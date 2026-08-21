package gameserver

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// NpcTemplate is the Java NpcTemplate / CreatureTemplate row used at spawn time.
type NpcTemplate struct {
	ID            int32
	Name          string
	Title         string
	Type          string
	Level         int32
	HP            int32
	MP            int32
	Exp           int64
	SP            int32
	PAtk          int32
	PDef          int32
	MAtk          int32
	MDef          int32
	PAtkSpd       int32
	RunSpeed      int32
	WalkSpeed     int32
	AttackRange   int32
	AggroRange    int32
	Radius        float64
	Height        float64
	RHand         int32
	LHand         int32
	CanBeAttacked bool
	Drops         []DropCategory
}

// DropCategory is Java model/item/DropCategory.
type DropCategory struct {
	Type   string
	Chance float64
	Drops  []DropData
}

// DropData is Java model/item/DropData.
type DropData struct {
	ItemID int32
	Min    int32
	Max    int32
	Chance float64
}

var (
	npcMu      sync.RWMutex
	npcByID    = map[int32]NpcTemplate{}
	npcsLoaded bool
)

func GetNpcTemplate(id int32) *NpcTemplate {
	npcMu.RLock()
	defer npcMu.RUnlock()
	t, ok := npcByID[id]
	if !ok {
		return nil
	}
	cp := t
	return &cp
}

func NpcTemplateCount() int {
	npcMu.RLock()
	defer npcMu.RUnlock()
	return len(npcByID)
}

type xmlNpcList struct {
	NPCs []xmlNpc `xml:"npc"`
}

type xmlNpc struct {
	ID    int32    `xml:"id,attr"`
	Name  string   `xml:"name,attr"`
	Title string   `xml:"title,attr"`
	Sets  []xmlSet `xml:"set"`
	Drops xmlDrops `xml:"drops"`
}

type xmlDrops struct {
	Categories []xmlDropCat `xml:"category"`
}

type xmlDropCat struct {
	Type   string    `xml:"type,attr"`
	Chance float64   `xml:"chance,attr"`
	Drops  []xmlDrop `xml:"drop"`
}

type xmlDrop struct {
	ItemID int32   `xml:"itemid,attr"`
	Min    int32   `xml:"min,attr"`
	Max    int32   `xml:"max,attr"`
	Chance float64 `xml:"chance,attr"`
}

func loadNpcXML(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	next := map[int32]NpcTemplate{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".xml") {
			continue
		}
		body, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return err
		}
		var list xmlNpcList
		if err := xml.Unmarshal(body, &list); err != nil {
			return fmt.Errorf("%s: %w", e.Name(), err)
		}
		for _, n := range list.NPCs {
			if n.ID == 0 {
				continue
			}
			next[n.ID] = parseNpcTemplate(n)
		}
	}
	npcMu.Lock()
	npcByID = next
	npcsLoaded = len(next) > 0
	npcMu.Unlock()
	return nil
}

func parseNpcTemplate(n xmlNpc) NpcTemplate {
	vals := map[string]string{}
	for _, s := range n.Sets {
		vals[s.Name] = s.Val
	}
	t := NpcTemplate{
		ID:            n.ID,
		Name:          n.Name,
		Title:         n.Title,
		Type:          vals["type"],
		Level:         atoi32(vals["level"]),
		HP:            int32(atof(vals["hp"])),
		MP:            int32(atof(vals["mp"])),
		Exp:           int64(atof(vals["exp"])),
		SP:            int32(atof(vals["sp"])),
		PAtk:          int32(atof(vals["pAtk"])),
		PDef:          int32(atof(vals["pDef"])),
		MAtk:          int32(atof(vals["mAtk"])),
		MDef:          int32(atof(vals["mDef"])),
		PAtkSpd:       int32(atof(vals["atkSpd"])),
		RunSpeed:      int32(atof(vals["runSpd"])),
		WalkSpeed:     int32(atof(vals["walkSpd"])),
		AttackRange:   atoi32(vals["baseAttackRange"]),
		AggroRange:    atoi32(vals["aggroRange"]),
		Radius:        atof(vals["radius"]),
		Height:        atof(vals["height"]),
		RHand:         atoi32(vals["rHand"]),
		LHand:         atoi32(vals["lHand"]),
		CanBeAttacked: parseBoolDefault(vals["canBeAttacked"], true),
	}
	for _, cat := range n.Drops.Categories {
		dc := DropCategory{Type: cat.Type, Chance: cat.Chance}
		for _, d := range cat.Drops {
			dc.Drops = append(dc.Drops, DropData{ItemID: d.ItemID, Min: d.Min, Max: d.Max, Chance: d.Chance})
		}
		t.Drops = append(t.Drops, dc)
	}
	return t
}

// ApplyNpcTemplate copies Java NpcData fields onto a live NPC when the XML is loaded.
func ApplyNpcTemplate(n *NPC) {
	tpl := GetNpcTemplate(n.NPCID)
	if tpl == nil {
		return
	}
	if n.Name == "" {
		n.Name = tpl.Name
	}
	if n.Title == "" {
		n.Title = tpl.Title
	}
	if n.Level == 0 {
		n.Level = tpl.Level
	}
	if n.MaxHP == 0 && tpl.HP > 0 {
		n.MaxHP = tpl.HP
		n.CurHP = tpl.HP
	}
	if n.MaxMP == 0 && tpl.MP > 0 {
		n.MaxMP = tpl.MP
		n.CurMP = tpl.MP
	}
	if n.Exp == 0 {
		n.Exp = tpl.Exp
	}
	if n.SP == 0 {
		n.SP = tpl.SP
	}
	if n.PAtk == 0 {
		n.PAtk = tpl.PAtk
	}
	if n.PDef == 0 {
		n.PDef = tpl.PDef
	}
	if n.MAtk == 0 {
		n.MAtk = tpl.MAtk
	}
	if n.MDef == 0 {
		n.MDef = tpl.MDef
	}
	if n.PAtkSpd == 0 && tpl.PAtkSpd > 0 {
		n.PAtkSpd = tpl.PAtkSpd
	}
	if n.RunSpeed == 0 && tpl.RunSpeed > 0 {
		n.RunSpeed = tpl.RunSpeed
	}
	if n.WalkSpeed == 0 && tpl.WalkSpeed > 0 {
		n.WalkSpeed = tpl.WalkSpeed
	}
	if n.AttackRange == 0 && tpl.AttackRange > 0 {
		n.AttackRange = tpl.AttackRange
	}
	if n.AggroRange == 0 {
		n.AggroRange = tpl.AggroRange
	}
	if n.CollisionRadius == 0 && tpl.Radius > 0 {
		n.CollisionRadius = tpl.Radius
	}
	if n.CollisionHeight == 0 && tpl.Height > 0 {
		n.CollisionHeight = tpl.Height
	}
	if n.RHand == 0 {
		n.RHand = tpl.RHand
	}
	if n.LHand == 0 {
		n.LHand = tpl.LHand
	}
	n.IsAttackable = tpl.CanBeAttacked && isMonsterType(tpl.Type)
}

func isMonsterType(typ string) bool {
	switch strings.ToLower(typ) {
	case "monster", "raidboss", "grandboss", "minion", "chest", "festivalmonster":
		return true
	default:
		return false
	}
}
