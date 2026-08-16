package gameserver

import (
	"bytes"
	"context"
	"io"
	"log"
	"net"
	"sync"

	"github.com/pblaravel/game-server-l2-go/internal/crypt"
	"github.com/pblaravel/game-server-l2-go/internal/packet"
	"github.com/pblaravel/game-server-l2-go/internal/session"
)

// GameClient matches Java GameClient.
type GameClient struct {
	conn   net.Conn
	server *Server
	crypt  *crypt.GameCrypt
	key    []byte

	mu      sync.Mutex
	state   ClientState
	account string
	sess    session.Key
	slots   []*Character
	player  *Character
	closed  bool
	target  int32
}

func NewGameClient(conn net.Conn, srv *Server) *GameClient {
	key := crypt.RandomGameKey()
	gc := crypt.NewGameCrypt(srv.cfg.UseBlowfishCipher)
	gc.SetKey(key)
	return &GameClient{
		conn:   conn,
		server: srv,
		crypt:  gc,
		key:    key,
		state:  StateConnected,
	}
}

func (c *GameClient) EnableCrypt() []byte { return c.key }
func (c *GameClient) SetState(s ClientState) { c.state = s }
func (c *GameClient) State() ClientState     { return c.state }
func (c *GameClient) SetAccountName(v string) { c.account = v }
func (c *GameClient) AccountName() string     { return c.account }
func (c *GameClient) SetSessionKey(k session.Key) { c.sess = k }
func (c *GameClient) SessionKey() session.Key     { return c.sess }
func (c *GameClient) SetSlots(s []*Character)     { c.slots = s }
func (c *GameClient) Slots() []*Character         { return c.slots }
func (c *GameClient) SetPlayer(p *Character)      { c.player = p }
func (c *GameClient) Player() *Character          { return c.player }
func (c *GameClient) ctx() context.Context        { return context.Background() }

func (c *GameClient) Serve() {
	defer c.Close()
	for {
		body, err := packet.ReadFrame(c.conn)
		if err != nil {
			return
		}
		c.crypt.Decrypt(body)
		c.server.handle(c, body)
	}
}

func (c *GameClient) Send(payload []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	dup := bytes.Clone(payload)
	c.crypt.Encrypt(dup)
	if err := packet.WriteFrame(c.conn, dup); err != nil {
		log.Printf("send: %v", err)
	}
}

func (c *GameClient) Close() {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.closed = true
	c.mu.Unlock()
	if c.player != nil {
		c.server.world.RemovePlayer(c.player.ObjectID)
		c.server.login.SendLogout(c.account)
	}
	_ = c.conn.Close()
}

func (c *GameClient) CloseNow() { c.Close() }

func (c *GameClient) Broadcast(payload []byte) {
	c.server.Broadcast(payload, c)
}

func hostOnly(addr net.Addr) string {
	if addr == nil {
		return ""
	}
	host, _, err := net.SplitHostPort(addr.String())
	if err != nil {
		return addr.String()
	}
	return host
}

var _ = io.EOF
