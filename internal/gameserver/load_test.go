package gameserver

import (
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pblaravel/game-server-l2-go/internal/config"
	"github.com/pblaravel/game-server-l2-go/internal/packet"
)

func TestGameClientAcceptLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("load test")
	}
	cfg := config.DefaultGameConfig()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()
	cfg.GameserverPort = port
	cfg.LoginHost = "127.0.0.1"
	cfg.LoginPort = 1 // will fail reconnect loop, unused for this test
	cfg.UseBlowfishCipher = false

	srv := NewServer(cfg, NewMemoryCharacterStore())
	// don't start login thread reconnect spam: mark stopped
	srv.login.Stop()

	ctxDone := make(chan struct{})
	go func() {
		l, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", itoa(port)))
		if err != nil {
			t.Error(err)
			return
		}
		defer l.Close()
		go func() {
			<-ctxDone
			_ = l.Close()
		}()
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			c := NewGameClient(conn, srv)
			go c.Serve()
		}
	}()
	time.Sleep(30 * time.Millisecond)

	const n = 40
	var ok atomic.Int64
	var wg sync.WaitGroup
	wg.Add(n)
	start := time.Now()
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", itoa(port)), 2*time.Second)
			if err != nil {
				return
			}
			defer conn.Close()
			_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
			w := packet.NewWriter()
			w.WriteC(0x00)
			w.WriteD(737)
			if err := packet.WriteFrame(conn, w.Bytes()); err != nil {
				return
			}
			body, err := packet.ReadFrame(conn)
			if err == nil && len(body) > 0 && body[0] == 0x00 {
				ok.Add(1)
			}
		}()
	}
	wg.Wait()
	close(ctxDone)
	if ok.Load() < n {
		t.Fatalf("only %d/%d protocol handshakes succeeded", ok.Load(), n)
	}
	t.Logf("gameserver handshakes %d in %s", n, time.Since(start))
}

func BenchmarkUserInfo(b *testing.B) {
	ch := DefaultCharacter("a", "Hero", 0, 0, 0, 0, 0, 0, 1, nil)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = UserInfo(ch)
	}
}
