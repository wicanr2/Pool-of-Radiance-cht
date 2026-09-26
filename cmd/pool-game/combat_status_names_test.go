package main

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// 戰鬥狀態列其餘的訊息（issue #110，spec 098〈AI 戰鬥訊息的字串與印法〉的表）：
// 原版每一句只經 overlay-25 entry 20／19 印，名字取記錄 `+0`，沒有一支印名冊編號。
// 全部從 Update() 送鍵，英繁兩語。

// newNamedDuel 是 CASTER（隊員，(5,5)）對 KOBOLD（敵方，foe 那一格）。
func newNamedDuel(t *testing.T, lang language, foe combat.CombatantCell, spells ...uint8) (*app, *tacticalState) {
	t.Helper()
	application, state := newSpellBoard(t,
		spellCasterWith(int(gamepack.ClassSlotMagicUser), 5, spells...),
		combat.CombatantCell{X: 5, Y: 5, FootprintClass: 1}, foe)
	state.rememberRecordName(2, "KOBOLD")
	application.language = lang
	state.Text = application.text
	return application, state
}

// drainCombatNotices 把停拍中的影格空轉過去（每一格送一個沒人接的鍵）。停拍只是等待，
// 以預算量東西的迴圈呼叫它時不把這些影格算進去；上限只為了讓迴圈有終點。
func drainCombatNotices(t *testing.T, application *app) {
	t.Helper()
	for guard := 0; guard < 1<<20 && combatNoticeHolding(application); guard++ {
		if err := press(application, combatNoticeIdleKey); err != nil {
			t.Fatal(err)
		}
	}
	if combatNoticeHolding(application) {
		t.Fatalf("a combat notice never finished: %+v", application.tactical.Notices)
	}
}

// attackNoticeIn 找佇列裡攻擊那一則（overlay-13 entry 4）。
func attackNoticeIn(state *tacticalState) (combatNotice, bool) {
	for _, notice := range state.Notices {
		if notice.Target != "" {
			return notice, true
		}
	}
	return combatNotice{}, false
}

// 一次攻擊：entry 20(攻擊者, "Attacks", 0Ah, 0)、entry 22 在第 0Ch 列印目標名字、
// 下一列起 "Hitting for N points of damage"／"and Misses"，之後等一拍。
// 名字顏色：隊員 0Bh、敵方 0Eh（`1865h`）。
func TestAttackNoticeNamesAttackerAndTarget(t *testing.T) {
	for _, tc := range []struct {
		lang   language
		roll   int
		status string
		detail string
		text   string
	}{
		{languageEnglish, 19, "HIT KOBOLD FOR 4 (HP 26)", "HITTING FOR 4 POINTS OF DAMAGE", "ATTACKS"},
		{languageEnglish, 1, "ATTACK KOBOLD MISSED (D20 1)", "AND MISSES", "ATTACKS"},
		{languageTraditionalChinese, 19, "打中 KOBOLD 造成 4（剩 26 生命力）", "造成 4 點傷害", "出手攻擊"},
		{languageTraditionalChinese, 1, "攻擊 KOBOLD 落空（D20 1）", "沒有打中", "出手攻擊"},
	} {
		application, state := newNamedDuel(t, tc.lang, combat.CombatantCell{X: 6, Y: 5, FootprintClass: 1})
		application.roller = fixedRoller{tc.roll}
		attackWithKeys(t, application, state, 1, 2)
		if state.Status != tc.status {
			t.Fatalf("%v roll %d: status %q, want %q", tc.lang, tc.roll, state.Status, tc.status)
		}
		notice, ok := attackNoticeIn(state)
		want := combatNotice{Name: "CASTER", NameInk: noticeInkParty, Text: tc.text, Row: noticeRowPanel,
			Target: "KOBOLD", TargetInk: noticeInkFoe, Detail: tc.detail}
		if !ok || notice != want {
			t.Fatalf("%v roll %d: attack notice %+v (found %v), want %+v", tc.lang, tc.roll, notice, ok, want)
		}
	}
}

// 攻擊那一拍：遊戲速度 4 是 54 影格，期間 ENTER 不作用（原版的 Delay 不讀鍵）。
func TestAttackNoticeHoldsForOneBeat(t *testing.T) {
	application, state := newNamedDuel(t, languageEnglish, combat.CombatantCell{X: 6, Y: 5, FootprintClass: 1})
	application.roller = fixedRoller{19}
	application.gameSpeed = 4
	attackWithKeys(t, application, state, 1, 2)
	notice, ok := attackNoticeIn(state)
	if !ok || notice.Ticks != 54 {
		t.Fatalf("attack notice %+v (found %v), want it holding 54 frames", notice, ok)
	}
	for frame := 0; frame < 54; frame++ {
		if !combatNoticeHolding(application) {
			t.Fatalf("frame %d: the beat ended early", frame)
		}
		pressAll(t, application, ebiten.KeyEnter)
	}
	if combatNoticeHolding(application) {
		t.Fatalf("still holding after 54 frames: %+v", state.Notices)
	}
}

