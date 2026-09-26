package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// tacticalPilot 是探索與主線探針用的「自己人怎麼打」。**這不是原版的演算法**，
// 是測試治具裡的玩家策略層（issue #22）：它只按玩家按得到的鍵。
//
// 每一回合的優先序：
//
//  1. 有隊友倒地就 B）ANDAGE（spec 138）；旁邊有敵人時撐到計時快到才包。
//  2. 記著催眠術而場上還有三個以上醒著的敵人就 C）AST 催眠（說明書 p.44）。
//  3. 旁邊有敵人就 A）IM，挑生命力最少的那一隻集火（N 鍵換目標）。
//  3. 否則往「站得上去、又貼著敵人」的最近一格走（先按 M 進移動）：步數表把
//     別人站的格子當牆，不再撞進同伴背後（以前四成的行動都是 BLOCKED）。
//  4. 走不動、走夠了或沒有路就結束回合。
type tacticalPilot struct {
	mover uint8
	round int
	tried bool
	moves int
	// aim 是這一回合按 A 之後要挑的目標；aimCycles 是換目標的 guard。
	aim       uint8
	aimCycles int
	// castID 是這一回合按 C 之後要挑的法術；castCycles 是移游標的 guard。
	castID     uint8
	castCycles int
	// spellAimPresses 是施法瞄準那一步按了幾下（#73）。超出射程、挑到重複的那一格
	// 按 ENTER 不會離開瞄準，所以交替按 N 換人，按夠了就 ESC（再答 Y 放棄）。
	spellAimPresses int
	// spellCentre 是催眠術要瞄的那一隻：以牠為中心的範圍裡沒有自己人（#73：範圍法術
	// 不分敵我，原版 AI 的 `0255h` 也是這樣挑）。
	spellCentre uint8
	// distance 是這一個回合的步數表（到任一「貼著敵人的空格」幾步），
	// 每一 tick 重算會讓探索的時間全花在廣度優先上。
	distance map[int]int
	// lastX／lastY 是上一次送出方向鍵之前站的格子；沒動就是被擋住了。
	lastX, lastY uint8
	stepped      bool
}

// exploreMaxCombatSteps 是一個角色一回合最多走幾步。原版有移動額度擋著，
// 這裡另外加一個上限，免得額度算法出錯時無限走下去。
const exploreMaxCombatSteps = 12

// bandageUrgency 是倒地計時到幾就算緊急：旁邊有醒著的敵人也放下武器先包紮。
// 計時在回合結束時加一、超過 9 才死（`combat.DyingRoundLimit`），所以看到 8 的
// 那一回合包下去還有一回合餘裕。以前是 6：量過七場（playtest 補七），輸的那幾場
// 每一個行動都要，早包等於少打一下。
const bandageUrgency = 8

