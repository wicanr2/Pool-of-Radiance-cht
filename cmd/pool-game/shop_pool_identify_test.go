package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
	pooltreasure "github.com/wicanr2/Pool-of-Radiance-cht/internal/treasure"
)

// 公款還有錢時離店，店主會問要不要回去拿（overlay-06 `0684h..0722h`）。
// Y 回到店裡、N 離開；離開時錢照樣留在公款（下一次進店才清），全程從 Update() 送鍵。
func TestLeavingTheShopWithPooledMoneyAsksFirst(t *testing.T) {
	a := newShopApp(t)
	a.state.PooledMoney[pooltreasure.Platinum] = 40
	pressKeys(t, a, ebiten.KeyEscape)
	if !a.shopActive || !a.shop.leaving || a.shop.message != a.text(msgShopLeaveMoney) {
		t.Fatalf("ESC with money in the pool: active %v leaving %v message %q", a.shopActive, a.shop.leaving, a.shop.message)
	}
	pressKeys(t, a, ebiten.KeyY)
	if !a.shopActive || a.shop.leaving {
		t.Fatal("Y (go back for the money) did not return to the shop menu")
	}
	pressKeys(t, a, ebiten.KeyEscape, ebiten.KeyN)
	if a.shopActive {
		t.Fatal("N (leave the money) did not leave the shop")
	}
	if a.state.PooledMoney[pooltreasure.Platinum] != 40 {
		t.Fatalf("leaving changed the pool to %v; the original only clears it on the next shop entry", a.state.PooledMoney)
	}
}

func TestLeavingTheShopWithAnEmptyPoolDoesNotAsk(t *testing.T) {
	a := newShopApp(t)
	pressKeys(t, a, ebiten.KeyEscape)
	if a.shopActive {
		t.Fatal("ESC with an empty pool did not leave the shop")
	}
}

// 回去拿錢的路：S）hare 把公款平分給隊員（overlay-21 entry 7，spec 040），之後離店不再問。
func TestSharingThePoolInTheShopThenLeaving(t *testing.T) {
	a := newShopApp(t)
	a.state.PooledMoney[pooltreasure.Platinum] = 40
	pressKeys(t, a, ebiten.KeyS)
	if a.state.PooledMoney != ([pooltreasure.CurrencyCount]uint32{}) {
		t.Fatalf("pool after S: %v", a.state.PooledMoney)
	}
	if got := a.state.Party[0].Money[pooltreasure.Platinum]; got != 40 {
		t.Fatalf("the only member holds %d platinum after sharing, want 40", got)
	}
	pressKeys(t, a, ebiten.KeyEscape)
	if a.shopActive {
		t.Fatal("ESC after sharing asked again")
	}
}

// hiddenLongSword 是 `ITEM8.DAX/20h` 的第 0 筆：`+30h` 是 A2h「+1」，被 `+35h` 位元 1 藏著，
// 內嵌名稱是 `Long Sword`。
func hiddenLongSword(t *testing.T) poolsave.Item {
	t.Helper()
	records, err := gamepack.ReadDOSTreasureItemBlock(dosZIPForTests, 8, 0x20)
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	record := records[0]
	if record.Name != "Long Sword" || record.Raw[0x30] != 0xa2 || record.Raw[0x35]&2 == 0 {
		t.Fatalf("ITEM8.DAX/20h#0 is %q parts %X hidden %02X", record.Name, record.Raw[0x2f:0x32], record.Raw[0x35])
	}
	return poolsave.Item{Name: record.Name, Raw: append([]byte(nil), record.Raw[:]...)}
}

func newIdentifyApp(t *testing.T, money [7]uint16) *app {
	t.Helper()
	table, err := gamepack.ReadDOSItemNameTable(dosZIPForTests)
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	a := newSellApp(t, money, "Dagger")
	a.itemNames = &table
	member := &a.state.Party[0]
	member.Inventory = append(member.Inventory, hiddenLongSword(t))
	a.state.CharacterLibrary[0] = *member
	return a
}

// V → 選長劍 → I → Y：付 200 金，`+35h` 清成 0，名稱變成 `Long Sword +1 `
// （原版每段字詞後面都接空白，記錄裡照存）。
func TestIdentifyingThroughUpdateRevealsTheBonus(t *testing.T) {
	a := newIdentifyApp(t, [7]uint16{pooltreasure.Gold: 3, pooltreasure.Platinum: 50})
	pressKeys(t, a, ebiten.KeyV, ebiten.KeyDown, ebiten.KeyI)
	if a.shop.sellStage != sellIdentify || !strings.Contains(a.shop.message, "Long Sword") {
		t.Fatalf("I offered %q at stage %d", a.shop.message, a.shop.sellStage)
	}
	pressKeys(t, a, ebiten.KeyY)
	member := a.state.Party[0]
	item := member.Inventory[1]
	if item.Name != "Long Sword +1 " || item.Raw[0x35] != 0 || string(item.Raw[1:1+item.Raw[0]]) != item.Name {
		t.Fatalf("after identify: %q hidden %02X raw name %q", item.Name, item.Raw[0x35], item.Raw[1:1+item.Raw[0]])
	}
	// 253 金 − 200 = 53 → 重鑄成 10 白金 3 金（entry 15）。
	want := [7]uint16{pooltreasure.Gold: 3, pooltreasure.Platinum: 10}
	if member.Money != want {
		t.Fatalf("money %v, want %v", member.Money, want)
	}
	if library := a.state.CharacterLibrary[0]; library.Money != want || library.Inventory[1].Name != item.Name {
		t.Fatalf("the library copy did not follow: %+v", library)
	}
	if !strings.Contains(a.shop.message, "Long Sword +1") {
		t.Fatalf("message %q", a.shop.message)
	}
	// 再鑑定一次：沒有藏字，照樣收 200。
	a.state.Party[0].Money[pooltreasure.Platinum] = 40
	pressKeys(t, a, ebiten.KeyI, ebiten.KeyY)
	if got := a.state.Party[0].Money; got != ([7]uint16{pooltreasure.Gold: 3}) {
		t.Fatalf("second identify left money %v, want the 200 charged anyway", got)
	}
	if !strings.Contains(a.shop.message, "Long Sword +1") || a.shop.message == a.text(msgShopIdentifyNoMoney) {
		t.Fatalf("second identify message %q", a.shop.message)
	}
}

