package gameserver

import (
	"github.com/pblaravel/game-server-l2-go/internal/packet"
)

// Java Henna.DRAW_AMOUNT / REMOVE_AMOUNT.
const (
	hennaDrawAmount   int32 = 10
	hennaRemoveAmount int32 = 5
	hennaMaxSlots     int32 = 3
)

func hennaCanBeUsedBy(h Henna, classID int32) bool {
	if len(h.Classes) == 0 {
		return true
	}
	for _, id := range h.Classes {
		if id == classID {
			return true
		}
	}
	return false
}

func hennaSlotsUsed(p *Character) int {
	n := 0
	for _, id := range p.Hennas {
		if id != 0 {
			n++
		}
	}
	return n
}

func applyHennaStats(p *Character) {
	for _, id := range p.Hennas {
		if id == 0 {
			continue
		}
		h := GetHenna(id)
		if h == nil {
			continue
		}
		p.STR += h.STR
		p.CON += h.CON
		p.DEX += h.DEX
		p.INT += h.INT
		p.WIT += h.WIT
		p.MEN += h.MEN
	}
}

func hennaStatSum(p *Character) (int, int, int, int, int, int) {
	var i, s, c, m, d, w int
	for _, id := range p.Hennas {
		if id == 0 {
			continue
		}
		h := GetHenna(id)
		if h == nil {
			continue
		}
		i += int(h.INT)
		s += int(h.STR)
		c += int(h.CON)
		m += int(h.MEN)
		d += int(h.DEX)
		w += int(h.WIT)
	}
	return i, s, c, m, d, w
}

func availableHennas(p *Character) []Henna {
	tableMu.RLock()
	defer tableMu.RUnlock()
	out := make([]Henna, 0)
	for _, h := range hennas {
		if !hennaCanBeUsedBy(h, p.ClassID) {
			continue
		}
		if FindItemByID(p, h.DyeID) == nil {
			continue
		}
		out = append(out, h)
	}
	return out
}

func HennaInfo(p *Character) []byte {
	i, s, c, m, d, w := hennaStatSum(p)
	used := make([]int32, 0, 3)
	for _, id := range p.Hennas {
		if id != 0 {
			used = append(used, id)
		}
	}
	return gsWrite(func(wtr *packet.Writer) {
		wtr.WriteC(0xe4)
		wtr.WriteC(i)
		wtr.WriteC(s)
		wtr.WriteC(c)
		wtr.WriteC(m)
		wtr.WriteC(d)
		wtr.WriteC(w)
		wtr.WriteD(hennaMaxSlots)
		wtr.WriteD(int32(len(used)))
		for _, id := range used {
			wtr.WriteD(id)
			if h := GetHenna(id); h != nil && hennaCanBeUsedBy(*h, p.ClassID) {
				wtr.WriteD(id)
			} else {
				wtr.WriteD(0)
			}
		}
	})
}

func HennaEquipList(p *Character) []byte {
	list := availableHennas(p)
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0xe2)
		w.WriteD(AdenaCount(p))
		w.WriteD(hennaMaxSlots)
		w.WriteD(int32(len(list)))
		for _, h := range list {
			w.WriteD(h.SymbolID)
			w.WriteD(h.DyeID)
			w.WriteD(hennaDrawAmount)
			w.WriteD(h.Price)
			w.WriteD(1)
		}
	})
}

func HennaItemInfo(h Henna, p *Character) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0xe3)
		w.WriteD(h.SymbolID)
		w.WriteD(h.DyeID)
		w.WriteD(hennaDrawAmount)
		w.WriteD(h.Price)
		w.WriteD(1)
		w.WriteD(AdenaCount(p))
		w.WriteD(p.INT)
		w.WriteC(int(p.INT + h.INT))
		w.WriteD(p.STR)
		w.WriteC(int(p.STR + h.STR))
		w.WriteD(p.CON)
		w.WriteC(int(p.CON + h.CON))
		w.WriteD(p.MEN)
		w.WriteC(int(p.MEN + h.MEN))
		w.WriteD(p.DEX)
		w.WriteC(int(p.DEX + h.DEX))
		w.WriteD(p.WIT)
		w.WriteC(int(p.WIT + h.WIT))
	})
}

func HennaUnequipList(p *Character) []byte {
	used := make([]Henna, 0, 3)
	for _, id := range p.Hennas {
		if id == 0 {
			continue
		}
		if h := GetHenna(id); h != nil {
			used = append(used, *h)
		}
	}
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0xe5)
		w.WriteD(AdenaCount(p))
		w.WriteD(hennaMaxSlots - int32(len(used)))
		w.WriteD(int32(len(used)))
		for _, h := range used {
			w.WriteD(h.SymbolID)
			w.WriteD(h.DyeID)
			w.WriteD(hennaRemoveAmount)
			w.WriteD(h.Price / hennaRemoveAmount)
			w.WriteD(1)
		}
	})
}

