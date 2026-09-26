package main

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// 命中擲骰的群組 10／16 與掛效果前的群組 9（spec 112，issue #86）。全部從 Update() 送鍵：
// A 出手、C 施法。盤面是 sideEffectBoard：隊員 2 出手打敵人 3（移到 (9,5) 貼著），
// THAC0 內部值 40、AC 內部值 50，命中骰要調整到 10 才中；每下傷害 4（1d4 取 4）。

// hitCase 是一次出手：setup 設好效果與欄位，roll 是固定的骰值，wantHit 是結果。
// 每一條都同時跑一次不做 setup 的對照組，對照組的結果必須相反——拿掉修正就會紅。
type hitCase struct {
	name     string
	attacker uint8
	target   uint8
	roll     int
	setup    func(state *tacticalState)
	wantHit  bool
	// sameWithout 為真代表這一條是反例：條件不成立，結果應與對照組相同。
	sameWithout bool
}

func runHitCase(t *testing.T, tc hitCase, apply bool) bool {
	t.Helper()
	application, state := sideEffectBoard(t, int(gamepack.ClassSlotCleric), 1, gamepack.SpellIDBless, tc.roll)
	state.Roster[3] = combat.CombatantCell{X: 9, Y: 5, FootprintClass: 1}
	if apply {
		tc.setup(state)
	}
	before := state.HitPoints[tc.target]
	attackWithKeys(t, application, state, tc.attacker, tc.target)
	return state.HitPoints[tc.target] < before
}

func checkHitCases(t *testing.T, cases []hitCase) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.attacker == 0 {
				tc.attacker, tc.target = 2, 3
			}
			got := runHitCase(t, tc, true)
			if got != tc.wantHit {
				t.Fatalf("hit = %v, want %v", got, tc.wantHit)
			}
			control := runHitCase(t, tc, false)
			if tc.sameWithout && control != got {
				t.Fatalf("the unmet condition changed the outcome: %v without, %v with", control, got)
			}
			if !tc.sameWithout && control == got {
				t.Fatalf("the effect made no difference: %v with and without", got)
			}
		})
	}
}

func withEffect(index int, code uint8, level uint8) func(*tacticalState) {
	return func(state *tacticalState) {
		state.Effects[index] = state.Effects[index].Append(gamepack.NewEffectNode(code, 5, level, false))
	}
}

func both(steps ...func(*tacticalState)) func(*tacticalState) {
	return func(state *tacticalState) {
		for _, step := range steps {
			step(state)
		}
	}
}

func targetKind(index int, creature, size uint8, name string) func(*tacticalState) {
	return func(state *tacticalState) {
		state.CreatureType[index], state.BodySize[index] = creature, size
		state.rememberRecordName(index, name)
	}
}

