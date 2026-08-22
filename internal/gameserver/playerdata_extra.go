package gameserver

import (
	"encoding/xml"
	"fmt"
	"os"
	"sync"
)

// ArmorSet is Java model.item.ArmorSet, keyed by chest item id.
type ArmorSet struct {
	Name          string
	Chest         int32
	Legs          int32
	Head          int32
	Gloves        int32
	Feet          int32
	SkillID       int32
	Shield        int32
	ShieldSkillID int32
	Enchant6Skill int32
}

type Spellbook struct {
	SkillID int32
	ItemID  int32
}

type HealSpsRow struct {
	SkillID    int32
	SkillLevel int32
	MagicLevel int32
	Correction float64
	NeededMAtk int32
}

type NewbieBuff struct {
	SkillID      int32
	SkillLevel   int32
	LowerLevel   int32
	UpperLevel   int32
	IsMagicClass bool
}

type FishingSkillNode struct {
	ID        int32
	Level     int32
	MinLvl    int32
	ItemID    int32
	ItemCount int32
}

type ClanSkillNode struct {
	ID     int32
	Level  int32
	Cost   int32
	MinLvl int32
	ItemID int32
}

type EnchantSkillNode struct {
	ID         int32
	Level      int32
	Exp        int64
	SP         int32
	Rate76     int32
	Rate77     int32
	Rate78     int32
	Rate79     int32
	Rate80     int32
	ItemNeeded [2]int32
}

var (
	extraPlayerMu sync.RWMutex
	armorSets     = map[int32]ArmorSet{}
	spellbooks    = map[int32]int32{}
	healSpsRows   []HealSpsRow
	newbieBuffs   []NewbieBuff
	fishingSkills []FishingSkillNode
	clanSkills    []ClanSkillNode
	enchantSkills []EnchantSkillNode
)

func ArmorSetCount() int {
	extraPlayerMu.RLock()
	defer extraPlayerMu.RUnlock()
	return len(armorSets)
}

func SpellbookCount() int {
	extraPlayerMu.RLock()
	defer extraPlayerMu.RUnlock()
	return len(spellbooks)
}

func HealSpsCount() int {
	extraPlayerMu.RLock()
	defer extraPlayerMu.RUnlock()
	return len(healSpsRows)
}

func NewbieBuffCount() int {
	extraPlayerMu.RLock()
	defer extraPlayerMu.RUnlock()
	return len(newbieBuffs)
}

func FishingSkillCount() int {
	extraPlayerMu.RLock()
	defer extraPlayerMu.RUnlock()
	return len(fishingSkills)
}

func ClanSkillCount() int {
	extraPlayerMu.RLock()
	defer extraPlayerMu.RUnlock()
	return len(clanSkills)
}

func EnchantSkillCount() int {
	extraPlayerMu.RLock()
	defer extraPlayerMu.RUnlock()
	return len(enchantSkills)
}

func GetArmorSet(chestID int32) *ArmorSet {
	extraPlayerMu.RLock()
	defer extraPlayerMu.RUnlock()
	s, ok := armorSets[chestID]
	if !ok {
		return nil
	}
	cp := s
	return &cp
}

func (s ArmorSet) containsAll(p *Character) bool {
	if s.Legs != 0 && equippedItemID(p, PaperLegs) != s.Legs {
		return false
	}
	if s.Head != 0 && equippedItemID(p, PaperHead) != s.Head {
		return false
	}
	if s.Gloves != 0 && equippedItemID(p, PaperGloves) != s.Gloves {
		return false
	}
	if s.Feet != 0 && equippedItemID(p, PaperFeet) != s.Feet {
		return false
	}
	return true
}

func equippedItemID(p *Character, slot Paperdoll) int32 {
	if int(slot) < 0 || int(slot) >= len(p.PaperdollItem) {
		return 0
	}
	return p.PaperdollItem[slot]
}

func armorSetFuncs(p *Character) []FuncTemplate {
	chest := equippedItemID(p, PaperChest)
	if chest == 0 {
		return nil
	}
	set := GetArmorSet(chest)
	if set == nil || !set.containsAll(p) {
		return nil
	}
	var out []FuncTemplate
	add := func(id int32) {
		if id == 0 {
			return
		}
		if tpl := GetSkill(id, 1); tpl != nil {
			out = append(out, tpl.Funcs...)
			for _, e := range tpl.Effects {
				out = append(out, e.Funcs...)
			}
		}
	}
	add(set.SkillID)
	if set.Shield != 0 && equippedItemID(p, PaperLHand) == set.Shield {
		add(set.ShieldSkillID)
	}
	return out
}

// BookForSkill is Java SpellbookData.getBookForSkill.
func BookForSkill(skillID, level int32) int32 {
	if skillID == 1405 { // Divine Inspiration
		switch level {
		case 1:
			return 8618
		case 2:
			return 8619
		case 3:
			return 8620
		case 4:
			return 8621
		default:
			return 0
		}
	}
	if level != 1 {
		return 0
	}
	extraPlayerMu.RLock()
	defer extraPlayerMu.RUnlock()
	return spellbooks[skillID]
}

