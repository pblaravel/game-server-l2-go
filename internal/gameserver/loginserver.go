package gameserver

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"sync"
	"time"

	"github.com/pblaravel/game-server-l2-go/internal/config"
	"github.com/pblaravel/game-server-l2-go/internal/crypt"
	"github.com/pblaravel/game-server-l2-go/internal/packet"
	"github.com/pblaravel/game-server-l2-go/internal/session"
)

const loginRevision = 0x0102

// LoginServerThread matches Java LoginServerThread public API.
type LoginServerThread struct {
	cfg   config.GameConfig
	world *World

	mu       sync.Mutex
	clients  map[string]*GameClient
	bf       *crypt.NewCrypt
	conn     net.Conn
	hexID    []byte
	reqID    int
	maxP     int
	name     string
	serverID int
	typ      int
	stopped  bool
}

func NewLoginServerThread(cfg config.GameConfig, world *World) *LoginServerThread {
	hex := cfg.HexID
	req := cfg.RequestID
	if len(hex) == 0 {
		hex = make([]byte, 16)
		_, _ = rand.Read(hex)
		log.Printf("Generated random hexID; will save to %s after login auth", cfg.HexIDFile)
	} else {
		req = cfg.ServerID
		if req == 0 {
			req = cfg.RequestID
		}
		log.Printf("Using saved hexID from %s (server %d)", cfg.HexIDFile, req)
	}
	return &LoginServerThread{
		cfg:     cfg,
		world:   world,
		clients: make(map[string]*GameClient),
		hexID:   hex,
		reqID:   req,
		maxP:    cfg.MaximumOnlineUsers,
		typ:     0,
	}
}

func (t *LoginServerThread) Run() {
	for !t.stopped {
		if err := t.connectOnce(); err != nil {
			log.Printf("No connection found with loginserver, next try in 10 seconds.")
			time.Sleep(10 * time.Second)
		}
	}
}

func (t *LoginServerThread) Stop() { t.stopped = true }

func (t *LoginServerThread) connectOnce() error {
	log.Printf("Connecting to login on %s:%d.", t.cfg.LoginHost, t.cfg.LoginPort)
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(t.cfg.LoginHost, itoa(t.cfg.LoginPort)), 5*time.Second)
	if err != nil {
		return err
	}
	t.mu.Lock()
	t.conn = conn
	t.bf = crypt.New(crypt.DefaultGSBlowfishKey)
	t.mu.Unlock()
	defer conn.Close()

	bfKey := make([]byte, 40)
	_, _ = rand.Read(bfKey)

	for {
		body, err := packet.ReadFrame(conn)
		if err != nil {
			return err
		}
		t.mu.Lock()
		t.bf.Decrypt(body, 0, len(body))
		t.mu.Unlock()
		if !crypt.VerifyChecksum(body) {
			return io.ErrUnexpectedEOF
		}
		if len(body) == 0 {
			continue
		}
		t.logLSIn(body)
		switch body[0] {
		case 0x00:
			if err := t.onInitLS(body, bfKey); err != nil {
				return err
			}
		case 0x01:
			log.Printf("LoginServer registration failed.")
			return io.EOF
		case 0x02:
			t.onAuthResponse(body)
		case 0x03:
			t.onPlayerAuth(body)
		case 0x04:
			t.onKick(body)
		}
	}
}

func (t *LoginServerThread) onInitLS(body, bfKey []byte) error {
	r := packet.NewReader(body)
	r.SkipOpcode()
	rev := r.ReadD()
	if rev != loginRevision {
		log.Printf("Revision mismatch between LS and GS.")
		return nil
	}
	n := int(r.ReadD())
	mod := r.ReadB(n)
	pub := &rsa.PublicKey{N: new(big.Int).SetBytes(mod), E: 65537}
	enc, err := crypt.EncryptNoPadding(pub, bfKey)
	if err != nil {
		return err
	}
	if err := t.send(blowFishKeyPacket(enc)); err != nil {
		return err
	}
	t.mu.Lock()
	t.bf = crypt.New(bfKey)
	t.mu.Unlock()
	host := t.cfg.Hostname
	if host == "*" {
		host = "*"
	}
	log.Printf("Sending auth request with params ID: %d Hostname: %s Port: %d MaxPlayers: %d", t.reqID, host, t.cfg.GameserverPort, t.maxP)
	return t.send(authRequestPacket(t.reqID, t.cfg.AcceptAlternateID, t.hexID, host, t.cfg.GameserverPort, t.cfg.ReserveHostOnLogin, t.maxP))
}

