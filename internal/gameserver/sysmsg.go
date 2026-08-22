package gameserver

// System message ids used by the server, taken from Java
// gameserver/network/SystemMessageId.java.
const (
	SMTargetTooFar                 int32 = 22
	SMNotEnoughHP                  int32 = 23
	SMNotEnoughMP                  int32 = 24
	SMPickedUpS2S1                 int32 = 29
	SMPickedUpS1                   int32 = 30
	SMCantMoveSitting              int32 = 31
	SMItemEquipped                 int32 = 49
	SMEffectWornOff                int32 = 92
	SMNotEnoughSP                  int32 = 93
	SMEarnedExpAndSP               int32 = 95
	SMLevelIncreased               int32 = 96
	SMCannotDiscardThisItem        int32 = 98
	SMYouInvitedToParty            int32 = 105
	SMLeftParty                    int32 = 108
	SMInvalidTarget                int32 = 109
	SMFeelEffect                   int32 = 110
	SMSlotsFull                    int32 = 129
	SMOnlyLeaderCanInvite          int32 = 154
	SMPartyFull                    int32 = 155
	SMAlreadyInParty               int32 = 160
	SMWaitingForReply              int32 = 164
	SMSelectPartyTarget            int32 = 185
	SMYouLeftParty                 int32 = 200
	SMItemMissingToLearn           int32 = 276
	SMLearnedSkill                 int32 = 277
	SMNotEnoughAdena               int32 = 279
	SMWeightLimitExceeded          int32 = 422
	SMExceededInputQty             int32 = 1036
	SMRequestForTrade              int32 = 118
	SMDeniedTradeRequest           int32 = 119
	SMTradeSuccessful              int32 = 123
	SMCanceledTrade                int32 = 124
	SMAddedToFriends               int32 = 132
	SMAlreadyTrading               int32 = 142
	SMTargetIncorrect              int32 = 144
	SMTargetNotFound               int32 = 145
	SMCannotDestroyNumber          int32 = 163
	SMCannotAddYourself            int32 = 165
	SMNotEnoughAdenaFee            int32 = 281
	SMNoItemInWarehouse            int32 = 282
	SMJoinedAsFriend               int32 = 479
	SMUserNotInFriendsList         int32 = 486
	SMExchangeEnded                int32 = 1266
	SMEarnedS2S1S                  int32 = 53
	SMEarnedItemS1                 int32 = 54
	SMSuccessfullyEnchanted        int32 = 62
	SMSuccessfullyEnchantedS1S2    int32 = 63
	SMCannotDiscardDistance        int32 = 151
	SMYouDroppedS1                 int32 = 298
	SMSelectItemToEnchant          int32 = 303
	SMNotEnoughItems               int32 = 351
	SMInappropriateEnchant         int32 = 355
	SMCreateLvlTooLow              int32 = 404
	SMCrystallizeLevelTooLow       int32 = 562
	SMItemMixingFailed             int32 = 719
	SMRecipeAlreadyRegistered      int32 = 840
	SMS1Added                      int32 = 851
	SMSymbolAdded                  int32 = 877
	SMSymbolDeleted                int32 = 878
	SMCantDrawSymbol               int32 = 899
	SMSymbolsFull                  int32 = 900
	SMCantRegisterNoAbility        int32 = 1061
	SMNoRecipeBookWhileCasting     int32 = 1124
	SMS1Crystallized               int32 = 1258
	SMBlessedEnchantFailed         int32 = 1517
	SMSuccessfullyTradedWithNPC    int32 = 1656
	SMSoulshotsGradeMismatch       int32 = 337
	SMNotEnoughSoulshots           int32 = 338
	SMCannotUseSoulshots           int32 = 339
	SMEnabledSoulshot              int32 = 342
	SMSpiritshotsGradeMismatch     int32 = 530
	SMNotEnoughSpiritshots         int32 = 531
	SMCannotUseSpiritshots         int32 = 532
	SMEnabledSpiritshot            int32 = 533
	SMCantOperateStoreDuringCombat int32 = 1135
	SMUseOfS1WillBeAuto            int32 = 1433
	SMAutoUseOfS1Cancelled         int32 = 1434
	SMS1HasBeenDeleted             int32 = 848
	SMCantAlterRecipebook          int32 = 853
	SMStuckTransportInFiveMinutes  int32 = 809
	SMLocTalkingIsland             int32 = 910
	SMTimeS1S2InTheDay             int32 = 927
	SMTimeS1S2InTheNight           int32 = 928
	SMPartyInformation             int32 = 1030
	SMLootingFindersKeepers        int32 = 1031
	SMLootingRandom                int32 = 1032
	SMLootingRandomIncludeSpoil    int32 = 1033
	SMLootingByTurn                int32 = 1034
	SMLootingByTurnIncludeSpoil    int32 = 1035
	SMOnlyLeaderCanTransferRights  int32 = 1399
	SMPartyLeaderS1                int32 = 1611
)

const (
	StoreNone        int32 = 0
	StoreSell        int32 = 1
	StoreSellManage  int32 = 2
	StoreBuy         int32 = 3
	StoreBuyManage   int32 = 4
	StorePackageSell int32 = 8
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
