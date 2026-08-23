package loginserver_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"math/big"
	"net"
	"testing"
	"time"

	"github.com/pblaravel/game-server-l2-go/internal/config"
	"github.com/pblaravel/game-server-l2-go/internal/crypt"
	"github.com/pblaravel/game-server-l2-go/internal/loginserver"
	"github.com/pblaravel/game-server-l2-go/internal/packet"
)

func TestInterludeInitBlowfishSurvivesXOR(t *testing.T) {
	mod := bytes.Repeat([]byte{0x11}, 128)
	bf := bytes.Repeat([]byte{0x22}, 16)
	raw := loginserver.InterludeInitPacket(mod, bf, 42)
	enc := append([]byte(nil), raw...)
	lc := crypt.NewLoginCrypt(bf)
	if err := lc.EncryptWithXORKey(enc, 0x12345678); err != nil {
		t.Fatal(err)
	}
	_, _, _, got := decryptInterludeInit(t, enc)
	if !bytes.Equal(got, bf) {
		t.Fatalf("blowfish after Init XOR:\n got %x\nwant %x", got, bf)
	}
}

func TestInterludeInitPacketLayout(t *testing.T) {
	mod := bytes.Repeat([]byte{0x11}, 128)
	bf := bytes.Repeat([]byte{0x22}, 16)
	p := loginserver.InterludeInitPacket(mod, bf, 42)
	if p[0] != loginserver.ServerInit {
		t.Fatalf("opcode 0x%02x", p[0])
	}
	r := packet.NewReader(p)
	r.SkipOpcode()
	if r.ReadD() != 42 {
		t.Fatal("session")
	}
	if r.ReadD() != loginserver.InterludeLoginProtocol {
		t.Fatalf("protocol want 0x%08x", loginserver.InterludeLoginProtocol)
	}
	if !bytes.Equal(r.ReadB(128), mod) {
		t.Fatal("rsa modulus")
	}
	_ = r.ReadD()
	_ = r.ReadD()
	_ = r.ReadD()
	_ = r.ReadD()
	if !bytes.Equal(r.ReadB(16), bf) {
		t.Fatal("blowfish")
	}
}

func TestGGAuthPacket(t *testing.T) {
	p := loginserver.GGAuthPacket(7)
	if p[0] != loginserver.ServerGGAuth {
		t.Fatalf("opcode 0x%02x", p[0])
	}
	r := packet.NewReader(p)
	r.SkipOpcode()
	if r.ReadD() != 7 {
		t.Fatal("response")
	}
}

func TestParseInterludeAuthCredentials(t *testing.T) {
	plain := make([]byte, 113)
	copy(plain[79:], []byte("HeroAcc"))
	copy(plain[93:], []byte("s3cret"))
	c, err := loginserver.ParseAuthCredentials(plain)
	if err != nil {
		t.Fatal(err)
	}
	if c.Account != "heroacc" {
		t.Fatalf("account %q", c.Account)
	}
	if !bytes.Equal(c.PassHashBytes, []byte("s3cret")) {
		t.Fatalf("password %q", c.PassHashBytes)
	}
}

