package clientapi

// Pair is one Unity-client C→S packet and the S→C opcodes it must produce.
type Pair struct {
	Name     string
	C2S      byte
	State    string
	S2C      []byte
	Optional bool
}

// LoginPairs is the Interlude login sequence from l2-unity Assets/Scripts/Networking.
func LoginPairs() []Pair {
	return []Pair{
		{Name: "Init", C2S: 0xFF, State: "connect", S2C: []byte{0x00}},
		{Name: "AuthGameGuard", C2S: 0x07, State: "connected", S2C: []byte{0x0B}},
		{Name: "RequestAuthLogin", C2S: 0x00, State: "gg", S2C: []byte{0x03}},
		{Name: "RequestServerList", C2S: 0x05, State: "authed", S2C: []byte{0x04}},
		{Name: "RequestServerLogin", C2S: 0x02, State: "authed", S2C: []byte{0x07}},
	}
}

// GamePairs is every outgoing game packet the Unity client implements.
func GamePairs() []Pair {
	return []Pair{
		{Name: "ProtocolVersion", C2S: 0x00, State: "connected", S2C: []byte{0x00}},
		{Name: "AuthLogin", C2S: 0x08, State: "connected", S2C: []byte{0x13}},
		{Name: "NewCharacter", C2S: 0x0E, State: "authed", S2C: []byte{0x17}},
		{Name: "CharacterCreate", C2S: 0x0B, State: "authed", S2C: []byte{0x19, 0x13}},
		{Name: "CharacterDelete", C2S: 0x0C, State: "authed", S2C: []byte{0x13}},
		{Name: "CharacterSelect", C2S: 0x0D, State: "authed", S2C: []byte{0x15}},
		{Name: "EnterWorld", C2S: 0x03, State: "entering", S2C: []byte{0x04, 0x1B, 0x58, 0x45, 0x80}},
		{Name: "Appearing", C2S: 0x30, State: "ingame", S2C: []byte{0x04}},
		{Name: "MoveBackwardToLocation", C2S: 0x01, State: "ingame", S2C: []byte{0x01}},
		{Name: "ValidatePosition", C2S: 0x48, State: "ingame", S2C: nil, Optional: true},
		{Name: "ClickAction", C2S: 0x04, State: "ingame", S2C: []byte{0xA6}},
		{Name: "RequestTargetCanceld", C2S: 0x37, State: "ingame", S2C: []byte{0x2A}},
		{Name: "RequestMagicSkillUse", C2S: 0x2F, State: "ingame", S2C: []byte{0x25}},
		{Name: "RequestRestartPoint", C2S: 0x6D, State: "ingame", S2C: nil, Optional: true},
		{Name: "UseItem", C2S: 0x14, State: "ingame", S2C: nil, Optional: true},
		{Name: "RequestDestroyItem", C2S: 0x59, State: "ingame", S2C: nil, Optional: true},
		{Name: "RequestEnchantItem", C2S: 0x58, State: "ingame", S2C: []byte{0x81}},
		{Name: "RequestBuyItem", C2S: 0x1F, State: "ingame", S2C: []byte{0x25}},
		{Name: "RequestSellItem", C2S: 0x1E, State: "ingame", S2C: []byte{0x25}},
		{Name: "RequestPreviewItem", C2S: 0xC6, State: "ingame", S2C: []byte{0xF0}},
		{Name: "MultiSellChoose", C2S: 0xA7, State: "ingame", S2C: []byte{0x25}},
		{Name: "AnswerTradeRequest", C2S: 0x44, State: "ingame", S2C: nil, Optional: true},
		{Name: "AddTradeItem", C2S: 0x16, State: "ingame", S2C: nil, Optional: true},
		{Name: "TradeDone", C2S: 0x17, State: "ingame", S2C: nil, Optional: true},
		{Name: "SendWarehouseDepositList", C2S: 0x31, State: "ingame", S2C: nil, Optional: true},
		{Name: "SendWarehouseWithdrawList", C2S: 0x32, State: "ingame", S2C: nil, Optional: true},
		{Name: "RequestPackageSendableItemList", C2S: 0x9E, State: "ingame", S2C: []byte{0xC3}},
		{Name: "RequestPackageSend", C2S: 0x9F, State: "ingame", S2C: []byte{0x25}},
		{Name: "RequestSay2", C2S: 0x38, State: "ingame", S2C: []byte{0x4A}},
		{Name: "RequestBypassToServer", C2S: 0x21, State: "ingame", S2C: nil, Optional: true},
		{Name: "RequestShowBoard", C2S: 0x57, State: "ingame", S2C: []byte{0x6E}},
		{Name: "RequestUserCommand", C2S: 0xAA, State: "ingame", S2C: []byte{0x64}},
		{Name: "RequestSkillCoolTime", C2S: 0x9D, State: "ingame", S2C: []byte{0xC1}},
		{Name: "RequestAcquireSkillInfo", C2S: 0x6B, State: "ingame", S2C: nil, Optional: true},
		{Name: "RequestAcquireSkill", C2S: 0x6C, State: "ingame", S2C: nil, Optional: true},
		{Name: "RequestShortCutReg", C2S: 0x33, State: "ingame", S2C: []byte{0x44}},
		{Name: "RequestShortCutDel", C2S: 0x35, State: "ingame", S2C: []byte{0x46}},
		{Name: "RequestQuestAbort", C2S: 0x64, State: "ingame", S2C: []byte{0x80}},
		{Name: "RequestAnswerJoinParty", C2S: 0x2A, State: "ingame", S2C: nil, Optional: true},
		{Name: "RequestPledgeInfo", C2S: 0x66, State: "ingame", S2C: []byte{0x83}},
		{Name: "RequestJoinPledge", C2S: 0x24, State: "ingame", S2C: []byte{0x25}},
		{Name: "RequestWithdrawPledge", C2S: 0x26, State: "ingame", S2C: []byte{0x04}},
		{Name: "RequestOustPledgeMember", C2S: 0x27, State: "ingame", S2C: []byte{0x25}},
		{Name: "RequestGiveNickName", C2S: 0x55, State: "ingame", S2C: []byte{0x04}},
		{Name: "RequestPledgePower", C2S: 0xC0, State: "ingame", S2C: []byte{0x30}},
		{Name: "RequestRecipeBookOpen", C2S: 0xAC, State: "ingame", S2C: []byte{0xD6}},
		{Name: "RequestRecipeBookDestroy", C2S: 0xAD, State: "ingame", S2C: []byte{0xD6}},
		{Name: "RequestRecipeItemMakeInfo", C2S: 0xAE, State: "ingame", S2C: []byte{0xD7}},
		{Name: "RequestRecipeItemMakeSelf", C2S: 0xAF, State: "ingame", S2C: []byte{0xD7}},
	}
}

func ContainsOpcode(got []byte, want byte) bool {
	for _, o := range got {
		if o == want {
			return true
		}
	}
	return false
}
