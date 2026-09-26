package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// combatNoticeIdleKey 是停拍中空轉一個影格時送的鍵：遊戲裡沒有任何地方接它。
const combatNoticeIdleKey = ebiten.KeyPause

// combatNoticeHolding 說這一影格是不是停拍中（tacticalInput 不讀鍵）。以預算或
// 駕駛計數量東西的測試拿它把等待的影格扣掉。
func combatNoticeHolding(application *app) bool {
	state := application.tactical
	return state != nil && !state.Finished && len(state.Notices) > 0 && state.Notices[0].Ticks > 0
}

// AI 戰鬥訊息帶名字、施法前的 "Casts a Spell" 與停拍（issue #104，spec 098
// 〈AI 戰鬥訊息的字串與印法〉）。全部從 Update() 送鍵，由 tacticalInput 自己分派 AI。

// 施法前那一句：`0D23h` 經 `32D3h` 印名字、"Casts a Spell"，第 23 列 "Spell:" 加法名。
// 英文的法名照 START.EXE 的名稱表（spec 068），中文用說明書譯名。
func TestFoeCastAnnouncesCasterAndSpellByName(t *testing.T) {
	for _, tc := range []struct {
		language language
		log      func(application *app) string
		bottom   func(application *app) string
		text     string
	}{
		{languageEnglish,
			func(*app) string { return "LEVEL 6 MU CASTS A SPELL SPELL:MAGIC MISSILE" },
			func(*app) string { return "SPELL:MAGIC MISSILE" },
			"CASTS A SPELL"},
		{languageTraditionalChinese,
			func(application *app) string {
				return "LEVEL 6 MU 施放法術 法術：" + application.spellLabel(gamepack.SpellIDMagicMissile)
			},
			func(application *app) string {
				return "法術：" + application.spellLabel(gamepack.SpellIDMagicMissile)
			},
			"施放法術"},
	} {
		application, state := newFoeCastApp(t)
		application.language = tc.language
		state.Text = application.text
		giveFoeSpells(state, gamepack.SpellIDMagicMissile)
		if err := press(application, ebiten.KeyEnter); err != nil {
			t.Fatal(err)
		}
		if state.FoeLog != tc.log(application) {
			t.Fatalf("%v: log %q, want %q", tc.language, state.FoeLog, tc.log(application))
		}
		// 遊戲速度 0 不停拍：訊息排進佇列，下一個影格就丟掉。
		if len(state.Notices) != 1 {
			t.Fatalf("%v: notices %+v, want the one cast notice", tc.language, state.Notices)
		}
		notice := state.Notices[0]
		if notice.Name != "LEVEL 6 MU" || notice.Text != tc.text || notice.Row != noticeRowPanel ||
			notice.Bottom != tc.bottom(application) || notice.Ticks != 0 {
			t.Fatalf("%v: notice %+v", tc.language, notice)
		}
	}
}

// 停拍：遊戲速度 × 225 ms（overlay-37 entry 13）。停拍期間 tacticalInput 什麼都不做
// ——輪到隊員時按 ENTER 也不結束回合——停完那一個影格才照常處理按鍵。
func TestFoeCastHoldsForOneBeat(t *testing.T) {
	application, state := newFoeCastApp(t)
	application.gameSpeed = 4
	giveFoeSpells(state, gamepack.SpellIDMagicMissile)
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	beat := application.speedDelayTicks()
	if beat != 54 || len(state.Notices) != 1 || state.Notices[0].Ticks != beat {
		t.Fatalf("beat %d notices %+v, want one notice holding 54 frames", beat, state.Notices)
	}
	if state.Mover != 1 {
		t.Fatalf("after the foe cast the party member should be up, mover %d", state.Mover)
	}
	for frame := 0; frame < beat; frame++ {
		if err := press(application, ebiten.KeyEnter); err != nil {
			t.Fatal(err)
		}
		if state.Mover != 1 || len(state.Notices) != 1 {
			t.Fatalf("frame %d: the beat was cut short: mover %d notices %+v", frame, state.Mover, state.Notices)
		}
	}
	if state.Notices[0].Ticks != 0 {
		t.Fatalf("ticks left after the beat: %d", state.Notices[0].Ticks)
	}
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if len(state.Notices) != 0 || state.Mover == 1 {
		t.Fatalf("ENTER after the beat did not end the turn: mover %d notices %+v", state.Mover, state.Notices)
	}
}

