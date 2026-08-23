package gameserver

import (
	"fmt"
	"log"
	"strings"

	"github.com/pblaravel/game-server-l2-go/internal/packet"
)

func (s ClientState) String() string {
	switch s {
	case StateConnected:
		return "connected"
	case StateAuthed:
		return "authed"
	case StateEntering:
		return "entering"
	case StateInGame:
		return "ingame"
	default:
		return fmt.Sprintf("state(%d)", s)
	}
}

func clientOpcodeName(op byte, data []byte) string {
	if op == 0xD0 && len(data) >= 3 {
		ex := int(data[1]) | int(data[2])<<8
		if n, ok := clientExNames[ex]; ok {
			return fmt.Sprintf("Ex.%s", n)
		}
		return fmt.Sprintf("Ex.0x%02X", ex)
	}
	if n, ok := clientOpcodeNames[op]; ok {
		return n
	}
	return "Unknown"
}

func serverOpcodeName(op byte) string {
	if n, ok := serverOpcodeNames[op]; ok {
		return n
	}
	return "Unknown"
}

func lsInOpcodeName(op byte) string {
	switch op {
	case 0x00:
		return "InitLS"
	case 0x01:
		return "LoginServerFail"
	case 0x02:
		return "AuthResponse"
	case 0x03:
		return "PlayerAuthResponse"
	case 0x04:
		return "KickPlayer"
	default:
		return "Unknown"
	}
}

func lsOutOpcodeName(op byte) string {
	switch op {
	case 0x00:
		return "BlowFishKey"
	case 0x01:
		return "AuthRequest"
	case 0x02:
		return "PlayerInGame"
	case 0x03:
		return "PlayerLogout"
	case 0x04:
		return "ChangeAccessLevel"
	case 0x05:
		return "PlayerAuthRequest"
	case 0x06:
		return "ServerStatus"
	default:
		return "Unknown"
	}
}

var clientOpcodeNames = map[byte]string{
	0x00: "ProtocolVersion",
	0x01: "MoveBackwardToLocation",
	0x02: "PlayerMoveDirection",
	0x03: "EnterWorld",
	0x04: "Action",
	0x05: "RequestTarget",
	0x08: "AuthLogin",
	0x09: "Logout",
	0x0A: "AttackRequest",
	0x0B: "CharacterCreate",
	0x0C: "CharacterDelete",
	0x0D: "CharacterSelect",
	0x0E: "RequestNewCharacter",
	0x0F: "RequestItemList",
	0x10: "RequestInventoryUpdateOrder",
	0x11: "RequestUnEquipItem",
	0x12: "RequestDropItem",
	0x14: "UseItem",
	0x15: "TradeRequest",
	0x16: "AddTradeItem",
	0x17: "TradeDone",
	0x1C: "ChangeMoveType",
	0x1D: "RequestChangeWaitType",
	0x1E: "RequestSellItem",
	0x1F: "RequestBuyItem",
	0x21: "RequestBypassToServer",
	0x29: "RequestJoinParty",
	0x2A: "RequestAnswerJoinParty",
	0x2B: "RequestWithdrawParty",
	0x2C: "RequestOustPartyMember",
	0x2F: "RequestMagicSkillUse",
	0x30: "Appearing",
	0x31: "SendWarehouseDepositList",
	0x32: "SendWarehouseWithdrawList",
	0x33: "RequestShortCutReg",
	0x35: "RequestShortCutDel",
	0x36: "CannotMoveAnymore",
	0x38: "Say2",
	0x3F: "RequestSkillList",
	0x44: "AnswerTradeRequest",
	0x45: "RequestActionUse",
	0x46: "RequestRestart",
	0x48: "ValidatePosition",
	0x58: "RequestEnchantItem",
	0x59: "RequestDestroyItem",
	0x5E: "RequestFriendInvite",
	0x5F: "RequestAnswerFriendInvite",
	0x60: "RequestFriendList",
	0x61: "RequestFriendDel",
	0x62: "CharacterRestore",
	0x63: "RequestQuestList",
	0x6B: "RequestAcquireSkillInfo",
	0x6C: "RequestAcquireSkill",
	0x6D: "RequestRestartPoint",
	0x73: "RequestPrivateStoreManageSell",
	0x74: "SetPrivateStoreListSell",
	0x76: "RequestPrivateStoreQuitSell",
	0x77: "SetPrivateStoreMsgSell",
	0x79: "RequestPrivateStoreBuy",
	0x90: "RequestPrivateStoreManageBuy",
	0x91: "SetPrivateStoreListBuy",
	0x93: "RequestPrivateStoreQuitBuy",
	0x94: "SetPrivateStoreMsgBuy",
	0x96: "RequestPrivateStoreSell",
	0x72: "RequestCrystallizeItem",
	0xA7: "MultiSellChoose",
	0xAA: "RequestUserCommand",
	0xAC: "RequestRecipeBookOpen",
	0xAD: "RequestRecipeBookDestroy",
	0xAE: "RequestRecipeItemMakeInfo",
	0xAF: "RequestRecipeItemMakeSelf",
	0xBA: "RequestHennaItemList",
	0xBB: "RequestHennaItemInfo",
	0xBC: "RequestHennaEquip",
	0xBD: "RequestHennaUnequipList",
	0xBE: "RequestHennaUnequipInfo",
	0xBF: "RequestHennaUnequip",
	0xCD: "RequestShowMiniMap",
	0xD0: "ExPacket",
}

