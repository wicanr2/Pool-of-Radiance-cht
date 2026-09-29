package main

import (
	"fmt"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// 怪物接近之前的效果群組 0Eh 其餘三個碼（spec 161，issue #82）：`53h` 石化凝視、`54h` 魅惑凝視、
// `79h` 噴酸。規則常數在 internal/gamepack/approach_effects.go，派發在 breath.go 的
// foeApproachEffects。三支處理常式的目標都是自己 runtime `+0Ah`（`37B8h` 剛挑好的追擊目標）。

const (
	// msgFoeGazesStone 是 overlay-12 `1CA5h` "gazes..."（`53h`）。
	msgFoeGazesStone messageID = iota + 5400
	// msgFoeGazeReflected 是 `1CAEh` "reflects it!"。
	msgFoeGazeReflected
	// msgEffectStoned 是 `1CBBh` "is Stoned"。
	msgEffectStoned
	// msgFoeGazesCharm 是 `1E73h` "Gazes..."（`54h`）。
	msgFoeGazesCharm
	// msgEffectCharmed 是 `1E7Ch` "is charmed"。
	msgEffectCharmed
	// msgFoeSpitsAcid 是 `2C27h` "Spits Acid"（`79h`）。
	msgFoeSpitsAcid
	// msgEffectAnimated 是 overlay-22 `2037h` "is animated"（死靈術逐人那一句，spec 098）。
	msgEffectAnimated
)

func init() {
	for id, key := range map[messageID]string{
		msgFoeGazesStone:    "ui.foeGazesStone",
		msgFoeGazeReflected: "ui.foeGazeReflected",
		msgEffectStoned:     "ui.effectStoned",
		msgFoeGazesCharm:    "ui.foeGazesCharm",
		msgEffectCharmed:    "ui.effectCharmed",
		msgFoeSpitsAcid:     "ui.foeSpitsAcid",
		msgEffectAnimated:   "ui.effectAnimated",
	} {
		if existing, ok := messageKeys[id]; ok {
			panic(fmt.Sprintf("message id %d is already %q", id, existing))
		}
		messageKeys[id] = key
	}
}

// gazeTraceBudget 是 `54h` 交給 overlay-31 `0419h` 的預算。處理常式傳的是自己沒初始化的區域變數
// `[bp-2]`（`1F02h`）；那個位址在 `37B8h` 最後一次 `1087h` → 群組 0（吸血鬼帶 `7Eh`）的
// overlay-24 entry 1 裡存的是 `014Dh` 的 bp，一個堆疊位移（START.EXE 的初始 SP 是 2000h）。
// 所以上限 `預算 × 2 + 1` 遠大於盤面上任何一條線的成本（最長 49 格斜走 147），只剩地形擋不擋
// （strong inference，spec 161）。0FFh 與那個殘值在這張盤面上結果相同。
const gazeTraceBudget = 0xff

// foeGazeStone 是 overlay-12 entry 76（`1CC5h`，`53h`，BASILISK／MEDUSA）。沒有距離與視線
// 的判斷，也不結束行動。
//
//	1CCBh  6784h = 自己 runtime +0Ah（目標）
//	1CF0h  entry 20(自己, "gazes...", 0Ah, 0)；1D03h 組圖、1D42h 動畫（spec 166）
//	1D47h  entry 27(自己, 7Fh)：有才走目標的物品串列
//	1D8Ah  裝備中（+34h）而且 +2Fh／+30h／+31h 有一格是 76h（Mirror）→
//	1DC7h    entry 20(目標, "reflects it!", 0Ch, 0)、反向動畫、6784h = 自己
//	1E38h  entry 7(6784h, 1, 0)；沒過 → 005Ah(6784h, 7, "is Stoned")
func (a *app) foeGazeStone(state *tacticalState, mover, target uint8) string {
	line := a.panelNotice(state, mover, state.say(msgFoeGazesStone), noticeRowPanel, false)
	// `1D03h`／`1D42h`：自己往目標畫一道（槽 12h 的四格、45 毫秒；spec 166）。
	a.combatantMissile(state, mover, target, fourFrames(slotSpellBolt), missileFramesComposite, gazeMilliseconds)
	victim := target
	if state.hasEffect(int(mover), gamepack.GazeReflectableEffectCode) && state.ItemsOf != nil &&
		gamepack.MirrorReadied(state.ItemsOf(int(target))) {
		line += " " + a.panelNotice(state, target, state.say(msgFoeGazeReflected), noticeRowLost, false)
		// `1E11h`：反射回來，從目標往自己畫同一道。
		a.combatantMissile(state, target, mover, fourFrames(slotSpellBolt), missileFramesComposite, gazeMilliseconds)
		victim = mover
	}
	if a.savedAgainstCategory(state, victim, gamepack.GazeStoneSaveCategory, 0) {
		return line
	}
	line += " " + a.panelNotice(state, victim, state.say(msgEffectStoned), noticeRowPanel, false)
	a.statusOffBoard(state, victim, gamepack.StonedState)
	return line
}

// foeGazeCharm 是 overlay-12 entry 77（`1E87h`，`54h`，VAMPIRE）。不結束行動。
//
//	1ED0h  overlay-13 `1087h`(自己, 目標) 不過就返回（吸血鬼的 `7Eh` 認鏡子與聖徽）
//	1F0Bh  overlay-31 `0419h` 從自己走向目標，地形擋住就返回（預算見 gazeTraceBudget）
//	1F27h  entry 20(自己, "Gazes...", 0Ah, 0)；1F3Ah 組圖、1F79h 動畫（spec 166）
//	1F7Eh  DS:6779h = 0Ah（魅惑人類）
//	1FD4h  overlay-24 entry 20(目標, 0Bh, 持續 0, (自己 +10Eh << 7) + 0Ch, 有收尾 1, 規則 1,
//	         entry 7(目標, 4, FEh), "is charmed")
//	1FE9h  entry 27(目標, 0Bh) 找得到 → entry 1 以模式 0 叫 `0Bh` 的處理常式：倒戈
func (a *app) foeGazeCharm(state *tacticalState, mover, target uint8) string {
	if state.attackVetoed(target, mover) || !state.withinArea(mover, target, gazeTraceBudget) {
		return ""
	}
	line := a.panelNotice(state, mover, state.say(msgFoeGazesCharm), noticeRowPanel, false)
	// `1F3Ah`／`1F79h`：自己往目標畫一道（槽 12h 的四格、45 毫秒；spec 166）。
	a.combatantMissile(state, mover, target, fourFrames(slotSpellBolt), missileFramesComposite, gazeMilliseconds)
	// entry 20 的引數由左往右求值：豁免（entry 7）在 entry 20 的群組 9 之前擲。
	saved := a.savedAgainstCategory(state, target, gamepack.GazeCharmSaveCategory, gamepack.GazeCharmSaveModifier)
	// 群組 9 的魔法抗性用 `010Ah:00D4h(DS:6779h)`：魅惑人類是法師法術，讀 `DS:5CF0h`（吸血鬼）的
	// `+9Bh`。處理常式沒寫 `DS:6777h`，免疫檢查的傷害旗標是 0（`08BCh` 沒有經過）。
	if a.unaffectedBySpellEffect(state, target, gamepack.CharmPersonEffectCode, a.recordMagicUserLevel(state, mover)) {
		return line
	}
	if saved {
		// `1689h`：豁免成功而且規則是 1 → "is Unaffected"。
		return line + " " + a.panelNotice(state, target, state.say(msgCastUnaffected), noticeRowPanel, true)
	}
	state.applyCharm(int(target), gamepack.GazeCharmLevel)
	return line + " " + a.panelNotice(state, target, state.say(msgEffectCharmed), noticeRowPanel, true)
}

// recordMagicUserLevelOffset 是記錄 `+9Bh`（`+96h + 5`，法師等級）。
const recordMagicUserLevelOffset = 0x9b

// recordMagicUserLevel 是怪物記錄的 `+9Bh`（法師等級，spec 098〈施法者等級〉）。隊員是 0。
func (a *app) recordMagicUserLevel(state *tacticalState, index uint8) int {
	if int(index) < len(state.PartySlot) && state.PartySlot[index] >= 0 {
		return 0
	}
	if monster, ok := a.stagedMonsterFor(int(index), state.PartySlot, state.Friendly); ok {
		return int(monster.Record.Raw[recordMagicUserLevelOffset])
	}
	return 0
}

// foeAcidSpit 是 overlay-12 entry 115（`2C32h`，`79h`，AHNKHEG）。噴了就結束行動，而且只噴一次。
//
//	2C53h  Roll(1, 100) > 25 → 返回
//	2C6Dh  overlay-25 entry 33（`2591h`）量到目標的距離 >= 4 → 返回
//	2C7Fh  entry 34（`266Dh`）結束行動
//	2C97h  entry 20(自己, "Spits Acid", 0Ah, 1)；2CAAh 組圖、2CE9h 動畫（spec 166）
//	2CFCh  overlay-24 entry 9(8, 4)（骰數記進 DS:677Ah）
//	2D13h  entry 7(目標, 3, 0)
//	2D19h  overlay-24 entry 19(目標, 傷害, 規則 2, 豁免)
//	2D2Dh  entry 2(自己, 79h, 這個節點)；2D41h entry 2(自己, 50h, 空)
func (a *app) foeAcidSpit(state *tacticalState, mover, target uint8) (bool, error) {
	if a.rollDice(1, 100) > gamepack.AcidSpitPercent {
		return false, nil
	}
	distance, err := state.entry33Distance(mover, target)
	if err != nil {
		return false, err
	}
	if distance >= gamepack.AcidSpitReach {
		return false, nil
	}
	notice := a.panelNotice(state, mover, state.say(msgFoeSpitsAcid), noticeRowPanel, true)
	// `2CAAh`／`2CE9h`：entry 24 以槽 17h 組四格，只用第一格、30 毫秒（spec 166）。
	a.combatantMissile(state, mover, target, fourFrames(slotHurtSparkle), 1, acidSpitMilliseconds)
	damage := 0
	for die := 0; die < gamepack.AcidSpitDiceCount; die++ {
		damage += a.rollDice(1, gamepack.AcidSpitDiceSides)
	}
	a.diceCount = gamepack.AcidSpitDiceCount
	saved := a.savedAgainstCategory(state, target, gamepack.AcidSpitSaveCategory, 0)
	// 處理常式不寫 `DS:6779h`／`6777h`；remake 當成 0（spec 161，停止線）。
	state.SpellDamage = spellDamageContext{}
	damage = a.spellDamageAfterEffects(state, target, 0, damage)
	if saved {
		damage = gamepack.DamageAfterSave(gamepack.AcidSpitSaveRule, damage)
	}
	a.applySpellDamage(state, target, damage)
	state.Effects[mover] = gamepack.AcidSpitAfterUse(state.Effects[mover])
	state.FoeLog = notice + " " + state.Status
	state.endTurn(a.rollDice, false)
	return true, nil
}

// entry33Distance 是 overlay-25 entry 33（`2591h`）：`25B0h` 把 `DS:6674h` 的 `+6` 立起來（不判地形）、
// 以預算 FFh、不指定朝向叫 `0912h` 建名單，找到目標那一筆取成本除以二（`264Bh`）。
// 找不到時迴圈停在最後一筆（`2628h` 的 `jae`），取的是那一筆的成本。
func (state *tacticalState) entry33Distance(from, to uint8) (int, error) {
	if int(from) >= len(state.Roster) {
		return 0, nil
	}
	snapshot, err := state.tacticalSnapshot()
	if err != nil {
		return 0, err
	}
	snapshot.Map.IgnoreTerrain = true
	self := state.Roster[from]
	cells, err := combat.NearbyCells(combat.NearbyRequest{
		Map: snapshot.Map, Classes: snapshot.Classes, Cells: snapshot.Cells,
		Class: self.FootprintClass, Facing: combat.DirectionUnset, Budget: 0xff,
		BaseX: self.X, BaseY: self.Y,
	})
	if err != nil || len(cells) == 0 {
		return 0, err
	}
	pick := cells[len(cells)-1]
	for _, cell := range cells {
		if cell.CombatantIndex == to {
			pick = cell
			break
		}
	}
	return int(pick.Cost) / 2, nil
}

// statusOffBoard 是 overlay-12 `005Ah(記錄, 狀態, 訊息)` 落在盤面上（訊息由呼叫端印）：狀態已經是
// 6／7／8 就不動（`008Ch..009Bh`）；否則 `+10Ch = 狀態`、`+10Dh = 0`、`+11Bh`（目前生命）= 0，
// `00C2h`／`00D0h` 的 entry 13 與群組 13 走 combatantDown，還躺著就記進屍體表（`00E6h`）。
// 回傳這一下有沒有寫下去。
func (a *app) statusOffBoard(state *tacticalState, target uint8, status uint8) bool {
	index := int(target)
	if index >= len(state.HitPoints) || index >= len(state.States) || alreadyGone(state.States[index]) {
		return false
	}
	if member := a.partyMemberAt(state, target); member != nil {
		member.Status, member.CurrentHP = status, 0
		syncTrainedLibraryCharacter(&a.state, *member)
	}
	state.HitPoints[index] = 0
	state.rememberFootprint(index)
	state.Roster[index].FootprintClass = 0
	state.Scores[index], state.States[index] = 0, status
	if index < len(state.DyingCounters) {
		state.DyingCounters[index] = 0
	}
	a.combatantDown(state, index, deadOverkill)
	return true
}
