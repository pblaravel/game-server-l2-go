package gameserver

import "math"

// Stat computation ported from Java skills/Formulas.java and
// model/actor/status/{CreatureStatus,PlayerStatus}.java. Base values come from
// the class template (data/xml/classes) plus the equipped weapon and armour.

const maxStatValue = 100

// Base attack speed, magic attack speed and critical rate defaults of Java
// model/actor/template/CreatureTemplate.
const (
	basePAtkSpd  = 300.0
	baseMAtkSpd  = 333.0
	baseCritRate = 4.0
	baseMCritHit = 8.0
)

var (
	strBonus            [maxStatValue]float64
	intBonus            [maxStatValue]float64
	dexBonus            [maxStatValue]float64
	witBonus            [maxStatValue]float64
	conBonus            [maxStatValue]float64
	menBonus            [maxStatValue]float64
	baseEvasionAccuracy [maxStatValue]float64
)

func init() {
	// Java Formulas static block: floor(pow(base, i - shift) * 100 + 0.5) / 100.
	fill := func(dst *[maxStatValue]float64, base, shift float64) {
		for i := 0; i < maxStatValue; i++ {
			dst[i] = math.Floor(math.Pow(base, float64(i)-shift)*100+0.5) / 100
		}
	}
	fill(&strBonus, 1.036, 34.845)
	fill(&intBonus, 1.020, 31.375)
	fill(&dexBonus, 1.009, 19.360)
	fill(&witBonus, 1.050, 20.000)
	fill(&conBonus, 1.030, 27.632)
	fill(&menBonus, 1.010, -0.060)
	for i := 0; i < maxStatValue; i++ {
		baseEvasionAccuracy[i] = math.Sqrt(float64(i)) * 6
	}
}

func statIndex(v int32) int {
	if v < 0 {
		return 0
	}
	if int(v) >= maxStatValue {
		return maxStatValue - 1
	}
	return int(v)
}

func STRBonus(v int32) float64 { return strBonus[statIndex(v)] }
func INTBonus(v int32) float64 { return intBonus[statIndex(v)] }
func DEXBonus(v int32) float64 { return dexBonus[statIndex(v)] }
func WITBonus(v int32) float64 { return witBonus[statIndex(v)] }
func CONBonus(v int32) float64 { return conBonus[statIndex(v)] }
func MENBonus(v int32) float64 { return menBonus[statIndex(v)] }
func EvasionAccuracyBase(dex int32) float64 {
	return baseEvasionAccuracy[statIndex(dex)]
}

// LevelMod is Java CreatureStatus.getLevelMod.
func LevelMod(level int32) float64 { return (100.0 - 11 + float64(level)) / 100.0 }

// WeaponStats is the subset of ItemData that the stat engine needs. Java reads it
// from the item XML; only the starter equipment is known without that datapack.
type WeaponStats struct {
	PAtk       int32
	MAtk       int32
	PAtkSpd    int32
	CritRate   float64
	AttackRange int32
}

// ArmourStats is the paperdoll defence contribution of Java Inventory.
type ArmourStats struct {
	PDef int32
	MDef int32
}

var itemStats = map[int32]WeaponStats{
	// Squire's weapons and the apprentice wand, values from Interlude item data.
	2369: {PAtk: 8, MAtk: 6, PAtkSpd: 379, CritRate: 8, AttackRange: 40},  // Squire's Sword
	2370: {PAtk: 8, MAtk: 6, PAtkSpd: 379, CritRate: 8, AttackRange: 40},  // Squire's Sword (DE)
	2371: {PAtk: 8, MAtk: 6, PAtkSpd: 379, CritRate: 8, AttackRange: 40},  // Squire's Sword (Orc)
	2372: {PAtk: 8, MAtk: 6, PAtkSpd: 379, CritRate: 8, AttackRange: 40},  // Squire's Sword (Dwarf)
	99:   {PAtk: 4, MAtk: 8, PAtkSpd: 379, CritRate: 4, AttackRange: 40},  // Apprentice's Wand
	5588: {},
}

var armourStats = map[int32]ArmourStats{
	1146: {PDef: 22}, // Squire's Shirt
	425:  {PDef: 15}, // Apprentice's Tunic
	1147: {PDef: 14}, // Squire's Pants
	461:  {PDef: 10}, // Apprentice's Stockings
	1148: {PDef: 9},  // Squire's Shoes
}

