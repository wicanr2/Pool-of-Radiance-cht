package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
)

func TestTacticalPilotDropsGoalAfterRosterShrinks(t *testing.T) {
	state := &tacticalState{
		Roster:   make([]combat.CombatantCell, 9),
		Friendly: make([]bool, 9),
		Budgets:  make([]uint8, 9),
		Round:    3,
		Mover:    1,
	}
	state.Roster[1].FootprintClass = 1
	state.Friendly[1] = true
	state.Budgets[1] = 6
	pilot := &tacticalPilot{
		mover:    1,
		round:    3,
		tried:    true,
		goal:     9,
		distance: map[int]int{0: 1},
	}

	if got := pilot.key(&app{tactical: state}); got != ebiten.KeyEnter {
		t.Fatalf("stale target with no remaining foe returned %v, want Enter", got)
	}
	if pilot.goal != 0 || pilot.distance != nil {
		t.Fatalf("stale target cache survived roster shrink: goal=%d distance=%v",
			pilot.goal, pilot.distance)
	}
}
