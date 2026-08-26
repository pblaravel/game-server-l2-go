package apitest

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"math/big"
	"net"
	"time"

	"github.com/pblaravel/game-server-l2-go/internal/crypt"
	"github.com/pblaravel/game-server-l2-go/internal/loginserver"
	"github.com/pblaravel/game-server-l2-go/internal/packet"
)

// Target is a live login/game pair (Java or Go).
type Target struct {
	Login string
	GSReg string
	Game  string
}

func dial(addr string) (net.Conn, error) {
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		return nil, err
	}
	_ = conn.SetDeadline(time.Now().Add(8 * time.Second))
	return conn, nil
}

func readFrame(conn net.Conn) ([]byte, error) { return packet.ReadFrame(conn) }

func writeFrame(conn net.Conn, body []byte) error { return packet.WriteFrame(conn, body) }

// InitResult is Java InitPacket after static Blowfish + XOR undo.
type InitResult struct {
	Opcode      byte   `json:"opcode"`
	SessionID   int32  `json:"sessionId"`
	RSAModLen   int    `json:"rsaModLen"`
	BlowfishLen int    `json:"blowfishKeyLen"`
	BlowfishKey []byte `json:"-"`
	RSAMod      []byte `json:"-"`
}

func LoginInit(addr string) (*InitResult, error) {
	conn, err := dial(addr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	body, err := readFrame(conn)
	if err != nil {
		return nil, err
	}
	return parseInit(body)
}

func parseInit(body []byte) (*InitResult, error) {
	if len(body) < 16 || len(body)%8 != 0 {
		return nil, fmt.Errorf("init frame size %d", len(body))
	}
	dup := append([]byte(nil), body...)
	crypt.New(crypt.StaticBlowfishKey).Decrypt(dup, 0, len(dup))
	crypt.DecXORPass(dup)
	if dup[0] != loginserver.ServerInit {
		return nil, fmt.Errorf("expected Init 0x00, got 0x%02x", dup[0])
	}
	r := packet.NewReader(dup)
	r.SkipOpcode()
	sid := r.ReadD()
	modLen := int(r.ReadD())
	mod := r.ReadB(modLen)
	keyLen := int(r.ReadD())
	key := r.ReadB(keyLen)
	return &InitResult{
		Opcode: loginserver.ServerInit, SessionID: sid,
		RSAModLen: modLen, BlowfishLen: keyLen,
		BlowfishKey: key, RSAMod: mod,
	}, nil
}

type loginSession struct {
	conn net.Conn
	bf   *crypt.NewCrypt
	init *InitResult
}

func openLogin(addr string) (*loginSession, error) {
	conn, err := dial(addr)
	if err != nil {
		return nil, err
	}
	body, err := readFrame(conn)
	if err != nil {
		conn.Close()
		return nil, err
	}
	init, err := parseInit(body)
	if err != nil {
		conn.Close()
		return nil, err
	}
	if len(init.BlowfishKey) == 0 {
		conn.Close()
		return nil, fmt.Errorf("empty blowfish key")
	}
	return &loginSession{conn: conn, bf: crypt.New(init.BlowfishKey), init: init}, nil
}

func (s *loginSession) send(payload []byte) error {
	dup := append([]byte(nil), payload...)
	crypt.AppendChecksum(dup)
	s.bf.Crypt(dup, 0, len(dup))
	return writeFrame(s.conn, dup)
}

func (s *loginSession) recv() ([]byte, error) {
	body, err := readFrame(s.conn)
	if err != nil {
		return nil, err
	}
	s.bf.Decrypt(body, 0, len(body))
	if !crypt.VerifyChecksum(body) {
		return nil, fmt.Errorf("login checksum")
	}
	return body, nil
}

func (s *loginSession) Close() { _ = s.conn.Close() }

type PingResult struct {
	Opcode byte `json:"opcode"`
}

func LoginPing(addr string) (*PingResult, error) {
	s, err := openLogin(addr)
	if err != nil {
		return nil, err
	}
	defer s.Close()
	w := packet.NewWriter()
	w.WriteC(int(loginserver.ClientPing))
	w.PadTo8()
	if err := s.send(w.Bytes()); err != nil {
		return nil, err
	}
	body, err := s.recv()
	if err != nil {
		return nil, err
	}
	return &PingResult{Opcode: body[0]}, nil
}

type AuthResult struct {
	Opcode     byte  `json:"opcode"`
	LoginOk1   int32 `json:"loginOk1,omitempty"`
	LoginOk2   int32 `json:"loginOk2,omitempty"`
	FailReason byte  `json:"failReason,omitempty"`
}

func buildAuthPlain(account string, hash []byte) []byte {
	plain := make([]byte, 0, 1+len(account)+1+len(hash))
	plain = append(plain, byte(len(account)))
	plain = append(plain, account...)
	plain = append(plain, 0)
	plain = append(plain, hash...)
	return plain
}

func LoginAuth(addr, account string, passHash []byte) (*AuthResult, error) {
	s, err := openLogin(addr)
	if err != nil {
		return nil, err
	}
	defer s.Close()
	mod := crypt.UnscrambleModulus(s.init.RSAMod)
	pub := &rsa.PublicKey{N: new(big.Int).SetBytes(mod), E: 65537}
	plain := buildAuthPlain(account, passHash)
	enc, err := rsa.EncryptPKCS1v15(rand.Reader, pub, plain)
	if err != nil {
		return nil, err
	}
	w := packet.NewWriter()
	w.WriteC(int(loginserver.ClientAuthRequest))
	w.WriteB(enc)
	w.PadTo8()
	if err := s.send(w.Bytes()); err != nil {
		return nil, err
	}
	body, err := s.recv()
	if err != nil {
		return nil, err
	}
	out := &AuthResult{Opcode: body[0]}
	r := packet.NewReader(body)
	r.SkipOpcode()
	switch body[0] {
	case loginserver.ServerLoginOk:
		out.LoginOk1, out.LoginOk2 = r.ReadD(), r.ReadD()
	case loginserver.ServerLoginFail, loginserver.ServerAccountKicked:
		out.FailReason = byte(r.ReadC())
	}
	return out, nil
}

type ServerListResult struct {
	Opcode     byte             `json:"opcode"`
	Count      int              `json:"count"`
	LastServer int              `json:"lastServer"`
	Servers    []map[string]any `json:"servers"`
	CharSlots  int              `json:"charSlots"`
}

func LoginServers(addr, account string, passHash []byte) (*ServerListResult, error) {
	s, err := openLogin(addr)
	if err != nil {
		return nil, err
	}
	defer s.Close()
	auth, err := loginAuthOn(s, account, passHash)
	if err != nil {
		return nil, err
	}
	if auth.Opcode != loginserver.ServerLoginOk {
		return nil, fmt.Errorf("auth opcode 0x%02x", auth.Opcode)
	}
	w := packet.NewWriter()
	w.WriteC(int(loginserver.ClientRequestServerList))
	w.WriteD(auth.LoginOk1)
	w.WriteD(auth.LoginOk2)
	w.PadTo8()
	if err := s.send(w.Bytes()); err != nil {
		return nil, err
	}
	body, err := s.recv()
	if err != nil {
		return nil, err
	}
	return parseServerList(body)
}

func parseServerList(body []byte) (*ServerListResult, error) {
	if body[0] != loginserver.ServerServerList {
		return &ServerListResult{Opcode: body[0]}, nil
	}
	r := packet.NewReader(body)
	r.SkipOpcode()
	n := r.ReadC()
	last := r.ReadC()
	out := &ServerListResult{Opcode: body[0], Count: int(n), LastServer: int(last)}
	for i := 0; i < int(n); i++ {
		id := r.ReadC()
		ip0, ip1, ip2, ip3 := r.ReadC(), r.ReadC(), r.ReadC(), r.ReadC()
		port := r.ReadD()
		cur, maxp := r.ReadD(), r.ReadD()
		st := r.ReadC()
		out.Servers = append(out.Servers, map[string]any{
			"id": int(id), "ip": fmt.Sprintf("%d.%d.%d.%d", ip0, ip1, ip2, ip3),
			"port": port, "current": cur, "max": maxp, "status": int(st),
		})
	}
	if r.Remaining() > 0 {
		out.CharSlots = int(r.ReadC())
	}
	return out, nil
}

type PlayResult struct {
	Opcode     byte  `json:"opcode"`
	PlayOk1    int32 `json:"playOk1,omitempty"`
	PlayOk2    int32 `json:"playOk2,omitempty"`
	FailReason byte  `json:"failReason,omitempty"`
}

func LoginPlay(addr, account string, passHash []byte, serverID int) (*PlayResult, error) {
	s, err := openLogin(addr)
	if err != nil {
		return nil, err
	}
	defer s.Close()
	auth, err := loginAuthOn(s, account, passHash)
	if err != nil {
		return nil, err
	}
	if auth.Opcode != loginserver.ServerLoginOk {
		return nil, fmt.Errorf("auth opcode 0x%02x", auth.Opcode)
	}
	w := packet.NewWriter()
	w.WriteC(int(loginserver.ClientRequestServerLogin))
	w.WriteD(auth.LoginOk1)
	w.WriteD(auth.LoginOk2)
	w.WriteD(int32(serverID))
	w.PadTo8()
	if err := s.send(w.Bytes()); err != nil {
		return nil, err
	}
	body, err := s.recv()
	if err != nil {
		return nil, err
	}
	out := &PlayResult{Opcode: body[0]}
	r := packet.NewReader(body)
	r.SkipOpcode()
	switch body[0] {
	case loginserver.ServerPlayOk:
		out.PlayOk1, out.PlayOk2 = r.ReadD(), r.ReadD()
	case loginserver.ServerPlayFail:
		out.FailReason = byte(r.ReadC())
	}
	return out, nil
}

func loginAuthOn(s *loginSession, account string, passHash []byte) (*AuthResult, error) {
	mod := crypt.UnscrambleModulus(s.init.RSAMod)
	pub := &rsa.PublicKey{N: new(big.Int).SetBytes(mod), E: 65537}
	plain := buildAuthPlain(account, passHash)
	enc, err := rsa.EncryptPKCS1v15(rand.Reader, pub, plain)
	if err != nil {
		return nil, err
	}
	w := packet.NewWriter()
	w.WriteC(int(loginserver.ClientAuthRequest))
	w.WriteB(enc)
	w.PadTo8()
	if err := s.send(w.Bytes()); err != nil {
		return nil, err
	}
	body, err := s.recv()
	if err != nil {
		return nil, err
	}
	out := &AuthResult{Opcode: body[0]}
	r := packet.NewReader(body)
	r.SkipOpcode()
	if body[0] == loginserver.ServerLoginOk {
		out.LoginOk1, out.LoginOk2 = r.ReadD(), r.ReadD()
	}
	return out, nil
}

type InitLSResult struct {
	Opcode   byte  `json:"opcode"`
	Revision int32 `json:"revision"`
	RSALen   int   `json:"rsaModLen"`
}

func GSRegInit(addr string) (*InitLSResult, error) {
	conn, err := dial(addr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	body, err := readFrame(conn)
	if err != nil {
		return nil, err
	}
	bf := crypt.New(crypt.DefaultGSBlowfishKey)
	bf.Decrypt(body, 0, len(body))
	if !crypt.VerifyChecksum(body) {
		return nil, fmt.Errorf("InitLS checksum")
	}
	if body[0] != loginserver.LSInitLS {
		return nil, fmt.Errorf("expected InitLS 0x00, got 0x%02x", body[0])
	}
	r := packet.NewReader(body)
	r.SkipOpcode()
	rev := r.ReadD()
	n := int(r.ReadD())
	return &InitLSResult{Opcode: body[0], Revision: rev, RSALen: n}, nil
}

type GSRegResult struct {
	Opcode   byte   `json:"opcode"`
	ServerID int    `json:"serverId,omitempty"`
	Name     string `json:"name,omitempty"`
}

// GSHold keeps the login↔game TCP session open so the gameserver stays Authed
// (Java SetDown() runs as soon as this socket closes).
type GSHold struct {
	conn   net.Conn
	bf     *crypt.NewCrypt
	Result *GSRegResult
}

func (h *GSHold) Close() {
	if h != nil && h.conn != nil {
		_ = h.conn.Close()
		h.conn = nil
	}
}

func GSRegRegister(addr string, id int, hex []byte) (*GSRegResult, error) {
	h, err := OpenGSReg(addr, id, hex)
	if err != nil {
		return nil, err
	}
	h.Close()
	return h.Result, nil
}

func OpenGSReg(addr string, id int, hex []byte) (*GSHold, error) {
	conn, err := dial(addr)
	if err != nil {
		return nil, err
	}
	_ = conn.SetDeadline(time.Time{})
	bf := crypt.New(crypt.DefaultGSBlowfishKey)
	body, err := readFrame(conn)
	if err != nil {
		conn.Close()
		return nil, err
	}
	bf.Decrypt(body, 0, len(body))
	if !crypt.VerifyChecksum(body) {
		conn.Close()
		return nil, fmt.Errorf("InitLS checksum")
	}
	r := packet.NewReader(body)
	r.SkipOpcode()
	_ = r.ReadD()
	n := int(r.ReadD())
	mod := r.ReadB(n)
	pub := &rsa.PublicKey{N: new(big.Int).SetBytes(mod), E: 65537}
	key := bytes.Repeat([]byte{0x55}, 16)
	enc, err := crypt.EncryptNoPadding(pub, key)
	if err != nil {
		conn.Close()
		return nil, err
	}
	if err := sendGS(conn, bf, blowfishKeyPkt(enc)); err != nil {
		conn.Close()
		return nil, err
	}
	bf = crypt.New(key)
	if err := sendGS(conn, bf, authReqPkt(id, hex)); err != nil {
		conn.Close()
		return nil, err
	}
	resp, err := readFrame(conn)
	if err != nil {
		conn.Close()
		return nil, err
	}
	bf.Decrypt(resp, 0, len(resp))
	if !crypt.VerifyChecksum(resp) {
		conn.Close()
		return nil, fmt.Errorf("authresp checksum")
	}
	out := &GSRegResult{Opcode: resp[0]}
	if resp[0] == loginserver.LSAuthResponse {
		rr := packet.NewReader(resp)
		rr.SkipOpcode()
		out.ServerID = int(rr.ReadC())
		out.Name = rr.ReadS()
	} else {
		conn.Close()
		return &GSHold{Result: out}, fmt.Errorf("gs auth opcode 0x%02x", out.Opcode)
	}
	// Java GameServer sends ServerStatus after AuthResponse so the list is not STATUS_DOWN.
	if err := sendGS(conn, bf, serverStatusPkt(loginserver.StatusGood, 100)); err != nil {
		conn.Close()
		return nil, err
	}
	// Login applies ServerStatus asynchronously on the GS thread.
	time.Sleep(30 * time.Millisecond)
	return &GSHold{conn: conn, bf: bf, Result: out}, nil
}

func serverStatusPkt(status, maxPlayers int) []byte {
	w := packet.NewWriter()
	w.WriteC(int(loginserver.GSServerStatus))
	w.WriteD(2)
	w.WriteD(loginserver.AttrServerListStatus)
	w.WriteD(int32(status))
	w.WriteD(loginserver.AttrMaxPlayers)
	w.WriteD(int32(maxPlayers))
	w.PadTo8()
	return w.Bytes()
}

func sendGS(conn net.Conn, bf *crypt.NewCrypt, payload []byte) error {
	dup := append([]byte(nil), payload...)
	crypt.AppendChecksum(dup)
	bf.Crypt(dup, 0, len(dup))
	return writeFrame(conn, dup)
}

func blowfishKeyPkt(enc []byte) []byte {
	w := packet.NewWriter()
	w.WriteC(int(loginserver.GSBlowFishKey))
	w.WriteD(int32(len(enc)))
	w.WriteB(enc)
	w.PadTo8()
	return w.Bytes()
}

func authReqPkt(id int, hex []byte) []byte {
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

type VersionCheckResult struct {
	Opcode       byte   `json:"opcode"`
	OK           byte   `json:"ok"`
	KeyLen       int    `json:"keyLen"`
	BlowfishFlag int32  `json:"blowfishFlag"`
	Trailer      int32  `json:"trailer"`
	Key8         []byte `json:"-"`
}

func GameProtocol(addr string, version int32) (*VersionCheckResult, error) {
	conn, err := dial(addr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	w := packet.NewWriter()
	w.WriteC(0x00)
	w.WriteD(version)
	if err := writeFrame(conn, w.Bytes()); err != nil {
		return nil, err
	}
	body, err := readFrame(conn)
	if err != nil {
		return nil, err
	}
	if len(body) == 0 || body[0] != 0x00 {
		op := byte(0xff)
		if len(body) > 0 {
			op = body[0]
		}
		return nil, fmt.Errorf("expected VersionCheck 0x00, got 0x%02x len=%d", op, len(body))
	}
	r := packet.NewReader(body)
	r.SkipOpcode()
	ok := byte(r.ReadC())
	key := r.ReadB(8)
	flag, trailer := int32(0), int32(0)
	if r.Remaining() >= 4 {
		flag = r.ReadD()
	}
	if r.Remaining() >= 4 {
		trailer = r.ReadD()
	}
	return &VersionCheckResult{
		Opcode: 0x00, OK: ok, KeyLen: len(key),
		BlowfishFlag: flag, Trailer: trailer, Key8: key,
	}, nil
}
