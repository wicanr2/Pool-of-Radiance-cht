package main

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/golden-box-remake-engine/eclvm"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 士氣與逃跑（overlay-09 entry 8／entry 5 `0B9Fh`／`07E8h` 的逃跑分支、overlay-13
// entry 7，spec 096，issue #74）。全部從 Update() 送 ENTER：輪到敵方時 tacticalInput
// 自己分派 foeTurn。
//
// 盤面是 newFoeCastApp 那一張：隊員（1）在 (5,5)、敵人（2）在 (10,5)，兩邊腳程 9。

// newFleeApp 把敵人擺到 (x,y)，隊伍朝向 facing（地圖上的 0..3），敵人記錄的 `+84h` 是 raw。
func newFleeApp(t *testing.T, x, y, facing, raw uint8, roller interface{ Roll(int, int) int }) (*app, *tacticalState) {
	t.Helper()
	application, state := newFoeCastApp(t)
	application.roller = roller
	application.spawn.Facing = facing
	state.Roster[2].X, state.Roster[2].Y = x, y
	state.rememberMorale(2, raw, 10)
	return application, state
}

// countSides 數 asked 裡有幾顆是 sides 面。
func countSides(asked []int, sides int) int {
	count := 0
	for _, value := range asked {
		if value == sides {
			count++
		}
	}
	return count
}

// 被轉變的不死生物（runtime `+10h`）這一回合逃跑：印 `is forced to flee`（`1116h`），
// 每一次 `07E8h` 擲一顆 d2 當模式，朝隊伍朝向算出的基準方向走——朝向 0 是西北（7）；
// 從 (1,1) 走一步到 (0,0)，再一步就踏出盤面，對面比牠慢，逃掉（`Got Away`）。
func TestTurnedUndeadFleesOffTheBoard(t *testing.T) {
	roller := &sequenceRoller{values: []int{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}}
	application, state := newFleeApp(t, 1, 1, 0, 0, roller)
	state.Undead.Turned = map[int]bool{2: true}
	state.BaseMovement[1] = 6 // 隊員比牠慢：對面最快 6 < 自己 9
	hp := state.HitPoints[2]
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if state.Roster[2].FootprintClass != 0 || state.States[2] != gamepack.FledState {
		t.Fatalf("the turned undead is still on the board: %+v state %d, log %q",
			state.Roster[2], state.States[2], state.FoeLog)
	}
	if state.HitPoints[2] != hp {
		t.Fatalf("got away keeps its hit points (0F73h): %d → %d", hp, state.HitPoints[2])
	}
	if !strings.Contains(state.FoeLog, "GOT AWAY") {
		t.Fatalf("log %q, want GOT AWAY", state.FoeLog)
	}
	// 兩次 07E8h：第一次走到 (0,0)，第二次踏出盤面；相同快慢才擲的那顆 d2 不擲。
	if got := countSides(roller.asked, 2); got != 2 {
		t.Fatalf("d2 asked %d times, want one per 07E8h call (2): %v", got, roller.asked)
	}
	if !askedRun(roller.asked, 7, 7, 2, 2) {
		t.Fatalf("dice %v, want the d7 pair then the two flee-mode d2", roller.asked)
	}
	if state.Scores[2] != 0 {
		t.Fatal("leaving the board did not end the turn (entry 34)")
	}
	if counts := state.sideCounts(); counts.Foes != 0 {
		t.Fatalf("the fled foe still counts toward the combat: %+v", counts)
	}
}

