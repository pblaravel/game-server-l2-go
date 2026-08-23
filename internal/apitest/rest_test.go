package apitest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pblaravel/game-server-l2-go/internal/config"
	"github.com/pblaravel/game-server-l2-go/internal/gameserver"
	"github.com/pblaravel/game-server-l2-go/internal/loginserver"
	"github.com/pblaravel/game-server-l2-go/internal/packet"
)

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

func startGoLogin(t *testing.T) (Target, func()) {
	t.Helper()
	cfg := config.DefaultLoginConfig()
	cfg.LoginServerPort = freePort(t)
	cfg.GameServerPort = freePort(t)
	cfg.ShowLicense = true
	cfg.AutoCreateAccount = true
	cfg.AcceptNewGameServer = true
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
	waitListen(t, cfg.LoginServerPort)
	waitListen(t, cfg.GameServerPort)
	return Target{
			Login: net.JoinHostPort("127.0.0.1", itoa(cfg.LoginServerPort)),
			GSReg: net.JoinHostPort("127.0.0.1", itoa(cfg.GameServerPort)),
		}, func() {
			cancel()
			<-done
		}
}

func startGoGame(t *testing.T) (string, func()) {
	t.Helper()
	cfg := config.DefaultGameConfig()
	cfg.LoginHost = "127.0.0.1"
	cfg.LoginPort = 1
	port := freePort(t)
	cfg.GameserverPort = port
	srv := gameserver.NewServer(cfg, gameserver.NewMemoryCharacterStore())
	srv.Login().Stop()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		l, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", itoa(port)))
		if err != nil {
			t.Error(err)
			close(done)
			return
		}
		defer l.Close()
		go func() {
			<-ctx.Done()
			_ = l.Close()
		}()
		for {
			conn, err := l.Accept()
			if err != nil {
				close(done)
				return
			}
			c := gameserver.NewGameClient(conn, srv)
			go c.Serve()
		}
	}()
	waitListen(t, port)
	return net.JoinHostPort("127.0.0.1", itoa(port)), func() {
		cancel()
		<-done
	}
}

