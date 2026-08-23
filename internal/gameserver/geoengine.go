package gameserver

import "math"

type moveDir struct {
	stepX, stepY     int
	signumX, signumY int
	offsetX, offsetY int
	dirX, dirY       byte
}

func newMoveDir(signumX, signumY int) moveDir {
	d := moveDir{
		stepX:   signumX * cellSize,
		stepY:   signumY * cellSize,
		signumX: signumX,
		signumY: signumY,
		offsetX: 0,
		offsetY: 0,
	}
	if signumX >= 0 {
		d.offsetX = cellSize - 1
	}
	if signumY >= 0 {
		d.offsetY = cellSize - 1
	}
	if signumX < 0 {
		d.dirX = cellFlagW
	} else if signumX > 0 {
		d.dirX = cellFlagE
	}
	if signumY < 0 {
		d.dirY = cellFlagN
	} else if signumY > 0 {
		d.dirY = cellFlagS
	}
	return d
}

func directionOf(gdx, gdy int) moveDir {
	if gdx == 0 {
		if gdy < 0 {
			return newMoveDir(0, -1)
		}
		return newMoveDir(0, 1)
	}
	if gdy == 0 {
		if gdx < 0 {
			return newMoveDir(-1, 0)
		}
		return newMoveDir(1, 0)
	}
	sx, sy := 1, 1
	if gdx < 0 {
		sx = -1
	}
	if gdy < 0 {
		sy = -1
	}
	return newMoveDir(sx, sy)
}

// CalculateGeoObject is Java GeoEngine.calculateGeoObject.
func CalculateGeoObject(inside [][]bool) [][]byte {
	if len(inside) == 0 {
		return nil
	}
	width := len(inside)
	height := len(inside[0])
	result := make([][]byte, width)
	for ix := 0; ix < width; ix++ {
		result[ix] = make([]byte, height)
		for iy := 0; iy < height; iy++ {
			if inside[ix][iy] {
				result[ix][iy] = cellFlagNone
				continue
			}
			nswe := cellFlagAll
			if iy < height-1 && inside[ix][iy+1] {
				nswe &^= cellFlagS
			}
			if iy > 0 && inside[ix][iy-1] {
				nswe &^= cellFlagN
			}
			if ix < width-1 && inside[ix+1][iy] {
				nswe &^= cellFlagE
			}
			if ix > 0 && inside[ix-1][iy] {
				nswe &^= cellFlagW
			}
			result[ix][iy] = nswe
		}
	}
	return result
}

func (e *GeoEngine) AddGeoObject(object GeoObject)    { e.toggleGeoObject(object, true) }
func (e *GeoEngine) RemoveGeoObject(object GeoObject) { e.toggleGeoObject(object, false) }

func (e *GeoEngine) toggleGeoObject(object GeoObject, add bool) {
	if object == nil {
		return
	}
	geoData := object.ObjectGeoData()
	if len(geoData) == 0 || len(geoData[0]) == 0 {
		return
	}
	minGX, minGY := object.GeoX(), object.GeoY()
	minBX := minGX / blockCellsX
	maxBX := (minGX + len(geoData) - 1) / blockCellsX
	minBY := minGY / blockCellsY
	maxBY := (minGY + len(geoData[0]) - 1) / blockCellsY

	e.mu.Lock()
	defer e.mu.Unlock()
	for bx := minBX; bx <= maxBX; bx++ {
		for by := minBY; by <= maxBY; by++ {
			r := e.regionOfBlock(bx, by)
			if r == nil {
				continue
			}
			lx, ly := bx%regionBlocksX, by%regionBlocksY
			block := r.blocks[lx][ly]
			dyn, ok := block.(geoDynamic)
			if !ok {
				if _, isNull := block.(*blockNull); isNull {
					continue
				}
				switch t := block.(type) {
				case *blockFlat:
					dyn = newComplexDynamicFromFlat(bx, by, t)
				case *blockComplex:
					dyn = newComplexDynamicFromComplex(bx, by, t)
				case *blockMultilayer:
					dyn = newMultilayerDynamic(bx, by, t)
				default:
					continue
				}
				r.blocks[lx][ly] = dyn
			}
			if add {
				dyn.addGeoObject(object)
			} else {
				dyn.removeGeoObject(object)
			}
		}
	}
}

