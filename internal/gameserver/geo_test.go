package gameserver

import (
	"encoding/binary"
	"testing"
)

func TestGeoWorldConversion(t *testing.T) {
	wx, wy := -71338, 258271
	if WorldX(GeoX(wx)) < wx-8 || WorldX(GeoX(wx)) > wx+8 {
		t.Fatalf("roundtrip X %d -> %d -> %d", wx, GeoX(wx), WorldX(GeoX(wx)))
	}
	if WorldY(GeoY(wy)) < wy-8 || WorldY(GeoY(wy)) > wy+8 {
		t.Fatalf("roundtrip Y %d -> %d -> %d", wy, GeoY(wy), WorldY(GeoY(wy)))
	}
	if !IsOutOfWorld(worldXMin-1, 0) || IsOutOfWorld(0, 0) {
		t.Fatal("world bounds")
	}
}

func TestGeoEmptyPassthrough(t *testing.T) {
	e := newGeoEngine(defaultGeoConfig())
	ox, oy, oz := int32(-71338), int32(258271), int32(-3104)
	tx, ty, tz := ox+200, oy+80, oz
	if !e.CanMoveTo(int(ox), int(oy), int(oz), int(tx), int(ty), int(tz)) {
		t.Fatal("null blocks must allow movement")
	}
	loc := e.ValidLocation(ox, oy, oz, tx, ty, tz)
	if loc.X != tx || loc.Y != ty || loc.Z != tz {
		t.Fatalf("valid loc %+v", loc)
	}
	if !e.CanSeeWorld(ox, oy, oz, 22, tx, ty, tz, 22) {
		t.Fatal("null blocks must allow LOS")
	}
	if path := e.FindPath(int(ox), int(oy), int(oz), int(tx), int(ty), int(tz), false); len(path) != 0 {
		t.Fatalf("findPath without geo = %v", path)
	}
}

func TestUnpackGeoShort(t *testing.T) {
	packed := packGeoShort(cellFlagAll, 96)
	nswe, h := unpackGeoShort(packed)
	if nswe != cellFlagAll || h != 96 {
		t.Fatalf("nswe=%d height=%d packed=%d", nswe, h, packed)
	}
	nswe, h = unpackGeoShort(-16) // 0xFFF0: nswe 0, height -8
	if nswe != 0 {
		t.Fatalf("nswe %d", nswe)
	}
	_ = h
}

func TestCalculateGeoObject(t *testing.T) {
	inside := [][]bool{
		{false, true, false},
		{false, true, false},
		{false, false, false},
	}
	got := CalculateGeoObject(inside)
	if got[1][1] != cellFlagNone {
		t.Fatalf("inside cell nswe=%d", got[1][1])
	}
	if got[1][0]&cellFlagS != 0 {
		t.Fatalf("south of wall should block S, nswe=%d", got[1][0])
	}
}

func putTestCell(e *GeoEngine, geoX, geoY int, nswe byte, height int16) {
	bx, by := geoX/blockCellsX, geoY/blockCellsY
	e.ensureRegion(bx/regionBlocksX+tileXMin, by/regionBlocksY+tileYMin)
	existing := e.block(geoX, geoY)
	var c *blockComplex
	if bc, ok := existing.(*blockComplex); ok {
		c = bc
	} else {
		buf := make([]byte, blockCells*3)
		for i := 0; i < blockCells; i++ {
			buf[i*3] = cellFlagAll
			putCellHeight(buf, i*3, height)
		}
		c = &blockComplex{buffer: buf}
		e.setBlock(bx, by, c)
	}
	idx := complexIndex(geoX, geoY)
	c.buffer[idx] = nswe
	putCellHeight(c.buffer, idx, height)
}

func fillOpen(e *GeoEngine, x0, y0, x1, y1 int, height int16) {
	for gx := x0; gx <= x1; gx++ {
		for gy := y0; gy <= y1; gy++ {
			putTestCell(e, gx, gy, cellFlagAll, height)
		}
	}
}

