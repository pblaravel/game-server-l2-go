package gameserver

import (
	"math"
	"strings"

	"github.com/pblaravel/game-server-l2-go/internal/packet"
)

// Java L2Skill.SKILL_CRYSTALLIZE.
const skillCrystallize int32 = 248

// Java Config enchant defaults (players.properties).
const (
	enchantChanceArmor            = 0.66
	enchantChanceWeaponMagic      = 0.4
	enchantChanceWeaponMagic15    = 0.2
	enchantChanceWeaponNonMagic   = 0.7
	enchantChanceWeaponNonMagic15 = 0.35
	enchantSafeMax                = 3
	enchantSafeMaxFull            = 4
)

type enchantScroll struct {
	weapon  bool
	blessed bool
	crystal bool
	grade   string
}

// Java AbstractEnchantPacket._scrolls.
var enchantScrolls = map[int32]enchantScroll{
	729: {true, false, false, "a"}, 947: {true, false, false, "b"}, 951: {true, false, false, "c"},
	955: {true, false, false, "d"}, 959: {true, false, false, "s"},
	730: {false, false, false, "a"}, 948: {false, false, false, "b"}, 952: {false, false, false, "c"},
	956: {false, false, false, "d"}, 960: {false, false, false, "s"},
	6569: {true, true, false, "a"}, 6571: {true, true, false, "b"}, 6573: {true, true, false, "c"},
	6575: {true, true, false, "d"}, 6577: {true, true, false, "s"},
	6570: {false, true, false, "a"}, 6572: {false, true, false, "b"}, 6574: {false, true, false, "c"},
	6576: {false, true, false, "d"}, 6578: {false, true, false, "s"},
	731: {true, false, true, "a"}, 949: {true, false, true, "b"}, 953: {true, false, true, "c"},
	957: {true, false, true, "d"}, 961: {true, false, true, "s"},
	732: {false, false, true, "a"}, 950: {false, false, true, "b"}, 954: {false, false, true, "c"},
	958: {false, false, true, "d"}, 962: {false, false, true, "s"},
}

// enchantRoll is Java Rnd.nextDouble; tests replace it.
var enchantRoll = func() float64 { return rndDouble() }

func GetEnchantScroll(itemID int32) (enchantScroll, bool) {
	s, ok := enchantScrolls[itemID]
	return s, ok
}

func crystalItemID(grade string) int32 {
	switch strings.ToLower(grade) {
	case "d":
		return 1458
	case "c":
		return 1459
	case "b":
		return 1460
	case "a":
		return 1461
	case "s":
		return 1462
	default:
		return 0
	}
}

func crystalGradeID(grade string) int {
	switch strings.ToLower(grade) {
	case "d":
		return 1
	case "c":
		return 2
	case "b":
		return 3
	case "a":
		return 4
	case "s":
		return 5
	default:
		return 0
	}
}

func crystalEnchantBonus(grade string, weapon bool) int32 {
	switch strings.ToLower(grade) {
	case "d":
		if weapon {
			return 90
		}
		return 11
	case "c":
		if weapon {
			return 45
		}
		return 6
	case "b":
		if weapon {
			return 67
		}
		return 11
	case "a":
		if weapon {
			return 144
		}
		return 19
	case "s":
		if weapon {
			return 250
		}
		return 25
	default:
		return 0
	}
}

func CrystalCountFor(tpl *ItemTemplate, enchant int16) int32 {
	if tpl == nil {
		return 0
	}
	base := tpl.CrystalCount
	if base <= 0 {
		return 0
	}
	weapon := tpl.Type2 == Type2Weapon
	bonus := crystalEnchantBonus(tpl.CrystalType, weapon)
	if enchant > 3 {
		if weapon {
			return base + bonus*(2*int32(enchant)-3)
		}
		return base + bonus*(3*int32(enchant)-6)
	}
	if enchant > 0 {
		if weapon {
			return base + bonus*int32(enchant)
		}
		return base + bonus*int32(enchant)
	}
	return base
}

func isEnchantable(item *Item) bool {
	if item == nil {
		return false
	}
	tpl := GetItem(item.ItemID)
	if tpl == nil {
		return false
	}
	if strings.EqualFold(tpl.Kind, "EtcItem") {
		return false
	}
	if tpl.Type2 != Type2Weapon && tpl.Type2 != Type2ShieldArmor && tpl.Type2 != Type2Accessory {
		return false
	}
	if item.Loc != "INVENTORY" && item.Loc != "PAPERDOLL" && !item.Equipped {
		return false
	}
	return true
}

func (s enchantScroll) validFor(item *Item) bool {
	if !isEnchantable(item) {
		return false
	}
	tpl := GetItem(item.ItemID)
	if tpl == nil {
		return false
	}
	weapon := tpl.Type2 == Type2Weapon
	if s.weapon != weapon {
		return false
	}
	return strings.EqualFold(s.grade, tpl.CrystalType)
}

