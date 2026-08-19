# Java → Go coverage

Reference trees: `reference/l2-unity-loginserver` (75 `.java`),
`reference/l2-unity-gameserver` (2301 `.java`, ~339k lines).
Go implementation: 42 non-test files, ~7.2k lines.

The **wire API is the contract**: opcodes, packet layouts, Blowfish/RSA/GameCrypt,
session keys, ports. Gameplay systems that Java loads from XML/geodata are not a
line-by-line port of the aCis-derived tree.

This file was re-verified against the sources: packet field-type sequences were
compared write-by-write, the client opcode table against Java
`network/GamePacketHandler.java`, and the startup checklist against the
`GameServer.java` constructor. Line numbers below point at the current code.

## Login server — protocol complete, service gaps

Matches Java `LoginClientThread` + `GameServerListener` / `GameServerThread`:

| Area | Status |
|------|--------|
| Init / Ping / AuthRequest / ServerList / PlayOk | yes |
| Autocreate, inactive, banned, already-on-LS/GS, kick | yes |
| ShowLicense skip of login-pair check | yes (Java quirk kept) |
| GS BlowFishKey RSA-512 NoPadding, AuthRequest, AuthResponse | yes |
| PlayerInGame / Logout / PlayerAuth / Kick / ServerStatus / ReplyCharacters | yes |
| Accounts + gameservers in PostgreSQL | yes |

Password handling is the Java behaviour: the client sends hash bytes, the server
stores `Base64(bytes)` and compares strings
(`ClientPacketHandler.java:116-121` → `internal/loginserver/packets.go:158`).

Known gaps against Java:

- **No idle disconnect.** Java arms a timer on `server.connection.timeout.ms`
  (`ClientPacketHandler.java:90-102`). Go records `lastEcho`
  (`internal/loginserver/client.go:39,120`) but never reads it, and
  `ConnectionTimeoutMS` (`internal/config/login.go:57`) is unused.
- **Harsher failure path.** A bad checksum/decrypt closes the socket
  (`internal/loginserver/client.go:110-113`); Java logs and keeps the connection
  (`ClientPacketHandler.java:52-63`).
- **`logger.print.*` flags are parsed and ignored** in the login server
  (`internal/config/login.go:70-71`, no readers in `internal/loginserver`).
- **Shutdown closes listeners only** (`internal/loginserver/server.go:47-50`);
  Java `ServerShutdownService` also disconnects live clients and gameservers.

`GGAuth`, `ChangeAccessLevel` and `RequestTempBan` are unimplemented in Java too.
Java login has **no SQL seed rows**; `gameservers` is filled when a GS registers.

## Game server — client opcodes

Handled in `internal/gameserver/handler.go`: `0x00`, `0x03`, `0x08`, `0x09`,
`0x0B`, `0x0C`, `0x0D`, `0x0E`, `0x62` (auth/selection) and `0x01`, `0x02`,
`0x04`, `0x05`, `0x0A`, `0x0F`, `0x10`, `0x11`, `0x12`, `0x14`, `0x1C`, `0x2F`,
`0x30`, `0x38`, `0x3F`, `0x45`, `0x48`, `0x6D` in game. Everything else is
logged as unhandled; extended `0xD0` packets are read and dropped
(`handler.go:61-64`).

Wrong opcode mapping (Java `GamePacketHandler.java`):

| Opcode | Java | Go |
|--------|------|-----|
| `0x45` | `RequestActionUse`, reads `D,D,C` (`clientpackets/combat/RequestActionUse.java:79-81`) | treated as a social action, reads one `D` (`handler.go:191-194`) |
| `0x1B` | `RequestSocialAction` (`GamePacketHandler.java:189`) | not handled |
| `0x6D` | `RequestRestartPoint`, reads `D requestType` (`clientpackets/combat/RequestRestartPoint.java:28`) | answers `TeleportToLocation` to the current position, no respawn (`handler.go:200-201`) |
| `0x01` | `MoveBackwardToLocation`, reads target + origin + `moveMovement` (`clientpackets/movement/legacy/MoveBackwardToLocation.java:37-46`) | reads only the target triple, replies `StopMove`; server packet `MoveToLocation` (0x01) is missing (`handler.go:74-80`) |

Handled by Java, missing in Go: `0x1E RequestSellItem`, `0x1F RequestBuyItem`,
`0x37 RequestTargetCancel`, `0x46 RequestRestart`,
`0x6B RequestAcquireSkillInfo`, `0x6C RequestAcquireSkill`.

Acknowledged without game logic: `0x0A` attacks deal a fixed 10 damage
(`handler.go:117-124`), `0x2F` only broadcasts `MagicSkillUse` with the template
timings, `0x11`/`0x12`/`0x14` answer `ItemList` without changing the inventory.

## Game server — server packet layouts

Field-type sequence verified identical to Java: `VersionCheck`, `UserInfo`,
`CharSelectInfo`, `CharSelected`, `ItemList`, `SkillList`, `StatusUpdate`,
`MoveDirection`, `ActionFailed`, `ShortCutInit`, `Attack` (single hit) and
`SystemMessage` with no parameters.

Diverging from Java:

- **`CharInfo` (0x03).** Java writes `writeLoc(x,y,z)`, the boat object id, then
  12 paperdoll item ids plus the augmentation `H` blocks — about 103 fields
  (`serverpackets/auth/CharInfo.java:35-165`). Go writes `x,y,z,heading` (no
  boat), 17 paperdoll ids and a truncated tail — about 59 fields
  (`internal/gameserver/packets.go:310-355`).
