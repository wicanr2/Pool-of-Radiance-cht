package main

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	"github.com/wicanr2/golden-box-remake-engine/geometry"
)

const (
	tacticalCellSize = 10
	tacticalLeft     = 70
	tacticalTop      = 76
)

// geoDetailForDirection 取出 GEO cell 在該方向的 detail 位元。
// engine 的 DetailDirections 依 0／2／4／6 的順序存放。
func geoDetailForDirection(cell geometry.Cell, direction uint8) uint8 {
	switch direction {
	case 0:
		return cell.DetailDirections[0]
	case 2:
		return cell.DetailDirections[1]
	case 4:
		return cell.DetailDirections[2]
	case 6:
		return cell.DetailDirections[3]
	}
	return 0
}

// geoWallProbe 重現 overlay-10 `0138h`：把 GEO 資料換成戰術層的 0／1／3。
// engine 的 Wall 用的就是 0／2／4／6 這組方向碼，與戰術層同一套，不必轉換。
//
// 原版不對座標取模：超出 16×16 一律當成牆，只有「與隊伍同一列、且方向是東或西」
// 例外，那一格回開放，好讓走廊可以延伸出盤面。界內則是「沒有牆 → 0；
// 有牆且該方向的 detail 位元為 0 → 1；有牆且 detail 非 0 → 3」。
func geoWallProbe(grid geometry.Grid, partyY int) combat.WallProbe {
	return func(direction uint8, x, y int) (uint8, error) {
		if x < 0 || x >= geometry.Width || y < 0 || y >= geometry.Height {
			if y == partyY && (direction == combat.WallDirectionEast || direction == combat.WallDirectionWest) {
				return combat.WallOpen, nil
			}
			return combat.WallBlocking, nil
		}
		wall, ok := grid.Wall(x, y, int(direction))
		if !ok || wall == 0 {
			return combat.WallOpen, nil
		}
		cell, _ := grid.Cell(x, y)
		if geoDetailForDirection(cell, direction) != 0 {
			return combat.WallAlternate, nil
		}
		return combat.WallBlocking, nil
	}
}

func fillTacticalCell(screen *ebiten.Image, column, row int, ink color.Color) {
	left := tacticalLeft + column*tacticalCellSize
	top := tacticalTop + row*tacticalCellSize
	for y := top; y < top+tacticalCellSize-1; y++ {
		for x := left; x < left+tacticalCellSize-1; x++ {
			screen.Set(x, y, ink)
		}
	}
}

// drawTactical 畫出由目前地城位置生成的戰術戰場。這一版只呈現地形，
// 還沒有 combatant、輸入或回合流程。
func drawTactical(screen *ebiten.Image, a *app, foreground, accent color.Color) {
	drawFrame(screen, foreground, accent)
	drawText(screen, "TACTICAL MAP PREVIEW", 232, 52, accent)
	if a.initialMap == nil {
		drawText(screen, "DUNGEON MAP IS NOT LOADED", 196, 190, foreground)
		return
	}

	if a.tactical == nil {
		drawText(screen, "TACTICAL STATE IS NOT BUILT", 190, 190, foreground)
		return
	}
	grid := a.tactical.Grid
	classes := a.tactical.Classes
	floor := color.RGBA{40, 72, 72, 255}
	wall := color.RGBA{170, 255, 255, 255}
	if a.modern {
		floor = color.RGBA{46, 54, 66, 255}
		wall = color.RGBA{238, 232, 207, 255}
	}

	painted := 0
	blocking := 0
	for row := 0; row < combat.TacticalMapHeight; row++ {
		for column := 0; column < combat.TacticalMapWidth; column++ {
			code := grid.Terrain[row*combat.TacticalRowStride+column]
			if code == combat.UnpaintedCellClass {
				continue
			}
			painted++
			ink := floor
			if int(code) < len(classes) && classes[code].EntryThreshold == 0xFF {
				ink = wall
				blocking++
			}
			fillTacticalCell(screen, column, row, ink)
		}
	}

	roster, friendly := a.tactical.Roster, a.tactical.Friendly
	occupancy, err := combat.RebuildOccupancy(roster)
	if err != nil {
		drawText(screen, "OCCUPANCY ERROR", 232, 190, accent)
		return
	}
	partyInk := color.RGBA{85, 255, 85, 255}
	foeInk := color.RGBA{255, 85, 85, 255}
	for row := 0; row < combat.TacticalMapHeight; row++ {
		for column := 0; column < combat.TacticalMapWidth; column++ {
			index := occupancy[row*combat.TacticalRowStride+column]
			if index == 0 {
				continue
			}
			ink := foeInk
			if int(index) < len(friendly) && friendly[index] {
				ink = partyInk
			}
			fillTacticalCell(screen, column, row, ink)
		}
	}

	party, foes := 0, 0
	for index := 1; index < len(roster); index++ {
		if friendly[index] {
			party++
		} else {
			foes++
		}
	}

	drawText(screen, fmt.Sprintf("DUNGEON %d,%d  CELLS %d  BLOCKING %d  PARTY %d  FOES %d",
		a.spawn.X, a.spawn.Y, painted, blocking, party, foes), 70, 330, foreground)
	drawText(screen, fmt.Sprintf("ROUND %d  MOVER %d  SCORE %d  BUDGET %d (%s)  %s",
		a.tactical.Round, a.tactical.Mover, a.tactical.Scores[a.tactical.Mover],
		a.tactical.Budget(), a.tactical.BudgetSource, a.tactical.Status), 70, 348, foreground)
	drawText(screen, "H I M Q P O K G: STEP   ENTER: END TURN   D: DELAY", 70, 366, accent)
	drawText(screen, "F5: BACK", 500, 330, foreground)
}

