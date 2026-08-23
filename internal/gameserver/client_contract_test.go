package gameserver

import (
	"testing"

	"github.com/pblaravel/game-server-l2-go/internal/clientapi"
	"github.com/pblaravel/game-server-l2-go/internal/config"
	"github.com/pblaravel/game-server-l2-go/internal/packet"
)

func hasOp(ops []byte, want byte) bool {
	return clientapi.ContainsOpcode(ops, want)
}

func TestClientCatalogCoversUnityOutgoing(t *testing.T) {
	if n := len(clientapi.GamePairs()); n < 40 {
		t.Fatalf("catalog too small: %d", n)
	}
}

func TestProtocol746SendsInterludeKey(t *testing.T) {
	addr, stop := startHandshakeServer(t, config.DefaultGameConfig())
	defer stop()
	body := sendProtocol(t, addr, 746)
	if body[0] != 0x00 || body[1] != 0x01 {
		t.Fatalf("InterludeKey/VersionCheck %x", body)
	}
	if len(body) < 14 {
		t.Fatalf("key+flags missing: %d", len(body))
	}
}

func TestNewCharacterReturnsCharTemplates(t *testing.T) {
	srv := combatTestServer(t)
	p := DefaultCharacter("acc", "Tmpl", 0, 0, 0, 1, 1, 1, 268450001, nil)
	c, stop := testClient(t, srv, p)
	defer stop()
	c.SetState(StateAuthed)
	c.SetPlayer(nil)
	c.RecordSends()
	srv.handle(c, []byte{0x0E})
	if !hasOp(c.SentOpcodes(), 0x17) {
		t.Fatalf("NewCharacter must return CharTemplates 0x17, got %x", c.SentOpcodes())
	}
}

func TestCharacterLifecycleOpcodes(t *testing.T) {
	srv := combatTestServer(t)
	store := srv.store.(*MemoryCharacterStore)
	_ = store
	p := DefaultCharacter("heroacc", "Hero", 0, 0, 0, 1, 1, 1, 268450010, nil)
	c, stop := testClient(t, srv, p)
	defer stop()
	c.SetAccountName("heroacc")
	c.SetPlayer(nil)
	c.SetState(StateAuthed)
	c.RecordSends()

	w := packet.NewWriter()
	w.WriteC(0x0B)
	w.WriteS("NewHero")
	w.WriteD(0)
	w.WriteD(0)
	w.WriteD(0)
	for i := 0; i < 6; i++ {
		w.WriteD(0)
	}
	w.WriteD(1)
	w.WriteD(1)
	w.WriteD(1)
	srv.handle(c, w.Bytes())
	ops := c.SentOpcodes()
	if !hasOp(ops, 0x19) || !hasOp(ops, 0x13) {
		t.Fatalf("create: want CharCreateOk 0x19 + CharSelectInfo 0x13, got %x", ops)
	}

	c.RecordSends()
	w = packet.NewWriter()
	w.WriteC(0x0C)
	w.WriteD(0)
	srv.handle(c, w.Bytes())
	if !hasOp(c.SentOpcodes(), 0x13) {
		t.Fatalf("delete: want CharSelectInfo 0x13, got %x", c.SentOpcodes())
	}

	c.RecordSends()
	w = packet.NewWriter()
	w.WriteC(0x0B)
	w.WriteS("NewHero")
	w.WriteD(0)
	w.WriteD(0)
	w.WriteD(0)
	for i := 0; i < 6; i++ {
		w.WriteD(0)
	}
	w.WriteD(1)
	w.WriteD(1)
	w.WriteD(1)
	srv.handle(c, w.Bytes())
	if !hasOp(c.SentOpcodes(), 0x19) {
		t.Fatalf("recreate: want CharCreateOk 0x19, got %x", c.SentOpcodes())
	}

	c.RecordSends()
	w = packet.NewWriter()
	w.WriteC(0x0D)
	w.WriteD(0)
	srv.handle(c, w.Bytes())
	if !hasOp(c.SentOpcodes(), 0x15) {
		t.Fatalf("select: want CharSelected 0x15, got %x", c.SentOpcodes())
	}
	if c.State() != StateEntering {
		t.Fatalf("state %s", c.State())
	}

	c.RecordSends()
	srv.handle(c, []byte{0x03})
	ops = c.SentOpcodes()
	for _, want := range []byte{0x04, 0x1B, 0x58, 0x45, 0x80, 0xE4, 0xE7, 0xFA, 0xF3, 0x7F} {
		if !hasOp(ops, want) {
			t.Fatalf("EnterWorld missing 0x%02x in %x", want, ops)
		}
	}
}

