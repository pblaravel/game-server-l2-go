package gameserver

import (
	"sync"
	"time"

	"github.com/pblaravel/game-server-l2-go/internal/packet"
)

// LootRule ordinals from Java enums/LootRule.
const (
	LootFinder    int32 = 0
	LootRandom    int32 = 1
	LootRandomAll int32 = 2
	LootByTurn    int32 = 3
	LootByTurnAll int32 = 4
)

const partyMaxMembers = 9
const partyInviteTTL = 15 * time.Second

type Party struct {
	ID       int32
	LeaderID int32
	Loot     int32
	Members  []int32
	pending  bool
	inviteAt time.Time
}

type partyInvite struct {
	from int32
	loot int32
	at   time.Time
}

type partyState struct {
	mu      sync.Mutex
	nextID  int32
	parties map[int32]*Party
	invites map[int32]partyInvite
}

func newPartyState() *partyState {
	return &partyState{parties: map[int32]*Party{}, invites: map[int32]partyInvite{}, nextID: 1}
}

func (s *Server) partyOf(p *Character) *Party {
	if p == nil || p.PartyID == 0 {
		return nil
	}
	s.parties.mu.Lock()
	defer s.parties.mu.Unlock()
	return s.parties.parties[p.PartyID]
}

func (s *Server) partyMembers(pt *Party) []*Character {
	if pt == nil {
		return nil
	}
	out := make([]*Character, 0, len(pt.Members))
	for _, id := range pt.Members {
		if m := s.world.GetPlayer(id); m != nil {
			out = append(out, m)
		}
	}
	return out
}

func (s *Server) broadcastParty(pt *Party, payload []byte, except int32) {
	for _, m := range s.partyMembers(pt) {
		if m.ObjectID == except {
			continue
		}
		if c := s.clientOf(m.ObjectID); c != nil {
			c.Send(payload)
		}
	}
}

func (s *Server) onJoinParty(c *GameClient, r *packet.Reader) {
	p := c.Player()
	name := r.ReadS()
	loot := int32(0)
	if r.Remaining() >= 4 {
		loot = r.ReadD()
	}
	target := s.world.GetPlayerByName(name)
	if target == nil {
		if c.target != 0 {
			target = s.world.GetPlayer(c.target)
		}
	}
	if target == nil {
		c.Send(SystemMessage(SMSelectPartyTarget))
		return
	}
	if target.ObjectID == p.ObjectID {
		c.Send(ActionFailed())
		return
	}
	if target.PartyID != 0 {
		c.Send(SystemMessage(SMAlreadyInParty, SysText(target.Name)))
		return
	}
	if pt := s.partyOf(p); pt != nil {
		if pt.LeaderID != p.ObjectID {
			c.Send(SystemMessage(SMOnlyLeaderCanInvite))
			return
		}
		if len(pt.Members) >= partyMaxMembers {
			c.Send(SystemMessage(SMPartyFull))
			return
		}
		s.parties.mu.Lock()
		if pt.pending && time.Since(pt.inviteAt) < partyInviteTTL {
			s.parties.mu.Unlock()
			c.Send(SystemMessage(SMWaitingForReply))
			return
		}
		pt.pending = true
		pt.inviteAt = time.Now()
		s.parties.mu.Unlock()
	}
	tc := s.clientOf(target.ObjectID)
	if tc == nil {
		c.Send(SystemMessage(SMSelectPartyTarget))
		return
	}
	s.parties.mu.Lock()
	s.parties.invites[target.ObjectID] = partyInvite{from: p.ObjectID, loot: loot, at: time.Now()}
	s.parties.mu.Unlock()
	c.Send(SystemMessage(SMYouInvitedToParty, SysText(target.Name)))
	tc.Send(AskJoinParty(p.Name, loot))
}

func (s *Server) onAnswerJoinParty(c *GameClient, r *packet.Reader) {
	p := c.Player()
	resp := r.ReadD()
	s.parties.mu.Lock()
	inv, ok := s.parties.invites[p.ObjectID]
	delete(s.parties.invites, p.ObjectID)
	s.parties.mu.Unlock()
	if !ok || time.Since(inv.at) > partyInviteTTL {
		return
	}
	from := s.clientOf(inv.from)
	if from == nil {
		return
	}
	from.Send(JoinParty(resp))
	leader := from.Player()
	if leader == nil {
		return
	}
	if pt := s.partyOf(leader); pt != nil {
		s.parties.mu.Lock()
		pt.pending = false
		s.parties.mu.Unlock()
	}
	if resp != 1 {
		return
	}
	s.addToParty(leader, p, inv.loot)
}

