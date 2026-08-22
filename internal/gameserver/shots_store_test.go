package gameserver

import (
	"testing"

	"github.com/pblaravel/game-server-l2-go/internal/packet"
)

func TestSoulshotChargesAndDoublesMelee(t *testing.T) {
	if GetItem(1835) == nil || shotKind(GetItem(1835)) != shotKindSS {
		t.Fatal("soulshot 1835 missing")
	}
	srv := combatTestServer(t)
	p := DefaultCharacter("acc", "Shooter", 0, 0, 0, 1, 1, 1, 268442000, nil)
	p.Level = 10
	RecalcStats(p)
	c, stop := testClient(t, srv, p)
	defer stop()
	ss := AddItem(p, 1835, 10, srv.nextItemID)
	srv.chargeShot(c, ss, false)
	if !p.ChargedSS {
		t.Fatal("soulshot should be charged")
	}
	if ItemCountOf(p, 1835) != 9 {
		t.Fatalf("ss count = %d, want 9", ItemCountOf(p, 1835))
	}
	npc := &NPC{ObjectID: srv.world.NextID(), NPCID: 20001, Name: "Dummy",
		X: p.X, Y: p.Y, Z: p.Z, Level: 1, MaxHP: 5000, CurHP: 5000, IsAttackable: true, PDef: 77}
	npc.NpcDefaults()
	npc.PDef = 77
	srv.world.AddNPC(npc)
	c.target = npc.ObjectID
	// Force a hit: retry until soulshot is consumed (miss leaves it charged).
	for i := 0; i < 40 && p.ChargedSS; i++ {
		srv.doAttackHit(c, npc)
	}
	if p.ChargedSS {
		t.Fatal("soulshot should have been consumed on a hit")
	}
}

func TestAutoSoulShotExOpcode(t *testing.T) {
	srv := combatTestServer(t)
	p := DefaultCharacter("acc", "Auto", 0, 0, 0, 1, 1, 1, 268442001, nil)
	c, stop := testClient(t, srv, p)
	defer stop()
	AddItem(p, 1835, 5, srv.nextItemID)
	w := packet.NewWriter()
	w.WriteC(0xD0)
	w.WriteH(5)
	w.WriteD(1835)
	w.WriteD(1)
	srv.handle(c, w.Bytes())
	if !hasAutoShot(p, 1835) {
		t.Fatal("auto soulshot should be enabled")
	}
	if !p.ChargedSS {
		t.Fatal("first shot should charge on auto enable")
	}
	w = packet.NewWriter()
	w.WriteC(0xD0)
	w.WriteH(5)
	w.WriteD(1835)
	w.WriteD(0)
	srv.handle(c, w.Bytes())
	if hasAutoShot(p, 1835) {
		t.Fatal("auto soulshot should be cancelled")
	}
}

func TestBlessedSpiritshotMagicFormula(t *testing.T) {
	base := CalcMagicDamage(100, 91, 10, false, false, false)
	sps := CalcMagicDamage(100, 91, 10, false, true, false)
	bss := CalcMagicDamage(100, 91, 10, false, false, true)
	if sps <= base {
		t.Fatalf("SPS damage %v should exceed base %v", sps, base)
	}
	if bss <= sps {
		t.Fatalf("BSS damage %v should exceed SPS %v", bss, sps)
	}
	// Java: BSS *4 mAtk => sqrt(4)=2x the no-shot damage.
	if bss < base*2-0.01 || bss > base*2+0.01 {
		t.Fatalf("BSS = %v, want 2x base %v", bss, base)
	}
}

