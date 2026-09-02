package combat

import "testing"

func TestDeploymentTemplateIndexMatchesTheOriginalStrides(t *testing.T) {
	got, err := DeploymentTemplateIndex(0, 0, 0, 0)
	if err != nil || got != 0 {
		t.Fatalf("index %d err %v", got, err)
	}
	got, _ = DeploymentTemplateIndex(0, 1, 0, 0)
	if got != 0x42 {
		t.Fatalf("formation stride gave %d, want 0x42", got)
	}
	got, _ = DeploymentTemplateIndex(1, 0, 0, 0)
	if got != 0x108 {
		t.Fatalf("set stride gave %d, want 0x108", got)
	}
	got, _ = DeploymentTemplateIndex(0, 0, 1, 0)
	if got != 0x0B {
		t.Fatalf("row stride gave %d, want 0x0B", got)
	}
}

func TestDeploymentTemplateIndexRejectsOutOfRange(t *testing.T) {
	if _, err := DeploymentTemplateIndex(0, 0, DeploymentTemplateRows, 0); err == nil {
		t.Fatal("row past the template was accepted")
	}
	if _, err := DeploymentTemplateIndex(0, 0, 0, DeploymentTemplateCols); err == nil {
		t.Fatal("column past the template was accepted")
	}
	if _, err := DeploymentTemplateIndex(0, DeploymentsPerSet, 0, 0); err == nil {
		t.Fatal("formation past the set was accepted")
	}
}

// 部署與地圖建構器用同一個斜投影，X 原點多一。
func TestDeploymentCellSharesTheBuilderProjection(t *testing.T) {
	x, y := DeploymentCell(0, 0, 0, 0)
	if x != DeploymentOriginX || y != IndoorOriginY {
		t.Fatalf("origin (%d,%d)", x, y)
	}
	builderX, builderY, _ := IndoorTacticalCell(0, 0, 0, 1)
	if x != builderX || y != builderY {
		t.Fatalf("deployment (%d,%d) does not line up with builder subB 1 (%d,%d)", x, y, builderX, builderY)
	}
	x, y = DeploymentCell(0, 1, 0, 0)
	if x != DeploymentOriginX+IndoorStepXPerRow || y != IndoorOriginY+IndoorStepYPerRow {
		t.Fatalf("one row down gave (%d,%d); the projection must stay skewed", x, y)
	}
}

func TestRebuildOccupancyMarksEveryFootprintCell(t *testing.T) {
	cells := []CombatantCell{
		{},
		{X: 10, Y: 10, FootprintClass: 4},
		{X: 20, Y: 12, FootprintClass: 1},
		{X: 30, Y: 12, FootprintClass: 0},
	}
	occupancy, err := RebuildOccupancy(cells)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range [][3]int{
		{10, 10, 1}, {11, 10, 1}, {10, 11, 1}, {11, 11, 1},
		{20, 12, 2},
	} {
		got := occupancy[want[1]*TacticalRowStride+want[0]]
		if int(got) != want[2] {
			t.Fatalf("cell (%d,%d) holds %d, want %d", want[0], want[1], got, want[2])
		}
	}
	if occupancy[12*TacticalRowStride+30] != 0 {
		t.Fatal("a combatant with footprint class 0 was placed")
	}
}

func TestRebuildOccupancyStartsFromAClearGrid(t *testing.T) {
	occupancy, err := RebuildOccupancy([]CombatantCell{{}})
	if err != nil {
		t.Fatal(err)
	}
	if len(occupancy) != TacticalMapCellCount {
		t.Fatalf("grid has %d cells", len(occupancy))
	}
	for _, code := range occupancy {
		if code != 0 {
			t.Fatal("an empty roster produced a non-empty grid")
		}
	}
}

// 畫面相對座標是位元組減法，會繞回。
func TestScreenRelativeCellsWrapLikeTheOriginal(t *testing.T) {
	cells := []CombatantCell{{}, {X: 30, Y: 12}, {X: 2, Y: 1}}
	offsetsX, offsetsY := ScreenRelativeCells(cells, ViewportOrigin{X: 5, Y: 3})
	if offsetsX[1] != 25 || offsetsY[1] != 9 {
		t.Fatalf("first combatant offsets (%d,%d)", offsetsX[1], offsetsY[1])
	}
	if offsetsX[2] != 253 || offsetsY[2] != 254 {
		t.Fatalf("negative offsets did not wrap: (%d,%d)", offsetsX[2], offsetsY[2])
	}
}

func TestTryPlaceCombatantFollowsTheOriginalOrder(t *testing.T) {
	var classes CellClasses
	classes[5] = cellClassOpenGround()
	classes[1] = blockedCellClass()

	if got, _ := TryPlaceCombatant(0, 0, 5, classes); got != PlacementTemplateSlotUsed {
		t.Fatalf("a consumed template slot gave %v", got)
	}
	if got, _ := TryPlaceCombatant(0xFF, 3, 5, classes); got != PlacementCellOccupied {
		t.Fatalf("an occupied cell gave %v", got)
	}
	if got, _ := TryPlaceCombatant(0xFF, 0, OffBoardDestinationClass, classes); got != PlacementCellNotEnterable {
		t.Fatalf("an off-board cell gave %v", got)
	}
	if got, _ := TryPlaceCombatant(0xFF, 0, 1, classes); got != PlacementCellNotEnterable {
		t.Fatalf("a blocking cell gave %v", got)
	}
	if got, err := TryPlaceCombatant(0xFF, 0, 5, classes); err != nil || got != PlacementAccepted {
		t.Fatalf("an open cell gave %v err %v", got, err)
	}
}
