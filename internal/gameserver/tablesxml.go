package gameserver

import (
	"encoding/xml"
	"sync"
)

type InstantTeleport struct {
	X, Y, Z int32
}

type Announcement struct {
	Message  string
	Critical bool
	Auto     bool
}

type Henna struct {
	SymbolID                     int32
	DyeID                        int32
	Price                        int32
	STR, CON, DEX, INT, WIT, MEN int32
	Classes                      []int32
}

type SummonItem struct {
	ItemID     int32
	NPCID      int32
	SummonType int32
}

type CursedWeapon struct {
	ItemID   int32
	SkillID  int32
	Name     string
	DropRate int32
	Duration int32
}

type SoulCrystal struct {
	Level   int32
	Initial int32
	Staged  int32
	Broken  int32
}

type Fish struct {
	ID    int32
	Level int32
	HP    int32
	Type  int32
	Group int32
}

type BufferSkill struct {
	Category string
	ID       int32
	Desc     string
}

type AccessLevel struct {
	Level int32
	Name  string
	IsGM  bool
}

type AdminCommand struct {
	Name        string
	AccessLevel int32
	Desc        string
}

type ScriptIndex struct {
	Path string
}

type AugmentSkill struct {
	ID         int32
	SkillID    int32
	SkillLevel int32
	Type       string
}

var (
	tableMu          sync.RWMutex
	instantTeles     = map[int32][]InstantTeleport{}
	announcements    []Announcement
	hennas           = map[int32]Henna{}
	summonItems      = map[int32]SummonItem{}
	cursedWeapons    []CursedWeapon
	soulCrystals     []SoulCrystal
	fishes           []Fish
	bufferSkills     []BufferSkill
	accessLevels     = map[int32]AccessLevel{}
	adminCommands    []AdminCommand
	scriptIndex      []ScriptIndex
	augmentSkills    []AugmentSkill
	augmentStatCount int
)

func InstantTeleportCount() int {
	tableMu.RLock()
	defer tableMu.RUnlock()
	n := 0
	for _, v := range instantTeles {
		n += len(v)
	}
	return n
}

func InstantTeleports(npcID int32) []InstantTeleport {
	tableMu.RLock()
	defer tableMu.RUnlock()
	return append([]InstantTeleport(nil), instantTeles[npcID]...)
}

func AnnouncementCount() int {
	tableMu.RLock()
	defer tableMu.RUnlock()
	return len(announcements)
}

func LoginAnnouncements() []Announcement {
	tableMu.RLock()
	defer tableMu.RUnlock()
	out := make([]Announcement, 0)
	for _, a := range announcements {
		if !a.Auto {
			out = append(out, a)
		}
	}
	return out
}

func HennaCount() int {
	tableMu.RLock()
	defer tableMu.RUnlock()
	return len(hennas)
}

func GetHenna(id int32) *Henna {
	tableMu.RLock()
	defer tableMu.RUnlock()
	h, ok := hennas[id]
	if !ok {
		return nil
	}
	cp := h
	return &cp
}

func SummonItemCount() int {
	tableMu.RLock()
	defer tableMu.RUnlock()
	return len(summonItems)
}

func GetSummonItem(itemID int32) *SummonItem {
	tableMu.RLock()
	defer tableMu.RUnlock()
	s, ok := summonItems[itemID]
	if !ok {
		return nil
	}
	cp := s
	return &cp
}

func CursedWeaponCount() int {
	tableMu.RLock()
	defer tableMu.RUnlock()
	return len(cursedWeapons)
}

func SoulCrystalCount() int {
	tableMu.RLock()
	defer tableMu.RUnlock()
	return len(soulCrystals)
}

func FishCount() int {
	tableMu.RLock()
	defer tableMu.RUnlock()
	return len(fishes)
}

func BufferSkillCount() int {
	tableMu.RLock()
	defer tableMu.RUnlock()
	return len(bufferSkills)
}

func AccessLevelCount() int {
	tableMu.RLock()
	defer tableMu.RUnlock()
	return len(accessLevels)
}

func AdminCommandCount() int {
	tableMu.RLock()
	defer tableMu.RUnlock()
	return len(adminCommands)
}

func ScriptIndexCount() int {
	tableMu.RLock()
	defer tableMu.RUnlock()
	return len(scriptIndex)
}

func AugmentSkillCount() int {
	tableMu.RLock()
	defer tableMu.RUnlock()
	return len(augmentSkills)
}

func GetAccessLevel(level int32) *AccessLevel {
	tableMu.RLock()
	defer tableMu.RUnlock()
	a, ok := accessLevels[level]
	if !ok {
		return nil
	}
	cp := a
	return &cp
}

type xmlInstantRoot struct {
	Lists []xmlInstantList `xml:"telPosList"`
}

type xmlInstantList struct {
	NpcID int32    `xml:"npcId,attr"`
	Locs  []xmlXYZ `xml:"loc"`
}