func TestPrivateSellStoreBuyout(t *testing.T) {
	srv := combatTestServer(t)
	seller := DefaultCharacter("acc", "Seller", 0, 0, 0, 1, 1, 1, 268442002, nil)
	buyer := DefaultCharacter("acc2", "Buyer", 0, 0, 0, 1, 1, 1, 268442003, nil)
	cs, stopS := testClient(t, srv, seller)
	defer stopS()
	cb, stopB := testClient(t, srv, buyer)
	defer stopB()
	club := AddItem(seller, 4, 1, srv.nextItemID)
	if !ReduceAdena(buyer, AdenaCount(buyer)) && AdenaCount(buyer) != 0 {
		t.Fatal("clear buyer adena")
	}
	AddAdena(buyer, 500, srv.nextItemID)

	w := packet.NewWriter()
	w.WriteC(0x74)
	w.WriteD(0)
	w.WriteD(1)
	w.WriteD(club.ObjectID)
	w.WriteD(1)
	w.WriteD(200)
	r := packet.NewReader(w.Bytes())
	r.SkipOpcode()
	srv.onSetPrivateStoreListSell(cs, r)
	if seller.PrivateStore != StoreSell {
		t.Fatalf("store type = %d, want sell", seller.PrivateStore)
	}

	w = packet.NewWriter()
	w.WriteC(0x79)
	w.WriteD(seller.ObjectID)
	w.WriteD(1)
	w.WriteD(club.ObjectID)
	w.WriteD(1)
	w.WriteD(200)
	r = packet.NewReader(w.Bytes())
	r.SkipOpcode()
	srv.onPrivateStoreBuy(cb, r)
	if FindItemByID(buyer, 4) == nil {
		t.Fatal("buyer should have received the club")
	}
	if FindItemByID(seller, 4) != nil {
		t.Fatal("seller should have given the club")
	}
	if AdenaCount(buyer) != 300 {
		t.Fatalf("buyer adena = %d, want 300", AdenaCount(buyer))
	}
	if AdenaCount(seller) < 200 {
		t.Fatalf("seller adena = %d, want at least 200", AdenaCount(seller))
	}
	if seller.PrivateStore != StoreNone {
		t.Fatal("empty store should have closed")
	}
}

func TestPrivateBuyStorePurchase(t *testing.T) {
	srv := combatTestServer(t)
	buyer := DefaultCharacter("acc", "BuyerStore", 0, 0, 0, 1, 1, 1, 268442005, nil)
	seller := DefaultCharacter("acc2", "SellerWalk", 0, 0, 0, 1, 1, 1, 268442006, nil)
	cb, stopB := testClient(t, srv, buyer)
	defer stopB()
	cs, stopS := testClient(t, srv, seller)
	defer stopS()
	club := AddItem(seller, 4, 1, srv.nextItemID)
	if AdenaCount(buyer) < 200 {
		AddAdena(buyer, 200, srv.nextItemID)
	}

	w := packet.NewWriter()
	w.WriteC(0x91)
	w.WriteD(1)
	w.WriteD(4)
	w.WriteH(0)
	w.WriteH(0)
	w.WriteD(1)
	w.WriteD(200)
	r := packet.NewReader(w.Bytes())
	r.SkipOpcode()
	srv.onSetPrivateStoreListBuy(cb, r)
	if buyer.PrivateStore != StoreBuy {
		t.Fatalf("store type = %d, want buy", buyer.PrivateStore)
	}

	w = packet.NewWriter()
	w.WriteC(0x96)
	w.WriteD(buyer.ObjectID)
	w.WriteD(1)
	w.WriteD(club.ObjectID)
	w.WriteD(4)
	w.WriteH(0)
	w.WriteH(0)
	w.WriteD(1)
	w.WriteD(200)
	r = packet.NewReader(w.Bytes())
	r.SkipOpcode()
	srv.onPrivateStoreSell(cs, r)
	if FindItemByID(buyer, 4) == nil {
		t.Fatal("buy-store owner should have received the club")
	}
	if FindItemByID(seller, 4) != nil {
		t.Fatal("walker should have sold the club")
	}
	if buyer.PrivateStore != StoreNone {
		t.Fatal("empty buy store should have closed")
	}
}

func TestActionUseOpensSellManage(t *testing.T) {
	srv := combatTestServer(t)
	p := DefaultCharacter("acc", "Merchant", 0, 0, 0, 1, 1, 1, 268442004, nil)
	c, stop := testClient(t, srv, p)
	defer stop()
	w := packet.NewWriter()
	w.WriteC(0x45)
	w.WriteD(10)
	w.WriteD(0)
	w.WriteC(0)
	r := packet.NewReader(w.Bytes())
	r.SkipOpcode()
	srv.onActionUse(c, r)
	if p.PrivateStore != StoreSellManage {
		t.Fatalf("store type = %d, want sell manage", p.PrivateStore)
	}
}
