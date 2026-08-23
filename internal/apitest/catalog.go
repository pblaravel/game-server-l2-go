package apitest

// Endpoint documents one Java login/game request and the packet it returns.
// The HTTP path is a test facade: Java itself is binary TCP, not HTTP.
type Endpoint struct {
	Method   string `json:"method"`
	Path     string `json:"path"`
	Channel  string `json:"channel"`
	Request  string `json:"request"`
	Response string `json:"response"`
}

// Catalog is the Java wire contract used by the REST tests.
func Catalog() []Endpoint {
	return []Endpoint{
		{
			Method: "GET", Path: "/api/login/init", Channel: "login-client :2107",
			Request:  "TCP connect (no payload)",
			Response: "S→C Init 0x00: sessionId i32, rsaModLen i32, scrambled RSA-1024 (128), blowfishLen i32, blowfishKey (16). Static Blowfish + EncXORPass.",
		},
		{
			Method: "POST", Path: "/api/login/ping", Channel: "login-client :2107",
			Request:  "C→S Ping 0x00",
			Response: "S→C Ping 0x63",
		},
		{
			Method: "POST", Path: "/api/login/auth", Channel: "login-client :2107",
			Request:  "C→S AuthRequest 0x01 + 128-byte RSA PKCS1 [len][account][0][hash]. Server stores Base64(hash).",
			Response: "S→C LoginOk 0x03 (loginOk1, loginOk2) when showLicense=true; ServerList 0x04 when false; LoginFail 0x01 reason; AccountKicked 0x02 reason.",
		},
		{
			Method: "POST", Path: "/api/login/servers", Channel: "login-client :2107",
			Request:  "C→S RequestServerList 0x02 + loginOk1 i32 + loginOk2 i32",
			Response: "S→C ServerList 0x04: count u8, lastServer u8, {id u8, ip 4×u8, port i32, current i32, max i32, status u8}×count, charSlots u8; or LoginFail 0x01.",
		},
		{
			Method: "POST", Path: "/api/login/play", Channel: "login-client :2107",
			Request:  "C→S RequestServerLogin 0x03 + loginOk1 i32 + loginOk2 i32 + serverId i32",
			Response: "S→C PlayOk 0x07 (playOk1, playOk2) or PlayFail 0x06 reason 0x0F if server down/full.",
		},
		{
			Method: "GET", Path: "/api/gsreg/init", Channel: "login↔game :9015",
			Request:  "TCP connect (no payload)",
			Response: "LS→GS InitLS 0x00: revision 0x0102, rsaModLen i32, RSA-512 modulus. Default GS Blowfish + checksum.",
		},
		{
			Method: "POST", Path: "/api/gsreg/register", Channel: "login↔game :9015",
			Request:  "GS→LS BlowFishKey 0x00 then AuthRequest 0x01 (id, hexid, host, port, max)",
			Response: "LS→GS AuthResponse 0x02 (serverId, name) or Fail 0x01.",
		},
		{
			Method: "POST", Path: "/api/game/protocol", Channel: "game-client :7778",
			Request:  "C→S ProtocolVersion 0x00 + version i32 (737/740/744/746)",
			Response: "S→C VersionCheck 0x00: ok u8 (1), key[8], blowfishFlag i32, trailer i32=1. Java closes unknown versions; Go accepts unless StrictProtocol.",
		},
	}
}
