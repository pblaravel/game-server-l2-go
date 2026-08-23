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
		min := minutes % 60
		id := SMTimeS1S2InTheDay
		if hour < 6 {
			id = SMTimeS1S2InTheNight
		}
		c.Send(SystemMessage(id, SysNumber(hour), SysText(fmt.Sprintf("%02d", min))))
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
