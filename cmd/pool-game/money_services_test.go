package main

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
	pooltreasure "github.com/wicanr2/Pool-of-Radiance-cht/internal/treasure"
)

// 公款的 P／T 在商店與神殿（spec 067〈公款〉，issue #79）。全部從 Update() 送鍵。

func typeChars(t *testing.T, a *app, text string) {
	t.Helper()
	a.keys = scriptedChars(text)
	if err := a.Update(); err != nil {
		t.Fatalf("type %q: %v", text, err)
	}
}

// 商店的 P）ool 把錢收進公款、T）ake 挑幣別與數量拿回目前的買家（overlay-06 `0636h`、`062Ah`）。
func TestShopPoolThenTakeBackThroughUpdate(t *testing.T) {
	a := newShopApp(t)
	pressKeys(t, a, ebiten.KeyT)
	if a.shop.take != nil {
		t.Fatal("T opened the take page with an empty pool; the menu without money has no Take")
	}
	pressKeys(t, a, ebiten.KeyP)
	if a.state.PooledMoney[pooltreasure.Gold] != 100 || a.state.Party[0].Money[pooltreasure.Gold] != 0 {
		t.Fatalf("after P: pool %v wallet %v", a.state.PooledMoney, a.state.Party[0].Money)
	}
	pressKeys(t, a, ebiten.KeyT)
	if a.shop.take == nil {
		t.Fatal("T with money in the pool did not open the take page")
	}
	pressKeys(t, a, ebiten.KeyEnter)
	if !a.shop.take.amountStage || a.shop.take.currency != pooltreasure.Gold {
		t.Fatalf("Enter did not pick gold: %+v", a.shop.take)
	}
	typeChars(t, a, "30")
	pressKeys(t, a, ebiten.KeyEnter)
	if a.state.PooledMoney[pooltreasure.Gold] != 70 || a.state.Party[0].Money[pooltreasure.Gold] != 30 {
		t.Fatalf("after taking 30: pool %v wallet %v", a.state.PooledMoney, a.state.Party[0].Money)
	}
	if a.shop.take == nil || a.shop.take.amountStage {
		t.Fatal("money is still in the pool; the original goes back to the coin list (`0F1Dh`)")
	}
	pressKeys(t, a, ebiten.KeyEnter)
	typeChars(t, a, "70")
	pressKeys(t, a, ebiten.KeyEnter)
	if a.shop.take != nil || a.hasPooledMoney() || a.state.Party[0].Money[pooltreasure.Gold] != 100 {
		t.Fatalf("emptying the pool: take %+v pool %v wallet %v", a.shop.take, a.state.PooledMoney, a.state.Party[0].Money)
	}
	pressKeys(t, a, ebiten.KeyEscape)
	if a.shopActive {
		t.Fatal("ESC with an empty pool did not leave")
	}
}

// 公款付款：P 之後錢包是空的，買東西由公款出（spec 116〈付款〉）。
func TestShopBuysFromThePoolAfterPooling(t *testing.T) {
	a := newShopApp(t)
	pressKeys(t, a, ebiten.KeyP, ebiten.KeyB, ebiten.KeyEnter)
	if len(a.state.Party[0].Inventory) != 1 {
		t.Fatalf("the long sword was not bought from the pool (%q)", a.shop.message)
	}
	// 100 − 15 = 85 金，重鑄成白金 17（overlay-21 entry 16）。
	if a.state.PooledMoney != [pooltreasure.CurrencyCount]uint32{pooltreasure.Platinum: 17} {
		t.Fatalf("pool after the purchase: %v", a.state.PooledMoney)
	}
}