- **`NpcInfo` (0x16).** Java sends r-hand/chest/l-hand ids, five state bytes,
  abnormal effect, clan/ally ids, move type, a second collision pair, enchant,
  flying flag, attack range, `D maxHp`, `D level`
  (`serverpackets/actor/AbstractNpcInfo.java:134-199`). Go omits all of those and
  sends `F maxHp`, `F curHp`, `D 0` instead (`internal/gameserver/packets.go:358-400`).

Where the layout matches, several values are still placeholders: the `UserInfo`
exp bar, weight and weapon load (`packets.go:207,219-221`), and the
`CharSelectInfo` delete timer, enchant and augmentation id
(`packets.go:125,132-133`). `Attack` sends the client-supplied origin instead of
the attacker position (`handler.go:119-124`).

`MagicSkillLaunched`, `ShortCutRegister`, `AcquireSkill*`, `MoveToLocation` and
the whole `0xFE` extended family have no Go counterpart.

## Game server — subsystems not ported

The Java `GameServer` constructor (`GameServer.java:114-320`) runs roughly 79
init steps. `cmd/gameserver/main.go` runs config load, DB connect, datapack load
(skills + class trees), spawn seeding and the two listeners. Missing:

- `ItemData`, `SkillTreeData`, `HennaData`, `MultisellData`, `RecipeData`,
  `ArmorSetData`, `FishData`, `SpellbookData`, `SoulCrystalData`,
  `AugmentationData`, `SummonItemData`, `DoorData`, `TeleportData`,
  `RestartPointData`, `NewbieBuffData`, `StaticObjectData`, `WalkerRouteData`,
  `ObserverGroupData`, `AnnouncementData`, `AdminData`, `ScriptData`, `BoatData`
  (27 of 29 XML loaders; Go parses only `data/xml/skills` and `data/xml/classes`)
- Skill execution: `skills/effects` (60), `skills/conditions` (32),
  `skills/funcs` (18), `skills/basefuncs` (12), `skills/l2skills` (13),
  `handler/skillhandlers` (33). Go keeps skill metadata only, and says so at
  `internal/gameserver/skilltable.go:13-15`
- `GeoEngine`, `ZoneManager`, all 13 task managers, `model/actor/ai`,
  `model/actor/move`, attack/cast pipelines, HP/MP regeneration and the
  `Calculator`/`Formulas` stat system (Go stats are constants set at creation,
  `internal/gameserver/model.go:275-276`)
- Quests and AI scripts (`scripting/`, 343 quest files plus the monster AI tree)
- Clans/allies/wars/crests, party and command channel, trade, private stores,
  buy lists, multisell, warehouse, freight
- Castle siege, clan hall, manor, Seven Signs and Festival runtime, Olympiad,
  heroes, community board, petitions, admin commands
- Boats, fishing, cursed weapons, augmentation, cubics, henna, macros, recipes,
  subclasses, punishment state
- Handler registries (`handler/` item/chat/user-command/target handlers);
  Go inlines a single switch in `handler.go`
- `ThreadPool`, `HtmCache`, `CrestCache`, `GlobalMemo`, `PlayerInfoTable`,
  server-crash tracking in `server_memo`

`internal/gameserver/model.go` carries no effect list, party, clan object,
subclass, quest, henna, macro, recipe or punishment state, and columns such as
`online`, `onlinetime`, `nobless`, `hero`, `punish_level`, `death_penalty_level`,
`expbeforedeath`, `rec_have`/`rec_left` exist in `sql/001_init.sql` but are never
read or written. About 30 Java tables (community board, auctions, manor,
petitions, olympiad fights, cursed weapons, …) are absent from the Go schema.

`internal/config/game.go` covers roughly 25 settings; Java `Config` exposes
several hundred (all `RATE_*`, `ENCHANT_*`, `OLY_*`, `SEVEN_SIGNS_*`,
`GEODATA_PATH`, `INVENTORY_*`, `AUTO_LOOT`, …).

Tables for the unported systems exist in `sql/001_init.sql` so a later datapack
load can persist state, but there is no runtime manager behind them.

## Seed data on a blank start

Java SQL `INSERT`s exist only in five files. Almost all world content is XML
under `data/xml/` (not copied into this repo).

| Java source | Go seed | Applied on empty DB |
|-------------|---------|---------------------|
| `castle.sql` (9 castles) | `sql/002_seed.sql` | yes |
| `clanhall.sql` (ids 21–64) | `sql/002_seed.sql` | yes |
| `seven_signs_status.sql` | `sql/002_seed.sql` | yes |
| `seven_signs_festival.sql` | `sql/002_seed.sql` | yes |
| `mdt_bets.sql` | `sql/002_seed.sql` | yes |
| `gameservers.sql` (schema only) | none (GS self-registers) | n/a |
| PlayerLevelData XML | `player_levels` 1–81 | yes |
| PlayerData class templates | `class_templates` + `data/xml/classes` + `class_skills` | yes |
| SkillTable XML | `data/xml/skills` → `skill_templates` | yes (parsed at GS start, upserted to DB) |
| NpcData + SpawnManager XML | newbie `npc_templates` / `npc_spawns` | yes (Talking Island + race gates + starter mobs) |

How a zero-state server gets this data:

1. `docker compose up` runs `sql/001_init.sql` then `sql/002_seed.sql`.
2. Both `cmd/loginserver` and `cmd/gameserver` call `db.ApplySchemaAndSeed` on
   connect, so a DB created without compose still gets schema + seed.
3. If PostgreSQL is down, the process uses in-memory stores and the gameserver
   still loads `DefaultNewbieSpawns`.
4. Character create grants the class starter items and **autoGet skills**
   (`cost="0"` in the class XML, same as Java `getAvailableAutoGetSkills`).
   Learn-from-NPC skills (Power Strike, etc.) stay on the class tree until
   a trainer handler is added.
