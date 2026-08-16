package packet

import (
	"bytes"
	"testing"
)

func TestWriteReadPrimitives(t *testing.T) {
	w := NewWriter()
	w.WriteC(0x13)
	w.WriteD(7)
	w.WriteH(0x1234)
	w.WriteQ(99)
	w.WriteS("Bartz")
	w.WriteF64(1.5)

	r := NewReader(w.Bytes())
	if r.ReadC() != 0x13 {
		t.Fatal("opcode")
	}
	if r.ReadD() != 7 {
		t.Fatal("int")
	}
	if r.ReadH() != 0x1234 {
		t.Fatal("short")
	}
	if r.ReadQ() != 99 {
		t.Fatal("long")
	}
	if r.ReadS() != "Bartz" {
		t.Fatal("string")
	}
	if r.ReadF64() != 1.5 {
		t.Fatal("double")
	}
}

func TestPadTo8(t *testing.T) {
	w := NewWriter()
	w.WriteC(0x03)
	w.WriteD(1)
	w.WriteD(2)
	w.PadTo8()
	if w.Len()%8 != 0 {
		t.Fatalf("not padded: %d", w.Len())
	}
}

func TestFrameRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	payload := []byte{0x00, 0x01, 0x02, 0x03}
	if err := WriteFrame(&buf, payload); err != nil {
		t.Fatal(err)
	}
	got, err := ReadFrame(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("%v != %v", got, payload)
	}
}

func TestUTF16EmptyString(t *testing.T) {
	w := NewWriter()
	w.WriteS("")
	r := NewReader(w.Bytes())
	if r.ReadS() != "" {
		t.Fatal("empty string")
	}
}
