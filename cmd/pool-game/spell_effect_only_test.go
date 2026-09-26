package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 只掛效果的那一批（spec 098〈只掛效果的那一批〉，issue #89）。全部從 Update() 送鍵：
// C 開施法清單、Enter 施、瞄準時 N 換人、Enter 確定；A 出手；U 用物品。

// effectOnlyBoard：施法者 1 在 (5,5)，隊友 2 在 (6,5)，對面 3 在 (5,6)——兩個都貼著施法者，
// 碰觸法術（參數表 +2 是 FFh，射程 1）搆得到。施法者牧師、法師各 3 級，魅力 12。
// THAC0 內部值 40、AC 內部值 50：命中骰要 10 才中；豁免目標值一律 20（自然 20 才過）。
func effectOnlyBoard(t *testing.T, roll int, spells ...uint8) (*app, *tacticalState) {
	t.Helper()
	caster := spellCasterWith(int(gamepack.ClassSlotCleric), 3, spells...)
	caster.ClassLevels[gamepack.ClassSlotMagicUser] = 3
	caster.Abilities = [6]int{12, 12, 12, 12, 12, 12}
	application, state := newSpellBoard(t, caster,
		combat.CombatantCell{X: 5, Y: 5, FootprintClass: 1},
		combat.CombatantCell{X: 6, Y: 5, FootprintClass: 1},
		combat.CombatantCell{X: 5, Y: 6, FootprintClass: 1})
	state.Friendly[2], state.PartySlot[2] = true, 1
	application.state.Party = append(application.state.Party,
		poolsave.Character{Name: "ALLY", Age: 20, MaxHP: 30, CurrentHP: 30,
			Abilities: [6]int{12, 12, 12, 12, 12, 12}})
	for index := range state.AIDriven {
		state.AIDriven[index] = false
	}
	state.PartyAged = application.agePartyMember(state)
	// 與開打時建的盤面同一條（tactical.go）：隊員的效果到期時跑收尾、寫回角色。
	state.PartyEffectTeardown = func(index int, node gamepack.EffectNode) {
		if party := state.PartySlot[index]; party >= 0 {
			application.expiredEffectTeardown(party, node, state.Effects[index])
		}
	}
	application.roller = fixedRoller{roll}
	return application, state
}

// castFromMenuAt 讓 caster 輪到，C 開清單、游標移到第 menuIndex 條、Enter 施；施法時間不為 0
// 的按 Enter 推到放出去。要瞄準的就把瞄準移到 target 再 Enter；要挑好幾個的（模式 1..7）
// 挑完第一個之後按 ESC，`23BBh` 把要的數量減到 0 就照收到的放。回傳放出去那一下按鍵之前的
// 回合數：放完如果剛好換回合，效果會在回合初始化時先減一。
func castFromMenuAt(t *testing.T, application *app, state *tacticalState, caster uint8,
	menuIndex int, target uint8) int {
	t.Helper()
	released := state.Round
	press := func(key ebiten.Key) {
		t.Helper()
		released = state.Round
		pressAll(t, application, key)
	}
	state.Mover, state.Scores[caster] = caster, 6
	press(ebiten.KeyC)
	if !application.castOpen {
		t.Fatalf("C did not open the spell list: %q", state.Status)
	}
	for guard := 0; guard < len(application.castOptions) && application.castCursor != menuIndex; guard++ {
		press(ebiten.KeyArrowDown)
	}
	press(ebiten.KeyEnter)
	for guard := 0; guard < 64 && state.Casting.Pending[int(caster)] != 0 && !application.castTargeting; guard++ {
		press(ebiten.KeyEnter)
	}
	if !application.castTargeting {
		return released
	}
	for guard := 0; guard <= len(application.castTargets); guard++ {
		if application.castTargets[application.castTargetCursor] == target {
			break
		}
		press(ebiten.KeyN)
	}
	if application.castTargets[application.castTargetCursor] != target {
		t.Fatalf("aiming never reached %d: %v", target, application.castTargets)
	}
	press(ebiten.KeyEnter)
	for guard := 0; guard < 8 && application.castTargeting; guard++ {
		press(ebiten.KeyEscape)
	}
	if application.castTargeting || application.castAim != nil {
		t.Fatalf("aiming did not release the spell: %q", state.Status)
	}
	return released
}

