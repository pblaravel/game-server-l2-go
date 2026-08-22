package gameserver

import (
	"strings"
	"testing"
)

// Each expected sequence below is the writeC/H/D/Q/F/S order of the matching Java
// class in reference/l2-unity-gameserver. walkLayout consumes the produced packet
// with that sequence; a packet whose layout drifted either runs out of bytes or
// leaves bytes behind.
//
// C = writeC, H = writeH, D = writeD, Q = writeQ, F = writeF (double), S = writeS.

func walkLayout(t *testing.T, name string, data []byte, sequence string) {
	t.Helper()
	pos := 0
	need := func(n int, i int, code byte) bool {
		if pos+n > len(data) {
			t.Errorf("%s: field %d (%c) needs %d bytes, only %d left (packet %d bytes)",
				name, i, code, n, len(data)-pos, len(data))
			return false
		}
		pos += n
		return true
	}
	for i := 0; i < len(sequence); i++ {
		switch code := sequence[i]; code {
		case 'C':
			if !need(1, i, code) {
				return
			}
		case 'H':
			if !need(2, i, code) {
				return
			}
		case 'D':
			if !need(4, i, code) {
				return
			}
		case 'Q', 'F':
			if !need(8, i, code) {
				return
			}
		case 'S':
			end := pos
			for end+1 < len(data) && (data[end] != 0 || data[end+1] != 0) {
				end += 2
			}
			pos = end + 2
			if pos > len(data) {
				t.Errorf("%s: field %d (S) is not terminated", name, i)
				return
			}
		default:
			t.Fatalf("%s: bad sequence code %q", name, string(code))
		}
	}
	if pos != len(data) {
		t.Errorf("%s: layout consumed %d of %d bytes (%d left over)", name, pos, len(data), len(data)-pos)
	}
}

func rep(code string, n int) string { return strings.Repeat(code, n) }

func layoutTestPlayer() *Character {
	p := DefaultCharacter("acc", "Tester", 0, 0, 0, 1, 1, 1, 268437456, nil)
	p.Title = "hello"
	p.Level = 10
	p.Exp = 5000
	RecalcStats(p)
	return p
}

func layoutTestNPC() *NPC {
	n := &NPC{ObjectID: 500000001, NPCID: 20120, Name: "Wolf", Title: "Lv 4",
		X: 1, Y: 2, Z: 3, Level: 4, MaxHP: 80, CurHP: 80, IsAttackable: true}
	n.NpcDefaults()
	return n
}

func TestCharInfoLayoutMatchesJava(t *testing.T) {
	// serverpackets/auth/CharInfo.java
	seq := "C" + "DDD" + "D" + "D" + "S" + "DDD" +
		rep("D", 12) + rep("H", 4) + "D" + rep("H", 12) + "D" + rep("H", 4) +
		rep("D", 6) + rep("D", 8) + "FF" + "FF" + "DDD" + "S" + rep("D", 4) + "D" +
		rep("C", 5) + "CC" + "H" + "C" + "D" + "C" + "H" + "D" + "DD" + "CC" + "D" +
		"CCC" + "DDD" + "D" + "D" + "DD" + "D" + "D" + "D"
	walkLayout(t, "CharInfo", CharInfo(layoutTestPlayer()), seq)
}

func TestNpcInfoLayoutMatchesJava(t *testing.T) {
	// serverpackets/actor/AbstractNpcInfo.NpcInfo
	seq := "C" + "DDD" + "DDDD" + "D" + "DD" + rep("D", 8) + "FF" + "FF" + "DDD" +
		rep("C", 5) + "SS" + "DDD" + "D" + rep("D", 4) + "CC" + "FF" + "DD" + "DDD"
	walkLayout(t, "NpcInfo", NpcInfo(layoutTestNPC()), seq)
}

