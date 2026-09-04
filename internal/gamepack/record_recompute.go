package gamepack

import (
	"encoding/binary"
	"fmt"
)

// 「重算這個角色的戰鬥數值」——overlay-25 `0E36h`（spec 063）。
//
// 原版每次裝備變動、載入角色與升級之後都跑這一支：把三個基礎值搬進來，
// 依備妥的武器算 THAC0 與傷害，再沿物品鏈累積負重與護甲。spec 063 寫下它時
// 兩支助手（`0281h` 護甲、`039Fh` 負重）還沒讀完，所以那一節標成 DRAFT；
// spec 080 與 spec 079 之後分別把它們閉合，這裡把三份規格接起來。
//
// 匯出 DOS 角色檔時需要它：remake 的角色模型只存「輸入」（能力值、等級、
// 背包），衍生欄位是現算現用的，不重算就會寫出一份 THAC0 與 AC 全是 0 的記錄。
const (
	// BaseThac0Offset 是建角寫下的基礎 THAC0 internal（`+2Dh`）。
	BaseThac0Offset = 0x2d
	// CurrentThac0Offset 是裝上武器之後的 THAC0 internal（`+110h`）。
	CurrentThac0Offset = 0x110
	// DamageDiceCountOffset、DamageDieSidesOffset、DamageBonusOffset 是
	// 武器算出來的三個傷害欄位（`+115h`／`+117h`／`+119h`）。
	DamageDiceCountOffset = 0x115
	DamageDieSidesOffset  = 0x117
	DamageBonusOffset     = 0x119
	// ClassLevelOffset 是每職業等級陣列（`+96h`，八格）。
	ClassLevelOffset = 0x96
	// AbilityBonusFlagOffset 是 `+0AAh`：非零才套用兩個力量修正。
	AbilityBonusFlagOffset = 0xaa
	// RaceOffset 是種族碼（`+2Eh`）；spec 063 第 9 步用它是否為 2 決定
	// 特定武器型別再加 1。
	RaceOffset = 0x2e
	// ItemCategoryWeapon 是物品型別表 `+0` 的武器類別，對應記錄的 `+0CCh` 槽。
	ItemCategoryWeapon = 0
	// LauncherOffset 與 AmmunitionOffset 是 `+0F8h`／`+0FCh` 那兩個物品槽。
	// 檔案裡存的是上次執行的遠指標，載入後沒有意義，所以重算時不追它們。
	LauncherOffset   = 0xf8
	AmmunitionOffset = 0xfc

	recomputeMinimumSize = DamageBonusOffset + 1
)

