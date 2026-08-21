package gameserver

import (
	"time"

	"github.com/pblaravel/game-server-l2-go/internal/packet"
)

func gsWrite(write func(w *packet.Writer)) []byte {
	w := packet.NewWriter()
	write(w)
	return w.Bytes()
}

func VersionCheck(key8 []byte, useCipher bool) []byte {
	return VersionCheckReply(key8, useCipher, true)
}

func VersionCheckReply(key8 []byte, useCipher bool, ok bool) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x00)
		if ok {
			w.WriteC(0x01)
		} else {
			w.WriteC(0x00)
		}
		w.WriteB(key8)
		if useCipher {
			w.WriteD(1)
		} else {
			w.WriteD(0)
		}
		w.WriteD(0x01)
	})
}

func AuthLoginFail(reason int32) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x14)
		w.WriteD(reason)
	})
}

func CharCreateOk() []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x19)
		w.WriteD(1)
	})
}

func CharCreateFail(reason int32) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x1A)
		w.WriteD(reason)
	})
}

func CharDeleteOk() []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x23)
		w.WriteD(1)
	})
}

func CharDeleteFail(reason int32) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x24)
		w.WriteD(reason)
	})
}

func ActionFailed() []byte {
	return gsWrite(func(w *packet.Writer) { w.WriteC(0x25) })
}

func ServerClose() []byte {
	return gsWrite(func(w *packet.Writer) { w.WriteC(0x26) })
}

func LeaveWorld() []byte {
	return gsWrite(func(w *packet.Writer) { w.WriteC(0x7E) })
}

func CharSelectInfo(login string, sessionID int32, slots []*Character) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x13)
		w.WriteD(int32(len(slots)))
		active := 0
		var last int64
		for i, s := range slots {
			if s.LastAccess > last {
				last = s.LastAccess
				active = i
			}
		}
		for i, s := range slots {
			w.WriteS(s.Name)
			w.WriteD(s.ObjectID)
			w.WriteS(login)
			w.WriteD(sessionID)
			w.WriteD(s.ClanID)
			w.WriteD(0)
			w.WriteD(s.Sex)
			w.WriteD(s.Race)
			w.WriteD(s.BaseClass)
			w.WriteD(1)
			w.WriteD(s.X)
			w.WriteD(s.Y)
			w.WriteD(s.Z)
			w.WriteF64(s.CurHP)
			w.WriteF64(s.CurMP)
			w.WriteD(s.SP)
			w.WriteQ(s.Exp)
			w.WriteF64(ExpPercent(int(s.Level), s.Exp))
			w.WriteD(s.Level)
			w.WriteD(s.Karma)
			w.WriteD(s.PKKills)
			w.WriteD(s.PvPKills)
			for j := 0; j < 7; j++ {
				w.WriteD(0)
			}
			writePaperdoll17(w, s.PaperdollObj)
			writePaperdoll17(w, s.PaperdollItem)
			w.WriteD(s.HairStyle)
			w.WriteD(s.HairColor)
			w.WriteD(s.Face)
			w.WriteF64(float64(s.MaxHP))
			w.WriteF64(float64(s.MaxMP))
			w.WriteD(deleteTimerSeconds(s))
			w.WriteD(s.ClassID)
			writeBool(w, i == active, 1, 0)
			w.WriteC(int(min32(127, s.EnchantEffect)))
			w.WriteD(s.AugmentRHand)
		}
	})
}

func min32(a, b int32) int32 {
	if a < b {
		return a
	}
	return b
}

// deleteTimerSeconds is Java CharSelectInfo: -1 for a banned slot, otherwise the
// remaining seconds of a scheduled deletion.
func deleteTimerSeconds(s *Character) int32 {
	if s.AccessLevel <= -1 {
		return -1
	}
	if s.DeleteTime <= 0 {
		return 0
	}
	left := (s.DeleteTime - time.Now().UnixMilli()) / 1000
	if left < 0 {
		return 0
	}
	return int32(left)
}

func writePaperdoll17(w *packet.Writer, slots [PaperCount]int32) {
	order := []Paperdoll{
		PaperHairAll, PaperRear, PaperLear, PaperNeck, PaperRFinger, PaperLFinger,
		PaperHead, PaperRHand, PaperLHand, PaperGloves, PaperChest, PaperLegs,
		PaperFeet, PaperCloak, PaperRHand, PaperHair, PaperFace,
	}
	for _, p := range order {
		w.WriteD(slots[p])
	}
}

