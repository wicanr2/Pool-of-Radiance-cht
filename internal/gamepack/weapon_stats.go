package gamepack

import "fmt"

// 能力值修正與武器數值。三張表都由 overlay-25 逐分支讀出來，不是照 AD&D 規則書
// 補的：原版在幾個邊界上與規則書不同（力量表索引 0 與 31 以上都回 0，
// 敏捷 0..2 回 -4），照規則書寫會在那些邊界上與原版不一致。

// StrengthTableIndex 是 overlay-25 `1203h`：把力量與特殊力量百分比折成一個索引，
// 力量修正與負重容量（spec 035）都用它。
//
// 18 點的百分比分段是 AD&D 的 18/01-50、18/51-75、18/76-90、18/91-99、18/00，
// 原版把 18/00 寫成百分比 100。
func StrengthTableIndex(strength, exceptional int) (uint8, error) {
	if strength < 0 || strength > 25 {
		return 0, fmt.Errorf("Pool strength %d is outside 0..25", strength)
	}
	switch {
	case strength <= 17:
		return uint8(strength), nil
	case strength == 18:
		switch {
		case exceptional == 0:
			return 0x12, nil
		case exceptional >= 1 && exceptional <= 50:
			return 0x13, nil
		case exceptional >= 51 && exceptional <= 75:
			return 0x14, nil
		case exceptional >= 76 && exceptional <= 90:
			return 0x15, nil
		case exceptional >= 91 && exceptional <= 99:
			return 0x16, nil
		case exceptional == 100:
			return 0x17, nil
		}
		return 0, fmt.Errorf("Pool exceptional strength %d is outside 0..100", exceptional)
	default: // 19..25
		return uint8(strength + 5), nil
	}
}

// StrengthHitAdjustment 是 overlay-25 `12AEh`，加到 internal THAC0 上。
// 回傳的是加到 internal 值的量：internal 越小代表越好命中，所以正的修正
// 在 typed THAC0 上是變好。
func StrengthHitAdjustment(index uint8) int {
	switch {
	case index >= 1 && index <= 3:
		return -3
	case index >= 4 && index <= 5:
		return -2
	case index >= 6 && index <= 7:
		return -1
	case index >= 0x11 && index <= 0x13:
		return 1
	case index >= 0x14 && index <= 0x16:
		return 2
	case index >= 0x17 && index <= 0x19:
		return 3
	case index >= 0x1a && index <= 0x1b:
		return 4
	case index >= 0x1c && index <= 0x1e:
		return int(index) - 0x17
	}
	return 0
}

// StrengthDamageAdjustment 是 overlay-25 `1366h`，加到 record `+119h` 上。
func StrengthDamageAdjustment(index uint8) int {
	switch {
	case index >= 1 && index <= 2:
		return -2
	case index >= 3 && index <= 5:
		return -1
	case index == 0x10:
		return 1
	case index >= 0x11 && index <= 0x13:
		return int(index) - 0x10
	case index >= 0x14 && index <= 0x1d:
		return int(index) - 0x11
	case index == 0x1e:
		return 14
	}
	return 0
}

// DexterityMissileAdjustment 是 overlay-25 `1173h`，投射武器的命中修正。
// 它直接讀角色的敏捷，不經過力量那張索引表。
func DexterityMissileAdjustment(dexterity int) int {
	switch {
	case dexterity >= 0 && dexterity <= 2:
		return -4
	case dexterity >= 3 && dexterity <= 5:
		return dexterity - 6
	case dexterity >= 0x10 && dexterity <= 0x12:
		return dexterity - 0x0f
	case dexterity >= 0x13 && dexterity <= 0x14:
		return 3
	case dexterity >= 0x15 && dexterity <= 0x17:
		return 4
	case dexterity >= 0x18 && dexterity <= 0x19:
		return 5
	}
	return 0
}

