package gameserver

import (
	"github.com/pblaravel/game-server-l2-go/internal/packet"
)

func (p *Character) inStoreMode() bool {
	return p.PrivateStore == StoreSell || p.PrivateStore == StoreBuy || p.PrivateStore == StorePackageSell
}

func (s *Server) canOpenPrivateStore(c *GameClient) bool {
	p := c.Player()
	if p.inStoreMode() {
		return !p.AlikeDead()
	}
	if p.InCombat || p.PvPFlag > 0 {
		p.PrivateStore = StoreNone
		c.Send(SystemMessage(SMCantOperateStoreDuringCombat))
		return false
	}
	if p.AlikeDead() || s.tradeOf(p.ObjectID) != nil {
		p.PrivateStore = StoreNone
		return false
	}
	return true
}

func (s *Server) openSellStoreManage(c *GameClient, pack bool) {
	p := c.Player()
	if !s.canOpenPrivateStore(c) {
		c.Send(ActionFailed())
		return
	}
	if p.Sitting {
		s.setSitting(c, false)
	}
	p.StorePack = pack
	p.PrivateStore = StoreSellManage
	c.Send(PrivateStoreManageListSell(p, pack))
}

func (s *Server) openBuyStoreManage(c *GameClient) {
	p := c.Player()
	if !s.canOpenPrivateStore(c) {
		c.Send(ActionFailed())
		return
	}
	if p.Sitting {
		s.setSitting(c, false)
	}
	p.PrivateStore = StoreBuyManage
	c.Send(PrivateStoreManageListBuy(p))
}

func (s *Server) quitPrivateStore(c *GameClient) {
	p := c.Player()
	p.PrivateStore = StoreNone
	p.StoreItems = nil
	p.StoreMsg = ""
	p.StorePack = false
	if p.Sitting {
		s.setSitting(c, false)
	}
	c.Send(UserInfo(p))
	c.Broadcast(CharInfo(p))
}

func (s *Server) onPrivateStoreManageSell(c *GameClient, r *packet.Reader) {
	_ = r
	s.openSellStoreManage(c, false)
}

func (s *Server) onSetPrivateStoreListSell(c *GameClient, r *packet.Reader) {
	p := c.Player()
	pack := r.ReadD() == 1
	count := r.ReadD()
	if count < 1 || count > 80 {
		s.quitPrivateStore(c)
		return
	}
	if !s.canOpenPrivateStore(c) {
		return
	}
	offers := make([]StoreOffer, 0, count)
	for i := int32(0); i < count; i++ {
		oid := r.ReadD()
		cnt := r.ReadD()
		price := r.ReadD()
		it := FindItem(p, oid)
		if it == nil || it.Equipped || !IsTradable(it.ItemID) || cnt < 1 || cnt > it.Count || price < 0 {
			s.quitPrivateStore(c)
			return
		}
		offers = append(offers, StoreOffer{ObjectID: oid, ItemID: it.ItemID, Count: cnt, Price: price, Enchant: it.Enchant})
	}
	p.StoreItems = offers
	p.StorePack = pack
	if pack {
		p.PrivateStore = StorePackageSell
	} else {
		p.PrivateStore = StoreSell
	}
	s.setSitting(c, true)
	c.Send(UserInfo(p))
	c.Broadcast(CharInfo(p))
	c.Broadcast(PrivateStoreMsgSell(p))
	c.logChange("opened sell store lines=%d pack=%v", len(offers), pack)
}

