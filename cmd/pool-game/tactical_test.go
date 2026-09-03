package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
	"github.com/wicanr2/golden-box-remake-engine/geometry"
)

// 沒有牆 → 0；有牆而該方向的 detail 為 0 → 1；有牆且 detail 非 0 → 3。
func TestGeoWallProbeMapsGeoDataToTheThreeOriginalValues(t *testing.T) {
	var grid geometry.Grid
	grid.Cells[5][4].WallDirections = [4]uint8{0, 3, 0, 1}
	grid.Cells[5][4].DetailDirections = [4]uint8{0, 2, 0, 0}
	probe := geoWallProbe(grid, 5)

	got, err := probe(combat.WallDirectionNorth, 4, 5)
	if err != nil {
		t.Fatal(err)
	}
	if got != combat.WallOpen {
		t.Fatalf("an absent wall mapped to %d, want %d", got, combat.WallOpen)
	}
	if got, _ = probe(combat.WallDirectionWest, 4, 5); got != combat.WallBlocking {
		t.Fatalf("a wall with no detail mapped to %d, want %d", got, combat.WallBlocking)
	}
	if got, _ = probe(combat.WallDirectionEast, 4, 5); got != combat.WallAlternate {
		t.Fatalf("a wall with detail mapped to %d, want %d", got, combat.WallAlternate)
	}
}

// 原版不取模：界外一律是牆，只有隊伍那一列的東西向例外。
func TestGeoWallProbeTreatsOutsideTheGridAsWall(t *testing.T) {
	var grid geometry.Grid
	probe := geoWallProbe(grid, 5)

	if got, _ := probe(combat.WallDirectionNorth, -1, 5); got != combat.WallBlocking {
		t.Fatalf("north of the grid mapped to %d", got)
	}
	if got, _ := probe(combat.WallDirectionEast, -1, 5); got != combat.WallOpen {
		t.Fatalf("east on the party row mapped to %d, want open", got)
	}
	if got, _ := probe(combat.WallDirectionEast, -1, 7); got != combat.WallBlocking {
		t.Fatalf("east off the party row mapped to %d, want blocking", got)
	}
}

// 生成器要能吃下這個 probe 並鋪滿整張盤面。
func TestGeoWallProbeDrivesTheGenerator(t *testing.T) {
	var grid geometry.Grid
	built, err := combat.GenerateIndoorTacticalGrid(8, 8, geoWallProbe(grid, 8))
	if err != nil {
		t.Fatal(err)
	}
	for index, code := range built.Terrain {
		if code == combat.UnpaintedCellClass {
			t.Fatalf("cell %d was left unpainted", index)
		}
	}
}

func fixedRoll(int, int) int { return 3 }

func newRoundState(members int) *tacticalState {
	size := members + 1
	state := &tacticalState{
		Roster:        make([]combat.CombatantCell, size),
		Friendly:      make([]bool, size),
		Dexterity:     make([]uint8, size),
		Scores:        make([]uint8, size),
		Budgets:       make([]uint8, size),
		BaseMovement:  make([]uint8, size),
		States:        make([]uint8, size),
		DyingCounters: make([]uint8, size),
	}
	for index := 1; index < size; index++ {
		state.Dexterity[index] = 12
		state.BaseMovement[index] = 9
		state.Roster[index].FootprintClass = 1
	}
	// 兩邊都要有人站著，回合收尾才不會把測試盤面判成已經分出勝負。
	state.Friendly[1] = true
	return state
}

// 每個回合都重擲先攻並重設移動預算，不是整場排一次。
func TestStartRoundResetsBudgetsAndScores(t *testing.T) {
	state := newRoundState(2)
	state.startRound(fixedRoll)
	if state.Round != 1 {
		t.Fatalf("round %d, want 1", state.Round)
	}
	want := combat.InitialMovementBudgetBeforeEffects(9, false, 0)
	for index := 1; index < len(state.Budgets); index++ {
		if state.Budgets[index] != want {
			t.Fatalf("combatant %d budget %d, want %d", index, state.Budgets[index], want)
		}
		if state.Scores[index] == 0 {
			t.Fatalf("combatant %d was not given an initiative score", index)
		}
	}
	if state.Mover == 0 {
		t.Fatal("no actor was selected")
	}
}