// RecalcStats is Java Creature.getStatus() recalculation: every derived stat is
// rebuilt from the class template, the level and the equipped items.
func RecalcStats(p *Character) {
	tpl := GetClassTemplate(p.ClassID)
	level := p.Level
	if level <= 0 {
		level = 1
	}
	lvlMod := LevelMod(level)

	basePAtk, basePDef, baseMAtk, baseMDef := 4.0, 80.0, 6.0, 41.0
	baseRun, baseWalk, baseSwim := 115.0, 80.0, 50.0
	if tpl != nil {
		if tpl.PAtk > 0 {
			basePAtk = float64(tpl.PAtk)
		}
		if tpl.PDef > 0 {
			basePDef = float64(tpl.PDef)
		}
		if tpl.MAtk > 0 {
			baseMAtk = float64(tpl.MAtk)
		}
		if tpl.MDef > 0 {
			baseMDef = float64(tpl.MDef)
		}
		if tpl.RunSpd > 0 {
			baseRun = float64(tpl.RunSpd)
		}
		if tpl.WalkSpd > 0 {
			baseWalk = float64(tpl.WalkSpd)
		}
		if tpl.SwimSpd > 0 {
			baseSwim = float64(tpl.SwimSpd)
		}
		if tpl.STR > 0 {
			p.STR, p.CON, p.DEX = tpl.STR, tpl.CON, tpl.DEX
			p.INT, p.WIT, p.MEN = tpl.INT, tpl.WIT, tpl.MEN
		}
	}

	weapon, armour := equippedStats(p)
	atkSpd := basePAtkSpd
	critRate := baseCritRate
	p.AttackRange = 40
	if weapon.PAtkSpd > 0 {
		atkSpd = float64(weapon.PAtkSpd)
	}
	if weapon.CritRate > 0 {
		critRate = weapon.CritRate
	}
	if weapon.AttackRange > 0 {
		p.AttackRange = weapon.AttackRange
	}

	// Java FuncMaxHpMul / FuncMaxMpMul / FuncMaxCpMul.
	if tpl != nil && len(tpl.HPTable) > 0 {
		p.MaxHP = int32(tpl.MaxHPAt(level) * CONBonus(p.CON))
		p.MaxMP = int32(tpl.MaxMPAt(level) * MENBonus(p.MEN))
		p.MaxCP = int32(tpl.MaxCPAt(level) * CONBonus(p.CON))
	}
	if p.MaxHP <= 0 {
		p.MaxHP = 80
	}
	if p.MaxMP <= 0 {
		p.MaxMP = 30
	}

	// Java FuncPAtkMod / FuncPDefMod / FuncMAtkMod / FuncMDefMod.
	p.PAtk = int32((basePAtk + float64(weapon.PAtk)) * STRBonus(p.STR) * lvlMod)
	p.PDef = int32((basePDef + float64(armour.PDef)) * lvlMod)
	intMod := INTBonus(p.INT)
	p.MAtk = int32((baseMAtk + float64(weapon.MAtk)) * (lvlMod * lvlMod) * (intMod * intMod))
	p.MDef = int32((baseMDef + float64(armour.MDef)) * MENBonus(p.MEN) * lvlMod)

	// Java FuncPAtkSpeed / FuncMAtkSpeed / FuncAtkCritical / accuracy / evasion.
	p.PAtkSpd = int32(atkSpd * DEXBonus(p.DEX))
	p.MAtkSpd = int32(baseMAtkSpd * WITBonus(p.WIT))
	p.Crit = min32(int32(critRate*10), 500)
	p.Accuracy = int32(EvasionAccuracyBase(p.DEX) + float64(level))
	p.Evasion = int32(EvasionAccuracyBase(p.DEX) + float64(level))

	// Java FuncMoveSpeed keeps walk speed unscaled.
	p.RunSpeed = int32(baseRun * DEXBonus(p.DEX))
	p.WalkSpeed = int32(baseWalk)
	p.SwimSpeed = int32(baseSwim)

	// Java Player.getWeightLimit and Inventory.refreshWeight.
	p.WeightLimit = int32(69000 * CONBonus(p.CON))
	p.CurrentWeight = CurrentWeight(p)

	radius, height := CollisionFor(p.ClassID, p.Sex)
	p.CollisionRadius, p.CollisionHeight = radius, height

	// Buffs and passive skills modify the computed values, like Java Calculator.
	applyStatFuncs(p)

	if p.CurHP > float64(p.MaxHP) {
		p.CurHP = float64(p.MaxHP)
	}
	if p.CurMP > float64(p.MaxMP) {
		p.CurMP = float64(p.MaxMP)
	}
	if p.CurCP > float64(p.MaxCP) {
		p.CurCP = float64(p.MaxCP)
	}
}

func equippedStats(p *Character) (WeaponStats, ArmourStats) {
	var weapon WeaponStats
	var armour ArmourStats
	for _, it := range p.Items {
		if !it.Equipped {
			continue
		}
		if w, ok := itemStats[it.ItemID]; ok {
			weapon.PAtk += w.PAtk
			weapon.MAtk += w.MAtk
			if w.PAtkSpd > 0 {
				weapon.PAtkSpd = w.PAtkSpd
			}
			if w.CritRate > 0 {
				weapon.CritRate = w.CritRate
			}
			if w.AttackRange > 0 {
				weapon.AttackRange = w.AttackRange
			}
		}
		if a, ok := armourStats[it.ItemID]; ok {
			armour.PDef += a.PDef
			armour.MDef += a.MDef
		}
	}
	return weapon, armour
}

// HPRegen / MPRegen / CPRegen are Java FuncRegen*Mul over the template tables.
func HPRegen(p *Character) float64 {
	base := 1.5
	if tpl := GetClassTemplate(p.ClassID); tpl != nil && len(tpl.HPRegenTable) > 0 {
		base = tpl.HPRegenAt(p.Level)
	}
	return base * CONBonus(p.CON) * LevelMod(p.Level)
}

func MPRegen(p *Character) float64 {
	base := 0.9
	if tpl := GetClassTemplate(p.ClassID); tpl != nil && len(tpl.MPRegenTable) > 0 {
		base = tpl.MPRegenAt(p.Level)
	}
	return base * MENBonus(p.MEN) * LevelMod(p.Level)
}

func CPRegen(p *Character) float64 {
	base := 1.5
	if tpl := GetClassTemplate(p.ClassID); tpl != nil && len(tpl.CPRegenTable) > 0 {
		base = tpl.CPRegenAt(p.Level)
	}
	return base * CONBonus(p.CON) * LevelMod(p.Level)
}