func waitListen(t *testing.T, port int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	addr := net.JoinHostPort("127.0.0.1", itoa(port))
	for time.Now().Before(deadline) {
		c, err := net.DialTimeout("tcp", addr, 50*time.Millisecond)
		if err == nil {
			_ = c.Close()
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("nothing listening on %s", addr)
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}

func TestRESTCatalogDocumentsJavaPackets(t *testing.T) {
	api := &REST{}
	ts := httptest.NewServer(api.Handler())
	defer ts.Close()
	res, err := http.Get(ts.URL + "/api/catalog")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var eps []Endpoint
	if err := json.NewDecoder(res.Body).Decode(&eps); err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{
		"/api/login/init": true, "/api/login/ping": true, "/api/login/auth": true,
		"/api/login/servers": true, "/api/login/play": true,
		"/api/gsreg/init": true, "/api/gsreg/register": true,
		"/api/game/protocol": true,
	}
	for _, e := range eps {
		delete(want, e.Path)
	}
	if len(want) > 0 {
		t.Fatalf("catalog missing %v", want)
	}
}

func TestRESTAgainstGoMatchesJavaContract(t *testing.T) {
	login, stopL := startGoLogin(t)
	defer stopL()
	game, stopG := startGoGame(t)
	defer stopG()
	login.Game = game

	api := &REST{Target: login}
	ts := httptest.NewServer(api.Handler())
	defer ts.Close()
	defer api.Close()

	get := func(path string) map[string]any {
		t.Helper()
		res, err := http.Get(ts.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode != 200 {
			b, _ := io.ReadAll(res.Body)
			t.Fatalf("%s status %d: %s", path, res.StatusCode, b)
		}
		var m map[string]any
		if err := json.NewDecoder(res.Body).Decode(&m); err != nil {
			t.Fatal(err)
		}
		return m
	}
	post := func(path string, body any) map[string]any {
		t.Helper()
		raw, _ := json.Marshal(body)
		res, err := http.Post(ts.URL+path, "application/json", bytes.NewReader(raw))
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode != 200 {
			b, _ := io.ReadAll(res.Body)
			t.Fatalf("%s status %d: %s", path, res.StatusCode, b)
		}
		var m map[string]any
		if err := json.NewDecoder(res.Body).Decode(&m); err != nil {
			t.Fatal(err)
		}
		return m
	}

	init := get("/api/login/init")
	if int(init["opcode"].(float64)) != 0x00 {
		t.Fatalf("init opcode %v", init["opcode"])
	}
	if int(init["rsaModLen"].(float64)) != 128 {
		t.Fatalf("rsa %v", init["rsaModLen"])
	}
	if int(init["blowfishKeyLen"].(float64)) != 16 {
		t.Fatalf("bf %v", init["blowfishKeyLen"])
	}

	ping := post("/api/login/ping", nil)
	if int(ping["opcode"].(float64)) != 0x63 {
		t.Fatalf("ping %v", ping)
	}

	auth := post("/api/login/auth", map[string]any{"account": "restuser", "passwordHash": "AQIDBA=="})
	if int(auth["opcode"].(float64)) != 0x03 {
		t.Fatalf("auth %v", auth)
	}

	fail := post("/api/login/auth", map[string]any{"account": "restuser", "passwordHash": "CQkJCQ=="})
	if int(fail["opcode"].(float64)) != 0x01 || int(fail["failReason"].(float64)) != 0x02 {
		t.Fatalf("auth fail %v", fail)
	}

	gs := post("/api/gsreg/register", map[string]any{"serverId": 1})
	if int(gs["opcode"].(float64)) != 0x02 || gs["name"] != "Bartz" {
		t.Fatalf("gsreg %v", gs)
	}

	servers := post("/api/login/servers", map[string]any{"account": "restuser", "passwordHash": "AQIDBA=="})
	if int(servers["opcode"].(float64)) != 0x04 || int(servers["count"].(float64)) < 1 {
		t.Fatalf("servers %v", servers)
	}
	list := servers["servers"].([]any)
	first := list[0].(map[string]any)
	if int(first["id"].(float64)) != 1 || int(first["port"].(float64)) != 7778 {
		t.Fatalf("server entry %v", first)
	}
	if int(first["status"].(float64)) != 1 {
		t.Fatalf("expected STATUS_NORMAL=1, got %v", first["status"])
	}

	play := post("/api/login/play", map[string]any{"account": "restuser", "passwordHash": "AQIDBA==", "serverId": 1})
	if int(play["opcode"].(float64)) != 0x07 {
		t.Fatalf("play %v", play)
	}

	down := post("/api/login/play", map[string]any{"account": "restuser", "passwordHash": "AQIDBA==", "serverId": 99})
	if int(down["opcode"].(float64)) != 0x06 || int(down["failReason"].(float64)) != 0x0F {
		t.Fatalf("play down %v", down)
	}

	ls := get("/api/gsreg/init")
	if int(ls["opcode"].(float64)) != 0x00 || int(ls["revision"].(float64)) != 0x0102 {
		t.Fatalf("initls %v", ls)
	}
	if n := int(ls["rsaModLen"].(float64)); n != 64 && n != 65 {
		t.Fatalf("initls rsa %v (Java BigInteger is 64 or 65)", ls)
	}

	ver := post("/api/game/protocol", map[string]any{"version": 740})
	if int(ver["opcode"].(float64)) != 0x00 || int(ver["ok"].(float64)) != 1 {
		t.Fatalf("protocol %v", ver)
	}
	if int(ver["keyLen"].(float64)) != 8 || int(ver["trailer"].(float64)) != 1 {
		t.Fatalf("protocol layout %v", ver)
	}
}

func TestGoSnapshotMatchesJavaSourceContract(t *testing.T) {
	login, stopL := startGoLogin(t)
	defer stopL()
	game, stopG := startGoGame(t)
	defer stopG()
	login.Game = game

	snap := Capture("go", login, "contractuser", []byte{1, 2, 3, 4})
	defer snap.Close()
	if diffs := snap.MatchContract(ExpectedJavaContract()); len(diffs) > 0 {
		t.Fatalf("go diverges from Java contract:\n%s\nsnap=%s", diffs, snap.JSON())
	}
	if len(snap.Errors) > 0 {
		t.Fatalf("capture errors: %v", snap.Errors)
	}
}

func TestDecXORPassUsedByInit(t *testing.T) {
	login, stop := startGoLogin(t)
	defer stop()
	init, err := LoginInit(login.Login)
	if err != nil {
		t.Fatal(err)
	}
	if init.SessionID == 0 {
		t.Fatal("sessionId should be readable after DecXORPass")
	}
	if len(init.RSAMod) != 128 || len(init.BlowfishKey) != 16 {
		t.Fatalf("mod=%d key=%d", len(init.RSAMod), len(init.BlowfishKey))
	}
}

func TestPlayPacketUsesInt32ServerID(t *testing.T) {
	// Java RequestServerLoginPacket reads serverId with readI() (int32).
	w := packet.NewWriter()
	w.WriteC(0x03)
	w.WriteD(1)
	w.WriteD(2)
	w.WriteD(7)
	s1, s2, id := loginserver.ParseRequestServerLogin(w.Bytes())
	if s1 != 1 || s2 != 2 || id != 7 {
		t.Fatalf("%d %d %d", s1, s2, id)
	}
}