func CharSelected(p *Character, sessionID int32, gameTime int32) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x15)
		w.WriteS(p.Name)
		w.WriteD(p.ObjectID)
		w.WriteS(p.Title)
		w.WriteD(sessionID)
		w.WriteD(p.ClanID)
		w.WriteD(0)
		w.WriteD(p.Sex)
		w.WriteD(p.Race)
		w.WriteD(p.ClassID)
		w.WriteD(1)
		w.WriteD(p.X)
		w.WriteD(p.Y)
		w.WriteD(p.Z)
		w.WriteF64(p.CurHP)
		w.WriteF64(p.CurMP)
		w.WriteD(p.SP)
		w.WriteQ(p.Exp)
		w.WriteD(p.Level)
		w.WriteD(p.Karma)
		w.WriteD(p.PKKills)
		w.WriteD(p.INT)
		w.WriteD(p.STR)
		w.WriteD(p.CON)
		w.WriteD(p.MEN)
		w.WriteD(p.DEX)
		w.WriteD(p.WIT)
		for i := 0; i < 30; i++ {
			w.WriteD(0)
		}
		w.WriteD(0)
		w.WriteD(0)
		w.WriteD(gameTime)
		w.WriteD(0)
		w.WriteD(p.ClassID)
		w.WriteD(0)
		w.WriteD(0)
		w.WriteD(0)
		w.WriteD(0)
	})
}

func UserInfo(p *Character) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x04)
		w.WriteD(p.X)
		w.WriteD(p.Y)
		w.WriteD(p.Z)
		w.WriteD(p.Heading)
		w.WriteD(p.ObjectID)
		w.WriteS(p.Name)
		w.WriteD(p.Race)
		w.WriteD(p.Sex)
		w.WriteD(p.BaseClass)
		w.WriteD(p.Level)
		w.WriteQ(p.Exp)
		w.WriteF64(ExpPercent(int(p.Level), p.Exp))
		w.WriteD(p.STR)
		w.WriteD(p.DEX)
		w.WriteD(p.CON)
		w.WriteD(p.INT)
		w.WriteD(p.WIT)
		w.WriteD(p.MEN)
		w.WriteD(p.MaxHP)
		w.WriteD(int32(p.CurHP))
		w.WriteD(p.MaxMP)
		w.WriteD(int32(p.CurMP))
		w.WriteD(p.SP)
		w.WriteD(p.CurrentWeight)
		w.WriteD(p.WeightLimit)
		if p.PaperdollItem[PaperRHand] != 0 {
			w.WriteD(40)
		} else {
			w.WriteD(20)
		}
		writePaperdoll17(w, p.PaperdollObj)
		writePaperdoll17(w, p.PaperdollItem)
		for i := 0; i < 14; i++ {
			w.WriteH(0)
		}
		w.WriteD(0)
		for i := 0; i < 12; i++ {
			w.WriteH(0)
		}
		w.WriteD(0)
		for i := 0; i < 4; i++ {
			w.WriteH(0)
		}
		w.WriteD(p.PAtk)
		w.WriteD(p.PAtkSpd)
		w.WriteD(p.PDef)
		w.WriteD(p.Evasion)
		w.WriteD(p.Accuracy)
		w.WriteD(p.Crit)
		w.WriteD(p.MAtk)
		w.WriteD(p.MAtkSpd)
		w.WriteD(p.PAtkSpd)
		w.WriteD(p.MDef)
		w.WriteD(p.PvPFlag)
		w.WriteD(p.Karma)
		w.WriteD(p.RunSpeed)
		w.WriteD(p.WalkSpeed)
		w.WriteD(p.SwimSpeed)
		w.WriteD(p.SwimSpeed)
		w.WriteD(0)
		w.WriteD(0)
		if p.Flying {
			w.WriteD(p.RunSpeed)
			w.WriteD(p.WalkSpeed)
		} else {
			w.WriteD(0)
			w.WriteD(0)
		}
		w.WriteF64(p.MoveMultiplier)
		w.WriteF64(p.AttackMultiplier)
		w.WriteF64(p.CollisionRadius)
		w.WriteF64(p.CollisionHeight)
		w.WriteD(p.HairStyle)
		w.WriteD(p.HairColor)
		w.WriteD(p.Face)
		writeBool(w, p.AccessLevel > 0, 1, 0)
		w.WriteS(p.Title)
		w.WriteD(p.ClanID)
		w.WriteD(p.ClanCrestID)
		w.WriteD(p.AllyID)
		w.WriteD(p.AllyCrestID)
		w.WriteD(0) // relation
		w.WriteC(int(p.MountType))
		w.WriteC(int(p.PrivateStore))
		w.WriteC(0) // crystallize
		w.WriteD(p.PKKills)
		w.WriteD(p.PvPKills)
		w.WriteH(len(p.Cubics))
		for _, id := range p.Cubics {
			w.WriteH(int(id))
		}
		writeBoolC(w, p.InPartyMatchRoom, 1, 0)
		w.WriteD(p.AbnormalEffect)
		w.WriteC(0)
		w.WriteD(0) // clan privileges
		w.WriteH(int(p.RecomLeft))
		w.WriteH(int(p.RecomHave))
		w.WriteD(0) // mount npc id
		w.WriteH(int(p.InventoryLimit))
		w.WriteD(p.ClassID)
		w.WriteD(0)
		w.WriteD(p.MaxCP)
		w.WriteD(int32(p.CurCP))
		if p.MountType != 0 {
			w.WriteC(0)
		} else {
			w.WriteC(int(p.EnchantEffect))
		}
		w.WriteC(int(p.Team))
		w.WriteD(p.ClanCrestLargeID)
		writeBoolC(w, p.Nobless, 1, 0)
		writeBoolC(w, p.Hero, 1, 0)
		writeBoolC(w, p.Fishing, 1, 0)
		w.WriteD(p.FishX)
		w.WriteD(p.FishY)
		w.WriteD(p.FishZ)
		w.WriteD(p.NameColor)
		writeBoolC(w, p.Running, 1, 0)
		w.WriteD(p.PledgeClass)
		w.WriteD(p.PledgeType)
		w.WriteD(p.TitleColor)
		w.WriteD(p.CursedWeaponLvl)
		w.WriteD(p.AttackRange)
	})
}

