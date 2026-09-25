package main

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 戰鬥中施法（spec 098）。C 開清單、上下挑、Enter 施、ESC 取消。
//
// **只列得出已經讀過處理常式的法術**（`SpellIsImplemented`）。沒讀過的不列，
// 而不是列了之後失敗——玩家看得到的清單就是實際做得到的事。
//
// **施法時間與瞄準照原版**（spec 098〈施法時間與打斷〉〈收目標〉，spell_targets.go）：
// 施法時間不為 0 的先「開始施法」，輪到下一次才瞄準、放出去；瞄準照參數表 `+6`
// 分成挑一個、逐個挑 (模式 & 3) + 1 個、挑一點收範圍三種（overlay-13 `20AEh`）。
// 模式 0Ah（整邊）與 8（閃電束的射線）還是 remake 自己決定作用在誰身上。

type castOption struct {
	// Slot 是記憶陣列裡的位置，施完要清掉那一格。
	Slot int
	// ID 是 1-based 的法術編號。
	ID uint8
	// Label 是給玩家看的名字。
	Label string
}

// openCastMenu 列出目前這個角色記著、而且處理常式已經讀出來的法術。
func (a *app) openCastMenu() {
	state := a.tactical
	if state == nil || state.Mover == 0 {
		return
	}
	index, ok := a.moverPartyIndex(state.Mover)
	if !ok {
		a.tacticalStatus(state, a.text(msgCastNotACaster))
		return
	}
	// overlay-08 `072Fh`：runtime +1（這一回合還能施法）為 0 的，指令列不接 "Cast "。
	// 受過傷、沉默或咳嗽都會清它（spec 096〈entry 4〉）。
	if state.castingDisrupted(int(state.Mover)) {
		a.tacticalStatus(state, a.text(msgCastCannotNow))
		return
	}
	options := a.spellOptionsFor(a.state.Party[index])
	if len(options) == 0 {
		a.tacticalStatus(state, a.text(msgCastNothingReady))
		return
	}
	a.castOptions, a.castCursor, a.castOpen = options, 0, true
}

// spellOptionsFor 列出這個人**記著、記完了、而且處理常式已經讀出來**的法術。
// 戰鬥中與探索時共用同一份清單：看得到的就是施得出來的。
func (a *app) spellOptionsFor(member poolsave.Character) []castOption {
	options := make([]castOption, 0, gamepack.MemorisedSpellSlots)
	for slot, value := range member.Memorised {
		// 還沒記完的（第 7 位還在）施不出來，要休息過（spec 070）。
		if !gamepack.MemorisedSpellIsReady(value) {
			continue
		}
		id := value & 0x7f
		if a.spellCaster == nil || !a.spellCaster.Implemented(id) {
			continue
		}
		label := fmt.Sprintf("%d", id)
		if a.spells != nil {
			if spell, err := a.spells.catalogue.SpellByID(id); err == nil {
				label = spell.Text
			}
		}
		options = append(options, castOption{Slot: slot, ID: id, Label: label})
	}
	return options
}

// moverPartyIndex 把戰場上的位置換回隊伍索引。
func (a *app) moverPartyIndex(mover uint8) (int, bool) {
	if int(mover) >= len(a.tactical.PartySlot) {
		return 0, false
	}
	index := a.tactical.PartySlot[mover]
	if index < 0 || index >= len(a.state.Party) {
		return 0, false
	}
	return index, true
}

// castInput 處理施法清單的按鍵。
func (a *app) castInput() error {
	switch {
	case a.justPressed(ebiten.KeyEscape):
		a.castOpen = false
	case a.justPressed(ebiten.KeyArrowUp):
		a.castCursor = (a.castCursor + len(a.castOptions) - 1) % len(a.castOptions)
	case a.justPressed(ebiten.KeyArrowDown):
		a.castCursor = (a.castCursor + 1) % len(a.castOptions)
	case a.justPressed(ebiten.KeyEnter), a.justPressed(ebiten.KeySpace):
		return a.resolveCast()
	}
	return nil
}

// beginCastTargeting 對「挑一個目標」那幾種模式開出選目標的步驟。
//
// 原版是 overlay-13 `1E09h` 進到 `352Ch` 的互動介面，選單列寫著
// `Aim: Next Prev Manual Center Exit`，而且允許打自己人
//（`Attack Ally:`）。這裡做的是同一件事：N／P 或左右鍵換人、`M` 進格子
// 游標、`C` 把視窗捲到目標身上、Enter 確定，預設停在繞得過去的最近敵人
// 身上——**選項列那五個字母全部接上了**（spec 127）。
func (a *app) beginCastTargeting(option castOption) bool {
	state := a.tactical
	candidates := make([]uint8, 0, len(state.Roster))
	for index := 1; index < len(state.Roster); index++ {
		if state.Roster[index].FootprintClass == 0 {
			continue
		}
		candidates = append(candidates, uint8(index))
	}
	if len(candidates) == 0 {
		return false
	}
	cursor := 0
	if target, found := state.nearestReachableOpposing(state.Mover); found {
		for index, candidate := range candidates {
			if candidate == target {
				cursor = index
				break
			}
		}
	}
	a.castTargets, a.castTargetCursor, a.castPending = candidates, cursor, option
	a.castTargeting = true
	return true
}