// 全部行動完才進下一回合，且預算會重設。
func TestEndTurnAdvancesTheRoundOnlyWhenNobodyIsLeft(t *testing.T) {
	state := newRoundState(2)
	state.startRound(fixedRoll)
	state.Budgets[state.Mover] = 0

	state.endTurn(fixedRoll, false)
	if state.Round != 1 {
		t.Fatalf("the round advanced with an actor still to go: round %d", state.Round)
	}
	if state.Mover == 0 {
		t.Fatal("the second actor was not selected")
	}

	state.endTurn(fixedRoll, false)
	if state.Round != 2 {
		t.Fatalf("round %d after everyone acted, want 2", state.Round)
	}
	want := combat.InitialMovementBudgetBeforeEffects(9, false, 0)
	for index := 1; index < len(state.Budgets); index++ {
		if state.Budgets[index] != want {
			t.Fatalf("combatant %d budget %d was not reset", index, state.Budgets[index])
		}
	}
}

// Delay 把分數寫成 1 而不是 0，所以那名角色稍後還會被選到。
func TestDelayKeepsTheActorSelectable(t *testing.T) {
	state := newRoundState(1)
	state.startRound(fixedRoll)
	actor := state.Mover
	state.endTurn(fixedRoll, true)
	if state.Round != 1 {
		t.Fatalf("delay advanced the round to %d", state.Round)
	}
	if state.Mover != actor {
		t.Fatalf("mover %d after delay, want the same actor %d", state.Mover, actor)
	}
	if state.Scores[actor] != combat.DelayInitiative() {
		t.Fatalf("delayed score %d, want %d", state.Scores[actor], combat.DelayInitiative())
	}
}

// fixedRoller 依骰面夾住，免得 d20 的固定值被拿去當 d8 的結果。
type fixedRoller struct{ value int }

func (roller fixedRoller) Roll(_, sides int) int {
	if roller.value > sides {
		return sides
	}
	return roller.value
}

func newAttackState() *tacticalState {
	state := newRoundState(2)
	state.Friendly[1] = true
	state.HitPoints = []int{0, 10, 6}
	state.THAC0 = []uint8{0, 40, 40}
	state.ArmorClass = []int{0, 50, 50}
	state.Damage = []combat.DamageDice{{}, {Count: 1, Sides: 8}, {Count: 1, Sides: 8}}
	state.Roster[1].FootprintClass = 1
	state.Roster[2].FootprintClass = 1
	state.Mover = 1
	return state
}

// d20 為 1 一定失手，目標的 HP 不動。
func TestResolveTacticalAttackMisses(t *testing.T) {
	state := newAttackState()
	a := &app{roller: fixedRoller{1}, tactical: state}
	if err := a.resolveTacticalAttack(state, 2); err != nil {
		t.Fatal(err)
	}
	if state.HitPoints[2] != 6 {
		t.Fatalf("target hit points %d after a miss", state.HitPoints[2])
	}
}

// 打倒目標之後它的體型類別歸零、轉成倒地狀態；戰鬥不在這一刻結束——
// spec 062 契約 6 的結束旗標是回合收尾產出的。
func TestResolveTacticalAttackDownsTheTargetWithoutEndingTheCombat(t *testing.T) {
	state := newAttackState()
	a := &app{roller: fixedRoller{20}, tactical: state}
	if err := a.resolveTacticalAttack(state, 2); err != nil {
		t.Fatal(err)
	}
	if state.HitPoints[2] != 0 {
		t.Fatalf("target hit points %d, want 0", state.HitPoints[2])
	}
	if state.Roster[2].FootprintClass != 0 {
		t.Fatal("a downed combatant still occupies its cells")
	}
	if state.States[2] != combat.DyingState {
		t.Fatalf("downed combatant state %d, want %d", state.States[2], combat.DyingState)
	}
	if state.Finished {
		t.Fatal("the attack ended the combat; only the round end may do that")
	}
}

// 兩邊都還有人時回合收尾不結束戰鬥，直接開下一回合。
func TestEndRoundStartsTheNextRoundWhileBothSidesStand(t *testing.T) {
	state := newAttackState()
	state.endRound(fixedRoll)
	if state.Finished || state.Prompt {
		t.Fatalf("finished %v prompt %v, want an ongoing combat", state.Finished, state.Prompt)
	}
	if state.Round != 1 {
		t.Fatalf("round %d, want the next round to have started", state.Round)
	}
}