func TestUserInfoLayoutMatchesJava(t *testing.T) {
	// serverpackets/actor/UserInfo.java
	seq := "C" + "DDD" + "D" + "D" + "S" + "DDD" + "D" + "Q" + "F" + rep("D", 6) +
		rep("D", 4) + "D" + "DDD" + rep("D", 17) + rep("D", 17) +
		rep("H", 14) + "D" + rep("H", 12) + "D" + rep("H", 4) +
		rep("D", 10) + "DD" + rep("D", 8) + "FF" + "FF" + "DDD" + "D" + "S" +
		rep("D", 4) + "D" + "CCC" + "DD" + "H" + "C" + "D" + "C" + "D" + "HH" + "D" +
		"H" + "D" + "D" + "DD" + "CC" + "D" + "CCC" + "DDD" + "D" + "C" + "DD" + "D" + "D" + "D"
	walkLayout(t, "UserInfo", UserInfo(layoutTestPlayer()), seq)
}

func TestCharSelectInfoLayoutMatchesJava(t *testing.T) {
	// serverpackets/auth/CharSelectInfo.java, one slot
	slot := "S" + "D" + "S" + "D" + "D" + "D" + "DDD" + "D" + "DDD" + "FF" + "D" + "Q" +
		"F" + "D" + "DDD" + rep("D", 7) + rep("D", 17) + rep("D", 17) + "DDD" + "FF" +
		"D" + "D" + "D" + "C" + "D"
	seq := "C" + "D" + slot
	walkLayout(t, "CharSelectInfo", CharSelectInfo("acc", 1, []*Character{layoutTestPlayer()}), seq)
}

func TestCharSelectedLayoutMatchesJava(t *testing.T) {
	data := CharSelected(layoutTestPlayer(), 1, 100)
	if len(data) == 0 {
		t.Fatal("CharSelected produced no bytes")
	}
	if data[0] != 0x15 {
		t.Errorf("CharSelected opcode = 0x%02X, want 0x15", data[0])
	}
}

func TestItemListLayoutMatchesJava(t *testing.T) {
	// serverpackets/item/ItemList.java with one item
	p := layoutTestPlayer()
	one := []Item{p.Items[0]}
	seq := "C" + "HH" + "H" + "DDD" + "HHH" + "D" + "HH" + "D" + "D" + "D"
	walkLayout(t, "ItemList", ItemList(one, true), seq)
}

func TestSkillListLayoutMatchesJava(t *testing.T) {
	seq := "C" + "D" + "DDD" + "C"
	walkLayout(t, "SkillList", SkillList([]Skill{{ID: 3, Level: 1}}), seq)
}

func TestMovementAndCombatLayoutsMatchJava(t *testing.T) {
	walkLayout(t, "MoveToLocation", MoveToLocation(1, 2, 3, 4, 5, 6, 7), "C"+"D"+"DDD"+"DDD")
	walkLayout(t, "ChangeWaitType", ChangeWaitType(1, WaitTypeSitting, 2, 3, 4), "C"+"D"+"D"+"DDD")
	walkLayout(t, "StopMove", StopMove(1, 2, 3, 4, 5), "C"+"D"+"DDD"+"D")
	walkLayout(t, "Attack", Attack(1, 2, 3, 0, 4, 5, 6), "C"+"D"+"D"+"D"+"C"+"DDD"+"H")
	walkLayout(t, "Die", Die(1, false, false, false, false, false), "C"+"D"+rep("D", 6))
	walkLayout(t, "Revive", Revive(1), "CD")
	walkLayout(t, "StatusUpdate", StatusUpdate(1, [][2]int32{{StatusCurHP, 10}}), "C"+"D"+"D"+"DD")
	walkLayout(t, "ChangeMoveType", ChangeMoveType(1, true), "C"+"D"+"D"+"D")
	walkLayout(t, "SocialAction", SocialAction(1, 2), "C"+"D"+"D")
	walkLayout(t, "TeleportToLocation", TeleportToLocation(1, 2, 3, 4), "C"+"D"+"DDD")
}

