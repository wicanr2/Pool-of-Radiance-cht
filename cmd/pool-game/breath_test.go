package main

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// 吐息（`58h`，overlay-22 `3092h`，#82）：輪到帶著它的怪物時，overlay-09 entry 5 開場的
// 群組 0Eh 先問它。從 Update() 送 ENTER，輪到敵方時 tacticalInput 自己分派 foeTurn。
//
// 盤面是 newFoeCastApp：隊員（1）在 (5,5)，敵人（2）擺到 (8,5)，三格——閃電束的挑法
// 射程是 4（龍沒有法師等級）。敵人在右邊，起點是 (7,5)，射線往左經過隊員。
func newBreathApp(t *testing.T, level uint8) (*app, *tacticalState) {
	t.Helper()
	application, state := newFoeCastApp(t)
	state.Roster[2].X = 8
	state.Effects[2] = state.Effects[2].Append(gamepack.NewEffectNode(gamepack.BreathEffectCode, 0, level, false))
	return application, state
}

func TestDragonBreathesDownALine(t *testing.T) {
	application, state := newBreathApp(t, 0xff)
	before := state.Roster[2]
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	// 傷害是生命上限 30（`+32h`）；d20 擲 7 過不了豁免，整份吃下。
	if state.HitPoints[1] != 10 {
		t.Fatalf("party hp %d, want 40 − 30 (log %q)", state.HitPoints[1], state.FoeLog)
	}
	if !strings.Contains(state.FoeLog, "Breathes!") {
		t.Fatalf("no Breathes! notice: %q", state.FoeLog)
	}
	if state.Roster[2] != before || state.Activity.FoeAttacks != 0 {
		t.Fatalf("the breath did not end the action: %+v → %+v, attacks %d", before, state.Roster[2],
			state.Activity.FoeAttacks)
	}
	at, ok := state.Effects[2].IndexOf(gamepack.BreathEffectCode)
	if !ok || state.Effects[2][at].Payload[2] != 0xfe {
		t.Fatalf("the breath node should count down FFh → FEh: %+v", state.Effects[2])
	}
}

// 第三次（+3 是 FDh）吐完摘掉節點；相位不是 0 時擲 d100，50 以下這一回合不吐、照常接近。
func TestDragonBreathRunsOutAndSkipsHalfTheLaterPhases(t *testing.T) {
	application, state := newBreathApp(t, 0xfd)
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if state.hasEffect(2, gamepack.BreathEffectCode) || state.HitPoints[1] != 10 {
		t.Fatalf("the last breath: node %+v, party hp %d", state.Effects[2], state.HitPoints[1])
	}

	application, state = newBreathApp(t, 0xff)
	state.AttackPhase = 1 // fixedRoller 的 d100 是 7
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(state.FoeLog, "Breathes!") || !state.hasEffect(2, gamepack.BreathEffectCode) {
		t.Fatalf("d100 7 should skip the breath: %q", state.FoeLog)
	}
	if state.Roster[2].X == 8 && state.Activity.FoeAttacks == 0 {
		t.Fatalf("without the breath the foe should close in: %+v", state.Roster[2])
	}
}
