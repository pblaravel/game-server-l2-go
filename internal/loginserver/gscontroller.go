package loginserver

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"log"
	"sync"

	"github.com/pblaravel/game-server-l2-go/internal/config"
)

const gsRSAKeys = 10

// GameServerController matches Java GameServerController public API.
type GameServerController struct {
	cfg   config.LoginConfig
	store GameServerStore

	mu      sync.Mutex
	servers map[int]*GameServerInfo
	keyPairs []*rsa.PrivateKey
}

func NewGameServerController(cfg config.LoginConfig, store GameServerStore) (*GameServerController, error) {
	c := &GameServerController{
		cfg:     cfg,
		store:   store,
		servers: make(map[int]*GameServerInfo),
	}
	if err := c.loadRegisteredGameServers(); err != nil {
		return nil, err
	}
	c.keyPairs = make([]*rsa.PrivateKey, gsRSAKeys)
	for i := 0; i < gsRSAKeys; i++ {
		k, err := rsa.GenerateKey(rand.Reader, 512)
		if err != nil {
			return nil, err
		}
		c.keyPairs[i] = k
	}
	log.Printf("GameServerController: loaded %d registered Game Servers, cached %d RSA keys.", len(c.servers), len(c.keyPairs))
	return c, nil
}

func (c *GameServerController) loadRegisteredGameServers() error {
	rows, err := c.store.GetAllGameServers(context.Background())
	if err != nil {
		return err
	}
	for _, row := range rows {
		c.servers[row.ServerID] = NewGameServerInfo(row.ServerID, StringToHex(row.HexID), nil)
	}
	return nil
}

func (c *GameServerController) GetRegisteredGameServers() map[int]*GameServerInfo {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make(map[int]*GameServerInfo, len(c.servers))
	for k, v := range c.servers {
		out[k] = v
	}
	return out
}

func (c *GameServerController) GetRegisteredGameServerById(id int) *GameServerInfo {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.servers[id]
}

func (c *GameServerController) HasRegisteredGameServerOnId(id int) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, ok := c.servers[id]
	return ok
}

func (c *GameServerController) RegisterWithFirstAvailableId(gsi *GameServerInfo) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i := 0; i < 16; i++ {
		if _, ok := c.servers[i]; !ok {
			c.servers[i] = gsi
			gsi.SetID(i)
			return true
		}
	}
	return false
}

func (c *GameServerController) Register(id int, gsi *GameServerInfo) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.servers[id]; ok {
		return false
	}
	c.servers[id] = gsi
	return true
}

func (c *GameServerController) RegisterServerOnDB(gsi *GameServerInfo) {
	c.RegisterServerOnDBBytes(gsi.HexID(), gsi.ID(), gsi.Hostname())
}

func (c *GameServerController) RegisterServerOnDBBytes(hexID []byte, id int, host string) {
	c.Register(id, NewGameServerInfo(id, hexID, nil))
	_ = c.store.AddGameServer(context.Background(), GameServerRow{
		ServerID: id,
		HexID:    HexToString(hexID),
		Host:     host,
	})
}

func (c *GameServerController) GetKeyPair() *rsa.PrivateKey {
	return c.keyPairs[rndN(len(c.keyPairs))]
}
