package main

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// 最小重現：城堡院子 → 二樓 → 三樓（區塊 7）→ 覲見廳。從城門那條既有的
// 治具起跑（`walkStojanowGateIntoTheCastle` 帶著注入的索寇旗標，只用來把
// 隊伍送到院子），只驗 spec 137 第 9～11 段的走法，不是主線收據。
func TestCastleStairsLeadToTheAudienceHall(t *testing.T) {
	application := walkStojanowGateIntoTheCastle(t)
	driver := &mainlineDriver{t: t, a: application, pilot: &tacticalPilot{},
		step: func(key ebiten.Key) {
			if err := press(application, key); err != nil {
				t.Fatal(err)
			}
		}}
	driver.castleUpstairs()
	if got := application.eventSession.CurrentBlockID(); got != 7 {
		t.Fatalf("the keep stairs led to block %d, want 7", got)
	}
	driver.note("in block 7 at (%d,%d)", application.spawn.X, application.spawn.Y)
	saw, ending, pages := driver.audienceHall()
	// 這條只驗走法：覲見廳那一格要真的站上去、衛兵那一場要真的開打。誰贏
	// 由隊伍強度決定——治具的 60 HP 一級矮人隊伍照樣被十二名 8 級戰士打光，
	// 停在 The END!。結局的收據在 `TestDefeatingTyranthraxusSetsTheVictoryFlag`
	// 與主線探針（spec 137）。
	if application.eventMachine == nil || application.eventMachine.Memory[0x4A6D]&8 == 0 {
		t.Fatalf("the audience hall text never ran: %s", driver.flags())
	}
	if !strings.Contains(driver.log[len(driver.log)-1], "audience hall done") {
		t.Fatalf("audience hall did not settle: %s", driver.flags())
	}
	t.Logf("tyranthraxus=%t ending=%t pages=%d gameOver=%t %s", saw, ending, pages,
		application.gameOver, driver.flags())
}
