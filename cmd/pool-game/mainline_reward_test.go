package main

// 委任獎金怎麼花（#37，spec 137「建議順序」）：原版訓練所一位收 1000 金（spec 097），
// 諾里斯的獎金 250 金＋200 白金分六份沒有人付得起，原版玩家的做法是在寶物畫面 Pool
// 集中、Take 給要升級的人；剩下的留在隊伍的 pool，武具店會從 pool 付（#30，spec 116）。
// 這裡把這兩手接進主線駕駛，全部走正常按鍵：寶物選單、輸入列打金額、商店、裝備畫面。

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	pooltreasure "github.com/wicanr2/Pool-of-Radiance-cht/internal/treasure"
)

// takeRewardTo 在寶物畫面把全隊的錢 Pool 起來，再 Take 給 trainee 剛好夠學費的白金
// （1000 金 = 200 白金，spec 097），其餘留在隊伍的 pool。回 true 表示這一影格處理了。
func (d *mainlineDriver) takeRewardTo(trainee int, platinum uint32) bool {
	a := d.a
	if !a.treasureActive || !a.cellWaitingMenu {
		return false
	}
	pick := func(label string) {
		if err := selectMenuOption(d.t, a, label); err != nil {
			d.fatalf("takeRewardTo: %v", err)
		}
	}
	switch a.treasureStage {
	case treasureMain:
		pooledPlatinum := a.state.PooledMoney[pooltreasure.Platinum]
		// 先把大家的錢 Pool 起來（受訓的人 Take 過之後身上有錢，不算）。
		needPool := false
		for index, member := range a.state.Party {
			if index != trainee && member.Money != ([7]uint16{}) {
				needPool = true
			}
		}
		switch {
		case needPool:
			pick("Pool")
		case platinum != 0 && a.state.Party[trainee].Money[pooltreasure.Platinum] < uint16(platinum) && pooledPlatinum != 0:
			pick("Take")
		default:
			d.note("reward: pool=%v; %s holds %d pp", a.state.PooledMoney, strings.TrimSpace(a.state.Party[trainee].Name),
				a.state.Party[trainee].Money[pooltreasure.Platinum])
			pick("Exit")
		}
	case treasureTake:
		pick("Money")
	case treasureMoneyCurrency:
		want := fmt.Sprintf("%s %d", pooltreasure.Names[pooltreasure.Platinum], a.state.PooledMoney[pooltreasure.Platinum])
		if a.state.Party[trainee].Money[pooltreasure.Platinum] >= uint16(platinum) {
			want = "Exit"
		}
		pick(want)
	case treasureMoneyCharacter:
		for guard := 0; a.cellMenuCursor != trainee && guard < 8; guard++ {
			d.step(ebiten.KeyArrowDown)
		}
		d.step(ebiten.KeyEnter)
	case treasureMoneyAmount:
		amount := platinum - uint32(a.state.Party[trainee].Money[pooltreasure.Platinum])
		if amount > a.state.PooledMoney[pooltreasure.Platinum] {
			amount = a.state.PooledMoney[pooltreasure.Platinum]
		}
		d.chars(strconv.FormatUint(uint64(amount), 10))
		d.step(ebiten.KeyEnter)
	case treasureConfirmExit:
		pick("Yes")
	default:
		pick("Exit")
	}
	return true
}

// trainee 挑要升級的人：經驗值已過門檻的第一個；沒有就回 -1。
func (d *mainlineDriver) trainee() int {
	a := d.a
	for index, member := range a.state.Party {
		levels := memberClassLevels(member)
		if eligible, _ := a.levelUpTables.EligibleTrainingMask(levels, member.Experience, a.experienceTable); eligible != 0 {
			return index
		}
	}
	return -1
}

// buyArmourFromPool 走進武具店，讓 member 買 name（自己的錢不夠就由隊伍的 pool 付，#30），
// 買到就裝上。回買前買後的七個錢欄。
func (d *mainlineDriver) buyArmourFromPool(member int, name string) string {
	d.t.Helper()
	a := d.a
	if a.eclArchive != 3 || a.eventSession.CurrentBlockID() != 0 {
		d.fatalf("buyArmourFromPool: not in the city")
	}
	shop := func(x, y int) bool { return d.terrain(x, y) == armouryTerrain }
	d.wantShop = true
	defer func() { d.wantShop = false }()
	if !d.walkTo("armoury", shop, func(x, y int) bool { return d.terrain(x, y) != 0 && !shop(x, y) }) {
		d.fatalf("buyArmourFromPool: cannot reach the armoury")
	}
	for guard := 0; guard < 64 && !a.shopActive; guard++ {
		d.step(ebiten.KeyEnter)
	}
	if !a.shopActive || a.shop == nil {
		d.fatalf("buyArmourFromPool: the armoury did not open its shop")
	}
	for guard := 0; guard < 8 && a.shop.buyer != member; guard++ {
		d.step(ebiten.KeyTab)
	}
	target := -1
	for position, item := range a.shop.items {
		if item.Name == name {
			target = position
		}
	}
	if target < 0 {
		d.fatalf("buyArmourFromPool: the armoury has no %q", name)
	}
	for guard := 0; guard < 64 && a.shop.cursor != target; guard++ {
		d.step(ebiten.KeyDown)
	}
	before := fmt.Sprintf("%s money=%v pool=%v", strings.TrimSpace(a.state.Party[member].Name),
		a.state.Party[member].Money, a.state.PooledMoney)
	count := len(a.state.Party[member].Inventory)
	d.step(ebiten.KeyEnter)
	line := fmt.Sprintf("bought %s: before %s → after money=%v pool=%v (%s)", name, before,
		a.state.Party[member].Money, a.state.PooledMoney, a.shop.message)
	if len(a.state.Party[member].Inventory) != count+1 {
		line = "could not buy " + name + ": " + a.shop.message + " (" + before + ")"
	}
	d.note("%s", line)
	d.wantShop = false
	d.step(ebiten.KeyEscape)
	d.settle()
	if len(a.state.Party[member].Inventory) == count+1 {
		d.step(ebiten.KeyI)
		if !a.equipmentOpen {
			d.fatalf("buyArmourFromPool: I did not open the equipment screen")
		}
		for guard := 0; guard < 8 && a.equipment.member != member; guard++ {
			d.step(ebiten.KeyTab)
		}
		// 只裝剛買的那一件（在最後）。原版那一格有東西就印 "already using"、不換手
		// （overlay-19 entry 7，spec 149），所以跟玩家一樣：先把同一類的舊甲卸下再裝。
		item := len(a.state.Party[member].Inventory) - 1
		bought, _ := a.itemCategory(a.state.Party[member].Inventory[item])
		for old := 0; old < item; old++ {
			worn := a.state.Party[member].Inventory[old]
			if category, ok := a.itemCategory(worn); !ok || category != bought || worn.Raw[itemReadyOffset] == 0 {
				continue
			}
			for guard := 0; guard < 16 && a.equipment.item != old; guard++ {
				d.step(ebiten.KeyDown)
			}
			d.step(ebiten.KeyEnter)
		}
		for guard := 0; guard < 16 && a.equipment.item != item; guard++ {
			d.step(ebiten.KeyDown)
		}
		d.step(ebiten.KeyEnter)
		d.step(ebiten.KeyEscape)
		armour, _, err := a.memberDefenceStats(a.state.Party[member], creationArmorClassInternal, creationBaseMovement)
		d.note("%s now wears %s: armour internal %d (surface AC %d, err=%v)", strings.TrimSpace(a.state.Party[member].Name),
			name, armour, 60-armour, err)
	}
	return line
}
