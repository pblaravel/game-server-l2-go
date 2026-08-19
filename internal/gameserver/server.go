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
	ai    *aiState

	mu      sync.Mutex
	clients []*GameClient
}

func NewServer(cfg config.GameConfig, store CharacterStore) *Server {
	if !DatapackLoaded() {
		if err := LoadDatapack(FindDataDir()); err != nil {
			log.Printf("datapack: %v (using builtin newbie kits)", err)
		}
	}
	w := NewWorld()
	s := &Server{cfg: cfg, world: w, store: store, ai: newAIState()}
	s.login = NewLoginServerThread(cfg, w)
	s.SeedWorld(nil)
	return s
}

// SeedWorld replaces world NPCs. Empty list falls back to the Interlude newbie set.
func (s *Server) SeedWorld(npcs []NPC) {
	s.world.ClearNPCs()
	if len(npcs) == 0 {
		s.world.LoadDefaultSpawns()
		return
	}
	for _, n := range npcs {
		cp := n
		if cp.ObjectID == 0 {
			cp.ObjectID = s.world.NextID()
		}
		if cp.CurHP == 0 {
			cp.CurHP = cp.MaxHP
		}
		cp.NpcDefaults()
		s.world.AddNPC(&cp)
	}
}

func (s *Server) Login() *LoginServerThread { return s.login }
func (s *Server) World() *World             { return s.world }

func (s *Server) Run(ctx context.Context) error {
	go s.login.Run()
	s.RunTaskManagers(ctx)
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
		if tc, ok := conn.(*net.TCPConn); ok {
			_ = tc.SetNoDelay(true)
			_ = tc.SetKeepAlive(true)
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
