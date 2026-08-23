package crypt

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"math/big"
	"testing"
)

func TestChecksumRoundTrip(t *testing.T) {
	raw := []byte{0x03, 0x01, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	AppendChecksum(raw)
	if !VerifyChecksum(raw) {
		t.Fatal("checksum should verify after append")
	}
	raw[1] ^= 0xFF
	if VerifyChecksum(raw) {
		t.Fatal("tampered packet must fail checksum")
	}
}

func TestBlowfishRoundTrip(t *testing.T) {
	key := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	plain := []byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77}
	enc := append([]byte(nil), plain...)
	c := New(key)
	c.Crypt(enc, 0, 8)
	if bytes.Equal(enc, plain) {
		t.Fatal("ciphertext must differ")
	}
	c.Decrypt(enc, 0, 8)
	if !bytes.Equal(enc, plain) {
		t.Fatalf("roundtrip failed: %v != %v", enc, plain)
	}
}

func TestLoginCryptInitThenDynamic(t *testing.T) {
	key := bytes.Repeat([]byte{0x41}, 16)
	lc := NewLoginCrypt(key)
	init := make([]byte, 16)
	copy(init, []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f})
	if err := lc.EncryptWithXORKey(init, 0x12345678); err != nil {
		t.Fatal(err)
	}

	// Subsequent packet uses dynamic key + checksum.
	pkt := []byte{0x03, 0xAA, 0x00, 0x00, 0x00, 0xBB, 0x00, 0x00}
	if err := lc.Encrypt(pkt); err != nil {
		t.Fatal(err)
	}
	lc2 := NewLoginCrypt(key)
	// skip static state by encrypting a dummy first
	dummy := make([]byte, 16)
	_ = lc2.EncryptWithXORKey(dummy, 1)
	if err := lc2.Decrypt(pkt); err != nil {
		t.Fatal(err)
	}
	if pkt[0] != 0x03 || pkt[1] != 0xAA {
		t.Fatalf("dynamic decrypt mismatch: %v", pkt)
	}
}

func TestScrambleModulusLength(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	s := ScrambleModulus(key.N)
	if len(s) != 128 {
		t.Fatalf("scrambled modulus must be 128 bytes, got %d", len(s))
	}
	raw := key.N.Bytes()
	if len(raw) == 0x81 && raw[0] == 0x00 {
		raw = raw[1:]
	}
	if len(raw) < 0x80 {
		padded := make([]byte, 0x80)
		copy(padded[0x80-len(raw):], raw)
		raw = padded
	}
	got := UnscrambleModulus(s)
	if !bytes.Equal(got, raw[:0x80]) {
		t.Fatal("unscramble did not restore modulus")
	}
}

func TestRSANoPaddingRoundTrip(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 512)
	if err != nil {
		t.Fatal(err)
	}
	plain := bytes.Repeat([]byte{0x7A}, 40)
	enc, err := EncryptNoPadding(&key.PublicKey, plain)
	if err != nil {
		t.Fatal(err)
	}
	dec, err := DecryptNoPadding(key, enc)
	if err != nil {
		t.Fatal(err)
	}
	got := StripLeadingZeros(dec)
	if !bytes.Equal(got, plain) {
		t.Fatalf("rsa nopadding mismatch %v != %v", got, plain)
	}
}

func TestGameCryptStream(t *testing.T) {
	key := RandomGameKey()
	if len(key) != 16 || !bytes.Equal(key[8:], StaticCryptKeyTail) {
		t.Fatalf("key format: %v", key)
	}
	c := NewGameCrypt(true)
	c.SetKey(key)
	// first encrypt enables cipher without transforming
	first := []byte{0x00, 0x01}
	c.Encrypt(first)
	if first[0] != 0x00 {
		t.Fatal("first outbound packet is not encrypted")
	}
	second := []byte{0x04, 0x11, 0x22, 0x33}
	orig := append([]byte(nil), second...)
	c.Encrypt(second)
	if bytes.Equal(second, orig) {
		t.Fatal("second packet should be encrypted")
	}

	d := NewGameCrypt(true)
	d.SetKey(key)
	d.enabled = true
	d.Decrypt(second)
	if !bytes.Equal(second, orig) {
		t.Fatalf("gamecrypt decrypt mismatch %v != %v", second, orig)
	}
}

func TestJavaModulusBytesAddsSignByte(t *testing.T) {
	n := new(big.Int).SetBytes([]byte{0x80, 0x00})
	got := JavaModulusBytes(n)
	if len(got) != 3 || got[0] != 0x00 || got[1] != 0x80 {
		t.Fatalf("%x", got)
	}
	n = new(big.Int).SetBytes([]byte{0x7F, 0x00})
	got = JavaModulusBytes(n)
	if len(got) != 2 || got[0] != 0x7F {
		t.Fatalf("%x", got)
	}
}

func TestDecXORPassRoundTrip(t *testing.T) {
	plain := []byte{
		0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77,
		0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff,
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
	}
	enc := append([]byte(nil), plain...)
	EncXORPass(enc, 0x12345678)
	if bytes.Equal(enc[4:len(enc)-8], plain[4:len(plain)-8]) {
		t.Fatal("payload between offset 4 and size-8 must change")
	}
	DecXORPass(enc)
	if !bytes.Equal(enc[:len(plain)-8], plain[:len(plain)-8]) {
		t.Fatalf("DecXORPass did not restore Init fields:\n got %x\nwant %x", enc, plain)
	}
}

func TestEncXORPassDeterministic(t *testing.T) {
	a := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	b := append([]byte(nil), a...)
	EncXORPass(a, 42)
	EncXORPass(b, 42)
	if !bytes.Equal(a, b) {
		t.Fatal("xor pass not deterministic")
	}
	if bytes.Equal(a[:8], []byte{1, 2, 3, 4, 5, 6, 7, 8}) {
		// first 4 bytes stay, bytes 4..size-8 change
	}
}
