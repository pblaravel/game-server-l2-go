package gameserver

import (
	"sync"
	"time"
)

// Monster AI ported from Java model/actor/ai/type/AttackableAI: a monster that is
// attacked or that sees a player inside its aggro range chases and hits back.
// Java drives this from AiTaskManager; here each engaged monster owns a goroutine.

const (
	// Java NpcTemplate aggroRange for the newbie zone monsters.
	defaultAggroRange = 400
	// Java AttackableAI stops chasing past this distance from the spawn point.
	maxChaseDistance = 2000
	aggroScanPeriod  = 1 * time.Second
)

type aiState struct {
	mu      sync.Mutex
	engaged map[int32]bool // npc object id
}

func newAIState() *aiState { return &aiState{engaged: map[int32]bool{}} }

func (a *aiState) tryEngage(npcID int32) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.engaged[npcID] {
		return false
	}
	a.engaged[npcID] = true
	return true
}

func (a *aiState) release(npcID int32) {
	a.mu.Lock()
	delete(a.engaged, npcID)
	a.mu.Unlock()
}

// runAggroScan is the Java AttackableAI idle thinking: look for a target nearby.
func (s *Server) runAggroScan() {
	for _, npc := range s.world.NPCs() {
		if !npc.IsAttackable || npc.AlikeDead() || npc.AggroRange <= 0 {
			continue
		}
		if s.ai.engagedID(npc.ObjectID) {
			continue
		}
		for _, c := range s.inGameClients() {
			p := c.Player()
			if p == nil || p.AlikeDead() {
				continue
			}
			if Distance3D(p.X, p.Y, p.Z, npc.X, npc.Y, npc.Z) <= float64(npc.AggroRange) &&
				Geo().CanSeeWorld(npc.X, npc.Y, npc.Z, npc.CollisionHeight, p.X, p.Y, p.Z, p.CollisionHeight) {
				s.engageNPC(npc, c)
				break
			}
		}
	}
}

func (a *aiState) engagedID(npcID int32) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.engaged[npcID]
}

// engageNPC starts the attack routine of a monster against one player.
func (s *Server) engageNPC(npc *NPC, c *GameClient) {
	if !npc.IsAttackable || npc.AlikeDead() {
		return
	}
	if !s.ai.tryEngage(npc.ObjectID) {
		return
	}
	go s.npcCombatLoop(npc, c)
}

func (s *Server) npcCombatLoop(npc *NPC, c *GameClient) {
	defer s.ai.release(npc.ObjectID)
	for {
		p := c.Player()
		if p == nil || c.Closed() || npc.AlikeDead() || p.AlikeDead() {
			npc.InCombat = false
			s.broadcastNpcIdle(npc)
			return
		}
		if Distance3D(npc.SpawnX, npc.SpawnY, npc.SpawnZ, npc.X, npc.Y, npc.Z) > maxChaseDistance {
			s.returnNPCHome(npc)
			return
		}
		reach := float64(npc.AttackRange) + npc.CollisionRadius + p.CollisionRadius
		dist := Distance3D(p.X, p.Y, p.Z, npc.X, npc.Y, npc.Z)
		if dist > reach {
			if dist > maxChaseDistance {
				s.returnNPCHome(npc)
				return
			}
			s.moveNPCToward(npc, p)
			time.Sleep(500 * time.Millisecond)
			continue
		}
		s.npcAttack(npc, c)
		time.Sleep(TimeBetweenAttacks(npc.PAtkSpd))
	}
}