// 士氣兩關都沒過（`+84h = FFh`：第一關士氣 0；第二關敵方整體只剩一成，比不過
// 100 − 70），對面沒有比牠快 → 逃。朝向 1 的基準方向是東（2），腳程 20 走十步，
// 每一步一顆 d2；腳程用完（`084Dh`）交給 entry 6，沒有打人。
func TestMoraleFailureFleesAwayFromTheParty(t *testing.T) {
	roller := &sequenceRoller{values: []int{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}}
	application, state := newFleeApp(t, 10, 5, 1, 0xFF, roller)
	state.HitPoints[2] = 3 // 3 × 20 ÷ 30 × 5 = 10
	state.refreshSideMorale()
	state.Morale.Party = 70
	if state.Morale.Side != 10 {
		t.Fatalf("DS:6D22h = %d, want 10", state.Morale.Side)
	}
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if state.Roster[2].X != 20 || state.Roster[2].Y != 5 {
		t.Fatalf("the fleeing foe ended at (%d,%d), want (20,5) running east; log %q",
			state.Roster[2].X, state.Roster[2].Y, state.FoeLog)
	}
	if state.HitPoints[1] != 40 || state.Roster[2].FootprintClass == 0 {
		t.Fatalf("a fleeing foe should neither attack nor leave here: hp %d, %+v", state.HitPoints[1], state.Roster[2])
	}
	if got := countSides(roller.asked, 2); got != 10 {
		t.Fatalf("d2 asked %d times, want one per step (10): %v", got, roller.asked)
	}
	// 腳程剛好用完時 `0B9Fh` 的條件（`+6 > 0`）不成立，照原版回到接近迴圈，
	// 那裡的 `07E8h` 再交給 entry 6——一樣是這一回合結束、不再擲骰。
	if state.Activity.FoeSteps != 10 || state.Scores[2] != 0 {
		t.Fatalf("steps %d score %d, want the turn to end after ten steps (log %q)",
			state.Activity.FoeSteps, state.Scores[2], state.FoeLog)
	}
}

// 同一個盤面，第二關過得了（整體生命九成五，95 >= 30）就照常走向隊伍。
func TestMoraleHoldsAndTheFoeCloses(t *testing.T) {
	roller := &sequenceRoller{values: []int{1, 1, 1, 1, 1, 1, 1, 1, 1, 1}}
	application, state := newFleeApp(t, 10, 5, 1, 0xFF, roller)
	state.HitPoints[2] = 29
	state.refreshSideMorale()
	state.Morale.Party = 70
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if state.Roster[2].X >= 10 {
		t.Fatalf("a steady foe should close on the party: at (%d,%d), log %q",
			state.Roster[2].X, state.Roster[2].Y, state.FoeLog)
	}
	if countSides(roller.asked, 2) != 0 {
		t.Fatalf("a steady foe rolled a flee-mode d2: %v", roller.asked)
	}
}

// 士氣崩了、對面比牠快：智力大於 5 就投降（`1259h`）——收掉、`+10Ch = 4`、生命值 0，
// 回合結束；智力不到就什麼都不做，照常行動。
func TestMoraleFailureWithFasterFoesSurrendersOrStands(t *testing.T) {
	for _, tc := range []struct {
		name         string
		intelligence uint8
		surrenders   bool
	}{
		{"smart", 10, true},
		{"dim", 5, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			roller := &sequenceRoller{values: []int{1, 1, 1, 1, 1, 1, 1, 1, 1}}
			application, state := newFleeApp(t, 10, 5, 1, 0xFF, roller)
			state.rememberMorale(2, 0xFF, tc.intelligence)
			state.BaseMovement[1] = 12 // 隊員比牠快
			state.HitPoints[2] = 3
			state.refreshSideMorale()
			state.Morale.Party = 70
			if err := press(application, ebiten.KeyEnter); err != nil {
				t.Fatal(err)
			}
			gone := state.Roster[2].FootprintClass == 0
			if gone != tc.surrenders {
				t.Fatalf("surrendered %v, want %v: log %q", gone, tc.surrenders, state.FoeLog)
			}
			if tc.surrenders {
				if state.States[2] != gamepack.SurrenderedState || state.HitPoints[2] != 0 ||
					!strings.Contains(state.FoeLog, "SURRENDERS") || state.Scores[2] != 0 {
					t.Fatalf("surrender: state %d hp %d log %q score %d",
						state.States[2], state.HitPoints[2], state.FoeLog, state.Scores[2])
				}
				if countSides(roller.asked, 7) != 0 {
					t.Fatalf("a surrender ends the turn before entry 3 rolls its d7: %v", roller.asked)
				}
				return
			}
			if state.Roster[2].X >= 10 || countSides(roller.asked, 2) != 0 {
				t.Fatalf("a cornered foe should act normally: at (%d,%d), dice %v",
					state.Roster[2].X, state.Roster[2].Y, roller.asked)
			}
		})
	}
}

