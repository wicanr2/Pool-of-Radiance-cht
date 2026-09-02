package gamepack

import "fmt"

// 護甲等級（spec 080）。overlay-25 `0C17h` 的「重算全部衍生值」在走物品鏈時
// 對每件穿戴中的物品呼叫 `0281h`，把 AC 累進五個槽，走完再結算成 `+111h`
// 與 `+112h`。
//
// 內部 AC 是「60 減去檯面上的 AC」，數字越大越好；結算時 `0F9Ah` 取
// 累加器[4] 與基礎 AC 的較大者，等於「盔甲比裸身差就不算」。
const (
	// BaseArmourClassOffset 是建角寫進角色記錄的基礎 AC（`+A9h`）。
	// 七名預設人物都是 50，也就是**不含敏捷修正**——敏捷在重算時才加。
	BaseArmourClassOffset = 0xa9
	// InternalArmourClassOffset 是重算出來的內部 AC。
	InternalArmourClassOffset = 0x111
	// RearArmourClassOffset 是不含敏捷與盾、再減 2 的那一份（`+112h`）。
	RearArmourClassOffset = 0x112
	// ArmourClassScale 是內部值與檯面值的換算基準：檯面 AC = 60 − 內部值。
	ArmourClassScale = 60

	// ArmourClassAccumulators 是 `0281h` 用的槽數。
	ArmourClassAccumulators = 5

	// ItemTypeArmourClassOffset 是物品型別表裡的 AC 欄位。
	ItemTypeArmourClassOffset = 0x06
	// itemTypeArmourClassEnabled 是該欄位的最高位：沒設就與 AC 無關。
	itemTypeArmourClassEnabled = 0x80

	// ItemCategoryShield 是盾（型別表 `+0` = 1），獨占累加器[1]。
	ItemCategoryShield = 1
	// ItemCategoryProtectionRing 是護符戒指（型別表 `+0` = 9）。
	ItemCategoryProtectionRing = 9

	// ItemSaveBonusOffset 是物品記錄裡的豁免修正（spec 075）。
	ItemSaveBonusOffset = 0x33
	// DexterityOffset 是角色記錄裡的敏捷；`10E3h` 讀的就是它。
	DexterityOffset = 0x13

	// rearArmourClassPenalty 是 `0FE1h` 結算時減掉的常數。
	rearArmourClassPenalty = 2
)

// ArmourClass 是一次重算的結果。Accumulators 保留下來是為了讓測試能對著
// spec 080 的表逐槽比對，而不是只比最後一個數字。
type ArmourClass struct {
	// Internal 是 `+111h`：五個槽相加。
	Internal int
	// Rear 是 `+112h`：累加器[4] + [3] + [2] − 2，不含敏捷與盾。
	Rear int
	// Accumulators 是結算前的五個槽。
	Accumulators [ArmourClassAccumulators]int
	// MagicArmour 是 `0281h` 立的旗標：穿著有加值的類別 2 盔甲。
	MagicArmour bool
	// SaveBonus 是 `+101h`：AC 值為 0 的物品把 `+33h` 累進來。
	SaveBonus int
}

// Tabletop 是檯面上的 AC，數字越小越好。
func (a ArmourClass) Tabletop() int { return ArmourClassScale - a.Internal }

// DexterityArmourAdjustment 是 overlay-25 `10E3h`，讀角色記錄的 `+13h`
// 算出累加器[0]。回傳的是**內部值方向**：正數代表防禦更好，檯面上的修正
// 是它取負。
//
// 與共用引擎的 ability.DexterityDefenceAdjustment 在 1..25 這段完全一致
// （TestDexterityArmourAdjustmentMatchesTheSharedEngine 盯著）；原版對
// 26 以上回 0，但遊戲裡到不了那裡。
func DexterityArmourAdjustment(dexterity int) int {
	switch {
	case dexterity >= 1 && dexterity <= 3:
		return -4
	case dexterity >= 4 && dexterity <= 6:
		return dexterity - 7
	case dexterity >= 15 && dexterity <= 18:
		return dexterity - 14
	case dexterity >= 19 && dexterity <= 20:
		return 4
	case dexterity >= 21 && dexterity <= 23:
		return 5
	case dexterity >= 24 && dexterity <= 25:
		return 6
	default:
		return 0
	}
}

// accumulateArmourClass 是 `0281h`：把一件物品記進 acc，必要時立旗標。
func accumulateArmourClass(acc *[ArmourClassAccumulators]int, magicArmour *bool, saveBonus *int, entry ItemTypeEntry, item []byte) {
	raw := entry.Raw[ItemTypeArmourClassOffset]
	if raw&itemTypeArmourClassEnabled == 0 {
		return
	}
	value := int(raw &^ itemTypeArmourClassEnabled)
	category := entry.Category()
	plus := int(int8(item[ItemPlusOffset]))

	if category == ItemCategoryShield {
		acc[1] = value + plus
		return
	}
	if value != 0 {
		if total := value + plus; total > acc[4] {
			acc[4] = total
			// 有加值的類別 2 盔甲壓掉護符戒指（`0382h`）。
			if plus > 0 && category == ItemCategoryArmour {
				*magicArmour = true
			}
		}
		return
	}
	if category == ItemCategoryProtectionRing {
		if plus > acc[3] {
			acc[3] = plus
		}
	} else {
		acc[2] += plus
	}
	// AC 值為 0 的兩支都會累加豁免修正（`032Ch`）。
	*saveBonus += int(int8(item[ItemSaveBonusOffset]))
}

// ArmourClassFor 走完整條鏈：清空 → 逐件累加 → 結算。items 只有 `+34h ≠ 0`
// 的會算數；順序沿用物品鏈的順序，因為累加器[1] 是覆寫而不是取大者。
func ArmourClassFor(base, dexterity int, items [][]byte, types *ItemTypeTable) (ArmourClass, error) {
	result := ArmourClass{}
	for index, item := range items {
		if len(item) <= ItemSaveBonusOffset {
			return ArmourClass{}, fmt.Errorf("item %d has %d bytes", index, len(item))
		}
		if item[ItemReadiedOffset] == 0 {
			continue
		}
		entry, err := types.Entry(item[ItemTypeOffset])
		if err != nil {
			return ArmourClass{}, fmt.Errorf("item %d: %w", index, err)
		}
		accumulateArmourClass(&result.Accumulators, &result.MagicArmour, &result.SaveBonus, entry, item)
	}
	if result.MagicArmour {
		result.Accumulators[3] = 0
	}
	result.Accumulators[0] = DexterityArmourAdjustment(dexterity)
	if base > result.Accumulators[4] {
		result.Accumulators[4] = base
	}
	for _, slot := range result.Accumulators {
		result.Internal += slot
	}
	result.Rear = result.Accumulators[4] + result.Accumulators[3] + result.Accumulators[2] - rearArmourClassPenalty
	return result, nil
}

// ArmourClassFromRecord 從原版 285-byte 記錄與它的物品算 AC。
func ArmourClassFromRecord(record []byte, items [][]byte, types *ItemTypeTable) (ArmourClass, error) {
	if len(record) <= RearArmourClassOffset {
		return ArmourClass{}, fmt.Errorf("character record has %d bytes, armour class needs %d", len(record), RearArmourClassOffset+1)
	}
	return ArmourClassFor(int(record[BaseArmourClassOffset]), int(record[DexterityOffset]), items, types)
}
