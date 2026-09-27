package main

import (
	"fmt"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/creation"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// 倒下時的群組 13、再生、群組 5 其餘的碼與麻痺（spec 155，issue #113）。規則在
// internal/gamepack/death_effects.go 與 melee_target_effects.go。

const (
	// msgEffectParalyzed 是 overlay-12 `15EAh` "is Paralyzed"（`15F7h`）。
	msgEffectParalyzed messageID = iota + 4600
	// msgEffectFallsDead 是 overlay-12 `25EEh` "Falls dead"（`5Fh` 的收尾）。
	msgEffectFallsDead
)

func init() {
	for id, key := range map[messageID]string{
		msgEffectParalyzed: "ui.effectParalyzed",
		msgEffectFallsDead: "ui.effectFallsDead",
	} {
		if existing, ok := messageKeys[id]; ok {
			panic(fmt.Sprintf("message id %d is already %q", id, existing))
		}
		messageKeys[id] = key
	}
}

// deadOverkill 是傳給 combatantDown 的「沒有打穿點數可言」：`005Ah` 直接寫狀態 6，`63h` 看到的
// 狀態不是 4 也不是 5，不會站起來。
const deadOverkill = -1

// combatantDown 是三個倒下入口共用的那一段（overlay-13 `05F5h..0608h`、overlay-24 `1610h..1620h`、
// overlay-12 `00C2h..00D0h`）：overlay-24 entry 13（`1004h`）摘 `DS:0C28h` 那十六個碼，再派發群組 13。
// 呼叫端已經把那一格記成倒下（體型 0，對應 `+10Dh = 0`）。overkill 是倒下那一下打穿了幾點
// （生命值扣到的負數取正），`005Ah` 那一條傳 deadOverkill。
func (a *app) combatantDown(state *tacticalState, index int, overkill int) {
	if index <= 0 || index >= len(state.Effects) {
		return
	}
	// ov32 entry 20 的 `0F55h..0FF9h`：記進屍體表（`6673h` 加一，`6634h` 那一筆是記錄與 X／Y）。
	state.Corpses = append(state.Corpses, index)
	// entry 13：每個碼摘最早掛上的那一個；`+4` 立著就先以模式 1 叫處理常式（entry 2 `0028h`）。
	for _, code := range gamepack.DeathStrippedEffects {
		list := state.Effects[index]
		at, ok := list.IndexOf(code)
		if !ok {
			continue
		}
		node := list[at]
		state.Effects[index] = list.RemoveAt(at)
		if code == gamepack.CharmPersonEffectCode {
			// `0Bh`（entry 14 `040Ah`）模式 1：`+10Eh` 還原成節點 `+3` 位元 6 記的那一邊。魅惑術的節點
			// 由 `08BCh` 帶效果參數 1 掛上（spec 098），`+4` 立著。
			state.releaseCharmFrom(index, list)
			continue
		}
		if node.NeedsTeardown() {
			state.effectTeardown(index, node)
		}
	}
	// 群組 13 的順序是 `63h 64h 67h 4Bh 4Ah`。
	if state.hasEffect(index, gamepack.BoarRallyEffectCode) {
		// `63h`（`2727h`）：打穿 0..5 點 → entry 22 以 6 − 打穿點數站起來；站得起來才掛 `5Fh`、摘 `63h`。
		if rally := gamepack.BoarRallyHitPoints(overkill); rally > 0 && a.poisonRecover(state, index, rally) {
			state.Effects[index] = gamepack.BoarRallied(state.Effects[index], a.rollDice)
		}
	}
	// `64h`（`27D0h`）：讀的是這一下的 `DS:6777h`。
	state.Effects[index], _ = gamepack.TrollRevival(state.Effects[index], state.SpellDamage.Flags, a.rollDice)
	// `67h`（`28B8h`）：ov32 entry 22 換圖、ov24 entry 11(記錄, 3, "Turns gaseous and escapes")——
	// entry 11 一進來 `0F19h..0F24h` 看 `+10Dh`，是 0 就整支返回。三個派發點都在 `+10Dh = 0` 之後，
	// 所以吸血鬼的氣化逃走在 DOS 版走不到，這裡什麼都不做。
	// `4Bh`／`4Ah`（`178Bh`、overlay-13 `3A55h`）：模式 0 而一方已倒 → 摘夥伴的 `3Ah`、摘自己。
	// entry 13 已經先摘掉第一個，remake 也沒有掛這兩個碼的來源（群組 2 的 `4Ch`／`49h` 沒接）。
}

// deathTeardown 是 `3Bh`／`5Fh`／`66h` 到期（或被摘）時的收尾，隊員與怪物同一支（三支都不看
// `+10Eh`）。
func (a *app) deathTeardown(state *tacticalState, index int, node gamepack.EffectNode) {
	if !node.NeedsTeardown() || !gamepack.IsDeathTeardownEffect(node.Code) ||
		index <= 0 || index >= len(state.Effects) || index >= len(state.Roster) {
		return
	}
	switch node.Code {
	case gamepack.RegenerationPendingEffectCode:
		// `147Ch`：entry 10(記錄, 62h, 0, FFh, 0)。
		state.Effects[index] = gamepack.RegenerationBegins(state.Effects[index])
	case gamepack.BoarFallsEffectCode:
		// `25F9h`：`+10Dh` 不是 0（還站著）→ `005Ah(記錄, 6, "Falls dead")`。
		if state.Roster[index].FootprintClass == 0 {
			return
		}
		a.tacticalStatus(state, fmt.Sprintf(a.text(msgEffectFallsDead), a.combatantName(state, uint8(index))))
		a.poisonKill(state, uint8(index))
	case gamepack.TrollRisesEffectCode:
		// `285Fh`：entry 22(記錄, `+32h`)；站不起來 → `0021h(記錄, 66h, 節點 +3, 1)`。
		if index >= len(state.MaxHitPoints) {
			return
		}
		if !a.poisonRecover(state, index, state.MaxHitPoints[index]) {
			state.Effects[index] = gamepack.TrollRisesRetry(state.Effects[index], node.Magnitude())
		}
	}
}

// regenerate 是群組 19 的 `62h`（overlay-08 `08A3h`，回合收尾在 entry 4 減計時之前）。
func (state *tacticalState) regenerate(index int) {
	if index <= 0 || index >= len(state.Effects) || index >= len(state.HitPoints) ||
		index >= len(state.MaxHitPoints) {
		return
	}
	state.HitPoints[index] = gamepack.Regenerate(state.Effects[index], state.HitPoints[index],
		state.MaxHitPoints[index])
}

// raceCode 是那一格記錄的 `+2Eh`（種族）：隊員依建角的種族、NPC 讀它帶的記錄、怪物讀開打時抄下的那一格。
func (a *app) raceCode(state *tacticalState, index uint8) uint8 {
	if member := a.partyMemberAt(state, index); member != nil {
		if member.NPC && len(member.Record) > recordRaceOffset {
			return member.Record[recordRaceOffset]
		}
		code, _ := creation.RaceDOSCode(member.RaceID)
		return code
	}
	if int(index) < len(state.SleepFlag) {
		return state.SleepFlag[index]
	}
	return 0
}

// recordRaceOffset 是角色記錄的 `+2Eh`（spec 145）。
const recordRaceOffset = 0x2e

// meleeTargetGroup 是群組 5（overlay-13 `022Ch`）在鏡影之後的部分：先 `29h`（normalMissileAvoided），
// 擋掉了就不往下；再跑其後的十二個碼（gamepack.MeleeTargetEffects）。處理常式看的行動者是
// `DS:5CF0h`，也就是 state.Mover（反應攻擊時是正在移動的那一個）。回傳改過的傷害與這一下有沒有被擋掉。
func (a *app) meleeTargetGroup(state *tacticalState, attacker, target uint8, damage int) (int, bool, error) {
	if missed, err := a.normalMissileAvoided(state, attacker, target); err != nil || missed {
		return 0, missed, err
	}
	if int(target) >= len(state.Effects) {
		return damage, false, nil
	}
	actor := state.Mover
	gear, items, _, err := a.missileGear(state, actor)
	if err != nil {
		return damage, false, err
	}
	input := gamepack.MeleeTargetEffects{
		Effects:     state.Effects[target],
		DamageFlags: state.SpellDamage.Flags,
		Dice:        a.diceCount,
		Roll:        a.rollDice,
		Types:       a.itemTypes,
		ActorRace:   a.raceCode(state, actor),
	}
	if missile := gear.MissileOf(); missile >= 0 && missile < len(items) {
		input.Hitting = items[missile].Raw
	}
	if gear.Weapon >= 0 && gear.Weapon < len(items) {
		input.Wielded = items[gear.Weapon].Raw
	}
	if int(actor) < len(state.HitDice) {
		input.ActorHitDice = state.HitDice[actor]
	}
	input.Distance, _ = state.tacticalRange(target, actor)
	outcome := input.Apply(damage)
	state.Effects[target] = outcome.Effects
	a.diceCount = outcome.Dice
	return outcome.Damage, outcome.Avoided, nil
}

// paralysisSpecialAttack 是群組 2／3 裡的 `43h 44h 45h`（overlay-12 `15F7h`）：攻擊者身上每帶一個碼，
// 目標擲一次類別 0、修正 0 的豁免；失敗就印 "is Paralyzed"、掛 `34h`（持續照碼、`+3` 0Ch、不收尾）。
// form 是攻擊形態（1 → 群組 2、2 → 群組 3）。
func (a *app) paralysisSpecialAttack(state *tacticalState, attacker, target uint8, form int) {
	if int(attacker) >= len(state.Effects) || int(target) >= len(state.Effects) {
		return
	}
	durations := gamepack.ParalysisAttacks(state.Effects[attacker], form, a.raceCode(state, target), a.rollDice)
	for _, duration := range durations {
		if a.savedAgainstCategory(state, target, 0, 0) {
			continue
		}
		state.Effects[target] = state.Effects[target].Append(gamepack.NewParalysisNode(duration))
		a.tacticalStatus(state, fmt.Sprintf(a.text(msgEffectParalyzed), a.combatantName(state, target)))
	}
}

// corpseAt 是 Manual 瞄準的 `2F9Fh..3002h`：游標那一格沒有站著的人，而地形類別是 1Fh（屍體；被雲蓋住
// 時是 1Eh）→ 從 `DS:6634h` 那張表找 X／Y 相符的，後面的蓋掉前面的。回傳那一格的戰鬥員序號，沒有回 0。
func (state *tacticalState) corpseAt(x, y int) uint8 {
	if code, err := state.Grid.TerrainAt(x, y); err != nil || code == gamepack.CloudTerrain {
		return 0
	}
	found := 0
	for _, index := range state.Corpses {
		if index <= 0 || index >= len(state.Roster) || state.Roster[index].FootprintClass != 0 {
			continue
		}
		if int(state.Roster[index].X) == x && int(state.Roster[index].Y) == y {
			found = index
		}
	}
	return uint8(found)
}

// forgetCorpse 是 ov32 entry 21 以 `[bp+6] = 1` 被 entry 22 叫的那一段：站起來的人從屍體表上摘掉。
func (state *tacticalState) forgetCorpse(index int) {
	kept := state.Corpses[:0]
	for _, corpse := range state.Corpses {
		if corpse != index {
			kept = append(kept, corpse)
		}
	}
	state.Corpses = kept
}

// healOnBoard 是 overlay-24 entry 21（`175Dh`）以旗標 0 落在盤面上（法術三個呼叫端都推 0）：
// 狀態 {0, 1, 4, 5} 才治、身上有 `32h` 不治、補到 `+32h` 為止、倒著的瀕死改成昏迷（戰鬥中不站起來）。
// 回傳有沒有治（entry 21 的回傳值）。
func (a *app) healOnBoard(state *tacticalState, index uint8, amount int) bool {
	if int(index) >= len(state.HitPoints) || int(index) >= len(state.States) ||
		int(index) >= len(state.Effects) || int(index) >= len(state.Roster) {
		return false
	}
	// 治具可能沒有填上限；真的盤面開打時每一格都有（tactical.go）。
	maximum := 0xff
	if int(index) < len(state.MaxHitPoints) && state.MaxHitPoints[index] > 0 {
		maximum = state.MaxHitPoints[index]
	}
	result := gamepack.HealByEntry21(state.Effects[index], state.States[index],
		state.Roster[index].FootprintClass == 0, state.HitPoints[index], maximum, amount, true)
	state.HitPoints[index], state.States[index] = result.HitPoints, result.State
	return result.Healed
}
