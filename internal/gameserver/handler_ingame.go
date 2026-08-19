package gameserver

import (
	"log"
	"time"

	"github.com/pblaravel/game-server-l2-go/internal/packet"
)

// Handlers for the in-game opcodes of Java network/GamePacketHandler.

// onMoveBackwardToLocation is Java clientpackets/movement/legacy/MoveBackwardToLocation.
func (s *Server) onMoveBackwardToLocation(c *GameClient, r *packet.Reader) {
	p := c.Player()
	destX, destY, destZ := r.ReadD(), r.ReadD(), r.ReadD()
	originX, originY, originZ := r.ReadD(), r.ReadD(), r.ReadD()
	if r.Remaining() >= 4 {
		_ = r.ReadD() // moveMovement: 0 keyboard, 1 mouse
	}
	if p.Sitting {
		c.Send(SystemMessage(SMCantMoveSitting))
		c.Send(ActionFailed())
		return
	}
	if p.AlikeDead() {
		c.Send(ActionFailed())
		return
	}
	p.X, p.Y, p.Z = originX, originY, originZ
	p.DestX, p.DestY, p.DestZ = destX, destY, destZ
	p.Heading = headingTo(originX, originY, destX, destY)
	c.logChange("move to (%d,%d,%d) from (%d,%d,%d)", destX, destY, destZ, originX, originY, originZ)
	pkt := MoveToLocation(p.ObjectID, destX, destY, destZ, originX, originY, originZ)
	c.Send(pkt)
	c.Broadcast(pkt)
}

// onAction is Java clientpackets/combat/Action: first click selects a target,
// a click on an already selected attackable starts the attack.
func (s *Server) onAction(c *GameClient, r *packet.Reader) {
	p := c.Player()
	targetID := r.ReadD()
	_, _, _ = r.ReadD(), r.ReadD(), r.ReadD() // origin
	shift := false
	if r.Remaining() >= 1 {
		shift = r.ReadC() != 0
	}
	if p.AlikeDead() {
		c.Send(ActionFailed())
		return
	}
	npc := s.world.GetNPC(targetID)
	if npc == nil {
		if other := s.world.GetPlayer(targetID); other != nil {
			c.target = targetID
			c.Send(MyTargetSelected(targetID, 0))
			return
		}
		c.Send(ActionFailed())
		return
	}
	if c.target != targetID {
		c.target = targetID
		c.logChange("target=%d (%s) via Action", targetID, npc.Name)
		c.Send(MyTargetSelected(targetID, targetColor(p, npc)))
		c.Send(ValidateLocation(npc.ObjectID, npc.X, npc.Y, npc.Z, npc.Heading))
		if npc.IsAttackable {
			c.Send(StatusUpdate(npc.ObjectID, [][2]int32{
				{StatusMaxHP, npc.MaxHP}, {StatusCurHP, npc.CurHP},
			}))
		}
		return
	}
	if npc.IsAttackable && !shift {
		s.startAttack(c, npc)
		return
	}
	c.Send(ActionFailed())
}

// onAttackRequest is Java clientpackets/combat/AttackRequest.
func (s *Server) onAttackRequest(c *GameClient, r *packet.Reader) {
	p := c.Player()
	targetID := r.ReadD()
	_, _, _ = r.ReadD(), r.ReadD(), r.ReadD() // origin
	if r.Remaining() >= 1 {
		_ = r.ReadC() // shift
	}
	if p.AlikeDead() || p.Sitting {
		c.Send(ActionFailed())
		return
	}
	npc := s.world.GetNPC(targetID)
	if npc == nil || !npc.IsAttackable {
		c.Send(SystemMessage(SMInvalidTarget))
		c.Send(ActionFailed())
		return
	}
	c.target = targetID
	s.startAttack(c, npc)
}

// onInventoryUpdateOrder is Java clientpackets/item/RequestInventoryUpdateOrder.
func (s *Server) onInventoryUpdateOrder(c *GameClient, r *packet.Reader) {
	p := c.Player()
	n := int(r.ReadD())
	if n > 125 {
		n = 125
	}
	for i := 0; i < n; i++ {
		oid := r.ReadD()
		order := r.ReadD()
		for j := range p.Items {
			if p.Items[j].ObjectID == oid {
				p.Items[j].Slot = order
			}
		}
	}
	c.Send(ItemList(p.Items, false))
}

