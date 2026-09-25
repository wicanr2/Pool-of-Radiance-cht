package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// AI 的轉變不死生物（overlay-09 entry 2 `0203h`，spec 096／111，issue #71）。
// 全部從 Update() 送鍵：交給電腦的隊員輪到時 tacticalInput 自己分派，這裡按 ENTER。

// newQuickTurnApp 是一場一對一：Q）UICK 過的隊員（1）在 (5,5)、一隻不死生物（2）
// 在 (10,5)。column 是牠的 `+76h`，clericLevel 是隊員的牧師等級（0 就是戰士）。
func newQuickTurnApp(t *testing.T, clericLevel uint8, column uint8, roller interface {
	Roll(int, int) int
}) (*app, *tacticalState) {
	t.Helper()
	application, state := newFoeCastApp(t)
	table, err := gamepack.ReadDOSTurnUndeadTable(filepath.Join("..", "..", "Pool of Radiance (1988).zip"))
	if err != nil {
		t.Fatal(err)
	}
	application.turnUndeadTable = &table
	application.roller = roller
	levels := make([]uint8, gamepack.ClassThac0ClassCount)
	if clericLevel > 0 {
		levels[gamepack.ClassSlotCleric] = clericLevel
	} else {
		levels[gamepack.ClassSlotFighter] = 5
	}
	application.state.Party[0] = poolsave.Character{Name: "A", ClassLevels: levels, Quick: true}
	var record gamepack.MonsterRecord
	record.Name = "SKELETON"
	record.Raw[gamepack.UndeadTurnColumnOffset] = column
	state.rememberSpellbook(2, record)
	state.rememberUndeadColumn(2, record)
	state.AIDriven[1] = true
	state.Mover = 1
	return application, state
}

// askedRun 說 asked 裡有沒有連續的 want。
func askedRun(asked []int, want ...int) bool {
	for start := 0; start+len(want) <= len(asked); start++ {
		match := true
		for offset, sides := range want {
			match = match && asked[start+offset] == sides
		}
		if match {
			return true
		}
	}
	return false
}

// 八級牧師對骷髏（欄 1）：門檻 −1，擲什麼都摧毀——從盤面收掉、`+10Ch = 8`。
// 骰序照原版：entry 3 的 d7，接著 116Ah 的 1d12（`11B3h`）與 1d20（`11C1h`）。
func TestQuickClericDestroysUndead(t *testing.T) {
	roller := &sequenceRoller{values: []int{1, 1, 1, 1, 1, 1, 1, 1, 1, 1}}
	application, state := newQuickTurnApp(t, 8, 1, roller)
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(state.FoeLog, "TURNS UNDEAD") || !strings.Contains(state.FoeLog, "2 IS DESTROYED") {
		t.Fatalf("the quick cleric did not destroy the skeleton: %q", state.FoeLog)
	}
	if state.Roster[2].FootprintClass != 0 || state.States[2] != turnDestroyedState {
		t.Fatalf("the destroyed skeleton is still on the board: %+v state %d", state.Roster[2], state.States[2])
	}
	if !askedRun(roller.asked, 7, 12, 20) {
		t.Fatalf("dice %v, want d7 then the 116Ah pair d12 d20", roller.asked)
	}
	if !state.Undead.Tried[1] {
		t.Fatal("runtime +11h was not set (116Ah `11AAh`)")
	}
}

// 一級牧師對骷髏：門檻 10。擲 15 轉變（runtime +10h），擲 3 什麼也沒發生；
// 兩種都用掉這一場唯一的一次（`+11h`），下一次輪到他就不再轉、一顆 d12 都不擲。
func TestQuickClericTurnsOncePerFight(t *testing.T) {
	application, state := newQuickTurnApp(t, 1, 1, &turnScript{d20: 15})
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(state.FoeLog, "2 IS TURNED") || !state.Undead.Turned[2] {
		t.Fatalf("the skeleton was not turned: %q", state.FoeLog)
	}
	if state.Roster[2].FootprintClass == 0 {
		t.Fatal("a turned skeleton left the board; only the destroy branch removes it")
	}

	script := &turnScript{d20: 20}
	application.roller = script
	state.Mover = 1
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(state.FoeLog, "TURNS UNDEAD") || script.asked(12) {
		t.Fatalf("the cleric turned twice in one fight: %q", state.FoeLog)
	}
}

func TestQuickClericFailingTheRollStillSpendsTheTurn(t *testing.T) {
	application, state := newQuickTurnApp(t, 1, 1, &turnScript{d20: 3})
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(state.FoeLog, "NOTHING HAPPENS") || state.Undead.Turned[2] {
		t.Fatalf("a failed turn should say nothing happens: %q", state.FoeLog)
	}
	if state.Scores[1] != 0 {
		t.Fatal("the failed turn did not use up the action (entry 34)")
	}
}

// 不是牧師（`+96h` 為 0）的不轉；是牧師而對面沒有不死生物（`+76h` 為 0）的也不轉，
// 而且兩者都不擲 116Ah 的 d12（entry 13 挑不到就回 0）。
func TestNonClericsAndLivingFoesAreNotTurned(t *testing.T) {
	for _, tc := range []struct {
		name           string
		cleric, column uint8
	}{
		{"fighter vs skeleton", 0, 1},
		{"cleric vs a living foe", 8, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			script := &turnScript{d20: 20}
			application, state := newQuickTurnApp(t, tc.cleric, tc.column, script)
			if err := press(application, ebiten.KeyEnter); err != nil {
				t.Fatal(err)
			}
			if strings.Contains(state.FoeLog, "TURNS UNDEAD") || script.asked(12) || state.Undead.Tried[1] {
				t.Fatalf("turned without a cleric level or an undead target: %q, dice %v",
					state.FoeLog, script.sides)
			}
		})
	}
}

// turnScript 回每一種骰面的固定值：d20 給指定值，其餘給 1。記下問過的面數。
type turnScript struct {
	d20   int
	sides []int
}

func (script *turnScript) Roll(_, sides int) int {
	script.sides = append(script.sides, sides)
	if sides == 20 && script.d20 > 0 {
		return script.d20
	}
	return 1
}

func (script *turnScript) asked(sides int) bool {
	for _, value := range script.sides {
		if value == sides {
			return true
		}
	}
	return false
}
