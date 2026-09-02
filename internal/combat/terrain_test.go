package combat

import (
	"reflect"
	"testing"
)

// 表有 32 筆，且只有這幾筆的 Block 是 0——出貨資料裡 Level 全部是 0，
// 所以這幾筆就是可通行的地形碼。
func TestOriginalTerrainRulesMatchTheDump(t *testing.T) {
	rules := OriginalTerrainRules()
	if len(rules) != TerrainRuleCount {
		t.Fatalf("table has %d rules, want %d", len(rules), TerrainRuleCount)
	}
	var passable []int
	for code, rule := range rules {
		if rule.Level != 0 {
			t.Fatalf("code %d has Level %d; the shipped table is all zero", code, rule.Level)
		}
		if rule.Block == 0 {
			passable = append(passable, code)
		}
	}
	want := []int{5, 9, 11, 13, 15, 17, 23, 24, 26, 27, 28, 29, 30, 31}
	if !reflect.DeepEqual(passable, want) {
		t.Fatalf("passable codes %v, want %v", passable, want)
	}
	if rules[0].Block != 0xFF {
		t.Fatalf("code 0 Block %d, want 255", rules[0].Block)
	}
}

func TestOriginalTerrainRulesAreCopies(t *testing.T) {
	first := OriginalTerrainRules()
	first[5].Block = 0x7F
	if OriginalTerrainRules()[5].Block != 0 {
		t.Fatal("mutating the returned slice changed the table")
	}
}

func TestTerrainRuleAtRejectsCodesOutsideTheTable(t *testing.T) {
	rules := OriginalTerrainRules()
	if _, err := TerrainRuleAt(rules, TerrainRuleCount); err == nil {
		t.Fatal("a code past the end of the table was accepted")
	}
	rule, err := TerrainRuleAt(rules, 5)
	if err != nil {
		t.Fatal(err)
	}
	if rule.Block != 0 {
		t.Fatalf("code 5 Block %d, want 0", rule.Block)
	}
}

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

	// 反過來，正向可以越過斜向。
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
