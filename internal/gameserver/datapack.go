package gameserver

// Interlude newbie datapack used when XML data/ is not present.
// Starter items/skills match aCis PlayerData class templates.

type NewbieItem struct {
	ItemID   int32
	Count    int32
	Equipped bool
	Type1    int16
	Type2    int16
	BodyPart int32
}

type ClassKit struct {
	Skills []Skill
	Items  []NewbieItem
}

var classKits = map[int32]ClassKit{
	0: { // Human Fighter
		Skills: []Skill{{ID: 194, Level: 1, Passive: true}, {ID: 3, Level: 1}, {ID: 56, Level: 1}, {ID: 226, Level: 1}},
		Items: []NewbieItem{
			{2369, 1, true, 0, 0, 0x0080}, // Squire's Sword
			{1146, 1, true, 0, 0, 0x0400}, // Squire's Shirt
			{1147, 1, true, 0, 0, 0x0800}, // Squire's Pants
			{1148, 1, true, 0, 0, 0x1000}, // Squire's Shoes
			{5588, 1, false, 0, 0, 0},     // Tutorial Guide
			{57, 10000, false, 0, 0, 0},   // Adena
		},
	},
	10: { // Human Mystic
		Skills: []Skill{{ID: 194, Level: 1, Passive: true}, {ID: 1177, Level: 1}, {ID: 1011, Level: 1}, {ID: 1184, Level: 1}},
		Items: []NewbieItem{
			{99, 1, true, 0, 0, 0x0080}, // Apprentice's Wand
			{425, 1, true, 0, 0, 0x0400},
			{461, 1, true, 0, 0, 0x0800},
			{5588, 1, false, 0, 0, 0},
			{57, 10000, false, 0, 0, 0},
		},
	},
	18: { // Elven Fighter
		Skills: []Skill{{ID: 194, Level: 1, Passive: true}, {ID: 3, Level: 1}, {ID: 56, Level: 1}},
		Items: []NewbieItem{
			{2369, 1, true, 0, 0, 0x0080},
			{1146, 1, true, 0, 0, 0x0400},
			{1147, 1, true, 0, 0, 0x0800},
			{1148, 1, true, 0, 0, 0x1000},
			{5588, 1, false, 0, 0, 0},
			{57, 10000, false, 0, 0, 0},
		},
	},
	25: { // Elven Mystic
		Skills: []Skill{{ID: 194, Level: 1, Passive: true}, {ID: 1177, Level: 1}, {ID: 1011, Level: 1}},
		Items: []NewbieItem{
			{99, 1, true, 0, 0, 0x0080},
			{425, 1, true, 0, 0, 0x0400},
			{461, 1, true, 0, 0, 0x0800},
			{5588, 1, false, 0, 0, 0},
			{57, 10000, false, 0, 0, 0},
		},
	},
	31: { // Dark Fighter
		Skills: []Skill{{ID: 194, Level: 1, Passive: true}, {ID: 3, Level: 1}, {ID: 56, Level: 1}},
		Items: []NewbieItem{
			{2370, 1, true, 0, 0, 0x0080},
			{1146, 1, true, 0, 0, 0x0400},
			{1147, 1, true, 0, 0, 0x0800},
			{1148, 1, true, 0, 0, 0x1000},
			{5588, 1, false, 0, 0, 0},
			{57, 10000, false, 0, 0, 0},
		},
	},
	38: { // Dark Mystic
		Skills: []Skill{{ID: 194, Level: 1, Passive: true}, {ID: 1177, Level: 1}, {ID: 1011, Level: 1}},
		Items: []NewbieItem{
			{99, 1, true, 0, 0, 0x0080},
			{425, 1, true, 0, 0, 0x0400},
			{461, 1, true, 0, 0, 0x0800},
			{5588, 1, false, 0, 0, 0},
			{57, 10000, false, 0, 0, 0},
		},
	},
	44: { // Orc Fighter
		Skills: []Skill{{ID: 194, Level: 1, Passive: true}, {ID: 3, Level: 1}, {ID: 1001, Level: 1}},
		Items: []NewbieItem{
			{2371, 1, true, 0, 0, 0x0080},
			{1146, 1, true, 0, 0, 0x0400},
			{1147, 1, true, 0, 0, 0x0800},
			{1148, 1, true, 0, 0, 0x1000},
			{5588, 1, false, 0, 0, 0},
			{57, 10000, false, 0, 0, 0},
		},
	},
	49: { // Orc Mystic
		Skills: []Skill{{ID: 194, Level: 1, Passive: true}, {ID: 1001, Level: 1}, {ID: 1011, Level: 1}},
		Items: []NewbieItem{
			{99, 1, true, 0, 0, 0x0080},
			{425, 1, true, 0, 0, 0x0400},
			{461, 1, true, 0, 0, 0x0800},
			{5588, 1, false, 0, 0, 0},
			{57, 10000, false, 0, 0, 0},
		},
	},
	53: { // Dwarven Fighter
		Skills: []Skill{{ID: 194, Level: 1, Passive: true}, {ID: 1320, Level: 1, Passive: true}, {ID: 1321, Level: 1, Passive: true}, {ID: 3, Level: 1}},
		Items: []NewbieItem{
			{2372, 1, true, 0, 0, 0x0080},
			{1146, 1, true, 0, 0, 0x0400},
			{1147, 1, true, 0, 0, 0x0800},
			{1148, 1, true, 0, 0, 0x1000},
			{5588, 1, false, 0, 0, 0},
			{57, 10000, false, 0, 0, 0},
		},
	},
}

