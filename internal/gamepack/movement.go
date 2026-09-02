package gamepack

import (
	"encoding/binary"
	"fmt"
)

// 移動力。`29h ENCOUNTER MENU`（spec 078）的逃跑與逼近判定要拿隊伍最慢與最快
// 的移動力去比門檻，所以這是那條 opcode 的前置條件。
//
// 算法是三段管線，順序不能換：
//
//  1. 重設：`+11Ch = +72h`（overlay-25 `0E36h`，同一段還重設 AC 與 THAC0）。
//  2. 盔甲：穿在身上的盔甲依重量把它壓成 12／9／6，魔法盔甲再加回 3
//     （overlay-25 `01F8h`）。
//  3. 負重：`+102h` 減掉力量的重量寬容之後分段再壓，而且**只會往下壓**
//     （overlay-25 `039Fh`）。
const (
	// BaseMovementRate 是建角寫進 `+72h` 的基礎移動力。
	BaseMovementRate = 12
	// AnimatedDeadMovementRate 是 Animate Dead 叫起來的不死生物
	// （overlay-22 `2115h`）。
	AnimatedDeadMovementRate = 6

	// BaseMovementOffset 是基礎移動力在角色記錄裡的位置。
	BaseMovementOffset = 0x72
	// CurrentMovementOffset 是目前移動力。
	CurrentMovementOffset = 0x11c
	// CarriedWeightOffset 是總負重（word）。
	CarriedWeightOffset = 0x102
	// MoneyOffset 是七種貨幣的枚數（七個 word）在角色記錄裡的起點。
	MoneyOffset = 0x88
	// MoneyCurrencies 是貨幣種類數（spec 040）。
	MoneyCurrencies = 7
	// StrengthOffset 與 ExceptionalStrengthOffset 是力量與例外力量。
	StrengthOffset            = 0x10
	ExceptionalStrengthOffset = 0x16

	// ItemTypeOffset 是物品記錄裡的型別索引。
	ItemTypeOffset = 0x2e
	// ItemPlusOffset 是魔法加值。
	ItemPlusOffset = 0x32
	// ItemReadiedOffset 是「穿戴中」旗標。
	ItemReadiedOffset = 0x34
	// ItemWeightOffset 是重量（word）。
	ItemWeightOffset = 0x37
	// ItemCountOffset 是數量；0 當成 1。
	ItemCountOffset = 0x39

	// ItemCategoryArmour 是物品型別表 `+0` 代表盔甲的值；只有它會影響移動力。
	ItemCategoryArmour = 2
	// itemRecordSize 是一筆物品記錄的大小。
	itemRecordSize = 63
)

// CarriedWeight 是總負重：每件物品的重量乘上數量（數量 0 當 1），再加上
// **身上所有硬幣的枚數**——一枚就是一單位（overlay-25 `0C17h` 起的累加，
// 加上 `0F67h` 的下限）。七名預設人物的 `+102h` 逐一對得上。
func CarriedWeight(items [][]byte, money [MoneyCurrencies]uint16) (int, error) {
	total := 0
	for index, item := range items {
		if len(item) <= ItemWeightOffset+1 {
			return 0, fmt.Errorf("item %d has %d bytes", index, len(item))
		}
		weight := int(binary.LittleEndian.Uint16(item[ItemWeightOffset:]))
		if count := int(item[ItemCountOffset]); count > 0 {
			weight *= count
		}
		total += weight
	}
	for _, coins := range money {
		total += int(coins)
	}
	return total, nil
}

// CarriedWeightFromRecord 從原版 285-byte 記錄與它的物品算總負重。
func CarriedWeightFromRecord(record []byte, items [][]byte) (int, error) {
	if len(record) < MoneyOffset+2*MoneyCurrencies {
		return 0, fmt.Errorf("character record has %d bytes, money needs %d", len(record), MoneyOffset+2*MoneyCurrencies)
	}
	var money [MoneyCurrencies]uint16
	for index := range money {
		money[index] = binary.LittleEndian.Uint16(record[MoneyOffset+2*index:])
	}
	return CarriedWeight(items, money)
}

