# Java → Go coverage

Reference trees: `reference/l2-unity-loginserver` (75 `.java`),
`reference/l2-unity-gameserver` (2301 `.java`, ~339k lines).
Go implementation: 58 non-test files, ~12k lines.

The **wire API is the contract**: opcodes, packet layouts, Blowfish/RSA/GameCrypt,
session keys, ports. Gameplay logic is ported from the Java sources wherever the
data it needs is available; the XML/geodata datapack is **not vendored**
(`reference/l2-unity-gameserver/gameserver/data/README.md`), which is what limits
the remaining systems.

Packet layouts are checked by `internal/gameserver/layout_test.go`, which walks
every server packet with the field sequence transcribed from its Java class.
Formulas are checked against the Java values in `internal/gameserver/stats_test.go`.
The client opcode table was compared against Java `network/GamePacketHandler.java`,
and the leftover startup checklist against the `GameServer.java` constructor.

## Login server — complete

Matches Java `LoginClientThread` + `GameServerListener` / `GameServerThread`:

| Area | Status |
|------|--------|
| Init / Ping / AuthRequest / ServerList / PlayOk | yes |
| Autocreate, inactive, banned, already-on-LS/GS, kick | yes |
| ShowLicense skip of login-pair check | yes (Java quirk kept) |
| GS BlowFishKey RSA-512 NoPadding, AuthRequest, AuthResponse | yes |
| PlayerInGame / Logout / PlayerAuth / Kick / ServerStatus / ReplyCharacters | yes |
| Idle disconnect on `server.connection.timeout.ms` | yes |
| Bad checksum keeps the connection (Java behaviour) | yes |
| `logger.print.received/sent-packets` | yes |
| Shutdown disconnects clients and gameservers | yes |
| Accounts + gameservers in PostgreSQL | yes |

Password handling is the Java behaviour: the client sends hash bytes, the server
stores `Base64(bytes)` and compares strings
(`ClientPacketHandler.java` → `internal/loginserver/packets.go`).

`GGAuth`, `ChangeAccessLevel` and `RequestTempBan` are unimplemented in Java too.
Java login has **no SQL seed rows**; `gameservers` is filled when a GS registers.

## Game server — protocol

Client opcodes follow Java `network/GamePacketHandler`: protocol version, auth,
character create/delete/select/restore, enter world, movement (`0x01` legacy and
`0x02` Unity), action and attack, target select and cancel, item list and reorder,
equip/unequip/drop/use, social actions (`0x1B`), move type (`0x1C`), skill use
(`0x2F`), skill list, action use (`0x45`: sit/stand, walk/run), restart (`0x46`),
validate position, skill learning (`0x6B`/`0x6C`), respawn (`0x6D`) and the Java
`DummyPacket` no-ops.

Server packets implemented with the Java field order: `VersionCheck`, `CharInfo`,
`UserInfo`, `NpcInfo`, `CharSelectInfo`, `CharSelected`, `CharCreate*`,
`CharDelete*`, `ItemList`, `SkillList`, `ShortCutInit`, `ShortCutRegister`,
`ActionFailed`, `ServerClose`, `LeaveWorld`, `MoveToLocation`, `MoveDirection`,
`StopMove`, `ValidateLocation`, `ChangeMoveType`, `ChangeWaitType`, `Attack`,
`Die`, `Revive`, `StatusUpdate`, `TargetSelected`, `MyTargetSelected`,
`DeleteObject`, `TeleportToLocation`, `SocialAction`, `CreatureSay`,
`SystemMessage` (with typed parameters), `MagicSkillUse`, `MagicSkillLaunched`,
`MagicSkillCanceled`, `SetupGauge`, `AcquireSkillList/Info/Done`,
`RestartResponse`, `AuthLoginFail`.

The GS↔LS bridge sends the full Java `ServerStatus` block (status, clock,
brackets, age limit, test server, PvP server, max players).

## Game server — gameplay

| System | Java source | Status |
|--------|-------------|--------|
| Stat engine (HP/MP/CP, regen, PAtk/PDef/MAtk/MDef, accuracy, evasion, crit, attack/cast speed, run speed, weight limit, collision) | `Formulas`, `CreatureStatus`, `PlayerStatus`, `skills/funcs` | ported over the class XML tables |
| Melee combat (hit/miss, critical, position and damage formulas, attack pacing, CP before HP) | `Formulas`, `model/actor/attack` | ported |
| Exp/SP rewards, level difference decay, level up, auto-granted skills | `Monster.calculateExpAndSp`, `Player.addExpAndSp` | ported |
| Death, respawn to the nearest town, revive restore | `Player.doDie`, `RequestRestartPoint` | ported (town list stands in for `RestartPointData`) |
| Skill casting (MP cost, reuse, cast time, target and range checks) | `CreatureCast`, `L2Skill` | ported |
| Skill effects and stat modifiers from the `<for>` blocks, stack type/order rules | `AbstractEffect`, `EffectList`, `skills/basefuncs` | ported for heal, mana heal, physical, magical and buff skills |
| Monster AI (aggro range, chase, retaliation, chase abandon) | `model/actor/ai/type/AttackableAI` | ported |
| NPC respawn | `Spawn.doRespawn` | ported (fixed delay; per-spawn delays live in the XML) |
| Inventory (equip/unequip, two handed, drop, weight) | `Inventory`, `PcInventory` | ported for the items the server can hand out |
| Skill learning from the class tree | `PlayerData`, `RequestAcquireSkill` | ported |
| Task managers (HP/MP/CP regen, effect expiry, PvP flag, combat stance, aggro scan) | `taskmanager/` | ported |
| SkillTable + class skill trees | `SkillTable`, `PlayerData` | ported from `data/xml` |
| ItemData XML | `ItemData`, `DocumentItem` | ported (`data/xml/items`) |
| NpcData XML | `NpcData` | ported (`data/xml/npcs`); applied on spawn |
| Buy lists + buy/sell | `BuyListManager`, `RequestBuyItem`, `RequestSellItem` | ported |
| Gatekeeper teleports | `TeleportData`, `Npc.onBypassFeedback` | ported (`goto` bypass) |
| Restart points | `RestartPointData` | ported (town / race areas; no castle/clan hall) |
| NPC talk HTML | `NpcHtmlMessage`, `RequestBypassToServer` | generated merchant/gatekeeper windows |
| Party invite / join / leave / kick | `RequestJoinParty` and `PartySmallWindow*` | ported |
| Monster drops | `DropCategory` / `DropData` | ported (no spoil, items go to inventory) |

