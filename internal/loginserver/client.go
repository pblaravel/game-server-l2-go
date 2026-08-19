package loginserver

import (
	"bytes"
	"context"
	"crypto/rsa"
	"encoding/base64"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"github.com/pblaravel/game-server-l2-go/internal/crypt"
	"github.com/pblaravel/game-server-l2-go/internal/packet"
	"github.com/pblaravel/game-server-l2-go/internal/session"
)

// LoginClient matches Java LoginClientThread.
type LoginClient struct {
	conn       net.Conn
	ip         string
	ls         *LoginServerController
	gsc        *GameServerController
	crypt      *crypt.LoginCrypt
	pair       *crypt.ScrambledKeyPair
	blowfish   []byte
	sessionID  int32
	sessionKey session.Key
	state      LoginClientState

	mu            sync.Mutex
	username      string
	accessLevel   int
	lastGS        int
	joinedGS      bool
	charsOnServ   map[int]int
	expectedChars int
	lastEcho      time.Time
	closed        bool
}

func NewLoginClient(conn net.Conn, ls *LoginServerController, gsc *GameServerController) *LoginClient {
	pair := ls.GetScrambledRSAKeyPair()
	key := ls.GetBlowfishKey()
	return &LoginClient{
		conn:      conn,
		ip:        hostOf(conn.RemoteAddr()),
		ls:        ls,
		gsc:       gsc,
		crypt:     crypt.NewLoginCrypt(key),
		pair:      pair,
		blowfish:  key,
		sessionID: rndInt32(),
		state:     ClientConnected,
	}
}

func (c *LoginClient) Username() string            { return c.username }
func (c *LoginClient) AccessLevel() int            { return c.accessLevel }
func (c *LoginClient) LastGameserver() int         { return c.lastGS }
func (c *LoginClient) SessionKey() session.Key     { return c.sessionKey }
func (c *LoginClient) ConnectionIP() string        { return c.ip }
func (c *LoginClient) SetUsername(v string)        { c.username = v }
func (c *LoginClient) SetAccessLevel(v int)        { c.accessLevel = v }
func (c *LoginClient) SetLastGameserver(v int)     { c.lastGS = v }
func (c *LoginClient) SetSessionKey(k session.Key) { c.sessionKey = k }
func (c *LoginClient) SetJoinedGS(v bool)          { c.joinedGS = v }
func (c *LoginClient) SetLoginClientState(s LoginClientState) {
	c.state = s
}

func (c *LoginClient) SetExpectedCharacterCount(n int) { c.expectedChars = n }

func (c *LoginClient) SetCharsOnServ(servID, chars int) {
	c.mu.Lock()
	if c.charsOnServ == nil {
		c.charsOnServ = make(map[int]int)
	}
	c.charsOnServ[servID] = chars
	ready := len(c.charsOnServ) >= c.expectedChars
	c.mu.Unlock()
	if ready {
		if c.ls.Config().ShowLicense {
			c.Send(LoginOkPacket(c.sessionKey))
		} else {
			c.Send(c.buildServerList())
		}
	}
}

func (c *LoginClient) CharsOnServ() map[int]int { return c.charsOnServ }

func (c *LoginClient) Serve() {
	defer c.Disconnect()
	c.ls.AddClient(c)
	if err := c.Send(InitPacket(c.pair.ScrambledModulus, c.blowfish, c.sessionID)); err != nil {
		return
	}
	c.lastEcho = time.Now()
	go c.watchTimeout()
	for {
		body, err := packet.ReadFrame(c.conn)
		if err != nil {
			return
		}
		c.handle(body)
	}
}

// watchTimeout is the Java ClientPacketHandler timer on
// server.connection.timeout.ms: a client that stops sending Ping is dropped.
func (c *LoginClient) watchTimeout() {
	timeout := time.Duration(c.ls.Config().ConnectionTimeoutMS) * time.Millisecond
	if timeout <= 0 {
		return
	}
	t := time.NewTicker(timeout / 2)
	defer t.Stop()
	for range t.C {
		if c.Closed() {
			return
		}
		if time.Since(c.LastEcho()) >= timeout {
			log.Printf("[CLIENT] %s connection timed out after %s", c.ip, timeout)
			c.Disconnect()
			return
		}
	}
}

