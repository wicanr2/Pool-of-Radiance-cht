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
		{First: 1, Second: 2, CombatantIndex: 3},
		{First: 4, Second: 5, CombatantIndex: 7},
		{First: 6, Second: 7, CombatantIndex: 9},
		{First: 8, Second: 9, CombatantIndex: 4},
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
		{First: 1, CombatantIndex: 3},
		{First: 2, CombatantIndex: 7},
		{First: 3, CombatantIndex: 9},
	}
	lookup := sides(map[uint8]uint8{3: 0xFF, 7: 0x00, 9: 0xFF})
	compacted, err := CompactOpposingNearby(cells, 0xFF, lookup)
	if err != nil {
		t.Fatal(err)
	}
	want := []NearbyCell{{First: 1, CombatantIndex: 3}, {First: 3, CombatantIndex: 9}}
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