## Game server — still not ported

These need the remaining datapack (geodata, quests, HTML, spawnlist) or the
subsystems built on top of it:

- Remaining XML loaders Java starts in `GameServer.java`: `SkillTreeData`,
  `HennaData`, `MultisellData`, `RecipeData`, `ArmorSetData`, `FishData`,
  `SpellbookData`, `SoulCrystalData`, `AugmentationData`, `SummonItemData`,
  `DoorData`, `NewbieBuffData`, `StaticObjectData`, `WalkerRouteData`,
  `ObserverGroupData`, `AnnouncementData`, `AdminData`, `ScriptData`, `BoatData`
- `GeoEngine`, pathfinding, zones, doors, `StaticObjectData`
- Quests and `ScriptData` (343 quest files plus the monster AI script tree)
- Multisell, recipes, warehouse, trade, private stores
- Clans, allies, wars, crests, command channel
- Castle siege, clan hall, manor, Seven Signs, Festival, Olympiad, heroes
- Community board, petitions, admin commands
- Boats, fishing, cursed weapons, augmentation, cubics, henna, macros, subclasses
- Debuff and status effects on NPCs (monsters have no effect list yet)
- Handler registries (`handler/` item/chat/user-command/target handlers);
  Go still inlines most of this in `handler.go` / `handler_ingame.go`
- `ThreadPool`, `HtmCache`, `CrestCache`, `GlobalMemo`, `PlayerInfoTable`,
  server-crash tracking in `server_memo`

`internal/config/game.go` covers the settings this port needs; Java `Config`
exposes several hundred more (`RATE_*`, `ENCHANT_*`, `OLY_*`, `SEVEN_SIGNS_*`,
`GEODATA_PATH`, `INVENTORY_*`, `AUTO_LOOT`, …). About 30 Java tables
(community board, auctions, manor, petitions, olympiad fights, cursed
weapons, …) are still absent from the Go schema.

Tables for the unported systems exist in `sql/001_init.sql` so a later datapack
load can persist state, but there is no runtime manager behind them.

## Seed data on a blank start

Java SQL `INSERT`s exist only in five files. Almost all world content is XML
under `data/xml/` (skills, classes, items, NPCs, buy lists, teleports and
restart points are vendored here; geodata and HTML are not).

| Java source | Go seed | Applied on empty DB |
|-------------|---------|---------------------|
| `castle.sql` (9 castles) | `sql/002_seed.sql` | yes |
| `clanhall.sql` (ids 21–64) | `sql/002_seed.sql` | yes |
| `seven_signs_status.sql` | `sql/002_seed.sql` | yes |
| `seven_signs_festival.sql` | `sql/002_seed.sql` | yes |
| `mdt_bets.sql` | `sql/002_seed.sql` | yes |
| `gameservers.sql` (schema only) | none (GS self-registers) | n/a |
| PlayerLevelData XML | `player_levels` 1–81 | yes |
| PlayerData class templates | `class_templates` + `data/xml/classes` + `class_skills` | yes (stats, HP/MP/CP and regen tables parsed at start) |
| SkillTable XML | `data/xml/skills` → `skill_templates` | yes (effects and funcs parsed too) |
| ItemData XML | `data/xml/items` | yes (parsed at start; used for weight, price, equip stats) |
| NpcData XML | `data/xml/npcs` | yes (parsed at start; applied to live NPCs) |
| BuyListManager XML | `data/xml/buyLists.xml` | yes |
| TeleportData / RestartPointData | `data/xml/teleports.xml`, `restartPointAreas.xml` | yes |
| SpawnManager XML | newbie `npc_templates` / `npc_spawns` | yes (Talking Island + race gates + starter mobs; full spawnlist is disabled upstream) |

How a zero-state server gets this data:

1. `docker compose up` runs `sql/001_init.sql` then `sql/002_seed.sql`.
2. Both `cmd/loginserver` and `cmd/gameserver` call `db.ApplySchemaAndSeed` on
   connect, so a DB created without compose still gets schema + seed.
3. If PostgreSQL is down, the process uses in-memory stores and the gameserver
   still loads `DefaultNewbieSpawns`.
4. Character create grants the class starter items and **autoGet skills**
   (`cost="0"` in the class XML, same as Java `getAvailableAutoGetSkills`).
   Learn-from-NPC skills are offered through `AcquireSkillList` once the player
   reaches their `minLvl` and can pay the SP cost.
