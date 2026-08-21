package gameserver

import (
	"strings"
	"sync"
	"time"

	"github.com/pblaravel/game-server-l2-go/internal/packet"
)

const friendInviteTTL = 15 * time.Second

type friendInvite struct {
	from int32
	at   time.Time
}

type friendState struct {
	mu      sync.Mutex
	invites map[int32]friendInvite
}

func newFriendState() *friendState {
	return &friendState{invites: map[int32]friendInvite{}}
}

func hasFriend(p *Character, id int32) bool {
	for _, f := range p.Friends {
		if f.ObjectID == id {
			return true
		}
	}
	return false
}

func addFriend(p *Character, other *Character) {
	if hasFriend(p, other.ObjectID) {
		return
	}
	p.Friends = append(p.Friends, Friend{ObjectID: other.ObjectID, Name: other.Name})
}

func removeFriend(p *Character, id int32) bool {
	for i, f := range p.Friends {
		if f.ObjectID == id {
			p.Friends = append(p.Friends[:i], p.Friends[i+1:]...)
			return true
		}
	}
	return false
}

func friendByName(p *Character, name string) *Friend {
	for i := range p.Friends {
		if strings.EqualFold(p.Friends[i].Name, name) {
			return &p.Friends[i]
		}
	}
	return nil
}

func (s *Server) friendOnline(friends []Friend) map[int32]bool {
	out := map[int32]bool{}
	for _, f := range friends {
		if p := s.world.GetPlayer(f.ObjectID); p != nil && p.Online {
			out[f.ObjectID] = true
		}
	}
	return out
}

func (s *Server) onFriendInvite(c *GameClient, r *packet.Reader) {
	p := c.Player()
	name := r.ReadS()
	target := s.world.GetPlayerByName(name)
	if target == nil {
		c.Send(SystemMessage(SMTargetNotFound))
		c.Send(FriendAddRequestResult(false))
		return
	}
	if target.ObjectID == p.ObjectID {
		c.Send(SystemMessage(SMCannotAddYourself))
		c.Send(FriendAddRequestResult(false))
		return
	}
	if hasFriend(p, target.ObjectID) {
		c.Send(SystemMessage(SMAddedToFriends, SysText(target.Name)))
		c.Send(FriendAddRequestResult(false))
		return
	}
	tc := s.clientOf(target.ObjectID)
	if tc == nil {
		c.Send(SystemMessage(SMTargetNotFound))
		c.Send(FriendAddRequestResult(false))
		return
	}
	s.friends.mu.Lock()
	if inv, ok := s.friends.invites[target.ObjectID]; ok && time.Since(inv.at) < friendInviteTTL {
		s.friends.mu.Unlock()
		c.Send(SystemMessage(SMWaitingForReply))
		c.Send(FriendAddRequestResult(false))
		return
	}
	s.friends.invites[target.ObjectID] = friendInvite{from: p.ObjectID, at: time.Now()}
	s.friends.mu.Unlock()
	tc.Send(FriendAddRequest(p.Name))
}

func (s *Server) onAnswerFriendInvite(c *GameClient, r *packet.Reader) {
	p := c.Player()
	resp := r.ReadD()
	s.friends.mu.Lock()
	inv, ok := s.friends.invites[p.ObjectID]
	delete(s.friends.invites, p.ObjectID)
	s.friends.mu.Unlock()
	if !ok || time.Since(inv.at) > friendInviteTTL {
		return
	}
	from := s.clientOf(inv.from)
	if from == nil {
		return
	}
	if resp != 1 {
		from.Send(FriendAddRequestResult(false))
		return
	}
	fp := from.Player()
	if fp == nil {
		return
	}
	addFriend(fp, p)
	addFriend(p, fp)
	from.Send(FriendAddRequestResult(true))
	from.Send(SystemMessage(SMAddedToFriends, SysText(p.Name)))
	from.Send(L2Friend(1, p.Name, p.ObjectID, true))
	c.Send(FriendAddRequestResult(true))
	c.Send(SystemMessage(SMJoinedAsFriend, SysText(fp.Name)))
	c.Send(L2Friend(1, fp.Name, fp.ObjectID, true))
	_ = s.store.Update(from.ctx(), fp)
	_ = s.store.Update(c.ctx(), p)
}

func (s *Server) onFriendList(c *GameClient) {
	p := c.Player()
	c.Send(FriendList(p.Friends, s.friendOnline(p.Friends)))
}

func (s *Server) onFriendDel(c *GameClient, r *packet.Reader) {
	p := c.Player()
	name := r.ReadS()
	f := friendByName(p, name)
	if f == nil {
		c.Send(SystemMessage(SMUserNotInFriendsList))
		return
	}
	id, fname := f.ObjectID, f.Name
	removeFriend(p, id)
	c.Send(L2Friend(3, fname, id, false))
	if other := s.world.GetPlayer(id); other != nil {
		removeFriend(other, p.ObjectID)
		if oc := s.clientOf(id); oc != nil {
			oc.Send(L2Friend(3, p.Name, p.ObjectID, false))
			_ = s.store.Update(oc.ctx(), other)
		}
	}
	_ = s.store.Update(c.ctx(), p)
}
