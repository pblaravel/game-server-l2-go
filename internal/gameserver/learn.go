package gameserver

import "sort"

// Skill learning from the class tree, matching Java PlayerData skill nodes and
// Player.getAvailableSkills(): a node is learnable when the player reached its
// minLvl and already knows the previous level.

func knownSkillLevels(p *Character) map[int32]int32 {
	known := make(map[int32]int32, len(p.Skills))
	for _, s := range p.Skills {
		if s.Level > known[s.ID] {
			known[s.ID] = s.Level
		}
	}
	return known
}

// LearnableSkills is Java Player.getAvailableSkills for AcquireSkillType.USUAL.
func LearnableSkills(p *Character) []ClassSkillNode {
	known := knownSkillLevels(p)
	best := map[int32]ClassSkillNode{}
	for _, n := range ClassSkills(p.ClassID) {
		if n.MinLvl > p.Level || n.Cost <= 0 {
			continue
		}
		if n.Level != known[n.ID]+1 {
			continue
		}
		if cur, ok := best[n.ID]; !ok || n.Level < cur.Level {
			best[n.ID] = n
		}
	}
	out := make([]ClassSkillNode, 0, len(best))
	for _, n := range best {
		out = append(out, n)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ID != out[j].ID {
			return out[i].ID < out[j].ID
		}
		return out[i].Level < out[j].Level
	})
	return out
}

func findLearnableNode(p *Character, id, level int32) (ClassSkillNode, bool) {
	for _, n := range LearnableSkills(p) {
		if n.ID == id && n.Level == level {
			return n, true
		}
	}
	return ClassSkillNode{}, false
}

// AddOrUpgradeSkill is Java Player.addSkill: a known skill is replaced by the
// higher level instead of being duplicated.
func AddOrUpgradeSkill(p *Character, node ClassSkillNode) {
	passive := false
	if tpl := GetSkill(node.ID, node.Level); tpl != nil {
		passive = tpl.IsPassive()
	}
	for i := range p.Skills {
		if p.Skills[i].ID == node.ID {
			if p.Skills[i].Level < node.Level {
				p.Skills[i].Level = node.Level
				p.Skills[i].Passive = passive
			}
			return
		}
	}
	p.Skills = append(p.Skills, Skill{ID: node.ID, Level: node.Level, Passive: passive})
}

// AutoLearnOnLevelUp grants the cost-free skills of the class tree, like Java
// Player.giveAvailableAutoGetSkills after a level change.
func AutoLearnOnLevelUp(p *Character) []Skill {
	known := knownSkillLevels(p)
	var added []Skill
	for _, n := range AutoGetSkills(p.ClassID, p.Level) {
		if known[n.ID] >= n.Level {
			continue
		}
		AddOrUpgradeSkill(p, n)
		added = append(added, Skill{ID: n.ID, Level: n.Level})
	}
	return added
}
