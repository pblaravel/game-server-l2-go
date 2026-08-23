package gameserver

import "log"

// DoorInstance is a spawned Java Door used as IGeoObject.
type DoorInstance struct {
	ID      int32
	Name    string
	Opened  bool
	geoX    int
	geoY    int
	geoZ    int
	height  int
	geoData [][]byte
}

func (d *DoorInstance) GeoX() int               { return d.geoX }
func (d *DoorInstance) GeoY() int               { return d.geoY }
func (d *DoorInstance) GeoZ() int               { return d.geoZ }
func (d *DoorInstance) Height() int             { return d.height }
func (d *DoorInstance) ObjectGeoData() [][]byte { return d.geoData }

var liveDoors []*DoorInstance

func LiveDoors() []*DoorInstance { return liveDoors }

func polygonContains(xs, ys []int, x, y int) bool {
	inside := false
	j := len(xs) - 1
	for i := 0; i < len(xs); i++ {
		xi, yi := xs[i], ys[i]
		xj, yj := xs[j], ys[j]
		if (yi > y) != (yj > y) && yj != yi && x < (xj-xi)*(y-yi)/(yj-yi)+xi {
			inside = !inside
		}
		j = i
	}
	return inside
}

// attachDoorGeo is Java DoorData.parseDocument geo footprint + spawn addGeoObject.
func attachDoorGeo() {
	e := Geo()
	worldMu.RLock()
	templates := make([]DoorTemplate, 0, len(doors))
	for _, d := range doors {
		templates = append(templates, d)
	}
	worldMu.RUnlock()

	next := make([]*DoorInstance, 0, len(templates))
	for _, tpl := range templates {
		inst := buildDoorGeo(e, tpl)
		if inst == nil {
			continue
		}
		next = append(next, inst)
		if !inst.Opened {
			e.AddGeoObject(inst)
		}
	}
	liveDoors = next
	if n := len(next); n > 0 {
		log.Printf("geodata: registered %d door geo objects", n)
	}
}

func buildDoorGeo(e *GeoEngine, tpl DoorTemplate) *DoorInstance {
	if len(tpl.Coords) < 3 {
		return nil
	}
	minX, maxX := tpl.Coords[0][0], tpl.Coords[0][0]
	minY, maxY := tpl.Coords[0][1], tpl.Coords[0][1]
	xs := make([]int, len(tpl.Coords))
	ys := make([]int, len(tpl.Coords))
	for i, c := range tpl.Coords {
		xs[i], ys[i] = int(c[0]), int(c[1])
		if c[0] < minX {
			minX = c[0]
		}
		if c[0] > maxX {
			maxX = c[0]
		}
		if c[1] < minY {
			minY = c[1]
		}
		if c[1] > maxY {
			maxY = c[1]
		}
	}
	if IsOutOfWorld(int(minX), int(minY)) || IsOutOfWorld(int(maxX), int(maxY)) {
		return nil
	}
	x := GeoX(int(minX)) - 1
	y := GeoY(int(minY)) - 1
	sizeX := (GeoX(int(maxX)) + 1) - x + 1
	sizeY := (GeoY(int(maxY)) + 1) - y + 1
	if sizeX <= 0 || sizeY <= 0 || sizeX > 256 || sizeY > 256 {
		return nil
	}
	geoX := GeoX(int(tpl.X))
	geoY := GeoY(int(tpl.Y))
	geoZ := int(e.HeightNearest(geoX, geoY, int(tpl.Z), nil))
	height := int(tpl.Height)
	block := e.block(geoX, geoY)
	if i := block.indexAbove(geoX, geoY, geoZ, nil); i >= 0 {
		layerDiff := int(block.heightAt(i, nil)) - geoZ
		if height > layerDiff {
			height = layerDiff - cellIgnoreHeight
		}
	}
	limit := cellIgnoreHeight
	if tpl.Type == "WALL" {
		limit = cellIgnoreHeight * 4
	}
	inside := make([][]bool, sizeX)
	for ix := 0; ix < sizeX; ix++ {
		inside[ix] = make([]bool, sizeY)
		for iy := 0; iy < sizeY; iy++ {
			gx, gy := x+ix, y+iy
			z := int(e.HeightNearest(gx, gy, int(tpl.Z), nil))
			if absInt(z-int(tpl.Z)) > limit {
				continue
			}
			worldX, worldY := WorldX(gx), WorldY(gy)
		cell:
			for wix := worldX - 6; wix <= worldX+6; wix += 2 {
				for wiy := worldY - 6; wiy <= worldY+6; wiy += 2 {
					if polygonContains(xs, ys, wix, wiy) {
						inside[ix][iy] = true
						break cell
					}
				}
			}
		}
	}
	return &DoorInstance{
		ID: tpl.ID, Name: tpl.Name, Opened: tpl.Opened,
		geoX: x, geoY: y, geoZ: geoZ, height: height,
		geoData: CalculateGeoObject(inside),
	}
}

func (d *DoorInstance) Open(e *GeoEngine) {
	if d.Opened {
		return
	}
	d.Opened = true
	e.RemoveGeoObject(d)
}

func (d *DoorInstance) Close(e *GeoEngine) {
	if !d.Opened {
		return
	}
	d.Opened = false
	e.AddGeoObject(d)
}
