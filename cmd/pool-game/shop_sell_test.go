package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	poolcharacter "github.com/wicanr2/Pool-of-Radiance-cht/internal/character"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
	pooltreasure "github.com/wicanr2/Pool-of-Radiance-cht/internal/treasure"
)

// newSellApp 是站在武具店裡、身上帶著 names 那幾件存貨的角色（spec 067〈賣出〉）。
// 角色欄位填到存檔驗得過，存讀檔那一條才能用同一份治具。
func newSellApp(t *testing.T, money [7]uint16, names ...string) *app {
	t.Helper()
	a := newShopApp(t)
	member := &a.state.Party[0]
	member.PortraitHead, member.PortraitBody, member.IconSize = 1, 1, 1
	member.MaxHP, member.CurrentHP = 10, 10
	member.Money = money
	for _, name := range names {
		found := false
		for _, record := range a.shop.items {
			if record.Name == name {
				member.Inventory = append(member.Inventory,
					poolsave.Item{Name: record.Name, Raw: append([]byte(nil), record.Raw[:]...)})
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("the armoury stock has no %s", name)
		}
	}
	a.state.CharacterLibrary[0] = *member
	return a
}

func pressKeys(t *testing.T, a *app, keys ...ebiten.Key) {
	t.Helper()
	for _, key := range keys {
		if err := press(a, key); err != nil {
			t.Fatalf("press %v: %v", key, err)
		}
	}
}

// 商店裡 V → S → Y 賣一件：出價是 `+3Ah` 的一半，錢以白金＋金直接加進錢包，
// 物品從隊伍與角色庫兩邊摘掉。全程從 Update() 送鍵。
func TestSellingThroughUpdatePaysHalfTheValue(t *testing.T) {
	a := newSellApp(t, [7]uint16{pooltreasure.Silver: 3, pooltreasure.Platinum: 10}, "Dagger", "Long Sword")
	pressKeys(t, a, ebiten.KeyV)
	if !a.shop.selling {
		t.Fatalf("V did not open the items to sell (message %q)", a.shop.message)
	}
	pressKeys(t, a, ebiten.KeyDown, ebiten.KeyS)
	if a.shop.sellStage != sellOffer || a.shop.sellPrice != 7 {
		t.Fatalf("S offered %d at stage %d, want 7 for the long sword", a.shop.sellPrice, a.shop.sellStage)
	}
	pressKeys(t, a, ebiten.KeyY)
	member := a.state.Party[0]
	want := [7]uint16{pooltreasure.Silver: 3, pooltreasure.Gold: 2, pooltreasure.Platinum: 11}
	if member.Money != want {
		t.Fatalf("money %v after selling a 15 gold long sword, want %v", member.Money, want)
	}
	if len(member.Inventory) != 1 || member.Inventory[0].Name != "Dagger" {
		t.Fatalf("inventory %+v, want only the dagger", member.Inventory)
	}
	if library := a.state.CharacterLibrary[0]; library.Money != want || len(library.Inventory) != 1 {
		t.Fatalf("the library copy did not follow the sale: %+v", library)
	}
	if a.shop.message != a.text(msgShopSellSold) {
		t.Fatalf("message %q after the sale", a.shop.message)
	}
	// ESC 回到架上，再 ESC 才離開商店。
	pressKeys(t, a, ebiten.KeyEscape)
	if a.shop == nil || a.shop.selling || !a.shopActive {
		t.Fatal("ESC from the sell page did not return to the stock list")
	}
}

// N 不賣：什麼都不動。
func TestSellingDeclinedKeepsTheItem(t *testing.T) {
	a := newSellApp(t, [7]uint16{}, "Long Sword")
	pressKeys(t, a, ebiten.KeyV, ebiten.KeyS, ebiten.KeyN)
	member := a.state.Party[0]
	if len(member.Inventory) != 1 || member.Money != [7]uint16{} || a.shop.sellStage != sellPicking {
		t.Fatalf("declining changed the character: %+v stage %d", member, a.shop.sellStage)
	}
}

// 穿戴中的東西不收（entry 20 `0D7Fh` 的 `Must be unreadied`），也不出價。
func TestSellingRefusesAReadiedItem(t *testing.T) {
	a := newSellApp(t, [7]uint16{}, "Long Sword")
	a.state.Party[0].Inventory[0].Raw[itemReadyOffset] = 1
	pressKeys(t, a, ebiten.KeyV, ebiten.KeyS, ebiten.KeyY)
	member := a.state.Party[0]
	if len(member.Inventory) != 1 || member.Money != [7]uint16{} {
		t.Fatalf("a readied item was sold: %+v", member)
	}
	if a.shop.message != a.text(msgShopSellUnready) {
		t.Fatalf("message %q, want the unready refusal", a.shop.message)
	}
}

// 帶著士氣的 NPC（記錄 `+84h` 位元 7）站在場上時，物品選單沒有 Sell（`10C9h`）。
func TestSellingIsNotOfferedForAMoraleNPC(t *testing.T) {
	a := newSellApp(t, [7]uint16{}, "Long Sword")
	member := &a.state.Party[0]
	member.NPC = true
	member.Record = make([]byte, poolsave.NPCRecordSize)
	member.Record[0x84], member.Record[0x10d] = 0xb3, 1
	pressKeys(t, a, ebiten.KeyV, ebiten.KeyS, ebiten.KeyY)
	if len(a.state.Party[0].Inventory) != 1 {
		t.Fatal("an NPC's item was sold")
	}
	if !strings.Contains(a.shop.message, member.Name) {
		t.Fatalf("message %q does not explain the refusal", a.shop.message)
	}
}

// 超重的那一段從按鍵走一次：錢包塞到只剩 2 枚的空間，賣 30 金的雙手劍換 3 白金 →
// 角色拿 2、公款拿 1，訊息接上 `Overloaded.  Money will be put in pool.`。
func TestSellingOverloadedPutsThePlatinumInThePool(t *testing.T) {
	// 治具角色的負重上限減掉雙手劍 250，再留 2 枚。
	a := newSellApp(t, [7]uint16{}, "Two-Handed Sword")
	capacity, err := poolcharacter.CarryCapacity(a.state.Party[0].Abilities[0], a.state.Party[0].ExceptionalStrength)
	if err != nil {
		t.Fatal(err)
	}
	a.state.Party[0].Money[pooltreasure.Copper] = uint16(capacity - 250 - 2)
	pressKeys(t, a, ebiten.KeyV, ebiten.KeyS, ebiten.KeyY)
	member := a.state.Party[0]
	if member.Money[pooltreasure.Platinum] != 2 || a.state.PooledMoney[pooltreasure.Platinum] != 1 {
		t.Fatalf("platinum %d, pool %d, want 2 and 1", member.Money[pooltreasure.Platinum],
			a.state.PooledMoney[pooltreasure.Platinum])
	}
	if !strings.HasSuffix(a.shop.message, a.text(msgShopSellOverloaded)) {
		t.Fatalf("message %q does not mention the overload", a.shop.message)
	}
}

// 賣完存檔再讀回來：錢與物品照賣完的樣子。
func TestSaleSurvivesSaveAndLoad(t *testing.T) {
	a := newSellApp(t, [7]uint16{pooltreasure.Platinum: 10}, "Dagger", "Long Sword")
	pressKeys(t, a, ebiten.KeyV, ebiten.KeyDown, ebiten.KeyS, ebiten.KeyY)
	a.state.Schema = poolsave.Schema
	path := filepath.Join(t.TempDir(), "save.json")
	if err := poolsave.WriteAtomic(path, a.state); err != nil {
		t.Fatal(err)
	}
	loaded, err := poolsave.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, character := range []poolsave.Character{loaded.Party[0], loaded.CharacterLibrary[0]} {
		if character.Money[pooltreasure.Platinum] != 11 || character.Money[pooltreasure.Gold] != 2 {
			t.Fatalf("%s reloaded with money %v, want 11 platinum 2 gold", character.Name, character.Money)
		}
		if len(character.Inventory) != 1 || character.Inventory[0].Name != "Dagger" {
			t.Fatalf("%s reloaded with inventory %+v", character.Name, character.Inventory)
		}
	}
}

// dosgolem 收據（docs/audit/dosgolem-shop-sell.json）：原版在武具店買一件、V I S Y 賣掉，
// 記錄五個錢欄與公款白金的前後。remake 拿同名的存貨、同一個錢包賣同一件，要逐欄相同。
func TestShopSaleMatchesTheDosgolemReceipt(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "docs", "audit", "dosgolem-shop-sell.json"))
	if err != nil {
		t.Fatal(err)
	}
	var receipt struct {
		Scenarios map[string]struct {
			Item          string         `json:"item"`
			Offer         int            `json:"offer"`
			Before        map[string]int `json:"wallet_before_sale"`
			After         map[string]int `json:"wallet_after_sale"`
			PoolPlatinumB int            `json:"pool_platinum_before_sale"`
			PoolPlatinumA int            `json:"pool_platinum_after_sale"`
			Strength      int            `json:"strength"`
			Load          int            `json:"load_before_sale"`
		} `json:"scenarios"`
	}
	if err := json.Unmarshal(raw, &receipt); err != nil {
		t.Fatal(err)
	}
	if len(receipt.Scenarios) == 0 {
		t.Fatal("the receipt has no scenarios")
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
		a := newSellApp(t, toMoney(scenario.Before), scenario.Item)
		a.state.Party[0].Abilities[0] = scenario.Strength
		a.state.Party[0].ExceptionalStrength = 0
		a.state.PooledMoney[pooltreasure.Platinum] = uint32(scenario.PoolPlatinumB)
		// 超重判定用的是原版記錄的 `+102h`；remake 用同一條算式現算，先對一次。
		member := a.state.Party[0]
		load, err := gamepack.CarriedWeight([][]byte{member.Inventory[0].Raw}, member.Money)
		if err != nil || load != scenario.Load {
			t.Fatalf("%s: remake load %d (%v), original +102h %d", name, load, err, scenario.Load)
		}
		pressKeys(t, a, ebiten.KeyV, ebiten.KeyS)
		if int(a.shop.sellPrice) != scenario.Offer {
			t.Fatalf("%s: remake offers %d for %s, original %d", name, a.shop.sellPrice, scenario.Item, scenario.Offer)
		}
		pressKeys(t, a, ebiten.KeyY)
		if got, want := a.state.Party[0].Money, toMoney(scenario.After); got != want {
			t.Fatalf("%s: remake wallet %v, original %v", name, got, want)
		}
		if got := a.state.PooledMoney[pooltreasure.Platinum]; got != uint32(scenario.PoolPlatinumA) {
			t.Fatalf("%s: remake pool platinum %d, original %d", name, got, scenario.PoolPlatinumA)
		}
		if len(a.state.Party[0].Inventory) != 0 {
			t.Fatalf("%s: the item is still carried", name)
		}
		t.Logf("%s: %s offered %d, wallet %v → %v matches the original", name, scenario.Item,
			scenario.Offer, scenario.Before, scenario.After)
	}
}