func ApplyStarterKit(ch *Character, nextItemID func() int32) {
	items := []NewbieItem(nil)
	if tpl := GetClassTemplate(ch.ClassID); tpl != nil && len(tpl.Items) > 0 {
		items = tpl.Items
	} else if kit, ok := classKits[ch.ClassID]; ok {
		items = kit.Items
	} else {
		items = classKits[0].Items
	}

	if DatapackLoaded() {
		ch.Skills = NodesToSkills(AutoGetSkills(ch.ClassID, ch.Level))
	} else {
		kit, ok := classKits[ch.ClassID]
		if !ok {
			kit = classKits[0]
		}
		ch.Skills = append([]Skill(nil), kit.Skills...)
	}

	ch.Items = ch.Items[:0]
	slot := int32(0)
	fallback := int32(0)
	for _, it := range items {
		oid := int32(0)
		if nextItemID != nil {
			oid = nextItemID()
		}
		if oid == 0 {
			fallback++
			oid = ch.ObjectID + 10000 + fallback
		}
		loc := "INVENTORY"
		if it.Equipped {
			loc = "PAPERDOLL"
		}
		item := Item{
			ObjectID: oid, ItemID: it.ItemID, Count: it.Count,
			Equipped: it.Equipped, Type1: it.Type1, Type2: it.Type2,
			BodyPart: it.BodyPart, Slot: slot, Loc: loc, ManaLeft: -1,
		}
		ApplyItemTemplate(&item)
		ch.Items = append(ch.Items, item)
		if it.Equipped {
			EquipPaperdoll(ch, it.BodyPart, it.ItemID, oid)
		}
		slot++
	}
	applyNewbieShortcuts(ch)
}

func EquipPaperdoll(ch *Character, bodyPart, itemID, objectID int32) {
	var slot Paperdoll = -1
	switch bodyPart {
	case 0x0080:
		slot = PaperRHand
	case 0x0400:
		slot = PaperChest
	case 0x0800:
		slot = PaperLegs
	case 0x1000:
		slot = PaperFeet
	case 0x0040:
		slot = PaperHead
	case 0x0200:
		slot = PaperGloves
	}
	if slot < 0 {
		return
	}
	ch.PaperdollItem[slot] = itemID
	ch.PaperdollObj[slot] = objectID
}

func applyNewbieShortcuts(ch *Character) {
	// Java RequestCharacterCreate default bars.
	ch.Shortcuts = []Shortcut{
		{Slot: 0, Page: 0, Type: 3, ID: 2, Level: -1, CharacterType: 1},  // ACTION attack
		{Slot: 3, Page: 0, Type: 3, ID: 5, Level: -1, CharacterType: 1},  // ACTION take
		{Slot: 10, Page: 0, Type: 3, ID: 0, Level: -1, CharacterType: 1}, // ACTION sit
	}
	for _, it := range ch.Items {
		if it.ItemID == 5588 {
			ch.Shortcuts = append(ch.Shortcuts, Shortcut{Slot: 11, Page: 0, Type: 1, ID: it.ObjectID, Level: -1, CharacterType: 1})
		}
	}
	for _, sk := range ch.Skills {
		if sk.ID == 1001 || sk.ID == 1177 {
			ch.Shortcuts = append(ch.Shortcuts, Shortcut{Slot: 1, Page: 0, Type: 2, ID: sk.ID, Level: 1, CharacterType: 1})
		}
		if sk.ID == 1216 {
			ch.Shortcuts = append(ch.Shortcuts, Shortcut{Slot: 9, Page: 0, Type: 2, ID: sk.ID, Level: 1, CharacterType: 1})
		}
	}
}

