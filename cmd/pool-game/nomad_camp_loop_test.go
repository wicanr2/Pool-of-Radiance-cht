package main

// ECL7/17（遊牧營地）的格子事件在探索器裡無限迴圈（#23）。
//
// 探索器的 log：換到 `GEO7/17` 之後連續七趟「格子事件 300000」，位置停在 (7,15) 一步不走。
// 這一條是最小重現：直接開那個區塊、站到那一格、按 ENTER，斷言事件會結束。

import (
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

func TestNomadCampCellEventEnds(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	archive, ok := application.eclCatalog.Archive(7)
	if !ok {
		t.Fatal("ECL7 archive is absent")
	}
	session, err := gamepack.NewDOSECLArchiveSession(archive, 17, 0x9914)
	if err != nil {
		t.Fatal(err)
	}
	hero := poolsave.Character{
		Name: "HERO", RaceID: "dwarf", GenderID: "male", ClassID: "fighter",
		AlignmentID: "lawful-good", MaxHP: 20, CurrentHP: 20,
		PortraitHead: 1, PortraitBody: 1, IconSize: 1,
	}
	application.mode, application.introDone = modeAdventure, true
	application.eventSession, application.eventMachine = session, session.Machine()
	application.eclArchive = 7
	if err := application.configureEventSession(session); err != nil {
		t.Fatal(err)
	}
	application.state.Party = []poolsave.Character{hero}
	application.state.CharacterLibrary = []poolsave.Character{hero}
	geometryMap, ok := application.geometryCatalog.MapByBlock(17)
	if !ok {
		t.Fatal("GEO block 17 is absent")
	}
	application.initialMap = &geometryMap
	application.spawn = gamepack.Spawn{Map: geometryMap.Key, X: 7, Y: 15, Facing: 0}

	result, err := gamepack.RunInitialSessionCellEntry(session, geometryMap.Grid, application.spawn)
	if err != nil {
		t.Fatalf("跑入口 0：%v", err)
	}
	if err := application.consumeInitialSearch(result); err != nil {
		t.Fatalf("跑入口 0：%v", err)
	}
	// 事件要在合理次數內結束。**上限給寬**：中間可能有好幾頁文字與選單，
	// 卡住的症狀是「按幾百次還在同一個畫面」，不是「多按了幾次」。
	screens := map[string]int{}
	for press := 0; press < 300; press++ {
		if !application.cellEventPending && !application.cellWaitingMenu {
			t.Logf("按了 %d 次結束；走過的畫面：%v", press, screens)
			return
		}
		screens[application.screenName()]++
		if err := pressKey(application, ebiten.KeyEnter); err != nil {
			t.Fatal(err)
		}
	}
	t.Fatalf("按了 300 次事件還沒結束：畫面=%s 選項=%v 文字=%q 走過的畫面次數=%v",
		application.screenName(), application.cellMenuOptions, application.eventText, screens)
}

// pressKey 與 press 相同，只是名字不撞既有治具。
func pressKey(application *app, key ebiten.Key) error {
	return press(application, key)
}
