package config

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const defaultHexIDFile = "conf/gameserver/hexid.txt"

func defaultHexIDPaths() []string {
	return []string{defaultHexIDFile, "conf/hexid.txt"}
}

// LoadHexID reads Java conf/hexid.txt (ServerID + HexID).
func LoadHexID(paths ...string) (serverID int, hexID []byte, used string, err error) {
	if len(paths) == 0 {
		paths = defaultHexIDPaths()
	}
	var last error
	for _, path := range paths {
		if path == "" {
			continue
		}
		p, e := LoadProperties(path)
		if e != nil {
			last = e
			continue
		}
		raw := strings.TrimSpace(p.String("HexID", ""))
		if raw == "" {
			last = fmt.Errorf("hexid: empty HexID in %s", path)
			continue
		}
		if len(raw)%2 == 1 {
			raw = "0" + raw
		}
		b, e := hex.DecodeString(raw)
		if e != nil {
			last = fmt.Errorf("hexid: %s: %w", path, e)
			continue
		}
		id := p.Int("ServerID", 1)
		if id <= 0 {
			id = 1
		}
		return id, b, path, nil
	}
	if last == nil {
		last = fmt.Errorf("hexid: not found")
	}
	return 0, nil, "", last
}

// SaveHexID writes Java-style hexid.txt so the next start reuses the same gameservers row.
func SaveHexID(path string, serverID int, hexID []byte) error {
	if path == "" {
		path = defaultHexIDFile
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	body := fmt.Sprintf("# the hexID to auth into login\nServerID=%d\nHexID=%s\n", serverID, hex.EncodeToString(hexID))
	return os.WriteFile(path, []byte(body), 0o600)
}

func applyHexIDFile(cfg *GameConfig) {
	paths := defaultHexIDPaths()
	if cfg.HexIDFile != "" {
		paths = append([]string{cfg.HexIDFile}, paths...)
	} else {
		cfg.HexIDFile = defaultHexIDFile
	}
	id, hexID, used, err := LoadHexID(paths...)
	if err != nil || len(hexID) == 0 {
		return
	}
	cfg.HexIDFile = used
	cfg.HexID = hexID
	cfg.ServerID = id
	cfg.RequestID = id
}
