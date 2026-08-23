package config

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

// Properties is a Java-style key=value file loader.
type Properties map[string]string

func LoadProperties(paths ...string) (Properties, error) {
	p := Properties{}
	var lastErr error
	loaded := false
	for _, path := range paths {
		f, err := os.Open(path)
		if err != nil {
			lastErr = err
			continue
		}
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			k, v, ok := strings.Cut(line, "=")
			if !ok {
				// Java Properties: a bare key (geoengine region list) has an empty value.
				p[line] = ""
				continue
			}
			p[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
		_ = f.Close()
		loaded = true
	}
	if !loaded && lastErr != nil {
		return p, lastErr
	}
	return p, nil
}

func (p Properties) String(key, def string) string {
	if v, ok := p[key]; ok {
		return v
	}
	return def
}

func (p Properties) Int(key string, def int) int {
	v, ok := p[key]
	if !ok {
		return def
	}
	if strings.HasPrefix(v, "0x") || strings.HasPrefix(v, "0X") {
		n, err := strconv.ParseInt(v[2:], 16, 64)
		if err != nil {
			return def
		}
		return int(n)
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func (p Properties) Bool(key string, def bool) bool {
	v, ok := p[key]
	if !ok {
		return def
	}
	switch strings.ToLower(v) {
	case "true", "1", "yes":
		return true
	case "false", "0", "no":
		return false
	default:
		return def
	}
}

// ParseIntList parses "737,740,744" into ints.
func ParseIntList(s string) []int {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == ';' || r == ' ' || r == '|'
	})
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			continue
		}
		out = append(out, n)
	}
	return out
}
