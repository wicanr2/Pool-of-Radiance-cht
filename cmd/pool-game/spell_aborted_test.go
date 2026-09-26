package main

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// AI 施法挑不到目標（issue #102，spec 098〈瞄不到目標：Abort Spell〉）。
// 開始施法之後輪到那一格時由 `012Eh` 放出去，這裡直接把 runtime +0 設成定身術，
// 再從 Update() 按 ENTER 讓 tacticalInput 自己分派 AI 那一格。

// giveFoeHoldPerson 讓敵方那一格是一個 7 級牧師，記著一條定身術，而且已經開始施法。
func giveFoeHoldPerson(state *tacticalState) {
	var record gamepack.MonsterRecord
	record.Name = "7TH LVL CLERIC"
	record.Raw[gamepack.AISpellArrayOffset+1] = gamepack.SpellIDHoldPerson
	record.Raw[0x96] = 7
	state.rememberSpellbook(2, record)
	state.Casting.Pending = map[int]uint8{2: gamepack.SpellIDHoldPerson}
}

func foeStillKnows(state *tacticalState, spell uint8) bool {
	for _, value := range state.Casting.Spells[2] {
		if value == spell {
			return true
		}
	}
	return false
}

// 反例：唯一的敵人身上已經有定身（34h），定身術又掛 34h——`1FC5h` 二十次都劃掉，
// `20AEh` 收到 0 個，`0EE7h` 印固定字串 "Spell Aborted"，法術從記憶清掉、行動用掉。
func TestFoeSpellAbortsWhenEveryTargetIsAlreadyHeld(t *testing.T) {
	for _, tc := range []struct {
		language language
		want     string
	}{
		{languageEnglish, "SPELL ABORTED"},
		{languageTraditionalChinese, "施法中止"},
	} {
		application, state := newFoeCastApp(t)
		application.language = tc.language
		giveFoeHoldPerson(state)
		state.addEffect(1, gamepack.HoldPersonEffectCode, 10, 7)
		if err := press(application, ebiten.KeyEnter); err != nil {
			t.Fatal(err)
		}
		if state.Status != tc.want || state.FoeLog != tc.want {
			t.Fatalf("%v: status %q log %q, want %q", tc.language, state.Status, state.FoeLog, tc.want)
		}
		if foeStillKnows(state, gamepack.SpellIDHoldPerson) {
			t.Fatal("an aborted spell stays memorised; 0F06h clears it")
		}
		if len(state.Casting.Pending) != 0 || state.Mover == 2 {
			t.Fatalf("the abort did not use up the action: pending %v mover %d", state.Casting.Pending, state.Mover)
		}
		if state.Activity.FoeCasts != 0 {
			t.Fatal("an aborted spell was counted as cast")
		}
	}
}

// 正例：同一個盤面，敵人身上沒有那四個效果——目標收得到，照常放出去。
func TestFoeSpellReleasesWhenATargetIsNotYetHeld(t *testing.T) {
	application, state := newFoeCastApp(t)
	giveFoeHoldPerson(state)
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(state.Status, "ABORTED") || !strings.Contains(state.FoeLog, "CASTS") {
		t.Fatalf("hold person did not go out: status %q log %q", state.Status, state.FoeLog)
	}
	if foeStillKnows(state, gamepack.SpellIDHoldPerson) || state.Activity.FoeCasts != 1 {
		t.Fatal("the released spell was not consumed")
	}
}