// charInfoPaperdoll is the 12-slot order of Java CharInfo (RHAND twice).
var charInfoPaperdoll = []Paperdoll{
	PaperHairAll, PaperHead, PaperRHand, PaperLHand, PaperGloves, PaperChest,
	PaperLegs, PaperFeet, PaperCloak, PaperRHand, PaperHair, PaperFace,
}

func writeBool(w *packet.Writer, v bool, yes, no int32) {
	if v {
		w.WriteD(yes)
	} else {
		w.WriteD(no)
	}
}

func writeBoolC(w *packet.Writer, v bool, yes, no int) {
	if v {
		w.WriteC(yes)
	} else {
		w.WriteC(no)
	}
}

// CharInfo mirrors Java serverpackets/auth/CharInfo.
func CharInfo(p *Character) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x03)
		w.WriteD(p.X)
		w.WriteD(p.Y)
		w.WriteD(p.Z)
		w.WriteD(0) // boat object id
		w.WriteD(p.ObjectID)
		w.WriteS(p.Name)
		w.WriteD(p.Race)
		w.WriteD(p.Sex)
		w.WriteD(p.BaseClass)
		for _, slot := range charInfoPaperdoll {
			w.WriteD(p.PaperdollItem[slot])
		}
		for i := 0; i < 4; i++ {
			w.WriteH(0)
		}
		w.WriteD(p.AugmentRHand)
		for i := 0; i < 12; i++ {
			w.WriteH(0)
		}
		w.WriteD(p.AugmentLHand)
		for i := 0; i < 4; i++ {
			w.WriteH(0)
		}
		w.WriteD(p.PvPFlag)
		w.WriteD(p.Karma)
		w.WriteD(p.MAtkSpd)
		w.WriteD(p.PAtkSpd)
		w.WriteD(p.PvPFlag)
		w.WriteD(p.Karma)
		w.WriteD(p.RunSpeed)
		w.WriteD(p.WalkSpeed)
		w.WriteD(p.SwimSpeed)
		w.WriteD(p.SwimSpeed)
		w.WriteD(p.RunSpeed)
		w.WriteD(p.WalkSpeed)
		if p.Flying {
			w.WriteD(p.RunSpeed)
			w.WriteD(p.WalkSpeed)
		} else {
			w.WriteD(0)
			w.WriteD(0)
		}
		w.WriteF64(p.MoveMultiplier)
		w.WriteF64(p.AttackMultiplier)
		w.WriteF64(p.CollisionRadius)
		w.WriteF64(p.CollisionHeight)
		w.WriteD(p.HairStyle)
		w.WriteD(p.HairColor)
		w.WriteD(p.Face)
		w.WriteS(p.Title)
		w.WriteD(p.ClanID)
		w.WriteD(p.ClanCrestID)
		w.WriteD(p.AllyID)
		w.WriteD(p.AllyCrestID)
		w.WriteD(0)
		writeBoolC(w, p.Sitting, 0, 1)
		writeBoolC(w, p.Running, 1, 0)
		writeBoolC(w, p.InCombat, 1, 0)
		writeBoolC(w, p.AlikeDead(), 1, 0)
		writeBoolC(w, p.Invisible, 1, 0)
		w.WriteC(int(p.MountType))
		w.WriteC(int(p.PrivateStore))
		w.WriteH(len(p.Cubics))
		for _, id := range p.Cubics {
			w.WriteH(int(id))
		}
		writeBoolC(w, p.InPartyMatchRoom, 1, 0)
		w.WriteD(p.AbnormalEffect)
		w.WriteC(int(p.RecomLeft))
		w.WriteH(int(p.RecomHave))
		w.WriteD(p.ClassID)
		w.WriteD(p.MaxCP)
		w.WriteD(int32(p.CurCP))
		if p.MountType != 0 {
			w.WriteC(0)
		} else {
			w.WriteC(int(p.EnchantEffect))
		}
		w.WriteC(int(p.Team))
		w.WriteD(p.ClanCrestLargeID)
		writeBoolC(w, p.Nobless, 1, 0)
		writeBoolC(w, p.Hero, 1, 0)
		writeBoolC(w, p.Fishing, 1, 0)
		w.WriteD(p.FishX)
		w.WriteD(p.FishY)
		w.WriteD(p.FishZ)
		w.WriteD(p.NameColor)
		w.WriteD(p.Heading)
		w.WriteD(p.PledgeClass)
		w.WriteD(p.PledgeType)
		w.WriteD(p.TitleColor)
		w.WriteD(p.CursedWeaponLvl)
		w.WriteD(p.AttackRange)
	})
}

