package main

import (
	"fmt"
	"strings"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 中毒（spec 153，issue #106）：有毒的怪物第二形態咬中、豁免失敗就當場死亡並掛 `37h`；緩毒術把人
// 暫時扶起來，到期時還中著毒就再死一次；神殿與編號 58 解毒。規則在 internal/gamepack/poison_effects.go。

const (
	// msgEffectPoisoned 是 overlay-12 `153Dh` "is Poisoned" 接 `1549h` "is killed"（`1553h`）。
	msgEffectPoisoned messageID = iota + 4300
	// msgEffectDiesFromPoison 是 `077Ah` "dies from poison"（`16h` 的收尾）。
	msgEffectDiesFromPoison
	// msgEffectGetsBackUp 是 overlay-24 `185Ch` "gets back up"（entry 22，`+10Eh` 不是 1）。
	msgEffectGetsBackUp
	// msgEffectStandsUpAndGrins 是 overlay-24 `1848h` "stands up and grins"（`+10Eh` 是 1）。
	msgEffectStandsUpAndGrins
	// msgEffectUnaffected 是 overlay-12 `2527h` "is unaffected"（群組 6 的 `5Bh`）。
	msgEffectUnaffected
)

func init() {
	for id, key := range map[messageID]string{
		msgEffectPoisoned:         "ui.effectPoisoned",
		msgEffectDiesFromPoison:   "ui.effectDiesFromPoison",
		msgEffectGetsBackUp:       "ui.effectGetsBackUp",
		msgEffectStandsUpAndGrins: "ui.effectStandsUpAndGrins",
		msgEffectUnaffected:       "ui.effectUnaffected",
	} {
		if existing, ok := messageKeys[id]; ok {
			panic(fmt.Sprintf("message id %d is already %q", id, existing))
		}
		messageKeys[id] = key
	}
}

// poisonSpecialAttack 是群組 3 裡的 `40h 41h 42h 46h`（overlay-12 `1553h`）：第二攻擊形態成功造成傷害、
// 目標還在場上（overlay-13 `1735h..174Dh`）之後，攻擊者身上每帶一個毒碼就讓目標擲一次類別 0 的
// 豁免；失敗就掛 `37h` 並跑 `005Ah(目標, 6, "is killed")`。回傳目標有沒有因此死掉。
func (a *app) poisonSpecialAttack(state *tacticalState, attacker, target uint8) bool {
	if int(attacker) >= len(state.Effects) || int(target) >= len(state.Effects) {
		return false
	}
	killed := false
	for _, modifier := range gamepack.PoisonAttackSaves(state.Effects[attacker]) {
		if a.savedAgainstCategory(state, target, 0, modifier) {
			continue
		}
		state.Effects[target] = state.Effects[target].Append(gamepack.NewPoisonNode())
		a.tacticalStatus(state, fmt.Sprintf(a.text(msgEffectPoisoned), a.combatantName(state, target)))
		if a.poisonKill(state, target) {
			killed = true
		}
	}
	return killed
}

// poisonKill 是 overlay-12 `005Ah(記錄, 6, 訊息)` 落在盤面上：狀態已經是 6／7／8 就不動
// （`008Ch..009Bh`）；否則 `+10Ch = 6`、`+10Dh = 0`、生命 0，離場（`00E6h` overlay-32 entry 20）。
// 回傳這一下有沒有把人弄死。`00C2h`／`00D0h` 的 entry 13 與群組 13 走 combatantDown（spec 155）。
func (a *app) poisonKill(state *tacticalState, target uint8) bool {
	index := int(target)
	if index >= len(state.HitPoints) || index >= len(state.States) || alreadyGone(state.States[index]) {
		return false
	}
	if member := a.partyMemberAt(state, target); member != nil {
		member.Status, member.CurrentHP = combat.DeadState, 0
		syncTrainedLibraryCharacter(&a.state, *member)
	}
	state.HitPoints[index] = 0
	state.rememberFootprint(index)
	state.Roster[index].FootprintClass = 0
	state.Scores[index], state.States[index] = 0, combat.DeadState
	if index < len(state.DyingCounters) {
		state.DyingCounters[index] = 0
	}
	a.combatantDown(state, index, deadOverkill)
	return true
}

// alreadyGone 是 `005Ah` 的 `008Ch..009Bh`：狀態 7、6、8 就不再寫。
func alreadyGone(status uint8) bool {
	return status == 6 || status == 7 || status == 8
}

// poisonTeardown 是戰場上 `0Fh`／`16h`／`4Eh` 到期時的收尾（隊員與怪物同一支，三支常式都不看
// `+10Eh`）。
func (a *app) poisonTeardown(state *tacticalState, index int, node gamepack.EffectNode) {
	if !node.NeedsTeardown() || !gamepack.IsPoisonTeardownEffect(node.Code) ||
		index < 0 || index >= len(state.Effects) || index >= len(state.HitPoints) {
		return
	}
	before := state.HitPoints[index]
	result := gamepack.PoisonTeardownOf(node, state.Effects[index], before)
	state.Effects[index] = result.Effects
	if result.Drained {
		// `0622h..0636h`：`6777h = 0`、entry 19(記錄, 1, 0, 0)——受傷打斷照走。
		state.HitPoints[index] = result.HitPoints
		a.woundCombatant(state, uint8(index), before-result.HitPoints)
	}
	if result.DiesFromPoison {
		a.tacticalStatus(state, fmt.Sprintf(a.text(msgEffectDiesFromPoison),
			a.combatantName(state, uint8(index))))
		a.poisonKill(state, uint8(index))
	}
	if result.Recover && !a.poisonRecover(state, index, state.HitPoints[index]) {
		state.Effects[index] = gamepack.PoisonRecoveryRetry(state.Effects[index], node.Magnitude())
	}
}

// poisonRecover 是 overlay-24 entry 22（`1869h`）在戰場上：overlay-32 entry 21（`1091h`）把這一位的
// 體型放回原地，拿方向 8 探自己的佔格（`1104h` 叫 `0CB9h`）；撞到別人、落在盤面外、或那一類地形是
// FFh（`110Bh..1123h`）就站不起來。站得起來：`+10Ch = 0`、`+10Dh = 1`、生命 = hitPoints，印
// "gets back up"（`+10Eh` 是 1 的印 "stands up and grins"）。
func (a *app) poisonRecover(state *tacticalState, index, hitPoints int) bool {
	if index <= 0 || index >= len(state.Roster) || index >= len(state.Footprint) ||
		index >= len(state.States) || index >= len(state.HitPoints) {
		return false
	}
	previous := state.Roster[index].FootprintClass
	if state.Footprint[index] != 0 {
		state.Roster[index].FootprintClass = state.Footprint[index]
	}
	if !state.standsOnItsOwnCells(index) {
		state.Roster[index].FootprintClass = previous
		return false
	}
	state.States[index], state.HitPoints[index] = 0, hitPoints
	if index < len(state.DyingCounters) {
		state.DyingCounters[index] = 0
	}
	state.forgetCorpse(index)
	message := msgEffectGetsBackUp
	if side, ok := state.sideOf(uint8(index)); ok && side == 1 {
		message = msgEffectStandsUpAndGrins
	}
	if member := a.partyMemberAt(state, uint8(index)); member != nil {
		member.Status, member.CurrentHP = 0, hitPoints
		syncTrainedLibraryCharacter(&a.state, *member)
	}
	a.tacticalStatus(state, fmt.Sprintf(a.text(message), a.combatantName(state, uint8(index))))
	return true
}

// standsOnItsOwnCells 是 overlay-32 `1104h..1123h`：`0CB9h(記錄, 8)` 回的佔用者非 0、地形類別是 0
// （盤面外），或 `DS:2758h[類別 × 4]` 是 FFh，都算站不上去。
func (state *tacticalState) standsOnItsOwnCells(index int) bool {
	snapshot, err := state.tacticalSnapshot()
	if err != nil {
		return false
	}
	occupant, class, err := combat.ProbeDestination(snapshot, uint8(index), combat.DirectionAny)
	if err != nil || occupant != 0 || class == combat.OffBoardDestinationClass {
		return false
	}
	record, err := combat.CellClassAt(state.Classes, class)
	return err == nil && record.EntryThreshold != combat.SpellRayWallClass
}

// slowPoisonAftermath 是緩毒術在 `08BCh` 之後那兩步（overlay-22 `18BBh..18E5h`），對表上第一格：
//
//	18CDh  entry 1(4Eh, 記錄, NULL, 模式 1)  ; 4Eh 的常式 `19F5h`：entry 22 扶起來，站不起來就重掛
//	18E5h  entry 10(記錄, 0Fh, 0Ah, FFh, 1)
//
// `19F5h` 讀的節點是 NULL，`1A1Fh` 的 `26 8A 45 03` 讀到的是 `0000:0003`（中斷向量表），那個 byte
// 只會存進重掛的 `4Eh` 的 `+3`；remake 存 FFh（spec 153〈卡點〉）。
func (a *app) slowPoisonAftermath(state *tacticalState, index uint8) {
	if int(index) >= len(state.Effects) || int(index) >= len(state.HitPoints) {
		return
	}
	if !a.poisonRecover(state, int(index), state.HitPoints[index]) {
		state.Effects[index] = gamepack.PoisonRecoveryRetry(state.Effects[index], gamepack.EffectUndispellable)
	}
	state.Effects[index] = state.Effects[index].Append(gamepack.NewPoisonDrainNode())
}

// slowPoisonAftermathInCamp 是同兩步在戰鬥外：overlay-32 entry 21 在 `DS:4954h != 5` 時直接回 1
// （`1097h..109Eh`），所以一定站得起來。回傳 entry 22 印的那一句。
func (a *app) slowPoisonAftermathInCamp(member *poolsave.Character) string {
	list := combatEffects(member.Effects)
	member.Status = 0
	list = list.Append(gamepack.NewPoisonDrainNode())
	member.Effects = storedEffects(list)
	syncTrainedLibraryCharacter(&a.state, *member)
	return fmt.Sprintf(a.text(msgEffectGetsBackUp), strings.TrimSpace(member.Name))
}

// mapPoisonTeardown 是 `0Fh`／`16h`／`4Eh` 在地圖上到期（overlay-20 entry 4 → overlay-24 entry 2）。
// 戰鬥外 overlay-32 entry 21 一定回 1，`4Eh` 一定站得起來。
func (a *app) mapPoisonTeardown(member *poolsave.Character, node gamepack.EffectNode,
	list gamepack.EffectList) gamepack.EffectList {
	if member == nil || !node.NeedsTeardown() || !gamepack.IsPoisonTeardownEffect(node.Code) {
		return list
	}
	result := gamepack.PoisonTeardownOf(node, list, member.CurrentHP)
	member.CurrentHP = result.HitPoints
	name := strings.TrimSpace(member.Name)
	if result.DiesFromPoison {
		a.statusLine = fmt.Sprintf(a.text(msgEffectDiesFromPoison), name)
		if !alreadyGone(member.Status) {
			member.Status, member.CurrentHP = combat.DeadState, 0
		}
	}
	if result.Recover {
		member.Status = 0
		a.statusLine = fmt.Sprintf(a.text(msgEffectGetsBackUp), name)
	}
	syncTrainedLibraryCharacter(&a.state, *member)
	return result.Effects
}

// neutralizesPoison 是編號 58（overlay-22 `2E02h`）的第一支：表上第一格身上有 `37h` 就用 entry 15
// 摘掉（印 "is Cured"）、`677Dh` 立著摘 `16h`（它的收尾 `078Bh` 順手摘 `0Fh`），整支結束——不治療、
// 不解病、不救活。沒中毒回 false，照舊走解病與治療。
func neutralizesPoison(list gamepack.EffectList) (gamepack.EffectList, bool) {
	return gamepack.NeutralizePoison(list)
}

// neutralizeOnBoard 是編號 58 在戰場上，整支照 `2E02h` 的先後：表上第一格（沒挑過是施法者自己）中毒就
// 只解毒（`2E08h..2E3Ch`）；否則 `2E3Fh` 走解病鏈 `225Bh`，解到東西就結束；什麼都沒解到才 `2E4Eh`
// 以 Roll(1, 4) ＋ 8 走 entry 21（旗標 0）。串列改的是盤面那一份，收場時 storeCombatEffects 寫回。
// 回 true 讓 castSpell 不再往下走（spec 155）。heal 是 CastEffect 先擲好的那一個 Roll(1, 4) ＋ 8
// ——remake 的施法在放出去時就擲，原版只在治療那一支才擲（spec 155〈與原版不同〉）。
func (a *app) neutralizeOnBoard(state *tacticalState, targets spellTargets, option castOption, heal int) bool {
	first := state.Mover
	if target, chosen := targets.first(); chosen {
		first = target
	}
	if int(first) >= len(state.Effects) {
		return false
	}
	if list, ok := neutralizesPoison(state.Effects[first]); ok {
		state.Effects[first] = list
		a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastCured), a.combatantName(state, first), 1))
		return true
	}
	if list, cured := gamepack.CureDiseaseChain(state.Effects[first]); cured {
		state.Effects[first] = list
		a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastCured), a.combatantName(state, first), 1))
		return true
	}
	before := 0
	if int(first) < len(state.HitPoints) {
		before = state.HitPoints[first]
	}
	if a.healOnBoard(state, first, heal) {
		a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastHealed), a.combatantName(state, state.Mover),
			option.Label, state.HitPoints[first]-before))
	}
	return true
}
