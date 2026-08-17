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

	t.Setenv("PACKET_LOG", "true")
	gc := DefaultGameConfig()
	ApplyGameEnv(&gc)
	if gc.LoginHost != "loginserver" || gc.LoginPort != 9015 {
		t.Fatal(gc)
	}
	if !gc.PacketHandlerDebug || !gc.PrintReceivedPackets || !gc.PrintSentPackets {
		t.Fatal("PACKET_LOG should enable packet traces")
	}

	t.Setenv("ALLOWED_PROTOCOL_VERSIONS", "740")
	t.Setenv("STRICT_PROTOCOL", "false")
	ApplyGameEnv(&gc)
	if len(gc.AllowedProtocolVers) != 1 || gc.AllowedProtocolVers[0] != 740 {
		t.Fatal(gc.AllowedProtocolVers)
	}
}
