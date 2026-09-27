package main

import "github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"

// 殺了目標、還有剩的攻擊次數時回合繼續（spec 160，issue #109）。
//
// 原版每回合初始化（overlay-13 `0000h..006Eh`）替每一格數好兩個形態這一回合揮幾下，寫在
// 記錄的 `+113h`／`+114h`，並把 runtime `+8` 清成 0。攻擊核心 `1404h` 一開頭把 `+8` 寫 1
// （`1440h`），每揮一下先扣那一格的次數（`16A3h`），目標 `+10Dh` 變 0 就停手（`1755h`）；
// `1799h..17C2h` 只要兩格任一格還大於 0 就把完成旗標改回 0，不呼叫 entry 34，回合繼續：
//
//	玩家  指令迴圈（overlay-08 `036Ch..05C8h`）看完成旗標為 0 就再畫一次指令列；撞上去的
//	      移動迴圈（`0A13h..0CF6h`）不看完成旗標、只看腳程，腳程沒被 entry 34 清掉就接著走
//	電腦  entry 5 `0F2Ch..0F37h` 目標已倒下就離開接近迴圈，回傳 0；entry 1 `01B2h..01FBh`
//	      只要 runtime `+3` 還大於 0、`37B8h` 挑得到目標，就再叫一次 entry 5
//
// remake 不在回合初始化先數，而是在這一回合第一次出手時照 volleySwings 現數（中間沒有
// 東西會改它：群組 18 在回合初始化就算好了，換武器走 recountSwings）；之後的出手照剩下的打。

// remainingSwings 是這一次出手要揮的那幾下：這一回合還沒出過手就是 fresh（剛數的），出過手
// 就照 `+113h`／`+114h` 剩下的次數重排（攻擊區段由第二形態倒數到第一，同 volleySwings）。
func (state *tacticalState) remainingSwings(mover uint8, fresh []combat.DamageDice, form2 int) ([]combat.DamageDice, int) {
	left, ok := state.swingsLeft[mover]
	if !ok || int(mover) >= len(state.AttackForms) {
		return fresh, form2
	}
	forms := state.AttackForms[mover]
	var swings []combat.DamageDice
	counts := [2]int{0, 0}
	for slot := len(forms); slot >= 1; slot-- {
		dice := forms[slot-1]
		if slot > len(left) || dice.Count == 0 || dice.Sides == 0 {
			continue
		}
		for swing := uint8(0); swing < left[slot-1]; swing++ {
			swings = append(swings, dice)
		}
		counts[slot-1] = int(left[slot-1])
	}
	return swings, counts[1]
}

// spendSwings 在攻擊核心回來之後扣掉實際揮出去的次數（`lastSwings`，第二形態先扣），
// 回傳回合是不是繼續（`1799h..17C2h`：任一格還大於 0）。
func (state *tacticalState) spendSwings(mover uint8, swings []combat.DamageDice, form2 int) bool {
	left := [2]uint8{uint8(len(swings) - form2), uint8(form2)}
	used := state.lastSwings
	for slot := 1; slot >= 0 && used > 0; slot-- {
		spent := int(left[slot])
		if spent > used {
			spent = used
		}
		left[slot] -= uint8(spent)
		used -= spent
	}
	if state.swingsLeft == nil {
		state.swingsLeft = map[uint8][2]uint8{}
	}
	state.swingsLeft[mover] = left
	return left[0]+left[1] > 0
}

// recountSwings 是回合中途再叫一次 overlay-13 entry 8（`0D29h`）：玩家的物品選單回來之後
// （overlay-08 `03FDh`）、電腦 entry 5 換武器之後（overlay-09 `0DFCh` → entry 9 `1813h`）。
// 它只動第一形態 `+113h`（exact）：
//
//	0D36  舊 = +113h；0D49 +113h = +0A1h；0DC8 新 = 現數（射擊再以彈藥封頂）
//	0E09  runtime +8 == 0 → 寫回新
//	0E18  新 < 舊 → 寫回新
//	0E34  新 ≥ 舊 × 2 → 不寫（+113h 留著 0D49 寫的 +0A1h）
//	0E41  射擊（entry 43 且 entry 45）→ 不寫；否則寫回新
//
// 還沒出過手（`+8` 為 0）的不必動：下一次出手本來就照現數。
func (a *app) recountSwings(state *tacticalState, mover uint8) error {
	left, ok := state.swingsLeft[mover]
	if !ok || int(mover) >= len(state.AttackRates) {
		return nil
	}
	gear, items, _, err := a.missileGear(state, mover)
	if err != nil {
		return err
	}
	fresh, form2, err := a.volleySwings(state, mover, gear, items)
	if err != nil {
		return err
	}
	count, old := uint8(len(fresh)-form2), left[0]
	switch {
	case count < old:
		left[0] = count
	case int(count) >= 2*int(old) || (gear.Ranged && gear.CanFire):
		left[0] = state.AttackRates[mover][0]
	default:
		left[0] = count
	}
	state.swingsLeft[mover] = left
	return nil
}
