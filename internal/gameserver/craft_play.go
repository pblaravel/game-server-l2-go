package gameserver

import (
	"github.com/pblaravel/game-server-l2-go/internal/packet"
)

// Java L2Skill.SKILL_CREATE_COMMON / SKILL_CREATE_DWARVEN.
const (
	skillCreateCommon  int32 = 1320
	skillCreateDwarven int32 = 172
	recipeBookLimit    int32 = 50
)

// recipeRoll is Java Rnd.get(100); tests replace it.
var recipeRoll = func() int { return int(rndDouble() * 100) }

func SkillLevelOf(p *Character, id int32) int32 {
	for _, sk := range p.Skills {
		if sk.ID == id {
			return sk.Level
		}
	}
	return 0
}

func HasRecipe(p *Character, id int32) bool {
	for _, r := range p.Recipes {
		if r == id {
			return true
		}
	}
	return false
}

func recipesOf(p *Character, dwarven bool) []int32 {
	out := make([]int32, 0)
	for _, id := range p.Recipes {
		r := GetRecipe(id)
		if r != nil && r.IsDwarven == dwarven {
			out = append(out, id)
		}
	}
	return out
}

func RecipeBookItemList(p *Character, dwarven bool) []byte {
	list := recipesOf(p, dwarven)
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0xD6)
		if dwarven {
			w.WriteD(0)
		} else {
			w.WriteD(1)
		}
		w.WriteD(p.MaxMP)
		w.WriteD(int32(len(list)))
		for i, id := range list {
			w.WriteD(id)
			w.WriteD(int32(i + 1))
		}
	})
}

func RecipeItemMakeInfo(id int32, p *Character, status int32) []byte {
	r := GetRecipe(id)
	if r == nil {
		return nil
	}
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0xD7)
		w.WriteD(id)
		if r.IsDwarven {
			w.WriteD(0)
		} else {
			w.WriteD(1)
		}
		w.WriteD(int32(p.CurMP))
		w.WriteD(p.MaxMP)
		w.WriteD(status)
	})
}

func (s *Server) learnRecipe(c *GameClient, item *Item) {
	p := c.Player()
	rec := GetRecipeByItem(item.ItemID)
	if rec == nil {
		c.Send(ActionFailed())
		return
	}
	if HasRecipe(p, rec.ID) {
		c.Send(SystemMessage(SMRecipeAlreadyRegistered))
		return
	}
	skillID := skillCreateCommon
	if rec.IsDwarven {
		skillID = skillCreateDwarven
	}
	lvl := SkillLevelOf(p, skillID)
	if lvl <= 0 {
		c.Send(SystemMessage(SMCantRegisterNoAbility))
		return
	}
	if rec.Level > lvl {
		c.Send(SystemMessage(SMCreateLvlTooLow))
		return
	}
	if int32(len(recipesOf(p, rec.IsDwarven))) >= recipeBookLimit {
		c.Send(ActionFailed())
		return
	}
	if !RemoveItemCount(p, item.ObjectID, 1) {
		c.Send(ActionFailed())
		return
	}
	p.Recipes = append(p.Recipes, rec.ID)
	c.Send(SystemMessage(SMS1Added, SysItem(item.ItemID)))
	c.Send(RecipeBookItemList(p, rec.IsDwarven))
	c.Send(ItemList(p.Items, false))
	_ = s.store.Update(c.ctx(), p)
	c.logChange("learned recipe=%d item=%d", rec.ID, item.ItemID)
}

func (s *Server) onRecipeBookOpen(c *GameClient, r *packet.Reader) {
	dwarven := r.ReadD() == 0
	c.Send(RecipeBookItemList(c.Player(), dwarven))
}

func (s *Server) onRecipeItemMakeInfo(c *GameClient, r *packet.Reader) {
	id := r.ReadD()
	if pkt := RecipeItemMakeInfo(id, c.Player(), -1); pkt != nil {
		c.Send(pkt)
	}
}

func (s *Server) onRecipeItemMakeSelf(c *GameClient, r *packet.Reader) {
	s.craftRecipe(c, r.ReadD())
}

func (s *Server) onRecipeBookDestroy(c *GameClient, r *packet.Reader) {
	p := c.Player()
	id := r.ReadD()
	if p.PrivateStore != StoreNone {
		c.Send(SystemMessage(SMCantAlterRecipebook))
		return
	}
	rec := GetRecipe(id)
	if rec == nil || !HasRecipe(p, id) {
		return
	}
	kept := p.Recipes[:0]
	for _, rid := range p.Recipes {
		if rid != id {
			kept = append(kept, rid)
		}
	}
	p.Recipes = append([]int32(nil), kept...)
	c.Send(SystemMessage(SMS1HasBeenDeleted, SysItem(rec.ItemID)))
	c.Send(RecipeBookItemList(p, rec.IsDwarven))
	_ = s.store.Update(c.ctx(), p)
	c.logChange("forgot recipe=%d", id)
}

func (s *Server) craftRecipe(c *GameClient, recipeID int32) {
	p := c.Player()
	rec := GetRecipe(recipeID)
	if rec == nil || !HasRecipe(p, recipeID) {
		c.Send(ActionFailed())
		return
	}
	skillID := skillCreateCommon
	if rec.IsDwarven {
		skillID = skillCreateDwarven
	}
	if rec.Level > SkillLevelOf(p, skillID) {
		c.Send(ActionFailed())
		return
	}
	if p.CurMP < float64(rec.MPConsume) {
		c.Send(SystemMessage(SMNotEnoughMP))
		c.Send(ActionFailed())
		return
	}
	for _, mat := range rec.Materials {
		if ItemCountOf(p, mat.ItemID) < mat.Count {
			c.Send(SystemMessage(SMNotEnoughItems))
			c.Send(ActionFailed())
			return
		}
	}
	weight := int64(ItemWeight(rec.ProductID)) * int64(rec.ProductCnt)
	if p.CurrentWeight+int32(weight) > p.WeightLimit {
		c.Send(SystemMessage(SMWeightLimitExceeded))
		return
	}
	p.CurMP -= float64(rec.MPConsume)
	for _, mat := range rec.Materials {
		if !RemoveItemByID(p, mat.ItemID, mat.Count) {
			c.Send(ActionFailed())
			return
		}
	}
	success := recipeRoll() < int(rec.SuccessRate)
	if success {
		AddItem(p, rec.ProductID, rec.ProductCnt, s.nextItemID)
		if rec.ProductCnt > 1 {
			c.Send(SystemMessage(SMEarnedS2S1S, SysItem(rec.ProductID), SysNumber(rec.ProductCnt)))
		} else {
			c.Send(SystemMessage(SMEarnedItemS1, SysItem(rec.ProductID)))
		}
	} else {
		c.Send(SystemMessage(SMItemMixingFailed))
	}
	status := int32(0)
	if success {
		status = 1
	}
	if pkt := RecipeItemMakeInfo(rec.ID, p, status); pkt != nil {
		c.Send(pkt)
	}
	c.Send(ItemList(p.Items, false))
	s.sendWeightAndStats(c)
	_ = s.store.Update(c.ctx(), p)
	c.logChange("craft recipe=%d success=%v product=%d", rec.ID, success, rec.ProductID)
}