// 每一支落到 default 的法術都照原版掛上參數表 +0Ah：碼、持續（`07C7h`）、節點 +3（等級或
// 等級覆寫推進來的整個 byte）與 +4（`[bp+0Eh]`）逐項對。fixedRoller{19}：豁免必敗、碰觸必中；
// 持續的特例照 fixedRoller 的取值（Roll(n, s) 回 min(19, s)）算。
func TestEffectOnlySpellsAttachTheParameterCode(t *testing.T) {
	const (
		self = 1
		ally = 2
		foe  = 3
	)
	cases := []struct {
		spell     uint8
		target    uint8
		duration  uint16
		magnitude uint8
		teardown  bool
	}{
		{5, self, 10, 3, false},    // Detect Magic：+4 = 0Ah
		{6, ally, 9, 3, false},     // Protection From Evil：3 × 3
		{7, ally, 9, 3, false},     // Protection from Good
		{8, ally, 30, 3, false},    // Resist Cold：0Ah × 3
		{11, self, 6, 3, false},    // Detect Magic（法師）
		{14, self, 3, 12, true},    // Friends：節點 +3 是施法前的魅力，`[bp+0Eh]` 1
		{16, ally, 6, 3, false},    // Protection From Evil（法師）
		{17, ally, 6, 3, false},    // Protection From Good（法師）
		{18, self, 6, 3, false},    // Read Magic → 10h
		{19, self, 15, 3, false},   // Shield → 11h
		{22, self, 30, 3, false},   // Find Traps：+4 = 1Eh
		{24, ally, 30, 3, false},   // Resist Fire
		{25, foe, 6, 3, false},     // Silence, 15' Radius（規則 3，豁免不擋）
		{28, self, 3, 3, true},     // Spiritual Hammer：`[bp+0Eh]` 1
		{29, self, 15, 3, false},   // Detect Invisibility
		{30, ally, 0, 3, false},    // Invisibility：持續 0，出手才摘
		{32, self, 6, 4, false},    // Mirror Image：影像數 Roll(1, 4) 借等級那一格
		{33, foe, 3, 3, false},     // Ray of Enfeeblement
		{38, foe, 0, 3, false},     // Cause Blindness（碰觸）
		{40, foe, 60, 3, true},     // Cause Disease（碰觸）：`07C7h` Roll(1, 6) × 10
		{42, self, 3, 3, false},    // Prayer：(0 << 4) + 3
		{44, foe, 30, 3, false},    // Bestow Curse（碰觸）
		{45, self, 3, 3, false},    // Blink
		{50, ally, 0, 3, false},    // Invisibility, 10' Radius
		{52, ally, 6, 3, false},    // Protection From Evil, 10' Radius
		{53, ally, 6, 3, false},    // Protection From Good, 10' Radius
		{54, ally, 30, 3, false},   // Protection From Normal Missiles
		{57, self, 4, 12, false},   // 39h：`07C7h` Roll(5, 4)
		{61, foe, 4, 12, false},    // 3Dh：`07C7h` Roll(5, 4)
		{63, ally, 100, 12, false}, // 3Fh：戰鬥中 Roll(2, 10) × 10
		{67, self, 0x5a0, 0xff, true},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("spell %d", tc.spell), func(t *testing.T) {
			application, state := effectOnlyBoard(t, 19, tc.spell)
			code := application.spellParameters[tc.spell].EffectCode()
			released := castFromMenuAt(t, application, state, 1, 0, tc.target)
			at, ok := state.Effects[tc.target].IndexOf(code)
			if !ok {
				t.Fatalf("combatant %d carries no %02Xh node: %+v (status %q)",
					tc.target, code, state.Effects[tc.target], state.Status)
			}
			node := state.Effects[tc.target][at]
			if tc.duration != 0 {
				// 放完剛好換回合的，回合初始化已經先減過一次（tickEffects）。
				tc.duration -= uint16(state.Round - released)
			}
			if node.Duration() != tc.duration || node.Magnitude() != tc.magnitude ||
				node.NeedsTeardown() != tc.teardown {
				t.Fatalf("node %02Xh: duration %d magnitude %#x teardown %v, want %d %#x %v",
					code, node.Duration(), node.Magnitude(), node.NeedsTeardown(),
					tc.duration, tc.magnitude, tc.teardown)
			}
		})
	}
}

// 開鎖術（`1A34h`）是泛型版型但參數表 +0Ah 為 0：`0A18h` 跳過，什麼都不掛。
func TestKnockAttachesNothing(t *testing.T) {
	application, state := effectOnlyBoard(t, 19, 31)
	castFromMenuAt(t, application, state, 1, 0, 1)
	for index := 1; index < len(state.Effects); index++ {
		if len(state.Effects[index]) != 0 {
			t.Fatalf("knock attached %+v to %d", state.Effects[index], index)
		}
	}
	if !strings.Contains(state.Status, "casts") {
		t.Fatalf("knock printed %q", state.Status)
	}
}

