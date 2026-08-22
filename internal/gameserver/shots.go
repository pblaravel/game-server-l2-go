package gameserver

import (
	"strings"

	"github.com/pblaravel/game-server-l2-go/internal/packet"
)

const (
	shotKindNone = iota
	shotKindSS
	shotKindSPS
	shotKindBSS
)

func crystalGrade(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return "none"
	}
	return s
}

func shotKind(tpl *ItemTemplate) int {
	if tpl == nil {
		return shotKindNone
	}
	if strings.EqualFold(tpl.Handler, "BlessedSpiritShots") || (tpl.ID >= 3947 && tpl.ID <= 3952) {
		return shotKindBSS
	}
	if strings.EqualFold(tpl.Action, "soulshot") || strings.EqualFold(tpl.Handler, "SoulShots") {
		return shotKindSS
	}
	if strings.EqualFold(tpl.Action, "spiritshot") || strings.EqualFold(tpl.Handler, "SpiritShots") {
		return shotKindSPS
	}
	return shotKindNone
}

func equippedWeapon(p *Character) *Item {
	oid := p.PaperdollObj[PaperRHand]
	if oid == 0 {
		return nil
	}
	return FindItem(p, oid)
}

func weaponShotCounts(p *Character) (ss, sps int32) {
	w := equippedWeapon(p)
	if w == nil {
		return 0, 0
	}
	if tpl := GetItem(w.ItemID); tpl != nil {
		return tpl.Soulshots, tpl.Spiritshots
	}
	return 1, 1
}

func weaponCrystal(p *Character) string {
	w := equippedWeapon(p)
	if w == nil {
		return ""
	}
	if tpl := GetItem(w.ItemID); tpl != nil {
		return crystalGrade(tpl.CrystalType)
	}
	return "none"
}

func hasAutoShot(p *Character, itemID int32) bool {
	for _, id := range p.AutoShots {
		if id == itemID {
			return true
		}
	}
	return false
}

func addAutoShot(p *Character, itemID int32) {
	if hasAutoShot(p, itemID) {
		return
	}
	p.AutoShots = append(p.AutoShots, itemID)
}

func removeAutoShot(p *Character, itemID int32) bool {
	kept := p.AutoShots[:0]
	found := false
	for _, id := range p.AutoShots {
		if id == itemID {
			found = true
			continue
		}
		kept = append(kept, id)
	}
	p.AutoShots = append([]int32(nil), kept...)
	return found
}

func ExAutoSoulShot(itemID, typ int32) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0xFE)
		w.WriteH(0x12)
		w.WriteD(itemID)
		w.WriteD(typ)
	})
}

func (s *Server) chargeShot(c *GameClient, item *Item, fromAuto bool) bool {
	p := c.Player()
	tpl := GetItem(item.ItemID)
	kind := shotKind(tpl)
	if kind == shotKindNone {
		return false
	}
	w := equippedWeapon(p)
	ssNeed, spsNeed := weaponShotCounts(p)
	need := ssNeed
	if kind != shotKindSS {
		need = spsNeed
	}
	if w == nil || need <= 0 {
		if !fromAuto {
			if kind == shotKindSS {
				c.Send(SystemMessage(SMCannotUseSoulshots))
			} else {
				c.Send(SystemMessage(SMCannotUseSpiritshots))
			}
		}
		return false
	}
	if crystalGrade(tpl.CrystalType) != weaponCrystal(p) {
		if !fromAuto {
			if kind == shotKindSS {
				c.Send(SystemMessage(SMSoulshotsGradeMismatch))
			} else {
				c.Send(SystemMessage(SMSpiritshotsGradeMismatch))
			}
		}
		return false
	}
	switch kind {
	case shotKindSS:
		if p.ChargedSS {
			return true
		}
	case shotKindSPS:
		if p.ChargedSPS || p.ChargedBSS {
			return true
		}
	case shotKindBSS:
		if p.ChargedBSS {
			return true
		}
	}
	if item.Count < need {
		if fromAuto {
			removeAutoShot(p, item.ItemID)
			c.Send(ExAutoSoulShot(item.ItemID, 0))
		}
		if kind == shotKindSS {
			c.Send(SystemMessage(SMNotEnoughSoulshots))
		} else {
			c.Send(SystemMessage(SMNotEnoughSpiritshots))
		}
		return false
	}
	if !RemoveItemCount(p, item.ObjectID, need) {
		return false
	}
	switch kind {
	case shotKindSS:
		p.ChargedSS = true
		c.Send(SystemMessage(SMEnabledSoulshot))
	case shotKindSPS:
		p.ChargedSPS = true
		c.Send(SystemMessage(SMEnabledSpiritshot))
	case shotKindBSS:
		p.ChargedBSS = true
		c.Send(SystemMessage(SMEnabledSpiritshot))
	}
	if tpl.SkillID > 0 {
		pkt := MagicSkillUse(p.ObjectID, p.ObjectID, tpl.SkillID, 1, 0, 0, p.X, p.Y, p.Z, p.X, p.Y, p.Z, true)
		c.Send(pkt)
		c.Broadcast(pkt)
	}
	c.Send(ItemList(p.Items, false))
	return true
}

func (s *Server) rechargeShots(c *GameClient, physical, magic bool) {
	p := c.Player()
	if p == nil || len(p.AutoShots) == 0 {
		return
	}
	for _, id := range append([]int32(nil), p.AutoShots...) {
		it := FindItemByID(p, id)
		if it == nil {
			removeAutoShot(p, id)
			c.Send(ExAutoSoulShot(id, 0))
			continue
		}
		tpl := GetItem(id)
		kind := shotKind(tpl)
		if physical && kind == shotKindSS {
			s.chargeShot(c, it, true)
		}
		if magic && (kind == shotKindSPS || kind == shotKindBSS) {
			s.chargeShot(c, it, true)
		}
	}
}

func (s *Server) onAutoSoulShot(c *GameClient, r *packet.Reader) {
	p := c.Player()
	itemID := r.ReadD()
	typ := r.ReadD()
	if p.AlikeDead() || p.PrivateStore != 0 {
		return
	}
	it := FindItemByID(p, itemID)
	if it == nil {
		return
	}
	if typ == 1 {
		if itemID >= 6535 && itemID <= 6540 {
			return
		}
		addAutoShot(p, itemID)
		c.Send(ExAutoSoulShot(itemID, 1))
		s.chargeShot(c, it, true)
		c.Send(SystemMessage(SMUseOfS1WillBeAuto, SysItem(itemID)))
		return
	}
	if typ == 0 {
		removeAutoShot(p, itemID)
		c.Send(ExAutoSoulShot(itemID, 0))
		c.Send(SystemMessage(SMAutoUseOfS1Cancelled, SysItem(itemID)))
	}
}

func consumeSoulshot(p *Character) bool {
	if !p.ChargedSS {
		return false
	}
	p.ChargedSS = false
	return true
}

func consumeSpiritshot(p *Character) (sps, bss bool) {
	if p.ChargedBSS {
		p.ChargedBSS = false
		return false, true
	}
	if p.ChargedSPS {
		p.ChargedSPS = false
		return true, false
	}
	return false, false
}
