# L2 Unity Game/Login Server (Go + PostgreSQL)

Go rewrite of:

- https://github.com/shnok/l2-unity-loginserver
- https://github.com/shnok/l2-unity-gameserver

Java sources stay under `reference/` as the protocol/behavior spec.
The **client and inter-server binary API is unchanged** (opcodes, layouts,
Blowfish/RSA/GameCrypt, session keys, ports).

## Layout

```
cmd/loginserver   # Java login server
cmd/gameserver    # Java game server
internal/crypt    # L2J Blowfish, NewCrypt, LoginCrypt, GameCrypt, scramble
internal/packet   # LE writer/reader + length framing
internal/loginserver
internal/gameserver
sql/001_init.sql  # PostgreSQL schema (accounts, characters, clans, …)
sql/002_seed.sql  # Java castle/clanhall/7s/mdt + newbie NPCs/levels
sql/003_skills.sql  # skill_templates + class_skills tables
data/xml/skills   # Java SkillTable
data/xml/classes  # Java PlayerData skill trees + newbie items
docs/COVERAGE.md  # Java → Go functionality and seed audit
reference/        # original Java trees
```

## Run

```bash
make docker-up             # postgres + login (2107/9015) + game (7778)
make docker-smoke          # checks login Init, InitLS, game VersionCheck
```

Or without Docker:

```bash
docker compose up -d postgres
make build
./bin/loginserver
./bin/gameserver
```

`make docker-up` also turns off `net.bridge.bridge-nf-call-iptables` when possible.
Some nested Docker VMs drop container-to-container TCP while that sysctl is `1`.

Defaults match Java:

| Service | Port |
|---------|------|
| Login clients | 2107 |
| Game server registration | 9015 |
| Game clients | 7778 |

Configs: `conf/loginserver/server.properties`, `conf/gameserver/server.properties`.

Game server packet traces are on by default (`PacketHandlerDebug`, `logger.print.received-packets`, `logger.print.sent-packets`).
Look for `GS RECV` / `GS SEND` / `GS CHANGE` / `GS STATE` / `[GAME]` in the process log, or `docker compose logs -f gameserver`.
The first client packet is `ProtocolVersion` (Unity uses **740**); the server must answer with `VersionCheck` and must not close the TCP socket.
Set `PACKET_LOG=false` to turn them off.

If PostgreSQL is down, both processes fall back to in-memory stores (dev/tests).

## Tests

```bash
make test-short          # unit + protocol
make test                # includes load tests
make load                # handshake load only
```

## Protocol

See [docs/PROTOCOL.md](docs/PROTOCOL.md). Java vs Go coverage: [docs/COVERAGE.md](docs/COVERAGE.md).
