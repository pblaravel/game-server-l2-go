package config

// GameConfig mirrors Java Config / conf/server.properties used by the gameserver.
type GameConfig struct {
	Hostname             string
	GameserverPort       int
	LoginHost            string
	LoginPort            int
	RequestID            int
	AcceptAlternateID    bool
	ReserveHostOnLogin   bool
	MaximumOnlineUsers   int
	UseBlowfishCipher    bool
	ServerGMOnly         bool
	ServerListClock      bool
	ServerListBracket    bool
	ServerListAge        int
	ServerListTestServer bool
	ServerListPvPServer  bool
	AllowedProtocolVers  []int
	AllowedPosDesync     float64
	DatabaseURL          string
	HexID                []byte
	ServerID             int
	Developer            bool
	PacketHandlerDebug   bool
	PrintReceivedPackets bool
	PrintSentPackets     bool
}

func DefaultGameConfig() GameConfig {
	return GameConfig{
		Hostname:            "*",
		GameserverPort:      7778,
		LoginHost:           "127.0.0.1",
		LoginPort:           9015,
		RequestID:           1,
		AcceptAlternateID:   true,
		ReserveHostOnLogin:  false,
		MaximumOnlineUsers:  100,
		UseBlowfishCipher:   true,
		AllowedProtocolVers: []int{737, 740, 744, 746},
		AllowedPosDesync:    0.5,
		DatabaseURL:         "postgres://l2unity:l2unity@localhost:5432/l2unity?sslmode=disable",
	}
}

func LoadGameConfig(paths ...string) (GameConfig, error) {
	cfg := DefaultGameConfig()
	p, err := LoadProperties(paths...)
	if err != nil && len(p) == 0 {
		return cfg, err
	}
	cfg.Hostname = p.String("GameserverHostname", p.String("gameserver.hostname", cfg.Hostname))
	cfg.GameserverPort = p.Int("GameserverPort", p.Int("gameserver.port", cfg.GameserverPort))
	cfg.LoginHost = p.String("LoginHost", p.String("loginserver.host", cfg.LoginHost))
	cfg.LoginPort = p.Int("LoginPort", p.Int("loginserver.port", cfg.LoginPort))
	cfg.RequestID = p.Int("RequestServerID", p.Int("request.id", cfg.RequestID))
	cfg.AcceptAlternateID = p.Bool("AcceptAlternateID", cfg.AcceptAlternateID)
	cfg.MaximumOnlineUsers = p.Int("MaximumOnlineUsers", cfg.MaximumOnlineUsers)
	cfg.UseBlowfishCipher = p.Bool("UseBlowfishCipher", cfg.UseBlowfishCipher)
	cfg.ServerGMOnly = p.Bool("ServerGMOnly", cfg.ServerGMOnly)
	cfg.Developer = p.Bool("Developer", cfg.Developer)
	cfg.PacketHandlerDebug = p.Bool("PacketHandlerDebug", cfg.PacketHandlerDebug)
	cfg.PrintReceivedPackets = p.Bool("logger.print.received-packets", cfg.PrintReceivedPackets)
	cfg.PrintSentPackets = p.Bool("logger.print.sent-packets", cfg.PrintSentPackets)
	if v := p.String("database.url", ""); v != "" {
		cfg.DatabaseURL = v
	} else if v := p.String("URL", ""); v != "" {
		cfg.DatabaseURL = jdbcToPostgres(v, p.String("Login", p.String("database.jdbc.username", "l2unity")), p.String("Password", p.String("database.jdbc.password", "l2unity")))
	}
	ApplyGameEnv(&cfg)
	return cfg, nil
}