// 群組 10：攻擊者身上的碼。
func TestAttackerEffectsAdjustTheHitRoll(t *testing.T) {
	checkHitCases(t, []hitCase{
		// 21h 失明（entry 31 `0BBEh`）與 24h 詛咒（entry 34 `0C2Dh`）：−4。
		{name: "21h −4", roll: 13, setup: withEffect(2, 0x21, 1), wantHit: false},
		{name: "24h −4", roll: 13, setup: withEffect(2, 0x24, 1), wantHit: false},
		// 03h（entry 7 `0141h`）：目標 +9Fh == 4 才 +2。
		{name: "03h vs undead +2", roll: 8,
			setup: both(withEffect(2, 0x03, 1), targetKind(3, 4, 1, "")), wantHit: true},
		{name: "03h vs living", roll: 8,
			setup: both(withEffect(2, 0x03, 1), targetKind(3, 0, 1, "")), wantHit: false, sameWithout: true},
		// 06h（entry 9 `01C9h`）：0Ah → +1、9／0Ch → +2、4 → +3，其餘 0。
		{name: "06h vs troll +1", roll: 9,
			setup: both(withEffect(2, 0x06, 1), targetKind(3, 0x0a, 1, "")), wantHit: true},
		{name: "06h vs 0Ch +2", roll: 8,
			setup: both(withEffect(2, 0x06, 1), targetKind(3, 0x0c, 1, "")), wantHit: true},
		{name: "06h vs undead +3", roll: 7,
			setup: both(withEffect(2, 0x06, 1), targetKind(3, 4, 1, "")), wantHit: true},
		{name: "06h vs human", roll: 9,
			setup: both(withEffect(2, 0x06, 1), targetKind(3, 0, 1, "")), wantHit: false, sameWithout: true},
		// 12h（entry 20 `068Bh`）：類人、體型 1、名字在 DS:0356h 那張表上 → +1。
		{name: "12h vs GOBLIN +1", roll: 9,
			setup: both(withEffect(2, 0x12, 1), targetKind(3, 1, 1, "GOBLIN")), wantHit: true},
		{name: "12h vs ORC", roll: 9,
			setup: both(withEffect(2, 0x12, 1), targetKind(3, 1, 1, "ORC")), wantHit: false, sameWithout: true},
		// 1Ah（entry 26 `0956h`）：八個名字；體型不是 1 就不比。
		{name: "1Ah vs HOBGOBLIN +1", roll: 9,
			setup: both(withEffect(2, 0x1a, 1), targetKind(3, 1, 1, "HOBGOBLIN")), wantHit: true},
		{name: "1Ah vs large HOBGOBLIN", roll: 9,
			setup: both(withEffect(2, 0x1a, 1), targetKind(3, 1, 2, "HOBGOBLIN")), wantHit: false, sameWithout: true},
	})
}

// 31h 祈禱（entry 46 `12C1h`）：`014Dh` 半徑 6 內有人帶著就算；節點 +3 的位元 4 是
// 施法者那一邊，和出手的人同邊 +1、不同邊 −1。帶著的是隊員 1，節點位元 4 為 0（隊伍這一邊）。
func TestPrayerAreaAdjustsBothSides(t *testing.T) {
	prayer := func(x, y uint8) func(*tacticalState) {
		return func(state *tacticalState) {
			state.Roster[1] = combat.CombatantCell{X: x, Y: y, FootprintClass: 1}
			withEffect(1, gamepack.PrayerAreaEffectCode, 3)(state)
		}
	}
	checkHitCases(t, []hitCase{
		{name: "ally in range +1", roll: 9, setup: prayer(7, 5), wantHit: true},
		{name: "foe in range −1", attacker: 3, target: 2, roll: 10, setup: prayer(7, 5), wantHit: false},
		{name: "ally out of range", roll: 9, setup: prayer(0, 0), wantHit: false, sameWithout: true},
	})
}

// 群組 16：目標身上的碼。
func TestTargetEffectsAdjustTheHitRoll(t *testing.T) {
	checkHitCases(t, []hitCase{
		// 19h（entry 25 `0927h`）與 47h（entry 66 `1737h`）：對它出手 −4。
		{name: "19h −4", roll: 13, setup: withEffect(3, gamepack.InvisibilityEffectCode, 1), wantHit: false},
		{name: "47h −4", roll: 13, setup: withEffect(3, 0x47, 1), wantHit: false},
		// 25h（entry 35 `0C40h`）：目標 runtime +3 大於 0 → 命中骰 FFh，自然 20 也落空。
		{name: "25h before it acts", roll: 20, setup: withEffect(3, gamepack.PhaseEffectCode, 1), wantHit: false},
		{name: "25h after it acted", roll: 10,
			setup:   both(withEffect(3, gamepack.PhaseEffectCode, 1), func(state *tacticalState) { state.Scores[3] = 0 }),
			wantHit: true, sameWithout: true},
		// 30h（entry 45 `1283h`）：比的是 `DS:5CF0h`（輪到的人）的名字。敵人 3 是 BUGBEAR、
		// 帶著 30h 的是隊員 2。
		{name: "30h vs BUGBEAR −4", attacker: 3, target: 2, roll: 13,
			setup: both(withEffect(2, 0x30, 1), targetKind(3, 1, 2, "BUGBEAR")), wantHit: false},
		{name: "30h vs ORC", attacker: 3, target: 2, roll: 13,
			setup: both(withEffect(2, 0x30, 1), targetKind(3, 1, 2, "ORC")), wantHit: true, sameWithout: true},
	})
}