func (e *GeoEngine) noGeoLine(ox, oy, tx, ty int) bool {
	return !e.HasGeo(ox, oy) && !e.HasGeo(tx, ty)
}

func (e *GeoEngine) CanSeeTarget(ox, oy, oz int, oheight float64, tx, ty, tz int, theight float64, ignore GeoObject) bool {
	return e.canSee(ox, oy, oz, oheight, tx, ty, tz, theight, ignore) &&
		e.canSee(tx, ty, tz, theight, ox, oy, oz, oheight, ignore)
}

func (e *GeoEngine) CanSeeWorld(ox, oy, oz int32, oCol float64, tx, ty, tz int32, tCol float64) bool {
	return e.CanSeeTarget(int(ox), int(oy), int(oz), losHeight(oCol), int(tx), int(ty), int(tz), losHeight(tCol), nil)
}

// canSee is the Java GeoEngine.canSee body that is commented out upstream
// (the Java method currently returns true after the world-bounds check).
func (e *GeoEngine) canSee(ox, oy, oz int, oheight float64, tx, ty, tz int, theight float64, ignore GeoObject) bool {
	if IsOutOfWorld(ox, oy) || IsOutOfWorld(tx, ty) {
		return false
	}
	if e.noGeoLine(ox, oy, tx, ty) {
		return true
	}
	gox, goy := GeoX(ox), GeoY(oy)
	gtx, gty := GeoX(tx), GeoY(ty)
	block := e.block(gox, goy)
	index := block.indexBelow(gox, goy, oz+cellHeight, ignore)
	if index < 0 {
		return false
	}
	if gox == gtx && goy == gty {
		return index == e.block(gtx, gty).indexBelow(gtx, gty, tz+cellHeight, ignore)
	}
	groundZ := int(block.heightAt(index, ignore))
	nswe := block.nsweAt(index, ignore)
	dx := tx - ox
	dy := ty - oy
	dz := (float64(tz) + theight) - (float64(oz) + oheight)
	m := divSlope(dy, dx)
	distXY := math.Sqrt(float64(dx*dx + dy*dy))
	mz := 0.0
	if distXY != 0 {
		mz = dz / distXY
	}
	mdt := directionOf(gtx-gox, gty-goy)
	gridX, gridY := cellGrid(ox), cellGrid(oy)
	for gox != gtx || goy != gty {
		checkX := gridX + mdt.offsetX
		checkY := int(float64(oy) + m*float64(checkX-ox))
		var dir byte
		if mdt.stepX != 0 && GeoY(checkY) == goy {
			gridX += mdt.stepX
			gox += mdt.signumX
			dir = mdt.dirX
		} else {
			checkY = gridY + mdt.offsetY
			if m == 0 {
				checkX = ox
			} else {
				checkX = int(float64(ox) + float64(checkY-oy)/m)
			}
			checkX = clampInt(checkX, gridX, gridX+15)
			gridY += mdt.stepY
			goy += mdt.signumY
			dir = mdt.dirY
		}
		block = e.block(gox, goy)
		losz := float64(oz) + oheight + float64(e.cfg.MaxObstacleHeight)
		losz += mz * math.Hypot(float64(checkX-ox), float64(checkY-oy))
		canMove := nswe&dir != 0
		if canMove {
			index = block.indexBelow(gox, goy, groundZ+cellIgnoreHeight, ignore)
		} else {
			index = block.indexAbove(gox, goy, groundZ-2*cellHeight, ignore)
		}
		if index < 0 {
			return false
		}
		z := int(block.heightAt(index, ignore))
		if float64(z) > losz {
			return false
		}
		groundZ = z
		nswe = block.nsweAt(index, ignore)
	}
	return true
}

func divSlope(dy, dx int) float64 {
	if dx == 0 {
		if dy == 0 {
			return 0
		}
		if dy > 0 {
			return math.Inf(1)
		}
		return math.Inf(-1)
	}
	return float64(dy) / float64(dx)
}

