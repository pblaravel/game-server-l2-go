package gameserver

import (
	"log"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/pblaravel/game-server-l2-go/internal/packet"
)

var nameRE = regexp.MustCompile(`^[A-Za-z0-9]{1,16}$`)

func (s *Server) handle(c *GameClient, data []byte) {
	if len(data) == 0 {
		return
	}
	op := data[0]
	if op == 0xD0 && len(data) >= 3 {
		s.handleEx(c, data)
		return
	}
	switch c.State() {
	case StateConnected:
		switch op {
		case 0x00:
			s.onProtocolVersion(c, data)
		case 0x08:
			s.onAuthLogin(c, data)
		default:
			log.Printf("[GAME] %s unhandled CONNECTED opcode 0x%02X %s (expected ProtocolVersion 0x00)", c.tag(), op, hexPreview(data, 64))
		}
	case StateAuthed:
		switch op {
		case 0x09:
			s.onLogout(c)
		case 0x0B:
			s.onCharCreate(c, data)
		case 0x0C:
			s.onCharDelete(c, data)
		case 0x0D:
			s.onGameStart(c, data)
		case 0x0E:
			c.Send(CharSelectInfo(c.AccountName(), c.SessionKey().PlayOkID1, c.Slots()))
		case 0x62:
			s.onCharRestore(c, data)
		}
	case StateEntering:
		switch op {
		case 0x03:
			s.onEnterWorld(c)
		case 0x3F:
			c.Send(SkillList(c.Player().Skills))
		}
	case StateInGame:
		s.handleInGame(c, op, data)
	}
}

func (s *Server) handleEx(c *GameClient, data []byte) {
	if c.State() != StateInGame || c.Player() == nil {
		return
	}
	r := packet.NewReader(data)
	r.SkipOpcode()
	switch r.ReadH() {
	case 4: // RequestChangePartyLeader
		s.onChangePartyLeader(c, r)
	case 5: // RequestAutoSoulShot
		s.onAutoSoulShot(c, r)
	}
}

