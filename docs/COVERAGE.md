# Java → Go coverage

Reference trees: `reference/l2-unity-loginserver` (75 `.java`),
`reference/l2-unity-gameserver` (2301 `.java`, ~339k lines).
Go implementation: 81 non-test files.

The **wire API is the contract**: opcodes, packet layouts, Blowfish/RSA/GameCrypt,
session keys, ports. Gameplay logic is ported from the Java sources wherever the
data it needs is available. The Java `data/xml` tree is vendored and parsed
except HTML, the disabled spawnlist and the Java quest script sources.
Geodata binaries are not vendored; `GeoEngine` loads `data/geodata/*.l2j` /
`*_conv.dat` when present and uses Null blocks otherwise.

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

`ChangeAccessLevel` and `RequestTempBan` are unimplemented in Java too.
`GGAuth 0x0B` is implemented for the Interlude/Unity login sequence
(`server.client.interlude` / `INTERLUDE_CLIENT`). Java login has **no SQL seed
rows**; `gameservers` is filled when a GS registers.

Unity Interlude (protocol 746) outgoing packets are listed in
`internal/clientapi` and checked by `internal/gameserver/client_contract_test.go`
plus `internal/loginserver/interlude_test.go`.

## Game server — protocol

Client opcodes follow Java `network/GamePacketHandler`: protocol version, auth,
character create/delete/select/restore, enter world, movement (`0x01` legacy and
`0x02` Unity), action and attack, target select and cancel, item list and reorder,
equip/unequip/drop/use, social actions (`0x1B`), move type (`0x1C`), skill use
(`0x2F`), skill list, action use (`0x45`: sit/stand, walk/run, private store
10/28/61), restart (`0x46`), validate position, enchant (`0x58`), skill learning
(`0x6B`/`0x6C`), respawn (`0x6D`), crystallize (`0x72`), private store sell/buy
(`0x73`/`0x74`/`0x76`/`0x77`/`0x79`, `0x90`/`0x91`/`0x93`/`0x94`/`0x96`),
CannotMoveAnymore (`0x36`), quest list (`0x63`), crystallize (`0x72`), private
store sell/buy (`0x73`/`0x74`/`0x76`/`0x77`/`0x79`, `0x90`/`0x91`/`0x93`/`0x94`/`0x96`),
user commands (`0xAA`), recipe book (`0xAC`/`0xAD`/`0xAE`/`0xAF`), henna
(`0xBA`–`0xBF`), mini-map (`0xCD`), auto soulshot / party leader (`0xD0` + short 5/4)
and the Java `DummyPacket` no-ops.

Server packets implemented with the Java field order: `VersionCheck`, `CharInfo`,
`UserInfo`, `NpcInfo`, `CharSelectInfo`, `CharSelected`, `CharCreate*`,
`CharDelete*`, `ItemList`, `SkillList`, `ShortCutInit`, `ShortCutRegister`,
`ActionFailed`, `ServerClose`, `LeaveWorld`, `MoveToLocation`, `MoveDirection`,
`StopMove`, `ValidateLocation`, `ChangeMoveType`, `ChangeWaitType`, `Attack`,
`Die`, `Revive`, `StatusUpdate`, `TargetSelected`, `MyTargetSelected`,
`DeleteObject`, `TeleportToLocation`, `SocialAction`, `CreatureSay`,
`SystemMessage` (with typed parameters), `MagicSkillUse`, `MagicSkillLaunched`,
`MagicSkillCanceled`, `SetupGauge`, `AcquireSkillList/Info/Done`,
`RestartResponse`, `AuthLoginFail`, `DropItem`, `SpawnItem`, `GetItem`,
`MultiSellList`, `EnchantResult`, `ChooseInventoryItem`, `RecipeBookItemList`,
`RecipeItemMakeInfo`, `HennaInfo`, `HennaEquipList`, `HennaItemInfo`,
`HennaUnequipList`, `HennaItemUnequipInfo`, `ExAutoSoulShot`, `QuestList`,
`ShowMiniMap`, `PrivateStoreManageListSell/ListSell/MsgSell`,
`PrivateStoreManageListBuy/ListBuy/MsgBuy`.

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
| Party invite / join / leave / kick / leader | `RequestJoinParty`, `RequestChangePartyLeader` | ported |
| Monster drops | `DropCategory` / `DropData` | ported (no spoil, items go to inventory) |
| Player trade | `TradeRequest`, `TradeList` | ported |
| Soul / spirit / blessed shots | `SoulShots`, `SpiritShots`, `BlessedSpiritShots`, `RequestAutoSoulShot` | ported (consume on hit / magic; no pet or fishing shots) |
| Private stores | `SetPrivateStoreList*`, `RequestPrivateStoreBuy/Sell`, ActionUse 10/28/61 | ported (sell / buy / package; no manufacture, no `NO_STORE` zone) |
| Private warehouse | `PcWarehouse`, `WarehouseKeeper` | ported (no clan/freight) |
| Shortcuts register/delete | `RequestShortCutReg` / `RequestShortCutDel` | ported |
| Destroy item | `RequestDestroyItem` | ported |
| Item skills (potions, SOE) | `UseItem` + `item_skill` | ported (`RECALL` and HOT/buff skills) |
| Friends | `RequestFriendInvite` and `FriendList` | ported (in-memory + SQL) |
| Ground items / pickup | `RequestDropItem`, `ItemInstance.dropMe` / `onAction` | ported (`DropItem` / `GetItem`; no auto-despawn timer) |
| Enchant | `RequestEnchantItem`, `AbstractEnchantPacket` | ported (Java scroll table and chances) |
| Crystallize | `RequestCrystallizeItem` | ported (skill 248 + crystal_type / crystal_count) |
| Multisell | `MultiSellList`, `MultiSellChoose` | ported (NPC `Multisell` bypass; no clan reputation) |
| Recipe book / self-craft | `Recipes` handler, `RecipeItemMaker` | ported (learn, craft, forget: `0xAC`/`0xAD`/`0xAE`/`0xAF`) |
| Henna | `RequestHennaEquip` / `Unequip` / info / lists | ported (3 slots, stat bonuses; no dye shop HTML) |
| User commands | `RequestUserCommand` | ported `/loc`, `/time`, `/unstuck` (instant), `/partyinfo` |
| Quest list packet | `RequestQuestList` | empty `QuestList` (no quest scripts) |
| Mini-map | `RequestShowMiniMap` | ported (`ShowMiniMap` 1665) |
| Geodata / pathfinding | `GeoEngine`, `PathFinder`, `DoorData` geo objects | ported (L2J/L2OFF load, LOS, `canMove` / `getValidLocation`, A* path, door footprints). Binary region files are optional; missing files stay Null / unrestricted |