// NpcInfo mirrors Java serverpackets/actor/AbstractNpcInfo.NpcInfo.
func NpcInfo(n *NPC) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x16)
		w.WriteD(n.ObjectID)
		w.WriteD(n.NPCID + 1000000)
		writeBool(w, n.IsAttackable, 1, 0)
		w.WriteD(n.X)
		w.WriteD(n.Y)
		w.WriteD(n.Z)
		w.WriteD(n.Heading)
		w.WriteD(0)
		w.WriteD(n.MAtkSpd)
		w.WriteD(n.PAtkSpd)
		for i := 0; i < 4; i++ {
			w.WriteD(n.RunSpeed)
			w.WriteD(n.WalkSpeed)
		}
		w.WriteF64(n.MoveMultiplier)
		w.WriteF64(n.AttackMultiplier)
		w.WriteF64(n.CollisionRadius)
		w.WriteF64(n.CollisionHeight)
		w.WriteD(n.RHand)
		w.WriteD(n.Chest)
		w.WriteD(n.LHand)
		w.WriteC(1) // name above char
		writeBoolC(w, n.Running, 1, 0)
		writeBoolC(w, n.InCombat, 1, 0)
		writeBoolC(w, n.AlikeDead(), 1, 0)
		w.WriteC(2) // summoned
		w.WriteS(n.Name)
		w.WriteS(n.Title)
		w.WriteD(0)
		w.WriteD(0)
		w.WriteD(0)
		w.WriteD(n.AbnormalEffect)
		w.WriteD(n.ClanID)
		w.WriteD(n.ClanCrest)
		w.WriteD(n.AllyID)
		w.WriteD(n.AllyCrest)
		w.WriteC(int(n.MoveType))
		w.WriteC(0)
		w.WriteF64(n.CollisionRadius)
		w.WriteF64(n.CollisionHeight)
		w.WriteD(n.EnchantEffect)
		writeBool(w, n.Flying, 1, 0)
		w.WriteD(n.AttackRange)
		w.WriteD(n.MaxHP)
		w.WriteD(n.Level)
	})
}

func ItemList(items []Item, show bool) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x1B)
		if show {
			w.WriteH(1)
		} else {
			w.WriteH(0)
		}
		w.WriteH(len(items))
		for _, it := range items {
			w.WriteH(int(it.Type1))
			w.WriteD(it.ObjectID)
			w.WriteD(it.ItemID)
			w.WriteD(it.Count)
			w.WriteH(int(it.Type2))
			w.WriteH(int(it.Custom1))
			if it.Equipped {
				w.WriteH(1)
			} else {
				w.WriteH(0)
			}
			w.WriteD(it.BodyPart)
			w.WriteH(int(it.Enchant))
			w.WriteH(int(it.Custom2))
			w.WriteD(it.Augment)
			w.WriteD(it.ManaLeft)
			w.WriteD(it.Slot)
		}
	})
}

