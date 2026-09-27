package main

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	"github.com/wicanr2/golden-box-remake-engine/eclvm"
)

// 倒下的狀態與反魔法區（spec 156，issue #115／#117）。全部從 Update() 送鍵。

// overlay-25 entry 28（`2266h`）：打穿 0 點 → 4、1..9 點 → 5（倒地計數 `+0Eh` = 打穿的點數）、
// 10 點以上 → 6；原本狀態是 1 的一律 6。近戰（overlay-13 `048Dh`）與法術（overlay-24 `14FBh`）
// 叫的是同一支。fixedRoller{19} 下 1dN 擲 N。
func TestDownStateFollowsTheOverkill(t *testing.T) {
	for _, tc := range []struct {
		name    string
		hp      int
		sides   uint8
		before  uint8
		state   uint8
		counter uint8
	}{
		{"exactly zero", 6, 6, 0, gamepack.UnconsciousState, 0},
		{"three through", 3, 6, 0, gamepack.DyingState, 3},
		{"nine through", 1, 10, 0, gamepack.DyingState, 9},
		{"ten through", 1, 11, 0, gamepack.DeadState, 0},
		{"animated, exactly zero", 6, 6, gamepack.AnimatedState, gamepack.DeadState, 0},
	} {
		t.Run("melee "+tc.name, func(t *testing.T) {
			application, state := deathBoard(t, 19)
			state.HitPoints[3], state.States[3] = tc.hp, tc.before
			state.setSingleAttackForm(2, combat.DamageDice{Count: 1, Sides: tc.sides})
			attackWithKeys(t, application, state, 2, 3)
			if state.Roster[3].FootprintClass != 0 || state.States[3] != tc.state ||
				state.DyingCounters[3] != tc.counter {
				t.Fatalf("state %d counter %d footprint %d (%q), want state %d counter %d", state.States[3],
					state.DyingCounters[3], state.Roster[3].FootprintClass, state.Status, tc.state, tc.counter)
			}
		})
	}
	// 法術：法師 3 級的魔法飛彈是 Roll(1, 4) + 1，fixedRoller{4} 下 5 點。
	for _, tc := range []struct {
		name    string
		hp      int
		state   uint8
		counter uint8
	}{{"exactly zero", 5, gamepack.UnconsciousState, 0}, {"two through", 3, gamepack.DyingState, 2}} {
		t.Run("spell "+tc.name, func(t *testing.T) {
			application, state := deathBoard(t, 4, gamepack.SpellIDMagicMissile)
			state.HitPoints[3] = tc.hp
			castFromMenuAt(t, application, state, 1, 0, 3)
			if state.Roster[3].FootprintClass != 0 || state.States[3] != tc.state ||
				state.DyingCounters[3] != tc.counter {
				t.Fatalf("state %d counter %d footprint %d (%q), want state %d counter %d", state.States[3],
					state.DyingCounters[3], state.Roster[3].FootprintClass, state.Status, tc.state, tc.counter)
			}
		})
	}
}

// 倒地計數從打穿的點數起算：打穿 7 點，回合收尾（overlay-08 `08C6h` 加一、`08D2h` 大於 9 → 6）
// 三回合後死透，不是十回合。
func TestDyingCounterStartsAtTheOverkill(t *testing.T) {
	application, state := deathBoard(t, 19)
	state.HitPoints[3] = 1
	state.setSingleAttackForm(2, combat.DamageDice{Count: 1, Sides: 8})
	attackWithKeys(t, application, state, 2, 3)
	if state.States[3] != gamepack.DyingState || state.DyingCounters[3] != 7 {
		t.Fatalf("after the hit: state %d counter %d", state.States[3], state.DyingCounters[3])
	}
	for round := 0; round < 3; round++ {
		endRoundWithKeys(t, application, state)
	}
	if state.States[3] != gamepack.DeadState {
		t.Fatalf("three rounds later: state %d counter %d", state.States[3], state.DyingCounters[3])
	}
}