// 我方全倒就是敗，而且不問要不要繼續。
func TestEndRoundReportsDefeatWhenThePartyIsGone(t *testing.T) {
	state := newAttackState()
	state.Roster[1].FootprintClass = 0
	state.endRound(fixedRoll)
	if !state.Finished || state.Outcome != combat.CombatDefeat {
		t.Fatalf("finished %v outcome %v, want defeat", state.Finished, state.Outcome)
	}
	if state.Prompt {
		t.Fatal("a defeated party was asked whether to fight on")
	}
}

// spec 062 契約 5：清光敵人之後要先問一次，答 N 才結束。
func TestEndRoundAsksBeforeEndingAClearedBattle(t *testing.T) {
	state := newAttackState()
	state.Roster[2].FootprintClass = 0
	state.endRound(fixedRoll)
	if state.Finished {
		t.Fatal("clearing the foes ended the battle without asking")
	}
	if !state.Prompt {
		t.Fatal("the continue prompt did not come up")
	}
}

// 倒地者每個回合加一，撐過第九回合才轉成另一個狀態。
func TestEndRoundAdvancesTheDyingCounter(t *testing.T) {
	state := newAttackState()
	state.States[2] = combat.DyingState
	for round := 0; round < int(combat.DyingRoundLimit); round++ {
		state.endRound(fixedRoll)
		if state.States[2] != combat.DyingState {
			t.Fatalf("state %d after %d rounds, want it still dying", state.States[2], round+1)
		}
	}
	state.endRound(fixedRoll)
	if state.States[2] != combat.DeadState {
		t.Fatalf("state %d after the limit, want %d", state.States[2], combat.DeadState)
	}
}

// Spec 046 契約 5：戰敗不得續跑戰後 ECL，而且要停下那條玩家路徑。
// 這個測試刻意不給 eventSession——一旦 defeat 走到續跑就會 panic，
// 所以它同時證明那條路徑碰不到 session。
//
// 排好的遭遇也要一起清掉。留著的話同一場架會被重新排出來，而戰鬥的生命值
// 是另一份陣列、沒有寫回隊伍，於是隊伍又是滿血——打輸、重來、再打輸，
// 那個迴圈不會停。
func TestFinishCombatDoesNotRunThePostCombatScriptOnDefeat(t *testing.T) {
	a := &app{combatActive: true, tacticalPreview: true, tactical: &tacticalState{},
		combatMonsters: []stagedMonster{{}}}
	if err := a.finishCombat(combat.CombatDefeat); err != nil {
		t.Fatal(err)
	}
	if a.combatActive || a.combatMonsters != nil {
		t.Fatal("defeat left the encounter staged; the same fight restarts for ever")
	}
	if a.tacticalPreview || a.tactical != nil {
		t.Fatal("the tactical screen stayed open after defeat")
	}
}

// stepTowards 反查的就是原版那張方向表：0 向上、順時針一圈。
func TestStepTowardsUsesTheOriginalDirectionTable(t *testing.T) {
	for _, test := range []struct {
		name           string
		toX, toY, want uint8
	}{
		{"east", 11, 10, 2},
		{"north", 10, 9, 0},
		{"south west", 9, 11, 5},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, ok := stepTowards(10, 10, test.toX, test.toY)
			if !ok || got != test.want {
				t.Fatalf("got %d (ok %v), want %d", got, ok, test.want)
			}
		})
	}
	if _, ok := stepTowards(10, 10, 10, 10); ok {
		t.Fatal("a combatant standing on the target produced a direction")
	}
}

func newFoeTurnState(foeX, foeY, partyX, partyY uint8, budget uint8) *tacticalState {
	state := newRoundState(2)
	cellCount := combat.TacticalRowStride * (combat.TacticalMaxY + 1)
	// 地形碼 0 在目的格探測裡代表盤面外，所以測試盤面要鋪一個非 0 的可通行碼。
	const openTerrain = 5
	terrain := make([]uint8, cellCount)
	for index := range terrain {
		terrain[index] = openTerrain
	}
	state.Grid = combat.TacticalGrid{Terrain: terrain}
	state.Classes[openTerrain] = gamepack.CombatCellClass{EntryThreshold: 1}
	state.Friendly[1] = true
	state.Roster[1] = combat.CombatantCell{X: partyX, Y: partyY, FootprintClass: 1}
	state.Roster[2] = combat.CombatantCell{X: foeX, Y: foeY, FootprintClass: 1}
	state.HitPoints = []int{0, 10, 10}
	state.THAC0 = []uint8{0, 40, 40}
	state.ArmorClass = []int{0, 50, 50}
	state.Damage = []combat.DamageDice{{}, {Count: 1, Sides: 8}, {Count: 1, Sides: 8}}
	state.Scores[1], state.Scores[2] = 5, 5
	state.Budgets[1], state.Budgets[2] = budget, budget
	state.Mover = 2
	return state
}