func TestInGameClientPacketsMatchUnityContract(t *testing.T) {
	srv := combatTestServer(t)
	p := DefaultCharacter("acc", "Unity", 0, 0, 0, 1, 1, 1, 268450020, nil)
	c, stop := testClient(t, srv, p)
	defer stop()
	c.RecordSends()

	npc := &NPC{ObjectID: srv.world.NextID(), NPCID: 30001, Name: "Guard",
		X: p.X, Y: p.Y, Z: p.Z, Level: 1, MaxHP: 100, CurHP: 100}
	npc.NpcDefaults()
	srv.world.AddNPC(npc)

	if rec := GetRecipe(1); rec != nil {
		p.Recipes = []int32{rec.ID}
	}
	c.activeEnchant = 1

	send := func(name string, body []byte, want ...byte) {
		t.Helper()
		c.RecordSends()
		srv.handle(c, body)
		ops := c.SentOpcodes()
		for _, w := range want {
			if !hasOp(ops, w) {
				t.Fatalf("%s: want 0x%02x, got %x", name, w, ops)
			}
		}
	}

	w := packet.NewWriter()
	w.WriteC(0x01)
	w.WriteD(p.X + 10)
	w.WriteD(p.Y)
	w.WriteD(p.Z)
	w.WriteD(p.X)
	w.WriteD(p.Y)
	w.WriteD(p.Z)
	w.WriteD(1)
	send("MoveBackwardToLocation", w.Bytes(), 0x01)

	w = packet.NewWriter()
	w.WriteC(0x30)
	send("Appearing", w.Bytes(), 0x04)

	w = packet.NewWriter()
	w.WriteC(0x04)
	w.WriteD(npc.ObjectID)
	w.WriteD(p.X)
	w.WriteD(p.Y)
	w.WriteD(p.Z)
	w.WriteC(0)
	send("ClickAction", w.Bytes(), 0xA6)

	w = packet.NewWriter()
	w.WriteC(0x37)
	w.WriteH(0)
	send("RequestTargetCanceld", w.Bytes(), 0x2A)

	w = packet.NewWriter()
	w.WriteC(0x2F)
	w.WriteD(99999)
	w.WriteD(0)
	w.WriteC(0)
	send("RequestMagicSkillUse", w.Bytes(), 0x25)

	w = packet.NewWriter()
	w.WriteC(0x38)
	w.WriteS("hello")
	w.WriteD(0)
	send("RequestSay2", w.Bytes(), 0x4A)

	w = packet.NewWriter()
	w.WriteC(0x57)
	send("RequestShowBoard", w.Bytes(), 0x6E)

	w = packet.NewWriter()
	w.WriteC(0xAA)
	w.WriteD(0)
	send("RequestUserCommand /loc", w.Bytes(), 0x64)

	w = packet.NewWriter()
	w.WriteC(0x9D)
	send("RequestSkillCoolTime", w.Bytes(), 0xC1)

	w = packet.NewWriter()
	w.WriteC(0x64)
	w.WriteD(1)
	send("RequestQuestAbort", w.Bytes(), 0x80)

	w = packet.NewWriter()
	w.WriteC(0x66)
	w.WriteD(0)
	send("RequestPledgeInfo", w.Bytes(), 0x83)

	w = packet.NewWriter()
	w.WriteC(0x24)
	w.WriteD(1)
	send("RequestJoinPledge", w.Bytes(), 0x25)

	w = packet.NewWriter()
	w.WriteC(0x26)
	send("RequestWithdrawPledge", w.Bytes(), 0x04)

	w = packet.NewWriter()
	w.WriteC(0x27)
	w.WriteS("Nobody")
	send("RequestOustPledgeMember", w.Bytes(), 0x25)

	w = packet.NewWriter()
	w.WriteC(0x55)
	w.WriteS("Unity")
	w.WriteS("Title")
	send("RequestGiveNickName", w.Bytes(), 0x04)

	w = packet.NewWriter()
	w.WriteC(0xC0)
	w.WriteD(1)
	w.WriteD(0)
	w.WriteD(0)
	send("RequestPledgePower", w.Bytes(), 0x30)

	w = packet.NewWriter()
	w.WriteC(0x9E)
	w.WriteD(p.ObjectID)
	send("RequestPackageSendableItemList", w.Bytes(), 0xC3)

	w = packet.NewWriter()
	w.WriteC(0x9F)
	w.WriteD(p.ObjectID)
	send("RequestPackageSend", w.Bytes(), 0x25)

	w = packet.NewWriter()
	w.WriteC(0xC6)
	w.WriteD(0)
	w.WriteD(1)
	send("RequestPreviewItem", w.Bytes(), 0xF0)

	w = packet.NewWriter()
	w.WriteC(0x33)
	w.WriteD(int32(ShortcutItem))
	w.WriteD(0)
	w.WriteD(57)
	w.WriteD(1)
	send("RequestShortCutReg", w.Bytes(), 0x44)

	w = packet.NewWriter()
	w.WriteC(0x35)
	w.WriteD(0)
	send("RequestShortCutDel", w.Bytes(), 0x46)

	w = packet.NewWriter()
	w.WriteC(0xAC)
	w.WriteD(0)
	send("RequestRecipeBookOpen", w.Bytes(), 0xD6)

	if len(p.Recipes) > 0 {
		w = packet.NewWriter()
		w.WriteC(0xAE)
		w.WriteD(p.Recipes[0])
		send("RequestRecipeItemMakeInfo", w.Bytes(), 0xD7)

		w = packet.NewWriter()
		w.WriteC(0xAD)
		w.WriteD(p.Recipes[0])
		send("RequestRecipeBookDestroy", w.Bytes(), 0xD6)
	}

	p.Dead = true
	w = packet.NewWriter()
	w.WriteC(0x6D)
	w.WriteD(0)
	c.RecordSends()
	srv.handle(c, w.Bytes())
	if !hasOp(c.SentOpcodes(), 0x28) && !hasOp(c.SentOpcodes(), 0x07) {
		t.Fatalf("RequestRestartPoint: want Teleport 0x28 or Revive 0x07, got %x", c.SentOpcodes())
	}
}

func TestEnterWorldKnownlistOpcodes(t *testing.T) {
	srv := combatTestServer(t)
	other := DefaultCharacter("o", "Other", 0, 0, 0, 1, 1, 1, 268450030, nil)
	other.Online = true
	srv.world.AddPlayer(other)

	p := DefaultCharacter("acc", "Self", 0, 0, 0, 1, 1, 1, 268450031, nil)
	c, stop := testClient(t, srv, p)
	defer stop()
	c.SetState(StateEntering)
	c.RecordSends()
	srv.handle(c, []byte{0x03})
	if !hasOp(c.SentOpcodes(), 0x03) {
		t.Fatalf("EnterWorld should push CharInfo 0x03 for other players, got %x", c.SentOpcodes())
	}
}
