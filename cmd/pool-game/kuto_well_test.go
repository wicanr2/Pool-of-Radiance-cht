package main

import (
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 古托井的入口 0（`ecl8/29 99DAh`）每一步都先 `CALL C01Eh` 往前看一格、再把
// `C04B`／`C04C` 寫回原值，人沒動。這一步要由引擎走完：以前只要腳本叫過
// `C01Eh` 引擎就不再走（那是為了換區那一步不走兩次），於是那張圖一步都走不動。
// 現在要叫過而且座標真的變了才算。最小重現：站在 (15,4) 朝西按一下，人要在
// (14,4)，而且沒有東西在等。
func TestKutoWellLookAheadDoesNotEatTheStep(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	archive, ok := application.eclCatalog.Archive(8)
	if !ok {
		t.Fatal("ECL8 archive is absent")
	}
	session, err := gamepack.NewDOSECLArchiveSession(archive, 29, 0x9914)
	if err != nil {
		t.Fatal(err)
	}
	geoMap, ok := application.geometryCatalog.Map(gamepack.MapKey{Archive: 8, BlockID: 29})
	if !ok {
		t.Fatal("GEO8/29 is absent")
	}
	application.eventSession, application.eventMachine = session, session.Machine()
	application.eclArchive = 8
	application.initialMap = &geoMap
	application.spawn = gamepack.Spawn{Map: geoMap.Key, X: 15, Y: 4, Facing: 3}
	application.introDone, application.mode = true, modeAdventure
	if err := application.configureEventSession(session); err != nil {
		t.Fatal(err)
	}
	hero := poolsave.Character{Name: "HERO", RaceID: "human", GenderID: "male", ClassID: "fighter",
		AlignmentID: "lawful-good", Abilities: [6]int{16, 10, 10, 10, 10, 10}, MaxHP: 10, CurrentHP: 10,
		PortraitHead: 1, PortraitBody: 1, IconSize: 1}
	application.state.Party = []poolsave.Character{hero}
	application.saveState = func(poolsave.State) error { return nil }
	// 隨機遭遇（十分之一）會讓這一步停在遭遇選單上；固定 seed 讓它不出現。
	application.eclSeed = 1
	for step := 0; step < 3; step++ {
		before := application.spawn
		if err := press(application, ebiten.KeyArrowUp); err != nil {
			t.Fatal(err)
		}
		if application.encounter != nil || application.combatActive {
			t.Fatalf("a random encounter fired on step %d with eclSeed 1; the seed is part of the fixture", step)
		}
		if application.cellEventPending || application.cellWaitingMenu {
			t.Fatalf("step %d left something pending: text=%q status=%q", step,
				application.eventText, application.statusLine)
		}
		if application.spawn.X != before.X-1 || application.spawn.Y != before.Y {
			t.Fatalf("step %d: %+v → %+v, want one cell west", step, before, application.spawn)
		}
	}
}

// 古托井北緣每一格北面都是牆，而它的換圖表北向指到封存檔 10h（不存在的
// `ECL16`，spec 101 的表）。被牆擋住的那一步 remake 照樣跑入口 0（樓梯常朝著
// 牆，spec 101），於是 `NEWECL` 解到不存在的封存檔——以前是硬錯誤，探索器
// 把它記成硬失敗（`TestRandomWalkReachesKnownContentWithoutFailing` 種子 7）。
// 現在當成牆：人不動、沒有錯誤、下一步照走。
func TestKutoWellNorthWallIsJustAWall(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	archive, ok := application.eclCatalog.Archive(8)
	if !ok {
		t.Fatal("ECL8 archive is absent")
	}
	session, err := gamepack.NewDOSECLArchiveSession(archive, 29, 0x9914)
	if err != nil {
		t.Fatal(err)
	}
	geoMap, ok := application.geometryCatalog.Map(gamepack.MapKey{Archive: 8, BlockID: 29})
	if !ok {
		t.Fatal("GEO8/29 is absent")
	}
	application.eventSession, application.eventMachine = session, session.Machine()
	application.eclArchive = 8
	application.initialMap = &geoMap
	application.spawn = gamepack.Spawn{Map: geoMap.Key, X: 4, Y: 0, Facing: 0}
	application.introDone, application.mode = true, modeAdventure
	if err := application.configureEventSession(session); err != nil {
		t.Fatal(err)
	}
	hero := poolsave.Character{Name: "HERO", RaceID: "human", GenderID: "male", ClassID: "fighter",
		AlignmentID: "lawful-good", Abilities: [6]int{16, 10, 10, 10, 10, 10}, MaxHP: 10, CurrentHP: 10,
		PortraitHead: 1, PortraitBody: 1, IconSize: 1}
	application.state.Party = []poolsave.Character{hero}
	application.saveState = func(poolsave.State) error { return nil }
	application.eclSeed = 1
	if geoMap.Grid.CanMoveDungeonWrapped(4, 0, 0) {
		t.Fatal("(4,0) is expected to have a wall to the north")
	}
	if err := press(application, ebiten.KeyArrowUp); err != nil {
		t.Fatalf("walking into the north wall returned an error: %v", err)
	}
	if application.spawn.Map != geoMap.Key || application.spawn.X != 4 || application.spawn.Y != 0 {
		t.Fatalf("the party moved to %+v", application.spawn)
	}
	if application.eclArchive != 8 || application.eventSession.CurrentBlockID() != 29 {
		t.Fatalf("the session left ECL8/29: ECL%d/%d", application.eclArchive, application.eventSession.CurrentBlockID())
	}
	// 轉身往南走一步要走得動。
	if err := press(application, ebiten.KeyArrowRight); err != nil {
		t.Fatal(err)
	}
	if err := press(application, ebiten.KeyArrowRight); err != nil {
		t.Fatal(err)
	}
	if err := press(application, ebiten.KeyArrowUp); err != nil {
		t.Fatalf("the next step returned an error: %v", err)
	}
	if application.spawn.Y != 1 {
		t.Fatalf("after the wall the party could not walk south: %+v", application.spawn)
	}
}
