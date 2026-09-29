package main

// AI 代打中途交還（#123，spec 167）：overlay-09 entry 7 在回合開頭、換完武器之後、每一步之前
// 問鍵；SPACE 收回之後分數寫 14h、不叫 entry 34，重選又是同一位，剩下的腳程歸玩家。

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// handBackFixture：隊員（roster 1）在 (4,5)，敵方在 (12,5)，要走好幾步才搆得到。
func handBackFixture() (*app, *tacticalState) {
	state := newFoeTurnState(12, 5, 4, 5, 20)
	state.PartySlot = []int{-1, 0, -1}
	state.AIDriven = aiDriven(state.PartySlot)
	state.FoeTargets = make([]uint8, 3)
	state.TacticModes = make([]uint8, 3)
	state.Scores[1], state.Scores[2] = 5, 10
	state.Mover = 1
	application := &app{mode: modeAdventure, tacticalPreview: true, roller: fixedRoller{1},
		tactical: state, language: languageEnglish, gameSpeed: campSpeedDefault}
	application.state.Party = []poolsave.Character{{Name: "A"}}
	return application, state
}

// Q 之後 AI 走一步停一個影格；走了一步之後按 SPACE，那一位停在原地、交回玩家。
func TestSpaceTakesTheQuickCharacterBackMidWalk(t *testing.T) {
	application, state := handBackFixture()
	start := state.Roster[1]
	if err := press(application, ebiten.KeyQ); err != nil {
		t.Fatal(err)
	}
	if application.foeRun == nil || state.Roster[1] != start {
		t.Fatalf("Q should pause the AI at entry 1 `01ACh` before it walks: run %v cell %+v", application.foeRun != nil,
			state.Roster[1])
	}
	// `01ACh` 放行，停在第一步之前（`0843h`）；再放行走一步，停在第二步之前。
	for frame := 0; frame < 2; frame++ {
		if err := press(application, ebiten.KeyEnter); err != nil {
			t.Fatal(err)
		}
	}
	moved := state.Roster[1]
	if dx, dy := int(moved.X)-int(start.X), int(moved.Y)-int(start.Y); max(abs(dx), abs(dy)) != 1 {
		t.Fatalf("after two frames the AI stands at %+v from %+v, want exactly one step", moved, start)
	}
	budget := state.Budgets[1]
	if err := press(application, ebiten.KeySpace); err != nil {
		t.Fatal(err)
	}
	if application.foeRun != nil || application.state.Party[0].Quick || state.aiDrives(1) {
		t.Fatalf("SPACE did not hand the character back: run %v quick %v ai %v", application.foeRun != nil,
			application.state.Party[0].Quick, state.aiDrives(1))
	}
	// `109Ah` 14h → 重選還是他（敵方 10 分）→ entry 3 `0248h` 改 13h；位置與腳程不動。
	if state.Mover != 1 || state.Scores[1] != resumedInitiative || state.Roster[1] != moved || state.Budgets[1] != budget {
		t.Fatalf("mover %d score %d cell %+v budget %d; want 1, 13h, %+v, %d", state.Mover, state.Scores[1],
			state.Roster[1], state.Budgets[1], moved, budget)
	}
	// 交回來之後是玩家的指令迴圈：M 進移動。
	if err := press(application, ebiten.KeyM); err != nil {
		t.Fatal(err)
	}
	if !state.Moving {
		t.Fatal("the handed-back character does not take commands")
	}
}

// 回合一開頭按 SPACE（entry 1 `001Ch`）：戰術模式照擲，分數 14h→13h，同一位由玩家走。
func TestSpaceAtTheStartOfAQuickTurnHandsItBack(t *testing.T) {
	application, state := handBackFixture()
	application.state.Party[0].Quick = true
	state.AIDriven[1] = true
	if err := press(application, ebiten.KeySpace); err != nil {
		t.Fatal(err)
	}
	if state.aiDrives(1) || state.Mover != 1 || state.Scores[1] != resumedInitiative {
		t.Fatalf("ai %v mover %d score %d; want the player's turn at 13h", state.aiDrives(1), state.Mover, state.Scores[1])
	}
	// fixedRoller 1：`0054h` 模式 0 重擲 → 1d8 = 1 → 1d4 = 1。
	if state.TacticModes[1] != 1 {
		t.Fatalf("tactic mode %d, want the roll of `0045h..00B1h` written back", state.TacticModes[1])
	}
	if state.Roster[1].X != 4 {
		t.Fatal("the character moved although it was taken back before acting")
	}
}
