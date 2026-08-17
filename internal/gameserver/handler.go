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
	// Extended Interlude opcodes are accepted and ignored unless implemented.
	_ = data
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
		x := r.ReadD()
		y := r.ReadD()
		z := r.ReadD()
		p.X, p.Y, p.Z = x, y, z
		c.logChange("pos=(%d,%d,%d) heading=%d via MoveBackwardToLocation", x, y, z, p.Heading)
		c.Broadcast(StopMove(p.ObjectID, x, y, z, p.Heading))
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
		target := r.ReadD()
		c.target = target
		c.logChange("target=%d via Action", target)
		c.Send(MyTargetSelected(target, 0))
		c.Send(TargetSelected(p.ObjectID, target, p.X, p.Y, p.Z))
	case 0x05: // RequestTarget
		target := r.ReadD()
		c.target = target
		c.logChange("target=%d via RequestTarget", target)
		c.Send(MyTargetSelected(target, 0))
	case 0x09:
		s.onLogout(c)
	case 0x0A: // AttackRequest
		target := r.ReadD()
		ox, oy, oz := r.ReadD(), r.ReadD(), r.ReadD()
		_ = r.ReadC()
		c.target = target
		c.logChange("attack target=%d from=(%d,%d,%d)", target, ox, oy, oz)
		c.Send(Attack(p.ObjectID, target, 10, 0, ox, oy, oz))
		c.Broadcast(Attack(p.ObjectID, target, 10, 0, ox, oy, oz))
	case 0x0F:
		c.Send(ItemList(p.Items, true))
	case 0x10: // RequestInventoryUpdateOrder
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
	case 0x11, 0x12, 0x14:
		c.Send(ItemList(p.Items, false))
	case 0x1C:
		c.Send(ChangeMoveType(p.ObjectID, true))
	case 0x2F: // RequestMagicSkillUse
		skillID := r.ReadD()
		_ = r.ReadD() // ctrlPressed
		_ = r.ReadC() // shiftPressed
		var lvl int32
		for _, sk := range p.Skills {
			if sk.ID == skillID {
				lvl = sk.Level
			}
		}
		if lvl == 0 {
			c.Send(ActionFailed())
			break
		}
		hit, reuse := int32(500), int32(1000)
		if tpl := GetSkill(skillID, lvl); tpl != nil {
			if tpl.HitTime > 0 {
				hit = tpl.HitTime
			}
			if tpl.ReuseDelay > 0 {
				reuse = tpl.ReuseDelay
			}
		}
		target := c.target
		if target == 0 {
			target = p.ObjectID
		}
		c.logChange("skill=%d lvl=%d hit=%d reuse=%d target=%d pos=(%d,%d,%d)", skillID, lvl, hit, reuse, target, p.X, p.Y, p.Z)
		pkt := MagicSkillUse(p.ObjectID, target, skillID, lvl, hit, reuse, p.X, p.Y, p.Z)
		c.Send(pkt)
		c.Broadcast(pkt)
	case 0x30:
		c.Send(UserInfo(p))
	case 0x38: // Say2
		text := r.ReadS()
		sayType := r.ReadD()
		if sayType == 2 && r.Remaining() > 0 {
			_ = r.ReadS()
		}
		c.logChange("say type=%d text=%q", sayType, text)
		msg := CreatureSay(p.ObjectID, sayType, p.Name, text)
		c.Send(msg)
		s.Broadcast(msg, nil)
	case 0x3F:
		c.Send(SkillList(p.Skills))
	case 0x45:
		action := r.ReadD()
		c.Send(SocialAction(p.ObjectID, action))
		c.Broadcast(SocialAction(p.ObjectID, action))
	case 0x48: // ValidatePosition
		x, y, z := r.ReadD(), r.ReadD(), r.ReadD()
		heading := r.ReadD()
		p.X, p.Y, p.Z, p.Heading = x, y, z, heading
		c.logChange("pos=(%d,%d,%d) heading=%d via ValidatePosition", x, y, z, heading)
	case 0x6D:
		c.Send(TeleportToLocation(p.ObjectID, p.X, p.Y, p.Z))
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
	s.world.AddPlayer(p)
	c.SetState(StateInGame)
	c.logChange("enter world name=%q oid=%d class=%d pos=(%d,%d,%d) players=%d npcs=%d", p.Name, p.ObjectID, p.ClassID, p.X, p.Y, p.Z, len(s.world.Players()), len(s.world.NPCs()))
	c.Send(UserInfo(p))
	c.Send(ItemList(p.Items, false))
	c.Send(SkillList(p.Skills))
	c.Send(ShortCutInit(p.Shortcuts))
	for _, other := range s.world.Players() {
		if other.ObjectID == p.ObjectID {
			continue
		}
		c.Send(CharInfo(other))
		// notify others
	}
	for _, n := range s.world.NPCs() {
		c.Send(NpcInfo(n))
	}
	s.Broadcast(CharInfo(p), c)
	_ = s.store.Update(c.ctx(), p)
}