// 59h 幻影移位（entry 83 `2461h`）：節點 +3 位元 4 沒立時這一擊落空並立起它，之後照常；
// 相位 `6CD7h` 為 0 而且調整後的命中骰剛好是 0 時清掉高四位。
func TestDisplacementMissesTheFirstAttack(t *testing.T) {
	application, state := sideEffectBoard(t, int(gamepack.ClassSlotCleric), 1, gamepack.SpellIDBless, 20)
	state.Roster[3] = combat.CombatantCell{X: 9, Y: 5, FootprintClass: 1}
	state.Effects[3] = state.Effects[3].Append(gamepack.NewEffectNode(gamepack.DisplacementEffectCode, 0, 1, false))
	attackWithKeys(t, application, state, 2, 3)
	if state.HitPoints[3] != 30 {
		t.Fatalf("the first attack on a displaced foe landed: %d hp", state.HitPoints[3])
	}
	at, _ := state.Effects[3].IndexOf(gamepack.DisplacementEffectCode)
	if state.Effects[3][at].Payload[2]&0x10 == 0 {
		t.Fatalf("the spent bit was not set: %+v", state.Effects[3][at])
	}
	attackWithKeys(t, application, state, 2, 3)
	if state.HitPoints[3] != 26 {
		t.Fatalf("the second attack should land normally: %d hp", state.HitPoints[3])
	}

	// 相位 0、命中骰調到 0（骰 4、攻擊者帶 24h 的 −4）：位元清掉，這一擊照常比 AC（落空）。
	application.roller = fixedRoller{4}
	state.AttackPhase = 0
	state.Effects[2] = state.Effects[2].Append(gamepack.NewEffectNode(0x24, 5, 1, false))
	attackWithKeys(t, application, state, 2, 3)
	at, _ = state.Effects[3].IndexOf(gamepack.DisplacementEffectCode)
	if state.Effects[3][at].Payload[2]&0xf0 != 0 {
		t.Fatalf("phase 0 with a zero roll should clear the high bits: %+v", state.Effects[3][at])
	}
	// 相位不是 0 時同一骰不清，而是再讓一擊落空（位元已清，這次重新立起）。
	state.AttackPhase = 1
	attackWithKeys(t, application, state, 2, 3)
	at, _ = state.Effects[3].IndexOf(gamepack.DisplacementEffectCode)
	if state.Effects[3][at].Payload[2]&0x10 == 0 {
		t.Fatalf("outside phase 0 the bit should be set again: %+v", state.Effects[3][at])
	}
}

// `0FCCh`：出手之前攻擊者身上的 19h 全部摘掉，連擲出 1 的那一下也一樣。
func TestAttackingDropsInvisibility(t *testing.T) {
	for _, roll := range []int{1, 15} {
		application, state := sideEffectBoard(t, int(gamepack.ClassSlotCleric), 1, gamepack.SpellIDBless, roll)
		state.Roster[3] = combat.CombatantCell{X: 9, Y: 5, FootprintClass: 1}
		for range 2 {
			state.Effects[2] = state.Effects[2].Append(
				gamepack.NewEffectNode(gamepack.InvisibilityEffectCode, 5, 1, false))
		}
		state.Effects[3] = state.Effects[3].Append(
			gamepack.NewEffectNode(gamepack.InvisibilityEffectCode, 5, 1, false))
		attackWithKeys(t, application, state, 2, 3)
		if state.hasEffect(2, gamepack.InvisibilityEffectCode) {
			t.Fatalf("roll %d: the attacker is still invisible: %+v", roll, state.Effects[2])
		}
		if !state.hasEffect(3, gamepack.InvisibilityEffectCode) {
			t.Fatalf("roll %d: the target lost its invisibility", roll)
		}
	}
}

