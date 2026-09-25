package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 玩家施法的時間、打斷與瞄準（spec 098〈施法時間與打斷〉〈收目標〉，issue #72／#73）。
// 全部從 Update() 送鍵。

// newSpellBoard 是一張空曠的盤面：1 號是隊員（施法者），其餘照 cells 擺成敵方。
// 參數表與處理常式照原版 ZIP 讀。
func newSpellBoard(t *testing.T, caster poolsave.Character, casterCell combat.CombatantCell,
	foes ...combat.CombatantCell) (*app, *tacticalState) {
	t.Helper()
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	parameters, err := gamepack.ReadDOSSpellParameters(zipPath)
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	spellCaster, err := gamepack.ReadDOSSpellCaster(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	count := 1 + len(foes)
	state := newRoundState(count)
	size := count + 1
	cellCount := combat.TacticalRowStride * (combat.TacticalMaxY + 1)
	const openTerrain = 5
	terrain := make([]uint8, cellCount)
	for index := range terrain {
		terrain[index] = openTerrain
	}
	state.Grid = combat.TacticalGrid{Terrain: terrain}
	state.Classes[openTerrain] = gamepack.CombatCellClass{EntryThreshold: 1}
	state.Roster[1] = casterCell
	for index, cell := range foes {
		state.Roster[index+2] = cell
	}
	state.HitPoints = make([]int, size)
	state.MaxHitPoints = make([]int, size)
	state.THAC0 = make([]uint8, size)
	state.ArmorClass = make([]int, size)
	state.HitDice = make([]uint8, size)
	state.SleepFlag = make([]uint8, size)
	state.CreatureType = make([]uint8, size)
	state.BodySize = make([]uint8, size)
	state.SaveTargets = make([][gamepack.SavingThrowCategories]uint8, size)
	state.SaveBonus = make([]int, size)
	state.PartySlot = make([]int, size)
	state.FoeTargets = make([]uint8, size)
	state.TacticModes = make([]uint8, size)
	for index := 1; index < size; index++ {
		state.HitPoints[index], state.MaxHitPoints[index] = 30, 30
		state.THAC0[index], state.ArmorClass[index] = 40, 50
		state.HitDice[index] = 1
		state.BodySize[index] = 1
		state.PartySlot[index] = -1
		state.Budgets[index] = 12
		state.setSingleAttackForm(index, combat.DamageDice{Count: 1, Sides: 4})
		for category := range state.SaveTargets[index] {
			state.SaveTargets[index][category] = gamepack.SavingThrowWorstTarget
		}
	}
	state.PartySlot[1] = 0
	state.AIDriven = aiDriven(state.PartySlot)
	state.Mover = 1
	state.Scores[1] = 6
	for index := 2; index < size; index++ {
		state.Scores[index] = 2
	}
	state.startCastingRound()
	caster.Memorised = append([]uint8(nil), caster.Memorised...)
	application := &app{mode: modeAdventure, tacticalPreview: true, roller: fixedRoller{1},
		tactical: state, language: languageEnglish, spellParameters: parameters, spellCaster: spellCaster}
	application.state.Party = []poolsave.Character{caster}
	return application, state
}

func spellCasterWith(class int, level uint8, spells ...uint8) poolsave.Character {
	levels := make([]uint8, gamepack.ClassThac0ClassCount)
	levels[class] = level
	memorised := make([]uint8, gamepack.MemorisedSpellSlots)
	copy(memorised, spells)
	return poolsave.Character{Name: "CASTER", ClassLevels: levels, Memorised: memorised,
		MaxHP: 30, CurrentHP: 30}
}

func pressAll(t *testing.T, application *app, keys ...ebiten.Key) {
	t.Helper()
	for _, key := range keys {
		if err := press(application, key); err != nil {
			t.Fatal(err)
		}
	}
}

// 火球術的施法時間是 1（+0Ch 3 ÷ 3）：按下去只是「開始施法」，先攻扣 1、記憶不動；
// 重選之後輪到自己（分數仍最高）才瞄準、放出去（overlay-13 `24E7h`、overlay-08 `031Bh`）。
func TestPlayerFireballGoesOffOnTheNextTurn(t *testing.T) {
	application, state := newSpellBoard(t,
		spellCasterWith(int(gamepack.ClassSlotMagicUser), 5, gamepack.SpellIDFireball),
		combat.CombatantCell{X: 5, Y: 5, FootprintClass: 1},
		combat.CombatantCell{X: 9, Y: 5, FootprintClass: 1})
	pressAll(t, application, ebiten.KeyC, ebiten.KeyEnter)
	if state.Casting.Pending[1] != gamepack.SpellIDFireball || !strings.Contains(state.Status, "BEGINS CASTING") {
		t.Fatalf("fireball did not begin casting: pending %v status %q", state.Casting.Pending, state.Status)
	}
	if state.Scores[1] != 5 {
		t.Fatalf("initiative after beginning: %d, want 6 − 1", state.Scores[1])
	}
	if application.state.Party[0].Memorised[0] != gamepack.SpellIDFireball {
		t.Fatal("memory was spent when casting began (14ECh runs on release)")
	}
	if state.HitPoints[2] != 30 || application.castTargeting {
		t.Fatalf("beginning to cast already hit or aimed: hp %d aiming %v", state.HitPoints[2], application.castTargeting)
	}
	if state.Mover != 1 {
		t.Fatalf("mover %d; the caster still has the highest score", state.Mover)
	}
	// 輪到時先放出去：瞄準在這一刻開。
	pressAll(t, application, ebiten.KeyEnter)
	if !application.castTargeting || len(state.Casting.Pending) != 0 {
		t.Fatalf("the pending fireball did not open aiming: aiming %v pending %v status %q",
			application.castTargeting, state.Casting.Pending, state.Status)
	}
	pressAll(t, application, ebiten.KeyEnter)
	if state.HitPoints[2] >= 30 {
		t.Fatalf("the fireball never went off: hp %d status %q", state.HitPoints[2], state.Status)
	}
	if application.state.Party[0].Memorised[0] != 0 {
		t.Fatal("the released fireball is still memorised")
	}
	if state.Mover == 1 {
		t.Fatal("releasing the spell did not end the caster's action (entry 34)")
	}
}

// 開始施法之後挨了打：輪到時法術沒了（從記憶清掉），而這一回合的指令列不再有
// Cast（runtime +1 為 0，overlay-08 `072Fh`）。
func TestPlayerLosesAPendingSpellWhenWounded(t *testing.T) {
	application, state := newSpellBoard(t,
		spellCasterWith(int(gamepack.ClassSlotMagicUser), 5, gamepack.SpellIDFireball, gamepack.SpellIDMagicMissile),
		combat.CombatantCell{X: 5, Y: 5, FootprintClass: 1},
		combat.CombatantCell{X: 9, Y: 5, FootprintClass: 1})
	application.combatCommands = []gamepack.CombatCommandSegment{
		{Key: gamepack.CombatCommandMove, Text: "Move "},
		{Key: gamepack.CombatCommandCast, Text: "Cast "},
		{Key: gamepack.CombatCommandQuickDone, Text: "Quick Done"},
	}
	if !strings.Contains(application.combatCommandBar(), "Cast") {
		t.Fatalf("an unhurt caster should see Cast: %q", application.combatCommandBar())
	}
	pressAll(t, application, ebiten.KeyC, ebiten.KeyEnter)
	if state.Casting.Pending[1] == 0 {
		t.Fatal("fireball did not begin casting")
	}
	// 受傷的當下就丟失（overlay-24 entry 19 `150Fh..155Ch`），不等輪到。
	application.applySpellDamage(state, 1, 3)
	if !strings.Contains(state.Status, "LOST A SPELL") || len(state.Casting.Pending) != 0 {
		t.Fatalf("the wounded caster kept the spell: status %q pending %v", state.Status, state.Casting.Pending)
	}
	if application.state.Party[0].Memorised[0] != 0 {
		t.Fatal("the lost fireball is still memorised (14ECh clears it)")
	}
	if application.castTargeting || state.HitPoints[2] != 30 {
		t.Fatal("a lost spell still aimed or hit")
	}
	if state.Mover != 1 {
		t.Fatalf("losing the spell used up the turn: mover %d", state.Mover)
	}
	// 同一回合再按 C：Cast 不在指令列上，清單不開。
	pressAll(t, application, ebiten.KeyC)
	if application.castOpen || !strings.Contains(state.Status, "NO SPELLCASTING") {
		t.Fatalf("cast menu opened after a wound: open %v status %q", application.castOpen, state.Status)
	}
	if strings.Contains(application.combatCommandBar(), "Cast") {
		t.Fatalf("command bar still offers Cast: %q", application.combatCommandBar())
	}
}

// 催眠術瞄的是一個點（`1E09h` 的第三個引數是 1）：Manual 游標停在空格子也選得到，
// 以那一格為中心、預算 +6 & 7 = 1 收人（`220Fh`）。遠處那一隻不在範圍裡。
func TestSleepAimsAtAnEmptyCellAndCollectsTheArea(t *testing.T) {
	application, state := newSpellBoard(t,
		spellCasterWith(int(gamepack.ClassSlotMagicUser), 3, gamepack.SpellIDSleep),
		combat.CombatantCell{X: 5, Y: 5, FootprintClass: 1},
		combat.CombatantCell{X: 9, Y: 5, FootprintClass: 1},
		combat.CombatantCell{X: 9, Y: 7, FootprintClass: 1},
		combat.CombatantCell{X: 14, Y: 5, FootprintClass: 1})
	application.roller = fixedRoller{4} // 額度 Roll(4, 4)：fixedRoller 回 4，放倒兩個 1 HD 夠用
	pressAll(t, application, ebiten.KeyC, ebiten.KeyEnter)
	if !application.castTargeting || application.castAim == nil || !application.castAim.plan.PointAim {
		t.Fatalf("sleep did not open point aiming: aiming %v", application.castTargeting)
	}
	if application.castTargets[application.castTargetCursor] != 2 {
		t.Fatalf("default target %d, want the nearest foe 2", application.castTargets[application.castTargetCursor])
	}
	// M 進 Manual（停在 2 號那一格 9,5），P 往南一格到空格 9,6，ENTER 選定。
	pressAll(t, application, ebiten.KeyM, ebiten.KeyP)
	if application.castManualX != 9 || application.castManualY != 6 {
		t.Fatalf("manual cursor at %d,%d", application.castManualX, application.castManualY)
	}
	pressAll(t, application, ebiten.KeyEnter)
	if application.castTargeting {
		t.Fatalf("an empty cell was not accepted as the centre: %q", state.Status)
	}
	for _, index := range []int{2, 3} {
		if !state.hasEffect(index, gamepack.SleepEffectCode) {
			t.Errorf("foe %d next to the centre is awake", index)
		}
	}
	if state.hasEffect(4, gamepack.SleepEffectCode) {
		t.Error("the far foe fell asleep; the area budget is 1")
	}
}

// 單體法術瞄的是人：Manual 停在空格子按 ENTER 什麼都不做（`30DEh`）。
func TestSingleTargetSpellRefusesAnEmptyCell(t *testing.T) {
	application, state := newSpellBoard(t,
		spellCasterWith(int(gamepack.ClassSlotMagicUser), 4, gamepack.SpellIDMagicMissile),
		combat.CombatantCell{X: 5, Y: 5, FootprintClass: 1},
		combat.CombatantCell{X: 8, Y: 5, FootprintClass: 1})
	pressAll(t, application, ebiten.KeyC, ebiten.KeyEnter, ebiten.KeyM, ebiten.KeyP, ebiten.KeyEnter)
	if !application.castTargeting || !application.castManual {
		t.Fatalf("an empty cell ended single-target aiming: %q", state.Status)
	}
	if application.state.Party[0].Memorised[0] != gamepack.SpellIDMagicMissile {
		t.Fatal("magic missile was spent on an empty cell")
	}
}

// 超出射程的目標沒有 "Target" 可按（`2AF2h`）：按 ENTER 停在瞄準裡，記憶不動。
func TestSpellTargetOutOfRangeStaysInAiming(t *testing.T) {
	application, state := newSpellBoard(t,
		spellCasterWith(int(gamepack.ClassSlotMagicUser), 4, gamepack.SpellIDMagicMissile),
		combat.CombatantCell{X: 5, Y: 5, FootprintClass: 1},
		combat.CombatantCell{X: 30, Y: 5, FootprintClass: 1})
	// 魔法飛彈 +2 = 6、+3 = 4：4 級 22 格，放在 25 格外。預設游標找的是這一回合
	// 移動額度內繞得到的最近敵人，額度給滿，游標才會停在那一隻上。
	state.Budgets[1] = 0xFF
	pressAll(t, application, ebiten.KeyC, ebiten.KeyEnter, ebiten.KeyEnter)
	if !application.castTargeting || state.HitPoints[2] != 30 {
		t.Fatalf("an out-of-range target was hit: aiming %v hp %d status %q",
			application.castTargeting, state.HitPoints[2], state.Status)
	}
	if !strings.Contains(state.Status, "RANGE") {
		t.Fatalf("no range message: %q", state.Status)
	}
}

// 定身術逐個收 3 個（`22BEh`）：重複的印 "Already been targeted" 不算數；Exit 讓要的
// 數量減一，減完就照收到的那幾個放。豁免修正看收到幾個（2 個是 −1）。
func TestHoldPersonCollectsTargetsOneByOne(t *testing.T) {
	application, state := newSpellBoard(t,
		spellCasterWith(int(gamepack.ClassSlotCleric), 3, gamepack.SpellIDHoldPerson),
		combat.CombatantCell{X: 5, Y: 5, FootprintClass: 1},
		combat.CombatantCell{X: 7, Y: 5, FootprintClass: 1},
		combat.CombatantCell{X: 7, Y: 7, FootprintClass: 1},
		combat.CombatantCell{X: 8, Y: 3, FootprintClass: 1})
	// 施法時間 1：開始施法，輪到下一次才瞄。
	pressAll(t, application, ebiten.KeyC, ebiten.KeyEnter)
	if state.Casting.Pending[1] != gamepack.SpellIDHoldPerson {
		t.Fatalf("hold person did not begin casting: %q", state.Status)
	}
	pressAll(t, application, ebiten.KeyEnter)
	aim := application.castAim
	if !application.castTargeting || aim == nil || aim.remaining != 3 {
		t.Fatalf("hold person should ask for 3 targets: aiming %v aim %+v", application.castTargeting, aim)
	}
	first := application.castTargets[application.castTargetCursor]
	pressAll(t, application, ebiten.KeyEnter)
	if len(aim.picks) != 1 || aim.picks[0] != first || !application.castTargeting {
		t.Fatalf("first pick not collected: %+v", aim)
	}
	// 重新開的瞄準預設又停在同一個：再按一次是重複的。
	pressAll(t, application, ebiten.KeyEnter)
	if len(aim.picks) != 1 || !strings.Contains(state.Status, "ALREADY BEEN TARGETED") {
		t.Fatalf("a duplicate was counted: picks %v status %q", aim.picks, state.Status)
	}
	// N 換到下一個敵人（跳過施法者自己那一格）再收。
	for guard := 0; guard < len(application.castTargets); guard++ {
		pressAll(t, application, ebiten.KeyN)
		next := application.castTargets[application.castTargetCursor]
		if next != first && next != 1 {
			break
		}
	}
	second := application.castTargets[application.castTargetCursor]
	pressAll(t, application, ebiten.KeyEnter)
	if len(aim.picks) != 2 || aim.picks[1] != second {
		t.Fatalf("second pick not collected: %+v", aim)
	}
	// Exit：還要的那一個不要了，照收到的兩個放。
	pressAll(t, application, ebiten.KeyEscape)
	if application.castTargeting || application.castAim != nil {
		t.Fatalf("exit with two picks did not release the spell: %q", state.Status)
	}
	for _, index := range []uint8{first, second} {
		if !state.hasEffect(int(index), gamepack.HoldPersonEffectCode) {
			t.Errorf("target %d was not held: %q", index, state.Status)
		}
	}
	for index := 2; index < len(state.Roster); index++ {
		if uint8(index) != first && uint8(index) != second && state.hasEffect(index, gamepack.HoldPersonEffectCode) {
			t.Errorf("unpicked foe %d was held", index)
		}
	}
	if application.state.Party[0].Memorised[0] != 0 {
		t.Fatal("hold person is still memorised")
	}
}

// 一個都沒收到就按 Exit：問 "Abort Spell?"，N 回去重瞄，Y 放棄——法術從記憶清掉、
// 行動用掉（overlay-22 `0EC0h..0F0Bh`）。
func TestAbortSpellForgetsItAndEndsTheAction(t *testing.T) {
	application, state := newSpellBoard(t,
		spellCasterWith(int(gamepack.ClassSlotMagicUser), 4, gamepack.SpellIDMagicMissile),
		combat.CombatantCell{X: 5, Y: 5, FootprintClass: 1},
		combat.CombatantCell{X: 8, Y: 5, FootprintClass: 1})
	pressAll(t, application, ebiten.KeyC, ebiten.KeyEnter, ebiten.KeyEscape)
	if !application.castAborting() || !strings.Contains(state.Status, "ABORT SPELL") {
		t.Fatalf("exit with nothing picked did not ask to abort: %q", state.Status)
	}
	pressAll(t, application, ebiten.KeyN)
	if !application.castTargeting || application.castAborting() {
		t.Fatal("N did not go back to aiming")
	}
	pressAll(t, application, ebiten.KeyEscape, ebiten.KeyY)
	if application.castAim != nil || application.castTargeting {
		t.Fatal("Y did not leave the spell")
	}
	if application.state.Party[0].Memorised[0] != 0 {
		t.Fatal("an aborted spell stays memorised; 0F06h clears it")
	}
	if !strings.Contains(state.Status, "SPELL ABORTED") || state.Mover == 1 || state.HitPoints[2] != 30 {
		t.Fatalf("abort did not end the action cleanly: status %q mover %d hp %d",
			state.Status, state.Mover, state.HitPoints[2])
	}
}

// AI 那一側與玩家同一張 SpellTargetPlan：定身術收 (模式 & 3) + 1 個，擲到重複的直接
// 少算一個（`237Fh`），收到的整張表都交給效果。
func TestFoeHoldPersonTargetsSeveralPartyMembers(t *testing.T) {
	application, state := newSpellBoard(t,
		spellCasterWith(int(gamepack.ClassSlotCleric), 3),
		combat.CombatantCell{X: 5, Y: 5, FootprintClass: 1},
		combat.CombatantCell{X: 7, Y: 5, FootprintClass: 1},
		combat.CombatantCell{X: 5, Y: 7, FootprintClass: 1})
	// 2 號改成隊員，3 號是施定身術的敵方牧師。
	state.Friendly[2], state.PartySlot[2] = true, 1
	application.state.Party = append(application.state.Party, spellCasterWith(int(gamepack.ClassSlotFighter), 3))
	state.AIDriven = aiDriven(state.PartySlot)
	var record gamepack.MonsterRecord
	record.Name = "PRIEST"
	record.Raw[0x96] = 5
	state.rememberSpellbook(3, record)
	// 20 次重挑裡每一次擲 Roll(1, n)：第一次擲到 1、第二次擲到 2、第三次又擲到 1。
	application.roller = &sequenceRoller{values: []int{1, 2, 1}}
	caster, ok := application.foeSpellcasterFor(state, 3)
	if !ok {
		t.Fatal("no spellbook for the foe")
	}
	state.Mover = 3
	targets, found, err := application.foeSpellTargets(state, 3, gamepack.SpellIDHoldPerson, caster)
	if err != nil || !found {
		t.Fatalf("no targets: %v", err)
	}
	if len(targets.List) != 2 {
		t.Fatalf("hold person collected %v, want both party members (the third roll is a duplicate)", targets.List)
	}
}