func (t *LoginServerThread) onAuthResponse(body []byte) {
	r := packet.NewReader(body)
	r.SkipOpcode()
	t.serverID = int(r.ReadC())
	t.name = r.ReadS()
	path := t.cfg.HexIDFile
	if path == "" {
		path = "conf/gameserver/hexid.txt"
	}
	if err := config.SaveHexID(path, t.serverID, t.hexID); err != nil {
		log.Printf("Failed to save hex ID to %s: %v", path, err)
	} else {
		log.Printf("Saved hex ID to %s (server %d). Next start will reuse gameservers row.", path, t.serverID)
	}
	log.Printf("Registered as server: [%d] %s.", t.serverID, t.name)
	// Java LoginServerThread sends the whole server list block after registration.
	status := serverTypeAuto
	if t.cfg.ServerGMOnly {
		status = serverTypeGMOnly
	}
	_ = t.send(serverStatusPacket([][2]int32{
		{attrStatus, status},
		{attrClock, boolAttr(t.cfg.ServerListClock)},
		{attrBrackets, boolAttr(t.cfg.ServerListBrackets)},
		{attrAgeLimit, int32(t.cfg.ServerListAge)},
		{attrTestServer, boolAttr(t.cfg.ServerListTestServer)},
		{attrPvPServer, boolAttr(t.cfg.ServerListPvPServer)},
		{attrMaxPlayers, int32(t.cfg.MaximumOnlineUsers)},
	}))
	var names []string
	for _, p := range t.world.Players() {
		names = append(names, p.Account)
	}
	if len(names) > 0 {
		_ = t.send(playerInGamePacket(names))
	}
}

// Attribute and server type ids from Java commons/network/AttributeType and
// enums/ServerType.
const (
	attrStatus     int32 = 1
	attrClock      int32 = 2
	attrBrackets   int32 = 3
	attrAgeLimit   int32 = 4
	attrTestServer int32 = 5
	attrPvPServer  int32 = 6
	attrMaxPlayers int32 = 7

	serverTypeAuto   int32 = 0
	serverTypeGMOnly int32 = 5
)

func boolAttr(v bool) int32 {
	if v {
		return 1
	}
	return 0
}

func (t *LoginServerThread) onPlayerAuth(body []byte) {
	r := packet.NewReader(body)
	r.SkipOpcode()
	account := r.ReadS()
	authed := r.ReadC() == 1
	t.mu.Lock()
	client := t.clients[account]
	t.mu.Unlock()
	if client == nil {
		return
	}
	if authed {
		_ = t.send(playerInGamePacket([]string{account}))
		client.SetState(StateAuthed)
		slots, _ := client.server.store.ListByAccount(client.ctx(), account)
		client.SetSlots(slots)
		client.Send(CharSelectInfo(account, client.SessionKey().PlayOkID1, slots))
	} else {
		client.Send(AuthLoginFail(1))
		client.Close()
	}
}

func (t *LoginServerThread) onKick(body []byte) {
	r := packet.NewReader(body)
	r.SkipOpcode()
	t.KickPlayer(r.ReadS())
}

func (t *LoginServerThread) AddClient(loginName string, loginKey1, loginKey2, playKey1, playKey2 int32, client *GameClient) {
	t.mu.Lock()
	if old, ok := t.clients[loginName]; ok {
		old.Close()
	}
	t.clients[loginName] = client
	t.mu.Unlock()
	client.SetAccountName(loginName)
	// Java: new SessionKey(loginKey1, loginKey2, playKey1, playKey2) on record (play, play, login, login)
	client.SetSessionKey(session.NewGameClientKey(loginKey1, loginKey2, playKey1, playKey2))
	_ = t.send(playerAuthRequestPacket(loginName, client.SessionKey()))
}

func (t *LoginServerThread) SendLogout(account string) {
	if account == "" {
		return
	}
	_ = t.send(playerLogoutPacket(account))
	t.mu.Lock()
	delete(t.clients, account)
	t.mu.Unlock()
}

func (t *LoginServerThread) SendAccessLevel(account string, level int32) {
	_ = t.send(changeAccessLevelPacket(account, level))
}

func (t *LoginServerThread) KickPlayer(account string) {
	t.mu.Lock()
	c := t.clients[account]
	t.mu.Unlock()
	if c != nil {
		c.Close()
	}
}