func HennaItemUnequipInfo(h Henna, p *Character) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0xe6)
		w.WriteD(h.SymbolID)
		w.WriteD(h.DyeID)
		w.WriteD(hennaRemoveAmount)
		w.WriteD(h.Price / hennaRemoveAmount)
		w.WriteD(1)
		w.WriteD(AdenaCount(p))
		w.WriteD(p.INT)
		w.WriteC(int(p.INT - h.INT))
		w.WriteD(p.STR)
		w.WriteC(int(p.STR - h.STR))
		w.WriteD(p.CON)
		w.WriteC(int(p.CON - h.CON))
		w.WriteD(p.MEN)
		w.WriteC(int(p.MEN - h.MEN))
		w.WriteD(p.DEX)
		w.WriteC(int(p.DEX - h.DEX))
		w.WriteD(p.WIT)
		w.WriteC(int(p.WIT - h.WIT))
	})
}

func (s *Server) onHennaItemList(c *GameClient, r *packet.Reader) {
	_ = r.ReadD()
	c.Send(HennaEquipList(c.Player()))
}

func (s *Server) onHennaItemInfo(c *GameClient, r *packet.Reader) {
	h := GetHenna(r.ReadD())
	if h == nil {
		return
	}
	c.Send(HennaItemInfo(*h, c.Player()))
}

func (s *Server) onHennaUnequipList(c *GameClient, r *packet.Reader) {
	_ = r.ReadD()
	c.Send(HennaUnequipList(c.Player()))
}

func (s *Server) onHennaUnequipInfo(c *GameClient, r *packet.Reader) {
	h := GetHenna(r.ReadD())
	if h == nil {
		return
	}
	c.Send(HennaItemUnequipInfo(*h, c.Player()))
}

func (s *Server) onHennaEquip(c *GameClient, r *packet.Reader) {
	s.equipHenna(c, r.ReadD())
}

func (s *Server) equipHenna(c *GameClient, symbolID int32) {
	p := c.Player()
	h := GetHenna(symbolID)
	if h == nil {
		return
	}
	if !hennaCanBeUsedBy(*h, p.ClassID) {
		c.Send(SystemMessage(SMCantDrawSymbol))
		return
	}
	if hennaSlotsUsed(p) >= int(hennaMaxSlots) {
		c.Send(SystemMessage(SMSymbolsFull))
		return
	}
	if ItemCountOf(p, h.DyeID) < hennaDrawAmount {
		c.Send(SystemMessage(SMCantDrawSymbol))
		return
	}
	if !ReduceAdena(p, h.Price) {
		c.Send(SystemMessage(SMNotEnoughAdena))
		return
	}
	if !RemoveItemByID(p, h.DyeID, hennaDrawAmount) {
		AddAdena(p, h.Price, s.nextItemID)
		c.Send(ActionFailed())
		return
	}
	for i := range p.Hennas {
		if p.Hennas[i] == 0 {
			p.Hennas[i] = symbolID
			break
		}
	}
	c.Send(HennaInfo(p))
	c.Send(SystemMessage(SMSymbolAdded))
	c.Send(ItemList(p.Items, false))
	s.sendWeightAndStats(c)
	_ = s.store.Update(c.ctx(), p)
	c.logChange("henna equipped symbol=%d", symbolID)
}

func (s *Server) onHennaUnequip(c *GameClient, r *packet.Reader) {
	s.unequipHenna(c, r.ReadD())
}

func (s *Server) unequipHenna(c *GameClient, symbolID int32) {
	p := c.Player()
	h := GetHenna(symbolID)
	if h == nil {
		return
	}
	slot := -1
	for i, id := range p.Hennas {
		if id == symbolID {
			slot = i
			break
		}
	}
	if slot < 0 {
		return
	}
	price := h.Price / hennaRemoveAmount
	if !ReduceAdena(p, price) {
		c.Send(SystemMessage(SMNotEnoughAdena))
		return
	}
	p.Hennas[slot] = 0
	AddItem(p, h.DyeID, hennaRemoveAmount, s.nextItemID)
	c.Send(HennaInfo(p))
	c.Send(SystemMessage(SMSymbolDeleted))
	c.Send(ItemList(p.Items, false))
	s.sendWeightAndStats(c)
	_ = s.store.Update(c.ctx(), p)
	c.logChange("henna removed symbol=%d", symbolID)
}
