package gameserver

import "github.com/pblaravel/game-server-l2-go/internal/packet"

// Interlude C→S packets the Unity client actually sends (see clientapi contract).

func (s *Server) onShowBoard(c *GameClient) {
	c.Send(ShowBoard("<html><body>Community Board</body></html>"))
}

func (s *Server) onSkillCoolTime(c *GameClient) {
	c.Send(SkillCoolTime())
}

func (s *Server) onQuestAbort(c *GameClient, r *packet.Reader) {
	_ = r.ReadD()
	c.Send(QuestList())
}

func (s *Server) onPledgeInfo(c *GameClient, r *packet.Reader) {
	id := r.ReadD()
	name, ally := "", ""
	if p := c.Player(); p != nil && p.ClanID == id && id != 0 {
		name = "Clan"
	}
	c.Send(PledgeInfo(id, name, ally))
}

func (s *Server) onJoinPledge(c *GameClient, r *packet.Reader) {
	_ = r.ReadD()
	c.Send(ActionFailed())
}

func (s *Server) onWithdrawPledge(c *GameClient) {
	if p := c.Player(); p != nil {
		p.ClanID = 0
		c.Send(UserInfo(p))
	}
}

func (s *Server) onOustPledgeMember(c *GameClient, r *packet.Reader) {
	_ = r.ReadS()
	c.Send(ActionFailed())
}

func (s *Server) onGiveNickName(c *GameClient, r *packet.Reader) {
	_ = r.ReadS()
	title := r.ReadS()
	if p := c.Player(); p != nil {
		p.Title = title
		c.Send(UserInfo(p))
	}
}

func (s *Server) onPledgePower(c *GameClient, r *packet.Reader) {
	rank, action, privs := r.ReadD(), r.ReadD(), r.ReadD()
	c.Send(ManagePledgePower(rank, action, privs))
}

func (s *Server) onPackageSendable(c *GameClient, r *packet.Reader) {
	_ = r.ReadD()
	adena := int32(0)
	if p := c.Player(); p != nil {
		adena = AdenaCount(p)
	}
	c.Send(PackageSendableList(adena))
}

func (s *Server) onPackageSend(c *GameClient, r *packet.Reader) {
	_ = r.ReadD()
	c.Send(ActionFailed())
}

func (s *Server) onPreviewItem(c *GameClient, r *packet.Reader) {
	_ = r.ReadD()
	_ = r.ReadD()
	c.Send(ShopPreviewInfo())
}
