package gameserver

import (
	"net"
	"testing"
	"time"

	"github.com/pblaravel/game-server-l2-go/internal/config"
	"github.com/pblaravel/game-server-l2-go/internal/packet"
)

func startHandshakeServer(t *testing.T, cfg config.GameConfig) (addr string, stop func()) {
	t.Helper()
	cfg.LoginHost = "127.0.0.1"
	cfg.LoginPort = 1
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()
	cfg.GameserverPort = port

	srv := NewServer(cfg, NewMemoryCharacterStore())
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
			if tc, ok := conn.(*net.TCPConn); ok {
				_ = tc.SetNoDelay(true)
			}
			c := NewGameClient(conn, srv)
			go c.Serve()
		}
	}()
	time.Sleep(20 * time.Millisecond)
	return net.JoinHostPort("127.0.0.1", itoa(port)), func() { close(ctxDone) }
}

func sendProtocol(t *testing.T, addr string, version int32) []byte {
	t.Helper()
	conn, err := net.DialTimeout("tcp", addr, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
	w := packet.NewWriter()
	w.WriteC(0x00)
	w.WriteD(version)
	if err := packet.WriteFrame(conn, w.Bytes()); err != nil {
		t.Fatal(err)
	}
	body, err := packet.ReadFrame(conn)
	if err != nil {
		t.Fatalf("no VersionCheck: %v", err)
	}
	return body
}

func TestProtocolVersion740SendsVersionCheck(t *testing.T) {
	addr, stop := startHandshakeServer(t, config.DefaultGameConfig())
	defer stop()
	body := sendProtocol(t, addr, 740)
	if body[0] != 0x00 || body[1] != 0x01 {
		t.Fatalf("VersionCheck %x", body)
	}
}

func TestUnknownProtocolStillGetsVersionCheck(t *testing.T) {
	addr, stop := startHandshakeServer(t, config.DefaultGameConfig())
	defer stop()
	body := sendProtocol(t, addr, 999)
	if len(body) < 2 || body[0] != 0x00 || body[1] != 0x01 {
		t.Fatalf("Unity clients must receive VersionCheck, got %x", body)
	}
}

func TestStrictProtocolRejectsUnknown(t *testing.T) {
	cfg := config.DefaultGameConfig()
	cfg.StrictProtocol = true
	addr, stop := startHandshakeServer(t, cfg)
	defer stop()
	body := sendProtocol(t, addr, 999)
	if len(body) < 2 || body[0] != 0x00 || body[1] != 0x00 {
		t.Fatalf("strict reject should send VersionCheck fail, got %x", body)
	}
}

func TestParseProtocolVersion(t *testing.T) {
	w := packet.NewWriter()
	w.WriteC(0x00)
	w.WriteD(740)
	if parseProtocolVersion(w.Bytes()) != 740 {
		t.Fatal("dword")
	}
	w = packet.NewWriter()
	w.WriteC(0x00)
	w.WriteH(740)
	if parseProtocolVersion(w.Bytes()) != 740 {
		t.Fatal("short")
	}
}

func TestVersionCheckReplyFailLayout(t *testing.T) {
	p := VersionCheckReply([]byte{1, 2, 3, 4, 5, 6, 7, 8}, true, false)
	r := packet.NewReader(p)
	if r.ReadC() != 0x00 || r.ReadC() != 0x00 {
		t.Fatal(p)
	}
}
