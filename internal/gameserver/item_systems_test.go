package gameserver

import (
	"testing"

	"github.com/pblaravel/game-server-l2-go/internal/packet"
)

func TestDropAndPickupGroundItem(t *testing.T) {
	srv := combatTestServer(t)
	p := DefaultCharacter("acc", "Dropper", 0, 0, 0, 1, 1, 1, 268441000, nil)
	c, stop := testClient(t, srv, p)
	defer stop()

	it := AddItem(p, AdenaID, 50, srv.nextItemID)
	w := packet.NewWriter()
	w.WriteC(0x12)
	w.WriteD(it.ObjectID)
	w.WriteD(20)
	w.WriteD(p.X + 10)
	w.WriteD(p.Y)
	w.WriteD(p.Z)
	r := packet.NewReader(w.Bytes())
	r.SkipOpcode()
	srv.onDropItem(c, r)

	if AdenaCount(p) != 30 {
		t.Fatalf("adena after drop = %d, want 30", AdenaCount(p))
	}
	ground := srv.world.GroundItems()
	if len(ground) != 1 || ground[0].ItemID != AdenaID || ground[0].Count != 20 {
		t.Fatalf("ground = %#v", ground)
	}

	w = packet.NewWriter()
	w.WriteC(0x04)
	w.WriteD(ground[0].ObjectID)
	w.WriteD(p.X)
	w.WriteD(p.Y)
	w.WriteD(p.Z)
	w.WriteC(0)
	r = packet.NewReader(w.Bytes())
	r.SkipOpcode()
	srv.onAction(c, r)
	if AdenaCount(p) != 50 {
		t.Fatalf("adena after pickup = %d, want 50", AdenaCount(p))
	}
	if len(srv.world.GroundItems()) != 0 {
		t.Fatal("ground item should have been removed")
	}
}

func TestDropRejectedWhenTooFar(t *testing.T) {
	srv := combatTestServer(t)
	p := DefaultCharacter("acc", "Far", 0, 0, 0, 1, 1, 1, 268441001, nil)
	c, stop := testClient(t, srv, p)
	defer stop()
	it := AddItem(p, AdenaID, 10, srv.nextItemID)
	w := packet.NewWriter()
	w.WriteC(0x12)
	w.WriteD(it.ObjectID)
	w.WriteD(1)
	w.WriteD(p.X + 10000)
	w.WriteD(p.Y)
	w.WriteD(p.Z)
	r := packet.NewReader(w.Bytes())
	r.SkipOpcode()
	srv.onDropItem(c, r)
	if AdenaCount(p) != 10 {
		t.Fatal("far drop should have been rejected")
	}
	if len(srv.world.GroundItems()) != 0 {
		t.Fatal("no ground item expected")
	}
}

func TestEnchantScrollSucceedsBelowSafeMax(t *testing.T) {
	if _, ok := GetEnchantScroll(955); !ok {
		t.Fatal("D-grade weapon scroll 955 missing")
	}
	if GetItem(69) == nil || GetItem(69).CrystalType != "D" {
		t.Fatal("Bastard Sword 69 should be D-grade")
	}
	srv := combatTestServer(t)
	p := DefaultCharacter("acc", "Enchanter", 0, 0, 0, 1, 1, 1, 268441002, nil)
	c, stop := testClient(t, srv, p)
	defer stop()
	sword := AddItem(p, 69, 1, srv.nextItemID)
	scroll := AddItem(p, 955, 1, srv.nextItemID)
	srv.beginEnchant(c, scroll)
	w := packet.NewWriter()
	w.WriteC(0x58)
	w.WriteD(sword.ObjectID)
	r := packet.NewReader(w.Bytes())
	r.SkipOpcode()
	srv.onEnchantItem(c, r)
	if sword.Enchant != 1 {
		t.Fatalf("enchant = %d, want 1 (safe +0)", sword.Enchant)
	}
	if FindItemByID(p, 955) != nil {
		t.Fatal("scroll should have been consumed")
	}
}

func TestCrystallizeDGradeWeapon(t *testing.T) {
	srv := combatTestServer(t)
	p := DefaultCharacter("acc", "Smith", 0, 0, 0, 1, 1, 1, 268441003, nil)
	p.Skills = append(p.Skills, Skill{ID: skillCrystallize, Level: 1})
	c, stop := testClient(t, srv, p)
	defer stop()
	sword := AddItem(p, 69, 1, srv.nextItemID)
	w := packet.NewWriter()
	w.WriteC(0x72)
	w.WriteD(sword.ObjectID)
	w.WriteD(1)
	r := packet.NewReader(w.Bytes())
	r.SkipOpcode()
	srv.onCrystallizeItem(c, r)
	if FindItemByID(p, 69) != nil {
		t.Fatal("weapon should have been crystallized")
	}
	if ItemCountOf(p, 1458) < 1 {
		t.Fatal("expected D-grade crystals 1458")
	}
}

