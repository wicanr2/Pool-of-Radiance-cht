package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
)

// 方向照原版 0..7（spec 053）：東 2、西 6。
const (
	reactionEast = 2
	reactionWest = 6
)

// reactionBoard：隊員（1）在 (10,10)，敵人（2）在東邊 (11,10)，盤面全開。
// 敵人的朝向由 foeFacing 決定；隊員輪到行動、已經按過 M。
func reactionBoard(t *testing.T, foeFacing func(state *tacticalState) uint8) (*app, *tacticalState) {
	t.Helper()
	state := newFoeTurnState(11, 10, 10, 10, 24)
	state.Mover = 1
	state.Moving = true
	state.ensureFacings()
	state.Facings[2] = foeFacing(state)
	return &app{roller: fixedRoller{20}, tactical: state, language: languageEnglish}, state
}

func facingTowardsMover(state *tacticalState) uint8 {
	facing, _ := combatFacingTowards(state.Roster[2].X, state.Roster[2].Y, state.Roster[1].X, state.Roster[1].Y)
	return facing
}

func stepWith(t *testing.T, a *app, direction int) {
	t.Helper()
	a.keys = scriptedKeys{tacticalStepKeys[direction]: true}
	if err := a.tacticalInput(); err != nil {
		t.Fatal(err)
	}
}

// 往西走開：脫離了面向自己的敵人，敵人打一下，這一步照走（spec 059，#58）。
func TestDisengagingFromAFacingFoeDrawsAReactionAttack(t *testing.T) {
	a, state := reactionBoard(t, facingTowardsMover)
	stepWith(t, a, reactionWest)
	if state.HitPoints[1] >= 10 {
		t.Fatalf("the foe did not react: party HP %d", state.HitPoints[1])
	}
	if state.HitPoints[1] > 0 && (state.Roster[1].X != 9 || state.Roster[1].Y != 10) {
		t.Fatalf("the step was not committed: mover at (%d,%d)", state.Roster[1].X, state.Roster[1].Y)
	}
	// 反應攻擊以第三個引數 1 呼叫攻擊包裝，`1898h` 跳過目標轉身（spec 059）：走的人還是朝西。
	if state.Facings[1] != reactionWest {
		t.Fatalf("mover facing %d after the reaction, want %d (a reaction does not turn it)", state.Facings[1], reactionWest)
	}
}

// 還沒輪到的反應者（先攻 5）一回合兩下（編碼 4）：反應攻擊把兩下都打完，次數扣光，entry 34
// 清掉它的先攻——走的人按 Enter 結束之後，這一回合輪不到它（spec 059）。
func TestAReactionSpendsTheReactorsTurn(t *testing.T) {
	a, state := reactionBoard(t, facingTowardsMover)
	state.HitPoints[1] = 100
	state.AttackRates[2] = [2]uint8{4, 0}
	state.Casting.Pending = map[int]uint8{2: 1}
	stepWith(t, a, reactionWest)
	if state.Activity.FoeHits != 2 || state.HitPoints[1] != 100-16 {
		t.Fatalf("reaction: %d hits, mover HP %d; want both swings (84)", state.Activity.FoeHits, state.HitPoints[1])
	}
	if state.Scores[2] != 0 || state.swingsLeft[2] != [2]uint8{} || state.Budgets[2] != 0 {
		t.Fatalf("reactor: score %d, left %v, budget %d; want entry 34 to clear it", state.Scores[2], state.swingsLeft[2], state.Budgets[2])
	}
	if _, ok := state.Casting.Pending[2]; ok {
		t.Fatal("entry 34 left the reactor's pending spell")
	}
	round := state.Round
	a.keys = scriptedKeys{ebiten.KeyEnter: true}
	if err := a.tacticalInput(); err != nil {
		t.Fatal(err)
	}
	if state.Round == round || state.Activity.FoeAttacks != 1 {
		t.Fatalf("after Enter: round %d → %d, %d foe attacks; want a new round with no further attack",
			round, state.Round, state.Activity.FoeAttacks)
	}
}

// 反應攻擊第一下就打倒了走的人：剩下那一下留著（`+113h` 1），先攻不清，輪到它時照剩下的打。
func TestAReactionThatDownsTheMoverKeepsTheRest(t *testing.T) {
	a, state := reactionBoard(t, facingTowardsMover)
	state.HitPoints[1] = 1
	state.AttackRates[2] = [2]uint8{4, 0}
	stepWith(t, a, reactionWest)
	if state.HitPoints[1] > 0 || state.Activity.FoeHits != 1 {
		t.Fatalf("mover HP %d after %d hits; want it down after one", state.HitPoints[1], state.Activity.FoeHits)
	}
	if state.swingsLeft[2] != [2]uint8{1, 0} || state.Scores[2] != 5 {
		t.Fatalf("reactor: left %v, score %d; want [1 0] and its initiative kept", state.swingsLeft[2], state.Scores[2])
	}
}

