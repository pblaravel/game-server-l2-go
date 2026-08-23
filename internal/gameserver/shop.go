package gameserver

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pblaravel/game-server-l2-go/internal/packet"
)

const interactRange = 150.0

func (s *Server) nextItemID() int32 { return s.world.NextID() }

func sellableItems(p *Character) []Item {
	out := make([]Item, 0)
	for _, it := range p.Items {
		if it.Equipped || !IsSellable(it.ItemID) {
			continue
		}
		out = append(out, it)
	}
	return out
}

func (s *Server) canInteract(p *Character, npc *NPC) bool {
	if p == nil || npc == nil || npc.AlikeDead() {
		return false
	}
	return Distance3D(p.X, p.Y, p.Z, npc.X, npc.Y, npc.Z) <= interactRange
}

func (s *Server) openNpcWindow(c *GameClient, npc *NPC) {
	p := c.Player()
	if !s.canInteract(p, npc) {
		c.Send(ActionFailed())
		return
	}
	lists := BuyListsForNPC(npc.NPCID)
	teles := TeleportsForNPC(npc.NPCID)
	instants := InstantTeleports(npc.NPCID)
	ms := MultisellsForNPC(npc.NPCID)
	wh := isWarehouseNPC(npc)
	guide := isNewbieGuide(npc)
	symbol := isSymbolMaker(npc)
	if len(lists) == 0 && len(teles) == 0 && len(instants) == 0 && len(ms) == 0 && !wh && !guide && !symbol {
		c.Send(ActionFailed())
		return
	}
	var b strings.Builder
	b.WriteString("<html><body>")
	if npc.Title != "" {
		fmt.Fprintf(&b, "%s<br>%s<br><br>", npc.Name, npc.Title)
	} else {
		fmt.Fprintf(&b, "%s<br><br>", npc.Name)
	}
	if wh {
		fmt.Fprintf(&b, "<a action=\"bypass -h npc_%d_DepositP\">Deposit</a><br>", npc.ObjectID)
		fmt.Fprintf(&b, "<a action=\"bypass -h npc_%d_WithdrawP\">Withdraw</a><br>", npc.ObjectID)
	}
	if guide {
		fmt.Fprintf(&b, "<a action=\"bypass -h npc_%d_SupportMagic\">Newbie support magic</a><br>", npc.ObjectID)
	}
	if symbol {
		fmt.Fprintf(&b, "<a action=\"bypass -h npc_%d_Draw\">Draw a symbol</a><br>", npc.ObjectID)
	}
	for i, list := range ms {
		label := "Exchange"
		if i > 0 {
			label = fmt.Sprintf("Exchange %d", i+1)
		}
		fmt.Fprintf(&b, "<a action=\"bypass -h npc_%d_Multisell %d\">%s</a><br>", npc.ObjectID, list.ID, label)
	}
	for i, list := range lists {
		label := "Buy"
		if i > 0 {
			label = fmt.Sprintf("Buy %d", i+1)
		}
		fmt.Fprintf(&b, "<a action=\"bypass -h npc_%d_Buy %d\">%s</a><br>", npc.ObjectID, list.ID, label)
	}
	if len(lists) > 0 {
		fmt.Fprintf(&b, "<a action=\"bypass -h npc_%d_Sell\">Sell</a><br>", npc.ObjectID)
	}
	for i, loc := range teles {
		if loc.Type != "" && !strings.EqualFold(loc.Type, "STANDARD") {
			continue
		}
		fmt.Fprintf(&b, "<a action=\"bypass -h npc_%d_goto %d\">%s</a><br1>", npc.ObjectID, i, loc.Desc)
	}
	for i := range instants {
		fmt.Fprintf(&b, "<a action=\"bypass -h npc_%d_instant %d\">Instant teleport %d</a><br1>", npc.ObjectID, i, i+1)
	}
	b.WriteString("</body></html>")
	c.Send(NpcHtmlMessage(npc.ObjectID, b.String()))
}

func (s *Server) showBuyWindow(c *GameClient, npc *NPC, listID int32, openTab bool) {
	list := GetBuyList(listID)
	if list == nil || (list.NpcID != 0 && list.NpcID != npc.NPCID) {
		c.Send(ActionFailed())
		return
	}
	c.Send(SellList(AdenaCount(c.Player()), sellableItems(c.Player()), !openTab))
	c.Send(BuyList(*list, AdenaCount(c.Player()), openTab))
}

func (s *Server) showSellWindow(c *GameClient, npc *NPC) {
	_ = npc
	c.Send(SellList(AdenaCount(c.Player()), sellableItems(c.Player()), true))
}

func (s *Server) onBypassToServer(c *GameClient, r *packet.Reader) {
	cmd := strings.TrimSpace(r.ReadS())
	if cmd == "" {
		return
	}
	if strings.HasPrefix(cmd, "npc_") {
		s.onNpcBypass(c, cmd)
		return
	}
	c.Send(ActionFailed())
}

