package gamepack

import "fmt"

// 豁免判定。overlay-24 entry 7（code `0D61h`）是全遊戲唯一一支：
//
//	roll = d20
//	roll = 1  → 必定失敗
//	roll = 20 → 必定成功
//	roll += 記錄 +101h 的修正 + 呼叫端傳入的修正
//	目標值 = 記錄 [+6Dh + 類別]
//	成功 ⟺ 目標值 ≤ roll
//
// 角色記錄 `+6Dh` 起的五個 byte 就是 AD&D 的豁免表，順序與規則書同：
// 七名預設人物（戰士 8／戰士 4／牧師 6／賊 9／法師 6）逐列與規則書的數字
// 相同，五個職業等級組合全中。
const (
	// SavingThrowOffset 是五個目標值在角色記錄裡的起點。
	SavingThrowOffset = 0x6d
	// SavingThrowCategories 是類別數。
	SavingThrowCategories = 5
	// SavingThrowBonusOffset 是記錄裡的豁免修正，帶正負號。
	SavingThrowBonusOffset = 0x101
	// SavingThrowDie 是骰面。
	SavingThrowDie = 20
)

// SaveCategory 是豁免類別，順序與 AD&D 規則書的欄位相同。
type SaveCategory uint8

const (
	// SaveParalyzation 是癱瘓／毒／死亡魔法。
	SaveParalyzation SaveCategory = 0
	// SavePetrification 是石化／變形。
	SavePetrification SaveCategory = 1
	// SaveRodStaffWand 是法杖／魔杖／權杖。
	SaveRodStaffWand SaveCategory = 2
	// SaveBreathWeapon 是吐息。
	SaveBreathWeapon SaveCategory = 3
	// SaveSpell 是法術。
	SaveSpell SaveCategory = 4
)

// SavingThrowTargets 取出角色記錄裡的五個目標值。
func SavingThrowTargets(record []byte) ([SavingThrowCategories]uint8, error) {
	var targets [SavingThrowCategories]uint8
	if len(record) < SavingThrowOffset+SavingThrowCategories {
		return targets, fmt.Errorf("character record has %d bytes, the saving throw table needs %d", len(record), SavingThrowOffset+SavingThrowCategories)
	}
	copy(targets[:], record[SavingThrowOffset:])
	return targets, nil
}

// SavingThrowBonus 取出記錄裡的豁免修正。
func SavingThrowBonus(record []byte) (int, error) {
	if len(record) <= SavingThrowBonusOffset {
		return 0, fmt.Errorf("character record has %d bytes, the saving throw bonus is at %#x", len(record), SavingThrowBonusOffset)
	}
	return int(int8(record[SavingThrowBonusOffset])), nil
}

// SavingThrow 依原版的順序判定：自然 1 與自然 20 在加修正之前就決定結果，
// 其餘才把修正加上去和目標值比。少了那兩個提前返回，一個修正夠大的角色會
// 連自然 1 都豁免成功。
func SavingThrow(record []byte, category SaveCategory, modifier, roll int) (bool, error) {
	if category >= SavingThrowCategories {
		return false, fmt.Errorf("saving throw category %d is outside 0..%d", category, SavingThrowCategories-1)
	}
	if roll < 1 || roll > SavingThrowDie {
		return false, fmt.Errorf("saving throw roll %d is outside 1..%d", roll, SavingThrowDie)
	}
	if roll == 1 {
		return false, nil
	}
	if roll == SavingThrowDie {
		return true, nil
	}
	targets, err := SavingThrowTargets(record)
	if err != nil {
		return false, err
	}
	bonus, err := SavingThrowBonus(record)
	if err != nil {
		return false, err
	}
	return int(targets[category]) <= roll+bonus+modifier, nil
}
