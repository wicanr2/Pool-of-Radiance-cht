package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 戰鬥中物品選單的 Ready、Drop、Halve、Join、卷軸與參數表 +0Bh 為 0 的物品法術
// （overlay-19 entry 6／7／8／12／14／15、overlay-22 entry 7，spec 144，issue #84）。
// 全部從 Update() 送鍵。

const (
	testTypeSword     = 0x01 // 1d8，單手，戰士／聖騎士
	testTypeTwoHander = 0x26 // 1d10，雙手
	testTypeMUScroll  = 0x3d // 法師卷軸（類別 0Bh，雙手）
)

// newItemMenuApp 是一個玩家操作的隊員（職業 slot 等級 level）對一隻敵人，身上帶著 items。
func newItemMenuApp(t *testing.T, class int, level uint8, items ...poolsave.Item) (*app, *tacticalState) {
	t.Helper()
	application, state := newFoeCastApp(t)
	types, err := gamepack.ReadDOSItemTypeTable(filepath.Join("..", "..", "Pool of Radiance (1988).zip"))
	if err != nil {
		t.Fatal(err)
	}
	application.itemTypes = types
	levels := make([]uint8, gamepack.ClassThac0ClassCount)
	levels[class] = level
	application.state.Party[0] = poolsave.Character{Name: "A", ClassID: "", ClassLevels: levels,
		Abilities: [6]int{12, 12, 12, 12, 12, 12}, Inventory: items}
	state.AIDriven[1] = false
	state.Mover = 1
	state.Scores[1], state.Scores[2] = 5, 1
	application.combatCommands = turnUseSegments
	return application, state
}

func itemOf(name string, itemType uint8, readied bool, count uint8) poolsave.Item {
	raw := make([]byte, 63)
	raw[gamepack.ItemTypeOffset] = itemType
	if readied {
		raw[gamepack.ItemReadiedOffset] = 1
	}
	raw[gamepack.ItemCountOffset] = count
	return poolsave.Item{Name: name, Raw: raw}
}

// Ready 換武器：原版不自動換手——拿著長劍再裝雙手劍印 "already using"；先卸下長劍
// 再裝，傷害當場換成 1d10（`146Fh` 重算），選單還開著，這個行動沒用掉。
func TestCombatReadySwapsWeaponWithoutSpendingTheAction(t *testing.T) {
	application, state := newItemMenuApp(t, int(gamepack.ClassSlotFighter), 5,
		itemOf("SWORD", testTypeSword, true, 0), itemOf("TWO-HANDED SWORD", testTypeTwoHander, false, 0))
	pressAll(t, application, ebiten.KeyU)
	if application.combatItems == nil {
		t.Fatal("U did not open the item menu")
	}
	if footer := application.combatItemFooter(state, 0); footer != "READY USE DROP HALVE JOIN EXIT" {
		t.Fatalf("item menu footer %q", footer)
	}
	pressAll(t, application, ebiten.KeyArrowDown, ebiten.KeyR)
	if !strings.Contains(state.Status, "ALREADY USING SWORD") {
		t.Fatalf("readying a second weapon printed %q", state.Status)
	}
	pressAll(t, application, ebiten.KeyArrowUp, ebiten.KeyR, ebiten.KeyArrowDown, ebiten.KeyR)
	inventory := application.state.Party[0].Inventory
	if inventory[0].Raw[gamepack.ItemReadiedOffset] != 0 || inventory[1].Raw[gamepack.ItemReadiedOffset] != 1 {
		t.Fatalf("readied flags %d %d", inventory[0].Raw[gamepack.ItemReadiedOffset], inventory[1].Raw[gamepack.ItemReadiedOffset])
	}
	if state.AttackForms[1][0].Sides != 10 || state.Damage[1].Sides != 10 {
		t.Fatalf("damage after the swap %+v, want 1d10", state.AttackForms[1][0])
	}
	if application.combatItems == nil || state.Mover != 1 || state.Scores[1] != 5 {
		t.Fatalf("ready spent the action: menu %v mover %d score %d", application.combatItems, state.Mover, state.Scores[1])
	}
	pressAll(t, application, ebiten.KeyEscape)
	if application.combatItems != nil || state.Mover != 1 {
		t.Fatal("ESC did not return to the command bar of the same mover")
	}
}