// 一樣快的時候 overlay-13 entry 7 擲一顆 d2：擲 2 逃不掉（`Escape is blocked`），
// 那一隻留在原地、這一回合結束，下一回合重新判定士氣。
func TestEscapeTieRollsOneD2(t *testing.T) {
	for _, tc := range []struct {
		roll    int
		escaped bool
	}{{1, true}, {2, false}} {
		// 模式骰與 d7 都回 1；兩次 07E8h 的 d2 回 1；相同快慢的那一顆回 tc.roll。
		roller := &sequenceRoller{values: []int{1, 1, 1, 1, 1, 1, 1, tc.roll, 1, 1, 1}}
		application, state := newFleeApp(t, 1, 1, 0, 0, roller)
		state.Undead.Turned = map[int]bool{2: true}
		if err := press(application, ebiten.KeyEnter); err != nil {
			t.Fatal(err)
		}
		if got := countSides(roller.asked, 2); got != 3 {
			t.Fatalf("roll %d: d2 asked %d times, want 2 flee modes + 1 tie: %v", tc.roll, got, roller.asked)
		}
		gone := state.Roster[2].FootprintClass == 0
		if gone != tc.escaped {
			t.Fatalf("roll %d: escaped %v, want %v (log %q)", tc.roll, gone, tc.escaped, state.FoeLog)
		}
		if !tc.escaped && (!strings.Contains(state.FoeLog, "ESCAPE IS BLOCKED") || state.Scores[2] != 0 ||
			state.Roster[2].X != 0 || state.Roster[2].Y != 0) {
			t.Fatalf("blocked: log %q score %d at (%d,%d)", state.FoeLog, state.Scores[2],
				state.Roster[2].X, state.Roster[2].Y)
		}
	}
}

// 逃掉的（`+10Ch == 3`）戰後不算經驗值；投降的（4）照算（overlay-05 entry 2 `0079h`）。
func TestFledFoesGiveNoExperience(t *testing.T) {
	var orc gamepack.MonsterRecord
	orc.Raw[0xB8] = 10
	orc.Raw[0xBA] = 1
	orc.Raw[0x32] = 5
	application := &app{combatMonsters: []stagedMonster{{Spawn: eclvm.MonsterSpawn{Count: 3}, Record: orc}}}
	application.state = poolsave.State{Party: []poolsave.Character{
		{Name: "A", ClassID: "fighter", Abilities: [6]int{10, 10, 10, 10, 10, 10}},
	}}
	state := newRoundState(3)
	state.Friendly = []bool{false, true, false, false}
	state.leaveBoard(2, gamepack.FledState)
	state.leaveBoard(3, gamepack.SurrenderedState)
	fled := application.fledFoeRecords(state)
	if len(fled) != 1 {
		t.Fatalf("fled records %d, want only the one that got away", len(fled))
	}
	application.awardCombatExperienceExcept(fled)
	// 三隻各 15，逃掉一隻 → 30。
	if got := application.state.Party[0].Experience; got != 30 {
		t.Fatalf("experience %d, want 30 (the fled ORC is not counted)", got)
	}
}

// DS:6D22h 在戰鬥佈置時算一次、每個回合收尾重算（overlay-10 `2030h`、overlay-08
// `087Dh`），回合中間不變；隊伍 `+58Ch` 佈置時夾到 100 並寫回 ECL 記憶體。
func TestSideMoraleIsRefreshedAtRoundEnd(t *testing.T) {
	application, state := newFoeCastApp(t)
	application.eventMachine = &eclvm.Machine{Memory: map[uint16]uint16{partyMoraleAddress: 250}}
	application.setupMorale(state)
	if state.Morale.Party != 100 || application.eventMachine.Memory[partyMoraleAddress] != 100 {
		t.Fatalf("+58Ch = %d (memory %d), want clamped to 100", state.Morale.Party,
			application.eventMachine.Memory[partyMoraleAddress])
	}
	if state.Morale.Side != 100 {
		t.Fatalf("DS:6D22h at setup %d, want 100", state.Morale.Side)
	}
	state.HitPoints[2] = 15
	if state.Morale.Side != 100 {
		t.Fatal("DS:6D22h changed in the middle of a round")
	}
	state.endRound(fixedRoller{1}.Roll)
	if state.Morale.Side != 50 {
		t.Fatalf("DS:6D22h after the round %d, want 50", state.Morale.Side)
	}
}