func SkillList(skills []Skill) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x58)
		w.WriteD(int32(len(skills)))
		for _, s := range skills {
			if s.Passive {
				w.WriteD(1)
			} else {
				w.WriteD(0)
			}
			w.WriteD(s.Level)
			w.WriteD(s.ID)
			if s.Disabled {
				w.WriteC(1)
			} else {
				w.WriteC(0)
			}
		}
	})
}

func CreatureSay(objectID int32, sayType int32, name, text string) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x4A)
		w.WriteD(objectID)
		w.WriteD(sayType)
		w.WriteS(name)
		w.WriteS(text)
	})
}

// SysMsgParam types from Java SystemMessage.
const (
	sysMsgText       int32 = 0
	sysMsgNumber     int32 = 1
	sysMsgNpc        int32 = 2
	sysMsgItem       int32 = 3
	sysMsgSkill      int32 = 4
	sysMsgItemNumber int32 = 6
)

type SysMsgParam struct {
	kind  int32
	num   int32
	num2  int32
	value string
}

func SysText(v string) SysMsgParam     { return SysMsgParam{kind: sysMsgText, value: v} }
func SysNumber(v int32) SysMsgParam    { return SysMsgParam{kind: sysMsgNumber, num: v} }
func SysNpc(id int32) SysMsgParam      { return SysMsgParam{kind: sysMsgNpc, num: id + 1000000} }
func SysItem(id int32) SysMsgParam     { return SysMsgParam{kind: sysMsgItem, num: id} }
func SysItemCount(n int32) SysMsgParam { return SysMsgParam{kind: sysMsgItemNumber, num: n} }
func SysSkill(id, lvl int32) SysMsgParam {
	return SysMsgParam{kind: sysMsgSkill, num: id, num2: lvl}
}

// SystemMessage mirrors Java SystemMessage: id, parameter count, typed parameters.
func SystemMessage(id int32, params ...SysMsgParam) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x64)
		w.WriteD(id)
		w.WriteD(int32(len(params)))
		for _, p := range params {
			w.WriteD(p.kind)
			switch p.kind {
			case sysMsgText:
				w.WriteS(p.value)
			case sysMsgSkill:
				w.WriteD(p.num)
				w.WriteD(p.num2)
			default:
				w.WriteD(p.num)
			}
		}
	})
}

func MoveDirection(objectID, dirY, dirX, vert, x, y, z int32, ts int64) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0xC6)
		w.WriteD(objectID)
		w.WriteD(dirY)
		w.WriteD(dirX)
		w.WriteD(vert)
		w.WriteD(x)
		w.WriteD(y)
		w.WriteD(z)
		w.WriteQ(ts)
	})
}

func StopMove(objectID, x, y, z, heading int32) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x47)
		w.WriteD(objectID)
		w.WriteD(x)
		w.WriteD(y)
		w.WriteD(z)
		w.WriteD(heading)
	})
}

func ValidateLocation(objectID, x, y, z, heading int32) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x61)
		w.WriteD(objectID)
		w.WriteD(x)
		w.WriteD(y)
		w.WriteD(z)
		w.WriteD(heading)
	})
}

func Attack(attacker, target, damage int32, flags byte, x, y, z int32) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x05)
		w.WriteD(attacker)
		w.WriteD(target)
		w.WriteD(damage)
		w.WriteC(int(flags))
		w.WriteD(x)
		w.WriteD(y)
		w.WriteD(z)
		w.WriteH(0)
	})
}

// Die mirrors Java combat/Die: village is always available, clan hall / castle /
// siege HQ need a clan, sweepable marks a spoiled corpse.
func Die(objectID int32, toClanHall, toCastle, toSiegeHQ, sweepable, fixedRes bool) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x06)
		w.WriteD(objectID)
		w.WriteD(1) // to nearest village
		writeBool(w, toClanHall, 1, 0)
		writeBool(w, toCastle, 1, 0)
		writeBool(w, toSiegeHQ, 1, 0)
		writeBool(w, sweepable, 1, 0)
		writeBool(w, fixedRes, 1, 0)
	})
}

func Revive(objectID int32) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x07)
		w.WriteD(objectID)
	})
}

func StatusUpdate(objectID int32, attrs [][2]int32) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x0E)
		w.WriteD(objectID)
		w.WriteD(int32(len(attrs)))
		for _, a := range attrs {
			w.WriteD(a[0])
			w.WriteD(a[1])
		}
	})
}

func TargetSelected(objectID, target, x, y, z int32) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x29)
		w.WriteD(objectID)
		w.WriteD(target)
		w.WriteD(x)
		w.WriteD(y)
		w.WriteD(z)
	})
}

