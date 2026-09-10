package main

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// 紮營被打斷之後，原版把棒子交給 ECL 的入口 3（spec 114）。貧民區那一份是
// 城衛隊：印一句話，再讓玩家選 GO 或 STAY。
//
// 這一條走玩家真的會走的路：真的紮營、真的被打斷、真的按鍵推進文字，
// 而不是直接去跑那個入口——直接跑證明不了呼叫端有接上。
func TestCampInterruptionHandsOffToTheCityWatchScript(t *testing.T) {
	application := bootCityParty(t, dosZIPForTests)
	if application.spawn.Map.BlockID != 0 {
		t.Skipf("開場沒有停在貧民區，而是 GEO%d/%d",
			application.spawn.Map.Archive, application.spawn.Map.BlockID)
	}

	// spec 114 的兩個參數：每幾刻問一次、百分位門檻。
	// 設成「每一刻都問，而且一定中」，打斷才是這條測試的前提而不是運氣。
	application.eventMachine.Memory[gamepack.RestInterruptionPeriodAddress] = 1
	application.eventMachine.Memory[gamepack.RestInterruptionThresholdAddress] = 100
	for step := 0; step < 8; step++ {
		application.restDuration = application.restDuration.Increase(gamepack.RestFieldHours)
	}

	application.restParty()

	for tick := 0; tick < 200 && !application.cellWaitingMenu; tick++ {
		if err := press(application, ebiten.KeyEnter); err != nil {
			t.Fatal(err)
		}
	}

	// 原版量到的是 `0, 4 W 00:05`：被打斷的那一刻正好睡了一刻（spec 114 的
	// 基準畫面）。時鐘沒走就代表休息沒有推進時間，效果也不會跟著到期。
	if got := application.gameTime.Minutes(); got != gamepack.RestMinutesPerTick {
		t.Fatalf("被打斷之後時鐘走了 %d 分，原版是 %d", got, gamepack.RestMinutesPerTick)
	}
	if !strings.Contains(strings.ToUpper(application.eventText), "CITY WATCH") {
		t.Fatalf("打斷之後的文字是 %q，該是城衛隊那一句", application.eventText)
	}
	if !application.cellWaitingMenu {
		t.Fatalf("城衛隊那一問沒有開出選單：文字 %q", application.eventText)
	}
	want := []string{"GO", "STAY"}
	if len(application.cellMenuOptions) != len(want) {
		t.Fatalf("選單是 %v，要 %v", application.cellMenuOptions, want)
	}
	for index, option := range want {
		if application.cellMenuOptions[index] != option {
			t.Fatalf("選單第 %d 項是 %q，要 %q", index, application.cellMenuOptions[index], option)
		}
	}
}

// 負對照：這一區不打擾的時候（Period 是 0，也是原版的初值），紮營就只是紮營，
// 不該有人來問話。少了它，上面那條證不了因果——文字有可能是別的地方印的。
func TestRestingWithoutInterruptionNeverCallsTheScript(t *testing.T) {
	application := bootCityParty(t, dosZIPForTests)
	if application.spawn.Map.BlockID != 0 {
		t.Skipf("開場沒有停在貧民區，而是 GEO%d/%d",
			application.spawn.Map.Archive, application.spawn.Map.BlockID)
	}

	application.eventMachine.Memory[gamepack.RestInterruptionPeriodAddress] = 0
	application.eventMachine.Memory[gamepack.RestInterruptionThresholdAddress] = 0
	for step := 0; step < 8; step++ {
		application.restDuration = application.restDuration.Increase(gamepack.RestFieldHours)
	}

	application.restParty()

	if strings.Contains(strings.ToUpper(application.eventText), "CITY WATCH") {
		t.Fatalf("沒有被打斷卻跑出城衛隊：%q", application.eventText)
	}
	if application.cellWaitingMenu {
		t.Fatalf("沒有被打斷卻停在選單上：%v", application.cellMenuOptions)
	}
}