func (s *Server) handleInGame(c *GameClient, op byte, data []byte) {
	p := c.Player()
	if p == nil {
		return
	}
	r := packet.NewReader(data)
	r.SkipOpcode()
	switch op {
	case 0x01: // MoveBackwardToLocation
		s.onMoveBackwardToLocation(c, r)
	case 0x02: // PlayerMoveDirection (Unity)
		need := r.ReadC() == 1
		dirY := r.ReadF64()
		dirX := r.ReadF64()
		heading := r.ReadD()
		vert := r.ReadF64()
		x := r.ReadD()
		y := r.ReadD()
		z := r.ReadD()
		ts := r.ReadQ()
		p.Heading = heading
		p.X, p.Y, p.Z = x, y, z
		p.MoveDirX = int32(dirX * 100)
		p.MoveDirY = int32(dirY * 100)
		p.VerticalVel = int32(vert * 100)
		p.LastPacketTS = ts
		c.logChange("pos=(%d,%d,%d) heading=%d dir=(%d,%d) vert=%d via PlayerMoveDirection", x, y, z, heading, p.MoveDirX, p.MoveDirY, p.VerticalVel)
		pkt := MoveDirection(p.ObjectID, p.MoveDirY, p.MoveDirX, p.VerticalVel, x, y, z, ts)
		c.Send(pkt)
		c.Broadcast(pkt)
		if need {
			// ActionFailed only when blocked; here movement is accepted.
		}
	case 0x04: // Action
		s.onAction(c, r)
	case 0x05: // RequestTarget
		target := r.ReadD()
		c.target = target
		c.logChange("target=%d via RequestTarget", target)
		c.Send(MyTargetSelected(target, 0))
	case 0x09: // Logout
		s.onLogout(c)
	case 0x0A: // AttackRequest
		s.onAttackRequest(c, r)
	case 0x0F: // RequestItemList
		c.Send(ItemList(p.Items, true))
	case 0x10: // RequestInventoryUpdateOrder
		s.onInventoryUpdateOrder(c, r)
	case 0x11: // RequestUnEquipItem
		s.onUnEquipItem(c, r)
	case 0x12: // RequestDropItem
		s.onDropItem(c, r)
	case 0x14: // UseItem
		s.onUseItem(c, r)
	case 0x15: // TradeRequest
		s.onTradeRequest(c, r)
	case 0x16: // AddTradeItem
		s.onAddTradeItem(c, r)
	case 0x17: // TradeDone
		s.onTradeDone(c, r)
	case 0x1A, 0x23, 0x2E, 0x34, 0x3E: // Java DummyPacket
	case 0x1B: // RequestSocialAction
		s.onSocialAction(c, r)
	case 0x1C: // RequestChangeMoveType
		s.onChangeMoveType(c)
	case 0x1D: // RequestChangeWaitType
		s.onChangeWaitType(c, r)
	case 0x1E: // RequestSellItem
		s.onSellItem(c, r)
	case 0x1F: // RequestBuyItem
		s.onBuyItem(c, r)
	case 0x21: // RequestBypassToServer
		s.onBypassToServer(c, r)
	case 0x29: // RequestJoinParty
		s.onJoinParty(c, r)
	case 0x2A: // RequestAnswerJoinParty
		s.onAnswerJoinParty(c, r)
	case 0x2B: // RequestWithdrawParty
		s.onWithdrawParty(c)
	case 0x2C: // RequestOustPartyMember
		s.onOustPartyMember(c, r)
	case 0x2F: // RequestMagicSkillUse
		s.onMagicSkillUse(c, r)
	case 0x30: // Appearing
		s.onAppearing(c)
	case 0x31: // SendWarehouseDepositList
		s.onWarehouseDeposit(c, r)
	case 0x32: // SendWarehouseWithdrawList
		s.onWarehouseWithdraw(c, r)
	case 0x33: // RequestShortCutReg
		s.onShortCutReg(c, r)
	case 0x35: // RequestShortCutDel
		s.onShortCutDel(c, r)
	case 0x36: // CannotMoveAnymore
		s.onCannotMoveAnymore(c, r)
	case 0x37: // RequestTargetCancel
		s.onTargetCancel(c, r)
	case 0x38: // Say2
		s.onSay2(c, r)
	case 0x3F: // RequestSkillList
		c.Send(SkillList(p.Skills))
	case 0x44: // AnswerTradeRequest
		s.onAnswerTradeRequest(c, r)
	case 0x45: // RequestActionUse
		s.onActionUse(c, r)
	case 0x46: // RequestRestart
		s.onRestart(c)
	case 0x48: // ValidatePosition
		s.onValidatePosition(c, r)
	case 0x58: // RequestEnchantItem
		s.onEnchantItem(c, r)
	case 0x59: // RequestDestroyItem
		s.onDestroyItem(c, r)
	case 0x5E: // RequestFriendInvite
		s.onFriendInvite(c, r)
	case 0x5F: // RequestAnswerFriendInvite
		s.onAnswerFriendInvite(c, r)
	case 0x60: // RequestFriendList
		s.onFriendList(c)
	case 0x61: // RequestFriendDel
		s.onFriendDel(c, r)
	case 0x63: // RequestQuestList
		s.onQuestList(c)
	case 0x6B: // RequestAcquireSkillInfo
		s.onAcquireSkillInfo(c, r)
	case 0x6C: // RequestAcquireSkill
		s.onAcquireSkill(c, r)
	case 0x6D: // RequestRestartPoint
		s.onRestartPoint(c, r)
	case 0x72: // RequestCrystallizeItem
		s.onCrystallizeItem(c, r)
	case 0x73: // RequestPrivateStoreManageSell
		s.onPrivateStoreManageSell(c, r)
	case 0x74: // SetPrivateStoreListSell
		s.onSetPrivateStoreListSell(c, r)
	case 0x76: // RequestPrivateStoreQuitSell
		s.quitPrivateStore(c)
	case 0x77: // SetPrivateStoreMsgSell
		s.onSetPrivateStoreMsgSell(c, r)
	case 0x79: // RequestPrivateStoreBuy
		s.onPrivateStoreBuy(c, r)
	case 0x90: // RequestPrivateStoreManageBuy
		s.openBuyStoreManage(c)
	case 0x91: // SetPrivateStoreListBuy
		s.onSetPrivateStoreListBuy(c, r)
	case 0x93: // RequestPrivateStoreQuitBuy
		s.quitPrivateStore(c)
	case 0x94: // SetPrivateStoreMsgBuy
		s.onSetPrivateStoreMsgBuy(c, r)
	case 0x96: // RequestPrivateStoreSell
		s.onPrivateStoreSell(c, r)
	case 0xA7: // MultiSellChoose
		s.onMultiSellChoose(c, r)
	case 0xAA: // RequestUserCommand
		s.onUserCommand(c, r)
	case 0xAC: // RequestRecipeBookOpen
		s.onRecipeBookOpen(c, r)
	case 0xAD: // RequestRecipeBookDestroy
		s.onRecipeBookDestroy(c, r)
	case 0xAE: // RequestRecipeItemMakeInfo
		s.onRecipeItemMakeInfo(c, r)
	case 0xAF: // RequestRecipeItemMakeSelf
		s.onRecipeItemMakeSelf(c, r)
	case 0xBA: // RequestHennaItemList
		s.onHennaItemList(c, r)
	case 0xBB: // RequestHennaItemInfo
		s.onHennaItemInfo(c, r)
	case 0xBC: // RequestHennaEquip
		s.onHennaEquip(c, r)
	case 0xBD: // RequestHennaUnequipList
		s.onHennaUnequipList(c, r)
	case 0xBE: // RequestHennaUnequipInfo
		s.onHennaUnequipInfo(c, r)
	case 0xBF: // RequestHennaUnequip
		s.onHennaUnequip(c, r)
	case 0xCD: // RequestShowMiniMap
		s.onShowMiniMap(c)
	default:
		if s.cfg.PacketHandlerDebug || s.cfg.PrintReceivedPackets || s.cfg.Developer {
			log.Printf("GS UNHANDLED %s opcode 0x%02X %s", c.tag(), op, hexPreview(data, 32))
		}
	}
}

