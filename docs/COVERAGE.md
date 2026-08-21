# Java → Go coverage

Reference trees: `reference/l2-unity-loginserver` (75 `.java`),
`reference/l2-unity-gameserver` (2301 `.java`, ~339k lines).
Go implementation: 63 non-test files.

The **wire API is the contract**: opcodes, packet layouts, Blowfish/RSA/GameCrypt,
session keys, ports. Gameplay logic is ported from the Java sources wherever the
data it needs is available. Skills, classes, items, NPCs, shops, teleports,
restart points and the player exp table are vendored under `data/xml/`. The rest
of the Java datapack (geodata, HTML, quests, spawnlist, zones, doors, …) is not.

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
| Death, respawn to the nearest town, revive restore | `Player.doDie`, `RequestRestartPoint` | ported from `RestartPointData` |
| Player exp / karma / death-penalty table | `PlayerLevelData` | ported from `playerLevels.xml` |
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
| NPC talk HTML | `NpcHtmlMessage`, `RequestBypassToServer` | generated merchant/gatekeeper/warehouse windows |
| Party invite / join / leave / kick | `RequestJoinParty` and `PartySmallWindow*` | ported |
| Monster drops | `DropCategory` / `DropData` | ported (no spoil, items go to inventory) |
| Player trade | `TradeRequest`, `TradeList` | ported (no private store) |
| Private warehouse | `PcWarehouse`, `WarehouseKeeper` | ported (no clan/freight) |
| Shortcuts register/delete | `RequestShortCutReg` / `RequestShortCutDel` | ported |
| Destroy item | `RequestDestroyItem` | ported |
| Item skills (potions, SOE) | `UseItem` + `item_skill` | ported (`RECALL` and HOT/buff skills) |
| Friends | `RequestFriendInvite` and `FriendList` | ported (in-memory + SQL) |

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
- Multisell, recipes, private stores, clan warehouse / freight
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

## Java XML loaders vs Go

Java `GameServer` constructor starts these XML sources. Copied = file exists
under `data/xml/`. Parsed = `LoadDatapack` reads it. Used = a live handler
consults the loaded table.

| Java loader | Path | Copied | Parsed | Used |
|-------------|------|--------|--------|------|
| `SkillTable` | `xml/skills/` | yes | yes | combat / cast / learn |
| `PlayerData` | `xml/classes/` | yes | yes | kits, stats, skill tree, create spawn |
| `ItemData` | `xml/items/` | yes | partial | weight, price, body part, `<for>` PAtk/PDef, tradable/destroyable, `item_skill` |
| `NpcData` | `xml/npcs/` | yes | partial | combat stats, drops, type; NPC `<skills>` / AI ignored |
| `BuyListManager` | `xml/buyLists.xml` | yes | yes | buy / sell |
| `TeleportData` | `xml/teleports.xml` | yes | yes | gatekeeper `goto` |
| `RestartPointData` | `xml/restartPointAreas.xml` | yes | yes | death / restart (no castle / clan hall) |
| `PlayerLevelData` | `xml/playerLevels.xml` | yes | yes | exp table, karma / death-loss rows stored |
| `SkillTreeData` | `xml/skillstrees/` | no | — | fishing / clan / enchant trees only |
| `InstantTeleportData` | `xml/instantTeleports.xml` | no | — | |
| `HealSpsData` | `xml/healSps.xml` | no | — | |
| `NewbieBuffData` | `xml/newbieBuffs.xml` | no | — | |
| `HennaData` | `xml/hennas.xml` | no | — | |
| `MultisellData` | `xml/multisell/` | no | — | |
| `RecipeData` | `xml/recipes.xml` | no | — | |
| `ArmorSetData` | `xml/armorSets.xml` | no | — | |
| `SpellbookData` | `xml/spellbooks.xml` | no | — | |
| `SummonItemData` | `xml/summonItems.xml` | no | — | |
| `SoulCrystalData` | `xml/soulCrystals.xml` | no | — | |
| `AugmentationData` | `xml/augmentation/` | no | — | |
| `FishData` | `xml/fish.xml` | no | — | |
| `CursedWeaponManager` | `xml/cursedWeapons.xml` | no | — | |
| `DoorData` | `xml/doors.xml` | no | — | |
| `StaticObjectData` | `xml/staticObjects.xml` | no | — | |
| `WalkerRouteData` | `xml/walkerRoutes.xml` | no | — | |
| `SpawnManager` | `xml/spawnlist/` | no | — | upstream also has `spawnlist_disabled/` |
| `ZoneManager` | `xml/zones/` | no | — | |
| `AnnouncementData` | `xml/announcements.xml` | no | — | |
| `AdminData` | `xml/accessLevels.xml`, `adminCommands.xml` | no | — | |
| `BufferManager` | `xml/bufferSkills.xml` | no | — | |
| `CastleManager` | `xml/castles.xml` | no | — | SQL seed has castle rows only |
| `ClanHallManager` | `xml/clanHalls.xml`, `clanHallDeco.xml` | no | — | SQL seed has hall rows only |
| `ManorAreaData` / `CastleManorManager` | `xml/manorAreas.xml`, `manors.xml` | no | — | |
| `ObserverGroupData` | `xml/observerGroups.xml` | no | — | |
| `BoatData` | `xml/boatRoutes.xml` | no | — | |
| `ScriptData` | `xml/scripts.xml` + quest tree | no | — | |
| `HtmCache` | `data/html/` | no | — | |
| `GeoEngine` | `data/geodata/` | no | — | |

Item XML still skips `crystal_type`, `soulshots`, `is_dropable`, `weapon_type`,
`armor_type`, `etcitem_type`. Npc XML still skips `<skills>` and AI fields.

## Seed data on a blank start

Java SQL `INSERT`s exist only in five files. Almost all world content is XML.

| Java source | Go seed | Applied on empty DB |
|-------------|---------|---------------------|
| `castle.sql` (9 castles) | `sql/002_seed.sql` | yes |
| `clanhall.sql` (ids 21–64) | `sql/002_seed.sql` | yes |
| `seven_signs_status.sql` | `sql/002_seed.sql` | yes |
| `seven_signs_festival.sql` | `sql/002_seed.sql` | yes |
| `mdt_bets.sql` | `sql/002_seed.sql` | yes |
| `gameservers.sql` (schema only) | none (GS self-registers) | n/a |
| PlayerLevelData XML | `player_levels` 1–81 + `data/xml/playerLevels.xml` | yes (runtime table is the XML) |
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
