package gameserver

import (
	"github.com/pblaravel/game-server-l2-go/internal/packet"
)

// Java Npc.INTERACTION_DISTANCE, used for drop and pickup range.
const itemInteractRange = 150.0

func (w *World) AddGroundItem(it *GroundItem) {
	if it == nil {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.items[it.ObjectID] = it
	w.objects[it.ObjectID] = it
}

func (w *World) RemoveGroundItem(id int32) *GroundItem {
	w.mu.Lock()
	defer w.mu.Unlock()
	it := w.items[id]
	delete(w.items, id)
	delete(w.objects, id)
	return it
}

func (w *World) GetGroundItem(id int32) *GroundItem {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.items[id]
}

func (w *World) GroundItems() []*GroundItem {
	w.mu.RLock()
	defer w.mu.RUnlock()
	out := make([]*GroundItem, 0, len(w.items))
	for _, it := range w.items {
		out = append(out, it)
	}
	return out
}

func (g *GroundItem) isStackable() bool {
	if tpl := GetItem(g.ItemID); tpl != nil {
		return tpl.Stackable
	}
	return g.ItemID == AdenaID
}

// dropPlayerItem is Java Player.dropItem(objectId, count, x, y, z).
func (s *Server) dropPlayerItem(c *GameClient, objectID, count, x, y, z int32) *GroundItem {
	p := c.Player()
	item := FindItem(p, objectID)
	if item == nil || count <= 0 || count > item.Count {
		return nil
	}
	enchant := item.Enchant
	itemID := item.ItemID
	groundOID := objectID
	if item.Count > count {
		item.Count -= count
		groundOID = s.nextItemID()
	} else {
		if item.Equipped && item.BodyPart != 0 {
			UnequipBodyPart(p, item.BodyPart)
		}
		RemoveItemCount(p, objectID, count)
	}
	g := &GroundItem{
		ObjectID: groundOID, ItemID: itemID, Count: count, Enchant: enchant,
		X: x, Y: y, Z: z, Dropper: p.ObjectID,
	}
	s.world.AddGroundItem(g)
	return g
}

func (s *Server) broadcastGroundItem(g *GroundItem, except *GameClient) {
	pkt := DropItem(g)
	if except != nil {
		except.Send(pkt)
	}
	s.Broadcast(pkt, except)
}

func (s *Server) pickupGroundItem(c *GameClient, objectID int32) bool {
	p := c.Player()
	g := s.world.GetGroundItem(objectID)
	if g == nil {
		return false
	}
	if Distance3D(p.X, p.Y, p.Z, g.X, g.Y, g.Z) > itemInteractRange {
		c.Send(SystemMessage(SMTargetTooFar))
		c.Send(ActionFailed())
		return false
	}
	if g.OwnerID != 0 && g.OwnerID != p.ObjectID {
		c.Send(ActionFailed())
		return false
	}
	weight := int64(ItemWeight(g.ItemID)) * int64(g.Count)
	if p.CurrentWeight+int32(weight) > p.WeightLimit {
		c.Send(SystemMessage(SMWeightLimitExceeded))
		c.Send(ActionFailed())
		return false
	}
	if !isStackable(g.ItemID) || FindItemByID(p, g.ItemID) == nil {
		if int32(len(p.Items))+1 > p.InventoryLimit {
			c.Send(SystemMessage(SMSlotsFull))
			c.Send(ActionFailed())
			return false
		}
	}
	if s.world.RemoveGroundItem(objectID) == nil {
		c.Send(ActionFailed())
		return false
	}
	added := AddItem(p, g.ItemID, g.Count, s.nextItemID)
	if added != nil && g.Enchant > 0 {
		added.Enchant = g.Enchant
	}
	get := GetItemPacket(p.ObjectID, g)
	del := DeleteObject(g.ObjectID)
	c.Send(get)
	c.Broadcast(get)
	c.Send(del)
	s.Broadcast(del, c)
	if g.Count > 1 {
		c.Send(SystemMessage(SMPickedUpS2S1, SysItemCount(g.Count), SysItem(g.ItemID)))
	} else {
		c.Send(SystemMessage(SMPickedUpS1, SysItem(g.ItemID)))
	}
	c.Send(ItemList(p.Items, false))
	s.sendWeightAndStats(c)
	_ = s.store.Update(c.ctx(), p)
	c.logChange("picked up item=%d count=%d oid=%d", g.ItemID, g.Count, g.ObjectID)
	return true
}

func (s *Server) sendNearbyGroundItems(c *GameClient) {
	for _, g := range s.world.GroundItems() {
		c.Send(DropItem(g))
	}
}

func DropItem(g *GroundItem) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x0c)
		w.WriteD(g.Dropper)
		w.WriteD(g.ObjectID)
		w.WriteD(g.ItemID)
		w.WriteD(g.X)
		w.WriteD(g.Y)
		w.WriteD(g.Z)
		if g.isStackable() {
			w.WriteD(1)
		} else {
			w.WriteD(0)
		}
		w.WriteD(g.Count)
		w.WriteD(1)
	})
}

func SpawnItem(g *GroundItem) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x0b)
		w.WriteD(g.ObjectID)
		w.WriteD(g.ItemID)
		w.WriteD(g.X)
		w.WriteD(g.Y)
		w.WriteD(g.Z)
		if g.isStackable() {
			w.WriteD(1)
		} else {
			w.WriteD(0)
		}
		w.WriteD(g.Count)
		w.WriteD(0)
	})
}

func GetItemPacket(playerID int32, g *GroundItem) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x0d)
		w.WriteD(playerID)
		w.WriteD(g.ObjectID)
		w.WriteD(g.X)
		w.WriteD(g.Y)
		w.WriteD(g.Z)
	})
}
