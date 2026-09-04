package main

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 存檔／讀檔在**戰鬥中**與**戰鬥後**的抽樣。
//
// 戰鬥中存不了檔：原版的 SAVE 在營地，戰鬥畫面沒有那個入口；而這裡的
// Campaign 只存 ECL session 與座標，`tactical` 與 `combatMonsters` 都不在
// 裡面——存了讀回來怪物會整批消失。這一則同時釘住「按了 F10 也不會把視窗
// 收掉」：`Update` 回傳非 Termination 的 error 時 ebiten 會直接結束。
func TestSavingIsRefusedDuringCombatAndWorksAfterIt(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	character := poolsave.Character{Name: "HERO", RaceID: "dwarf", GenderID: "male", ClassID: "fighter",
		AlignmentID: "lawful-good", Abilities: [6]int{16, 10, 10, 13, 10, 10}, MaxHP: 8, CurrentHP: 8,
		PortraitHead: 1, PortraitBody: 1, IconSize: 1}
	application.state = poolsave.State{Schema: poolsave.Schema,
		CharacterLibrary: []poolsave.Character{character}, Party: []poolsave.Character{character}}
	saved := poolsave.State{}
	application.saveState = func(state poolsave.State) error { saved = cloneSaveState(state); return nil }
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
		t.Fatal("the opening never finished")
	}

	reached := walkThisAreaUntil(t, application, 60000, func() bool { return application.combatActive })
	if !reached {
		t.Skipf("這一趟走完整區都沒打到架，最後在 %+v", application.spawn)
	}

	// 剛進戰鬥時遭遇文字還開著，那時擋下 F10 的是對話閘。要驗戰鬥閘得先按
	// ENTER 把隊伍送進戰術地圖（`enterCombatStaging` 留了那一步）。
	for tick := 0; tick < 512 && application.tactical == nil; tick++ {
		if err := press(application, ebiten.KeyEnter); err != nil {
			t.Fatalf("進戰術地圖時第 %d tick：%v", tick, err)
		}
		if !application.combatActive {
			t.Skip("這一場在進到戰術地圖之前就結束了")
		}
	}
	if application.tactical == nil {
		t.Skip("這一場沒有進到戰術地圖")
	}
	// 戰鬥中：F10 應該被擋下來，而且不能結束程式。
	saved = poolsave.State{}
	if err := press(application, ebiten.KeyF10); err != nil {
		t.Fatalf("戰鬥中按 F10 回傳 %v，應該只是拒絕存檔", err)
	}
	if saved.Campaign != nil {
		t.Error("戰鬥中存了檔；tactical 與 combatMonsters 都不在 Campaign 裡，讀回來怪物會消失")
	}
	if !strings.Contains(application.statusLine, "battle") {
		t.Errorf("戰鬥中拒絕存檔的理由是 %q，應該是戰鬥那一條", application.statusLine)
	}
	t.Logf("戰鬥中按 F10：%s", application.statusLine)

	// 打完這一場。隊伍不還手，讓怪物自己解決；打不完就跳過，這一則要驗的
	// 是存檔而不是戰鬥平衡。
	for tick := 0; tick < 60000 && application.combatActive; tick++ {
		if err := press(application, ebiten.KeyEnter); err != nil {
			t.Fatalf("打完這一場時第 %d tick：%v", tick, err)
		}
	}
	if application.combatActive {
		t.Skip("這一場在預算內沒打完")
	}
	for tick := 0; tick < 512 && (application.cellEventPending || application.cellWaitingMenu); tick++ {
		if err := press(application, ebiten.KeyEnter); err != nil {
			t.Fatalf("讀完戰後文字時：%v", err)
		}
	}

	// 戰鬥後：存得起來，而且讀回來還走得動。
	if err := press(application, ebiten.KeyF10); !errors.Is(err, ebiten.Termination) {
		t.Fatalf("戰鬥後按 F10 回傳 %v，應該存檔後結束", err)
	}
	if saved.Campaign == nil {
		t.Fatal("戰鬥後 F10 沒有存下 campaign")
	}
	restored, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	restored.mode = modeMenu
	restored.loadState = func() (poolsave.State, error) { return cloneSaveState(saved), nil }
	if err := press(restored, ebiten.KeyL); err != nil {
		t.Fatal(err)
	}
	if restored.mode != modeAdventure || restored.spawn != application.spawn {
		t.Fatalf("讀回來 mode=%d spawn=%+v，原本是 %+v", restored.mode, restored.spawn, application.spawn)
	}
	if restored.combatActive || restored.tactical != nil {
		t.Error("讀回來還帶著戰鬥狀態；存檔不該保存戰鬥")
	}
	// 「欄位全對但玩家按下一步就卡住」是這一類存檔缺陷的樣子，所以要真的走一步。
	before := restored.spawn
	moved := false
	for tick := 0; tick < 64 && !moved; tick++ {
		if err := press(restored, ebiten.KeyArrowUp); err != nil {
			t.Fatalf("讀回來之後走一步：%v", err)
		}
		if restored.spawn != before || restored.cellEventPending || restored.combatActive {
			moved = true
		}
		if err := press(restored, ebiten.KeyArrowRight); err != nil {
			t.Fatalf("讀回來之後轉向：%v", err)
		}
	}
	if !moved {
		t.Errorf("讀回來之後動不了：%+v 狀態列 %q", restored.spawn, restored.statusLine)
	}
	t.Logf("戰鬥後存讀檔通過：存在 GEO%d/%d (%d,%d)",
		application.spawn.Map.Archive, application.spawn.Map.BlockID,
		application.spawn.X, application.spawn.Y)
}
