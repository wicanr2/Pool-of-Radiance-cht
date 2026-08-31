package temple

import (
	"reflect"
	"testing"

	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

type fixedRoller int

func (value fixedRoller) Roll(count, sides int) int { return int(value) }

func wounded(gold, pooled int) poolsave.State {
	character := poolsave.Character{Name: "HERO", RaceID: "dwarf", GenderID: "male", ClassID: "fighter", AlignmentID: "lawful-good", Gold: gold, MaxHP: 20, CurrentHP: 2, PortraitHead: 1, PortraitBody: 1, IconSize: 1}
	return poolsave.State{Schema: poolsave.Schema, PooledGold: pooled, CharacterLibrary: []poolsave.Character{character}, Party: []poolsave.Character{character}}
}

func TestCureWoundsPaysCharacterFirstAndCapsHP(t *testing.T) {
	state := wounded(600, 900)
	result, err := CureWounds(&state, 0, 2, fixedRoller(21))
	if err != nil {
		t.Fatal(err)
	}
	if result.PaidFrom != "character" || result.Cost != 600 || result.Healed != 18 || state.Party[0].Gold != 0 || state.PooledGold != 900 || state.Party[0].CurrentHP != 20 || !reflect.DeepEqual(state.CharacterLibrary[0], state.Party[0]) {
		t.Fatalf("result=%+v state=%+v", result, state)
	}
}

func TestCureWoundsFallsBackToPoolWithoutCombiningFunds(t *testing.T) {
	state := wounded(99, 100)
	result, err := CureWounds(&state, 0, 0, fixedRoller(5))
	if err != nil {
		t.Fatal(err)
	}
	if result.PaidFrom != "pool" || state.Party[0].Gold != 99 || state.PooledGold != 0 || state.Party[0].CurrentHP != 7 {
		t.Fatalf("result=%+v state=%+v", result, state)
	}
	state = wounded(99, 99)
	if _, err := CureWounds(&state, 0, 0, fixedRoller(5)); err == nil || state.Party[0].Gold != 99 || state.PooledGold != 99 || state.Party[0].CurrentHP != 2 {
		t.Fatalf("insufficient payment mutated state: %+v err=%v", state, err)
	}
}
