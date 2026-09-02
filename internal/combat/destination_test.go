package combat

import (
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// 測試用的類別碼：5 可進入（EntryThreshold 1），1 進不去（0FFh）。
// 兩個值取自原始表的同名索引，見 gamepack 的 66 筆 fixture。
const (
	openTerrain    = 5
	blockedTerrain = 1
)

func destinationTestClasses() CellClasses {
	var classes CellClasses
	classes[openTerrain] = gamepack.CombatCellClass{EntryThreshold: 1, PresentationCode: 4}
	classes[blockedTerrain] = gamepack.CombatCellClass{EntryThreshold: 0xFF, PathByte2: 2}
	return classes
}

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
		Classes: destinationTestClasses(),
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
	outcome, err := ResolveDestination(state, 1, 2, 0)
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Action != MovementAttack || outcome.Leaving {
		t.Fatalf("outcome %+v, want attack", outcome)
	}
	if outcome.Target != 3 {
		t.Fatalf("target %d, want 3", outcome.Target)
	}
}

func TestResolveDestinationEntersAndBlocks(t *testing.T) {
	state := newTacticalState(1, 10, 10)
	outcome, err := ResolveDestination(state, 1, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Action != MovementEnter || outcome.Leaving {
		t.Fatalf("outcome %+v, want enter", outcome)
	}

	state.setTerrain(11, 10, blockedTerrain)
	outcome, err = ResolveDestination(state, 1, 2, 200)
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Action != MovementBlocked || outcome.Leaving {
		t.Fatalf("outcome %+v, want blocked", outcome)
	}
}

// 盤面外不是「擋住」，原版會問玩家要不要離開戰鬥。
func TestResolveDestinationAsksBeforeLeavingTheBoard(t *testing.T) {
	state := newTacticalState(1, 0, 10)
	outcome, err := ResolveDestination(state, 1, 6, 10)
	if err != nil {
		t.Fatal(err)
	}
	if !outcome.Leaving {
		t.Fatalf("outcome %+v, want the leave-combat prompt", outcome)
	}
}
