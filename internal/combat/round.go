package combat

import "fmt"

// SideCounts 是 DS:6772h 與 DS:6773h 兩個存活數（spec 062）。
// 戰鬥迴圈以「兩邊都還有人」為繼續條件。
type SideCounts struct {
	Party uint8 // DS:6772h
	Foes  uint8 // DS:6773h
}

// 倒地狀態的兩個編號與計數上限，取自 overlay-08 `0868h` 的回合收尾。
const (
	DyingState      uint8 = 5
	DeadState       uint8 = 6
	DyingRoundLimit uint8 = 9
)

// AdvanceDyingCounter 重現回合收尾對倒地者的處理：狀態為 DyingState 的
// combatant 每回合把計數加一，超過 DyingRoundLimit 就轉成 DeadState。
// 其他狀態不動。計數以位元組遞增，與原版一致。
func AdvanceDyingCounter(state, counter uint8) (uint8, uint8) {
	if state != DyingState {
		return state, counter
	}
	counter++
	if counter > DyingRoundLimit {
		return DeadState, counter
	}
	return state, counter
}

// RoundEndsCombat 重現 overlay-08 `0868h` 結尾的兩段判斷。
//
// 任一邊歸零就結束。但「我方還有人、敵方歸零、且 DS:4955h 為 0」時，原版會
// 多問一次；玩家答 Y 就把結束旗標改回 0，戰鬥繼續——所以敵人清光不必然
// 立刻結束戰鬥。
func RoundEndsCombat(counts SideCounts, lingerFlag uint8, continueRequested bool) bool {
	over := counts.Party == 0 || counts.Foes == 0
	if counts.Party > 0 && counts.Foes == 0 && lingerFlag == 0 && continueRequested {
		over = false
	}
	return over
}

// CombatOutcome 是戰鬥結束時的結果。
type CombatOutcome int

const (
	CombatOngoing CombatOutcome = iota
	CombatVictory
	CombatDefeat
)

// ResolveCombatOutcome 由兩個存活數判定結果。兩邊同時歸零時原版沒有額外分支，
// 這裡照它的判斷順序先看我方——我方歸零就是敗。
func ResolveCombatOutcome(counts SideCounts) CombatOutcome {
	switch {
	case counts.Party == 0:
		return CombatDefeat
	case counts.Foes == 0:
		return CombatVictory
	default:
		return CombatOngoing
	}
}

// RoundSteps 是一個回合裡原版依序做的事，由呼叫端提供實作。
// 拆成介面是為了讓順序本身可以被測試釘住——順序錯了，行為就跟原版不同。
type RoundSteps struct {
	// StartRound 對應 overlay-25 entry 31，每回合開頭跑一次。
	StartRound func() error
	// InitCombatants 對應沿 DS:5CF4h 串列逐一呼叫 overlay-13 entry 1
	//（移動預算初始化，spec 053）。
	InitCombatants func() error
	// ClearRoundState 對應把 `[DS:4937h + 596h]` 清零。
	ClearRoundState func() error
	// SelectActor 對應 overlay-08 entry 2 的先攻選擇；沒有人可行動時回 false。
	SelectActor func() (int, bool, error)
	// TakeTurn 對應 overlay-08 entry 3。
	TakeTurn func(actor int) error
	// EndRound 對應 overlay-08 entry 6，回報戰鬥是否結束。
	EndRound func() (bool, error)
}

// RunCombatRounds 重現 overlay-08 entry 1（`0071h`）的迴圈骨架：
// 先判斷是否已經結束，再跑一個完整回合，回合收尾決定要不要繼續。
//
// initiallyOver 對應原版進迴圈前先做的那一次存活數檢查——戰鬥有可能一回合
// 都沒跑就結束。
func RunCombatRounds(initiallyOver bool, steps RoundSteps) (int, error) {
	if steps.SelectActor == nil || steps.TakeTurn == nil || steps.EndRound == nil {
		return 0, fmt.Errorf("Pool combat rounds need SelectActor, TakeTurn and EndRound")
	}
	rounds := 0
	over := initiallyOver
	for !over {
		if steps.StartRound != nil {
			if err := steps.StartRound(); err != nil {
				return rounds, err
			}
		}
		if steps.InitCombatants != nil {
			if err := steps.InitCombatants(); err != nil {
				return rounds, err
			}
		}
		if steps.ClearRoundState != nil {
			if err := steps.ClearRoundState(); err != nil {
				return rounds, err
			}
		}
		for {
			actor, ok, err := steps.SelectActor()
			if err != nil {
				return rounds, err
			}
			if !ok {
				break
			}
			if err := steps.TakeTurn(actor); err != nil {
				return rounds, err
			}
		}
		finished, err := steps.EndRound()
		if err != nil {
			return rounds, err
		}
		rounds++
		over = finished
	}
	return rounds, nil
}
