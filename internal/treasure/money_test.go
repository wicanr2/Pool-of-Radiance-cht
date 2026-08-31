package treasure

import (
	"errors"
	"math"
	"reflect"
	"testing"

	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

func moneyCharacter(name string, strength int) poolsave.Character {
	return poolsave.Character{Name: name, RaceID: "human", GenderID: "male", ClassID: "fighter", AlignmentID: "lawful-good", Abilities: [6]int{strength, 10, 10, 10, 10, 10}, MaxHP: 8, CurrentHP: 8, PortraitHead: 1, PortraitBody: 1, IconSize: 1}
}

func moneyState(characters ...poolsave.Character) poolsave.State {
	return poolsave.State{Schema: poolsave.Schema, CharacterLibrary: append([]poolsave.Character(nil), characters...), Party: append([]poolsave.Character(nil), characters...)}
}

func TestPoolMoneyPreservesSevenDenominationsAndSynchronizesLibrary(t *testing.T) {
	a, b := moneyCharacter("A", 18), moneyCharacter("B", 18)
	a.Money = [7]uint16{1, 2, 3, 4, 5, 6, 7}
	b.Money = [7]uint16{10, 20, 30, 40, 50, 60, 70}
	state := moneyState(a, b)
	state.PooledMoney = [7]uint32{100, 200, 300, 400, 500, 600, 700}
	if err := PoolMoney(&state); err != nil {
		t.Fatal(err)
	}
	want := [7]uint32{111, 222, 333, 444, 555, 666, 777}
	if state.PooledMoney != want || state.Party[0].Money != ([7]uint16{}) || state.Party[1].Money != ([7]uint16{}) || !reflect.DeepEqual(state.Party, state.CharacterLibrary) {
		t.Fatalf("pooled=%v party=%+v library=%+v", state.PooledMoney, state.Party, state.CharacterLibrary)
	}
}

func TestPoolMoneyOverflowIsAtomic(t *testing.T) {
	character := moneyCharacter("A", 18)
	character.Money[Copper] = 1
	state := moneyState(character)
	state.PooledMoney[Copper] = math.MaxUint32
	before := state
	if err := PoolMoney(&state); err == nil || !reflect.DeepEqual(state, before) {
		t.Fatalf("overflow err=%v state=%+v", err, state)
	}
}

func TestShareMoneyDistributesQuotientAndRemainderHighToLow(t *testing.T) {
	a, b, c := moneyCharacter("A", 18), moneyCharacter("B", 18), moneyCharacter("C", 18)
	state := moneyState(a, b, c)
	state.PooledMoney = [7]uint32{8, 0, 0, 7, 0, 0, 5}
	if err := ShareMoney(&state); err != nil {
		t.Fatal(err)
	}
	if state.PooledMoney != ([7]uint32{}) {
		t.Fatalf("pooled remainder=%v", state.PooledMoney)
	}
	if got := [3]uint16{state.Party[0].Money[Copper], state.Party[1].Money[Copper], state.Party[2].Money[Copper]}; got != [3]uint16{3, 3, 2} {
		t.Fatalf("copper=%v", got)
	}
	if got := [3]uint16{state.Party[0].Money[Gold], state.Party[1].Money[Gold], state.Party[2].Money[Gold]}; got != [3]uint16{3, 2, 2} {
		t.Fatalf("gold=%v", got)
	}
	if got := [3]uint16{state.Party[0].Money[Jewelry], state.Party[1].Money[Jewelry], state.Party[2].Money[Jewelry]}; got != [3]uint16{2, 2, 1} {
		t.Fatalf("jewelry=%v", got)
	}
	if !reflect.DeepEqual(state.Party, state.CharacterLibrary) {
		t.Fatal("library was not synchronized")
	}
}

func TestShareMoneyLeavesCapacityLimitedRemainderInPool(t *testing.T) {
	character := moneyCharacter("A", 3) // original capacity 1150
	character.Money[Copper] = 1149
	state := moneyState(character)
	state.PooledMoney[Gold] = 3
	if err := ShareMoney(&state); err != nil {
		t.Fatal(err)
	}
	if state.Party[0].Money[Gold] != 1 || state.PooledMoney[Gold] != 2 {
		t.Fatalf("wallet=%v pool=%v", state.Party[0].Money, state.PooledMoney)
	}
}

func TestTakeMoneySuccessAndFailuresAreAtomic(t *testing.T) {
	character := moneyCharacter("A", 3)
	state := moneyState(character)
	state.PooledMoney[Gold] = 10
	if err := TakeMoney(&state, 0, Gold, 4); err != nil {
		t.Fatal(err)
	}
	if state.PooledMoney[Gold] != 6 || state.Party[0].Money[Gold] != 4 || state.CharacterLibrary[0].Money[Gold] != 4 {
		t.Fatalf("state=%+v", state)
	}
	for _, test := range []struct {
		name   string
		mutate func(*poolsave.State)
		amount uint32
		want   error
	}{
		{name: "pool", amount: 7, want: ErrNotEnoughPool},
		{name: "capacity", mutate: func(value *poolsave.State) {
			value.Party[0].Money[Copper] = 1146
			value.CharacterLibrary[0] = value.Party[0]
		}, amount: 1, want: ErrOverloaded},
		{name: "wallet", mutate: func(value *poolsave.State) {
			value.Party[0].Money[Gold] = math.MaxUint16
			value.CharacterLibrary[0] = value.Party[0]
		}, amount: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			candidate := state
			candidate.Party = append([]poolsave.Character(nil), state.Party...)
			candidate.CharacterLibrary = append([]poolsave.Character(nil), state.CharacterLibrary...)
			if test.mutate != nil {
				test.mutate(&candidate)
			}
			before := candidate
			before.Party = append([]poolsave.Character(nil), candidate.Party...)
			before.CharacterLibrary = append([]poolsave.Character(nil), candidate.CharacterLibrary...)
			err := TakeMoney(&candidate, 0, Gold, test.amount)
			if err == nil || (test.want != nil && !errors.Is(err, test.want)) || !reflect.DeepEqual(candidate, before) {
				t.Fatalf("err=%v state=%+v before=%+v", err, candidate, before)
			}
		})
	}
}
