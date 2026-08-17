package config

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestHexIDRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hexid.txt")
	want := []byte{0x81, 0xa8, 0xba, 0x90, 0xdb, 0x0e, 0x77, 0xd3, 0x03, 0x39, 0x73, 0x88, 0xe2, 0x5e, 0xce, 0xfa}
	if err := SaveHexID(path, 1, want); err != nil {
		t.Fatal(err)
	}
	id, got, used, err := LoadHexID(path)
	if err != nil {
		t.Fatal(err)
	}
	if used != path || id != 1 || hex.EncodeToString(got) != hex.EncodeToString(want) {
		t.Fatalf("id=%d used=%s hex=%x", id, used, got)
	}
}

func TestLoadHexIDJavaFormat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hexid.txt")
	body := "#the hexID to auth into login\n#Sun Mar 30 14:27:30 SGT 2025\nHexID=81a8ba90db0e77d303397388e25ecefa\nServerID=1\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	id, hexID, _, err := LoadHexID(path)
	if err != nil {
		t.Fatal(err)
	}
	if id != 1 || hex.EncodeToString(hexID) != "81a8ba90db0e77d303397388e25ecefa" {
		t.Fatalf("id=%d hex=%x", id, hexID)
	}
}

func TestLoadGameConfigUsesHexIDFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hexid.txt")
	if err := SaveHexID(path, 3, []byte{1, 2, 3, 4}); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HEXID_FILE", path)
	cfg := DefaultGameConfig()
	ApplyGameEnv(&cfg)
	applyHexIDFile(&cfg)
	if cfg.ServerID != 3 || cfg.RequestID != 3 || len(cfg.HexID) != 4 {
		t.Fatalf("%+v", cfg)
	}
}
