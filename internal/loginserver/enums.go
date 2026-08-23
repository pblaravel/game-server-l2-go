package loginserver

// Client opcodes (ClientPacketType).
const (
	ClientPing              byte = 0x00
	ClientAuthRequest       byte = 0x01
	ClientRequestServerList byte = 0x02
	ClientRequestServerLogin byte = 0x03
	// Interlude / Unity client (l2-unity Assets/Scripts/Networking).
	ClientAuthGameGuard              byte = 0x07
	ClientRequestServerListInterlude byte = 0x05
)

// Server opcodes (ServerPacketType).
const (
	ServerInit          byte = 0x00
	ServerLoginFail     byte = 0x01
	ServerAccountKicked byte = 0x02
	ServerLoginOk       byte = 0x03
	ServerServerList    byte = 0x04
	ServerPlayFail      byte = 0x06
	ServerPlayOk        byte = 0x07
	ServerGGAuth        byte = 0x0B
	ServerPing          byte = 0x63
)

// GameServer → LoginServer opcodes (GameServerPacketType).
const (
	GSBlowFishKey       byte = 0x00
	GSAuthRequest       byte = 0x01
	GSPlayerInGame      byte = 0x02
	GSPlayerLogout      byte = 0x03
	GSChangeAccessLevel byte = 0x04
	GSPlayerAuthRequest byte = 0x05
	GSServerStatus      byte = 0x06
	GSReplyCharacters   byte = 0x07
)

// LoginServer → GameServer opcodes (LoginServerPacketType).
const (
	LSInitLS             byte = 0x00
	LSFail               byte = 0x01
	LSAuthResponse       byte = 0x02
	LSPlayerAuthResponse byte = 0x03
	LSKickPlayer         byte = 0x04
	LSReceivableList     byte = 0x05
	LSRequestCharacters  byte = 0x06
)

type GameServerState int

const (
	GSConnected GameServerState = iota
	GSBFConnected
	GSAAuthed
)

type LoginClientState int

const (
	ClientConnected LoginClientState = iota
	ClientAuthedLogin
)

type AuthLoginResult int

const (
	AuthInvalidPassword AuthLoginResult = iota
	AuthAccountInactive
	AuthAccountBanned
	AuthAlreadyOnLS
	AuthAlreadyOnGS
	AuthSuccess
)

// LoginFailReason / PlayFailReason codes used by the Java server.
const (
	ReasonNoMessage              byte = 0x00
	ReasonSystemErrorLoginLater  byte = 0x01
	ReasonUserOrPassWrong        byte = 0x02
	ReasonAccountInUse           byte = 0x07
	ReasonServerOverloaded       byte = 0x0F
	ReasonAccessFailed           byte = 0x15
	ReasonInactive               byte = 0x24
)

const (
	KickDataStealer        byte = 0x01
	KickGenericViolation   byte = 0x08
	Kick7DaysSuspended     byte = 0x10
	KickPermanentlyBanned  byte = 0x20
)

const (
	FailInvalidGameServerVersion = 0
	FailIPBanned                 = 1
	FailIPReserved               = 2
	FailWrongHexID               = 3
	FailIDReserved               = 4
	FailNoFreeID                 = 5
	FailNotAuthed                = 6
	FailAlreadyLoggedIn          = 7
)

const (
	StatusAuto   = 0
	StatusGood   = 1
	StatusNormal = 2
	StatusFull   = 3
	StatusDown   = 4
	StatusGMOnly = 5
)

const (
	AttrServerListStatus = 0x01
	AttrClock            = 0x02
	AttrBrackets         = 0x03
	AttrAgeLimit         = 0x04
	AttrTestServer       = 0x05
	AttrPvPServer        = 0x06
	AttrMaxPlayers       = 0x07
)
