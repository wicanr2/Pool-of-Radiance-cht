package main

import (
	"fmt"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// 效果系統在戰鬥裡的兩個掛勾點（spec 112，issue #86）：命中擲骰時的群組 10／16
// （overlay-24 entry 6 `0CB5h`），與掛效果之前的群組 9（overlay-24 entry 20 `1656h`）。
// 規則在 internal/gamepack/hit_roll_effects.go 與 effect_immunity.go，這裡只把盤面接上去。

const (
	// msgCastUnaffected 是 entry 20 的 `1648h`："is Unaffected"。
	msgCastUnaffected messageID = iota + 2860
)

func init() {
	for id, key := range map[messageID]string{
		msgCastUnaffected: "ui.castUnaffected",
	} {
		if existing, ok := messageKeys[id]; ok {
			panic(fmt.Sprintf("message id %d is already %q", id, existing))
		}
		messageKeys[id] = key
	}
}

// prayerAreaRadius 是 `014Dh` 的 `01EEh..01FBh`：碼是 31h 時半徑 6，其餘三個有範圍的碼 1。
const prayerAreaRadius = 6

// hitRollAfterEffects 是 entry 6 除了擲骰以外的部分。roll 是剛擲出的 d20。
//
//	0CBFh  0FCCh(攻擊者)：摘掉攻擊者身上所有 19h（隱形出手就現形）
//	0CD6h  d20 <= 1 → 落空，效果系統一支都不問
//	0CE9h  群組 10（攻擊者）→ 0CF6h 群組 16（目標）
//	0D28h  `DS:6780h` 有號小於 0 → 落空
//
// 回傳交給 `combat.ResolveHit` 的修正（調整後的命中骰減掉原本的分數），missed 為真時
// 不必比 AC。
func (state *tacticalState) hitRollAfterEffects(attacker, target, roll uint8) (modifier int, missed bool) {
	if int(attacker) < len(state.Effects) {
		state.Effects[attacker] = gamepack.DropInvisibility(state.Effects[attacker])
	}
	if roll <= 1 || int(attacker) >= len(state.Effects) || int(target) >= len(state.Effects) {
		return 0, false
	}
	base := gamepack.HitRollBase(roll)
	value, targetEffects := gamepack.HitRollEffects{
		Attacker:    state.hitRollCombatant(attacker),
		Target:      state.hitRollCombatant(target),
		Actor:       state.hitRollCombatant(state.Mover),
		AttackPhase: state.AttackPhase,
		TargetAim:   state.targetAimName(target),
		AreaNode: func(code uint8) (gamepack.EffectNode, bool) {
			return state.areaEffectNode(attacker, code, prayerAreaRadius)
		},
	}.Apply(base)
	state.Effects[target] = targetEffects
	if value < 0 {
		return 0, true
	}
	// `0D03h..0D25h`：出手的人那一邊的遭遇腳本修正（`+6E2h`／`+6E0h`，side_adjust.go）。
	return int(value) - int(base) + state.sideHitBonus(attacker), false
}

// hitRollCombatant 收一個戰鬥者在命中擲骰時被讀到的幾格。
func (state *tacticalState) hitRollCombatant(index uint8) gamepack.HitRollCombatant {
	var combatant gamepack.HitRollCombatant
	if int(index) < len(state.Effects) {
		combatant.Effects = state.Effects[index]
	}
	if int(index) < len(state.CreatureType) {
		combatant.CreatureType = state.CreatureType[index]
	}
	if int(index) < len(state.BodySize) {
		combatant.BodySize = state.BodySize[index]
	}
	if int(index) < len(state.RecordNames) {
		combatant.Name = state.RecordNames[index]
	}
	combatant.Side, _ = state.sideOf(index)
	if int(index) < len(state.Scores) {
		combatant.Score = state.Scores[index]
	}
	return combatant
}

// targetAimName 是 index 的 runtime `+0Ah`（它正在打的那一個）那一筆記錄的名字；
// 攻擊包裝在每一次出手時寫（resolveAttackSwings），AI 挑目標時也寫（setFoeTarget）。
func (state *tacticalState) targetAimName(index uint8) string {
	if int(index) >= len(state.FoeTargets) {
		return ""
	}
	aim := state.FoeTargets[index]
	if aim == 0 || int(aim) >= len(state.RecordNames) {
		return ""
	}
	return state.RecordNames[aim]
}

// rememberRecordName 記下那一格記錄 `+0` 的名字（`12h`／`1Ah`／`30h` 拿它比名字表）。
func (state *tacticalState) rememberRecordName(index int, name string) {
	if index < 0 || index >= len(state.Roster) {
		return
	}
	if len(state.RecordNames) < len(state.Roster) {
		names := make([]string, len(state.Roster))
		copy(names, state.RecordNames)
		state.RecordNames = names
	}
	state.RecordNames[index] = name
}

// areaEffectNode 是 `014Dh` 的第二條路（`019Bh..0292h`）：沿戰鬥者串列找帶著 code 的人，
// 以他為中心、半徑 radius 做近鄰查詢（overlay-31 `0138h:003Eh`，朝向 0FFh、體型是他的），
// 結果裡有 me 就用他的節點。找到第一個就停。
//
// 串列的順序取 roster 的順序（strong inference：`5CF4h` 由開打時依序建）；已經離場的
// （體型 0）不當中心——它不在戰術地圖上，近鄰查詢沒有格子可比。
func (state *tacticalState) areaEffectNode(me uint8, code uint8, radius int) (gamepack.EffectNode, bool) {
	for bearer := 1; bearer < len(state.Roster) && bearer < len(state.Effects); bearer++ {
		at, ok := state.Effects[bearer].IndexOf(code)
		if !ok || state.Roster[bearer].FootprintClass == 0 {
			continue
		}
		snapshot, err := state.tacticalSnapshot()
		if err != nil {
			return gamepack.EffectNode{}, false
		}
		cells, err := combat.NearbyCells(combat.NearbyRequest{
			Map: snapshot.Map, Classes: snapshot.Classes, Cells: snapshot.Cells,
			Class: state.Roster[bearer].FootprintClass, Facing: combat.DirectionUnset,
			Budget: uint16(radius),
			BaseX:  state.Roster[bearer].X, BaseY: state.Roster[bearer].Y,
		})
		if err != nil {
			continue
		}
		for _, cell := range cells {
			if cell.CombatantIndex == me {
				return state.Effects[bearer][at], true
			}
		}
	}
	return gamepack.EffectNode{}, false
}

// unaffectedBySpellEffect 是 entry 20 的 `166Fh..1687h`：群組 9 把這個碼擋掉就印
// "is Unaffected"、不掛。damageFlags 是 `DS:6777h`——`08BCh` 在沒有傷害時寫 0，
// 目前走到這裡的四條路（模式 0Ah、魅惑、定身）都沒有傷害。
func (a *app) unaffectedBySpellEffect(state *tacticalState, index uint8, code uint8,
	casterLevel int) bool {
	if int(index) >= len(state.Effects) {
		return false
	}
	if !gamepack.SpellEffectImmunity(state.Effects[index], code, 0, casterLevel, a.rollDice) {
		return false
	}
	// overlay-24 `16B0h`：entry 20(目標, "is Unaffected", 0Ah, 1)。
	a.tacticalStatus(state, a.panelNotice(state, index, state.say(msgCastUnaffected), noticeRowPanel, true))
	return true
}