// 這一段的部署是暫定的。原版由 DS:43A2h 的陣型樣板決定誰站哪一格，而那張表
// 是執行期填的、不在檔案裡（spec 061），填寫者還沒解出來。在解出來之前，
// 這裡只從部署投影的格子裡挑得進去的來放，並在畫面上標明它是暫定版面。
const (
	provisionalPartyOffsetX = -1
	provisionalFoeOffsetX   = 2
)

// deploymentCandidates 依部署投影列出某個地城格偏移底下可以站人的格子，
// 順序與原版走樣板的順序相同：外層 row、內層 col。
func deploymentCandidates(grid combat.TacticalGrid, classes combat.CellClasses, offsetX int, taken map[[2]int]bool) [][2]int {
	candidates := make([][2]int, 0, combat.DeploymentTemplateSize)
	for row := 0; row < combat.DeploymentTemplateRows; row++ {
		for column := 0; column < combat.DeploymentTemplateCols; column++ {
			x, y := combat.DeploymentCell(offsetX, 0, row, column)
			if x < 0 || x > combat.TacticalMaxX || y < 0 || y > combat.TacticalMaxY {
				continue
			}
			if taken[[2]int{x, y}] {
				continue
			}
			code := grid.Terrain[y*combat.TacticalRowStride+x]
			if code == combat.UnpaintedCellClass {
				continue
			}
			decision, err := combat.TryPlaceCombatant(0xFF, 0, code, classes)
			if err != nil || decision != combat.PlacementAccepted {
				continue
			}
			candidates = append(candidates, [2]int{x, y})
		}
	}
	return candidates
}

// provisionalRoster 把隊伍與已 staged 的怪物擺上戰場，回傳 1-based 的位置表
// 與各筆屬於哪一方。放不下的就不放——不擠、不重疊、不自行擴大範圍。
func provisionalRoster(a *app, grid combat.TacticalGrid, classes combat.CellClasses) ([]combat.CombatantCell, []bool) {
	cells := []combat.CombatantCell{{}}
	friendly := []bool{false}
	taken := map[[2]int]bool{}

	assign := func(count, offsetX int, isParty bool) {
		for _, spot := range deploymentCandidates(grid, classes, offsetX, taken) {
			if count == 0 {
				return
			}
			taken[[2]int{spot[0], spot[1]}] = true
			cells = append(cells, combat.CombatantCell{
				X: uint8(spot[0]), Y: uint8(spot[1]), FootprintClass: 1,
			})
			friendly = append(friendly, isParty)
			count--
		}
	}

	assign(len(a.state.Party), provisionalPartyOffsetX, true)
	foes := 0
	for _, monster := range a.combatMonsters {
		foes += int(monster.Spawn.Count)
	}
	assign(foes, provisionalFoeOffsetX, false)
	return cells, friendly
}

// tacticalState 是戰術預覽跨影格保留的狀態。Scores 對應原版 runtime 的 `+3`
// 先攻排序分數，Budgets 對應 `+6` 的剩餘步數，兩者都是 1-based。
type tacticalState struct {
	Grid         combat.TacticalGrid
	Classes      combat.CellClasses
	Roster       []combat.CombatantCell
	Friendly     []bool
	Dexterity    []uint8
	Scores       []uint8
	Budgets      []uint8
	BaseMovement []uint8
	HitPoints    []int
	THAC0        []uint8
	ArmorClass   []int
	Damage       []combat.DamageDice
	StatsSource  string
	Round        int
	Mover        uint8
	Finished     bool
	Outcome      combat.CombatOutcome
	BudgetSource string
	Status       string
}

