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

type ClassTemplate struct {
	ID      int32
	Name    string
	BaseLvl int32
	Items   []NewbieItem
	Skills  []ClassSkillNode // this class only; parents are merged in ClassSkills()
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
		if s.BaseLvl != "" {
			tpl.BaseLvl = parseI32(s.BaseLvl)
		}
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