// 瞄準列上沒有 Target 的那幾句是 remake 自己的說明（原版只是選單上少一項，overlay-13
// `2AEFh..2B8Bh`），帶名字不帶名冊編號。
func TestAimOutOfRangeNamesTheTarget(t *testing.T) {
	for _, tc := range []struct {
		lang   language
		prefix string
	}{
		{languageEnglish, "KOBOLD is "},
		{languageTraditionalChinese, "KOBOLD 在 "},
	} {
		application, state := newNamedDuel(t, tc.lang, combat.CombatantCell{X: 9, Y: 5, FootprintClass: 1})
		attackWithKeys(t, application, state, 1, 2)
		if !strings.HasPrefix(state.Status, tc.prefix) || state.Mover != 1 {
			t.Fatalf("%v: status %q mover %d, want %q… and the turn kept", tc.lang, state.Status, state.Mover, tc.prefix)
		}
	}
}

// 包紮：overlay-08 `0FE9h` 的 entry 20(被包紮的那一位, "is bandaged", 0Ah, 1)。
func TestBandageNoticeNamesTheMember(t *testing.T) {
	for _, tc := range []struct {
		lang language
		want string
		text string
	}{
		{languageEnglish, "ALLY IS BANDAGED", "IS BANDAGED"},
		{languageTraditionalChinese, "ALLY 已包紮", "已包紮"},
	} {
		application, state := sideEffectBoard(t, int(gamepack.ClassSlotCleric), 1, gamepack.SpellIDBless, 1)
		application.language = tc.lang
		state.Text = application.text
		state.States[2] = combat.DyingState
		state.Roster[2].FootprintClass = 0
		pressAll(t, application, ebiten.KeyB)
		if state.Status != tc.want {
			t.Fatalf("%v: status %q, want %q", tc.lang, state.Status, tc.want)
		}
		want := combatNotice{Name: "ALLY", NameInk: noticeInkOut, Text: tc.text, Row: noticeRowPanel}
		if len(state.Notices) != 1 || state.Notices[0] != want {
			t.Fatalf("%v: notices %+v, want %+v", tc.lang, state.Notices, want)
		}
	}
}

// 玩家施法同樣經 `0D23h`（overlay-08 `0410h` 推 `[bp+0Ah]` = 1）：挑目標之前印名字、
// "Casts a Spell" 與第 23 列的 "Spell:" 加法名，停一拍；停拍中瞄準按 ENTER 不作用。
// 施法清單的法名英文照 START.EXE 名稱表（spellLabel）。
func TestPlayerCastAnnouncesBeforeAiming(t *testing.T) {
	for _, tc := range []struct {
		lang   language
		label  string
		text   string
		bottom string
	}{
		{languageEnglish, "Magic Missile", "CASTS A SPELL", "SPELL:MAGIC MISSILE"},
		{languageTraditionalChinese, "魔法飛彈", "施放法術", "法術：魔法飛彈"},
	} {
		application, state := newNamedDuel(t, tc.lang, combat.CombatantCell{X: 7, Y: 5, FootprintClass: 1},
			gamepack.SpellIDMagicMissile)
		application.gameSpeed = 4
		// 施法清單的目錄平常在打開法術頁時才建（spellOptionsFor 沒有目錄就印編號）。
		application.spellLabel(gamepack.SpellIDMagicMissile)
		pressAll(t, application, ebiten.KeyC)
		if len(application.castOptions) != 1 || application.castOptions[0].Label != tc.label {
			t.Fatalf("%v: cast menu %+v, want %q", tc.lang, application.castOptions, tc.label)
		}
		pressAll(t, application, ebiten.KeyEnter)
		want := combatNotice{Name: "CASTER", NameInk: noticeInkParty, Text: tc.text, Row: noticeRowPanel,
			Bottom: tc.bottom, Ticks: 54}
		if len(state.Notices) != 1 || state.Notices[0] != want {
			t.Fatalf("%v: notices %+v, want %+v", tc.lang, state.Notices, want)
		}
		hp := state.HitPoints[2]
		for frame := 0; frame < 54; frame++ {
			pressAll(t, application, ebiten.KeyEnter)
		}
		if state.HitPoints[2] != hp || !application.castTargeting {
			t.Fatalf("%v: the spell went off during the beat: hp %d → %d", tc.lang, hp, state.HitPoints[2])
		}
		pressAll(t, application, ebiten.KeyEnter)
		if state.HitPoints[2] >= hp {
			t.Fatalf("%v: ENTER after the beat did not cast: hp %d → %d (%q)", tc.lang, hp, state.HitPoints[2], state.Status)
		}
	}
}

