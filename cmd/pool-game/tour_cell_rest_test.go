package main

// 導覽終點那一格紮營休息兩小時（#42）。
//
// 原版（dosgolem，`docs/audit/dos-tour-cell-rest.json`）：`e`→`r`→`h`→`i`→`i`→`r`
// 之後時鐘走到 00:05 就被打斷，畫面是城衛隊那一問
// （`YOU ARE ROUSTED BY THE CITY WATCH AND TOLD TO MOVE ALONG. WHAT DO YOU DO`，底列 `GO STAY`）。
// **不是**把導覽的結尾句再印一次。

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// atTourEnd 從標題以正常按鍵走完建角與羅夫導覽，停在城區 (0,4)。
func atTourEnd(t *testing.T) *app {
	t.Helper()
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	party := make([]poolsave.Character, 0, 6)
	for index := 0; index < 6; index++ {
		party = append(party, poolsave.Character{Name: string(rune('A' + index)),
			RaceID: "dwarf", GenderID: "male", ClassID: "fighter", AlignmentID: "lawful-good",
			Abilities: [6]int{16, 10, 10, 13, 10, 10}, MaxHP: 8, CurrentHP: 8,
			PortraitHead: 1, PortraitBody: 1, IconSize: 1})
	}
	application.state = poolsave.State{Schema: poolsave.Schema, CharacterLibrary: party, Party: party}
	application.saveState = func(poolsave.State) error { return nil }
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if err := press(application, ebiten.KeyB); err != nil {
		t.Fatal(err)
	}
	for tick := 0; tick < 20000 && !application.introDone; tick++ {
		if application.introWaiting || application.tourPage >= 0 {
			if err := press(application, ebiten.KeyEnter); err != nil {
				t.Fatal(err)
			}
			continue
		}
		application.keys = scriptedKeys{}
		if err := application.Update(); err != nil {
			t.Fatal(err)
		}
	}
	if !application.introDone {
		t.Fatal("羅夫導覽沒有結束")
	}
	if application.spawn.X != 0 || application.spawn.Y != 4 {
		t.Fatalf("導覽結束停在 (%d,%d)，原版是 (0,4)", application.spawn.X, application.spawn.Y)
	}
	return application
}

func TestRestingAtTheTourCellIsInterruptedByTheWatch(t *testing.T) {
	application := atTourEnd(t)
	tourLine := application.eventText
	if !strings.Contains(tourLine, application.gameText.Translate("FOR A COMMISSION.  GOOD-BYE AND GOOD")) &&
		!strings.Contains(tourLine, "COMMISSION") {
		t.Logf("導覽結尾句：%q", tourLine)
	}
	for _, key := range []ebiten.Key{ebiten.KeyE, ebiten.KeyR, ebiten.KeyH, ebiten.KeyI, ebiten.KeyI, ebiten.KeyR} {
		if err := press(application, key); err != nil {
			t.Fatal(err)
		}
	}
	// 休息本身是逐格推進的，跑到打斷為止。
	for tick := 0; tick < 4000 && application.campOpen; tick++ {
		application.keys = scriptedKeys{}
		if err := application.Update(); err != nil {
			t.Fatal(err)
		}
	}
	if application.campOpen {
		t.Fatalf("休息沒有結束：畫面=%s", application.screenName())
	}
	// 比的是句子裡最有識別力的那一段：原版與 remake 的空白與句尾標點不一定一樣
	// （原版是兩個空白、沒有問號），拿整句比會在無關的地方紅。
	watch := application.gameText.Translate("ROUSTED BY THE CITY WATCH")
	if !strings.Contains(application.eventText, watch) {
		t.Fatalf("休息完的文字是 %q，原版是城衛隊那一問（畫面=%s）",
			application.eventText, application.screenName())
	}
	// 原版量到的是走到 **00:05** 被打斷（`docs/audit/dos-tour-cell-rest.json` 的
	// `rest-5-r`：排了兩小時，第五分鐘就被攔下來）。
	if want := (gamepack.GameTime{0, 5}); application.gameTime != want {
		t.Fatalf("被打斷時的時鐘是 %v，原版是 %v", application.gameTime, want)
	}
	if !hasMenuOption(application, "GO") || !hasMenuOption(application, "STAY") {
		t.Fatalf("沒有 GO／STAY 兩個選項：%v（畫面=%s）",
			application.cellMenuOptions, application.screenName())
	}
}