// onUnEquipItem is Java clientpackets/item/RequestUnEquipItem (slot is a body part).
func (s *Server) onUnEquipItem(c *GameClient, r *packet.Reader) {
	p := c.Player()
	slot := r.ReadD()
	if p.AlikeDead() {
		c.Send(ActionFailed())
		return
	}
	if !UnequipBodyPart(p, slot) {
		c.Send(ActionFailed())
		return
	}
	s.refreshAppearance(c)
}

// onUseItem is Java clientpackets/item/UseItem: equipable items toggle equipment.
func (s *Server) onUseItem(c *GameClient, r *packet.Reader) {
	p := c.Player()
	objectID := r.ReadD()
	if r.Remaining() >= 4 {
		_ = r.ReadD() // ctrlPressed
	}
	if p.AlikeDead() {
		c.Send(ActionFailed())
		return
	}
	item := FindItem(p, objectID)
	if item == nil {
		c.Send(ActionFailed())
		return
	}
	if item.BodyPart == 0 {
		// Java hands non-equipable items to an ItemHandler; none are ported yet.
		c.Send(ActionFailed())
		return
	}
	if item.Equipped {
		UnequipBodyPart(p, item.BodyPart)
	} else {
		EquipItem(p, objectID)
		c.Send(SystemMessage(SMItemEquipped, SysItem(item.ItemID)))
	}
	s.refreshAppearance(c)
}

// onDropItem is Java clientpackets/item/RequestDropItem.
func (s *Server) onDropItem(c *GameClient, r *packet.Reader) {
	p := c.Player()
	objectID := r.ReadD()
	count := r.ReadD()
	x, y, z := r.ReadD(), r.ReadD(), r.ReadD()
	item := FindItem(p, objectID)
	if item == nil || count <= 0 || count > item.Count {
		c.Send(ActionFailed())
		return
	}
	if item.Equipped {
		c.Send(SystemMessage(SMCannotDiscardThisItem))
		c.Send(ActionFailed())
		return
	}
	RemoveItemCount(p, objectID, count)
	c.logChange("dropped item=%d count=%d at (%d,%d,%d)", item.ItemID, count, x, y, z)
	c.Send(ItemList(p.Items, false))
	s.sendWeightAndStats(c)
}

// onSocialAction is Java clientpackets/RequestSocialAction.
func (s *Server) onSocialAction(c *GameClient, r *packet.Reader) {
	p := c.Player()
	action := r.ReadD()
	if action < 2 || action > 13 {
		return
	}
	if p.AlikeDead() || p.PrivateStore != 0 {
		return
	}
	pkt := SocialAction(p.ObjectID, action)
	c.Send(pkt)
	c.Broadcast(pkt)
}

// onChangeMoveType is Java clientpackets/movement/RequestChangeMoveType.
func (s *Server) onChangeMoveType(c *GameClient) {
	p := c.Player()
	p.Running = !p.Running
	pkt := ChangeMoveType(p.ObjectID, p.Running)
	c.Send(pkt)
	c.Broadcast(pkt)
}

// onActionUse is Java clientpackets/combat/RequestActionUse. Pet, mount and
// private store actions need subsystems that are not ported.
func (s *Server) onActionUse(c *GameClient, r *packet.Reader) {
	p := c.Player()
	action := r.ReadD()
	if r.Remaining() >= 4 {
		_ = r.ReadD() // ctrlPressed
	}
	if r.Remaining() >= 1 {
		_ = r.ReadC() // shiftPressed
	}
	if p.AlikeDead() {
		c.Send(ActionFailed())
		return
	}
	switch action {
	case 0: // sit / stand
		p.Sitting = !p.Sitting
		waitType := WaitTypeStanding
		if p.Sitting {
			waitType = WaitTypeSitting
		}
		pkt := ChangeWaitType(p.ObjectID, waitType, p.X, p.Y, p.Z)
		c.Send(pkt)
		c.Broadcast(pkt)
	case 1: // walk / run
		s.onChangeMoveType(c)
	default:
		log.Printf("[GAME] %s unhandled action type %d", c.tag(), action)
		c.Send(ActionFailed())
	}
}

// onTargetCancel is Java clientpackets/combat/RequestTargetCancel.
func (s *Server) onTargetCancel(c *GameClient, r *packet.Reader) {
	unselect := r.ReadH()
	if unselect == 0 {
		// Java aborts the current cast here; without a cast pipeline nothing to abort.
		c.Send(ActionFailed())
		return
	}
	c.target = 0
}