func (s enchantScroll) chance(item *Item) float64 {
	if !s.validFor(item) {
		return -1
	}
	tpl := GetItem(item.ItemID)
	fullBody := tpl != nil && tpl.BodyPart == SlotFullArmor
	safe := int16(enchantSafeMax)
	if fullBody {
		safe = enchantSafeMaxFull
	}
	if item.Enchant < safe {
		return 1
	}
	if tpl != nil && tpl.Type2 != Type2Weapon {
		return math.Pow(enchantChanceArmor, float64(item.Enchant-2))
	}
	magic := tpl != nil && tpl.MAtk > 0
	if magic {
		if item.Enchant > 14 {
			return enchantChanceWeaponMagic15
		}
		return enchantChanceWeaponMagic
	}
	if item.Enchant > 14 {
		return enchantChanceWeaponNonMagic15
	}
	return enchantChanceWeaponNonMagic
}

func EnchantResult(result int32) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x81)
		w.WriteD(result)
	})
}

func ChooseInventoryItem(itemID int32) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x6f)
		w.WriteD(itemID)
	})
}

func (s *Server) beginEnchant(c *GameClient, scroll *Item) {
	c.activeEnchant = scroll.ObjectID
	c.Send(SystemMessage(SMSelectItemToEnchant))
	c.Send(ChooseInventoryItem(scroll.ItemID))
}

func (s *Server) onEnchantItem(c *GameClient, r *packet.Reader) {
	p := c.Player()
	objectID := r.ReadD()
	if objectID == 0 || c.activeEnchant == 0 {
		c.activeEnchant = 0
		return
	}
	if s.tradeOf(p.ObjectID) != nil || p.PrivateStore != 0 {
		c.activeEnchant = 0
		c.Send(EnchantResult(2))
		return
	}
	item := FindItem(p, objectID)
	scroll := FindItem(p, c.activeEnchant)
	if item == nil || scroll == nil {
		c.activeEnchant = 0
		c.Send(EnchantResult(2))
		return
	}
	tpl, ok := GetEnchantScroll(scroll.ItemID)
	if !ok || !tpl.validFor(item) || !isEnchantable(item) {
		c.Send(SystemMessage(SMInappropriateEnchant))
		c.activeEnchant = 0
		c.Send(EnchantResult(2))
		return
	}
	chance := tpl.chance(item)
	if !RemoveItemCount(p, scroll.ObjectID, 1) {
		c.Send(SystemMessage(SMNotEnoughItems))
		c.activeEnchant = 0
		c.Send(EnchantResult(2))
		return
	}
	if enchantRoll() < chance {
		if item.Enchant == 0 {
			c.Send(SystemMessage(SMSuccessfullyEnchanted, SysItem(item.ItemID)))
		} else {
			c.Send(SystemMessage(SMSuccessfullyEnchantedS1S2, SysNumber(int32(item.Enchant)), SysItem(item.ItemID)))
		}
		item.Enchant++
		c.Send(EnchantResult(0))
	} else if tpl.blessed {
		c.Send(SystemMessage(SMBlessedEnchantFailed))
		item.Enchant = 0
		c.Send(EnchantResult(3))
	} else {
		itemTpl := GetItem(item.ItemID)
		crystalID := int32(0)
		count := int32(1)
		if itemTpl != nil {
			crystalID = crystalItemID(itemTpl.CrystalType)
			base := itemTpl.CrystalCount
			count = CrystalCountFor(itemTpl, item.Enchant) - (base+1)/2
			if count < 1 {
				count = 1
			}
		}
		RemoveItemCount(p, item.ObjectID, item.Count)
		if crystalID != 0 {
			AddItem(p, crystalID, count, s.nextItemID)
			c.Send(SystemMessage(SMEarnedS2S1S, SysItem(crystalID), SysNumber(count)))
			c.Send(EnchantResult(1))
		} else {
			c.Send(EnchantResult(4))
		}
	}
	c.activeEnchant = 0
	s.refreshAppearance(c)
}

func (s *Server) onCrystallizeItem(c *GameClient, r *packet.Reader) {
	p := c.Player()
	objectID := r.ReadD()
	count := r.ReadD()
	if count <= 0 {
		return
	}
	level := SkillLevelOf(p, skillCrystallize)
	if level <= 0 {
		c.Send(SystemMessage(SMCrystallizeLevelTooLow))
		return
	}
	item := FindItem(p, objectID)
	if item == nil {
		return
	}
	tpl := GetItem(item.ItemID)
	if tpl == nil || tpl.CrystalCount <= 0 || crystalItemID(tpl.CrystalType) == 0 {
		return
	}
	grade := crystalGradeID(tpl.CrystalType)
	if grade >= 2 && level <= int32(grade-1) {
		c.Send(SystemMessage(SMCrystallizeLevelTooLow))
		c.Send(ActionFailed())
		return
	}
	if count > item.Count {
		count = item.Count
	}
	if item.Equipped && item.BodyPart != 0 {
		UnequipBodyPart(p, item.BodyPart)
	}
	crystals := CrystalCountFor(tpl, item.Enchant)
	if crystals < 1 {
		crystals = 1
	}
	RemoveItemCount(p, objectID, count)
	AddItem(p, crystalItemID(tpl.CrystalType), crystals, s.nextItemID)
	c.Send(SystemMessage(SMS1Crystallized, SysItem(item.ItemID)))
	c.Send(ItemList(p.Items, false))
	s.sendWeightAndStats(c)
	_ = s.store.Update(c.ctx(), p)
}
