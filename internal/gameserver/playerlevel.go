package gameserver

import (
	"encoding/xml"
	"fmt"
	"os"
	"sync"
)

// defaultPlayerLevelExp is used when playerLevels.xml is missing.
var defaultPlayerLevelExp = []int64{
	0, // unused index 0
	0, // 1
	68, 363, 1168, 2884, 6038, 11287, 19423, 31378, 48229,
	71201, 101676, 141192, 191417, 254161, 331466, 425503, 538475, 672708, 830564,
	1014620, 1226951, 1471172, 1750753, 2069420, 2431166, 2840261, 3301266, 3819031, 4398706,
	5045741, 5765886, 6565191, 7450026, 8427071, 9503326, 10686081, 11983236, 13401491, 14953046,
	16651201, 18510456, 20546511, 22776366, 25218321, 27892076, 30818731, 34020886, 37522541, 41349196,
	45527951, 50087406, 55057761, 60470816, 66360071, 72760626, 79709281, 87244536, 95406691, 104237446,
	113798201, 124139956, 135337711, 147472466, 160631221, 174907976, 190403731, 207226486, 225491241, 245320996,
	266846751, 290207506, 315551261, 343035016, 372824771, 405096526, 440036281, 477840036, 518714791, 562878546,
	610560301, // 81 — first unreachable if XML is absent
}

// PlayerLevelExp is cumulative exp required to *be* at this level (Java requiredExpToLevelUp).
var PlayerLevelExp = append([]int64(nil), defaultPlayerLevelExp...)

// PlayerLevelRow is Java model.records.PlayerLevel.
type PlayerLevelRow struct {
	Level        int32
	Exp          int64
	Karma        float64
	ExpLossDeath float64
}

var (
	playerLevelMu   sync.RWMutex
	playerLevelByLv = map[int32]PlayerLevelRow{}
)

func ExpForLevel(level int) int64 {
	playerLevelMu.RLock()
	defer playerLevelMu.RUnlock()
	if level <= 0 || level >= len(PlayerLevelExp) {
		return PlayerLevelExp[len(PlayerLevelExp)-1]
	}
	return PlayerLevelExp[level]
}

func ExpPercent(level int, exp int64) float64 {
	cur := ExpForLevel(level)
	next := ExpForLevel(level + 1)
	if next <= cur {
		return 0
	}
	return float64(exp-cur) / float64(next-cur)
}

func GetPlayerLevel(level int32) (PlayerLevelRow, bool) {
	playerLevelMu.RLock()
	defer playerLevelMu.RUnlock()
	row, ok := playerLevelByLv[level]
	return row, ok
}

func PlayerLevelCount() int {
	playerLevelMu.RLock()
	defer playerLevelMu.RUnlock()
	return len(playerLevelByLv)
}

// MaxPlayerLevel is Java PlayerLevelData.getMaxLevel: first unreachable level.
func MaxPlayerLevel() int32 {
	playerLevelMu.RLock()
	defer playerLevelMu.RUnlock()
	if len(PlayerLevelExp) < 2 {
		return 1
	}
	return int32(len(PlayerLevelExp) - 1)
}

type xmlPlayerLevels struct {
	Levels []xmlPlayerLevel `xml:"playerLevel"`
}

type xmlPlayerLevel struct {
	Level   int32   `xml:"level,attr"`
	Exp     int64   `xml:"requiredExpToLevelUp,attr"`
	Karma   float64 `xml:"karmaModifier,attr"`
	ExpLoss float64 `xml:"expLossAtDeath,attr"`
}

func loadPlayerLevelXML(path string) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var root xmlPlayerLevels
	if err := xml.Unmarshal(body, &root); err != nil {
		return err
	}
	if len(root.Levels) == 0 {
		return fmt.Errorf("no playerLevel rows in %s", path)
	}
	next := make(map[int32]PlayerLevelRow, len(root.Levels))
	maxLv := 0
	for _, lv := range root.Levels {
		if int(lv.Level) > maxLv {
			maxLv = int(lv.Level)
		}
		next[lv.Level] = PlayerLevelRow{
			Level:        lv.Level,
			Exp:          lv.Exp,
			Karma:        lv.Karma,
			ExpLossDeath: lv.ExpLoss,
		}
	}
	table := make([]int64, maxLv+1)
	for i := 1; i <= maxLv; i++ {
		if row, ok := next[int32(i)]; ok {
			table[i] = row.Exp
		}
	}
	playerLevelMu.Lock()
	PlayerLevelExp = table
	playerLevelByLv = next
	playerLevelMu.Unlock()
	return nil
}
