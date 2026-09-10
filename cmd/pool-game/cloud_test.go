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
	if !state.placeCloud(0, 5, 5, 3) {
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
	state.placeCloud(0, 5, 5, 3)
	if state.stinkingCloudTurn(1) {
		t.Error("腳下是雲但身上沒有效果，不該收掉回合")
	}
	if state.ArmorClass[1] != 0x3C {
		t.Errorf("沒結算卻把 AC 動成 %#x", state.ArmorClass[1])
	}

	// 兩個都有：回合沒了，AC 一次差 2 點，壓到 32h 為止。
	state = cloudFixture()
	state.placeCloud(0, 5, 5, 3)
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

// 雲的壽命就是施法者身上那個 `28h` 節點的持續：回合邊界減到 0、摘節點時走
// 收尾，那一下才收雲（spec 121）。**沒有人每回合輪詢盤面上的雲。**
func TestCloudDispersesWhenItsEffectNodeExpires(t *testing.T) {
	state := cloudFixture()
	if !state.placeCloud(1, 5, 5, 2) {
		t.Fatal("放不下這一團雲")
	}

	at, ok := state.Effects[1].IndexOf(gamepack.CloudObjectEffectCode)
	if !ok {
		t.Fatal("施法者身上沒有雲物件節點")
	}
	node := state.Effects[1][at]
	if got := node.Duration(); got != 2 {
		t.Errorf("持續是 %d，預期 2（＝施法者等級）", got)
	}
	if !node.NeedsTeardown() {
		t.Error("`+4` 沒立起來，到期就不會走收尾")
	}
	if got := node.CloudIndex(); got != 1 {
		t.Errorf("雲序號讀成 %d，預期 1", got)
	}
	if got := node.CasterLevel(); got != 2 {
		t.Errorf("等級讀成 %d，預期 2", got)
	}

	// 第一個回合邊界：持續剩 1，雲還在盤上。
	state.tickEffects(1)
	if len(state.Clouds) != 1 {
		t.Fatalf("才過一個回合就散了：串列上有 %d 團", len(state.Clouds))
	}
	if got, _ := state.Grid.TerrainAt(5, 5); got != gamepack.CloudTerrain {
		t.Errorf("雲心的地形是 %#x，預期還是雲", got)
	}

	// 第二個回合邊界：節點歸零摘掉，收尾把雲收走，四格還原成原本的地形。
	state.tickEffects(1)
	if len(state.Clouds) != 0 {
		t.Fatalf("節點到期了雲還在：串列上有 %d 團", len(state.Clouds))
	}
	for _, cell := range [gamepack.CloudCells][2]int{{5, 5}, {6, 5}, {6, 6}, {5, 6}} {
		got, err := state.Grid.TerrainAt(cell[0], cell[1])
		if err != nil {
			t.Fatal(err)
		}
		if got != 1 {
			t.Errorf("(%d,%d) 收雲後是 %#x，預期還原成 1", cell[0], cell[1], got)
		}
	}
	if state.Effects[1].Has(gamepack.CloudObjectEffectCode) {
		t.Error("節點到期了卻還掛在身上")
	}
}

// 反對照：收尾只認雲物件那個代碼，也只在 `+4` 立著時跑。別的效果到期不該
// 順手把雲收掉——`addEffect` 掛的節點 `+4` 是 0。
func TestExpiringAnotherEffectLeavesTheCloudAlone(t *testing.T) {
	state := cloudFixture()
	if !state.placeCloud(1, 5, 5, 5) {
		t.Fatal("放不下這一團雲")
	}
	state.addEffect(1, gamepack.StinkingCloudEffectCode, 1, 1)

	state.tickEffects(1)
	if state.Effects[1].Has(gamepack.StinkingCloudEffectCode) {
		t.Error("持續 1 的效果過了一個回合邊界還在")
	}
	if len(state.Clouds) != 1 {
		t.Fatalf("別的效果到期把雲收掉了：串列上有 %d 團", len(state.Clouds))
	}
	if got, _ := state.Grid.TerrainAt(5, 5); got != gamepack.CloudTerrain {
		t.Errorf("雲心的地形是 %#x，預期還是雲", got)
	}
}

// 找不到那一團就什麼都不做，與原版 `0D4Ah` 的「找不到 → 直接結束」相同。
func TestDisperseCloudIgnoresACloudThatIsNotThere(t *testing.T) {
	state := cloudFixture()
	if !state.placeCloud(1, 5, 5, 3) {
		t.Fatal("放不下這一團雲")
	}
	if state.disperseCloud(1, 7) {
		t.Error("收了一團序號不存在的雲")
	}
	if state.disperseCloud(0, 1) {
		t.Error("收了別人的雲")
	}
	if len(state.Clouds) != 1 {
		t.Fatalf("串列上有 %d 團雲，預期原封不動", len(state.Clouds))
	}
}
