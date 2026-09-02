package main

import (
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// 只用正常按鍵，從標題一路走到「隊伍在地圖上動了一步」。這條路徑上任何一段
// 需要測試自己塞狀態才走得通，就表示玩家也走不通——所以這裡不碰
// application.state、eventSession 或 spawn，全部靠 press。
func TestNormalKeysReachTheFirstDungeonStep(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	text := &scriptedTextKeys{scriptedKeys: scriptedKeys{}}
	application.keys = text
	step := func(what string, key ebiten.Key, chars ...rune) {
		t.Helper()
		text.scriptedKeys[key] = true
		text.chars = chars
		if err := application.Update(); err != nil {
			t.Fatalf("%s: %v", what, err)
		}
	}
	idle := func() {
		t.Helper()
		text.chars = nil
		if err := application.Update(); err != nil {
			t.Fatalf("idle: %v", err)
		}
	}

	step("標題", ebiten.KeyEnter)
	if application.mode != modeMenu {
		t.Fatalf("title did not reach the menu, mode=%d", application.mode)
	}
	step("開始建角", ebiten.KeyC)
	for _, what := range []string{"種族", "性別", "職業", "陣營"} {
		step(what, ebiten.KeyEnter)
	}
	idle() // 骰值在第一次更新時產生，與真正的畫格路徑相同
	step("接受骰值", ebiten.KeyEnter)
	step("輸入姓名", ebiten.KeyEnter, 'H', 'E', 'R', 'O')
	step("保留肖像", ebiten.KeyK)
	step("確認造形", ebiten.KeyEnter)
	step("造形 OK", ebiten.KeyY)
	if len(application.state.CharacterLibrary) != 1 {
		t.Fatalf("character library holds %d after creation", len(application.state.CharacterLibrary))
	}
	step("加入隊伍", ebiten.KeyA)
	if len(application.state.Party) != 1 {
		t.Fatalf("party holds %d after A", len(application.state.Party))
	}
	step("開始冒險", ebiten.KeyB)
	if application.mode != modeAdventure {
		t.Fatalf("B did not enter the adventure, mode=%d status=%q", application.mode, application.statusLine)
	}

	// 開場與 34 步導覽是原版的自動流程，玩家只能按 ENTER 推進。
	for tick := 0; tick < 20000 && !application.introDone; tick++ {
		if application.introWaiting || application.tourPage >= 0 {
			step("推進開場", ebiten.KeyEnter)
			continue
		}
		idle()
	}
	if !application.introDone {
		t.Fatalf("the opening never finished: waiting=%v tour=%v step=%d status=%q",
			application.introWaiting, application.tourActive, application.tourStep, application.statusLine)
	}

	before := application.spawn
	moved := false
	// 轉向是 45 度一格，但前進只認四個正方向，所以每次要按兩下才換到下一個
	// 可走的朝向。四個方向都試過還沒動，才算真的走不了。
	for attempt := 0; attempt < 4 && !moved; attempt++ {
		facing := application.spawn.Facing
		step("前進", ebiten.KeyArrowUp)
		t.Logf("朝向 %d 前進後 (%d,%d) 事件=%v 戰鬥=%v 狀態=%q",
			facing, application.spawn.X, application.spawn.Y,
			application.cellEventPending, application.combatActive, application.statusLine)
		if application.spawn.X != before.X || application.spawn.Y != before.Y {
			moved = true
			break
		}
		step("右轉", ebiten.KeyArrowRight)
		step("右轉", ebiten.KeyArrowRight)
	}
	if !moved {
		t.Fatalf("the party never moved from (%d,%d) facing %d: 事件=%v 狀態=%q",
			before.X, before.Y, application.spawn.Facing, application.cellEventPending, application.statusLine)
	}
	t.Log(fmt.Sprintf("走到 %+v，狀態列：%s", application.spawn, application.statusLine))

	if err := press(application, ebiten.KeyF10); !errors.Is(err, ebiten.Termination) {
		t.Fatalf("F10 save=%v", err)
	}
}
