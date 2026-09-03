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
	// 記憶的那一格用掉了，不論打不打得中——原版也是先耗掉才判定。
	member.Memorised[option.Slot] = 0
	syncTrainedLibraryCharacter(&a.state, *member)

	// 挑目標照原版的模式（參數表 `+6` 的低四位，spec 074）：模式 0 作用在
	// 施法者自己、模式 0Ah 作用在整邊、模式 8／9／0Bh 是範圍。
	// **模式 4 那三十支原版是讓玩家自己瞄**（overlay-13 `1E09h`），
	// 那條還沒讀，所以這裡治療打自己、傷害打繞得過去的最近敵人。
	mode := a.spellParameters[option.ID].TargetMode()
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
	case effect.SleepBudget > 0:
		a.applySleep(state, member.Name, option.Label, effect.SleepBudget)
	case effect.Heal > 0:
		healed := state.Mover
		if chosen {
			healed = target
		}
		before := state.HitPoints[healed]
		state.HitPoints[healed] += effect.Heal
		a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastHealed),
			strings.TrimSpace(member.Name), option.Label, state.HitPoints[healed]-before))
	case effect.Damage > 0 && a.spellParameters[option.ID].AffectsArea():
		// 範圍：對面每一個都吃一份。原版是以一格為中心算範圍
		// （overlay-31 `0138h:003Eh`），那條還沒讀。
		hit := 0
		for index := 1; index < len(state.Roster); index++ {
			if state.Roster[index].FootprintClass == 0 ||
				state.Friendly[index] == state.Friendly[state.Mover] {
				continue
			}
			a.applySpellDamage(state, uint8(index), effect.Damage)
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
		a.applySpellDamage(state, picked, effect.Damage)
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

// applySleep 依額度逐個放倒對面的人（spec 098）。
//
// 原版走的是這一次施法挑出來的目標清單（`DS:6B85h`），這裡沒有瞄準那一層，
// 所以走整個敵方，順序就是位置順序。花費照 `SleepHitDiceCost`。
func (a *app) applySleep(state *tacticalState, caster, label string, budget int) {
	slept := 0
	for index := 1; index < len(state.Roster); index++ {
		if state.Roster[index].FootprintClass == 0 ||
			state.Friendly[index] == state.Friendly[state.Mover] || state.Asleep[index] {
			continue
		}
		cost := gamepack.SleepHitDiceCost(int(state.HitDice[index]), state.SleepFlag[index])
		if cost > budget {
			continue
		}
		budget -= cost
		state.Asleep[index] = true
		slept++
	}
	if slept == 0 {
		a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastSleptNone), strings.TrimSpace(caster)))
		return
	}
	a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastSlept), strings.TrimSpace(caster), slept))
	_ = label
}