var clientExNames = map[int]string{
	4: "RequestChangePartyLeader",
	5: "RequestAutoSoulShot",
}

var serverOpcodeNames = map[byte]string{
	0x00: "VersionCheck",
	0x03: "CharInfo",
	0x04: "UserInfo",
	0x05: "Attack",
	0x06: "Die",
	0x0B: "SpawnItem",
	0x0C: "DropItem",
	0x0D: "GetItem",
	0x0E: "StatusUpdate",
	0x12: "DeleteObject",
	0x13: "CharSelectInfo",
	0x14: "AuthLoginFail",
	0x15: "CharSelected",
	0x16: "NpcInfo",
	0x17: "CharTemplates",
	0x19: "CharCreateOk",
	0x1A: "CharCreateFail",
	0x1B: "ItemList",
	0x23: "CharDeleteOk",
	0x24: "CharDeleteFail",
	0x25: "ActionFailed",
	0x26: "ServerClose",
	0x28: "TeleportToLocation",
	0x29: "TargetSelected",
	0x2D: "SocialAction",
	0x1E: "TradeStart",
	0x20: "TradeOwnAdd",
	0x21: "TradeOtherAdd",
	0x22: "SendTradeDone",
	0x2E: "ChangeMoveType",
	0x41: "WarehouseDepositList",
	0x42: "WarehouseWithdrawList",
	0x44: "ShortCutRegister",
	0x45: "ShortCutInit",
	0x5E: "SendTradeRequest",
	0x75: "TradePressOwnOk",
	0x77: "FriendAddRequestResult",
	0x7C: "TradePressOtherOk",
	0x7D: "FriendAddRequest",
	0xFA: "FriendList",
	0xFB: "L2Friend",
	0x47: "StopMove",
	0x48: "MagicSkillUse",
	0x4A: "CreatureSay",
	0x58: "SkillList",
	0x6F: "ChooseInventoryItem",
	0x81: "EnchantResult",
	0xD0: "MultiSellList",
	0xD6: "RecipeBookItemList",
	0xD7: "RecipeItemMakeInfo",
	0x80: "QuestList",
	0x9D: "ShowMiniMap",
	0xE2: "HennaEquipList",
	0xE3: "HennaItemInfo",
	0xE4: "HennaInfo",
	0xE5: "HennaUnequipList",
	0xE6: "HennaItemUnequipInfo",
	0x9A: "PrivateStoreManageListSell",
	0x9B: "PrivateStoreListSell",
	0x9C: "PrivateStoreMsgSell",
	0xB7: "PrivateStoreManageListBuy",
	0xB8: "PrivateStoreListBuy",
	0xB9: "PrivateStoreMsgBuy",
	0x5F: "RestartResponse",
	0x61: "ValidateLocation",
	0x64: "SystemMessage",
	0x7E: "LeaveWorld",
	0xA6: "MyTargetSelected",
	0xC6: "MoveDirection",
}