// 敵方就在旁邊時這一回合直接攻擊，而且回合會結束、換人行動。
func TestFoeTurnAttacksAnAdjacentPartyMember(t *testing.T) {
	state := newFoeTurnState(11, 10, 10, 10, 20)
	a := &app{roller: fixedRoller{20}, tactical: state}
	if err := a.foeTurn(state); err != nil {
		t.Fatal(err)
	}
	if state.HitPoints[1] == 10 {
		t.Fatalf("the party member was untouched: %s", state.FoeLog)
	}
	if state.Scores[2] != 0 {
		t.Fatalf("the foe kept its initiative score %d after acting", state.Scores[2])
	}
	if state.Mover != 1 {
		t.Fatalf("mover %d after the foe acted, want the party member", state.Mover)
	}
}

// 走不到就用完預算往目標靠，不會憑空攻擊。
func TestFoeTurnClosesTheDistanceWhenItCannotReach(t *testing.T) {
	state := newFoeTurnState(20, 10, 10, 10, 6)
	a := &app{roller: fixedRoller{20}, tactical: state}
	if err := a.foeTurn(state); err != nil {
		t.Fatal(err)
	}
	if state.HitPoints[1] != 10 {
		t.Fatalf("the foe attacked from out of reach: %s", state.FoeLog)
	}
	if state.Roster[2].X != 17 {
		t.Fatalf("the foe walked to x=%d on a budget of 6, want 17", state.Roster[2].X)
	}
	if state.Budgets[2] != 0 {
		t.Fatalf("the foe kept %d movement after closing", state.Budgets[2])
	}
}

// F5 開的預覽盤面沒有 ECL 遭遇，勝利也不能去續跑腳本。這個測試同樣刻意不給
// eventSession，走到續跑就會 panic。
func TestFinishCombatDoesNotRunAScriptForThePreviewBoard(t *testing.T) {
	a := &app{tacticalPreview: true, tactical: &tacticalState{}}
	if err := a.finishCombat(combat.CombatVictory); err != nil {
		t.Fatal(err)
	}
	if a.tacticalPreview || a.tactical != nil {
		t.Fatal("the tactical screen stayed open after the preview combat ended")
	}
}

// spec 063：隊伍的 THAC0 由職業查表得到，AC 與移動用建角寫下的基礎值。
// 1 級的每個職業 THAC0 都是 20，所以這個測試釘住的是來源，不是數字大小。
func TestPartyCombatStatsComeFromTheClassTable(t *testing.T) {
	for _, classID := range []string{"fighter", "cleric", "magic-user", "thief", "fighter-magic-user-thief"} {
		thac0, armor, movement, err := partyCombatStats(poolsave.Character{Name: "HERO", ClassID: classID})
		if err != nil {
			t.Fatalf("%s: %v", classID, err)
		}
		if 60-int(thac0) != 20 {
			t.Fatalf("%s THAC0 %d, want 20 at level 1", classID, 60-int(thac0))
		}
		if 60-armor != 10 {
			t.Fatalf("%s armour class %d, want 10", classID, 60-armor)
		}
		if movement != creationBaseMovement {
			t.Fatalf("%s movement %d, want %d", classID, movement, creationBaseMovement)
		}
	}
}

// 不認得的職業要失敗即關閉，不能默默當成單職業硬解。
func TestPartyCombatStatsRejectsAnUnknownClass(t *testing.T) {
	if _, _, _, err := partyCombatStats(poolsave.Character{Name: "HERO", ClassID: "bard"}); err == nil {
		t.Fatal("an unknown class was given combat stats")
	}
}

