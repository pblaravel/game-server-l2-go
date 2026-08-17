package db

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSplitSQL(t *testing.T) {
	script := `
-- comment
CREATE TABLE IF NOT EXISTS foo (id INTEGER);
INSERT INTO foo (id) VALUES (1),
(2);
`
	stmts := splitSQL(script)
	if len(stmts) != 2 {
		t.Fatalf("got %d stmts: %#v", len(stmts), stmts)
	}
	if !strings.Contains(stmts[0], "CREATE TABLE") {
		t.Fatal(stmts[0])
	}
	if !strings.Contains(stmts[1], "INSERT INTO foo") {
		t.Fatal(stmts[1])
	}
}

func TestFindSQLDir(t *testing.T) {
	dir := FindSQLDir()
	if _, err := os.Stat(filepath.Join(dir, "001_init.sql")); err != nil {
		t.Fatalf("001_init.sql not found from %s: %v", dir, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "003_skills.sql")); err != nil {
		t.Fatalf("003_skills.sql not found from %s: %v", dir, err)
	}
}

func TestSplitSQLRealFiles(t *testing.T) {
	dir := FindSQLDir()
	for _, name := range []string{"001_init.sql", "002_seed.sql", "003_skills.sql"} {
		body, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		stmts := splitSQL(string(body))
		if len(stmts) < 2 {
			t.Fatalf("%s: too few statements (%d)", name, len(stmts))
		}
		for i, s := range stmts {
			if strings.TrimSpace(s) == "" {
				t.Fatalf("%s: empty statement %d", name, i)
			}
		}
	}
}

func TestSeedFileHasJavaInserts(t *testing.T) {
	body, err := os.ReadFile(filepath.Join(FindSQLDir(), "002_seed.sql"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(body)
	for _, want := range []string{
		"INSERT INTO castle",
		"INSERT INTO clanhall",
		"INSERT INTO seven_signs_status",
		"INSERT INTO seven_signs_festival",
		"INSERT INTO mdt_bets",
		"INSERT INTO npc_templates",
		"INSERT INTO npc_spawns",
		"INSERT INTO player_levels",
		"INSERT INTO class_templates",
		"'Gludio'",
		"'Fortress of Resistance'",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("seed missing %q", want)
		}
	}
}
