package main

import (
	"reflect"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	pooltreasure "github.com/wicanr2/Pool-of-Radiance-cht/internal/treasure"
)

// 原版武具店按 `b` 之後的第一頁（dosgolem `workplace/dosgolem-ref-shop/80-b`，
// 雜湊 `4d5cbdcf`），逐行讀自那一幀。價格欄另有一條（記錄是 0 的上架改成 1，
// spec 168〈價格欄〉），這裡只對名稱。
var dosArmouryFirstPage = []string{
	"BATTLE AXE", "HAND AXE", "BARDICHE", "BEC DE CORBIN", "BILL-GUISARME",
	"BO STICK", "CLUB", "DAGGER", "4 DARTS", "FAUCHARD", "FAUCHARD-FORK", "FLAIL",
	"MILITARY FORK", "GLAIVE", "GLAIVE-GUISARME", "GUISARME", "GUISARME-VOULGE",
	"HALBERD", "LUCERN HAMMER",
}

func newShopListApp(t *testing.T) *app {
	t.Helper()
	a := newShopApp(t)
	names, err := gamepack.ReadDOSItemNameTable(dosZIPForTests)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	a.itemNames = &names
	return a
}

func (a *app) shopListPage() []string {
	page := []string{}
	for row := 0; row < shopListLines && a.shop.top+row < len(a.shop.items); row++ {
		page = append(page, strings.ToUpper(a.shopListName(a.shop.items[a.shop.top+row])))
	}
	return page
}

// 清單順序、名稱與一打開的反白都照原版：第一頁 19 行與基準逐行相同，
// 反白在第二項 HAND AXE。
func TestShopListFirstPageMatchesTheDOSShot(t *testing.T) {
	a := newShopListApp(t)
	pressKeys(t, a, ebiten.KeyB)
	if got := a.shopListPage(); !reflect.DeepEqual(got, dosArmouryFirstPage) {
		t.Fatalf("first page\n got %q\nwant %q", got, dosArmouryFirstPage)
	}
	if a.shop.cursor != 1 || a.shop.top != 0 {
		t.Fatalf("cursor %d top %d, want the highlight on row 1 (HAND AXE) of the first page",
			a.shop.cursor, a.shop.top)
	}
	want := []messageID{msgShopCommandBuy, msgShopListNext, msgShopCommandExit}
	if got := a.shopListCommands(); !reflect.DeepEqual(got, want) {
		t.Fatalf("first page commands %v, want Buy Next Exit", got)
	}
}

// 翻頁照原版：End／Home 移一行，N 整頁跳而頁內那一行不變，中間頁有 NEXT 與 PREV，
// 最後一頁只剩 PREV、最後一行是 SHIELD；離開清單再開，反白回到 HAND AXE，
// 起點只移到看得見它（dosgolem `81-End`..`88-b`）。
func TestShopListPagesLikeTheOriginal(t *testing.T) {
	a := newShopListApp(t)
	pressKeys(t, a, ebiten.KeyB, ebiten.KeyEnd, ebiten.KeyEnd, ebiten.KeyHome)
	if a.shop.cursor != 2 {
		t.Fatalf("End End Home left the highlight on %d, want 2", a.shop.cursor)
	}
	pressKeys(t, a, ebiten.KeyN)
	page := a.shopListPage()
	if a.shop.top != shopListLines || a.shop.cursor != shopListLines+2 ||
		page[0] != "HAMMER" || page[2] != "JO STICK" {
		t.Fatalf("NEXT: top %d cursor %d page %q, want HAMMER on top and JO STICK highlighted",
			a.shop.top, a.shop.cursor, page)
	}
	want := []messageID{msgShopCommandBuy, msgShopListNext, msgShopListPrev, msgShopCommandExit}
	if got := a.shopListCommands(); !reflect.DeepEqual(got, want) {
		t.Fatalf("middle page commands %v, want Buy Next Prev Exit", got)
	}
	pressKeys(t, a, ebiten.KeyN)
	page = a.shopListPage()
	if page[0] != "TRIDENT" || page[len(page)-1] != "SHIELD" ||
		strings.ToUpper(a.shopListName(a.shop.items[a.shop.cursor])) != "COMPOSITE LONG BOW" {
		t.Fatalf("last page %q, cursor on %q", page, a.shop.items[a.shop.cursor].Name)
	}
	want = []messageID{msgShopCommandBuy, msgShopListPrev, msgShopCommandExit}
	if got := a.shopListCommands(); !reflect.DeepEqual(got, want) {
		t.Fatalf("last page commands %v, want Buy Prev Exit", got)
	}
	top := a.shop.top
	pressKeys(t, a, ebiten.KeyN)
	if a.shop.top != top {
		t.Fatal("N on the last page turned the page")
	}
	pressKeys(t, a, ebiten.KeyE)
	if a.shop.buying || !a.shopActive {
		t.Fatal("E in the list should return to the shop menu")
	}
	pressKeys(t, a, ebiten.KeyB)
	if a.shop.cursor != 1 || a.shop.top != 1 || a.shopListPage()[0] != "HAND AXE" {
		t.Fatalf("reopened with cursor %d top %d, want HAND AXE on top and highlighted",
			a.shop.cursor, a.shop.top)
	}
	// PREV 從起點 1 往回整頁：夾到第一頁（這一步 dosgolem 沒拍，是 remake 的夾法）。
	pressKeys(t, a, ebiten.KeyP)
	if a.shop.top != 0 || a.shop.cursor != 0 {
		t.Fatalf("PREV: top %d cursor %d, want the first page from the top", a.shop.top, a.shop.cursor)
	}
}

