// Package treasure owns Pool-specific seven-currency treasure service rules.
package treasure

import (
	"errors"
	"fmt"
	"math"

	poolcharacter "github.com/wicanr2/Pool-of-Radiance-cht/internal/character"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

const (
	Copper = iota
	Silver
	Electrum
	Gold
	Platinum
	Gems
	Jewelry
	CurrencyCount
)

var Names = [CurrencyCount]string{"Copper", "Silver", "Electrum", "Gold", "Platinum", "Gems", "Jewelry"}

var (
	ErrNoParty       = errors.New("no active party members")
	ErrNotEnoughPool = errors.New("not enough pooled treasure")
	ErrOverloaded    = errors.New("character is overloaded")
)

// PoolMoney reproduces overlay-21:0506..05D7: move every wallet entry into
// the corresponding 32-bit party pool, then clear the character wallets.
func PoolMoney(state *poolsave.State) error {
	if state == nil {
		return fmt.Errorf("Pool money state is nil")
	}
	next := *state
	next.Party = append([]poolsave.Character(nil), state.Party...)
	for partyIndex := range next.Party {
		for currency, amount := range next.Party[partyIndex].Money {
			if uint64(next.PooledMoney[currency])+uint64(amount) > math.MaxUint32 {
				return fmt.Errorf("pooled %s overflows uint32", Names[currency])
			}
			next.PooledMoney[currency] += uint32(amount)
			next.Party[partyIndex].Money[currency] = 0
		}
		syncLibraryCharacter(&next, next.Party[partyIndex])
	}
	*state = next
	return nil
}

// ShareMoney reproduces overlay-21:062E..09E8. Each denomination is divided
// independently; capacity-limited remainders stay in the 32-bit party pool.
func ShareMoney(state *poolsave.State) error {
	if state == nil {
		return fmt.Errorf("Pool share state is nil")
	}
	if len(state.Party) == 0 {
		return ErrNoParty
	}
	next := *state
	next.Party = append([]poolsave.Character(nil), state.Party...)
	for currency := CurrencyCount - 1; currency >= 0; currency-- {
		share := next.PooledMoney[currency] / uint32(len(next.Party))
		leftover := next.PooledMoney[currency] % uint32(len(next.Party))
		for partyIndex := range next.Party {
			available, err := availableCoinCapacity(next.Party[partyIndex])
			if err != nil {
				return err
			}
			give := uint32(0)
			if share > available {
				give = available
				leftover += share - available
			} else {
				give = share
				available -= share
				if leftover != 0 && available != 0 {
					give++
					leftover--
				}
			}
			walletRoom := uint32(math.MaxUint16 - next.Party[partyIndex].Money[currency])
			if give > walletRoom {
				leftover += give - walletRoom
				give = walletRoom
			}
			next.Party[partyIndex].Money[currency] += uint16(give)
		}
		next.PooledMoney[currency] = leftover
	}
	for partyIndex := range next.Party {
		syncLibraryCharacter(&next, next.Party[partyIndex])
	}
	*state = next
	return nil
}

// TakeMoney transfers one selected denomination atomically from the pooled
// treasure to one character, preserving the original capacity gate.
func TakeMoney(state *poolsave.State, partyIndex, currency int, amount uint32) error {
	if state == nil || partyIndex < 0 || partyIndex >= len(state.Party) {
		return fmt.Errorf("Pool money party index %d is invalid", partyIndex)
	}
	if currency < 0 || currency >= CurrencyCount {
		return fmt.Errorf("Pool money currency index %d is invalid", currency)
	}
	if amount > state.PooledMoney[currency] {
		return ErrNotEnoughPool
	}
	if amount > uint32(math.MaxUint16-state.Party[partyIndex].Money[currency]) {
		return fmt.Errorf("%s wallet overflows uint16", Names[currency])
	}
	available, err := availableCoinCapacity(state.Party[partyIndex])
	if err != nil {
		return err
	}
	if amount > available {
		return ErrOverloaded
	}
	next := *state
	next.Party = append([]poolsave.Character(nil), state.Party...)
	next.PooledMoney[currency] -= amount
	next.Party[partyIndex].Money[currency] += uint16(amount)
	syncLibraryCharacter(&next, next.Party[partyIndex])
	*state = next
	return nil
}

func availableCoinCapacity(character poolsave.Character) (uint32, error) {
	capacity, err := poolcharacter.CarryCapacity(character.Abilities[0], character.ExceptionalStrength)
	if err != nil {
		return 0, err
	}
	load := 0
	for _, amount := range character.Money {
		load += int(amount)
	}
	for index, item := range character.Inventory {
		weight, err := poolcharacter.ItemLoad(item.Raw)
		if err != nil {
			return 0, fmt.Errorf("inventory item %d: %w", index, err)
		}
		load += weight
	}
	if load >= capacity {
		return 0, nil
	}
	return uint32(capacity - load), nil
}

func syncLibraryCharacter(state *poolsave.State, character poolsave.Character) {
	for index := range state.CharacterLibrary {
		if state.CharacterLibrary[index].Name == character.Name {
			state.CharacterLibrary[index] = character
			return
		}
	}
}