func (pilot *tacticalPilot) key(app *app) ebiten.Key {
	// 停拍中 tacticalInput 不讀鍵（combat_notice.go）：這一影格送一個沒有人接的鍵，
	// 駕駛自己的計數也不動——不然「按了沒反應」會被當成被擋住或瞄不到。
	if combatNoticeHolding(app) {
		return combatNoticeIdleKey
	}
	state := app.tactical
	if state.Prompt {
		// 敵方清光之後那一次問的是「還要不要繼續打」（spec 062）。答 Y
		// 會再開一輪，於是那一場永遠結束不了——開打與收工共用同一個旗標。
		if state.sideCounts().Foes == 0 {
			return ebiten.KeyN
		}
		return ebiten.KeyY
	}
	if app.castOpen {
		// 法術清單：↓ 移到要施的那一條再 ENTER；找不到就 ESC 回去照常打。
		if pilot.castID != 0 && pilot.castCycles < len(app.castOptions) &&
			app.castCursor < len(app.castOptions) && app.castOptions[app.castCursor].ID != pilot.castID {
			pilot.castCycles++
			return ebiten.KeyArrowDown
		}
		if pilot.castID == 0 || app.castCursor >= len(app.castOptions) ||
			app.castOptions[app.castCursor].ID != pilot.castID {
			pilot.castID = 0
			return ebiten.KeyEscape
		}
		return ebiten.KeyEnter
	}
	if app.castTargeting {
		if app.castAim != nil && !app.castTargetingAttack {
			// 施法的瞄準：第一下 ENTER 打預設那一個（最近的敵人）；停在原地就是
			// 超出射程或挑過了，換下一個再試。guard 給到每個候選都試過兩輪。
			pilot.spellAimPresses++
			if pilot.spellAimPresses > 4*len(app.castTargets)+4 {
				return ebiten.KeyEscape
			}
			if pilot.spellCentre != 0 && pilot.spellAimPresses <= 2*len(app.castTargets) {
				if app.castTargets[app.castTargetCursor] != pilot.spellCentre {
					return ebiten.KeyN
				}
				return ebiten.KeyEnter
			}
			if pilot.spellAimPresses%2 == 0 {
				return ebiten.KeyN
			}
			return ebiten.KeyEnter
		}
		// 集火：游標預設停在最近的敵人，N 往下一個，轉到想打的那一隻再 ENTER。
		if pilot.aim != 0 && len(app.castTargets) > 0 && pilot.aimCycles < len(app.castTargets) &&
			app.castTargets[app.castTargetCursor] != pilot.aim {
			pilot.aimCycles++
			return ebiten.KeyN
		}
		return ebiten.KeyEnter
	}
	pilot.spellAimPresses = 0
	if app.castAborting() {
		// 瞄不到任何人：放棄這個法術（原版 "Abort Spell?" 答 Y）。
		return ebiten.KeyY
	}
	if state.Mover == 0 || int(state.Mover) >= len(state.Friendly) ||
		!state.Friendly[state.Mover] {
		return ebiten.KeyEnter
	}
	if pilot.mover != state.Mover || pilot.round != state.Round {
		pilot.mover, pilot.round = state.Mover, state.Round
		pilot.tried, pilot.moves = false, 0
		pilot.aim, pilot.aimCycles = 0, 0
		pilot.castID, pilot.castCycles = 0, 0
		pilot.spellCentre = 0
		pilot.distance, pilot.stepped = nil, false
	}
	here := state.Roster[state.Mover]
	if pilot.stepped && here.X == pilot.lastX && here.Y == pilot.lastY {
		// 方向鍵送出去人沒動：被擋住或額度用完，這一回合到此為止。
		return ebiten.KeyEnter
	}
	pilot.stepped = false
	if !pilot.tried {
		pilot.tried = true
		// 玩家策略（issue #22／#25）：有隊友倒地就先包紮。原版的 B）ANDAGE
		// 不看距離（spec 138），代價是這一個行動。旁邊有敵人時多撐幾回合
		// 再包——倒地計時到 9 才轉死亡，`bandageUrgency` 之前先把身邊的打掉。
		if target, ok := state.bandageTarget(); ok &&
			(!pilot.adjacentToAwakeFoe(state) || state.DyingCounters[target] >= bandageUrgency) {
			return ebiten.KeyB
		}
		if id, ok := pilot.spellToCast(app); ok {
			pilot.castID = id
			return ebiten.KeyC
		}
		if target, ok := pilot.focusTarget(state); ok {
			pilot.aim = target
			return ebiten.KeyA
		}
	}
	if pilot.moves >= exploreMaxCombatSteps {
		return ebiten.KeyEnter
	}
	if pilot.distance == nil {
		pilot.distance = pilot.approachDistances(state)
	}
	current, ok := pilot.distance[tacticalCellKey(here.X, here.Y)]
	if !ok || current == 0 {
		return ebiten.KeyEnter
	}
	best := -1
	for direction := uint8(0); direction < combat.DirectionCount; direction++ {
		x, y, err := combat.AdvanceTacticalCoordinate(here.X, here.Y, direction)
		if err != nil {
			continue
		}
		if next, ok := pilot.distance[tacticalCellKey(x, y)]; ok && next == current-1 {
			best = int(direction)
			break
		}
	}
	if best < 0 {
		return ebiten.KeyEnter
	}
	// 原版的方向鍵要先按 M）OVE 才算方向（頂層的 Q 是 Q）UICK）。
	if !state.Moving {
		return ebiten.KeyM
	}
	pilot.moves++
	pilot.lastX, pilot.lastY, pilot.stepped = here.X, here.Y, true
	// 走到敵人旁邊才值得再試一次瞄準。每走一步都按一次 A 會讓一場架的
	// tick 數翻倍，探索的預算就全花在打不到的瞄準上。
	pilot.tried = !pilot.adjacentToFoe(state)
	return tacticalStepKeys[best]
}

