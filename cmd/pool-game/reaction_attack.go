package main

import (
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
)

// 戰場上的朝向（原版 combatant 的 `+108h` 結構 `+9`）。反應攻擊的朝向窗以它為
// 基準（spec 059），所以要跟原版在同樣的時機改它（#58）：
//
//   - 部署：overlay-10 `13F4h..143Ch`，`DS:[2F0h + 隊伍朝向/2]`，敵方（`+10Eh == 1`）
//     再轉 180 度。`DS:2F0h` 起四個 byte 是 `07 02 03 06`（START.EXE 檔案位移
//     `30640 + 2F0h`；以 `DS:2880h` 的 `33 34 35 1F` 對過基準）。
//   - 走一步：玩家 overlay-08 `0BAEh`、怪物 overlay-09 `0AB3h` 都先以這一步的方向
//     呼叫 overlay-32 entry 14（`13Dh:0066h`）轉向，**然後**才呼叫 overlay-13 entry 6。
//   - 被攻擊：攻擊包裝 overlay-13 `1883h` 讓**目標**轉身面向攻擊者，方向是 `261Bh`
//     從 0 起第一個讓 overlay-31 `0579h`（`FacingArcContains`）成立的朝向。原版另有
//     兩個條件（目標 `+108h` 的 `+0Fh < 3`、呼叫端旗標為 0），語意還沒讀，這裡一律轉。
var deploymentFacings = [4]uint8{0x07, 0x02, 0x03, 0x06}

// deploymentFacing 是部署時的朝向。partyFacing 是地圖上的 0..3。
func deploymentFacing(partyFacing uint8, friendly bool) uint8 {
	facing := deploymentFacings[partyFacing&3]
	if !friendly {
		facing = (facing + 4) % combat.DirectionCount
	}
	return facing
}

// combatFacingTowards 是 `261Bh`：從 from 看出去，第一個把 to 收進弧內的朝向。
func combatFacingTowards(fromX, fromY, toX, toY uint8) (uint8, bool) {
	for direction := uint8(0); direction < combat.DirectionCount; direction++ {
		inside, err := combat.FacingArcContains(fromX, fromY, toX, toY, direction)
		if err == nil && inside {
			return direction, true
		}
	}
	return 0, false
}

// ensureFacings 讓 Facings 與盤面一樣長。正式戰鬥開打時就配好了，這一支是給
// 只組了部分狀態的測試盤面用的：補上的格子朝向 0。
func (state *tacticalState) ensureFacings() {
	for len(state.Facings) < len(state.Roster) {
		state.Facings = append(state.Facings, 0)
	}
}

// turnToFace 讓 index 轉身面向 other（攻擊包裝 `1883h`）。
func (state *tacticalState) turnToFace(index, other uint8) {
	state.ensureFacings()
	if int(index) >= len(state.Facings) || int(other) >= len(state.Roster) {
		return
	}
	from, to := state.Roster[index], state.Roster[other]
	if facing, ok := combatFacingTowards(from.X, from.Y, to.X, to.Y); ok {
		state.Facings[index] = facing
	}
}

// 否決攻擊的效果代碼（overlay-13 `1087h`，spec 112〈`DS:677Ch` 是誰立起來的〉）。
// 群組 1 問被打的一方，群組 0 問出手的一方。
const (
	vetoEffectNeedsLeader  = 0x19 // 要看 `DS:5CF0h` 指的那一位有沒有 18h
	vetoEffectInitiative   = 0x25 // runtime `+3`（先攻分數）大於 0 才否決
	vetoEffectAlways       = 0x47
	vetoEffectTargetsItems = 0x7E // 讀出手者當下目標的物品串列
	// overlay-25 entry 27 的兩個查詢碼（spec 059 閘門 4、5），有就不打。
	reactionBlockedEffectA = 0x4B
	reactionBlockedEffectB = 0x4A
)