func (e *GeoEngine) CanMoveTo(ox, oy, oz, tx, ty, tz int) bool {
	if IsOutOfWorld(tx, ty) {
		return false
	}
	if e.noGeoLine(ox, oy, tx, ty) {
		return true
	}
	gox, goy := GeoX(ox), GeoY(oy)
	goz := int(e.HeightNearest(gox, goy, oz, nil))
	gtx, gty := GeoX(tx), GeoY(ty)
	if gox == gtx && goy == gty {
		return goz == int(e.Height(tx, ty, tz))
	}
	nswe := int(e.NsweNearest(gox, goy, goz, nil))
	dx, dy := tx-ox, ty-oy
	m := divSlope(dy, dx)
	mdt := directionOf(gtx-gox, gty-goy)
	gridX, gridY := cellGrid(ox), cellGrid(oy)
	nx, ny := gox, goy
	for gox != gtx || goy != gty {
		checkX := gridX + mdt.offsetX
		checkY := int(float64(oy) + m*float64(checkX-ox))
		var dir byte
		if mdt.stepX != 0 && GeoY(checkY) == goy {
			gridX += mdt.stepX
			nx += mdt.signumX
			dir = mdt.dirX
		} else {
			checkY = gridY + mdt.offsetY
			if m == 0 {
				checkX = ox
			} else {
				checkX = int(float64(ox) + float64(checkY-oy)/m)
			}
			checkX = clampInt(checkX, gridX, gridX+15)
			gridY += mdt.stepY
			ny += mdt.signumY
			dir = mdt.dirY
		}
		if nswe&int(dir) == 0 {
			return false
		}
		block := e.block(nx, ny)
		if !block.hasGeoPos() {
			gox, goy = nx, ny
			nswe = int(cellFlagAll)
			continue
		}
		i := block.indexBelow(nx, ny, goz+cellIgnoreHeight, nil)
		if i < 0 {
			return false
		}
		gox, goy = nx, ny
		goz = int(block.heightAt(i, nil))
		nswe = int(block.nsweAt(i, nil))
	}
	return goz == int(e.Height(tx, ty, tz))
}

func (e *GeoEngine) ValidLocation(ox, oy, oz, tx, ty, tz int32) GeoLoc {
	return e.validLocation(int(ox), int(oy), int(oz), int(tx), int(ty), int(tz))
}

func (e *GeoEngine) validLocation(ox, oy, oz, tx, ty, tz int) GeoLoc {
	if e.noGeoLine(ox, oy, tx, ty) {
		return GeoLoc{int32(tx), int32(ty), int32(tz)}
	}
	gox, goy := GeoX(ox), GeoY(oy)
	goz := int(e.HeightNearest(gox, goy, oz, nil))
	nswe := int(e.NsweNearest(gox, goy, goz, nil))
	gtx, gty := GeoX(tx), GeoY(ty)
	gtz := int(e.HeightNearest(gtx, gty, tz, nil))
	dx, dy := tx-ox, ty-oy
	m := divSlope(dy, dx)
	mdt := directionOf(gtx-gox, gty-goy)
	gridX, gridY := cellGrid(ox), cellGrid(oy)
	nx, ny := gox, goy
	for gox != gtx || goy != gty {
		checkX := gridX + mdt.offsetX
		checkY := int(float64(oy) + m*float64(checkX-ox))
		var dir byte
		if mdt.stepX != 0 && GeoY(checkY) == goy {
			gridX += mdt.stepX
			nx += mdt.signumX
			dir = mdt.dirX
		} else {
			checkY = gridY + mdt.offsetY
			if m == 0 {
				checkX = ox
			} else {
				checkX = int(float64(ox) + float64(checkY-oy)/m)
			}
			checkX = clampInt(checkX, gridX, gridX+15)
			gridY += mdt.stepY
			ny += mdt.signumY
			dir = mdt.dirY
		}
		if nx < 0 || nx >= geoCellsX || ny < 0 || ny >= geoCellsY {
			return GeoLoc{int32(checkX), int32(checkY), int32(goz)}
		}
		if nswe&int(dir) == 0 {
			return GeoLoc{int32(checkX), int32(checkY), int32(goz)}
		}
		block := e.block(nx, ny)
		if !block.hasGeoPos() {
			gox, goy = nx, ny
			nswe = int(cellFlagAll)
			continue
		}
		i := block.indexBelow(nx, ny, goz+cellIgnoreHeight, nil)
		if i < 0 {
			return GeoLoc{int32(checkX), int32(checkY), int32(goz)}
		}
		gox, goy = nx, ny
		goz = int(block.heightAt(i, nil))
		nswe = int(block.nsweAt(i, nil))
	}
	if goz == gtz {
		return GeoLoc{int32(tx), int32(ty), int32(gtz)}
	}
	return GeoLoc{int32(ox), int32(oy), int32(oz)}
}

