package gameserver

import (
	"github.com/pblaravel/game-server-l2-go/internal/packet"
)

// Java MultisellData.PAGE_SIZE.
const multiSellPageSize = 40

const clanReputationItem int32 = 65336

func MultiSellList(list *MultisellList, index int) []byte {
	if list == nil {
		return nil
	}
	size := len(list.Entries) - index
	finished := 1
	if size > multiSellPageSize {
		size = multiSellPageSize
		finished = 0
	}
	if size < 0 {
		size = 0
	}
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0xd0)
		w.WriteD(list.ID)
		w.WriteD(int32(1 + index/multiSellPageSize))
		w.WriteD(int32(finished))
		w.WriteD(multiSellPageSize)
		w.WriteD(int32(size))
		for i := 0; i < size; i++ {
			ent := list.Entries[index+i]
			w.WriteD(int32(index + i + 1))
			w.WriteD(0)
			w.WriteD(0)
			stackable := 0
			if len(ent.Products) > 0 && isStackable(ent.Products[0].ItemID) {
				stackable = 1
			}
			w.WriteC(stackable)
			w.WriteH(len(ent.Products))
			w.WriteH(len(ent.Ingredients))
			for _, ing := range ent.Products {
				w.WriteH(int(ing.ItemID))
				if tpl := GetItem(ing.ItemID); tpl != nil {
					w.WriteD(tpl.BodyPart)
					w.WriteH(int(tpl.Type2))
				} else {
					w.WriteD(0)
					w.WriteH(65535)
				}
				w.WriteD(ing.Count)
				w.WriteH(int(ing.Enchant))
				w.WriteD(0)
				w.WriteD(0)
			}
			for _, ing := range ent.Ingredients {
				w.WriteH(int(ing.ItemID))
				if tpl := GetItem(ing.ItemID); tpl != nil {
					w.WriteH(int(tpl.Type2))
				} else {
					w.WriteH(65535)
				}
				w.WriteD(ing.Count)
				w.WriteH(int(ing.Enchant))
				w.WriteD(0)
				w.WriteD(0)
			}
		}
	})
}

func (s *Server) showMultisell(c *GameClient, list *MultisellList) {
	if list == nil || len(list.Entries) == 0 {
		c.Send(ActionFailed())
		return
	}
	c.multiSellID = list.ID
	for i := 0; i < len(list.Entries); i += multiSellPageSize {
		if pkt := MultiSellList(list, i); pkt != nil {
			c.Send(pkt)
		}
	}
}

func (s *Server) onMultiSellChoose(c *GameClient, r *packet.Reader) {
	listID := r.ReadD()
	entryID := r.ReadD()
	amount := r.ReadD()
	s.chooseMultisell(c, listID, entryID, amount)
}

func (s *Server) chooseMultisell(c *GameClient, listID, entryID, amount int32) {
	p := c.Player()
	if amount < 1 || amount > 9999 {
		c.multiSellID = 0
		return
	}
	if c.multiSellID != 0 && c.multiSellID != listID {
		c.multiSellID = 0
		return
	}
	list := GetMultisell(listID)
	if list == nil || entryID < 1 || int(entryID) > len(list.Entries) {
		c.multiSellID = 0
		return
	}
	npc := s.world.GetNPC(c.target)
	if npc != nil && !s.canInteract(p, npc) {
		c.multiSellID = 0
		c.Send(ActionFailed())
		return
	}
	if len(list.NpcIDs) > 0 {
		ok := npc != nil && npcAllowed(list, npc.NPCID)
		if !ok {
			c.multiSellID = 0
			c.Send(ActionFailed())
			return
		}
	}
	entry := list.Entries[entryID-1]
	if !entryStackable(entry) && amount > 1 {
		c.multiSellID = 0
		return
	}
	var slots, weight int64
	for _, prod := range entry.Products {
		if prod.ItemID < 0 {
			continue
		}
		if !isStackable(prod.ItemID) {
			slots += int64(prod.Count) * int64(amount)
		} else if FindItemByID(p, prod.ItemID) == nil {
			slots++
		}
		weight += int64(prod.Count) * int64(amount) * int64(ItemWeight(prod.ItemID))
	}
	if p.CurrentWeight+int32(weight) > p.WeightLimit {
		c.Send(SystemMessage(SMWeightLimitExceeded))
		return
	}
	if int32(len(p.Items))+int32(slots) > p.InventoryLimit {
		c.Send(SystemMessage(SMSlotsFull))
		return
	}
	needed := mergeIngredients(entry.Ingredients)
	for _, ing := range needed {
		if ing.ItemID == clanReputationItem {
			c.Send(SystemMessage(SMNotEnoughItems))
			return
		}
		if ItemCountOf(p, ing.ItemID) < ing.Count*amount {
			c.Send(SystemMessage(SMNotEnoughItems))
			return
		}
	}
	for _, ing := range needed {
		if !RemoveItemByID(p, ing.ItemID, ing.Count*amount) {
			c.Send(ActionFailed())
			return
		}
	}
	for _, prod := range entry.Products {
		if prod.ItemID < 0 {
			continue
		}
		added := AddItem(p, prod.ItemID, prod.Count*amount, s.nextItemID)
		if added != nil && list.MaintainEnchantment && prod.Enchant > 0 {
			added.Enchant = int16(prod.Enchant)
		}
		total := prod.Count * amount
		if total > 1 {
			c.Send(SystemMessage(SMEarnedS2S1S, SysItem(prod.ItemID), SysNumber(total)))
		} else {
			c.Send(SystemMessage(SMEarnedItemS1, SysItem(prod.ItemID)))
		}
	}
	c.Send(SystemMessage(SMSuccessfullyTradedWithNPC))
	c.Send(ItemList(p.Items, true))
	s.sendWeightAndStats(c)
	_ = s.store.Update(c.ctx(), p)
	c.logChange("multisell list=%d entry=%d amount=%d", listID, entryID, amount)
}

func npcAllowed(list *MultisellList, npcID int32) bool {
	for _, id := range list.NpcIDs {
		if id == npcID {
			return true
		}
	}
	return false
}

func entryStackable(e MultisellEntry) bool {
	if len(e.Products) == 0 {
		return false
	}
	return isStackable(e.Products[0].ItemID)
}

func mergeIngredients(in []MultisellIngredient) []MultisellIngredient {
	out := make([]MultisellIngredient, 0, len(in))
	for _, e := range in {
		merged := false
		for i := range out {
			if out[i].ItemID == e.ItemID && out[i].Enchant == e.Enchant {
				out[i].Count += e.Count
				merged = true
				break
			}
		}
		if !merged {
			out = append(out, e)
		}
	}
	return out
}
