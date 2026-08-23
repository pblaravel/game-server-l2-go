package crypt

import "fmt"

// StaticBlowfishKey is the Init-packet-only key from Java LoginCrypt.
var StaticBlowfishKey = []byte{
	0x6b, 0x60, 0xcb, 0x5b, 0x82, 0xce, 0x90, 0xb1,
	0xcc, 0x2b, 0x6c, 0x55, 0x6c, 0x6c, 0x6c, 0x6c,
}

// DefaultGSBlowfishKey is the hardcoded GS↔LS key until BlowFishKey is exchanged.
// Java: new NewCrypt("_;v.]05-31!|+-%xT!^[$\00")
var DefaultGSBlowfishKey = append([]byte("_;v.]05-31!|+-%xT!^[$"), 0x00)

// NewCrypt is the Java NewCrypt wrapper around BlowfishEngine.
type NewCrypt struct {
	cipher *Engine
}

func New(key []byte) *NewCrypt {
	return &NewCrypt{cipher: NewEngine(key)}
}

func NewFromString(key string) *NewCrypt {
	return New([]byte(key))
}

func (c *NewCrypt) Decrypt(raw []byte, offset, size int) {
	for i := offset; i < offset+size; i += 8 {
		c.cipher.DecryptBlock(raw, i)
	}
}

func (c *NewCrypt) Crypt(raw []byte, offset, size int) {
	for i := offset; i < offset+size; i += 8 {
		c.cipher.EncryptBlock(raw, i)
	}
}

func VerifyChecksum(raw []byte) bool {
	return VerifyChecksumAt(raw, 0, len(raw))
}

func VerifyChecksumAt(raw []byte, offset, size int) bool {
	if size&3 != 0 || size <= 4 {
		return false
	}
	var chksum uint32
	count := size - 4
	i := offset
	for ; i < count; i += 4 {
		check := uint32(raw[i]) |
			uint32(raw[i+1])<<8 |
			uint32(raw[i+2])<<16 |
			uint32(raw[i+3])<<24
		chksum ^= check
	}
	check := uint32(raw[i]) |
		uint32(raw[i+1])<<8 |
		uint32(raw[i+2])<<16 |
		uint32(raw[i+3])<<24
	return check == chksum
}

func AppendChecksum(raw []byte) {
	AppendChecksumAt(raw, 0, len(raw))
}

func AppendChecksumAt(raw []byte, offset, size int) {
	var chksum uint32
	count := size - 4
	i := offset
	for ; i < count; i += 4 {
		ecx := uint32(raw[i]) |
			uint32(raw[i+1])<<8 |
			uint32(raw[i+2])<<16 |
			uint32(raw[i+3])<<24
		chksum ^= ecx
	}
	raw[i] = byte(chksum)
	raw[i+1] = byte(chksum >> 8)
	raw[i+2] = byte(chksum >> 16)
	raw[i+3] = byte(chksum >> 24)
}

func EncXORPass(raw []byte, key int32) {
	EncXORPassAt(raw, 0, len(raw), key)
}

func EncXORPassAt(raw []byte, offset, size int, key int32) {
	stop := size - 8
	pos := 4 + offset
	ecx := key
	for pos < stop {
		edx := int32(raw[pos]) |
			int32(raw[pos+1])<<8 |
			int32(raw[pos+2])<<16 |
			int32(raw[pos+3])<<24
		ecx += edx
		edx ^= ecx
		raw[pos] = byte(edx)
		raw[pos+1] = byte(edx >> 8)
		raw[pos+2] = byte(edx >> 16)
		raw[pos+3] = byte(edx >> 24)
		pos += 4
	}
	raw[pos] = byte(ecx)
	raw[pos+1] = byte(ecx >> 8)
	raw[pos+2] = byte(ecx >> 16)
	raw[pos+3] = byte(ecx >> 24)
}

// DecXORPass reverses EncXORPass so a login client can read Init fields.
func DecXORPass(raw []byte) {
	DecXORPassAt(raw, 0, len(raw))
}

func DecXORPassAt(raw []byte, offset, size int) {
	if size < 12 {
		return
	}
	pos := offset + size - 8
	ecx := int32(raw[pos]) |
		int32(raw[pos+1])<<8 |
		int32(raw[pos+2])<<16 |
		int32(raw[pos+3])<<24
	pos -= 4
	for pos >= 4+offset {
		edx := int32(raw[pos]) |
			int32(raw[pos+1])<<8 |
			int32(raw[pos+2])<<16 |
			int32(raw[pos+3])<<24
		edx ^= ecx
		raw[pos] = byte(edx)
		raw[pos+1] = byte(edx >> 8)
		raw[pos+2] = byte(edx >> 16)
		raw[pos+3] = byte(edx >> 24)
		ecx -= edx
		pos -= 4
	}
}

func RequireBlockSize(size int) error {
	if size%8 != 0 {
		return fmt.Errorf("size have to be multiple of 8")
	}
	return nil
}
