package main

import (
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 部署時就不在場的隊員（spec 061〈逐筆部署〉、spec 060〈生成之後的寫入者〉，issue #69）。
//
// overlay-10 `1A99h` 沿 `DS:5CF4h` 逐筆呼叫 `1609h`，不看記錄 `+10Dh`；放上了之後（exact）：
//
//	1D3A  26 80 BD 0D 01 00   +10Dh != 0 → 在場，下一筆
//	1D50  C6 85 88 5E 00      體型類別 5E88h[i] = 0（不佔格）
//	1D55  80 3E 9A 82 00      DS:829Ah != 0（決鬥）→ 不登記
//	1D67  26 80 7D 13 00      runtime +13h != 0（怪物）→ 不登記
//	1D95  FE 06 73 66         6673h 加一；663Ah[7n] = 原地形
//	1DDB  26 C6 45 07 1F      那一格地形寫 1Fh（屍體）
//	1DF1  89 8D 34 66 …       6634h[7n] = 記錄、6638h／6639h = X／Y
//
// `+10Dh` 在狀態寫成 {0, 1} 以外的值時清成 0（spec 084），remake 以狀態 4／5／6 判它；決鬥時
// overlay-07 `1AFFh` 把目前角色以外的人清成 0（spec 150）。

// partyMemberAbsent 是隊員記錄的 `+10Dh` 為 0：昏迷、倒地、死亡，或決鬥時不上場。
func (a *app) partyMemberAbsent(slot int) bool {
	if slot < 0 || slot >= len(a.state.Party) {
		return false
	}
	if partyMemberDown(a.state.Party[slot]) {
		return true
	}
	return !a.duelDeploys(slot)
}

// partyMemberDown 是昏迷（4）、倒地（5）與死亡（6）：戰後寫回的狀態（storeCombatHitPoints）。
// 石化（7）也是：overlay-12 `005Ah` 把 `+10Dh` 清成 0，戰鬥外只有戰後把狀態 0／1／3 的設回 1
// （spec 150），所以石化的人下一場照樣只擺屍體（spec 161）。
func partyMemberDown(member poolsave.Character) bool {
	return member.Status == 4 || member.Status == combat.DyingState || member.Status == combat.DeadState ||
		member.Status == gamepack.StonedState
}

// deployedCorpses 把部署時體型改成 0 的隊員登記進屍體表（`DS:6634h`，spec 155 的 Manual
// 瞄準讀的就是這一張），順序照部署的順序。remake 不改盤面地形（spec 155〈與原版不同〉），
// 屍體格由屍體表認。倒著的人帶著自己的狀態與生命值進場，回合收尾照樣推進倒地計時
// （overlay-08 `0868h` 沿同一條串列走，spec 062）。
func (a *app) deployedCorpses(state *tacticalState) {
	for index := 1; index < len(state.Roster) && index < len(state.PartySlot); index++ {
		slot := state.PartySlot[index]
		if state.Roster[index].FootprintClass != 0 || !a.partyMemberAbsent(slot) {
			continue
		}
		// 站起來時放回的體型（ov32 entry 21 讀記錄 `+6Ch AND 7`；隊員都是 1）。
		if index < len(state.Footprint) {
			state.Footprint[index] = 1
		}
		member := a.state.Party[slot]
		if partyMemberDown(member) {
			hp := member.CurrentHP
			if hp < 0 {
				hp = 0
			}
			state.HitPoints[index], state.States[index] = hp, member.Status
		}
		if a.duel {
			continue
		}
		state.Corpses = append(state.Corpses, index)
	}
}