// 祈禱術：`249Dh` 把 `(邊 << 4) + 等級` 推在等級覆寫那一格，`08BCh` 掛在施法者身上。
// 半徑 6 內同一邊出手 +1、另一邊 −1（entry 46 `12C1h`）：命中骰 9 原本差一點，隊友打得中；
// 對面擲 10 原本剛好，祈禱之後打不中。
func TestPrayerCastFromTheMenuShiftsBothSides(t *testing.T) {
	application, state := effectOnlyBoard(t, 9, gamepack.SpellIDPrayer)
	castFromMenuAt(t, application, state, 1, 0, 1)
	if !state.hasEffect(1, gamepack.PrayerAreaEffectCode) {
		t.Fatalf("the caster carries no prayer node: %+v", state.Effects[1])
	}
	attackWithKeys(t, application, state, 2, 3)
	if state.HitPoints[3] != 26 {
		t.Fatalf("a praying ally should hit on 9: foe has %d hp, status %q", state.HitPoints[3], state.Status)
	}
	application.roller = fixedRoller{10}
	attackWithKeys(t, application, state, 3, 2)
	if state.HitPoints[2] != 30 {
		t.Fatalf("the foe should miss on 10 under prayer: ally has %d hp", state.HitPoints[2])
	}
}

// 同一盤不施祈禱：兩下都照原本的骰，這是上一條的對照組。
func TestWithoutPrayerTheSameRollsMissAndHit(t *testing.T) {
	application, state := effectOnlyBoard(t, 9, gamepack.SpellIDPrayer)
	attackWithKeys(t, application, state, 2, 3)
	if state.HitPoints[3] != 30 {
		t.Fatalf("an unaided 9 should miss: foe has %d hp", state.HitPoints[3])
	}
	application.roller = fixedRoller{10}
	attackWithKeys(t, application, state, 3, 2)
	if state.HitPoints[2] != 26 {
		t.Fatalf("an unaided 10 should hit: ally has %d hp", state.HitPoints[2])
	}
}

// 群組 12：祈禱讓對面的豁免 −1（`12FEh` `FE 0E 74 67`）。先施祈禱，再對敵人施定身術：
// 豁免目標值 10，擲 12、定身術 1 個目標 −2 → 10 剛好擋下；祈禱之後 9，定住。
func TestPrayerLowersTheOtherSidesSave(t *testing.T) {
	for _, pray := range []bool{false, true} {
		application, state := effectOnlyBoard(t, 12, gamepack.SpellIDPrayer, gamepack.SpellIDHoldPerson)
		for category := range state.SaveTargets[3] {
			state.SaveTargets[3][category] = 10
		}
		if pray {
			castFromMenuAt(t, application, state, 1, 0, 1)
			if !state.hasEffect(1, gamepack.PrayerAreaEffectCode) {
				t.Fatal("prayer did not attach")
			}
		}
		castFromMenuAt(t, application, state, 1, len(application.spellOptionsFor(application.state.Party[0]))-1, 3)
		if held := state.hasEffect(3, gamepack.HoldPersonEffectCode); held != pray {
			t.Fatalf("pray %v: foe held %v, want %v (status %q)", pray, held, pray, state.Status)
		}
	}
}

// 隱形術掛 19h 在隊友身上：對它出手 −4（entry 25 `0927h`）。擲 13 原本打得中，隱形之後
// 9 打不中；隊友自己出手就現形（`0FCCh`）。
func TestInvisibilityFromTheMenuMakesAttackersMiss(t *testing.T) {
	for _, cast := range []bool{false, true} {
		application, state := effectOnlyBoard(t, 13, 30)
		if cast {
			castFromMenuAt(t, application, state, 1, 0, 2)
		}
		attackWithKeys(t, application, state, 3, 2)
		if hit := state.HitPoints[2] < 30; hit == cast {
			t.Fatalf("invisible %v: the foe hit %v", cast, hit)
		}
		if cast {
			attackWithKeys(t, application, state, 2, 3)
			if state.hasEffect(2, gamepack.InvisibilityEffectCode) {
				t.Fatal("attacking did not drop the invisibility")
			}
		}
	}
}

