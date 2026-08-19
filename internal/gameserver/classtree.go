package gameserver

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// ClassSkillNode is Java GeneralSkillNode (id, lvl, cost, minLvl).
type ClassSkillNode struct {
	ID     int32
	Level  int32
	Cost   int32
	MinLvl int32
}

// ClassTemplate is Java PlayerTemplate: base stats, per-level tables and the
// class skill tree from data/xml/classes.
type ClassTemplate struct {
	ID      int32
	Name    string
	BaseLvl int32
	Items   []NewbieItem
	Skills  []ClassSkillNode // this class only; parents are merged in ClassSkills()

	Fists                                      int32
	STR, CON, DEX, INT, WIT, MEN               int32
	PAtk, PDef, MAtk, MDef                     int32
	RunSpd, WalkSpd, SwimSpd                   int32
	Radius, RadiusFemale, Height, HeightFemale float64
	HPTable, MPTable, CPTable                  []float64
	HPRegenTable, MPRegenTable, CPRegenTable   []float64
}

func tableAt(t []float64, level int32) float64 {
	if len(t) == 0 {
		return 0
	}
	i := int(level) - 1
	if i < 0 {
		i = 0
	}
	if i >= len(t) {
		i = len(t) - 1
	}
	return t[i]
}

// MaxHPAt / MaxMPAt / MaxCPAt are Java PlayerTemplate.getBaseHpMax(level) etc.
func (t *ClassTemplate) MaxHPAt(level int32) float64 { return tableAt(t.HPTable, level) }
func (t *ClassTemplate) MaxMPAt(level int32) float64 { return tableAt(t.MPTable, level) }
func (t *ClassTemplate) MaxCPAt(level int32) float64 { return tableAt(t.CPTable, level) }
func (t *ClassTemplate) HPRegenAt(level int32) float64 {
	return tableAt(t.HPRegenTable, level)
}
func (t *ClassTemplate) MPRegenAt(level int32) float64 {
	return tableAt(t.MPRegenTable, level)
}
func (t *ClassTemplate) CPRegenAt(level int32) float64 {
	return tableAt(t.CPRegenTable, level)
}

var (
	classMu       sync.RWMutex
	classByID     = map[int32]*ClassTemplate{}
	classesLoaded bool
)

func GetClassTemplate(id int32) *ClassTemplate {
	classMu.RLock()
	defer classMu.RUnlock()
	return classByID[id]
}

func ClassCount() int {
	classMu.RLock()
	defer classMu.RUnlock()
	return len(classByID)
}

func AllClassTemplates() []*ClassTemplate {
	classMu.RLock()
	defer classMu.RUnlock()
	out := make([]*ClassTemplate, 0, len(classByID))
	for _, c := range classByID {
		out = append(out, c)
	}
	return out
}

// ClassSkills returns this class tree plus every parent, like Java PlayerData.load().
func ClassSkills(classID int32) []ClassSkillNode {
	classMu.RLock()
	defer classMu.RUnlock()
	var out []ClassSkillNode
	seen := map[[2]int32]struct{}{}
	for id := classID; ; {
		if tpl := classByID[id]; tpl != nil {
			for _, s := range tpl.Skills {
				k := [2]int32{s.ID, s.Level}
				if _, ok := seen[k]; ok {
					continue
				}
				seen[k] = struct{}{}
				out = append(out, s)
			}
		}
		p, ok := classParent[id]
		if !ok || p < 0 {
			break
		}
		id = p
	}
	return out
}

// AutoGetSkills is Java Player.getAvailableAutoGetSkills:
// cost == 0, minLvl <= level, maximal level per skill id.
func AutoGetSkills(classID, level int32) []ClassSkillNode {
	best := map[int32]ClassSkillNode{}
	for _, s := range ClassSkills(classID) {
		if s.Cost != 0 || s.MinLvl > level {
			continue
		}
		if prev, ok := best[s.ID]; !ok || s.Level > prev.Level {
			best[s.ID] = s
		}
	}
	out := make([]ClassSkillNode, 0, len(best))
	for _, s := range best {
		out = append(out, s)
	}
	return out
}