func TestSkillPacketLayoutsMatchJava(t *testing.T) {
	walkLayout(t, "MagicSkillUse", MagicSkillUse(1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, true),
		"C"+"DD"+"DD"+"DD"+"DDD"+"D"+"H"+"DDD")
	walkLayout(t, "MagicSkillUse-fail", MagicSkillUse(1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, false),
		"C"+"DD"+"DD"+"DD"+"DDD"+"D"+"DDD")
	walkLayout(t, "MagicSkillLaunched", MagicSkillLaunched(1, 2, 3, []int32{4, 5}),
		"C"+"D"+"DD"+"D"+"DD")
	walkLayout(t, "MagicSkillLaunched-empty", MagicSkillLaunched(1, 2, 3, nil), "C"+"D"+"DD"+"DD")
	walkLayout(t, "MagicSkillCanceled", MagicSkillCanceled(1), "CD")
	walkLayout(t, "SetupGauge", SetupGauge(GaugeBlue, 100, 100), "C"+"DDD")
	walkLayout(t, "AcquireSkillList", AcquireSkillList(AcquireUsual, []ClassSkillNode{{ID: 3, Level: 1, Cost: 50}}),
		"C"+"D"+"D"+rep("D", 5))
	walkLayout(t, "AcquireSkillInfo", AcquireSkillInfo(3, 1, 50, 0, []SkillRequirement{{1, 57, 100, 0}}),
		"C"+rep("D", 4)+"D"+rep("D", 4))
	walkLayout(t, "AcquireSkillDone", AcquireSkillDone(), "C")
}

func TestShortcutLayoutsMatchJava(t *testing.T) {
	action := Shortcut{Slot: 0, Type: ShortcutAction, ID: 2, CharacterType: 1}
	item := Shortcut{Slot: 1, Type: ShortcutItem, ID: 10, CharacterType: 1}
	skill := Shortcut{Slot: 2, Type: ShortcutSkill, ID: 3, Level: 1, CharacterType: 1}
	walkLayout(t, "ShortCutInit", ShortCutInit([]Shortcut{action, item, skill}),
		"C"+"D"+ /*action*/ "DD"+"DD"+ /*item*/ "DD"+rep("D", 6)+ /*skill*/ "DD"+"DD"+"C"+"D")
	walkLayout(t, "ShortCutRegister-item", ShortCutRegister(item), "C"+"DD"+rep("D", 6))
	walkLayout(t, "ShortCutRegister-skill", ShortCutRegister(skill), "C"+"DD"+"DD"+"C"+"D")
}

func TestSystemMessageLayoutMatchesJava(t *testing.T) {
	walkLayout(t, "SystemMessage-plain", SystemMessage(SMLevelIncreased), "C"+"D"+"D")
	walkLayout(t, "SystemMessage-params",
		SystemMessage(SMEarnedExpAndSP, SysNumber(10), SysNumber(2)),
		"C"+"D"+"D"+"DD"+"DD")
	walkLayout(t, "SystemMessage-skill", SystemMessage(SMLearnedSkill, SysSkill(3, 1)),
		"C"+"D"+"D"+"D"+"DD")
	walkLayout(t, "SystemMessage-text", SystemMessage(SMLearnedSkill, SysText("abc")),
		"C"+"D"+"D"+"D"+"S")
}

func TestShopAndPartyLayoutsMatchJava(t *testing.T) {
	list := NpcBuyList{ID: 1, Items: []BuyProduct{{ItemID: 1, Price: 100}}}
	walkLayout(t, "BuyList", BuyList(list, 50, true),
		"C"+"C"+"D"+"D"+"H"+"H"+"DD"+"D"+"H"+"H"+"D"+"HHH"+"D")
	walkLayout(t, "SellList", SellList(50, []Item{{ObjectID: 1, ItemID: 2, Count: 1}}, true),
		"C"+"C"+"D"+"D"+"H"+"H"+"DD"+"D"+"HH"+"D"+"HHH"+"D")
	walkLayout(t, "NpcHtmlMessage", NpcHtmlMessage(1, "<html></html>"), "C"+"D"+"S"+"D")
	walkLayout(t, "AskJoinParty", AskJoinParty("A", 0), "C"+"S"+"D")
	walkLayout(t, "JoinParty", JoinParty(1), "C"+"D")
	p := &Character{ObjectID: 2, Name: "B", Level: 1}
	walkLayout(t, "PartySmallWindowAdd", PartySmallWindowAdd(p, 1, 0),
		"C"+"D"+"D"+"D"+"S"+rep("D", 10))
	walkLayout(t, "PartySmallWindowDelete", PartySmallWindowDelete(p), "C"+"D"+"S")
	walkLayout(t, "PartySmallWindowDeleteAll", PartySmallWindowDeleteAll(), "C")
}

