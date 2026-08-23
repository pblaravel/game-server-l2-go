package gameserver

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/pblaravel/game-server-l2-go/internal/config"
)

// GeoStructure / World constants from Java geoengine.geodata.GeoStructure and model.World.
const (
	cellFlagNone byte = 0x00
	cellFlagE    byte = 0x01
	cellFlagW    byte = 0x02
	cellFlagS    byte = 0x04
	cellFlagN    byte = 0x08
	cellFlagAll  byte = 0x0F

	cellSize         = 16
	cellHeight       = 8
	cellIgnoreHeight = cellHeight * 6

	typeFlatL2JL2OFF  byte = 0
	typeComplexL2J    byte = 1
	typeComplexL2OFF       = 0x40
	typeMultilayerL2J byte = 2

	blockCellsX = 8
	blockCellsY = 8
	blockCells  = blockCellsX * blockCellsY

	regionBlocksX = 256
	regionBlocksY = 256

	maxLayers = 127
)

var (
	geoRegionsX = tileXMax - tileXMin + 1
	geoRegionsY = tileYMax - tileYMin + 1
	geoBlocksX  = geoRegionsX * regionBlocksX
	geoBlocksY  = geoRegionsY * regionBlocksY
	geoCellsX   = geoBlocksX * blockCellsX
	geoCellsY   = geoBlocksY * blockCellsY
)

// GeoType is Java enums.GeoType.
type GeoType int

const (
	GeoL2OFF GeoType = iota
	GeoL2J
)

func (t GeoType) filename(rx, ry int) string {
	if t == GeoL2J {
		return fmt.Sprintf("%d_%d.l2j", rx, ry)
	}
	return fmt.Sprintf("%d_%d_conv.dat", rx, ry)
}

func (t GeoType) String() string {
	if t == GeoL2J {
		return "L2J"
	}
	return "L2OFF"
}

// GeoLoc is Java model.location.Location for geo results.
type GeoLoc struct {
	X, Y, Z int32
}

// GeoObject is Java geodata.IGeoObject (doors and other static geo modifiers).
type GeoObject interface {
	GeoX() int
	GeoY() int
	GeoZ() int
	Height() int
	ObjectGeoData() [][]byte
}

// GeoConfig is Java Config.loadGeoengine.
type GeoConfig struct {
	Path                  string
	Type                  GeoType
	PartOfCharacterHeight int
	MaxObstacleHeight     int
	MoveWeight            int
	MoveWeightDiag        int
	ObstacleWeight        int
	ObstacleWeightDiag    int
	HeuristicWeight       int
	MaxIterations         int
	MaxGeopathFailCount   int
	Enabled               map[string]bool
}

func defaultGeoConfig() GeoConfig {
	return GeoConfig{
		Path:                  "data/geodata/",
		Type:                  GeoL2OFF,
		PartOfCharacterHeight: 75,
		MaxObstacleHeight:     32,
		MoveWeight:            10,
		MoveWeightDiag:        14,
		ObstacleWeight:        30,
		ObstacleWeightDiag:    42, // 30 * sqrt(2) ≈ 42
		HeuristicWeight:       12,
		MaxIterations:         10000,
		MaxGeopathFailCount:   50,
		Enabled:               map[string]bool{},
	}
}

func loadGeoConfig(dataDir string) GeoConfig {
	cfg := defaultGeoConfig()
	paths := []string{
		filepath.Join(dataDir, "geodata", "geoengine.properties"),
		"data/geodata/geoengine.properties",
		"conf/geoengine.properties",
	}
	p, err := config.LoadProperties(paths...)
	if err != nil && len(p) == 0 {
		return cfg
	}
	cfg.Path = p.String("GeoDataPath", cfg.Path)
	if t := p.String("GeoDataType", "L2OFF"); t == "L2J" {
		cfg.Type = GeoL2J
	}
	cfg.PartOfCharacterHeight = p.Int("PartOfCharacterHeight", cfg.PartOfCharacterHeight)
	cfg.MaxObstacleHeight = p.Int("MaxObstacleHeight", cfg.MaxObstacleHeight)
	cfg.MoveWeight = p.Int("MoveWeight", cfg.MoveWeight)
	cfg.MoveWeightDiag = p.Int("MoveWeightDiag", cfg.MoveWeightDiag)
	cfg.ObstacleWeight = p.Int("ObstacleWeight", cfg.ObstacleWeight)
	cfg.ObstacleWeightDiag = int(float64(cfg.ObstacleWeight) * 1.414213562)
	cfg.HeuristicWeight = p.Int("HeuristicWeight", cfg.HeuristicWeight)
	cfg.MaxIterations = p.Int("MaxIterations", cfg.MaxIterations)
	cfg.MaxGeopathFailCount = p.Int("MaxGeopathFailCount", cfg.MaxGeopathFailCount)
	if cfg.MaxGeopathFailCount < 15 {
		cfg.MaxGeopathFailCount = 15
	}
	cfg.Enabled = map[string]bool{}
	for rx := tileXMin; rx <= tileXMax; rx++ {
		for ry := tileYMin; ry <= tileYMax; ry++ {
			key := fmt.Sprintf("%d_%d", rx, ry)
			if _, ok := p[key]; ok {
				cfg.Enabled[key] = true
			}
		}
	}
	return cfg
}

