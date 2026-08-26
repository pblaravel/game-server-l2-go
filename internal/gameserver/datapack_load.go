package gameserver

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// FindDataDir locates Java-style data/xml (skills + classes).
func FindDataDir() string {
	candidates := []string{"data", "./data"}
	if wd, err := os.Getwd(); err == nil {
		dir := wd
		for i := 0; i < 6; i++ {
			candidates = append(candidates, filepath.Join(dir, "data"))
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	for _, c := range candidates {
		if _, err := os.Stat(filepath.Join(c, "xml", "skills")); err == nil {
			return c
		}
	}
	return "data"
}

// LoadDatapack parses the Java XML loaders that are vendored under data/xml.
func LoadDatapack(dataDir string) error {
	if dataDir == "" {
		dataDir = FindDataDir()
	}
	if err := loadPlayerLevelXML(filepath.Join(dataDir, "xml", "playerLevels.xml")); err != nil {
		log.Printf("datapack player levels: %v (hardcoded exp table stays in use)", err)
	}
	if err := loadSkillXML(filepath.Join(dataDir, "xml", "skills")); err != nil {
		return fmt.Errorf("skills: %w", err)
	}
	if err := loadClassXML(filepath.Join(dataDir, "xml", "classes")); err != nil {
		return fmt.Errorf("classes: %w", err)
	}
	if err := loadItemXML(filepath.Join(dataDir, "xml", "items")); err != nil {
		log.Printf("datapack items: %v (starter item fallbacks stay in use)", err)
	}
	if err := loadNpcXML(filepath.Join(dataDir, "xml", "npcs")); err != nil {
		log.Printf("datapack npcs: %v (newbie spawn fallbacks stay in use)", err)
	}
	if err := loadBuyListXML(filepath.Join(dataDir, "xml", "buyLists.xml")); err != nil {
		log.Printf("datapack buylists: %v", err)
	}
	if err := loadTeleportXML(filepath.Join(dataDir, "xml", "teleports.xml")); err != nil {
		log.Printf("datapack teleports: %v", err)
	}
	if err := loadRestartXML(filepath.Join(dataDir, "xml", "restartPointAreas.xml")); err != nil {
		log.Printf("datapack restart points: %v", err)
	}
	loadExtraDatapack(dataDir)
	if err := LoadGeoEngine(dataDir); err != nil {
		log.Printf("datapack geodata: %v", err)
	}
	attachDoorGeo()
	log.Printf("datapack: %d skill levels, %d classes, %d items, %d npcs, %d buylists, %d teleport npcs, %d restart points, %d player levels, %d recipes, %d armor sets, %d zones, %d doors, %d hennas, %d multisell, %d geo regions, %d clan hall decos, %d manor areas, %d manor crops, %d observer entries, %d augment stats",
		SkillCount(), ClassCount(), ItemCount(), NpcTemplateCount(), BuyListCount(), TeleportListCount(), RestartPointCount(), PlayerLevelCount(),
		RecipeCount(), ArmorSetCount(), ZoneCount(), DoorCount(), HennaCount(), MultisellCount(), Geo().LoadedRegions(),
		ClanHallDecoCount(), ManorAreaCount(), ManorCropCount(), ObserverEntryCount(), AugmentStatCount())
	return nil
}

func DatapackLoaded() bool {
	skillMu.RLock()
	okSkills := skillsLoaded
	skillMu.RUnlock()
	classMu.RLock()
	okClass := classesLoaded
	classMu.RUnlock()
	return okSkills && okClass
}