// spellToCast 說這一個行動者現在該不該施法、施哪一條：記著催眠術、而且場上
// 醒著的敵人還有三個以上（催眠一次放倒 2d4 個生命骰，說明書 p.44）。
func (pilot *tacticalPilot) spellToCast(app *app) (uint8, bool) {
	state := app.tactical
	index, ok := app.moverPartyIndex(state.Mover)
	if !ok || index >= len(app.state.Party) {
		return 0, false
	}
	// 這一回合挨過打的指令列上沒有 Cast（overlay-08 `072Fh`），玩家不會去按它。
	if state.castingDisrupted(int(state.Mover)) {
		return 0, false
	}
	hasSleep := false
	for _, option := range app.spellOptionsFor(app.state.Party[index]) {
		if option.ID == gamepack.SpellIDSleep {
			hasSleep = true
		}
	}
	if !hasSleep {
		return 0, false
	}
	awake := 0
	for foe := 1; foe < len(state.Roster); foe++ {
		if !standing(state, foe) || foe >= len(state.Friendly) ||
			state.Friendly[foe] == state.Friendly[state.Mover] {
			continue
		}
		if !state.hasEffect(foe, gamepack.SleepEffectCode) {
			awake++
		}
	}
	if awake < 3 {
		return 0, false
	}
	// 範圍不分敵我：找一隻醒著、周圍（預算 +6 & 7）沒有自己人的當中心，找不到就不放。
	if int(gamepack.SpellIDSleep) >= len(app.spellParameters) {
		return 0, false
	}
	budget := app.spellParameters[gamepack.SpellIDSleep].TargetPlan().AreaBudget
	for foe := 1; foe < len(state.Roster); foe++ {
		if !standing(state, foe) || state.Friendly[foe] == state.Friendly[state.Mover] ||
			state.hasEffect(foe, gamepack.SleepEffectCode) {
			continue
		}
		members, err := state.spellAreaMembers(int(state.Roster[foe].X), int(state.Roster[foe].Y), budget)
		if err != nil {
			continue
		}
		crowded := false
		for _, member := range members {
			crowded = crowded || state.Friendly[member] == state.Friendly[state.Mover]
		}
		if !crowded {
			pilot.spellCentre = uint8(foe)
			return gamepack.SpellIDSleep, true
		}
	}
	return 0, false
}

// standing 說第 index 格還在場上（體型類別非 0）。
func standing(state *tacticalState, index int) bool {
	return index > 0 && index < len(state.Roster) && state.Roster[index].FootprintClass != 0
}

// foeCells 列出所有還站著的敵方 combatant 佔的格子。
func (pilot *tacticalPilot) foeCells(state *tacticalState) []combat.FootprintCell {
	cells := []combat.FootprintCell{}
	for index := 1; index < len(state.Roster); index++ {
		if !standing(state, index) || index >= len(state.Friendly) ||
			state.Friendly[index] == state.Friendly[state.Mover] {
			continue
		}
		for _, cell := range combat.FootprintCells(state.Roster[index].FootprintClass,
			state.Roster[index].X, state.Roster[index].Y) {
			if cell.Valid() {
				cells = append(cells, cell)
			}
		}
	}
	return cells
}

// adjacentToFoe 說目前這一格旁邊有沒有站著還在場的敵人。
func (pilot *tacticalPilot) adjacentToFoe(state *tacticalState) bool {
	_, ok := pilot.focusTarget(state)
	return ok
}

// adjacentToAwakeFoe 只算醒著的：睡著的敵人貼在旁邊不是威脅，不必為它延後包紮。
func (pilot *tacticalPilot) adjacentToAwakeFoe(state *tacticalState) bool {
	here := state.Roster[state.Mover]
	for index := 1; index < len(state.Roster); index++ {
		if !standing(state, index) || index == int(state.Mover) ||
			index >= len(state.Friendly) || state.Friendly[index] == state.Friendly[state.Mover] ||
			state.hasEffect(index, gamepack.SleepEffectCode) {
			continue
		}
		for _, cell := range combat.FootprintCells(state.Roster[index].FootprintClass,
			state.Roster[index].X, state.Roster[index].Y) {
			if cell.Valid() && chebyshev(here.X, here.Y, cell.X, cell.Y) <= 1 {
				return true
			}
		}
	}
	return false
}

