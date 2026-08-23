package apitest

import (
	"encoding/json"
	"fmt"
)

// Snapshot is a structural dump of every login/game request the Java server accepts.
type Snapshot struct {
	Backend   string              `json:"backend"`
	Init      *InitResult         `json:"loginInit"`
	Ping      *PingResult         `json:"loginPing"`
	Auth      *AuthResult         `json:"loginAuth"`
	AuthFail  *AuthResult         `json:"loginAuthWrongPassword"`
	GSReg     *GSRegResult        `json:"gsRegister"`
	Servers   *ServerListResult   `json:"loginServers"`
	Play      *PlayResult         `json:"loginPlay"`
	PlayDown  *PlayResult         `json:"loginPlayMissingServer"`
	InitLS    *InitLSResult       `json:"gsregInit"`
	Protocol  *VersionCheckResult `json:"gameProtocol740"`
	Errors    map[string]string   `json:"errors,omitempty"`
	hold      *GSHold
}

func (s *Snapshot) Close() {
	if s != nil && s.hold != nil {
		s.hold.Close()
		s.hold = nil
	}
}

func Capture(backend string, t Target, account string, hash []byte) *Snapshot {
	s := &Snapshot{Backend: backend, Errors: map[string]string{}}
	var err error
	if s.Init, err = LoginInit(t.Login); err != nil {
		s.Errors["loginInit"] = err.Error()
	}
	if s.Ping, err = LoginPing(t.Login); err != nil {
		s.Errors["loginPing"] = err.Error()
	}
	if s.Auth, err = LoginAuth(t.Login, account, hash); err != nil {
		s.Errors["loginAuth"] = err.Error()
	}
	if s.AuthFail, err = LoginAuth(t.Login, account, []byte{9, 9, 9, 9}); err != nil {
		s.Errors["loginAuthWrongPassword"] = err.Error()
	}
	if s.InitLS, err = GSRegInit(t.GSReg); err != nil {
		s.Errors["gsregInit"] = err.Error()
	}
	hex := []byte{0x81, 0xa8, 0xba, 0x90, 0xdb, 0x0e, 0x77, 0xd3, 0x03, 0x39, 0x73, 0x88, 0xe2, 0x5e, 0xce, 0xfa}
	hold, err := OpenGSReg(t.GSReg, 1, hex)
	if err != nil {
		s.Errors["gsRegister"] = err.Error()
	} else {
		s.hold = hold
		s.GSReg = hold.Result
	}
	if s.Servers, err = LoginServers(t.Login, account, hash); err != nil {
		s.Errors["loginServers"] = err.Error()
	}
	if s.Play, err = LoginPlay(t.Login, account, hash, 1); err != nil {
		s.Errors["loginPlay"] = err.Error()
	}
	if s.PlayDown, err = LoginPlay(t.Login, account, hash, 99); err != nil {
		s.Errors["loginPlayMissingServer"] = err.Error()
	}
	if t.Game != "" {
		if s.Protocol, err = GameProtocol(t.Game, 740); err != nil {
			s.Errors["gameProtocol740"] = err.Error()
		}
	}
	return s
}

