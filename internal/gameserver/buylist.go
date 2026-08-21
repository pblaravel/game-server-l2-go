package gameserver

import (
	"encoding/xml"
	"fmt"
	"os"
	"sync"
)

// NpcBuyList is Java model/buylist/NpcBuyList.
type NpcBuyList struct {
	ID    int32
	NpcID int32
	Items []BuyProduct
}

// BuyProduct is Java model/buylist/Product.
type BuyProduct struct {
	ItemID int32
	Price  int32
}

var (
	buyMu      sync.RWMutex
	buyByID    = map[int32]NpcBuyList{}
	buyByNpc   = map[int32][]int32{}
	buysLoaded bool
)

func GetBuyList(id int32) *NpcBuyList {
	buyMu.RLock()
	defer buyMu.RUnlock()
	l, ok := buyByID[id]
	if !ok {
		return nil
	}
	cp := l
	return &cp
}

func BuyListsForNPC(npcID int32) []NpcBuyList {
	buyMu.RLock()
	defer buyMu.RUnlock()
	ids := buyByNpc[npcID]
	out := make([]NpcBuyList, 0, len(ids))
	for _, id := range ids {
		if l, ok := buyByID[id]; ok {
			out = append(out, l)
		}
	}
	return out
}

func BuyListCount() int {
	buyMu.RLock()
	defer buyMu.RUnlock()
	return len(buyByID)
}

func (l NpcBuyList) Product(itemID int32) *BuyProduct {
	for i := range l.Items {
		if l.Items[i].ItemID == itemID {
			cp := l.Items[i]
			return &cp
		}
	}
	return nil
}

type xmlBuyRoot struct {
	Lists []xmlBuyList `xml:"buyList"`
}

type xmlBuyList struct {
	ID       int32        `xml:"id,attr"`
	NpcID    int32        `xml:"npcId,attr"`
	Products []xmlProduct `xml:"product"`
}

type xmlProduct struct {
	ID    int32 `xml:"id,attr"`
	Price int32 `xml:"price,attr"`
}

func loadBuyListXML(path string) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var root xmlBuyRoot
	if err := xml.Unmarshal(body, &root); err != nil {
		return fmt.Errorf("buyLists: %w", err)
	}
	next := map[int32]NpcBuyList{}
	byNpc := map[int32][]int32{}
	for _, l := range root.Lists {
		bl := NpcBuyList{ID: l.ID, NpcID: l.NpcID}
		for _, p := range l.Products {
			price := p.Price
			if price == 0 {
				if tpl := GetItem(p.ID); tpl != nil {
					price = tpl.Price
				}
			}
			bl.Items = append(bl.Items, BuyProduct{ItemID: p.ID, Price: price})
		}
		next[l.ID] = bl
		byNpc[l.NpcID] = append(byNpc[l.NpcID], l.ID)
	}
	buyMu.Lock()
	buyByID = next
	buyByNpc = byNpc
	buysLoaded = len(next) > 0
	buyMu.Unlock()
	return nil
}