func TestTradeWarehouseFriendLayoutsMatchJava(t *testing.T) {
	it := Item{ObjectID: 1, ItemID: 57, Count: 10, Type1: 4, Type2: 4}
	walkLayout(t, "SendTradeRequest", SendTradeRequest(7), "C"+"D")
	walkLayout(t, "TradeStart", TradeStart(7, []Item{it}),
		"C"+"D"+"H"+"H"+"DD"+"D"+"HH"+"D"+"HHH")
	walkLayout(t, "TradeOwnAdd", TradeOwnAdd(it, 3),
		"C"+"H"+"H"+"DD"+"D"+"HH"+"D"+"HHH")
	walkLayout(t, "TradeOtherAdd", TradeOtherAdd(it, 3),
		"C"+"H"+"H"+"DD"+"D"+"HH"+"D"+"HHH")
	walkLayout(t, "SendTradeDone", SendTradeDone(true), "C"+"D")
	walkLayout(t, "TradePressOwnOk", TradePressOwnOk(), "C")
	walkLayout(t, "TradePressOtherOk", TradePressOtherOk(), "C")
	walkLayout(t, "WarehouseDepositList", WarehouseDepositList(50, []Item{it}),
		"C"+"H"+"D"+"H"+"H"+"DD"+"D"+"HH"+"D"+"HHH"+"D"+"Q")
	walkLayout(t, "WarehouseWithdrawList", WarehouseWithdrawList(50, []Item{it}),
		"C"+"H"+"D"+"H"+"H"+"DD"+"D"+"HH"+"D"+"HHH"+"D"+"Q")
	walkLayout(t, "FriendAddRequest", FriendAddRequest("A"), "C"+"S")
	walkLayout(t, "FriendAddRequestResult", FriendAddRequestResult(true), "C"+"D")
	walkLayout(t, "FriendList", FriendList([]Friend{{ObjectID: 1, Name: "B"}}, map[int32]bool{1: true}),
		"C"+"D"+"D"+"S"+"DD")
	walkLayout(t, "L2Friend", L2Friend(1, "B", 1, true), "C"+"DD"+"S"+"DD")
}

func TestItemSystemLayoutsMatchJava(t *testing.T) {
	g := &GroundItem{ObjectID: 9, ItemID: 57, Count: 10, X: 1, Y: 2, Z: 3, Dropper: 7}
	walkLayout(t, "DropItem", DropItem(g), "C"+rep("D", 9))
	walkLayout(t, "SpawnItem", SpawnItem(g), "C"+rep("D", 8))
	walkLayout(t, "GetItem", GetItemPacket(7, g), "C"+rep("D", 5))
	walkLayout(t, "EnchantResult", EnchantResult(0), "CD")
	walkLayout(t, "ChooseInventoryItem", ChooseInventoryItem(955), "CD")
	p := layoutTestPlayer()
	walkLayout(t, "RecipeBookItemList-empty", RecipeBookItemList(p, true), "C"+rep("D", 3))
	p.Recipes = []int32{1}
	walkLayout(t, "RecipeBookItemList", RecipeBookItemList(p, true), "C"+rep("D", 3)+"DD")
	if pkt := RecipeItemMakeInfo(1, p, -1); pkt != nil {
		walkLayout(t, "RecipeItemMakeInfo", pkt, "C"+rep("D", 5))
	}
	walkLayout(t, "HennaInfo-empty", HennaInfo(p), "C"+rep("C", 6)+"DD")
	walkLayout(t, "HennaEquipList-empty", HennaEquipList(p), "C"+rep("D", 3))
	tiny := &MultisellList{ID: 3, Entries: []MultisellEntry{{
		Products:    []MultisellIngredient{{ItemID: 152, Count: 1}},
		Ingredients: []MultisellIngredient{{ItemID: 4, Count: 1}, {ItemID: 57, Count: 10}},
	}}}
	walkLayout(t, "MultiSellList", MultiSellList(tiny, 0),
		"C"+rep("D", 5)+rep("D", 3)+"C"+"HH"+"HDHDHDD"+rep("HHDHDD", 2))
}