// attackVetoed 是 overlay-13 `1087h`（mover 被打、opponent 出手）。
//
// 四個會立起 `677Ch` 的代碼裡，`25h`、`47h` 的條件讀得完整。`19h` 要看 `DS:5CF0h`
// 指到誰、`7Eh` 要看物品記錄 `+2Eh + i` 的 i 範圍，這兩處還沒讀，所以照 spec 059
// 契約 6 **保守處理**：身上有就當成否決，不預設放行。
func (state *tacticalState) attackVetoed(mover, opponent uint8) bool {
	if mover == opponent {
		return false
	}
	switch {
	case state.hasEffect(int(mover), vetoEffectNeedsLeader),
		state.hasEffect(int(mover), vetoEffectAlways),
		state.hasEffect(int(mover), vetoEffectInitiative) && int(mover) < len(state.Scores) && state.Scores[mover] > 0,
		state.hasEffect(int(opponent), vetoEffectTargetsItems):
		return true
	}
	return false
}

// disengageReactions 重現 overlay-13 entry 6：mover 朝 direction 踏出去之前，
// 因這一步而脫離鄰接的每個對手，照 spec 059 的閘門各打一次。回傳 mover 是否已經
// 不在場上——原版兩個呼叫端接著看 `+10Dh`，不在就不提交這一步、結束這一回合
// （overlay-08 `0C85h`、overlay-09 `0AD3h`）。
func (a *app) disengageReactions(state *tacticalState, mover, direction uint8) (bool, error) {
	state.ensureFacings()
	if int(mover) < len(state.Facings) {
		state.Facings[mover] = direction
	}
	snapshot, err := state.tacticalSnapshot()
	if err != nil {
		return false, err
	}
	side, ok := state.sideOf(mover)
	if !ok {
		return false, nil
	}
	leaving, err := combat.LeavingOpponentsAfterStep(snapshot, mover, direction, 1-side, state.sideOf)
	if err != nil {
		return false, err
	}
	for _, opponent := range leaving {
		if state.Roster[mover].FootprintClass == 0 {
			break
		}
		if !state.canReact(mover, opponent) {
			continue
		}
		if err := a.reactionAttack(state, opponent, mover); err != nil {
			return false, err
		}
	}
	return state.Roster[mover].FootprintClass == 0, nil
}

// canReact 是 spec 059 的閘門 2～6。
func (state *tacticalState) canReact(mover, opponent uint8) bool {
	if int(opponent) >= len(state.Roster) || state.Roster[opponent].FootprintClass == 0 {
		return false
	}
	var codes []uint8
	if int(opponent) < len(state.Effects) {
		for _, node := range state.Effects[opponent] {
			codes = append(codes, node.Code)
		}
	}
	if combat.IsReactionDisabled(codes) || state.attackVetoed(mover, opponent) ||
		state.hasEffect(int(opponent), reactionBlockedEffectA) ||
		state.hasEffect(int(opponent), reactionBlockedEffectB) {
		return false
	}
	// 朝向窗：對手目前朝向的前後兩格（`(base+6)..(base+10) mod 8`）。原版在
	// `+108h` 的 `+3 > 0` 或 `+0Fh == 0` 時直接視為成立，那兩格的語意還沒讀，
	// 這裡一律照弧判定。
	from, to := state.Roster[opponent], state.Roster[mover]
	for _, facing := range combat.ReactionFacings(state.Facings[opponent]) {
		inside, err := combat.FacingArcContains(from.X, from.Y, to.X, to.Y, facing)
		if err == nil && inside {
			return true
		}
	}
	return false
}

// reactionAttack 以 spec 059 選出的攻擊槽打一次。phase 計數（`+113h`／`+114h`）
// remake 沒有逐次扣的那一份，拿這一相位該揮幾下代替（hypothesis）。
func (a *app) reactionAttack(state *tacticalState, attacker, target uint8) error {
	var counts [2]uint8
	for slot := 0; slot < 2; slot++ {
		count, err := combat.AttacksThisPhase(state.AttackRates[attacker][slot], state.AttackPhase&1)
		if err != nil {
			return err
		}
		counts[slot] = count
	}
	pick := combat.SelectReactionAttackSlot(state.AttackRates[attacker][0], counts)
	dice := state.AttackForms[attacker][pick.Slot-1]
	if dice.Count == 0 || dice.Sides == 0 {
		dice = state.Damage[attacker]
	}
	swings := make([]combat.DamageDice, pick.PhaseCounts[pick.Slot-1])
	for index := range swings {
		swings[index] = dice
	}
	return a.resolveAttackSwings(state, attacker, target, swings)
}