// 84h 在自己的物品選單裡被弄倒（overlay-24 entry 19 `1610h..161Dh`）：倒下走 combatantDown（屍體表、
// 十六個碼、群組 13），狀態照 entry 28 分；overlay-19 的選單不收（`0F42h`／`0F51h`），但 `0F8Fh`
// `+10Dh` 為 0 之後 Use 不接。ESC 回到 overlay-08 的指令迴圈（`05A6h` → `036Ch`，不再看 `+10Dh`），
// 輪到的還是那一位。
func TestAlignedSwordDownInOwnItemMenu(t *testing.T) {
	for _, tc := range []struct {
		name  string
		hp    int
		state uint8
	}{{"five through", 10, gamepack.DyingState}, {"ten through", 5, gamepack.DeadState}} {
		t.Run(tc.name, func(t *testing.T) {
			// 魔杖不穿著：長劍要空出手來才裝得上。Use 開著時 U 會印 "must be readied"，關著什麼都不印。
			wand := itemOf("WAND", testTypeSword, false, 0)
			wand.Raw[gamepack.AIItemChargesOffset] = 2
			wand.Raw[gamepack.AIItemSpellOffset] = 18
			application, state := newItemMenuApp(t, int(gamepack.ClassSlotFighter), 5, alignedSwordF0(t), wand)
			application.state.Party[0].AlignmentID = "chaotic-good"
			state.HitPoints[1] = tc.hp
			pressAll(t, application, ebiten.KeyU)
			if footer := application.combatItemFooter(state, 0); footer != "READY USE DROP HALVE JOIN EXIT" {
				t.Fatalf("standing footer %q", footer)
			}
			pressAll(t, application, ebiten.KeyR)
			for guard := 0; guard < 600 && len(state.Notices) > 0; guard++ {
				idleFrame(t, application)
			}
			if state.Roster[1].FootprintClass != 0 || state.States[1] != tc.state {
				t.Fatalf("after the sword: state %d footprint %d", state.States[1], state.Roster[1].FootprintClass)
			}
			if tc.state == gamepack.DyingState && state.DyingCounters[1] != 5 {
				t.Fatalf("dying counter %d, want the 5 points through", state.DyingCounters[1])
			}
			if len(state.Corpses) != 1 || state.Corpses[0] != 1 {
				t.Fatalf("ov32 entry 20 corpse table %v", state.Corpses)
			}
			if application.combatItems == nil {
				t.Fatal("the item menu closed on its own")
			}
			if footer := application.combatItemFooter(state, 0); footer != "READY DROP HALVE JOIN EXIT" {
				t.Fatalf("downed footer %q", footer)
			}
			state.Status = ""
			pressAll(t, application, ebiten.KeyArrowDown, ebiten.KeyU)
			if state.Status != "" || application.castTargeting {
				t.Fatalf("U after going down: %q targeting %v", state.Status, application.castTargeting)
			}
			pressAll(t, application, ebiten.KeyEscape)
			if application.combatItems != nil || state.Mover != 1 {
				t.Fatalf("ESC: menu %v mover %d", application.combatItems != nil, state.Mover)
			}
		})
	}
}

// 反魔法區（`[4933h]+1CAh`，ECL `@49E5`；唯一的寫入端是 ECL8 block 16 `9BEEh`）：戰鬥指令列的
// Cast（overlay-08 `073Ah`）不接，C 不開法術清單；營地的施法（overlay-15 entry 2 → entry 8 模式 1，
// `036Eh`）印 "cannot cast spells in this area"、不列法術。
func TestAntiMagicShellBlocksCasting(t *testing.T) {
	application, state := deathBoard(t, 4, gamepack.SpellIDMagicMissile)
	application.combatCommands = turnUseSegments
	application.eventMachine = &eclvm.Machine{Memory: map[uint16]uint16{antiMagicAddress: 1}}
	state.Mover, state.Scores[1] = 1, 6
	if application.combatSegmentShown(gamepack.CombatCommandCast) {
		t.Fatal("Cast is on the command bar inside the shell")
	}
	pressAll(t, application, ebiten.KeyC)
	if application.castOpen {
		t.Fatalf("C opened the spell list inside the shell: %q", state.Status)
	}
	application.eventMachine.Memory[antiMagicAddress] = 0
	if !application.combatSegmentShown(gamepack.CombatCommandCast) {
		t.Fatal("Cast is missing after the shell is gone")
	}
	pressAll(t, application, ebiten.KeyC)
	if !application.castOpen {
		t.Fatalf("C did not open the spell list after the shell is gone: %q", state.Status)
	}

	camp := campCastApp(t, campCaster("A", 3, gamepack.SpellIDMagicMissile))
	camp.eventMachine = &eclvm.Machine{Memory: map[uint16]uint16{antiMagicAddress: 1}}
	pressAll(t, camp, ebiten.KeyC)
	if !camp.fieldCastOpen {
		t.Fatal("C did not open the caster picker")
	}
	pressAll(t, camp, ebiten.KeyEnter)
	if camp.fieldCastStage != fieldCastPickCaster ||
		!strings.Contains(strings.ToUpper(camp.fieldCastMessage), "CANNOT CAST SPELLS IN THIS AREA") {
		t.Fatalf("camp cast inside the shell: stage %d %q", camp.fieldCastStage, camp.fieldCastMessage)
	}
	camp.eventMachine.Memory[antiMagicAddress] = 0
	pressAll(t, camp, ebiten.KeyEnter)
	if camp.fieldCastStage != fieldCastPickSpell {
		t.Fatalf("camp cast after the shell is gone: stage %d %q", camp.fieldCastStage, camp.fieldCastMessage)
	}
}
