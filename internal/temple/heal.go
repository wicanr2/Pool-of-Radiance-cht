// Package temple owns Pool-specific temple service rules proven from the DOS
// overlays. UI presentation remains in cmd/pool-game.
package temple

import (
	"errors"
	"fmt"

	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

var ErrNotEnoughMoney = errors.New("not enough money")

type Roller interface{ Roll(count, sides int) int }

type WoundService struct {
	Name  string
	Cost  int
	Count int
	Sides int
	Bonus int
}

var WoundServices = []WoundService{
	{Name: "Cure Light Wounds", Cost: 100, Count: 1, Sides: 8},
	{Name: "Cure Serious Wounds", Cost: 350, Count: 2, Sides: 8, Bonus: 1},
	{Name: "Cure Critical Wounds", Cost: 600, Count: 3, Sides: 8, Bonus: 3},
}

type Result struct {
	PaidFrom string
	Cost     int
	Healed   int
}

// CureWounds follows overlay-04 sub_BF: try the selected character's money
// first; only when it is insufficient, try pooled money for the whole cost.
// The two sources are never combined.
func CureWounds(state *poolsave.State, partyIndex, serviceIndex int, roller Roller) (Result, error) {
	if state == nil || partyIndex < 0 || partyIndex >= len(state.Party) {
		return Result{}, fmt.Errorf("Pool temple party index %d is invalid", partyIndex)
	}
	if serviceIndex < 0 || serviceIndex >= len(WoundServices) {
		return Result{}, fmt.Errorf("Pool wound service index %d is invalid", serviceIndex)
	}
	if roller == nil {
		return Result{}, fmt.Errorf("Pool temple wound roller is unavailable")
	}
	service := WoundServices[serviceIndex]
	character := &state.Party[partyIndex]
	result := Result{Cost: service.Cost}
	if int(character.Money[3]) >= service.Cost {
		character.Money[3] -= uint16(service.Cost)
		result.PaidFrom = "character"
	} else if uint64(state.PooledMoney[3]) >= uint64(service.Cost) {
		state.PooledMoney[3] -= uint32(service.Cost)
		result.PaidFrom = "pool"
	} else {
		return Result{}, ErrNotEnoughMoney
	}
	healed := roller.Roll(service.Count, service.Sides) + service.Bonus
	before := character.CurrentHP
	character.CurrentHP += healed
	if character.CurrentHP > character.MaxHP {
		character.CurrentHP = character.MaxHP
	}
	result.Healed = character.CurrentHP - before
	syncLibraryCharacter(state, *character)
	return result, nil
}

func syncLibraryCharacter(state *poolsave.State, character poolsave.Character) {
	for index := range state.CharacterLibrary {
		if state.CharacterLibrary[index].Name == character.Name {
			state.CharacterLibrary[index] = character
			return
		}
	}
}
