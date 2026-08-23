package clientapi

import "testing"

func TestLoginSequenceMatchesUnity(t *testing.T) {
	pairs := LoginPairs()
	if len(pairs) != 5 {
		t.Fatalf("login sequence %d, want 5", len(pairs))
	}
	want := []struct {
		name string
		c2s  byte
		s2c  byte
	}{
		{"Init", 0xFF, 0x00},
		{"AuthGameGuard", 0x07, 0x0B},
		{"RequestAuthLogin", 0x00, 0x03},
		{"RequestServerList", 0x05, 0x04},
		{"RequestServerLogin", 0x02, 0x07},
	}
	for i, w := range want {
		if pairs[i].Name != w.name || pairs[i].C2S != w.c2s || !ContainsOpcode(pairs[i].S2C, w.s2c) {
			t.Fatalf("login[%d]=%s 0x%02x → %x, want %s 0x%02x → 0x%02x",
				i, pairs[i].Name, pairs[i].C2S, pairs[i].S2C, w.name, w.c2s, w.s2c)
		}
	}
}

func TestGameCatalogCoversImplementedOutgoing(t *testing.T) {
	seen := map[byte]string{}
	for _, p := range GamePairs() {
		if prev, ok := seen[p.C2S]; ok {
			t.Fatalf("duplicate C→S 0x%02x (%s and %s)", p.C2S, prev, p.Name)
		}
		seen[p.C2S] = p.Name
	}
	required := []byte{
		0x00, 0x08, 0x0E, 0x0B, 0x0C, 0x0D, 0x03, 0x30, 0x01, 0x48,
		0x04, 0x37, 0x2F, 0x6D, 0x14, 0x59, 0x58, 0x1F, 0x1E, 0xC6,
		0xA7, 0x44, 0x16, 0x17, 0x31, 0x32, 0x9E, 0x9F, 0x38, 0x21,
		0x57, 0xAA, 0x9D, 0x6B, 0x6C, 0x33, 0x35, 0x64, 0x2A, 0x66,
		0x24, 0x26, 0x27, 0x55, 0xC0, 0xAC, 0xAD, 0xAE, 0xAF,
	}
	for _, op := range required {
		if _, ok := seen[op]; !ok {
			t.Fatalf("missing outgoing 0x%02x from Unity catalog", op)
		}
	}
	if len(GamePairs()) < 49 {
		t.Fatalf("catalog %d, want at least the 49 implemented Unity packets", len(GamePairs()))
	}
}

func TestContainsOpcode(t *testing.T) {
	if !ContainsOpcode([]byte{0x04, 0x1B, 0x58}, 0x1B) {
		t.Fatal("should find 0x1B")
	}
	if ContainsOpcode([]byte{0x04}, 0x1B) {
		t.Fatal("should not find 0x1B")
	}
}
