# Java datapack

Vendored from https://github.com/shnok/l2-unity-gameserver `gameserver/data/xml`.
Geodata, HTML dialogs, the disabled spawnlist and the Java quest script tree
are still not copied.

The gameserver parses every vendored XML file on start. Character create grants
**autoGet** skills only (`cost="0"` and `minLvl` ≤ level), same as Java, and
uses the first `<spawns>` point from the class XML.

| Path | Java loader | Rows |
|------|-------------|------|
| `xml/skills/*.xml` | `SkillTable` | ~2650 skills / ~15k levels |
| `xml/classes/*.xml` | `PlayerData` | 89 class trees, skill nodes, newbie items, create spawns |
| `xml/items/*.xml` | `ItemData` | weapons, armour, etc (weight, price, body part, `<for>` stats) |
| `xml/npcs/*.xml` | `NpcData` | templates, combat stats, drop lists |
| `xml/buyLists.xml` | `BuyListManager` | merchant product lists |
| `xml/teleports.xml` | `TeleportData` | gatekeeper destinations |
| `xml/restartPointAreas.xml` | `RestartPointData` | town restart points and race areas |
| `xml/playerLevels.xml` | `PlayerLevelData` | exp / karma / death-penalty table |

Live systems that already consult the extra tables:

- armor set bonuses in `RecalcStats`
- spellbooks on skill learn
- heal SPS correction
- newbie guide `SupportMagic`
- instant teleports
- peace-zone lookup
- XML spawnlist NPCs next to the hardcoded Talking Island set
- `is_dropable` / `crystal_type` / `soulshots` on items
- EtcItem `item_skill` entries (potions, Scroll of Escape) used by `UseItem`
- Warehouse keepers (`WarehouseKeeper`) open deposit/withdraw windows
