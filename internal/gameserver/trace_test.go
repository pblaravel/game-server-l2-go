package gameserver

import (
	"strings"
	"testing"

	"github.com/pblaravel/game-server-l2-go/internal/packet"
)

func TestOpcodeNames(t *testing.T) {
	if clientOpcodeName(0x02, []byte{0x02}) != "PlayerMoveDirection" {
		t.Fatal("client 0x02")
	}
	if serverOpcodeName(0xC6) != "MoveDirection" {
		t.Fatal("server 0xC6")
	}
	if ClientState(StateInGame).String() != "ingame" {
		t.Fatal("state")
	}
}

func TestHexPreview(t *testing.T) {
	got := hexPreview([]byte{0x00, 0x01, 0x02}, 8)
	if got != "000102 (3B)" {
		t.Fatal(got)
	}
	long := make([]byte, 40)
	p := hexPreview(long, 4)
	if p != "00000000… (40B)" {
		t.Fatal(p)
	}
}

func TestRecvMoveSummary(t *testing.T) {
	w := packet.NewWriter()
	w.WriteC(0x02)
	w.WriteC(1)
	w.WriteF64(0.5)
	w.WriteF64(-0.25)
	w.WriteD(8192)
	w.WriteF64(0)
	w.WriteD(-71338)
	w.WriteD(258271)
	w.WriteD(-3104)
	w.WriteQ(99)
	s := recvSummary(0x02, w.Bytes())
	if !strings.Contains(s, "pos=(-71338,258271,-3104)") || !strings.Contains(s, "heading=8192") {
		t.Fatal(s)
	}
}

func TestSendMoveSummary(t *testing.T) {
	p := MoveDirection(10, 20, 30, 40, -71338, 258271, -3104, 99)
	s := sendSummary(0xC6, p)
	if !strings.Contains(s, "oid=10") || !strings.Contains(s, "pos=(-71338,258271,-3104)") {
		t.Fatal(s)
	}
}