// Diff reports opcode/layout mismatches. Session keys and RSA/blowfish secrets are ignored.
func Diff(java, goSnap *Snapshot) []string {
	var out []string
	check := func(name string, ok bool, detail string) {
		if !ok {
			out = append(out, name+": "+detail)
		}
	}
	if java.Init != nil && goSnap.Init != nil {
		check("loginInit.opcode", java.Init.Opcode == goSnap.Init.Opcode, fmt.Sprintf("java 0x%02x go 0x%02x", java.Init.Opcode, goSnap.Init.Opcode))
		check("loginInit.rsaModLen", java.Init.RSAModLen == goSnap.Init.RSAModLen, fmt.Sprintf("java %d go %d", java.Init.RSAModLen, goSnap.Init.RSAModLen))
		check("loginInit.blowfishKeyLen", java.Init.BlowfishLen == goSnap.Init.BlowfishLen, fmt.Sprintf("java %d go %d", java.Init.BlowfishLen, goSnap.Init.BlowfishLen))
	} else {
		check("loginInit", java.Init != nil && goSnap.Init != nil, "missing on one side")
	}
	if java.Ping != nil && goSnap.Ping != nil {
		check("loginPing.opcode", java.Ping.Opcode == goSnap.Ping.Opcode, fmt.Sprintf("java 0x%02x go 0x%02x", java.Ping.Opcode, goSnap.Ping.Opcode))
	}
	if java.Auth != nil && goSnap.Auth != nil {
		check("loginAuth.opcode", java.Auth.Opcode == goSnap.Auth.Opcode, fmt.Sprintf("java 0x%02x go 0x%02x", java.Auth.Opcode, goSnap.Auth.Opcode))
	}
	if java.AuthFail != nil && goSnap.AuthFail != nil {
		check("loginAuthWrongPassword.opcode", java.AuthFail.Opcode == goSnap.AuthFail.Opcode, fmt.Sprintf("java 0x%02x go 0x%02x", java.AuthFail.Opcode, goSnap.AuthFail.Opcode))
		check("loginAuthWrongPassword.failReason", java.AuthFail.FailReason == goSnap.AuthFail.FailReason, fmt.Sprintf("java 0x%02x go 0x%02x", java.AuthFail.FailReason, goSnap.AuthFail.FailReason))
	}
	if java.InitLS != nil && goSnap.InitLS != nil {
		check("gsregInit.opcode", java.InitLS.Opcode == goSnap.InitLS.Opcode, fmt.Sprintf("java 0x%02x go 0x%02x", java.InitLS.Opcode, goSnap.InitLS.Opcode))
		check("gsregInit.revision", java.InitLS.Revision == goSnap.InitLS.Revision, fmt.Sprintf("java 0x%x go 0x%x", java.InitLS.Revision, goSnap.InitLS.Revision))
		check("gsregInit.rsaModLen", java.InitLS.RSALen == goSnap.InitLS.RSALen, fmt.Sprintf("java %d go %d", java.InitLS.RSALen, goSnap.InitLS.RSALen))
	}
	if java.GSReg != nil && goSnap.GSReg != nil {
		check("gsRegister.opcode", java.GSReg.Opcode == goSnap.GSReg.Opcode, fmt.Sprintf("java 0x%02x go 0x%02x", java.GSReg.Opcode, goSnap.GSReg.Opcode))
		check("gsRegister.serverId", java.GSReg.ServerID == goSnap.GSReg.ServerID, fmt.Sprintf("java %d go %d", java.GSReg.ServerID, goSnap.GSReg.ServerID))
		check("gsRegister.name", java.GSReg.Name == goSnap.GSReg.Name, fmt.Sprintf("java %q go %q", java.GSReg.Name, goSnap.GSReg.Name))
	}
	if java.Servers != nil && goSnap.Servers != nil {
		check("loginServers.opcode", java.Servers.Opcode == goSnap.Servers.Opcode, fmt.Sprintf("java 0x%02x go 0x%02x", java.Servers.Opcode, goSnap.Servers.Opcode))
		check("loginServers.count", java.Servers.Count == goSnap.Servers.Count, fmt.Sprintf("java %d go %d", java.Servers.Count, goSnap.Servers.Count))
		if java.Servers.Count > 0 && goSnap.Servers.Count > 0 {
			jp, _ := java.Servers.Servers[0]["port"].(int32)
			gp, _ := goSnap.Servers.Servers[0]["port"].(int32)
			if jp == 0 {
				if v, ok := java.Servers.Servers[0]["port"].(int); ok {
					jp = int32(v)
				}
			}
			if gp == 0 {
				if v, ok := goSnap.Servers.Servers[0]["port"].(int); ok {
					gp = int32(v)
				}
			}
			check("loginServers[0].port", jp == gp, fmt.Sprintf("java %v go %v", java.Servers.Servers[0]["port"], goSnap.Servers.Servers[0]["port"]))
		}
	}
	if java.Play != nil && goSnap.Play != nil {
		check("loginPlay.opcode", java.Play.Opcode == goSnap.Play.Opcode, fmt.Sprintf("java 0x%02x go 0x%02x", java.Play.Opcode, goSnap.Play.Opcode))
	}
	if java.PlayDown != nil && goSnap.PlayDown != nil {
		check("loginPlayMissingServer.opcode", java.PlayDown.Opcode == goSnap.PlayDown.Opcode, fmt.Sprintf("java 0x%02x go 0x%02x", java.PlayDown.Opcode, goSnap.PlayDown.Opcode))
		check("loginPlayMissingServer.failReason", java.PlayDown.FailReason == goSnap.PlayDown.FailReason, fmt.Sprintf("java 0x%02x go 0x%02x", java.PlayDown.FailReason, goSnap.PlayDown.FailReason))
	}
	if java.Protocol != nil && goSnap.Protocol != nil {
		check("gameProtocol.opcode", java.Protocol.Opcode == goSnap.Protocol.Opcode, fmt.Sprintf("java 0x%02x go 0x%02x", java.Protocol.Opcode, goSnap.Protocol.Opcode))
		check("gameProtocol.ok", java.Protocol.OK == goSnap.Protocol.OK, fmt.Sprintf("java %d go %d", java.Protocol.OK, goSnap.Protocol.OK))
		check("gameProtocol.keyLen", java.Protocol.KeyLen == goSnap.Protocol.KeyLen, fmt.Sprintf("java %d go %d", java.Protocol.KeyLen, goSnap.Protocol.KeyLen))
		check("gameProtocol.blowfishFlag", java.Protocol.BlowfishFlag == goSnap.Protocol.BlowfishFlag, fmt.Sprintf("java %d go %d", java.Protocol.BlowfishFlag, goSnap.Protocol.BlowfishFlag))
		check("gameProtocol.trailer", java.Protocol.Trailer == goSnap.Protocol.Trailer, fmt.Sprintf("java %d go %d", java.Protocol.Trailer, goSnap.Protocol.Trailer))
	}
	return out
}