// onSay2 is Java clientpackets/Say2.
func (s *Server) onSay2(c *GameClient, r *packet.Reader) {
	p := c.Player()
	text := r.ReadS()
	sayType := r.ReadD()
	target := ""
	if sayType == 2 && r.Remaining() > 0 {
		target = r.ReadS()
	}
	if text == "" {
		return
	}
	c.logChange("say type=%d text=%q", sayType, text)
	msg := CreatureSay(p.ObjectID, sayType, p.Name, text)
	if sayType == 2 && target != "" {
		c.Send(msg)
		if other := s.world.GetPlayerByName(target); other != nil {
			if oc := s.clientOf(other.ObjectID); oc != nil {
				oc.Send(msg)
			}
		}
		return
	}
	c.Send(msg)
	s.Broadcast(msg, c)
}

// onAppearing is Java clientpackets/Appearing (sent after a teleport).
func (s *Server) onAppearing(c *GameClient) {
	p := c.Player()
	c.Send(UserInfo(p))
	c.Broadcast(CharInfo(p))
}

// onValidatePosition is Java clientpackets/movement/ValidatePosition.
func (s *Server) onValidatePosition(c *GameClient, r *packet.Reader) {
	p := c.Player()
	x, y, z := r.ReadD(), r.ReadD(), r.ReadD()
	heading := r.ReadD()
	p.X, p.Y, p.Z, p.Heading = x, y, z, heading
	c.logChange("pos=(%d,%d,%d) heading=%d via ValidatePosition", x, y, z, heading)
}

// onRestart is Java clientpackets/auth/RequestRestart.
func (s *Server) onRestart(c *GameClient) {
	p := c.Player()
	if p == nil {
		return
	}
	if p.InCombat {
		c.Send(RestartResponse(false))
		return
	}
	_ = s.store.Update(c.ctx(), p)
	s.Broadcast(DeleteObject(p.ObjectID), c)
	s.world.RemovePlayer(p.ObjectID)
	p.Online = false
	c.SetPlayer(nil)
	c.target = 0
	c.SetState(StateAuthed)
	c.Send(RestartResponse(true))
	slots, _ := s.store.ListByAccount(c.ctx(), c.AccountName())
	c.SetSlots(slots)
	c.Send(CharSelectInfo(c.AccountName(), c.SessionKey().PlayOkID1, slots))
}

// onRestartPoint is Java clientpackets/combat/RequestRestartPoint. Java picks the
// destination from RestartPointData XML, which is not vendored; the town list in
// respawnPoints stands in for it.
func (s *Server) onRestartPoint(c *GameClient, r *packet.Reader) {
	p := c.Player()
	_ = r.ReadD() // requestType: village / clan hall / castle / fixed
	if !p.Dead {
		return
	}
	loc := nearestRespawn(p.X, p.Y, p.Z)
	p.X, p.Y, p.Z = loc[0], loc[1], loc[2]
	s.revive(c, RespawnHPPercent)
	c.Send(TeleportToLocation(p.ObjectID, p.X, p.Y, p.Z))
	c.Broadcast(TeleportToLocation(p.ObjectID, p.X, p.Y, p.Z))
	c.logChange("respawned at (%d,%d,%d) hp=%.0f", p.X, p.Y, p.Z, p.CurHP)
}

// onSellItem / onBuyItem are Java clientpackets/item/RequestSellItem and
// RequestBuyItem. Both need BuyListManager, which loads XML that is not vendored.
func (s *Server) onSellItem(c *GameClient, r *packet.Reader) {
	listID := r.ReadD()
	log.Printf("[GAME] %s RequestSellItem list=%d ignored: no BuyListManager data", c.tag(), listID)
	c.Send(ActionFailed())
}

func (s *Server) onBuyItem(c *GameClient, r *packet.Reader) {
	listID := r.ReadD()
	log.Printf("[GAME] %s RequestBuyItem list=%d ignored: no BuyListManager data", c.tag(), listID)
	c.Send(ActionFailed())
}

// onAcquireSkillInfo is Java clientpackets/skill/RequestAcquireSkillInfo.
func (s *Server) onAcquireSkillInfo(c *GameClient, r *packet.Reader) {
	p := c.Player()
	skillID := r.ReadD()
	skillLevel := r.ReadD()
	skillType := r.ReadD()
	if skillType != AcquireUsual {
		c.Send(ActionFailed())
		return
	}
	node, ok := findLearnableNode(p, skillID, skillLevel)
	if !ok {
		c.Send(ActionFailed())
		return
	}
	c.Send(AcquireSkillInfo(node.ID, node.Level, node.Cost, 0, nil))
}

