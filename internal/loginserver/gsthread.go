package loginserver

import (
	"bytes"
	"crypto/rsa"
	"io"
	"log"
	"net"
	"sync"

	"github.com/pblaravel/game-server-l2-go/internal/crypt"
	"github.com/pblaravel/game-server-l2-go/internal/packet"
	"github.com/pblaravel/game-server-l2-go/internal/session"
)

// GameServerThread matches Java GameServerThread.
type GameServerThread struct {
	conn       net.Conn
	ip         string
	ls         *LoginServerController
	gsc        *GameServerController
	priv       *rsa.PrivateKey
	blowfish   *crypt.NewCrypt
	state      GameServerState
	gsi        *GameServerInfo

	mu       sync.Mutex
	accounts map[string]struct{}
	closed   bool
}

func NewGameServerThread(conn net.Conn, ls *LoginServerController, gsc *GameServerController) *GameServerThread {
	return &GameServerThread{
		conn:     conn,
		ip:       hostOf(conn.RemoteAddr()),
		ls:       ls,
		gsc:      gsc,
		priv:     gsc.GetKeyPair(),
		blowfish: crypt.New(crypt.DefaultGSBlowfishKey),
		state:    GSConnected,
		accounts: make(map[string]struct{}),
	}
}

func (t *GameServerThread) Serve() {
	defer t.Disconnect()
	mod := t.priv.PublicKey.N.Bytes()
	if err := t.Send(InitLSPacket(int32(t.ls.Config().Revision), mod)); err != nil {
		return
	}
	for {
		body, err := packet.ReadFrame(t.conn)
		if err != nil {
			return
		}
		t.handle(body)
	}
}

func (t *GameServerThread) handle(data []byte) {
	t.blowfish.Decrypt(data, 0, len(data))
	if !crypt.VerifyChecksum(data) {
		log.Printf("gameserver packet checksum failed from %s", t.ip)
		return
	}
	if len(data) == 0 {
		return
	}
	typ := data[0]
	switch t.state {
	case GSConnected:
		if typ == GSBlowFishKey {
			t.onBlowfish(data)
		} else {
			t.ForceClose(FailNotAuthed)
		}
	case GSBFConnected:
		if typ == GSAuthRequest {
			t.onAuth(data)
		} else {
			t.ForceClose(FailNotAuthed)
		}
	case GSAAuthed:
		switch typ {
		case GSServerStatus:
			t.onStatus(data)
		case GSPlayerInGame:
			for _, acc := range ParsePlayerInGame(data) {
				t.AddAccountOnGameServer(acc)
			}
		case GSPlayerLogout:
			t.RemoveAccountOnGameServer(ParsePlayerLogout(data))
		case GSReplyCharacters:
			acc, n := ParseReplyCharacters(data)
			t.ls.SetCharactersOnServer(acc, n, t.GetServerId())
		case GSPlayerAuthRequest:
			t.onPlayerAuth(data)
		}
	}
}

func (t *GameServerThread) onBlowfish(data []byte) {
	key, err := ParseBlowFishKey(data, func(ct []byte) ([]byte, error) {
		return crypt.DecryptNoPadding(t.priv, ct)
	})
	if err != nil {
		log.Printf("blowfish key decrypt: %v", err)
		t.ForceClose(FailNotAuthed)
		return
	}
	t.SetBlowfish(crypt.New(key))
	t.SetLoginConnectionState(GSBFConnected)
}

func (t *GameServerThread) onAuth(data []byte) {
	pkt := ParseGameServerAuth(data)
	if t.handleRegProcess(pkt) {
		_ = t.Send(AuthResponsePacket(t.gsi.ID()))
		t.SetLoginConnectionState(GSAAuthed)
		log.Printf("Game Server %d enabled.", t.gsi.ID())
	}
}

