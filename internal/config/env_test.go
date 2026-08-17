package config

import (
	"os"
	"testing"
)

func TestApplyEnvOverrides(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x:y@db:5432/l2?sslmode=disable")
	t.Setenv("LOGIN_HOST", "loginserver")
	t.Setenv("LOGIN_PORT", "9015")
	t.Setenv("LOGINSERVER_PORT", "2107")
	t.Setenv("GAMESERVER_REGISTER_PORT", "9015")

	lc := DefaultLoginConfig()
	ApplyLoginEnv(&lc)
	if lc.DatabaseURL != os.Getenv("DATABASE_URL") {
		t.Fatal(lc.DatabaseURL)
	}
	if lc.LoginServerPort != 2107 || lc.GameServerPort != 9015 {
		t.Fatal(lc)
	}

	gc := DefaultGameConfig()
	ApplyGameEnv(&gc)
	if gc.LoginHost != "loginserver" || gc.LoginPort != 9015 {
		t.Fatal(gc)
	}
}
