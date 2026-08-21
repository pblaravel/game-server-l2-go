package gameserver

import (
	"math"
	"math/rand"
	"strings"
	"time"
)

// Melee combat ported from Java skills/Formulas.java and the attack pipeline of
// model/actor/attack/CreatureAttack.

// RespawnHPPercent is Java Config.RESPAWN_RESTORE_HP.
const RespawnHPPercent = 0.7

// Attack packet flags from Java combat/Attack.Hit.
const (
	hitFlagMiss     byte = 0x80
	hitFlagShield   byte = 0x40
	hitFlagCritical byte = 0x20
	hitFlagUseSS    byte = 0x10
)

// ShieldDefense values of Java enums.ShieldDefense.
const (
	shieldFailed = iota
	shieldSuccess
	shieldPerfect
)

// TimeBetweenAttacks is Java Formulas.calculateTimeBetweenAttacks.
func TimeBetweenAttacks(pAtkSpd int32) time.Duration {
	if pAtkSpd <= 0 {
		pAtkSpd = 1
	}
	ms := 500000 / int(pAtkSpd)
	if ms < 100 {
		ms = 100
	}
	return time.Duration(ms) * time.Millisecond
}

// CalcAtkSpd is Java Formulas.calcAtkSpd for skill cast times.
func CalcAtkSpd(skillTime float64, magic bool, pAtkSpd, mAtkSpd int32) int32 {
	spd := pAtkSpd
	if magic {
		spd = mAtkSpd
	}
	if spd <= 0 {
		spd = 1
	}
	return int32(skillTime * 333 / float64(spd))
}

// CalcHitMiss is Java Formulas.calcHitMiss (weather and position bonuses included).
func CalcHitMiss(accuracy, evasion, attackerZ, targetZ int32, behind, inFront bool) bool {
	diff := accuracy - evasion
	diffZ := attackerZ - targetZ
	if diffZ > 50 {
		diff += 3
	} else if diffZ < -50 {
		diff -= 3
	}
	if behind {
		diff += 10
	} else if !inFront {
		diff += 5
	}
	chance := (90 + 2*diff) * 10
	if chance < 300 {
		chance = 300
	}
	if chance > 980 {
		chance = 980
	}
	return int32(rand.Intn(1000)) >= chance
}

// CalcCrit is Java Formulas.calcCrit: the rate is out of 1000.
func CalcCrit(rate int32) bool { return int(rate) > rand.Intn(1000) }

// PosMul is Java Formulas.getPosMul.
func PosMul(behind, inFront, crit bool) float64 {
	if behind {
		if crit {
			return 1.1
		}
		return 1.2
	}
	if !inFront {
		if crit {
			return 1.025
		}
		return 1.05
	}
	return 1
}

// RandomDamageMultiplier is Java Creature.getRandomDamageMultiplier (weapon
// random damage, 10% for the starter weapons).
func RandomDamageMultiplier(randomDamage int32) float64 {
	if randomDamage <= 0 {
		return 1
	}
	return 1 + float64(rand.Intn(int(2*randomDamage+1))-int(randomDamage))/100
}

// CalcPhysicalAttackDamage is Java Formulas.calcPhysicalAttackDamage. Elemental,
// race and PvP modifiers stay at 1 because their data sources are not ported.
func CalcPhysicalAttackDamage(pAtk, pDef int32, shield int, crit, ss bool, posMul, rndMul float64) float64 {
	if shield == shieldPerfect {
		return 1
	}
	defence := float64(pDef)
	if shield == shieldSuccess {
		defence += 0 // no shield item data, Java adds target.getShldDef()
	}
	if defence <= 0 {
		defence = 1
	}
	attackPower := float64(pAtk)
	var damage float64
	if crit {
		damage = attackPower * 2 * posMul * rndMul * 77 / defence
	} else {
		damage = attackPower * posMul * rndMul * 77 / defence
	}
	if ss {
		damage *= 2
	}
	if damage < 1 {
		damage = 1
	}
	return damage
}

