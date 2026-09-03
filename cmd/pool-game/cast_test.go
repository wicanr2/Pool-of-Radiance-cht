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
