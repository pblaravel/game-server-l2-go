package gameserver

import (
	"testing"

	"github.com/pblaravel/game-server-l2-go/internal/packet"
)

func TestVersionCheckLayout(t *testing.T) {
	key := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	p := VersionCheck(key, true)
	r := packet.NewReader(p)
	if r.ReadC() != 0x00 {
		t.Fatal("opcode")
	}
	if r.ReadC() != 0x01 {
		t.Fatal("ok")
	}
	got := r.ReadB(8)
	for i := 0; i < 8; i++ {
		if got[i] != key[i] {
			t.Fatal("key")
		}
	}
	if r.ReadD() != 1 {
		t.Fatal("cipher flag")
	}
}

func TestCharSelectInfoOpcode(t *testing.T) {
	ch := DefaultCharacter("acc", "Hero", 0, 0, 0, 0, 0, 0, 100, nil)
	p := CharSelectInfo("acc", 42, []*Character{ch})
	if p[0] != 0x13 {
		t.Fatalf("opcode %d", p[0])
	}
	r := packet.NewReader(p)
	r.SkipOpcode()
	if r.ReadD() != 1 {
		t.Fatal("count")
	}
	if r.ReadS() != "Hero" {
		t.Fatal("name")
	}
}

func TestMoveDirectionLayout(t *testing.T) {
	p := MoveDirection(10, 20, 30, 40, 1, 2, 3, 99)
	r := packet.NewReader(p)
	if r.ReadC() != 0xC6 {
		t.Fatal("opcode")
	}
	if r.ReadD() != 10 || r.ReadD() != 20 || r.ReadD() != 30 {
		t.Fatal("ids")
	}
}

func TestItemListUnitySlot(t *testing.T) {
	p := ItemList([]Item{{ObjectID: 1, ItemID: 57, Count: 100, Slot: 3}}, true)
	r := packet.NewReader(p)
	if r.ReadC() != 0x1B {
		t.Fatal("opcode")
	}
	if r.ReadH() != 1 {
		t.Fatal("show")
	}
	if r.ReadH() != 1 {
		t.Fatal("count")
	}
}

func TestStartingClasses(t *testing.T) {
	for _, id := range []int32{0, 10, 18, 25, 31, 38, 44, 49, 53} {
		if _, ok := startingClasses[id]; !ok {
			t.Fatalf("missing class %d", id)
		}
	}
}

func TestCharCreateValidation(t *testing.T) {
	if !nameRE.MatchString("Hero123") {
		t.Fatal("valid name")
	}
	if nameRE.MatchString("bad name") {
		t.Fatal("space")
	}
}
