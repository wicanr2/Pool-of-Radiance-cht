package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"strings"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/journal"
)

func newTestJournal(t *testing.T) *journalState {
	t.Helper()
	corpus, err := journal.TraditionalChinese()
	if err != nil {
		t.Fatal(err)
	}
	state := newJournalState(corpus)
	state.reflow()
	return state
}

// 開起來停在第五章第 1 條，因為遊戲最常引用的就是線索報導。
func TestJournalOpensOnTheFirstClue(t *testing.T) {
	state := newTestJournal(t)
	entry, ok := state.current()
	if !ok {
		t.Fatal("no current entry")
	}
	if entry.Kind != journal.Clue || entry.ID != "1" {
		t.Fatalf("opened on %s %s", entry.Kind, entry.ID)
	}
	if len(state.rendered) == 0 {
		t.Fatal("the entry produced no lines")
	}
}

// 換章之後編號各自記著，換回來要回到原來那一條——玩家在兩章之間來回查時，
// 每次都被丟回第 1 條會很難用。
func TestJournalRemembersThePositionOfEachChapter(t *testing.T) {
	state := newTestJournal(t)
	state.selectEntry(5)
	state.selectKind(1)
	if kind := state.currentKind(); kind != journal.Rumour {
		t.Fatalf("TAB moved to %s", kind)
	}
	state.selectKind(-1)
	entry, _ := state.current()
	if entry.ID != "6" {
		t.Fatalf("returned to clue %s, want 6", entry.ID)
	}
}

// 條目在頭尾之間繞回去，不會走到範圍外。
func TestJournalEntrySelectionWraps(t *testing.T) {
	state := newTestJournal(t)
	state.selectEntry(-1)
	entry, _ := state.current()
	if entry.ID != "58" {
		t.Fatalf("stepping back from the first clue reached %s, want 58", entry.ID)
	}
	state.selectEntry(1)
	entry, _ = state.current()
	if entry.ID != "1" {
		t.Fatalf("stepping forward from the last clue reached %s, want 1", entry.ID)
	}
}

// 打編號跳條目，是這個畫面存在的理由：遊戲說「線索報導 46」，玩家就打 46。
func TestJournalJumpsToAQuotedNumber(t *testing.T) {
	state := newTestJournal(t)
	if !state.jump("46") {
		t.Fatal("clue 46 was not found")
	}
	entry, _ := state.current()
	if entry.ID != "46" {
		t.Fatalf("jumped to %s", entry.ID)
	}
	if state.jump("999") {
		t.Fatal("a nonexistent clue reported success")
	}
	if entry, _ := state.current(); entry.ID != "46" {
		t.Fatalf("a failed jump moved the cursor to %s", entry.ID)
	}
}

// 公告用羅馬數字當編號，跳轉要吃得下。
func TestJournalJumpsToARomanProclamation(t *testing.T) {
	state := newTestJournal(t)
	state.selectKind(2)
	if kind := state.currentKind(); kind != journal.Proclamation {
		t.Fatalf("two TABs reached %s", kind)
	}
	if !state.jump("CXXXIV") {
		t.Fatal("proclamation CXXXIV was not found")
	}
	entry, _ := state.current()
	if entry.ID != "CXXXIV" {
		t.Fatalf("jumped to %s", entry.ID)
	}
}

// 捲動夾在範圍內，且長條目真的捲得動。
func TestJournalScrollStaysInRange(t *testing.T) {
	state := newTestJournal(t)
	state.jump("37") // 最長的一條：湯姆斯的地圖集。
	if len(state.rendered) <= journalLineCount {
		t.Fatalf("clue 37 rendered %d lines, expected more than one screen", len(state.rendered))
	}
	state.scrollBy(-5)
	if state.scroll != 0 {
		t.Fatalf("scrolled above the top to %d", state.scroll)
	}
	state.scrollBy(10000)
	if want := len(state.rendered) - journalLineCount; state.scroll != want {
		t.Fatalf("scrolled to %d, want the last page at %d", state.scroll, want)
	}
	state.selectEntry(1)
	if state.scroll != 0 {
		t.Fatalf("moving to the next entry kept the scroll at %d", state.scroll)
	}
}

// 每一行都要排得進畫面寬度，否則右邊會被切掉而看不出是漏字還是版面錯。
// 上限是欄數加上收尾標點的溢位額度：64+2 個半形格 ＝ 528 px，內文從 x=48
// 起算，右界 576，還在框線（x=608）之內。
func TestJournalLinesFitTheColumnBudget(t *testing.T) {
	state := newTestJournal(t)
	for _, kind := range journal.Kinds {
		for index := range state.corpus.Entries(kind) {
			state.kind = indexOfKind(kind)
			state.entry[kind] = index
			state.reflow()
			for _, line := range state.rendered {
				width := 0
				for _, r := range line {
					width += runeWidth(r)
				}
				if width > journalColumns+closingPunctuationSlack {
					t.Fatalf("%s %d produced a %d-column line: %q",
						kind, index+1, width, line)
				}
			}
		}
	}
}

func indexOfKind(kind journal.Kind) int {
	for index, candidate := range journal.Kinds {
		if candidate == kind {
			return index
		}
	}
	return 0
}

// 英文模式不開手冊：內建字型沒有漢字，畫出來是一片空白。
func TestJournalStaysClosedInEnglish(t *testing.T) {
	a := &app{language: languageEnglish}
	if err := a.openJournal(); err != nil {
		t.Fatal(err)
	}
	if a.journalOpen {
		t.Fatal("the journal opened in English")
	}
	if !strings.Contains(a.statusLine, "-lang zh") {
		t.Fatalf("status line %q does not say how to open it", a.statusLine)
	}
}

// 走完整條輸入路徑：J 開手冊、逐格打進「46」、ENTER 跳過去。
// 直接測 journalInput 而不是只測 jump，是因為畫面上真正會壞的是按鍵接線
// （某個 case 先攔截、字元沒收到），不是查表本身。
func TestJournalInputTypesANumberAndJumps(t *testing.T) {
	corpus, err := journal.TraditionalChinese()
	if err != nil {
		t.Fatal(err)
	}
	a := &app{language: languageTraditionalChinese, mode: modeAdventure}
	a.journal = newJournalState(corpus)
	a.journal.reflow()
	a.journalOpen = true

	for _, digit := range []rune{'4', '6'} {
		a.keys = &scriptedTextKeys{scriptedKeys: scriptedKeys{}, chars: []rune{digit}}
		a.journalInput()
	}
	if a.journal.typed != "46" {
		t.Fatalf("typed buffer is %q after two digits", a.journal.typed)
	}
	a.keys = scriptedKeys{ebiten.KeyEnter: true}
	a.journalInput()
	entry, _ := a.journal.current()
	if entry.ID != "46" {
		t.Fatalf("ENTER landed on clue %s", entry.ID)
	}
	if a.journal.typed != "" {
		t.Fatalf("the typed buffer was not cleared: %q", a.journal.typed)
	}
	a.keys = scriptedKeys{ebiten.KeyEscape: true}
	a.journalInput()
	if a.journalOpen {
		t.Fatal("ESC did not close the journal")
	}
}
