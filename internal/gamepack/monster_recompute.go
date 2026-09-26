package gamepack

import "fmt"

// RecomputeMonsterCombatFields 是開打時對怪物跑的那一次 overlay-25 entry 7（`0BBEh`）。
//
// overlay-10 開打初始化 `1ED6h` 在 `1F9Ch` 叫 `1380h`，沿 `DS:5CF4h` 的 combatant 串列對
// **每一筆**呼叫 entry 7（`13A7h`：`9A 43 00 0A 01`，spec 063）。隊員與怪物走同一支，
// 所以這裡與 RecomputeCombatFields 共用 recomputeFromBase，差別只有一個：entry 7 不寫
// `+2Dh`（整顆 overlay-25 沒有 `3C16h` 職業表的引用），怪物的 `+2Dh` 照樣板
// （獸人 41 不是一級戰士表的 40，dosgolem 收據）。
//
// items 是這一隻的物品串列（MONnITM.DAX，記錄 `+C8h`，spec 142），一件 63 bytes，
// 順序照串列。回傳的記錄裡：`+110h` THAC0、`+111h` AC、`+112h` 背後 AC、
// `+115h..+11Ah` 兩種形態的傷害骰（第一種有武器時由武器決定）、`+11Ch` 腳程、
// `+102h` 負重。
func RecomputeMonsterCombatFields(record MonsterRecord, items [][]byte, types *ItemTypeTable) (MonsterRecord, error) {
	if types == nil {
		return MonsterRecord{}, fmt.Errorf("Pool monster %q recompute needs an item type table", record.Name)
	}
	raw, err := recomputeFromBase(append([]byte(nil), record.Raw[:]...), items, types)
	if err != nil {
		return MonsterRecord{}, fmt.Errorf("Pool monster %q recompute: %w", record.Name, err)
	}
	result := record
	copy(result.Raw[:], raw)
	return result, nil
}

// RuntimeAttackDamage 是 entry 7 重算之後、執行期那一段（`+114h+n` 起）的第 n 種
// 形態骰子。樣板裡那一段是殘值（見 AttackDamage 的說明），只有重算過的記錄才讀它。
func (record MonsterRecord) RuntimeAttackDamage(slot uint8) (MonsterAttackDamage, error) {
	if slot < 1 || slot > MonsterAttackSlots {
		return MonsterAttackDamage{}, fmt.Errorf("Pool monster attack slot %d is outside 1..%d", slot, MonsterAttackSlots)
	}
	index := int(slot)
	return MonsterAttackDamage{
		Count: record.Raw[runtimeAttackBase+index],
		Sides: record.Raw[runtimeAttackBase+2+index],
		Bonus: int8(record.Raw[runtimeAttackBase+4+index]),
	}, nil
}
