package gameserver

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// SkillTemplate is the Java SkillTable row used by SkillList / MagicSkillUse.
// Effects (<for>) are not executed; operateType/timings/mp are enough for the wire API.
type SkillTemplate struct {
	ID          int32
	Level       int32
	Name        string
	OperateType string // PASSIVE, ACTIVE, TOGGLE
	SkillType   string
	IsMagic     bool
	MPConsume   int32
	HitTime     int32
	ReuseDelay  int32
	CoolTime    int32
	Power       float64
	MagicLvl    int32
	TargetType  string
	CastRange   int32
	EffectRange int32
}

func (t SkillTemplate) IsPassive() bool {
	return strings.EqualFold(t.OperateType, "PASSIVE")
}

func skillHash(id, level int32) int32 { return id*256 + level }

var (
	skillMu      sync.RWMutex
	skillByHash  = map[int32]SkillTemplate{}
	skillMaxLvl  = map[int32]int32{}
	skillsLoaded bool
)

func GetSkill(id, level int32) *SkillTemplate {
	skillMu.RLock()
	defer skillMu.RUnlock()
	t, ok := skillByHash[skillHash(id, level)]
	if !ok {
		return nil
	}
	cp := t
	return &cp
}

func SkillCount() int {
	skillMu.RLock()
	defer skillMu.RUnlock()
	return len(skillByHash)
}

func AllSkills() []SkillTemplate {
	skillMu.RLock()
	defer skillMu.RUnlock()
	out := make([]SkillTemplate, 0, len(skillByHash))
	for _, t := range skillByHash {
		out = append(out, t)
	}
	return out
}

func SkillMaxLevel(id int32) int32 {
	skillMu.RLock()
	defer skillMu.RUnlock()
	return skillMaxLvl[id]
}

type xmlSkillList struct {
	XMLName xml.Name   `xml:"list"`
	Skills  []xmlSkill `xml:"skill"`
}

type xmlSkill struct {
	ID         int32     `xml:"id,attr"`
	Levels     int       `xml:"levels,attr"`
	Name       string    `xml:"name,attr"`
	Enchant1   int       `xml:"enchantLevels1,attr"`
	Enchant2   int       `xml:"enchantLevels2,attr"`
	Tables     []xmlTbl  `xml:"table"`
	Sets       []xmlSet  `xml:"set"`
	Enchant1s  []xmlSet  `xml:"enchant1"`
	Enchant2s  []xmlSet  `xml:"enchant2"`
}

type xmlTbl struct {
	Name string `xml:"name,attr"`
	Text string `xml:",chardata"`
}

type xmlSet struct {
	Name string `xml:"name,attr"`
	Val  string `xml:"val,attr"`
}

func loadSkillXML(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	next := map[int32]SkillTemplate{}
	maxLvl := map[int32]int32{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".xml") {
			continue
		}
		body, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return err
		}
		var list xmlSkillList
		if err := xml.Unmarshal(body, &list); err != nil {
			return fmt.Errorf("%s: %w", e.Name(), err)
		}
		for _, sk := range list.Skills {
			expandSkill(sk, next, maxLvl)
		}
	}
	skillMu.Lock()
	skillByHash = next
	skillMaxLvl = maxLvl
	skillsLoaded = len(next) > 0
	skillMu.Unlock()
	return nil
}

func expandSkill(sk xmlSkill, out map[int32]SkillTemplate, maxLvl map[int32]int32) {
	if sk.ID == 0 || sk.Levels <= 0 {
		return
	}
	tables := map[string][]string{}
	for _, t := range sk.Tables {
		name := strings.TrimSpace(t.Name)
		tables[name] = splitWS(t.Text)
	}
	apply := func(level int32, tableIdx int, extra []xmlSet) {
		vals := map[string]string{}
		for _, s := range sk.Sets {
			vals[s.Name] = resolveTable(s.Val, tables, tableIdx)
		}
		for _, s := range extra {
			vals[s.Name] = resolveTable(s.Val, tables, tableIdx)
		}
		tpl := SkillTemplate{
			ID:          sk.ID,
			Level:       level,
			Name:        sk.Name,
			OperateType: vals["operateType"],
			SkillType:   vals["skillType"],
			IsMagic:     parseBool(vals["isMagic"]),
			MPConsume:   parseI32(vals["mpConsume"]),
			HitTime:     parseI32(vals["hitTime"]),
			ReuseDelay:  parseI32(vals["reuseDelay"]),
			CoolTime:    parseI32(vals["coolTime"]),
			Power:       parseF64(vals["power"]),
			MagicLvl:    parseI32(vals["magicLvl"]),
			TargetType:  vals["target"],
			CastRange:   parseI32(vals["castRange"]),
			EffectRange: parseI32(vals["effectRange"]),
		}
		out[skillHash(tpl.ID, tpl.Level)] = tpl
		if tpl.Level < 99 && tpl.Level > maxLvl[tpl.ID] {
			maxLvl[tpl.ID] = tpl.Level
		}
	}
	for i := 1; i <= sk.Levels; i++ {
		apply(int32(i), i, nil)
	}
	// Enchant routes use the last normal table index, then overlay enchant1/2 (Java DocumentSkill).
	for i := 0; i < sk.Enchant1; i++ {
		apply(int32(101+i), sk.Levels, sk.Enchant1s)
	}
	for i := 0; i < sk.Enchant2; i++ {
		apply(int32(141+i), sk.Levels, sk.Enchant2s)
	}
}

func resolveTable(val string, tables map[string][]string, level int) string {
	val = strings.TrimSpace(val)
	if val == "" || val[0] != '#' {
		return val
	}
	row := tables[val]
	if len(row) == 0 {
		return ""
	}
	idx := level - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(row) {
		idx = len(row) - 1
	}
	return row[idx]
}

func splitWS(s string) []string {
	return strings.Fields(s)
}

func parseI32(s string) int32 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	if i, err := strconv.ParseInt(s, 10, 32); err == nil {
		return int32(i)
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return int32(f)
	}
	return 0
}

func parseF64(s string) float64 {
	f, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return f
}

func parseBool(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes":
		return true
	}
	return false
}
