package loginserver_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"io"
	"net"
	"testing"
	"time"

	"math/big"

	"github.com/pblaravel/game-server-l2-go/internal/config"
	"github.com/pblaravel/game-server-l2-go/internal/crypt"
	"github.com/pblaravel/game-server-l2-go/internal/loginserver"
	"github.com/pblaravel/game-server-l2-go/internal/packet"
	"github.com/pblaravel/game-server-l2-go/internal/session"
)

func TestSessionKeyLicenseModes(t *testing.T) {
	a := session.New(1, 2, 3, 4)
	b := session.New(1, 2, 3, 4)
	c := session.New(9, 9, 3, 4)
	if !a.Equals(b, true) {
		t.Fatal("full key should match")
	}
	if a.Equals(c, true) {
		t.Fatal("login pair differs")
	}
	if !a.Equals(c, false) {
		t.Fatal("play pair only when license hidden")
	}
}

func TestAuthCredentialsParse(t *testing.T) {
	// [len][account][pad][hash...]
	plain := []byte{5, 'a', 'd', 'm', 'i', 'n', 0x00, 1, 2, 3, 4}
	c, err := loginserver.ParseAuthCredentials(plain)
	if err != nil {
		t.Fatal(err)
	}
	if c.Account != "admin" {
		t.Fatalf("account %q", c.Account)
	}
	if !bytes.Equal(c.PassHashBytes, []byte{1, 2, 3, 4}) {
		t.Fatalf("hash %v", c.PassHashBytes)
	}
}

func TestServerNameAPI(t *testing.T) {
	if loginserver.ServerName(1) != "Bartz" {
		t.Fatal(loginserver.ServerName(1))
	}
	if loginserver.ServerName(999) != "Undefined" {
		t.Fatal("fallback")
	}
}

func TestHexRoundTrip(t *testing.T) {
	in := []byte{0xde, 0xad, 0xbe, 0xef}
	s := loginserver.HexToString(in)
	out := loginserver.StringToHex(s)
	if !bytes.Equal(in, out) {
		t.Fatalf("%s %v", s, out)
	}
}

func TestPacketOpcodes(t *testing.T) {
	if loginserver.InitPacket(make([]byte, 128), make([]byte, 16), 1)[0] != loginserver.ServerInit {
		t.Fatal("init opcode")
	}
	if loginserver.LoginOkPacket(session.New(1, 2, 3, 4))[0] != loginserver.ServerLoginOk {
		t.Fatal("loginok")
	}
	if loginserver.PlayOkPacket(session.New(1, 2, 3, 4))[0] != loginserver.ServerPlayOk {
		t.Fatal("playok")
	}
	if loginserver.PingPacket()[0] != loginserver.ServerPing {
		t.Fatal("ping")
	}
	if loginserver.AuthResponsePacket(1)[0] != loginserver.LSAuthResponse {
		t.Fatal("authresp")
	}
}

func startLogin(t *testing.T) (*loginserver.Server, func()) {
	t.Helper()
	cfg := config.DefaultLoginConfig()
	cfg.LoginServerPort = freePort(t)
	cfg.GameServerPort = freePort(t)
	cfg.ShowLicense = true
	cfg.AutoCreateAccount = true
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

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	p := l.Addr().(*net.TCPAddr).Port
	_ = l.Close()
	return p
}

func TestLoginClientReceivesInit(t *testing.T) {
	srv, stop := startLogin(t)
	defer stop()

	conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", itoa(srv.LS.Config().LoginServerPort)), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
	body, err := packet.ReadFrame(conn)
	if err != nil {
		t.Fatal(err)
	}
	if len(body) < 8 || len(body)%8 != 0 {
		t.Fatalf("init frame size %d", len(body))
	}
	// Init is encrypted with static blowfish + xor; decrypt to check opcode.
	static := crypt.New(crypt.StaticBlowfishKey)
	static.Decrypt(body, 0, len(body))
	if body[0] != loginserver.ServerInit {
		t.Fatalf("expected init opcode after static decrypt, got 0x%02x", body[0])
	}
}

func TestGameServerHandshake(t *testing.T) {
	srv, stop := startLogin(t)
	defer stop()

	conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", itoa(srv.LS.Config().GameServerPort)), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))

	bf := crypt.New(crypt.DefaultGSBlowfishKey)
	body, err := packet.ReadFrame(conn)
	if err != nil {
		t.Fatal(err)
	}
	bf.Decrypt(body, 0, len(body))
	if !crypt.VerifyChecksum(body) {
		t.Fatal("initls checksum")
	}
	if body[0] != loginserver.LSInitLS {
		t.Fatalf("opcode %d", body[0])
	}
	r := packet.NewReader(body)
	r.SkipOpcode()
	rev := r.ReadD()
	if rev != 0x0102 {
		t.Fatalf("revision %d", rev)
	}
	n := int(r.ReadD())
	mod := r.ReadB(n)
	pub := &rsa.PublicKey{N: new(big.Int).SetBytes(mod), E: 65537}

	key := bytes.Repeat([]byte{0x55}, 16)
	enc, err := crypt.EncryptNoPadding(pub, key)
	if err != nil {
		t.Fatal(err)
	}
	if err := sendGS(conn, bf, blowfishKey(enc)); err != nil {
		t.Fatal(err)
	}
	bf = crypt.New(key)

	hex := bytes.Repeat([]byte{0xAB}, 16)
	if err := sendGS(conn, bf, authReq(1, hex)); err != nil {
		t.Fatal(err)
	}
	resp, err := packet.ReadFrame(conn)
	if err != nil {
		t.Fatal(err)
	}
	bf.Decrypt(resp, 0, len(resp))
	if !crypt.VerifyChecksum(resp) {
		t.Fatal("authresp checksum")
	}
	if resp[0] != loginserver.LSAuthResponse {
		t.Fatalf("got opcode 0x%02x", resp[0])
	}
	if srv.GSC.GetRegisteredGameServerById(1) == nil || !srv.GSC.GetRegisteredGameServerById(1).IsAuthed() {
		t.Fatal("gameserver should be registered and authed")
	}
}

func sendGS(conn net.Conn, bf *crypt.NewCrypt, payload []byte) error {
	dup := bytes.Clone(payload)
	crypt.AppendChecksum(dup)
	bf.Crypt(dup, 0, len(dup))
	return packet.WriteFrame(conn, dup)
}

func blowfishKey(enc []byte) []byte {
	w := packet.NewWriter()
	w.WriteC(int(loginserver.GSBlowFishKey))
	w.WriteD(int32(len(enc)))
	w.WriteB(enc)
	w.PadTo8()
	return w.Bytes()
}

func authReq(id int, hex []byte) []byte {
	w := packet.NewWriter()
	w.WriteC(int(loginserver.GSAuthRequest))
	w.WriteC(id)
	w.WriteC(1)
	w.WriteC(0)
	w.WriteS("*")
	w.WriteH(7778)
	w.WriteD(100)
	w.WriteD(int32(len(hex)))
	w.WriteB(hex)
	w.PadTo8()
	return w.Bytes()
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [16]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

// compile-time sanity
var _ = io.EOF
var _ = rand.Reader