// onAcquireSkill is Java clientpackets/skill/RequestAcquireSkill.
func (s *Server) onAcquireSkill(c *GameClient, r *packet.Reader) {
	p := c.Player()
	skillID := r.ReadD()
	skillLevel := r.ReadD()
	skillType := r.ReadD()
	if skillType != AcquireUsual {
		c.Send(ActionFailed())
		return
	}
	node, ok := findLearnableNode(p, skillID, skillLevel)
	if !ok {
		c.Send(ActionFailed())
		return
	}
	if p.SP < node.Cost {
		c.Send(SystemMessage(SMNotEnoughSP))
		return
	}
	p.SP -= node.Cost
	AddOrUpgradeSkill(p, node)
	c.Send(SystemMessage(SMLearnedSkill, SysSkill(node.ID, node.Level)))
	c.Send(AcquireSkillDone())
	c.Send(AcquireSkillList(AcquireUsual, LearnableSkills(p)))
	c.Send(SkillList(p.Skills))
	c.Send(UserInfo(p))
	_ = s.store.Update(c.ctx(), p)
	c.logChange("learned skill=%d lvl=%d cost=%d sp_left=%d", node.ID, node.Level, node.Cost, p.SP)
}

// onMagicSkillUse is Java clientpackets/skill/RequestMagicSkillUse.
func (s *Server) onMagicSkillUse(c *GameClient, r *packet.Reader) {
	p := c.Player()
	skillID := r.ReadD()
	if r.Remaining() >= 4 {
		_ = r.ReadD() // ctrlPressed
	}
	if r.Remaining() >= 1 {
		_ = r.ReadC() // shiftPressed
	}
	if p.AlikeDead() {
		c.Send(ActionFailed())
		return
	}
	var level int32
	for _, sk := range p.Skills {
		if sk.ID == skillID {
			level = sk.Level
		}
	}
	if level == 0 {
		c.Send(ActionFailed())
		return
	}
	s.castSkill(c, skillID, level)
}

func (s *Server) clientOf(objectID int32) *GameClient {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, c := range s.clients {
		if p := c.Player(); p != nil && p.ObjectID == objectID {
			return c
		}
	}
	return nil
}

// refreshAppearance resends the packets Java sends after an equipment change.
func (s *Server) refreshAppearance(c *GameClient) {
	p := c.Player()
	RecalcStats(p)
	c.Send(ItemList(p.Items, false))
	c.Send(UserInfo(p))
	c.Broadcast(CharInfo(p))
	_ = s.store.Update(c.ctx(), p)
}

func (s *Server) sendWeightAndStats(c *GameClient) {
	p := c.Player()
	RecalcStats(p)
	c.Send(UserInfo(p))
}

// respawnPoints are the Interlude town centers Java would read from
// RestartPointData (data/xml/restartPoints.xml is not vendored).
var respawnPoints = [][3]int32{
	{-71338, 258271, -3104}, // Talking Island
	{45470, 48328, -3059},   // Gludio
	{-12672, 122776, -3116}, // Gludin
	{15670, 142983, -2705},  // Dion
	{83043, 147923, -3403},  // Giran
	{111409, 219364, -3545}, // Heine
	{147134, -55413, -2735}, // Oren
	{82956, 53162, -1495},   // Hunter Village
	{116713, 76994, -2714},  // Aden
	{9745, 15606, -4574},    // Dark Elf Village
	{46934, 51467, -2977},   // Elven Village
	{-44836, -112524, -235}, // Orc Village
	{115113, -178212, -899}, // Dwarven Village
	{-80826, 149775, -3043}, // Floran
}

func nearestRespawn(x, y, z int32) [3]int32 {
	best := respawnPoints[0]
	bestDist := int64(-1)
	for _, loc := range respawnPoints {
		dx := int64(loc[0] - x)
		dy := int64(loc[1] - y)
		dz := int64(loc[2] - z)
		d := dx*dx + dy*dy + dz*dz
		if bestDist < 0 || d < bestDist {
			bestDist = d
			best = loc
		}
	}
	return best
}

// targetColor is Java MyTargetSelected: level difference tints the target frame.
func targetColor(p *Character, n *NPC) int16 {
	if !n.IsAttackable {
		return 0
	}
	return int16(p.Level - n.Level)
}

func (s *Server) touchPlayer(p *Character) {
	p.LastAccess = time.Now().UnixMilli()
}
