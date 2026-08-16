package gameserver

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/pblaravel/game-server-l2-go/internal/config"
)

// Server is the gameserver process (Java GameServer).
type Server struct {
	cfg   config.GameConfig
	world *World
	store CharacterStore
	login *LoginServerThread

	mu      sync.Mutex
	clients []*GameClient
}

func NewServer(cfg config.GameConfig, store CharacterStore) *Server {
	w := NewWorld()
	s := &Server{cfg: cfg, world: w, store: store}
	s.login = NewLoginServerThread(cfg, w)
	s.seedNPCs()
	return s
}

func (s *Server) Login() *LoginServerThread { return s.login }
func (s *Server) World() *World             { return s.world }

func (s *Server) Run(ctx context.Context) error {
	go s.login.Run()
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", s.cfg.GameserverPort))
	if err != nil {
		return fmt.Errorf("gameserver listen: %w", err)
	}
	log.Printf("GameServer listening for clients on :%d", s.cfg.GameserverPort)
	go func() {
		<-ctx.Done()
		s.login.Stop()
		_ = ln.Close()
	}()
	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		c := NewGameClient(conn, s)
		s.mu.Lock()
		s.clients = append(s.clients, c)
		s.mu.Unlock()
		go c.Serve()
	}
}

func (s *Server) Broadcast(payload []byte, except *GameClient) {
	s.mu.Lock()
	clients := append([]*GameClient(nil), s.clients...)
	s.mu.Unlock()
	for _, c := range clients {
		if c == except || c.State() != StateInGame {
			continue
		}
		c.Send(payload)
	}
}

func (s *Server) seedNPCs() {
	// Talking Island newbie helpers so EnterWorld is not empty.
	s.world.AddNPC(&NPC{
		ObjectID: s.world.NextID(), NPCID: 30006, Name: "Gatekeeper Roxxy",
		X: -71338, Y: 258271, Z: -3104, Heading: 0, Level: 70,
		MaxHP: 10000, CurHP: 10000, IsAttackable: false,
	})
	s.world.AddNPC(&NPC{
		ObjectID: s.world.NextID(), NPCID: 30001, Name: "Weapon Merchant Lector",
		X: -71400, Y: 258200, Z: -3104, Heading: 0, Level: 70,
		MaxHP: 8000, CurHP: 8000, IsAttackable: false,
	})
}
