package gameserver

import (
	"math"
	"net"
	"testing"
	"time"

	"github.com/pblaravel/game-server-l2-go/internal/config"
)

// testClient wires a GameClient to a discarded socket so the handlers can run
// without a real client on the other end.
func testClient(t *testing.T, srv *Server, p *Character) (*GameClient, func()) {
	t.Helper()
	server, client := net.Pipe()
	go func() {
		buf := make([]byte, 4096)
		for {
			if _, err := client.Read(buf); err != nil {
				return
			}
		}
	}()
	c := NewGameClient(server, srv)
	c.SetAccountName(p.Account)
	c.SetPlayer(p)
	c.SetState(StateInGame)
	p.Online = true
	srv.mu.Lock()
	srv.clients = append(srv.clients, c)
	srv.mu.Unlock()
	srv.world.AddPlayer(p)
	return c, func() {
		_ = client.Close()
		c.Close()
	}
}

func combatTestServer(t *testing.T) *Server {
	t.Helper()
	cfg := config.DefaultGameConfig()
	cfg.LoginHost = "127.0.0.1"
	cfg.LoginPort = 1
	srv := NewServer(cfg, NewMemoryCharacterStore())
	srv.login.Stop()
	return srv
}

func TestPlayerAttackDamagesAndKillsMonster(t *testing.T) {
	srv := combatTestServer(t)
	p := DefaultCharacter("acc", "Fighter", 0, 0, 0, 1, 1, 1, 268440000, nil)
	p.Level = 10
	RecalcStats(p)
	c, stop := testClient(t, srv, p)
	defer stop()

	npc := &NPC{ObjectID: srv.world.NextID(), NPCID: 20001, Name: "Gremlin",
		X: p.X, Y: p.Y, Z: p.Z, Level: 1, MaxHP: 40, CurHP: 40, IsAttackable: true}
	npc.NpcDefaults()
	srv.world.AddNPC(npc)
	c.target = npc.ObjectID

	startExp := p.Exp
	// Hitting can miss, so keep swinging until the monster dies.
	for i := 0; i < 200 && !npc.AlikeDead(); i++ {
		srv.doAttackHit(c, npc)
	}
	if !npc.AlikeDead() {
		t.Fatalf("monster survived 200 hits with %d HP left", npc.CurHP)
	}
	if p.Exp <= startExp {
		t.Errorf("exp = %d, want more than %d after the kill", p.Exp, startExp)
	}
	if !p.InCombat {
		t.Error("the player should be in combat stance after attacking")
	}
}

func TestMonsterRetaliatesAndCanKillPlayer(t *testing.T) {
	srv := combatTestServer(t)
	p := DefaultCharacter("acc", "Victim", 0, 0, 0, 1, 1, 1, 268440001, nil)
	RecalcStats(p)
	p.CurHP = 5
	p.CurCP = 0
	c, stop := testClient(t, srv, p)
	defer stop()

	npc := &NPC{ObjectID: srv.world.NextID(), NPCID: 20120, Name: "Wolf",
		X: p.X, Y: p.Y, Z: p.Z, Level: 40, MaxHP: 500, CurHP: 500, IsAttackable: true}
	npc.NpcDefaults()
	srv.world.AddNPC(npc)

	for i := 0; i < 200 && !p.Dead; i++ {
		srv.npcAttack(npc, c)
	}
	if !p.Dead {
		t.Fatalf("player survived 200 monster hits with %.0f HP", p.CurHP)
	}
	if p.CurHP != 0 {
		t.Errorf("dead player HP = %.0f, want 0", p.CurHP)
	}

	// Respawn restores the configured share of the maximum values.
	srv.revive(c, RespawnHPPercent)
	if p.Dead {
		t.Error("player is still dead after revive")
	}
	if want := float64(p.MaxHP) * RespawnHPPercent; p.CurHP != want {
		t.Errorf("HP after revive = %.1f, want %.1f", p.CurHP, want)
	}
}

func TestAggroScanEngagesNearbyMonster(t *testing.T) {
	srv := combatTestServer(t)
	p := DefaultCharacter("acc", "Walker", 0, 0, 0, 1, 1, 1, 268440002, nil)
	RecalcStats(p)
	_, stop := testClient(t, srv, p)
	defer stop()

	far := &NPC{ObjectID: srv.world.NextID(), NPCID: 20120, Name: "FarWolf",
		X: p.X + 5000, Y: p.Y, Z: p.Z, Level: 4, MaxHP: 80, CurHP: 80, IsAttackable: true}
	far.NpcDefaults()
	srv.world.AddNPC(far)
	srv.runAggroScan()
	if srv.ai.engagedID(far.ObjectID) {
		t.Error("a monster 5000 units away should not aggro")
	}

	near := &NPC{ObjectID: srv.world.NextID(), NPCID: 20120, Name: "NearWolf",
		X: p.X + 100, Y: p.Y, Z: p.Z, Level: 4, MaxHP: 80, CurHP: 80, IsAttackable: true}
	near.NpcDefaults()
	srv.world.AddNPC(near)
	srv.runAggroScan()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) && !srv.ai.engagedID(near.ObjectID) {
		time.Sleep(10 * time.Millisecond)
	}
	if !srv.ai.engagedID(near.ObjectID) {
		t.Error("a monster inside the aggro range should engage the player")
	}
}

// Java Monster.calculateExpAndSp: pow(5/6, diff - 5) once the gap exceeds five.
func TestNpcRewardFallsOffWithLevelDifference(t *testing.T) {
	npc := &NPC{Level: 10, Exp: 1000, SP: 100}
	if exp, sp := npcReward(npc, 10); exp != 1000 || sp != 100 {
		t.Errorf("same level reward = (%d, %d), want (1000, 100)", exp, sp)
	}
	if exp, sp := npcReward(npc, 15); exp != 1000 || sp != 100 {
		t.Errorf("five level gap reward = (%d, %d), want the full (1000, 100)", exp, sp)
	}
	wantExp := int64(1000 * math.Pow(5.0/6.0, 3))
	if exp, _ := npcReward(npc, 18); exp != wantExp {
		t.Errorf("eight level gap reward = %d, want %d", exp, wantExp)
	}
	if exp, _ := npcReward(npc, 90); exp != 0 {
		t.Errorf("huge level gap reward = %d, want 0", exp)
	}
}
