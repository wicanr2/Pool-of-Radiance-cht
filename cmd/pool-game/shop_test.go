package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/golden-box-remake-engine/eclvm"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
	pooltreasure "github.com/wicanr2/Pool-of-Radiance-cht/internal/treasure"
)

// 真檔的防具店存貨：ITEM3.DAX 的 block 35h，由 ECL3/block0 的
// `TREASURE 0,0,0,0,0,0,0,53` 指定。
func armouryStock(t *testing.T) []gamepack.TreasureItemRecord {
	t.Helper()
	records, err := gamepack.ReadDOSTreasureItemBlock(dosZIPForTests, 3, 0x35)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	return records
}

// 原版的價目要跟得上：長劍 15、板甲 400、匕首 2、鏈甲 75。四個都與
// AD&D 玩家手冊相同，說明價格欄讀對了。
func TestArmouryStockCarriesTheOriginalPrices(t *testing.T) {
	prices := map[string]uint16{}
	for _, record := range armouryStock(t) {
		prices[record.Name] = record.Price()
	}
	for name, want := range map[string]uint16{
		"Long Sword": 15, "Plate Mail": 400, "Dagger": 2, "Chain Mail": 75,
		"Two-Handed Sword": 30, "Shield": 15,
	} {
		if got, ok := prices[name]; !ok || got != want {
			t.Fatalf("%s costs %d (present=%v), want %d", name, got, ok, want)
		}
	}
}

func newShopApp(t *testing.T) *app {
	t.Helper()
	stock := armouryStock(t)
	member := poolsave.Character{Name: "HERO", ClassID: "fighter",
		Abilities: [6]int{18, 10, 10, 12, 10, 10}}
	member.Money[pooltreasure.Gold] = 100
	a := &app{
		shop:      &shopState{items: stock},
		shopActive: true,
		state:     poolsave.State{Party: []poolsave.Character{member}},
	}
	a.state.CharacterLibrary = append(a.state.CharacterLibrary, member)
	for index, record := range stock {
		if record.Name == "Long Sword" {
			a.shop.cursor = index
		}
	}
	return a
}

// 買下去：扣金幣、物品進背包，隊伍與角色庫都要同步。
func TestBuyingDeductsGoldAndAddsTheItem(t *testing.T) {
	a := newShopApp(t)
	a.keys = scriptedKeys{ebiten.KeyEnter: true}
	if err := a.shopInput(); err != nil {
		t.Fatal(err)
	}
	member := a.state.Party[0]
	// 原版付完錢把餘額重鑄成白金＋金（overlay-21 entry 15，spec 116）：85 金 → 17 白金 0 金。
	if member.Money[pooltreasure.Platinum] != 17 || member.Money[pooltreasure.Gold] != 0 {
		t.Fatalf("money is %v after buying a 15 gold long sword from 100 gold, want 17 platinum", member.Money)
	}
	if len(member.Inventory) != 1 || member.Inventory[0].Name != "Long Sword" {
		t.Fatalf("inventory is %+v", member.Inventory)
	}
	if len(a.state.CharacterLibrary[0].Inventory) != 1 {
		t.Fatal("the character library did not receive the purchase")
	}
}

// 錢不夠就不賣，也不扣錢——扣了錢卻沒東西是玩家看得見的損失。
func TestBuyingRefusesWhenTheGoldIsShort(t *testing.T) {
	a := newShopApp(t)
	a.state.Party[0].Money[pooltreasure.Gold] = 10
	a.keys = scriptedKeys{ebiten.KeyEnter: true}
	if err := a.shopInput(); err != nil {
		t.Fatal(err)
	}
	if got := a.state.Party[0].Money[pooltreasure.Gold]; got != 10 {
		t.Fatalf("gold changed to %d on a refused purchase", got)
	}
	if len(a.state.Party[0].Inventory) != 0 {
		t.Fatal("a refused purchase still added the item")
	}
	if a.shop.message == "" {
		t.Fatal("nothing explained why the purchase failed")
	}
}