func loadInstantTeleportXML(path string) error {
	var root xmlInstantRoot
	if err := readXML(path, &root); err != nil {
		return err
	}
	next := map[int32][]InstantTeleport{}
	for _, list := range root.Lists {
		for _, loc := range list.Locs {
			next[list.NpcID] = append(next[list.NpcID], InstantTeleport{X: loc.X, Y: loc.Y, Z: loc.Z})
		}
	}
	tableMu.Lock()
	instantTeles = next
	tableMu.Unlock()
	return nil
}

type xmlAnnounceRoot struct {
	Rows []xmlAnnounce `xml:"announcement"`
}

type xmlAnnounce struct {
	Message  string `xml:"message,attr"`
	Critical string `xml:"critical,attr"`
	Auto     string `xml:"auto,attr"`
}

func loadAnnouncementXML(path string) error {
	var root xmlAnnounceRoot
	if err := readXML(path, &root); err != nil {
		return err
	}
	next := make([]Announcement, 0, len(root.Rows))
	for _, a := range root.Rows {
		if a.Message == "" {
			continue
		}
		next = append(next, Announcement{Message: a.Message, Critical: parseBool(a.Critical), Auto: parseBool(a.Auto)})
	}
	tableMu.Lock()
	announcements = next
	tableMu.Unlock()
	return nil
}

type xmlHennaRoot struct {
	Rows []xmlHenna `xml:"henna"`
}

type xmlHenna struct {
	SymbolID int32  `xml:"symbolId,attr"`
	DyeID    int32  `xml:"dyeId,attr"`
	Price    int32  `xml:"price,attr"`
	STR      int32  `xml:"STR,attr"`
	CON      int32  `xml:"CON,attr"`
	DEX      int32  `xml:"DEX,attr"`
	INT      int32  `xml:"INT,attr"`
	WIT      int32  `xml:"WIT,attr"`
	MEN      int32  `xml:"MEN,attr"`
	Classes  string `xml:"classes,attr"`
}

func loadHennaXML(path string) error {
	var root xmlHennaRoot
	if err := readXML(path, &root); err != nil {
		return err
	}
	next := map[int32]Henna{}
	for _, h := range root.Rows {
		next[h.SymbolID] = Henna{
			SymbolID: h.SymbolID, DyeID: h.DyeID, Price: h.Price,
			STR: h.STR, CON: h.CON, DEX: h.DEX, INT: h.INT, WIT: h.WIT, MEN: h.MEN,
			Classes: parseCSVInts(h.Classes),
		}
	}
	tableMu.Lock()
	hennas = next
	tableMu.Unlock()
	return nil
}

type xmlSummonRoot struct {
	Items []xmlSummonItem `xml:"item"`
}

type xmlSummonItem struct {
	ID         int32 `xml:"id,attr"`
	NPCID      int32 `xml:"npcId,attr"`
	SummonType int32 `xml:"summonType,attr"`
}

func loadSummonItemXML(path string) error {
	var root xmlSummonRoot
	if err := readXML(path, &root); err != nil {
		return err
	}
	next := map[int32]SummonItem{}
	for _, it := range root.Items {
		next[it.ID] = SummonItem{ItemID: it.ID, NPCID: it.NPCID, SummonType: it.SummonType}
	}
	tableMu.Lock()
	summonItems = next
	tableMu.Unlock()
	return nil
}

type xmlCursedRoot struct {
	Items []xmlCursed `xml:"item"`
}

type xmlCursed struct {
	ID       int32  `xml:"id,attr"`
	SkillID  int32  `xml:"skillId,attr"`
	Name     string `xml:"name,attr"`
	DropRate int32  `xml:"dropRate,attr"`
	Duration int32  `xml:"duration,attr"`
}

func loadCursedWeaponXML(path string) error {
	var root xmlCursedRoot
	if err := readXML(path, &root); err != nil {
		return err
	}
	next := make([]CursedWeapon, 0, len(root.Items))
	for _, it := range root.Items {
		next = append(next, CursedWeapon{ItemID: it.ID, SkillID: it.SkillID, Name: it.Name, DropRate: it.DropRate, Duration: it.Duration})
	}
	tableMu.Lock()
	cursedWeapons = next
	tableMu.Unlock()
	return nil
}

type xmlSoulRoot struct {
	Crystals []xmlSoulCrystal `xml:"crystals>crystal"`
}

type xmlSoulCrystal struct {
	Level   int32 `xml:"level,attr"`
	Initial int32 `xml:"initial,attr"`
	Staged  int32 `xml:"staged,attr"`
	Broken  int32 `xml:"broken,attr"`
}

func loadSoulCrystalXML(path string) error {
	var root xmlSoulRoot
	if err := readXML(path, &root); err != nil {
		return err
	}
	next := make([]SoulCrystal, 0, len(root.Crystals))
	for _, c := range root.Crystals {
		next = append(next, SoulCrystal{Level: c.Level, Initial: c.Initial, Staged: c.Staged, Broken: c.Broken})
	}
	tableMu.Lock()
	soulCrystals = next
	tableMu.Unlock()
	return nil
}

