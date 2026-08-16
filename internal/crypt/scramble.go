package crypt

import (
	"crypto/rand"
	"crypto/rsa"
	"math/big"
)

// ScrambleModulus applies the L2 client RSA modulus scramble (128 bytes).
func ScrambleModulus(modulus *big.Int) []byte {
	scrambled := modulus.Bytes()
	if len(scrambled) == 0x81 && scrambled[0] == 0x00 {
		scrambled = scrambled[1:]
	}
	if len(scrambled) != 0x80 {
		out := make([]byte, 0x80)
		copy(out[0x80-len(scrambled):], scrambled)
		scrambled = out
	} else {
		scrambled = append([]byte(nil), scrambled...)
	}
	for i := 0; i < 4; i++ {
		scrambled[i], scrambled[0x4d+i] = scrambled[0x4d+i], scrambled[i]
	}
	for i := 0; i < 0x40; i++ {
		scrambled[i] ^= scrambled[0x40+i]
	}
	for i := 0; i < 4; i++ {
		scrambled[0x0d+i] ^= scrambled[0x34+i]
	}
	for i := 0; i < 0x40; i++ {
		scrambled[0x40+i] ^= scrambled[i]
	}
	return scrambled
}

type ScrambledKeyPair struct {
	Private          *rsa.PrivateKey
	ScrambledModulus []byte
}

func NewScrambledKeyPair(bits int) (*ScrambledKeyPair, error) {
	key, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, err
	}
	return &ScrambledKeyPair{
		Private:          key,
		ScrambledModulus: ScrambleModulus(key.N),
	}, nil
}

func RSAModulusBytes(key *rsa.PublicKey) []byte {
	return key.N.Bytes()
}

// UnscrambleModulus reverses ScrambleModulus (client-side).
func UnscrambleModulus(in []byte) []byte {
	s := append([]byte(nil), in...)
	if len(s) != 0x80 {
		return s
	}
	for i := 0; i < 0x40; i++ {
		s[0x40+i] ^= s[i]
	}
	for i := 0; i < 4; i++ {
		s[0x0d+i] ^= s[0x34+i]
	}
	for i := 0; i < 0x40; i++ {
		s[i] ^= s[0x40+i]
	}
	for i := 0; i < 4; i++ {
		s[i], s[0x4d+i] = s[0x4d+i], s[i]
	}
	return s
}