// 只帶白金也買得到（#30）：白金 5 金一枚，100 白金 = 500 金；買 15 金的長劍剩 485 →
// 重鑄成 97 白金 0 金。
func TestBuyingWithPlatinumOnly(t *testing.T) {
	a := newShopApp(t)
	a.state.Party[0].Money = [7]uint16{}
	a.state.Party[0].Money[pooltreasure.Platinum] = 100
	a.keys = scriptedKeys{ebiten.KeyEnter: true}
	if err := a.shopInput(); err != nil {
		t.Fatal(err)
	}
	member := a.state.Party[0]
	if member.Money[pooltreasure.Platinum] != 97 || member.Money[pooltreasure.Gold] != 0 {
		t.Fatalf("money is %v after paying 15 gold with 100 platinum, want 97 platinum", member.Money)
	}
	if len(member.Inventory) != 1 || member.Inventory[0].Name != "Long Sword" {
		t.Fatalf("inventory is %+v", member.Inventory)
	}
	if a.state.CharacterLibrary[0].Money != member.Money {
		t.Fatal("the character library copy kept the old money")
	}
}

// 角色不夠才動隊伍公款（overlay-06 `03E7h`：entry 17 算、entry 16 寫回），不混付。
func TestBuyingFallsBackToThePartyPool(t *testing.T) {
	a := newShopApp(t)
	a.state.Party[0].Money[pooltreasure.Gold] = 10
	a.state.PooledMoney[pooltreasure.Gold] = 100
	a.keys = scriptedKeys{ebiten.KeyEnter: true}
	if err := a.shopInput(); err != nil {
		t.Fatal(err)
	}
	if got := a.state.Party[0].Money[pooltreasure.Gold]; got != 10 {
		t.Fatalf("the character's 10 gold changed to %d although the pool paid", got)
	}
	if a.state.PooledMoney[pooltreasure.Platinum] != 17 || a.state.PooledMoney[pooltreasure.Gold] != 0 {
		t.Fatalf("pool is %v after paying 15 of 100 gold, want 17 platinum", a.state.PooledMoney)
	}
	if len(a.state.Party[0].Inventory) != 1 {
		t.Fatal("the purchase paid by the pool did not arrive")
	}
	if a.shop.message == "" {
		t.Fatal("no message for the pool purchase")
	}
}

// 買來的東西預設沒裝備上，跟撿到的一樣。
func TestBoughtItemsArriveUnreadied(t *testing.T) {
	a := newShopApp(t)
	a.keys = scriptedKeys{ebiten.KeyEnter: true}
	if err := a.shopInput(); err != nil {
		t.Fatal(err)
	}
	if _, ok := a.readiedWeapon(a.state.Party[0]); ok {
		t.Fatal("a bought item was already readied")
	}
}

// 商店與戰利品走同一條 ECL 邊界，判別靠三個旗標。分不出來的話，
// 走進商店會把整櫃存貨當成免費戰利品發下去。
func TestShopBoundaryNeedsTheServiceFlags(t *testing.T) {
	machine := &eclvm.Machine{Memory: map[uint16]uint16{}}
	a := &app{eventMachine: machine}
	result := eclvm.Result{TreasureRequests: []eclvm.TreasureRequest{{ItemBlock: 0x35}}}

	if a.isShopBoundary(result) {
		t.Fatal("a bare treasure request was taken for a shop")
	}
	machine.Memory[gamepack.ShopServiceKindAddress] = gamepack.ShopServiceKind
	machine.Memory[gamepack.ShopServiceEnabledAddress] = 1
	machine.Memory[gamepack.ShopServicePartyAddress] = 1
	if !a.isShopBoundary(result) {
		t.Fatal("the shop flags were not recognised")
	}
	// 帶錢的請求是戰利品，不是店——墓園那筆七種貨幣都有數量。
	withMoney := eclvm.Result{TreasureRequests: []eclvm.TreasureRequest{
		{Amounts: [7]uint16{0, 0, 0, 500}, ItemBlock: 0x33},
	}}
	if a.isShopBoundary(withMoney) {
		t.Fatal("a treasure request carrying money was taken for a shop")
	}
}

