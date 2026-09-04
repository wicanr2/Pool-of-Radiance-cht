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
		Effects:     make([]gamepack.EffectList, 2),
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
	state.addEffect(1, gamepack.HoldPersonEffectCode, 2, 6)
	state.Mover = 1
	application := &app{roller: fixedRoller{20}, tactical: state, language: languageEnglish}
	if err := application.tacticalInput(); err != nil {
		t.Fatal(err)
	}
	if state.Mover == 1 {
		t.Error("被定住的那一格還在行動")
	}
	before := state.effectRounds(1, gamepack.HoldPersonEffectCode)
	state.startRound(application.rollDice)
	if got := state.effectRounds(1, gamepack.HoldPersonEffectCode); got != before-1 {
		t.Errorf("回合開始之後剩 %d 回合，預期 %d", got, before-1)
	}
	// 減到 0 就整個摘掉，不是留一個持續 0 的節點賴著。
	state.startRound(application.rollDice)
	if state.hasEffect(1, gamepack.HoldPersonEffectCode) {
		t.Error("定身到期之後節點還掛著")
	}
}

// 解除魔法走的是目標身上的效果節點串列（spec 098 的 `2356h`）：
// 每一個各擲一次，`+3` 是 `0FFh` 的解不掉。
//
// 這一條同時擋住「兩份真相」：睡眠、定身與魅惑必須真的存在那條串列上，
// 不是另外幾個旗標——存成旗標的話解除魔法解到的會是空的。
func TestDispelMagicWalksTheEffectList(t *testing.T) {
	state := newAttackState()
	state.addEffect(2, gamepack.SleepEffectCode, 0, 3)
	state.addEffect(2, gamepack.HoldPersonEffectCode, 5, 3)
	state.Effects[2] = state.Effects[2].Append(
		gamepack.NewEffectNode(0x3D, 0, gamepack.EffectUndispellable, false))

	// 必成：骰 1。解不掉的那一個要留著。
	removed := state.dispelEffects(2, 6, func() int { return 1 })
	if removed != 2 {
		t.Fatalf("拿掉了 %d 個，預期 2 個", removed)
	}
	if state.hasEffect(2, gamepack.SleepEffectCode) ||
		state.hasEffect(2, gamepack.HoldPersonEffectCode) {
		t.Fatal("睡眠或定身沒有被解掉")
	}
	if !state.hasEffect(2, 0x3D) {
		t.Fatal("`+3` 是 0FFh 的效果不該被解掉")
	}
	// 必敗：骰 100 大於施法者 6 對效果 3 的 65。
	state.addEffect(2, gamepack.SleepEffectCode, 0, 3)
	if got := state.dispelEffects(2, 6, func() int { return 100 }); got != 0 {
		t.Fatalf("必敗卻拿掉了 %d 個", got)
	}
}

// 睡眠與魅惑掛在同一條串列上，而且行動判定讀的就是它。
func TestSleepAndCharmLiveOnTheEffectList(t *testing.T) {
	state := newAttackState()
	state.addEffect(1, gamepack.SleepEffectCode, 0, 6)
	state.Mover = 1
	application := &app{roller: fixedRoller{20}, tactical: state, language: languageEnglish}
	if err := application.tacticalInput(); err != nil {
		t.Fatal(err)
	}
	if state.Mover == 1 {
		t.Error("睡著的那一格還在行動")
	}
	// 持續 0 代表沒有回合計時：回合邊界不該把它摘掉。
	state.startRound(application.rollDice)
	if !state.hasEffect(1, gamepack.SleepEffectCode) {
		t.Error("持續 0 的睡眠被回合邊界摘掉了")
	}
}

// 被迷住的會**倒戈**（spec 112 的 overlay-12 entry 14）：陣營變成施法者
// 那一邊、改由 AI 分派行動，原本的陣營記在節點 `+3` 的位元 6。
// 解掉之後兩者都還原。
func TestCharmSwitchesSidesAndDispelRestoresThem(t *testing.T) {
	state := newAttackState()
	state.PartySlot = []int{-1, 0, -1}
	state.AIDriven = []bool{false, false, true}
	state.Mover = 1 // 我方施法
	if state.Friendly[2] {
		t.Fatal("目標一開始就該是敵方")
	}
	state.applyCharm(2, 6)
	if !state.Friendly[2] {
		t.Fatal("被迷住之後沒有倒戈到施法者那一邊")
	}
	if !state.aiDrives(2) {
		t.Fatal("倒戈之後應該仍由 AI 分派，不是交給玩家")
	}
	// 節點記著原本的陣營。
	at, ok := state.Effects[2].IndexOf(gamepack.CharmPersonEffectCode)
	if !ok {
		t.Fatal("魅惑的節點沒有掛上")
	}
	if state.Effects[2][at].OriginalSide() != 0 {
		t.Fatalf("原陣營記成 %d，預期 0", state.Effects[2][at].OriginalSide())
	}
	// 套過的不重複套：再迷一次不該把「原陣營」覆寫成現在這一邊。
	state.applyCharm(2, 6)
	at, _ = state.Effects[2].IndexOf(gamepack.CharmPersonEffectCode)
	if state.Effects[2][at].OriginalSide() != 0 {
		t.Fatal("重複施放把原陣營覆寫掉了")
	}
	// 解掉之後還原。
	if removed := state.dispelEffects(2, 6, func() int { return 1 }); removed == 0 {
		t.Fatal("必成的解除魔法什麼都沒解掉")
	}
	if state.Friendly[2] {
		t.Fatal("解掉魅惑之後陣營沒有還原")
	}
	if !state.aiDrives(2) {
		t.Fatal("怪物還原之後仍該由 AI 走")
	}
}