// CalculateHealSps is Java HealSpsData.calculateHealSps.
func CalculateHealSps(tpl *SkillTemplate, mAtk int32) float64 {
	if tpl == nil {
		return 0
	}
	extraPlayerMu.RLock()
	rows := healSpsRows
	extraPlayerMu.RUnlock()
	var match *HealSpsRow
	for i := range rows {
		if rows[i].SkillID == tpl.ID && rows[i].SkillLevel == tpl.Level {
			match = &rows[i]
			break
		}
	}
	if match == nil && tpl.MagicLvl > 0 {
		var best *HealSpsRow
		for i := range rows {
			if rows[i].SkillID != 0 {
				continue
			}
			if rows[i].MagicLevel <= tpl.MagicLvl && (best == nil || rows[i].MagicLevel > best.MagicLevel) {
				best = &rows[i]
			}
		}
		match = best
	}
	if match == nil {
		return 0
	}
	amount := match.Correction
	diff := match.NeededMAtk - mAtk
	if diff <= 0 {
		return amount
	}
	return amount - float64(diff)/2
}

func isMageClass(classID int32) bool {
	switch classID {
	case 10, 11, 12, 13, 14, 15, 16, 17,
		25, 26, 27, 28, 29, 30,
		38, 39, 40, 41, 42, 43,
		49, 50, 51, 52,
		94, 95, 96, 97, 98,
		103, 104, 105, 110, 111, 112:
		return true
	default:
		return false
	}
}

func ValidNewbieBuffs(isMage bool, level int32) []NewbieBuff {
	extraPlayerMu.RLock()
	defer extraPlayerMu.RUnlock()
	out := make([]NewbieBuff, 0)
	for _, b := range newbieBuffs {
		if b.IsMagicClass == isMage && level >= b.LowerLevel && level <= b.UpperLevel {
			out = append(out, b)
		}
	}
	return out
}

func GetFishingSkills() []FishingSkillNode {
	extraPlayerMu.RLock()
	defer extraPlayerMu.RUnlock()
	return append([]FishingSkillNode(nil), fishingSkills...)
}

func GetClanSkills() []ClanSkillNode {
	extraPlayerMu.RLock()
	defer extraPlayerMu.RUnlock()
	return append([]ClanSkillNode(nil), clanSkills...)
}

type xmlArmorSetList struct {
	Sets []xmlArmorSet `xml:"armorset"`
}

type xmlArmorSet struct {
	Name          string `xml:"name,attr"`
	Chest         int32  `xml:"chest,attr"`
	Legs          int32  `xml:"legs,attr"`
	Head          int32  `xml:"head,attr"`
	Gloves        int32  `xml:"gloves,attr"`
	Feet          int32  `xml:"feet,attr"`
	SkillID       int32  `xml:"skillId,attr"`
	Shield        int32  `xml:"shield,attr"`
	ShieldSkillID int32  `xml:"shieldSkillId,attr"`
	Enchant6Skill int32  `xml:"enchant6Skill,attr"`
}

func loadArmorSetXML(path string) error {
	var root xmlArmorSetList
	if err := readXML(path, &root); err != nil {
		return err
	}
	next := map[int32]ArmorSet{}
	for _, s := range root.Sets {
		if s.Chest == 0 {
			continue
		}
		next[s.Chest] = ArmorSet{
			Name: s.Name, Chest: s.Chest, Legs: s.Legs, Head: s.Head,
			Gloves: s.Gloves, Feet: s.Feet, SkillID: s.SkillID,
			Shield: s.Shield, ShieldSkillID: s.ShieldSkillID, Enchant6Skill: s.Enchant6Skill,
		}
	}
	extraPlayerMu.Lock()
	armorSets = next
	extraPlayerMu.Unlock()
	return nil
}

type xmlSpellbookList struct {
	Books []xmlSpellbook `xml:"book"`
}

type xmlSpellbook struct {
	SkillID int32 `xml:"skillId,attr"`
	ItemID  int32 `xml:"itemId,attr"`
}

func loadSpellbookXML(path string) error {
	var root xmlSpellbookList
	if err := readXML(path, &root); err != nil {
		return err
	}
	next := map[int32]int32{}
	for _, b := range root.Books {
		if b.SkillID > 0 {
			next[b.SkillID] = b.ItemID
		}
	}
	extraPlayerMu.Lock()
	spellbooks = next
	extraPlayerMu.Unlock()
	return nil
}

type xmlHealSpsList struct {
	Rows []xmlHealSps `xml:"healSps"`
}

type xmlHealSps struct {
	SkillID    int32   `xml:"skillId,attr"`
	SkillLevel int32   `xml:"skillLevel,attr"`
	MagicLevel int32   `xml:"magicLevel,attr"`
	Correction float64 `xml:"correction,attr"`
	NeededMAtk int32   `xml:"neededMatk,attr"`
}