func (s *Server) onSetPrivateStoreListBuy(c *GameClient, r *packet.Reader) {
	p := c.Player()
	count := r.ReadD()
	if count < 1 || count > 80 {
		s.quitPrivateStore(c)
		return
	}
	if !s.canOpenPrivateStore(c) {
		return
	}
	offers := make([]StoreOffer, 0, count)
	var cost int64
	for i := int32(0); i < count; i++ {
		// Java SetPrivateStoreListBuy: itemId D, enchant H, unused H, count D, price D.
		itemID := r.ReadD()
		enchant := int16(r.ReadH())
		_ = r.ReadH()
		cnt := r.ReadD()
		price := r.ReadD()
		if itemID < 1 || cnt < 1 || price < 0 || !IsTradable(itemID) {
			s.quitPrivateStore(c)
			return
		}
		cost += int64(price) * int64(cnt)
		offers = append(offers, StoreOffer{ItemID: itemID, Count: cnt, Price: price, Enchant: enchant})
	}
	if int64(AdenaCount(p)) < cost {
		c.Send(SystemMessage(SMNotEnoughAdena))
		s.quitPrivateStore(c)
		return
	}
	p.StoreItems = offers
	p.PrivateStore = StoreBuy
	s.setSitting(c, true)
	c.Send(UserInfo(p))
	c.Broadcast(CharInfo(p))
	c.Broadcast(PrivateStoreMsgBuy(p))
	c.logChange("opened buy store lines=%d", len(offers))
}

func (s *Server) onSetPrivateStoreMsgSell(c *GameClient, r *packet.Reader) {
	c.Player().StoreMsg = r.ReadS()
	c.Broadcast(PrivateStoreMsgSell(c.Player()))
}

func (s *Server) onSetPrivateStoreMsgBuy(c *GameClient, r *packet.Reader) {
	c.Player().StoreMsg = r.ReadS()
	c.Broadcast(PrivateStoreMsgBuy(c.Player()))
}

func (s *Server) showPlayerStore(c *GameClient, other *Character) {
	switch other.PrivateStore {
	case StoreSell, StorePackageSell:
		c.Send(PrivateStoreListSell(c.Player(), other))
	case StoreBuy:
		c.Send(PrivateStoreListBuy(c.Player(), other))
	default:
		c.Send(ActionFailed())
	}
}

func (s *Server) onPrivateStoreBuy(c *GameClient, r *packet.Reader) {
	p := c.Player()
	storeID := r.ReadD()
	count := r.ReadD()
	if count < 1 || count > 80 {
		return
	}
	store := s.world.GetPlayer(storeID)
	if store == nil || !store.inStoreMode() || (store.PrivateStore != StoreSell && store.PrivateStore != StorePackageSell) {
		return
	}
	if Distance3D(p.X, p.Y, p.Z, store.X, store.Y, store.Z) > itemInteractRange {
		c.Send(SystemMessage(SMTargetTooFar))
		return
	}
	type req struct{ oid, cnt, price int32 }
	reqs := make([]req, 0, count)
	for i := int32(0); i < count; i++ {
		reqs = append(reqs, req{r.ReadD(), r.ReadD(), r.ReadD()})
	}
	if store.PrivateStore == StorePackageSell && len(reqs) < len(store.StoreItems) {
		return
	}
	sc := s.clientOf(store.ObjectID)
	for _, req := range reqs {
		off := findStoreOffer(store, req.oid)
		if off == nil || req.cnt < 1 || req.cnt > off.Count || req.price != off.Price {
			return
		}
		cost := int64(off.Price) * int64(req.cnt)
		if int64(AdenaCount(p)) < cost {
			c.Send(SystemMessage(SMNotEnoughAdena))
			return
		}
		it := FindItem(store, off.ObjectID)
		if it == nil || it.Count < req.cnt || it.Equipped {
			return
		}
		if !ReduceAdena(p, int32(cost)) {
			c.Send(SystemMessage(SMNotEnoughAdena))
			return
		}
		AddAdena(store, int32(cost), s.nextItemID)
		RemoveItemCount(store, it.ObjectID, req.cnt)
		added := AddItem(p, off.ItemID, req.cnt, s.nextItemID)
		if added != nil && off.Enchant > 0 {
			added.Enchant = off.Enchant
		}
		off.Count -= req.cnt
	}
	store.StoreItems = compactStore(store.StoreItems)
	if len(store.StoreItems) == 0 {
		if sc != nil {
			s.quitPrivateStore(sc)
		} else {
			store.PrivateStore = StoreNone
			store.StoreItems = nil
		}
	}
	c.Send(ItemList(p.Items, true))
	s.sendWeightAndStats(c)
	if sc != nil {
		sc.Send(ItemList(store.Items, true))
		s.sendWeightAndStats(sc)
	}
	_ = s.store.Update(c.ctx(), p)
	_ = s.store.Update(c.ctx(), store)
}

