# Java datapack (subset)

Vendored from https://github.com/shnok/l2-unity-gameserver `gameserver/data/xml`.

| Path | Java loader | Rows |
|------|-------------|------|
| `xml/skills/*.xml` | `SkillTable` | ~2650 skills / ~15k levels |
| `xml/classes/*.xml` | `PlayerData` | 89 class trees, skill nodes + newbie items |

The gameserver parses these on start and upserts `skill_templates` + `class_skills`.
Character create grants **autoGet** skills only (`cost="0"` and `minLvl` ≤ level), same as Java.
