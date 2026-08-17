package gameserver

import "testing"

func TestDatapackSkillsLoaded(t *testing.T) {
	if !DatapackLoaded() {
		t.Fatal("expected data/xml skills+classes")
	}
	if SkillCount() < 10000 {
		t.Fatalf("skill levels %d", SkillCount())
	}
	if ClassCount() < 80 {
		t.Fatalf("classes %d", ClassCount())
	}
	lucky := GetSkill(194, 1)
	if lucky == nil || !lucky.IsPassive() || lucky.Name != "Lucky" {
		t.Fatalf("lucky: %+v", lucky)
	}
	ps := GetSkill(3, 1)
	if ps == nil || ps.HitTime != 1080 || ps.ReuseDelay != 13000 || ps.Name != "Power Strike" {
		t.Fatalf("power strike: %+v", ps)
	}
	ws := GetSkill(1177, 1)
	if ws == nil || ws.OperateType != "ACTIVE" {
		t.Fatalf("wind strike: %+v", ws)
	}
}

func TestHumanFighterAutoGet(t *testing.T) {
	nodes := AutoGetSkills(0, 1)
	got := map[int32]int32{}
	for _, n := range nodes {
		got[n.ID] = n.Level
	}
	if got[194] != 1 {
		t.Fatalf("Lucky missing: %v", got)
	}
	if got[1320] != 1 {
		t.Fatalf("Create Common missing: %v", got)
	}
	if _, ok := got[3]; ok {
		t.Fatal("Power Strike is not autoGet at lvl 1")
	}
	ch := DefaultCharacter("a", "H", 0, 0, 0, 0, 0, 0, 1, nil)
	ids := map[int32]bool{}
	for _, s := range ch.Skills {
		ids[s.ID] = true
		if s.ID == 194 && !s.Passive {
			t.Fatal("Lucky should be passive")
		}
	}
	if !ids[194] {
		t.Fatal("created fighter missing Lucky")
	}
}

func TestMysticAutoGetWindStrike(t *testing.T) {
	nodes := AutoGetSkills(10, 1)
	found := false
	for _, n := range nodes {
		if n.ID == 1177 {
			found = true
		}
	}
	if !found {
		t.Fatal("Human Mystic should autoGet Wind Strike 1177")
	}
}

func TestClassTreeParents(t *testing.T) {
	// Gladiator (2) inherits Human Fighter (0) + Warrior (1).
	ids := map[int32]bool{}
	for _, s := range ClassSkills(2) {
		ids[s.ID] = true
	}
	if !ids[194] || !ids[3] {
		t.Fatalf("gladiator tree missing parent skills: lucky=%v powerStrike=%v size=%d", ids[194], ids[3], len(ids))
	}
}
