package main

import (
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// applyMonsterGearStats 是開打時對怪物跑的 overlay-25 entry 7（`0BBEh`，spec 147）：
// overlay-10 `1F9Ch` → `1380h` 沿 combatant 串列對每一筆都叫它，怪物身上的物品串列
// （`+C8h`，spec 142）照樣認回武器、盔甲與盾。隊員那一側是同一支
// （applyPartyGearStats），差別只在 `+2Dh`：entry 7 不寫它，怪物照樣板。
//
// 蓋掉的是模板值：THAC0（`+110h`）、AC（`+111h`）、腳程（`+11Ch`）、兩種形態的傷害骰
// （`+115h..+11Ah`，有武器時第一種由武器決定），以及射程（overlay-13 `358Dh` 讀
// `+0CCh` 那件的型別表 `+0Ch`，spec 065）。攻擊次數（`+0A1h`／`+0A2h`）entry 7 不動。
//
// 沒有型別表時（只建了盤面的測試）照舊讀模板，不猜。
func (a *app) applyMonsterGearStats(state *tacticalState, index int, record gamepack.MonsterRecord) error {
	if a.itemTypes == nil {
		return nil
	}
	inventory := state.FoeItems[index]
	items := make([][]byte, 0, len(inventory))
	for _, item := range inventory {
		items = append(items, item.Raw)
	}
	runtime, err := gamepack.RecomputeMonsterCombatFields(record, items, a.itemTypes)
	if err != nil {
		return err
	}
	state.THAC0[index] = runtime.Raw[gamepack.CurrentThac0Offset]
	state.ArmorClass[index] = int(runtime.Raw[gamepack.InternalArmourClassOffset])
	state.BaseMovement[index] = runtime.Raw[gamepack.CurrentMovementOffset]
	primary := -1
	for slot := uint8(1); slot <= gamepack.MonsterAttackSlots; slot++ {
		damage, err := runtime.RuntimeAttackDamage(slot)
		if err != nil {
			return err
		}
		dice := combat.DamageDice{Count: damage.Count, Sides: damage.Sides, Bonus: damage.Bonus}
		state.AttackForms[index][slot-1] = dice
		// 畫面與法術用的那一組：第一種有骰子就用它，否則第二種（同 primaryAttackSlot）。
		if primary < 0 && (damage.Count != 0 || damage.Sides != 0) {
			primary = int(slot) - 1
		}
	}
	if primary < 0 {
		primary = gamepack.MonsterAttackSlots - 1
	}
	state.Damage[index] = state.AttackForms[index][primary]
	reach := 1
	// 認武器與隊員同一條規則（穿戴中、類別 0、最後一件贏，`0C76h`）。
	if weapon, ok := a.readiedWeapon(poolsave.Character{Inventory: inventory}); ok {
		reach = a.weaponAttackRange(weapon)
	}
	if index < len(state.AttackRange) {
		state.AttackRange[index] = reach
	}
	return nil
}