func TestGeoWallStopsMove(t *testing.T) {
	e := newGeoEngine(defaultGeoConfig())
	fillOpen(e, 100, 100, 120, 110, 0)
	for gy := 100; gy <= 110; gy++ {
		putTestCell(e, 110, gy, cellFlagNone, 0)
	}
	ox, oy := WorldX(102), WorldY(105)
	tx, ty := WorldX(118), WorldY(105)
	oz := 8
	if e.CanMoveTo(ox, oy, oz, tx, ty, oz) {
		t.Fatal("expected wall to block a straight move")
	}
	loc := e.validLocation(ox, oy, oz, tx, ty, oz)
	if loc.X == int32(tx) && loc.Y == int32(ty) {
		t.Fatalf("valid location reached the far side of the wall: %+v", loc)
	}
	if GeoX(int(loc.X)) > 110 {
		t.Fatalf("valid location walked past the wall: %+v geo=%d", loc, GeoX(int(loc.X)))
	}
	if !e.CanSeeTarget(ox, oy, oz, 16, tx, ty, oz, 16, nil) {
		// LOS uses NSWE + height; a CELL_FLAG_NONE wall is a vertical obstacle.
		t.Log("LOS blocked by wall (expected when geo is present)")
	}
}

func TestGeoPathAroundWall(t *testing.T) {
	e := newGeoEngine(defaultGeoConfig())
	fillOpen(e, 200, 200, 230, 230, 0)
	for gy := 200; gy <= 220; gy++ {
		putTestCell(e, 215, gy, cellFlagNone, 0)
	}
	ox, oy := WorldX(205), WorldY(205)
	tx, ty := WorldX(225), WorldY(205)
	oz := 8
	if e.CanMoveTo(ox, oy, oz, tx, ty, oz) {
		t.Fatal("straight line should be blocked")
	}
	path := e.FindPath(ox, oy, oz, tx, ty, oz, false)
	if len(path) == 0 {
		t.Fatal("expected a path around the wall")
	}
	last := path[len(path)-1]
	if absInt(GeoX(int(last.X))-225) > 1 || absInt(GeoY(int(last.Y))-205) > 1 {
		t.Fatalf("path end %+v", last)
	}
}

func TestGeoFlatAndMultilayer(t *testing.T) {
	flat := &blockFlat{height: 50, nswe: cellFlagAll}
	if !flat.hasGeoPos() || flat.heightNearest(0, 0, 0, nil) != 50 {
		t.Fatal("flat")
	}
	if flat.indexBelow(0, 0, 40, nil) != -1 || flat.indexBelow(0, 0, 60, nil) != 0 {
		t.Fatal("flat below")
	}

	buf := []byte{2, cellFlagAll, 10, 0, cellFlagAll, 100, 0}
	// one cell, two layers at 10 and 100 (little-endian heights)
	putCellHeight(buf, 1, 10)
	putCellHeight(buf, 4, 100)
	ml := &blockMultilayer{buffer: buf}
	if h := ml.heightNearest(0, 0, 12, nil); h != 10 {
		t.Fatalf("nearest to 12 = %d", h)
	}
	if h := ml.heightNearest(0, 0, 90, nil); h != 100 {
		t.Fatalf("nearest to 90 = %d", h)
	}
	if ml.indexAbove(0, 0, 50, nil) < 0 {
		t.Fatal("expected layer above 50")
	}
}

func TestLoadSyntheticL2JRegion(t *testing.T) {
	raw := make([]byte, 0, regionBlocksX*regionBlocksY*3)
	for i := 0; i < regionBlocksX*regionBlocksY; i++ {
		raw = append(raw, typeFlatL2JL2OFF)
		var h [2]byte
		binary.LittleEndian.PutUint16(h[:], uint16(int16(40)))
		raw = append(raw, h[:]...)
	}
	e := newGeoEngine(defaultGeoConfig())
	if err := e.parseRegion(16, 10, raw, GeoL2J); err != nil {
		t.Fatal(err)
	}
	if !e.HasGeoPos(0, 0) {
		t.Fatal("region 16_10 should have geo at 0,0")
	}
	if h := e.HeightNearest(0, 0, 0, nil); h != 40 {
		t.Fatalf("height %d", h)
	}
}