// beginAimedAttack 是 A 鍵：拿現在裝備的武器瞄一個目標打。
//
// 原版把它掛在 `View Aim` 指令上（overlay-08 `05E1h` 起的指令字串），
// 射程來自物品型別表的 `+0Ch`（spec 065），距離則是
// **TraceMovement 的成本除以二**（spec 098，overlay-25 `2591h`）。
// 地形擋住走不到就打不到，與原版一樣。
func (a *app) beginAimedAttack() bool {
	state := a.tactical
	if state == nil || state.Mover == 0 {
		return false
	}
	if !a.beginCastTargeting(castOption{Label: a.text(msgAimAttack)}) {
		return false
	}
	a.castTargetingAttack = true
	return true
}

// weaponAttackRange 是這件武器搆得到幾格（spec 065）。沒有型別表、型別查不到
// 或記錄短了一截都當成相鄰一格——原版把型別表 `+0Ch` 的 0 與 FFh 都當 1。
func (a *app) weaponAttackRange(weapon poolsave.Item) int {
	if a.itemTypes == nil || len(weapon.Raw) <= itemTypeOffset {
		return 1
	}
	entry, err := a.itemTypes.Entry(weapon.Raw[itemTypeOffset])
	if err != nil {
		return 1
	}
	return entry.AttackRange()
}

// moverAttackRange 是現在這個角色打得到幾格。**它與敵方 AI 讀同一欄**
// （`tacticalState.AttackRange`，建 roster 時填）：兩份真相會讓「玩家瞄得到
// 但同一把武器在 AI 手上搆不到」這種不對稱悄悄出現，而那在畫面上看不出來。
func (a *app) moverAttackRange() int {
	if a.tactical == nil {
		return 1
	}
	return a.tactical.attackRangeOf(a.tactical.Mover)
}

// resolveAimedAttack 對挑中的目標打一次。超出射程就不打，也不消耗回合。
func (a *app) resolveAimedAttack(target uint8) error {
	state := a.tactical
	distance, reachable := state.tacticalRange(state.Mover, target)
	if !reachable {
		a.tacticalStatus(state, fmt.Sprintf(a.text(msgAimBlocked), target))
		return nil
	}
	if reach := a.moverAttackRange(); distance > reach {
		a.tacticalStatus(state, fmt.Sprintf(a.text(msgAimOutOfRange), target, distance, reach))
		return nil
	}
	if same, err := state.sameSide(state.Mover, target); err != nil {
		return err
	} else if same {
		// 原版允許打自己人（`Attack Ally:` 會先問一句），那一句還沒讀，
		// 所以這裡直接擋下來。
		state.Status = state.say(msgStatusBlocked)
		return nil
	}
	if err := a.resolveTacticalAttack(state, target); err != nil {
		return err
	}
	state.endTurnAfterAction(a.rollDice)
	if state.Finished {
		return a.finishCombat(state.Outcome)
	}
	return nil
}

// castTargetingInput 處理選目標那一步的按鍵。
func (a *app) castTargetingInput() error {
	// Manual 的格子游標自己吃掉按鍵（spec 127）。
	if handled, err := a.manualAimInput(); handled || err != nil {
		return err
	}
	switch {
	case a.justPressed(ebiten.KeyM):
		// 原版瞄準列的第三項。進去之後游標停在目前挑到的那一格。
		a.beginManualAim()
	case a.justPressed(ebiten.KeyC):
		// 原版瞄準列的第四項（overlay-13 `3714h`）：把 7×7 視窗捲到
		// 讓目前這個目標落在正中央（餘裕 0），選到誰不變。
		a.centreOnTarget()
	case a.justPressed(ebiten.KeyEscape):
		if a.castAim != nil && !a.castTargetingAttack {
			// 施法的瞄準按 Exit：`1E09h` 回 0，交給 `20AEh` 的計數或 Abort Spell。
			return a.cancelSpellPick()
		}
		a.castTargeting, a.castTargetingAttack, a.castManual = false, false, false
	case a.justPressed(ebiten.KeyP), a.justPressed(ebiten.KeyArrowLeft),
		a.justPressed(ebiten.KeyArrowUp):
		a.castTargetCursor = (a.castTargetCursor + len(a.castTargets) - 1) % len(a.castTargets)
	case a.justPressed(ebiten.KeyN), a.justPressed(ebiten.KeyArrowRight),
		a.justPressed(ebiten.KeyArrowDown):
		a.castTargetCursor = (a.castTargetCursor + 1) % len(a.castTargets)
	case a.justPressed(ebiten.KeyEnter), a.justPressed(ebiten.KeySpace):
		target := a.castTargets[a.castTargetCursor]
		if a.castAim != nil && !a.castTargetingAttack {
			// 施法照 `20AEh` 收目標：射程、重複與還要挑幾個都在那一支判斷，
			// 過不了就停在瞄準裡。
			cell := a.tactical.Roster[target]
			return a.confirmSpellCell(int(cell.X), int(cell.Y), target)
		}
		a.castTargeting = false
		if a.castTargetingAttack {
			a.castTargetingAttack = false
			return a.resolveAimedAttack(target)
		}
		return a.finishCast(a.castPending, target, true)
	}
	return nil
}