// Drop：穿戴中的先要卸下（entry 20）；沒穿的問 "Drop It?"，Y 就摘掉。
func TestCombatDropAsksAndRemoves(t *testing.T) {
	application, state := newItemMenuApp(t, int(gamepack.ClassSlotFighter), 5,
		itemOf("SWORD", testTypeSword, true, 0), itemOf("TWO-HANDED SWORD", testTypeTwoHander, false, 0))
	pressAll(t, application, ebiten.KeyU, ebiten.KeyD)
	if !strings.Contains(state.Status, "MUST BE UNREADIED") || len(application.state.Party[0].Inventory) != 2 {
		t.Fatalf("dropping a readied sword: %q", state.Status)
	}
	pressAll(t, application, ebiten.KeyArrowDown, ebiten.KeyD)
	if !strings.Contains(state.Status, "WILL BE GONE FOREVER") {
		t.Fatalf("drop prompt %q", state.Status)
	}
	pressAll(t, application, ebiten.KeyN)
	if len(application.state.Party[0].Inventory) != 2 {
		t.Fatal("N dropped the item")
	}
	pressAll(t, application, ebiten.KeyD, ebiten.KeyY)
	inventory := application.state.Party[0].Inventory
	if len(inventory) != 1 || inventory[0].Name != "SWORD" {
		t.Fatalf("after Y: %+v", inventory)
	}
	if application.combatItems == nil || state.Scores[1] != 5 {
		t.Fatal("drop closed the menu or spent the action")
	}
}

// Halve：十一支箭分成 6 與 5，新的一疊不穿戴、接在後面；一支的印 "Can't halve that"。
func TestCombatHalveSplitsAStack(t *testing.T) {
	application, state := newItemMenuApp(t, int(gamepack.ClassSlotFighter), 5,
		itemOf("ARROWS", gamepack.ItemTypeArrow, true, 11), itemOf("ARROW", gamepack.ItemTypeArrow, false, 1))
	pressAll(t, application, ebiten.KeyU, ebiten.KeyH)
	inventory := application.state.Party[0].Inventory
	if len(inventory) != 3 || inventory[0].Raw[gamepack.ItemCountOffset] != 6 ||
		inventory[1].Raw[gamepack.ItemCountOffset] != 5 || inventory[1].Raw[gamepack.ItemReadiedOffset] != 0 {
		t.Fatalf("after halving: %d items", len(inventory))
	}
	pressAll(t, application, ebiten.KeyArrowDown, ebiten.KeyArrowDown, ebiten.KeyH)
	if !strings.Contains(state.Status, "CAN'T HALVE THAT") || len(application.state.Party[0].Inventory) != 3 {
		t.Fatalf("halving one arrow: %q", state.Status)
	}
}

// Join：同一種的疊回選中那一件；超過 255 的留在後面那一疊。
func TestCombatJoinMergesStacks(t *testing.T) {
	application, _ := newItemMenuApp(t, int(gamepack.ClassSlotFighter), 5,
		itemOf("ARROWS", gamepack.ItemTypeArrow, false, 5), itemOf("SWORD", testTypeSword, true, 0),
		itemOf("ARROWS", gamepack.ItemTypeArrow, false, 5), itemOf("ARROWS", gamepack.ItemTypeArrow, false, 250))
	pressAll(t, application, ebiten.KeyU, ebiten.KeyJ)
	inventory := application.state.Party[0].Inventory
	if len(inventory) != 3 || inventory[0].Raw[gamepack.ItemCountOffset] != 255 ||
		inventory[2].Raw[gamepack.ItemCountOffset] != 5 {
		counts := []uint8{}
		for _, item := range inventory {
			counts = append(counts, item.Raw[gamepack.ItemCountOffset])
		}
		t.Fatalf("after join: counts %v", counts)
	}
}

func scrollItem(hidden uint8, spells ...uint8) poolsave.Item {
	item := itemOf("MAGIC USER SCROLL", testTypeMUScroll, true, 0)
	item.Raw[gamepack.ItemNameWordOffset+1] = 0xd1 + uint8(len(spells))
	copy(item.Raw[gamepack.AIItemChargesOffset:], spells)
	item.Raw[gamepack.ItemHiddenNameOffset] = hidden
	return item
}