func loadHealSpsXML(path string) error {
	var root xmlHealSpsList
	if err := readXML(path, &root); err != nil {
		return err
	}
	next := make([]HealSpsRow, 0, len(root.Rows))
	for _, r := range root.Rows {
		next = append(next, HealSpsRow{
			SkillID: r.SkillID, SkillLevel: r.SkillLevel, MagicLevel: r.MagicLevel,
			Correction: r.Correction, NeededMAtk: r.NeededMAtk,
		})
	}
	extraPlayerMu.Lock()
	healSpsRows = next
	extraPlayerMu.Unlock()
	return nil
}

type xmlNewbieBuffList struct {
	Buffs []xmlNewbieBuff `xml:"buff"`
}

type xmlNewbieBuff struct {
	SkillID    int32  `xml:"skillId,attr"`
	SkillLevel int32  `xml:"skillLevel,attr"`
	Lower      int32  `xml:"lowerLevel,attr"`
	Upper      int32  `xml:"upperLevel,attr"`
	Magic      string `xml:"isMagicClass,attr"`
}

func loadNewbieBuffXML(path string) error {
	var root xmlNewbieBuffList
	if err := readXML(path, &root); err != nil {
		return err
	}
	next := make([]NewbieBuff, 0, len(root.Buffs))
	for _, b := range root.Buffs {
		next = append(next, NewbieBuff{
			SkillID: b.SkillID, SkillLevel: b.SkillLevel,
			LowerLevel: b.Lower, UpperLevel: b.Upper,
			IsMagicClass: parseBool(b.Magic),
		})
	}
	extraPlayerMu.Lock()
	newbieBuffs = next
	extraPlayerMu.Unlock()
	return nil
}

type xmlSkillTreeRoot struct {
	Fishing []xmlFishingSkill `xml:"fishingSkill"`
	Clan    []xmlClanSkill    `xml:"clanSkill"`
	Enchant []xmlEnchantSkill `xml:"enchantSkill"`
}

type xmlFishingSkill struct {
	ID        int32 `xml:"id,attr"`
	Lvl       int32 `xml:"lvl,attr"`
	MinLvl    int32 `xml:"minLvl,attr"`
	ItemID    int32 `xml:"itemId,attr"`
	ItemCount int32 `xml:"itemCount,attr"`
}

type xmlClanSkill struct {
	ID     int32 `xml:"id,attr"`
	Lvl    int32 `xml:"lvl,attr"`
	Cost   int32 `xml:"cost,attr"`
	MinLvl int32 `xml:"minLvl,attr"`
	ItemID int32 `xml:"itemId,attr"`
}

type xmlEnchantSkill struct {
	ID         int32  `xml:"id,attr"`
	Lvl        int32  `xml:"lvl,attr"`
	Exp        int64  `xml:"exp,attr"`
	SP         int32  `xml:"sp,attr"`
	Rate76     int32  `xml:"rate76,attr"`
	Rate77     int32  `xml:"rate77,attr"`
	Rate78     int32  `xml:"rate78,attr"`
	Rate79     int32  `xml:"rate79,attr"`
	Rate80     int32  `xml:"rate80,attr"`
	ItemNeeded string `xml:"itemNeeded,attr"`
}

func loadSkillTreeXML(dir string) error {
	var fish []FishingSkillNode
	var clan []ClanSkillNode
	var ench []EnchantSkillNode
	err := walkXMLFiles(dir, func(_ string, body []byte) error {
		var root xmlSkillTreeRoot
		if err := xml.Unmarshal(body, &root); err != nil {
			return err
		}
		for _, n := range root.Fishing {
			fish = append(fish, FishingSkillNode{ID: n.ID, Level: n.Lvl, MinLvl: n.MinLvl, ItemID: n.ItemID, ItemCount: n.ItemCount})
		}
		for _, n := range root.Clan {
			clan = append(clan, ClanSkillNode{ID: n.ID, Level: n.Lvl, Cost: n.Cost, MinLvl: n.MinLvl, ItemID: n.ItemID})
		}
		for _, n := range root.Enchant {
			row := EnchantSkillNode{
				ID: n.ID, Level: n.Lvl, Exp: n.Exp, SP: n.SP,
				Rate76: n.Rate76, Rate77: n.Rate77, Rate78: n.Rate78, Rate79: n.Rate79, Rate80: n.Rate80,
			}
			if id, cnt, ok := parseSkillRef(n.ItemNeeded); ok {
				row.ItemNeeded = [2]int32{id, cnt}
			}
			ench = append(ench, row)
		}
		return nil
	})
	if err != nil {
		return err
	}
	extraPlayerMu.Lock()
	fishingSkills, clanSkills, enchantSkills = fish, clan, ench
	extraPlayerMu.Unlock()
	return nil
}

func readXML(path string, dest any) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := xml.Unmarshal(body, dest); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}