// Budget 回傳目前行動者的剩餘步數。
func (state *tacticalState) Budget() uint8 {
	if state.Mover == 0 || int(state.Mover) >= len(state.Budgets) {
		return 0
	}
	return state.Budgets[state.Mover]
}

// startRound 重現 spec 062 的回合開頭：逐一初始化移動預算，再擲先攻。
// 原版每回合都重擲，不是整場排一次。
func (state *tacticalState) startRound(roll func(count, sides int) int) {
	state.Round++
	for index := 1; index < len(state.Roster); index++ {
		state.Budgets[index] = combat.InitialMovementBudgetBeforeEffects(state.BaseMovement[index], false, 0)
		modifier := combat.DexterityInitiativeModifier(state.Dexterity[index])
		score, err := combat.ResolveInitiativeScore(modifier, uint8(roll(1, 6)), false)
		if err != nil {
			score = 0
		}
		state.Scores[index] = score
	}
	state.selectActor(roll)
}

// selectActor 重現 spec 062 的「每次行動後重選」：分數為 0 的不參與，
// 全部為 0 時代表這一回合沒有人可以行動。
func (state *tacticalState) selectActor(roll func(count, sides int) int) {
	scores := make([]uint8, 0, len(state.Roster))
	ties := make([]uint8, 0, len(state.Roster))
	for index := 1; index < len(state.Roster); index++ {
		scores = append(scores, state.Scores[index])
		ties = append(ties, uint8(roll(1, 100)))
	}
	selected, ok, err := combat.SelectInitiativeActor(scores, ties)
	if err != nil || !ok {
		state.Mover = 0
		return
	}
	state.Mover = uint8(selected + 1)
}

// endTurn 把目前行動者的分數歸零（Delay 時改成 1，與原版的 D 命令一致），
// 再重選；選不到人就跑回合收尾並開始下一回合。
func (state *tacticalState) endTurn(roll func(count, sides int) int, delay bool) {
	if state.Mover == 0 {
		return
	}
	if delay {
		state.Scores[state.Mover] = combat.DelayInitiative()
		state.Status = "DELAYED"
	} else {
		state.Scores[state.Mover] = 0
		state.Status = "TURN ENDED"
	}
	state.selectActor(roll)
	if state.Mover == 0 {
		state.startRound(roll)
		state.Status = fmt.Sprintf("ROUND %d", state.Round)
	}
}

// placeholderBaseMovement 是隊伍成員的暫定移動值。remake 的角色記錄目前沒有
// 這個欄位，原版是 285-byte record 的 +11Ch；在它接上來之前，這個數字只是
// 讓移動判定可以被實際走一次，畫面上會標明它的來源。
const placeholderBaseMovement = 12

// 隊伍成員的戰鬥數值暫定值。remake 的角色記錄目前只有 HP，沒有 AC、THAC0
// 與傷害骰；原版那三項在 285-byte record 的 +110h／+111h／+115h..+119h。
// THAC0 與 AC 這裡存的是原版的內部編碼（60 減去顯示值），與 ResolveHit 一致。
const (
	placeholderHitPoints          = 8
	placeholderInternalTHAC0      = 40
	placeholderInternalArmorClass = 50
)

// dexterityAbilityIndex 是能力值陣列裡的 DEX，順序為 STR／INT／WIS／DEX／CON／CHA。
// placeholderDexterity 給沒有能力值的一方用；怪物的 DEX 在 285-byte record 的
// `+13h`，還沒接上來。
const (
	dexterityAbilityIndex = 3
	placeholderDexterity  = 12
)