// 角色與公款都不到 200 金：`Not Enough Money`，物品與錢都不動。N 則是根本不付。
func TestIdentifyingWithoutMoneyIsRefused(t *testing.T) {
	a := newIdentifyApp(t, [7]uint16{pooltreasure.Gold: 199})
	pressKeys(t, a, ebiten.KeyV, ebiten.KeyDown, ebiten.KeyI, ebiten.KeyN)
	if a.shop.sellStage != sellPicking || a.state.Party[0].Money[pooltreasure.Gold] != 199 {
		t.Fatal("N still charged or stayed on the offer")
	}
	pressKeys(t, a, ebiten.KeyI, ebiten.KeyY)
	item := a.state.Party[0].Inventory[1]
	if a.shop.message != a.text(msgShopIdentifyNoMoney) || item.Name != "Long Sword" || item.Raw[0x35] == 0 ||
		a.state.Party[0].Money[pooltreasure.Gold] != 199 {
		t.Fatalf("refusal: message %q item %q hidden %02X money %v",
			a.shop.message, item.Name, item.Raw[0x35], a.state.Party[0].Money)
	}
}

func TestIdentifiedItemSurvivesSaveAndLoad(t *testing.T) {
	a := newIdentifyApp(t, [7]uint16{pooltreasure.Platinum: 40})
	pressKeys(t, a, ebiten.KeyV, ebiten.KeyDown, ebiten.KeyI, ebiten.KeyY)
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
		item := character.Inventory[1]
		if item.Name != "Long Sword +1 " || item.Raw[0x35] != 0 || character.Money != ([7]uint16{}) {
			t.Fatalf("%s reloaded with %q hidden %02X money %v", character.Name, item.Name, item.Raw[0x35], character.Money)
		}
	}
}

// dosgolem 收據（docs/audit/dosgolem-shop-pool-identify.json）：原版買一把 PARTISAN 再
// V I I Y 鑑定。PARTISAN 沒有藏字，情境是「付了錢、看不出新東西」（角色付、公款付）與
// 「錢不夠」，比的是按 Y 前後的錢包與公款。remake 拿同一件存貨、同一個錢包從 Update() 按鍵。
func TestShopIdentifyMatchesTheDosgolemReceipt(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "docs", "audit", "dosgolem-shop-pool-identify.json"))
	if err != nil {
		t.Fatal(err)
	}
	type frame struct {
		Label  string         `json:"label"`
		Wallet map[string]int `json:"wallet"`
		Pool   map[string]int `json:"pool"`
	}
	var receipt struct {
		Scenarios map[string]struct {
			Frames []frame `json:"frames"`
		} `json:"scenarios"`
	}
	if err := json.Unmarshal(raw, &receipt); err != nil {
		t.Fatal(err)
	}
	table, err := gamepack.ReadDOSItemNameTable(dosZIPForTests)
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	coins := []string{"copper", "silver", "electrum", "gold", "platinum"}
	for _, name := range []string{"identify-paid", "identify-short", "identify-pool"} {
		frames := receipt.Scenarios[name].Frames
		pay := -1
		for index, f := range frames {
			if f.Label == "y" && index > 0 && frames[index-1].Label == "i" {
				pay = index
			}
		}
		if pay < 0 {
			t.Fatalf("%s: the receipt has no I → Y step", name)
		}
		before, after := frames[pay-1], frames[pay]
		a := newSellApp(t, [7]uint16{}, "Partisan")
		a.itemNames = &table
		for index, coin := range coins {
			a.state.Party[0].Money[index] = uint16(before.Wallet[coin])
			a.state.PooledMoney[index] = uint32(before.Pool[coin])
		}
		a.state.CharacterLibrary[0] = a.state.Party[0]
		pressKeys(t, a, ebiten.KeyV, ebiten.KeyI, ebiten.KeyY)
		for index, coin := range coins {
			if got := int(a.state.Party[0].Money[index]); got != after.Wallet[coin] {
				t.Errorf("%s: %s %d, the original has %d", name, coin, got, after.Wallet[coin])
			}
			if got := int(a.state.PooledMoney[index]); got != after.Pool[coin] {
				t.Errorf("%s: pool %s %d, the original has %d", name, coin, got, after.Pool[coin])
			}
		}
		t.Logf("%s: wallet %v pool %v → wallet %v pool %v matches the original",
			name, before.Wallet, before.Pool, after.Wallet, after.Pool)
	}
}
