package main

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// B）ANDAGE（spec 138，issue #25）：按鍵從 `Update()` 送進去，倒地的隊友變成
// 昏迷、計時歸零、包紮的人這一回合用掉；之後再多少回合都不會轉死亡。
func TestBandageStopsTheBleedingAndUsesTheTurn(t *testing.T) {
	state := newRoundState(3)
	state.Friendly[2] = true
	state.HitPoints = []int{0, 10, 0, 6}
	state.States[2] = combat.DyingState
	state.DyingCounters[2] = combat.DyingRoundLimit
	state.Mover = 1
	application := &app{mode: modeAdventure, tacticalPreview: true, roller: fixedRoller{20},
		tactical: state, language: languageEnglish}
	if err := press(application, ebiten.KeyB); err != nil {
		t.Fatal(err)
	}
	if got := state.States[2]; got != gamepack.UnconsciousState {
		t.Fatalf("state %d after bandaging, want %d (unconscious)", got, gamepack.UnconsciousState)
	}
	if state.DyingCounters[2] != 0 {
		t.Fatalf("dying counter %d after bandaging, want 0", state.DyingCounters[2])
	}
	if state.HitPoints[2] != 0 {
		t.Fatalf("bandaging changed hit points to %d; it only stops the bleeding", state.HitPoints[2])
	}
	if state.Mover == 1 {
		t.Fatal("bandaging did not use up the actor's turn")
	}
	if !strings.Contains(state.Status, "BANDAGED") {
		t.Fatalf("status %q, want the bandaged message", state.Status)
	}
	// 包紮過的人不再流血：再走過計時上限也還是昏迷。
	for round := 0; round <= int(combat.DyingRoundLimit)+1; round++ {
		state.endRound(fixedRoll)
	}
	if got := state.States[2]; got != gamepack.UnconsciousState {
		t.Fatalf("state %d after the rounds, want the bandaged member still unconscious", got)
	}
}

// 沒有人倒地時 B 什麼都不做，行動者也還在（原版連 `Bandage` 這一項都不會列出來）。
func TestBandageWithoutADyingTeammateDoesNothing(t *testing.T) {
	state := newAttackState()
	state.States[2] = combat.DyingState // 敵方倒地不算
	application := &app{mode: modeAdventure, tacticalPreview: true, roller: fixedRoller{20},
		tactical: state, language: languageEnglish}
	if err := press(application, ebiten.KeyB); err != nil {
		t.Fatal(err)
	}
	if state.Mover != 1 {
		t.Fatal("B with nobody to bandage consumed the turn")
	}
	if state.States[2] != combat.DyingState {
		t.Fatal("B bandaged a foe")
	}
}
