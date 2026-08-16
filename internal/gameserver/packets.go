package gameserver

import "github.com/pblaravel/game-server-l2-go/internal/packet"

func gsWrite(write func(w *packet.Writer)) []byte {
	w := packet.NewWriter()
	write(w)
	return w.Bytes()
}

func VersionCheck(key8 []byte, useCipher bool) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x00)
		w.WriteC(0x01)
		w.WriteB(key8)
		if useCipher {
			w.WriteD(1)
		} else {
			w.WriteD(0)
		}
		w.WriteD(1)
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
			w.WriteF64(0)
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
			w.WriteD(0)
			w.WriteD(s.ClassID)
			if i == active {
				w.WriteD(1)
			} else {
				w.WriteD(0)
			}
			w.WriteC(0)
			w.WriteD(0)
		}
	})
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
		w.WriteF64(0)
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
		w.WriteD(0)
		w.WriteD(80000)
		w.WriteD(20)
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
		w.WriteD(0)
		w.WriteD(p.Karma)
		w.WriteD(p.RunSpeed)
		w.WriteD(p.WalkSpeed)
		w.WriteD(p.RunSpeed)
		w.WriteD(p.RunSpeed)
		w.WriteD(0)
		w.WriteD(0)
		w.WriteD(0)
		w.WriteD(0)
		w.WriteF64(1)
		w.WriteF64(1)
		w.WriteF64(9)
		w.WriteF64(23)
		w.WriteD(p.HairStyle)
		w.WriteD(p.HairColor)
		w.WriteD(p.Face)
		if p.AccessLevel > 0 {
			w.WriteD(1)
		} else {
			w.WriteD(0)
		}
		w.WriteS(p.Title)
		w.WriteD(p.ClanID)
		w.WriteD(0)
		w.WriteD(0)
		w.WriteD(0)
		w.WriteD(0)
		w.WriteC(0)
		w.WriteC(0)
		w.WriteC(0)
		w.WriteD(p.PKKills)
		w.WriteD(p.PvPKills)
		w.WriteH(0)
		w.WriteC(0)
		w.WriteD(0)
		w.WriteC(0)
		w.WriteD(0)
		w.WriteH(0)
		w.WriteH(0)
		w.WriteD(0)
		w.WriteH(int(p.InventoryLimit))
		w.WriteD(p.ClassID)
		w.WriteD(0)
		w.WriteD(p.MaxCP)
		w.WriteD(int32(p.CurCP))
		w.WriteC(0)
		w.WriteC(0)
		w.WriteD(0)
		w.WriteC(0)
		w.WriteC(0)
		w.WriteC(0)
		w.WriteD(0)
		w.WriteD(0)
		w.WriteD(0)
		w.WriteD(p.NameColor)
		w.WriteC(1)
		w.WriteD(0)
		w.WriteD(0)
		w.WriteD(p.TitleColor)
		w.WriteD(0)
		w.WriteD(40)
	})
}

func CharInfo(p *Character) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x03)
		w.WriteD(p.X)
		w.WriteD(p.Y)
		w.WriteD(p.Z)
		w.WriteD(p.Heading)
		w.WriteD(p.ObjectID)
		w.WriteS(p.Name)
		w.WriteD(p.Race)
		w.WriteD(p.Sex)
		w.WriteD(p.ClassID)
		writePaperdoll17(w, p.PaperdollItem)
		w.WriteD(p.PvPKills)
		w.WriteD(p.Karma)
		w.WriteD(p.MAtkSpd)
		w.WriteD(p.PAtkSpd)
		w.WriteD(p.PvPKills)
		w.WriteD(p.RunSpeed)
		w.WriteD(p.WalkSpeed)
		w.WriteD(p.RunSpeed)
		w.WriteD(p.RunSpeed)
		w.WriteD(0)
		w.WriteD(0)
		w.WriteD(0)
		w.WriteD(0)
		w.WriteF64(1)
		w.WriteF64(1)
		w.WriteF64(9)
		w.WriteF64(23)
		w.WriteD(0)
		w.WriteS(p.Title)
		w.WriteD(p.ClanID)
		w.WriteD(0)
		w.WriteD(0)
		w.WriteD(0)
		w.WriteD(0)
		w.WriteC(1)
		w.WriteC(0)
		w.WriteF64(float64(p.MaxHP))
		w.WriteF64(p.CurHP)
		w.WriteD(p.ClassID)
		w.WriteD(p.Heading)
		w.WriteC(0)
		w.WriteC(int(p.Level))
	})
}

func NpcInfo(n *NPC) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x16)
		w.WriteD(n.ObjectID)
		w.WriteD(n.NPCID + 1000000)
		if n.IsAttackable {
			w.WriteD(1)
		} else {
			w.WriteD(0)
		}
		w.WriteD(n.X)
		w.WriteD(n.Y)
		w.WriteD(n.Z)
		w.WriteD(n.Heading)
		w.WriteD(0)
		w.WriteD(333)
		w.WriteD(300)
		w.WriteD(120)
		w.WriteD(80)
		w.WriteD(120)
		w.WriteD(80)
		w.WriteD(0)
		w.WriteD(0)
		w.WriteD(0)
		w.WriteD(0)
		w.WriteF64(1)
		w.WriteF64(1)
		w.WriteF64(8)
		w.WriteF64(22)
		w.WriteD(0)
		w.WriteS(n.Name)
		w.WriteS(n.Title)
		w.WriteD(0)
		w.WriteD(0)
		w.WriteD(0)
		w.WriteD(0)
		w.WriteC(0)
		w.WriteC(0)
		w.WriteF64(float64(n.MaxHP))
		w.WriteF64(float64(n.CurHP))
		w.WriteD(0)
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

func SystemMessage(id int32) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x64)
		w.WriteD(id)
		w.WriteD(0)
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

func Die(objectID int32) []byte {
	return gsWrite(func(w *packet.Writer) {
		w.WriteC(0x06)
		w.WriteD(objectID)
		w.WriteD(1)
		w.WriteD(0)
		w.WriteD(0)
		w.WriteD(0)
		w.WriteD(0)
		w.WriteD(0)
		w.WriteD(0)
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

func MagicSkillUse(caster, target, skillID, level, hitTime, reuse, x, y, z int32) []byte {
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
		w.WriteD(0)
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
			w.WriteD(sc.Type)
			w.WriteD(sc.Slot + sc.Page*12)
			switch sc.Type {
			case 1: // ITEM
				w.WriteD(sc.ID)
				if sc.CharacterType == 0 {
					w.WriteD(1)
				} else {
					w.WriteD(sc.CharacterType)
				}
				w.WriteD(0)
				w.WriteD(0)
				w.WriteD(0)
				w.WriteD(0)
			case 2: // SKILL
				w.WriteD(sc.ID)
				w.WriteD(sc.Level)
				w.WriteC(0)
				if sc.CharacterType == 0 {
					w.WriteD(1)
				} else {
					w.WriteD(sc.CharacterType)
				}
			default:
				w.WriteD(sc.ID)
				if sc.CharacterType == 0 {
					w.WriteD(1)
				} else {
					w.WriteD(sc.CharacterType)
				}
			}
		}
	})
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
	CharCreateFailed          int32 = 0x00
	CharCreateNameExists      int32 = 0x02
	CharCreateTooMany         int32 = 0x03
	CharCreateIncorrectName   int32 = 0x04
)
