package gameserver

// Inventory operations from Java model/actor/container/player/Inventory and
// PcInventory. Item weight and body parts come from ItemData XML in Java; that
// datapack is not vendored, so ItemWeight/BodyPartForItem provide the values for
// the starter items the server hands out.

// paperdollForBodyPart maps a Java Item.bodyPart mask to its paperdoll slot.
func paperdollForBodyPart(bodyPart int32) Paperdoll {
	switch bodyPart {
	case 0x01:
		return PaperHairAll
	case 0x02:
		return PaperRear
	case 0x04:
		return PaperLear
	case 0x08:
		return PaperNeck
	case 0x10:
		return PaperRFinger
	case 0x20:
		return PaperLFinger
	case 0x40:
		return PaperHead
	case 0x80:
		return PaperRHand
	case 0x100:
		return PaperLHand
	case 0x200:
		return PaperGloves
	case 0x400:
		return PaperChest
	case 0x800:
		return PaperLegs
	case 0x1000:
		return PaperFeet
	case 0x2000:
		return PaperCloak
	case 0x4000: // two handed weapon
		return PaperRHand
	case 0x8000:
		return PaperHair
	case 0x10000:
		return PaperFace
	default:
		return -1
	}
}

func FindItem(p *Character, objectID int32) *Item {
	for i := range p.Items {
		if p.Items[i].ObjectID == objectID {
			return &p.Items[i]
		}
	}
	return nil
}

func FindItemByID(p *Character, itemID int32) *Item {
	for i := range p.Items {
		if p.Items[i].ItemID == itemID {
			return &p.Items[i]
		}
	}
	return nil
}

// EquipItem is Java Inventory.equipItem: the previous item in the slot is
// unequipped first, then the paperdoll is updated.
func EquipItem(p *Character, objectID int32) bool {
	item := FindItem(p, objectID)
	if item == nil || item.BodyPart == 0 {
		return false
	}
	slot := paperdollForBodyPart(item.BodyPart)
	if slot < 0 {
		return false
	}
	if prev := p.PaperdollObj[slot]; prev != 0 && prev != objectID {
		unequipSlot(p, slot)
	}
	if item.BodyPart == 0x4000 { // two handed: the left hand must be free
		unequipSlot(p, PaperLHand)
	}
	item.Equipped = true
	item.Loc = "PAPERDOLL"
	item.LocData = int32(slot)
	p.PaperdollItem[slot] = item.ItemID
	p.PaperdollObj[slot] = item.ObjectID
	if slot == PaperRHand {
		p.AugmentRHand = item.Augment
	}
	if slot == PaperLHand {
		p.AugmentLHand = item.Augment
	}
	return true
}

// UnequipBodyPart is Java Inventory.unEquipItemInBodySlot.
func UnequipBodyPart(p *Character, bodyPart int32) bool {
	slot := paperdollForBodyPart(bodyPart)
	if slot < 0 {
		return false
	}
	return unequipSlot(p, slot)
}

func unequipSlot(p *Character, slot Paperdoll) bool {
	objectID := p.PaperdollObj[slot]
	if objectID == 0 {
		return false
	}
	if item := FindItem(p, objectID); item != nil {
		item.Equipped = false
		item.Loc = "INVENTORY"
		item.LocData = 0
	}
	p.PaperdollItem[slot] = 0
	p.PaperdollObj[slot] = 0
	if slot == PaperRHand {
		p.AugmentRHand = 0
	}
	if slot == PaperLHand {
		p.AugmentLHand = 0
	}
	return true
}

// RemoveItemCount is Java Inventory.destroyItem: stackables lose count, the rest
// leave the inventory.
func RemoveItemCount(p *Character, objectID, count int32) bool {
	for i := range p.Items {
		if p.Items[i].ObjectID != objectID {
			continue
		}
		if p.Items[i].Count > count {
			p.Items[i].Count -= count
			return true
		}
		if p.Items[i].Equipped {
			unequipSlot(p, paperdollForBodyPart(p.Items[i].BodyPart))
		}
		p.Items = append(p.Items[:i], p.Items[i+1:]...)
		return true
	}
	return false
}

// AddItem is Java Inventory.addItem; stackable ids merge into one entry.
func AddItem(p *Character, itemID, count int32, nextObjectID func() int32) *Item {
	if isStackable(itemID) {
		if existing := FindItemByID(p, itemID); existing != nil {
			existing.Count += count
			return existing
		}
	}
	objectID := int32(0)
	if nextObjectID != nil {
		objectID = nextObjectID()
	}
	slot := int32(0)
	for _, it := range p.Items {
		if it.Slot >= slot {
			slot = it.Slot + 1
		}
	}
	p.Items = append(p.Items, Item{
		ObjectID: objectID, ItemID: itemID, Count: count,
		BodyPart: BodyPartForItem(itemID), Loc: "INVENTORY", Slot: slot, ManaLeft: -1,
	})
	return &p.Items[len(p.Items)-1]
}

func isStackable(itemID int32) bool {
	// Adena and the newbie consumables are the stackables the server can hand out.
	switch itemID {
	case 57, 1147, 1146, 5588:
		return itemID == 57
	default:
		return false
	}
}

// ItemWeight is the subset of ItemData weights needed for the starter kits.
func ItemWeight(itemID int32) int32 {
	switch itemID {
	case 57: // adena
		return 0
	case 2369, 2370, 2371, 2372: // squire's weapons
		return 1500
	case 99: // apprentice's wand
		return 1200
	case 1146, 425: // shirts
		return 430
	case 1147, 461: // pants
		return 240
	case 1148: // shoes
		return 250
	case 5588: // tutorial guide
		return 120
	default:
		return 100
	}
}

// CurrentWeight is Java Inventory.refreshWeight.
func CurrentWeight(p *Character) int32 {
	total := int32(0)
	for _, it := range p.Items {
		total += ItemWeight(it.ItemID) * it.Count
	}
	return total
}
