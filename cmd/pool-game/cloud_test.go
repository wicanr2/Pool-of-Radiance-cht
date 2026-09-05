package main

import (
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// cloudFixture 是一張空地形的小盤，第 1 格站一個佔一格的人。
func cloudFixture() *tacticalState {
	state := &tacticalState{
		Grid: combat.TacticalGrid{
			Terrain: make([]uint8, combat.TacticalRowStride*combat.TacticalRowStride),
		},
		Roster:     []combat.CombatantCell{{}, {X: 5, Y: 5, FootprintClass: 1}},
		Effects:    make([]gamepack.EffectList, 2),
		ArmorClass: []int{0, 0x3C},
	}
	for index := range state.Grid.Terrain {
		state.Grid.Terrain[index] = 1
	}
	return state
}

// 放一團雲下去：四格蓋成地形 `1Eh`，站在裡面的人就算站在雲裡。
func TestPlacingACloudStampsTheTwoByTwoBlock(t *testing.T) {
	state := cloudFixture()
	if state.standingInCloud(1) {
		t.Fatal("還沒放雲就說站在雲裡")
	}
	if !state.placeCloud(0, 5, 5) {
		t.Fatal("放不下這一團雲")
	}
	if len(state.Clouds) != 1 {
		t.Fatalf("串列上有 %d 團雲，預期 1", len(state.Clouds))
	}
	// 雲心、東、東南、南——四格都是 `1Eh`，其餘不動。
	for _, cell := range [gamepack.CloudCells][2]int{{5, 5}, {6, 5}, {6, 6}, {5, 6}} {
		got, err := state.Grid.TerrainAt(cell[0], cell[1])
		if err != nil {
			t.Fatalf("(%d,%d) 取不到地形：%v", cell[0], cell[1], err)
		}
		if got != gamepack.CloudTerrain {
			t.Errorf("(%d,%d) 的地形是 %#x，預期 %#x", cell[0], cell[1], got, gamepack.CloudTerrain)
		}
	}
	if got, _ := state.Grid.TerrainAt(4, 5); got != 1 {
		t.Errorf("雲西邊那一格被蓋成 %#x，不該動", got)
	}
	if !state.standingInCloud(1) {
		t.Error("人站在雲心卻不算站在雲裡")
	}
	// 收掉之後地形要還原，而且人不再算在雲裡。
	list, err := state.Clouds.RemoveAt(0, tacticalBoard{state})
	if err != nil {
		t.Fatalf("收雲失敗：%v", err)
	}
	state.Clouds = list
	if state.standingInCloud(1) {
		t.Error("雲收掉了還說站在雲裡")
	}
	if got, _ := state.Grid.TerrainAt(5, 5); got != 1 {
		t.Errorf("收雲之後 (5,5) 是 %#x，預期還原成 1", got)
	}
}

// 結算要**兩個條件都成立**：身上有 `1Eh`，而且腳下還是雲。
// 只有其中一個的時候什麼都不做——AC 也不能動。
func TestStinkingCloudTurnNeedsBothTheEffectAndTheTerrain(t *testing.T) {
	// 只有效果，沒有雲。
	state := cloudFixture()
	state.addEffect(1, gamepack.StinkingCloudEffectCode, 0, 1)
	if state.stinkingCloudTurn(1) {
		t.Error("身上有效果但腳下不是雲，不該收掉回合")
	}
	if state.ArmorClass[1] != 0x3C {
		t.Errorf("沒結算卻把 AC 動成 %#x", state.ArmorClass[1])
	}

	// 只有雲，身上沒有效果——走進別人放的雲不會自己中招，
	// 因為「誰在人走進來時掛上 `1Eh`」原版哪一段做的還沒讀（spec 121）。
	state = cloudFixture()
	state.placeCloud(0, 5, 5)
	if state.stinkingCloudTurn(1) {
		t.Error("腳下是雲但身上沒有效果，不該收掉回合")
	}
	if state.ArmorClass[1] != 0x3C {
		t.Errorf("沒結算卻把 AC 動成 %#x", state.ArmorClass[1])
	}

	// 兩個都有：回合沒了，AC 一次差 2 點，壓到 32h 為止。
	state = cloudFixture()
	state.placeCloud(0, 5, 5)
	state.addEffect(1, gamepack.StinkingCloudEffectCode, 0, 1)
	for round, want := range []int{0x3A, 0x38, 0x36, 0x34, 0x32, 0x32} {
		if !state.stinkingCloudTurn(1) {
			t.Fatalf("第 %d 次結算沒有收掉回合", round+1)
		}
		if state.ArmorClass[1] != want {
			t.Errorf("第 %d 次結算後內部 AC 是 %#x，預期 %#x",
				round+1, state.ArmorClass[1], want)
		}
	}
}
