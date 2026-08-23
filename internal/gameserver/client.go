package gameserver

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

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

	attacking atomic.Bool
	casting   atomic.Bool
	reuse     sync.Map // skill id -> time.Time when it becomes usable again
	lastHit   time.Time
	lastPvP   time.Time

	activeEnchant int32
	multiSellID   int32

	record bool
	sent   [][]byte
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

func (c *GameClient) SetState(s ClientState) {
	old := c.state
	c.state = s
	if old != s && c.logTraceEnabled() {
		log.Printf("GS STATE %s %s -> %s", c.tag(), old, s)
	}
}

func (c *GameClient) State() ClientState { return c.state }
func (c *GameClient) SetAccountName(v string) {
	c.account = v
	c.logChange("account=%q", v)
}
func (c *GameClient) AccountName() string         { return c.account }
func (c *GameClient) SetSessionKey(k session.Key) { c.sess = k }
func (c *GameClient) SessionKey() session.Key     { return c.sess }
func (c *GameClient) SetSlots(s []*Character)     { c.slots = s }
func (c *GameClient) Slots() []*Character         { return c.slots }
func (c *GameClient) SetPlayer(p *Character) {
	c.player = p
	if p != nil {
		c.logChange("selected char name=%q oid=%d class=%d pos=(%d,%d,%d)", p.Name, p.ObjectID, p.ClassID, p.X, p.Y, p.Z)
	}
}
func (c *GameClient) Player() *Character   { return c.player }
func (c *GameClient) ctx() context.Context { return context.Background() }

func (c *GameClient) Serve() {
	if c.logTraceEnabled() {
		log.Printf("GS ACCEPT %s", c.tag())
	}
	defer c.Close()
	for {
		body, err := packet.ReadFrame(c.conn)
		if err != nil {
			if c.logTraceEnabled() && err != io.EOF {
				log.Printf("GS DISCONNECT %s err=%v", c.tag(), err)
			} else if c.logTraceEnabled() {
				log.Printf("GS DISCONNECT %s", c.tag())
			}
			return
		}
		c.crypt.Decrypt(body)
		c.logRecv(body)
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
	if c.record {
		c.sent = append(c.sent, bytes.Clone(payload))
	}
	c.logSend(payload)
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
	if c.logTraceEnabled() {
		who := c.account
		if c.player != nil {
			who = fmt.Sprintf("%s/%s#%d", c.account, c.player.Name, c.player.ObjectID)
		}
		log.Printf("GS CLOSE %s %s", hostOnly(c.conn.RemoteAddr()), who)
	}
	if c.player != nil {
		if pt := c.server.partyOf(c.player); pt != nil {
			c.server.removeFromParty(pt, c.player, true)
		}
		c.server.world.RemovePlayer(c.player.ObjectID)
		c.server.login.SendLogout(c.account)
	}
	_ = c.conn.Close()
}

func (c *GameClient) RecordSends() {
	c.mu.Lock()
	c.record = true
	c.sent = nil
	c.mu.Unlock()
}

func (c *GameClient) Sent() [][]byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([][]byte, len(c.sent))
	for i, p := range c.sent {
		out[i] = append([]byte(nil), p...)
	}
	return out
}

func (c *GameClient) SentOpcodes() []byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]byte, 0, len(c.sent))
	for _, p := range c.sent {
		if len(p) > 0 {
			out = append(out, p[0])
		}
	}
	return out
}

func (c *GameClient) CloseNow() { c.Close() }

func (c *GameClient) Closed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}

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