type geoRegion struct {
	blocks [regionBlocksX][regionBlocksY]geoBlock
}

// GeoEngine is Java geoengine.GeoEngine.
type GeoEngine struct {
	mu      sync.RWMutex
	cfg     GeoConfig
	regions [][]*geoRegion
	nullBlk *blockNull
	loaded  int
	failed  int
}

func newGeoEngine(cfg GeoConfig) *GeoEngine {
	e := &GeoEngine{
		cfg:     cfg,
		nullBlk: &blockNull{},
		regions: make([][]*geoRegion, geoRegionsX),
	}
	for i := range e.regions {
		e.regions[i] = make([]*geoRegion, geoRegionsY)
	}
	return e
}

func (e *GeoEngine) ensureRegion(rx, ry int) *geoRegion {
	ix := rx - tileXMin
	iy := ry - tileYMin
	if e.regions[ix][iy] == nil {
		r := &geoRegion{}
		for x := 0; x < regionBlocksX; x++ {
			for y := 0; y < regionBlocksY; y++ {
				r.blocks[x][y] = e.nullBlk
			}
		}
		e.regions[ix][iy] = r
	}
	return e.regions[ix][iy]
}

func (e *GeoEngine) regionOfBlock(bx, by int) *geoRegion {
	rx := bx / regionBlocksX
	ry := by / regionBlocksY
	if rx < 0 || ry < 0 || rx >= geoRegionsX || ry >= geoRegionsY {
		return nil
	}
	return e.regions[rx][ry]
}

func (e *GeoEngine) block(geoX, geoY int) geoBlock {
	if geoX < 0 || geoY < 0 || geoX >= geoCellsX || geoY >= geoCellsY {
		return e.nullBlk
	}
	bx := geoX / blockCellsX
	by := geoY / blockCellsY
	r := e.regionOfBlock(bx, by)
	if r == nil {
		return e.nullBlk
	}
	return r.blocks[bx%regionBlocksX][by%regionBlocksY]
}

func (e *GeoEngine) setBlock(bx, by int, b geoBlock) {
	rx := bx/regionBlocksX + tileXMin
	ry := by/regionBlocksY + tileYMin
	r := e.ensureRegion(rx, ry)
	r.blocks[bx%regionBlocksX][by%regionBlocksY] = b
}

func GeoX(worldX int) int { return (worldX - worldXMin) >> 4 }
func GeoY(worldY int) int { return (worldY - worldYMin) >> 4 }
func WorldX(geoX int) int { return (geoX << 4) + worldXMin + 8 }
func WorldY(geoY int) int { return (geoY << 4) + worldYMin + 8 }

func IsOutOfWorld(x, y int) bool {
	return x < worldXMin || x > worldXMax || y < worldYMin || y > worldYMax
}

func cellGrid(v int) int { return int(int32(v) & -16) }

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func cellHeightAt(buf []byte, index int) int16 {
	lo := int(buf[index+1])
	hi := int(int8(buf[index+2]))
	return int16(lo | hi<<8)
}

func putCellHeight(buf []byte, index int, h int16) {
	buf[index+1] = byte(h)
	buf[index+2] = byte(h >> 8)
}

func unpackGeoShort(data int16) (nswe byte, height int16) {
	nswe = byte(data & 0x000F)
	masked := int16(int(data) & 0xFFF0)
	return nswe, masked >> 1
}

func packGeoShort(nswe byte, height int16) int16 {
	return int16(int(height)<<1)&-16 | int16(nswe&0x0F)
}

var (
	geoMu     sync.RWMutex
	geoEngine *GeoEngine
)

func Geo() *GeoEngine {
	geoMu.RLock()
	e := geoEngine
	geoMu.RUnlock()
	if e != nil {
		return e
	}
	geoMu.Lock()
	defer geoMu.Unlock()
	if geoEngine == nil {
		geoEngine = newGeoEngine(defaultGeoConfig())
	}
	return geoEngine
}

func setGeoEngine(e *GeoEngine) {
	geoMu.Lock()
	geoEngine = e
	geoMu.Unlock()
}