// enterTacticalPreview 生成戰場、擺人、決定移動預算。
func (a *app) enterTacticalPreview() error {
	if a.initialMap == nil {
		return fmt.Errorf("Pool dungeon map is not loaded")
	}
	grid, err := combat.GenerateIndoorTacticalGrid(int(a.spawn.X), int(a.spawn.Y),
		geoWallProbe(a.initialMap.Grid, int(a.spawn.Y)))
	if err != nil {
		return err
	}
	classes := gamepack.OriginalCombatCellClassTable()
	roster, friendly := provisionalRoster(a, grid, classes)

	base, source := uint8(placeholderBaseMovement), "PLACEHOLDER"
	if len(a.combatMonsters) > 0 {
		base, source = a.combatMonsters[0].Record.Movement(), "STAGED MONSTER"
	}
	size := len(roster)
	state := &tacticalState{
		Grid:         grid,
		Classes:      classes,
		Roster:       roster,
		Friendly:     friendly,
		Dexterity:    make([]uint8, size),
		Scores:       make([]uint8, size),
		Budgets:      make([]uint8, size),
		BaseMovement: make([]uint8, size),
		BudgetSource: source,
	}
	state.HitPoints = make([]int, size)
	state.THAC0 = make([]uint8, size)
	state.ArmorClass = make([]int, size)
	state.Damage = make([]combat.DamageDice, size)
	state.StatsSource = "PARTY PLACEHOLDER"
	if len(a.combatMonsters) > 0 {
		state.StatsSource = "MONSTER RECORD / PARTY PLACEHOLDER"
	}
	party := 0
	for index := 1; index < size; index++ {
		state.BaseMovement[index] = base
		state.Dexterity[index] = placeholderDexterity
		state.HitPoints[index] = placeholderHitPoints
		state.THAC0[index] = placeholderInternalTHAC0
		state.ArmorClass[index] = placeholderInternalArmorClass
		state.Damage[index] = combat.DamageDice{Count: 1, Sides: 8}
		if friendly[index] && party < len(a.state.Party) {
			member := a.state.Party[party]
			state.Dexterity[index] = uint8(member.Abilities[dexterityAbilityIndex])
			if member.CurrentHP > 0 {
				state.HitPoints[index] = member.CurrentHP
			}
			party++
			continue
		}
		if record, ok := a.stagedRecordFor(index, friendly); ok {
			state.BaseMovement[index] = record.Movement()
			state.HitPoints[index] = int(record.CurrentHitPoints())
			state.THAC0[index] = uint8(60 - record.THAC0())
			state.ArmorClass[index] = 60 - record.ArmorClass()
			state.Damage[index] = combat.DamageDice{
				Count: record.DamageDiceCount(),
				Sides: record.DamageDieSides(),
				Bonus: record.DamageBonus(),
			}
		}
	}
	state.startRound(a.rollDice)
	state.Status = fmt.Sprintf("ROUND %d", state.Round)
	a.tactical = state
	return nil
}

// rollDice 把 app 的骰子接成回合流程要的形狀。
func (a *app) rollDice(count, sides int) int { return a.roller.Roll(count, sides) }

// tacticalStepKeys 是原版 Move 命令的八個方向鍵，依 spec 053 對到 direction 0..7。
var tacticalStepKeys = [8]ebiten.Key{
	ebiten.KeyH, ebiten.KeyI, ebiten.KeyM, ebiten.KeyQ,
	ebiten.KeyP, ebiten.KeyO, ebiten.KeyK, ebiten.KeyG,
}

// tacticalInput 讓那八個鍵驅動 ResolveDestination，並在允許進入時提交這一步。
func (a *app) tacticalInput() error {
	state := a.tactical
	if state == nil {
		return nil
	}
	if a.justPressed(ebiten.KeyEnter) {
		state.endTurn(a.rollDice, false)
		return nil
	}
	if a.justPressed(ebiten.KeyD) {
		state.endTurn(a.rollDice, true)
		return nil
	}
	if state.Mover == 0 {
		return nil
	}
	for direction, key := range tacticalStepKeys {
		if !a.justPressed(key) {
			continue
		}
		occupancy, err := combat.RebuildOccupancy(state.Roster)
		if err != nil {
			return err
		}
		tactical := combat.TacticalState{
			Map:       state.Grid,
			Occupancy: occupancy,
			Cells:     state.Roster,
			Classes:   state.Classes,
		}
		outcome, err := combat.ResolveDestination(tactical, state.Mover, uint8(direction), state.Budget())
		if err != nil {
			return err
		}
		switch {
		case outcome.Leaving:
			state.Status = "OFF BOARD: LEAVE COMBAT PROMPT"
		case outcome.Action == combat.MovementAttack:
			if err := a.resolveTacticalAttack(state, outcome.Target); err != nil {
				return err
			}
			if state.Finished {
				return a.finishCombat(state.Outcome)
			}
		case outcome.Action == combat.MovementBlocked:
			state.Status = "BLOCKED"
		default:
			mover := state.Roster[state.Mover]
			x, y, err := combat.AdvanceTacticalCoordinate(mover.X, mover.Y, uint8(direction))
			if err != nil {
				return err
			}
			budget, err := combat.SpendMovementStep(state.Budget(), uint8(direction))
			if err != nil {
				return err
			}
			state.Roster[state.Mover].X, state.Roster[state.Mover].Y = x, y
			state.Budgets[state.Mover] = budget
			state.Status = fmt.Sprintf("MOVED %d", direction)
		}
		return nil
	}
	return nil
}