func (s *Snapshot) JSON() []byte {
	b, _ := json.MarshalIndent(s, "", "  ")
	return b
}

// JavaContract is the opcode/layout contract taken from the Java sources
// (reference/l2-unity-loginserver and reference/l2-unity-gameserver).
type JavaContract struct {
	InitOpcode       byte
	InitRSAModLen    int
	InitBlowfishLen  int
	PingOpcode       byte
	LoginOkOpcode    byte
	LoginFailOpcode  byte
	UserOrPassWrong  byte
	ServerListOpcode byte
	PlayOkOpcode     byte
	PlayFailOpcode   byte
	ServerOverloaded byte
	InitLSOpcode     byte
	InitLSRevision   int32
	InitLSRSALen     int
	AuthRespOpcode   byte
	AuthRespName     string
	VersionOpcode    byte
	VersionOK        byte
	VersionKeyLen    int
	VersionTrailer   int32
}

func ExpectedJavaContract() JavaContract {
	return JavaContract{
		InitOpcode: 0x00, InitRSAModLen: 128, InitBlowfishLen: 16,
		PingOpcode: 0x63,
		LoginOkOpcode: 0x03, LoginFailOpcode: 0x01, UserOrPassWrong: 0x02,
		ServerListOpcode: 0x04,
		PlayOkOpcode: 0x07, PlayFailOpcode: 0x06, ServerOverloaded: 0x0F,
		InitLSOpcode: 0x00, InitLSRevision: 0x0102, InitLSRSALen: 64,
		AuthRespOpcode: 0x02, AuthRespName: "Bartz",
		VersionOpcode: 0x00, VersionOK: 0x01, VersionKeyLen: 8, VersionTrailer: 1,
	}
}

