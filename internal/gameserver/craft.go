package gameserver

import (
	"encoding/xml"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

type RecipeMaterial struct {
	ItemID int32
	Count  int32
}

type Recipe struct {
	ID          int32
	Alias       string
	ItemID      int32
	Level       int32
	MPConsume   int32
	SuccessRate int32
	IsDwarven   bool
	Materials   []RecipeMaterial
	ProductID   int32
	ProductCnt  int32
}

type MultisellIngredient struct {
	ItemID  int32
	Count   int32
	Enchant int32
}

type MultisellEntry struct {
	Ingredients []MultisellIngredient
	Products    []MultisellIngredient
}

type MultisellList struct {
	ID                  int32
	Name                string
	ApplyTaxes          bool
	MaintainEnchantment bool
	NpcIDs              []int32
	Entries             []MultisellEntry
}

var (
	craftMu    sync.RWMutex
	recipes    = map[int32]Recipe{}
	recipeItem = map[int32]int32{}
	multisells = map[int32]*MultisellList{}
)

func RecipeCount() int {
	craftMu.RLock()
	defer craftMu.RUnlock()
	return len(recipes)
}

func MultisellCount() int {
	craftMu.RLock()
	defer craftMu.RUnlock()
	seen := map[string]struct{}{}
	for _, list := range multisells {
		seen[list.Name] = struct{}{}
	}
	return len(seen)
}

func GetRecipe(id int32) *Recipe {
	craftMu.RLock()
	defer craftMu.RUnlock()
	r, ok := recipes[id]
	if !ok {
		return nil
	}
	cp := r
	return &cp
}

func GetRecipeByItem(itemID int32) *Recipe {
	craftMu.RLock()
	id, ok := recipeItem[itemID]
	craftMu.RUnlock()
	if !ok {
		return nil
	}
	return GetRecipe(id)
}

func GetMultisell(id int32) *MultisellList {
	craftMu.RLock()
	defer craftMu.RUnlock()
	return multisells[id]
}

func MultisellsForNPC(npcID int32) []*MultisellList {
	craftMu.RLock()
	defer craftMu.RUnlock()
	out := make([]*MultisellList, 0)
	for _, list := range multisells {
		for _, id := range list.NpcIDs {
			if id == npcID {
				out = append(out, list)
				break
			}
		}
	}
	return out
}

type xmlRecipeList struct {
	Recipes []xmlRecipe `xml:"recipe"`
}

type xmlRecipe struct {
	Alias       string `xml:"alias,attr"`
	ID          int32  `xml:"id,attr"`
	Material    string `xml:"material,attr"`
	Product     string `xml:"product,attr"`
	ItemID      int32  `xml:"itemId,attr"`
	Level       int32  `xml:"level,attr"`
	MPConsume   int32  `xml:"mpConsume,attr"`
	SuccessRate int32  `xml:"successRate,attr"`
	IsDwarven   string `xml:"isDwarven,attr"`
}

func loadRecipeXML(path string) error {
	var root xmlRecipeList
	if err := readXML(path, &root); err != nil {
		return err
	}
	next := map[int32]Recipe{}
	byItem := map[int32]int32{}
	for _, r := range root.Recipes {
		rec := Recipe{
			ID: r.ID, Alias: r.Alias, ItemID: r.ItemID, Level: r.Level,
			MPConsume: r.MPConsume, SuccessRate: r.SuccessRate,
			IsDwarven: parseBoolDefault(r.IsDwarven, true),
		}
		for _, pair := range parseIDCountPairs(r.Material) {
			rec.Materials = append(rec.Materials, RecipeMaterial{ItemID: pair[0], Count: pair[1]})
		}
		if pairs := parseIDCountPairs(r.Product); len(pairs) > 0 {
			rec.ProductID, rec.ProductCnt = pairs[0][0], pairs[0][1]
		}
		next[rec.ID] = rec
		if rec.ItemID > 0 {
			byItem[rec.ItemID] = rec.ID
		}
	}
	craftMu.Lock()
	recipes, recipeItem = next, byItem
	craftMu.Unlock()
	return nil
}

type xmlMultisellRoot struct {
	ApplyTaxes          string             `xml:"applyTaxes,attr"`
	MaintainEnchantment string             `xml:"maintainEnchantment,attr"`
	Npcs                []xmlMultisellNPC  `xml:"npcs>npc"`
	Items               []xmlMultisellItem `xml:"item"`
}

type xmlMultisellNPC struct {
	Text string `xml:",chardata"`
}

type xmlMultisellItem struct {
	Ingredients []xmlMSIng `xml:"ingredient"`
	Products    []xmlMSIng `xml:"production"`
}

type xmlMSIng struct {
	ID      int32 `xml:"id,attr"`
	Count   int32 `xml:"count,attr"`
	Enchant int32 `xml:"enchant,attr"`
}

func loadMultisellXML(dir string) error {
	next := map[int32]*MultisellList{}
	err := walkXMLFiles(dir, func(name string, body []byte) error {
		var root xmlMultisellRoot
		if err := xml.Unmarshal(body, &root); err != nil {
			return err
		}
		stem := strings.TrimSuffix(name, filepath.Ext(name))
		hash := javaStringHash(stem)
		list := &MultisellList{
			ID:                  hash,
			Name:                stem,
			ApplyTaxes:          parseBool(root.ApplyTaxes),
			MaintainEnchantment: parseBool(root.MaintainEnchantment),
		}
		if n, err := strconv.Atoi(stem); err == nil {
			list.ID = int32(n)
		}
		for _, n := range root.Npcs {
			id, err := strconv.Atoi(strings.TrimSpace(n.Text))
			if err == nil {
				list.NpcIDs = append(list.NpcIDs, int32(id))
			}
		}
		for _, it := range root.Items {
			entry := MultisellEntry{}
			for _, ing := range it.Ingredients {
				entry.Ingredients = append(entry.Ingredients, MultisellIngredient{ItemID: ing.ID, Count: ing.Count, Enchant: ing.Enchant})
			}
			for _, prod := range it.Products {
				entry.Products = append(entry.Products, MultisellIngredient{ItemID: prod.ID, Count: prod.Count, Enchant: prod.Enchant})
			}
			list.Entries = append(list.Entries, entry)
		}
		next[list.ID] = list
		if hash != list.ID {
			next[hash] = list
		}
		return nil
	})
	if err != nil {
		return err
	}
	craftMu.Lock()
	multisells = next
	craftMu.Unlock()
	return nil
}
