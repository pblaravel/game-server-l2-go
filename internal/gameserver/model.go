package gameserver

import (
	"math"
	"sync"
	"time"
)

type ClientState int

const (
	StateConnected ClientState = iota
	StateAuthed
	StateEntering
	StateInGame
)

type Paperdoll int

const (
	PaperHairAll Paperdoll = iota
	PaperRear
	PaperLear
	PaperNeck
	PaperRFinger
	PaperLFinger
	PaperHead
	PaperRHand
	PaperLHand
	PaperGloves
	PaperChest
	PaperLegs
	PaperFeet
	PaperCloak
	PaperHair
	PaperFace
	PaperCount
)

type Item struct {
	ObjectID int32
	ItemID   int32
	Count    int32
	Enchant  int16
	Loc      string
	LocData  int32
	Type1    int16
	Type2    int16
	Custom1  int16
	Custom2  int16
	Equipped bool
	BodyPart int32
	Augment  int32
	ManaLeft int32
	Slot     int32
}

type Character struct {
	ObjectID                     int32
	Account                      string
	Name                         string
	Title                        string
	Level                        int32
	MaxHP                        int32
	CurHP                        float64
	MaxMP                        int32
	CurMP                        float64
	MaxCP                        int32
	CurCP                        float64
	Face                         int32
	HairStyle                    int32
	HairColor                    int32
	Sex                          int32
	Heading                      int32
	X, Y, Z                      int32
	Exp                          int64
	SP                           int32
	Karma                        int32
	PvPKills                     int32
	PKKills                      int32
	ClanID                       int32
	Race                         int32
	ClassID                      int32
	BaseClass                    int32
	DeleteTime                   int64
	AccessLevel                  int32
	LastAccess                   int64
	STR, DEX, CON, INT, WIT, MEN int32
	PAtk, PDef, MAtk, MDef       int32
	PAtkSpd, MAtkSpd             int32
	Accuracy, Evasion, Crit      int32
	RunSpeed, WalkSpeed          int32
	NameColor                    int32
	TitleColor                   int32
	InventoryLimit               int32
	PaperdollObj                 [PaperCount]int32
	PaperdollItem                [PaperCount]int32
	Items                        []Item
	Skills                       []Skill
	Shortcuts                    []Shortcut
	MoveDirX                     int32
	MoveDirY                     int32
	VerticalVel                  int32
	LastPacketTS                 int64
	Online                       bool

	// Broadcast state from Java Player / Appearance / CreatureStatus.
	PvPFlag          int32
	RecomLeft        int32
	RecomHave        int32
	Nobless          bool
	Hero             bool
	Sitting          bool
	Running          bool
	InCombat         bool
	Dead             bool
	Invisible        bool
	PrivateStore     int32
	MountType        int32
	EnchantEffect    int32
	Team             int32
	AbnormalEffect   int32
	Cubics           []int32
	InPartyMatchRoom bool
	ClanCrestID      int32
	ClanCrestLargeID int32
	AllyID           int32
	AllyCrestID      int32
	PledgeClass      int32
	PledgeType       int32
	CursedWeaponLvl  int32
	Fishing          bool
	FishX            int32
	FishY            int32
	FishZ            int32
	Flying           bool
	SwimSpeed        int32
	CollisionRadius  float64
	CollisionHeight  float64
	MoveMultiplier   float64
	AttackMultiplier float64
	AttackRange      int32
	AugmentRHand     int32
	AugmentLHand     int32
	CurrentWeight    int32
	WeightLimit      int32
	DestX            int32
	DestY            int32
	DestZ            int32
	Effects          []ActiveEffect
}

// AlikeDead is Java Creature.isAlikeDead (dead or fake death).
func (c *Character) AlikeDead() bool { return c.Dead }

// ApplyRuntimeDefaults sets the state Java derives at runtime instead of storing
// it in `characters`: stance, speed multipliers, collision box and attack range.
func (c *Character) ApplyRuntimeDefaults() {
	c.Running = true
	c.Dead = c.CurHP <= 0
	if c.MoveMultiplier == 0 {
		c.MoveMultiplier = 1
	}
	if c.AttackMultiplier == 0 {
		c.AttackMultiplier = 1
	}
	if c.SwimSpeed == 0 {
		c.SwimSpeed = 50
	}
	if c.AttackRange == 0 {
		c.AttackRange = 40
	}
	if c.InventoryLimit == 0 {
		c.InventoryLimit = 80
	}
	radius, height := CollisionFor(c.ClassID, c.Sex)
	if c.CollisionRadius == 0 {
		c.CollisionRadius = radius
	}
	if c.CollisionHeight == 0 {
		c.CollisionHeight = height
	}
}

// headingTo is Java MoveData heading: atan2 scaled to the 16-bit client heading.
func headingTo(fromX, fromY, toX, toY int32) int32 {
	dx := float64(toX - fromX)
	dy := float64(toY - fromY)
	if dx == 0 && dy == 0 {
		return 0
	}
	heading := int32(math.Atan2(-dy, -dx) * 10430.378350470453)
	return heading + 32768
}

