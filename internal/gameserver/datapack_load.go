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

// LoadDatapack parses Java SkillTable + PlayerData XML.
func LoadDatapack(dataDir string) error {
	if dataDir == "" {
		dataDir = FindDataDir()
	}
	skillDir := filepath.Join(dataDir, "xml", "skills")
	classDir := filepath.Join(dataDir, "xml", "classes")
	if err := loadSkillXML(skillDir); err != nil {
		return fmt.Errorf("skills: %w", err)
	}
	if err := loadClassXML(classDir); err != nil {
		return fmt.Errorf("classes: %w", err)
	}
	log.Printf("datapack: %d skill levels, %d class templates", SkillCount(), ClassCount())
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
