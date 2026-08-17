package config

// LoginConfig mirrors Java ServerConfig / conf/server.properties.
type LoginConfig struct {
	LoginServerHost            string
	LoginServerPort            int
	GameServerHost             string
	GameServerPort             int
	Revision                   int
	ConnectionTimeoutMS        int
	AcceptNewGameServer        bool
	AutoCreateAccount          bool
	AutoCreateAccountAccessLvl int
	AccountInactiveLevel       int
	ShowLicense                bool
	GameServerRSAPadding       string
	ClientRSAPadding           string
	DatabaseURL                string
	PrintReceivedPackets       bool
	PrintSentPackets           bool
	PrintCryptography          bool
}

func DefaultLoginConfig() LoginConfig {
	return LoginConfig{
		LoginServerHost:            "*",
		LoginServerPort:            2107,
		GameServerHost:             "127.0.0.1",
		GameServerPort:             9015,
		Revision:                   0x0102,
		ConnectionTimeoutMS:        10000,
		AcceptNewGameServer:        true,
		AutoCreateAccount:          true,
		AutoCreateAccountAccessLvl: 0,
		AccountInactiveLevel:       -127,
		ShowLicense:                true,
		GameServerRSAPadding:       "RSA/ECB/nopadding",
		ClientRSAPadding:           "RSA/ECB/PKCS1Padding",
		DatabaseURL:                "postgres://l2unity:l2unity@localhost:5432/l2unity?sslmode=disable",
		PrintReceivedPackets:       false,
		PrintSentPackets:           false,
		PrintCryptography:          false,
	}
}

func LoadLoginConfig(paths ...string) (LoginConfig, error) {
	cfg := DefaultLoginConfig()
	p, err := LoadProperties(paths...)
	if err != nil && len(p) == 0 {
		return cfg, err
	}
	cfg.LoginServerHost = p.String("loginserver.host", cfg.LoginServerHost)
	cfg.LoginServerPort = p.Int("loginserver.port", cfg.LoginServerPort)
	cfg.GameServerHost = p.String("gameserver.host", cfg.GameServerHost)
	cfg.GameServerPort = p.Int("gameserver.port", cfg.GameServerPort)
	cfg.Revision = p.Int("revision", cfg.Revision)
	cfg.ConnectionTimeoutMS = p.Int("server.connection.timeout.ms", cfg.ConnectionTimeoutMS)
	cfg.AcceptNewGameServer = p.Bool("accept.new.gameserver", cfg.AcceptNewGameServer)
	cfg.AutoCreateAccount = p.Bool("server.account.autocreate", cfg.AutoCreateAccount)
	cfg.AutoCreateAccountAccessLvl = p.Int("server.account.autocreate.access.level", cfg.AutoCreateAccountAccessLvl)
	cfg.AccountInactiveLevel = p.Int("server.account.inactive.access.level", cfg.AccountInactiveLevel)
	cfg.ShowLicense = p.Bool("server.show.license", cfg.ShowLicense)
	cfg.GameServerRSAPadding = p.String("rsa.padding.mode.gameserver", cfg.GameServerRSAPadding)
	cfg.ClientRSAPadding = p.String("rsa.padding.mode.client", cfg.ClientRSAPadding)
	if v := p.String("database.url", ""); v != "" {
		cfg.DatabaseURL = v
	} else if v := p.String("database.jdbc.url", ""); v != "" {
		cfg.DatabaseURL = jdbcToPostgres(v, p.String("database.jdbc.username", "l2unity"), p.String("database.jdbc.password", "l2unity"))
	}
	cfg.PrintReceivedPackets = p.Bool("logger.print.received-packets", cfg.PrintReceivedPackets)
	cfg.PrintSentPackets = p.Bool("logger.print.sent-packets", cfg.PrintSentPackets)
	cfg.PrintCryptography = p.Bool("logger.print.cryptography", cfg.PrintCryptography)
	ApplyLoginEnv(&cfg)
	return cfg, nil
}

func jdbcToPostgres(jdbc, user, pass string) string {
	// jdbc:mariadb://localhost/l2unity → postgres://user:pass@localhost:5432/l2unity
	const prefix = "jdbc:mariadb://"
	const prefix2 = "jdbc:postgresql://"
	hostdb := jdbc
	switch {
	case len(jdbc) > len(prefix) && jdbc[:len(prefix)] == prefix:
		hostdb = jdbc[len(prefix):]
	case len(jdbc) > len(prefix2) && jdbc[:len(prefix2)] == prefix2:
		hostdb = jdbc[len(prefix2):]
	}
	if hostdb == jdbc {
		return jdbc
	}
	host, db, ok := splitHostDB(hostdb)
	if !ok {
		return "postgres://" + user + ":" + pass + "@localhost:5432/l2unity?sslmode=disable"
	}
	if !containsPort(host) {
		host += ":5432"
	}
	return "postgres://" + user + ":" + pass + "@" + host + "/" + db + "?sslmode=disable"
}

func splitHostDB(s string) (host, db string, ok bool) {
	i := 0
	for i < len(s) && s[i] != '/' {
		i++
	}
	if i == 0 || i >= len(s)-1 {
		return "", "", false
	}
	return s[:i], s[i+1:], true
}

func containsPort(host string) bool {
	for i := len(host) - 1; i >= 0; i-- {
		if host[i] == ']' {
			return false
		}
		if host[i] == ':' {
			return true
		}
	}
	return false
}
