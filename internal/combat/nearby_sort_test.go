package combat

import "testing"

// 主鍵是成本遞增。
func TestSortNearbyCellsOrdersByCost(t *testing.T) {
	cells := []NearbyCell{
		{CombatantIndex: 1, Cost: 9, Facing: 0},
		{CombatantIndex: 2, Cost: 3, Facing: 0},
		{CombatantIndex: 3, Cost: 6, Facing: 0},
	}
	SortNearbyCells(cells)
	want := []uint8{2, 3, 1}
	for index, expected := range want {
		if cells[index].CombatantIndex != expected {
			t.Fatalf("position %d holds combatant %d, want %d", index, cells[index].CombatantIndex, expected)
		}
	}
}

// 成本相同時朝向小的排前面。
func TestSortNearbyCellsBreaksTiesByFacing(t *testing.T) {
	cells := []NearbyCell{
		{CombatantIndex: 1, Cost: 4, Facing: 6},
		{CombatantIndex: 2, Cost: 4, Facing: 2},
		{CombatantIndex: 3, Cost: 4, Facing: 4},
	}
	SortNearbyCells(cells)
	want := []uint8{2, 3, 1}
	for index, expected := range want {
		if cells[index].CombatantIndex != expected {
			t.Fatalf("position %d holds combatant %d, want %d", index, cells[index].CombatantIndex, expected)
		}
	}
}

// 斜向（奇數朝向）不得越過正向（偶數朝向），即使它的索引比較小。
func TestSortNearbyCellsKeepsDiagonalsBehindAxisFacings(t *testing.T) {
	cells := []NearbyCell{
		{CombatantIndex: 1, Cost: 4, Facing: 2},
		{CombatantIndex: 2, Cost: 4, Facing: 1},
	}
	SortNearbyCells(cells)
	if cells[0].CombatantIndex != 1 {
		t.Fatalf("the diagonal facing overtook the axis facing: %+v", cells)
	}

	cells = []NearbyCell{
		{CombatantIndex: 1, Cost: 4, Facing: 3},
		{CombatantIndex: 2, Cost: 4, Facing: 2},
	}
	SortNearbyCells(cells)
	if cells[0].CombatantIndex != 2 {
		t.Fatalf("the axis facing did not overtake the diagonal: %+v", cells)
	}
}

func TestSortNearbyCellsHandlesShortTables(t *testing.T) {
	SortNearbyCells(nil)
	single := []NearbyCell{{CombatantIndex: 7, Cost: 3}}
	SortNearbyCells(single)
	if single[0].CombatantIndex != 7 {
		t.Fatal("a single-entry table was disturbed")
	}
}
