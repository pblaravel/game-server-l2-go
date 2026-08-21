package gameserver

import (
	"testing"

	"github.com/pblaravel/game-server-l2-go/internal/packet"
)

func TestItemDataLoadsStarterSword(t *testing.T) {
	if !ItemsLoaded() {
		t.Skip("item XML not loaded")
	}
	tpl := GetItem(2369)
	if tpl == nil {
		t.Fatal("missing Squire's Sword 2369")
	}
	if tpl.BodyPart != SlotRHand {
		t.Errorf("body part = %#x, want rhand %#x", tpl.BodyPart, SlotRHand)
	}
	if tpl.PAtk <= 0 || tpl.PAtkSpd <= 0 {
		t.Errorf("weapon stats pAtk=%d pAtkSpd=%d", tpl.PAtk, tpl.PAtkSpd)
	}
	if GetItem(AdenaID) == nil || !GetItem(AdenaID).Stackable {
		t.Fatal("adena must be stackable")
	}
}

func TestNpcDataLoadsWolfAndMerchant(t *testing.T) {
	wolf := GetNpcTemplate(20120)
	if wolf == nil {
		t.Fatal("missing Wolf 20120")
	}
	if wolf.Level != 4 || wolf.HP <= 0 || wolf.Exp <= 0 {
		t.Errorf("wolf level=%d hp=%d exp=%d", wolf.Level, wolf.HP, wolf.Exp)
	}
	if !wolf.CanBeAttacked || !isMonsterType(wolf.Type) {
		t.Errorf("wolf should be an attackable monster, type=%s", wolf.Type)
	}
	if len(wolf.Drops) == 0 {
		t.Error("wolf should have drop categories")
	}
	lector := GetNpcTemplate(30001)
	if lector == nil {
		t.Fatal("missing Lector 30001")
	}
	if lector.CanBeAttacked {
		t.Error("merchant Lector must not be attackable")
	}
	if len(BuyListsForNPC(30001)) == 0 {
		t.Error("Lector should have buy lists")
	}
}

func TestTeleportAndRestartData(t *testing.T) {
	locs := TeleportsForNPC(30006)
	if len(locs) == 0 {
		t.Fatal("Roxxy 30006 should have teleport destinations")
	}
	if RestartPointCount() == 0 {
		t.Fatal("expected restart points from XML")
	}
	p := &Character{X: -71338, Y: 258271, Z: -3104, Race: 0}
	loc := NearestRestartLocation(p)
	if loc[0] == 0 && loc[1] == 0 {
		t.Fatalf("restart loc is zero: %#v", loc)
	}
	// Talking Island maps 17;25 should resolve to talking_island_town.
	if loc[0] > -80000 || loc[0] < -90000 {
		t.Errorf("Talking Island restart X = %d, want around -84000", loc[0])
	}
}

func TestBuyAndSellAgainstBuyList(t *testing.T) {
	if GetBuyList(1) == nil {
		t.Skip("buy lists not loaded")
	}
	srv := combatTestServer(t)
	p := DefaultCharacter("acc", "Buyer", 0, 0, 0, 1, 1, 1, 268440010, nil)
	if !ReduceAdena(p, AdenaCount(p)) && AdenaCount(p) != 0 {
		t.Fatal("could not clear starter adena")
	}
	AddAdena(p, 100000, srv.nextItemID)
	c, stop := testClient(t, srv, p)
	defer stop()

	npc := &NPC{ObjectID: srv.world.NextID(), NPCID: 30001, Name: "Lector",
		X: p.X, Y: p.Y, Z: p.Z}
	ApplyNpcTemplate(npc)
	npc.NpcDefaults()
	srv.world.AddNPC(npc)
	c.target = npc.ObjectID

	before := AdenaCount(p)
	w := packet.NewWriter()
	w.WriteC(0x1F)
	w.WriteD(1) // list id
	w.WriteD(1) // one item
	w.WriteD(1) // short sword
	w.WriteD(1)
	r := packet.NewReader(w.Bytes())
	r.SkipOpcode()
	srv.onBuyItem(c, r)
	if FindItemByID(p, 1) == nil {
		t.Fatal("expected short sword after buy")
	}
	if AdenaCount(p) >= before {
		t.Errorf("adena did not decrease: before=%d after=%d", before, AdenaCount(p))
	}

	sword := FindItemByID(p, 1)
	before = AdenaCount(p)
	w = packet.NewWriter()
	w.WriteC(0x1E)
	w.WriteD(0)
	w.WriteD(1)
	w.WriteD(sword.ObjectID)
	w.WriteD(1)
	w.WriteD(1)
	r = packet.NewReader(w.Bytes())
	r.SkipOpcode()
	srv.onSellItem(c, r)
	if FindItemByID(p, 1) != nil {
		t.Fatal("short sword should have been sold")
	}
	if AdenaCount(p) <= before {
		t.Errorf("adena did not increase after sell: before=%d after=%d", before, AdenaCount(p))
	}
}

func TestPartyInviteAndAccept(t *testing.T) {
	srv := combatTestServer(t)
	a := DefaultCharacter("acc", "Leader", 0, 0, 0, 1, 1, 1, 268440011, nil)
	b := DefaultCharacter("acc2", "Member", 0, 0, 0, 1, 1, 1, 268440012, nil)
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

	if a.PartyID == 0 || a.PartyID != b.PartyID {
		t.Fatalf("party ids leader=%d member=%d", a.PartyID, b.PartyID)
	}
	pt := srv.partyOf(a)
	if pt == nil || len(pt.Members) != 2 || pt.LeaderID != a.ObjectID {
		t.Fatalf("party = %#v", pt)
	}
	srv.onWithdrawParty(cb)
	if b.PartyID != 0 {
		t.Error("member should have left")
	}
}

func TestNpcTemplateAppliedToDefaultSpawns(t *testing.T) {
	w := NewWorld()
	w.LoadDefaultSpawns()
	var wolf *NPC
	for _, n := range w.NPCs() {
		if n.NPCID == 20120 {
			wolf = n
			break
		}
	}
	if wolf == nil {
		t.Fatal("expected a wolf spawn")
	}
	if tpl := GetNpcTemplate(20120); tpl != nil && wolf.Exp != tpl.Exp {
		t.Errorf("wolf exp = %d, want template %d", wolf.Exp, tpl.Exp)
	}
}