func hexPreview(b []byte, max int) string {
	if len(b) == 0 {
		return "(empty)"
	}
	if max <= 0 {
		max = 48
	}
	if len(b) <= max {
		return fmt.Sprintf("%x (%dB)", b, len(b))
	}
	return fmt.Sprintf("%x… (%dB)", b[:max], len(b))
}

func (c *GameClient) tag() string {
	addr := ""
	if c.conn != nil {
		addr = hostOnly(c.conn.RemoteAddr())
	}
	who := "-"
	if c.account != "" {
		who = c.account
	}
	if c.player != nil {
		who = fmt.Sprintf("%s/%s#%d", c.account, c.player.Name, c.player.ObjectID)
	}
	return fmt.Sprintf("%s %s [%s]", addr, who, c.state)
}

func (c *GameClient) logRecvEnabled() bool {
	return c.server != nil && (c.server.cfg.PacketHandlerDebug || c.server.cfg.PrintReceivedPackets || c.server.cfg.Developer)
}

func (c *GameClient) logSendEnabled() bool {
	return c.server != nil && (c.server.cfg.PacketHandlerDebug || c.server.cfg.PrintSentPackets || c.server.cfg.Developer)
}

func (c *GameClient) logTraceEnabled() bool {
	return c.logRecvEnabled() || c.logSendEnabled()
}

func (c *GameClient) logRecv(data []byte) {
	if !c.logRecvEnabled() || len(data) == 0 {
		return
	}
	op := data[0]
	name := clientOpcodeName(op, data)
	extra := recvSummary(op, data)
	if extra != "" {
		log.Printf("GS RECV %s %s 0x%02X %s %s", c.tag(), name, op, extra, hexPreview(data, 32))
		return
	}
	log.Printf("GS RECV %s %s 0x%02X %s", c.tag(), name, op, hexPreview(data, 32))
}

func (c *GameClient) logSend(data []byte) {
	if !c.logSendEnabled() || len(data) == 0 {
		return
	}
	op := data[0]
	name := serverOpcodeName(op)
	extra := sendSummary(op, data)
	if extra != "" {
		log.Printf("GS SEND %s %s 0x%02X %s %s", c.tag(), name, op, extra, hexPreview(data, 32))
		return
	}
	log.Printf("GS SEND %s %s 0x%02X %s", c.tag(), name, op, hexPreview(data, 32))
}

func (c *GameClient) logChange(format string, args ...any) {
	if !c.logTraceEnabled() {
		return
	}
	msg := fmt.Sprintf(format, args...)
	log.Printf("GS CHANGE %s %s", c.tag(), msg)
}

func recvSummary(op byte, data []byte) string {
	r := packet.NewReader(data)
	r.SkipOpcode()
	switch op {
	case 0x00:
		return fmt.Sprintf("protocol=%d", r.ReadD())
	case 0x01:
		return fmt.Sprintf("to=(%d,%d,%d)", r.ReadD(), r.ReadD(), r.ReadD())
	case 0x02:
		need := r.ReadC()
		dirY := r.ReadF64()
		dirX := r.ReadF64()
		heading := r.ReadD()
		vert := r.ReadF64()
		x, y, z := r.ReadD(), r.ReadD(), r.ReadD()
		ts := r.ReadQ()
		return fmt.Sprintf("need=%d dir=(%.3f,%.3f) vert=%.3f heading=%d pos=(%d,%d,%d) ts=%d", need, dirX, dirY, vert, heading, x, y, z, ts)
	case 0x04, 0x05, 0x0A:
		return fmt.Sprintf("target=%d", r.ReadD())
	case 0x08:
		return fmt.Sprintf("login=%q", strings.ToLower(r.ReadS()))
	case 0x0B:
		name := r.ReadS()
		race, sex, classID := r.ReadD(), r.ReadD(), r.ReadD()
		return fmt.Sprintf("name=%q race=%d sex=%d class=%d", name, race, sex, classID)
	case 0x0C, 0x0D, 0x62:
		return fmt.Sprintf("slot=%d", r.ReadD())
	case 0x2F:
		return fmt.Sprintf("skill=%d", r.ReadD())
	case 0x38:
		text := r.ReadS()
		sayType := r.ReadD()
		return fmt.Sprintf("type=%d text=%q", sayType, text)
	case 0x45:
		return fmt.Sprintf("action=%d", r.ReadD())
	case 0x48:
		x, y, z, h := r.ReadD(), r.ReadD(), r.ReadD(), r.ReadD()
		return fmt.Sprintf("pos=(%d,%d,%d) heading=%d", x, y, z, h)
	default:
		return ""
	}
}

