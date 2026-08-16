# Java → Go coverage

Reference trees: `reference/l2-unity-loginserver`, `reference/l2-unity-gameserver`.

The **wire API is the contract**: opcodes, packet layouts, Blowfish/RSA/GameCrypt,
session keys, ports. Gameplay systems that Java loads from XML/geodata are not a
line-by-line port of the ~2300-file aCis tree.

## Login server — complete

Matches Java `LoginClientThread` + `GameServerListener` / `GameServerThread`:

| Area | Status |
|------|--------|
| Init / Ping / AuthRequest / ServerList / PlayOk | yes |
| Autocreate, inactive, banned, already-on-LS/GS, kick | yes |
| ShowLicense skip of login-pair check | yes (Java quirk kept) |
| GS BlowFishKey RSA-512 NoPadding, AuthRequest, AuthResponse | yes |
| PlayerInGame / Logout / PlayerAuth / Kick / ServerStatus / ReplyCharacters | yes |
| Accounts + gameservers in PostgreSQL | yes |

Java login has **no SQL seed rows**. `gameservers` is filled when a GS registers.

## Game server — protocol core

Implemented and client-compatible:

- VersionCheck / AuthLogin via login bridge
- Char create / delete / select / EnterWorld
- Starter items, skills, shortcuts (Java `RequestCharacterCreate`)
- Unity `0x02` PlayerMoveDirection / `0xC6` MoveDirection
- Chat, attack, target, inventory list + reorder, skill list/use
- UserInfo / CharInfo / NpcInfo / CharSelectInfo / CharSelected
- Newbie-zone NPC + mob spawns so EnterWorld is not empty

## Game server — not ported (Java XML/datapack)

Java `GameServer` constructor loads these; Go does **not** run them:

- ItemData / NpcData / SkillTable / SkillTree / PlayerData XML
- Full Interlude spawn list (`SpawnManager.spawn()`)
- GeoEngine / pathfinding / zones / doors
- Quests and `ScriptData`
- Castle siege, clan hall, manor runtime
- Seven Signs / Festival of Darkness runtime
- Olympiad / heroes
- Clans, allies, wars, crests
- BuyList / multisell / recipes / warehouse / trade / party
- Community board, petitions, admin commands
- Boats, fishing, cursed weapons, augmentation
- AI scripts and task managers (decay, PvP flag, water, …)

Those tables exist in `sql/001_init.sql` so a later datapack load can persist
state, but there is no runtime manager behind them.

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
| PlayerData class templates | `class_templates` + in-memory kits | yes |
| NpcData + SpawnManager XML | newbie `npc_templates` / `npc_spawns` | yes (Talking Island + race gates + starter mobs) |

How a zero-state server gets this data:

1. `docker compose up` runs `sql/001_init.sql` then `sql/002_seed.sql`.
2. Both `cmd/loginserver` and `cmd/gameserver` call `db.ApplySchemaAndSeed` on
   connect, so a DB created without compose still gets schema + seed.
3. If PostgreSQL is down, the process uses in-memory stores and the gameserver
   still loads `DefaultNewbieSpawns`.
4. Character create always grants the class starter kit (items/skills/shortcuts)
   and persists them to `items` / `character_skills` / `character_shortcuts`.