// focusTarget 挑旁邊生命力最少的敵人（同分取編號小的）：集火先打倒一隻，
// 比六個人各打一隻讓六隻都活著划算——倒下的不再出手。
func (pilot *tacticalPilot) focusTarget(state *tacticalState) (uint8, bool) {
	here := state.Roster[state.Mover]
	best, bestHP := 0, 0
	for index := 1; index < len(state.Roster); index++ {
		if !standing(state, index) || index == int(state.Mover) ||
			index >= len(state.Friendly) || state.Friendly[index] == state.Friendly[state.Mover] {
			continue
		}
		adjacent := false
		for _, cell := range combat.FootprintCells(state.Roster[index].FootprintClass,
			state.Roster[index].X, state.Roster[index].Y) {
			if cell.Valid() && chebyshev(here.X, here.Y, cell.X, cell.Y) <= 1 {
				adjacent = true
				break
			}
		}
		if !adjacent {
			continue
		}
		hp := 0
		if index < len(state.HitPoints) {
			hp = state.HitPoints[index]
		}
		if best == 0 || hp < bestHP {
			best, bestHP = index, hp
		}
	}
	return uint8(best), best != 0
}

// approachDistances 是到任一「空著、可走、貼著敵人」的格子要幾步。多源廣度
// 優先，別人（含同伴）站的格子當牆，只有自己現在站的那一格例外。
func (pilot *tacticalPilot) approachDistances(state *tacticalState) map[int]int {
	occupied := map[int]bool{}
	for index := 1; index < len(state.Roster); index++ {
		if !standing(state, index) || index == int(state.Mover) {
			continue
		}
		for _, cell := range combat.FootprintCells(state.Roster[index].FootprintClass,
			state.Roster[index].X, state.Roster[index].Y) {
			if cell.Valid() {
				occupied[tacticalCellKey(cell.X, cell.Y)] = true
			}
		}
	}
	passable := func(x, y uint8) bool {
		if occupied[tacticalCellKey(x, y)] {
			return false
		}
		terrain, err := state.Grid.TerrainAt(int(x), int(y))
		if err != nil {
			return false
		}
		record, err := combat.CellClassAt(state.Classes, terrain)
		return err == nil && record.EntryThreshold < 0xFF
	}
	distance := map[int]int{}
	queue := [][2]uint8{}
	for _, foe := range pilot.foeCells(state) {
		for direction := uint8(0); direction < combat.DirectionCount; direction++ {
			x, y, err := combat.AdvanceTacticalCoordinate(foe.X, foe.Y, direction)
			if err != nil {
				continue
			}
			key := tacticalCellKey(x, y)
			if _, seen := distance[key]; seen || !passable(x, y) {
				continue
			}
			distance[key] = 0
			queue = append(queue, [2]uint8{x, y})
		}
	}
	for len(queue) != 0 {
		cell := queue[0]
		queue = queue[1:]
		step := distance[tacticalCellKey(cell[0], cell[1])] + 1
		for direction := uint8(0); direction < combat.DirectionCount; direction++ {
			x, y, err := combat.AdvanceTacticalCoordinate(cell[0], cell[1], direction)
			if err != nil {
				continue
			}
			key := tacticalCellKey(x, y)
			if _, seen := distance[key]; seen || !passable(x, y) {
				continue
			}
			distance[key] = step
			queue = append(queue, [2]uint8{x, y})
		}
	}
	return distance
}

func TestTacticalPilotEndsTheTurnWhenNoFoeRemains(t *testing.T) {
	state := &tacticalState{
		Roster:   make([]combat.CombatantCell, 9),
		Friendly: make([]bool, 9),
		Budgets:  make([]uint8, 9),
		Round:    3,
		Mover:    1,
	}
	state.Roster[1].FootprintClass = 1
	state.Friendly[1] = true
	state.Budgets[1] = 6
	pilot := &tacticalPilot{mover: 1, round: 3, tried: true, distance: map[int]int{0: 1}}
	if got := pilot.key(&app{tactical: state}); got != ebiten.KeyEnter {
		t.Fatalf("no remaining foe returned %v, want Enter", got)
	}
}