func sendSummary(op byte, data []byte) string {
	r := packet.NewReader(data)
	r.SkipOpcode()
	switch op {
	case 0x00:
		ok := r.ReadC()
		return fmt.Sprintf("ok=%d", ok)
	case 0x03:
		x, y, z := r.ReadD(), r.ReadD(), r.ReadD()
		_ = r.ReadD()
		oid := r.ReadD()
		name := r.ReadS()
		return fmt.Sprintf("oid=%d name=%q pos=(%d,%d,%d)", oid, name, x, y, z)
	case 0x04:
		x, y, z := r.ReadD(), r.ReadD(), r.ReadD()
		_ = r.ReadD()
		oid := r.ReadD()
		name := r.ReadS()
		return fmt.Sprintf("oid=%d name=%q pos=(%d,%d,%d)", oid, name, x, y, z)
	case 0x05:
		return fmt.Sprintf("attacker=%d target=%d dmg=%d", r.ReadD(), r.ReadD(), r.ReadD())
	case 0x13:
		return fmt.Sprintf("chars=%d", r.ReadD())
	case 0x15:
		name := r.ReadS()
		oid := r.ReadD()
		return fmt.Sprintf("name=%q oid=%d", name, oid)
	case 0x16:
		oid := r.ReadD()
		npcID := r.ReadD()
		_ = r.ReadD()
		x, y, z := r.ReadD(), r.ReadD(), r.ReadD()
		return fmt.Sprintf("oid=%d npc=%d pos=(%d,%d,%d)", oid, npcID, x, y, z)
	case 0x1B:
		show, n := r.ReadH(), r.ReadH()
		return fmt.Sprintf("show=%d items=%d", show, n)
	case 0x29:
		return fmt.Sprintf("oid=%d target=%d pos=(%d,%d,%d)", r.ReadD(), r.ReadD(), r.ReadD(), r.ReadD(), r.ReadD())
	case 0x47:
		return fmt.Sprintf("oid=%d pos=(%d,%d,%d) heading=%d", r.ReadD(), r.ReadD(), r.ReadD(), r.ReadD(), r.ReadD())
	case 0x48:
		return fmt.Sprintf("caster=%d target=%d skill=%d lvl=%d", r.ReadD(), r.ReadD(), r.ReadD(), r.ReadD())
	case 0x4A:
		oid := r.ReadD()
		sayType := r.ReadD()
		name := r.ReadS()
		text := r.ReadS()
		return fmt.Sprintf("oid=%d type=%d name=%q text=%q", oid, sayType, name, text)
	case 0x58:
		return fmt.Sprintf("skills=%d", r.ReadD())
	case 0x61:
		return fmt.Sprintf("oid=%d pos=(%d,%d,%d) heading=%d", r.ReadD(), r.ReadD(), r.ReadD(), r.ReadD(), r.ReadD())
	case 0xA6:
		return fmt.Sprintf("target=%d", r.ReadD())
	case 0xC6:
		oid := r.ReadD()
		dirY, dirX, vert := r.ReadD(), r.ReadD(), r.ReadD()
		x, y, z := r.ReadD(), r.ReadD(), r.ReadD()
		ts := r.ReadQ()
		return fmt.Sprintf("oid=%d dir=(%d,%d) vert=%d pos=(%d,%d,%d) ts=%d", oid, dirX, dirY, vert, x, y, z, ts)
	default:
		return ""
	}
}