// LoadGeoEngine is Java GeoEngine constructor: load listed region files, Null elsewhere.
// Missing files do not abort the process (the repo does not vendor binary geodata).
func LoadGeoEngine(dataDir string) error {
	if dataDir == "" {
		dataDir = FindDataDir()
	}
	cfg := loadGeoConfig(dataDir)
	if !filepath.IsAbs(cfg.Path) {
		cfg.Path = filepath.Join(dataDir, "geodata") + string(os.PathSeparator)
	}
	e := newGeoEngine(cfg)
	for rx := tileXMin; rx <= tileXMax; rx++ {
		for ry := tileYMin; ry <= tileYMax; ry++ {
			key := fmt.Sprintf("%d_%d", rx, ry)
			if !cfg.Enabled[key] {
				continue
			}
			if e.loadGeoBlocks(rx, ry) {
				e.loaded++
			} else {
				e.failed++
			}
		}
	}
	if e.loaded > 0 {
		log.Printf("geodata: loaded %d %s region files", e.loaded, cfg.Type)
	} else {
		log.Printf("geodata: no region files under %s (Null blocks, movement unrestricted)", cfg.Path)
	}
	if e.failed > 0 {
		log.Printf("geodata: failed to load %d listed %s region files", e.failed, cfg.Type)
	}
	setGeoEngine(e)
	return nil
}

func (e *GeoEngine) loadGeoBlocks(regionX, regionY int) bool {
	name := e.cfg.Type.filename(regionX, regionY)
	path := filepath.Join(e.cfg.Path, name)
	raw, err := os.ReadFile(path)
	if err != nil {
		log.Printf("geodata: missing %s", path)
		return false
	}
	if err := e.parseRegion(regionX, regionY, raw, e.cfg.Type); err != nil {
		log.Printf("geodata: %s: %v", name, err)
		return false
	}
	return true
}

func (e *GeoEngine) parseRegion(regionX, regionY int, raw []byte, typ GeoType) error {
	off := 0
	if typ == GeoL2OFF {
		if len(raw) < 18 {
			return fmt.Errorf("truncated L2OFF header")
		}
		off = 18
	}
	e.ensureRegion(regionX, regionY)
	blockX := (regionX - tileXMin) * regionBlocksX
	blockY := (regionY - tileYMin) * regionBlocksY
	mlTemp := make([]byte, 0, blockCells*maxLayers*3)
	for ix := 0; ix < regionBlocksX; ix++ {
		for iy := 0; iy < regionBlocksY; iy++ {
			var b geoBlock
			var err error
			if typ == GeoL2J {
				if off >= len(raw) {
					return fmt.Errorf("truncated L2J block type at %d,%d", ix, iy)
				}
				kind := raw[off]
				off++
				switch kind {
				case typeFlatL2JL2OFF:
					b, off, err = readBlockFlat(raw, off, typ)
				case typeComplexL2J:
					b, off, err = readBlockComplex(raw, off)
				case typeMultilayerL2J:
					b, off, mlTemp, err = readBlockMultilayer(raw, off, typ, mlTemp)
				default:
					return fmt.Errorf("unknown L2J block type %d", kind)
				}
			} else {
				if off+2 > len(raw) {
					return fmt.Errorf("truncated L2OFF block type at %d,%d", ix, iy)
				}
				kind := int16(binary.LittleEndian.Uint16(raw[off:]))
				off += 2
				switch kind {
				case int16(typeFlatL2JL2OFF):
					b, off, err = readBlockFlat(raw, off, typ)
				case typeComplexL2OFF:
					b, off, err = readBlockComplex(raw, off)
				default:
					b, off, mlTemp, err = readBlockMultilayer(raw, off, typ, mlTemp)
				}
			}
			if err != nil {
				return err
			}
			e.setBlock(blockX+ix, blockY+iy, b)
		}
	}
	if rem := len(raw) - off; rem > 0 {
		log.Printf("geodata: region %d_%d has %d trailing bytes", regionX, regionY, rem)
	}
	return nil
}

func (e *GeoEngine) HasGeoPos(geoX, geoY int) bool { return e.block(geoX, geoY).hasGeoPos() }
func (e *GeoEngine) HasGeo(worldX, worldY int) bool {
	return e.HasGeoPos(GeoX(worldX), GeoY(worldY))
}

func (e *GeoEngine) HeightNearest(geoX, geoY, worldZ int, ignore GeoObject) int16 {
	return e.block(geoX, geoY).heightNearest(geoX, geoY, worldZ, ignore)
}

func (e *GeoEngine) NsweNearest(geoX, geoY, worldZ int, ignore GeoObject) byte {
	return e.block(geoX, geoY).nsweNearest(geoX, geoY, worldZ, ignore)
}

func (e *GeoEngine) Height(worldX, worldY, worldZ int) int32 {
	return int32(e.HeightNearest(GeoX(worldX), GeoY(worldY), worldZ, nil))
}

func (e *GeoEngine) LoadedRegions() int { return e.loaded }

func losHeight(collision float64) float64 {
	return collision * 2 * float64(Geo().cfg.PartOfCharacterHeight) / 100
}
