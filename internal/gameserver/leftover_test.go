package gameserver

import (
	"testing"

	"github.com/pblaravel/game-server-l2-go/internal/packet"
)

func TestRecipeBookDestroyRemovesRecipe(t *testing.T) {
	rec := GetRecipe(1)
	if rec == nil {
		t.Fatal("recipe 1 missing")
	}
	srv := combatTestServer(t)
	p := DefaultCharacter("acc", "Forget", 0, 0, 0, 1, 1, 1, 268442100, nil)
	c, stop := testClient(t, srv, p)
	defer stop()
	p.Recipes = []int32{rec.ID}
	w := packet.NewWriter()
	w.WriteC(0xAD)
	w.WriteD(rec.ID)
	r := packet.NewReader(w.Bytes())
	r.SkipOpcode()
	srv.onRecipeBookDestroy(c, r)
	if HasRecipe(p, rec.ID) {
		t.Fatal("recipe should have been removed")
	}
}

func TestChangePartyLeader(t *testing.T) {
	srv := combatTestServer(t)
	a := DefaultCharacter("acc", "Leader", 0, 0, 0, 1, 1, 1, 268442101, nil)
	b := DefaultCharacter("acc2", "Member", 0, 0, 0, 1, 1, 1, 268442102, nil)
	ca, stopA := testClient(t, srv, a)
	defer stopA()
	cb, stopB := testClient(t, srv, b)
	defer stopB()

	w := packet.NewWriter()
	w.WriteC(0x29)
	w.WriteS("Member")
	w.WriteD(0)
	r := packet.NewReader(w.Bytes())
	r.SkipOpcode()
	srv.onJoinParty(ca, r)

	w = packet.NewWriter()
	w.WriteC(0x2A)
	w.WriteD(1)
	r = packet.NewReader(w.Bytes())
	r.SkipOpcode()
	srv.onAnswerJoinParty(cb, r)

	w = packet.NewWriter()
	w.WriteC(0xD0)
	w.WriteH(4)
	w.WriteS("Member")
	srv.handle(ca, w.Bytes())
	pt := srv.partyOf(a)
	if pt == nil || pt.LeaderID != b.ObjectID {
		t.Fatalf("leader = %#v, want %d", pt, b.ObjectID)
	}
}

func TestUserCommandLocAndTime(t *testing.T) {
	srv := combatTestServer(t)
	p := DefaultCharacter("acc", "Cmd", 0, 0, 0, 1, 1, 1, 268442103, nil)
	c, stop := testClient(t, srv, p)
	defer stop()
	w := packet.NewWriter()
	w.WriteC(0xAA)
	w.WriteD(0)
	r := packet.NewReader(w.Bytes())
	r.SkipOpcode()
	srv.onUserCommand(c, r)
	w = packet.NewWriter()
	w.WriteC(0xAA)
	w.WriteD(77)
	r = packet.NewReader(w.Bytes())
	r.SkipOpcode()
	srv.onUserCommand(c, r)
}

func TestQuestListAndMiniMapOpcodes(t *testing.T) {
	srv := combatTestServer(t)
	p := DefaultCharacter("acc", "Map", 0, 0, 0, 1, 1, 1, 268442104, nil)
	c, stop := testClient(t, srv, p)
	defer stop()
	w := packet.NewWriter()
	w.WriteC(0x63)
	srv.handle(c, w.Bytes())
	w = packet.NewWriter()
	w.WriteC(0xCD)
	srv.handle(c, w.Bytes())
}