// RecomputeCombatFields 依 overlay-25 `0E36h` 把衍生欄位填回記錄。
// items 是這個角色的物品鏈，一件 63 bytes，順序照檔案。
//
// 不動的欄位：`+6Dh..+71h` 的豁免表與 `+73h` 的生命骰。它們由建角與升級寫下，
// 不在這一支的職責裡，remake 也還沒有產生端。
func RecomputeCombatFields(record []byte, items [][]byte, types *ItemTypeTable) ([]byte, error) {
	if len(record) < recomputeMinimumSize {
		return nil, fmt.Errorf("character record has %d bytes, the recompute needs %d",
			len(record), recomputeMinimumSize)
	}
	if types == nil {
		return nil, fmt.Errorf("Pool combat recompute needs an item type table")
	}
	result := append([]byte(nil), record...)

	// 1. 基礎 THAC0：八個職業取 internal 最大者。
	var levels [ClassThac0ClassCount]uint8
	copy(levels[:], result[ClassLevelOffset:])
	base, err := BaseThac0Internal(levels)
	if err != nil {
		return nil, err
	}
	result[BaseThac0Offset] = base

	// 2. 三個基礎值搬進來。
	result[InternalArmourClassOffset] = result[BaseArmourClassOffset]
	result[CurrentMovementOffset] = result[BaseMovementOffset]
	result[CurrentThac0Offset] = result[BaseThac0Offset]

	// 3. 備妥的武器決定 THAC0 與傷害。沒有武器就維持原值——原版那支直接返回，
	//    不補徒手傷害（spec 063 契約第 5 條）。
	weapon, hasWeapon, err := readiedWeapon(items, types)
	if err != nil {
		return nil, err
	}
	if !hasWeapon {
		// 沒有武器時原版補上兩個能力值加值（spec 063 的 `0E36h` 那一行）：
		// 命中加給 `+110h`、傷害加給 `+119h`。TARRY（`chrdatd5`）身上沒有
		// 備妥的武器，`+110h` 正好是基礎值加上 18/00 的 +3、`+119h` 是 +6。
		//
		// 「用不用 `+0AAh` 這個旗標決定要不要補」是**強推論**：那個 byte 在
		// 有武器那條路上就是這個作用，兩條路共用同一個開關最說得通，但
		// `0E36h` 這一行本身還沒逐指令讀過。七名預設人物的 `+0AAh` 都是 1，
		// 分不出兩種讀法。
		if result[AbilityBonusFlagOffset] != 0 {
			index, err := StrengthTableIndex(int(result[StrengthOffset]),
				int(result[ExceptionalStrengthOffset]))
			if err != nil {
				return nil, err
			}
			result[CurrentThac0Offset] = byte(int(result[CurrentThac0Offset]) + StrengthHitAdjustment(index))
			result[DamageBonusOffset] = byte(int(int8(result[DamageBonusOffset])) + StrengthDamageAdjustment(index))
		}
	}
	if hasWeapon {
		bearer := WeaponBearer{
			BaseThac0Internal:     result[BaseThac0Offset],
			Strength:              int(result[StrengthOffset]),
			ExceptionalStrength:   int(result[ExceptionalStrengthOffset]),
			Dexterity:             int(result[DexterityOffset]),
			AbilityBonusesEnabled: result[AbilityBonusFlagOffset] != 0,
			ClassBonusApplies:     result[RaceOffset] == 2,
		}
		stats, err := WeaponCombatStats(types, weapon[ItemTypeOffset],
			int(int8(weapon[ItemPlusOffset])), bearer)
		if err != nil {
			return nil, err
		}
		result[CurrentThac0Offset] = stats.Thac0Internal
		result[DamageDiceCountOffset] = stats.DamageCount
		result[DamageDieSidesOffset] = stats.DamageSides
		result[DamageBonusOffset] = byte(stats.DamageBonus)
	}

	// 4. 負重要先算，移動力的負重分段讀的是寫回去的 `+102h`。
	carried, err := CarriedWeightFromRecord(result, items)
	if err != nil {
		return nil, err
	}
	binary.LittleEndian.PutUint16(result[CarriedWeightOffset:], uint16(carried))

	// 5. 護甲與移動力。
	armour, err := ArmourClassFromRecord(result, items, types)
	if err != nil {
		return nil, err
	}
	result[InternalArmourClassOffset] = byte(armour.Internal)
	result[RearArmourClassOffset] = byte(armour.Rear)
	rate, err := MovementRate(result, items, types)
	if err != nil {
		return nil, err
	}
	result[CurrentMovementOffset] = byte(rate)
	return result, nil
}

// readiedWeapon 挑出物品鏈上備妥的武器。原版的記錄只有一個武器槽（`+0CCh`），
// 檔案裡那個槽存的是上次執行的遠指標，所以載入之後要從鏈上重新認：
// 備妥（`+34h ≠ 0`）而且型別表的類別是 0 的那一件。
func readiedWeapon(items [][]byte, types *ItemTypeTable) ([]byte, bool, error) {
	var weapon []byte
	found := false
	for index, item := range items {
		if len(item) <= ItemReadiedOffset || item[ItemReadiedOffset] == 0 {
			continue
		}
		entry, err := types.Entry(item[ItemTypeOffset])
		if err != nil {
			return nil, false, fmt.Errorf("item %d: %w", index, err)
		}
		if entry.Category() != ItemCategoryWeapon {
			continue
		}
		weapon, found = item, true
	}
	return weapon, found, nil
}
