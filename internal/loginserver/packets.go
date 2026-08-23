package loginserver

import (
	"encoding/base64"
	"net"

	"github.com/pblaravel/game-server-l2-go/internal/crypt"
	"github.com/pblaravel/game-server-l2-go/internal/packet"
	"github.com/pblaravel/game-server-l2-go/internal/session"
)

func buildLS(opcode byte, write func(w *packet.Writer)) []byte {
	w := packet.NewWriter()
	w.WriteC(int(opcode))
	write(w)
	w.PadTo8()
	return w.Bytes()
}

func InitPacket(scrambledMod, blowfishKey []byte, sessionID int32) []byte {
	return buildLS(ServerInit, func(w *packet.Writer) {
		w.WriteD(sessionID)
		w.WriteD(int32(len(scrambledMod)))
		w.WriteB(scrambledMod)
		w.WriteD(int32(len(blowfishKey)))
		w.WriteB(blowfishKey)
	})
}

// InterludeLoginProtocol is the Init.Protocol value the Unity client expects.
const InterludeLoginProtocol int32 = 0x0000c621

// InterludeInitPacket is the classic Interlude Init: sessionId, protocol,
// 128-byte RSA modulus (no length prefix), GG challenge ×4, blowfish, 0x00.
func InterludeInitPacket(publicKey, blowfishKey []byte, sessionID int32) []byte {
	mod := publicKey
	if len(mod) == 0x81 && mod[0] == 0x00 {
		mod = mod[1:]
	}
	if len(mod) > 128 {
		mod = mod[len(mod)-128:]
	}
	if len(mod) < 128 {
		padded := make([]byte, 128)
		copy(padded[128-len(mod):], mod)
		mod = padded
	}
	bf := blowfishKey
	if len(bf) > 16 {
		bf = bf[:16]
	}
	return buildLS(ServerInit, func(w *packet.Writer) {
		w.WriteD(sessionID)
		w.WriteD(InterludeLoginProtocol)
		w.WriteB(mod)
		w.WriteD(0)
		w.WriteD(0)
		w.WriteD(0)
		w.WriteD(0)
		w.WriteB(bf)
		w.WriteC(0)
	})
}

func GGAuthPacket(response int32) []byte {
	return buildLS(ServerGGAuth, func(w *packet.Writer) { w.WriteD(response) })
}

func LoginFailPacket(reason byte) []byte {
	return buildLS(ServerLoginFail, func(w *packet.Writer) { w.WriteC(int(reason)) })
}

func AccountKickedPacket(reason byte) []byte {
	return buildLS(ServerAccountKicked, func(w *packet.Writer) { w.WriteC(int(reason)) })
}

func LoginOkPacket(key session.Key) []byte {
	return buildLS(ServerLoginOk, func(w *packet.Writer) {
		w.WriteD(key.LoginOkID1)
		w.WriteD(key.LoginOkID2)
	})
}

func PlayOkPacket(key session.Key) []byte {
	return buildLS(ServerPlayOk, func(w *packet.Writer) {
		w.WriteD(key.PlayOkID1)
		w.WriteD(key.PlayOkID2)
	})
}

func PlayFailPacket(reason byte) []byte {
	return buildLS(ServerPlayFail, func(w *packet.Writer) { w.WriteC(int(reason)) })
}

func PingPacket() []byte {
	return buildLS(ServerPing, func(w *packet.Writer) {})
}

type ServerListEntry struct {
	ID             int
	IP             [4]byte
	Port           int32
	CurrentPlayers int32
	MaxPlayers     int32
	Status         byte
}

func ServerListPacket(lastServer byte, servers []ServerListEntry, charsOnServers map[int]int) []byte {
	return buildLS(ServerServerList, func(w *packet.Writer) {
		w.WriteC(len(servers))
		w.WriteC(int(lastServer))
		for _, s := range servers {
			w.WriteC(s.ID)
			w.WriteC(int(s.IP[0]))
			w.WriteC(int(s.IP[1]))
			w.WriteC(int(s.IP[2]))
			w.WriteC(int(s.IP[3]))
			w.WriteD(s.Port)
			w.WriteD(s.CurrentPlayers)
			w.WriteD(s.MaxPlayers)
			w.WriteC(int(s.Status))
		}
		if len(charsOnServers) == 0 {
			w.WriteC(0)
			return
		}
		w.WriteC(len(charsOnServers))
		for id, n := range charsOnServers {
			w.WriteC(id)
			w.WriteC(n)
		}
	})
}