// 閃現術掛 25h：目標 runtime +3（先攻分數）大於 0 時命中骰寫 FFh，自然 20 也落空
// （entry 35 `0C40h`）。
func TestBlinkFromTheMenuMakesTheFirstAttacksMiss(t *testing.T) {
	for _, cast := range []bool{false, true} {
		application, state := effectOnlyBoard(t, 20, 45)
		if cast {
			castFromMenuAt(t, application, state, 1, 0, 1)
		}
		state.Scores[1] = 3
		attackWithKeys(t, application, state, 3, 1)
		if hit := state.HitPoints[1] < 30; hit == cast {
			t.Fatalf("blinking %v: the foe hit %v", cast, hit)
		}
	}
}

// 護盾術（11h，entry 19 `065Eh`）：群組 11 把被打的那一個 AC 內部值墊到 39h，擲 10 原本剛好
// 打中的變成打不中；群組 6 把魔法飛彈（`DS:6779h` == 0Fh）的傷害寫 0。
func TestShieldFromTheMenuRaisesTheArmourAndStopsMagicMissile(t *testing.T) {
	for _, cast := range []bool{false, true} {
		application, state := effectOnlyBoard(t, 10, 19)
		if cast {
			castFromMenuAt(t, application, state, 1, 0, 1)
			if got := state.hitCheckArmourClass(1); got != gamepack.ShieldArmourClassFloor {
				t.Fatalf("shielded AC is %d, want 57", got)
			}
		}
		attackWithKeys(t, application, state, 3, 1)
		if hit := state.HitPoints[1] < 30; hit == cast {
			t.Fatalf("shield %v: the foe hit %v", cast, hit)
		}
		// 隊友（法師 5 級）對施法者放魔法飛彈：Roll(2, 4) + 2 = 6。
		ally := &application.state.Party[1]
		ally.ClassLevels = make([]uint8, gamepack.ClassThac0ClassCount)
		ally.ClassLevels[gamepack.ClassSlotMagicUser] = 5
		ally.Memorised = make([]uint8, gamepack.MemorisedSpellSlots)
		ally.Memorised[0] = gamepack.SpellIDMagicMissile
		before := state.HitPoints[1]
		castFromMenuAt(t, application, state, 2, 0, 1)
		if lost := before - state.HitPoints[1]; (lost == 0) != cast {
			t.Fatalf("shield %v: magic missile took %d hp (status %q)", cast, lost, state.Status)
		}
	}
}

// 閱讀魔法（編號 18，10h）：用魔杖放出去（+0Bh 為 0，不用掉行動），同一個人接著打開藏字的
// 法師卷軸，`ScrollReadable` 看見 10h 就揭開（spec 144）。沒放閱讀魔法時同一卷揭不開。
func TestReadMagicCastInCombatRevealsTheScroll(t *testing.T) {
	for _, cast := range []bool{false, true} {
		wand := itemOf("WAND", testTypeSword, true, 0)
		wand.Raw[gamepack.AIItemChargesOffset] = 2
		wand.Raw[gamepack.AIItemSpellOffset] = 18
		application, state := newItemMenuApp(t, int(gamepack.ClassSlotFighter), 5,
			wand, scrollItem(0x02, gamepack.SpellIDMagicMissile))
		if cast {
			pressAll(t, application, ebiten.KeyU, ebiten.KeyU)
			if application.castTargeting {
				pressAll(t, application, ebiten.KeyEnter)
			}
			if !state.hasEffect(1, gamepack.ReadMagicEffectCode) || state.Mover != 1 {
				t.Fatalf("read magic from the wand: effects %+v mover %d", state.Effects[1], state.Mover)
			}
		}
		pressAll(t, application, ebiten.KeyU, ebiten.KeyArrowDown, ebiten.KeyU)
		revealed := application.combatItems != nil && application.combatItems.stage == combatItemScroll
		if revealed != cast {
			t.Fatalf("read magic %v: scroll revealed %v", cast, revealed)
		}
	}
}