func MyTargetSelected(target int32, color int16) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0xA6)
		w.WriteD(target)
		w.WriteH(int(color))
	})
}

func DeleteObject(objectID int32) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x12)
		w.WriteD(objectID)
		w.WriteD(0)
	})
}

// MagicSkillUse mirrors Java skill/MagicSkillUse, including the success block
// that the client uses to play the cast animation.
func MagicSkillUse(caster, target, skillID, level, hitTime, reuse, x, y, z, tx, ty, tz int32, success bool) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x48)
		w.WriteD(caster)
		w.WriteD(target)
		w.WriteD(skillID)
		w.WriteD(level)
		w.WriteD(hitTime)
		w.WriteD(reuse)
		w.WriteD(x)
		w.WriteD(y)
		w.WriteD(z)
		if success {
			w.WriteD(1)
			w.WriteH(0)
		} else {
			w.WriteD(0)
		}
		w.WriteD(tx)
		w.WriteD(ty)
		w.WriteD(tz)
	})
}

func MagicSkillLaunched(caster, skillID, level int32, targets []int32) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x76)
		w.WriteD(caster)
		w.WriteD(skillID)
		w.WriteD(level)
		if len(targets) == 0 {
			w.WriteD(0)
			w.WriteD(0)
			return
		}
		w.WriteD(int32(len(targets)))
		for _, t := range targets {
			w.WriteD(t)
		}
	})
}

func MagicSkillCanceled(objectID int32) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x49)
		w.WriteD(objectID)
	})
}

// SetupGauge colors from Java SetupGauge.GaugeColor.
const (
	GaugeBlue  int32 = 0
	GaugeRed   int32 = 1
	GaugeCyan  int32 = 2
	GaugeGreen int32 = 3
)

func SetupGauge(color, current, max int32) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x6D)
		w.WriteD(color)
		w.WriteD(current)
		w.WriteD(max)
	})
}

// MoveToLocation mirrors Java movement/MoveToLocation.
func MoveToLocation(objectID, destX, destY, destZ, curX, curY, curZ int32) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x01)
		w.WriteD(objectID)
		w.WriteD(destX)
		w.WriteD(destY)
		w.WriteD(destZ)
		w.WriteD(curX)
		w.WriteD(curY)
		w.WriteD(curZ)
	})
}

// ChangeWaitType mirrors Java actor/ChangeWaitType (sit down / stand up).
const (
	WaitTypeSitting        int32 = 0
	WaitTypeStanding       int32 = 1
	WaitTypeStartFakeDeath int32 = 2
	WaitTypeStopFakeDeath  int32 = 3
)

func ChangeWaitType(objectID, waitType, x, y, z int32) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x2F)
		w.WriteD(objectID)
		w.WriteD(waitType)
		w.WriteD(x)
		w.WriteD(y)
		w.WriteD(z)
	})
}

func TeleportToLocation(objectID, x, y, z int32) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x28)
		w.WriteD(objectID)
		w.WriteD(x)
		w.WriteD(y)
		w.WriteD(z)
	})
}

func SocialAction(objectID, action int32) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x2D)
		w.WriteD(objectID)
		w.WriteD(action)
	})
}

func ChangeMoveType(objectID int32, running bool) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x2E)
		w.WriteD(objectID)
		if running {
			w.WriteD(1)
		} else {
			w.WriteD(0)
		}
		w.WriteD(0)
	})
}

func ShortCutInit(shortcuts []Shortcut) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x45)
		w.WriteD(int32(len(shortcuts)))
		for _, sc := range shortcuts {
			writeShortcut(w, sc)
		}
	})
}

// writeShortcut is the shared body of Java ShortCutInit and ShortCutRegister.
func writeShortcut(w *packet.Writer, sc Shortcut) {
	charType := sc.CharacterType
	if charType == 0 {
		charType = 1
	}
	w.WriteD(sc.Type)
	w.WriteD(sc.Slot + sc.Page*12)
	switch sc.Type {
	case ShortcutItem:
		w.WriteD(sc.ID)
		w.WriteD(charType)
		w.WriteD(sc.SharedReuseGroup)
		w.WriteD(0)
		w.WriteD(0)
		w.WriteD(0)
	case ShortcutSkill:
		w.WriteD(sc.ID)
		w.WriteD(sc.Level)
		w.WriteC(0)
		w.WriteD(charType)
	default:
		w.WriteD(sc.ID)
		w.WriteD(charType)
	}
}

// ShortCutRegister mirrors Java ShortCutRegister (single slot update).
func ShortCutRegister(sc Shortcut) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x44)
		writeShortcut(w, sc)
	})
}