// resolveCast 施出選中的那一條。
func (a *app) resolveCast() error {
	state := a.tactical
	a.castOpen = false
	if state == nil || a.castCursor >= len(a.castOptions) {
		return nil
	}
	option := a.castOptions[a.castCursor]
	if _, ok := a.moverPartyIndex(state.Mover); !ok {
		return nil
	}
	if int(option.ID) < len(a.spellParameters) {
		// overlay-13 `24AAh`：施法時間 = 參數表 +0Ch ÷ 3。不為 0 就先開始施法，
		// 目標與記憶都留到輪到下一次放出去的時候（spell_targets.go）。
		if cost := a.spellParameters[option.ID].CastingCost(); cost > 0 {
			return a.beginPlayerCasting(option, cost)
		}
	}
	return a.aimSpell(option, false)
}

// finishCast 真的把法術施出去。chosen 為真時 target 是玩家挑的那一個。
func (a *app) finishCast(option castOption, target uint8, chosen bool) error {
	state := a.tactical
	if state == nil {
		return nil
	}
	targets := spellTargets{}
	if chosen {
		targets = state.singleSpellTarget(target)
	}
	return a.finishCastTargets(option, targets)
}

// finishCastTargets 是 finishCast 帶整份目標表的那一支（overlay-13 `20AEh` 收好的）。
func (a *app) finishCastTargets(option castOption, targets spellTargets) error {
	// 用物品的那一件不動記憶陣列，交給 overlay-19 entry 8 的後半（combat_commands.go）。
	if handled, err := a.finishCombatItem(option, targets); handled {
		return err
	}
	state := a.tactical
	if state == nil {
		return nil
	}
	index, ok := a.moverPartyIndex(state.Mover)
	if !ok {
		return nil
	}
	member := &a.state.Party[index]
	if option.Slot >= len(member.Memorised) || member.Memorised[option.Slot]&0x7f != option.ID {
		// 記憶陣列在開清單之後被動過，重新來一次比施錯法術好。
		a.tacticalStatus(state, a.text(msgCastNothingReady))
		return nil
	}
	levels := memberClassLevels(*member)
	casterLevel := gamepack.CasterLevelFor(a.spellParameters[option.ID],
		int(levels[gamepack.ClassSlotCleric]), int(levels[gamepack.ClassSlotMagicUser]), false)
	if err := a.castSpell(state, spellCasting{
		name: member.Name, member: member, partySlot: index, level: casterLevel,
		consume: func() {
			member.Memorised[option.Slot] = 0
			syncTrainedLibraryCharacter(&a.state, *member)
		},
	}, option, targets); err != nil {
		return err
	}
	if state.Finished {
		return a.finishCombat(state.Outcome)
	}
	return nil
}

// spellCasting 是「誰在施法」：玩家下指令（finishCast）與 AI 施法（foe_cast.go）
// 共用 castSpell 這一支，差別只在施法者是誰、記憶陣列哪一格要用掉。
type spellCasting struct {
	// name 是訊息裡的施法者名字。
	name string
	// member 是施法的隊員；怪物施法時是 nil。
	member *poolsave.Character
	// partySlot 是施法者的隊伍索引，怪物是 −1。
	partySlot int
	// level 是施法者等級（overlay-25 `26F8h`）。
	level int
	// consume 用掉記憶陣列裡的那一格。
	consume func()
}

