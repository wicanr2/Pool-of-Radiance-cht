package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// 效果掛上、解除、老化時逐人印的那一行（issue #87，spec 163）。全部從 Update() 送鍵。

// effectAttachMessages 的英文要與原版處理常式推給 `08BCh` 的那一段 Pascal 字串逐格相同
// （`ReadDOSSpellDispatchTable` 讀 overlay-22 的派發表與字串）。
func TestEffectAttachMessagesMatchTheDispatchTable(t *testing.T) {
	table, err := gamepack.ReadDOSSpellDispatchTable(filepath.Join("..", "..", "Pool of Radiance (1988).zip"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	messages := map[uint8]string{}
	for _, entry := range table {
		messages[uint8(entry.SpellID)] = entry.Message
	}
	for spell, id := range effectAttachMessages {
		want := strings.ToUpper(messages[spell])
		if want == "" {
			t.Errorf("spell %d has no handler message in the dispatch table", spell)
			continue
		}
		if got := packMessage(id, "en"); got != want {
			t.Errorf("spell %d: English %q, the handler pushes %q", spell, got, want)
		}
	}
}

// waitCombatNotices 等佇列裡每一則停拍都停完。drainCombatNotices 只看第一則，而一次施法
// 會排好幾則，不停拍的那一則（速度 0 的 "Casts a Spell"）可能排在閃光那一則前面；
// holdCombatNotice 會先丟掉它、停在後面那一則，這時送出的鍵會被吃掉。按的是閒置鍵，只是等待。
func waitCombatNotices(t *testing.T, application *app) {
	t.Helper()
	holding := func() bool {
		state := application.tactical
		if state == nil || state.Finished {
			return false
		}
		for _, notice := range state.Notices {
			if notice.Ticks > 0 {
				return true
			}
		}
		return false
	}
	for guard := 0; guard < 1<<20 && holding(); guard++ {
		pressAll(t, application, combatNoticeIdleKey)
	}
	if holding() {
		t.Fatalf("a combat notice never finished: %+v", application.tactical.Notices)
	}
}

// drainEffectBeats 等戰鬥外的逐人訊息停完（停拍中不讀鍵，按的是閒置鍵）。
func drainEffectBeats(t *testing.T, application *app) {
	t.Helper()
	for guard := 0; guard < 1<<20 && application.effectBeats.target != nil; guard++ {
		pressAll(t, application, combatNoticeIdleKey)
	}
	if application.effectBeats.target != nil {
		t.Fatalf("the effect beats never finished: %+v", application.effectBeats)
	}
}

// noticeTexts 是佇列裡每一則的「名字 句子」，依排入的先後。
func noticeTexts(state *tacticalState) []string {
	var lines []string
	for _, notice := range state.Notices {
		if notice.Name != "" || notice.Text != "" {
			lines = append(lines, notice.Name+" "+notice.Text)
		}
	}
	return lines
}

// indexOfLine 是 want 在 lines 裡第一次出現的位置，沒有回 -1。
func indexOfLine(lines []string, want string) int {
	for index, line := range lines {
		if line == want {
			return index
		}
	}
	return -1
}

// 祝福術：entry 20 `171Fh` 對每一個掛上的人經 entry 26 旗標 1 印「名字 is Blessed」，停的長度是
// 閃光動畫（遊戲速度 0：一輪四格、每格 70 ms）。按一下閒置鍵，畫面上那一則就是它。
func TestBlessNamesEachBlessedTarget(t *testing.T) {
	for _, tc := range []struct {
		language language
		want     string
	}{
		{languageEnglish, "ALLY IS BLESSED"},
		{languageTraditionalChinese, "ALLY 受到祝福"},
	} {
		application, state := sideEffectBoard(t, int(gamepack.ClassSlotCleric), 1, gamepack.SpellIDBless, 9)
		application.language = tc.language
		state.Text = application.text
		castAndAimAtIndex(t, application, state, 2)
		lines := noticeTexts(state)
		at := indexOfLine(lines, tc.want)
		if at < 0 {
			t.Fatalf("%v: no %q in the notices %q", tc.language, tc.want, lines)
		}
		if indexOfLine(lines, "FOE "+application.text(msgNoticeBlessed)) >= 0 {
			t.Fatalf("%v: the foe was named as blessed: %q", tc.language, lines)
		}
		pressAll(t, application, combatNoticeIdleKey)
		shown, ok := state.shownNotice()
		if !ok || shown.Name+" "+shown.Text != tc.want {
			t.Fatalf("%v: the screen shows %+v, want %q", tc.language, shown, tc.want)
		}
		if want := 4 * turnedSparkleMilliseconds * 60 / 1000; shown.Ticks < want-1 || shown.Ticks > want {
			t.Errorf("%v: the blessed line holds %d ticks, want the sparkle %d", tc.language, shown.Ticks, want)
		}
		waitCombatNotices(t, application)
		if !state.hasEffect(2, gamepack.BlessEffectCode) {
			t.Fatalf("%v: the ally carries no bless node", tc.language)
		}
	}
}

// 急速術：`08BCh` 先印「名字 is Hasted」，接著 `2724h` 的群組 18 讓那個人老一歲、印「名字 ages」
// （overlay-12 `0C98h`，停一拍）。之後回合初始化再問群組 18 不再老、也不再印。
func TestHasteNamesTheTargetAndItsAging(t *testing.T) {
	for _, tc := range []struct {
		language     language
		hasted, aged string
	}{
		{languageEnglish, "ALLY IS HASTED", "ALLY AGES"},
		{languageTraditionalChinese, "ALLY 加快了速度", "ALLY 老了一歲"},
	} {
		application, state := sideEffectBoard(t, int(gamepack.ClassSlotMagicUser), 5, gamepack.SpellIDHaste, 19)
		application.language = tc.language
		application.gameSpeed = 2
		state.Text = application.text
		pressAll(t, application, ebiten.KeyC, ebiten.KeyEnter)
		waitCombatNotices(t, application) // "Begins Casting"／"Casts a Spell" 那一拍
		if state.Casting.Pending[1] != 0 {
			pressAll(t, application, ebiten.KeyEnter)
			waitCombatNotices(t, application)
		}
		if !application.castTargeting || len(application.castTargets) == 0 {
			t.Fatalf("%v: haste did not open aiming: %q", tc.language, state.Status)
		}
		for guard := 0; guard <= len(application.castTargets) &&
			application.castTargets[application.castTargetCursor] != 2; guard++ {
			pressAll(t, application, ebiten.KeyN)
		}
		pressAll(t, application, ebiten.KeyEnter)
		lines := noticeTexts(state)
		hasted, aged := indexOfLine(lines, tc.hasted), indexOfLine(lines, tc.aged)
		if hasted < 0 || aged < 0 || aged < hasted {
			t.Fatalf("%v: want %q then %q, got %q", tc.language, tc.hasted, tc.aged, lines)
		}
		for _, notice := range state.Notices {
			if notice.Name+" "+notice.Text == tc.aged && notice.Ticks != application.speedDelayTicks() {
				t.Errorf("%v: the aging line holds %d ticks, want one beat %d",
					tc.language, notice.Ticks, application.speedDelayTicks())
			}
		}
		if got := application.state.Party[1].Age; got != 21 {
			t.Fatalf("%v: the hasted ally is %d, want 21", tc.language, got)
		}
		waitCombatNotices(t, application)
		endRoundWithKeys(t, application, state)
		if indexOfLine(noticeTexts(state), tc.aged) >= 0 {
			t.Fatalf("%v: the ally aged again at the round start: %q", tc.language, noticeTexts(state))
		}
	}
}

// 急速遇上緩速：`2724h` 以 entry 15 摘掉緩速，印「名字 is Cured」，這一次不掛、不老、不印 is Hasted。
func TestHasteOnTheSlowedNamesTheCure(t *testing.T) {
	application, state := sideEffectBoard(t, int(gamepack.ClassSlotMagicUser), 5, gamepack.SpellIDHaste, 19)
	state.addEffect(2, gamepack.SlowEffectCode, 5, 5)
	castAndAimAtIndex(t, application, state, 2)
	lines := noticeTexts(state)
	if indexOfLine(lines, "ALLY IS CURED") < 0 {
		t.Fatalf("no cure line: %q", lines)
	}
	if indexOfLine(lines, "ALLY IS HASTED") >= 0 || indexOfLine(lines, "ALLY AGES") >= 0 {
		t.Fatalf("a cured ally was still hasted: %q", lines)
	}
}

// 解病術在戰場上：解病鏈 `225Bh` 問 entry 15，身上有病就印「名字 is Cured」、停一拍。
func TestCureDiseaseOnTheBoardNamesTheCure(t *testing.T) {
	for _, tc := range []struct {
		language language
		want     string
	}{
		{languageEnglish, "CASTER IS CURED"},
		{languageTraditionalChinese, "CASTER 解除了"},
	} {
		application, state := deathBoard(t, 4, gamepack.SpellIDCureDisease)
		application.language = tc.language
		application.gameSpeed = 2
		state.Text = application.text
		disease := gamepack.NewEffectNode(gamepack.DiseaseEffectCode, 0, gamepack.EffectUndispellable, false)
		state.Effects[1] = gamepack.EffectList{disease}
		application.state.Party[0].Effects = storedEffects(gamepack.EffectList{disease})
		castFromMenuAt(t, application, state, 1, 0, 1)
		if state.hasEffect(1, gamepack.DiseaseEffectCode) {
			t.Fatalf("%v: the disease is still on: %+v", tc.language, state.Effects[1])
		}
		lines := noticeTexts(state)
		if indexOfLine(lines, tc.want) < 0 {
			t.Fatalf("%v: no %q among the notices %q", tc.language, tc.want, lines)
		}
		for _, notice := range state.Notices {
			if notice.Name+" "+notice.Text == tc.want && notice.Ticks != application.speedDelayTicks() {
				t.Errorf("%v: the cure line holds %d ticks, want one beat", tc.language, notice.Ticks)
			}
		}
		waitCombatNotices(t, application)
	}
}

// 編號 57（`2DB7h`，物品 +3Dh 50h 換算成 39h）：身上有緩速就經 entry 15 印「名字 is Cured」、
// 整支返回；沒有才走 `08BCh`，掛 27h、印「名字 is Speedy」。
func TestSpeedyItemNamesTheCureOrTheSpeed(t *testing.T) {
	for _, slowed := range []bool{false, true} {
		application, state := effectOnlyBoard(t, 19, gamepack.SpellIDGuardedGeneric)
		if slowed {
			state.addEffect(1, gamepack.SlowEffectCode, 5, 5)
		}
		castFromMenuAt(t, application, state, 1, 0, 1)
		lines := noticeTexts(state)
		cured, speedy := indexOfLine(lines, "CASTER IS CURED"), indexOfLine(lines, "CASTER IS SPEEDY")
		if slowed && (cured < 0 || speedy >= 0) {
			t.Fatalf("slowed: want only the cure line, got %q", lines)
		}
		if !slowed && (speedy < 0 || cured >= 0) {
			t.Fatalf("not slowed: want only the speedy line, got %q", lines)
		}
		if !slowed && !state.hasEffect(1, gamepack.HasteEffectCode) {
			t.Fatalf("39h attached no 27h: %+v", state.Effects[1])
		}
	}
}

// 營地的祝福與急速：戰鬥外 entry 26 走 `21C1h`，在下方訊息框印「名字 那一句」、停一拍；
// 停拍中不讀鍵，逐行放完才回到原本那一句。
func TestCampHasteShowsEachLineForABeat(t *testing.T) {
	for _, tc := range []struct {
		language language
		want     []string
	}{
		{languageEnglish, []string{"B IS HASTED", "B AGES"}},
		{languageTraditionalChinese, []string{"B 加快了速度", "B 老了一歲"}},
	} {
		application := campCastApp(t, campCaster("A", 5, gamepack.SpellIDHaste), campCaster("B", 1))
		application.language = tc.language
		application.gameSpeed = 2
		pressAll(t, application, ebiten.KeyC, ebiten.KeyEnter)
		for guard := 0; guard < 8 && application.fieldCastOptions[application.fieldCastCursor].ID != gamepack.SpellIDHaste; guard++ {
			pressAll(t, application, ebiten.KeyArrowDown)
		}
		pressAll(t, application, ebiten.KeyEnter)
		if application.fieldCastStage == fieldCastPickTarget {
			pressAll(t, application, ebiten.KeyArrowDown, ebiten.KeyEnter)
		}
		var seen []string
		for guard := 0; guard < 1<<12 && application.effectBeats.target != nil; guard++ {
			if len(seen) == 0 || seen[len(seen)-1] != application.fieldCastMessage {
				seen = append(seen, application.fieldCastMessage)
			}
			// 停拍中按 ESC 不作用（原版的 Delay 不讀鍵）。
			pressAll(t, application, ebiten.KeyEscape)
			if !application.fieldCastOpen {
				t.Fatalf("%v: ESC closed the page during a beat", tc.language)
			}
		}
		for _, line := range tc.want {
			if indexOfLine(seen, line) < 0 {
				t.Fatalf("%v: no %q among the shown lines %q", tc.language, line, seen)
			}
		}
		if indexOfLine(seen, tc.want[0]) > indexOfLine(seen, tc.want[1]) {
			t.Fatalf("%v: the aging line came first: %q", tc.language, seen)
		}
		if application.state.Party[1].Age != 21 {
			t.Fatalf("%v: B is %d, want 21", tc.language, application.state.Party[1].Age)
		}
	}
}