// 用真的 ECL bytes 跑到防具店那條服務邊界：從 `A919h`（CLEARMONSTERS）開始，
// VM 會走過 TREASURE 與三個 SAVE，在 COMBAT 停下。判別要認出這是商店而不是
// 戰利品，並且把 ITEM3.DAX 的 block 35h 當存貨載進來。
func TestRealArmouryBytesEnterTheShopService(t *testing.T) {
	zipPath := dosZIPForTests
	event, err := gamepack.ReadDOSInitialEvent(zipPath)
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	fixture := event
	fixture.HandlerAddress = 0xA919
	fixture.ScriptBlock = nil
	fixture.ScriptBlocks = map[uint16][]byte{0: event.ScriptBlocks[0]}
	session, err := gamepack.NewInitialEventSession(fixture)
	if err != nil {
		t.Fatal(err)
	}
	result, err := session.Machine().RunUntilEvent(64, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	a := &app{eventSession: session, eventMachine: session.Machine(),
		spawn: gamepack.Spawn{Map: gamepack.MapKey{Archive: 3}}}
	a.loadTreasure = func(archive, block uint8) ([]gamepack.TreasureItemRecord, error) {
		return gamepack.ReadDOSTreasureItemBlock(zipPath, archive, block)
	}
	if !a.isShopBoundary(result) {
		t.Fatalf("the real armoury boundary was not recognised as a shop: %+v", result)
	}
	if err := a.consumeInitialSearch(result); err != nil {
		t.Fatal(err)
	}
	if !a.shopActive {
		t.Fatal("the shop service did not open")
	}
	if a.treasureActive {
		t.Fatal("the shop was dispatched as free treasure")
	}
	names := map[string]bool{}
	for _, record := range a.shop.items {
		names[record.Name] = true
	}
	for _, want := range []string{"Long Sword", "Plate Mail", "Composite Long Bow"} {
		if !names[want] {
			t.Fatalf("the armoury stock has no %s (%d items)", want, len(a.shop.items))
		}
	}
}

// dosgolem 收據（docs/audit/dosgolem-shop-payment.json，#30）：同一個角色兩種錢包走進武具店
// 按 b、Return 買清單第一件，原版記錄的五個錢欄前後——remake 用同一份清單、同一個游標
// 買，錢包要逐欄相同（entry 11 的四捨五入、entry 15 的重鑄）。
func TestShopPaymentMatchesTheDosgolemReceipt(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "docs", "audit", "dosgolem-shop-payment.json"))
	if err != nil {
		t.Fatal(err)
	}
	var receipt struct {
		Scenarios map[string]struct {
			Frames []struct {
				Label  string         `json:"label"`
				Wallet map[string]int `json:"wallet"`
			} `json:"frames"`
		} `json:"scenarios"`
	}
	if err := json.Unmarshal(raw, &receipt); err != nil {
		t.Fatal(err)
	}
	names := []string{"copper", "silver", "electrum", "gold", "platinum"}
	toMoney := func(wallet map[string]int) [7]uint16 {
		var money [7]uint16
		for coin, name := range names {
			money[coin] = uint16(wallet[name])
		}
		return money
	}
	for name, scenario := range receipt.Scenarios {
		a := newShopApp(t)
		// 原版按 b 之後 Return 買到的是 HAND AXE（1 金）：500 金剩 499 → 99 白金 4 金。
		// 這一家的存貨是 ITEM3.DAX block 35h（防具店的鐵匠），清單順序 remake 照檔案，
		// 原版畫面照另一個順序排，所以用名字找同一件。
		a.shop.cursor = -1
		for index, record := range a.shop.items {
			if strings.EqualFold(record.Name, "Hand Axe") {
				a.shop.cursor = index
			}
		}
		if a.shop.cursor < 0 {
			t.Fatal("the armoury stock has no Hand Axe")
		}
		a.state.Party[0].Money = toMoney(scenario.Frames[0].Wallet)
		a.state.CharacterLibrary[0].Money = a.state.Party[0].Money
		a.keys = scriptedKeys{ebiten.KeyEnter: true}
		if err := a.shopInput(); err != nil {
			t.Fatal(err)
		}
		bought := a.shop.items[a.shop.cursor]
		want := toMoney(scenario.Frames[len(scenario.Frames)-1].Wallet)
		if got := a.state.Party[0].Money; got != want {
			t.Fatalf("%s: remake wallet %v after buying %q (%d gold), original %v (message %q)",
				name, got, bought.Name, bought.Price(), want, a.shop.message)
		}
		t.Logf("%s: %v → %v buying %q for %d gold matches the original", name,
			scenario.Frames[0].Wallet, scenario.Frames[len(scenario.Frames)-1].Wallet, bought.Name, bought.Price())
	}
}
