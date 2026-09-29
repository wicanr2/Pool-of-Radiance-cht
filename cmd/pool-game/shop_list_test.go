package main

import (
	"reflect"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// 原版武具店按 `b` 之後的第一頁（dosgolem `workplace/dosgolem-ref-shop/80-b`，
// 雜湊 `4d5cbdcf`），逐行讀自那一幀。價格欄 `4 DARTS` 原版顯示 1，記錄是 0，
// 那一欄另外說明（spec 168〈價格欄〉），這裡只對名稱。
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