// 群組 9：詛咒術（模式 0Ah）掛到帶著魔法抗性的敵人身上之前先擲 d100（`2910h`）。
// 門檻 = byte(百分比 − (11 − 施法者等級) × 5)，無號比較：
//
//	6Ah（15%）、1 級施法者：15 − 50 繞成 221，一定抗掉
//	6Ah（15%）、11 級：門檻 15，骰 19 抗不掉
//	69h（50%）、11 級：門檻 50，骰 19 抗掉
func TestMagicResistanceBlocksTheCurse(t *testing.T) {
	for _, tc := range []struct {
		name       string
		carried    uint8
		level      uint8
		unaffected bool
	}{
		{"15% vs level 1 wraps", 0x6a, 1, true},
		{"15% vs level 11", 0x6a, 11, false},
		{"50% vs level 11", 0x69, 11, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			application, state := sideEffectBoard(t, int(gamepack.ClassSlotCleric), tc.level, gamepack.SpellIDCurse, 19)
			state.Effects[3] = state.Effects[3].Append(gamepack.NewEffectNode(tc.carried, 0, 0, false))
			castAndAimAtIndex(t, application, state, 2)
			if got := !state.hasEffect(3, gamepack.CurseEffectCode); got != tc.unaffected {
				t.Fatalf("unaffected = %v, want %v: %+v", got, tc.unaffected, state.Effects[3])
			}
		})
	}
	// 對照：沒有抗性就一定掛上。
	application, state := sideEffectBoard(t, int(gamepack.ClassSlotCleric), 1, gamepack.SpellIDCurse, 19)
	castAndAimAtIndex(t, application, state, 2)
	if !state.hasEffect(3, gamepack.CurseEffectCode) {
		t.Fatal("the curse did not land without resistance")
	}
}

// 群組 9 的點名免疫：定身術的碼 34h 被 6Dh／6Fh／7Dh 擋掉，印 "is Unaffected"；
// 6Ch 只擋 0Bh／35h，不擋定身。
func TestHoldImmunityCodesBlockTheHold(t *testing.T) {
	for _, tc := range []struct {
		carried uint8
		blocked bool
	}{
		{0, false}, {0x6d, true}, {0x6f, true}, {0x7d, true}, {0x6c, false},
	} {
		application, state := newSpellBoard(t,
			spellCasterWith(int(gamepack.ClassSlotCleric), 3, gamepack.SpellIDHoldPerson),
			combat.CombatantCell{X: 5, Y: 5, FootprintClass: 1},
			combat.CombatantCell{X: 7, Y: 5, FootprintClass: 1})
		application.roller = fixedRoller{19}
		if tc.carried != 0 {
			state.Effects[2] = state.Effects[2].Append(gamepack.NewEffectNode(tc.carried, 0, 0, false))
		}
		pressAll(t, application, ebiten.KeyC, ebiten.KeyEnter)
		if state.Casting.Pending[1] != 0 {
			pressAll(t, application, ebiten.KeyEnter)
		}
		// 收一個，然後 Exit 把要的數量減到 1 就放出去（`22BEh`）。
		pressAll(t, application, ebiten.KeyEnter)
		for guard := 0; guard < 8 && application.castTargeting; guard++ {
			pressAll(t, application, ebiten.KeyEscape)
		}
		if application.castTargeting {
			t.Fatalf("carrying %#x: hold person was never released: %q", tc.carried, state.Status)
		}
		held := state.hasEffect(2, gamepack.HoldPersonEffectCode)
		if held == tc.blocked {
			t.Errorf("carrying %#x: held %v, want blocked %v (status %q)", tc.carried, held, tc.blocked, state.Status)
		}
		if tc.blocked && !strings.Contains(state.Status, state.say(msgCastUnaffected, 2)) {
			t.Errorf("carrying %#x: no Unaffected message: %q", tc.carried, state.Status)
		}
	}
}

// castAtTarget 按 C、ENTER 選第一格記憶（施法時間不為 0 再按一次 ENTER），瞄準清單上
// N 換到 index，ENTER 放出去。單一目標的法術走這一條（不是瞄一點）。
func castAtTarget(t *testing.T, application *app, state *tacticalState, index uint8) {
	t.Helper()
	pressAll(t, application, ebiten.KeyC, ebiten.KeyEnter)
	if state.Casting.Pending[1] != 0 {
		pressAll(t, application, ebiten.KeyEnter)
	}
	if !application.castTargeting {
		t.Fatalf("the spell did not open aiming: %q", state.Status)
	}
	for guard := 0; guard <= len(application.castTargets); guard++ {
		if application.castTargets[application.castTargetCursor] == index {
			break
		}
		pressAll(t, application, ebiten.KeyN)
	}
	if application.castTargets[application.castTargetCursor] != index {
		t.Fatalf("aiming never reached combatant %d: %v", index, application.castTargets)
	}
	pressAll(t, application, ebiten.KeyEnter)
	if application.castTargeting {
		t.Fatalf("aiming did not release the spell: %q", state.Status)
	}
}