func InitLSPacket(revision int32, pubModulus []byte) []byte {
	return buildLS(LSInitLS, func(w *packet.Writer) {
		w.WriteD(revision)
		w.WriteD(int32(len(pubModulus)))
		w.WriteB(pubModulus)
	})
}

func LoginServerFailPacket(reason int32) []byte {
	return buildLS(LSFail, func(w *packet.Writer) {
		w.WriteC(0)
		w.WriteC(0)
		w.WriteD(reason)
	})
}

func AuthResponsePacket(id int) []byte {
	return buildLS(LSAuthResponse, func(w *packet.Writer) {
		w.WriteC(id)
		w.WriteS(ServerName(id))
	})
}

func PlayerAuthResponsePacket(account string, authed bool) []byte {
	return buildLS(LSPlayerAuthResponse, func(w *packet.Writer) {
		w.WriteS(account)
		if authed {
			w.WriteC(1)
		} else {
			w.WriteC(0)
		}
	})
}

func KickPlayerPacket(account string) []byte {
	return buildLS(LSKickPlayer, func(w *packet.Writer) {
		w.WriteS(account)
	})
}

func RequestCharactersPacket(account string) []byte {
	return buildLS(LSRequestCharacters, func(w *packet.Writer) {
		w.WriteS(account)
	})
}

type AuthCredentials struct {
	Account       string
	PassHashBytes []byte
	HashBase64    string
}

func ParseAuthCredentials(decrypted []byte) (AuthCredentials, error) {
	if looksLikeJavaAuth(decrypted) {
		return parseJavaAuth(decrypted)
	}
	if c, err := parseInterludeAuth(decrypted); err == nil && c.Account != "" {
		return c, nil
	}
	return parseJavaAuth(decrypted)
}

func looksLikeJavaAuth(b []byte) bool {
	if len(b) < 3 {
		return false
	}
	n := int(b[0])
	if n < 1 || n > 14 || 1+n >= len(b) {
		return false
	}
	if b[1+n] != 0 {
		return false
	}
	for _, c := range b[1 : 1+n] {
		if c < 33 || c > 126 {
			return false
		}
	}
	return true
}

func parseJavaAuth(decrypted []byte) (AuthCredentials, error) {
	if len(decrypted) < 3 {
		return AuthCredentials{}, errShort
	}
	n := int(decrypted[0])
	if n < 0 || 1+n >= len(decrypted) {
		return AuthCredentials{}, errShort
	}
	account := string(decrypted[1 : 1+n])
	pass := append([]byte(nil), decrypted[n+2:]...)
	return AuthCredentials{
		Account:       toLowerTrim(account),
		PassHashBytes: pass,
		HashBase64:    base64.StdEncoding.EncodeToString(pass),
	}, nil
}

// parseInterludeAuth reads account/password at offsets 79 and 93 of the RSA
// plaintext (Unity client: ≤15 UTF-8 bytes each).
func parseInterludeAuth(decrypted []byte) (AuthCredentials, error) {
	if len(decrypted) < 108 {
		return AuthCredentials{}, errShort
	}
	acc := cstring(decrypted[79:94])
	pass := decrypted[93:]
	if len(pass) > 15 {
		pass = pass[:15]
	}
	for i, b := range pass {
		if b == 0 {
			pass = pass[:i]
			break
		}
	}
	acc = toLowerTrim(acc)
	if acc == "" {
		return AuthCredentials{}, errShort
	}
	return AuthCredentials{
		Account:       acc,
		PassHashBytes: append([]byte(nil), pass...),
		HashBase64:    base64.StdEncoding.EncodeToString(pass),
	}, nil
}

func cstring(b []byte) string {
	for i, v := range b {
		if v == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}

func DecryptAuthRequest(data []byte, decrypt func([]byte) ([]byte, error)) (AuthCredentials, error) {
	if len(data) < 1+128 {
		return AuthCredentials{}, errShort
	}
	plain, err := decrypt(data[1 : 1+128])
	if err != nil {
		return AuthCredentials{}, err
	}
	return ParseAuthCredentials(plain)
}

var errShort = errString("short packet")

type errString string

func (e errString) Error() string { return string(e) }

func toLowerTrim(s string) string {
	b := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			continue
		}
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b = append(b, c)
	}
	return string(b)
}

