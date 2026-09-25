package main

import (
	"math/rand"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/assets"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// recordTileSets 把圖塊載入換成只記名字的假貨，用來看戰場挑了哪一組圖塊
// （overlay-10 `12EBh`）。
func recordTileSets(a *app) map[string]bool {
	loaded := map[string]bool{}
	a.combatTiles = nil
	a.loadCombatTiles = func(name string) ([]*ebiten.Image, error) {
		loaded[name] = true
		return nil, nil
	}
	return loaded
}

// assertOutdoorBattlefield 核對一張由 `1255h` 生成的戰場：模式是野外、
// 整面沒有室內那種未落筆的格子、平地的圖塊取自 WildCom。
func assertOutdoorBattlefield(t *testing.T, a *app) {
	t.Helper()
	if a.tactical == nil {
		t.Fatal("no tactical board")
	}
	if mode := a.combatAreaMode(); mode == gamepack.CombatAreaIndoor {
		t.Fatalf("combat area mode is indoor in wilderness block %d (@49E6=%d)",
			a.eventSession.CurrentBlockID(), a.eventMachine.Memory[encounterWalkFlagAddress])
	}
	grid := a.tactical.Grid
	if !grid.Outdoor {
		t.Fatal("the wilderness fight was generated as an indoor battlefield")
	}
	open, detail := 0, 0
	for index, class := range grid.Terrain {
		if class == combat.UnpaintedCellClass {
			t.Fatalf("cell %d is unpainted; the outdoor generator fills every cell", index)
		}
		if class == combat.OpenGroundCellClass {
			open++
		} else {
			detail++
		}
	}
	if open == 0 || detail == 0 {
		t.Fatalf("outdoor field has %d open and %d detail cells", open, detail)
	}
	if len(a.tactical.Roster) < 2 {
		t.Fatalf("deployment placed %d combatants", len(a.tactical.Roster))
	}
	loaded := recordTileSets(a)
	a.combatTerrainTile(combat.OpenGroundCellClass)
	if !loaded[assets.WildernessCombatTiles] || loaded[assets.DungeonCombatTiles] {
		t.Fatalf("open ground on the outdoor field loaded %v, want WildCom only", loaded)
	}
	t.Logf("野外戰場：區塊 %d 野外座標 (%d,%d) 模式 %d，%d 格平地、%d 格細節",
		a.eventSession.CurrentBlockID(), a.eventMachine.Memory[wildernessX],
		a.eventMachine.Memory[wildernessY], a.combatAreaMode(), open, detail)
}

// 野外遭遇：從東邊登陸點起，只用按鍵走到隨機遭遇、選 COMBAT，戰場要是
// `1255h` 的野外戰場。
func TestAWildernessEncounterFightsOnTheOutdoorBattlefield(t *testing.T) {
	a := sailEastIntoTheWilderness(t)
	random := rand.New(rand.NewSource(59))
	turns := []ebiten.Key{ebiten.KeyArrowUp, ebiten.KeyArrowUp, ebiten.KeyArrowUp, ebiten.KeyArrowLeft, ebiten.KeyArrowRight}
	sawEncounter := false
	for step := 0; step < 6000 && a.tactical == nil; step++ {
		var err error
		switch {
		case a.encounter != nil:
			sawEncounter = true
			if a.cellMenuOptions[a.cellMenuCursor] != "COMBAT" {
				err = press(a, ebiten.KeyArrowRight)
			} else {
				err = press(a, ebiten.KeyEnter)
			}
		case a.combatActive:
			err = press(a, ebiten.KeyEnter)
		case a.cellWaitingMenu && len(a.cellMenuOptions) > 1 && a.cellMenuCursor != len(a.cellMenuOptions)-1:
			// 地點問句的最後一項是拒絕（不上船、不進城），留在野外。
			err = press(a, ebiten.KeyArrowRight)
		case a.cellEventPending || a.cellWaitingMenu:
			err = press(a, ebiten.KeyEnter)
		default:
			if !a.inWildernessOverland() {
				t.Fatalf("left the wilderness overland: block %d 4A9E=%d",
					a.eventSession.CurrentBlockID(), a.eventMachine.Memory[wildernessArea])
			}
			err = press(a, turns[random.Intn(len(turns))])
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	if a.tactical == nil {
		t.Fatalf("6000 key presses in the wilderness never started a fight; at (%d,%d)",
			a.eventMachine.Memory[wildernessX], a.eventMachine.Memory[wildernessY])
	}
	// 真的遭遇是「遭遇選單 → COMBAT → 開打訊息按 ENTER」，combatActive 會立著；
	// F5 的預覽不會。
	if !sawEncounter || !a.combatActive {
		t.Fatalf("the fight did not come from the encounter menu (menu seen=%t, combatActive=%t)", sawEncounter, a.combatActive)
	}
	assertOutdoorBattlefield(t, a)
}

// F5 的戰術預覽走同一條生成路徑：站在野外地形上是野外戰場。
func TestTacticalPreviewInTheWildernessIsOutdoor(t *testing.T) {
	a := sailEastIntoTheWilderness(t)
	if !a.inWildernessOverland() {
		t.Fatalf("not on the wilderness overland: 4A9E=%d", a.eventMachine.Memory[wildernessArea])
	}
	if err := press(a, ebiten.KeyF5); err != nil {
		t.Fatal(err)
	}
	if !a.tacticalPreview {
		t.Fatalf("F5 did not open the tactical preview: %q", a.statusLine)
	}
	assertOutdoorBattlefield(t, a)
}

// 室內回歸：城裡按 F5，戰場仍是 `0820h` 由地城牆面生成的那一張，
// 逐格等於直接呼叫室內生成器的結果，圖塊取自 DungCom。
func TestTacticalPreviewInTheCityStaysIndoor(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	a := bootCityParty(t, zipPath)
	if mode := a.combatAreaMode(); mode != gamepack.CombatAreaIndoor {
		t.Fatalf("city block %d is mode %d", a.eventSession.CurrentBlockID(), mode)
	}
	if err := press(a, ebiten.KeyF5); err != nil {
		t.Fatal(err)
	}
	if !a.tacticalPreview || a.tactical == nil {
		t.Fatalf("F5 did not open the tactical preview: %q", a.statusLine)
	}
	if a.tactical.Grid.Outdoor {
		t.Fatal("the city battlefield was generated outdoors")
	}
	want, err := combat.GenerateIndoorTacticalGrid(int(a.spawn.X), int(a.spawn.Y),
		geoWallProbe(a.initialMap.Grid, int(a.spawn.Y)))
	if err != nil {
		t.Fatal(err)
	}
	for index := range want.Terrain {
		if a.tactical.Grid.Terrain[index] != want.Terrain[index] {
			t.Fatalf("cell %d = %02Xh, indoor generator gives %02Xh", index,
				a.tactical.Grid.Terrain[index], want.Terrain[index])
		}
	}
	loaded := recordTileSets(a)
	a.combatTerrainTile(combat.OpenGroundCellClass)
	if !loaded[assets.DungeonCombatTiles] || loaded[assets.WildernessCombatTiles] {
		t.Fatalf("open ground on the indoor field loaded %v, want DungCom only", loaded)
	}
}
