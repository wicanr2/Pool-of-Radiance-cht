package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// 士氣逃跑那幾處的原版細節（spec 096〈士氣崩了之後〉，#83）。全部從 Update() 送 ENTER，
// 輪到敵方時 tacticalInput 自己分派 foeTurn。

// overlay-09 entry 5：每一輪先問搆不搆得到（`0D51h` 010Ah:00C0h），搆得到就打；
// 搆不到才叫 `07E8h`，腳程不夠（`084Dh` 的 `+6 ÷ 2 <= 0`）是在那裡面收工的。
// 所以腳程只剩 1 點、身邊有人時照樣打；不在身邊就不動。
func TestFoeWithNoStepLeftStillAttacksItsNeighbour(t *testing.T) {
	for _, tc := range []struct {
		name    string
		foeX    uint8
		attacks int
	}{
		{"adjacent", 6, 1},
		{"two cells away", 7, 0},
	} {
		application, state := newFoeCastApp(t)
		state.Roster[2].X = tc.foeX
		state.Budgets[2] = 1
		before := state.Roster[2]
		if err := press(application, ebiten.KeyEnter); err != nil {
			t.Fatal(err)
		}
		if state.Activity.FoeAttacks != tc.attacks {
			t.Fatalf("%s: foe attacks %d, want %d (log %q)", tc.name, state.Activity.FoeAttacks,
				tc.attacks, state.FoeLog)
		}
		if tc.attacks == 0 && state.Roster[2] != before {
			t.Fatalf("%s: a foe with one point of movement stepped: %+v → %+v", tc.name, before, state.Roster[2])
		}
	}
}

// overlay-13 entry 7 比的腳程是 `0123h`，而 `0123h` 在 `0150h..0182h` 派發群組 12h：
// 對面最快的那一個被緩速（`2Ah`，腳程減半）時，一樣快的局面變成自己比較快，
// 不擲那一顆 d2 就逃掉。與 TestEscapeTieRollsOneD2 同一盤，tie 那一格排的是 2（擋住）。
func TestSlowedOpponentCannotBlockTheEscape(t *testing.T) {
	roller := &sequenceRoller{values: []int{1, 1, 1, 1, 1, 1, 1, 2, 1, 1, 1}}
	application, state := newFleeApp(t, 1, 1, 0, 0, roller)
	state.Undead.Turned = map[int]bool{2: true}
	state.Effects[1] = state.Effects[1].Append(gamepack.NewEffectNode(gamepack.SlowEffectCode, 5, 1, false))
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if got := countSides(roller.asked, 2); got != 2 {
		t.Fatalf("d2 asked %d times, want only the two flee modes: %v", got, roller.asked)
	}
	if state.Roster[2].FootprintClass != 0 {
		t.Fatalf("the foe did not get away from a slowed party: log %q", state.FoeLog)
	}
}
