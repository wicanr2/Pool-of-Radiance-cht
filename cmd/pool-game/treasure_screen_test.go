package main

// 戰利品畫面（#47）：主選單依錢與物品的有無組、`View` 是人物頁、`Share` 的結果寫得出來。
// 原版那一側的收據是 `docs/audit/dos-treasure-screens.json`（市政廳交件的獎金：250 金、50 白金、1 珠寶，
// 五個隊員各拿 50 金與 10 白金，餘下的 1 件珠寶給鏈上第一個人，pool 歸零）。
// 全部從 `Update()` 送鍵。

import (
	"slices"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// treasurePartyWithReward 把隊伍縮到 members 人、pool 塞成原版那一筆獎金。
func treasurePartyWithReward(t *testing.T, members int) *app {
	t.Helper()
	a := bootCityParty(t, dosZIPForTests)
	a.state.Party = a.state.Party[:members]
	for index := range a.state.Party {
		a.state.Party[index].Money = [7]uint16{}
	}
	a.state.PooledMoney = [7]uint32{}
	a.state.PooledMoney[3] = 250 // 金
	a.state.PooledMoney[4] = 50  // 白金
	a.state.PooledMoney[6] = 1   // 珠寶
	a.treasureActive, a.treasureItems = true, nil
	a.cellEventPending, a.cellWaitingMenu = true, true
	a.enterTreasureMain()
	return a
}

func TestTreasureMenuFollowsMoneyAndItemPresence(t *testing.T) {
	a := treasurePartyWithReward(t, 5)
	if got := a.cellMenuOptions; !slices.Equal(got, []string{"View", "Take", "Pool", "Share", "Exit"}) {
		t.Fatalf("有錢時的主選單是 %v，原版是 VIEW TAKE POOL SHARE EXIT", got)
	}
	if a.screenName() != "treasure" {
		t.Fatalf("畫面識別字 %q，要 treasure", a.screenName())
	}
	// 分完之後錢在隊員身上、pool 空了：原版只剩 VIEW POOL EXIT。
	if err := selectMenuOption(t, a, "Share"); err != nil {
		t.Fatal(err)
	}
	if got := a.cellMenuOptions; !slices.Equal(got, []string{"View", "Pool", "Exit"}) {
		t.Fatalf("分完之後的主選單是 %v，原版是 VIEW POOL EXIT", got)
	}
}

// 分錢的數字對上原版：五個人各 50 金、10 白金，餘下的 1 件珠寶給第一個人，pool 歸零。
func TestTreasureShareFromKeysMatchesTheOriginalSplit(t *testing.T) {
	a := treasurePartyWithReward(t, 5)
	if err := selectMenuOption(t, a, "Share"); err != nil {
		t.Fatal(err)
	}
	if a.state.PooledMoney != ([7]uint32{}) {
		t.Fatalf("分完 pool 還剩 %v，原版是全部發完", a.state.PooledMoney)
	}
	for index, member := range a.state.Party {
		want := [7]uint16{}
		want[3], want[4] = 50, 10
		if index == 0 {
			want[6] = 1
		}
		if member.Money != want {
			t.Fatalf("%s 分到 %v，原版是 %v", strings.TrimSpace(member.Name), member.Money, want)
		}
	}
	// 狀態列要寫得出誰拿到什麼、pool 剩什麼——原版只換選單，這一行是 remake 自己的。
	for _, want := range []string{"+50 Gold", "+10 Platinum", "+1 Jewelry", "pool holds nothing"} {
		if !strings.Contains(a.statusLine, want) {
			t.Fatalf("分錢的狀態列 %q 少了 %q", a.statusLine, want)
		}
	}
}

// `View` 開的是人物頁（原版底列 `VIEW:TRADE DROP EXIT`），ESC 兩次回戰利品選單。
func TestTreasureViewOpensTheCharacterSheetAndReturns(t *testing.T) {
	a := treasurePartyWithReward(t, 5)
	if err := selectMenuOption(t, a, "View"); err != nil {
		t.Fatal(err)
	}
	if !a.viewSheetOpen || a.screenName() != "view-pick" {
		t.Fatalf("View 沒有開人物頁：open=%t screen=%s", a.viewSheetOpen, a.screenName())
	}
	if err := press(a, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if !a.viewSheetShown || a.screenName() != "view-sheet" {
		t.Fatalf("選了人之後沒有進資料頁：shown=%t screen=%s", a.viewSheetShown, a.screenName())
	}
	for range 2 {
		if err := press(a, ebiten.KeyEscape); err != nil {
			t.Fatal(err)
		}
	}
	if a.viewSheetOpen || a.screenName() != "treasure" || len(a.cellMenuOptions) == 0 {
		t.Fatalf("關掉人物頁沒有回戰利品選單：open=%t screen=%s options=%v",
			a.viewSheetOpen, a.screenName(), a.cellMenuOptions)
	}
}

// 拿錢那一頁的文字框要看得到整份幣種清單（原版 `SELECT TYPE OF COIN` 那一頁把它列在畫面上）。
func TestTreasureTakeMoneyListsEveryCurrency(t *testing.T) {
	a := treasurePartyWithReward(t, 5)
	if err := selectMenuOption(t, a, "Take"); err != nil {
		t.Fatal(err)
	}
	if a.screenName() != "treasure-take-money" {
		t.Fatalf("畫面識別字 %q，要 treasure-take-money（這一場沒有物品，Take 直接進錢）", a.screenName())
	}
	for _, want := range []string{"Gold 250", "Platinum 50", "Jewelry 1"} {
		if !strings.Contains(a.eventText, want) {
			t.Fatalf("拿錢那一頁的文字 %q 少了 %q", a.eventText, want)
		}
	}
}

// 負對照：沒有錢、只有物品時，主選單沒有 Share（spec 034 的 `View Take Pool`）。
func TestTreasureMenuWithoutMoneyHasNoShare(t *testing.T) {
	a := treasurePartyWithReward(t, 5)
	a.state.PooledMoney = [7]uint32{}
	a.treasureItems = []gamepack.TreasureItemRecord{{Name: "Scroll 1"}}
	a.enterTreasureMain()
	if got := a.cellMenuOptions; !slices.Equal(got, []string{"View", "Take", "Pool", "Exit"}) {
		t.Fatalf("只有物品時的主選單是 %v，要沒有 Share", got)
	}
}