// AcquireSkillType values are the ordinals of Java AcquireSkillType.
const (
	AcquireUsual   int32 = 0
	AcquireFishing int32 = 1
	AcquireClan    int32 = 2
)

// AcquireSkillList mirrors Java skill/AcquireSkillList for the USUAL type.
func AcquireSkillList(skillType int32, nodes []ClassSkillNode) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x8A)
		w.WriteD(skillType)
		w.WriteD(int32(len(nodes)))
		for _, n := range nodes {
			w.WriteD(n.ID)
			w.WriteD(n.Level)
			w.WriteD(n.Level)
			w.WriteD(n.Cost)
			w.WriteD(0)
		}
	})
}

// SkillRequirement is Java AcquireSkillInfo.SkillRequirement.
type SkillRequirement struct {
	Type   int32
	ItemID int32
	Count  int32
	Unk    int32
}

func AcquireSkillInfo(id, level, spCost, mode int32, reqs []SkillRequirement) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x8B)
		w.WriteD(id)
		w.WriteD(level)
		w.WriteD(spCost)
		w.WriteD(mode)
		w.WriteD(int32(len(reqs)))
		for _, r := range reqs {
			w.WriteD(r.Type)
			w.WriteD(r.ItemID)
			w.WriteD(r.Count)
			w.WriteD(r.Unk)
		}
	})
}

func AcquireSkillDone() []byte {
	return gsWrite(func(w *packet.Writer) { w.WriteC(0x8E) })
}

func RestartResponse(ok bool) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x5F)
		if ok {
			w.WriteD(1)
		} else {
			w.WriteD(0)
		}
	})
}

const (
	CharCreateFailed        int32 = 0x00
	CharCreateNameExists    int32 = 0x02
	CharCreateTooMany       int32 = 0x03
	CharCreateIncorrectName int32 = 0x04
)

// BuyList mirrors Java serverpackets/item/BuyList (opcode 0x11 plus the Unity openTab byte).
func BuyList(list NpcBuyList, adena int32, openTab bool) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x11)
		if openTab {
			w.WriteC(1)
		} else {
			w.WriteC(0)
		}
		w.WriteD(adena)
		w.WriteD(list.ID)
		w.WriteH(len(list.Items))
		for _, p := range list.Items {
			tpl := GetItem(p.ItemID)
			type1, type2, body := Type1QuestAdena, Type2Other, int32(0)
			if tpl != nil {
				type1, type2, body = tpl.Type1, tpl.Type2, tpl.BodyPart
			}
			w.WriteH(int(type1))
			w.WriteD(p.ItemID)
			w.WriteD(p.ItemID)
			w.WriteD(0)
			w.WriteH(int(type2))
			w.WriteH(0)
			w.WriteD(body)
			w.WriteH(0)
			w.WriteH(0)
			w.WriteH(0)
			w.WriteD(p.Price)
		}
	})
}

// SellList mirrors Java serverpackets/item/SellList.
func SellList(adena int32, items []Item, openTab bool) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x10)
		if openTab {
			w.WriteC(1)
		} else {
			w.WriteC(0)
		}
		w.WriteD(adena)
		w.WriteD(0)
		w.WriteH(len(items))
		for _, it := range items {
			w.WriteH(int(it.Type1))
			w.WriteD(it.ObjectID)
			w.WriteD(it.ItemID)
			w.WriteD(it.Count)
			w.WriteH(int(it.Type2))
			w.WriteH(int(it.Custom1))
			w.WriteD(it.BodyPart)
			w.WriteH(int(it.Enchant))
			w.WriteH(int(it.Custom2))
			w.WriteH(0)
			w.WriteD(ReferencePrice(it.ItemID) / 2)
		}
	})
}

// NpcHtmlMessage mirrors Java unused/NpcHtmlMessage.
func NpcHtmlMessage(objectID int32, html string) []byte {
	if len(html) > 8192 {
		html = "<html><body>Html was too long.</body></html>"
	}
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x0F)
		w.WriteD(objectID)
		w.WriteS(html)
		w.WriteD(0)
	})
}

func AskJoinParty(name string, lootRule int32) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x39)
		w.WriteS(name)
		w.WriteD(lootRule)
	})
}

func JoinParty(response int32) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x3A)
		w.WriteD(response)
	})
}

