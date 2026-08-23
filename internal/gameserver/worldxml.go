package gameserver

import (
	"encoding/xml"
	"sync"
)

type Zone struct {
	Kind   string
	MinZ   int32
	MaxZ   int32
	Xs, Ys []int32
}

type DoorTemplate struct {
	ID      int32
	Name    string
	Type    string
	Level   int32
	X, Y, Z int32
	HP      int32
	PDef    int32
	MDef    int32
	Height  int32
	Opened  bool
	Coords  [][2]int32
}

type StaticObject struct {
	ID      int32
	X, Y, Z int32
	Type    int32
	Texture string
	MapX    int32
	MapY    int32
}

type WalkerRoute struct {
	Name  string
	Nodes [][3]int32
}

type BoatRoute struct {
	Dock1   string
	Dock2   string
	Item1   int32
	Item2   int32
	Heading int32
	Nodes   [][3]int32
}

type CastleData struct {
	ID        int32
	Name      string
	Alias     string
	CircletID int32
}

type ClanHallData struct {
	ID   int32
	Name string
	Loc  string
}

type ManorArea struct {
	Name     string
	CastleID int32
	MinZ     int32
	MaxZ     int32
}

type ManorCrop struct {
	ID     int32
	SeedID int32
	Level  int32
}

type ObserverEntry struct {
	GroupID int32
	LocID   int32
	X, Y, Z int32
	Cost    int32
	Castle  int32
}

type ClanHallDeco struct {
	Name  string
	Type  int32
	Level int32
	Price int32
}

type SpawnMakerNPC struct {
	NPCID   int32
	Total   int32
	Respawn int32
	X, Y, Z int32
}

var (
	worldMu         sync.RWMutex
	zones           []Zone
	doors           = map[int32]DoorTemplate{}
	staticObjects   []StaticObject
	walkerRoutes    []WalkerRoute
	boatRoutes      []BoatRoute
	castles         []CastleData
	clanHalls       []ClanHallData
	clanHallDecos   []ClanHallDeco
	manorAreas      []ManorArea
	manorCrops      []ManorCrop
	observerEntries []ObserverEntry
	xmlSpawns       []SpawnMakerNPC
)

func ZoneCount() int {
	worldMu.RLock()
	defer worldMu.RUnlock()
	return len(zones)
}

func DoorCount() int {
	worldMu.RLock()
	defer worldMu.RUnlock()
	return len(doors)
}

func StaticObjectCount() int {
	worldMu.RLock()
	defer worldMu.RUnlock()
	return len(staticObjects)
}

func WalkerRouteCount() int {
	worldMu.RLock()
	defer worldMu.RUnlock()
	return len(walkerRoutes)
}

func BoatRouteCount() int {
	worldMu.RLock()
	defer worldMu.RUnlock()
	return len(boatRoutes)
}

func CastleDataCount() int {
	worldMu.RLock()
	defer worldMu.RUnlock()
	return len(castles)
}

func ClanHallDataCount() int {
	worldMu.RLock()
	defer worldMu.RUnlock()
	return len(clanHalls)
}

func SpawnXMLCount() int {
	worldMu.RLock()
	defer worldMu.RUnlock()
	return len(xmlSpawns)
}

func XMLSpawns() []SpawnMakerNPC {
	worldMu.RLock()
	defer worldMu.RUnlock()
	return append([]SpawnMakerNPC(nil), xmlSpawns...)
}

func InPeaceZone(x, y, z int32) bool {
	return inZoneKind("PeaceZone", x, y, z)
}

func inZoneKind(kind string, x, y, z int32) bool {
	worldMu.RLock()
	defer worldMu.RUnlock()
	for _, zn := range zones {
		if zn.Kind != kind {
			continue
		}
		if z < zn.MinZ || z > zn.MaxZ {
			continue
		}
		if pointInPoly(x, y, zn.Xs, zn.Ys) {
			return true
		}
	}
	return false
}

func pointInPoly(x, y int32, xs, ys []int32) bool {
	if len(xs) < 3 || len(xs) != len(ys) {
		return false
	}
	inside := false
	j := len(xs) - 1
	for i := 0; i < len(xs); i++ {
		xi, yi := xs[i], ys[i]
		xj, yj := xs[j], ys[j]
		intersect := (yi > y) != (yj > y) &&
			float64(x) < float64(xj-xi)*float64(y-yi)/float64(yj-yi)+float64(xi)
		if intersect {
			inside = !inside
		}
		j = i
	}
	return inside
}

type xmlZoneFile struct {
	Zones []xmlZone `xml:"zone"`
}

type xmlZone struct {
	Shape string        `xml:"shape,attr"`
	MinZ  int32         `xml:"minZ,attr"`
	MaxZ  int32         `xml:"maxZ,attr"`
	Nodes []xmlZoneNode `xml:"node"`
}

