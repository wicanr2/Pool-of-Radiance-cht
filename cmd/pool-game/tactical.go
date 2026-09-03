package main

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/creation"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
	"github.com/wicanr2/golden-box-remake-engine/geometry"
)

const (
	tacticalCellSize = 10
	tacticalLeft     = 70
	// 盤面往上挪，讓底下擠得下四行資訊加功能鍵列。25 列 × 10 像素從 58 畫到 307，
	// 四行基線 322／338／354／370，功能鍵列 386；漢字字型的 ascent 是 14，
	// 16 像素行距剛好不相疊，也不會壓到下框。
	tacticalTop = 58
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
	drawText(screen, a.text(msgTacticalTitle), 232, 44, accent)
	if a.initialMap == nil {
		drawText(screen, a.text(msgTacticalNoMap), 196, 190, foreground)
		return
	}

	if a.tactical == nil {
		drawText(screen, a.text(msgTacticalNoState), 190, 190, foreground)
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

	drawText(screen, fmt.Sprintf(a.text(msgTacticalBoard),
		a.spawn.X, a.spawn.Y, painted, blocking, party, foes), 70, 322, foreground)
	drawText(screen, fmt.Sprintf(a.text(msgTacticalRound),
		a.tactical.Round, a.tactical.Mover, a.tactical.Scores[a.tactical.Mover],
		a.tactical.Budget(), a.tactical.BudgetSource, a.tactical.Status), 70, 338, foreground)
	drawText(screen, fmt.Sprintf("%s   %s", a.text(msgTacticalProvisional), a.tactical.FoeLog),
		70, 354, foreground)
	hint := a.text(msgTacticalKeys) + "  " + a.text(msgCastHint)
	if a.tactical.Prompt {
		hint = a.text(msgTacticalPrompt)
	}
	drawText(screen, hint, 70, 370, accent)
	drawText(screen, a.text(msgTacticalBack), 500, 322, foreground)
	drawCastMenu(screen, a, foreground, accent)
	drawCastTargeting(screen, a, accent)
}

// drawCastTargeting 標出選目標那一步停在誰身上。原版的選單列是
// `Next Prev Manual`，格子游標（Manual）那一半還沒接。
func drawCastTargeting(screen *ebiten.Image, a *app, accent color.Color) {
	if !a.castTargeting || len(a.castTargets) == 0 {
		return
	}
	target := a.castTargets[a.castTargetCursor]
	drawText(screen, fmt.Sprintf(a.text(msgCastAiming), a.castPending.Label, target),
		70, 306, accent)
}

// drawCastMenu 把施法清單畫在盤面右邊。只列得出已經讀過處理常式的法術，
// 所以看得到的就是做得到的。
func drawCastMenu(screen *ebiten.Image, a *app, foreground, accent color.Color) {
	if !a.castOpen || len(a.castOptions) == 0 {
		return
	}
	for index, option := range a.castOptions {
		cursor, ink := " ", foreground
		if index == a.castCursor {
			cursor, ink = ">", accent
		}
		drawText(screen, fmt.Sprintf("%s%s", cursor, option.Label),
			420, 100+index*16, ink)
	}
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

// provisionalRoster 把隊伍與已 staged 的怪物擺上戰場，回傳 1-based 的位置表、
// 各筆屬於哪一方，以及每一格對回哪一個隊伍成員（不是隊伍成員的是 −1）。
// 放不下的就不放——不擠、不重疊、不自行擴大範圍。
//
// **陣營是逐人看的**：`36h ADD NPC` 加進來的 NPC 記錄 `+10Eh` 非零時站在
// 對面（spec 091），所以不能整批當成我方；隊伍索引也因此要另外記，
// 不能靠「友方槽依序對應隊伍」那個假設。
func provisionalRoster(a *app, grid combat.TacticalGrid, classes combat.CellClasses) ([]combat.CombatantCell, []bool, []int) {
	cells := []combat.CombatantCell{{}}
	friendly := []bool{false}
	partySlot := []int{-1}
	taken := map[[2]int]bool{}

	assign := func(members []int, offsetX int, isParty bool) {
		next := 0
		for _, spot := range deploymentCandidates(grid, classes, offsetX, taken) {
			if next >= len(members) {
				return
			}
			taken[[2]int{spot[0], spot[1]}] = true
			cells = append(cells, combat.CombatantCell{
				X: uint8(spot[0]), Y: uint8(spot[1]), FootprintClass: 1,
			})
			friendly = append(friendly, isParty)
			partySlot = append(partySlot, members[next])
			next++
		}
	}

	allies, traitors := make([]int, 0, len(a.state.Party)), make([]int, 0, 1)
	for index, member := range a.state.Party {
		if member.Side != 0 {
			traitors = append(traitors, index)
			continue
		}
		allies = append(allies, index)
	}
	assign(allies, provisionalPartyOffsetX, true)
	foes := 0
	for _, monster := range a.combatMonsters {
		foes += int(monster.Spawn.Count)
	}
	opposing := append([]int(nil), traitors...)
	for index := 0; index < foes; index++ {
		opposing = append(opposing, -1)
	}
	assign(opposing, provisionalFoeOffsetX, false)
	return cells, friendly, partySlot
}

// tacticalState 是戰術預覽跨影格保留的狀態。Scores 對應原版 runtime 的 `+3`
// 先攻排序分數，Budgets 對應 `+6` 的剩餘步數，兩者都是 1-based。
type tacticalState struct {
	Grid          combat.TacticalGrid
	Classes       combat.CellClasses
	Roster        []combat.CombatantCell
	Friendly      []bool
	// PartySlot 把戰場上的位置換回隊伍索引，−1 代表那一格不是隊員。
	// 施法要用它才找得到「這個位置是誰」的記憶陣列。
	PartySlot []int
	// HitDice 是每一格的 `+73h`（最高職業等級，怪物就是生命骰）。
	// 催眠術用它算要花多少額度（spec 098）。
	HitDice []uint8
	// SleepFlag 是每一格的 `+2Eh`，催眠術第 5 段要看它。
	SleepFlag []uint8
	// Asleep 是被催眠的格子。睡著的一輪到就直接結束回合。
	Asleep []bool
	Dexterity     []uint8
	Scores        []uint8
	Budgets       []uint8
	States        []uint8
	DyingCounters []uint8
	BaseMovement  []uint8
	HitPoints     []int
	THAC0         []uint8
	ArmorClass    []int
	Damage        []combat.DamageDice
	Round         int
	Mover         uint8
	Finished      bool
	Prompt        bool
	Outcome       combat.CombatOutcome
	BudgetSource  string
	Status        string
	FoeLog        string
	// Text 由建立者接上 app.text，讓狀態列的訊息也能翻譯。測試直接建構
	// tacticalState 時不設它，say 會退回英文，所以測試不必知道語言這件事。
	Text func(messageID) string
}

// say 取出一則狀態訊息的目前語言版本。
func (state *tacticalState) say(id messageID, args ...any) string {
	format := messages[id][0]
	if state != nil && state.Text != nil {
		format = state.Text(id)
	}
	if len(args) == 0 {
		return format
	}
	return fmt.Sprintf(format, args...)
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
		state.Status = state.say(msgStatusDelayed)
	} else {
		state.Scores[state.Mover] = 0
		state.Status = state.say(msgStatusTurnEnded)
	}
	state.selectActor(roll)
	if state.Mover == 0 {
		state.endRound(roll)
	}
}

// endRound 重現 overlay-08 `0868h` 的回合收尾（spec 062）：先推進倒地計時，
// 再判結束；「我方還在、敵方清光」那一支要多問一次要不要繼續，不是直接結束。
//
// 原版另有一個不問的條件（`DS:4955h` 非 0），但全遊戲只有 overlay-11 `03C1h`
// 的啟動設定寫它，寫的是 0——所以正常遊玩時一律會問，這裡照做（spec 062）。
func (state *tacticalState) endRound(roll func(count, sides int) int) {
	for index := 1; index < len(state.Roster); index++ {
		state.States[index], state.DyingCounters[index] =
			combat.AdvanceDyingCounter(state.States[index], state.DyingCounters[index])
	}
	counts := state.sideCounts()
	if counts.Party > 0 && counts.Foes == 0 {
		state.Prompt = true
		state.Status = state.say(msgStatusContinuePrompt)
		return
	}
	if combat.RoundEndsCombat(counts, 0, false) {
		state.Finished, state.Outcome = true, combat.ResolveCombatOutcome(counts)
		state.Status = state.say(msgStatusDefeat)
		if state.Outcome == combat.CombatVictory {
			state.Status = state.say(msgStatusVictory)
		}
		return
	}
	state.startRound(roll)
	state.Status = state.say(msgStatusRound, state.Round)
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
	roster, friendly, partySlot := provisionalRoster(a, grid, classes)

	base, source := uint8(placeholderBaseMovement), a.text(msgBudgetPlaceholder)
	if len(a.combatMonsters) > 0 {
		base, source = a.combatMonsters[0].Record.Movement(), a.text(msgBudgetStagedMonster)
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
		PartySlot:    partySlot,
		Text:         a.text,
	}
	state.States = make([]uint8, size)
	state.DyingCounters = make([]uint8, size)
	state.HitPoints = make([]int, size)
	state.THAC0 = make([]uint8, size)
	state.ArmorClass = make([]int, size)
	state.Damage = make([]combat.DamageDice, size)
	state.HitDice = make([]uint8, size)
	state.SleepFlag = make([]uint8, size)
	state.Asleep = make([]bool, size)
	for index := 1; index < size; index++ {
		state.BaseMovement[index] = base
		state.Dexterity[index] = placeholderDexterity
		state.HitPoints[index] = placeholderHitPoints
		state.THAC0[index] = placeholderInternalTHAC0
		state.ArmorClass[index] = placeholderInternalArmorClass
		state.Damage[index] = combat.DamageDice{Count: 1, Sides: 8}
		if party := partySlot[index]; party >= 0 && party < len(a.state.Party) {
			member := a.state.Party[party]
			// NPC 沒有走過建角，戰鬥數值直接讀它帶著的原版記錄。
			if member.NPC {
				if err := applyNPCCombatStats(state, index, member); err != nil {
					return err
				}
				continue
			}
			state.Dexterity[index] = uint8(member.Abilities[dexterityAbilityIndex])
			if member.CurrentHP > 0 {
				state.HitPoints[index] = member.CurrentHP
			}
			thac0, armor, movement, err := partyCombatStats(member)
			if err != nil {
				return err
			}
			state.THAC0[index] = thac0
			// `+73h` 是最高職業等級（spec 072 的 overlay-23）。催眠術用它。
			levels := memberClassLevels(member)
			for _, level := range levels {
				if level > state.HitDice[index] {
					state.HitDice[index] = level
				}
			}
			// 穿在身上的東西改 AC 與腳程（spec 079／080）：`partyCombatStats`
			// 回的是建角值，那是「脫光了」的角色。
			armor, movement, err = a.memberDefenceStats(member, armor, movement)
			if err != nil {
				return err
			}
			state.ArmorClass[index] = armor
			state.BaseMovement[index] = movement
			// 手上有裝備好的武器時，THAC0 與傷害改由武器決定（spec 065）。
			if weapon, ok := a.readiedWeapon(member); ok {
				stats, err := a.weaponCombatStats(weapon, member, thac0)
				if err != nil {
					return err
				}
				state.THAC0[index] = stats.Thac0Internal
				state.Damage[index] = combat.DamageDice{
					Count: stats.DamageCount, Sides: stats.DamageSides, Bonus: stats.DamageBonus,
				}
			}
			continue
		}
		if record, ok := a.stagedRecordFor(index, friendly); ok {
			state.BaseMovement[index] = record.Movement()
			state.HitDice[index] = record.Raw[0x73]
			state.SleepFlag[index] = record.Raw[0x2e]
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
	state.Status = state.say(msgStatusRound, state.Round)
	a.tactical = state
	return nil
}

// tacticalSnapshot 把跨影格的狀態組成 combat 層要的那份 DS 快照。佔用格每次
// 重建，與原版每個回合開頭呼叫 overlay-32 entry 20 的做法一致（spec 061）。
func (state *tacticalState) tacticalSnapshot() (combat.TacticalState, error) {
	occupancy, err := combat.RebuildOccupancy(state.Roster)
	if err != nil {
		return combat.TacticalState{}, err
	}
	return combat.TacticalState{
		Map:       state.Grid,
		Occupancy: occupancy,
		Cells:     state.Roster,
		Classes:   state.Classes,
	}, nil
}

// sideOf 把 Friendly 對回原版 combatant record 的 +10Eh 陣營欄位：隊伍 0、敵方 1。
// 查不到的索引一律回 false，讓陣營篩選失敗即關閉。
func (state *tacticalState) sideOf(index uint8) (uint8, bool) {
	if index == 0 || int(index) >= len(state.Friendly) {
		return 0, false
	}
	if state.Friendly[index] {
		return 0, true
	}
	return 1, true
}

// foeSearchBudget 是敵方找目標時給直線追蹤的預算。原版怎麼挑目標還沒讀出來，
// 這個值只用來保證整個盤面都落在搜尋範圍內。
const foeSearchBudget = 128

// foeMaxStepsPerTurn 是一回合內允許的步數上限。預算本身每步遞減、迴圈一定會停，
// 這個上限只是防止未來改動把它變成不會停的迴圈。
const foeMaxStepsPerTurn = 32

// stepTowards 由座標差反查原版方向表，得到朝目標前進的那一個方向。
func stepTowards(fromX, fromY, toX, toY uint8) (uint8, bool) {
	deltaX, deltaY := sign(int(toX)-int(fromX)), sign(int(toY)-int(fromY))
	if deltaX == 0 && deltaY == 0 {
		return 0, false
	}
	for direction := uint8(0); direction < combat.DirectionCount; direction++ {
		step, err := combat.DirectionStep(direction)
		if err != nil {
			continue
		}
		if int(step.X) == deltaX && int(step.Y) == deltaY {
			return direction, true
		}
	}
	return 0, false
}

func sign(value int) int {
	switch {
	case value > 0:
		return 1
	case value < 0:
		return -1
	default:
		return 0
	}
}

// foeTurn 讓敵方的行動者走完一回合。原版的怪物 AI 還沒反組譯，所以「挑哪個
// 目標」與「走哪一步」是暫定策略，畫面上標成 PROVISIONAL AI；策略之外的每一步
// 都走已閉合的原版規則——目標由 spec 056 的鄰近成本表取成本最小者，每一步過
// ResolveDestination（spec 058），攻擊走 spec 050／051。
func (a *app) foeTurn(state *tacticalState) error {
	mover := state.Mover
	snapshot, err := state.tacticalSnapshot()
	if err != nil {
		return err
	}
	side, ok := state.sideOf(mover)
	if !ok {
		return fmt.Errorf("Pool mover %d has no side", mover)
	}
	targets, err := combat.OpposingNearbyAt(snapshot, mover,
		state.Roster[mover].X, state.Roster[mover].Y, foeSearchBudget, 1-side, state.sideOf)
	if err != nil {
		return err
	}
	target := uint8(0)
	if len(targets) != 0 {
		target = targets[0]
	} else if nearest, ok := state.nearestReachableOpposing(mover); ok {
		// 反應距離內沒人時，改追盤面上最近的敵人。
		//
		// **這不是原版的演算法**：原版的敵方回合在 overlay-09 entry 1
		// （code `000Fh`），還沒讀出來。`OpposingNearbyAt` 重現的是
		// overlay-25 entry 32 的「鄰接反應」搜尋（spec 059），拿它當目標選擇
		// 本來就是借用。少了這個退路，站得遠的怪物會回報找不到目標然後原地
		// 結束回合——實測索寇要塞那一場，最後兩隻殭屍與隊伍隔著 22 格互相
		// 不動，戰鬥永遠打不完。
		target = nearest
	}
	if target == 0 {
		state.FoeLog = state.say(msgFoeNoTarget, mover)
		state.endTurn(a.rollDice, false)
		return nil
	}

	// 目標在這一回合裡不會換，所以步數表只算一次。
	goalCell := state.Roster[target]
	stepDistance := tacticalStepDistances(state.Grid, state.Classes, goalCell.X, goalCell.Y)

	steps := 0
	for ; steps < foeMaxStepsPerTurn; steps++ {
		snapshot, err = state.tacticalSnapshot()
		if err != nil {
			return err
		}
		here := state.Roster[mover]
		goal := state.Roster[target]
		// 八個方向都問一次，挑「走得進去而且離目標最近」的那一個。
		//
		// 只問 `stepTowards` 給的那一個方向是不夠的：實測最後一隻殭屍站在
		// (42,10)、目標在西邊，而它西邊那兩格是牆，其餘六個方向全都走得進去
		// ——原本的寫法在那一個方向上撞牆就收工，回報「走了零步」，於是雙方
		// 隔著地形永遠對峙。
		//
		// **繞路的規則不是原版的**：原版的敵方回合在 overlay-09 entry 1
		// （code `000Fh`），還沒讀。這裡只挑一步，不做完整選路。
		bestDirection, bestDistance := -1, tacticalDistanceAt(stepDistance, here.X, here.Y, goal)
		// 先問原版的方向表要的那一格；走得進去就走，繞路只是備案。
		preferred := -1
		if direction, ok := stepTowards(here.X, here.Y, goal.X, goal.Y); ok {
			preferred = int(direction)
		}
		order := make([]uint8, 0, 8)
		if preferred >= 0 {
			order = append(order, uint8(preferred))
		}
		for direction := uint8(0); direction < 8; direction++ {
			if int(direction) != preferred {
				order = append(order, direction)
			}
		}
		for _, direction := range order {
			outcome, err := combat.ResolveDestination(snapshot, mover, direction, state.Budget())
			if err != nil {
				return err
			}
			if outcome.Action == combat.MovementAttack {
				if same, err := state.sameSide(mover, outcome.Target); err != nil {
					return err
				} else if same {
					continue
				}
				if err := a.resolveTacticalAttack(state, outcome.Target); err != nil {
					return err
				}
				state.FoeLog = state.say(msgFoeAttacked, mover, steps, state.Status)
				state.endTurn(a.rollDice, false)
				return nil
			}
			if outcome.Action != combat.MovementEnter {
				continue
			}
			x, y, err := combat.AdvanceTacticalCoordinate(here.X, here.Y, direction)
			if err != nil {
				return err
			}
			distance := tacticalDistanceAt(stepDistance, x, y, goal)
			if distance >= bestDistance {
				continue
			}
			bestDirection, bestDistance = int(direction), distance
			if int(direction) == preferred {
				// 方向表要的那一格走得進去，就不必再看別的。
				break
			}
		}
		if bestDirection < 0 {
			break
		}
		direction := uint8(bestDirection)
		x, y, err := combat.AdvanceTacticalCoordinate(here.X, here.Y, direction)
		if err != nil {
			return err
		}
		budget, err := combat.SpendMovementStep(state.Budget(), direction)
		if err != nil {
			return err
		}
		if budget == state.Budget() {
			break
		}
		state.Roster[mover].X, state.Roster[mover].Y = x, y
		state.Budgets[mover] = budget
	}
	state.FoeLog = state.say(msgFoeClosed, mover, steps, target)
	state.endTurn(a.rollDice, false)
	return nil
}

// 建角寫下的三個基礎值，逐一取自 overlay-16（spec 063）：AC internal 32h
// （typed 10）、THAC0 internal 28h（typed 20）、基礎移動 0Ch。
const (
	creationArmorClassInternal = 0x32
	creationThac0Internal      = 0x28
	creationBaseMovement       = 0x0C
)

// firstCharacterLevel 是還沒訓練過的角色的等級。建角每個組成職業各寫下第 1 級
// （spec 072），存檔裡沒有 ClassLevels 就是這個狀態。
const firstCharacterLevel = 1

// partyCombatStats 依 spec 063 由職業算出隊伍成員的基礎戰鬥數值：THAC0 逐個
// component 查 DS:3C16h 的表取最好的一個，AC 與移動用建角寫下的基礎值。
//
// 裝備尚未接進戰鬥，所以這裡回的是「沒有裝備」的角色——原版穿上裝備之後還會
// 重算 AC，那條鏈（overlay-25 的 sub_281／sub_39F）還沒閉合。
func partyCombatStats(member poolsave.Character) (thac0Internal uint8, armorInternal int, movement uint8, err error) {
	levels, err := partyClassLevels(member)
	if err != nil {
		return 0, 0, 0, err
	}
	thac0Internal, err = gamepack.BaseThac0Internal(levels)
	if err != nil {
		return 0, 0, 0, err
	}
	return thac0Internal, creationArmorClassInternal, creationBaseMovement, nil
}

// memberDefenceStats 把角色身上的東西算進 AC 與移動力（spec 079／080）。
// 兩者共用同一條物品鏈，原版也是在同一支 overlay-25 `0C17h` 裡一起算的，
// 分開走會出現「AC 算了裝備、腳程沒算」這種只在特定隊伍才看得出來的偏差。
func (a *app) memberDefenceStats(member poolsave.Character, baseArmor int, baseMovement uint8) (int, uint8, error) {
	// 沒有任何物品也要走完：敏捷的 AC 調整與硬幣的重量都不看物品鏈，
	// 提早返回會讓空手的角色少掉敏捷那一項。
	items := make([][]byte, 0, len(member.Inventory))
	for _, item := range member.Inventory {
		items = append(items, item.Raw)
	}
	armour, err := gamepack.ArmourClassFor(baseArmor,
		member.Abilities[dexterityAbilityIndex], items, a.itemTypes)
	if err != nil {
		return 0, 0, fmt.Errorf("Pool character %q armour class: %w", member.Name, err)
	}
	carried, err := gamepack.CarriedWeight(items, member.Money)
	if err != nil {
		return 0, 0, fmt.Errorf("Pool character %q carried weight: %w", member.Name, err)
	}
	movement, err := gamepack.MovementRateFor(int(baseMovement), member.Abilities[0],
		member.ExceptionalStrength, carried, items, a.itemTypes)
	if err != nil {
		return 0, 0, fmt.Errorf("Pool character %q movement: %w", member.Name, err)
	}
	return armour.Internal, uint8(movement), nil
}

// applyNPCCombatStats 用 NPC 自己帶的 285-byte 記錄填戰鬥數值，做法與已
// staged 的怪物同一套（spec 091）。NPC 不走建角那一組欄位，硬套會得到
// 一個「一級戰士」，而那與原版差很多。
func applyNPCCombatStats(state *tacticalState, index int, member poolsave.Character) error {
	if len(member.Record) != poolsave.NPCRecordSize {
		return fmt.Errorf("Pool NPC %q has a %d-byte record", member.Name, len(member.Record))
	}
	var record gamepack.MonsterRecord
	copy(record.Raw[:], member.Record)
	record.Name = member.Name
	state.BaseMovement[index] = record.Movement()
	state.HitPoints[index] = int(record.CurrentHitPoints())
	state.THAC0[index] = uint8(60 - record.THAC0())
	state.ArmorClass[index] = 60 - record.ArmorClass()
	state.Damage[index] = combat.DamageDice{
		Count: record.DamageDiceCount(),
		Sides: record.DamageDieSides(),
		Bonus: record.DamageBonus(),
	}
	return nil
}

// tacticalDistanceAt 查步數表；查不到（那一格與目標之間沒有通路）就退回
// 直線距離，讓行為不會比先前差。
func tacticalDistanceAt(distance map[int]int, x, y uint8, goal combat.CombatantCell) int {
	if step, ok := distance[tacticalCellKey(x, y)]; ok {
		return step
	}
	return chebyshev(x, y, goal.X, goal.Y) + len(distance)
}

// tacticalCellKey 把一格壓成一個查表用的鍵。
func tacticalCellKey(x, y uint8) int { return int(y)*256 + int(x) }

// tacticalStepDistances 從目標往外做一次寬度優先，回傳每一格到目標的步數。
//
// 挑方向要用**繞得過去的實際步數**，不是直線距離：戰場是原版的斜投影又多牆
// （spec 060），直線距離會把人帶進死角然後在那裡來回，盤面一擠就再也靠不近。
//
// 只看地形擋不擋路，不看誰站在那裡——佔用格每一步都在變，把它算進去會讓
// 同一條路每走一步就得到不同的答案。
//
// **這不是原版的選路**：原版（overlay-31 entry 6，spec 096）是先列出目標
// 周圍可站的格子再挑。這裡只是讓「繞得過去」這件事成立。
func tacticalStepDistances(grid combat.TacticalGrid, classes combat.CellClasses, targetX, targetY uint8) map[int]int {
	distance := map[int]int{tacticalCellKey(targetX, targetY): 0}
	queue := [][2]uint8{{targetX, targetY}}
	for len(queue) != 0 {
		cell := queue[0]
		queue = queue[1:]
		step := distance[tacticalCellKey(cell[0], cell[1])] + 1
		for direction := uint8(0); direction < 8; direction++ {
			x, y, err := combat.AdvanceTacticalCoordinate(cell[0], cell[1], direction)
			if err != nil {
				continue
			}
			key := tacticalCellKey(x, y)
			if _, seen := distance[key]; seen {
				continue
			}
			terrain, err := grid.TerrainAt(int(x), int(y))
			if err != nil {
				continue
			}
			record, err := combat.CellClassAt(classes, terrain)
			if err != nil || record.EntryThreshold >= 0xFF {
				continue
			}
			distance[key] = step
			queue = append(queue, [2]uint8{x, y})
		}
	}
	return distance
}

// partyClassLevels 把角色攤成原版記錄 `+96h` 起那八個職業等級。THAC0
// （spec 063）與豁免目標值（spec 075）查的是同一組索引，所以只算一次。
func partyClassLevels(member poolsave.Character) ([gamepack.ClassThac0ClassCount]uint8, error) {
	var levels [gamepack.ClassThac0ClassCount]uint8
	// 訓練過的角色帶著自己的八個等級（spec 097）；沒有的照建角的第 1 級算。
	if len(member.ClassLevels) > 0 {
		copy(levels[:], member.ClassLevels)
		return levels, nil
	}
	components, ok := creation.ClassComponents(member.ClassID)
	if !ok {
		return levels, fmt.Errorf("Pool character %q has unknown class %q", member.Name, member.ClassID)
	}
	for _, component := range components {
		index, ok := creation.ComponentClassIndex(component)
		if !ok {
			return levels, fmt.Errorf("Pool class component %q has no index", component)
		}
		if int(index) >= len(levels) {
			return levels, fmt.Errorf("Pool class component %q index %d is outside the table", component, index)
		}
		levels[index] = firstCharacterLevel
	}
	return levels, nil
}

// 物品記錄裡本規格用到的三個欄位（spec 033／035／063）。
const (
	itemTypeOffset  = 0x2e // 物品型別索引，查 DS:54E0h 那張表用
	itemPlusOffset  = 0x32 // 附魔值，武器的 +1／+2
	itemReadyOffset = 0x34 // 非零代表這件已經裝備上

	// itemCategoryWeapon 是型別表 `+0` 的武器類別，對應角色記錄的 `+CCh` 槽。
	itemCategoryWeapon = 0
)

// readiedWeapon 取出角色手上的武器。
//
// 原版把每件穿戴中的物品依型別表的類別放進 `+CCh + 類別 × 4` 的槽
//（overlay-25 `0C76h`，類別 0..8；類別 9 另外走 `+F0h`／`+F4h` 兩個戒指槽），
// 而判斷「有沒有武器」讀的是類別 0 那一格（`0E81h` 檢查 `+CCh`／`+CEh` 是不是
// 空指標，空的就走徒手那一支）。所以武器是**類別 0** 的那一件，不是物品鏈上
// 第一件標成裝備的東西——預設人物身上第一件裝備多半是戒指，那東西的傷害骰
// 是 0d0，拿它當武器整隊會打不出傷害。
//
// 槽是覆寫不是累加，所以同類別有多件時**最後一件**贏，這裡照同樣的順序。
func (a *app) readiedWeapon(member poolsave.Character) (poolsave.Item, bool) {
	var weapon poolsave.Item
	found := false
	for _, item := range member.Inventory {
		if len(item.Raw) <= itemReadyOffset || item.Raw[itemReadyOffset] == 0 {
			continue
		}
		entry, err := a.itemTypes.Entry(item.Raw[itemTypeOffset])
		if err != nil || entry.Category() != itemCategoryWeapon {
			continue
		}
		weapon, found = item, true
	}
	return weapon, found
}

// weaponCombatStats 是畫面與戰鬥共用的那一條規則：兩邊分開算，會出現
// 裝備頁顯示一組數字、打起來卻是另一組。
func (a *app) weaponCombatStats(weapon poolsave.Item, member poolsave.Character, baseThac0Internal uint8) (gamepack.WeaponStats, error) {
	if len(weapon.Raw) <= itemPlusOffset {
		return gamepack.WeaponStats{}, fmt.Errorf("Pool item %q is too short to be a weapon", weapon.Name)
	}
	return gamepack.WeaponCombatStats(a.itemTypes,
		weapon.Raw[itemTypeOffset], int(int8(weapon.Raw[itemPlusOffset])),
		weaponBearerFor(member, baseThac0Internal))
}

// weaponBearerFor 把角色接成武器規則要的形狀。
//
// `AbilityBonusesEnabled` 對應角色記錄的 `+0AAh`：原版以它決定要不要套用兩個
// 力量修正，但那個 byte 由誰寫、代表什麼還沒閉合。remake 的角色都是正常
// 建角出來的，所以先照「開著」接；等 `+0AAh` 的 producer 讀出來再改。
func weaponBearerFor(member poolsave.Character, baseThac0Internal uint8) gamepack.WeaponBearer {
	return gamepack.WeaponBearer{
		BaseThac0Internal:     baseThac0Internal,
		Strength:              member.Abilities[0],
		ExceptionalStrength:   member.ExceptionalStrength,
		Dexterity:             member.Abilities[dexterityAbilityIndex],
		AbilityBonusesEnabled: true,
	}
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
	if state.Prompt {
		if a.justPressed(ebiten.KeyY) {
			state.Prompt = false
			state.startRound(a.rollDice)
			state.Status = state.say(msgStatusRound, state.Round)
		}
		if a.justPressed(ebiten.KeyN) {
			state.Prompt = false
			state.Finished, state.Outcome = true, combat.ResolveCombatOutcome(state.sideCounts())
			return a.finishCombat(state.Outcome)
		}
		return nil
	}
	// 睡著的一輪到就直接結束回合（spec 098 的催眠術）。原版是把效果碼掛上去
	// 之後由行動判定擋下來；這裡先用一個旗標，效果串列還沒接進戰鬥。
	if state.Mover != 0 && int(state.Mover) < len(state.Asleep) && state.Asleep[state.Mover] {
		state.Status = state.say(msgStatusAsleep, state.Mover)
		state.endTurn(a.rollDice, false)
		if state.Finished {
			return a.finishCombat(state.Outcome)
		}
		return nil
	}
	if state.Mover != 0 && int(state.Mover) < len(state.Friendly) && !state.Friendly[state.Mover] {
		if err := a.foeTurn(state); err != nil {
			return err
		}
		if state.Finished {
			return a.finishCombat(state.Outcome)
		}
		return nil
	}
	if a.castTargeting {
		return a.castTargetingInput()
	}
	if a.castOpen {
		return a.castInput()
	}
	if a.justPressed(ebiten.KeyC) && state.Mover != 0 {
		a.openCastMenu()
		return nil
	}
	if a.justPressed(ebiten.KeyA) && state.Mover != 0 {
		a.beginAimedAttack()
		return nil
	}
	if a.justPressed(ebiten.KeyEnter) {
		state.endTurn(a.rollDice, false)
		if state.Finished {
			return a.finishCombat(state.Outcome)
		}
		return nil
	}
	if a.justPressed(ebiten.KeyD) {
		state.endTurn(a.rollDice, true)
		if state.Finished {
			return a.finishCombat(state.Outcome)
		}
		return nil
	}
	if state.Mover == 0 {
		return nil
	}
	for direction, key := range tacticalStepKeys {
		if !a.justPressed(key) {
			continue
		}
		tactical, err := state.tacticalSnapshot()
		if err != nil {
			return err
		}
		outcome, err := combat.ResolveDestination(tactical, state.Mover, uint8(direction), state.Budget())
		if err != nil {
			return err
		}
		switch {
		case outcome.Leaving:
			state.Status = state.say(msgStatusOffBoard)
		case outcome.Action == combat.MovementAttack:
			// 撞到自己人不打自己人。`ProbeDestination` 忠實重現原版，它只回報
			// 「那一格站著誰」——原版的格位表（`DS:5E89h`）本來就沒有陣營，
			// 陣營在角色記錄的 `+10Eh`，所以這個判斷是呼叫端的責任。
			// 少了它，隊伍排成一列時最左邊那個往右走就會砍死自己的同伴，
			// 而戰鬥永遠打不完。
			//
			// 待證：原版撞到同伴是「擋住」還是「換位」。這裡先擋住。
			if same, err := state.sameSide(state.Mover, outcome.Target); err != nil {
				return err
			} else if same {
				state.Status = state.say(msgStatusBlocked)
				return nil
			}
			if err := a.resolveTacticalAttack(state, outcome.Target); err != nil {
				return err
			}
			// 攻擊就用掉這一次行動。原版的玩家指令迴圈（overlay-08 `0307h`）
			// 把「這一回合結束了嗎」的旗標位址交給攻擊常式
			// （`0096h:0089h`，`03D7h` 那個 `lea -2(bp)`），由它決定要不要
			// 回到 `036Ch` 再問下一個指令；敵方回合（`foeTurn`）打完也是直接
			// `endTurn`。少了這一步，同一個角色可以對同一個目標無限連打。
			//
			// 待證：戰士的多次攻擊（記錄 `+A1h`，spec 051）還沒接，接上之後
			// 這裡要改成「打完所有攻擊次數才結束」。
			state.endTurn(a.rollDice, false)
			if state.Finished {
				return a.finishCombat(state.Outcome)
			}
		case outcome.Action == combat.MovementBlocked:
			state.Status = state.say(msgStatusBlocked)
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
			state.Status = state.say(msgStatusMoved, direction)
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
		state.Status = state.say(msgStatusMissed, target, roll)
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
		state.Status = state.say(msgStatusHit, target, damage, state.HitPoints[target])
		return nil
	}
	state.HitPoints[target] = 0
	state.Roster[target].FootprintClass = 0
	state.Scores[target] = 0
	state.States[target] = combat.DyingState
	state.Status = state.say(msgStatusDown, target)
	return nil
}

// finishCombat 依 spec 046 契約 5 處理戰後：只有勝利才從 COMBAT 邊界停下的 PC
// 續跑戰後腳本；戰敗不得續跑，也不得用自動勝利代替戰鬥結果。
func (a *app) finishCombat(outcome combat.CombatOutcome) error {
	staged := a.combatActive
	a.tacticalPreview, a.tactical = false, nil
	a.castOpen, a.castOptions, a.castCursor = false, nil, 0
	a.castTargeting, a.castTargets, a.castTargetCursor = false, nil, 0
	a.castTargetingAttack = false
	if outcome != combat.CombatVictory {
		a.statusLine = "Party defeated; the post-combat script does not run."
		return nil
	}
	if !staged {
		// F5 開的是預覽盤面，不是 ECL 排出來的遭遇，所以沒有戰後腳本可以續跑。
		a.statusLine = "Tactical preview finished; no encounter was staged."
		return nil
	}
	a.awardCombatExperience()
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

// sideCounts 數出兩邊還站著的人，對應原版的 DS:6772h 與 DS:6773h。
// nearestReachableOpposing 先挑「直線走得到」的敵人，沒有才退回最近的那個。
//
// 走得到與否用 `combat.TraceMovement`——那支是 overlay-31 `0419h` 的重現
// （spec 057），原版本來就是用它判斷地形擋不擋路。少了這一層，怪物會盯著
// 一個隔著牆的目標，然後每回合往牆上撞、回報走了零步。
//
// **挑目標的規則本身還不是原版的**：原版的敵方回合在 overlay-09 entry 1，
// 還沒讀。這裡只是讓「盯著走不到的目標」不再發生。
func (state *tacticalState) nearestReachableOpposing(mover uint8) (uint8, bool) {
	best, bestDistance := uint8(0), 0
	from := state.Roster[mover]
	budget := uint16(state.Budgets[mover])
	for index := 1; index < len(state.Roster); index++ {
		if state.Friendly[index] == state.Friendly[mover] || state.Roster[index].FootprintClass == 0 {
			continue
		}
		to := state.Roster[index]
		trace, err := combat.TraceMovement(state.Grid, state.Classes,
			int(from.X), int(from.Y), int(to.X), int(to.Y), budget)
		if err != nil || !trace.Complete {
			continue
		}
		// 距離用原版的算法：TraceMovement 的成本除以二（spec 098）。
		// 成本本來就以半格為單位（`limit := budget*2 + 1`），所以斜走與
		// 直走不同價；切比雪夫距離在斜投影的盤面上會低估。
		distance := int(trace.Cost) / 2
		if best == 0 || distance < bestDistance {
			best, bestDistance = uint8(index), distance
		}
	}
	if best != 0 {
		return best, true
	}
	return state.nearestOpposing(mover)
}

// nearestOpposing 回報盤面上離 mover 最近、還站著的敵對參戰者。
// 距離用原版走位的切比雪夫距離（八方向一步一格）。
func (state *tacticalState) nearestOpposing(mover uint8) (uint8, bool) {
	if int(mover) >= len(state.Friendly) || mover == 0 {
		return 0, false
	}
	best, bestDistance := uint8(0), 0
	from := state.Roster[mover]
	for index := 1; index < len(state.Roster); index++ {
		if state.Friendly[index] == state.Friendly[mover] {
			continue
		}
		if state.Roster[index].FootprintClass == 0 {
			continue
		}
		to := state.Roster[index]
		distance := chebyshev(from.X, from.Y, to.X, to.Y)
		if best == 0 || distance < bestDistance {
			best, bestDistance = uint8(index), distance
		}
	}
	return best, best != 0
}

// chebyshev 是八方向走位下的步數距離。
func chebyshev(ax, ay, bx, by uint8) int {
	dx, dy := int(ax)-int(bx), int(ay)-int(by)
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	if dx > dy {
		return dx
	}
	return dy
}

// sameSide 說兩個參戰者是不是同一邊。
func (state *tacticalState) sameSide(a, b uint8) (bool, error) {
	if int(a) >= len(state.Friendly) || int(b) >= len(state.Friendly) || a == 0 || b == 0 {
		return false, fmt.Errorf("Pool combatant index %d or %d is outside the roster", a, b)
	}
	return state.Friendly[a] == state.Friendly[b], nil
}

func (state *tacticalState) sideCounts() combat.SideCounts {
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
	return counts
}

// awardCombatExperience 把這一場的經驗值發給隊伍（spec 097）。
//
// 原版是先把所有敵方的經驗值加總、除以「有資格分的人數」，再由每個人依自己的
// 複合職業碼調整：純職業的主屬性超過 15 多拿十分之一，複合職業除以職業數。
//
// **有資格的判準還沒讀完**：原版跳過 `+10Dh` 為 0 與狀態為 1 的成員，兩個欄位
// 的語意都還沒閉合，所以這裡讓全隊都分。倒下的成員在原版一樣分得到——
// 它擋的不是死亡。
func (a *app) awardCombatExperience() {
	if len(a.state.Party) == 0 || len(a.combatMonsters) == 0 {
		return
	}
	total := uint32(0)
	for _, monster := range a.combatMonsters {
		value := monster.Record.ExperienceValue(int(monster.Record.MaxHitPoints()))
		total += value * uint32(monster.Spawn.Count)
	}
	share := gamepack.DivideExperience(total, len(a.state.Party))
	if share == 0 {
		return
	}
	for index := range a.state.Party {
		member := &a.state.Party[index]
		code, ok := creation.ClassDOSCode(member.ClassID)
		if !ok {
			// NPC 帶的是自己的 285-byte 記錄，職業碼在 `+2Fh`。
			if len(member.Record) > gamepack.ClassCodeOffset {
				code = member.Record[gamepack.ClassCodeOffset]
			} else {
				continue
			}
		}
		member.Experience += gamepack.ExperienceShare(share, code, member.Abilities)
	}
}


// tacticalRange 是原版算兩個參戰者之間距離的方式（spec 098）：
// overlay-25 `2591h` 用 overlay-31 entry 6 建的清單找到目標那一格，
// 取**路徑成本除以二**。成本由 `0419h`（TraceMovement，spec 057）算，
// 本來就以半格為單位。
//
// 走不到（地形擋住或超出預算）回 false，呼叫端要當成「打不到」。
func (state *tacticalState) tacticalRange(from, to uint8) (int, bool) {
	if int(from) >= len(state.Roster) || int(to) >= len(state.Roster) {
		return 0, false
	}
	source, target := state.Roster[from], state.Roster[to]
	// 預算給滿：算距離不受這一回合剩多少步影響。
	trace, err := combat.TraceMovement(state.Grid, state.Classes,
		int(source.X), int(source.Y), int(target.X), int(target.Y), 0xFF)
	if err != nil || !trace.Complete {
		return 0, false
	}
	return int(trace.Cost) / 2, true
}