// 反應攻擊比的是目標的背後 AC（`+112h`，`1604h..161Ch`）：正面擋得住、背後擋不住的那一骰會中。
func TestAReactionHitsTheRearArmourClass(t *testing.T) {
	for _, rear := range []bool{true, false} {
		a, state := reactionBoard(t, facingTowardsMover)
		a.roller = fixedRoller{10}
		front, back := -1, -1
		for armour := 0; armour <= 120 && (front < 0 || back < 0); armour++ {
			hit, err := combat.ResolveHit(10, state.THAC0[2], armour, 0)
			if err != nil {
				t.Fatal(err)
			}
			if hit && back < 0 {
				back = armour
			}
			if !hit && front < 0 && back >= 0 {
				front = armour
			}
		}
		if front < 0 || back < 0 {
			t.Fatal("fixture: no armour class pair splits a roll of 10")
		}
		state.ArmorClass[1] = front
		if rear {
			state.setRearArmour(1, back)
		}
		stepWith(t, a, reactionWest)
		if hurt := state.HitPoints[1] < 10; hurt != rear {
			t.Errorf("rear AC %v: mover HP %d; want hurt only against the rear AC", rear, state.HitPoints[1])
		}
	}
}

// 敵人背對著：朝向窗（前後兩格）收不進隊員，不打。
func TestDisengagingFromAFoeFacingAwayIsFree(t *testing.T) {
	away := func(state *tacticalState) uint8 { return (facingTowardsMover(state) + 4) % 8 }
	a, state := reactionBoard(t, away)
	for _, facing := range combat.ReactionFacings(state.Facings[2]) {
		inside, err := combat.FacingArcContains(11, 10, 10, 10, facing)
		if err != nil {
			t.Fatal(err)
		}
		if inside {
			t.Fatalf("fixture: facing window of %d still contains the mover (facing %d)", state.Facings[2], facing)
		}
	}
	stepWith(t, a, reactionWest)
	if state.HitPoints[1] != 10 {
		t.Fatalf("a foe facing away reacted: party HP %d", state.HitPoints[1])
	}
	if state.Roster[1].X != 9 {
		t.Fatalf("the step was not committed: x=%d", state.Roster[1].X)
	}
}

// 往北走一格仍然貼著敵人（斜角）：不是脫離，不打。
func TestSteppingAlongAFoeIsNotDisengaging(t *testing.T) {
	a, state := reactionBoard(t, facingTowardsMover)
	stepWith(t, a, 0)
	if state.HitPoints[1] != 10 {
		t.Fatalf("staying adjacent drew an attack: party HP %d", state.HitPoints[1])
	}
}

// 否決代碼 47h 在被打的人身上（`1087h`）：不打。
func TestReactionAttackRespectsTheVetoEffect(t *testing.T) {
	a, state := reactionBoard(t, facingTowardsMover)
	state.addEffect(1, vetoEffectAlways, 10, 1)
	stepWith(t, a, reactionWest)
	if state.HitPoints[1] != 10 {
		t.Fatalf("a vetoed reaction still hit: party HP %d", state.HitPoints[1])
	}
}

// 致能效果（`DS:2880h` 的四個代碼）在敵人身上：不打。
func TestADisabledFoeDoesNotReact(t *testing.T) {
	a, state := reactionBoard(t, facingTowardsMover)
	state.addEffect(2, combat.DisablingEffectCodes[0], 10, 1)
	stepWith(t, a, reactionWest)
	if state.HitPoints[1] != 10 {
		t.Fatalf("a disabled foe reacted: party HP %d", state.HitPoints[1])
	}
}

// 反應攻擊打倒了隊員：這一步不走，回合結束（overlay-08 `0C85h`）。
func TestAMoverDownedByTheReactionDoesNotStep(t *testing.T) {
	a, state := reactionBoard(t, facingTowardsMover)
	state.HitPoints[1] = 1
	stepWith(t, a, reactionWest)
	if state.HitPoints[1] > 0 {
		t.Fatalf("fixture: the reaction did not down the mover (HP %d)", state.HitPoints[1])
	}
	if state.Roster[1].X != 10 {
		t.Fatalf("a downed mover still stepped to x=%d", state.Roster[1].X)
	}
	if state.Mover == 1 {
		t.Fatal("the downed mover kept the turn")
	}
}

// 怪物這一側走同一支（overlay-09 `0ACBh` 呼叫同一個 entry 6）：敵人往東走開，
// 面向牠的隊員打一下。怪物 AI 搆得到人就先打，鄰接的敵人不會自己走開，
// 所以這裡直接叫 `disengageReactions`，驗的是閘門對兩邊一樣；`foeTurn` 裡
// 那個呼叫點與玩家那一側同形。
func TestAFoeWalkingAwayDrawsAReactionFromTheParty(t *testing.T) {
	state := newFoeTurnState(11, 10, 10, 10, 24)
	state.ensureFacings()
	facing, _ := combatFacingTowards(10, 10, 11, 10)
	state.Facings[1] = facing
	a := &app{roller: fixedRoller{20}, tactical: state}
	down, err := a.disengageReactions(state, 2, reactionEast)
	if err != nil {
		t.Fatal(err)
	}
	if state.HitPoints[2] >= 10 {
		t.Fatalf("the party member did not react to the foe walking away: foe HP %d", state.HitPoints[2])
	}
	if down != (state.HitPoints[2] <= 0) {
		t.Fatalf("down=%t with foe HP %d", down, state.HitPoints[2])
	}
}