// castSpell 是施法的共用後半段：擲效果、用掉記憶的那一格、套用效果、結束這個
// 行動。玩家與 AI 走同一支，所以法術效果只有這一份。戰鬥打完（state.Finished）
// 由呼叫端收：玩家那一側是 finishCast，AI 那一側是 foeTurn 的呼叫端。
func (a *app) castSpell(state *tacticalState, caster spellCasting, option castOption,
	targets spellTargets) error {
	target, chosen := targets.first()
	member := caster.member
	casterLevel := caster.level
	if state.isFriendly(state.Mover) {
		state.Activity.PartyCasts++
	}
	effect, err := a.spellCaster.Cast(option.ID, a.spellParameters, casterLevel, a.roller)
	if err != nil {
		return err
	}
	// 有前提的那幾支：目標身上已經有那個效果就整支不做（原版先問
	// `0100h:006Bh`，中了就直接返回）。記憶那一格照樣用掉。
	if effect.BlockedByEffect != 0 && chosen && int(target) < len(state.Roster) {
		if slot, ok := a.moverPartyIndex(target); ok {
			for _, node := range a.state.Party[slot].Effects {
				if node.Code == effect.BlockedByEffect {
					caster.consume()
					a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastNoEffect),
						option.Label, target))
					state.endTurn(a.rollDice, false)
					return nil
				}
			}
		}
	}
	// 反過來的那一支：縮小術要求目標**身上有**效果 `0Ch`（被變大過），
	// 沒有就整支不做（`1382h` 問 `0100h:006Bh(目標, 0Ch)`，為零就返回）。
	if effect.RequiresEffect != 0 {
		has := false
		if chosen && int(target) < len(state.Roster) {
			if slot, ok := a.moverPartyIndex(target); ok {
				for _, node := range a.state.Party[slot].Effects {
					if node.Code == effect.RequiresEffect {
						has = true
					}
				}
			}
		}
		if !has {
			caster.consume()
			a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastNoEffect),
				option.Label, target))
			state.endTurn(a.rollDice, false)
			return nil
		}
	}
	// 記憶的那一格用掉了，不論打不打得中——原版也是先耗掉才判定。
	caster.consume()

	// 目標是 overlay-13 `20AEh` 收好的那一份（spell_targets.go，玩家瞄、AI 擲骰）：
	// 模式 0 作用在施法者自己、模式 0Ah 以一點收再分邊、模式 8 是射線、9／0Bh 是範圍。
	// 沒挑過（targets.Chosen 為假）的照舊：治療打自己、傷害打繞得過去的最近敵人。
	mode := a.spellParameters[option.ID].TargetMode()
	// 緩毒術那一類：把倒在 0 的人墊回 1。原版問的是選中的目標。
	if effect.MinimumHitPoints > 0 {
		revived := state.Mover
		if chosen {
			revived = target
		}
		if int(revived) < len(state.HitPoints) &&
			state.HitPoints[revived] < effect.MinimumHitPoints {
			state.HitPoints[revived] = effect.MinimumHitPoints
		}
	}
	switch {
	case len(effect.RemoveEffects) > 0:
		// 解病術這一類：拿掉**選中的目標**身上那幾個效果碼。原版
		// `225Bh` 是逐個 `lcall 0100h:006Bh(目標, …)` 再 `002Ah(目標, …)`
		// （spec 098），問的一直是目標，不是施法者；沒挑目標時才是自己。
		subject := member
		if chosen {
			if index, ok := a.moverPartyIndex(target); ok {
				subject = &a.state.Party[index]
			}
		}
		if subject == nil {
			// 怪物施法而目標不是隊員：效果串列只存在隊員身上。
			a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastNoTarget), option.Label))
			break
		}
		removed := 0
		for _, code := range effect.RemoveEffects {
			for index, node := range subject.Effects {
				if node.Code == code {
					subject.Effects = append(subject.Effects[:index], subject.Effects[index+1:]...)
					removed++
					break
				}
			}
		}
		syncTrainedLibraryCharacter(&a.state, *subject)
		a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastCured),
			strings.TrimSpace(subject.Name), removed))
	case effect.EffectCode == gamepack.HoldPersonEffectCode:
		// 定身術：規則 1（豁免成功完全無效，spec 074）。中了就照參數表的
		// 持續回合數定住，那一格輪到就直接結束回合。
		//
		// `1650h` 逐一走 `20AEh` 收好的那張表（`DS:6B85h`，定身術 3 個、定身
		// 怪物 4 個），每一個各丟一次豁免。沒挑過（remake 自己挑）時是最近的敵人。
		victims := targets.List
		if !chosen {
			if picked, found := state.nearestReachableOpposing(state.Mover); found {
				victims = []uint8{picked}
			} else {
				victims = nil
			}
		}
		if len(victims) == 0 {
			a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastNoTarget), option.Label))
			break
		}
		rounds := a.spellParameters[option.ID].Duration(casterLevel)
		if rounds < 1 {
			rounds = 1
		}
		// 豁免修正看這一次選了幾個目標（overlay-22 `1656h`）：1 個時定身術 −2、
		// 定身怪物 −3，2 個 −1，3 或 4 個 0。
		modifier := 0
		if effect.SaveModifierByTargetCount {
			modifier = gamepack.HoldPersonSaveModifier(option.ID, len(victims))
		}
		for _, picked := range victims {
			if int(picked) >= len(state.Roster) || state.Roster[picked].FootprintClass == 0 {
				continue
			}
			// 不是人的目標一律當作豁免成功（`175Dh` 直接把結果設成 1）。
			if effect.PersonOnly && !state.affectsPerson(picked) {
				a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastNotPerson),
					picked, option.Label))
				continue
			}
			if a.savedAgainstSpellWithModifier(state, picked, option.ID, modifier) {
				a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastResisted), picked, option.Label))
				continue
			}
			state.addEffect(int(picked), gamepack.HoldPersonEffectCode, rounds, casterLevel)
			a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastHeld), picked, rounds))
		}
	case effect.StrengthValue > 0 || effect.StrengthFromTarget:
		// 力量那一組（變大術、力量術、編號 59）走同一支
		// overlay-24 entry 18：只往上調，調不動就什麼都不做。
		cell := int(state.Mover)
		if chosen {
			cell = int(target)
		}
		if cell <= 0 || cell >= len(state.PartySlot) || state.PartySlot[cell] < 0 ||
			state.PartySlot[cell] >= len(a.state.Party) {
			a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastNoTarget), option.Label))
			break
		}
		slot := state.PartySlot[cell]
		subject := &a.state.Party[slot]
		current := uint8(subject.Abilities[gamepack.AbilityStrength])
		currentPercentile := uint8(subject.ExceptionalStrength)
		value, percentile := effect.StrengthValue, effect.StrengthPercentile
		if effect.StrengthFromTarget {
			value, percentile = gamepack.StrengthSpellResult(
				memberClassLevels(*subject), current, currentPercentile, a.roller)
		}
		duration := a.spellParameters[option.ID].Duration(casterLevel)
		list, value, percentile, raised := gamepack.ApplyStrengthEffect(
			state.Effects[cell], effect.EffectCode, duration,
			current, currentPercentile, value, percentile)
		state.Effects[cell] = list
		subject.Effects = storedEffects(list)
		if !raised {
			syncTrainedLibraryCharacter(&a.state, *subject)
			a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastNoEffect),
				option.Label, target))
			break
		}
		subject.Abilities[gamepack.AbilityStrength] = int(value)
		subject.ExceptionalStrength = int(percentile)
		syncTrainedLibraryCharacter(&a.state, *subject)
		a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastStronger),
			strings.TrimSpace(subject.Name), value, percentile))
	case effect.HitPointBudgetFromCaster:
		// 迷蛇術：額度是施法者的目前生命值。
		a.applyCharmByHitPoints(state, caster.name, effect,
			state.HitPoints[state.Mover], casterLevel)
	case effect.PersonOnly:
		// 魅惑人類：只對「人」有效，中了就不再行動。
		picked, found := target, chosen
		if !found {
			picked, found = state.nearestReachableOpposing(state.Mover)
		}
		if !found {
			a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastNoTarget), option.Label))
			break
		}
		if !state.affectsPerson(picked) {
			a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastNotPerson),
				picked, option.Label))
			break
		}
		if a.savedAgainstSpell(state, picked, option.ID) {
			a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastResisted), picked, option.Label))
			break
		}
		state.applyCharm(int(picked), casterLevel)
		a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastCharmed),
			strings.TrimSpace(caster.name), 1))
	case effect.SleepBudget > 0:
		// 瞄過一點的（`20AEh` 收好的表）走那張表；沒瞄過的照舊走整個敵方。
		var candidates []uint8
		if targets.Area && chosen {
			candidates = targets.List
		}
		a.applySleep(state, caster.name, option.Label, effect.SleepBudget, casterLevel, candidates)
	case effect.Heal > 0:
		healed := state.Mover
		if chosen {
			healed = target
		}
		before := state.HitPoints[healed]
		state.HitPoints[healed] += effect.Heal
		a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastHealed),
			strings.TrimSpace(caster.name), option.Label, state.HitPoints[healed]-before))
	case effect.Cloud:
		// 臭雲術（overlay-22 `1AF6h`，spec 121）：在盤上生一團 2×2 的雲，
		// 蓋成地形 `1Eh`；效果碼 `1Eh` 掛在**當下站在那四格裡的人**身上，
		// 之後每次輪到他行動就結算一次（stinkingCloudTurn）。
		centreX, centreY := int(state.Roster[state.Mover].X), int(state.Roster[state.Mover].Y)
		if targets.Area && chosen {
			// 雲心是瞄準選的那一點（`DS:6CADh`／`6CAEh`），可以是空格子。
			centreX, centreY = targets.X, targets.Y
		} else if chosen && int(target) < len(state.Roster) {
			centreX, centreY = int(state.Roster[target].X), int(state.Roster[target].Y)
		} else if picked, ok := state.nearestReachableOpposing(state.Mover); ok {
			centreX, centreY = int(state.Roster[picked].X), int(state.Roster[picked].Y)
		}
		// 雲記在施法者的**戰鬥員序號**上，不是隊伍欄位——原版節點 `+0..+3`
		// 記的是施法者的記錄，而效果串列也是按戰鬥員索引的。兩邊用同一把尺，
		// 節點到期時才找得回那一團雲（spec 121）。
		if !state.placeCloud(int(state.Mover), centreX, centreY, casterLevel) {
			a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastNoTarget), option.Label))
			break
		}
		for cell := 1; cell < len(state.Roster); cell++ {
			if state.Roster[cell].FootprintClass == 0 || !state.standingInCloud(cell) {
				continue
			}
			state.addEffect(cell, effect.EffectCode, 0, casterLevel)
		}
		a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastCloud),
			strings.TrimSpace(caster.name)))
	case effect.Restore:
		// 恢復術（overlay-22 `2C01h`）：把能量吸取的欠帳還一級。
		// 沒欠就整支直接返回——原版的 `2C16h` 就是這樣。
		slot := caster.partySlot
		if chosen {
			if party, ok := a.moverPartyIndex(target); ok {
				slot = party
			}
		}
		if slot < 0 || slot >= len(a.state.Party) {
			a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastNoTarget), option.Label))
			break
		}
		subject := &a.state.Party[slot]
		levels := memberClassLevels(*subject)
		outcome, restoredLevels, restoredExperience := gamepack.RestoreDrainedLevel(
			levels, subject.Experience, subject.DrainedLevels,
			subject.DrainedHitPoints, a.experienceTable)
		if !outcome.Restored {
			a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastNothingToRestore),
				strings.TrimSpace(subject.Name)))
			break
		}
		subject.DrainedLevels, subject.DrainedHitPoints =
			outcome.DrainedLevels, outcome.DrainedHitPoints
		subject.ClassLevels = append([]uint8(nil), restoredLevels[:]...)
		subject.Experience = restoredExperience
		subject.MaxHP += outcome.HitPoints
		subject.CurrentHP += outcome.HitPoints
		subject.RawHP += outcome.HitPoints
		syncTrainedLibraryCharacter(&a.state, *subject)
		if slot < len(state.PartySlot) {
			for cell := 1; cell < len(state.PartySlot); cell++ {
				if state.PartySlot[cell] == slot && cell < len(state.HitPoints) {
					state.HitPoints[cell] += outcome.HitPoints
					if cell < len(state.MaxHitPoints) {
						state.MaxHitPoints[cell] += outcome.HitPoints
					}
				}
			}
		}
		a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastRestored),
			strings.TrimSpace(subject.Name), outcome.HitPoints))
	case effect.AnimateDead:
		// 死靈術：把死掉的人類屍體叫起來，換到施法者那一邊（spec 098）。
		raised := state.animateDead(casterLevel)
		if raised == 0 {
			a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastNoTarget), option.Label))
			break
		}
		a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastAnimated), raised))
	case effect.Dispel:
		// 解除魔法（overlay-22 `2356h`）：沿著目標身上的效果節點串列走，
		// 每一個各擲一次。`+3` 是 `0FFh` 的解不掉。
		//
		// 原版的外層是 `for i := 1 to DS:6B88h`，但**每一輪都從 DS:6B89h 取
		// 同一個遠指標**（絕對定址，沒有索引暫存器），所以實際上只作用在
		// 一個目標身上。這裡照那個行為接：挑一個目標，不是整個範圍。
		picked, found := target, chosen
		if !found {
			picked, found = state.nearestReachableOpposing(state.Mover)
		}
		if !found {
			a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastNoTarget), option.Label))
			break
		}
		removed := state.dispelEffects(int(picked), casterLevel, func() int {
			return a.roller.Roll(1, 100)
		})
		a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastDispelled), picked, removed))
	case effect.Ray != nil:
		// 閃電束與編號 3Ch：由瞄準的那一點拉射線（spell_targets.go 的 castSpellRay）。
		// 沒瞄過（remake 自己挑）時瞄繞得過去的最近敵人。
		x, y, found := targets.X, targets.Y, chosen
		if !found {
			if picked, ok := state.nearestReachableOpposing(state.Mover); ok {
				x, y, found = int(state.Roster[picked].X), int(state.Roster[picked].Y), true
			}
		}
		if !found {
			a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastNoTarget), option.Label))
			break
		}
		hit, err := a.castSpellRay(state, effect, x, y)
		if err != nil {
			return err
		}
		if hit == 0 {
			a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastNoTarget), option.Label))
		} else {
			a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastArea),
				option.Label, hit, effect.Damage))
		}
	case effect.Damage > 0 && a.spellParameters[option.ID].AffectsArea() &&
		targets.Area && chosen:
		// 範圍：`20AEh` 以瞄準的那一點收好的表，**不分敵我**每一個都吃一份
		// （`08BCh` 的 `090Ch..0A62h` 逐一走 `DS:6B85h`）。火球術在 `@49E6` 為 0 時
		// 以同一點、預算 2 重收一次（overlay-22 `2661h..26DAh`）。
		victims := targets.List
		walkFlag := uint8(0)
		if a.eventMachine != nil && a.eventMachine.Memory[encounterWalkFlagAddress] != 0 {
			walkFlag = 1
		}
		if budget := gamepack.FireballOutdoorAreaBudget(option.ID, walkFlag); budget > 0 {
			members, err := state.spellAreaMembers(targets.X, targets.Y, budget)
			if err != nil {
				return err
			}
			victims = members
		}
		hit := 0
		for _, index := range victims {
			if int(index) >= len(state.Roster) || state.Roster[index].FootprintClass == 0 {
				continue
			}
			a.applySpellDamage(state, index, a.damageAfterSave(state, index, option.ID, effect.Damage))
			hit++
		}
		if hit == 0 {
			a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastNoTarget), option.Label))
		} else {
			a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastArea),
				option.Label, hit, effect.Damage))
		}
	case effect.Damage > 0 && a.spellParameters[option.ID].AffectsArea():
		// 沒瞄過的範圍（remake 自己挑的）：對面每一個都吃一份。
		// 原版是以一格為中心算範圍（overlay-31 `0138h:003Eh`）：中心是挑中的目標，
		// 沒挑目標時中心退回施法者自己。
		centre := state.Mover
		if chosen && int(target) < len(state.Roster) {
			centre = target
		}
		hit := 0
		for index := 1; index < len(state.Roster); index++ {
			if state.Roster[index].FootprintClass == 0 ||
				state.Friendly[index] == state.Friendly[state.Mover] {
				continue
			}
			// 有讀出預算的就照原版收人：那個預算內走得到才算在範圍裡
			// （`0912h` 把預算交給 `0419h` 當上限）。沒讀出來的先收整邊。
			if effect.AreaBudget > 0 && !state.withinArea(centre, uint8(index),
				uint16(effect.AreaBudget)) {
				continue
			}
			a.applySpellDamage(state, uint8(index),
				a.damageAfterSave(state, uint8(index), option.ID, effect.Damage))
			hit++
		}
		if hit == 0 {
			a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastNoTarget), option.Label))
		} else {
			a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastArea),
				option.Label, hit, effect.Damage))
		}
	case effect.Damage > 0:
		picked, found := target, chosen
		if !found {
			picked, found = state.nearestReachableOpposing(state.Mover)
		}
		if !found {
			a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastNoTarget), option.Label))
			break
		}
		a.applySpellDamage(state, picked,
			a.damageAfterSave(state, picked, option.ID, effect.Damage))
	case mode == gamepack.SpellTargetWholeSide:
		// 模式 0Ah：`20AEh` 以瞄準的那一點、預算 2 收好表，處理常式（`0F35h`／`2724h`）
		// 再從表裡分邊（sideSpellTargets）。沒瞄過（remake 自己挑）時照舊數施法者那一邊。
		affected := 0
		if chosen {
			kept, err := state.sideSpellTargets(option.ID, targets.List, casterLevel)
			if err != nil {
				return err
			}
			affected = len(kept)
		} else {
			for index := 1; index < len(state.Roster); index++ {
				if state.Roster[index].FootprintClass == 0 ||
					state.Friendly[index] != state.Friendly[state.Mover] {
					continue
				}
				affected++
			}
		}
		a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastWholeSide),
			strings.TrimSpace(caster.name), option.Label, affected))
	default:
		a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastTookEffect),
			strings.TrimSpace(caster.name), option.Label))
	}
	state.endTurnAfterAction(a.rollDice)
	return nil
}

