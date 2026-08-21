package gameserver

import (
	"sync"
	"time"

	"github.com/pblaravel/game-server-l2-go/internal/packet"
)

const tradeInviteTTL = 15 * time.Second

type tradeOffer struct {
	ObjectID int32
	Count    int32
}

type TradeSession struct {
	A, B     int32
	ItemsA   []tradeOffer
	ItemsB   []tradeOffer
	ConfirmA bool
	ConfirmB bool
	Locked   bool
}

type tradeInvite struct {
	from int32
	at   time.Time
}

type tradeState struct {
	mu      sync.Mutex
	invites map[int32]tradeInvite
	byOwner map[int32]*TradeSession
}

func newTradeState() *tradeState {
	return &tradeState{invites: map[int32]tradeInvite{}, byOwner: map[int32]*TradeSession{}}
}

func (s *Server) tradeOf(id int32) *TradeSession {
	s.trades.mu.Lock()
	defer s.trades.mu.Unlock()
	return s.trades.byOwner[id]
}

func tradableInventory(p *Character) []Item {
	out := make([]Item, 0)
	for _, it := range p.Items {
		if it.Equipped || !IsTradable(it.ItemID) {
			continue
		}
		out = append(out, it)
	}
	return out
}

func (s *Server) onTradeRequest(c *GameClient, r *packet.Reader) {
	p := c.Player()
	targetID := r.ReadD()
	if targetID == 0 {
		targetID = c.target
	}
	target := s.world.GetPlayer(targetID)
	if target == nil || target.ObjectID == p.ObjectID {
		c.Send(SystemMessage(SMTargetIncorrect))
		return
	}
	if s.tradeOf(p.ObjectID) != nil {
		c.Send(SystemMessage(SMAlreadyTrading))
		return
	}
	if s.tradeOf(target.ObjectID) != nil {
		c.Send(SystemMessage(SMAlreadyTrading))
		return
	}
	tc := s.clientOf(target.ObjectID)
	if tc == nil {
		c.Send(SystemMessage(SMTargetNotFound))
		return
	}
	s.trades.mu.Lock()
	if inv, ok := s.trades.invites[target.ObjectID]; ok && time.Since(inv.at) < tradeInviteTTL {
		s.trades.mu.Unlock()
		c.Send(SystemMessage(SMWaitingForReply))
		return
	}
	s.trades.invites[target.ObjectID] = tradeInvite{from: p.ObjectID, at: time.Now()}
	s.trades.mu.Unlock()
	c.Send(SystemMessage(SMRequestForTrade, SysText(target.Name)))
	tc.Send(SendTradeRequest(p.ObjectID))
}

func (s *Server) onAnswerTradeRequest(c *GameClient, r *packet.Reader) {
	p := c.Player()
	resp := r.ReadD()
	s.trades.mu.Lock()
	inv, ok := s.trades.invites[p.ObjectID]
	delete(s.trades.invites, p.ObjectID)
	s.trades.mu.Unlock()
	if !ok || time.Since(inv.at) > tradeInviteTTL {
		c.Send(SendTradeDone(false))
		c.Send(SystemMessage(SMTargetNotFound))
		return
	}
	from := s.clientOf(inv.from)
	if from == nil {
		c.Send(SendTradeDone(false))
		c.Send(SystemMessage(SMTargetNotFound))
		return
	}
	if resp != 1 {
		from.Send(SystemMessage(SMDeniedTradeRequest, SysText(p.Name)))
		return
	}
	s.startTrade(from, c)
}

func (s *Server) startTrade(a, b *GameClient) {
	pa, pb := a.Player(), b.Player()
	if pa == nil || pb == nil {
		return
	}
	sess := &TradeSession{A: pa.ObjectID, B: pb.ObjectID}
	s.trades.mu.Lock()
	s.trades.byOwner[pa.ObjectID] = sess
	s.trades.byOwner[pb.ObjectID] = sess
	s.trades.mu.Unlock()
	a.Send(TradeStart(pb.ObjectID, tradableInventory(pa)))
	b.Send(TradeStart(pa.ObjectID, tradableInventory(pb)))
}

