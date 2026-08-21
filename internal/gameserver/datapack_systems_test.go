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
	// Cedric's Training Hall area (the newbie spawn) maps to that restart point.
	hall := &Character{X: -71338, Y: 258271, Z: -3104, Race: 0}
	hallLoc := NearestRestartLocation(hall)
	if hallLoc[0] != -71338 {
		t.Errorf("Cedric hall restart X = %d, want -71338", hallLoc[0])
	}
	// Wilderness on geomap 17;25 (outside the school polygon) is talking_island_town.
	p := &Character{X: -80000, Y: 250000, Z: -3700, Race: 0}
	loc := NearestRestartLocation(p)
	if loc[0] == 0 && loc[1] == 0 {
		t.Fatalf("restart loc is zero: %#v", loc)
	}
	if loc[0] > -80000 || loc[0] < -90000 {
		t.Errorf("Talking Island town restart X = %d, want around -84000", loc[0])
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

func TestTradeExchange(t *testing.T) {
	srv := combatTestServer(t)
	a := DefaultCharacter("acc", "Alice", 0, 0, 0, 1, 1, 1, 268440020, nil)
	b := DefaultCharacter("acc2", "Bob", 0, 0, 0, 1, 1, 1, 268440021, nil)
	ca, stopA := testClient(t, srv, a)
	defer stopA()
	cb, stopB := testClient(t, srv, b)
	defer stopB()

	AddAdena(a, 500, srv.nextItemID)
	AddAdena(b, 100, srv.nextItemID)
	beforeA, beforeB := AdenaCount(a), AdenaCount(b)
	adena := FindItemByID(a, AdenaID)
	if adena == nil || adena.Count < 100 {
		t.Fatal("alice needs adena to trade")
	}

	w := packet.NewWriter()
	w.WriteC(0x15)
	w.WriteD(b.ObjectID)
	r := packet.NewReader(w.Bytes())
	r.SkipOpcode()
	srv.onTradeRequest(ca, r)

	w = packet.NewWriter()
	w.WriteC(0x44)
	w.WriteD(1)
	r = packet.NewReader(w.Bytes())
	r.SkipOpcode()
	srv.onAnswerTradeRequest(cb, r)
	if srv.tradeOf(a.ObjectID) == nil {
		t.Fatal("trade session missing")
	}

	if !IsTradable(AdenaID) {
		t.Fatal("adena must be tradable")
	}
	w = packet.NewWriter()
	w.WriteC(0x16)
	w.WriteD(0)
	w.WriteD(adena.ObjectID)
	w.WriteD(100)
	r = packet.NewReader(w.Bytes())
	r.SkipOpcode()
	srv.onAddTradeItem(ca, r)
	if sess := srv.tradeOf(a.ObjectID); sess == nil || len(sess.ItemsA)+len(sess.ItemsB) == 0 {
		t.Fatalf("trade offer missing: tradable=%v item=%#v", IsTradable(AdenaID), FindItemByID(a, AdenaID))
	}

	confirm := func(cl *GameClient) {
		w := packet.NewWriter()
		w.WriteC(0x17)
		w.WriteD(1)
		r := packet.NewReader(w.Bytes())
		r.SkipOpcode()
		srv.onTradeDone(cl, r)
	}
	confirm(ca)
	confirm(cb)

	if AdenaCount(a) != beforeA-100 {
		t.Fatalf("alice adena %d, want %d", AdenaCount(a), beforeA-100)
	}
	if AdenaCount(b) != beforeB+100 {
		t.Fatalf("bob adena %d, want %d", AdenaCount(b), beforeB+100)
	}
}

func TestWarehouseDepositAndWithdraw(t *testing.T) {
	srv := combatTestServer(t)
	p := DefaultCharacter("acc", "Porter", 0, 0, 0, 1, 1, 1, 268440022, nil)
	c, stop := testClient(t, srv, p)
	defer stop()

	npc := &NPC{ObjectID: srv.world.NextID(), NPCID: 30054, Name: "Rant",
		Title: "Warehouse Keeper", Type: "WarehouseKeeper", X: p.X, Y: p.Y, Z: p.Z}
	ApplyNpcTemplate(npc)
	npc.NpcDefaults()
	srv.world.AddNPC(npc)
	c.target = npc.ObjectID

	sword := FindItemByID(p, 2369)
	if sword == nil {
		t.Fatal("missing starter sword")
	}
	if sword.Equipped {
		UnequipBodyPart(p, sword.BodyPart)
	}
	AddAdena(p, 1000, srv.nextItemID)

	w := packet.NewWriter()
	w.WriteC(0x31)
	w.WriteD(1)
	w.WriteD(sword.ObjectID)
	w.WriteD(1)
	r := packet.NewReader(w.Bytes())
	r.SkipOpcode()
	srv.onWarehouseDeposit(c, r)
	if FindItemByID(p, 2369) != nil {
		t.Fatal("sword should be in the warehouse")
	}
	if len(p.Warehouse) == 0 {
		t.Fatal("warehouse empty after deposit")
	}

	oid := p.Warehouse[0].ObjectID
	w = packet.NewWriter()
	w.WriteC(0x32)
	w.WriteD(1)
	w.WriteD(oid)
	w.WriteD(1)
	r = packet.NewReader(w.Bytes())
	r.SkipOpcode()
	srv.onWarehouseWithdraw(c, r)
	if FindItemByID(p, 2369) == nil {
		t.Fatal("sword should be back in inventory")
	}
}

func TestShortcutRegDelAndDestroy(t *testing.T) {
	srv := combatTestServer(t)
	p := DefaultCharacter("acc", "Hotbar", 0, 0, 0, 1, 1, 1, 268440023, nil)
	c, stop := testClient(t, srv, p)
	defer stop()

	w := packet.NewWriter()
	w.WriteC(0x33)
	w.WriteD(ShortcutAction)
	w.WriteD(3)
	w.WriteD(2)
	w.WriteD(1)
	r := packet.NewReader(w.Bytes())
	r.SkipOpcode()
	srv.onShortCutReg(c, r)
	found := false
	for _, sc := range p.Shortcuts {
		if sc.Slot == 3 && sc.Type == ShortcutAction && sc.ID == 2 {
			found = true
		}
	}
	if !found {
		t.Fatal("shortcut was not registered")
	}

	w = packet.NewWriter()
	w.WriteC(0x35)
	w.WriteD(3)
	r = packet.NewReader(w.Bytes())
	r.SkipOpcode()
	srv.onShortCutDel(c, r)
	for _, sc := range p.Shortcuts {
		if sc.Slot == 3 && sc.Page == 0 {
			t.Fatal("shortcut should have been deleted")
		}
	}

	potion := AddItem(p, 1060, 2, srv.nextItemID)
	if potion == nil {
		t.Fatal("could not add potion")
	}
	w = packet.NewWriter()
	w.WriteC(0x59)
	w.WriteD(potion.ObjectID)
	w.WriteD(1)
	r = packet.NewReader(w.Bytes())
	r.SkipOpcode()
	srv.onDestroyItem(c, r)
	if it := FindItemByID(p, 1060); it == nil || it.Count != 1 {
		t.Fatalf("expected 1 potion left, got %#v", it)
	}
}

func TestFriendInviteAndList(t *testing.T) {
	srv := combatTestServer(t)
	a := DefaultCharacter("acc", "Ann", 0, 0, 0, 1, 1, 1, 268440024, nil)
	b := DefaultCharacter("acc2", "Ben", 0, 0, 0, 1, 1, 1, 268440025, nil)
	ca, stopA := testClient(t, srv, a)
	defer stopA()
	cb, stopB := testClient(t, srv, b)
	defer stopB()

	w := packet.NewWriter()
	w.WriteC(0x5E)
	w.WriteS("Ben")
	r := packet.NewReader(w.Bytes())
	r.SkipOpcode()
	srv.onFriendInvite(ca, r)

	w = packet.NewWriter()
	w.WriteC(0x5F)
	w.WriteD(1)
	r = packet.NewReader(w.Bytes())
	r.SkipOpcode()
	srv.onAnswerFriendInvite(cb, r)
	if !hasFriend(a, b.ObjectID) || !hasFriend(b, a.ObjectID) {
		t.Fatalf("friends a=%v b=%v", a.Friends, b.Friends)
	}

	w = packet.NewWriter()
	w.WriteC(0x61)
	w.WriteS("Ben")
	r = packet.NewReader(w.Bytes())
	r.SkipOpcode()
	srv.onFriendDel(ca, r)
	if hasFriend(a, b.ObjectID) {
		t.Fatal("friend should have been removed")
	}
}

func TestUseScrollOfEscapeConsumesItem(t *testing.T) {
	if GetItem(736) == nil || GetItem(736).SkillID == 0 {
		t.Skip("scroll of escape XML not loaded")
	}
	srv := combatTestServer(t)
	p := DefaultCharacter("acc", "Runner", 0, 0, 0, 1, 1, 1, 268440026, nil)
	c, stop := testClient(t, srv, p)
	defer stop()
	it := AddItem(p, 736, 1, srv.nextItemID)
	w := packet.NewWriter()
	w.WriteC(0x14)
	w.WriteD(it.ObjectID)
	r := packet.NewReader(w.Bytes())
	r.SkipOpcode()
	srv.onUseItem(c, r)
	if FindItemByID(p, 736) != nil {
		t.Fatal("scroll of escape should have been consumed")
	}
}