// damageAfterSave 讓目標擲一次豁免，再依法術參數 `+8` 的規則處置傷害
//（spec 074／075）：規則 0 不擲、1 豁免成功就完全無效、2 減半、其餘不動。
//
// 原版在共用施法常式 `08BCh` 裡先擲一次豁免（`096Bh` 呼叫 overlay-24 entry 7），
// 把布林結果和規則值一起交給 overlay-24 entry 19（`133Ah`）處置。
func (a *app) damageAfterSave(state *tacticalState, target, spell uint8, damage int) int {
	if !a.savedAgainstSpell(state, target, spell) {
		return damage
	}
	return gamepack.DamageAfterSave(a.spellParameters[spell].SaveRule(), damage)
}

// savedAgainstSpell 讓目標對這一支法術擲一次豁免。不用擲（規則 0）或資料
// 不齊時回 false，讓呼叫端照「沒豁免成功」處理。
func (a *app) savedAgainstSpell(state *tacticalState, target, spell uint8) bool {
	return a.savedAgainstSpellWithModifier(state, target, spell, 0)
}

// savedAgainstSpellWithModifier 是同一件事外加一個豁免修正。原版把修正
// 交給 overlay-24 entry 7（`0100h:0043h` 的第三個引數），常式把它加進
// d20；這裡與既有的 `+101h` 修正同一側處理。
func (a *app) savedAgainstSpellWithModifier(state *tacticalState,
	target, spell uint8, modifier int) bool {
	if state == nil || int(target) >= len(state.SaveTargets) {
		return false
	}
	if int(spell) >= len(a.spellParameters) {
		return false
	}
	parameters := a.spellParameters[spell]
	if parameters.SaveRule() == gamepack.SaveRuleNone {
		return false
	}
	return a.savedAgainstCategory(state, target, parameters.SaveCategory(), modifier)
}