// Distance2D / Distance3D are Java WorldObject helpers used by range checks.
func Distance2D(x1, y1, x2, y2 int32) float64 {
	dx := float64(x2 - x1)
	dy := float64(y2 - y1)
	return math.Sqrt(dx*dx + dy*dy)
}

func Distance3D(x1, y1, z1, x2, y2, z2 int32) float64 {
	dx := float64(x2 - x1)
	dy := float64(y2 - y1)
	dz := float64(z2 - z1)
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

// CollisionFor mirrors Java PlayerTemplate radius/height (female variants).
func CollisionFor(classID, sex int32) (float64, float64) {
	if tpl := GetClassTemplate(classID); tpl != nil && tpl.Radius > 0 {
		if sex != 0 && tpl.RadiusFemale > 0 {
			return tpl.RadiusFemale, tpl.HeightFemale
		}
		return tpl.Radius, tpl.Height
	}
	if sex != 0 {
		return 8, 23.5
	}
	return 9, 23
}

type Skill struct {
	ID       int32
	Level    int32
	Passive  bool
	Disabled bool
}

// Shortcut matches Java com.shnok.javaserver.gameserver.model.Shortcut.
type Shortcut struct {
	Slot             int32
	Page             int32
	Type             int32
	ID               int32
	Level            int32
	CharacterType    int32
	SharedReuseGroup int32
}

// ShortcutType ordinals from Java enums.ShortcutType.
const (
	ShortcutNone int32 = iota
	ShortcutItem
	ShortcutSkill
	ShortcutAction
	ShortcutMacro
	ShortcutRecipe
)

type NPC struct {
	ObjectID     int32
	NPCID        int32
	Name         string
	Title        string
	X, Y, Z      int32
	SpawnX       int32
	SpawnY       int32
	SpawnZ       int32
	Heading      int32
	Level        int32
	MaxHP        int32
	CurHP        int32
	MaxMP        int32
	CurMP        int32
	IsAttackable bool

	// Broadcast state from Java AbstractNpcInfo.NpcInfo.
	MAtkSpd          int32
	PAtkSpd          int32
	RunSpeed         int32
	WalkSpeed        int32
	CollisionRadius  float64
	CollisionHeight  float64
	MoveMultiplier   float64
	AttackMultiplier float64
	RHand            int32
	Chest            int32
	LHand            int32
	Running          bool
	InCombat         bool
	Dead             bool
	AbnormalEffect   int32
	ClanID           int32
	ClanCrest        int32
	AllyID           int32
	AllyCrest        int32
	MoveType         int32
	EnchantEffect    int32
	Flying           bool
	AttackRange      int32
	PAtk             int32
	PDef             int32
	MAtk             int32
	MDef             int32
	AggroRange       int32
	Exp              int64
	SP               int32
}

// NpcDefaults fills the template values Java reads from NpcData when the XML
// datapack is absent, so NpcInfo never broadcasts zeroed speeds or collision.
func (n *NPC) NpcDefaults() {
	if n.MAtkSpd == 0 {
		n.MAtkSpd = 333
	}
	if n.PAtkSpd == 0 {
		n.PAtkSpd = 300
	}
	if n.RunSpeed == 0 {
		n.RunSpeed = 120
	}
	if n.WalkSpeed == 0 {
		n.WalkSpeed = 80
	}
	if n.CollisionRadius == 0 {
		n.CollisionRadius = 8
	}
	if n.CollisionHeight == 0 {
		n.CollisionHeight = 22
	}
	if n.MoveMultiplier == 0 {
		n.MoveMultiplier = 1
	}
	if n.AttackMultiplier == 0 {
		n.AttackMultiplier = 1
	}
	if n.AttackRange == 0 {
		n.AttackRange = 40
	}
	if n.SpawnX == 0 && n.SpawnY == 0 && n.SpawnZ == 0 {
		n.SpawnX, n.SpawnY, n.SpawnZ = n.X, n.Y, n.Z
	}
	if n.MaxMP == 0 {
		n.MaxMP = n.MaxHP / 2
		n.CurMP = n.MaxMP
	}
	// NpcData XML holds the real combat stats; scale them off the level instead.
	if n.PAtk == 0 {
		n.PAtk = 8 + n.Level*3
	}
	if n.PDef == 0 {
		n.PDef = 40 + n.Level*4
	}
	if n.MAtk == 0 {
		n.MAtk = 6 + n.Level*2
	}
	if n.MDef == 0 {
		n.MDef = 20 + n.Level*3
	}
	n.Running = true
}

// AlikeDead is Java Creature.isAlikeDead.
func (n *NPC) AlikeDead() bool { return n.Dead || n.CurHP <= 0 }

type World struct {
	mu       sync.RWMutex
	objects  map[int32]any
	players  map[int32]*Character
	byName   map[string]*Character
	npcs     map[int32]*NPC
	nextID   int32
	gameTime int32
}

func NewWorld() *World {
	return &World{
		objects:  make(map[int32]any),
		players:  make(map[int32]*Character),
		byName:   make(map[string]*Character),
		npcs:     make(map[int32]*NPC),
		nextID:   500000000,
		gameTime: 0,
	}
}

func (w *World) NextID() int32 {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.nextID++
	return w.nextID
}

func (w *World) AddPlayer(p *Character) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.players[p.ObjectID] = p
	w.byName[p.Name] = p
	w.objects[p.ObjectID] = p
}