func ResolveIPv4(host string) [4]byte {
	ip := [4]byte{127, 0, 0, 1}
	if host == "" || host == "*" {
		return ip
	}
	parsed := net.ParseIP(host)
	if parsed == nil {
		addrs, err := net.LookupIP(host)
		if err != nil || len(addrs) == 0 {
			return ip
		}
		parsed = addrs[0]
	}
	v4 := parsed.To4()
	if v4 == nil {
		return ip
	}
	copy(ip[:], v4)
	return ip
}

func HexToString(b []byte) string {
	const hexdigits = "0123456789abcdef"
	out := make([]byte, len(b)*2)
	for i, v := range b {
		out[i*2] = hexdigits[v>>4]
		out[i*2+1] = hexdigits[v&0x0f]
	}
	return string(out)
}

func StringToHex(s string) []byte {
	if len(s)%2 != 0 {
		return nil
	}
	out := make([]byte, len(s)/2)
	for i := 0; i < len(s); i += 2 {
		out[i/2] = hexNibble(s[i])<<4 | hexNibble(s[i+1])
	}
	return out
}

func hexNibble(c byte) byte {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10
	}
	return 0
}

// ParseBlowFishKey decrypts the RSA-wrapped session key (NoPadding) and strips leading zeros.
func ParseBlowFishKey(data []byte, decrypt func([]byte) ([]byte, error)) ([]byte, error) {
	r := packet.NewReader(data)
	r.SkipOpcode()
	n := int(r.ReadD())
	enc := r.ReadB(n)
	plain, err := decrypt(enc)
	if err != nil {
		return nil, err
	}
	return crypt.StripLeadingZeros(plain), nil
}

type GameServerAuth struct {
	ID              byte
	AcceptAlternate bool
	ReserveHost     bool
	Host            string
	Port            int
	MaxPlayer       int32
	HexID           []byte
}

func ParseGameServerAuth(data []byte) GameServerAuth {
	r := packet.NewReader(data)
	r.SkipOpcode()
	a := GameServerAuth{}
	a.ID = r.ReadC()
	a.AcceptAlternate = r.ReadC() == 1
	a.ReserveHost = r.ReadC() == 1
	a.Host = r.ReadS()
	a.Port = r.ReadH()
	a.MaxPlayer = r.ReadD()
	n := int(r.ReadD())
	a.HexID = r.ReadB(n)
	return a
}

func ParsePlayerAuthRequest(data []byte) (account string, key session.Key) {
	r := packet.NewReader(data)
	r.SkipOpcode()
	account = r.ReadS()
	key.PlayOkID1 = r.ReadD()
	key.PlayOkID2 = r.ReadD()
	key.LoginOkID1 = r.ReadD()
	key.LoginOkID2 = r.ReadD()
	return
}

func ParsePlayerInGame(data []byte) []string {
	r := packet.NewReader(data)
	r.SkipOpcode()
	n := r.ReadH()
	out := make([]string, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, r.ReadS())
	}
	return out
}

func ParsePlayerLogout(data []byte) string {
	r := packet.NewReader(data)
	r.SkipOpcode()
	return r.ReadS()
}

func ParseReplyCharacters(data []byte) (account string, count int) {
	r := packet.NewReader(data)
	r.SkipOpcode()
	account = r.ReadS()
	count = int(r.ReadC())
	return
}

type StatusAttr struct {
	ID    int32
	Value int32
}

func ParseServerStatus(data []byte) []StatusAttr {
	r := packet.NewReader(data)
	r.SkipOpcode()
	n := int(r.ReadD())
	out := make([]StatusAttr, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, StatusAttr{ID: r.ReadD(), Value: r.ReadD()})
	}
	return out
}

func ParseRequestServerList(data []byte) (skey1, skey2 int32) {
	r := packet.NewReader(data)
	r.SkipOpcode()
	return r.ReadD(), r.ReadD()
}

func ParseRequestServerLogin(data []byte) (skey1, skey2, serverID int32) {
	r := packet.NewReader(data)
	r.SkipOpcode()
	return r.ReadD(), r.ReadD(), r.ReadD()
}