// savedAgainstCategory 是 overlay-24 entry 7（`0100h:0043h(記錄, 類別, 修正)`）本身：
// 類別直接由呼叫端給。射線（`287Ch`）推的是處理常式寫死的類別，不看參數表。
func (a *app) savedAgainstCategory(state *tacticalState, target uint8,
	category gamepack.SaveCategory, modifier int) bool {
	if state == nil || int(target) >= len(state.SaveTargets) {
		return false
	}
	if int(category) >= gamepack.SavingThrowCategories {
		return false
	}
	roll := a.rollDice(1, gamepack.SavingThrowDie)
	switch {
	case roll == 1:
		return false
	case roll == gamepack.SavingThrowDie:
		return true
	default:
		return int(state.SaveTargets[target][category]) <=
			roll+state.SaveBonus[target]+modifier
	}
}

// applySpellDamage 用與攻擊同一套的收尾：歸零就不再佔格、不再參與。
func (a *app) applySpellDamage(state *tacticalState, target uint8, damage int) {
	if int(target) >= len(state.HitPoints) {
		return
	}
	// 一擊斃命（spec 141）：施法者是隊員、豁免後傷害仍大於 0 時歸零。
	damage = a.cheatDamage(state, state.Mover, target, damage)
	state.HitPoints[target] -= damage
	// overlay-24 entry 19 `1500h..155Fh`：傷害大於 0 的當下清 runtime +1、丟失施法中的。
	a.woundCombatant(state, target, damage)
	defer a.announceLostSpells(state)
	if state.HitPoints[target] > 0 {
		a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastHit),
			target, damage, state.HitPoints[target]))
		return
	}
	state.HitPoints[target] = 0
	state.rememberFootprint(int(target))
	state.Roster[target].FootprintClass = 0
	state.Scores[target] = 0
	state.States[target] = combat.DyingState
	a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastDown), target))
}