func NodesToSkills(nodes []ClassSkillNode) []Skill {
	out := make([]Skill, 0, len(nodes))
	for _, n := range nodes {
		sk := Skill{ID: n.ID, Level: n.Level}
		if tpl := GetSkill(n.ID, n.Level); tpl != nil {
			sk.Passive = tpl.IsPassive()
		}
		out = append(out, sk)
	}
	return out
}

// Interlude ClassId parents (XML / ordinal ids; dummy 58–87 between 2nd and 3rd class).
var classParent = map[int32]int32{
	1: 0, 2: 1, 3: 1, 4: 0, 5: 4, 6: 4, 7: 0, 8: 7, 9: 7,
	11: 10, 12: 11, 13: 11, 14: 11, 15: 10, 16: 15, 17: 15,
	19: 18, 20: 19, 21: 19, 22: 18, 23: 22, 24: 22,
	26: 25, 27: 26, 28: 26, 29: 25, 30: 29,
	32: 31, 33: 32, 34: 32, 35: 31, 36: 35, 37: 35,
	39: 38, 40: 39, 41: 39, 42: 38, 43: 42,
	45: 44, 46: 45, 47: 44, 48: 47,
	50: 49, 51: 50, 52: 50,
	54: 53, 55: 54, 56: 53, 57: 56,
	88: 2, 89: 3, 90: 5, 91: 6, 92: 9, 93: 8,
	94: 12, 95: 13, 96: 14, 97: 16, 98: 17,
	99: 20, 100: 21, 101: 23, 102: 24, 103: 27, 104: 28, 105: 30,
	106: 33, 107: 34, 108: 36, 109: 37, 110: 40, 111: 41, 112: 43,
	113: 46, 114: 48, 115: 51, 116: 52,
	117: 55, 118: 57,
}

type xmlClassList struct {
	Classes []xmlClass `xml:"class"`
}

type xmlClass struct {
	Sets   []xmlClassSet `xml:"set"`
	Items  []xmlItem     `xml:"items>item"`
	Skills []xmlCSkill   `xml:"skills>skill"`
}

type xmlClassSet struct {
	ID      string `xml:"id,attr"`
	BaseLvl string `xml:"baseLvl,attr"`
	Name    string `xml:"name,attr"`
	Val     string `xml:"val,attr"`

	Fists        string `xml:"fists,attr"`
	STR          string `xml:"str,attr"`
	CON          string `xml:"con,attr"`
	DEX          string `xml:"dex,attr"`
	INT          string `xml:"int,attr"`
	WIT          string `xml:"wit,attr"`
	MEN          string `xml:"men,attr"`
	PAtk         string `xml:"pAtk,attr"`
	PDef         string `xml:"pDef,attr"`
	MAtk         string `xml:"mAtk,attr"`
	MDef         string `xml:"mDef,attr"`
	RunSpd       string `xml:"runSpd,attr"`
	WalkSpd      string `xml:"walkSpd,attr"`
	SwimSpd      string `xml:"swimSpd,attr"`
	Radius       string `xml:"radius,attr"`
	RadiusFemale string `xml:"radiusFemale,attr"`
	Height       string `xml:"height,attr"`
	HeightFemale string `xml:"heightFemale,attr"`
	HPTable      string `xml:"hpTable,attr"`
	MPTable      string `xml:"mpTable,attr"`
	CPTable      string `xml:"cpTable,attr"`
	HPRegenTable string `xml:"hpRegenTable,attr"`
	MPRegenTable string `xml:"mpRegenTable,attr"`
	CPRegenTable string `xml:"cpRegenTable,attr"`
}

func setI32(dst *int32, raw string) {
	if raw != "" {
		*dst = parseI32(raw)
	}
}

