package main

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/journal"
)

// 遊戲文字會說「成為線索報導 46」，玩家接著要去翻那一條。原版把手冊印成紙本，
// 這裡把同一批條目放進遊戲內，編號沿用說明書自己的編號。
//
// 版面沿用戰術畫面量過的那組數字：行距 16、內文左界 48，最下面一行留給
// 功能鍵列（基線 386），因為倚天字型的 ascent 是 14，再往下會被下框切到字腳。
const (
	journalTextLeft   = 48
	journalFirstLine  = 118
	journalLineHeight = 16
	journalLineCount  = 14
	journalColumns    = 64
)

type journalState struct {
	corpus   *journal.Corpus
	kind     int
	entry    map[journal.Kind]int
	scroll   int
	typed    string
	message  string
	rendered []string
}

func newJournalState(corpus *journal.Corpus) *journalState {
	return &journalState{corpus: corpus, entry: map[journal.Kind]int{}}
}

func (s *journalState) currentKind() journal.Kind { return journal.Kinds[s.kind] }

func (s *journalState) current() (journal.Entry, bool) {
	entries := s.corpus.Entries(s.currentKind())
	if len(entries) == 0 {
		return journal.Entry{}, false
	}
	index := s.entry[s.currentKind()]
	if index < 0 || index >= len(entries) {
		return journal.Entry{}, false
	}
	return entries[index], true
}

// reflow 在條目或章別改變之後重排內文。段落之間留一個空行，因為手冊裡有
// 對話與引文，擠在一起會看不出換人說話。
func (s *journalState) reflow() {
	s.scroll = 0
	s.rendered = nil
	entry, ok := s.current()
	if !ok {
		return
	}
	for index, paragraph := range strings.Split(entry.Text, "\n") {
		if index > 0 {
			s.rendered = append(s.rendered, "")
		}
		s.rendered = append(s.rendered, wrapDisplay(paragraph, journalColumns)...)
	}
}

func (s *journalState) selectKind(delta int) {
	s.kind = (s.kind + delta + len(journal.Kinds)) % len(journal.Kinds)
	s.typed = ""
	s.message = ""
	s.reflow()
}

func (s *journalState) selectEntry(delta int) {
	entries := s.corpus.Entries(s.currentKind())
	if len(entries) == 0 {
		return
	}
	index := (s.entry[s.currentKind()] + delta + len(entries)) % len(entries)
	s.entry[s.currentKind()] = index
	s.typed = ""
	s.message = ""
	s.reflow()
}

// jump 依編號跳到一條。查不到就留在原地並說明白——安靜地不動會讓玩家以為
// 是按鍵沒吃到。
func (s *journalState) jump(id string) bool {
	entries := s.corpus.Entries(s.currentKind())
	for index, item := range entries {
		if item.ID == id {
			s.entry[s.currentKind()] = index
			s.reflow()
			return true
		}
	}
	return false
}

func (s *journalState) scrollBy(delta int) {
	max := len(s.rendered) - journalLineCount
	if max < 0 {
		max = 0
	}
	s.scroll += delta
	if s.scroll < 0 {
		s.scroll = 0
	}
	if s.scroll > max {
		s.scroll = max
	}
}

var journalKindNames = map[journal.Kind]string{
	journal.Clue:         "線索報導",
	journal.Rumour:       "酒店傳言",
	journal.Proclamation: "議會公告",
}

// openJournal 只在繁中模式開得起來：手冊本身是軟體世界的中譯，英文模式下的
// 內建字型沒有漢字，畫出來會是一片空白，看起來像繪圖壞掉而不是沒有這個功能。
func (a *app) openJournal() error {
	if a.language != languageTraditionalChinese {
		a.statusLine = "The journal is the Traditional Chinese manual; run with -lang zh."
		return nil
	}
	if a.journal == nil {
		corpus, err := journal.TraditionalChinese()
		if err != nil {
			return err
		}
		a.journal = newJournalState(corpus)
		a.journal.reflow()
	}
	a.journalOpen = true
	return nil
}

func (a *app) journalInput() {
	state := a.journal
	switch {
	case a.justPressed(ebiten.KeyEscape), a.justPressed(ebiten.KeyJ):
		a.journalOpen = false
	case a.justPressed(ebiten.KeyTab):
		state.selectKind(1)
	case a.justPressed(ebiten.KeyRight), a.justPressed(ebiten.KeyPageDown):
		state.selectEntry(1)
	case a.justPressed(ebiten.KeyLeft), a.justPressed(ebiten.KeyPageUp):
		state.selectEntry(-1)
	case a.justPressed(ebiten.KeyDown):
		state.scrollBy(1)
	case a.justPressed(ebiten.KeyUp):
		state.scrollBy(-1)
	case a.justPressed(ebiten.KeyBackspace):
		if state.typed != "" {
			state.typed = state.typed[:len(state.typed)-1]
		}
	case a.justPressed(ebiten.KeyEnter):
		if state.typed == "" {
			break
		}
		if !state.jump(strings.ToUpper(state.typed)) {
			state.message = fmt.Sprintf("查無 %s %s", journalKindNames[state.currentKind()], state.typed)
		} else {
			state.message = ""
		}
		state.typed = ""
	default:
		// 直接打編號跳條目：遊戲畫面報的是編號，玩家記得的也是編號。
		// 公告的字號是羅馬數字，所以字母也收。
		for _, r := range a.inputChars() {
			if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				if len(state.typed) < 8 {
					state.typed += string(r)
				}
			}
		}
	}
}

func drawJournal(screen *ebiten.Image, a *app, background, foreground, accent color.Color) {
	state := a.journal
	for y := 40; y < 372; y++ {
		for x := 32; x < 608; x++ {
			screen.Set(x, y, background)
		}
	}
	drawText(screen, "探險者手冊", 268, 62, accent)

	var tabs []string
	for index, kind := range journal.Kinds {
		name := journalKindNames[kind]
		if index == state.kind {
			name = "〔" + name + "〕"
		}
		tabs = append(tabs, name)
	}
	drawText(screen, strings.Join(tabs, "  "), journalTextLeft, 86, foreground)

	entry, ok := state.current()
	if !ok {
		drawText(screen, "這一章沒有條目。", journalTextLeft, journalFirstLine, foreground)
		return
	}
	header := fmt.Sprintf("%s %s", journalKindNames[entry.Kind], entry.ID)
	if entry.Page != "" {
		header += fmt.Sprintf("　（說明書上冊 p.%s）", entry.Page)
	}
	drawText(screen, header, journalTextLeft, 104, accent)

	for offset := 0; offset < journalLineCount; offset++ {
		index := state.scroll + offset
		if index >= len(state.rendered) {
			break
		}
		drawText(screen, state.rendered[index], journalTextLeft,
			journalFirstLine+offset*journalLineHeight, foreground)
	}

	footer := "TAB 換章　左右換條目　上下捲動　打編號後 ENTER 跳到該條　J／ESC 關閉"
	if state.typed != "" {
		footer = fmt.Sprintf("跳到 %s %s_", journalKindNames[state.currentKind()], state.typed)
	}
	if state.message != "" {
		footer = state.message
	}
	drawText(screen, footer, journalTextLeft, footerBaseline, accent)
}
