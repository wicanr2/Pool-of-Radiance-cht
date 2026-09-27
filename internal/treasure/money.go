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
// 只收 `+84h` 是 00h 或 B3h 的人（`052Bh..053Ch`）：ADD NPC 帶進來、士氣 80h 以上的
// NPC 不進公款（spec 040）。
func PoolMoney(state *poolsave.State) error {
	if state == nil {
		return fmt.Errorf("Pool money state is nil")
	}
	next := *state
	next.Party = append([]poolsave.Character(nil), state.Party...)
	for partyIndex := range next.Party {
		if !pooledMember(next.Party[partyIndex]) {
			continue
		}
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

// animatedControl 是死靈術叫起來的人的 `+84h`（overlay-22 `2138h`，spec 098）。
const animatedControl = 0xB3

// memberControl 是記錄 `+84h`：玩家建的角色是 0；ADD NPC 帶進來的記錄照原值。
func memberControl(character poolsave.Character) uint8 {
	if character.NPC && len(character.Record) > 0x84 {
		return character.Record[0x84]
	}
	return 0
}

// pooledMember 是 entry 5／6 的條件（`052Bh`、`05F9h`）：`+84h` 等於 00h 或 B3h。
func pooledMember(character poolsave.Character) bool {
	control := memberControl(character)
	return control == 0 || control == animatedControl
}

// ShareMoney reproduces overlay-21 entry 7（`062Eh..09E8h`，spec 040〈Share〉）。
// 全部照原版的 16 位元算術：
//
//  1. 份數 n 是 entry 6（`05D8h`）數的人：`+84h` 等於 00h 或 B3h。
//  2. 七欄各自（公款當有號 dword，大於 0 才算）：每份 = 公款 ÷ n、餘數 = 公款 mod n，
//     兩者都只留低位字（`06B0h`、`06E1h` `mov dx, ax`）。
//  3. 第一輪沿隊伍走、`+84h` 小於 80h 的人才發（`0721h`），每人依 6→0 逐欄：
//     容量 helper `0058h`（現重＋數量，16 位元相加）超過上限就只發「上限 − 現重」、
//     差額加回餘數；沒超過就發一份，餘數大於 0 時**拿餘數整筆**再問一次容量
//     （`07B6h` 推的是餘數，不是 1），過得了才多給一枚。
//  4. 第二輪（`087Ah`）依 6→0，餘數大於 0 的那一欄沿**整隊**走（不看 `+84h`）：
//     「上限 − 現重」無號大於 0 就發，發到餘數用完。
//  5. 公款每一欄改寫成那一欄的餘數（`09A3h..09C2h`）。
//
// B3h 的人算進份數卻不在第一輪發，他那一份就從公款消失——那是原版。
// 超重的人「上限 − 現重」繞回成很大的無號數，照樣照繞回的值發，也是原版。
func ShareMoney(state *poolsave.State) error {
	if state == nil {
		return fmt.Errorf("Pool share state is nil")
	}
	count := 0
	for _, member := range state.Party {
		if pooledMember(member) {
			count++
		}
	}
	if count == 0 {
		// 原版在這裡除以 0（RTL `05BBh:0294h` → 執行期錯誤 200）；remake 失敗即關閉。
		return ErrNoParty
	}
	next := *state
	next.Party = append([]poolsave.Character(nil), state.Party...)
	var share, remainder [CurrencyCount]uint16
	for currency, amount := range next.PooledMoney {
		if int32(amount) > 0 {
			share[currency] = uint16(amount / uint32(count))
			remainder[currency] = uint16(amount % uint32(count))
		}
	}
	loads := make([]uint16, len(next.Party))
	limits := make([]uint16, len(next.Party))
	for index, member := range next.Party {
		limit, load, err := coinCapacityAndLoad(member)
		if err != nil {
			return err
		}
		limits[index], loads[index] = uint16(limit), uint16(load)
	}
	// overloaded 是 `0058h`：現重＋數量（16 位元）大於上限就回 1，另給「上限 − 現重」。
	overloaded := func(index int, amount uint16) (bool, uint16) {
		if loads[index]+amount > limits[index] {
			return true, limits[index] - loads[index]
		}
		return false, 0
	}
	for index := range next.Party {
		if memberControl(next.Party[index]) >= 0x80 {
			continue
		}
		wallet := &next.Party[index].Money
		for currency := CurrencyCount - 1; currency >= 0; currency-- {
			over, room := overloaded(index, share[currency])
			if over {
				wallet[currency] += room
				remainder[currency] = remainder[currency] + share[currency] - room
				loads[index] += room
				continue
			}
			wallet[currency] += share[currency]
			loads[index] += share[currency]
			if remainder[currency] == 0 {
				continue
			}
			if again, _ := overloaded(index, remainder[currency]); !again {
				wallet[currency]++
				loads[index]++
				remainder[currency]--
			}
		}
	}
	for currency := CurrencyCount - 1; currency >= 0; currency-- {
		if remainder[currency] == 0 {
			continue
		}
		for index := range next.Party {
			room := limits[index] - loads[index]
			if room == 0 {
				continue
			}
			give := remainder[currency]
			if give > room {
				give = room
			}
			next.Party[index].Money[currency] += give
			loads[index] += give
			remainder[currency] -= give
		}
	}
	for currency := range next.PooledMoney {
		next.PooledMoney[currency] = uint32(remainder[currency])
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
	capacity, load, err := coinCapacityAndLoad(character)
	if err != nil {
		return 0, err
	}
	if load >= capacity {
		return 0, nil
	}
	return uint32(capacity - load), nil
}

// coinCapacityAndLoad 是容量 helper 的兩個數：上限（overlay-21 `0000h`）與現重
// （記錄 `+102h`：物品重量加上每一枚錢）。
func coinCapacityAndLoad(character poolsave.Character) (int, int, error) {
	capacity, err := poolcharacter.CarryCapacity(character.Abilities[0], character.ExceptionalStrength)
	if err != nil {
		return 0, 0, err
	}
	load := 0
	for _, amount := range character.Money {
		load += int(amount)
	}
	for index, item := range character.Inventory {
		weight, err := poolcharacter.ItemLoad(item.Raw)
		if err != nil {
			return 0, 0, fmt.Errorf("inventory item %d: %w", index, err)
		}
		load += weight
	}
	return capacity, load, nil
}

func syncLibraryCharacter(state *poolsave.State, character poolsave.Character) {
	for index := range state.CharacterLibrary {
		if state.CharacterLibrary[index].Name == character.Name {
			state.CharacterLibrary[index] = character
			return
		}
	}
}