type xmlFishRoot struct {
	Rows []xmlFish `xml:"fish"`
}

type xmlFish struct {
	ID    int32 `xml:"id,attr"`
	Level int32 `xml:"level,attr"`
	HP    int32 `xml:"hp,attr"`
	Type  int32 `xml:"type,attr"`
	Group int32 `xml:"group,attr"`
}

func loadFishXML(path string) error {
	var root xmlFishRoot
	if err := readXML(path, &root); err != nil {
		return err
	}
	next := make([]Fish, 0, len(root.Rows))
	for _, f := range root.Rows {
		next = append(next, Fish{ID: f.ID, Level: f.Level, HP: f.HP, Type: f.Type, Group: f.Group})
	}
	tableMu.Lock()
	fishes = next
	tableMu.Unlock()
	return nil
}

type xmlBufferRoot struct {
	Cats []xmlBufferCat `xml:"category"`
}

type xmlBufferCat struct {
	Type  string           `xml:"type,attr"`
	Buffs []xmlBufferSkill `xml:"buff"`
}

type xmlBufferSkill struct {
	ID   int32  `xml:"id,attr"`
	Desc string `xml:"desc,attr"`
}

func loadBufferXML(path string) error {
	var root xmlBufferRoot
	if err := readXML(path, &root); err != nil {
		return err
	}
	var next []BufferSkill
	for _, c := range root.Cats {
		for _, b := range c.Buffs {
			next = append(next, BufferSkill{Category: c.Type, ID: b.ID, Desc: b.Desc})
		}
	}
	tableMu.Lock()
	bufferSkills = next
	tableMu.Unlock()
	return nil
}

type xmlAccessRoot struct {
	Rows []xmlAccess `xml:"access"`
}

type xmlAccess struct {
	Level int32  `xml:"level,attr"`
	Name  string `xml:"name,attr"`
	IsGM  string `xml:"isGM,attr"`
}

func loadAccessLevelXML(path string) error {
	var root xmlAccessRoot
	if err := readXML(path, &root); err != nil {
		return err
	}
	next := map[int32]AccessLevel{}
	for _, a := range root.Rows {
		next[a.Level] = AccessLevel{Level: a.Level, Name: a.Name, IsGM: parseBool(a.IsGM)}
	}
	tableMu.Lock()
	accessLevels = next
	tableMu.Unlock()
	return nil
}

type xmlAdminRoot struct {
	Rows []xmlAdmin `xml:"aCar"`
}

type xmlAdmin struct {
	Name        string `xml:"name,attr"`
	AccessLevel int32  `xml:"accessLevel,attr"`
	Desc        string `xml:"desc,attr"`
}

func loadAdminCommandXML(path string) error {
	var root xmlAdminRoot
	if err := readXML(path, &root); err != nil {
		return err
	}
	next := make([]AdminCommand, 0, len(root.Rows))
	for _, a := range root.Rows {
		next = append(next, AdminCommand{Name: a.Name, AccessLevel: a.AccessLevel, Desc: a.Desc})
	}
	tableMu.Lock()
	adminCommands = next
	tableMu.Unlock()
	return nil
}

type xmlScriptRoot struct {
	Rows []xmlScript `xml:"script"`
}

type xmlScript struct {
	Path string `xml:"path,attr"`
}

func loadScriptIndexXML(path string) error {
	var root xmlScriptRoot
	if err := readXML(path, &root); err != nil {
		return err
	}
	next := make([]ScriptIndex, 0, len(root.Rows))
	for _, s := range root.Rows {
		if s.Path != "" {
			next = append(next, ScriptIndex{Path: s.Path})
		}
	}
	tableMu.Lock()
	scriptIndex = next
	tableMu.Unlock()
	return nil
}

type xmlAugmentFile struct {
	Skills []xmlAugment `xml:"augmentation"`
	Sets   []xmlAugSet  `xml:"set"`
}

type xmlAugment struct {
	ID         int32  `xml:"id,attr"`
	SkillID    int32  `xml:"skillId,attr"`
	SkillLevel int32  `xml:"skillLevel,attr"`
	Type       string `xml:"type,attr"`
}

type xmlAugSet struct {
	Stats []xmlAugStat `xml:"stat"`
}

type xmlAugStat struct {
	Name string `xml:"name,attr"`
}

func loadAugmentationXML(dir string) error {
	var skills []AugmentSkill
	statCount := 0
	err := walkXMLFiles(dir, func(_ string, body []byte) error {
		var root xmlAugmentFile
		if err := xml.Unmarshal(body, &root); err != nil {
			return err
		}
		for _, s := range root.Skills {
			skills = append(skills, AugmentSkill{ID: s.ID, SkillID: s.SkillID, SkillLevel: s.SkillLevel, Type: s.Type})
		}
		for _, set := range root.Sets {
			statCount += len(set.Stats)
		}
		return nil
	})
	if err != nil {
		return err
	}
	tableMu.Lock()
	augmentSkills = skills
	augmentStatCount = statCount
	tableMu.Unlock()
	return nil
}