func TestMultisellNewbieExchange(t *testing.T) {
	list := GetMultisell(3)
	if list == nil || len(list.Entries) == 0 {
		t.Fatal("multisell 003 missing")
	}
	entry := list.Entries[0]
	srv := combatTestServer(t)
	p := DefaultCharacter("acc", "Trader", 0, 0, 0, 1, 1, 1, 268441004, nil)
	c, stop := testClient(t, srv, p)
	defer stop()
	npc := &NPC{ObjectID: srv.world.NextID(), NPCID: 30001, Name: "Lector",
		X: p.X, Y: p.Y, Z: p.Z, MaxHP: 100, CurHP: 100}
	srv.world.AddNPC(npc)
	c.target = npc.ObjectID
	c.multiSellID = list.ID
	for _, ing := range entry.Ingredients {
		AddItem(p, ing.ItemID, ing.Count, srv.nextItemID)
	}
	w := packet.NewWriter()
	w.WriteC(0xA7)
	w.WriteD(list.ID)
	w.WriteD(1)
	w.WriteD(1)
	r := packet.NewReader(w.Bytes())
	r.SkipOpcode()
	srv.onMultiSellChoose(c, r)
	if FindItemByID(p, entry.Products[0].ItemID) == nil {
		t.Fatalf("expected product %d", entry.Products[0].ItemID)
	}
	for _, ing := range entry.Ingredients {
		if ItemCountOf(p, ing.ItemID) != 0 && ing.ItemID != AdenaID {
			t.Fatalf("ingredient %d should have been consumed", ing.ItemID)
		}
	}
}

func TestRecipeLearnAndCraftWoodenArrow(t *testing.T) {
	rec := GetRecipe(1)
	if rec == nil {
		t.Fatal("recipe 1 missing")
	}
	srv := combatTestServer(t)
	p := DefaultCharacter("acc", "Crafter", 0, 0, 0, 1, 1, 1, 268441005, nil)
	p.Skills = append(p.Skills, Skill{ID: skillCreateDwarven, Level: 1})
	p.CurMP = 100
	p.MaxMP = 100
	c, stop := testClient(t, srv, p)
	defer stop()
	book := AddItem(p, rec.ItemID, 1, srv.nextItemID)
	srv.learnRecipe(c, book)
	if !HasRecipe(p, rec.ID) {
		t.Fatal("recipe should have been registered")
	}
	if FindItemByID(p, rec.ItemID) != nil {
		t.Fatal("recipe item should have been consumed")
	}
	for _, mat := range rec.Materials {
		AddItem(p, mat.ItemID, mat.Count, srv.nextItemID)
	}
	w := packet.NewWriter()
	w.WriteC(0xAF)
	w.WriteD(rec.ID)
	r := packet.NewReader(w.Bytes())
	r.SkipOpcode()
	srv.onRecipeItemMakeSelf(c, r)
	if ItemCountOf(p, rec.ProductID) != rec.ProductCnt {
		t.Fatalf("product count = %d, want %d", ItemCountOf(p, rec.ProductID), rec.ProductCnt)
	}
}

func TestHennaEquipAndUnequip(t *testing.T) {
	h := GetHenna(1)
	if h == nil {
		t.Fatal("henna 1 missing")
	}
	srv := combatTestServer(t)
	p := DefaultCharacter("acc", "Dyed", 0, 0, 0, 1, 1, 1, 268441006, nil)
	p.ClassID = 1
	c, stop := testClient(t, srv, p)
	defer stop()
	AddItem(p, h.DyeID, hennaDrawAmount, srv.nextItemID)
	AddAdena(p, h.Price, srv.nextItemID)
	RecalcStats(p)
	baseSTR := p.STR
	w := packet.NewWriter()
	w.WriteC(0xBC)
	w.WriteD(h.SymbolID)
	r := packet.NewReader(w.Bytes())
	r.SkipOpcode()
	srv.onHennaEquip(c, r)
	if p.Hennas[0] != h.SymbolID {
		t.Fatalf("henna slot = %v, want %d", p.Hennas, h.SymbolID)
	}
	RecalcStats(p)
	if p.STR != baseSTR+h.STR {
		t.Fatalf("STR = %d, want %d", p.STR, baseSTR+h.STR)
	}
	AddAdena(p, h.Price/hennaRemoveAmount, srv.nextItemID)
	w = packet.NewWriter()
	w.WriteC(0xBF)
	w.WriteD(h.SymbolID)
	r = packet.NewReader(w.Bytes())
	r.SkipOpcode()
	srv.onHennaUnequip(c, r)
	if p.Hennas[0] != 0 {
		t.Fatal("henna should have been removed")
	}
	if ItemCountOf(p, h.DyeID) != hennaRemoveAmount {
		t.Fatalf("returned dyes = %d, want %d", ItemCountOf(p, h.DyeID), hennaRemoveAmount)
	}
}

func TestEnchantChanceSafeAndArmorCurve(t *testing.T) {
	scroll := enchantScroll{weapon: true, grade: "d"}
	item := &Item{ItemID: 69, Enchant: 0, Loc: "INVENTORY"}
	if scroll.chance(item) != 1 {
		t.Fatalf("safe enchant chance = %v, want 1", scroll.chance(item))
	}
	armorScroll := enchantScroll{weapon: false, grade: "d"}
	// Need a D-grade armor. Skip if missing.
	var armorID int32
	for _, id := range []int32{348, 24, 27} {
		if tpl := GetItem(id); tpl != nil && tpl.Type2 == Type2ShieldArmor && tpl.CrystalType == "D" {
			armorID = id
			break
		}
	}
	if armorID == 0 {
		t.Skip("no D-grade armor in XML")
	}
	armor := &Item{ItemID: armorID, Enchant: 4, Loc: "INVENTORY"}
	got := armorScroll.chance(armor)
	want := enchantChanceArmor * enchantChanceArmor // 0.66^(4-2)
	if got < want-0.001 || got > want+0.001 {
		t.Fatalf("armor +4 chance = %v, want %v", got, want)
	}
}
