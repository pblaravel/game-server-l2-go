package loginserver

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/binary"
	"log"
	"sync"

	"github.com/pblaravel/game-server-l2-go/internal/config"
	"github.com/pblaravel/game-server-l2-go/internal/crypt"
	"github.com/pblaravel/game-server-l2-go/internal/session"
)

const (
	rsaKeyPairs     = 10
	blowfishKeyPool = 20
)

// LoginServerController matches Java LoginServerController public API.
type LoginServerController struct {
	cfg      config.LoginConfig
	accounts AccountStore

	mu      sync.Mutex
	clients []*LoginClient

	keyPairs     []*crypt.ScrambledKeyPair
	blowfishKeys [][]byte
}

func NewLoginServerController(cfg config.LoginConfig, accounts AccountStore) (*LoginServerController, error) {
	c := &LoginServerController{cfg: cfg, accounts: accounts}
	c.keyPairs = make([]*crypt.ScrambledKeyPair, rsaKeyPairs)
	for i := 0; i < rsaKeyPairs; i++ {
		kp, err := crypt.NewScrambledKeyPair(1024)
		if err != nil {
			return nil, err
		}
		c.keyPairs[i] = kp
	}
	c.blowfishKeys = make([][]byte, blowfishKeyPool)
	for i := 0; i < blowfishKeyPool; i++ {
		key := make([]byte, 16)
		for j := 0; j < 16; j++ {
			var b [1]byte
			_, _ = rand.Read(b[:])
			if b[0] == 0 {
				b[0] = 1
			}
			key[j] = b[0]
		}
		c.blowfishKeys[i] = key
	}
	log.Printf("Cached %d KeyPairs for RSA communication.", rsaKeyPairs)
	log.Printf("Stored %d keys for Blowfish communication.", blowfishKeyPool)
	return c, nil
}

func (c *LoginServerController) AddClient(client *LoginClient) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.clients = append(c.clients, client)
}

func (c *LoginServerController) GetClient(login string) *LoginClient {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, cl := range c.clients {
		if cl.Username() == login {
			return cl
		}
	}
	return nil
}

func (c *LoginServerController) RemoveClient(s *LoginClient) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i, cl := range c.clients {
		if cl == s {
			c.clients = append(c.clients[:i], c.clients[i+1:]...)
			return
		}
	}
}

func (c *LoginServerController) GetAllClients() []*LoginClient {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]*LoginClient, len(c.clients))
	copy(out, c.clients)
	return out
}

func (c *LoginServerController) GetScrambledRSAKeyPair() *crypt.ScrambledKeyPair {
	return c.keyPairs[rndN(len(c.keyPairs))]
}

func (c *LoginServerController) GetBlowfishKey() []byte {
	return c.blowfishKeys[rndN(len(c.blowfishKeys))]
}

func (c *LoginServerController) GetNewSessionKey() session.Key {
	return session.New(rndInt32(), rndInt32(), rndInt32(), rndInt32())
}

func (c *LoginServerController) GetCharactersOnAccount(client *LoginClient, account string, gsc *GameServerController) {
	servers := gsc.GetRegisteredGameServers()
	client.SetExpectedCharacterCount(len(servers))
	for _, gsi := range servers {
		if gsi.IsAuthed() && gsi.Thread() != nil {
			gsi.Thread().RequestCharacters(account)
		} else {
			client.SetCharsOnServ(gsi.ID(), 0)
		}
	}
}

func (c *LoginServerController) SetCharactersOnServer(account string, charsNum, serverID int) {
	client := c.GetClient(account)
	if client == nil {
		return
	}
	client.SetCharsOnServ(serverID, charsNum)
}

func (c *LoginServerController) IsLoginPossible(client *LoginClient, serverID int, gsc *GameServerController) bool {
	gsi := gsc.GetRegisteredGameServerById(serverID)
	access := client.AccessLevel()
	if gsi != nil && gsi.IsAuthed() {
		loginOk := (gsi.CurrentPlayerCount() < gsi.MaxPlayers() && gsi.Status() != StatusGMOnly) || access > 0
		if loginOk && client.LastGameserver() != serverID {
			_ = c.accounts.UpdateAccountLastServer(context.Background(), client.Username(), serverID)
		}
		return loginOk
	}
	return false
}

func (c *LoginServerController) GetKeyForAccount(account string) (session.Key, bool) {
	client := c.GetClient(account)
	if client == nil {
		return session.Key{}, false
	}
	return client.SessionKey(), true
}

func (c *LoginServerController) RemoveAuthedClient(account string) {
	if client := c.GetClient(account); client != nil {
		c.RemoveClient(client)
	}
}

func (c *LoginServerController) Accounts() AccountStore { return c.accounts }
func (c *LoginServerController) Config() config.LoginConfig {
	return c.cfg
}

func rndN(n int) int {
	if n <= 0 {
		return 0
	}
	var b [8]byte
	_, _ = rand.Read(b[:])
	return int(binary.LittleEndian.Uint64(b[:]) % uint64(n))
}

func rndInt32() int32 {
	var b [4]byte
	_, _ = rand.Read(b[:])
	return int32(binary.LittleEndian.Uint32(b[:]))
}

// RSAPrivateFromPKCS1 is a helper for tests.
func RSAPrivateFromPKCS1(der []byte) (*rsa.PrivateKey, error) {
	return x509.ParsePKCS1PrivateKey(der)
}