// tacticalStatus 把一行字放進戰場的狀態列。
func (a *app) tacticalStatus(state *tacticalState, line string) {
	state.Status = line
	a.statusLine = line
}

// affectsPerson 回答「這一格算不算人」：種類 `+9Fh` 不大於 1、
// 體型 `+6Ch` 不大於 1（overlay-22 `11DAh`／`174Bh`）。
func (state *tacticalState) affectsPerson(index uint8) bool {
	if int(index) >= len(state.CreatureType) || int(index) >= len(state.BodySize) {
		return false
	}
	return gamepack.SpellAffectsPerson(state.CreatureType[index], state.BodySize[index])
}

// applyCharmByHitPoints 是迷蛇術：沿著對面走，種類對得上而且目前生命值
// 扣得動的就迷住（overlay-22 `18F9h`）。
//
// 原版走的是這一次挑出來的目標清單（`DS:6B85h`），這裡沒有瞄準那一層，
// 所以走整個敵方，順序就是位置順序——與催眠術同一個近似。
func (a *app) applyCharmByHitPoints(state *tacticalState, caster string,
	effect gamepack.CastEffect, budget, casterLevel int) {
	charmed := 0
	for index := 1; index < len(state.Roster); index++ {
		if index >= len(state.Effects) || index >= len(state.CreatureType) {
			break
		}
		if state.Roster[index].FootprintClass == 0 ||
			state.Friendly[index] == state.Friendly[state.Mover] ||
			state.hasEffect(index, gamepack.CharmPersonEffectCode) {
			continue
		}
		if effect.CreatureTypeFiltered && state.CreatureType[index] != effect.CreatureType {
			continue
		}
		cost := state.HitPoints[index]
		if cost > budget {
			continue
		}
		budget -= cost
		state.applyCharm(index, casterLevel)
		charmed++
	}
	if charmed == 0 {
		a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastCharmedNone),
			strings.TrimSpace(caster)))
		return
	}
	a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastCharmed),
		strings.TrimSpace(caster), charmed))
}

