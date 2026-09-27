package main

import (
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	"github.com/wicanr2/golden-box-remake-engine/eclvm"
)

// 遭遇腳本寫的 `@6E70..6E72`（隊伍記錄 `+6E0h..+6E4h`，side_adjust.go，#83）。
// 命中那兩格從 Update() 按 A 出手量；腳程那一格按 ENTER 過完一回合量。

func TestEncounterScriptAdjustsTheHitRoll(t *testing.T) {
	run := func(t *testing.T, attacker, target uint8, roll int, adjust sideAdjustments) bool {
		t.Helper()
		application, state := sideEffectBoard(t, int(gamepack.ClassSlotCleric), 1, gamepack.SpellIDBless, roll)
		state.Roster[3] = combat.CombatantCell{X: 9, Y: 5, FootprintClass: 1}
		state.SideAdjust = adjust
		before := state.HitPoints[target]
		attackWithKeys(t, application, state, attacker, target)
		return state.HitPoints[target] < before
	}
	// 命中骰調到 10 才中。隊伍那一邊讀 +6E2h（@6E71）：FCh 是 −4。
	if !run(t, 2, 3, 12, sideAdjustments{}) || run(t, 2, 3, 12, sideAdjustments{PartyHit: -4}) {
		t.Fatal("@6E71 = FCh should turn the party's 12 into a miss")
	}
	// 敵方讀 +6E0h（@6E70），隊伍那一格不影響它。
	if run(t, 3, 2, 8, sideAdjustments{PartyHit: 5}) || !run(t, 3, 2, 8, sideAdjustments{FoeHit: 2}) {
		t.Fatal("@6E70 = 02h should turn the foe's 8 into a hit, and only that one")
	}
}

func TestEncounterScriptSlowsThePartyOnly(t *testing.T) {
	application, state := sideEffectBoard(t, int(gamepack.ClassSlotCleric), 1, gamepack.SpellIDBless, 10)
	for index := range state.BaseMovement {
		state.BaseMovement[index] = 12
	}
	state.SideAdjust = sideAdjustments{PartyMove: 0xFC}
	endRoundWithKeys(t, application, state)
	// `0123h`：12 + FCh 寫回 byte 是 8，乘 2；敵方不加。
	if state.Budgets[1] != 16 || state.Budgets[2] != 16 || state.Budgets[3] != 24 {
		t.Fatalf("budgets %v, want party 16 and foe 24", state.Budgets[1:4])
	}
}

// 開打時讀、戰後主流程（overlay-05 `15D4h..15F0h`）清。
func TestEncounterScriptAdjustmentsAreReadAndCleared(t *testing.T) {
	machine := &eclvm.Machine{Memory: map[uint16]uint16{0x6E70: 0x02, 0x6E71: 0xFE, 0x6E72: 0xFC}}
	got := readSideAdjustments(machine)
	if got != (sideAdjustments{FoeHit: 2, PartyHit: -2, PartyMove: 0xFC}) {
		t.Fatalf("read %+v", got)
	}
	clearSideAdjustments(machine)
	if readSideAdjustments(machine) != (sideAdjustments{}) {
		t.Fatalf("not cleared: %v", machine.Memory)
	}
}