func (s *Snapshot) MatchContract(c JavaContract) []string {
	var out []string
	fail := func(name string, ok bool, detail string) {
		if !ok {
			out = append(out, name+": "+detail)
		}
	}
	if s.Init != nil {
		fail("init.opcode", s.Init.Opcode == c.InitOpcode, fmt.Sprintf("got 0x%02x", s.Init.Opcode))
		fail("init.rsaModLen", s.Init.RSAModLen == c.InitRSAModLen, fmt.Sprintf("got %d", s.Init.RSAModLen))
		fail("init.blowfishKeyLen", s.Init.BlowfishLen == c.InitBlowfishLen, fmt.Sprintf("got %d", s.Init.BlowfishLen))
	} else {
		fail("init", false, "missing")
	}
	if s.Ping != nil {
		fail("ping.opcode", s.Ping.Opcode == c.PingOpcode, fmt.Sprintf("got 0x%02x", s.Ping.Opcode))
	} else {
		fail("ping", false, "missing")
	}
	if s.Auth != nil {
		fail("auth.opcode", s.Auth.Opcode == c.LoginOkOpcode, fmt.Sprintf("got 0x%02x", s.Auth.Opcode))
		fail("auth.loginOkPair", s.Auth.LoginOk1 != 0 || s.Auth.LoginOk2 != 0, "empty session pair")
	} else {
		fail("auth", false, "missing")
	}
	if s.AuthFail != nil {
		fail("authFail.opcode", s.AuthFail.Opcode == c.LoginFailOpcode, fmt.Sprintf("got 0x%02x", s.AuthFail.Opcode))
		fail("authFail.reason", s.AuthFail.FailReason == c.UserOrPassWrong, fmt.Sprintf("got 0x%02x", s.AuthFail.FailReason))
	} else {
		fail("authFail", false, "missing")
	}
	if s.InitLS != nil {
		fail("initLS.opcode", s.InitLS.Opcode == c.InitLSOpcode, fmt.Sprintf("got 0x%02x", s.InitLS.Opcode))
		fail("initLS.revision", s.InitLS.Revision == c.InitLSRevision, fmt.Sprintf("got 0x%x", s.InitLS.Revision))
		fail("initLS.rsaModLen", s.InitLS.RSALen == c.InitLSRSALen, fmt.Sprintf("got %d", s.InitLS.RSALen))
	} else {
		fail("initLS", false, "missing")
	}
	if s.GSReg != nil {
		fail("gsReg.opcode", s.GSReg.Opcode == c.AuthRespOpcode, fmt.Sprintf("got 0x%02x", s.GSReg.Opcode))
		fail("gsReg.name", s.GSReg.Name == c.AuthRespName, fmt.Sprintf("got %q", s.GSReg.Name))
	} else {
		fail("gsReg", false, "missing")
	}
	if s.Servers != nil {
		fail("servers.opcode", s.Servers.Opcode == c.ServerListOpcode, fmt.Sprintf("got 0x%02x", s.Servers.Opcode))
		fail("servers.count", s.Servers.Count >= 1, fmt.Sprintf("got %d", s.Servers.Count))
	} else {
		fail("servers", false, "missing")
	}
	if s.Play != nil {
		fail("play.opcode", s.Play.Opcode == c.PlayOkOpcode, fmt.Sprintf("got 0x%02x", s.Play.Opcode))
	} else {
		fail("play", false, "missing")
	}
	if s.PlayDown != nil {
		fail("playDown.opcode", s.PlayDown.Opcode == c.PlayFailOpcode, fmt.Sprintf("got 0x%02x", s.PlayDown.Opcode))
		fail("playDown.reason", s.PlayDown.FailReason == c.ServerOverloaded, fmt.Sprintf("got 0x%02x", s.PlayDown.FailReason))
	} else {
		fail("playDown", false, "missing")
	}
	if s.Protocol != nil {
		fail("protocol.opcode", s.Protocol.Opcode == c.VersionOpcode, fmt.Sprintf("got 0x%02x", s.Protocol.Opcode))
		fail("protocol.ok", s.Protocol.OK == c.VersionOK, fmt.Sprintf("got %d", s.Protocol.OK))
		fail("protocol.keyLen", s.Protocol.KeyLen == c.VersionKeyLen, fmt.Sprintf("got %d", s.Protocol.KeyLen))
		fail("protocol.trailer", s.Protocol.Trailer == c.VersionTrailer, fmt.Sprintf("got %d", s.Protocol.Trailer))
	}
	return out
}
