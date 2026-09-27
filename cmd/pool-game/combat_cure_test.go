package main

import (
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// #116：戰場上的解病術改的是盤面那一份串列（state.Effects）。以前改的是隊伍那一份，收場時
// storeCombatEffects 用盤面那一份蓋回去，病又回來。
func TestCureDiseaseOnTheBoardStaysCuredAfterCombat(t *testing.T) {
	application, state := deathBoard(t, 4, gamepack.SpellIDCureDisease)
	// 參數表 `+0`（解病術那一格）是模式 0：`20AEh` 收的是施法者自己。
	if mode := application.spellParameters[gamepack.SpellIDCureDisease].TargetMode(); mode != 0 {
		t.Fatalf("cure disease target mode %d, the fixture assumes 0 (self)", mode)
	}
	disease := gamepack.NewEffectNode(gamepack.DiseaseEffectCode, 0, gamepack.EffectUndispellable, false)
	// 進場時 tactical.go 從隊伍抄一份到盤面：兩份都帶著病。
	state.Effects[1] = gamepack.EffectList{disease}
	application.state.Party[0].Effects = storedEffects(gamepack.EffectList{disease})
	castFromMenuAt(t, application, state, 1, 0, 1)
	if state.hasEffect(1, gamepack.DiseaseEffectCode) {
		t.Fatalf("the board still carries the disease: %+v (%q)", state.Effects[1], state.Status)
	}
	// 收場寫回（finishCombat 的第一步）。
	application.storeCombatEffects(state)
	if _, still := memberEffect(application.state.Party[0], gamepack.DiseaseEffectCode); still {
		t.Fatalf("the disease came back after combat: %+v", application.state.Party[0].Effects)
	}
}
