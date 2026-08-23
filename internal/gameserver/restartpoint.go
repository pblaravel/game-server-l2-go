package gameserver

import (
	"encoding/xml"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
)

// Java World tile constants used by RestartPointData.getRestartPoint and GeoEngine.
const (
	tileXMin  = 16
	tileXMax  = 26
	tileYMin  = 10
	tileYMax  = 25
	tileSize  = 32768
	worldXMin = (tileXMin - 20) * tileSize
	worldXMax = (tileXMax-19)*tileSize - 1
	worldYMin = (tileYMin - 18) * tileSize
	worldYMax = (tileYMax-17)*tileSize - 1
	worldZMax = 16410
)

// RestartPoint is Java model/restart/RestartPoint (town spawn list + map tiles).
type RestartPoint struct {
	Name   string
	Points [][3]int32
	Maps   [][2]int32
}

// RestartArea is Java model/restart/RestartArea: a polygon that overrides the
// geomap lookup by race.
type RestartArea struct {
	MinZ, MaxZ int32
	Xs, Ys     []int32
	RaceZone   map[int32]string
}

var (
	restartMu    sync.RWMutex
	restartPts   []RestartPoint
	restartAreas []RestartArea
)

func RestartPointCount() int {
	restartMu.RLock()
	defer restartMu.RUnlock()
	return len(restartPts)
}

type xmlRestartRoot struct {
	Areas  []xmlRestartArea  `xml:"area"`
	Points []xmlRestartPoint `xml:"point"`
}

type xmlRestartArea struct {
	MinZ     int32            `xml:"minZ,attr"`
	MaxZ     int32            `xml:"maxZ,attr"`
	Nodes    []xmlRestartNode `xml:"node"`
	Restarts []xmlRaceRestart `xml:"restart"`
}

type xmlRestartNode struct {
	X int32 `xml:"x,attr"`
	Y int32 `xml:"y,attr"`
}

type xmlRaceRestart struct {
	Race string `xml:"race,attr"`
	Zone string `xml:"zone,attr"`
}

type xmlRestartPoint struct {
	Sets []xmlSet `xml:"set"`
}

func loadRestartXML(path string) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var root xmlRestartRoot
	if err := xml.Unmarshal(body, &root); err != nil {
		return fmt.Errorf("restartPointAreas: %w", err)
	}
	pts := make([]RestartPoint, 0, len(root.Points))
	for _, p := range root.Points {
		rp := RestartPoint{}
		for _, s := range p.Sets {
			switch s.Name {
			case "name":
				rp.Name = s.Val
			case "point":
				if loc, ok := parseXYZ(s.Val); ok {
					rp.Points = append(rp.Points, loc)
				}
			case "map":
				if m, ok := parseMapTile(s.Val); ok {
					rp.Maps = append(rp.Maps, m)
				}
			}
		}
		if rp.Name != "" && len(rp.Points) > 0 {
			pts = append(pts, rp)
		}
	}
	areas := make([]RestartArea, 0, len(root.Areas))
	for _, a := range root.Areas {
		area := RestartArea{MinZ: a.MinZ, MaxZ: a.MaxZ, RaceZone: map[int32]string{}}
		for _, n := range a.Nodes {
			area.Xs = append(area.Xs, n.X)
			area.Ys = append(area.Ys, n.Y)
		}
		for _, r := range a.Restarts {
			area.RaceZone[raceOrdinal(r.Race)] = r.Zone
		}
		if len(area.Xs) >= 3 {
			areas = append(areas, area)
		}
	}
	restartMu.Lock()
	restartPts = pts
	restartAreas = areas
	restartMu.Unlock()
	return nil
}

func parseXYZ(val string) ([3]int32, bool) {
	parts := strings.Split(val, ";")
	if len(parts) < 3 {
		return [3]int32{}, false
	}
	x, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	y, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	z, err3 := strconv.Atoi(strings.TrimSpace(parts[2]))
	if err1 != nil || err2 != nil || err3 != nil {
		return [3]int32{}, false
	}
	return [3]int32{int32(x), int32(y), int32(z)}, true
}

func parseMapTile(val string) ([2]int32, bool) {
	parts := strings.Split(val, ";")
	if len(parts) < 2 {
		return [2]int32{}, false
	}
	x, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	y, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err1 != nil || err2 != nil {
		return [2]int32{}, false
	}
	return [2]int32{int32(x), int32(y)}, true
}

func raceOrdinal(name string) int32 {
	switch strings.ToUpper(name) {
	case "HUMAN":
		return 0
	case "ELF":
		return 1
	case "DARK_ELF", "DARKELF":
		return 2
	case "ORC":
		return 3
	case "DWARF":
		return 4
	default:
		return 0
	}
}

func mapTileOf(x, y int32) (int32, int32) {
	rx := (x-worldXMin)/tileSize + tileXMin
	ry := (y-worldYMin)/tileSize + tileYMin
	return rx, ry
}

func restartPointByName(name string) *RestartPoint {
	for i := range restartPts {
		if strings.EqualFold(restartPts[i].Name, name) {
			return &restartPts[i]
		}
	}
	return nil
}

func restartPointByTile(rx, ry int32) *RestartPoint {
	for i := range restartPts {
		for _, m := range restartPts[i].Maps {
			if m[0] == rx && m[1] == ry {
				return &restartPts[i]
			}
		}
	}
	return nil
}

func pointInArea(a RestartArea, x, y, z int32) bool {
	if z < a.MinZ || z > a.MaxZ || len(a.Xs) < 3 {
		return false
	}
	inside := false
	j := len(a.Xs) - 1
	for i := 0; i < len(a.Xs); i++ {
		xi, yi := a.Xs[i], a.Ys[i]
		xj, yj := a.Xs[j], a.Ys[j]
		if (yi > y) != (yj > y) && x < (xj-xi)*(y-yi)/(yj-yi)+xi {
			inside = !inside
		}
		j = i
	}
	return inside
}

// NearestRestartLocation is Java RestartPointData.getNearestRestartLocation
// without the clan-hall / castle / siege branches.
func NearestRestartLocation(p *Character) [3]int32 {
	restartMu.RLock()
	defer restartMu.RUnlock()
	if len(restartPts) == 0 {
		return nearestTownFallback(p.X, p.Y, p.Z)
	}
	for _, area := range restartAreas {
		if !pointInArea(area, p.X, p.Y, p.Z) {
			continue
		}
		if name := area.RaceZone[p.Race]; name != "" {
			if rp := restartPointByName(name); rp != nil && len(rp.Points) > 0 {
				return rp.Points[0]
			}
		}
	}
	rx, ry := mapTileOf(p.X, p.Y)
	if rp := restartPointByTile(rx, ry); rp != nil && len(rp.Points) > 0 {
		return rp.Points[0]
	}
	return nearestTownFallback(p.X, p.Y, p.Z)
}

func nearestTownFallback(x, y, z int32) [3]int32 {
	return nearestRespawn(x, y, z)
}
