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
		id := value & 0x7f
		if id == 0 || !gamepack.SpellIsImplemented(id) {
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

// resolveCast 施出選中的那一條。
func (a *app) resolveCast() error {
	state := a.tactical
	a.castOpen = false
	if state == nil || a.castCursor >= len(a.castOptions) {
		return nil
	}
	option := a.castOptions[a.castCursor]
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
	effect, err := gamepack.CastSpell(option.ID, a.spellParameters, casterLevel, a.roller)
	if err != nil {
		return err
	}
	// 記憶的那一格用掉了，不論打不打得中——原版也是先耗掉才判定。
	member.Memorised[option.Slot] = 0
	syncTrainedLibraryCharacter(&a.state, *member)

	switch {
	case effect.SleepBudget > 0:
		a.applySleep(state, member.Name, option.Label, effect.SleepBudget)
	case effect.Heal > 0:
		before := state.HitPoints[state.Mover]
		state.HitPoints[state.Mover] += effect.Heal
		a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastHealed),
			strings.TrimSpace(member.Name), option.Label, state.HitPoints[state.Mover]-before))
	case effect.Damage > 0:
		target, found := state.nearestReachableOpposing(state.Mover)
		if !found {
			a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastNoTarget), option.Label))
			break
		}
		a.applySpellDamage(state, target, effect.Damage)
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
