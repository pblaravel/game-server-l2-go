package config

import (
	"os"
	"strconv"
)

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
}
