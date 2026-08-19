package gameserver

import (
	"math"
	"testing"
)

// Expected values come from the Java formulas in skills/Formulas.java:
// bonus[i] = floor(pow(base, i - shift) * 100 + 0.5) / 100.
func TestStatBonusTablesMatchJava(t *testing.T) {
	cases := []struct {
		name  string
		got   float64
		base  float64
		shift float64
		stat  int32
	}{
		{"STR", STRBonus(40), 1.036, 34.845, 40},
		{"INT", INTBonus(21), 1.020, 31.375, 21},
		{"DEX", DEXBonus(30), 1.009, 19.360, 30},
		{"WIT", WITBonus(11), 1.050, 20.000, 11},
		{"CON", CONBonus(43), 1.030, 27.632, 43},
		{"MEN", MENBonus(25), 1.010, -0.060, 25},
	}
	for _, c := range cases {
		want := math.Floor(math.Pow(c.base, float64(c.stat)-c.shift)*100+0.5) / 100
		if c.got != want {
			t.Errorf("%sBonus(%d) = %v, want %v", c.name, c.stat, c.got, want)
		}
	}
}

func TestLevelModMatchesJava(t *testing.T) {
	// Java CreatureStatus.getLevelMod: (100 - 11 + level) / 100.
	for _, level := range []int32{1, 20, 80} {
		want := (100.0 - 11 + float64(level)) / 100.0
		if got := LevelMod(level); got != want {
			t.Errorf("LevelMod(%d) = %v, want %v", level, got, want)
		}
	}
}

func TestEvasionAccuracyBaseMatchesJava(t *testing.T) {
	// Java Formulas.BASE_EVASION_ACCURACY[i] = sqrt(i) * 6.
	for _, dex := range []int32{0, 30, 99} {
		want := math.Sqrt(float64(dex)) * 6
		if got := EvasionAccuracyBase(dex); got != want {
			t.Errorf("EvasionAccuracyBase(%d) = %v, want %v", dex, got, want)
		}
	}
}

func TestRecalcStatsUsesClassTemplate(t *testing.T) {
	if err := LoadDatapack(FindDataDir()); err != nil {
		t.Skipf("datapack unavailable: %v", err)
	}
	tpl := GetClassTemplate(0) // Human Fighter
	if tpl == nil || len(tpl.HPTable) == 0 {
		t.Skip("human fighter template has no stat tables")
	}
	p := DefaultCharacter("acc", "Fighter", 0, 0, 0, 1, 1, 1, 268437457, nil)
	p.Level = 10
	RecalcStats(p)

	wantHP := int32(tpl.MaxHPAt(10) * CONBonus(p.CON))
	if p.MaxHP != wantHP {
		t.Errorf("MaxHP = %d, want %d (hpTable[10]=%v * CON bonus)", p.MaxHP, wantHP, tpl.MaxHPAt(10))
	}
	wantAcc := int32(EvasionAccuracyBase(p.DEX) + 10)
	if p.Accuracy != wantAcc {
		t.Errorf("Accuracy = %d, want %d", p.Accuracy, wantAcc)
	}
	if p.WeightLimit != int32(69000*CONBonus(p.CON)) {
		t.Errorf("WeightLimit = %d, want %d", p.WeightLimit, int32(69000*CONBonus(p.CON)))
	}
	// The equipped squire's sword must raise attack over the bare template value.
	bare := int32(float64(tpl.PAtk) * STRBonus(p.STR) * LevelMod(10))
	if p.PAtk <= bare {
		t.Errorf("PAtk = %d, expected more than the unarmed %d", p.PAtk, bare)
	}
}

func TestTimeBetweenAttacksMatchesJava(t *testing.T) {
	// Java Formulas.calculateTimeBetweenAttacks: max(100, 500000 / pAtkSpd).
	if got := TimeBetweenAttacks(300).Milliseconds(); got != 1666 {
		t.Errorf("TimeBetweenAttacks(300) = %dms, want 1666ms", got)
	}
	if got := TimeBetweenAttacks(10000).Milliseconds(); got != 100 {
		t.Errorf("TimeBetweenAttacks(10000) = %dms, want the 100ms floor", got)
	}
}

func TestCalcAtkSpdMatchesJava(t *testing.T) {
	// Java Formulas.calcAtkSpd: skillTime * 333 / (m)AtkSpd.
	if got := CalcAtkSpd(1500, true, 300, 333); got != 1500 {
		t.Errorf("magic cast time = %d, want 1500", got)
	}
	if got := CalcAtkSpd(1500, false, 333, 300); got != 1500 {
		t.Errorf("physical cast time = %d, want 1500", got)
	}
}

func TestPosMulMatchesJava(t *testing.T) {
	// Java Formulas.getPosMul.
	cases := []struct {
		behind, inFront, crit bool
		want                  float64
	}{
		{true, false, false, 1.2},
		{true, false, true, 1.1},
		{false, false, false, 1.05},
		{false, false, true, 1.025},
		{false, true, false, 1.0},
	}
	for _, c := range cases {
		if got := PosMul(c.behind, c.inFront, c.crit); got != c.want {
			t.Errorf("PosMul(behind=%v, front=%v, crit=%v) = %v, want %v",
				c.behind, c.inFront, c.crit, got, c.want)
		}
	}
}

