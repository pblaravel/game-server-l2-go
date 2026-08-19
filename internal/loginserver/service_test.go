package loginserver_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/pblaravel/game-server-l2-go/internal/config"
	"github.com/pblaravel/game-server-l2-go/internal/loginserver"
	"github.com/pblaravel/game-server-l2-go/internal/packet"
)

func startLoginWith(t *testing.T, mutate func(*config.LoginConfig)) (*loginserver.Server, func()) {
	t.Helper()
	cfg := config.DefaultLoginConfig()
	cfg.LoginServerPort = freePort(t)
	cfg.GameServerPort = freePort(t)
	if mutate != nil {
		mutate(&cfg)
	}
	srv, err := loginserver.NewServer(cfg, loginserver.NewMemoryAccountStore(), loginserver.NewMemoryGameServerStore())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		_ = srv.Run(ctx)
		close(done)
	}()
	time.Sleep(50 * time.Millisecond)
	return srv, func() {
		cancel()
		<-done
	}
}

// Java ClientPacketHandler drops a client that stops answering with Ping after
// server.connection.timeout.ms.
func TestIdleClientIsDisconnected(t *testing.T) {
	srv, stop := startLoginWith(t, func(c *config.LoginConfig) {
		c.ConnectionTimeoutMS = 300
	})
	defer stop()

	conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", itoa(srv.LS.Config().LoginServerPort)), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
	if _, err := packet.ReadFrame(conn); err != nil {
		t.Fatalf("init frame: %v", err)
	}
	// The server must close the socket on its own while we stay silent.
	if _, err := packet.ReadFrame(conn); err == nil {
		t.Fatal("expected the idle connection to be closed by the server")
	}
}

// Java logs a bad checksum and keeps the connection open.
func TestBadPacketKeepsConnectionOpen(t *testing.T) {
	srv, stop := startLoginWith(t, nil)
	defer stop()

	conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", itoa(srv.LS.Config().LoginServerPort)), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
	if _, err := packet.ReadFrame(conn); err != nil {
		t.Fatalf("init frame: %v", err)
	}
	if err := packet.WriteFrame(conn, make([]byte, 16)); err != nil {
		t.Fatalf("write garbage: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	clients := srv.LS.GetAllClients()
	if len(clients) != 1 {
		t.Fatalf("clients = %d, want the connection to survive a bad packet", len(clients))
	}
}

// Java ServerShutdownService disconnects live clients when the server stops.
func TestShutdownDisconnectsClients(t *testing.T) {
	srv, stop := startLoginWith(t, nil)

	conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", itoa(srv.LS.Config().LoginServerPort)), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
	if _, err := packet.ReadFrame(conn); err != nil {
		t.Fatalf("init frame: %v", err)
	}
	stop()
	if clients := srv.LS.GetAllClients(); len(clients) != 0 {
		t.Fatalf("clients after shutdown = %d, want 0", len(clients))
	}
	if _, err := packet.ReadFrame(conn); err == nil {
		t.Fatal("expected the client socket to be closed on shutdown")
	}
}