func (s *Server) onPrivateStoreSell(c *GameClient, r *packet.Reader) {
	p := c.Player()
	storeID := r.ReadD()
	count := r.ReadD()
	if count < 1 || count > 80 {
		return
	}
	store := s.world.GetPlayer(storeID)
	if store == nil || store.PrivateStore != StoreBuy {
		return
	}
	if Distance3D(p.X, p.Y, p.Z, store.X, store.Y, store.Z) > itemInteractRange {
		c.Send(SystemMessage(SMTargetTooFar))
		return
	}
	sc := s.clientOf(store.ObjectID)
	for i := int32(0); i < count; i++ {
		// Java RequestPrivateStoreSell: objectId D, itemId D, enchant H, unused H, count D, price D.
		oid := r.ReadD()
		itemID := r.ReadD()
		ench := int16(r.ReadH())
		_ = r.ReadH()
		cnt := r.ReadD()
		price := r.ReadD()
		it := FindItem(p, oid)
		off := findStoreOfferByItem(store, itemID)
		if it == nil || off == nil || it.ItemID != itemID || it.Equipped || cnt < 1 || cnt > it.Count || cnt > off.Count || price != off.Price {
			return
		}
		if (off.Enchant != 0 && it.Enchant != off.Enchant) || (ench != 0 && it.Enchant != ench) {
			return
		}
		cost := int64(off.Price) * int64(cnt)
		if int64(AdenaCount(store)) < cost {
			c.Send(SystemMessage(SMNotEnoughAdena))
			return
		}
		if !ReduceAdena(store, int32(cost)) {
			return
		}
		AddAdena(p, int32(cost), s.nextItemID)
		RemoveItemCount(p, oid, cnt)
		AddItem(store, itemID, cnt, s.nextItemID)
		off.Count -= cnt
	}
	store.StoreItems = compactStore(store.StoreItems)
	if len(store.StoreItems) == 0 {
		if sc != nil {
			s.quitPrivateStore(sc)
		} else {
			store.PrivateStore = StoreNone
		}
	}
	c.Send(ItemList(p.Items, true))
	s.sendWeightAndStats(c)
	if sc != nil {
		sc.Send(ItemList(store.Items, true))
		s.sendWeightAndStats(sc)
	}
}

func findStoreOffer(p *Character, objectID int32) *StoreOffer {
	for i := range p.StoreItems {
		if p.StoreItems[i].ObjectID == objectID {
			return &p.StoreItems[i]
		}
	}
	return nil
}

func findStoreOfferByItem(p *Character, itemID int32) *StoreOffer {
	for i := range p.StoreItems {
		if p.StoreItems[i].ItemID == itemID {
			return &p.StoreItems[i]
		}
	}
	return nil
}

func compactStore(in []StoreOffer) []StoreOffer {
	out := in[:0]
	for _, o := range in {
		if o.Count > 0 {
			out = append(out, o)
		}
	}
	return append([]StoreOffer(nil), out...)
}

func sellableStoreItems(p *Character) []Item {
	return tradableInventory(p)
}

