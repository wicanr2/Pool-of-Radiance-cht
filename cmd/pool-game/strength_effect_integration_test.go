package main

import (
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 從玩家按得到的 C 鍵施放變大術，再讓原版參數表指定的十回合走完
// （spec 112）；這條同時驗「掛上、時間過去、能力回復」，不是直接呼叫收尾 helper。
func TestCastingEnlargeThroughUpdateRestoresStrengthWhenItExpires(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	parameters, err := gamepack.ReadDOSSpellParameters(zipPath)
	if err != nil {
		t.Skip("original DOS ZIP is intentionally not tracked")
	}
	caster, err := gamepack.ReadDOSSpellCaster(zipPath)
	if err != nil {
		t.Skip("original DOS ZIP is intentionally not tracked")
	}
	levels := make([]uint8, gamepack.ClassThac0ClassCount)
	levels[gamepack.ClassSlotMagicUser] = 1
	member := poolsave.Character{
		Name:        "A",
		ClassID:     "magic-user",
		ClassLevels: levels,
		Abilities:   [6]int{10, 10, 10, 10, 10, 10},
		Memorised:   make([]uint8, gamepack.MemorisedSpellSlots),
	}
	member.Memorised[0] = gamepack.SpellIDEnlarge
	state := newAttackState()
	state.PartySlot = []int{-1, 0, -1}
	application := &app{
		tactical:        state,
		tacticalPreview: true,
		mode:            modeAdventure,
		language:        languageEnglish,
		spellParameters: parameters,
		spellCaster:     caster,
		roller:          fixedRoller{1},
		state: poolsave.State{
			Schema:           poolsave.Schema,
			Party:            []poolsave.Character{member},
			CharacterLibrary: []poolsave.Character{member},
		},
	}
	state.PartyEffectTeardown = func(index int, node gamepack.EffectNode) {
		application.expiredEffectTeardown(state.PartySlot[index], node, state.Effects[index])
	}

	if err := press(application, ebiten.KeyC); err != nil {
		t.Fatal(err)
	}
	if !application.castOpen {
		t.Fatal("按 C 沒有開出施法清單")
	}
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if !application.castTargeting {
		t.Fatal("變大術沒有進入選目標步驟")
	}
	// 預設游標停在最近敵人；N 繞到施法者自己，再用 Enter 確認。
	if err := press(application, ebiten.KeyN); err != nil {
		t.Fatal(err)
	}
	if application.castTargets[application.castTargetCursor] != 1 {
		t.Fatalf("選目標游標沒有移到施法者：%v／%d",
			application.castTargets, application.castTargetCursor)
	}
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if got := application.state.Party[0].Abilities[gamepack.AbilityStrength]; got != 18 {
		t.Fatalf("施法後力量是 %d，預期 18", got)
	}
	if len(state.Effects[1]) != 1 ||
		state.Effects[1][0].Code != gamepack.EnlargeEffectCode ||
		!state.Effects[1][0].NeedsTeardown() {
		t.Fatalf("變大術沒有掛成可收尾節點：%v", state.Effects[1])
	}
	duration := parameters[gamepack.SpellIDEnlarge].Duration(1)
	for round := 0; round < duration; round++ {
		state.startRound(application.rollDice)
	}
	if len(state.Effects[1]) != 0 {
		t.Fatalf("%d 回合後效果還在：%v", duration, state.Effects[1])
	}
	if got := application.state.Party[0].Abilities[gamepack.AbilityStrength]; got != 10 {
		t.Fatalf("效果到期後力量是 %d，預期回復 10", got)
	}
	if got := application.state.CharacterLibrary[0].Abilities[gamepack.AbilityStrength]; got != 10 {
		t.Fatalf("角色庫力量是 %d，沒有同步回復", got)
	}
}
