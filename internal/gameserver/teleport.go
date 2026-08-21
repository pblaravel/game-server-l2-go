package gameserver

import (
	"encoding/xml"
	"fmt"
	"os"
	"sync"
)

// TeleportLocation is Java model/location/TeleportLocation.
type TeleportLocation struct {
	Desc       string
	Type       string
	PriceID    int32
	PriceCount int32
	X, Y, Z    int32
}

var (
	teleMu    sync.RWMutex
	teleByNpc = map[int32][]TeleportLocation{}
)

func TeleportsForNPC(npcID int32) []TeleportLocation {
	teleMu.RLock()
	defer teleMu.RUnlock()
	return append([]TeleportLocation(nil), teleByNpc[npcID]...)
}

func TeleportListCount() int {
	teleMu.RLock()
	defer teleMu.RUnlock()
	return len(teleByNpc)
}

type xmlTeleportRoot struct {
	Lists []xmlTelPosList `xml:"telPosList"`
}

type xmlTelPosList struct {
	NpcID int32    `xml:"npcId,attr"`
	Locs  []xmlLoc `xml:"loc"`
}

type xmlLoc struct {
	Desc       string `xml:"desc,attr"`
	Type       string `xml:"type,attr"`
	PriceID    int32  `xml:"priceId,attr"`
	PriceCount int32  `xml:"priceCount,attr"`
	X          int32  `xml:"x,attr"`
	Y          int32  `xml:"y,attr"`
	Z          int32  `xml:"z,attr"`
}

func loadTeleportXML(path string) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var root xmlTeleportRoot
	if err := xml.Unmarshal(body, &root); err != nil {
		return fmt.Errorf("teleports: %w", err)
	}
	next := map[int32][]TeleportLocation{}
	for _, list := range root.Lists {
		for _, loc := range list.Locs {
			next[list.NpcID] = append(next[list.NpcID], TeleportLocation{
				Desc: loc.Desc, Type: loc.Type,
				PriceID: loc.PriceID, PriceCount: loc.PriceCount,
				X: loc.X, Y: loc.Y, Z: loc.Z,
			})
		}
	}
	teleMu.Lock()
	teleByNpc = next
	teleMu.Unlock()
	return nil
}
