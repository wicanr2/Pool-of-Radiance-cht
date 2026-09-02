package combat

import "testing"

// nearbyTestState 擺一個 mover 與若干對手，全部是 1 格體型、地形全部可通行。
func nearbyTestState(cells ...CombatantCell) TacticalState {
	cellCount := TacticalRowStride * (TacticalMaxY + 1)
	table := make([]CombatantCell, 1, len(cells)+1)
	table = append(table, cells...)
	return TacticalState{
		Map:       TacticalGrid{Terrain: make([]uint8, cellCount)},
		Occupancy: make([]uint8, cellCount),
		Cells:     table,
		Classes:   testCellClasses(),
	}
}

func nearbyRequest(state TacticalState, moverIndex uint8, budget uint16) NearbyRequest {
	mover := state.Cells[moverIndex]
	return NearbyRequest{
		Map:     state.Map,
		Classes: state.Classes,
		Cells:   state.Cells,
		Class:   mover.FootprintClass,
		Facing:  DirectionUnset,
		Budget:  budget,
		BaseX:   mover.X,
		BaseY:   mover.Y,
	}
}

// 原版不跳過 mover 自己：自己的格對自己的格朝向弧無條件成立，成本 0，
// 所以排序後它一定排在最前面。剔除它是陣營篩選的工作。
func TestNearbyCellsKeepsTheMoverAndCostsTheStraightStep(t *testing.T) {
	state := nearbyTestState(
		CombatantCell{X: 10, Y: 10, FootprintClass: 1},
		CombatantCell{X: 11, Y: 10, FootprintClass: 1},
	)
	cells, err := NearbyCells(nearbyRequest(state, 1, NearbyReactionRange))
	if err != nil {
		t.Fatal(err)
	}
	if len(cells) != 2 {
		t.Fatalf("got %d nearby cells, want the mover and its neighbour: %+v", len(cells), cells)
	}
	if cells[0].CombatantIndex != 1 || cells[0].Cost != 0 {
		t.Fatalf("first cell %+v, want the mover itself at cost 0", cells[0])
	}
	if cells[1].CombatantIndex != 2 || cells[1].Cost != StraightStepCost {
		t.Fatalf("second cell %+v, want combatant 2 at cost %d", cells[1], StraightStepCost)
	}
	// 朝向未指定時要求出一個真正的方向，2 就是「向右」。
	if cells[1].Facing != 2 {
		t.Fatalf("facing %d, want 2 for a neighbour due east", cells[1].Facing)
	}
}

// 體型 0 的槽不參與，走出範圍的對手也不該進結果。
func TestNearbyCellsSkipsInactiveAndOutOfRangeCombatants(t *testing.T) {
	state := nearbyTestState(
		CombatantCell{X: 10, Y: 10, FootprintClass: 1},
		CombatantCell{X: 11, Y: 10},
		CombatantCell{X: 20, Y: 10, FootprintClass: 1},
	)
	cells, err := NearbyCells(nearbyRequest(state, 1, NearbyReactionRange))
	if err != nil {
		t.Fatal(err)
	}
	if len(cells) != 1 || cells[0].CombatantIndex != 1 {
		t.Fatalf("got %+v, want only the mover", cells)
	}
}

func nearbySides(sides ...uint8) func(uint8) (uint8, bool) {
	return func(index uint8) (uint8, bool) {
		if index == 0 || int(index) > len(sides) {
			return 0, false
		}
		return sides[index-1], true
	}
}

func TestOpposingNearbyAtDropsTheMoverAndItsAllies(t *testing.T) {
	state := nearbyTestState(
		CombatantCell{X: 10, Y: 10, FootprintClass: 1},
		CombatantCell{X: 11, Y: 10, FootprintClass: 1},
		CombatantCell{X: 9, Y: 10, FootprintClass: 1},
	)
	opponents, err := OpposingNearbyAt(state, 1, 10, 10, NearbyReactionRange, 1, nearbySides(0, 1, 0))
	if err != nil {
		t.Fatal(err)
	}
	if len(opponents) != 1 || opponents[0] != 2 {
		t.Fatalf("got %v, want just combatant 2", opponents)
	}
}

// 差集：往西走離開對手就是脫離，往北走仍然斜向相鄰就不是。
func TestLeavingOpponentsAfterStepIsTheDifferenceSet(t *testing.T) {
	for _, test := range []struct {
		name      string
		direction uint8
		want      []uint8
	}{
		{"west leaves the threat", 6, []uint8{2}},
		{"north stays adjacent", 0, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			state := nearbyTestState(
				CombatantCell{X: 10, Y: 10, FootprintClass: 1},
				CombatantCell{X: 11, Y: 10, FootprintClass: 1},
			)
			leaving, err := LeavingOpponentsAfterStep(state, 1, test.direction, 1, nearbySides(0, 1))
			if err != nil {
				t.Fatal(err)
			}
			if len(leaving) != len(test.want) {
				t.Fatalf("got %v, want %v", leaving, test.want)
			}
			for index := range test.want {
				if leaving[index] != test.want[index] {
					t.Fatalf("got %v, want %v", leaving, test.want)
				}
			}
			if state.Cells[1].X != 10 || state.Cells[1].Y != 10 {
				t.Fatalf("the mover stayed at (%d,%d); the probe must restore the position table",
					state.Cells[1].X, state.Cells[1].Y)
			}
		})
	}
}
