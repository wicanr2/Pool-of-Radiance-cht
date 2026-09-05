package main

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// 戰鬥中施法（spec 098）。C 開清單、上下挑、Enter 施、ESC 取消。
//
// **只列得出已經讀過處理常式的法術**（`SpellIsImplemented`）。沒讀過的不列，
// 而不是列了之後失敗——玩家看得到的清單就是實際做得到的事。
//
// **選目標是 remake 自己的作法**：傷害法術打「繞得過去的最近敵人」、
// 治療打自己、整邊的效果作用在自己這邊。原版讓玩家自己瞄（overlay-08 的
// `View Aim` 指令），那條還沒讀。

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
	member := a.state.Party[index]
	options := make([]castOption, 0, gamepack.MemorisedSpellSlots)
	for slot, value := range member.Memorised {
		// 還沒記完的（第 7 位還在）施不出來，要休息過（spec 070）。
		if !gamepack.MemorisedSpellIsReady(value) {
			continue
		}
		id := value & 0x7f
		if !a.spellCaster.Implemented(id) {
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
	if len(options) == 0 {
		a.tacticalStatus(state, a.text(msgCastNothingReady))
		return
	}
	a.castOptions, a.castCursor, a.castOpen = options, 0, true
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
// `Next Prev Manual`（overlay-13 的字串），而且允許打自己人
//（`Attack Ally:`）。這裡做的是同一件事的最小版本：N／P 或左右鍵換人、
// Enter 確定，預設停在繞得過去的最近敵人身上。`352Ch` 那一支還沒讀，
// 所以**格子游標（Manual）那一半沒有**。
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

// moverAttackRange 是現在這個角色打得到幾格。沒有裝備武器就是相鄰。
func (a *app) moverAttackRange() int {
	index, ok := a.moverPartyIndex(a.tactical.Mover)
	if !ok {
		return 1
	}
	weapon, ok := a.readiedWeapon(a.state.Party[index])
	if !ok || a.itemTypes == nil {
		return 1
	}
	if len(weapon.Raw) <= itemTypeOffset {
		return 1
	}
	entry, err := a.itemTypes.Entry(weapon.Raw[itemTypeOffset])
	if err != nil {
		return 1
	}
	return entry.AttackRange()
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
	state.endTurn(a.rollDice, false)
	if state.Finished {
		return a.finishCombat(state.Outcome)
	}
	return nil
}

// castTargetingInput 處理選目標那一步的按鍵。
func (a *app) castTargetingInput() error {
	switch {
	case a.justPressed(ebiten.KeyEscape):
		a.castTargeting, a.castTargetingAttack = false, false
	case a.justPressed(ebiten.KeyP), a.justPressed(ebiten.KeyArrowLeft),
		a.justPressed(ebiten.KeyArrowUp):
		a.castTargetCursor = (a.castTargetCursor + len(a.castTargets) - 1) % len(a.castTargets)
	case a.justPressed(ebiten.KeyN), a.justPressed(ebiten.KeyArrowRight),
		a.justPressed(ebiten.KeyArrowDown):
		a.castTargetCursor = (a.castTargetCursor + 1) % len(a.castTargets)
	case a.justPressed(ebiten.KeyEnter), a.justPressed(ebiten.KeySpace):
		a.castTargeting = false
		target := a.castTargets[a.castTargetCursor]
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
	// 要挑目標的那幾種模式先進選目標那一步。
	switch a.spellParameters[option.ID].TargetMode() {
	case gamepack.SpellTargetSingle, gamepack.SpellTargetHold,
		gamepack.SpellTargetHoldAlt, gamepack.SpellTargetPick:
		if a.beginCastTargeting(option) {
			return nil
		}
	}
	return a.finishCast(option, 0, false)
}

// finishCast 真的把法術施出去。chosen 為真時 target 是玩家挑的那一個。
func (a *app) finishCast(option castOption, target uint8, chosen bool) error {
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
	effect, err := a.spellCaster.Cast(option.ID, a.spellParameters, casterLevel, a.roller)
	if err != nil {
		return err
	}
	// 有前提的那幾支：目標身上已經有那個效果就整支不做（原版先問
	// `0100h:006Bh`，中了就直接返回）。記憶那一格照樣用掉。
	if effect.BlockedByEffect != 0 && chosen && int(target) < len(state.Roster) {
		if slot, ok := a.moverPartyIndex(target); ok {
			for _, value := range a.state.Party[slot].Effects {
				if value == effect.BlockedByEffect {
					member.Memorised[option.Slot] = 0
					syncTrainedLibraryCharacter(&a.state, *member)
					a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastNoEffect),
						option.Label, target))
					state.endTurn(a.rollDice, false)
					if state.Finished {
						return a.finishCombat(state.Outcome)
					}
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
				for _, value := range a.state.Party[slot].Effects {
					if value == effect.RequiresEffect {
						has = true
					}
				}
			}
		}
		if !has {
			member.Memorised[option.Slot] = 0
			syncTrainedLibraryCharacter(&a.state, *member)
			a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastNoEffect),
				option.Label, target))
			state.endTurn(a.rollDice, false)
			if state.Finished {
				return a.finishCombat(state.Outcome)
			}
			return nil
		}
	}
	// 記憶的那一格用掉了，不論打不打得中——原版也是先耗掉才判定。
	member.Memorised[option.Slot] = 0
	syncTrainedLibraryCharacter(&a.state, *member)

	// 挑目標照原版的模式（參數表 `+6` 的低四位，spec 074）：模式 0 作用在
	// 施法者自己、模式 0Ah 作用在整邊、模式 8／9／0Bh 是範圍。
	// **模式 4 那三十支原版是讓玩家自己瞄**（overlay-13 `1E09h`），
	// 那條還沒讀，所以這裡治療打自己、傷害打繞得過去的最近敵人。
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
		// 解病術這一類：從施法者身上拿掉那幾個效果碼。原版問的是選中的目標，
		// remake 還沒有瞄準那一層，所以先對自己。
		removed := 0
		for _, code := range effect.RemoveEffects {
			for index, value := range member.Effects {
				if value == code {
					member.Effects = append(member.Effects[:index], member.Effects[index+1:]...)
					removed++
					break
				}
			}
		}
		syncTrainedLibraryCharacter(&a.state, *member)
		a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastCured),
			strings.TrimSpace(member.Name), removed))
	case effect.EffectCode == gamepack.HoldPersonEffectCode:
		// 定身術：規則 1（豁免成功完全無效，spec 074）。中了就照參數表的
		// 持續回合數定住，那一格輪到就直接結束回合。
		picked, found := target, chosen
		if !found {
			picked, found = state.nearestReachableOpposing(state.Mover)
		}
		if !found {
			a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastNoTarget), option.Label))
			break
		}
		rounds := a.spellParameters[option.ID].Duration(casterLevel)
		if rounds < 1 {
			rounds = 1
		}
		// 不是人的目標一律當作豁免成功（`175Dh` 直接把結果設成 1）。
		if effect.PersonOnly && !state.affectsPerson(picked) {
			a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastNotPerson),
				picked, option.Label))
			break
		}
		// 豁免修正看這一次選了幾個目標。remake 一次只瞄一個，所以是 1 個
		// 那一格：定身術 −2、定身怪物 −3（overlay-22 `1656h`）。
		modifier := 0
		if effect.SaveModifierByTargetCount {
			modifier = gamepack.HoldPersonSaveModifier(option.ID, 1)
		}
		if a.savedAgainstSpellWithModifier(state, picked, option.ID, modifier) {
			a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastResisted), picked, option.Label))
			break
		}
		state.addEffect(int(picked), gamepack.HoldPersonEffectCode, rounds, casterLevel)
		a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastHeld), picked, rounds))
	case effect.StrengthValue > 0 || effect.StrengthFromTarget:
		// 力量那一組（變大術、力量術、編號 59）走同一支
		// overlay-24 entry 18：只往上調，調不動就什麼都不做。
		slot := index
		if chosen {
			if party, ok := a.moverPartyIndex(target); ok {
				slot = party
			}
		}
		subject := &a.state.Party[slot]
		current := uint8(subject.Abilities[gamepack.AbilityStrength])
		currentPercentile := uint8(subject.ExceptionalStrength)
		value, percentile := effect.StrengthValue, effect.StrengthPercentile
		if effect.StrengthFromTarget {
			value, percentile = gamepack.StrengthSpellResult(
				memberClassLevels(*subject), current, currentPercentile, a.roller)
		}
		value, percentile, raised := gamepack.RaiseStrength(
			current, currentPercentile, value, percentile)
		if !raised {
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
		a.applyCharmByHitPoints(state, member.Name, effect,
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
			strings.TrimSpace(member.Name), 1))
	case effect.SleepBudget > 0:
		a.applySleep(state, member.Name, option.Label, effect.SleepBudget, casterLevel)
	case effect.Heal > 0:
		healed := state.Mover
		if chosen {
			healed = target
		}
		before := state.HitPoints[healed]
		state.HitPoints[healed] += effect.Heal
		a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastHealed),
			strings.TrimSpace(member.Name), option.Label, state.HitPoints[healed]-before))
	case effect.Cloud:
		// 臭雲術（overlay-22 `1AF6h`，spec 121）：在盤上生一團 2×2 的雲，
		// 蓋成地形 `1Eh`；效果碼 `1Eh` 掛在**當下站在那四格裡的人**身上，
		// 之後每次輪到他行動就結算一次（stinkingCloudTurn）。
		centreX, centreY := int(state.Roster[state.Mover].X), int(state.Roster[state.Mover].Y)
		if chosen && int(target) < len(state.Roster) {
			centreX, centreY = int(state.Roster[target].X), int(state.Roster[target].Y)
		} else if picked, ok := state.nearestReachableOpposing(state.Mover); ok {
			centreX, centreY = int(state.Roster[picked].X), int(state.Roster[picked].Y)
		}
		if !state.placeCloud(index, centreX, centreY) {
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
			strings.TrimSpace(member.Name)))
	case effect.Restore:
		// 恢復術（overlay-22 `2C01h`）：把能量吸取的欠帳還一級。
		// 沒欠就整支直接返回——原版的 `2C16h` 就是這樣。
		slot := index
		if chosen {
			if party, ok := a.moverPartyIndex(target); ok {
				slot = party
			}
		}
		subject := &a.state.Party[slot]
		outcome := gamepack.Restore(subject.DrainedLevels, subject.DrainedHitPoints)
		if !outcome.Restored {
			a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastNothingToRestore),
				strings.TrimSpace(subject.Name)))
			break
		}
		subject.DrainedLevels, subject.DrainedHitPoints =
			outcome.DrainedLevels, outcome.DrainedHitPoints
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
	case effect.Damage > 0 && a.spellParameters[option.ID].AffectsArea():
		// 範圍：對面每一個都吃一份。原版是以一格為中心算範圍
		// （overlay-31 `0138h:003Eh`），那條還沒讀。
		hit := 0
		for index := 1; index < len(state.Roster); index++ {
			if state.Roster[index].FootprintClass == 0 ||
				state.Friendly[index] == state.Friendly[state.Mover] {
				continue
			}
			// 有讀出預算的就照原版收人：那個預算內走得到才算在範圍裡
			// （`0912h` 把預算交給 `0419h` 當上限）。沒讀出來的先收整邊。
			if effect.AreaBudget > 0 && !state.withinArea(state.Mover, uint8(index),
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
		// 模式 0Ah：整邊。原版走 0F35h，把效果掛給施法者那一邊的每個人。
		affected := 0
		for index := 1; index < len(state.Roster); index++ {
			if state.Roster[index].FootprintClass == 0 ||
				state.Friendly[index] != state.Friendly[state.Mover] {
				continue
			}
			affected++
		}
		a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastWholeSide),
			strings.TrimSpace(member.Name), option.Label, affected))
	default:
		a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastTookEffect),
			strings.TrimSpace(member.Name), option.Label))
	}
	state.endTurn(a.rollDice, false)
	if state.Finished {
		return a.finishCombat(state.Outcome)
	}
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
	category := parameters.SaveCategory()
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
	state.HitPoints[target] -= damage
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

// applySleep 依額度逐個放倒對面的人（spec 098）。
//
// 原版走的是這一次施法挑出來的目標清單（`DS:6B85h`），這裡沒有瞄準那一層，
// 所以走整個敵方，順序就是位置順序。花費照 `SleepHitDiceCost`。
func (a *app) applySleep(state *tacticalState, caster, label string, budget, casterLevel int) {
	slept := 0
	for index := 1; index < len(state.Roster); index++ {
		if state.Roster[index].FootprintClass == 0 ||
			state.Friendly[index] == state.Friendly[state.Mover] ||
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