func (e *GeoEngine) CanMoveAround(worldX, worldY, worldZ int) bool {
	geoX, geoY := GeoX(worldX), GeoY(worldY)
	for ix := -1; ix <= 1; ix++ {
		for iy := -1; iy <= 1; iy++ {
			if e.NsweNearest(geoX+ix, geoY+iy, worldZ, nil) != cellFlagAll {
				return false
			}
		}
	}
	return true
}

func (e *GeoEngine) CanFindPathTo(ox, oy, oz, tx, ty, tz int) bool {
	if e.CanMoveTo(ox, oy, oz, tx, ty, tz) {
		return true
	}
	return len(e.FindPath(ox, oy, oz, tx, ty, tz, false)) >= 2
}

func (e *GeoEngine) FindPath(ox, oy, oz, tx, ty, tz int, playable bool) []GeoLoc {
	_ = playable
	if IsOutOfWorld(tx, ty) {
		return nil
	}
	gox, goy := GeoX(ox), GeoY(oy)
	if !e.HasGeoPos(gox, goy) {
		return nil
	}
	goz := int(e.HeightNearest(gox, goy, oz, nil))
	gtx, gty := GeoX(tx), GeoY(ty)
	if !e.HasGeoPos(gtx, gty) {
		return nil
	}
	gtz := int(e.HeightNearest(gtx, gty, tz, nil))
	if absInt(gtz-tz) > 500 {
		return nil
	}
	path := newPathFinder(e).findPath(gox, goy, goz, gtx, gty, gtz)
	if len(path) == 0 {
		return nil
	}
	if len(path) < 3 {
		return path
	}
	nodeAx, nodeAy, nodeAz := ox, oy, goz
	i := 0
	for i < len(path)-1 {
		nodeB := path[i]
		nodeC := path[i+1]
		if e.CanMoveTo(nodeAx, nodeAy, nodeAz, int(nodeC.X), int(nodeC.Y), int(nodeC.Z)) {
			path = append(path[:i], path[i+1:]...)
			continue
		}
		nodeAx, nodeAy, nodeAz = int(nodeB.X), int(nodeB.Y), int(nodeB.Z)
		i++
	}
	return path
}

func (e *GeoEngine) StepToward(ox, oy, oz, tx, ty, tz int32) GeoLoc {
	if e.CanMoveTo(int(ox), int(oy), int(oz), int(tx), int(ty), int(tz)) {
		return GeoLoc{tx, ty, int32(e.Height(int(tx), int(ty), int(tz)))}
	}
	if path := e.FindPath(int(ox), int(oy), int(oz), int(tx), int(ty), int(tz), false); len(path) > 0 {
		return path[0]
	}
	return e.ValidLocation(ox, oy, oz, tx, ty, tz)
}

