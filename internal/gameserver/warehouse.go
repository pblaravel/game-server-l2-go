package gameserver

import (
	"strings"

	"github.com/pblaravel/game-server-l2-go/internal/packet"
)

const (
	warehouseSlots = 100
	warehouseFee   = 30
)

func isWarehouseNPC(npc *NPC) bool {
	if npc == nil {
		return false
	}
	if strings.Contains(strings.ToLower(npc.Type), "warehouse") {
		return true
	}
	if tpl := GetNpcTemplate(npc.NPCID); tpl != nil && strings.Contains(strings.ToLower(tpl.Type), "warehouse") {
		return true
	}
	return strings.Contains(strings.ToLower(npc.Title), "warehouse")
}

func depositableItems(p *Character) []Item {
	out := make([]Item, 0)
	for _, it := range p.Items {
		if it.Equipped {
			continue
		}
		if it.ItemID != AdenaID && !IsTradable(it.ItemID) {
			continue
		}
		out = append(out, it)
	}
	return out
}

func (s *Server) showWarehouseDeposit(c *GameClient, npc *NPC) {
	p := c.Player()
	if !isWarehouseNPC(npc) || !s.canInteract(p, npc) {
		c.Send(ActionFailed())
		return
	}
	c.Send(ActionFailed())
	c.Send(WarehouseDepositList(AdenaCount(p), depositableItems(p)))
}

func (s *Server) showWarehouseWithdraw(c *GameClient, npc *NPC) {
	p := c.Player()
	if !isWarehouseNPC(npc) || !s.canInteract(p, npc) {
		c.Send(ActionFailed())
		return
	}
	if len(p.Warehouse) == 0 {
		c.Send(SystemMessage(SMNoItemInWarehouse))
		return
	}
	c.Send(WarehouseWithdrawList(AdenaCount(p), p.Warehouse))
	c.Send(ActionFailed())
}

func (s *Server) onWarehouseDeposit(c *GameClient, r *packet.Reader) {
	p := c.Player()
	npc := s.world.GetNPC(c.target)
	if npc == nil || !isWarehouseNPC(npc) || !s.canInteract(p, npc) {
		c.Send(ActionFailed())
		return
	}
	if s.tradeOf(p.ObjectID) != nil {
		c.Send(SystemMessage(SMAlreadyTrading))
		return
	}
	count := r.ReadD()
	if count <= 0 || count > 100 {
		return
	}
	type req struct{ oid, cnt int32 }
	reqs := make([]req, 0, count)
	for i := int32(0); i < count; i++ {
		oid, cnt := r.ReadD(), r.ReadD()
		if oid < 1 || cnt < 1 {
			return
		}
		reqs = append(reqs, req{oid, cnt})
	}
	fee := int32(len(reqs)) * warehouseFee
	adena := AdenaCount(p)
	slots := int32(0)
	for _, req := range reqs {
		it := FindItem(p, req.oid)
		if it == nil || req.cnt > it.Count || it.Equipped {
			return
		}
		if it.ItemID == AdenaID {
			adena -= req.cnt
		}
		if !isStackable(it.ItemID) {
			slots += req.cnt
		} else if findWarehouseItemByID(p, it.ItemID) == nil {
			slots++
		}
	}
	if int32(len(p.Warehouse))+slots > warehouseSlots {
		c.Send(SystemMessage(SMExceededInputQty))
		return
	}
	if adena < fee || !ReduceAdena(p, fee) {
		c.Send(SystemMessage(SMNotEnoughAdenaFee))
		return
	}
	for _, req := range reqs {
		it := FindItem(p, req.oid)
		if it == nil {
			continue
		}
		moveItem(&p.Items, &p.Warehouse, req.oid, req.cnt, "WAREHOUSE", s.nextItemID)
	}
	c.Send(ItemList(p.Items, false))
	s.sendWeightAndStats(c)
	_ = s.store.Update(c.ctx(), p)
	c.logChange("warehouse deposit lines=%d fee=%d", len(reqs), fee)
}

func (s *Server) onWarehouseWithdraw(c *GameClient, r *packet.Reader) {
	p := c.Player()
	npc := s.world.GetNPC(c.target)
	if npc == nil || !isWarehouseNPC(npc) || !s.canInteract(p, npc) {
		c.Send(ActionFailed())
		return
	}
	if s.tradeOf(p.ObjectID) != nil {
		c.Send(SystemMessage(SMAlreadyTrading))
		return
	}
	count := r.ReadD()
	if count <= 0 || count > 100 {
		return
	}
	type req struct{ oid, cnt int32 }
	reqs := make([]req, 0, count)
	var weight, slots int32
	for i := int32(0); i < count; i++ {
		oid, cnt := r.ReadD(), r.ReadD()
		if oid < 1 || cnt < 1 {
			return
		}
		it := findWarehouseItem(p, oid)
		if it == nil || cnt > it.Count {
			return
		}
		weight += ItemWeight(it.ItemID) * cnt
		if !isStackable(it.ItemID) {
			slots += cnt
		} else if FindItemByID(p, it.ItemID) == nil {
			slots++
		}
		reqs = append(reqs, req{oid, cnt})
	}
	if p.CurrentWeight+weight > p.WeightLimit {
		c.Send(SystemMessage(SMWeightLimitExceeded))
		return
	}
	if int32(len(p.Items))+slots > p.InventoryLimit {
		c.Send(SystemMessage(SMSlotsFull))
		return
	}
	for _, req := range reqs {
		moveItem(&p.Warehouse, &p.Items, req.oid, req.cnt, "INVENTORY", s.nextItemID)
	}
	c.Send(ItemList(p.Items, false))
	s.sendWeightAndStats(c)
	_ = s.store.Update(c.ctx(), p)
	c.logChange("warehouse withdraw lines=%d", len(reqs))
}

func findWarehouseItem(p *Character, objectID int32) *Item {
	for i := range p.Warehouse {
		if p.Warehouse[i].ObjectID == objectID {
			return &p.Warehouse[i]
		}
	}
	return nil
}

func findWarehouseItemByID(p *Character, itemID int32) *Item {
	for i := range p.Warehouse {
		if p.Warehouse[i].ItemID == itemID {
			return &p.Warehouse[i]
		}
	}
	return nil
}