func (s *Server) onAddTradeItem(c *GameClient, r *packet.Reader) {
	p := c.Player()
	_ = r.ReadD() // trade id, unused in Java
	objectID := r.ReadD()
	count := r.ReadD()
	sess := s.tradeOf(p.ObjectID)
	if sess == nil || sess.Locked {
		return
	}
	if sess.ConfirmA || sess.ConfirmB {
		c.Send(SystemMessage(SMAlreadyTrading))
		return
	}
	it := FindItem(p, objectID)
	if it == nil || count <= 0 || count > it.Count || it.Equipped || !IsTradable(it.ItemID) {
		c.Send(SystemMessage(SMInvalidTarget))
		return
	}
	offers := s.offersOf(sess, p.ObjectID)
	for i := range *offers {
		if (*offers)[i].ObjectID == objectID {
			next := (*offers)[i].Count + count
			if next > it.Count {
				return
			}
			(*offers)[i].Count = next
			s.sendTradeAdd(c, sess, *it, count)
			return
		}
	}
	*offers = append(*offers, tradeOffer{ObjectID: objectID, Count: count})
	s.sendTradeAdd(c, sess, *it, count)
}

func (s *Server) offersOf(sess *TradeSession, owner int32) *[]tradeOffer {
	if sess.A == owner {
		return &sess.ItemsA
	}
	return &sess.ItemsB
}

func (s *Server) sendTradeAdd(c *GameClient, sess *TradeSession, it Item, count int32) {
	c.Send(TradeOwnAdd(it, count))
	partnerID := sess.B
	if sess.A != c.Player().ObjectID {
		partnerID = sess.A
	}
	if oc := s.clientOf(partnerID); oc != nil {
		oc.Send(TradeOtherAdd(it, count))
	}
}

func (s *Server) onTradeDone(c *GameClient, r *packet.Reader) {
	p := c.Player()
	resp := r.ReadD()
	sess := s.tradeOf(p.ObjectID)
	if sess == nil || sess.Locked {
		return
	}
	if resp != 1 {
		s.cancelTrade(sess, p.ObjectID)
		return
	}
	if sess.A == p.ObjectID {
		sess.ConfirmA = true
	} else {
		sess.ConfirmB = true
	}
	partnerID := sess.B
	if sess.A != p.ObjectID {
		partnerID = sess.A
	}
	if oc := s.clientOf(partnerID); oc != nil {
		oc.Send(TradePressOtherOk())
		c.Send(TradePressOwnOk())
	}
	if sess.ConfirmA && sess.ConfirmB {
		s.finishTrade(sess)
	}
}

