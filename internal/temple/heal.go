// Package temple owns Pool-specific temple service rules proven from the DOS
// overlays. UI presentation remains in cmd/pool-game.
package temple

import (
	"errors"
	"fmt"

	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
	pooltreasure "github.com/wicanr2/Pool-of-Radiance-cht/internal/treasure"
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
	paidFrom, err := pay(state, partyIndex, service.Cost)
	if err != nil {
		return Result{}, err
	}
	result.PaidFrom = paidFrom
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

// pay 是 entry 3（`00BFh`）按 Y 之後的 `016Ah..01D1h`：角色的金幣等值（overlay-19 entry 11，
// 只收低位字）夠就從角色扣、餘額重鑄成白金＋金；不夠才看公款（overlay-21 entry 17／16）。
// 與武具店同一條 treasure.PayGold（spec 018〈付款來源與順序〉、spec 067〈公款〉）。
func pay(state *poolsave.State, partyIndex, cost int) (string, error) {
	source, paid, err := pooltreasure.PayGold(state, partyIndex, int64(cost))
	if err != nil {
		return "", err
	}
	if !paid {
		return "", ErrNotEnoughMoney
	}
	return string(source), nil
}
