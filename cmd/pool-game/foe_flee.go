package main

import (
	"fmt"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// 士氣與逃跑（spec 096〈entry 8〉〈逃跑〉，issue #74）。規則本身在
// internal/gamepack/morale.go，這一檔是戰場那一層：
//
//	overlay-09 entry 1  `00B5h`  entry 8 士氣 → `00C8h` 印 `flees in panic`
//	overlay-09 entry 5  `0B9Fh`  runtime +14h 立著、+6 > 0、0 < +3 < 14h → 反覆叫 07E8h
//	overlay-09 `07E8h`  `08C5h`  逃跑那一支：模式 = 骰(1,2)，基準方向由隊伍朝向算
//	overlay-09 `0955h`           逃跑中踏出盤面 → overlay-13 entry 7（`0C6Ch`）
//	overlay-24 entry 11 `0F00h`  離開盤面：`+10Dh = 0`、`+10Ch` = 3／4
//
// runtime `+14h`（正在逃）在 entry 8 開頭清成 0、當回合重新判定，所以它只活在
// 一隻怪物的一個回合裡，這裡用 foeTurn 的區域變數表示，不另外存。

const (
	msgFoeForcedToFlee messageID = iota + 2740
	msgFoeFleesInPanic
	msgFoeSurrenders
	msgFoeGotAway
	msgFoeEscapeBlocked
	msgFoeFled
)

func init() {
	for id, key := range map[messageID]string{
		msgFoeForcedToFlee:  "ui.foeForcedToFlee",
		msgFoeFleesInPanic:  "ui.foeFleesInPanic",
		msgFoeSurrenders:    "ui.foeSurrenders",
		msgFoeGotAway:       "ui.foeGotAway",
		msgFoeEscapeBlocked: "ui.foeEscapeBlocked",
		msgFoeFled:          "ui.foeFled",
	} {
		if existing, ok := messageKeys[id]; ok {
			panic(fmt.Sprintf("message id %d is already %q", id, existing))
		}
		messageKeys[id] = key
	}
}

// foeMorale 是士氣判定跨行動要記的東西。
type foeMorale struct {
	// Raw 是每一格記錄的 `+84h`，Intelligence 是 `+11h`（建 roster 時記下）。
	// 玩家建的角色 `+84h` 是 0（spec 067），不做士氣判定。
	Raw          map[int]uint8
	Intelligence map[int]uint8
	// Side 是 `DS:6D22h`：敵方整體還剩幾成生命（overlay-13 entry 25），戰鬥佈置時
	// （overlay-10 `2030h`）與每個回合收尾（overlay-08 `087Dh`）重算，回合中間不變。
	Side uint8
	// Party 是隊伍的 `+58Ch`（ECL `@6DC6`，遭遇腳本寫的），佈置時夾到 100。
	Party uint16
}

// rememberMorale 在建 roster 時記下那一格的 `+84h` 與 `+11h`。
func (state *tacticalState) rememberMorale(index int, raw, intelligence uint8) {
	if state.Morale.Raw == nil {
		state.Morale.Raw = map[int]uint8{}
		state.Morale.Intelligence = map[int]uint8{}
	}
	state.Morale.Raw[index] = raw
	state.Morale.Intelligence[index] = intelligence
}

// setupMorale 是戰鬥佈置那兩步：overlay-10 `1F7Fh` 把隊伍 `+58Ch` 夾到 100
// （寫回 ECL 記憶體），`2030h` 算第一次 `DS:6D22h`。
func (a *app) setupMorale(state *tacticalState) {
	if a.eventMachine != nil {
		if a.eventMachine.Memory[partyMoraleAddress] > gamepack.PartyMoraleCap {
			a.eventMachine.Memory[partyMoraleAddress] = gamepack.PartyMoraleCap
		}
		state.Morale.Party = uint16(a.eventMachine.Memory[partyMoraleAddress])
	}
	state.refreshSideMorale()
}

// partyMoraleAddress 是 ECL `@6DC6`，換成引擎位移就是隊伍的 `+58Ch`
// （`(2A00h + 6DC6h × 2) mod 10000h`，spec 136）。
const partyMoraleAddress = 0x6DC6

// refreshSideMorale 是 overlay-13 entry 25（`285Dh`）：沿整條戰鬥者串列，敵方的
// 生命上限全部加進分母、還在場的目前生命加進分子。分母為 0 時原版不寫。
func (state *tacticalState) refreshSideMorale() {
	var current, maximum uint16
	for index := 1; index < len(state.Roster); index++ {
		if index >= len(state.Friendly) || state.Friendly[index] {
			continue
		}
		if state.Roster[index].FootprintClass != 0 && index < len(state.HitPoints) {
			current += uint16(uint8(state.HitPoints[index]))
		}
		if index < len(state.MaxHitPoints) {
			maximum += uint16(uint8(state.MaxHitPoints[index]))
		}
	}
	if value, ok := gamepack.SideMorale(current, maximum); ok {
		state.Morale.Side = value
	}
}

// speedOf 是 overlay-13 `0123h`（腳程的初值）除以 2。remake 的腳程初值與 startRound
// 同一支，同樣沒有加隊伍的 `+6E4h` 與效果群組 12h（spec 053）。
func (state *tacticalState) speedOf(index int) int {
	if index <= 0 || index >= len(state.BaseMovement) {
		return 0
	}
	return int(combat.InitialMovementBudgetBeforeEffects(state.BaseMovement[index], false, 0)) / 2
}

// fastestOpponent 是 overlay-13 entry 26（`28E7h`）：對面（`+10Eh` 是對立陣營值）
// 還在場的（`+10Dh`）裡，腳程 ÷ 2 最大的那一個。
func (state *tacticalState) fastestOpponent(mover uint8) int {
	fastest := 0
	for index := 1; index < len(state.Roster) && index < len(state.Friendly); index++ {
		if state.Friendly[index] == state.Friendly[mover] || state.Roster[index].FootprintClass == 0 {
			continue
		}
		if speed := state.speedOf(index); speed > fastest {
			fastest = speed
		}
	}
	return fastest
}

// foeMoralePhase 是 overlay-09 entry 8（`10DDh`），接在戰術模式擲骰之後、用物品之前
// （entry 1 `00B5h`）。回傳 fleeing 代表這一回合 runtime `+14h` 立著；acted 代表
// 投降了，這一隻的回合已經結束。整支不擲骰。
func (a *app) foeMoralePhase(state *tacticalState, mover uint8) (fleeing, acted bool) {
	index := int(mover)
	// `10F4h`：overlay-24 entry 14 先摘掉 4Ah／4Bh。
	for _, code := range gamepack.MoraleClearedEffects {
		state.removeEffect(index, code)
	}
	outcome := gamepack.ResolveMorale(gamepack.MoraleCheck{
		Turned:       state.Undead.Turned[index],
		Raw:          state.Morale.Raw[index],
		Boost:        state.hasEffect(index, gamepack.MoraleBoostEffectCode),
		Drop:         state.hasEffect(index, gamepack.MoraleDropEffectCode),
		HitPoints:    valueAt(state.HitPoints, index),
		MaxHitPoints: valueAt(state.MaxHitPoints, index),
		SideMorale:   state.Morale.Side,
		PartyMorale:  state.Morale.Party,
		FoeSide:      index < len(state.Friendly) && !state.Friendly[index],
		Intelligence: state.Morale.Intelligence[index],
		OwnSpeed:     state.speedOf(index),
		FastestFoe:   state.fastestOpponent(mover),
	})
	switch outcome {
	case gamepack.MoraleForcedFlee:
		// `1116h`：`is forced to flee`。
		a.tacticalStatus(state, a.panelNotice(state, mover, state.say(msgFoeForcedToFlee), noticeRowPanel, true))
		return true, false
	case gamepack.MoraleFlees:
		// `122Fh..1252h` 再摘一次 4Ah／4Bh；entry 1 `00C8h` 印 `flees in panic`
		// （`+14h` 立著而 `+10h` 沒立）。
		for _, code := range gamepack.MoraleClearedEffects {
			state.removeEffect(index, code)
		}
		a.tacticalStatus(state, a.panelNotice(state, mover, state.say(msgFoeFleesInPanic), noticeRowPanel, true))
		return true, false
	case gamepack.MoraleSurrenders:
		// `1263h..128Bh`：overlay-24 entry 11(記錄, 4, "Surrenders")，再 entry 34。
		state.leaveBoard(mover, gamepack.SurrenderedState)
		state.FoeLog = a.panelNotice(state, mover, state.say(msgFoeSurrenders), noticeRowPanel, true)
		a.tacticalStatus(state, state.FoeLog)
		state.endTurnAfterAction(a.rollDice)
		return false, true
	}
	return false, false
}

func valueAt(values []int, index int) int {
	if index < 0 || index >= len(values) {
		return 0
	}
	return values[index]
}

// leaveBoard 是 overlay-24 entry 11（`0F00h`）加上 `1004h`：從戰術地圖收掉、
// `+10Dh = 0`、`+10Ch` = status；不是 3（逃掉）就把生命值寫 0（`0F73h`）。
// 之後摘掉十五個戰鬥用的效果代碼（`1004h`，每個摘第一個節點）。
func (state *tacticalState) leaveBoard(index uint8, status uint8) {
	state.rememberFootprint(int(index))
	state.Roster[index].FootprintClass = 0
	if int(index) < len(state.Scores) {
		state.Scores[index] = 0
	}
	if int(index) < len(state.States) {
		state.States[index] = status
	}
	if status != gamepack.FledState && int(index) < len(state.HitPoints) {
		state.HitPoints[index] = 0
	}
	for _, code := range gamepack.EscapeStrippedEffects {
		state.removeEffect(int(index), code)
	}
}

// foeFleeRun 是 entry 5 開場清掉、接近迴圈與逃跑迴圈共用的幾個值：
// `DS:439Eh`（上一步的方向）、`DS:439Fh`（卡住次數）、runtime `+15h`（模式）與
// runtime `+0Ah`（目標）。
type foeFleeRun struct {
	mode          int
	lastDirection uint8
	stuck         int
	target        uint8
	steps         int
}

// foeFleeScoreLimit 是逃跑迴圈對先攻分數的上界（`0BC5h`：`+3 < 14h`）。
const foeFleeScoreLimit = 0x14

// foeFleeGuard 只是讓迴圈有終點：原版的逃跑迴圈靠腳程遞減與卡住三次收掉，
// 一次呼叫最多走到腳程用完，這個數字遠大於任何一場實際會跑到的次數。
const foeFleeGuard = 256

// foeFleeLoop 是 entry 5 的 `0B9Fh..0BD6h`：runtime `+14h` 立著、腳程 `+6 > 0`、
// 先攻分數 `0 < +3 < 14h` 時一直叫 `07E8h`。回傳 ended 代表這一隻的回合已經結束
// （entry 6／entry 34 已經叫過）；沒結束的（腳程剛好用完、卡住三次）照原版回到
// 接近迴圈。
func (a *app) foeFleeLoop(state *tacticalState, mover uint8, run *foeFleeRun,
	pickTarget func() (uint8, error)) (bool, error) {
	for guard := 0; guard < foeFleeGuard; guard++ {
		score := uint8(0)
		if int(mover) < len(state.Scores) {
			score = state.Scores[mover]
		}
		if state.Budget() == 0 || score == 0 || score >= foeFleeScoreLimit {
			return false, nil
		}
		ended, stop, err := a.foeFleeStep(state, mover, run, pickTarget)
		if ended || err != nil {
			return ended, err
		}
		if stop {
			// 卡住三次（`0A49h`）：腳程已經清成 0，迴圈的條件自己不成立。
			return false, nil
		}
	}
	return false, nil
}

// foeFleeStep 是 `07E8h` 逃跑的那一支。ended：這一隻的回合結束了；stop：這一次
// `07E8h` 回來時 `[bp-5]` 立著、但回合沒有結束（卡住三次）。
func (a *app) foeFleeStep(state *tacticalState, mover uint8, run *foeFleeRun,
	pickTarget func() (uint8, error)) (ended, stop bool, err error) {
	// `084Dh`：腳程 ÷ 2 不到一步就交給 entry 6。
	if state.Budget()/2 == 0 {
		a.foeFleeEnd(state, mover, run)
		return true, false, nil
	}
	// `08C5h`：模式 = 骰(1,2)，直接寫回 runtime `+15h`。
	run.mode = a.rollDice(1, 2)
	state.setTacticMode(mover, run.mode)
	partySide := int(mover) < len(state.Friendly) && state.Friendly[mover]
	base := gamepack.FleeBaseDirection(a.spawn.Facing, partySide)

	snapshot, err := state.tacticalSnapshot()
	if err != nil {
		return false, false, err
	}
	// `092Ah`：步 1..5 依序試，第一個進得去的就走；逃跑中踏到盤面外就交給
	// overlay-13 entry 7（`0955h`）。
	direction, found := uint8(0), false
	for step := 1; step <= gamepack.TacticSteps && !found; step++ {
		candidate, err := gamepack.DefaultTacticOffsets.Direction(run.mode, step, base)
		if err != nil {
			return false, false, err
		}
		_, class, err := combat.ProbeDestination(snapshot, mover, candidate)
		if err != nil {
			return false, false, err
		}
		if class == combat.OffBoardDestinationClass {
			// `066Eh` 先看盤面外（`06D5h`），比有沒有人擋著更早。
			a.foeLeaveCombat(state, mover, run)
			return true, false, nil
		}
		enterable, err := state.foeStepEnterable(snapshot, mover, candidate)
		if err != nil {
			return false, false, err
		}
		if enterable {
			direction, found = candidate, true
		}
	}

	// `09E7h..0A88h`：卡住的規則與接近時同一段。
	reverse := (int(direction) + combat.DirectionCount/2) % combat.DirectionCount
	if !found || int(run.lastDirection) == reverse {
		run.mode = gamepack.NextTacticMode(run.mode)
		state.setTacticMode(mover, run.mode)
		run.stuck++
		switch {
		case run.stuck > 2:
			// `0A50h`：腳程清成 0，`[bp-5] = 1`，但沒有叫 entry 34。
			state.setFoeTarget(mover, 0)
			state.Budgets[mover] = 0
			return false, true, nil
		case run.stuck == 2:
			// `0A37h`／`0A63h`：忘掉目標，`37B8h(記錄, 0FFh, 1, 0)` 重挑，挑不到交給 entry 6。
			state.setFoeTarget(mover, 0)
			picked, err := pickTarget()
			if err != nil {
				return false, false, err
			}
			if picked == 0 {
				a.foeFleeEnd(state, mover, run)
				return true, false, nil
			}
			run.target = picked
			state.setFoeTarget(mover, picked)
		}
		if !found {
			// `0A88h`：步 = 6，這一次不動。
			return false, false, nil
		}
	}

	budget, err := combat.SpendMovementStep(state.Budget(), direction)
	if err != nil {
		return false, false, err
	}
	// `0AB3h` 轉向、`0ACBh` overlay-13 entry 6（離開威脅區的反應攻擊），被打倒
	// （`0AD3h` 看 `+10Dh`）就交給 entry 34。
	down, err := a.disengageReactions(state, mover, direction)
	if err != nil {
		return false, false, err
	}
	if down {
		a.foeFleeEnd(state, mover, run)
		return true, false, nil
	}
	here := state.Roster[mover]
	x, y, err := combat.AdvanceTacticalCoordinate(here.X, here.Y, direction)
	if err != nil {
		return false, false, err
	}
	state.Roster[mover].X, state.Roster[mover].Y = x, y
	state.Budgets[mover] = budget
	run.lastDirection = direction
	run.steps++
	return false, false, nil
}

// foeLeaveCombat 是 overlay-13 entry 7（`0C6Ch`），由 `07E8h` 在逃跑中踏出盤面時叫：
// 逃掉就 overlay-24 entry 11(記錄, 3, "Got Away")，逃不掉印 `Escape is blocked`，
// 兩者都叫 entry 34（結束回合、恆回 1）；`07E8h` 接著在 `097Fh` 把腳程與 `+14h` 清掉。
func (a *app) foeLeaveCombat(state *tacticalState, mover uint8, run *foeFleeRun) {
	opponents, err := state.opposingWithin(mover, 0xFF, false)
	if err != nil {
		opponents = nil
	}
	escaped := gamepack.EscapeSucceeds(len(opponents), state.speedOf(int(mover)),
		state.fastestOpponent(mover), a.rollDice)
	var result string
	if escaped {
		state.leaveBoard(mover, gamepack.FledState)
		// overlay-24 entry 11(記錄, 3, "Got Away") → entry 20(記錄, 字串, 0Ah, 1)。
		result = a.panelNotice(state, mover, state.say(msgFoeGotAway), noticeRowPanel, true)
	} else {
		// `0D0Dh`：entry 19，不帶名字。
		result = a.footerNotice(state, state.say(msgFoeEscapeBlocked))
	}
	state.Budgets[mover] = 0
	state.Activity.FoeSteps += run.steps
	state.FoeLog = state.say(msgFoeFled, a.combatantName(state, mover), run.steps) + " " + result
	a.tacticalStatus(state, state.FoeLog)
	state.endTurnAfterAction(a.rollDice)
}

// foeFleeEnd 是逃跑中交給 entry 6／entry 34 的收尾。
func (a *app) foeFleeEnd(state *tacticalState, mover uint8, run *foeFleeRun) {
	state.Activity.FoeSteps += run.steps
	state.FoeLog = state.say(msgFoeFled, a.combatantName(state, mover), run.steps)
	state.endTurn(a.rollDice, false)
}

// fledFoeRecords 是戰後不發經驗值的那幾隻：overlay-05 entry 2 `0079h` 跳過
// `+10Ch == 3` 的敵方（經驗值、錢與物品整段都跳過）。投降的（4）照發。
func (a *app) fledFoeRecords(state *tacticalState) []gamepack.MonsterRecord {
	if state == nil {
		return nil
	}
	friendly := state.Friendly
	var records []gamepack.MonsterRecord
	for index := 1; index < len(state.Roster) && index < len(friendly) && index < len(state.States); index++ {
		if friendly[index] || state.States[index] != gamepack.FledState ||
			state.Roster[index].FootprintClass != 0 {
			continue
		}
		if monster, ok := a.stagedMonsterFor(index, friendly); ok {
			records = append(records, monster.Record)
		}
	}
	return records
}