// applySleep 依額度逐個放倒目標（spec 098）。
//
// 原版走的是這一次施法收出來的目標清單（`DS:6B85h`，overlay-22 `152Ch` 起），
// 表上的人不分敵我。candidates 就是那張表；nil 代表沒瞄過，照舊走整個敵方，
// 順序就是位置順序。花費照 `SleepHitDiceCost`。
func (a *app) applySleep(state *tacticalState, caster, label string, budget, casterLevel int,
	candidates []uint8) {
	if candidates == nil {
		for index := 1; index < len(state.Roster); index++ {
			if state.Friendly[index] != state.Friendly[state.Mover] {
				candidates = append(candidates, uint8(index))
			}
		}
	}
	slept := 0
	for _, cell := range candidates {
		index := int(cell)
		if index <= 0 || index >= len(state.Roster) || state.Roster[index].FootprintClass == 0 ||
			state.hasEffect(index, gamepack.SleepEffectCode) {
			continue
		}
		cost := gamepack.SleepHitDiceCost(int(state.HitDice[index]), state.SleepFlag[index])
		if cost > budget {
			continue
		}
		budget -= cost
		state.addEffect(index, gamepack.SleepEffectCode, 0, casterLevel)
		slept++
	}
	if slept == 0 {
		a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastSleptNone), strings.TrimSpace(caster)))
		return
	}
	a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastSlept), strings.TrimSpace(caster), slept))
	_ = label
}
