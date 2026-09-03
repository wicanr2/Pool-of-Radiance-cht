package main

import (
	"path/filepath"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// 傷害法術要先讓目標擲豁免，再依參數表 `+8` 的規則處置傷害（spec 074）。
// 少了這一條，火球術對豁免成功的目標照樣打滿。
func TestSpellDamageAppliesTheSavingThrowRule(t *testing.T) {
	parameters, err := gamepack.ReadDOSSpellParameters(
		filepath.Join("..", "..", "Pool of Radiance (1988).zip"))
	if err != nil {
		t.Skip("original DOS ZIP is intentionally not tracked")
	}
	state := &tacticalState{
		SaveTargets: make([][gamepack.SavingThrowCategories]uint8, 2),
		SaveBonus:   make([]int, 2),
	}
	// 目標值 20 代表「只有自然 20 才過」，所以骰子決定一切。
	for category := range state.SaveTargets[1] {
		state.SaveTargets[1][category] = gamepack.SavingThrowWorstTarget
	}
	for _, testCase := range []struct {
		name string
		roll int
		want int
	}{
		{"自然 20 一定豁免成功，火球術減半", 20, 10},
		{"自然 1 一定失敗，吃滿", 1, 21},
	} {
		application := &app{spellParameters: parameters, roller: fixedRoller{testCase.roll}}
		got := application.damageAfterSave(state, 1, gamepack.SpellIDFireball, 21)
		if got != testCase.want {
			t.Errorf("%s：得到 %d，預期 %d", testCase.name, got, testCase.want)
		}
	}
	// 規則 1 的法術豁免成功是完全無效。致病術（編號 40）就是規則 1。
	application := &app{spellParameters: parameters, roller: fixedRoller{20}}
	if got := application.damageAfterSave(state, 1, spellIDCauseDiseaseWithSave, 21); got != 0 {
		t.Errorf("定身術豁免成功還有 %d 點傷害，預期 0", got)
	}
}

// spellIDCauseDiseaseWithSave 是參數表裡規則 1 的一支（致病術，編號 40）。
const spellIDCauseDiseaseWithSave = uint8(gamepack.SpellIDCauseDisease)

// 定身術：規則 1，豁免成功完全無效，失敗就照參數表的回合數定住，
// 而被定住的那一格輪到就直接結束回合。
func TestHoldPersonHoldsATargetThatFailsItsSave(t *testing.T) {
	parameters, err := gamepack.ReadDOSSpellParameters(
		filepath.Join("..", "..", "Pool of Radiance (1988).zip"))
	if err != nil {
		t.Skip("original DOS ZIP is intentionally not tracked")
	}
	if got := parameters[gamepack.SpellIDHoldPerson].EffectCode(); got != gamepack.HoldPersonEffectCode {
		t.Fatalf("定身術的效果碼是 %#x，預期 %#x", got, gamepack.HoldPersonEffectCode)
	}
	state := &tacticalState{
		SaveTargets: make([][gamepack.SavingThrowCategories]uint8, 2),
		SaveBonus:   make([]int, 2),
		HeldRounds:  make([]int, 2),
	}
	for category := range state.SaveTargets[1] {
		state.SaveTargets[1][category] = gamepack.SavingThrowWorstTarget
	}
	// 自然 20 一定豁免成功。
	saved := (&app{spellParameters: parameters, roller: fixedRoller{20}}).
		savedAgainstSpell(state, 1, gamepack.SpellIDHoldPerson)
	if !saved {
		t.Error("自然 20 沒有豁免成功")
	}
	// 自然 1 一定失敗。
	saved = (&app{spellParameters: parameters, roller: fixedRoller{1}}).
		savedAgainstSpell(state, 1, gamepack.SpellIDHoldPerson)
	if saved {
		t.Error("自然 1 豁免成功了")
	}
}

// 被定住的一格輪到就結束回合，而且回合開始時剩餘回合數要減一——
// 少了遞減，一次定身術等於定到打完。
func TestHeldCombatantsLoseTheirTurnAndTheHoldWearsOff(t *testing.T) {
	state := newAttackState()
	state.HeldRounds = make([]int, len(state.Roster))
	state.HeldRounds[1] = 2
	state.Mover = 1
	application := &app{roller: fixedRoller{20}, tactical: state, language: languageEnglish}
	if err := application.tacticalInput(); err != nil {
		t.Fatal(err)
	}
	if state.Mover == 1 {
		t.Error("被定住的那一格還在行動")
	}
	before := state.HeldRounds[1]
	state.startRound(application.rollDice)
	if state.HeldRounds[1] != before-1 {
		t.Errorf("回合開始之後剩 %d 回合，預期 %d", state.HeldRounds[1], before-1)
	}
}