func TestCalcPhysicalAttackDamageMatchesJava(t *testing.T) {
	// Java: damage = attack * posMul * rnd * 77 / defence, doubled on crit and ss.
	base := CalcPhysicalAttackDamage(100, 77, shieldFailed, false, false, 1, 1)
	if base != 100 {
		t.Errorf("damage = %v, want 100 (pAtk 100 vs pDef 77)", base)
	}
	crit := CalcPhysicalAttackDamage(100, 77, shieldFailed, true, false, 1, 1)
	if crit != 200 {
		t.Errorf("critical damage = %v, want 200", crit)
	}
	ss := CalcPhysicalAttackDamage(100, 77, shieldFailed, false, true, 1, 1)
	if ss != 200 {
		t.Errorf("soulshot damage = %v, want 200", ss)
	}
	if perfect := CalcPhysicalAttackDamage(100, 77, shieldPerfect, false, false, 1, 1); perfect != 1 {
		t.Errorf("perfect shield block = %v, want 1", perfect)
	}
}

func TestFacingHelpersMatchJava(t *testing.T) {
	// A target looking towards +X (heading 0) is attacked from behind at -X.
	if !IsBehind(-100, 0, 0, 0, 0) {
		t.Error("attacker at -X should be behind a target with heading 0")
	}
	if IsBehind(100, 0, 0, 0, 0) {
		t.Error("attacker at +X should not be behind a target with heading 0")
	}
	if !IsInFrontOf(100, 0, 0, 0, 0) {
		t.Error("attacker at +X should be in front of a target with heading 0")
	}
}

func TestEffectStackingFollowsJavaEffectList(t *testing.T) {
	p := DefaultCharacter("acc", "Buffed", 0, 0, 0, 1, 1, 1, 268437458, nil)
	RecalcStats(p)
	basePAtk := p.PAtk

	weak := &SkillTemplate{ID: 1001, Level: 1, SkillType: "BUFF", OperateType: "ACTIVE",
		Effects: []EffectTemplate{{Name: "Buff", Duration: 60, StackType: "pa_up", StackOrder: 1,
			Funcs: []FuncTemplate{{Op: "mul", Stat: "pAtk", Value: 1.1}}}}}
	strong := &SkillTemplate{ID: 1002, Level: 1, SkillType: "BUFF", OperateType: "ACTIVE",
		Effects: []EffectTemplate{{Name: "Buff", Duration: 60, StackType: "pa_up", StackOrder: 3,
			Funcs: []FuncTemplate{{Op: "mul", Stat: "pAtk", Value: 1.5}}}}}

	AddEffects(p, weak)
	if len(p.Effects) != 1 {
		t.Fatalf("effects after first buff = %d, want 1", len(p.Effects))
	}
	if p.PAtk <= basePAtk {
		t.Errorf("PAtk = %d, expected the buff to raise it above %d", p.PAtk, basePAtk)
	}

	// Higher stack order replaces the same stack type instead of adding a second entry.
	AddEffects(p, strong)
	if len(p.Effects) != 1 {
		t.Fatalf("effects after the stronger buff = %d, want 1", len(p.Effects))
	}
	if p.Effects[0].SkillID != 1002 {
		t.Errorf("kept skill %d, want the higher stack order 1002", p.Effects[0].SkillID)
	}

	// A lower stack order must not push the stronger buff out.
	AddEffects(p, weak)
	if p.Effects[0].SkillID != 1002 {
		t.Errorf("weaker buff replaced the stronger one (skill %d)", p.Effects[0].SkillID)
	}
}

func TestLearnableSkillsFollowClassTree(t *testing.T) {
	if err := LoadDatapack(FindDataDir()); err != nil {
		t.Skipf("datapack unavailable: %v", err)
	}
	p := DefaultCharacter("acc", "Student", 0, 0, 0, 1, 1, 1, 268437459, nil)
	p.Level = 5
	nodes := LearnableSkills(p)
	if len(nodes) == 0 {
		t.Fatal("a level 5 human fighter should have learnable skills")
	}
	for _, n := range nodes {
		if n.MinLvl > p.Level {
			t.Errorf("skill %d level %d requires level %d", n.ID, n.Level, n.MinLvl)
		}
		if n.Cost <= 0 {
			t.Errorf("skill %d level %d has no SP cost, it should be auto-granted", n.ID, n.Level)
		}
	}
	// Learning the first node moves the tree on to the next level of that skill.
	first := nodes[0]
	AddOrUpgradeSkill(p, first)
	for _, n := range LearnableSkills(p) {
		if n.ID == first.ID && n.Level == first.Level {
			t.Errorf("skill %d level %d is still offered after being learned", n.ID, n.Level)
		}
	}
}

func TestInventoryEquipUnequip(t *testing.T) {
	p := DefaultCharacter("acc", "Equipper", 0, 0, 0, 1, 1, 1, 268437460, nil)
	weapon := FindItemByID(p, 2369)
	if weapon == nil {
		t.Skip("starter sword missing from the kit")
	}
	if !weapon.Equipped {
		t.Fatal("the starter sword should come equipped")
	}
	if p.PaperdollItem[PaperRHand] != weapon.ItemID {
		t.Errorf("paperdoll right hand = %d, want %d", p.PaperdollItem[PaperRHand], weapon.ItemID)
	}
	if !UnequipBodyPart(p, weapon.BodyPart) {
		t.Fatal("unequip failed")
	}
	if p.PaperdollItem[PaperRHand] != 0 || FindItemByID(p, 2369).Equipped {
		t.Error("the weapon is still equipped after unequip")
	}
	RecalcStats(p)
	unarmed := p.PAtk
	if !EquipItem(p, weapon.ObjectID) {
		t.Fatal("re-equip failed")
	}
	RecalcStats(p)
	if p.PAtk <= unarmed {
		t.Errorf("PAtk with weapon = %d, want more than unarmed %d", p.PAtk, unarmed)
	}
}