## Game server — still not ported

XML tables for the remaining Java loaders are now vendored and parsed. These
still need the gameplay managers / packets on top of that data:

- Live door / static-object spawn packets (`DoorInfo` is unused in the Unity tree)
- Quests and the Java script tree (`scripts.xml` is only an index)
- Clan warehouse / freight
- Clans, allies, wars, crests, command channel
- Castle siege, clan hall, manor, Seven Signs, Festival, Olympiad, heroes
- Community board, petitions, admin commands
- Boats, fishing, cursed weapons, augmentation, cubics, macros, subclasses
- Debuff and status effects on NPCs (monsters have no effect list yet)
- Handler registries (`handler/` item/chat/target handlers);
  Go still inlines most of this in `handler.go` / `handler_ingame.go`
- `ThreadPool`, `HtmCache`, `CrestCache`, `GlobalMemo`, `PlayerInfoTable`,
  server-crash tracking in `server_memo`

`internal/config/game.go` covers the settings this port needs; Java `Config`
exposes several hundred more (`RATE_*`, `ENCHANT_*`, `OLY_*`, `SEVEN_SIGNS_*`,
`INVENTORY_*`, `AUTO_LOOT`, …). Geoengine settings load from
`data/geodata/geoengine.properties`. About 30 Java tables
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
| `SkillTreeData` | `xml/skillstrees/` | yes | yes | tables loaded (fishing / clan / enchant) |
| `InstantTeleportData` | `xml/instantTeleports.xml` | yes | yes | NPC `instant` bypass |
| `HealSpsData` | `xml/healSps.xml` | yes | yes | heal amount correction |
| `NewbieBuffData` | `xml/newbieBuffs.xml` | yes | yes | newbie guide `SupportMagic` |
| `HennaData` | `xml/hennas.xml` | yes | yes | draw / remove + stat bonuses |
| `MultisellData` | `xml/multisell/` | yes | yes | NPC exchange + `MultiSellChoose` |
| `RecipeData` | `xml/recipes.xml` | yes | yes | learn + self-craft |
| `ArmorSetData` | `xml/armorSets.xml` | yes | yes | set bonuses in `RecalcStats` |
| `SpellbookData` | `xml/spellbooks.xml` | yes | yes | consumed on skill learn |
| `SummonItemData` | `xml/summonItems.xml` | yes | yes | table loaded |
| `SoulCrystalData` | `xml/soulCrystals.xml` | yes | yes | table loaded |
| `AugmentationData` | `xml/augmentation/` | yes | yes | skills / stat tables loaded |
| `FishData` | `xml/fish.xml` | yes | yes | table loaded |
| `CursedWeaponManager` | `xml/cursedWeapons.xml` | yes | yes | table loaded |
| `DoorData` | `xml/doors.xml` | yes | yes | templates loaded |
| `StaticObjectData` | `xml/staticObjects.xml` | yes | yes | table loaded |
| `WalkerRouteData` | `xml/walkerRoutes.xml` | yes | yes | table loaded |
| `SpawnManager` | `xml/spawnlist/` | yes | yes | Talking Island makers; `spawnlist_disabled/` not copied |
| `ZoneManager` | `xml/zones/` | yes | yes | `InPeaceZone` |
| `AnnouncementData` | `xml/announcements.xml` | yes | yes | sent on enter world (file is empty upstream) |
| `AdminData` | `xml/accessLevels.xml`, `adminCommands.xml` | yes | yes | tables loaded |
| `BufferManager` | `xml/bufferSkills.xml` | yes | yes | table loaded |
| `CastleManager` | `xml/castles.xml` | yes | yes | templates loaded (no siege manager) |
| `ClanHallManager` | `xml/clanHalls.xml`, `clanHallDeco.xml` | yes | yes | templates loaded |
| `ManorAreaData` / `CastleManorManager` | `xml/manorAreas.xml`, `manors.xml` | yes | yes | tables loaded |
| `ObserverGroupData` | `xml/observerGroups.xml` | yes | yes | table loaded |
| `BoatData` | `xml/boatRoutes.xml` | yes | yes | table loaded |
| `ScriptData` | `xml/scripts.xml` + quest tree | partial | index only | quest Java sources not ported |
| `HtmCache` | `data/html/` | no | — | |
| `GeoEngine` | `data/geodata/` | properties | yes | movement / LOS / AI / doors |

Item XML now also stores `crystal_type`, `crystal_count`, `is_dropable`, `soulshots`, `spiritshots` and `default_action`. Npc XML still skips
`<skills>` and AI fields.

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
