package main

import (
	"fmt"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// 怪物接近之前的效果群組 0Eh（overlay-09 entry 5 開場 `0B66h`，spec 096／098／161，issue #82）。
// 群組 0Eh 依序是 `53h 54h 58h 79h`；這個檔案是 `58h` 吐息（gamepack/breath.go），其餘三個在
// approach_effects.go。處理常式叫了 entry 34 結束行動，`0B73h` 看 runtime `+3` 為 0 就不進接近迴圈。

const (
	// msgFoeBreathes 是 overlay-22 `3088h` 的 "Breathes!"。
	msgFoeBreathes messageID = iota + 5001
)

func init() {
	for id, key := range map[messageID]string{
		msgFoeBreathes: "ui.foeBreathes",
	} {
		if existing, ok := messageKeys[id]; ok {
			panic(fmt.Sprintf("message id %d is already %q", id, existing))
		}
		messageKeys[id] = key
	}
}

// foeApproachEffects 是 `0B66h` 的群組 0Eh：overlay-24 entry 3 依 `53h 54h 58h 79h` 的順序，身上有
// 那個碼就以最早的節點叫一次它的處理常式（`014Dh`）。回傳 true 代表這一隻的行動已經結束
// （處理常式叫了 entry 34，`0B73h` 看 runtime `+3`）；吐息與噴酸不會同時長在一隻身上。
//
// target 是 `37B8h` 剛挑好的追擊目標（runtime `+0Ah`）。凝視把它石化或魅惑之後，接近迴圈
// `0C7Bh` 把它作廢，搆不到人時 `0D5Dh` 交給 `37B8h` 重挑；remake 的接近迴圈不逐步重挑，
// 所以凝視過後目標不再有效就在這裡重挑一次，挑不到就收工（spec 161）。
func (a *app) foeApproachEffects(state *tacticalState, mover uint8, target *uint8,
	pickTarget func() (uint8, error)) (bool, error) {
	gazed := false
	for _, code := range gamepack.ApproachEffectCodes {
		if !state.hasEffect(int(mover), code) {
			continue
		}
		ended := false
		var err error
		switch code {
		case gamepack.GazeStoneEffectCode:
			a.tacticalStatus(state, a.foeGazeStone(state, mover, *target))
			gazed = true
		case gamepack.GazeCharmEffectCode:
			if line := a.foeGazeCharm(state, mover, *target); line != "" {
				a.tacticalStatus(state, line)
			}
			gazed = true
		case gamepack.BreathEffectCode:
			ended, err = a.foeBreath(state, mover)
		case gamepack.AcidSpitEffectCode:
			ended, err = a.foeAcidSpit(state, mover, *target)
		}
		if err != nil || ended {
			return ended, err
		}
	}
	if !gazed {
		return false, nil
	}
	if _, ok := state.foeTarget(mover); ok {
		return false, nil
	}
	state.setFoeTarget(mover, 0)
	picked, err := pickTarget()
	if err != nil {
		return false, err
	}
	if picked == 0 {
		state.FoeLog = state.say(msgFoeNoTarget, a.combatantName(state, mover))
		state.endTurn(a.rollDice, false)
		return true, nil
	}
	*target = picked
	state.setFoeTarget(mover, picked)
	return false, nil
}

// foeBreath 是 overlay-22 `3092h`（gamepack/breath.go 有逐條位址）。
func (a *app) foeBreath(state *tacticalState, mover uint8) (bool, error) {
	index := int(mover)
	if gamepack.BreathSkipped(state.AttackPhase, func() int { return a.rollDice(1, 100) }) {
		return false, nil
	}
	// `30CDh`：entry 20(記錄, "Breathes!", 0Ah, 1)。
	notice := a.panelNotice(state, mover, state.say(msgFoeBreathes), noticeRowPanel, true)
	// `30EDh`：`DS:6A78h(33h, 1, …)` 是 AI 放閃電束的挑法（overlay-13 `20AEh`／`1E09h`），
	// 射程照閃電束算（`0723h`：`+2 + +3 × 等級`，龍沒有法師等級就是 4）。
	caster, ok := a.foeSpellcasterFor(state, mover)
	if !ok {
		caster = foeSpellcaster{party: -1}
	}
	targets, found, err := a.foeSpellTargets(state, mover, gamepack.SpellIDLightningBolt, caster)
	if err != nil {
		return false, err
	}
	if !found {
		// `1E09h` 二十次都挑不到時不寫 `DS:6CADh`／`6CAEh`，原版拿殘值吐出去（unknown，
		// 停止線）。remake 這一回合不吐，照常接近。
		return false, nil
	}
	self := state.Roster[mover]
	x, y := gamepack.BreathStart(int(self.X), int(self.Y), targets.X, targets.Y)
	// `315Ch`：overlay-24 entry 12（0FCCh）摘掉自己身上的隱形。
	state.Effects[index] = gamepack.DropInvisibility(state.Effects[index])
	damage := 0
	if index < len(state.MaxHitPoints) {
		damage = state.MaxHitPoints[index]
	}
	ray := gamepack.BreathRay(damage)
	// 吐息不是法術：`DS:6779h` 是 0（鏡影擋不下）。
	state.SpellDamage = spellDamageContext{}
	hits, err := a.castSpellRay(state, gamepack.CastEffect{Damage: damage, Ray: &ray}, x, y)
	state.SpellDamage = spellDamageContext{}
	if err != nil {
		return false, err
	}
	state.Effects[index] = gamepack.BreathAfterUse(state.Effects[index])
	state.FoeLog = notice + " " + fmt.Sprintf(a.text(msgCastArea), a.combatantName(state, mover), hits, damage)
	state.endTurn(a.rollDice, false)
	return true, nil
}
