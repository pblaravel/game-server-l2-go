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

// AdenaID is Java PcInventory.ADENA_ID.
const AdenaID int32 = 57

// Item body-part masks from Java model/item/kind/Item.SLOT_*.
const (
	SlotNone      int32 = 0x0000
	SlotUnderwear int32 = 0x0001
	SlotREar      int32 = 0x0002
	SlotLEar      int32 = 0x0004
	SlotNeck      int32 = 0x0008
	SlotRFinger   int32 = 0x0010
	SlotLFinger   int32 = 0x0020
	SlotHead      int32 = 0x0040
	SlotRHand     int32 = 0x0080
	SlotLHand     int32 = 0x0100
	SlotGloves    int32 = 0x0200
	SlotChest     int32 = 0x0400
	SlotLegs      int32 = 0x0800
	SlotFeet      int32 = 0x1000
	SlotBack      int32 = 0x2000
	SlotLRHand    int32 = 0x4000
	SlotFullArmor int32 = 0x8000
	SlotFace      int32 = 0x010000
	SlotHair      int32 = 0x040000
	SlotHairAll   int32 = 0x080000
)

// Item type1/type2 from Java Item.TYPE1_* / TYPE2_*.
const (
	Type1WeaponJewel int16 = 0
	Type1ShieldArmor int16 = 1
	Type1QuestAdena  int16 = 4
	Type2Weapon      int16 = 0
	Type2ShieldArmor int16 = 1
	Type2Accessory   int16 = 2
	Type2Quest       int16 = 3
	Type2Money       int16 = 4
	Type2Other       int16 = 5
)

// ItemTemplate is the Java Item / Weapon / Armor / EtcItem row.
type ItemTemplate struct {
	ID           int32
	Name         string
	Kind         string // Weapon, Armor, EtcItem
	Weight       int32
	Price        int32
	BodyPart     int32
	Stackable    bool
	Sellable     bool
	Type1        int16
	Type2        int16
	PAtk         int32
	MAtk         int32
	PDef         int32
	MDef         int32
	PAtkSpd      int32
	CritRate     float64
	AtkRange     int32
	SkillID      int32
	SkillLevel   int32
	Handler      string
	Tradable     bool
	Destroyable  bool
	Dropable     bool
	CrystalType  string
	CrystalCount int32
}

var (
	itemMu      sync.RWMutex
	itemByID    = map[int32]ItemTemplate{}
	itemsLoaded bool
)

func GetItem(id int32) *ItemTemplate {
	itemMu.RLock()
	defer itemMu.RUnlock()
	t, ok := itemByID[id]
	if !ok {
		return nil
	}
	cp := t
	return &cp
}

func ItemCount() int {
	itemMu.RLock()
	defer itemMu.RUnlock()
	return len(itemByID)
}

func ItemsLoaded() bool {
	itemMu.RLock()
	defer itemMu.RUnlock()
	return itemsLoaded
}

type xmlItemDefList struct {
	Items []xmlItemDef `xml:"item"`
}

type xmlItemDef struct {
	ID   int32    `xml:"id,attr"`
	Type string   `xml:"type,attr"`
	Name string   `xml:"name,attr"`
	Sets []xmlSet `xml:"set"`
	For  xmlFor   `xml:"for"`
}

func loadItemXML(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	next := map[int32]ItemTemplate{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".xml") {
			continue
		}
		body, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return err
		}
		var list xmlItemDefList
		if err := xml.Unmarshal(body, &list); err != nil {
			return fmt.Errorf("%s: %w", e.Name(), err)
		}
		for _, it := range list.Items {
			if it.ID == 0 {
				continue
			}
			next[it.ID] = parseItemTemplate(it)
		}
	}
	itemMu.Lock()
	itemByID = next
	itemsLoaded = len(next) > 0
	itemMu.Unlock()
	return nil
}