func (s *Server) onNpcBypass(c *GameClient, cmd string) {
	p := c.Player()
	rest := strings.TrimPrefix(cmd, "npc_")
	us := strings.IndexByte(rest, '_')
	if us <= 0 {
		c.Send(ActionFailed())
		return
	}
	oid, err := strconv.Atoi(rest[:us])
	if err != nil {
		c.Send(ActionFailed())
		return
	}
	npc := s.world.GetNPC(int32(oid))
	if npc == nil || !s.canInteract(p, npc) {
		c.Send(ActionFailed())
		return
	}
	action := strings.TrimSpace(rest[us+1:])
	fields := strings.Fields(action)
	if len(fields) == 0 {
		c.Send(ActionFailed())
		return
	}
	switch strings.ToLower(fields[0]) {
	case "buy":
		if len(fields) < 2 {
			if lists := BuyListsForNPC(npc.NPCID); len(lists) > 0 {
				s.showBuyWindow(c, npc, lists[0].ID, true)
			}
			return
		}
		id, _ := strconv.Atoi(fields[1])
		s.showBuyWindow(c, npc, int32(id), true)
	case "sell":
		s.showSellWindow(c, npc)
	case "goto":
		if len(fields) < 2 {
			c.Send(ActionFailed())
			return
		}
		idx, _ := strconv.Atoi(fields[1])
		s.doNpcTeleport(c, npc, idx)
	case "depositp":
		s.showWarehouseDeposit(c, npc)
	case "withdrawp":
		s.showWarehouseWithdraw(c, npc)
	case "supportmagic":
		s.giveNewbieBuffs(c)
	case "instant", "teleport":
		if len(fields) < 2 {
			c.Send(ActionFailed())
			return
		}
		idx, _ := strconv.Atoi(fields[1])
		s.doInstantTeleport(c, npc, idx)
	case "multisell":
		if len(fields) < 2 {
			if lists := MultisellsForNPC(npc.NPCID); len(lists) > 0 {
				s.showMultisell(c, lists[0])
			}
			return
		}
		id, _ := strconv.Atoi(fields[1])
		s.showMultisell(c, GetMultisell(int32(id)))
	case "draw":
		c.Send(HennaEquipList(p))
	default:
		c.Send(ActionFailed())
	}
}

func isSymbolMaker(npc *NPC) bool {
	if npc == nil {
		return false
	}
	name := strings.ToLower(npc.Name)
	title := strings.ToLower(npc.Title)
	if strings.Contains(name, "symbol maker") || strings.Contains(title, "symbol") {
		return true
	}
	return npc.NPCID == 30098 || npc.NPCID == 31046 || npc.NPCID == 31047 || npc.NPCID == 31048 || npc.NPCID == 31049
}

func isNewbieGuide(npc *NPC) bool {
	if npc == nil {
		return false
	}
	if strings.Contains(strings.ToLower(npc.Name), "newbie guide") || strings.Contains(strings.ToLower(npc.Title), "newbie") {
		return true
	}
	return npc.NPCID == 30009 || npc.NPCID == 30598 || npc.NPCID == 30599 || npc.NPCID == 30600 || npc.NPCID == 30601 || npc.NPCID == 30602
}

func (s *Server) giveNewbieBuffs(c *GameClient) {
	p := c.Player()
	buffs := ValidNewbieBuffs(isMageClass(p.ClassID), p.Level)
	if len(buffs) == 0 {
		c.Send(ActionFailed())
		return
	}
	for _, b := range buffs {
		if tpl := GetSkill(b.SkillID, b.SkillLevel); tpl != nil {
			AddEffects(p, tpl)
		}
	}
	RecalcStats(p)
	c.Send(UserInfo(p))
	c.Send(ActionFailed())
}

func (s *Server) doInstantTeleport(c *GameClient, npc *NPC, index int) {
	locs := InstantTeleports(npc.NPCID)
	if index < 0 || index >= len(locs) {
		c.Send(ActionFailed())
		return
	}
	loc := locs[index]
	s.teleportPlayer(c, loc.X, loc.Y, loc.Z)
}

func (s *Server) doNpcTeleport(c *GameClient, npc *NPC, index int) {
	p := c.Player()
	locs := TeleportsForNPC(npc.NPCID)
	if index < 0 || index >= len(locs) {
		c.Send(ActionFailed())
		return
	}
	loc := locs[index]
	if loc.PriceCount > 0 && loc.PriceID != 0 {
		if it := FindItemByID(p, loc.PriceID); it == nil || it.Count < loc.PriceCount {
			c.Send(SystemMessage(SMNotEnoughAdena))
			c.Send(ActionFailed())
			return
		}
		RemoveItemCount(p, FindItemByID(p, loc.PriceID).ObjectID, loc.PriceCount)
	}
	s.teleportPlayer(c, loc.X, loc.Y, loc.Z)
}