func (s *Server) onProtocolVersion(c *GameClient, data []byte) {
	ver := parseProtocolVersion(data)
	addr := ""
	if c.conn != nil {
		addr = c.conn.RemoteAddr().String()
	}
	allowed := s.protocolAllowed(ver)
	log.Printf("[GAME] %s ProtocolVersion opcode=0x%02X ver=%d raw=%s allowed=%v list=%v cipher=%v",
		addr, data[0], ver, hexPreview(data, 64), allowed, s.cfg.AllowedProtocolVers, s.cfg.UseBlowfishCipher)

	key := c.EnableCrypt()
	if len(key) < 8 {
		log.Printf("[GAME] %s crypt key missing, closing", addr)
		c.Close()
		return
	}
	if !allowed && s.cfg.StrictProtocol {
		log.Printf("[GAME] %s rejecting protocol %d (StrictProtocol), sending VersionCheck fail then close", addr, ver)
		c.Send(VersionCheckReply(key[:8], s.cfg.UseBlowfishCipher, false))
		c.Close()
		return
	}
	if !allowed {
		log.Printf("[GAME] %s protocol %d not in allowed list, accepting anyway (set StrictProtocol=true to reject)", addr, ver)
	}
	log.Printf("[GAME] %s sending VersionCheck ok=1 key8=%x", addr, key[:8])
	c.Send(VersionCheck(key[:8], s.cfg.UseBlowfishCipher))
}

func parseProtocolVersion(data []byte) int {
	r := packet.NewReader(data)
	r.SkipOpcode()
	switch {
	case r.Remaining() >= 4:
		return int(r.ReadD())
	case r.Remaining() >= 2:
		return r.ReadH()
	default:
		return 0
	}
}

func (s *Server) protocolAllowed(ver int) bool {
	if len(s.cfg.AllowedProtocolVers) == 0 {
		return true
	}
	for _, v := range s.cfg.AllowedProtocolVers {
		if v == ver {
			return true
		}
	}
	return false
}

func (s *Server) onAuthLogin(c *GameClient, data []byte) {
	r := packet.NewReader(data)
	r.SkipOpcode()
	login := strings.ToLower(r.ReadS())
	playKey2 := r.ReadD()
	playKey1 := r.ReadD()
	loginKey1 := r.ReadD()
	loginKey2 := r.ReadD()
	s.login.AddClient(login, loginKey1, loginKey2, playKey1, playKey2, c)
}

func (s *Server) onLogout(c *GameClient) {
	if p := c.Player(); p != nil {
		s.cancelTradeFor(p.ObjectID)
	}
	c.Send(LeaveWorld())
	c.Close()
}

func (s *Server) onCharCreate(c *GameClient, data []byte) {
	r := packet.NewReader(data)
	r.SkipOpcode()
	name := r.ReadS()
	race := r.ReadD()
	sex := r.ReadD()
	classID := r.ReadD()
	for i := 0; i < 6; i++ {
		_ = r.ReadD()
	}
	hair := r.ReadD()
	color := r.ReadD()
	face := r.ReadD()

	if race < 0 || race > 4 || face < 0 || face > 2 || color < 0 || color > 3 {
		c.Send(CharCreateFail(CharCreateFailed))
		return
	}
	if hair < 0 || (sex == 0 && hair > 4) || (sex != 0 && hair > 6) {
		c.Send(CharCreateFail(CharCreateFailed))
		return
	}
	if !nameRE.MatchString(name) || utf8.RuneCountInString(name) == 0 {
		c.Send(CharCreateFail(CharCreateIncorrectName))
		return
	}
	n, _ := s.store.CountByAccount(c.ctx(), c.AccountName())
	if n >= 7 {
		c.Send(CharCreateFail(CharCreateTooMany))
		return
	}
	if id, _ := s.store.GetObjectIDByName(c.ctx(), name); id > 0 {
		c.Send(CharCreateFail(CharCreateNameExists))
		return
	}
	if _, ok := startingClasses[classID]; !ok {
		c.Send(CharCreateFail(CharCreateFailed))
		return
	}
	oid, err := s.store.NextObjectID(c.ctx())
	if err != nil {
		c.Send(CharCreateFail(CharCreateFailed))
		return
	}
	ch := DefaultCharacter(c.AccountName(), name, classID, race, sex, hair, color, face, oid, func() int32 {
		id, err := s.store.NextObjectID(c.ctx())
		if err != nil {
			return 0
		}
		return id
	})
	ch.LastAccess = time.Now().UnixMilli()
	if err := s.store.Create(c.ctx(), ch); err != nil {
		c.Send(CharCreateFail(CharCreateFailed))
		return
	}
	c.logChange("created char name=%q oid=%d class=%d race=%d sex=%d spawn=(%d,%d,%d)", ch.Name, ch.ObjectID, ch.ClassID, ch.Race, ch.Sex, ch.X, ch.Y, ch.Z)
	c.Send(CharCreateOk())
	slots, _ := s.store.ListByAccount(c.ctx(), c.AccountName())
	c.SetSlots(slots)
	c.Send(CharSelectInfo(c.AccountName(), c.SessionKey().PlayOkID1, slots))
}

