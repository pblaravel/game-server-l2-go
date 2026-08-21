package gameserver

import (
	"math"
	"math/rand"
	"sort"
	"strings"
	"time"
)

// Skill execution ported from Java skills/L2Skill, skills/AbstractEffect and the
// cast pipeline of model/actor/cast/CreatureCast. Effects come from the <for>
// blocks of data/xml/skills.

// ActiveEffect is one entry of Java creature/EffectList.
type ActiveEffect struct {
	SkillID    int32
	SkillLevel int32
	Name       string
	StackType  string
	StackOrder int32
	Value      float64
	Funcs      []FuncTemplate
	Expires    time.Time
}

func (e ActiveEffect) SecondsLeft() int32 {
	d := time.Until(e.Expires)
	if d <= 0 {
		return 0
	}
	return int32(d / time.Second)
}

// AddEffects is Java EffectList.add: a new effect of the same stackType replaces
// the current one only when its stackOrder is at least as high.
func AddEffects(p *Character, tpl *SkillTemplate) []ActiveEffect {
	now := time.Now()
	var applied []ActiveEffect
	for _, e := range tpl.Effects {
		if e.Duration <= 0 {
			continue
		}
		eff := ActiveEffect{
			SkillID:    tpl.ID,
			SkillLevel: tpl.Level,
			Name:       e.Name,
			StackType:  e.StackType,
			StackOrder: e.StackOrder,
			Value:      e.Value,
			Funcs:      e.Funcs,
			Expires:    now.Add(time.Duration(e.Duration) * time.Second),
		}
		replaced := false
		for i := range p.Effects {
			same := p.Effects[i].SkillID == eff.SkillID
			sameStack := eff.StackType != "" && p.Effects[i].StackType == eff.StackType
			if !same && !sameStack {
				continue
			}
			if sameStack && p.Effects[i].StackOrder > eff.StackOrder {
				replaced = true
				break
			}
			p.Effects[i] = eff
			replaced = true
			break
		}
		if !replaced {
			p.Effects = append(p.Effects, eff)
		}
		applied = append(applied, eff)
	}
	if len(applied) > 0 {
		RecalcStats(p)
	}
	return applied
}

// PurgeExpiredEffects is the Java EffectList tick; it reports the effects that
// ran out so the caller can tell the client.
func PurgeExpiredEffects(p *Character) []ActiveEffect {
	if len(p.Effects) == 0 {
		return nil
	}
	now := time.Now()
	var expired []ActiveEffect
	kept := p.Effects[:0]
	for _, e := range p.Effects {
		if e.Expires.After(now) {
			kept = append(kept, e)
			continue
		}
		expired = append(expired, e)
	}
	p.Effects = kept
	if len(expired) > 0 {
		RecalcStats(p)
	}
	return expired
}

// activeFuncs collects the modifiers of buffs plus the passive skill funcs, the
// way Java Calculator stacks Func instances on a stat.
func activeFuncs(p *Character) []FuncTemplate {
	var out []FuncTemplate
	for _, sk := range p.Skills {
		if !sk.Passive {
			continue
		}
		if tpl := GetSkill(sk.ID, sk.Level); tpl != nil {
			out = append(out, tpl.Funcs...)
			for _, e := range tpl.Effects {
				out = append(out, e.Funcs...)
			}
		}
	}
	for _, e := range p.Effects {
		out = append(out, e.Funcs...)
	}
	return out
}

// applyStatFuncs is Java Calculator.calc: set, then add/sub, then mul.
func applyStatFuncs(p *Character) {
	funcs := activeFuncs(p)
	if len(funcs) == 0 {
		return
	}
	stats := map[string]*int32{
		"pAtk":      &p.PAtk,
		"pDef":      &p.PDef,
		"mAtk":      &p.MAtk,
		"mDef":      &p.MDef,
		"pAtkSpd":   &p.PAtkSpd,
		"mAtkSpd":   &p.MAtkSpd,
		"runSpd":    &p.RunSpeed,
		"accCombat": &p.Accuracy,
		"rEvas":     &p.Evasion,
		"rCrit":     &p.Crit,
		"maxHp":     &p.MaxHP,
		"maxMp":     &p.MaxMP,
		"maxCp":     &p.MaxCP,
	}
	ordered := append([]FuncTemplate(nil), funcs...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return funcOrder(ordered[i].Op) < funcOrder(ordered[j].Op)
	})
	for _, f := range ordered {
		dst, ok := stats[f.Stat]
		if !ok {
			continue
		}
		v := float64(*dst)
		switch f.Op {
		case "set":
			v = f.Value
		case "add":
			v += f.Value
		case "sub":
			v -= f.Value
		case "mul":
			v *= f.Value
		}
		if v < 0 {
			v = 0
		}
		*dst = int32(v)
	}
}

