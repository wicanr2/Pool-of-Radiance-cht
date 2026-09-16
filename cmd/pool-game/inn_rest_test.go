package main

// 旅店與「睡得著嗎」的那一環（#38，spec 114）。
//
// **收錢與開紮營已經有 `TestTheInnChargesAPlatinumAndOpensCamp` 釘著**
//（`inn_test.go`：走進去、一枚白金、`PROGRAM 9` 開紮營、睡滿回血），這裡不重複。
// 這三條測的是 #24 把入口 2 接上之後才存在的那一環：付錢的同一支腳本在
// `ecl3/0 A1A9h` 自己 `GOSUB 9A63h` 重跑城區入口 2，於是 `6DD2h`／`6DD3h` 從
// 1／101 變成 0／0——**那才是「旅店睡得著、街上睡不著」的機制**——以及房間
// 踏回街上就沒了（`AE6Ah`）。順便把駕駛的 `restAtTheInn()` 走一遍。
//
// 都從 `Update()` 送鍵。產品碼沒有為這一條改過。

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	pooltreasure "github.com/wicanr2/Pool-of-Radiance-cht/internal/treasure"
)

// innDriver 把城區的隊伍接上駕駛，並把大家的白金設成 want。
func innDriver(t *testing.T, platinum uint16) *mainlineDriver {
	t.Helper()
	application := bootCityParty(t, dosZIPForTests)
	if application.spawn.Map.Archive != 3 || application.spawn.Map.BlockID != 0 {
		t.Skipf("開場沒有停在城區，而是 GEO%d/%d", application.spawn.Map.Archive, application.spawn.Map.BlockID)
	}
	for index := range application.state.Party {
		application.state.Party[index].Money[pooltreasure.Platinum] = platinum
	}
	return &mainlineDriver{t: t, a: application, pilot: &tacticalPilot{},
		step: func(key ebiten.Key) {
			if err := press(application, key); err != nil {
				t.Fatal(err)
			}
		}}
}

// 開了房間，這一區的打斷設定就從 1／101 變成 0／0。
func TestTakingAnInnRoomStopsTheRestInterruption(t *testing.T) {
	d := innDriver(t, 3)
	a := d.a
	// 負對照：還沒開房間時城區是 1／101（`ecl3/0 9A63h`：還沒通關且 `4A07 == 0`）。
	if err := a.runRestEntry(); err != nil {
		t.Fatal(err)
	}
	if got := a.restInterruption(); got.Period != 1 || got.Threshold != 101 {
		t.Fatalf("開房間之前城區的打斷設定是 %d／%d，`9A63h` 是 1／101", got.Period, got.Threshold)
	}
	before := a.state.Party[0].Money[pooltreasure.Platinum]

	if !d.restAtTheInn() {
		t.Fatalf("旅店沒開成：4A07=%d 位置 %+v 文字 %q",
			a.eventMachine.Memory[0x4A07], a.spawn, strings.TrimSpace(a.eventText))
	}

	if got := a.eventMachine.Memory[0x4A07]; got != 1 {
		t.Fatalf("4A07=%d，付過錢該是 1", got)
	}
	if got := a.state.Party[0].Money[pooltreasure.Platinum]; got != before-1 {
		t.Fatalf("付錢的人剩 %d 枚白金，原本 %d，一晚該扣一枚", got, before)
	}
	// 腳本自己在 `A1A9h` 重跑了入口 2，這時 `4A07 == 1` 所以是 0／0。
	if got := a.restInterruption(); got.Period != 0 || got.Threshold != 0 {
		t.Fatalf("開了房間之後打斷設定還是 %d／%d，該是 0／0", got.Period, got.Threshold)
	}
}

// 負對照一：全隊都沒有白金就開不成房間，而且錢一毛都不能少。
func TestTheInnRefusesWithoutAPlatinumPiece(t *testing.T) {
	d := innDriver(t, 0)
	a := d.a
	if d.restAtTheInn() {
		t.Fatalf("沒有白金卻開成了房間：4A07=%d", a.eventMachine.Memory[0x4A07])
	}
	if got := a.eventMachine.Memory[0x4A07]; got != 0 {
		t.Fatalf("4A07=%d，沒付錢該是 0", got)
	}
	for index, member := range a.state.Party {
		if member.Money[pooltreasure.Platinum] != 0 {
			t.Fatalf("第 %d 個人憑空多了白金：%v", index, member.Money)
		}
		if member.Money[pooltreasure.Gold] != 500 {
			t.Fatalf("第 %d 個人的金幣被動了：%d，該是 500", index, member.Money[pooltreasure.Gold])
		}
	}
}

// 負對照二：房間只管這一次——踏回街上，`ecl3/0 AE6Ah` 開頭就把 `4A07` 清 0，
// 於是打斷設定又變回 1／101。少了這一條，「開了房間就一直有效」會被當成真的。
func TestLeavingTheInnCellGivesUpTheRoom(t *testing.T) {
	d := innDriver(t, 3)
	a := d.a
	if !d.restAtTheInn() {
		t.Fatalf("旅店沒開成：4A07=%d", a.eventMachine.Memory[0x4A07])
	}
	for guard := 0; guard < 8 && a.campOpen; guard++ {
		d.step(ebiten.KeyEscape)
	}
	if a.campOpen {
		t.Fatal("離不開紮營畫面")
	}
	street := func(x, y int) bool { return d.terrain(x, y) == 0 }
	facing, ok := d.approachIfPossible("back to the street", street, func(int, int) bool { return true })
	if !ok {
		t.Fatal("旅店那一格旁邊沒有街道")
	}
	d.face(facing)
	d.step(ebiten.KeyArrowUp)
	d.settle()
	if got := a.eventMachine.Memory[0x4A07]; got != 0 {
		t.Fatalf("踏回街上之後 4A07=%d，`AE6Ah` 該把它清成 0", got)
	}
	if err := a.runRestEntry(); err != nil {
		t.Fatal(err)
	}
	if got := a.restInterruption(); got.Period != 1 || got.Threshold != 101 {
		t.Fatalf("房間退掉之後打斷設定是 %d／%d，該回到 1／101", got.Period, got.Threshold)
	}
}