func (t *GameServerThread) handleRegProcess(pkt GameServerAuth) bool {
	gsi := t.gsc.GetRegisteredGameServerById(int(pkt.ID))
	if gsi != nil {
		if bytes.Equal(gsi.HexID(), pkt.HexID) {
			gsi.Lock()
			defer gsi.Unlock()
			if gsi.IsAuthed() {
				t.ForceClose(FailAlreadyLoggedIn)
				return false
			}
			t.AttachGameServerInfo(gsi, pkt.Port, pkt.Host, int(pkt.MaxPlayer))
			return true
		}
		if t.ls.Config().AcceptNewGameServer && pkt.AcceptAlternate {
			gsi = NewGameServerInfo(int(pkt.ID), pkt.HexID, t)
			if t.gsc.RegisterWithFirstAvailableId(gsi) {
				t.AttachGameServerInfo(gsi, pkt.Port, pkt.Host, int(pkt.MaxPlayer))
				t.gsc.RegisterServerOnDB(gsi)
				return true
			}
			t.ForceClose(FailNoFreeID)
			return false
		}
		t.ForceClose(FailWrongHexID)
		return false
	}
	if t.ls.Config().AcceptNewGameServer {
		gsi = NewGameServerInfo(int(pkt.ID), pkt.HexID, t)
		if t.gsc.Register(int(pkt.ID), gsi) {
			t.AttachGameServerInfo(gsi, pkt.Port, pkt.Host, int(pkt.MaxPlayer))
			t.gsc.RegisterServerOnDB(gsi)
			return true
		}
		t.ForceClose(FailIDReserved)
		return false
	}
	t.ForceClose(FailWrongHexID)
	return false
}

func (t *GameServerThread) onStatus(data []byte) {
	if t.gsi == nil {
		return
	}
	for _, a := range ParseServerStatus(data) {
		switch a.ID {
		case AttrServerListStatus:
			t.gsi.SetStatus(int(a.Value))
		case AttrMaxPlayers:
			t.gsi.SetMaxPlayers(int(a.Value))
		}
	}
}

func (t *GameServerThread) onPlayerAuth(data []byte) {
	account, incoming := ParsePlayerAuthRequest(data)
	// Rebuild the same way Java does: new SessionKey(loginOk1, loginOk2, playOk1, playOk2)
	incoming = session.New(incoming.LoginOkID1, incoming.LoginOkID2, incoming.PlayOkID1, incoming.PlayOkID2)
	key, ok := t.ls.GetKeyForAccount(account)
	authed := ok && key.Equals(incoming, t.ls.Config().ShowLicense)
	if authed {
		t.ls.RemoveAuthedClient(account)
	}
	_ = t.Send(PlayerAuthResponsePacket(account, authed))
}

func (t *GameServerThread) AttachGameServerInfo(gsi *GameServerInfo, port int, host string, maxPlayers int) {
	t.gsi = gsi
	gsi.SetThread(t)
	gsi.SetPort(port)
	t.SetGameHosts(host)
	gsi.SetMaxPlayers(maxPlayers)
	gsi.SetAuthed(true)
}

func (t *GameServerThread) SetGameHosts(hosts string) {
	if t.gsi == nil {
		return
	}
	if hosts != "*" {
		ip := ResolveIPv4(hosts)
		t.gsi.SetHostname(net.IP(ip[:]).String())
		return
	}
	t.gsi.SetHostname(t.ip)
}

func (t *GameServerThread) Send(payload []byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return io.ErrClosedPipe
	}
	dup := bytes.Clone(payload)
	crypt.AppendChecksum(dup)
	t.blowfish.Crypt(dup, 0, len(dup))
	return packet.WriteFrame(t.conn, dup)
}

func (t *GameServerThread) ForceClose(reason int) {
	_ = t.Send(LoginServerFailPacket(int32(reason)))
	t.Disconnect()
}

func (t *GameServerThread) Disconnect() {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return
	}
	t.closed = true
	t.mu.Unlock()
	if t.gsi != nil && t.gsi.IsAuthed() {
		t.gsi.SetDown()
		log.Printf("Server %s[%d] is now disconnected.", ServerName(t.gsi.ID()), t.gsi.ID())
	}
	_ = t.conn.Close()
}

func (t *GameServerThread) GetPlayerCount() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.accounts)
}

func (t *GameServerThread) HasAccountOnGameServer(account string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	_, ok := t.accounts[account]
	return ok
}

func (t *GameServerThread) AddAccountOnGameServer(account string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.accounts[account] = struct{}{}
}

func (t *GameServerThread) RemoveAccountOnGameServer(account string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.accounts, account)
}

func (t *GameServerThread) SetBlowfish(c *crypt.NewCrypt) { t.blowfish = c }
func (t *GameServerThread) SetLoginConnectionState(s GameServerState) {
	t.state = s
}
func (t *GameServerThread) GetServerId() int {
	if t.gsi == nil {
		return 0
	}
	return t.gsi.ID()
}
func (t *GameServerThread) GetServerName() string {
	if t.gsi == nil {
		return ""
	}
	return t.gsi.Name()
}
func (t *GameServerThread) RequestCharacters(account string) {
	_ = t.Send(RequestCharactersPacket(account))
}
func (t *GameServerThread) KickPlayer(account string) {
	_ = t.Send(KickPlayerPacket(toLowerTrim(account)))
}

func IsBannedGameServerIP(string) bool { return false }