func (e *GeoEngine) ValidSwimLocation(ox, oy, oz, tx, ty, tz int) GeoLoc {
	if IsOutOfWorld(tx, ty) {
		return GeoLoc{int32(ox), int32(oy), int32(oz)}
	}
	if e.noGeoLine(ox, oy, tx, ty) {
		return GeoLoc{int32(tx), int32(ty), int32(tz)}
	}
	gox, goy := GeoX(ox), GeoY(oy)
	block := e.block(gox, goy)
	index := block.indexBelow(gox, goy, oz+cellHeight, nil)
	gtx, gty := GeoX(tx), GeoY(ty)
	if gox == gtx && goy == gty {
		if index == block.indexBelow(gox, goy, tz+cellHeight, nil) {
			return GeoLoc{int32(tx), int32(ty), int32(tz)}
		}
		return GeoLoc{int32(ox), int32(oy), int32(oz)}
	}
	groundZ := int(block.heightAt(index, nil))
	nswe := block.nsweAt(index, nil)
	dx, dy, dz := tx-ox, ty-oy, tz-oz
	m := divSlope(dy, dx)
	distXY := math.Sqrt(float64(dx*dx + dy*dy))
	mz := 0.0
	if distXY != 0 {
		mz = float64(dz) / distXY
	}
	mdt := directionOf(gtx-gox, gty-goy)
	gridX, gridY := cellGrid(ox), cellGrid(oy)
	for gox != gtx || goy != gty {
		checkX := gridX + mdt.offsetX
		checkY := int(float64(oy) + m*float64(checkX-ox))
		var dir byte
		if mdt.stepX != 0 && GeoY(checkY) == goy {
			gridX += mdt.stepX
			gox += mdt.signumX
			dir = mdt.dirX
		} else {
			checkY = gridY + mdt.offsetY
			if m == 0 {
				checkX = ox
			} else {
				checkX = int(float64(ox) + float64(checkY-oy)/m)
			}
			checkX = clampInt(checkX, gridX, gridX+15)
			gridY += mdt.stepY
			goy += mdt.signumY
			dir = mdt.dirY
		}
		block = e.block(gox, goy)
		swimZ := float64(oz) + mz*math.Hypot(float64(checkX-ox), float64(checkY-oy))
		canMove := nswe&dir != 0
		if canMove {
			index = block.indexBelow(gox, goy, groundZ+cellIgnoreHeight, nil)
		} else {
			index = block.indexAbove(gox, goy, groundZ-2*cellHeight, nil)
		}
		if index < 0 {
			return GeoLoc{int32(gridX), int32(gridY), int32(swimZ)}
		}
		z := int(block.heightAt(index, nil))
		if canMove {
			if float64(z) >= swimZ {
				groundZ = z
				nswe = block.nsweAt(index, nil)
				continue
			}
		} else if float64(z) > swimZ {
			return GeoLoc{int32(checkX), int32(checkY), int32(swimZ)}
		}
		index = block.indexBelow(gox, goy, int(swimZ), nil)
		groundZ = int(block.heightAt(index, nil))
		nswe = block.nsweAt(index, nil)
	}
	return GeoLoc{int32(tx), int32(ty), int32(tz)}
}

func (e *GeoEngine) CanFlyTo(ox, oy, oz int, oheight float64, tx, ty, tz int) bool {
	if IsOutOfWorld(tx, ty) {
		return false
	}
	if e.noGeoLine(ox, oy, tx, ty) {
		return true
	}
	gox, goy := GeoX(ox), GeoY(oy)
	gtx, gty := GeoX(tx), GeoY(ty)
	dx, dy, dz := tx-ox, ty-oy, tz-oz
	m := divSlope(dy, dx)
	distXY := math.Sqrt(float64(dx*dx + dy*dy))
	mz := 0.0
	if distXY != 0 {
		mz = float64(dz) / distXY
	}
	mdt := directionOf(gtx-gox, gty-goy)
	gridX, gridY := cellGrid(ox), cellGrid(oy)
	for gox != gtx || goy != gty {
		checkX := gridX + mdt.offsetX
		checkY := int(float64(oy) + m*float64(checkX-ox))
		if mdt.stepX != 0 && GeoY(checkY) == goy {
			gridX += mdt.stepX
			gox += mdt.signumX
		} else {
			checkY = gridY + mdt.offsetY
			if m == 0 {
				checkX = ox
			} else {
				checkX = int(float64(ox) + float64(checkY-oy)/m)
			}
			checkX = clampInt(checkX, gridX, gridX+15)
			gridY += mdt.stepY
			goy += mdt.signumY
		}
		block := e.block(gox, goy)
		nextZ := oz + int(mz*math.Hypot(float64(checkX-ox), float64(checkY-oy)))
		index := block.indexBelow(gox, goy, nextZ+int(oheight), nil)
		if index < 0 {
			return false
		}
		goz := int(block.heightAt(index, nil))
		if goz > nextZ {
			return false
		}
		index = block.indexAbove(gox, goy, nextZ, nil)
		nextZ += int(oheight)
		if index >= 0 {
			goz = int(block.heightAt(index, nil))
			if goz < nextZ {
				return false
			}
		}
	}
	return true
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