func (c *LoginClient) handle(data []byte) {
	// Java logs and keeps the connection when decryption or the checksum fails.
	if err := c.crypt.Decrypt(data); err != nil {
		log.Printf("[CLIENT] %s error while decrypting client packet: %v", c.ip, err)
		return
	}
	if len(data) == 0 {
		return
	}
	if c.ls.Config().PrintReceivedPackets && data[0] != ClientPing {
		log.Printf("[CLIENT] %s received packet 0x%02X (%d bytes)", c.ip, data[0], len(data))
	}
	switch data[0] {
	case ClientPing:
		c.SetLastEcho(time.Now())
		_ = c.Send(PingPacket())
	case ClientAuthRequest:
		c.onAuth(data)
	case ClientRequestServerList:
		c.onServerList(data)
	case ClientRequestServerLogin:
		c.onServerLogin(data)
	}
}

func (c *LoginClient) onAuth(data []byte) {
	creds, err := DecryptAuthRequest(data, func(ct []byte) ([]byte, error) {
		return crypt.DecryptPKCS1(c.pair.Private, ct)
	})
	if err != nil {
		c.CloseLoginFail(ReasonUserOrPassWrong)
		return
	}
	ctx := context.Background()
	info, err := c.ls.Accounts().GetAccountInfo(ctx, creds.Account)
	if err != nil {
		c.CloseLoginFail(ReasonUserOrPassWrong)
		return
	}
	if info != nil {
		if info.PassHash != creds.HashBase64 {
			c.CloseLoginFail(ReasonUserOrPassWrong)
			return
		}
	} else if c.ls.Config().AutoCreateAccount {
		info = &AccountInfo{
			Login:      creds.Account,
			PassHash:   creds.HashBase64,
			LastActive: time.Now().UnixMilli(),
			LastIP:     c.ip,
		}
		if err := c.ls.Accounts().CreateAccount(ctx, *info); err != nil {
			c.CloseLoginFail(ReasonUserOrPassWrong)
			return
		}
		log.Printf("Autocreated account %s.", creds.Account)
	} else {
		c.CloseLoginFail(ReasonUserOrPassWrong)
		return
	}

	switch c.tryCheckin(*info) {
	case AuthSuccess:
		c.username = info.Login
		c.state = ClientAuthedLogin
		c.sessionKey = c.ls.GetNewSessionKey()
		if c.ls.Config().ShowLicense {
			_ = c.Send(LoginOkPacket(c.sessionKey))
		} else {
			_ = c.Send(c.buildServerList())
		}
	case AuthInvalidPassword:
		c.CloseLoginFail(ReasonUserOrPassWrong)
	case AuthAccountInactive:
		c.CloseLoginFail(ReasonInactive)
	case AuthAccountBanned:
		c.CloseKicked(KickPermanentlyBanned)
	case AuthAlreadyOnLS:
		if old := c.ls.GetClient(info.Login); old != nil && old != c {
			old.CloseLoginFail(ReasonAccountInUse)
		}
		c.CloseLoginFail(ReasonAccountInUse)
	case AuthAlreadyOnGS:
		c.CloseLoginFail(ReasonAccountInUse)
		if gsi := c.accountOnAnyGS(info.Login); gsi != nil && gsi.IsAuthed() && gsi.Thread() != nil {
			gsi.Thread().KickPlayer(info.Login)
		}
	}
}

func (c *LoginClient) tryCheckin(info AccountInfo) AuthLoginResult {
	if info.AccessLevel < 0 {
		if info.AccessLevel == c.ls.Config().AccountInactiveLevel {
			return AuthAccountInactive
		}
		return AuthAccountBanned
	}
	if !c.canCheckIn(info) {
		return AuthInvalidPassword
	}
	if c.accountOnAnyGS(info.Login) != nil {
		return AuthAlreadyOnGS
	}
	if existing := c.ls.GetClient(info.Login); existing != nil && existing != c {
		return AuthAlreadyOnLS
	}
	return AuthSuccess
}

