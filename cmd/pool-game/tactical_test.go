package main

import (
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
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
		Roster:       make([]combat.CombatantCell, size),
		Friendly:     make([]bool, size),
		Dexterity:    make([]uint8, size),
		Scores:       make([]uint8, size),
		Budgets:      make([]uint8, size),
		BaseMovement: make([]uint8, size),
	}
	for index := 1; index < size; index++ {
		state.Dexterity[index] = 12
		state.BaseMovement[index] = 9
	}
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

// 打倒目標之後它的體型類別歸零，不再佔格也不再參與；敵方清空即為勝。
func TestResolveTacticalAttackDownsTheTargetAndEndsTheCombat(t *testing.T) {
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
	if state.Status != "VICTORY" {
		t.Fatalf("status %q, want VICTORY", state.Status)
	}
}

// 兩邊都還有人時戰鬥繼續。
func TestCombatOutcomeStaysOngoingWhileBothSidesStand(t *testing.T) {
	state := newAttackState()
	over, outcome := state.combatOutcome()
	if over || outcome != combat.CombatOngoing {
		t.Fatalf("over %v outcome %v, want an ongoing combat", over, outcome)
	}
}

func TestCombatOutcomeReportsDefeatWhenThePartyIsGone(t *testing.T) {
	state := newAttackState()
	state.Roster[1].FootprintClass = 0
	over, outcome := state.combatOutcome()
	if !over || outcome != combat.CombatDefeat {
		t.Fatalf("over %v outcome %v, want defeat", over, outcome)
	}
}

// Spec 046 契約 5：戰敗不得續跑戰後 ECL。這個測試刻意不給 eventSession——
// 一旦 defeat 走到續跑就會 panic，所以它同時證明那條路徑碰不到 session。
func TestFinishCombatDoesNotRunThePostCombatScriptOnDefeat(t *testing.T) {
	a := &app{combatActive: true, tacticalPreview: true, tactical: &tacticalState{}}
	if err := a.finishCombat(combat.CombatDefeat); err != nil {
		t.Fatal(err)
	}
	if !a.combatActive {
		t.Fatal("defeat cleared the encounter; the post-combat script must not be reached")
	}
	if a.tacticalPreview || a.tactical != nil {
		t.Fatal("the tactical screen stayed open after defeat")
	}
}
