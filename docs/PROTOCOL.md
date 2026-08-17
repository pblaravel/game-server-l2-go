# L2 Unity protocol (Java-compatible)

This document is the wire contract copied from:

- `reference/l2-unity-loginserver`
- `reference/l2-unity-gameserver`

The Go servers speak the same TCP binary API. Clients and the login↔game
link must not need changes.

## Framing (all channels)

```
uint16 LE length   // includes these 2 bytes
payload            // length-2 bytes
```

Integers are little-endian. Strings are UTF-16LE + `0x0000`.

## Login client (default :2107)

| Dir | Opcode | Packet |
|-----|--------|--------|
| S→C | 0x00 | Init (sessionId, scrambled RSA-1024 modulus, blowfish key) |
| C→S | 0x00 | Ping |
| S→C | 0x63 | Ping |
| C→S | 0x01 | AuthRequest (128-byte RSA PKCS1 block) |
| S→C | 0x01 | LoginFail |
| S→C | 0x02 | AccountKicked |
| S→C | 0x03 | LoginOk (loginOk1, loginOk2) |
| S→C | 0x04 | ServerList |
| C→S | 0x02 | RequestServerList |
| C→S | 0x03 | RequestServerLogin |
| S→C | 0x06 | PlayFail |
| S→C | 0x07 | PlayOk (playOk1, playOk2) |

Init is encrypted with the static Blowfish key + XOR pass (no checksum).
Every later packet uses the session Blowfish key + XOR checksum.

Password is client-hashed; the server stores/compares Base64 of those bytes.

## Login ↔ Game (default :9015)

State: `CONNECTED → BF_CONNECTED → AUTHED`.

| Dir | Opcode | Packet |
|-----|--------|--------|
| LS→GS | 0x00 | InitLS (revision 0x0102, RSA-512 modulus) |
| GS→LS | 0x00 | BlowFishKey (RSA/ECB/NoPadding) |
| GS→LS | 0x01 | AuthRequest (id, hexid, host, port, max) |
| LS→GS | 0x02 | AuthResponse |
| LS→GS | 0x01 | Fail |
| GS→LS | 0x02 | PlayerInGame |
| GS→LS | 0x03 | PlayerLogout |
| GS→LS | 0x05 | PlayerAuthRequest |
| LS→GS | 0x03 | PlayerAuthResponse |
| LS→GS | 0x04 | KickPlayer |
| GS→LS | 0x06 | ServerStatus |
| GS→LS | 0x07 | ReplyCharacters |

Default GS blowfish key: `_;v.]05-31!|+-%xT!^[$\0`

## Game client (default :7778)

Protocol versions: 737, 740, 744, 746 (Unity clients typically send **740**).
Unknown versions still get `VersionCheck` unless `StrictProtocol=true`.
Handshake is always logged as `[GAME] … ProtocolVersion` / `[GAME] … sending VersionCheck`.

| State | Opcode | Packet |
|-------|--------|--------|
| CONNECTED | 0x00 | SendProtocolVersion / VersionCheck |
| CONNECTED | 0x08 | AuthLogin |
| AUTHED | 0x0B/0x0C/0x0D | create / delete / start |
| ENTERING | 0x03 | EnterWorld |
| IN_GAME | 0x02 / 0xC6 | Unity MoveDirection |
| IN_GAME | 0x10 | RequestInventoryUpdateOrder |
| IN_GAME | 0x38 / 0x4A | Say2 / CreatureSay |
| IN_GAME | 0x0A / 0x05 | Attack / AttackRequest |

Client↔GS encryption is the L2 XOR stream (`GameCrypt`), not block Blowfish.
The first 8 key bytes are sent in VersionCheck; bytes 8–15 are the static tail
`C8 27 93 01 A1 6C 31 97`.

## Service API (kept from Java)

Login: `LoginServerController` / `GameServerController` methods
(`AddClient`, `GetClient`, `GetNewSessionKey`, `IsLoginPossible`,
`GetKeyForAccount`, `Register`, `RegisterServerOnDB`, …).

Game: `LoginServerThread` methods
(`AddClient`, `SendLogout`, `SendAccessLevel`, `KickPlayer`,
`SetMaxPlayer`, `SetServerType`, `GetServerName`, …).
