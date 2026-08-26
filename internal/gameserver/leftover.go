package gameserver

import (
	"fmt"

	"github.com/pblaravel/game-server-l2-go/internal/packet"
)

func QuestList() []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x80)
		w.WriteH(0)
	})
}

// NewCharacterSuccess is Java unused/NewCharacterSuccess (opcode 0x17 CharTemplates).
func NewCharacterSuccess() []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x17)
		ids := []int32{0, 10, 18, 25, 31, 38, 44, 49, 53}
		w.WriteD(int32(len(ids)))
		for _, id := range ids {
			tpl := startingClasses[id]
			w.WriteD(tpl.Race)
			w.WriteD(id)
			w.WriteD(0x46)
			w.WriteD(tpl.STR)
			w.WriteD(0x0a)
			w.WriteD(0x46)
			w.WriteD(tpl.DEX)
			w.WriteD(0x0a)
			w.WriteD(0x46)
			w.WriteD(tpl.CON)
			w.WriteD(0x0a)
			w.WriteD(0x46)
			w.WriteD(tpl.INT)
			w.WriteD(0x0a)
			w.WriteD(0x46)
			w.WriteD(tpl.WIT)
			w.WriteD(0x0a)
			w.WriteD(0x46)
			w.WriteD(tpl.MEN)
			w.WriteD(0x0a)
		}
	})
}

func MacroList() []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0xE7)
		w.WriteD(0)
		w.WriteC(0)
		w.WriteC(0)
		w.WriteC(0)
	})
}

func EtcStatusUpdate(p *Character) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0xF3)
		w.WriteD(0) // charges
		w.WriteD(0) // weight penalty
		w.WriteD(0) // weapon
		w.WriteD(0) // chat banned
		w.WriteD(0) // danger area
		w.WriteD(0) // expertise
		_ = p       // weight/charge fields stay 0 until penalty tables are ported
	})
}

func AbnormalStatusUpdate() []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x7F)
		w.WriteH(0)
	})
}

func ShowBoard(html string) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x6E)
		w.WriteC(1)
		w.WriteS("")
		w.WriteS("")
		w.WriteS("")
		w.WriteS("")
		w.WriteS(html)
	})
}

func SkillCoolTime() []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0xC1)
		w.WriteD(0)
	})
}

func PledgeInfo(clanID int32, name, ally string) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x83)
		w.WriteD(clanID)
		w.WriteS(name)
		w.WriteS(ally)
	})
}

func ManagePledgePower(rank, action, privs int32) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x30)
		w.WriteD(rank)
		w.WriteD(action)
		w.WriteD(privs)
	})
}

func PackageSendableList(adena int32) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0xC3)
		w.WriteD(adena)
		w.WriteD(0)
		w.WriteD(0)
	})
}

func TargetUnselected(objectID, x, y, z int32) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x2A)
		w.WriteD(objectID)
		w.WriteD(x)
		w.WriteD(y)
		w.WriteD(z)
	})
}

func ShortCutDel(slot int32) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x46)
		w.WriteD(slot)
	})
}

func ShopPreviewInfo() []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0xF0)
		w.WriteD(0)
	})
}

func ShowMiniMap(mapID, period int32) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x9d)
		w.WriteD(mapID)
		w.WriteD(period)
	})
}

func (s *Server) onQuestList(c *GameClient) {
	c.Send(QuestList())
}

func (s *Server) onShowMiniMap(c *GameClient) {
	c.Send(ShowMiniMap(1665, 0))
}

func (s *Server) onCannotMoveAnymore(c *GameClient, r *packet.Reader) {
	p := c.Player()
	x, y, z, heading := r.ReadD(), r.ReadD(), r.ReadD(), r.ReadD()
	p.X, p.Y, p.Z, p.Heading = x, y, z, heading
	pkt := StopMove(p.ObjectID, x, y, z, heading)
	c.Send(pkt)
	c.Broadcast(pkt)
}

func lootRuleMessage(loot int32) int32 {
	switch loot {
	case LootRandom:
		return SMLootingRandom
	case LootRandomAll:
		return SMLootingRandomIncludeSpoil
	case LootByTurn:
		return SMLootingByTurn
	case LootByTurnAll:
		return SMLootingByTurnIncludeSpoil
	default:
		return SMLootingFindersKeepers
	}
}

func (s *Server) onUserCommand(c *GameClient, r *packet.Reader) {
	p := c.Player()
	switch r.ReadD() {
	case 0: // /loc
		c.Send(SystemMessage(SMLocTalkingIsland, SysNumber(p.X), SysNumber(p.Y), SysNumber(p.Z)))
	case 52: // /unstuck — Java casts skill 2099 (5 min); we teleport now.
		c.Send(SystemMessage(SMStuckTransportInFiveMinutes))
		loc := NearestRestartLocation(p)
		s.teleportPlayer(c, loc[0], loc[1], loc[2])
	case 77: // /time
		minutes := s.world.GameTime()
		hour := minutes / 60
		mins := minutes % 60
		id := SMTimeS1S2InTheDay
		if hour < 6 {
			id = SMTimeS1S2InTheNight
		}
		c.Send(SystemMessage(id, SysNumber(hour), SysText(fmt.Sprintf("%02d", mins))))
	case 81: // /partyinfo
		pt := s.partyOf(p)
		if pt == nil {
			return
		}
		leader := s.world.GetPlayer(pt.LeaderID)
		name := ""
		if leader != nil {
			name = leader.Name
		}
		c.Send(SystemMessage(SMPartyInformation))
		c.Send(SystemMessage(lootRuleMessage(pt.Loot)))
		c.Send(SystemMessage(SMPartyLeaderS1, SysText(name)))
	default:
		c.Send(ActionFailed())
	}
}
