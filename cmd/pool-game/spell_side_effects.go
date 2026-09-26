package main

import "github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"

// 祝福、詛咒、急速、緩速掛上去之後在戰鬥裡作用的那幾處（spec 098〈模式 0Ah：效果怎麼掛上去〉、
// spec 112〈群組 10／18〉，issue #81）。規則在 internal/gamepack/spell_side_effects.go，
// 這裡只把盤面接上去。

// applySpellEffect 是 `08BCh` 的 `0A5Ah` 呼叫的 overlay-24 entry 20 過了免疫與豁免之後
// 那一段：同碼的舊節點剩得比較短（或新的不計時）就先摘，再在尾端掛新的。
func (state *tacticalState) applySpellEffect(index int, code uint8, duration, casterLevel int) {
	if index < 0 || index >= len(state.Effects) {
		return
	}
	if duration < 0 {
		duration = 0
	}
	if casterLevel < 0 {
		casterLevel = 0
	}
	if casterLevel > gamepack.EffectLevelMask {
		casterLevel = gamepack.EffectLevelMask
	}
	state.Effects[index] = gamepack.ApplySpellEffectNode(state.Effects[index],
		gamepack.NewEffectNode(code, uint16(duration), uint8(casterLevel), false))
}

// 命中擲骰時的群組 10／16 在 hit_roll_effects.go（#86）。

// dispatchRateEffects 是群組 18 派發一次：回這個人身上帶著哪幾個碼，順帶跑 `27h`
// 處理常式開頭那一段（最早的急速節點第一次被問到時老一歲）。
//
// 原版在每回合初始化（overlay-13 entry 1）與急速／緩速放出去的當下（overlay-22
// `281Fh..2835h`，對每一個留下的目標）都派發群組 18。放出去當下那一次的 `DS:6778h`
// 是上一次留下的值、結果也沒有寫回任何地方，看得見的只有老化那一下。
func (state *tacticalState) dispatchRateEffects(index int) gamepack.RoundRateEffects {
	if index < 0 || index >= len(state.Effects) {
		return 0
	}
	list, aged := gamepack.MarkHasteAged(state.Effects[index])
	state.Effects[index] = list
	if aged && state.PartyAged != nil {
		state.PartyAged(index)
	}
	return gamepack.RoundRateEffectsOf(list)
}

// attackRateThisRound 是第 slot 形態（0 起算）的攻擊次數編碼，套過這一回合初始化時
// 記下的群組 18。原版把套過的結果換算成次數存在 `+113h`／`+114h`，這一回合中途才
// 掛上的急速或緩速要到下一回合初始化才生效。
func (state *tacticalState) attackRateThisRound(index uint8, slot int) uint8 {
	rate := state.AttackRates[index][slot]
	if int(index) >= len(state.RoundRates) {
		return rate
	}
	return gamepack.AttackRateAfterEffects(rate, state.RoundRates[index])
}

// agePartyMember 是 `27h` 的 `0CB0h`（`26 FF 45 30`：記錄 `+30h` 加一）落在隊員身上：
// 寫回角色與角色庫。怪物的年齡不存檔，不必寫。
func (a *app) agePartyMember(state *tacticalState) func(index int) {
	return func(index int) {
		if index < 0 || index >= len(state.PartySlot) {
			return
		}
		party := state.PartySlot[index]
		if party < 0 || party >= len(a.state.Party) {
			return
		}
		a.state.Party[party].Age++
		syncTrainedLibraryCharacter(&a.state, a.state.Party[party])
	}
}
