package gameserver

import "testing"

func TestRemainingDatapackLoaded(t *testing.T) {
	checks := []struct {
		name string
		got  int
		min  int
	}{
		{"armor sets", ArmorSetCount(), 20},
		{"spellbooks", SpellbookCount(), 100},
		{"heal sps", HealSpsCount(), 10},
		{"newbie buffs", NewbieBuffCount(), 10},
		{"recipes", RecipeCount(), 200},
		{"multisell", MultisellCount(), 50},
		{"hennas", HennaCount(), 50},
		{"zones", ZoneCount(), 50},
		{"doors", DoorCount(), 100},
		{"fishing skills", FishingSkillCount(), 20},
		{"clan skills", ClanSkillCount(), 10},
		{"enchant skills", EnchantSkillCount(), 100},
		{"instant teleports", InstantTeleportCount(), 10},
		{"summon items", SummonItemCount(), 5},
		{"fish", FishCount(), 50},
		{"soul crystals", SoulCrystalCount(), 20},
		{"cursed weapons", CursedWeaponCount(), 2},
		{"static objects", StaticObjectCount(), 10},
		{"walker routes", WalkerRouteCount(), 5},
		{"boats", BoatRouteCount(), 2},
		{"buffer skills", BufferSkillCount(), 10},
		{"access levels", AccessLevelCount(), 5},
		{"admin commands", AdminCommandCount(), 20},
		{"scripts", ScriptIndexCount(), 100},
		{"castles", CastleDataCount(), 9},
		{"clan halls", ClanHallDataCount(), 20},
		{"doors already counted", DoorCount(), 100},
		{"augmentation skills", AugmentSkillCount(), 100},
	}
	for _, c := range checks {
		if c.got < c.min {
			t.Errorf("%s = %d, want >= %d", c.name, c.got, c.min)
		}
	}
}

func TestSpellbookLookup(t *testing.T) {
	if BookForSkill(2, 1) != 1512 {
		t.Fatalf("Confusion book = %d", BookForSkill(2, 1))
	}
	if BookForSkill(2, 2) != 0 {
		t.Fatal("level 2 should not need a book")
	}
	if BookForSkill(1405, 1) != 8618 {
		t.Fatal("divine inspiration book")
	}
}

func TestArmorSetWooden(t *testing.T) {
	set := GetArmorSet(23)
	if set == nil || set.Legs != 2386 || set.SkillID != 3500 {
		t.Fatalf("wooden set = %#v", set)
	}
}

func TestHealSpsCorrection(t *testing.T) {
	tpl := &SkillTemplate{MagicLvl: 1}
	got := CalculateHealSps(tpl, 6)
	if got != 17 {
		t.Fatalf("heal sps magic 1 matk 6 = %v, want 17", got)
	}
	got = CalculateHealSps(tpl, 0)
	if got != 14 { // 17 - (6-0)/2
		t.Fatalf("heal sps low matk = %v, want 14", got)
	}
}

func TestRecipeWoodenArrow(t *testing.T) {
	r := GetRecipe(1)
	if r == nil || r.ProductID != 17 || r.ProductCnt != 500 || len(r.Materials) < 2 {
		t.Fatalf("recipe 1 = %#v", r)
	}
}

func TestTalkingIslandPeaceZone(t *testing.T) {
	if !InPeaceZone(-84000, 243000, -3700) {
		t.Fatal("Talking Island village should be a peace zone")
	}
	if InPeaceZone(0, 0, 0) {
		t.Fatal("origin should not be a peace zone")
	}
}

func TestFishingSkillTree(t *testing.T) {
	found := false
	for _, n := range GetFishingSkills() {
		if n.ID == 1312 && n.Level == 1 && n.ItemID == 57 {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("fishing skill 1312 missing")
	}
}

func TestInstantTeleportZiggurat(t *testing.T) {
	locs := InstantTeleports(31111)
	if len(locs) < 2 {
		t.Fatalf("ziggurat teleports = %d", len(locs))
	}
}

func TestIsDropableDefaults(t *testing.T) {
	if !IsDropable(AdenaID) {
		t.Fatal("adena should be dropable when no template override")
	}
}