// 集火：旁邊兩隻敵人，挑生命力少的那一隻；換目標用 N，到了才 ENTER。
func TestTacticalPilotFocusesTheWeakestAdjacentFoe(t *testing.T) {
	state := newFoeTurnState(0, 0, 0, 0, 6)
	state.Roster = append(state.Roster, combat.CombatantCell{X: 6, Y: 5, FootprintClass: 1})
	state.Friendly = append(state.Friendly, false)
	state.HitPoints = []int{0, 10, 8, 3}
	state.Roster[1] = combat.CombatantCell{X: 5, Y: 5, FootprintClass: 1}
	state.Roster[2] = combat.CombatantCell{X: 4, Y: 5, FootprintClass: 1}
	state.Mover = 1
	pilot := &tacticalPilot{}
	application := &app{tactical: state}
	if got := pilot.key(application); got != ebiten.KeyA {
		t.Fatalf("adjacent foes returned %v, want A", got)
	}
	if pilot.aim != 3 {
		t.Fatalf("aimed at %d, want the 3 HP foe (3)", pilot.aim)
	}
	application.castTargeting = true
	application.castTargets = []uint8{2, 3}
	application.castTargetCursor = 0
	if got := pilot.key(application); got != ebiten.KeyN {
		t.Fatalf("cursor on the wrong foe returned %v, want N", got)
	}
	application.castTargetCursor = 1
	if got := pilot.key(application); got != ebiten.KeyEnter {
		t.Fatalf("cursor on the weakest foe returned %v, want Enter", got)
	}
}

// 走位：同伴站在直線上就繞過去，不再往同伴背後撞。
func TestTacticalPilotWalksAroundTeammates(t *testing.T) {
	state := newFoeTurnState(0, 0, 0, 0, 6)
	state.Roster = append(state.Roster, combat.CombatantCell{X: 6, Y: 5, FootprintClass: 1})
	state.Friendly = append(state.Friendly, true)
	state.HitPoints = []int{0, 10, 8, 10}
	state.Roster[1] = combat.CombatantCell{X: 4, Y: 5, FootprintClass: 1} // 走的人
	state.Roster[3] = combat.CombatantCell{X: 5, Y: 5, FootprintClass: 1} // 同伴擋在中間
	state.Roster[2] = combat.CombatantCell{X: 6, Y: 5, FootprintClass: 1} // 敵人
	state.Mover = 1
	pilot := &tacticalPilot{}
	if got := pilot.key(&app{tactical: state}); got != ebiten.KeyM {
		t.Fatalf("the pilot should press M before stepping, got %v", got)
	}
	state.Moving = true
	got := pilot.key(&app{tactical: state})
	if got == ebiten.KeyEnter || got == ebiten.KeyA {
		t.Fatalf("blocked by a teammate the pilot returned %v, want a step", got)
	}
	if got == tacticalStepKeys[2] {
		t.Fatalf("the pilot stepped straight into the teammate")
	}
}

// 包紮的時機（playtest 補七之後改的）：旁邊只有睡著的敵人就先包；醒著的貼著時
// 計時未到 8 先打，到 8 才包。
func TestTacticalPilotBandagesUnlessAnAwakeFoeIsAdjacent(t *testing.T) {
	state := newFoeTurnState(6, 5, 5, 5, 6)
	state.Roster = append(state.Roster, combat.CombatantCell{X: 3, Y: 5, FootprintClass: 1})
	state.Friendly = append(state.Friendly, true)
	state.HitPoints = []int{0, 10, 8, 0}
	state.States = make([]uint8, 4)
	state.DyingCounters = make([]uint8, 4)
	state.Effects = make([]gamepack.EffectList, 4)
	state.States[3] = combat.DyingState
	state.DyingCounters[3] = 2
	state.Mover = 1
	// 旁邊的敵人 2 睡著：先包。
	state.addEffect(2, gamepack.SleepEffectCode, 0, 1)
	pilot := &tacticalPilot{}
	if got := pilot.key(&app{tactical: state}); got != ebiten.KeyB {
		t.Fatalf("with only a sleeping foe adjacent the pilot returned %v, want B", got)
	}
	// 敵人 2 醒著、計時 2：先打。
	state.removeEffect(2, gamepack.SleepEffectCode)
	pilot = &tacticalPilot{}
	if got := pilot.key(&app{tactical: state}); got != ebiten.KeyA {
		t.Fatalf("with an awake foe adjacent at counter 2 the pilot returned %v, want A", got)
	}
	// 計時 8：包。
	state.DyingCounters[3] = bandageUrgency
	pilot = &tacticalPilot{}
	if got := pilot.key(&app{tactical: state}); got != ebiten.KeyB {
		t.Fatalf("at counter %d the pilot returned %v, want B", bandageUrgency, got)
	}
}