// 神殿：進門清公款、P 收錢、Heal 由公款付、T 拿回、Exit 有錢先問（overlay-04 entry 1）。
func TestTemplePoolPaysTheCureAndAsksBeforeLeaving(t *testing.T) {
	hero := poolsave.Character{Name: "HERO", RaceID: "human", GenderID: "male", ClassID: "fighter",
		AlignmentID: "lawful-good", MaxHP: 12, CurrentHP: 2, Abilities: [6]int{18, 10, 10, 10, 10, 10}}
	ally := hero
	ally.Name, ally.CurrentHP = "ALLY", 12
	ally.Money[pooltreasure.Platinum] = 30 // 150 金
	a := &app{mode: modeAdventure, introDone: true, keys: scriptedKeys{}, roller: fixedTempleRoller(6),
		state: poolsave.State{Schema: poolsave.Schema,
			CharacterLibrary: []poolsave.Character{hero, ally}, Party: []poolsave.Character{hero, ally}}}
	a.state.PooledMoney[pooltreasure.Gold] = 999
	a.saveState = func(poolsave.State) error { return nil }
	if err := a.enterSuneTemple(); err != nil {
		t.Fatal(err)
	}
	if a.hasPooledMoney() {
		t.Fatalf("entering the temple kept the pool %v (`0D01h` FillChar)", a.state.PooledMoney)
	}
	if strings.Join(a.cellMenuOptions, " ") != "Heal View Pool Appraise Exit" {
		t.Fatalf("menu without money: %v", a.cellMenuOptions)
	}
	pressKeys(t, a, ebiten.KeyArrowRight, ebiten.KeyArrowRight, ebiten.KeyEnter)
	if a.state.PooledMoney[pooltreasure.Platinum] != 30 || a.state.Party[1].Money != ([7]uint16{}) {
		t.Fatalf("after Pool: pool %v ally %v", a.state.PooledMoney, a.state.Party[1].Money)
	}
	if strings.Join(a.cellMenuOptions, " ") != "Heal View Take Pool Share Appraise Exit" {
		t.Fatalf("menu with money: %v", a.cellMenuOptions)
	}
	// Heal → Cure Light Wounds（第 2 項）→ YES：HERO 身上沒錢，公款出 100 金。
	pressKeys(t, a, ebiten.KeyEnter, ebiten.KeyArrowRight, ebiten.KeyArrowRight, ebiten.KeyEnter, ebiten.KeyEnter)
	if a.state.Party[0].CurrentHP != 8 || a.state.PooledMoney != [pooltreasure.CurrencyCount]uint32{pooltreasure.Platinum: 10} {
		t.Fatalf("cure paid by the pool: HP %d pool %v (%q)", a.state.Party[0].CurrentHP, a.state.PooledMoney, a.eventText)
	}
	pressKeys(t, a, ebiten.KeyArrowLeft, ebiten.KeyEnter) // Heal 的 Exit
	pressKeys(t, a, ebiten.KeyArrowRight, ebiten.KeyArrowRight, ebiten.KeyEnter)
	if a.templeTake == nil || !strings.Contains(a.eventText, "Platinum 10") {
		t.Fatalf("Take did not list the pool: take %v text %q", a.templeTake, a.eventText)
	}
	pressKeys(t, a, ebiten.KeyEnter)
	typeChars(t, a, "4")
	pressKeys(t, a, ebiten.KeyEnter)
	if a.state.Party[0].Money[pooltreasure.Platinum] != 4 || a.state.PooledMoney[pooltreasure.Platinum] != 6 {
		t.Fatalf("take went to %v, pool %v; `0EC2h` gives it to the current character", a.state.Party[0].Money, a.state.PooledMoney)
	}
	pressKeys(t, a, ebiten.KeyArrowDown, ebiten.KeyEnter) // Exit
	if a.templeTake != nil || a.templeStage != templeMain {
		t.Fatal("Exit did not leave the take page")
	}
	pressKeys(t, a, ebiten.KeyArrowLeft, ebiten.KeyEnter)
	if a.templeStage != templeLeaving || !a.templeActive {
		t.Fatalf("Exit with money in the pool did not ask: stage %d", a.templeStage)
	}
	pressKeys(t, a, ebiten.KeyEnter) // Yes：回去拿
	if a.templeStage != templeMain || !a.templeActive {
		t.Fatal("Yes did not return to the temple menu")
	}
	pressKeys(t, a, ebiten.KeyArrowLeft, ebiten.KeyArrowLeft, ebiten.KeyArrowLeft, ebiten.KeyEnter) // Share
	if a.hasPooledMoney() || a.state.Party[0].Money[pooltreasure.Platinum] != 7 || a.state.Party[1].Money[pooltreasure.Platinum] != 3 {
		t.Fatalf("Share: pool %v hero %v ally %v", a.state.PooledMoney, a.state.Party[0].Money, a.state.Party[1].Money)
	}
}
