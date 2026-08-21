package gameserver

// System message ids used by the server, taken from Java
// gameserver/network/SystemMessageId.java.
const (
	SMTargetTooFar          int32 = 22
	SMNotEnoughHP           int32 = 23
	SMNotEnoughMP           int32 = 24
	SMPickedUpS2S1          int32 = 29
	SMPickedUpS1            int32 = 30
	SMCantMoveSitting       int32 = 31
	SMItemEquipped          int32 = 49
	SMEffectWornOff         int32 = 92
	SMNotEnoughSP           int32 = 93
	SMEarnedExpAndSP        int32 = 95
	SMLevelIncreased        int32 = 96
	SMCannotDiscardThisItem int32 = 98
	SMYouInvitedToParty     int32 = 105
	SMLeftParty             int32 = 108
	SMInvalidTarget         int32 = 109
	SMFeelEffect            int32 = 110
	SMSlotsFull             int32 = 129
	SMOnlyLeaderCanInvite   int32 = 154
	SMPartyFull             int32 = 155
	SMAlreadyInParty        int32 = 160
	SMWaitingForReply       int32 = 164
	SMSelectPartyTarget     int32 = 185
	SMYouLeftParty          int32 = 200
	SMLearnedSkill          int32 = 277
	SMNotEnoughAdena        int32 = 279
	SMWeightLimitExceeded   int32 = 422
	SMExceededInputQty      int32 = 1036
	SMRequestForTrade       int32 = 118
	SMDeniedTradeRequest    int32 = 119
	SMTradeSuccessful       int32 = 123
	SMCanceledTrade         int32 = 124
	SMAddedToFriends        int32 = 132
	SMAlreadyTrading        int32 = 142
	SMTargetIncorrect       int32 = 144
	SMTargetNotFound        int32 = 145
	SMCannotDestroyNumber   int32 = 163
	SMCannotAddYourself     int32 = 165
	SMNotEnoughAdenaFee     int32 = 281
	SMNoItemInWarehouse     int32 = 282
	SMJoinedAsFriend        int32 = 479
	SMUserNotInFriendsList  int32 = 486
	SMExchangeEnded         int32 = 1266
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
