package gameserver

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	if err := LoadDatapack(FindDataDir()); err != nil {
		os.Stderr.WriteString("datapack load: " + err.Error() + "\n")
	}
	os.Exit(m.Run())
}

func TestStarterKitHumanFighter(t *testing.T) {
	ch := DefaultCharacter("acc", "Hero", 0, 0, 0, 0, 0, 0, 100, nil)
	if len(ch.Items) == 0 {
		t.Fatal("expected starter items")
	}
	var hasSword, hasGuide bool
	for _, it := range ch.Items {
		if it.ObjectID == 0 {
			t.Fatal("item object id")
		}
		switch it.ItemID {
		case 2369:
			hasSword = true
			if !it.Equipped {
				t.Fatal("sword should be equipped")
			}
		case 5588:
			hasGuide = true
		}
	}
	if !hasSword || !hasGuide {
		t.Fatalf("kit incomplete sword=%v guide=%v items=%d", hasSword, hasGuide, len(ch.Items))
	}
	if len(ch.Skills) == 0 {
		t.Fatal("expected starter skills")
	}
	if len(ch.Shortcuts) < 3 {
		t.Fatalf("expected default shortcuts, got %d", len(ch.Shortcuts))
	}
}

func TestStarterKitAllClasses(t *testing.T) {
	next := int32(1000)
	for _, id := range []int32{0, 10, 18, 25, 31, 38, 44, 49, 53} {
		ch := DefaultCharacter("acc", "C", id, 0, 0, 0, 0, 0, 1, func() int32 {
			next++
			return next
		})
		if len(ch.Items) == 0 || len(ch.Skills) == 0 {
			t.Fatalf("class %d missing kit", id)
		}
	}
}

func TestLoadDefaultSpawns(t *testing.T) {
	w := NewWorld()
	w.LoadDefaultSpawns()
	if n := len(w.NPCs()); n < 10 {
		t.Fatalf("expected newbie spawns, got %d", n)
	}
	seen := map[int32]bool{}
	for _, n := range w.NPCs() {
		if n.ObjectID == 0 {
			t.Fatal("npc object id")
		}
		if seen[n.ObjectID] {
			t.Fatalf("duplicate object id %d", n.ObjectID)
		}
		seen[n.ObjectID] = true
	}
}

func TestExpTable(t *testing.T) {
	if ExpForLevel(1) != 0 {
		t.Fatal("level 1")
	}
	if ExpForLevel(2) != 68 {
		t.Fatal("level 2")
	}
	if PlayerLevelCount() == 0 {
		t.Fatal("playerLevels.xml was not loaded")
	}
	// Java PlayerLevelData: first unreachable level is 81.
	if ExpForLevel(14) != 191452 {
		t.Fatalf("level 14 exp = %d, want 191452 from playerLevels.xml", ExpForLevel(14))
	}
	if ExpForLevel(81) != 6299994999 {
		t.Fatalf("level 81 exp = %d, want 6299994999 from playerLevels.xml", ExpForLevel(81))
	}
	if p := ExpPercent(1, 34); p < 0.4 || p > 0.6 {
		t.Fatalf("percent %v", p)
	}
}

func TestClassSpawnFromXML(t *testing.T) {
	tpl := GetClassTemplate(0)
	if tpl == nil || len(tpl.Spawns) == 0 {
		t.Fatal("human fighter spawn list missing")
	}
	// Cedric hall coords are commented out; live spawn is Talking Island village.
	if tpl.Spawns[0][0] != -90875 || tpl.Spawns[0][1] != 248162 {
		t.Fatalf("human fighter spawn = %#v, want Talking Island village", tpl.Spawns[0])
	}
	ch := DefaultCharacter("acc", "Spawn", 0, 0, 0, 0, 0, 0, 300, nil)
	if ch.X != -90875 || ch.Y != 248162 {
		t.Fatalf("created at %d,%d want XML spawn", ch.X, ch.Y)
	}
}

func TestShortCutInitOpcode(t *testing.T) {
	ch := DefaultCharacter("acc", "Hero", 0, 0, 0, 0, 0, 0, 100, nil)
	p := ShortCutInit(ch.Shortcuts)
	if p[0] != 0x45 {
		t.Fatalf("opcode %d", p[0])
	}
}

func TestMemoryStorePersistsKit(t *testing.T) {
	st := NewMemoryCharacterStore()
	ch := DefaultCharacter("acc", "Hero", 0, 0, 0, 0, 0, 0, 200, nil)
	if err := st.Create(nil, ch); err != nil {
		t.Fatal(err)
	}
	got, err := st.GetByObjectID(nil, 200)
	if err != nil || got == nil {
		t.Fatal(err)
	}
	if len(got.Items) != len(ch.Items) || len(got.Skills) != len(ch.Skills) {
		t.Fatal("kit not persisted")
	}
	orig := ch.Items[0].Count
	got.Items[0].Count = 1
	again, _ := st.GetByObjectID(nil, 200)
	if again.Items[0].Count != orig {
		t.Fatal("store must deep-copy items")
	}
}
