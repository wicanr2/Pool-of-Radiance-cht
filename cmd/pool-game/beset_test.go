package main

// 被圍攻的計數（runtime `+0Fh`／`+12h`，beset.go，spec 059）。一般出手從 `Update()` 送鍵。

import (
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// besetBoard：隊友（2）在 (8,5)，敵人（3）貼在它東邊 (9,5)、背對它（朝向就是 2 → 3 的方向）。
func besetBoard(t *testing.T, roll int) (*app, *tacticalState) {
	t.Helper()
	application, state := sideEffectBoard(t, int(gamepack.ClassSlotCleric), 1, gamepack.SpellIDBless, roll)
	state.Roster[3] = combat.CombatantCell{X: 9, Y: 5, FootprintClass: 1}
	state.ensureFacings()
	away, ok := combatFacingTowards(8, 5, 9, 5)
	if !ok {
		t.Fatal("fixture: no facing from (8,5) to (9,5)")
	}
	state.Facings[3] = away
	return application, state
}

// entry 14 每一次一般出手都讓目標計數加一；包裝只在計數不到三時讓目標轉身（`1891h`）。
func TestTheThirdAttackNoLongerTurnsTheTarget(t *testing.T) {
	application, state := besetBoard(t, 1) // d20 一律 1：全部落空，目標不會倒下
	facing := func() uint8 { return state.Facings[3] }
	towardsAttacker, _ := combatFacingTowards(9, 5, 8, 5)
	for attack := 1; attack <= 3; attack++ {
		state.Facings[3] = (towardsAttacker + 4) % 8
		// 下一個輪到的是施法者（1），不是目標——目標自己的行動開頭會清掉計數。
		state.Scores[1], state.Scores[3] = 5, 0
		attackWithKeys(t, application, state, 2, 3)
		if got := state.beset[3].count; got != uint8(attack) {
			t.Fatalf("attack %d: +0Fh %d", attack, got)
		}
		turned := facing() == towardsAttacker
		if turned != (attack < 3) {
			t.Fatalf("attack %d: target turned %v; want it to turn only while +0Fh < 3", attack, turned)
		}
	}
	// 目標自己的行動開頭（`01F2h`）清掉。
	state.Mover = 0
	state.Scores[3], state.Scores[1], state.Scores[2] = 5, 0, 0
	state.selectActor(application.rollDice)
	if state.Mover != 3 || state.beset[3].count != 0 {
		t.Fatalf("mover %d, +0Fh %d after selection; want 3 and 0", state.Mover, state.beset[3].count)
	}
}

// `15C4h..1600h`：目標已被打過兩次（`+0Fh` 2、`+12h` 1），第三下從正後方來——目標不再轉身，
// `+12h` 加 4 變 5 > 4，攻擊者到目標的方向等於目標朝向 → 比背後 AC。負對照：第二下（計數 2，
// 目標還會轉身）不算背後。
func TestTheThirdAttackFromBehindHitsTheRearArmourClass(t *testing.T) {
	for _, third := range []bool{true, false} {
		application, state := besetBoard(t, 10)
		front, back := splitArmourClasses(t, state.THAC0[2], 10)
		state.ArmorClass[3] = front
		state.setRearArmour(3, back)
		state.beset = map[uint8]besetMark{3: {count: 1, turned: 1}}
		if third {
			state.beset[3] = besetMark{count: 2, turned: 1}
		}
		hp := state.HitPoints[3]
		state.Scores[1], state.Scores[3] = 5, 0
		attackWithKeys(t, application, state, 2, 3)
		if hurt := state.HitPoints[3] < hp; hurt != third {
			t.Errorf("third attack %v: target HP %d → %d; want hurt only from behind", third, hp, state.HitPoints[3])
		}
	}
}

// splitArmourClasses 找一對 AC：命中骰 roll 打不中 front、打得中 back。
func splitArmourClasses(t *testing.T, thac0 uint8, roll uint8) (int, int) {
	t.Helper()
	front, back := -1, -1
	for armour := 0; armour <= 120 && (front < 0 || back < 0); armour++ {
		hit, err := combat.ResolveHit(roll, thac0, armour, 0)
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
		t.Fatalf("fixture: no armour class pair splits a roll of %d", roll)
	}
	return front, back
}

// `0B0Ah..0B1Eh`：背對著的敵人照樣反應——還沒行動（先攻 > 0），或行動過但之後沒被出手過。
// 行動過又被打過的（先攻 0、`+0Fh` 1）才看朝向窗（TestDisengagingFromAFoeFacingAwayIsFree）。
func TestAFoeFacingAwayStillReactsWhenTheBypassHolds(t *testing.T) {
	away := func(state *tacticalState) uint8 { return (facingTowardsMover(state) + 4) % 8 }
	for _, test := range []struct {
		name  string
		score uint8
		count uint8
	}{
		{"not acted yet", 5, 1},
		{"not attacked since", 0, 0},
	} {
		a, state := reactionBoard(t, away)
		state.Scores[2] = test.score
		state.beset = map[uint8]besetMark{2: {count: test.count}}
		if test.count == 0 {
			state.beset = nil
		}
		stepWith(t, a, reactionWest)
		if state.HitPoints[1] >= 10 {
			t.Errorf("%s: a foe facing away did not react (party HP %d)", test.name, state.HitPoints[1])
		}
	}
}
