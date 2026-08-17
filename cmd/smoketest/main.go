package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/pblaravel/game-server-l2-go/internal/crypt"
	"github.com/pblaravel/game-server-l2-go/internal/loginserver"
	"github.com/pblaravel/game-server-l2-go/internal/packet"
)

func main() {
	login := flag.String("login", "127.0.0.1:2107", "login client TCP addr")
	gsreg := flag.String("gsreg", "127.0.0.1:9015", "login↔game registration TCP addr")
	game := flag.String("game", "127.0.0.1:7778", "game client TCP addr")
	wait := flag.Duration("wait", 60*time.Second, "how long to retry until all APIs respond")
	flag.Parse()

	deadline := time.Now().Add(*wait)
	checks := []struct {
		name string
		fn   func() error
	}{
		{"login Init", func() error { return checkLoginInit(*login) }},
		{"login InitLS", func() error { return checkInitLS(*gsreg) }},
		{"game VersionCheck", func() error { return checkGameVersion(*game) }},
	}
	for _, c := range checks {
		if err := retry(deadline, c.name, c.fn); err != nil {
			fmt.Fprintf(os.Stderr, "FAIL %s: %v\n", c.name, err)
			os.Exit(1)
		}
		fmt.Printf("OK   %s\n", c.name)
	}
	fmt.Println("all L2 APIs responded")
}

func retry(deadline time.Time, name string, fn func() error) error {
	var last error
	for time.Now().Before(deadline) {
		last = fn()
		if last == nil {
			return nil
		}
		fmt.Fprintf(os.Stderr, "wait %s: %v\n", name, last)
		time.Sleep(time.Second)
	}
	return last
}

func dial(addr string) (net.Conn, error) {
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		return nil, err
	}
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	return conn, nil
}

func checkLoginInit(addr string) error {
	conn, err := dial(addr)
	if err != nil {
		return err
	}
	defer conn.Close()
	body, err := packet.ReadFrame(conn)
	if err != nil {
		return err
	}
	if len(body) < 8 || len(body)%8 != 0 {
		return fmt.Errorf("init frame size %d", len(body))
	}
	static := crypt.New(crypt.StaticBlowfishKey)
	static.Decrypt(body, 0, len(body))
	if body[0] != loginserver.ServerInit {
		return fmt.Errorf("expected Init opcode 0x00, got 0x%02x", body[0])
	}
	return nil
}

func checkInitLS(addr string) error {
	conn, err := dial(addr)
	if err != nil {
		return err
	}
	defer conn.Close()
	body, err := packet.ReadFrame(conn)
	if err != nil {
		return err
	}
	bf := crypt.New(crypt.DefaultGSBlowfishKey)
	bf.Decrypt(body, 0, len(body))
	if !crypt.VerifyChecksum(body) {
		return fmt.Errorf("InitLS checksum")
	}
	if body[0] != loginserver.LSInitLS {
		return fmt.Errorf("expected InitLS 0x00, got 0x%02x", body[0])
	}
	r := packet.NewReader(body)
	r.SkipOpcode()
	rev := r.ReadD()
	if rev != 0x0102 {
		return fmt.Errorf("revision 0x%x want 0x0102", rev)
	}
	return nil
}

func checkGameVersion(addr string) error {
	conn, err := dial(addr)
	if err != nil {
		return err
	}
	defer conn.Close()
	w := packet.NewWriter()
	w.WriteC(0x00)
	w.WriteD(737)
	if err := packet.WriteFrame(conn, w.Bytes()); err != nil {
		return err
	}
	body, err := packet.ReadFrame(conn)
	if err != nil {
		return err
	}
	if len(body) == 0 || body[0] != 0x00 {
		op := byte(0xff)
		if len(body) > 0 {
			op = body[0]
		}
		return fmt.Errorf("expected VersionCheck 0x00, got 0x%02x len=%d", op, len(body))
	}
	return nil
}
