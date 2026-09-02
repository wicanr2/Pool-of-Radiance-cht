package combat

import "testing"

// 地形碼 5 的 EntryThreshold 是 1（可進入），碼 1 是 0FFh（進不去）。
const (
	openTerrain    = 5
	blockedTerrain = 1
)

func newTacticalState(moverClass uint8, moverX, moverY uint8) TacticalState {
	cellCount := TacticalRowStride * (TacticalMaxY + 1)
	terrain := make([]uint8, cellCount)
	for index := range terrain {
		terrain[index] = openTerrain
	}
	state := TacticalState{
		Map:       TacticalGrid{Terrain: terrain},
		Occupancy: make([]uint8, cellCount),
		Cells: []CombatantCell{
			{},
			{X: moverX, Y: moverY, FootprintClass: moverClass},
		},
		Rules: OriginalTerrainRules(),
	}
	return state
}

func (state TacticalState) put(x, y, occupant uint8) {
	state.Occupancy[int(y)*TacticalRowStride+int(x)] = occupant
}

func (state TacticalState) setTerrain(x, y, code uint8) {
	state.Map.Terrain[int(y)*TacticalRowStride+int(x)] = code
}

func TestCellAtReportsZeroesOutsideTheBoard(t *testing.T) {
	state := newTacticalState(1, 10, 10)
	occupant, class, err := state.CellAt(TacticalMaxX+1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if occupant != 0 || class != 0 {
		t.Fatalf("off-board cell reported occupant %d class %d", occupant, class)
	}
}

func TestProbeDestinationFindsAnOccupant(t *testing.T) {
	state := newTacticalState(1, 10, 10)
	state.put(11, 10, 3)
	target, class, err := ProbeDestination(state, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if target != 3 {
		t.Fatalf("target %d, want 3", target)
	}
	if class != openTerrain {
		t.Fatalf("class %d, want %d", class, openTerrain)
	}
}

// 撞到自己不算目標：橫向兩格的 mover 往右走時，前一格仍是自己。
func TestProbeDestinationIgnoresTheMoverItself(t *testing.T) {
	state := newTacticalState(3, 10, 10)
	state.put(10, 10, 1)
	state.put(11, 10, 1)
	target, _, err := ProbeDestination(state, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if target != 0 {
		t.Fatalf("target %d, want 0", target)
	}
}

// 大型 mover 的四個目的格取最難進的那一個。
func TestProbeDestinationTakesTheHardestCell(t *testing.T) {
	state := newTacticalState(4, 10, 10)
	state.setTerrain(12, 11, blockedTerrain)
	_, class, err := ProbeDestination(state, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if class != blockedTerrain {
		t.Fatalf("class %d, want the blocking terrain %d", class, blockedTerrain)
	}
}

func TestProbeDestinationReportsOffBoard(t *testing.T) {
	state := newTacticalState(1, 0, 10)
	_, class, err := ProbeDestination(state, 1, 6)
	if err != nil {
		t.Fatal(err)
	}
	if class != OffBoardDestinationClass {
		t.Fatalf("class %d, want %d", class, OffBoardDestinationClass)
	}
}

// 體型類別不在表內時原版四個槽全部略過，類別停在初值。
func TestProbeDestinationKeepsTheDefaultClassForAnUnknownFootprint(t *testing.T) {
	state := newTacticalState(0, 10, 10)
	target, class, err := ProbeDestination(state, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if target != 0 || class != DefaultDestinationClass {
		t.Fatalf("target %d class %d, want 0 and %d", target, class, DefaultDestinationClass)
	}
}

func TestProbeDestinationRejectsAnUnknownMover(t *testing.T) {
	state := newTacticalState(1, 10, 10)
	if _, _, err := ProbeDestination(state, 0, 2); err == nil {
		t.Fatal("mover index 0 was accepted")
	}
	if _, _, err := ProbeDestination(state, 9, 2); err == nil {
		t.Fatal("a mover index past the table was accepted")
	}
}

func TestResolveDestinationRoutesToAttackBeforeTheThresholdGate(t *testing.T) {
	state := newTacticalState(1, 10, 10)
	state.put(11, 10, 3)
	state.setTerrain(11, 10, blockedTerrain)
	action, prompt, err := ResolveDestination(state, 1, 2, 0)
	if err != nil {
		t.Fatal(err)
	}
	if action != MovementAttack || prompt {
		t.Fatalf("action %v prompt %v, want attack", action, prompt)
	}
}

func TestResolveDestinationEntersAndBlocks(t *testing.T) {
	state := newTacticalState(1, 10, 10)
	action, prompt, err := ResolveDestination(state, 1, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	if action != MovementEnter || prompt {
		t.Fatalf("action %v prompt %v, want enter", action, prompt)
	}

	state.setTerrain(11, 10, blockedTerrain)
	action, prompt, err = ResolveDestination(state, 1, 2, 200)
	if err != nil {
		t.Fatal(err)
	}
	if action != MovementBlocked || prompt {
		t.Fatalf("action %v prompt %v, want blocked", action, prompt)
	}
}

// 盤面外不是「擋住」，原版會問玩家要不要離開戰鬥。
func TestResolveDestinationAsksBeforeLeavingTheBoard(t *testing.T) {
	state := newTacticalState(1, 0, 10)
	action, prompt, err := ResolveDestination(state, 1, 6, 10)
	if err != nil {
		t.Fatal(err)
	}
	if !prompt {
		t.Fatalf("action %v prompt %v, want the leave-combat prompt", action, prompt)
	}
}