func PrivateStoreManageListSell(p *Character, pack bool) []byte {
	inv := sellableStoreItems(p)
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x9a)
		w.WriteD(p.ObjectID)
		if pack || p.StorePack {
			w.WriteD(1)
		} else {
			w.WriteD(0)
		}
		w.WriteD(AdenaCount(p))
		w.WriteD(int32(len(inv)))
		for _, it := range inv {
			w.WriteD(int32(it.Type2))
			w.WriteD(it.ObjectID)
			w.WriteD(it.ItemID)
			w.WriteD(it.Count)
			w.WriteH(0)
			w.WriteH(int(it.Enchant))
			w.WriteH(0)
			w.WriteD(it.BodyPart)
			w.WriteD(0)
		}
		w.WriteD(int32(len(p.StoreItems)))
		for _, it := range p.StoreItems {
			w.WriteD(int32(itemType2(it.ItemID)))
			w.WriteD(it.ObjectID)
			w.WriteD(it.ItemID)
			w.WriteD(it.Count)
			w.WriteH(0)
			w.WriteH(int(it.Enchant))
			w.WriteH(0)
			w.WriteD(BodyPartForItem(it.ItemID))
			w.WriteD(it.Price)
			w.WriteD(ReferencePrice(it.ItemID))
		}
	})
}

func PrivateStoreListSell(buyer, store *Character) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x9b)
		w.WriteD(store.ObjectID)
		if store.StorePack {
			w.WriteD(1)
		} else {
			w.WriteD(0)
		}
		w.WriteD(AdenaCount(buyer))
		w.WriteD(int32(len(store.StoreItems)))
		for _, it := range store.StoreItems {
			w.WriteD(int32(itemType2(it.ItemID)))
			w.WriteD(it.ObjectID)
			w.WriteD(it.ItemID)
			w.WriteD(it.Count)
			w.WriteH(0)
			w.WriteH(int(it.Enchant))
			w.WriteH(0)
			w.WriteD(BodyPartForItem(it.ItemID))
			w.WriteD(it.Price)
			w.WriteD(ReferencePrice(it.ItemID))
		}
	})
}

func PrivateStoreManageListBuy(p *Character) []byte {
	inv := sellableStoreItems(p)
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0xb7)
		w.WriteD(p.ObjectID)
		w.WriteD(AdenaCount(p))
		w.WriteD(int32(len(inv)))
		for _, it := range inv {
			w.WriteD(it.ItemID)
			w.WriteH(int(it.Enchant))
			w.WriteD(it.Count)
			w.WriteD(ReferencePrice(it.ItemID))
			w.WriteH(0)
			w.WriteD(it.BodyPart)
			w.WriteH(int(it.Type2))
		}
		w.WriteD(int32(len(p.StoreItems)))
		for _, it := range p.StoreItems {
			w.WriteD(it.ItemID)
			w.WriteH(int(it.Enchant))
			w.WriteD(it.Count)
			w.WriteD(ReferencePrice(it.ItemID))
			w.WriteH(0)
			w.WriteD(BodyPartForItem(it.ItemID))
			w.WriteH(int(itemType2(it.ItemID)))
			w.WriteD(it.Price)
			w.WriteD(ReferencePrice(it.ItemID))
		}
	})
}

func PrivateStoreListBuy(seller, store *Character) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0xb8)
		w.WriteD(store.ObjectID)
		w.WriteD(AdenaCount(seller))
		w.WriteD(int32(len(store.StoreItems)))
		for _, it := range store.StoreItems {
			w.WriteD(it.ObjectID)
			w.WriteD(it.ItemID)
			w.WriteH(int(it.Enchant))
			w.WriteD(ItemCountOf(seller, it.ItemID))
			w.WriteD(ReferencePrice(it.ItemID))
			w.WriteH(0)
			w.WriteD(BodyPartForItem(it.ItemID))
			w.WriteH(int(itemType2(it.ItemID)))
			w.WriteD(it.Price)
			w.WriteD(it.Count)
		}
	})
}

func PrivateStoreMsgSell(p *Character) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x9c)
		w.WriteD(p.ObjectID)
		w.WriteS(p.StoreMsg)
	})
}

func PrivateStoreMsgBuy(p *Character) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0xb9)
		w.WriteD(p.ObjectID)
		w.WriteS(p.StoreMsg)
	})
}

func itemType2(itemID int32) int16 {
	if tpl := GetItem(itemID); tpl != nil {
		return tpl.Type2
	}
	return Type2Other
}
