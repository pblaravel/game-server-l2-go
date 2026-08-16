package gameserver

import "testing"

func TestStarterKitHumanFighter(t *testing.T) {
	ch := DefaultCharacter("acc", "Hero", 0, 0, 0, 0, 0, 0, 100, nil)
	if len(ch.Items) == 0 {
		t.Fatal("expected starter items")
	}
	var hasSword, hasAdena, hasGuide bool
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
		case 57:
			hasAdena = it.Count == 10000
		case 5588:
			hasGuide = true
		}
	}
	if !hasSword || !hasAdena || !hasGuide {
		t.Fatalf("kit incomplete sword=%v adena=%v guide=%v items=%d", hasSword, hasAdena, hasGuide, len(ch.Items))
	}
	if ch.PaperdollItem[PaperRHand] != 2369 {
		t.Fatal("rhand paperdoll")
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
	if ExpForLevel(81) != 610560301 {
		t.Fatal("level 81")
	}
	if p := ExpPercent(1, 34); p < 0.4 || p > 0.6 {
		t.Fatalf("percent %v", p)
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