func PartySmallWindowAll(viewer *Character, members []*Character, leaderID, loot int32) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x4E)
		w.WriteD(leaderID)
		w.WriteD(loot)
		n := int32(0)
		for _, m := range members {
			if m.ObjectID != viewer.ObjectID {
				n++
			}
		}
		w.WriteD(n)
		for _, m := range members {
			if m.ObjectID == viewer.ObjectID {
				continue
			}
			writePartyMember(w, m)
		}
	})
}

func PartySmallWindowAdd(member *Character, leaderID, loot int32) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x4F)
		w.WriteD(leaderID)
		w.WriteD(loot)
		writePartyMember(w, member)
	})
}

func PartySmallWindowDeleteAll() []byte {
	return gsWrite(func(w *packet.Writer) { w.WriteC(0x50) })
}

func PartySmallWindowDelete(member *Character) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x51)
		w.WriteD(member.ObjectID)
		w.WriteS(member.Name)
	})
}

func writeTradeItem(w *packet.Writer, it Item, count int32) {
	w.WriteH(int(it.Type1))
	w.WriteD(it.ObjectID)
	w.WriteD(it.ItemID)
	w.WriteD(count)
	w.WriteH(int(it.Type2))
	w.WriteH(int(it.Custom1))
	w.WriteD(it.BodyPart)
	w.WriteH(int(it.Enchant))
	w.WriteH(int(it.Custom2))
	w.WriteH(0)
}

func writeWarehouseItem(w *packet.Writer, it Item) {
	writeTradeItem(w, it, it.Count)
	w.WriteD(it.ObjectID)
	w.WriteQ(0)
}

func SendTradeRequest(senderID int32) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x5E)
		w.WriteD(senderID)
	})
}

func TradeStart(partnerID int32, items []Item) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x1E)
		w.WriteD(partnerID)
		w.WriteH(len(items))
		for _, it := range items {
			writeTradeItem(w, it, it.Count)
		}
	})
}

func TradeOwnAdd(it Item, count int32) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x20)
		w.WriteH(1)
		writeTradeItem(w, it, count)
	})
}

func TradeOtherAdd(it Item, count int32) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x21)
		w.WriteH(1)
		writeTradeItem(w, it, count)
	})
}

func SendTradeDone(ok bool) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x22)
		if ok {
			w.WriteD(1)
		} else {
			w.WriteD(0)
		}
	})
}

func TradePressOwnOk() []byte {
	return gsWrite(func(w *packet.Writer) { w.WriteC(0x75) })
}

func TradePressOtherOk() []byte {
	return gsWrite(func(w *packet.Writer) { w.WriteC(0x7C) })
}

const (
	WarehousePrivate int16 = 1
	WarehouseClan    int16 = 2
)

func WarehouseDepositList(adena int32, items []Item) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x41)
		w.WriteH(int(WarehousePrivate))
		w.WriteD(adena)
		w.WriteH(len(items))
		for _, it := range items {
			writeWarehouseItem(w, it)
		}
	})
}

func WarehouseWithdrawList(adena int32, items []Item) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x42)
		w.WriteH(int(WarehousePrivate))
		w.WriteD(adena)
		w.WriteH(len(items))
		for _, it := range items {
			writeWarehouseItem(w, it)
		}
	})
}

func FriendAddRequest(name string) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x7D)
		w.WriteS(name)
	})
}

func FriendAddRequestResult(accepted bool) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x77)
		if accepted {
			w.WriteD(1)
		} else {
			w.WriteD(0)
		}
	})
}

func FriendList(friends []Friend, online map[int32]bool) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0xFA)
		w.WriteD(int32(len(friends)))
		for _, f := range friends {
			on := online[f.ObjectID]
			w.WriteD(f.ObjectID)
			w.WriteS(f.Name)
			if on {
				w.WriteD(1)
				w.WriteD(f.ObjectID)
			} else {
				w.WriteD(0)
				w.WriteD(0)
			}
		}
	})
}

func L2Friend(action int32, name string, objectID int32, online bool) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0xFB)
		w.WriteD(action)
		w.WriteD(0)
		w.WriteS(name)
		if online {
			w.WriteD(1)
		} else {
			w.WriteD(0)
		}
		w.WriteD(objectID)
	})
}

func writePartyMember(w *packet.Writer, m *Character) {
	w.WriteD(m.ObjectID)
	w.WriteS(m.Name)
	w.WriteD(int32(m.CurCP))
	w.WriteD(m.MaxCP)
	w.WriteD(int32(m.CurHP))
	w.WriteD(m.MaxHP)
	w.WriteD(int32(m.CurMP))
	w.WriteD(m.MaxMP)
	w.WriteD(m.Level)
	w.WriteD(m.ClassID)
	w.WriteD(0)
	w.WriteD(m.Race)
}
