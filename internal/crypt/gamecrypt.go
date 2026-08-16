package crypt

import "crypto/rand"

// StaticCryptKeyTail is written into bytes 8-15 of every GameCrypt key (BlowFishKeygen).
var StaticCryptKeyTail = []byte{0xc8, 0x27, 0x93, 0x01, 0xa1, 0x6c, 0x31, 0x97}

// GameCrypt is the L2 XOR stream cipher used on the client↔gameserver link.
type GameCrypt struct {
	inKey    [16]byte
	outKey   [16]byte
	enabled  bool
	useCipher bool
}

func NewGameCrypt(useCipher bool) *GameCrypt {
	return &GameCrypt{useCipher: useCipher}
}

func (c *GameCrypt) SetKey(key []byte) {
	copy(c.inKey[:], key[:16])
	copy(c.outKey[:], key[:16])
}

func (c *GameCrypt) Decrypt(raw []byte) {
	if !c.useCipher || !c.enabled {
		return
	}
	temp := 0
	for i := 0; i < len(raw); i++ {
		temp2 := int(raw[i] & 0xFF)
		raw[i] = byte(temp2 ^ int(c.inKey[i&15]) ^ temp)
		temp = temp2
	}
	old := int(c.inKey[8]) |
		int(c.inKey[9])<<8 |
		int(c.inKey[10])<<16 |
		int(c.inKey[11])<<24
	old += len(raw)
	c.inKey[8] = byte(old)
	c.inKey[9] = byte(old >> 8)
	c.inKey[10] = byte(old >> 16)
	c.inKey[11] = byte(old >> 24)
}

func (c *GameCrypt) Encrypt(raw []byte) {
	if !c.enabled {
		c.enabled = c.useCipher
		return
	}
	temp := 0
	for i := 0; i < len(raw); i++ {
		temp2 := int(raw[i] & 0xFF)
		temp = temp2 ^ int(c.outKey[i&15]) ^ temp
		raw[i] = byte(temp)
	}
	old := int(c.outKey[8]) |
		int(c.outKey[9])<<8 |
		int(c.outKey[10])<<16 |
		int(c.outKey[11])<<24
	old += len(raw)
	c.outKey[8] = byte(old)
	c.outKey[9] = byte(old >> 8)
	c.outKey[10] = byte(old >> 16)
	c.outKey[11] = byte(old >> 24)
}

// RandomGameKey returns a 16-byte GameCrypt key (8 random + 8 static tail).
func RandomGameKey() []byte {
	key := make([]byte, 16)
	if _, err := rand.Read(key[:8]); err != nil {
		for i := 0; i < 8; i++ {
			key[i] = byte(i * 17)
		}
	}
	copy(key[8:], StaticCryptKeyTail)
	return key
}
