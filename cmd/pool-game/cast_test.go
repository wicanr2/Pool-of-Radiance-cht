package main

import (
	"path/filepath"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
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

// 死靈術**不是**在盤面上生一個新的戰鬥員，是把已經死掉的人類屍體叫起來
// （spec 098 的 `2043h`）：換到施法者那一邊、改成不死、生命補滿、狀態變 1。
// 額度是施法者等級。
func TestAnimateDeadRaisesHumanCorpsesOnly(t *testing.T) {
	state := newRoundState(4)
	state.PartySlot = []int{-1, 0, -1, -1, -1}
	state.AIDriven = []bool{false, false, true, true, true}
	state.Friendly[1] = true
	state.HitPoints = []int{0, 10, 0, 0, 0}
	state.MaxHitPoints = []int{0, 10, 8, 8, 8}
	state.CreatureType = []uint8{0, 0, 0, 4, 0}
	state.States = []uint8{0, 0, combat.DeadState, combat.DeadState, combat.DyingState}
	state.Footprint = []uint8{0, 1, 1, 1, 1}
	for index := 2; index <= 4; index++ {
		state.Roster[index].FootprintClass = 0
	}
	state.Mover = 1

	raised := state.animateDead(1) // 施法者 1 級：額度 1
	if raised != 1 {
		t.Fatalf("叫起來 %d 具，額度 1 應該只叫得起 1 具", raised)
	}
	// 第 2 格是人類屍體，先被叫起來。
	if !state.Friendly[2] || !state.aiDrives(2) {
		t.Fatalf("第 2 格沒有換到施法者那一邊（%v / %v）", state.Friendly[2], state.aiDrives(2))
	}
	if state.CreatureType[2] != gamepack.CreatureTypeUndead {
		t.Fatalf("生物種類是 %d，預期 %d", state.CreatureType[2], gamepack.CreatureTypeUndead)
	}
	if state.HitPoints[2] != 8 || state.States[2] != gamepack.AnimatedState {
		t.Fatalf("生命 %d 狀態 %d", state.HitPoints[2], state.States[2])
	}
	if state.Roster[2].FootprintClass != 1 {
		t.Fatal("叫起來之後沒有站回盤面")
	}
	if state.BaseMovement[2] != gamepack.AnimatedDeadMovementRate {
		t.Fatalf("基礎移動是 %d，預期 %d", state.BaseMovement[2], gamepack.AnimatedDeadMovementRate)
	}
	if !state.hasEffect(2, gamepack.AnimateDeadEffectCode) {
		t.Fatal("效果碼 20h 沒有掛上")
	}
	// 第 3 格已經是不死（`+9Fh != 0`）、第 4 格只是瀕死（狀態 5），兩個都不動。
	if state.Friendly[3] || state.Friendly[4] {
		t.Fatal("不是人類屍體的也被叫起來了")
	}
	// 額度用完就停：再叫一次才輪到別人。
	if got := state.animateDead(5); got != 0 {
		t.Fatalf("剩下的都不合格，卻叫起來 %d 具", got)
	}
}

// 恢復術把能量吸取的欠帳還一級（spec 097 的 `2C01h`）：
// 還回來的 HP 是欠的 HP 除以欠的等級，沒欠就什麼都不做。
func TestRestorationPaysBackOneDrainedLevel(t *testing.T) {
	outcome := gamepack.Restore(0, 0)
	if outcome.Restored {
		t.Fatal("沒有欠帳卻還了")
	}
	// 欠 3 級 9 點：一次還 3 點。
	levels, points := 3, 9
	total := 0
	for round := 0; round < 3; round++ {
		outcome = gamepack.Restore(levels, points)
		if !outcome.Restored {
			t.Fatalf("第 %d 次應該還得動", round+1)
		}
		total += outcome.HitPoints
		levels, points = outcome.DrainedLevels, outcome.DrainedHitPoints
	}
	if levels != 0 || points != 0 {
		t.Fatalf("還完之後還欠 %d 級 %d 點", levels, points)
	}
	if total != 9 {
		t.Fatalf("還回來 %d 點，原本欠 9 點", total)
	}
}

// 解病術作用在**選中的目標**身上，不是施法者。原版 `225Bh` 逐個
// `lcall 0100h:006Bh(目標, …)` 再 `002Ah(目標, …)`（spec 098），
// 問的一直是目標；對自己施等於隊友中了病也治不好。
func TestCureDiseaseWorksOnTheChosenTarget(t *testing.T) {
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
	levels[gamepack.ClassSlotCleric] = 6
	party := make([]poolsave.Character, 2)
	for index := range party {
		party[index] = poolsave.Character{Name: string(rune('A' + index)),
			MaxHP: 20, CurrentHP: 20,
			ClassLevels: append([]uint8(nil), levels...),
			Memorised:   make([]uint8, gamepack.MemorisedSpellSlots)}
		party[index].Memorised[0] = gamepack.SpellIDCureDisease
	}
	// 第二個人身上有致病術的效果碼；施法者身上也放一個，用來證明
	// 拿掉的是目標那一份，不是順手把自己也治了。
	party[0].Effects = []uint8{gamepack.CureDiseaseEffectCodes[2]}
	party[1].Effects = []uint8{gamepack.CureDiseaseEffectCodes[2]}
	state := &tacticalState{
		Roster:      make([]combat.CombatantCell, 3),
		Friendly:    []bool{false, true, true},
		HitPoints:   []int{0, 20, 20},
		PartySlot:   []int{-1, 0, 1},
		States:      make([]uint8, 3),
		Scores:      []uint8{0, 5, 5},
		Budgets:     make([]uint8, 3),
		HitDice:     make([]uint8, 3),
		SleepFlag:   make([]uint8, 3),
		Effects:     make([]gamepack.EffectList, 3),
		ArmorClass:  make([]int, 3),
		THAC0:       make([]uint8, 3),
		Damage:      make([]combat.DamageDice, 3),
		SaveTargets: make([][gamepack.SavingThrowCategories]uint8, 3),
		SaveBonus:   make([]int, 3),
		Mover:       1,
	}
	for index := 1; index < 3; index++ {
		state.Roster[index] = combat.CombatantCell{X: uint8(index), Y: 1, FootprintClass: 1}
	}
	application := &app{tactical: state, tacticalPreview: true, mode: modeAdventure,
		spellParameters: parameters, spellCaster: caster, roller: fixedRoller{10},
		state: poolsave.State{Schema: poolsave.Schema, Party: party}}
	if err := application.finishCast(
		castOption{ID: gamepack.SpellIDCureDisease, Label: "解病術"}, 2, true); err != nil {
		t.Fatal(err)
	}
	if len(application.state.Party[1].Effects) != 0 {
		t.Errorf("目標身上還留著 %v，解病術沒有作用在他身上",
			application.state.Party[1].Effects)
	}
	if len(application.state.Party[0].Effects) != 1 {
		t.Errorf("施法者身上的效果被順手拿掉了：%v", application.state.Party[0].Effects)
	}
}