// 施法中止：先印 "Casts a Spell" 那一句（名字在這裡），再以 entry 19 在第 24 列印
// 固定字串 "Spell Aborted"。
func TestFoeAbortedCastStillNamesTheCaster(t *testing.T) {
	application, state := newFoeCastApp(t)
	giveFoeHoldPerson(state)
	state.addEffect(1, gamepack.HoldPersonEffectCode, 10, 7)
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if len(state.Notices) != 2 || state.Notices[0].Name != "7TH LVL CLERIC" ||
		state.Notices[0].Bottom != "SPELL:HOLD PERSON" || state.Notices[1].Footer != "SPELL ABORTED" {
		t.Fatalf("notices %+v, want the cast notice then SPELL ABORTED on row 24", state.Notices)
	}
}

// 開始施法（overlay-13 `24E7h`）：entry 20(記錄, "Begins Casting", 0Ah, 1)，不帶法名。
func TestFoeBeginsCastingByName(t *testing.T) {
	for _, tc := range []struct {
		language language
		want     string
	}{
		{languageEnglish, "LEVEL 6 MU BEGINS CASTING"},
		{languageTraditionalChinese, "LEVEL 6 MU 開始施法"},
	} {
		application, state := newFoeCastApp(t)
		application.language = tc.language
		state.Text = application.text
		giveFoeSpells(state, gamepack.SpellIDFireball)
		state.Scores[1], state.Scores[2] = 3, 5
		if err := press(application, ebiten.KeyEnter); err != nil {
			t.Fatal(err)
		}
		if state.FoeLog != tc.want {
			t.Fatalf("%v: log %q, want %q", tc.language, state.FoeLog, tc.want)
		}
	}
}

// 逃跑：`Got Away` 走 overlay-24 entry 11 → entry 20，帶名字；`Escape is blocked`
// 走 entry 19，不帶名字。記錄那一行的名字取記錄 `+0`，不再是名冊編號。
func TestFleeMessagesNameTheFoe(t *testing.T) {
	for _, tc := range []struct {
		language language
		roll     int
		want     string
		notice   combatNotice
	}{
		{languageEnglish, 1, "SKELETON FLED 1 STEPS. SKELETON GOT AWAY",
			combatNotice{Name: "SKELETON", Text: "GOT AWAY", Row: noticeRowPanel}},
		{languageEnglish, 2, "SKELETON FLED 1 STEPS. ESCAPE IS BLOCKED",
			combatNotice{Footer: "ESCAPE IS BLOCKED"}},
		{languageTraditionalChinese, 1, "SKELETON 逃了 1 步。 SKELETON 逃走了",
			combatNotice{Name: "SKELETON", Text: "逃走了", Row: noticeRowPanel}},
		{languageTraditionalChinese, 2, "SKELETON 逃了 1 步。 退路被擋住了",
			combatNotice{Footer: "退路被擋住了"}},
	} {
		roller := &sequenceRoller{values: []int{1, 1, 1, 1, 1, 1, 1, tc.roll, 1, 1, 1}}
		application, state := newFleeApp(t, 1, 1, 0, 0, roller)
		application.language = tc.language
		state.Text = application.text
		state.rememberRecordName(2, "SKELETON")
		state.Undead.Turned = map[int]bool{2: true}
		if err := press(application, ebiten.KeyEnter); err != nil {
			t.Fatal(err)
		}
		if state.FoeLog != tc.want {
			t.Fatalf("%v roll %d: log %q, want %q", tc.language, tc.roll, state.FoeLog, tc.want)
		}
		last := state.Notices[len(state.Notices)-1]
		if last != tc.notice {
			t.Fatalf("%v roll %d: last notice %+v, want %+v", tc.language, tc.roll, last, tc.notice)
		}
	}
}