func (s *Server) addToParty(leader, member *Character, loot int32) {
	pt := s.partyOf(leader)
	if pt == nil {
		s.parties.mu.Lock()
		s.parties.nextID++
		pt = &Party{ID: s.parties.nextID, LeaderID: leader.ObjectID, Loot: loot, Members: []int32{leader.ObjectID}}
		s.parties.parties[pt.ID] = pt
		leader.PartyID = pt.ID
		s.parties.mu.Unlock()
	}
	s.parties.mu.Lock()
	pt.Members = append(pt.Members, member.ObjectID)
	member.PartyID = pt.ID
	s.parties.mu.Unlock()

	members := s.partyMembers(pt)
	if lc := s.clientOf(leader.ObjectID); lc != nil {
		if len(pt.Members) == 2 {
			lc.Send(PartySmallWindowAll(leader, members, pt.LeaderID, pt.Loot))
		} else {
			lc.Send(PartySmallWindowAdd(member, pt.LeaderID, pt.Loot))
		}
	}
	if mc := s.clientOf(member.ObjectID); mc != nil {
		mc.Send(PartySmallWindowAll(member, members, pt.LeaderID, pt.Loot))
	}
	s.broadcastParty(pt, PartySmallWindowAdd(member, pt.LeaderID, pt.Loot), member.ObjectID)
}

func (s *Server) onWithdrawParty(c *GameClient) {
	p := c.Player()
	pt := s.partyOf(p)
	if pt == nil {
		return
	}
	s.removeFromParty(pt, p, true)
}

func (s *Server) onOustPartyMember(c *GameClient, r *packet.Reader) {
	p := c.Player()
	name := r.ReadS()
	pt := s.partyOf(p)
	if pt == nil || pt.LeaderID != p.ObjectID {
		return
	}
	target := s.world.GetPlayerByName(name)
	if target == nil || target.PartyID != pt.ID {
		return
	}
	s.removeFromParty(pt, target, false)
}

func (s *Server) onChangePartyLeader(c *GameClient, r *packet.Reader) {
	p := c.Player()
	name := r.ReadS()
	pt := s.partyOf(p)
	if pt == nil || pt.LeaderID != p.ObjectID {
		c.Send(SystemMessage(SMOnlyLeaderCanTransferRights))
		return
	}
	target := s.world.GetPlayerByName(name)
	if target == nil || target.PartyID != pt.ID || target.ObjectID == p.ObjectID {
		c.Send(SystemMessage(SMTargetIncorrect))
		return
	}
	s.parties.mu.Lock()
	pt.LeaderID = target.ObjectID
	s.parties.mu.Unlock()
	members := s.partyMembers(pt)
	for _, m := range members {
		if cl := s.clientOf(m.ObjectID); cl != nil {
			cl.Send(PartySmallWindowDeleteAll())
			cl.Send(PartySmallWindowAll(m, members, pt.LeaderID, pt.Loot))
			cl.Send(SystemMessage(SMPartyLeaderS1, SysText(target.Name)))
		}
	}
	c.logChange("party leader -> %s", target.Name)
}

func (s *Server) removeFromParty(pt *Party, member *Character, left bool) {
	s.parties.mu.Lock()
	kept := pt.Members[:0]
	for _, id := range pt.Members {
		if id != member.ObjectID {
			kept = append(kept, id)
		}
	}
	pt.Members = append([]int32(nil), kept...)
	member.PartyID = 0
	disband := len(pt.Members) < 2
	if !disband && pt.LeaderID == member.ObjectID {
		pt.LeaderID = pt.Members[0]
	}
	s.parties.mu.Unlock()

	if c := s.clientOf(member.ObjectID); c != nil {
		c.Send(PartySmallWindowDeleteAll())
		if left {
			c.Send(SystemMessage(SMYouLeftParty))
		}
	}
	s.broadcastParty(pt, PartySmallWindowDelete(member), member.ObjectID)
	if left {
		s.broadcastParty(pt, SystemMessage(SMLeftParty, SysText(member.Name)), 0)
	}
	if disband {
		s.disbandParty(pt)
	}
}

func (s *Server) disbandParty(pt *Party) {
	for _, m := range s.partyMembers(pt) {
		m.PartyID = 0
		if c := s.clientOf(m.ObjectID); c != nil {
			c.Send(PartySmallWindowDeleteAll())
		}
	}
	s.parties.mu.Lock()
	delete(s.parties.parties, pt.ID)
	s.parties.mu.Unlock()
}
