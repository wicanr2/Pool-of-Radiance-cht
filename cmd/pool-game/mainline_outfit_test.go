package main

// 玩家策略層第二條（issue #22）：開場先去武具店把身上的金幣換成裝備，再裝上。
// 新建的角色什麼都沒有（AC 10、空手），貧民窟第一場就死人；原版玩家的第一件事
// 就是買甲買武器。這裡照職業挑：戰士買買得起的最好的甲＋盾＋長劍，牧師買甲＋
// 盾＋釘頭錘（不用刃器），賊皮甲＋短劍，法師只買匕首。價格與存貨都是原版
// `ECL3.DAX` 武具店那一塊（`armouryStock`，spec 067）。所有動作都是正常按鍵：
// 走到武具店那一格、商店裡 TAB 換買家、↓ 移游標、ENTER 買、ESC 離開；
// `I` 開裝備、TAB 換人、↓ 選物、ENTER 裝上、ESC 關掉。

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	pooltreasure "github.com/wicanr2/Pool-of-Radiance-cht/internal/treasure"
)

// armouryTerrain 是城區地形碼表裡武具店那一個索引（spec 102）。
const armouryTerrain = 22

// outfitShoppingList 依職業與手上的金幣決定要買哪幾件（照存貨名稱）。
func outfitShoppingList(classID string, gold int) []string {
	class := strings.ToLower(classID)
	switch {
	case strings.Contains(class, "magic") && !strings.Contains(class, "fighter"):
		return []string{"Dagger"}
	case strings.Contains(class, "thief") && !strings.Contains(class, "fighter"):
		return []string{"Leather Armor", "Short Sword"}
	}
	weapon, weaponPrice := "Long Sword", 15
	if strings.Contains(class, "cleric") {
		weapon, weaponPrice = "Mace", 8
	}
	list := []string{}
	budget := gold - weaponPrice
	for _, armour := range []struct {
		name  string
		price int
	}{{"Banded Mail", 90}, {"Splint Mail", 80}, {"Chain Mail", 75}, {"Scale Mail", 45},
		{"Ring Mail", 30}, {"Studded Leather Armor", 15}, {"Leather Armor", 5}} {
		if budget >= armour.price {
			list = append(list, armour.name)
			budget -= armour.price
			break
		}
	}
	if budget >= 15 {
		list = append(list, "Shield")
		budget -= 15
	}
	if budget >= 0 {
		list = append(list, weapon)
	} else if gold >= 1 {
		list = append(list, "Spear")
	}
	return list
}

// outfitParty 從城區的任一格走到武具店、買、裝、再回到 (0,4) 朝西。
func (d *mainlineDriver) outfitParty() {
	d.t.Helper()
	a := d.a
	if a.eclArchive != 3 || a.eventSession.CurrentBlockID() != 0 {
		d.fatalf("outfitParty: not in the city")
	}
	shop := func(x, y int) bool { return d.terrain(x, y) == armouryTerrain }
	d.wantShop = true
	defer func() { d.wantShop = false }()
	if !d.walkTo("armoury", shop, func(x, y int) bool { return d.terrain(x, y) != 0 && !shop(x, y) }) {
		d.fatalf("outfitParty: cannot reach the armoury")
	}
	// 「CAN I SHOW YOU OUR WARES?」答第一項（YES）就開店。
	for guard := 0; guard < 64 && !a.shopActive; guard++ {
		d.step(ebiten.KeyEnter)
	}
	if !a.shopActive || a.shop == nil {
		d.fatalf("outfitParty: the armoury did not open its shop")
	}
	index := map[string]int{}
	for position, item := range a.shop.items {
		index[item.Name] = position
	}
	for member := range a.state.Party {
		for guard := 0; guard < 8 && a.shop.buyer != member; guard++ {
			d.step(ebiten.KeyTab)
		}
		gold := int(a.state.Party[member].Money[pooltreasure.Gold])
		list := outfitShoppingList(a.state.Party[member].ClassID, gold)
		for _, name := range list {
			target, ok := index[name]
			if !ok {
				d.fatalf("outfitParty: the armoury has no %q", name)
			}
			for guard := 0; guard < 64 && a.shop.cursor != target; guard++ {
				d.step(ebiten.KeyDown)
			}
			before := len(a.state.Party[member].Inventory)
			d.step(ebiten.KeyEnter)
			if len(a.state.Party[member].Inventory) != before+1 {
				d.note("outfit: %s could not buy %s (%s)", strings.TrimSpace(a.state.Party[member].Name), name, a.shop.message)
			}
		}
		d.note("outfit: %s (%s) %d gp → %v, %d gp left", strings.TrimSpace(a.state.Party[member].Name),
			a.state.Party[member].ClassID, gold, list, a.state.Party[member].Money[pooltreasure.Gold])
	}
	d.wantShop = false
	d.step(ebiten.KeyEscape)
	d.settle()
	// 買到的東西都是沒裝上的（`TestBoughtItemsArriveUnreadied`）；一件一件裝。
	d.step(ebiten.KeyI)
	if !a.equipmentOpen {
		d.fatalf("outfitParty: I did not open the equipment screen")
	}
	for member := range a.state.Party {
		for guard := 0; guard < 8 && a.equipment.member != member; guard++ {
			d.step(ebiten.KeyTab)
		}
		for item := range a.state.Party[member].Inventory {
			for guard := 0; guard < 16 && a.equipment.item != item; guard++ {
				d.step(ebiten.KeyDown)
			}
			if a.state.Party[member].Inventory[item].Raw[itemReadyOffset] == 0 {
				d.step(ebiten.KeyEnter)
			}
		}
	}
	d.step(ebiten.KeyEscape)
	if a.equipmentOpen {
		d.fatalf("outfitParty: the equipment screen did not close")
	}
	gate := func(x, y int) bool { return x == 0 && y == 4 }
	if !d.walkTo("city gate (0,4)", gate, func(x, y int) bool { return d.terrain(x, y) != 0 && !gate(x, y) }) {
		d.fatalf("outfitParty: cannot walk back to (0,4)")
	}
	d.settle()
	d.face(3)
}
