package gameserver

// Inventory operations from Java model/actor/container/player/Inventory and
// PcInventory. Weight, body part and stackable flags come from ItemData XML.

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
	case SlotLRHand:
		return PaperRHand
	case SlotFullArmor:
		return PaperChest
	case SlotHair:
		return PaperHair
	case SlotFace:
		return PaperFace
	case SlotHairAll:
		return PaperHairAll
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
	item := Item{
		ObjectID: objectID, ItemID: itemID, Count: count,
		BodyPart: BodyPartForItem(itemID), Loc: "INVENTORY", Slot: slot, ManaLeft: -1,
	}
	ApplyItemTemplate(&item)
	p.Items = append(p.Items, item)
	return &p.Items[len(p.Items)-1]
}

func isStackable(itemID int32) bool {
	if tpl := GetItem(itemID); tpl != nil {
		return tpl.Stackable
	}
	return itemID == AdenaID
}

// ItemWeight is Java Item.getWeight, with starter-kit fallbacks when XML is absent.
func ItemWeight(itemID int32) int32 {
	if tpl := GetItem(itemID); tpl != nil {
		return tpl.Weight
	}
	switch itemID {
	case AdenaID:
		return 0
	case 2369, 2370, 2371, 2372:
		return 1500
	case 99:
		return 1200
	case 1146, 425:
		return 430
	case 1147, 461:
		return 240
	case 1148:
		return 250
	case 5588:
		return 120
	default:
		return 100
	}
}

func AdenaCount(p *Character) int32 {
	if it := FindItemByID(p, AdenaID); it != nil {
		return it.Count
	}
	return 0
}

func ReduceAdena(p *Character, amount int32) bool {
	if amount <= 0 {
		return true
	}
	it := FindItemByID(p, AdenaID)
	if it == nil || it.Count < amount {
		return false
	}
	return RemoveItemCount(p, it.ObjectID, amount)
}

func AddAdena(p *Character, amount int32, nextObjectID func() int32) {
	if amount <= 0 {
		return
	}
	AddItem(p, AdenaID, amount, nextObjectID)
}

func IsSellable(itemID int32) bool {
	if itemID == AdenaID {
		return false
	}
	if tpl := GetItem(itemID); tpl != nil {
		return tpl.Sellable && tpl.Price > 0
	}
	return itemID != AdenaID
}

func ReferencePrice(itemID int32) int32 {
	if tpl := GetItem(itemID); tpl != nil {
		return tpl.Price
	}
	return 0
}

// CurrentWeight is Java Inventory.refreshWeight.
func CurrentWeight(p *Character) int32 {
	total := int32(0)
	for _, it := range p.Items {
		total += ItemWeight(it.ItemID) * it.Count
	}
	return total
}