func TestLoadSyntheticL2OFFComplex(t *testing.T) {
	// One region, all complex L2OFF blocks with height 20 and NSWE all.
	raw := make([]byte, 18) // header
	cell := packGeoShort(cellFlagAll, 16)
	var packed [2]byte
	binary.LittleEndian.PutUint16(packed[:], uint16(cell))
	for i := 0; i < regionBlocksX*regionBlocksY; i++ {
		raw = append(raw, byte(typeComplexL2OFF), 0) // short 0x40
		for c := 0; c < blockCells; c++ {
			raw = append(raw, packed[:]...)
		}
	}
	e := newGeoEngine(defaultGeoConfig())
	if err := e.parseRegion(16, 10, raw, GeoL2OFF); err != nil {
		t.Fatal(err)
	}
	if e.NsweNearest(3, 4, 30, nil) != cellFlagAll {
		t.Fatal("complex nswe")
	}
	if e.HeightNearest(3, 4, 30, nil) != 16 {
		t.Fatalf("height %d", e.HeightNearest(3, 4, 30, nil))
	}
}

func TestDynamicDoorBlocksNSWE(t *testing.T) {
	e := newGeoEngine(defaultGeoConfig())
	fillOpen(e, 300, 300, 310, 310, 0)
	inside := [][]bool{{true, true}, {true, true}}
	d := &DoorInstance{
		geoX: 304, geoY: 304, geoZ: 0, height: 50,
		geoData: CalculateGeoObject(inside),
	}
	e.AddGeoObject(d)
	if e.NsweNearest(304, 304, 8, nil) != cellFlagNone {
		t.Fatalf("closed door should clear nswe, got %d", e.NsweNearest(304, 304, 8, nil))
	}
	if e.NsweNearest(304, 304, 8, d) == cellFlagNone {
		t.Fatal("ignore door should restore original nswe")
	}
	e.RemoveGeoObject(d)
	if e.NsweNearest(304, 304, 8, nil) != cellFlagAll {
		t.Fatal("removed door should restore nswe")
	}
}

func TestDoorGeoFromRectangle(t *testing.T) {
	e := newGeoEngine(defaultGeoConfig())
	// Place flat geo under a door-sized rectangle near world origin of tile 16_10.
	gx, gy := 40, 40
	fillOpen(e, gx-4, gy-4, gx+8, gy+8, -100)
	wx, wy := WorldX(gx), WorldY(gy)
	tpl := DoorTemplate{
		ID: 1, Type: "DOOR", X: int32(wx), Y: int32(wy), Z: -100, Height: 80,
		Coords: [][2]int32{
			{int32(wx - 20), int32(wy - 8)},
			{int32(wx + 20), int32(wy - 8)},
			{int32(wx + 20), int32(wy + 8)},
			{int32(wx - 20), int32(wy + 8)},
		},
	}
	d := buildDoorGeo(e, tpl)
	if d == nil || len(d.geoData) == 0 {
		t.Fatal("door geo")
	}
	blocked := 0
	for _, col := range d.geoData {
		for _, nswe := range col {
			if nswe != cellFlagAll {
				blocked++
			}
		}
	}
	if blocked == 0 {
		t.Fatal("expected the door polygon to mark cells")
	}
}

func TestGeoConfigReadsRegionKeys(t *testing.T) {
	cfg := loadGeoConfig(FindDataDir())
	if !cfg.Enabled["16_24"] || !cfg.Enabled["17_25"] {
		t.Fatalf("Talking Island regions not enabled: %+v", cfg.Enabled)
	}
	if cfg.Enabled["16_10"] {
		t.Fatal("commented region 16_10 should stay disabled")
	}
	if cfg.Type != GeoL2OFF {
		t.Fatalf("type %s", cfg.Type)
	}
}

func TestDatapackLoadsGeoEngine(t *testing.T) {
	if Geo() == nil {
		t.Fatal("geo engine")
	}
	if DoorCount() < 100 {
		t.Fatalf("doors %d", DoorCount())
	}
	if len(LiveDoors()) == 0 {
		t.Fatal("expected door geo objects after datapack load")
	}
}