// stagedRecordFor 找出敵方第 index 筆對應的原版怪物記錄。staged 的每一筆帶著
// 數量，所以要依序展開才對得回去。
func (a *app) stagedRecordFor(index int, friendly []bool) (gamepack.MonsterRecord, bool) {
	position := 0
	for slot := 1; slot < index; slot++ {
		if !friendly[slot] {
			position++
		}
	}
	for _, monster := range a.combatMonsters {
		if position < int(monster.Spawn.Count) {
			return monster.Record, true
		}
		position -= int(monster.Spawn.Count)
	}
	return gamepack.MonsterRecord{}, false
}

// resolveTacticalAttack 以既有的命中與傷害規則（spec 050／051）解一次攻擊。
// 目標歸零時把它的體型類別寫 0，與原版一樣讓它不再佔格、也不再參與。
func (a *app) resolveTacticalAttack(state *tacticalState, target uint8) error {
	if int(target) >= len(state.HitPoints) {
		return fmt.Errorf("Pool attack target %d is outside the roster", target)
	}
	roll := uint8(a.rollDice(1, 20))
	hit, err := combat.ResolveHit(roll, state.THAC0[state.Mover], state.ArmorClass[target], 0)
	if err != nil {
		return err
	}
	if !hit {
		state.Status = fmt.Sprintf("ATTACK %d MISSED (D20 %d)", target, roll)
		return nil
	}
	dice := state.Damage[state.Mover]
	rolls := make([]uint8, dice.Count)
	for index := range rolls {
		rolls[index] = uint8(a.rollDice(1, int(dice.Sides)))
	}
	damage, err := combat.ResolveDamage(dice, rolls, 1)
	if err != nil {
		return err
	}
	state.HitPoints[target] -= damage
	if state.HitPoints[target] > 0 {
		state.Status = fmt.Sprintf("HIT %d FOR %d (HP %d)", target, damage, state.HitPoints[target])
		return nil
	}
	state.HitPoints[target] = 0
	state.Roster[target].FootprintClass = 0
	state.Scores[target] = 0
	state.Status = fmt.Sprintf("%d IS DOWN", target)
	if over, outcome := state.combatOutcome(); over {
		state.Finished, state.Outcome = true, outcome
		switch outcome {
		case combat.CombatVictory:
			state.Status = "VICTORY"
		case combat.CombatDefeat:
			state.Status = "DEFEAT"
		}
	}
	return nil
}

// finishCombat 依 spec 046 契約 5 處理戰後：只有勝利才從 COMBAT 邊界停下的 PC
// 續跑戰後腳本；戰敗不得續跑，也不得用自動勝利代替戰鬥結果。
func (a *app) finishCombat(outcome combat.CombatOutcome) error {
	a.tacticalPreview, a.tactical = false, nil
	if outcome != combat.CombatVictory {
		a.statusLine = "Party defeated; the post-combat script does not run."
		return nil
	}
	a.combatActive, a.combatMonsters = false, nil
	a.cellEventPending, a.cellWaitingMenu = false, false
	a.eventText, a.eventLabel = "", ""
	result, err := a.eventSession.RunUntilEvent(4096, nil, true)
	if err != nil {
		return fmt.Errorf("continue after Pool combat: %w", err)
	}
	a.applyCellECLResult(result)
	return nil
}

// combatOutcome 依兩邊還站著的人數判定戰鬥是否結束。清光敵人之後原版還會問
// 一次要不要繼續，這裡先不問，直接當成結束。
func (state *tacticalState) combatOutcome() (bool, combat.CombatOutcome) {
	counts := combat.SideCounts{}
	for index := 1; index < len(state.Roster); index++ {
		if state.Roster[index].FootprintClass == 0 {
			continue
		}
		if state.Friendly[index] {
			counts.Party++
		} else {
			counts.Foes++
		}
	}
	if !combat.RoundEndsCombat(counts, 0, false) {
		return false, combat.CombatOngoing
	}
	return true, combat.ResolveCombatOutcome(counts)
}
