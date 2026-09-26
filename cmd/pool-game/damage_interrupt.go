package main

import "strings"

// 受傷打斷施法（spec 098〈受傷打斷〉，issue #77）。
//
// 原版有兩支套用傷害的常式，兩支在傷害大於 0 的當下做同一件事：
//
//	overlay-13 entry 4（02FEh）   近戰／遠程攻擊（`1404h` 那一支攻擊常式的三個呼叫點
//	                               `14C1h`／`1732h`／`1796h`，全部 overlay 只有這三處）
//	  04E8  傷害 [bp+0Ah] == 0 → 跳過
//	  04F6  runtime(+108h) +1 = 0                         ; 這一回合不能施法
//	  0503  runtime +0 > 0 → 印 "lost a spell"（`02B0h`）、
//	        053A overlay-25 entry 16（14ECh）把那一格從記憶清掉、0547 runtime +0 = 0
//	overlay-24 entry 19（133Ah）  法術與效果的傷害（overlay-22 的處理常式 2 處、
//	                               overlay-12 的效果處理常式 7 處）
//	  137A  傷害（DS:6776h，豁免之後）== 0 → 整支跳過
//	  14FB  overlay-25 entry 28（2266h）扣生命值
//	  1500  DS:4954h == 5（戰鬥中）→ 150F runtime +1 = 0；runtime +0 > 0 →
//	        印 "lost a spell"（`12EAh`）、154F 14ECh、155C runtime +0 = 0
//
// remake 扣生命值的地方都從這裡過（resolveAttackSwings、applySpellDamage）。
// 吸取等級（applyEnergyDrainSpecialAttack）不過：原版 overlay-12 entry 78（211Eh）
// 直接 `dec +11Bh`，不清 +1——它只在一下命中造成傷害之後派發，那一下已經清過了。
//
// 玩家與 AI 走同一支，`+1` 清掉之後三處讀它：指令列的 "Cast "（overlay-08 `072Fh`）、
// AI 收法術清單（overlay-09 `0548h`）、開始施法的那一條放不放得出去（`012Eh`／`031Bh`
// 讀的其實是 `+0`，已經在這裡清成 0）。

// woundCombatant 是兩支傷害常式共同的那一段：傷害大於 0 的當下清 runtime `+1`，
// 開始施法還沒放出去的（runtime `+0`）當場丟失，從記憶清掉。
func (a *app) woundCombatant(state *tacticalState, target uint8, damage int) {
	if state == nil || damage <= 0 {
		return
	}
	index := int(target)
	if state.Casting.Wounded == nil {
		state.Casting.Wounded = map[int]bool{}
	}
	state.Casting.Wounded[index] = true
	spell := state.Casting.Pending[index]
	if spell == 0 {
		return
	}
	delete(state.Casting.Pending, index)
	if caster, ok := a.foeSpellcasterFor(state, target); ok {
		a.foeForgetSpell(state, caster, spell)
	}
	// overlay-13 `0529h`／overlay-24 `153Eh`：entry 20(記錄, "lost a spell", 0Ch, 1)。
	state.Casting.LostNotices = append(state.Casting.LostNotices,
		a.panelNotice(state, target, state.say(msgFoeLostSpell), noticeRowLost, true))
}

// announceLostSpells 把這一次傷害裡丟失的法術接在狀態列後面。原版先印傷害那一行，
// 再印 "lost a spell"（entry 4 的 `04DAh` 之後才到 `0529h`），所以接在後面。
func (a *app) announceLostSpells(state *tacticalState) {
	if state == nil || len(state.Casting.LostNotices) == 0 {
		return
	}
	notice := strings.Join(state.Casting.LostNotices, " ")
	state.Casting.LostNotices = nil
	previous := state.Status
	state.Status = strings.TrimSpace(previous + " " + notice)
	if a.statusLine == previous {
		a.statusLine = state.Status
	}
}