func funcOrder(op string) int {
	switch op {
	case "set":
		return 0
	case "add", "sub":
		return 1
	default:
		return 2
	}
}

// CalcHealAmount is Java Formulas.calcHealAmount without shot bonuses.
func CalcHealAmount(power float64, healEffectiveness float64) float64 {
	if healEffectiveness <= 0 {
		healEffectiveness = 1
	}
	return power * healEffectiveness
}

// castSkill is Java CreatureCast.doCast: validate, announce, wait the cast time
// and then apply the skill.
func (s *Server) castSkill(c *GameClient, skillID, level int32) {
	p := c.Player()
	tpl := GetSkill(skillID, level)
	if tpl == nil {
		c.Send(ActionFailed())
		return
	}
	if tpl.IsPassive() {
		c.Send(ActionFailed())
		return
	}
	if until, ok := c.reuse.Load(skillID); ok {
		if t, _ := until.(time.Time); time.Now().Before(t) {
			c.Send(ActionFailed())
			return
		}
	}
	if float64(tpl.MPConsume) > p.CurMP {
		c.Send(SystemMessage(SMNotEnoughMP))
		c.Send(ActionFailed())
		return
	}
	if c.casting.Swap(true) {
		c.Send(ActionFailed())
		return
	}

	target, npc := s.resolveSkillTarget(c, tpl)
	if target == 0 {
		c.casting.Store(false)
		c.Send(SystemMessage(SMInvalidTarget))
		c.Send(ActionFailed())
		return
	}
	if npc != nil && tpl.CastRange > 0 {
		if Distance3D(p.X, p.Y, p.Z, npc.X, npc.Y, npc.Z) > float64(tpl.CastRange)+p.CollisionRadius+npc.CollisionRadius {
			c.casting.Store(false)
			c.Send(SystemMessage(SMTargetTooFar))
			c.Send(ActionFailed())
			return
		}
	}

	hitTime := CalcAtkSpd(float64(tpl.HitTime), tpl.IsMagic, p.PAtkSpd, p.MAtkSpd)
	if hitTime <= 0 {
		hitTime = tpl.HitTime
	}
	reuse := tpl.ReuseDelay
	targetX, targetY, targetZ := p.X, p.Y, p.Z
	if npc != nil {
		targetX, targetY, targetZ = npc.X, npc.Y, npc.Z
	}

	p.CurMP -= float64(tpl.MPConsume)
	if p.CurMP < 0 {
		p.CurMP = 0
	}
	c.reuse.Store(skillID, time.Now().Add(time.Duration(reuse)*time.Millisecond))

	use := MagicSkillUse(p.ObjectID, target, skillID, level, hitTime, reuse,
		p.X, p.Y, p.Z, targetX, targetY, targetZ, true)
	c.Send(use)
	c.Broadcast(use)
	c.Send(SetupGauge(GaugeBlue, hitTime, hitTime))
	c.Send(StatusUpdate(p.ObjectID, [][2]int32{
		{StatusCurMP, int32(p.CurMP)}, {StatusMaxMP, p.MaxMP},
	}))
	c.logChange("cast skill=%d lvl=%d type=%s hit=%d reuse=%d target=%d", skillID, level, tpl.SkillType, hitTime, reuse, target)

	go func() {
		time.Sleep(time.Duration(hitTime) * time.Millisecond)
		defer c.casting.Store(false)
		if c.Closed() || p.AlikeDead() {
			return
		}
		launched := MagicSkillLaunched(p.ObjectID, skillID, level, []int32{target})
		c.Send(launched)
		c.Broadcast(launched)
		s.applySkill(c, tpl, npc)
	}()
}

func (s *Server) resolveSkillTarget(c *GameClient, tpl *SkillTemplate) (int32, *NPC) {
	p := c.Player()
	switch strings.ToUpper(tpl.TargetType) {
	case "SELF":
		return p.ObjectID, nil
	case "ONE", "AURA", "AREA", "FRONT_AREA", "BEHIND_AREA", "":
		if c.target != 0 {
			if npc := s.world.GetNPC(c.target); npc != nil && !npc.AlikeDead() {
				return npc.ObjectID, npc
			}
			if other := s.world.GetPlayer(c.target); other != nil {
				return other.ObjectID, nil
			}
		}
		if isOffensive(tpl) {
			return 0, nil
		}
		return p.ObjectID, nil
	default:
		return p.ObjectID, nil
	}
}

func isOffensive(tpl *SkillTemplate) bool {
	switch strings.ToUpper(tpl.SkillType) {
	case "PDAM", "MDAM", "BLOW", "DEBUFF", "DRAIN", "STUN", "ROOT", "SLEEP", "MANADAM", "CANCEL", "AGGDAMAGE":
		return true
	default:
		return false
	}
}

