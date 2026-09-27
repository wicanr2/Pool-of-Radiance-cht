package main

// 被圍攻的計數（runtime `+108h` 結構的 `+0Fh` 與 `+12h`，spec 059〈被圍攻〉，exact）。
//
// 寫入端（38 個 overlay 全掃 `les di,es:[di+108h]` 之後對 `+0Fh`／`+12h` 的存取）：
//
//	overlay-08 `01F2h`／`01FFh`  行動開頭（`01E4h`，玩家與電腦都走）兩格清 0
//	overlay-13 `0867h`／`0874h`  entry 5 提交一步：走的人兩格清 0
//	overlay-13 `1803h`／`1879h`  entry 14（`17F5h`）：目標 `+0Fh` 加一；`+12h` 加上目標朝向與
//	                             「目標看向攻擊者」的方向差（摺到 0..4），模 8
//
// entry 14 的呼叫端是一般出手的每一條路：瞄準 `2CF7h`、撞上去 overlay-08 `0DBEh`、電腦
// overlay-09 `0EA6h`、橫掃 `1042h`、`0630h` 的看守攻擊 `06E7h`；反應攻擊不叫。讀取端：
//
//	`1891h`  攻擊包裝：目標 `+0Fh < 3` 才轉身面向攻擊者
//	`15C4h..1600h`  攻擊核心：目標 `+0Fh > 1`、攻擊者到目標的方向等於目標朝向（站在正後方）、
//	         `+12h > 4` → 拿背後 AC `+112h`
//	`0B0Ah..0B1Eh`  反應攻擊：反應者先攻 `+3 > 0` 或 `+0Fh == 0` 就不看朝向窗
//	`25E3h`  entry 22 背刺（賊）：目標 `+0Fh > 1`（remake 沒有背刺）

// besetMark 是一格的 `+0Fh`（這一段時間被出手幾次）與 `+12h`（累計被迫轉過的方向差）。
type besetMark struct {
	count, turned uint8
}

// clearBeset 是 `01F2h`／`0867h`：行動開頭或提交一步，兩格清 0。
func (state *tacticalState) clearBeset(index uint8) {
	delete(state.beset, index)
}

// markAttacked 是 entry 14（`17F5h`）：目標計數加一，再把「目標朝向」到「看向攻擊者」的方向差
// （`261Bh`，自 0 起第一個弧內成立的方向；`1834h..1850h` 摺到 0..4）加進 `+12h`，模 8。
func (state *tacticalState) markAttacked(target, attacker uint8) {
	state.ensureFacings()
	if int(target) >= len(state.Facings) || int(attacker) >= len(state.Roster) {
		return
	}
	if state.beset == nil {
		state.beset = map[uint8]besetMark{}
	}
	mark := state.beset[target]
	mark.count++
	from, to := state.Roster[target], state.Roster[attacker]
	direction, ok := combatFacingTowards(from.X, from.Y, to.X, to.Y)
	if !ok {
		direction = 8 // 方向 8 恆真（spec 098）
	}
	difference := (int(direction) - int(state.Facings[target]) + 8) % 8
	if difference > 4 {
		difference = 8 - difference
	}
	mark.turned = uint8((int(mark.turned) + difference) % 8)
	state.beset[target] = mark
}

// besetTurns 是 `1891h`：目標被出手不到三次才轉身面向攻擊者（計數已含這一次）。
func (state *tacticalState) besetTurns(target uint8) bool {
	return state.beset[target].count < 3
}

// hitFromBehind 是 `15C4h..1600h`：目標 `+0Fh > 1`、攻擊者到目標的方向（`261Bh`）等於目標朝向、
// 目標 `+12h > 4`。成立就拿背後 AC（`1604h` 立 `[bp-13h]`）。
func (state *tacticalState) hitFromBehind(attacker, target uint8) bool {
	mark := state.beset[target]
	if mark.count <= 1 || mark.turned <= 4 || int(target) >= len(state.Facings) {
		return false
	}
	from, to := state.Roster[attacker], state.Roster[target]
	direction, ok := combatFacingTowards(from.X, from.Y, to.X, to.Y)
	return ok && direction == state.Facings[target]
}

// reactsWithoutFacing 是 `0B0Ah..0B1Eh`：反應者還沒行動（先攻 `+3 > 0`），或自從它上一次
// 行動、上一步之後還沒被出手過（`+0Fh == 0`），就不看朝向窗。
func (state *tacticalState) reactsWithoutFacing(reactor uint8) bool {
	if int(reactor) < len(state.Scores) && state.Scores[reactor] > 0 {
		return true
	}
	return state.beset[reactor].count == 0
}
