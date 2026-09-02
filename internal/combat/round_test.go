package combat

import (
	"reflect"
	"testing"
)

func TestAdvanceDyingCounterTurnsIntoDeathAfterTenRounds(t *testing.T) {
	state, counter := DyingState, uint8(0)
	for round := 1; round <= int(DyingRoundLimit); round++ {
		state, counter = AdvanceDyingCounter(state, counter)
		if state != DyingState {
			t.Fatalf("round %d turned the state into %d too early", round, state)
		}
	}
	state, counter = AdvanceDyingCounter(state, counter)
	if state != DeadState {
		t.Fatalf("state %d after the limit, want %d", state, DeadState)
	}
	if counter != DyingRoundLimit+1 {
		t.Fatalf("counter %d", counter)
	}
}

func TestAdvanceDyingCounterLeavesOtherStatesAlone(t *testing.T) {
	state, counter := AdvanceDyingCounter(3, 7)
	if state != 3 || counter != 7 {
		t.Fatalf("state %d counter %d changed", state, counter)
	}
}

func TestRoundEndsCombatWhenEitherSideIsGone(t *testing.T) {
	if !RoundEndsCombat(SideCounts{Party: 0, Foes: 4}, 0, false) {
		t.Fatal("a wiped party did not end the combat")
	}
	if !RoundEndsCombat(SideCounts{Party: 3, Foes: 0}, 0, false) {
		t.Fatal("no foes left did not end the combat")
	}
	if RoundEndsCombat(SideCounts{Party: 3, Foes: 4}, 0, false) {
		t.Fatal("both sides alive ended the combat")
	}
}

// 敵人清光時原版會多問一次；答 Y 就繼續打。
func TestRoundEndsCombatHonoursTheContinuePrompt(t *testing.T) {
	if RoundEndsCombat(SideCounts{Party: 3, Foes: 0}, 0, true) {
		t.Fatal("answering yes did not keep the combat going")
	}
	// 旗標非 0 時原版根本不問，所以回答無效。
	if !RoundEndsCombat(SideCounts{Party: 3, Foes: 0}, 1, true) {
		t.Fatal("the prompt ran although the linger flag was set")
	}
	// 我方全滅那一支沒有提示。
	if !RoundEndsCombat(SideCounts{Party: 0, Foes: 0}, 0, true) {
		t.Fatal("a wiped party was kept in combat by the prompt")
	}
}

func TestResolveCombatOutcome(t *testing.T) {
	if got := ResolveCombatOutcome(SideCounts{Party: 0, Foes: 2}); got != CombatDefeat {
		t.Fatalf("outcome %v, want defeat", got)
	}
	if got := ResolveCombatOutcome(SideCounts{Party: 2, Foes: 0}); got != CombatVictory {
		t.Fatalf("outcome %v, want victory", got)
	}
	if got := ResolveCombatOutcome(SideCounts{Party: 2, Foes: 2}); got != CombatOngoing {
		t.Fatalf("outcome %v, want ongoing", got)
	}
	if got := ResolveCombatOutcome(SideCounts{}); got != CombatDefeat {
		t.Fatalf("outcome %v when both sides are gone, want defeat", got)
	}
}

// 回合內的順序就是行為的一部分，用逐步紀錄釘住。
func TestRunCombatRoundsFollowsTheOriginalOrder(t *testing.T) {
	var log []string
	actors := []int{7, 3}
	next := 0
	rounds, err := RunCombatRounds(false, RoundSteps{
		StartRound:      func() error { log = append(log, "start"); return nil },
		InitCombatants:  func() error { log = append(log, "init"); return nil },
		ClearRoundState: func() error { log = append(log, "clear"); return nil },
		SelectActor: func() (int, bool, error) {
			if next >= len(actors) {
				return 0, false, nil
			}
			actor := actors[next]
			next++
			return actor, true, nil
		},
		TakeTurn: func(actor int) error {
			log = append(log, "turn")
			return nil
		},
		EndRound: func() (bool, error) { log = append(log, "end"); return true, nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	if rounds != 1 {
		t.Fatalf("ran %d rounds, want 1", rounds)
	}
	want := []string{"start", "init", "clear", "turn", "turn", "end"}
	if !reflect.DeepEqual(log, want) {
		t.Fatalf("order %v, want %v", log, want)
	}
}

// 進迴圈前就已經結束的戰鬥一個回合都不跑。
func TestRunCombatRoundsRespectsTheInitialCheck(t *testing.T) {
	rounds, err := RunCombatRounds(true, RoundSteps{
		SelectActor: func() (int, bool, error) { t.Fatal("a round ran"); return 0, false, nil },
		TakeTurn:    func(int) error { return nil },
		EndRound:    func() (bool, error) { return true, nil },
	})
	if err != nil || rounds != 0 {
		t.Fatalf("rounds %d err %v", rounds, err)
	}
}

func TestRunCombatRoundsKeepsGoingUntilTheEndRoundSaysStop(t *testing.T) {
	remaining := 3
	rounds, err := RunCombatRounds(false, RoundSteps{
		SelectActor: func() (int, bool, error) { return 0, false, nil },
		TakeTurn:    func(int) error { return nil },
		EndRound: func() (bool, error) {
			remaining--
			return remaining == 0, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if rounds != 3 {
		t.Fatalf("ran %d rounds, want 3", rounds)
	}
}

func TestRunCombatRoundsRequiresTheCoreSteps(t *testing.T) {
	if _, err := RunCombatRounds(false, RoundSteps{}); err == nil {
		t.Fatal("missing steps were accepted")
	}
}
