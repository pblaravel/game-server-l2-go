package crypt

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"math/big"
)

// DecryptPKCS1 decrypts a 128-byte client auth block (RSA/ECB/PKCS1Padding).
func DecryptPKCS1(priv *rsa.PrivateKey, ciphertext []byte) ([]byte, error) {
	return rsa.DecryptPKCS1v15(rand.Reader, priv, ciphertext)
}

// DecryptNoPadding performs raw RSA decryption (RSA/ECB/nopadding) used for GS BlowFishKey.
func DecryptNoPadding(priv *rsa.PrivateKey, ciphertext []byte) ([]byte, error) {
	c := new(big.Int).SetBytes(ciphertext)
	m := new(big.Int).Exp(c, priv.D, priv.N)
	out := m.Bytes()
	// Java Cipher.doFinal for 512-bit RSA returns 64 bytes (modulus size).
	size := (priv.N.BitLen() + 7) / 8
	if len(out) < size {
		padded := make([]byte, size)
		copy(padded[size-len(out):], out)
		out = padded
	}
	return out, nil
}

// EncryptNoPadding performs raw RSA encryption (RSA/ECB/nopadding).
func EncryptNoPadding(pub *rsa.PublicKey, plaintext []byte) ([]byte, error) {
	size := (pub.N.BitLen() + 7) / 8
	if len(plaintext) > size {
		return nil, fmt.Errorf("plaintext longer than modulus")
	}
	padded := plaintext
	if len(plaintext) < size {
		padded = make([]byte, size)
		copy(padded[size-len(plaintext):], plaintext)
	}
	m := new(big.Int).SetBytes(padded)
	if m.Cmp(pub.N) >= 0 {
		return nil, fmt.Errorf("plaintext representative out of range")
	}
	c := new(big.Int).Exp(m, big.NewInt(int64(pub.E)), pub.N)
	out := c.Bytes()
	if len(out) < size {
		full := make([]byte, size)
		copy(full[size-len(out):], out)
		out = full
	}
	return out, nil
}

// JavaModulusBytes is BigInteger.toByteArray(): unsigned bytes plus a leading
// 0x00 when the high bit is set so Java treats the value as positive.
func JavaModulusBytes(n *big.Int) []byte {
	b := n.Bytes()
	if len(b) > 0 && b[0]&0x80 != 0 {
		return append([]byte{0x00}, b...)
	}
	return b
}

// StripLeadingZeros matches Java BlowFishKeyPacket key cleanup.
func StripLeadingZeros(b []byte) []byte {
	i := 0
	for i < len(b) && b[i] == 0 {
		i++
	}
	return append([]byte(nil), b[i:]...)
}
