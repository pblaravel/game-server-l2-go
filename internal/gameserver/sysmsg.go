package gameserver

// System message ids used by the server, taken from Java
// gameserver/network/SystemMessageId.java.
const (
	SMTargetTooFar          int32 = 22
	SMNotEnoughHP           int32 = 23
	SMNotEnoughMP           int32 = 24
	SMCantMoveSitting       int32 = 31
	SMItemEquipped          int32 = 49
	SMEffectWornOff         int32 = 92
	SMNotEnoughSP           int32 = 93
	SMEarnedExpAndSP        int32 = 95
	SMLevelIncreased        int32 = 96
	SMCannotDiscardThisItem int32 = 98
	SMInvalidTarget         int32 = 109
	SMFeelEffect            int32 = 110
	SMWeightLimitExceeded   int32 = 422
	SMLearnedSkill          int32 = 277
)

// StatusUpdate attribute ids from Java enums/StatusType.java.
const (
	StatusLevel    int32 = 1
	StatusExp      int32 = 2
	StatusSTR      int32 = 3
	StatusDEX      int32 = 4
	StatusCON      int32 = 5
	StatusINT      int32 = 6
	StatusWIT      int32 = 7
	StatusMEN      int32 = 8
	StatusCurHP    int32 = 9
	StatusMaxHP    int32 = 10
	StatusCurMP    int32 = 11
	StatusMaxMP    int32 = 12
	StatusSP       int32 = 13
	StatusCurLoad  int32 = 14
	StatusMaxLoad  int32 = 15
	StatusPAtk     int32 = 17
	StatusAtkSpd   int32 = 18
	StatusPDef     int32 = 19
	StatusEvasion  int32 = 20
	StatusAccuracy int32 = 21
	StatusCritical int32 = 22
	StatusMAtk     int32 = 23
	StatusCastSpd  int32 = 24
	StatusMDef     int32 = 25
	StatusPvPFlag  int32 = 26
	StatusKarma    int32 = 27
	StatusCurCP    int32 = 33
	StatusMaxCP    int32 = 34
)
