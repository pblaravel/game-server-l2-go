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
reference/        # original Java trees
```

## Run

```bash
docker compose up -d postgres
psql postgres://l2unity:l2unity@localhost:5432/l2unity -f sql/001_init.sql   # if not using compose init
make build
./bin/loginserver
./bin/gameserver
```

Defaults match Java:

| Service | Port |
|---------|------|
| Login clients | 2107 |
| Game server registration | 9015 |
| Game clients | 7778 |

Configs: `conf/loginserver/server.properties`, `conf/gameserver/server.properties`.

If PostgreSQL is down, both processes fall back to in-memory stores (dev/tests).

## Tests

```bash
make test-short          # unit + protocol
make test                # includes load tests
make load                # handshake load only
```

## Protocol

See [docs/PROTOCOL.md](docs/PROTOCOL.md).
