package gameserver

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func loadExtraDatapack(dataDir string) {
	xml := filepath.Join(dataDir, "xml")
	type loader struct {
		name string
		path string
		fn   func(string) error
	}
	for _, l := range []loader{
		{"armor sets", "armorSets.xml", loadArmorSetXML},
		{"spellbooks", "spellbooks.xml", loadSpellbookXML},
		{"heal sps", "healSps.xml", loadHealSpsXML},
		{"newbie buffs", "newbieBuffs.xml", loadNewbieBuffXML},
		{"instant teleports", "instantTeleports.xml", loadInstantTeleportXML},
		{"announcements", "announcements.xml", loadAnnouncementXML},
		{"hennas", "hennas.xml", loadHennaXML},
		{"recipes", "recipes.xml", loadRecipeXML},
		{"summon items", "summonItems.xml", loadSummonItemXML},
		{"cursed weapons", "cursedWeapons.xml", loadCursedWeaponXML},
		{"soul crystals", "soulCrystals.xml", loadSoulCrystalXML},
		{"fish", "fish.xml", loadFishXML},
		{"static objects", "staticObjects.xml", loadStaticObjectXML},
		{"walker routes", "walkerRoutes.xml", loadWalkerRouteXML},
		{"boats", "boatRoutes.xml", loadBoatXML},
		{"buffer skills", "bufferSkills.xml", loadBufferXML},
		{"access levels", "accessLevels.xml", loadAccessLevelXML},
		{"admin commands", "adminCommands.xml", loadAdminCommandXML},
		{"scripts", "scripts.xml", loadScriptIndexXML},
		{"doors", "doors.xml", loadDoorXML},
		{"castles", "castles.xml", loadCastleXML},
		{"clan halls", "clanHalls.xml", loadClanHallXML},
		{"clan hall deco", "clanHallDeco.xml", loadClanHallDecoXML},
		{"manor areas", "manorAreas.xml", loadManorAreaXML},
		{"manors", "manors.xml", loadManorXML},
		{"observer groups", "observerGroups.xml", loadObserverXML},
		{"skill trees", "skillstrees", loadSkillTreeXML},
		{"multisell", "multisell", loadMultisellXML},
		{"zones", "zones", loadZoneXML},
		{"spawnlist", "spawnlist", loadSpawnXML},
		{"augmentation", "augmentation", loadAugmentationXML},
	} {
		if err := l.fn(filepath.Join(xml, l.path)); err != nil && !os.IsNotExist(err) {
			log.Printf("datapack %s: %v", l.name, err)
		}
	}
}

func walkXMLFiles(path string, handle func(name string, body []byte) error) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return handle(filepath.Base(path), body)
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".xml") {
			continue
		}
		body, err := os.ReadFile(filepath.Join(path, e.Name()))
		if err != nil {
			return err
		}
		if err := handle(e.Name(), body); err != nil {
			return err
		}
	}
	return nil
}

func parseCSVInts(raw string) []int32 {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ";")
	out := make([]int32, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			continue
		}
		out = append(out, int32(n))
	}
	return out
}

func parseIDCountPairs(raw string) [][2]int32 {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ";")
	out := make([][2]int32, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		id, cnt, ok := parseSkillRef(p)
		if !ok {
			continue
		}
		out = append(out, [2]int32{id, cnt})
	}
	return out
}

// javaStringHash is java.lang.String.hashCode for ASCII filenames.
func javaStringHash(s string) int32 {
	var h int32
	for i := 0; i < len(s); i++ {
		h = 31*h + int32(s[i])
	}
	return h
}

func parseRespawnSec(raw string) int32 {
	s := strings.TrimSpace(strings.ToLower(raw))
	switch {
	case strings.HasSuffix(s, "min"):
		n, _ := strconv.Atoi(strings.TrimSuffix(s, "min"))
		return int32(n * 60)
	case strings.HasSuffix(s, "sec"):
		n, _ := strconv.Atoi(strings.TrimSuffix(s, "sec"))
		return int32(n)
	case strings.HasSuffix(s, "hour"):
		n, _ := strconv.Atoi(strings.TrimSuffix(s, "hour"))
		return int32(n * 3600)
	default:
		n, _ := strconv.Atoi(s)
		if n <= 0 {
			return 60
		}
		return int32(n)
	}
}