// 玩家的 Abort Spell：`0EE7h` 走 entry 19，第 24 列固定字串、停一拍（與 AI 同一支）。
func TestPlayerAbortSpellHoldsOnTheFooter(t *testing.T) {
	for _, tc := range []struct {
		lang language
		want string
	}{
		{languageEnglish, "SPELL ABORTED"},
		{languageTraditionalChinese, "施法中止"},
	} {
		application, state := newNamedDuel(t, tc.lang, combat.CombatantCell{X: 7, Y: 5, FootprintClass: 1},
			gamepack.SpellIDMagicMissile)
		application.gameSpeed = 4
		pressAll(t, application, ebiten.KeyC, ebiten.KeyEnter)
		for combatNoticeHolding(application) {
			pressAll(t, application, combatNoticeIdleKey)
		}
		pressAll(t, application, ebiten.KeyEscape, ebiten.KeyY)
		last := state.Notices[len(state.Notices)-1]
		if last.Footer != tc.want || last.Ticks != 54 || state.Status != tc.want {
			t.Fatalf("%v: last notice %+v status %q, want %q holding 54 frames", tc.lang, last, state.Status, tc.want)
		}
	}
}

// 重複挑同一個：`2363h` 只有玩家（`[bp+0Ah]` 為 0）印 "Already been targeted"，走 entry 19
// （`2378h`），不帶名字。
func TestAlreadyTargetedIsAFixedFooter(t *testing.T) {
	for _, tc := range []struct {
		lang language
		want string
	}{
		{languageEnglish, "ALREADY BEEN TARGETED"},
		{languageTraditionalChinese, "已經選過了"},
	} {
		application, state := newSpellBoard(t,
			spellCasterWith(int(gamepack.ClassSlotCleric), 3, gamepack.SpellIDHoldPerson),
			combat.CombatantCell{X: 5, Y: 5, FootprintClass: 1},
			combat.CombatantCell{X: 7, Y: 5, FootprintClass: 1},
			combat.CombatantCell{X: 7, Y: 7, FootprintClass: 1})
		application.language = tc.lang
		state.Text = application.text
		pressAll(t, application, ebiten.KeyC, ebiten.KeyEnter)
		if state.Casting.Pending[1] != 0 {
			pressAll(t, application, ebiten.KeyEnter)
		}
		first := application.castTargets[application.castTargetCursor]
		pressAll(t, application, ebiten.KeyEnter)
		if !application.castTargeting || application.castTargets[application.castTargetCursor] != first {
			t.Fatalf("%v: hold person did not ask for a second target on %d", tc.lang, first)
		}
		pressAll(t, application, ebiten.KeyEnter)
		last := state.Notices[len(state.Notices)-1]
		if last.Footer != tc.want || state.Status != tc.want {
			t.Fatalf("%v: last notice %+v status %q, want footer %q", tc.lang, last, state.Status, tc.want)
		}
	}
}

// 用物品：overlay-19 `1B2Dh..1BB8h`，名字與 "uses an item"、第 23 列 "Item:" 加物品名，
// 物品名印完才等一拍（`1BB3h`）；用物品不印 "Casts a Spell"（`DS:6CB3h` 非 0）。
func TestItemUseHoldsAfterTheItemLine(t *testing.T) {
	application, state := newQuickWandApp(t, 2, 0, true)
	application.state.Party[0].Quick = false
	state.AIDriven[1] = false
	application.combatCommands = turnUseSegments
	application.gameSpeed = 4
	pressAll(t, application, ebiten.KeyU, ebiten.KeyU)
	want := combatNotice{Name: "A", NameInk: noticeInkParty, Text: "USES AN ITEM", Row: noticeRowPanel,
		Bottom: "ITEM:WAND", Ticks: 54}
	if len(state.Notices) != 1 || state.Notices[0] != want {
		t.Fatalf("notices %+v, want only %+v", state.Notices, want)
	}
}

// "is turned"：overlay-25 entry 26 旗標 1，停的是閃光動畫——(遊戲速度 + 1) × 4 格 × 70 ms。
func TestTurnedUndeadHoldsForTheSparkle(t *testing.T) {
	application, state := newQuickTurnApp(t, 1, 1, &turnScript{d20: 15})
	application.gameSpeed = 4
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	ticks := (4 + 1) * 4 * 70 * 60 / 1000
	for _, notice := range state.Notices {
		if notice.Name == "SKELETON" && notice.Text == "IS TURNED" && notice.Ticks == ticks &&
			notice.NameInk == noticeInkFoe {
			return
		}
	}
	t.Fatalf("notices %+v, want SKELETON IS TURNED holding %d frames", state.Notices, ticks)
}

// 名字的三種顏色（`1865h`）：離場 0Ch、敵方 0Eh、其餘 0Bh；畫的時候查主題色盤。
func TestNoticeNameInkFollowsTheRecord(t *testing.T) {
	state := newRoundState(3)
	state.Friendly[2] = false
	state.Friendly[3] = false
	state.Roster[3].FootprintClass = 0
	for index, want := range map[uint8]uint8{1: noticeInkParty, 2: noticeInkFoe, 3: noticeInkOut} {
		if got := state.nameInk(index); got != want {
			t.Errorf("index %d: ink %#x, want %#x", index, got, want)
		}
	}
}