// CalcMagicDamage is Java Formulas.calcMagicDam without shot and element modifiers.
func CalcMagicDamage(mAtk, mDef int32, power float64, mcrit, ss bool) float64 {
	defence := float64(mDef)
	if defence <= 0 {
		defence = 1
	}
	damage := 91 * math.Sqrt(float64(mAtk)) / defence * power
	if mcrit {
		damage *= 4
	}
	if ss {
		damage *= 2
	}
	if damage < 1 {
		damage = 1
	}
	return damage
}

// angleFrom / headingDegrees / isBehind / isInFrontOf are Java MathUtil and
// WorldObject facing helpers.
func angleFrom(x1, y1, x2, y2 int32) float64 {
	angle := math.Atan2(float64(y2-y1), float64(x2-x1)) * 180 / math.Pi
	if angle < 0 {
		angle += 360
	}
	return angle
}

func headingDegrees(heading int32) float64 {
	return float64(heading) / 182.04444444444444
}

func facing(attackerX, attackerY int32, targetX, targetY, targetHeading int32, maxAngleDiff float64) bool {
	angleChar := angleFrom(attackerX, attackerY, targetX, targetY)
	angleTarget := headingDegrees(targetHeading)
	diff := angleChar - angleTarget
	if diff <= -360+maxAngleDiff {
		diff += 360
	}
	if diff >= 360-maxAngleDiff {
		diff -= 360
	}
	return math.Abs(diff) <= maxAngleDiff
}

// IsBehind is Java WorldObject.isBehind (60 degree cone behind the target).
func IsBehind(attackerX, attackerY, targetX, targetY, targetHeading int32) bool {
	return facing(attackerX, attackerY, targetX, targetY, targetHeading, 60)
}

// IsInFrontOf is Java WorldObject.isInFrontOf.
func IsInFrontOf(attackerX, attackerY, targetX, targetY, targetHeading int32) bool {
	return facing(attackerX, attackerY, targetX, targetY, targetHeading+32768, 60)
}

// startAttack is Java Player.getAI().tryToAttack: range and state are checked,
// then hits are scheduled at the attack speed interval.
func (s *Server) startAttack(c *GameClient, npc *NPC) {
	p := c.Player()
	if p == nil || npc == nil {
		return
	}
	if npc.AlikeDead() {
		c.Send(SystemMessage(SMInvalidTarget))
		c.Send(ActionFailed())
		return
	}
	dist := Distance3D(p.X, p.Y, p.Z, npc.X, npc.Y, npc.Z)
	if dist > float64(p.AttackRange)+p.CollisionRadius+npc.CollisionRadius {
		c.Send(SystemMessage(SMTargetTooFar))
		c.Send(ActionFailed())
		return
	}
	// Java Attackable.addDamageHate makes the monster retaliate.
	s.engageNPC(npc, c)
	if c.attacking.Swap(true) {
		return
	}
	go s.attackLoop(c, npc)
}

func (s *Server) attackLoop(c *GameClient, npc *NPC) {
	defer c.attacking.Store(false)
	p := c.Player()
	for {
		if p == nil || c.Closed() || p.AlikeDead() || npc.AlikeDead() || c.target != npc.ObjectID {
			p.InCombat = false
			return
		}
		if Distance3D(p.X, p.Y, p.Z, npc.X, npc.Y, npc.Z) > float64(p.AttackRange)+p.CollisionRadius+npc.CollisionRadius+40 {
			c.Send(ActionFailed())
			p.InCombat = false
			return
		}
		s.doAttackHit(c, npc)
		if npc.AlikeDead() {
			return
		}
		time.Sleep(TimeBetweenAttacks(p.PAtkSpd))
	}
}

