package loginserver

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/pblaravel/game-server-l2-go/internal/config"
)

// Server is the combined login-server process (Java Main.runServer).
type Server struct {
	cfg config.LoginConfig
	LS  *LoginServerController
	GSC *GameServerController
}

func NewServer(cfg config.LoginConfig, accounts AccountStore, games GameServerStore) (*Server, error) {
	ls, err := NewLoginServerController(cfg, accounts)
	if err != nil {
		return nil, err
	}
	gsc, err := NewGameServerController(cfg, games)
	if err != nil {
		return nil, err
	}
	return &Server{cfg: cfg, LS: ls, GSC: gsc}, nil
}

func (s *Server) Run(ctx context.Context) error {
	clientLn, err := net.Listen("tcp", fmt.Sprintf(":%d", s.cfg.LoginServerPort))
	if err != nil {
		return fmt.Errorf("login client listen: %w", err)
	}
	gsLn, err := net.Listen("tcp", fmt.Sprintf(":%d", s.cfg.GameServerPort))
	if err != nil {
		_ = clientLn.Close()
		return fmt.Errorf("gameserver listen: %w", err)
	}
	log.Printf("LoginServer listening for clients on :%d", s.cfg.LoginServerPort)
	log.Printf("LoginServer listening for gameservers on :%d", s.cfg.GameServerPort)

	go s.acceptClients(clientLn)
	go s.acceptGameServers(gsLn)

	<-ctx.Done()
	_ = clientLn.Close()
	_ = gsLn.Close()
	s.shutdown()
	return nil
}

// shutdown is Java ServerShutdownService: the listeners stop first, then every
// live client and gameserver connection is dropped.
func (s *Server) shutdown() {
	clients := s.LS.GetAllClients()
	for _, c := range clients {
		c.Disconnect()
	}
	threads := s.GSC.GetRegisteredGameServers()
	for _, gsi := range threads {
		if t := gsi.Thread(); t != nil {
			t.Disconnect()
		}
	}
	log.Printf("LoginServer shutdown: closed %d clients and %d gameservers", len(clients), len(threads))
}

func (s *Server) acceptClients(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		go NewLoginClient(conn, s.LS, s.GSC).Serve()
	}
}

func (s *Server) acceptGameServers(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		go NewGameServerThread(conn, s.LS, s.GSC).Serve()
	}
}