func (s *Server) onCharDelete(c *GameClient, data []byte) {
	r := packet.NewReader(data)
	r.SkipOpcode()
	slot := int(r.ReadD())
	if slot < 0 || slot >= len(c.Slots()) {
		c.Send(CharDeleteFail(1))
		return
	}
	_ = s.store.Delete(c.ctx(), c.Slots()[slot].ObjectID)
	c.Send(CharDeleteOk())
	slots, _ := s.store.ListByAccount(c.ctx(), c.AccountName())
	c.SetSlots(slots)
	c.Send(CharSelectInfo(c.AccountName(), c.SessionKey().PlayOkID1, slots))
}

func (s *Server) onCharRestore(c *GameClient, data []byte) {
	c.Send(CharSelectInfo(c.AccountName(), c.SessionKey().PlayOkID1, c.Slots()))
}

func (s *Server) onGameStart(c *GameClient, data []byte) {
	r := packet.NewReader(data)
	r.SkipOpcode()
	slot := int(r.ReadD())
	if slot < 0 || slot >= len(c.Slots()) {
		return
	}
	ch, err := s.store.GetByObjectID(c.ctx(), c.Slots()[slot].ObjectID)
	if err != nil || ch == nil {
		return
	}
	c.SetPlayer(ch)
	c.SetState(StateEntering)
	c.Send(CharSelected(ch, c.SessionKey().PlayOkID1, s.world.GameTime()))
}

func (s *Server) onEnterWorld(c *GameClient) {
	p := c.Player()
	if p == nil {
		return
	}
	p.Online = true
	p.LastAccess = time.Now().UnixMilli()
	RecalcStats(p)
	s.world.AddPlayer(p)
	c.SetState(StateInGame)
	c.logChange("enter world name=%q oid=%d class=%d pos=(%d,%d,%d) players=%d npcs=%d", p.Name, p.ObjectID, p.ClassID, p.X, p.Y, p.Z, len(s.world.Players()), len(s.world.NPCs()))
	c.Send(UserInfo(p))
	c.Send(ItemList(p.Items, false))
	c.Send(SkillList(p.Skills))
	c.Send(ShortCutInit(p.Shortcuts))
	c.Send(StatusUpdate(p.ObjectID, [][2]int32{
		{StatusCurHP, int32(p.CurHP)}, {StatusMaxHP, p.MaxHP},
		{StatusCurMP, int32(p.CurMP)}, {StatusMaxMP, p.MaxMP},
		{StatusCurCP, int32(p.CurCP)}, {StatusMaxCP, p.MaxCP},
		{StatusCurLoad, p.CurrentWeight}, {StatusMaxLoad, p.WeightLimit},
	}))
	for _, other := range s.world.Players() {
		if other.ObjectID == p.ObjectID {
			continue
		}
		c.Send(CharInfo(other))
	}
	for _, n := range s.world.NPCs() {
		c.Send(NpcInfo(n))
	}
	s.sendNearbyGroundItems(c)
	if p.Dead {
		c.Send(Die(p.ObjectID, false, false, false, false, false))
	}
	s.Broadcast(CharInfo(p), c)
	for _, a := range LoginAnnouncements() {
		say := int32(10) // SayType.ANNOUNCEMENT
		if a.Critical {
			say = 23 // SayType.CRITICAL_ANNOUNCE
		}
		c.Send(CreatureSay(0, say, "", a.Message))
	}
	c.Send(ActionFailed())
	_ = s.store.Update(c.ctx(), p)
}
