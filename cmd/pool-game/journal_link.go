package main

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/journal"
)

// 遊戲文字報編號、玩家自己去翻手冊，是**紙本時代的做法**：原版把手冊印成
// 一本小冊子，畫面上只寫得下「成為線索報導 46」。remake 把手冊收進遊戲之後
// 就沒有理由再要玩家自己查——文字裡報了編號，按繼續的那一下就直接翻到那一則。
//
// 為什麼不把內文直接印進文字框：那個框是原版的尺寸，五行 × 68 欄
//（`dialogueLines`／`drawDialogue`）；一則線索報導動輒十幾行，塞進去要不是
// 截斷就是要把框放大，兩者都會讓探索畫面不再是原版那個版面。手冊頁本來就是
// 為長文排的（14 行、可捲動），翻過去讀完按 ESC 就回到原地。

// journalCitationHeads 是譯文裡引用手冊的三種說法。**三種都要認**：
// 起始地圖走幾步就到的市政廳外牆引用的是公告，酒館引用的是傳言，
// 只認線索報導的話那兩處按下去不會有反應。
//
// 「公告字號」同時是手冊裡那一章的章名（`PROCLAMATION` 的譯文）；章名後面
// 沒有編號，解析不出東西就不會產生引用。
var journalCitationHeads = []struct {
	Marker string
	Kind   journal.Kind
	Roman  bool
}{
	{"線索報導", journal.Clue, false},
	{"公告字號", journal.Proclamation, true},
	{"酒館裡的傳言", journal.Rumour, false},
	{"酒店傳言", journal.Rumour, false},
}

// journalCitationSeparators 是編號之間的分隔。原版列舉寫成
// 「公告字號 LXIV、LXXVIII、CIX 與 LIX。」與「成為線索報導 23 與 14。」
const journalCitationSeparators = " 　、，,和與及"

// journalCue 是「這段文字要玩家去讀哪一則」。
type journalCue struct {
	Kind journal.Kind
	ID   string
}

// journalCitationsIn 依出現順序取出文字裡的每一則引用。
//
// **一段話引用多則是常態**，不是例外：市政廳外牆一次報四則公告，
// 有兩處卷宗一次報兩則線索報導。只取第一則的話其餘幾則玩家還是得自己翻。
func journalCitationsIn(text string) []journalCue {
	var cues []journalCue
	runes := []rune(text)
	for index := 0; index < len(runes); {
		head, width := citationHeadAt(runes, index)
		if head < 0 {
			index++
			continue
		}
		kind := journalCitationHeads[head].Kind
		roman := journalCitationHeads[head].Roman
		index += width
		for {
			id, next, ok := readCitationID(runes, index, roman)
			if !ok {
				break
			}
			cues = append(cues, journalCue{Kind: kind, ID: id})
			index = next
		}
	}
	return cues
}

// citationHeadAt 看這個位置是不是某一種引用的開頭，回傳第幾種與吃掉幾個字。
// 較長的說法先比，免得「酒店傳言」被短的前綴切掉。
func citationHeadAt(runes []rune, index int) (int, int) {
	best, width := -1, 0
	for order, head := range journalCitationHeads {
		marker := []rune(head.Marker)
		if len(marker) <= width || index+len(marker) > len(runes) {
			continue
		}
		if string(runes[index:index+len(marker)]) != head.Marker {
			continue
		}
		best, width = order, len(marker)
	}
	return best, width
}

// readCitationID 跳過分隔符之後讀一個編號。羅馬數字只在公告那一種收，
// 否則譯文裡的英文專有名詞（`Sahuagin` 那類）會被當成字號。
func readCitationID(runes []rune, index int, roman bool) (string, int, bool) {
	for index < len(runes) && strings.ContainsRune(journalCitationSeparators, runes[index]) {
		index++
	}
	start := index
	for index < len(runes) && isCitationDigit(runes[index], roman) {
		index++
	}
	if index == start {
		return "", start, false
	}
	// 羅馬字號後面若還黏著別的字母，那是一個英文字而不是字號。
	if index < len(runes) && unicode.IsLetter(runes[index]) {
		return "", start, false
	}
	return string(runes[start:index]), index, true
}

func isCitationDigit(symbol rune, roman bool) bool {
	if symbol >= '0' && symbol <= '9' {
		return !roman
	}
	return roman && strings.ContainsRune("IVXLCDM", symbol)
}

// updateJournalCue 在文字框變動之後重算待翻清單。**英文模式不設**：手冊是
// 軟體世界的中譯本，`openJournal` 在英文模式會拒絕，設了只會讓 ENTER 沒反應。
func (a *app) updateJournalCue() {
	a.journalCues = nil
	if a.language != languageTraditionalChinese {
		return
	}
	for _, cue := range journalCitationsIn(a.eventText) {
		if a.journalCueDone[cue] {
			continue
		}
		a.journalCues = append(a.journalCues, cue)
	}
}

// takeJournalCue 取下一則並記成已翻。回傳 false 表示沒有待翻的。
//
// 記成已翻是為了不要在同一段文字上反覆彈出來：玩家讀完關掉手冊之後會回到
// 同一個文字框，再按一次 ENTER 應該是下一則，四則都翻完就是「繼續」。
// 這一格的腳本跑完（`finishCellBlock`）就忘掉，所以再走回這一格還會再翻。
func (a *app) takeJournalCue() (journalCue, bool) {
	if len(a.journalCues) == 0 {
		return journalCue{}, false
	}
	cue := a.journalCues[0]
	a.journalCues = a.journalCues[1:]
	if a.journalCueDone == nil {
		a.journalCueDone = map[journalCue]bool{}
	}
	a.journalCueDone[cue] = true
	return cue, true
}

// journalCuePrompt 是**文字框裡**最後一行的提示。玩家沒有理由知道按下去會
// 翻手冊，一次引用好幾則時也要看得出還有幾則。
//
// 畫在框裡而不是框外那一列：那一列是原版的指令列與「按 RETURN 繼續」的
// 位置——原版走到市政廳外時印的是 `AREA CAST VIEW ENCAMP SEARCH LOOK`，
// 提示放上去會把它擠掉（spec 132）。
func (a *app) journalCuePrompt() string {
	if len(a.journalCues) == 0 {
		return ""
	}
	// 面板蓋上來時對話框整個不畫，提示自然也不該出現。
	if a.panelOpen() {
		return ""
	}
	cue := a.journalCues[0]
	name := journalKindNames[cue.Kind]
	if remaining := len(a.journalCues) - 1; remaining > 0 {
		return fmt.Sprintf(a.text(msgJournalCueMore), name, cue.ID, remaining)
	}
	return fmt.Sprintf(a.text(msgJournalCue), name, cue.ID)
}


// openJournalAt 開手冊並停在指定的一則。
func (a *app) openJournalAt(cue journalCue) error {
	if err := a.openJournal(); err != nil {
		return err
	}
	if !a.journalOpen || a.journal == nil {
		return nil
	}
	if !a.journal.jumpTo(cue.Kind, cue.ID) {
		// 引用了一則手冊裡沒有的編號。手冊照開，並說明白——安靜地停在
		// 上一則會讓玩家以為自己看到的就是那一則。
		a.journal.message = "查無" + journalKindNames[cue.Kind] + " " + cue.ID
	}
	return nil
}