func parseItemTemplate(it xmlItemDef) ItemTemplate {
	vals := map[string]string{}
	for _, s := range it.Sets {
		vals[s.Name] = s.Val
	}
	t := ItemTemplate{
		ID:       it.ID,
		Name:     it.Name,
		Kind:     it.Type,
		Weight:   atoi32(vals["weight"]),
		Price:    atoi32(vals["price"]),
		BodyPart: bodyPartMask(vals["bodypart"]),
		Sellable: parseBoolDefault(vals["is_sellable"], true),
		AtkRange: atoi32(vals["attack_range"]),
	}
	if t.AtkRange == 0 && strings.EqualFold(it.Type, "Weapon") {
		t.AtkRange = 40
	}
	t.Stackable = parseBoolDefault(vals["is_stackable"], false)
	t.Tradable = parseBoolDefault(vals["is_tradable"], true)
	t.Destroyable = parseBoolDefault(vals["is_destroyable"], true)
	t.Dropable = parseBoolDefault(vals["is_dropable"], true)
	t.CrystalType = vals["crystal_type"]
	t.CrystalCount = atoi32(vals["crystal_count"])
	t.Handler = vals["handler"]
	if id, lvl, ok := parseSkillRef(vals["item_skill"]); ok {
		t.SkillID, t.SkillLevel = id, lvl
	}
	applyItemFuncs(&t, it.For)
	t.Type1, t.Type2 = itemTypes(it.Type, t.ID, t.BodyPart)
	return t
}

func parseSkillRef(val string) (int32, int32, bool) {
	parts := strings.Split(val, "-")
	if len(parts) < 2 {
		return 0, 0, false
	}
	id, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	lvl, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err1 != nil || err2 != nil || id <= 0 {
		return 0, 0, false
	}
	return int32(id), int32(lvl), true
}

func applyItemFuncs(t *ItemTemplate, block xmlFor) {
	apply := func(fn xmlFunc) {
		val := atof(fn.Val)
		switch strings.ToLower(fn.Stat) {
		case "patk":
			t.PAtk = int32(val)
		case "matk":
			t.MAtk = int32(val)
		case "pdef":
			t.PDef = int32(val)
		case "mdef":
			t.MDef = int32(val)
		case "patkspd":
			t.PAtkSpd = int32(val)
		case "rcrit", "crit":
			t.CritRate = val
		}
	}
	for _, fn := range block.Sets {
		apply(fn)
	}
	for _, fn := range block.Adds {
		apply(fn)
	}
}

func itemTypes(kind string, id, bodyPart int32) (int16, int16) {
	switch strings.ToLower(kind) {
	case "weapon":
		return Type1WeaponJewel, Type2Weapon
	case "armor":
		if bodyPart == SlotNeck || bodyPart == SlotFace || bodyPart == SlotHair || bodyPart == SlotHairAll ||
			bodyPart&SlotLEar != 0 || bodyPart&SlotLFinger != 0 || bodyPart == SlotBack {
			return Type1WeaponJewel, Type2Accessory
		}
		return Type1ShieldArmor, Type2ShieldArmor
	default:
		if id == AdenaID {
			return Type1QuestAdena, Type2Money
		}
		return Type1QuestAdena, Type2Other
	}
}

func bodyPartMask(name string) int32 {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "chest":
		return SlotChest
	case "fullarmor":
		return SlotFullArmor
	case "head":
		return SlotHead
	case "hair":
		return SlotHair
	case "face":
		return SlotFace
	case "hairall":
		return SlotHairAll
	case "underwear":
		return SlotUnderwear
	case "back":
		return SlotBack
	case "neck":
		return SlotNeck
	case "legs":
		return SlotLegs
	case "feet":
		return SlotFeet
	case "gloves":
		return SlotGloves
	case "rhand":
		return SlotRHand
	case "lhand":
		return SlotLHand
	case "lrhand":
		return SlotLRHand
	case "rear;lear":
		return SlotREar | SlotLEar
	case "rfinger;lfinger":
		return SlotRFinger | SlotLFinger
	default:
		return SlotNone
	}
}

func ApplyItemTemplate(it *Item) {
	tpl := GetItem(it.ItemID)
	if tpl == nil {
		if it.BodyPart == 0 {
			it.BodyPart = BodyPartForItem(it.ItemID)
		}
		return
	}
	if it.BodyPart == 0 {
		it.BodyPart = tpl.BodyPart
	}
	if it.Type1 == 0 && it.Type2 == 0 {
		it.Type1, it.Type2 = tpl.Type1, tpl.Type2
	}
}

func atoi32(s string) int32 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	n, _ := strconv.ParseInt(s, 10, 32)
	return int32(n)
}

func atof(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

func parseBoolDefault(s string, def bool) bool {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return def
	}
	return s == "true" || s == "1" || s == "yes"
}