// bySidesRoller 依骰子面數給固定值：d20 與 d100 要分開控制時用。
type bySidesRoller map[int]int

func (roller bySidesRoller) Roll(_, sides int) int {
	if value, ok := roller[sides]; ok && value <= sides {
		return value
	}
	return 1
}

// 魅惑（碼 0Bh）：6Ch 一定擋；6Bh（精靈）擲 d100 不大於 90 擋、7Ch（半精靈）不大於 30 擋。
// d20 固定 19（豁免失敗），d100 另外給。
func TestCharmImmunityCodesBlockTheCharm(t *testing.T) {
	for _, tc := range []struct {
		carried uint8
		d100    int
		blocked bool
	}{
		{0, 19, false}, {0x6c, 99, true}, {0x6d, 19, false},
		{0x6b, 90, true}, {0x6b, 91, false}, {0x7c, 30, true}, {0x7c, 31, false},
	} {
		application, state := sideEffectBoard(t, int(gamepack.ClassSlotMagicUser), 3, gamepack.SpellIDCharmPerson, 19)
		application.roller = bySidesRoller{20: 19, 100: tc.d100, 4: 4, 6: 6, 7: 7, 2: 2}
		if tc.carried != 0 {
			state.Effects[3] = state.Effects[3].Append(gamepack.NewEffectNode(tc.carried, 0, 0, false))
		}
		castAtTarget(t, application, state, 3)
		charmed := state.hasEffect(3, gamepack.CharmPersonEffectCode)
		if charmed == tc.blocked {
			t.Errorf("carrying %#x d100 %d: charmed %v, want blocked %v (status %q)",
				tc.carried, tc.d100, charmed, tc.blocked, state.Status)
		}
	}
}

// 群組 17（overlay-09 `1172h`／`11C5h`）：兩關都先把 `DS:6783h` 交給祝福（+5）與詛咒（−5，
// 不低於 0）再比。盤面是 newFleeApp：敵人整體只剩一成（`6D22h` = 10），第一關士氣 0。
//
//	隊伍 +58Ch 86 → 門檻 14：沒效果 10 過不了、會逃；祝福 15 撐住，照常逼近
//	隊伍 +58Ch 92 → 門檻 8：沒效果 10 撐住；詛咒 5 過不了、會逃
func TestBlessAndCurseMoveTheMoraleCheck(t *testing.T) {
	for _, tc := range []struct {
		name  string
		party uint16
		code  uint8
		flees bool
	}{
		{"plain vs 14 flees", 86, 0, true},
		{"blessed vs 14 holds", 86, gamepack.MoraleBoostEffectCode, false},
		{"plain vs 8 holds", 92, 0, false},
		{"cursed vs 8 flees", 92, gamepack.MoraleDropEffectCode, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ones := make([]int, 32)
			for index := range ones {
				ones[index] = 1
			}
			application, state := newFleeApp(t, 10, 5, 1, 0xFF, &sequenceRoller{values: ones})
			state.HitPoints[2] = 3
			state.refreshSideMorale()
			state.Morale.Party = tc.party
			if tc.code != 0 {
				state.Effects[2] = state.Effects[2].Append(gamepack.NewEffectNode(tc.code, 6, 1, false))
			}
			if err := press(application, ebiten.KeyEnter); err != nil {
				t.Fatal(err)
			}
			if fled := state.Roster[2].X > 10; fled != tc.flees {
				t.Fatalf("fled %v, want %v: at (%d,%d), log %q",
					fled, tc.flees, state.Roster[2].X, state.Roster[2].Y, state.FoeLog)
			}
		})
	}
}