// ArmourMovementRate 是一件盔甲把移動力壓成多少（overlay-25 `01F8h`）。
// weight 是物品的 `+37h`，plus 是 `+32h`，base 是角色的 `+72h`。
func ArmourMovementRate(weight int, plus int, base int) int {
	rate := AnimatedDeadMovementRate
	switch {
	case weight >= 0 && weight <= 150:
		rate = base
	case weight >= 151 && weight <= 399:
		rate = 9
	}
	// 有加值的盔甲等於輕一級；已經比 9 快的不再加。
	if plus != 0 && rate <= 9 {
		rate += 3
	}
	return rate
}

// StrengthWeightAllowance 是力量給的重量寬容（overlay-25 `13F8h`），
// 索引由 StrengthTableIndex 給（spec 063 同一套）。
func StrengthWeightAllowance(index int) int {
	switch {
	case index >= 1 && index <= 3:
		return -350
	case index >= 4 && index <= 5:
		return -250
	case index >= 6 && index <= 7:
		return -150
	case index >= 12 && index <= 13:
		return 100
	case index >= 14 && index <= 15:
		return 200
	case index == 16:
		return 350
	case index >= 17 && index <= 21:
		return 500 + (index-17)*250
	case index >= 22 && index <= 26:
		return 2000 + (index-22)*1000
	case index == 27:
		return 7500
	case index >= 28 && index <= 30:
		return 9000 + (index-28)*3000
	default:
		return 0
	}
}

// EncumbranceMovementRate 依「負重減寬容」的結果把移動力再往下壓
// （overlay-25 `039Fh`）。負值先夾成 0；**只會往下壓**，不會把盔甲壓低的
// 值調回去。
func EncumbranceMovementRate(carried, allowance, rate int) int {
	load := carried - allowance
	if load < 0 {
		load = 0
	}
	limited := rate
	switch {
	case load <= 512:
		// 維持原值。
	case load <= 768:
		limited = 9
	case load <= 1024:
		limited = 6
	default:
		limited = 3
	}
	if limited < rate {
		return limited
	}
	return rate
}

// MovementRateFor 走完整條管線：基礎值 → 盔甲 → 負重。
//
// 盔甲那一段只看**穿戴中**的物品：`01F8h` 自己不檢查 `+34h`，是呼叫端挑的；
// 沒穿在身上的盔甲不該拖慢腳步。
func MovementRateFor(base, strength, exceptional, carried int, items [][]byte, types *ItemTypeTable) (int, error) {
	rate := base
	for index, item := range items {
		if len(item) <= ItemWeightOffset+1 {
			return 0, fmt.Errorf("item %d has %d bytes", index, len(item))
		}
		if item[ItemReadiedOffset] == 0 {
			continue
		}
		entry, err := types.Entry(item[ItemTypeOffset])
		if err != nil {
			return 0, fmt.Errorf("item %d: %w", index, err)
		}
		if entry.Category() != ItemCategoryArmour {
			continue
		}
		rate = ArmourMovementRate(int(binary.LittleEndian.Uint16(item[ItemWeightOffset:])),
			int(int8(item[ItemPlusOffset])), base)
	}
	index, err := StrengthTableIndex(strength, exceptional)
	if err != nil {
		return 0, err
	}
	return EncumbranceMovementRate(carried, StrengthWeightAllowance(int(index)), rate), nil
}

// MovementRate 從原版 285-byte 記錄與它的物品算移動力。
func MovementRate(record []byte, items [][]byte, types *ItemTypeTable) (int, error) {
	if len(record) <= CurrentMovementOffset {
		return 0, fmt.Errorf("character record has %d bytes, movement needs %d", len(record), CurrentMovementOffset+1)
	}
	carried := int(binary.LittleEndian.Uint16(record[CarriedWeightOffset:]))
	return MovementRateFor(int(record[BaseMovementOffset]), int(record[StrengthOffset]),
		int(record[ExceptionalStrengthOffset]), carried, items, types)
}