// doAttackHit is one iteration of Java CreatureAttack.doAttack for a melee weapon.
func (s *Server) doAttackHit(c *GameClient, npc *NPC) {
	p := c.Player()
	p.InCombat = true
	npc.InCombat = true
	c.lastHit = time.Now()

	behind := IsBehind(p.X, p.Y, npc.X, npc.Y, npc.Heading)
	inFront := IsInFrontOf(p.X, p.Y, npc.X, npc.Y, npc.Heading)

	miss := CalcHitMiss(p.Accuracy, npcEvasion(npc), p.Z, npc.Z, behind, inFront)
	var flags byte
	var damage int32
	if miss {
		flags |= hitFlagMiss
	} else {
		crit := CalcCrit(p.Crit)
		if crit {
			flags |= hitFlagCritical
		}
		dmg := CalcPhysicalAttackDamage(p.PAtk, npc.PDef, shieldFailed, crit,
			false, PosMul(behind, inFront, crit), RandomDamageMultiplier(10))
		damage = int32(dmg)
	}

	pkt := Attack(p.ObjectID, npc.ObjectID, damage, flags, p.X, p.Y, p.Z)
	c.Send(pkt)
	c.Broadcast(pkt)

	if miss || damage <= 0 {
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

func npcEvasion(n *NPC) int32 {
	return int32(EvasionAccuracyBase(30) + float64(n.Level))
}

// killNPC is Java Attackable.doDie: rewards, death broadcast and respawn schedule.
func (s *Server) killNPC(c *GameClient, npc *NPC) {
	npc.Dead = true
	npc.InCombat = false
	pkt := Die(npc.ObjectID, false, false, false, false, false)
	c.Send(pkt)
	c.Broadcast(pkt)
	s.rewardKill(c, npc)
	c.logChange("killed npc=%d (%s)", npc.ObjectID, npc.Name)
	go s.scheduleRespawn(npc)
}

// rewardKill is Java Attackable.calculateRewards: exp/SP, optional party split, drops.
func (s *Server) rewardKill(c *GameClient, npc *NPC) {
	p := c.Player()
	exp, sp := npcReward(npc, p.Level)
	s.giveKillRewards(c, p, npc, exp, sp)
	s.rollDrops(c, npc)
}

func (s *Server) giveKillRewards(c *GameClient, p *Character, npc *NPC, exp int64, sp int32) {
	members := []*Character{p}
	if pt := s.partyOf(p); pt != nil {
		members = members[:0]
		for _, m := range s.partyMembers(pt) {
			if Distance2D(m.X, m.Y, npc.X, npc.Y) <= 1500 {
				members = append(members, m)
			}
		}
		if len(members) == 0 {
			members = []*Character{p}
		}
	}
	shareExp := exp / int64(len(members))
	shareSP := sp / int32(len(members))
	if shareExp <= 0 && shareSP <= 0 {
		return
	}
	for _, m := range members {
		m.Exp += shareExp
		m.SP += shareSP
		mc := s.clientOf(m.ObjectID)
		if mc == nil {
			continue
		}
		mc.Send(SystemMessage(SMEarnedExpAndSP, SysNumber(int32(shareExp)), SysNumber(shareSP)))
		if s.checkLevelUp(mc) {
			continue
		}
		mc.Send(UserInfo(m))
		_ = s.store.Update(mc.ctx(), m)
	}
}

// rollDrops is Java Attackable.doItemDrop without spoil: one roll per category.
func (s *Server) rollDrops(c *GameClient, npc *NPC) {
	tpl := GetNpcTemplate(npc.NPCID)
	if tpl == nil {
		return
	}
	p := c.Player()
	dropped := false
	for _, cat := range tpl.Drops {
		if strings.EqualFold(cat.Type, "SPOIL") {
			continue
		}
		if cat.Chance <= 0 || rndDouble()*100 > cat.Chance {
			continue
		}
		drop := pickDrop(cat.Drops)
		if drop == nil {
			continue
		}
		cnt := drop.Min
		if drop.Max > drop.Min {
			cnt = drop.Min + int32(rndDouble()*float64(drop.Max-drop.Min+1))
		}
		if cnt <= 0 {
			continue
		}
		AddItem(p, drop.ItemID, cnt, s.nextItemID)
		if cnt == 1 {
			c.Send(SystemMessage(SMPickedUpS1, SysItem(drop.ItemID)))
		} else {
			c.Send(SystemMessage(SMPickedUpS2S1, SysItemCount(cnt), SysItem(drop.ItemID)))
		}
		dropped = true
	}
	if dropped {
		c.Send(ItemList(p.Items, false))
		s.sendWeightAndStats(c)
	}
}

func pickDrop(drops []DropData) *DropData {
	if len(drops) == 0 {
		return nil
	}
	total := 0.0
	for _, d := range drops {
		total += d.Chance
	}
	if total <= 0 {
		return &drops[0]
	}
	roll := rndDouble() * total
	acc := 0.0
	for i := range drops {
		acc += drops[i].Chance
		if roll <= acc {
			return &drops[i]
		}
	}
	return &drops[len(drops)-1]
}

// npcReward is Java Monster.calculateExpAndSp: above a five level gap the reward
// decays by pow(5/6, diff - 5), and no XP means no SP either.
func npcReward(npc *NPC, playerLevel int32) (int64, int32) {
	exp, sp := float64(npc.Exp), float64(npc.SP)
	if exp == 0 && sp == 0 {
		// NpcData XML is not vendored; derive the Interlude ballpark from level.
		exp = float64(npc.Level) * float64(npc.Level) * 8
		sp = float64(npc.Level) * float64(npc.Level) / 2
	}
	if diff := playerLevel - npc.Level; diff > 5 {
		pow := math.Pow(5.0/6.0, float64(diff-5))
		exp *= pow
		sp *= pow
	}
	if exp <= 0 {
		return 0, 0
	}
	if sp < 0 {
		sp = 0
	}
	return int64(exp), int32(sp)
}

// checkLevelUp is Java Player.addExpAndSp level handling.
func (s *Server) checkLevelUp(c *GameClient) bool {
	p := c.Player()
	levelled := false
	for p.Level < int32(len(PlayerLevelExp)-1) && p.Exp >= ExpForLevel(int(p.Level)+1) {
		p.Level++
		levelled = true
	}
	if !levelled {
		return false
	}
	RecalcStats(p)
	p.CurHP = float64(p.MaxHP)
	p.CurMP = float64(p.MaxMP)
	p.CurCP = float64(p.MaxCP)
	learned := AutoLearnOnLevelUp(p)
	c.Send(SystemMessage(SMLevelIncreased))
	c.Send(SocialAction(p.ObjectID, 15)) // Java LEVEL_UP social action
	c.Broadcast(SocialAction(p.ObjectID, 15))
	c.Send(UserInfo(p))
	if len(learned) > 0 {
		c.Send(SkillList(p.Skills))
	}
	c.Broadcast(CharInfo(p))
	_ = s.store.Update(c.ctx(), p)
	c.logChange("level up to %d hp=%d mp=%d new_skills=%d", p.Level, p.MaxHP, p.MaxMP, len(learned))
	return true
}

// scheduleRespawn is Java Spawn.doRespawn.
func (s *Server) scheduleRespawn(npc *NPC) {
	time.Sleep(NPCRespawnDelay)
	npc.CurHP = npc.MaxHP
	npc.CurMP = npc.MaxMP
	npc.Dead = false
	npc.X, npc.Y, npc.Z = npc.SpawnX, npc.SpawnY, npc.SpawnZ
	s.Broadcast(NpcInfo(npc), nil)
}

// NPCRespawnDelay stands in for the per-spawn respawnDelay of the spawn XML.
const NPCRespawnDelay = 60 * time.Second

// killPlayer is Java Player.doDie.
func (s *Server) killPlayer(c *GameClient, killerName string) {
	p := c.Player()
	p.Dead = true
	p.CurHP = 0
	p.InCombat = false
	c.target = 0
	pkt := Die(p.ObjectID, false, false, false, false, false)
	c.Send(pkt)
	c.Broadcast(pkt)
	c.logChange("died to %s at (%d,%d,%d)", killerName, p.X, p.Y, p.Z)
	_ = s.store.Update(c.ctx(), p)
}

// revive is Java Creature.doRevive with the configured HP/MP/CP restore.
func (s *Server) revive(c *GameClient, percent float64) {
	p := c.Player()
	p.Dead = false
	p.CurHP = float64(p.MaxHP) * percent
	p.CurMP = float64(p.MaxMP) * percent
	p.CurCP = float64(p.MaxCP) * percent
	c.Send(Revive(p.ObjectID))
	c.Broadcast(Revive(p.ObjectID))
	c.Send(StatusUpdate(p.ObjectID, [][2]int32{
		{StatusCurHP, int32(p.CurHP)}, {StatusMaxHP, p.MaxHP},
		{StatusCurMP, int32(p.CurMP)}, {StatusMaxMP, p.MaxMP},
		{StatusCurCP, int32(p.CurCP)}, {StatusMaxCP, p.MaxCP},
	}))
	c.Send(UserInfo(p))
	_ = s.store.Update(c.ctx(), p)
}