// moveNPCToward is the server side of Java Move.moveToLocation: the monster steps
// towards its target and the clients are told where it is heading.
func (s *Server) moveNPCToward(npc *NPC, p *Character) {
	npc.InCombat = true
	npc.Heading = headingTo(npc.X, npc.Y, p.X, p.Y)
	step := float64(npc.RunSpeed) * 0.5 // half a second of running
	if Distance2D(npc.X, npc.Y, p.X, p.Y) <= 0 {
		return
	}
	fromX, fromY, fromZ := npc.X, npc.Y, npc.Z
	dest := Geo().StepToward(npc.X, npc.Y, npc.Z, p.X, p.Y, p.Z)
	stepDist := Distance2D(npc.X, npc.Y, dest.X, dest.Y)
	if stepDist > 0 {
		stepRatio := step / stepDist
		if stepRatio > 1 {
			stepRatio = 1
		}
		npc.X += int32(float64(dest.X-npc.X) * stepRatio)
		npc.Y += int32(float64(dest.Y-npc.Y) * stepRatio)
		npc.Z = dest.Z
	}
	s.Broadcast(MoveToLocation(npc.ObjectID, dest.X, dest.Y, dest.Z, fromX, fromY, fromZ), nil)
}

func (s *Server) returnNPCHome(npc *NPC) {
	from := [3]int32{npc.X, npc.Y, npc.Z}
	npc.X, npc.Y, npc.Z = npc.SpawnX, npc.SpawnY, npc.SpawnZ
	npc.InCombat = false
	s.Broadcast(MoveToLocation(npc.ObjectID, npc.X, npc.Y, npc.Z, from[0], from[1], from[2]), nil)
	s.broadcastNpcIdle(npc)
}

func (s *Server) broadcastNpcIdle(npc *NPC) {
	if npc.AlikeDead() {
		return
	}
	s.Broadcast(NpcInfo(npc), nil)
}

// npcAttack is Java CreatureAttack.doAttack from the monster side.
func (s *Server) npcAttack(npc *NPC, c *GameClient) {
	p := c.Player()
	npc.InCombat = true
	p.InCombat = true
	c.lastHit = time.Now()
	if !Geo().CanSeeWorld(npc.X, npc.Y, npc.Z, npc.CollisionHeight, p.X, p.Y, p.Z, p.CollisionHeight) {
		return
	}

	behind := IsBehind(npc.X, npc.Y, p.X, p.Y, p.Heading)
	inFront := IsInFrontOf(npc.X, npc.Y, p.X, p.Y, p.Heading)
	miss := CalcHitMiss(npcAccuracy(npc), p.Evasion, npc.Z, p.Z, behind, inFront)

	var flags byte
	var damage int32
	if miss {
		flags |= hitFlagMiss
	} else {
		crit := CalcCrit(int32(baseCritRate * 10))
		if crit {
			flags |= hitFlagCritical
		}
		damage = int32(CalcPhysicalAttackDamage(npc.PAtk, p.PDef, shieldFailed, crit, false,
			PosMul(behind, inFront, crit), RandomDamageMultiplier(10)))
	}

	pkt := Attack(npc.ObjectID, p.ObjectID, damage, flags, npc.X, npc.Y, npc.Z)
	c.Send(pkt)
	s.Broadcast(pkt, c)
	if miss || damage <= 0 {
		return
	}

	// Java Creature.reduceCurrentHp spends CP before HP.
	if p.CurCP > 0 {
		absorbed := min64f(p.CurCP, float64(damage))
		p.CurCP -= absorbed
		damage -= int32(absorbed)
	}
	if damage > 0 {
		p.CurHP -= float64(damage)
	}
	if p.CurHP < 0 {
		p.CurHP = 0
	}
	c.Send(StatusUpdate(p.ObjectID, [][2]int32{
		{StatusCurHP, int32(p.CurHP)}, {StatusMaxHP, p.MaxHP},
		{StatusCurCP, int32(p.CurCP)}, {StatusMaxCP, p.MaxCP},
	}))
	if p.CurHP == 0 {
		s.killPlayer(c, npc.Name)
	}
}

func npcAccuracy(n *NPC) int32 {
	return int32(EvasionAccuracyBase(30) + float64(n.Level))
}

func min64f(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