// applySkill is Java skill handler dispatch (handler/skillhandlers).
func (s *Server) applySkill(c *GameClient, tpl *SkillTemplate, npc *NPC) {
	p := c.Player()
	switch strings.ToUpper(tpl.SkillType) {
	case "HEAL", "HEAL_PERCENT", "HEAL_STATIC":
		amount := CalcHealAmount(tpl.Power, 1)
		if strings.EqualFold(tpl.SkillType, "HEAL_PERCENT") {
			amount = float64(p.MaxHP) * tpl.Power / 100
		}
		p.CurHP = math.Min(float64(p.MaxHP), p.CurHP+amount)
		c.Send(StatusUpdate(p.ObjectID, [][2]int32{
			{StatusCurHP, int32(p.CurHP)}, {StatusMaxHP, p.MaxHP},
		}))
		c.Send(SystemMessage(SMFeelEffect, SysSkill(tpl.ID, tpl.Level)))
	case "MANAHEAL", "MANARECHARGE", "MANAHEAL_PERCENT":
		amount := tpl.Power
		if strings.EqualFold(tpl.SkillType, "MANAHEAL_PERCENT") {
			amount = float64(p.MaxMP) * tpl.Power / 100
		}
		p.CurMP = math.Min(float64(p.MaxMP), p.CurMP+amount)
		c.Send(StatusUpdate(p.ObjectID, [][2]int32{
			{StatusCurMP, int32(p.CurMP)}, {StatusMaxMP, p.MaxMP},
		}))
	case "PDAM", "BLOW":
		if npc == nil {
			return
		}
		crit := CalcCrit(p.Crit)
		behind := IsBehind(p.X, p.Y, npc.X, npc.Y, npc.Heading)
		inFront := IsInFrontOf(p.X, p.Y, npc.X, npc.Y, npc.Heading)
		dmg := CalcPhysicalSkillDamage(p.PAtk, npc.PDef, tpl.Power, crit, false,
			PosMul(behind, inFront, crit), RandomDamageMultiplier(10))
		s.applySkillDamage(c, npc, int32(dmg), crit)
	case "MDAM", "DRAIN":
		if npc == nil {
			return
		}
		mcrit := CalcCrit(int32(baseMCritHit * 10))
		dmg := CalcMagicDamage(p.MAtk, npc.MDef, tpl.Power, mcrit, false)
		s.applySkillDamage(c, npc, int32(dmg), mcrit)
	case "RECALL":
		loc := NearestRestartLocation(p)
		s.teleportPlayer(c, loc[0], loc[1], loc[2])
	case "BUFF", "CONT", "HOT", "MPHOT", "CPHOT", "SELF_BUFF":
		applied := AddEffects(p, tpl)
		if len(applied) > 0 {
			c.Send(SystemMessage(SMFeelEffect, SysSkill(tpl.ID, tpl.Level)))
			c.Send(UserInfo(p))
			c.Broadcast(CharInfo(p))
		}
	case "DEBUFF", "STUN", "ROOT", "SLEEP", "MUTE", "PARALYZE":
		if npc == nil {
			return
		}
		// Java stores the effect on the target; NPC effect lists are not ported,
		// so only the client-visible result of the cast is sent.
		c.Send(SystemMessage(SMFeelEffect, SysSkill(tpl.ID, tpl.Level)))
	default:
		if len(tpl.Effects) > 0 {
			AddEffects(p, tpl)
			c.Send(UserInfo(p))
		}
	}
	_ = s.store.Update(c.ctx(), p)
}

func (s *Server) applySkillDamage(c *GameClient, npc *NPC, damage int32, crit bool) {
	if damage <= 0 {
		return
	}
	npc.CurHP -= damage
	if npc.CurHP < 0 {
		npc.CurHP = 0
	}
	c.Send(StatusUpdate(npc.ObjectID, [][2]int32{
		{StatusMaxHP, npc.MaxHP}, {StatusCurHP, npc.CurHP},
	}))
	if npc.CurHP == 0 {
		s.killNPC(c, npc)
	}
}

// CalcPhysicalSkillDamage is Java Formulas.calcPhysicalSkillDamage.
func CalcPhysicalSkillDamage(pAtk, pDef int32, power float64, crit, ss bool, posMul, rndMul float64) float64 {
	defence := float64(pDef)
	if defence <= 0 {
		defence = 1
	}
	damage := (float64(pAtk) + power) * 70 / defence * posMul * rndMul
	if crit {
		damage *= 2
	}
	if ss {
		damage *= 2
	}
	if damage < 1 {
		damage = 1
	}
	return damage
}

// rndDouble mirrors Java Rnd.get(double) usage in the effect handlers.
func rndDouble() float64 { return rand.Float64() }