// 清單裡的 B 是 BUY：原版按 `b` 開清單、再按 `b` 買到反白的 HAND AXE。
func TestShopListBBuysTheHighlightedItem(t *testing.T) {
	a := newShopListApp(t)
	pressKeys(t, a, ebiten.KeyB, ebiten.KeyB)
	inventory := a.state.Party[0].Inventory
	if len(inventory) != 1 || !strings.EqualFold(inventory[0].Name, "Hand Axe") {
		t.Fatalf("B B bought %v (%q), want the Hand Axe", inventory, a.shop.message)
	}
}

// 記錄價格為 0 的彈藥上架時是 1 金，清單印 1、買也收 1（overlay-06 `0071h`，
// spec 168〈價格欄〉）。收據是 dosgolem 在武具店買 `4 DARTS` 兩次
// （`workplace/dosgolem-probe-price`）：人物資料頁 GOLD 120 → PLATINUM 23 GOLD 4
// → PLATINUM 23 GOLD 3。全程從 Update() 送原版那一組鍵：b、End ×7、b、e。
func TestShopListChargesOneForZeroPricedAmmunition(t *testing.T) {
	a := newShopListApp(t)
	a.state.Party[0].Money = [7]uint16{}
	a.state.Party[0].Money[pooltreasure.Gold] = 120
	a.state.CharacterLibrary[0].Money = a.state.Party[0].Money
	want := [][2]uint16{{23, 4}, {23, 3}}
	for round, wallet := range want {
		pressKeys(t, a, ebiten.KeyB)
		for step := 0; step < 7; step++ {
			pressKeys(t, a, ebiten.KeyEnd)
		}
		record := a.shop.items[a.shop.cursor]
		if name := strings.ToUpper(a.shopListName(record)); name != "4 DARTS" || record.Price() != 1 {
			t.Fatalf("round %d: highlight on %q priced %d, want 4 DARTS priced 1", round, name, record.Price())
		}
		pressKeys(t, a, ebiten.KeyB, ebiten.KeyE)
		money := a.state.Party[0].Money
		if got := [2]uint16{money[pooltreasure.Platinum], money[pooltreasure.Gold]}; got != wallet {
			t.Fatalf("round %d: wallet platinum/gold %v, original %v (%q)", round, got, wallet, a.shop.message)
		}
	}
	inventory := a.state.Party[0].Inventory
	if len(inventory) != 2 {
		t.Fatalf("bought %d items, want two lots of darts", len(inventory))
	}
	bought := gamepack.TreasureItemRecord{}
	copy(bought.Raw[:], inventory[0].Raw)
	if bought.Price() != 1 {
		t.Fatalf("the darts carry price %d; the original writes 1 into the stocked record", bought.Price())
	}
}

// 上架只改價格為 0 的那幾筆；其餘照記錄（長劍 15、板甲 400）。
func TestShopStockOnlyRaisesZeroPrices(t *testing.T) {
	raw := armouryStock(t)
	stocked := shopStock(raw)
	changed := 0
	for index, record := range stocked {
		original := raw[len(raw)-1-index]
		switch {
		case original.Price() == 0 && record.Price() != 1:
			t.Fatalf("%s: zero price became %d, want 1", record.Name, record.Price())
		case original.Price() != 0 && record.Price() != original.Price():
			t.Fatalf("%s: price %d changed to %d", record.Name, original.Price(), record.Price())
		case original.Price() == 0:
			changed++
		}
	}
	if changed != 5 {
		t.Fatalf("%d records raised to 1, want 5 (darts, javelins, quarrels, arrows, sling)", changed)
	}
}
