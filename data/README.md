# Java datapack (subset)

Vendored from https://github.com/shnok/l2-unity-gameserver `gameserver/data/xml`.
Geodata, HTML dialogs and the disabled spawnlist are still not copied.

| Path | Java loader | Rows |
|------|-------------|------|
| `xml/skills/*.xml` | `SkillTable` | ~2650 skills / ~15k levels |
| `xml/classes/*.xml` | `PlayerData` | 89 class trees, skill nodes + newbie items |
| `xml/items/*.xml` | `ItemData` | weapons, armour, etc (weight, price, body part, `<for>` stats) |
| `xml/npcs/*.xml` | `NpcData` | templates, combat stats, drop lists |
| `xml/buyLists.xml` | `BuyListManager` | merchant product lists |
| `xml/teleports.xml` | `TeleportData` | gatekeeper destinations |
| `xml/restartPointAreas.xml` | `RestartPointData` | town restart points and race areas |
| `xml/playerLevels.xml` | `PlayerLevelData` | reference only; exp table is already seeded |

The gameserver parses these on start. Character create grants **autoGet**
skills only (`cost="0"` and `minLvl` ≤ level), same as Java.

EtcItem `item_skill` entries (potions, Scroll of Escape) are used by `UseItem`.
Warehouse keepers (`WarehouseKeeper`) open deposit/withdraw windows.