type xmlZoneNode struct {
	X int32 `xml:"x,attr"`
	Y int32 `xml:"y,attr"`
}

func loadZoneXML(dir string) error {
	var next []Zone
	err := walkXMLFiles(dir, func(name string, body []byte) error {
		kind := stringsTrimSuffixXML(name)
		var root xmlZoneFile
		if err := xml.Unmarshal(body, &root); err != nil {
			return err
		}
		for _, z := range root.Zones {
			zn := Zone{Kind: kind, MinZ: z.MinZ, MaxZ: z.MaxZ}
			for _, n := range z.Nodes {
				zn.Xs = append(zn.Xs, n.X)
				zn.Ys = append(zn.Ys, n.Y)
			}
			if len(zn.Xs) > 0 {
				next = append(next, zn)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	worldMu.Lock()
	zones = next
	worldMu.Unlock()
	return nil
}

func stringsTrimSuffixXML(name string) string {
	if len(name) > 4 && (name[len(name)-4:] == ".xml" || name[len(name)-4:] == ".XML") {
		return name[:len(name)-4]
	}
	return name
}

type xmlDoorFile struct {
	Doors []xmlDoor `xml:"door"`
}

type xmlDoor struct {
	ID       int32         `xml:"id,attr"`
	Type     string        `xml:"type,attr"`
	Level    int32         `xml:"level,attr"`
	Name     string        `xml:"name,attr"`
	Pos      xmlDoorPos    `xml:"position"`
	Stats    xmlDoorStats  `xml:"stats"`
	Function xmlDoorFunc   `xml:"function"`
	Coords   xmlDoorCoords `xml:"coordinates"`
}

type xmlDoorFunc struct {
	Opened bool `xml:"opened,attr"`
}

type xmlDoorCoords struct {
	Locs []xmlDoorLoc `xml:"loc"`
}

type xmlDoorLoc struct {
	X int32 `xml:"x,attr"`
	Y int32 `xml:"y,attr"`
}

type xmlDoorPos struct {
	X int32 `xml:"x,attr"`
	Y int32 `xml:"y,attr"`
	Z int32 `xml:"z,attr"`
}

type xmlDoorStats struct {
	HP     int32 `xml:"hp,attr"`
	PDef   int32 `xml:"pDef,attr"`
	MDef   int32 `xml:"mDef,attr"`
	Height int32 `xml:"height,attr"`
}

func loadDoorXML(path string) error {
	var root xmlDoorFile
	if err := readXML(path, &root); err != nil {
		return err
	}
	next := map[int32]DoorTemplate{}
	for _, d := range root.Doors {
		coords := make([][2]int32, 0, len(d.Coords.Locs))
		for _, loc := range d.Coords.Locs {
			coords = append(coords, [2]int32{loc.X, loc.Y})
		}
		next[d.ID] = DoorTemplate{
			ID: d.ID, Name: d.Name, Type: d.Type, Level: d.Level,
			X: d.Pos.X, Y: d.Pos.Y, Z: d.Pos.Z,
			HP: d.Stats.HP, PDef: d.Stats.PDef, MDef: d.Stats.MDef, Height: d.Stats.Height,
			Opened: d.Function.Opened, Coords: coords,
		}
	}
	worldMu.Lock()
	doors = next
	worldMu.Unlock()
	return nil
}

type xmlStaticFile struct {
	Objects []xmlStatic `xml:"object"`
}

type xmlStatic struct {
	ID      int32  `xml:"id,attr"`
	X       int32  `xml:"x,attr"`
	Y       int32  `xml:"y,attr"`
	Z       int32  `xml:"z,attr"`
	Type    int32  `xml:"type,attr"`
	Texture string `xml:"texture,attr"`
	MapX    int32  `xml:"mapX,attr"`
	MapY    int32  `xml:"mapY,attr"`
}

func loadStaticObjectXML(path string) error {
	var root xmlStaticFile
	if err := readXML(path, &root); err != nil {
		return err
	}
	next := make([]StaticObject, 0, len(root.Objects))
	for _, o := range root.Objects {
		next = append(next, StaticObject{
			ID: o.ID, X: o.X, Y: o.Y, Z: o.Z, Type: o.Type,
			Texture: o.Texture, MapX: o.MapX, MapY: o.MapY,
		})
	}
	worldMu.Lock()
	staticObjects = next
	worldMu.Unlock()
	return nil
}

type xmlWalkerFile struct {
	Routes []xmlWalkerRoute `xml:"route"`
}

type xmlWalkerRoute struct {
	Name string         `xml:"name,attr"`
	Npcs []xmlWalkerNPC `xml:"npc"`
}

type xmlWalkerNPC struct {
	Nodes []xmlXYZ `xml:"node"`
}

type xmlXYZ struct {
	X int32 `xml:"x,attr"`
	Y int32 `xml:"y,attr"`
	Z int32 `xml:"z,attr"`
}

func loadWalkerRouteXML(path string) error {
	var root xmlWalkerFile
	if err := readXML(path, &root); err != nil {
		return err
	}
	var next []WalkerRoute
	for _, r := range root.Routes {
		wr := WalkerRoute{Name: r.Name}
		for _, npc := range r.Npcs {
			for _, n := range npc.Nodes {
				wr.Nodes = append(wr.Nodes, [3]int32{n.X, n.Y, n.Z})
			}
		}
		next = append(next, wr)
	}
	worldMu.Lock()
	walkerRoutes = next
	worldMu.Unlock()
	return nil
}

type xmlBoatFile struct {
	Routes []xmlBoat `xml:"itinerary"`
}

type xmlBoat struct {
	Dock1   string        `xml:"dock1,attr"`
	Dock2   string        `xml:"dock2,attr"`
	Item1   int32         `xml:"item1,attr"`
	Item2   int32         `xml:"item2,attr"`
	Heading int32         `xml:"heading,attr"`
	Route   []xmlBoatNode `xml:"route>node"`
}

type xmlBoatNode struct {
	X int32 `xml:"x,attr"`
	Y int32 `xml:"y,attr"`
	Z int32 `xml:"z,attr"`
}

func loadBoatXML(path string) error {
	var root xmlBoatFile
	if err := readXML(path, &root); err != nil {
		return err
	}
	next := make([]BoatRoute, 0, len(root.Routes))
	for _, r := range root.Routes {
		br := BoatRoute{Dock1: r.Dock1, Dock2: r.Dock2, Item1: r.Item1, Item2: r.Item2, Heading: r.Heading}
		for _, n := range r.Route {
			br.Nodes = append(br.Nodes, [3]int32{n.X, n.Y, n.Z})
		}
		next = append(next, br)
	}
	worldMu.Lock()
	boatRoutes = next
	worldMu.Unlock()
	return nil
}

type xmlCastleFile struct {
	Castles []xmlCastle `xml:"castle"`
}

type xmlCastle struct {
	ID        int32  `xml:"id,attr"`
	Alias     string `xml:"alias,attr"`
	Name      string `xml:"name,attr"`
	CircletID int32  `xml:"circletId,attr"`
}

func loadCastleXML(path string) error {
	var root xmlCastleFile
	if err := readXML(path, &root); err != nil {
		return err
	}
	next := make([]CastleData, 0, len(root.Castles))
	for _, c := range root.Castles {
		next = append(next, CastleData{ID: c.ID, Name: c.Name, Alias: c.Alias, CircletID: c.CircletID})
	}
	worldMu.Lock()
	castles = next
	worldMu.Unlock()
	return nil
}

type xmlClanHallFile struct {
	Halls []xmlClanHall `xml:"clanHall"`
}

type xmlClanHall struct {
	ID   int32     `xml:"id,attr"`
	Name string    `xml:"name,attr"`
	Agit xmlCHAgit `xml:"agit"`
}

type xmlCHAgit struct {
	Loc string `xml:"loc,attr"`
}

func loadClanHallXML(path string) error {
	var root xmlClanHallFile
	if err := readXML(path, &root); err != nil {
		return err
	}
	next := make([]ClanHallData, 0, len(root.Halls))
	for _, h := range root.Halls {
		next = append(next, ClanHallData{ID: h.ID, Name: h.Name, Loc: h.Agit.Loc})
	}
	worldMu.Lock()
	clanHalls = next
	worldMu.Unlock()
	return nil
}

type xmlDecoFile struct {
	Decos []xmlDeco `xml:"deco"`
}

type xmlDeco struct {
	Name  string `xml:"name,attr"`
	Type  int32  `xml:"type,attr"`
	Level int32  `xml:"level,attr"`
	Price int32  `xml:"price,attr"`
}

func loadClanHallDecoXML(path string) error {
	var root xmlDecoFile
	if err := readXML(path, &root); err != nil {
		return err
	}
	next := make([]ClanHallDeco, 0, len(root.Decos))
	for _, d := range root.Decos {
		next = append(next, ClanHallDeco{Name: d.Name, Type: d.Type, Level: d.Level, Price: d.Price})
	}
	worldMu.Lock()
	clanHallDecos = next
	worldMu.Unlock()
	return nil
}

type xmlManorAreaFile struct {
	Areas []xmlManorArea `xml:"area"`
}

type xmlManorArea struct {
	Name     string `xml:"name,attr"`
	CastleID int32  `xml:"castleId,attr"`
	MinZ     int32  `xml:"minZ,attr"`
	MaxZ     int32  `xml:"maxZ,attr"`
}

func loadManorAreaXML(path string) error {
	var root xmlManorAreaFile
	if err := readXML(path, &root); err != nil {
		return err
	}
	next := make([]ManorArea, 0, len(root.Areas))
	for _, a := range root.Areas {
		next = append(next, ManorArea{Name: a.Name, CastleID: a.CastleID, MinZ: a.MinZ, MaxZ: a.MaxZ})
	}
	worldMu.Lock()
	manorAreas = next
	worldMu.Unlock()
	return nil
}

type xmlManorFile struct {
	Manors []xmlManor `xml:"manor"`
}

type xmlManor struct {
	Crops []xmlCrop `xml:"crop"`
}

type xmlCrop struct {
	ID     int32 `xml:"id,attr"`
	SeedID int32 `xml:"seedId,attr"`
	Level  int32 `xml:"level,attr"`
}

func loadManorXML(path string) error {
	var root xmlManorFile
	if err := readXML(path, &root); err != nil {
		return err
	}
	var next []ManorCrop
	for _, m := range root.Manors {
		for _, c := range m.Crops {
			next = append(next, ManorCrop{ID: c.ID, SeedID: c.SeedID, Level: c.Level})
		}
	}
	worldMu.Lock()
	manorCrops = next
	worldMu.Unlock()
	return nil
}

type xmlObserverFile struct {
	Groups []xmlObserverGroup `xml:"groups>group"`
}

type xmlObserverGroup struct {
	ID      int32              `xml:"id,attr"`
	Entries []xmlObserverEntry `xml:"entry"`
}

type xmlObserverEntry struct {
	LocID  int32 `xml:"locId,attr"`
	X      int32 `xml:"x,attr"`
	Y      int32 `xml:"y,attr"`
	Z      int32 `xml:"z,attr"`
	Cost   int32 `xml:"cost,attr"`
	Castle int32 `xml:"castle,attr"`
}

func loadObserverXML(path string) error {
	var root xmlObserverFile
	if err := readXML(path, &root); err != nil {
		return err
	}
	var next []ObserverEntry
	for _, g := range root.Groups {
		for _, e := range g.Entries {
			next = append(next, ObserverEntry{
				GroupID: g.ID, LocID: e.LocID, X: e.X, Y: e.Y, Z: e.Z, Cost: e.Cost, Castle: e.Castle,
			})
		}
	}
	worldMu.Lock()
	observerEntries = next
	worldMu.Unlock()
	return nil
}

type xmlSpawnFile struct {
	Territories []xmlTerritory `xml:"territory"`
	Makers      []xmlNpcMaker  `xml:"npcmaker"`
}

type xmlTerritory struct {
	Name  string        `xml:"name,attr"`
	MinZ  int32         `xml:"minZ,attr"`
	MaxZ  int32         `xml:"maxZ,attr"`
	Nodes []xmlZoneNode `xml:"node"`
}

type xmlNpcMaker struct {
	Territory string        `xml:"territory,attr"`
	Npcs      []xmlSpawnNpc `xml:"npc"`
}

type xmlSpawnNpc struct {
	ID      int32  `xml:"id,attr"`
	Total   int32  `xml:"total,attr"`
	Respawn string `xml:"respawn,attr"`
}

func loadSpawnXML(dir string) error {
	type terr struct {
		cx, cy, cz int32
	}
	var next []SpawnMakerNPC
	err := walkXMLFiles(dir, func(_ string, body []byte) error {
		var root xmlSpawnFile
		if err := xml.Unmarshal(body, &root); err != nil {
			return err
		}
		byName := map[string]terr{}
		for _, t := range root.Territories {
			if len(t.Nodes) == 0 {
				continue
			}
			var sx, sy int64
			for _, n := range t.Nodes {
				sx += int64(n.X)
				sy += int64(n.Y)
			}
			n := int64(len(t.Nodes))
			byName[t.Name] = terr{int32(sx / n), int32(sy / n), (t.MinZ + t.MaxZ) / 2}
		}
		for _, m := range root.Makers {
			loc := byName[m.Territory]
			for _, n := range m.Npcs {
				total := n.Total
				if total <= 0 {
					total = 1
				}
				for i := int32(0); i < total; i++ {
					next = append(next, SpawnMakerNPC{
						NPCID: n.ID, Total: 1, Respawn: parseRespawnSec(n.Respawn),
						X: loc.cx, Y: loc.cy, Z: loc.cz,
					})
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	worldMu.Lock()
	xmlSpawns = next
	worldMu.Unlock()
	return nil
}