// BodyPartForItem is used when loading items without XML ItemData.
func BodyPartForItem(itemID int32) int32 {
	switch itemID {
	case 2369, 2370, 2371, 2372, 99:
		return 0x0080
	case 1146, 425:
		return 0x0400
	case 1147, 461:
		return 0x0800
	case 1148:
		return 0x1000
	default:
		return 0
	}
}

// DefaultNewbieSpawns is used when the DB spawn table is empty.
var DefaultNewbieSpawns = []NPC{
	{NPCID: 30006, Name: "Roxxy", Title: "Gatekeeper", X: -71338, Y: 258271, Z: -3104, Level: 70, MaxHP: 10000, CurHP: 10000},
	{NPCID: 30001, Name: "Lector", Title: "Weapon Merchant", X: -71424, Y: 258191, Z: -3104, Level: 70, MaxHP: 8000, CurHP: 8000},
	{NPCID: 30002, Name: "Jackson", Title: "Armor Merchant", X: -71280, Y: 258191, Z: -3104, Level: 70, MaxHP: 8000, CurHP: 8000},
	{NPCID: 30003, Name: "Silvia", Title: "Accessory Merchant", X: -71338, Y: 258080, Z: -3104, Level: 70, MaxHP: 8000, CurHP: 8000},
	{NPCID: 30009, Name: "Newbie Guide", X: -71380, Y: 258400, Z: -3104, Level: 70, MaxHP: 8000, CurHP: 8000},
	{NPCID: 30031, Name: "Captain Bathis", Title: "Guard Captain", X: -72224, Y: 257788, Z: -3120, Level: 70, MaxHP: 12000, CurHP: 12000},
	{NPCID: 30054, Name: "Rant", Title: "Warehouse Keeper", X: -71220, Y: 258191, Z: -3104, Level: 70, MaxHP: 8000, CurHP: 8000},
	{NPCID: 30146, Name: "Mirabel", Title: "Gatekeeper", X: 45873, Y: 49688, Z: -3056, Level: 70, MaxHP: 10000, CurHP: 10000},
	{NPCID: 30134, Name: "Jasmine", Title: "Gatekeeper", X: 9690, Y: 15537, Z: -4570, Level: 70, MaxHP: 10000, CurHP: 10000},
	{NPCID: 30576, Name: "Tataru Zu Hestui", Title: "Gatekeeper", X: -45251, Y: -112400, Z: -240, Level: 70, MaxHP: 10000, CurHP: 10000},
	{NPCID: 30540, Name: "Wirphy", Title: "Gatekeeper", X: 115072, Y: -178176, Z: -880, Level: 70, MaxHP: 10000, CurHP: 10000},
	{NPCID: 20120, Name: "Wolf", X: -74720, Y: 245280, Z: -3616, Level: 4, MaxHP: 80, CurHP: 80, IsAttackable: true},
	{NPCID: 20120, Name: "Wolf", X: -75100, Y: 244900, Z: -3616, Level: 4, MaxHP: 80, CurHP: 80, IsAttackable: true},
	{NPCID: 20120, Name: "Wolf", X: -74300, Y: 245600, Z: -3616, Level: 4, MaxHP: 80, CurHP: 80, IsAttackable: true},
	{NPCID: 20481, Name: "Elder Keltir", X: -75600, Y: 246200, Z: -3648, Level: 3, MaxHP: 60, CurHP: 60, IsAttackable: true},
	{NPCID: 20481, Name: "Elder Keltir", X: -76000, Y: 245800, Z: -3648, Level: 3, MaxHP: 60, CurHP: 60, IsAttackable: true},
	{NPCID: 20006, Name: "Orc Archer", X: -77000, Y: 247000, Z: -3760, Level: 8, MaxHP: 160, CurHP: 160, IsAttackable: true},
	{NPCID: 20001, Name: "Gremlin", X: -73500, Y: 244500, Z: -3552, Level: 1, MaxHP: 40, CurHP: 40, IsAttackable: true},
	{NPCID: 20003, Name: "Goblin", X: -74000, Y: 246800, Z: -3680, Level: 5, MaxHP: 90, CurHP: 90, IsAttackable: true},
}

func (w *World) LoadDefaultSpawns() {
	for _, n := range DefaultNewbieSpawns {
		cp := n
		cp.ObjectID = w.NextID()
		ApplyNpcTemplate(&cp)
		cp.NpcDefaults()
		w.AddNPC(&cp)
	}
}
