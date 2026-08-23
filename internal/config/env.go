package config

import (
	"os"
	"strconv"
	"strings"
)

func envBool(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// ApplyEnv overlays Docker/k8s environment variables on top of .properties.
func ApplyLoginEnv(cfg *LoginConfig) {
	if v := os.Getenv("DATABASE_URL"); v != "" {
		cfg.DatabaseURL = v
	}
	if v := os.Getenv("LOGINSERVER_PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.LoginServerPort = n
		}
	}
	if envBool(os.Getenv("INTERLUDE_CLIENT")) {
		cfg.InterludeClient = true
	}
	if v := os.Getenv("GAMESERVER_REGISTER_PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.GameServerPort = n
		}
	}
}

func ApplyGameEnv(cfg *GameConfig) {
	if v := os.Getenv("DATABASE_URL"); v != "" {
		cfg.DatabaseURL = v
	}
	if v := os.Getenv("LOGIN_HOST"); v != "" {
		cfg.LoginHost = v
	}
	if v := os.Getenv("LOGIN_PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.LoginPort = n
		}
	}
	if v := os.Getenv("GAMESERVER_PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.GameserverPort = n
		}
	}
	if v := os.Getenv("GAMESERVER_HOSTNAME"); v != "" {
		cfg.Hostname = v
	}
	if v := os.Getenv("PACKET_LOG"); v != "" {
		on := envBool(v)
		cfg.PacketHandlerDebug = on
		cfg.PrintReceivedPackets = on
		cfg.PrintSentPackets = on
	}
	if v := os.Getenv("PACKET_HANDLER_DEBUG"); v != "" {
		cfg.PacketHandlerDebug = envBool(v)
	}
	if v := os.Getenv("ALLOWED_PROTOCOL_VERSIONS"); v != "" {
		if vers := ParseIntList(v); len(vers) > 0 {
			cfg.AllowedProtocolVers = vers
		}
	}
	if v := os.Getenv("STRICT_PROTOCOL"); v != "" {
		cfg.StrictProtocol = envBool(v)
	}
	if v := os.Getenv("HEXID_FILE"); v != "" {
		cfg.HexIDFile = v
	}
	if v := os.Getenv("SERVER_ID"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.ServerID = n
			cfg.RequestID = n
		}
	}
}
