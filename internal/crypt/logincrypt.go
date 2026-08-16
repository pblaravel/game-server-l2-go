package crypt

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
)

// LoginCrypt matches Java com.shnok.javaserver.security.LoginCrypt.
type LoginCrypt struct {
	static bool
	crypt  *NewCrypt
}

func NewLoginCrypt(dynamicKey []byte) *LoginCrypt {
	return &LoginCrypt{
		static: true,
		crypt:  New(dynamicKey),
	}
}

func (c *LoginCrypt) SetKey(key []byte) {
	c.crypt = New(key)
}

func (c *LoginCrypt) Decrypt(raw []byte) error {
	if len(raw)%8 != 0 {
		return fmt.Errorf("size have to be multiple of 8")
	}
	c.crypt.Decrypt(raw, 0, len(raw))
	if !VerifyChecksum(raw) {
		return fmt.Errorf("wrong checksum")
	}
	return nil
}

// Encrypt encrypts in-place. First packet uses static BF + XOR pass (Init).
func (c *LoginCrypt) Encrypt(raw []byte) error {
	if len(raw)%8 != 0 {
		return fmt.Errorf("size have to be multiple of 8")
	}
	if c.static {
		var kb [4]byte
		if _, err := rand.Read(kb[:]); err != nil {
			return err
		}
		key := int32(binary.LittleEndian.Uint32(kb[:]))
		EncXORPass(raw, key)
		static := New(StaticBlowfishKey)
		static.Crypt(raw, 0, len(raw))
		c.static = false
		return nil
	}
	AppendChecksum(raw)
	c.crypt.Crypt(raw, 0, len(raw))
	return nil
}

// EncryptWithXORKey is used by tests to make Init encryption deterministic.
func (c *LoginCrypt) EncryptWithXORKey(raw []byte, xorKey int32) error {
	if len(raw)%8 != 0 {
		return fmt.Errorf("size have to be multiple of 8")
	}
	if c.static {
		EncXORPass(raw, xorKey)
		static := New(StaticBlowfishKey)
		static.Crypt(raw, 0, len(raw))
		c.static = false
		return nil
	}
	AppendChecksum(raw)
	c.crypt.Crypt(raw, 0, len(raw))
	return nil
}