func (s *Server) teleportPlayer(c *GameClient, x, y, z int32) {
	p := c.Player()
	if Geo().HasGeo(int(x), int(y)) {
		z = Geo().Height(int(x), int(y), int(z))
	}
	s.Broadcast(DeleteObject(p.ObjectID), c)
	p.X, p.Y, p.Z = x, y, z
	pkt := TeleportToLocation(p.ObjectID, x, y, z)
	c.Send(pkt)
	c.Broadcast(pkt)
	c.Send(UserInfo(p))
	c.logChange("teleported to (%d,%d,%d)", x, y, z)
}

func (s *Server) onSellItem(c *GameClient, r *packet.Reader) {
	p := c.Player()
	listID := r.ReadD()
	count := r.ReadD()
	npc := s.world.GetNPC(c.target)
	if npc == nil || !s.canInteract(p, npc) {
		c.Send(ActionFailed())
		return
	}
	if listID > 1000000 && npc.NPCID != listID-1000000 {
		c.Send(ActionFailed())
		return
	}
	if count <= 0 || count > 100 {
		c.Send(ActionFailed())
		return
	}
	var total int64
	type sellReq struct{ oid, itemID, cnt int32 }
	reqs := make([]sellReq, 0, count)
	for i := int32(0); i < count; i++ {
		oid := r.ReadD()
		itemID := r.ReadD()
		cnt := r.ReadD()
		reqs = append(reqs, sellReq{oid, itemID, cnt})
	}
	for _, req := range reqs {
		it := FindItem(p, req.oid)
		if it == nil || it.ItemID != req.itemID || req.cnt <= 0 || req.cnt > it.Count || it.Equipped || !IsSellable(it.ItemID) {
			continue
		}
		price := int64(ReferencePrice(it.ItemID) / 2)
		total += price * int64(req.cnt)
		RemoveItemCount(p, req.oid, req.cnt)
	}
	if total > 0 {
		AddAdena(p, int32(total), s.nextItemID)
	}
	c.Send(ItemList(p.Items, true))
	s.sendWeightAndStats(c)
	_ = s.store.Update(c.ctx(), p)
	c.logChange("sold items for %d adena to npc=%d", total, npc.NPCID)
}

func (s *Server) onBuyItem(c *GameClient, r *packet.Reader) {
	p := c.Player()
	listID := r.ReadD()
	count := r.ReadD()
	list := GetBuyList(listID)
	if list == nil || count <= 0 || count > 100 {
		c.Send(ActionFailed())
		return
	}
	npc := s.world.GetNPC(c.target)
	if list.NpcID > 0 && (npc == nil || npc.NPCID != list.NpcID || !s.canInteract(p, npc)) {
		c.Send(ActionFailed())
		return
	}
	type buyReq struct{ itemID, cnt int32 }
	reqs := make([]buyReq, 0, count)
	for i := int32(0); i < count; i++ {
		itemID := r.ReadD()
		cnt := r.ReadD()
		reqs = append(reqs, buyReq{itemID, cnt})
	}
	var cost, weight, slots int64
	for _, req := range reqs {
		prod := list.Product(req.itemID)
		if prod == nil || req.cnt < 1 {
			c.Send(ActionFailed())
			return
		}
		tpl := GetItem(req.itemID)
		if tpl != nil && !tpl.Stackable && req.cnt > 1 {
			c.Send(SystemMessage(SMExceededInputQty))
			return
		}
		if prod.Price < 0 || (prod.Price == 0 && p.AccessLevel <= 0) {
			c.Send(ActionFailed())
			return
		}
		cost += int64(prod.Price) * int64(req.cnt)
		if tpl != nil {
			weight += int64(tpl.Weight) * int64(req.cnt)
			if tpl.Stackable {
				if FindItemByID(p, req.itemID) == nil {
					slots++
				}
			} else {
				slots += int64(req.cnt)
			}
		}
	}
	if p.CurrentWeight+int32(weight) > p.WeightLimit {
		c.Send(SystemMessage(SMWeightLimitExceeded))
		return
	}
	if int32(len(p.Items))+int32(slots) > p.InventoryLimit {
		c.Send(SystemMessage(SMSlotsFull))
		return
	}
	if !ReduceAdena(p, int32(cost)) {
		c.Send(SystemMessage(SMNotEnoughAdena))
		return
	}
	for _, req := range reqs {
		AddItem(p, req.itemID, req.cnt, s.nextItemID)
	}
	c.Send(ItemList(p.Items, true))
	s.sendWeightAndStats(c)
	_ = s.store.Update(c.ctx(), p)
	c.logChange("bought %d lines from list=%d cost=%d", len(reqs), listID, cost)
}
