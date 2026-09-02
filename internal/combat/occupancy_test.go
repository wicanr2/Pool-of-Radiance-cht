package combat

import (
	"reflect"
	"testing"
)

// 依 spec 056：只留下 +10Eh 等於 mover 對立值者，且保留原順序。
func sides(table map[uint8]uint8) func(uint8) (uint8, bool) {
	return func(index uint8) (uint8, bool) {
		side, ok := table[index]
		return side, ok
	}
}

func TestSelectOpposingNearbyKeepsOriginalOrder(t *testing.T) {
	cells := []NearbyCell{
		{CombatantIndex: 3, Cost: 2, Facing: 1},
		{CombatantIndex: 7, Cost: 5, Facing: 0},
		{CombatantIndex: 9, Cost: 7, Facing: 2},
		{CombatantIndex: 4, Cost: 9, Facing: 3},
	}
	lookup := sides(map[uint8]uint8{3: 0xFF, 7: 0x00, 9: 0xFF, 4: 0xFF})
	selected, err := SelectOpposingNearby(cells, 0xFF, lookup)
	if err != nil {
		t.Fatal(err)
	}
	if want := []uint8{3, 9, 4}; !reflect.DeepEqual(selected, want) {
		t.Fatalf("selected %v, want %v", selected, want)
	}
}

func TestSelectOpposingNearbyRejectsUnknownCombatant(t *testing.T) {
	cells := []NearbyCell{{CombatantIndex: 12}}
	if _, err := SelectOpposingNearby(cells, 0xFF, sides(map[uint8]uint8{})); err == nil {
		t.Fatal("cell referencing an absent combatant record was accepted")
	}
}

func TestSelectOpposingNearbyRequiresLookup(t *testing.T) {
	if _, err := SelectOpposingNearby(nil, 0xFF, nil); err == nil {
		t.Fatal("missing side lookup accepted")
	}
}

// 原版把命中者往前搬移並把筆數改為命中數，不重新排序。
func TestCompactOpposingNearbyMovesHitsToFront(t *testing.T) {
	cells := []NearbyCell{
		{CombatantIndex: 3, Cost: 1},
		{CombatantIndex: 7, Cost: 2},
		{CombatantIndex: 9, Cost: 3},
	}
	lookup := sides(map[uint8]uint8{3: 0xFF, 7: 0x00, 9: 0xFF})
	compacted, err := CompactOpposingNearby(cells, 0xFF, lookup)
	if err != nil {
		t.Fatal(err)
	}
	want := []NearbyCell{{CombatantIndex: 3, Cost: 1}, {CombatantIndex: 9, Cost: 3}}
	if !reflect.DeepEqual(compacted, want) {
		t.Fatalf("compacted %+v, want %+v", compacted, want)
	}
	if len(cells) != 3 {
		t.Fatalf("backing array length changed to %d", len(cells))
	}
	if !reflect.DeepEqual(cells[:2], want) {
		t.Fatalf("compaction did not happen in place: %+v", cells[:2])
	}
}

func TestCompactOpposingNearbyKeepsNoneWhenSidesMatchMover(t *testing.T) {
	cells := []NearbyCell{{CombatantIndex: 3}, {CombatantIndex: 9}}
	lookup := sides(map[uint8]uint8{3: 0x00, 9: 0x00})
	compacted, err := CompactOpposingNearby(cells, 0xFF, lookup)
	if err != nil {
		t.Fatal(err)
	}
	if len(compacted) != 0 {
		t.Fatalf("compacted %d cells, want 0", len(compacted))
	}
}

// DS:2860h 的四列，逐組對回 docs/audit/ida-ds-footprint-offset-table.json。
func TestFootprintOffsetsMatchOriginalTable(t *testing.T) {
	want := map[uint8][]FootprintOffset{
		1: {{X: 0, Y: 0}},
		2: {{X: 0, Y: 0}, {X: 0, Y: 1}},
		3: {{X: 0, Y: 0}, {X: 1, Y: 0}},
		4: {{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 0, Y: 1}, {X: 1, Y: 1}},
	}
	for class, expected := range want {
		got, err := FootprintOffsets(class)
		if err != nil {
			t.Fatalf("class %d: %v", class, err)
		}
		if !reflect.DeepEqual(got, expected) {
			t.Fatalf("class %d offsets %+v, want %+v", class, got, expected)
		}
	}
}

func TestFootprintOffsetsRejectsClassesOutsideTheTable(t *testing.T) {
	for _, class := range []uint8{0, FootprintClassCount + 1, 255} {
		if _, err := FootprintOffsets(class); err == nil {
			t.Fatalf("class %d accepted although the original table has %d rows", class, FootprintClassCount)
		}
	}
}

// 原版固定跑四個槽，無效槽寫 0FFh 佔位，槽的位置要保留。
func TestFootprintCellsKeepSlotPositions(t *testing.T) {
	cells := FootprintCells(2, 10, 20)
	if !cells[0].Valid() || cells[0] != (FootprintCell{X: 10, Y: 20}) {
		t.Fatalf("slot 0 = %+v", cells[0])
	}
	if !cells[1].Valid() || cells[1] != (FootprintCell{X: 10, Y: 21}) {
		t.Fatalf("slot 1 = %+v", cells[1])
	}
	for slot := 2; slot < FootprintSlots; slot++ {
		if cells[slot].Valid() {
			t.Fatalf("slot %d should be the 0FFh placeholder, got %+v", slot, cells[slot])
		}
	}
}

func TestFootprintCellsCoverAllFourShapes(t *testing.T) {
	want := map[uint8][]FootprintCell{
		1: {{X: 10, Y: 20}},
		2: {{X: 10, Y: 20}, {X: 10, Y: 21}},
		3: {{X: 10, Y: 20}, {X: 11, Y: 20}},
		4: {{X: 10, Y: 20}, {X: 11, Y: 20}, {X: 10, Y: 21}, {X: 11, Y: 21}},
	}
	for class, expected := range want {
		cells := FootprintCells(class, 10, 20)
		var valid []FootprintCell
		for _, cell := range cells {
			if cell.Valid() {
				valid = append(valid, cell)
			}
		}
		if !reflect.DeepEqual(valid, expected) {
			t.Fatalf("class %d cells %+v, want %+v", class, valid, expected)
		}
	}
}

// 體型類別 0 的 combatant 在原版不參與，四個槽全部無效。
func TestFootprintCellsRejectClassZero(t *testing.T) {
	for _, cell := range FootprintCells(0, 10, 20) {
		if cell.Valid() {
			t.Fatalf("class 0 produced a valid cell %+v", cell)
		}
	}
}

// 原版把加總截成一個位元組，之後再以有號位元組判定有效，
// 因此越過 127 的座標會被當成無效格。
func TestFootprintCellsTreatByteOverflowAsInvalid(t *testing.T) {
	cells := FootprintCells(3, 127, 20)
	if !cells[0].Valid() {
		t.Fatalf("slot 0 = %+v, want valid", cells[0])
	}
	if cells[1].Valid() {
		t.Fatalf("slot 1 = %+v, want invalid after the byte wrap", cells[1])
	}
}