func (t *LoginServerThread) SetMaxPlayer(n int) {
	t.maxP = n
	_ = t.send(serverStatusPacket([][2]int32{{7, int32(n)}}))
}

func (t *LoginServerThread) GetMaxPlayers() int { return t.maxP }

func (t *LoginServerThread) SetServerType(typ int) {
	t.typ = typ
	_ = t.send(serverStatusPacket([][2]int32{{1, int32(typ)}}))
}

func (t *LoginServerThread) GetServerType() int    { return t.typ }
func (t *LoginServerThread) GetServerName() string { return t.name }
func (t *LoginServerThread) GetServerID() int      { return t.serverID }

func (t *LoginServerThread) send(payload []byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.conn == nil {
		return io.ErrClosedPipe
	}
	t.logLSOut(payload)
	dup := bytes.Clone(payload)
	crypt.AppendChecksum(dup)
	t.bf.Crypt(dup, 0, len(dup))
	return packet.WriteFrame(t.conn, dup)
}

func (t *LoginServerThread) logLSIn(body []byte) {
	if !(t.cfg.PacketHandlerDebug || t.cfg.PrintReceivedPackets || t.cfg.Developer) || len(body) == 0 {
		return
	}
	op := body[0]
	log.Printf("GS<-LS RECV %s 0x%02X %s", lsInOpcodeName(op), op, hexPreview(body, 32))
}

func (t *LoginServerThread) logLSOut(body []byte) {
	if !(t.cfg.PacketHandlerDebug || t.cfg.PrintSentPackets || t.cfg.Developer) || len(body) == 0 {
		return
	}
	op := body[0]
	extra := ""
	switch op {
	case 0x01:
		extra = fmt.Sprintf(" id=%d", t.reqID)
	case 0x03:
		extra = " logout"
	case 0x05:
		extra = " auth"
	}
	log.Printf("GS->LS SEND %s 0x%02X%s %s", lsOutOpcodeName(op), op, extra, hexPreview(body, 32))
}

func blowFishKeyPacket(enc []byte) []byte {
	w := packet.NewWriter()
	w.WriteC(0x00)
	w.WriteD(int32(len(enc)))
	w.WriteB(enc)
	w.PadTo8()
	return w.Bytes()
}

func authRequestPacket(id int, acceptAlt bool, hex []byte, host string, port int, reserve bool, maxP int) []byte {
	w := packet.NewWriter()
	w.WriteC(0x01)
	w.WriteC(id)
	if acceptAlt {
		w.WriteC(1)
	} else {
		w.WriteC(0)
	}
	if reserve {
		w.WriteC(1)
	} else {
		w.WriteC(0)
	}
	w.WriteS(host)
	w.WriteH(port)
	w.WriteD(int32(maxP))
	w.WriteD(int32(len(hex)))
	w.WriteB(hex)
	w.PadTo8()
	return w.Bytes()
}

func playerAuthRequestPacket(account string, key session.Key) []byte {
	w := packet.NewWriter()
	w.WriteC(0x05)
	w.WriteS(account)
	w.WriteD(key.PlayOkID1)
	w.WriteD(key.PlayOkID2)
	w.WriteD(key.LoginOkID1)
	w.WriteD(key.LoginOkID2)
	w.PadTo8()
	return w.Bytes()
}

func playerInGamePacket(names []string) []byte {
	w := packet.NewWriter()
	w.WriteC(0x02)
	w.WriteH(len(names))
	for _, n := range names {
		w.WriteS(n)
	}
	w.PadTo8()
	return w.Bytes()
}

func playerLogoutPacket(account string) []byte {
	w := packet.NewWriter()
	w.WriteC(0x03)
	w.WriteS(account)
	w.PadTo8()
	return w.Bytes()
}

func changeAccessLevelPacket(account string, level int32) []byte {
	w := packet.NewWriter()
	w.WriteC(0x04)
	w.WriteS(account)
	w.WriteD(level)
	w.PadTo8()
	return w.Bytes()
}

func serverStatusPacket(attrs [][2]int32) []byte {
	w := packet.NewWriter()
	w.WriteC(0x06)
	w.WriteD(int32(len(attrs)))
	for _, a := range attrs {
		w.WriteD(a[0])
		w.WriteD(a[1])
	}
	w.PadTo8()
	return w.Bytes()
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [16]byte
	i := len(b)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