// 走進同伴那一格不會揮刀。原版的格位表沒有陣營，`ProbeDestination` 只回報
// 「那一格站著誰」，所以陣營判斷是呼叫端的責任；少了它，隊伍排成一列時最左邊
// 那個往右走就會砍死自己的同伴，而戰鬥永遠打不完。
func TestWalkingIntoAnAllyDoesNotAttack(t *testing.T) {
	state := newAttackState()
	state.Friendly[2] = true
	state.Roster[1].X, state.Roster[1].Y = 10, 10
	state.Roster[2].X, state.Roster[2].Y = 11, 10
	state.Grid = combat.TacticalGrid{IgnoreTerrain: true, Terrain: make([]uint8, 1250)}
	state.Budgets[1] = 24
	state.Mover = 1
	a := &app{roller: fixedRoller{20}, tactical: state, language: languageEnglish}

	east := -1
	for direction := 0; direction < 8; direction++ {
		x, y, err := combat.AdvanceTacticalCoordinate(10, 10, uint8(direction))
		if err == nil && x == 11 && y == 10 {
			east = direction
		}
	}
	if east < 0 {
		t.Fatal("no eastward direction")
	}
	before := state.HitPoints[2]
	a.keys = scriptedKeys{tacticalStepKeys[east]: true}
	if err := a.tacticalInput(); err != nil {
		t.Fatal(err)
	}
	if state.HitPoints[2] != before {
		t.Fatalf("an ally took %d damage", before-state.HitPoints[2])
	}
	if state.Roster[1].X != 10 || state.Roster[1].Y != 10 {
		t.Fatalf("the mover walked onto its ally at (%d,%d)", state.Roster[1].X, state.Roster[1].Y)
	}
}

// 對面的人照打。
func TestWalkingIntoAFoeStillAttacks(t *testing.T) {
	state := newAttackState()
	state.Roster[1].X, state.Roster[1].Y = 10, 10
	state.Roster[2].X, state.Roster[2].Y = 11, 10
	state.Grid = combat.TacticalGrid{IgnoreTerrain: true, Terrain: make([]uint8, 1250)}
	state.Budgets[1] = 24
	state.Mover = 1
	a := &app{roller: fixedRoller{20}, tactical: state, language: languageEnglish}
	east := -1
	for direction := 0; direction < 8; direction++ {
		x, y, err := combat.AdvanceTacticalCoordinate(10, 10, uint8(direction))
		if err == nil && x == 11 && y == 10 {
			east = direction
		}
	}
	before := state.HitPoints[2]
	a.keys = scriptedKeys{tacticalStepKeys[east]: true}
	if err := a.tacticalInput(); err != nil {
		t.Fatal(err)
	}
	if state.HitPoints[2] >= before {
		t.Fatalf("a foe took no damage (%d then %d)", before, state.HitPoints[2])
	}
}

// 戰術地圖上的 I 與 K 是移動鍵（方向 1 與方向 6，spec 053 的 H I M Q P O K G），
// 不是地圖上的「裝備」與「法術書」。少了這一條，隊伍往東北或西南走會開錯畫面，
// 那個角色的回合永遠結束不了——整場架就卡住。
func TestCombatMovementKeysAreNotMapShortcuts(t *testing.T) {
	for _, probe := range []struct {
		name string
		key  ebiten.Key
		open func(*app) bool
	}{
		{"裝備", ebiten.KeyI, func(a *app) bool { return a.equipmentOpen }},
		{"法術書", ebiten.KeyK, func(a *app) bool { return a.spellsOpen }},
		{"手札", ebiten.KeyJ, func(a *app) bool { return a.journalOpen }},
	} {
		state := newAttackState()
		state.Grid = combat.TacticalGrid{IgnoreTerrain: true, Terrain: make([]uint8, 1250)}
		state.Roster[1].X, state.Roster[1].Y = 10, 10
		state.Budgets[1] = 24
		state.Mover = 1
		application := &app{roller: fixedRoller{20}, tactical: state,
			language: languageEnglish, mode: modeAdventure, tacticalPreview: true}
		application.state.Party = []poolsave.Character{{Name: "HERO", ClassID: "fighter"}}
		if err := press(application, probe.key); err != nil {
			t.Fatalf("%s：%v", probe.name, err)
		}
		if probe.open(application) {
			t.Errorf("戰鬥中按 %v 開了%s", probe.key, probe.name)
		}
	}
}
