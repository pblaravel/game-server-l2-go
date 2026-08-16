package loginserver_test

import (
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pblaravel/game-server-l2-go/internal/packet"
)

func TestLoginAcceptLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("load test")
	}
	srv, stop := startLogin(t)
	defer stop()
	port := srv.LS.Config().LoginServerPort
	addr := net.JoinHostPort("127.0.0.1", itoa(port))

	const n = 50
	var ok atomic.Int64
	var wg sync.WaitGroup
	wg.Add(n)
	start := time.Now()
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
			if err != nil {
				return
			}
			defer conn.Close()
			_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
			body, err := packet.ReadFrame(conn)
			if err == nil && len(body) >= 8 {
				ok.Add(1)
			}
		}()
	}
	wg.Wait()
	elapsed := time.Since(start)
	if ok.Load() < n {
		t.Fatalf("only %d/%d clients received Init", ok.Load(), n)
	}
	t.Logf("accepted %d login inits in %s (%.0f conn/s)", n, elapsed, float64(n)/elapsed.Seconds())
}

func BenchmarkWriteFrame(b *testing.B) {
	payload := make([]byte, 64)
	var buf discard
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = packet.WriteFrame(buf, payload)
	}
}

type discard struct{}

func (discard) Write(p []byte) (int, error) { return len(p), nil }