// 卷軸：挑一行放出去，放完抹掉那一行（overlay-22 entry 7），名稱字詞 D3h → D2h；
// 卷軸不印 "uses an item"，行動用掉（參數表 +0Bh 非 0 → entry 34）。
func TestCombatScrollCastsAndErasesTheLine(t *testing.T) {
	application, state := newItemMenuApp(t, int(gamepack.ClassSlotMagicUser), 3,
		scrollItem(0, gamepack.SpellIDMagicMissile, gamepack.SpellIDSleep))
	hp := state.HitPoints[2]
	pressAll(t, application, ebiten.KeyU, ebiten.KeyU)
	menu := application.combatItems
	if menu == nil || menu.stage != combatItemScroll || len(menu.scroll) != 2 {
		t.Fatalf("the scroll list did not open: %+v", menu)
	}
	pressAll(t, application, ebiten.KeyEnter)
	if strings.Contains(state.Status, "USES AN ITEM") {
		t.Fatalf("a scroll printed %q", state.Status)
	}
	if application.castTargeting {
		pressAll(t, application, ebiten.KeyEnter)
	}
	if state.HitPoints[2] >= hp {
		t.Fatalf("the scroll's magic missile did no damage: %q", state.Status)
	}
	raw := application.state.Party[0].Inventory[0].Raw
	if raw[gamepack.AIItemChargesOffset] != 0 || raw[gamepack.AIItemSpellOffset] != gamepack.SpellIDSleep ||
		raw[gamepack.ItemNameWordOffset+1] != 0xd2 {
		t.Fatalf("scroll after casting: % X", raw[gamepack.ItemNameWordOffset:gamepack.ItemEffectOffset+1])
	}
	if state.Mover == 1 && state.Scores[1] != 0 {
		t.Fatal("reading the scroll did not spend the action")
	}
}

// 藏字沒揭開的法師卷軸，身上沒有閱讀魔法（效果 10h）：entry 12 一行也列不出來，
// 結果 0，留在物品選單，卷軸不動。
func TestCombatHiddenScrollNeedsReadMagic(t *testing.T) {
	application, state := newItemMenuApp(t, int(gamepack.ClassSlotMagicUser), 3,
		scrollItem(0x02, gamepack.SpellIDMagicMissile))
	pressAll(t, application, ebiten.KeyU, ebiten.KeyU)
	if application.combatItems == nil || application.combatItems.stage != combatItemPicking ||
		application.castTargeting || state.Scores[1] != 5 {
		t.Fatal("an unread scroll did something")
	}
	state.Effects[1] = gamepack.EffectList{gamepack.NewEffectNode(gamepack.ReadMagicEffectCode, 0, 1, false)}
	pressAll(t, application, ebiten.KeyU)
	if application.combatItems == nil || application.combatItems.stage != combatItemScroll ||
		application.state.Party[0].Inventory[0].Raw[gamepack.ItemHiddenNameOffset] != 0 {
		t.Fatal("read magic did not reveal the scroll")
	}
}

// 參數表 +0Bh 為 0 的法術（閱讀魔法，18）用物品放出：overlay-22 entry 5 在戰鬥中照放，
// 次數記帳；entry 8 `1BF4h` 不呼叫 entry 34，所以分數不歸零、同一個人重選到。
func TestCombatItemCampOnlySpellKeepsTheTurn(t *testing.T) {
	wand := itemOf("WAND", testTypeSword, true, 0)
	wand.Raw[gamepack.AIItemChargesOffset] = 2
	wand.Raw[gamepack.AIItemSpellOffset] = 18
	application, state := newItemMenuApp(t, int(gamepack.ClassSlotFighter), 5, wand)
	if !application.spellParameters[18].CampOnly() {
		t.Fatal("spell 18 is not a +0Bh == 0 spell")
	}
	pressAll(t, application, ebiten.KeyU, ebiten.KeyU)
	// 模式 0（打自己）不瞄準，"uses an item" 當場被施法那一句蓋過。
	if !strings.Contains(state.Status, "casts") {
		t.Fatalf("using the wand printed %q", state.Status)
	}
	if application.castTargeting {
		pressAll(t, application, ebiten.KeyEnter)
	}
	if charges := application.state.Party[0].Inventory[0].Raw[gamepack.AIItemChargesOffset]; charges != 1 {
		t.Fatalf("charges %d, want 1", charges)
	}
	if application.combatItems != nil || state.Mover != 1 || state.Scores[1] != 5 {
		t.Fatalf("the turn ended: menu %v mover %d score %d", application.combatItems, state.Mover, state.Scores[1])
	}
}