func (w *World) RemovePlayer(id int32) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if p, ok := w.players[id]; ok {
		delete(w.byName, p.Name)
		delete(w.players, id)
		delete(w.objects, id)
	}
}

func (w *World) GetPlayer(id int32) *Character {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.players[id]
}

func (w *World) GetPlayerByName(name string) *Character {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.byName[name]
}

func (w *World) Players() []*Character {
	w.mu.RLock()
	defer w.mu.RUnlock()
	out := make([]*Character, 0, len(w.players))
	for _, p := range w.players {
		out = append(out, p)
	}
	return out
}

func (w *World) ClearNPCs() {
	w.mu.Lock()
	defer w.mu.Unlock()
	for id := range w.npcs {
		delete(w.objects, id)
	}
	w.npcs = make(map[int32]*NPC)
}

func (w *World) AddNPC(n *NPC) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.npcs[n.ObjectID] = n
	w.objects[n.ObjectID] = n
}

func (w *World) GetNPC(id int32) *NPC {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.npcs[id]
}

func (w *World) NPCs() []*NPC {
	w.mu.RLock()
	defer w.mu.RUnlock()
	out := make([]*NPC, 0, len(w.npcs))
	for _, n := range w.npcs {
		out = append(out, n)
	}
	return out
}

func (w *World) GameTime() int32 {
	// Interlude game time: minutes since server start scaled.
	return int32(time.Now().Unix()/6) % 1440
}

// StartingClass is a newbie Interlude class template.
type StartingClass struct {
	ClassID                      int32
	Race                         int32
	X, Y, Z                      int32
	STR, DEX, CON, INT, WIT, MEN int32
	HP, MP, CP                   int32
}

var startingClasses = map[int32]StartingClass{
	0:  {0, 0, -71338, 258271, -3104, 40, 30, 43, 21, 11, 25, 80, 30, 40},
	10: {10, 0, -71338, 258271, -3104, 22, 21, 27, 41, 20, 39, 50, 80, 40},
	18: {18, 1, 46000, 49412, -3056, 36, 36, 36, 23, 14, 26, 80, 30, 40},
	25: {25, 1, 46000, 49412, -3056, 22, 27, 25, 37, 23, 38, 50, 80, 40},
	31: {31, 2, 28300, 11040, -4230, 41, 34, 32, 25, 12, 26, 80, 30, 40},
	38: {38, 2, 28300, 11040, -4230, 23, 24, 25, 44, 19, 35, 50, 80, 40},
	44: {44, 3, -56736, -113680, -672, 40, 26, 47, 18, 12, 27, 90, 30, 40},
	49: {49, 3, -56736, -113680, -672, 27, 24, 31, 31, 15, 34, 60, 70, 40},
	53: {53, 4, 108567, -174008, -400, 39, 29, 45, 20, 10, 27, 80, 30, 40},
}

func DefaultCharacter(account, name string, classID, race, sex, hair, color, face int32, objectID int32, nextItemID func() int32) *Character {
	tpl, ok := startingClasses[classID]
	if !ok {
		tpl = startingClasses[0]
		classID = 0
		race = 0
	}
	ch := &Character{
		ObjectID: objectID, Account: account, Name: name,
		Level: 1, MaxHP: tpl.HP, CurHP: float64(tpl.HP), MaxMP: tpl.MP, CurMP: float64(tpl.MP),
		MaxCP: tpl.CP, CurCP: float64(tpl.CP),
		Face: face, HairStyle: hair, HairColor: color, Sex: sex,
		X: tpl.X, Y: tpl.Y, Z: tpl.Z,
		Race: race, ClassID: classID, BaseClass: classID,
		STR: tpl.STR, DEX: tpl.DEX, CON: tpl.CON, INT: tpl.INT, WIT: tpl.WIT, MEN: tpl.MEN,
		PAtk: 10, PDef: 20, MAtk: 8, MDef: 20, PAtkSpd: 300, MAtkSpd: 333,
		Accuracy: 30, Evasion: 30, Crit: 40, RunSpeed: 120, WalkSpeed: 80,
		NameColor: 0xFFFFFF, TitleColor: 0xFFFF77, InventoryLimit: 80,
	}
	ch.ApplyRuntimeDefaults()
	if nextItemID == nil {
		n := objectID + 1000
		nextItemID = func() int32 { n++; return n }
	}
	ApplyStarterKit(ch, nextItemID)
	RecalcStats(ch)
	ch.CurHP = float64(ch.MaxHP)
	ch.CurMP = float64(ch.MaxMP)
	ch.CurCP = float64(ch.MaxCP)
	return ch
}