func setF64(dst *float64, raw string) {
	if raw != "" {
		*dst = parseF64(raw)
	}
}

func setTable(dst *[]float64, raw string) {
	if raw == "" {
		return
	}
	parts := strings.Split(raw, ";")
	out := make([]float64, 0, len(parts))
	for _, p := range parts {
		out = append(out, parseF64(strings.TrimSpace(p)))
	}
	*dst = out
}

type xmlItem struct {
	ID       int32  `xml:"id,attr"`
	Count    int32  `xml:"count,attr"`
	Equipped string `xml:"isEquipped,attr"`
}

type xmlCSkill struct {
	ID     int32 `xml:"id,attr"`
	Lvl    int32 `xml:"lvl,attr"`
	Cost   int32 `xml:"cost,attr"`
	MinLvl int32 `xml:"minLvl,attr"`
}

func loadClassXML(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	next := map[int32]*ClassTemplate{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".xml") {
			continue
		}
		body, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return err
		}
		var list xmlClassList
		if err := xml.Unmarshal(body, &list); err != nil {
			return fmt.Errorf("%s: %w", e.Name(), err)
		}
		for _, c := range list.Classes {
			tpl := parseClass(c)
			if tpl.ID == 0 && len(tpl.Skills) == 0 {
				// Human Fighter is id 0; keep it when it has skills or items.
				if len(tpl.Items) == 0 {
					continue
				}
			}
			next[tpl.ID] = tpl
		}
	}
	classMu.Lock()
	classByID = next
	classesLoaded = len(next) > 0
	classMu.Unlock()
	return nil
}

func parseClass(c xmlClass) *ClassTemplate {
	tpl := &ClassTemplate{}
	for _, s := range c.Sets {
		if s.ID != "" {
			tpl.ID = parseI32(s.ID)
		}
		setI32(&tpl.BaseLvl, s.BaseLvl)
		setI32(&tpl.Fists, s.Fists)
		setI32(&tpl.STR, s.STR)
		setI32(&tpl.CON, s.CON)
		setI32(&tpl.DEX, s.DEX)
		setI32(&tpl.INT, s.INT)
		setI32(&tpl.WIT, s.WIT)
		setI32(&tpl.MEN, s.MEN)
		setI32(&tpl.PAtk, s.PAtk)
		setI32(&tpl.PDef, s.PDef)
		setI32(&tpl.MAtk, s.MAtk)
		setI32(&tpl.MDef, s.MDef)
		setI32(&tpl.RunSpd, s.RunSpd)
		setI32(&tpl.WalkSpd, s.WalkSpd)
		setI32(&tpl.SwimSpd, s.SwimSpd)
		setF64(&tpl.Radius, s.Radius)
		setF64(&tpl.RadiusFemale, s.RadiusFemale)
		setF64(&tpl.Height, s.Height)
		setF64(&tpl.HeightFemale, s.HeightFemale)
		setTable(&tpl.HPTable, s.HPTable)
		setTable(&tpl.MPTable, s.MPTable)
		setTable(&tpl.CPTable, s.CPTable)
		setTable(&tpl.HPRegenTable, s.HPRegenTable)
		setTable(&tpl.MPRegenTable, s.MPRegenTable)
		setTable(&tpl.CPRegenTable, s.CPRegenTable)
	}
	for _, it := range c.Items {
		eq := true
		if it.Equipped != "" {
			eq = parseBool(it.Equipped)
		}
		tpl.Items = append(tpl.Items, NewbieItem{
			ItemID: it.ID, Count: it.Count, Equipped: eq,
			BodyPart: BodyPartForItem(it.ID),
		})
	}
	for _, sk := range c.Skills {
		tpl.Skills = append(tpl.Skills, ClassSkillNode{
			ID: sk.ID, Level: sk.Lvl, Cost: sk.Cost, MinLvl: sk.MinLvl,
		})
	}
	return tpl
}