// 致盲（38，碰觸）：參數表 +2 是 FFh，先擲命中（`09CEh` entry 6）。碰到了、豁免失敗就掛 21h：
// 牠出手 −4（群組 10），被打時 AC −4（群組 11 `0BC9h`）。碰不到（AC 內部值 70）就當豁免成功，
// 規則 1 → "is Unaffected"，什麼都不掛。
func TestCauseBlindnessNeedsATouchAndThenHampersBothWays(t *testing.T) {
	application, state := effectOnlyBoard(t, 19, 38)
	state.ArmorClass[3] = 70
	castFromMenuAt(t, application, state, 1, 0, 3)
	if state.hasEffect(3, gamepack.BlindnessEffectCode) {
		t.Fatal("a missed touch still blinded the foe")
	}

	application, state = effectOnlyBoard(t, 19, 38)
	castFromMenuAt(t, application, state, 1, 0, 3)
	if !state.hasEffect(3, gamepack.BlindnessEffectCode) {
		t.Fatalf("the touch landed but no blindness: %q", state.Status)
	}
	application.roller = fixedRoller{13}
	attackWithKeys(t, application, state, 3, 2)
	if state.HitPoints[2] != 30 {
		t.Fatalf("a blind foe should miss on 13: ally has %d hp", state.HitPoints[2])
	}
	application.roller = fixedRoller{7}
	attackWithKeys(t, application, state, 2, 3)
	if state.HitPoints[3] != 26 {
		t.Fatalf("a blind foe should be hit on 7 (AC −4): foe has %d hp", state.HitPoints[3])
	}
}

// 降咒（44，碰觸）掛 24h：牠出手 −4（群組 10）、豁免 −4（群組 12）。
func TestBestowCurseLowersTheFoesRollsFromTheMenu(t *testing.T) {
	application, state := effectOnlyBoard(t, 19, 44)
	castFromMenuAt(t, application, state, 1, 0, 3)
	if !state.hasEffect(3, gamepack.BestowCurseEffectCode) {
		t.Fatalf("bestow curse did not attach: %q", state.Status)
	}
	application.roller = fixedRoller{13}
	attackWithKeys(t, application, state, 3, 2)
	if state.HitPoints[2] != 30 {
		t.Fatalf("a cursed foe should miss on 13: ally has %d hp", state.HitPoints[2])
	}
	for category := range state.SaveTargets[3] {
		state.SaveTargets[3][category] = 10
	}
	application.roller = fixedRoller{12}
	if application.savedAgainstCategory(state, 3, 0, 0) {
		t.Fatal("a cursed 12 against 10 should fail (12 − 4)")
	}
}

// 緩毒術：`1873h` 先問中毒 37h，沒中毒就整支不做；中毒而且生命值 0 就墊成 1，掛 16h（解不掉）。
func TestSlowPoisonOnlyHelpsThePoisoned(t *testing.T) {
	for _, poisoned := range []bool{false, true} {
		application, state := effectOnlyBoard(t, 19, gamepack.SpellIDSlowPoison)
		state.HitPoints[2] = 0
		if poisoned {
			state.addEffect(2, gamepack.PoisonEffectCode, 0, 1)
		}
		castFromMenuAt(t, application, state, 1, 0, 2)
		if (state.HitPoints[2] == 1) != poisoned || state.hasEffect(2, 0x16) != poisoned {
			t.Fatalf("poisoned %v: hp %d effects %+v", poisoned, state.HitPoints[2], state.Effects[2])
		}
	}
}

// 友誼術：魅力加 Roll(2, 4)（夾在 25），節點 +3 記著原本的 12；三回合到期時 `05E0h` 還原。
func TestFriendsRaisesCharismaUntilItExpires(t *testing.T) {
	application, state := effectOnlyBoard(t, 19, gamepack.SpellIDFriends)
	castFromMenuAt(t, application, state, 1, 0, 1)
	if got := application.state.Party[0].Abilities[gamepack.AbilityCharisma]; got != 16 {
		t.Fatalf("charisma after friends is %d, want 12 + 4", got)
	}
	cast := state.Round
	for state.hasEffect(1, gamepack.FriendsEffectCode) {
		if state.Round > cast+3 {
			t.Fatalf("friends still on after round %d", state.Round)
		}
		endRoundWithKeys(t, application, state)
	}
	if got := application.state.Party[0].Abilities[gamepack.AbilityCharisma]; got != 12 {
		t.Fatalf("charisma after friends expired is %d, want 12", got)
	}
}

// 編號 39h（`2DB7h`）：身上有緩速 2Ah 就用 entry 15 摘掉、整支返回，不掛 27h。
func TestSpeedyItemCuresSlowInsteadOfHasting(t *testing.T) {
	application, state := effectOnlyBoard(t, 19, gamepack.SpellIDGuardedGeneric)
	state.addEffect(1, gamepack.SlowEffectCode, 5, 5)
	castFromMenuAt(t, application, state, 1, 0, 1)
	if state.hasEffect(1, gamepack.SlowEffectCode) || state.hasEffect(1, gamepack.HasteEffectCode) {
		t.Fatalf("39h should cure the slow and stop: %+v", state.Effects[1])
	}
}
