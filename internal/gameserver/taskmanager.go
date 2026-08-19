package gameserver

import (
	"context"
	"time"
)

// Periodic tasks ported from Java taskmanager/: RegenTaskManager (HP/MP/CP),
// the EffectList tick, PvpFlagTaskManager and AttackStanceTaskManager.

const (
	// Java Formulas.HP_REGENERATE_PERIOD.
	regenPeriod = 3 * time.Second
	// Java Config.PVP_TIME.
	pvpFlagDuration = 40 * time.Second
	// Java AttackStanceTaskManager.COMBAT_TIME.
	combatStanceDuration = 15 * time.Second
	effectTickPeriod     = 1 * time.Second
)

// RunTaskManagers starts the periodic loops for the whole world.
func (s *Server) RunTaskManagers(ctx context.Context) {
	go s.loop(ctx, regenPeriod, s.regenTick)
	go s.loop(ctx, effectTickPeriod, s.effectTick)
	go s.loop(ctx, aggroScanPeriod, s.runAggroScan)
}

func (s *Server) loop(ctx context.Context, period time.Duration, tick func()) {
	t := time.NewTicker(period)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			tick()
		}
	}
}

// regenTick is Java RegenTaskManager: sitting doubles HP/MP regeneration and
// standing still restores CP.
func (s *Server) regenTick() {
	for _, c := range s.inGameClients() {
		p := c.Player()
		if p == nil || p.AlikeDead() {
			continue
		}
		hpMul, mpMul := 1.0, 1.0
		if p.Sitting {
			hpMul, mpMul = 1.5, 1.5
		} else if p.InCombat {
			hpMul, mpMul = 0.7, 0.7
		}
		before := [3]int32{int32(p.CurHP), int32(p.CurMP), int32(p.CurCP)}
		p.CurHP = clampF(p.CurHP+HPRegen(p)*hpMul, 0, float64(p.MaxHP))
		p.CurMP = clampF(p.CurMP+MPRegen(p)*mpMul, 0, float64(p.MaxMP))
		p.CurCP = clampF(p.CurCP+CPRegen(p), 0, float64(p.MaxCP))
		if before == [3]int32{int32(p.CurHP), int32(p.CurMP), int32(p.CurCP)} {
			continue
		}
		c.Send(StatusUpdate(p.ObjectID, [][2]int32{
			{StatusCurHP, int32(p.CurHP)}, {StatusMaxHP, p.MaxHP},
			{StatusCurMP, int32(p.CurMP)}, {StatusMaxMP, p.MaxMP},
			{StatusCurCP, int32(p.CurCP)}, {StatusMaxCP, p.MaxCP},
		}))
	}
}

// effectTick expires buffs and clears the PvP flag and combat stance.
func (s *Server) effectTick() {
	now := time.Now()
	for _, c := range s.inGameClients() {
		p := c.Player()
		if p == nil {
			continue
		}
		if expired := PurgeExpiredEffects(p); len(expired) > 0 {
			for _, e := range expired {
				c.Send(SystemMessage(SMEffectWornOff, SysSkill(e.SkillID, e.SkillLevel)))
			}
			c.Send(UserInfo(p))
			c.Broadcast(CharInfo(p))
		}
		if p.PvPFlag != 0 && now.Sub(c.lastPvP) > pvpFlagDuration {
			p.PvPFlag = 0
			c.Send(UserInfo(p))
			c.Broadcast(CharInfo(p))
		}
		if p.InCombat && !c.attacking.Load() && now.Sub(c.lastHit) > combatStanceDuration {
			p.InCombat = false
			c.Broadcast(CharInfo(p))
		}
	}
}

func (s *Server) inGameClients() []*GameClient {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*GameClient, 0, len(s.clients))
	for _, c := range s.clients {
		if c.State() == StateInGame && !c.Closed() {
			out = append(out, c)
		}
	}
	return out
}

func clampF(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
