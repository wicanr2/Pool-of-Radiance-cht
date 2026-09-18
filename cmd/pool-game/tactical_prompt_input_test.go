package main

// 「Y 繼續戰鬥　N 結束戰鬥」那個提示：玩家按得到嗎（spec 062）。
//
// 直接呼叫 `tacticalInput` 證明的是規則對，不是玩家按得動——中間還隔著 `Update()`
// 的輸入分派，而那一層出錯時兩者在報表上長得一樣（CLAUDE.md §9 的門選單）。
// 2026-09-18 發行包擷圖就卡在這個提示上：畫面寫著 Y／N，xdotool 按下去沒反應。

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// stagedTacticalBattle 從正常按鍵走到戰術盤上。
func stagedTacticalBattle(t *testing.T) *app {
	t.Helper()
	application := newNormalSlumWanderingEncounter(t, 1, 1)
	if err := selectMenuOption(t, application, "COMBAT"); err != nil {
		t.Fatal(err)
	}
	for guard := 0; guard < 40 && application.tactical == nil; guard++ {
		if err := press(application, ebiten.KeyEnter); err != nil {
			t.Fatal(err)
		}
	}
	if application.tactical == nil {
		t.Fatalf("按 ENTER 進不了戰術盤：combat=%v preview=%v 畫面=%s",
			application.combatActive, application.tacticalPreview, application.screenName())
	}
	return application
}

func TestTacticalContinuePromptAnswersFromKeys(t *testing.T) {
	application := stagedTacticalBattle(t)
	state := application.tactical
	state.Prompt = true
	if err := press(application, ebiten.KeyY); err != nil {
		t.Fatal(err)
	}
	if state.Prompt {
		t.Fatalf("畫面寫著「Y 繼續戰鬥」，從 Update() 按 Y 卻沒反應（畫面=%s）", application.screenName())
	}

	application = stagedTacticalBattle(t)
	state = application.tactical
	state.Prompt = true
	if err := press(application, ebiten.KeyN); err != nil {
		t.Fatal(err)
	}
	if state.Prompt {
		t.Fatalf("按 N 也沒反應（畫面=%s）", application.screenName())
	}
}
