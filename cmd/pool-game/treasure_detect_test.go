package main

// 戰利品選單的 " Detect Exit"（spec 150〈Detect〉，issue #111），全部從 `Update()` 送鍵。

import (
	"slices"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

func TestTreasureDetectNeedsItemsAndAMemorisedDetectMagic(t *testing.T) {
	a := treasurePartyWithReward(t, 5)
	a.state.PooledMoney = [7]uint32{}
	a.currentCharacter = 0
	a.state.Party[0].Memorised = []uint8{0x05 | gamepack.MemorisedSpellFlag, 0, 0}
	a.treasureItems = []gamepack.TreasureItemRecord{{Name: "Scroll 1"}}
	a.enterTreasureMain()
	// 還沒記完的（第 7 位立著）不算：`0F00h` 比的是整個位元組。
	if got := a.cellMenuOptions; !slices.Equal(got, []string{"View", "Take", "Pool", "Exit"}) {
		t.Fatalf("menu %v with an unfinished Detect Magic", got)
	}
	a.state.Party[0].Memorised = []uint8{0x01, 0x05, 0x05}
	a.enterTreasureMain()
	if got := a.cellMenuOptions; !slices.Equal(got, []string{"View", "Take", "Pool", "Detect", "Exit"}) {
		t.Fatalf("menu %v, want VIEW TAKE POOL DETECT EXIT", got)
	}
	// 沒有物品就沒有 Detect（`0EE0h`）。
	items := a.treasureItems
	a.treasureItems = nil
	a.enterTreasureMain()
	if slices.Contains(a.cellMenuOptions, "Detect") {
		t.Fatalf("menu %v offers Detect without items", a.cellMenuOptions)
	}
	a.treasureItems = items
	a.enterTreasureMain()
	if err := selectMenuOption(t, a, "Detect"); err != nil {
		t.Fatal(err)
	}
	// 施掉的是第一個（`0EE6h` 由小到大），第二個還在，所以選項還在。
	if got := a.state.Party[0].Memorised; !slices.Equal(got, []uint8{0x01, 0x00, 0x05}) {
		t.Fatalf("memorised after Detect %v, want the first Detect Magic gone", got)
	}
	if !combatEffects(a.state.Party[0].Effects).Has(0x05) {
		t.Fatalf("effects %v, want Detect Magic (05h) on the caster", a.state.Party[0].Effects)
	}
	if !slices.Contains(a.cellMenuOptions, "Detect") || !a.treasureMenuShown() {
		t.Fatalf("menu %v after the first cast", a.cellMenuOptions)
	}
	if err := selectMenuOption(t, a, "Detect"); err != nil {
		t.Fatal(err)
	}
	if slices.Contains(a.cellMenuOptions, "Detect") {
		t.Fatalf("menu %v still offers Detect with nothing memorised", a.cellMenuOptions)
	}
}