func (c *LoginClient) canCheckIn(info AccountInfo) bool {
	c.accessLevel = info.AccessLevel
	c.lastGS = info.LastServer
	info.LastIP = c.ip
	info.LastActive = time.Now().UnixMilli()
	if err := c.ls.Accounts().UpdateAccount(context.Background(), info); err != nil {
		return false
	}
	return true
}

func (c *LoginClient) accountOnAnyGS(account string) *GameServerInfo {
	for _, gsi := range c.gsc.GetRegisteredGameServers() {
		if t := gsi.Thread(); t != nil && t.HasAccountOnGameServer(account) {
			return gsi
		}
	}
	return nil
}

func (c *LoginClient) onServerList(data []byte) {
	s1, s2 := ParseRequestServerList(data)
	if !c.sessionKey.CheckLoginPair(s1, s2) {
		c.CloseLoginFail(ReasonAccessFailed)
		return
	}
	_ = c.Send(c.buildServerList())
}

func (c *LoginClient) onServerLogin(data []byte) {
	s1, s2, serverID := ParseRequestServerLogin(data)
	// Java: if (server.showLicense() || sk.checkLoginPair(skey1, skey2))
	if !(c.ls.Config().ShowLicense || c.sessionKey.CheckLoginPair(s1, s2)) {
		c.CloseLoginFail(ReasonAccessFailed)
		return
	}
	if c.ls.IsLoginPossible(c, int(serverID), c.gsc) {
		c.joinedGS = true
		_ = c.Send(PlayOkPacket(c.sessionKey))
	} else {
		_ = c.Send(PlayFailPacket(ReasonServerOverloaded))
	}
}

func (c *LoginClient) buildServerList() []byte {
	var entries []ServerListEntry
	for _, gsi := range c.gsc.GetRegisteredGameServers() {
		entries = append(entries, ServerListEntry{
			ID:             gsi.ID(),
			IP:             ResolveIPv4(gsi.Hostname()),
			Port:           int32(gsi.Port()),
			CurrentPlayers: int32(gsi.CurrentPlayerCount()),
			MaxPlayers:     int32(gsi.MaxPlayers()),
			Status:         byte(gsi.Status()),
		})
	}
	return ServerListPacket(byte(c.lastGS), entries, c.charsOnServ)
}

func (c *LoginClient) Send(payload []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return io.ErrClosedPipe
	}
	if c.ls.Config().PrintSentPackets && len(payload) > 0 {
		log.Printf("[CLIENT] %s sending packet 0x%02X (%d bytes)", c.ip, payload[0], len(payload))
	}
	dup := bytes.Clone(payload)
	if err := c.crypt.Encrypt(dup); err != nil {
		return err
	}
	return packet.WriteFrame(c.conn, dup)
}

func (c *LoginClient) Closed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}

func (c *LoginClient) LastEcho() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.lastEcho
}

func (c *LoginClient) SetLastEcho(t time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastEcho = t
}

func (c *LoginClient) CloseLoginFail(reason byte) {
	_ = c.Send(LoginFailPacket(reason))
	c.Disconnect()
}

func (c *LoginClient) CloseKicked(reason byte) {
	_ = c.Send(AccountKickedPacket(reason))
	c.Disconnect()
}

func (c *LoginClient) Disconnect() {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.closed = true
	c.mu.Unlock()
	c.ls.RemoveClient(c)
	_ = c.conn.Close()
}

func (c *LoginClient) RSAPrivate() *rsa.PrivateKey { return c.pair.Private }

func hostOf(addr net.Addr) string {
	if addr == nil {
		return ""
	}
	host, _, err := net.SplitHostPort(addr.String())
	if err != nil {
		return addr.String()
	}
	return host
}

// EncodePasswordHash is the Java Base64.encode(passHashBytes) helper.
func EncodePasswordHash(passHashBytes []byte) string {
	return base64.StdEncoding.EncodeToString(passHashBytes)
}