// WeaponBearer 是算武器數值時要讀的角色狀態。
type WeaponBearer struct {
	// BaseThac0Internal 是 record `+2Dh`。
	BaseThac0Internal uint8
	// Strength、ExceptionalStrength、Dexterity 是 record `+10h`／`+16h`／`+13h`。
	Strength            int
	ExceptionalStrength int
	Dexterity           int
	// AbilityBonusesEnabled 對應 record `+0AAh`。原版以它是否為零決定要不要
	// 套用兩個力量修正；這個 byte 的語意還沒閉合，所以照它的行為傳進來，
	// 不在這裡替它取名或推導。
	AbilityBonusesEnabled bool
	// LauncherPlus、AmmunitionPlus 是 record `+0F8h`／`+0FCh` 那兩件物品的
	// `+32h`；沒有那件物品時傳 nil。
	LauncherPlus    *int
	AmmunitionPlus  *int
	ClassBonusApplies bool // record `+2Eh` 是否為 2；為真時特定武器型別再加 1
}

// WeaponStats 是 overlay-25 entry 1 寫回 record 的四個欄位。
type WeaponStats struct {
	Thac0Internal uint8
	DamageCount   uint8
	DamageSides   uint8
	DamageBonus   int8
}

// classBonusWeaponTypes 是 record `+2Eh == 2` 時額外加 1 的武器型別，
// 取自 overlay-25 的四個比較（`24h`、`25h`，以及 `29h..2Ch`）。
// 那個欄位的語意還沒閉合，因此這裡只照位元組列出型別，不替它取名。
func classBonusWeaponType(itemType uint8) bool {
	switch {
	case itemType == 0x24, itemType == 0x25:
		return true
	case itemType >= 0x29 && itemType <= 0x2c:
		return true
	}
	return false
}

// WeaponCombatStats 依 spec 063 的十一個步驟算出裝上這件武器之後的數值。
//
// 沒有武器時原版那支直接返回，`+115h`／`+117h`／`+119h` 維持原值，所以
// 呼叫端要自己決定徒手要怎麼辦——這裡不補徒手傷害。
func WeaponCombatStats(table *ItemTypeTable, itemType uint8, weaponPlus int, bearer WeaponBearer) (WeaponStats, error) {
	if table == nil {
		return WeaponStats{}, fmt.Errorf("Pool weapon stats need an item type table")
	}
	entry, err := table.Entry(itemType)
	if err != nil {
		return WeaponStats{}, err
	}
	flags := entry.Flags()
	thac0 := int(bearer.BaseThac0Internal)
	strengthIndex := uint8(0)
	if bearer.AbilityBonusesEnabled {
		strengthIndex, err = StrengthTableIndex(bearer.Strength, bearer.ExceptionalStrength)
		if err != nil {
			return WeaponStats{}, err
		}
	}
	if flags&ItemTypeFlagDexterityToHit != 0 {
		thac0 += DexterityMissileAdjustment(bearer.Dexterity)
	}
	damageBonus := int(int8(entry.DamageBonus()))
	if flags&ItemTypeFlagStrengthBonuses != 0 {
		thac0 += StrengthHitAdjustment(strengthIndex)
		damageBonus += StrengthDamageAdjustment(strengthIndex)
	}

	modifier := weaponPlus
	if flags&ItemTypeFlagUsesAmmunition != 0 && bearer.AmmunitionPlus != nil {
		modifier += *bearer.AmmunitionPlus
	}
	if flags&ItemTypeFlagNeedsLauncher != 0 && bearer.LauncherPlus != nil {
		modifier += *bearer.LauncherPlus
	}
	damageBonus += modifier
	if bearer.ClassBonusApplies && classBonusWeaponType(itemType) {
		modifier++
	}
	thac0 += modifier

	count, sides := entry.Damage()
	return WeaponStats{
		Thac0Internal: uint8(thac0),
		DamageCount:   count,
		DamageSides:   sides,
		DamageBonus:   int8(damageBonus),
	}, nil
}
