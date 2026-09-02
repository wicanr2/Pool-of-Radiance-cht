package combat

import (
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

func newTestGrid(ignoreTerrain bool) TacticalGrid {
	return TacticalGrid{
		IgnoreTerrain: ignoreTerrain,
		Terrain:       make([]uint8, TacticalRowStride*(TacticalMaxY+1)),
	}
}

// 測試用的類別表：碼 0 可通行，碼 1 擋住。真正的 66 筆由
// gamepack.ParseCombatCellClassTable 依原始 START.EXE bytes 解出。
func testCellClasses() CellClasses {
	var classes CellClasses
	classes[0] = gamepack.CombatCellClass{EntryThreshold: 1}
	classes[1] = gamepack.CombatCellClass{EntryThreshold: 0xFF, PathByte2: 2}
	return classes
}

// 主軸上的直走每步 2，副軸同時前進的斜走每步 3。
func TestStepWalkerStraightLineCosts(t *testing.T) {
	walker := NewStepWalker(0, 0, 3, 0)
	steps := 0
	for walker.Step() {
		steps++
		if walker.Direction != 2 {
			t.Fatalf("step %d direction %d, want 2", steps, walker.Direction)
		}
	}
	if steps != 3 {
		t.Fatalf("took %d steps, want 3", steps)
	}
	if walker.X != 3 || walker.Y != 0 {
		t.Fatalf("ended at (%d,%d), want (3,0)", walker.X, walker.Y)
	}
	if walker.Cost != 3*StraightStepCost {
		t.Fatalf("cost %d, want %d", walker.Cost, 3*StraightStepCost)
	}
}

func TestStepWalkerDiagonalCosts(t *testing.T) {
	walker := NewStepWalker(0, 0, 3, 3)
	steps := 0
	for walker.Step() {
		steps++
		if walker.Direction != 3 {
			t.Fatalf("step %d direction %d, want 3", steps, walker.Direction)
		}
	}
	if steps != 3 || walker.X != 3 || walker.Y != 3 {
		t.Fatalf("%d steps ending at (%d,%d)", steps, walker.X, walker.Y)
	}
	if walker.Cost != 3*DiagonalStepCost {
		t.Fatalf("cost %d, want %d", walker.Cost, 3*DiagonalStepCost)
	}
}

// 主軸較長時直走與斜走交錯，逐步核對位置與累積成本。
func TestStepWalkerMixedLineMatchesOriginalSequence(t *testing.T) {
	walker := NewStepWalker(0, 0, 4, 2)
	want := []struct {
		x, y int
		cost uint8
	}{
		{1, 1, 3},
		{2, 1, 5},
		{3, 2, 8},
		{4, 2, 10},
	}
	for index, expected := range want {
		if !walker.Step() {
			t.Fatalf("step %d did not move", index+1)
		}
		if walker.X != expected.x || walker.Y != expected.y || walker.Cost != expected.cost {
			t.Fatalf("step %d at (%d,%d) cost %d, want (%d,%d) cost %d",
				index+1, walker.X, walker.Y, walker.Cost, expected.x, expected.y, expected.cost)
		}
	}
	if walker.Step() {
		t.Fatal("walker moved past its goal")
	}
}

// 已經在終點時原版不動也不加成本，方向欄位歸為「無方向」。
func TestStepWalkerAtGoalDoesNotMove(t *testing.T) {
	walker := NewStepWalker(7, 7, 7, 7)
	if walker.Step() {
		t.Fatal("a zero-length walk reported a step")
	}
	if walker.Cost != 0 {
		t.Fatalf("cost %d, want 0", walker.Cost)
	}
	if walker.Direction != DirectionAny {
		t.Fatalf("direction %d, want %d", walker.Direction, DirectionAny)
	}
}

func TestTraceMovementCompletesWithinBudget(t *testing.T) {
	result, err := TraceMovement(newTestGrid(false), testCellClasses(), 0, 0, 4, 2, 5)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Complete {
		t.Fatalf("trace stopped at (%d,%d) cost %d", result.X, result.Y, result.Cost)
	}
	if result.X != 4 || result.Y != 2 || result.Cost != 10 {
		t.Fatalf("ended at (%d,%d) cost %d", result.X, result.Y, result.Cost)
	}
}

// 預算上限是 budget*2+1；成本 10 需要 budget 5，budget 4 會在半路停下。
func TestTraceMovementStopsWhenBudgetRunsOut(t *testing.T) {
	result, err := TraceMovement(newTestGrid(false), testCellClasses(), 0, 0, 4, 2, 4)
	if err != nil {
		t.Fatal(err)
	}
	if result.Complete {
		t.Fatal("trace completed although the budget was short")
	}
	if int(result.Cost) <= 4*2+1 {
		t.Fatalf("stopped at cost %d, which is still inside the budget", result.Cost)
	}
}

func TestTraceMovementStopsOnBlockingTerrain(t *testing.T) {
	grid := newTestGrid(false)
	grid.Terrain[1*TacticalRowStride+2] = 1
	result, err := TraceMovement(grid, testCellClasses(), 0, 0, 4, 2, 20)
	if err != nil {
		t.Fatal(err)
	}
	if result.Complete {
		t.Fatal("trace crossed a blocking cell")
	}
	if result.X != 2 || result.Y != 1 {
		t.Fatalf("stopped at (%d,%d), want (2,1)", result.X, result.Y)
	}
}

// 地圖 record 的 +6 非 0 時原版整段跳過地形判定，只剩預算限制。
func TestTraceMovementIgnoresTerrainWhenTheMapSaysSo(t *testing.T) {
	grid := newTestGrid(true)
	grid.Terrain[1*TacticalRowStride+2] = 1
	result, err := TraceMovement(grid, testCellClasses(), 0, 0, 4, 2, 20)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Complete {
		t.Fatalf("trace stopped at (%d,%d) although terrain is ignored", result.X, result.Y)
	}
}

// 判定發生在每一格的開頭，包含起點自己。
func TestTraceMovementStopsImmediatelyOnAnImpassableStart(t *testing.T) {
	grid := newTestGrid(false)
	grid.Terrain[0] = 1
	result, err := TraceMovement(grid, testCellClasses(), 0, 0, 4, 2, 20)
	if err != nil {
		t.Fatal(err)
	}
	if result.Complete || result.X != 0 || result.Y != 0 || result.Cost != 0 {
		t.Fatalf("result %+v, want an immediate stop at the origin", result)
	}
}

func TestTraceMovementRejectsCodesPastTheClassTable(t *testing.T) {
	grid := newTestGrid(false)
	grid.Terrain[0] = gamepack.CombatCellClassCount
	if _, err := TraceMovement(grid, testCellClasses(), 0, 0, 4, 2, 20); err == nil {
		t.Fatal("a cell class past the end of the table was accepted")
	}
}
