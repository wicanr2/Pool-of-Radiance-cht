package main

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/music"
)

type recordingSink struct{ calls []string }

func (s *recordingSink) Play(key int) { s.calls = append(s.calls, fmt.Sprintf("play %d", key)) }
func (s *recordingSink) Stop(key int) { s.calls = append(s.calls, fmt.Sprintf("stop %d", key)) }
func (s *recordingSink) Close() error { return nil }

// 從 Update() 走：城區（ECL 區塊 0，第 2 首）買東線船票、上船，到荒野
// （區塊 27，第 13 首）。換曲那一刻先停，800 毫秒後才放新的一首（spec 169）。
func TestSailingIntoTheWildernessSwitchesThePC98Song(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application := bootCityParty(t, zipPath)
	if application.spawn.Map.BlockID != 0 || application.eventSession.CurrentBlockID() != 0 {
		t.Fatalf("開場沒有停在城區：GEO%d/%d ECL block %d", application.spawn.Map.Archive,
			application.spawn.Map.BlockID, application.eventSession.CurrentBlockID())
	}
	sink := &recordingSink{}
	clock := time.Unix(0, 0)
	application.musicPlayer = music.NewPlayerWithSink(sink, music.SourcePC98, func() time.Time { return clock })
	idle := func() {
		application.keys = scriptedKeys{}
		if err := application.Update(); err != nil {
			t.Fatal(err)
		}
	}
	idle()
	clock = clock.Add(music.PC98SwitchSilence)
	idle()
	if got := application.musicPlayer.Sounding(); got != 2 {
		t.Fatalf("城區出聲第 %d 首，`$63CC` 是第 2 首", got)
	}

	application.eventMachine.Memory[0x4AA7] = 254
	application.eventMachine.Memory[0x4A01] = 255
	application.spawn.X, application.spawn.Y, application.spawn.Facing = 11, 2, 0
	if err := press(application, ebiten.KeyArrowUp); err != nil {
		t.Fatal(err)
	}
	for tick := 0; tick < 200 && !application.cellWaitingMenu; tick++ {
		if err := press(application, ebiten.KeyEnter); err != nil {
			t.Fatal(err)
		}
	}
	if err := press(application, ebiten.KeyArrowRight); err != nil {
		t.Fatal(err)
	}
	if application.cellMenuOptions[application.cellMenuCursor] != "EAST" {
		t.Fatalf("游標停在 %q", application.cellMenuOptions[application.cellMenuCursor])
	}
	for tick := 0; tick < 400 && application.cellEventPending; tick++ {
		if err := press(application, ebiten.KeyEnter); err != nil {
			t.Fatal(err)
		}
	}
	// 港務長那一段都還在區塊 0：同一首不重播。
	if fmt.Sprint(sink.calls) != "[play 2]" {
		t.Fatalf("還在城區就有別的呼叫：%v", sink.calls)
	}
	application.spawn.X, application.spawn.Y, application.spawn.Facing = 14, 1, 1
	if err := press(application, ebiten.KeyArrowUp); err != nil {
		t.Fatal(err)
	}
	for tick := 0; tick < 600 && application.eventSession.CurrentBlockID() == 0; tick++ {
		if err := press(application, ebiten.KeyEnter); err != nil {
			t.Fatal(err)
		}
	}
	if got := application.eventSession.CurrentBlockID(); got != 27 {
		t.Fatalf("上船之後停在 ECL block %d，要 27", got)
	}
	idle()
	if application.musicPlayer.Current() != 13 || application.musicPlayer.Sounding() != 0 {
		t.Fatalf("進荒野那一刻派 %d、出聲 %d；要派 13、先靜音",
			application.musicPlayer.Current(), application.musicPlayer.Sounding())
	}
	clock = clock.Add(music.PC98SwitchSilence)
	idle()
	if got := application.musicPlayer.Sounding(); got != 13 {
		t.Fatalf("荒野出聲第 %d 首，`$63CC` 是第 13 首", got)
	}
	if fmt.Sprint(sink.calls) != "[play 2 stop 2 play 13]" {
		t.Fatalf("呼叫順序 %v", sink.calls)
	}

	// Ctrl+O 關掉：停止；再按一次，在地圖上重派區域曲。
	application.keys = scriptedKeys{ebiten.KeyControl: true, ebiten.KeyO: true}
	if err := application.Update(); err != nil {
		t.Fatal(err)
	}
	if application.musicPlayer.Enabled() || application.musicPlayer.Sounding() != 0 {
		t.Fatal("Ctrl+O 沒有把音樂關掉")
	}
	application.keys = scriptedKeys{ebiten.KeyControl: true, ebiten.KeyO: true}
	if err := application.Update(); err != nil {
		t.Fatal(err)
	}
	clock = clock.Add(music.PC98SwitchSilence)
	idle()
	if got := application.musicPlayer.Sounding(); got != 13 {
		t.Fatalf("重新打開之後出聲 %d，要 13", got)
	}
}

// 畫面狀態怎麼對到派曲規則：標題、開始選單（區塊 0）、戰術盤面、最後一戰、商店、神殿、結局。
func TestMusicSceneReadsTheScreen(t *testing.T) {
	a := &app{mode: modeTitle}
	if scene := a.musicScene(); !scene.Title || scene.Block != music.PC98StartBlock {
		t.Fatalf("標題：%+v", scene)
	}
	a.mode = modeMenu
	if scene := a.musicScene(); scene.Title || scene.Block != 0 {
		t.Fatalf("開始選單：%+v（開機初始化把區塊設成 0）", scene)
	}
	a.mode = modeAdventure
	a.tactical, a.tacticalPreview = &tacticalState{}, true
	a.lastLoadedMonster = music.PC98BossMonsterID
	if scene := a.musicScene(); !scene.Combat || !scene.BossCombat {
		t.Fatalf("最後一戰：%+v", scene)
	}
	a.lastLoadedMonster = 13
	if scene := a.musicScene(); !scene.Combat || scene.BossCombat {
		t.Fatalf("一般戰鬥：%+v", scene)
	}
	a.tactical, a.tacticalPreview = nil, false
	a.shopActive = true
	if scene := a.musicScene(); !scene.Shop {
		t.Fatalf("商店：%+v", scene)
	}
	a.shopActive, a.templeActive = false, true
	if scene := a.musicScene(); !scene.Temple {
		t.Fatalf("神殿：%+v", scene)
	}
	a.templeActive, a.endingActive = false, true
	if scene := a.musicScene(); !scene.Ending {
		t.Fatalf("結局：%+v", scene)
	}
}