func (s *Server) finishTrade(sess *TradeSession) {
	sess.Locked = true
	a := s.world.GetPlayer(sess.A)
	b := s.world.GetPlayer(sess.B)
	ca, cb := s.clientOf(sess.A), s.clientOf(sess.B)
	if a == nil || b == nil || ca == nil || cb == nil {
		s.cancelTrade(sess, 0)
		return
	}
	RecalcStats(a)
	RecalcStats(b)
	if Distance3D(a.X, a.Y, a.Z, b.X, b.Y, b.Z) > interactRange {
		s.cancelTrade(sess, 0)
		return
	}
	if len(sess.ItemsA) == 0 && len(sess.ItemsB) == 0 {
		s.failTrade(sess)
		return
	}
	if !s.validateTradeOffers(a, sess.ItemsA) || !s.validateTradeOffers(b, sess.ItemsB) {
		s.cancelTrade(sess, 0)
		return
	}
	weightA, slotsA := tradeIncomingLoad(a, b, sess.ItemsB)
	weightB, slotsB := tradeIncomingLoad(b, a, sess.ItemsA)
	if a.WeightLimit > 0 && a.CurrentWeight+weightA > a.WeightLimit ||
		b.WeightLimit > 0 && b.CurrentWeight+weightB > b.WeightLimit {
		ca.Send(SystemMessage(SMWeightLimitExceeded))
		cb.Send(SystemMessage(SMWeightLimitExceeded))
		s.failTrade(sess)
		return
	}
	if a.InventoryLimit > 0 && int32(len(a.Items))+slotsA > a.InventoryLimit ||
		b.InventoryLimit > 0 && int32(len(b.Items))+slotsB > b.InventoryLimit {
		ca.Send(SystemMessage(SMSlotsFull))
		cb.Send(SystemMessage(SMSlotsFull))
		s.failTrade(sess)
		return
	}
	for _, off := range sess.ItemsA {
		moveItem(&a.Items, &b.Items, off.ObjectID, off.Count, "INVENTORY", s.nextItemID)
	}
	for _, off := range sess.ItemsB {
		moveItem(&b.Items, &a.Items, off.ObjectID, off.Count, "INVENTORY", s.nextItemID)
	}
	s.clearTrade(sess)
	ca.Send(SendTradeDone(true))
	cb.Send(SendTradeDone(true))
	ca.Send(SystemMessage(SMTradeSuccessful))
	cb.Send(SystemMessage(SMTradeSuccessful))
	ca.Send(ItemList(a.Items, false))
	cb.Send(ItemList(b.Items, false))
	s.sendWeightAndStats(ca)
	s.sendWeightAndStats(cb)
	_ = s.store.Update(ca.ctx(), a)
	_ = s.store.Update(cb.ctx(), b)
}

func (s *Server) validateTradeOffers(p *Character, offers []tradeOffer) bool {
	for _, off := range offers {
		it := FindItem(p, off.ObjectID)
		if it == nil || off.Count <= 0 || off.Count > it.Count || it.Equipped {
			return false
		}
	}
	return true
}

func tradeIncomingLoad(dest, src *Character, offers []tradeOffer) (weight, slots int32) {
	for _, off := range offers {
		it := FindItem(src, off.ObjectID)
		if it == nil {
			continue
		}
		weight += ItemWeight(it.ItemID) * off.Count
		if isStackable(it.ItemID) {
			if FindItemByID(dest, it.ItemID) == nil {
				slots++
			}
		} else {
			slots += off.Count
		}
	}
	return weight, slots
}

func (s *Server) cancelTrade(sess *TradeSession, by int32) {
	a, b := s.clientOf(sess.A), s.clientOf(sess.B)
	s.clearTrade(sess)
	if a != nil {
		a.Send(SendTradeDone(false))
		if by != 0 && by != sess.A {
			a.Send(SystemMessage(SMCanceledTrade, SysText(playerName(s, by))))
		}
	}
	if b != nil {
		b.Send(SendTradeDone(false))
		if by != 0 && by != sess.B {
			b.Send(SystemMessage(SMCanceledTrade, SysText(playerName(s, by))))
		}
	}
}

func (s *Server) failTrade(sess *TradeSession) {
	a, b := s.clientOf(sess.A), s.clientOf(sess.B)
	s.clearTrade(sess)
	if a != nil {
		a.Send(SendTradeDone(false))
		a.Send(SystemMessage(SMExchangeEnded))
	}
	if b != nil {
		b.Send(SendTradeDone(false))
		b.Send(SystemMessage(SMExchangeEnded))
	}
}

func (s *Server) clearTrade(sess *TradeSession) {
	s.trades.mu.Lock()
	delete(s.trades.byOwner, sess.A)
	delete(s.trades.byOwner, sess.B)
	s.trades.mu.Unlock()
}

func (s *Server) cancelTradeFor(id int32) {
	if sess := s.tradeOf(id); sess != nil {
		s.cancelTrade(sess, id)
	}
}

func playerName(s *Server, id int32) string {
	if p := s.world.GetPlayer(id); p != nil {
		return p.Name
	}
	return ""
}