func startInterludeLogin(t *testing.T) (*loginserver.Server, func()) {
	t.Helper()
	cfg := config.DefaultLoginConfig()
	cfg.LoginServerPort = freePort(t)
	cfg.GameServerPort = freePort(t)
	cfg.ShowLicense = true
	cfg.AutoCreateAccount = true
	cfg.InterludeClient = true
	cfg.ConnectionTimeoutMS = 30000
	srv, err := loginserver.NewServer(cfg, loginserver.NewMemoryAccountStore(), loginserver.NewMemoryGameServerStore())
	if err != nil {
		t.Fatal(err)
	}
	gsi := loginserver.NewGameServerInfo(1, []byte{0xAB, 0xCD}, nil)
	gsi.SetAuthed(true)
	gsi.SetMaxPlayers(100)
	gsi.SetStatus(loginserver.StatusGood)
	gsi.SetHostname("127.0.0.1")
	gsi.SetPort(7778)
	if !srv.GSC.Register(1, gsi) {
		t.Fatal("register gameserver")
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

func TestInterludeLoginSequence(t *testing.T) {
	srv, stop := startInterludeLogin(t)
	defer stop()

	conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", itoa(srv.LS.Config().LoginServerPort)), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))

	initBody, err := packet.ReadFrame(conn)
	if err != nil {
		t.Fatal(err)
	}
	sessionID, protocol, pub, bfKey := decryptInterludeInit(t, initBody)
	if protocol != loginserver.InterludeLoginProtocol {
		t.Fatalf("Init.Protocol 0x%08x", protocol)
	}
	if len(bfKey) != 16 {
		t.Fatalf("blowfish %d", len(bfKey))
	}

	// AuthGameGuard 0x07 → GGAuth 0x0B
	w := packet.NewWriter()
	w.WriteC(int(loginserver.ClientAuthGameGuard))
	w.WriteD(sessionID)
	w.WriteD(0)
	w.WriteD(0)
	w.WriteD(0)
	w.WriteD(0)
	w.PadTo8()
	if err := writeLogin(conn, bfKey, w.Bytes()); err != nil {
		t.Fatal(err)
	}
	gg := readLogin(t, conn, bfKey)
	if gg[0] != loginserver.ServerGGAuth {
		t.Fatalf("GGAuth opcode 0x%02x", gg[0])
	}

	// RequestAuthLogin 0x00: RSA block at offsets 79 / 93
	plain := make([]byte, 113)
	copy(plain[79:], []byte("unity"))
	copy(plain[93:], []byte("pass"))
	enc, err := rsa.EncryptPKCS1v15(rand.Reader, pub, plain)
	if err != nil {
		t.Fatal(err)
	}
	if len(enc) != 128 {
		t.Fatalf("rsa cipher %d", len(enc))
	}
	w = packet.NewWriter()
	w.WriteC(int(loginserver.ClientPing)) // Interlude RequestAuthLogin shares 0x00
	w.WriteB(enc)
	w.WriteD(sessionID)
	w.PadTo8()
	if err := writeLogin(conn, bfKey, w.Bytes()); err != nil {
		t.Fatal(err)
	}
	ok := readLogin(t, conn, bfKey)
	if ok[0] != loginserver.ServerLoginOk {
		t.Fatalf("LoginOk opcode 0x%02x body=%x", ok[0], ok[:min(16, len(ok))])
	}
	r := packet.NewReader(ok)
	r.SkipOpcode()
	key1, key2 := r.ReadD(), r.ReadD()

	// RequestServerList 0x05
	w = packet.NewWriter()
	w.WriteC(int(loginserver.ClientRequestServerListInterlude))
	w.WriteD(key1)
	w.WriteD(key2)
	w.PadTo8()
	if err := writeLogin(conn, bfKey, w.Bytes()); err != nil {
		t.Fatal(err)
	}
	list := readLogin(t, conn, bfKey)
	if list[0] != loginserver.ServerServerList {
		t.Fatalf("ServerList opcode 0x%02x", list[0])
	}

	// RequestServerLogin 0x02: serverId, sessionKey1, sessionKey2
	w = packet.NewWriter()
	w.WriteC(int(loginserver.ClientRequestServerList)) // Interlude play is 0x02
	w.WriteD(1)
	w.WriteD(key1)
	w.WriteD(key2)
	w.PadTo8()
	if err := writeLogin(conn, bfKey, w.Bytes()); err != nil {
		t.Fatal(err)
	}
	play := readLogin(t, conn, bfKey)
	if play[0] != loginserver.ServerPlayOk {
		t.Fatalf("PlayOk opcode 0x%02x", play[0])
	}
}

func decryptInterludeInit(t *testing.T, body []byte) (sessionID, protocol int32, pub *rsa.PublicKey, bf []byte) {
	t.Helper()
	static := crypt.New(crypt.StaticBlowfishKey)
	static.Decrypt(body, 0, len(body))
	crypt.DecXORPass(body)
	if body[0] != loginserver.ServerInit {
		t.Fatalf("Init opcode 0x%02x", body[0])
	}
	r := packet.NewReader(body)
	r.SkipOpcode()
	sessionID = r.ReadD()
	protocol = r.ReadD()
	mod := r.ReadB(128)
	_ = r.ReadD()
	_ = r.ReadD()
	_ = r.ReadD()
	_ = r.ReadD()
	bf = r.ReadB(16)
	pub = &rsa.PublicKey{N: new(big.Int).SetBytes(mod), E: 65537}
	return
}

func writeLogin(conn net.Conn, key, payload []byte) error {
	dup := bytes.Clone(payload)
	crypt.AppendChecksum(dup)
	bf := crypt.New(key)
	bf.Crypt(dup, 0, len(dup))
	return packet.WriteFrame(conn, dup)
}

func readLogin(t *testing.T, conn net.Conn, key []byte) []byte {
	t.Helper()
	body, err := packet.ReadFrame(conn)
	if err != nil {
		t.Fatal(err)
	}
	bf := crypt.New(key)
	bf.Decrypt(body, 0, len(body))
	if !crypt.VerifyChecksum(body) {
		t.Fatal("checksum")
	}
	return body
}
