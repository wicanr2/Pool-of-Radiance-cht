package main

import (
	"math/rand"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 敵方要在多牆的斜投影盤面上繞得過來。
//
// 這一條擋的是「敵方朝目標方向走一格」的選路：在 TestPassiveCombatTerminates
// 那種空曠小場地看不出來，換到史藍鎮那場 41 隻的大場面就變成敵方整場
// CLOSED 0 STEPS、站在原地不動——隊伍完全不還手也不會死，戰鬥難度與原版不同。
// 所以門檻訂在「完全不動不打的隊伍會被打死」，而不是「有人靠過來」。
func TestFoesCloseOnACrowdedBoard(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	application.roller = diceRoller{random: rand.New(rand.NewSource(29))}
	party := make([]poolsave.Character, 0, 6)
	for index := 0; index < 6; index++ {
		party = append(party, poolsave.Character{Name: string(rune('A' + index)), RaceID: "dwarf",
			GenderID: "male", ClassID: "fighter", AlignmentID: "lawful-good",
			Abilities: [6]int{18, 10, 10, 16, 10, 10}, MaxHP: 60, CurrentHP: 60,
			PortraitHead: 1, PortraitBody: 1, IconSize: 1})
	}
	application.state = poolsave.State{Schema: poolsave.Schema, CharacterLibrary: party, Party: party}
	application.saveState = func(poolsave.State) error { return nil }
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if err := press(application, ebiten.KeyB); err != nil {
		t.Fatal(err)
	}
	for tick := 0; tick < 20000 && !application.introDone; tick++ {
		if application.introWaiting || application.tourPage >= 0 {
			if err := press(application, ebiten.KeyEnter); err != nil {
				t.Fatal(err)
			}
			continue
		}
		application.keys = scriptedKeys{}
		if err := application.Update(); err != nil {
			t.Fatal(err)
		}
	}
	// 逐格走，走到**夠擠的那一場**為止：路上遇到的小場面照打完再繼續。
	// 原本靠種子 29 亂走碰運氣，世界一變大同一個種子就走去別的地方。
	crowded := func() bool {
		state := application.tactical
		if state == nil {
			return false
		}
		foes := 0
		for index := 1; index < len(state.Roster); index++ {
			if state.Roster[index].FootprintClass != 0 && !state.Friendly[index] {
				foes++
			}
		}
		return foes >= 20
	}
	if !walkThisAreaUntil(t, application, 120000, crowded) {
		t.Fatal("走完整區都沒碰到夠擠的那一場")
	}
	state := application.tactical
	if state == nil {
		t.Fatal("never reached combat")
	}
	foes := 0
	for index := 1; index < len(state.Roster); index++ {
		if state.Roster[index].FootprintClass != 0 && !state.Friendly[index] {
			foes++
		}
	}
	// 這場要夠擠才有意義；場面小的那條由 TestPassiveCombatTerminates 顧。
	if foes < 20 {
		t.Fatalf("這場只有 %d 隻敵人，擠不出繞路問題", foes)
	}
	t.Logf("擠的那一場在 GEO%d/%d：%d 隻敵人",
		application.spawn.Map.Archive, application.spawn.Map.BlockID, foes)
	for tick := 0; tick < 40000; tick++ {
		current := application.tactical
		if current == nil || current.Finished {
			break
		}
		key := ebiten.KeyEnter // 一律結束回合：完全不動不打
		if current.Prompt {
			key = ebiten.KeyY
		}
		if err := press(application, key); err != nil {
			t.Fatalf("combat tick %d: %v", tick, err)
		}
	}
	if application.tactical != nil {
		t.Fatalf("戰鬥沒收尾：round %d, foe log %q",
			application.tactical.Round, application.tactical.FoeLog)
	}
	if !strings.Contains(application.statusLine, "defeated") {
		t.Fatalf("%d 隻敵人打不死一支完全不還手的隊伍，敵方大概沒繞過來：%q",
			foes, application.statusLine)
	}
}
