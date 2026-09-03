package main

import (
	"fmt"
	"strings"
)

// 結局過場（spec 108）。`38h PROGRAM` 的值 8 是 overlay-18 entry 1，
// 唯一的呼叫點是 `ECL5/7` 的 `A82Ah`——打贏泰倫斯拉克斯之後。
//
// **只接了文字**：原版在兩頁台詞之間還會依序畫 `FINAL5.DAX` 的區塊
// 1／3／4／5／6，那幾張圖的位置還沒讀出來（`018Eh:004Dh` 的引數），
// 所以這裡先不畫。缺的是圖不是字，spec 108 寫明了。
func (a *app) enterEnding() error {
	pages := a.endingScript.Pages()
	if len(pages) == 0 {
		return fmt.Errorf("Pool ending script has no pages")
	}
	a.endingActive, a.endingPages, a.endingPage = true, pages, 0
	a.cellEventPending, a.cellWaitingMenu = true, false
	a.showEndingPage()
	return nil
}

// showEndingPage 把目前這一頁放進文字框。行的順序與列號都照原版
//（spec 108 的表），這裡只是逐行接起來。
func (a *app) showEndingPage() {
	lines := a.endingPages[a.endingPage]
	rendered := make([]string, 0, len(lines))
	for _, line := range lines {
		rendered = append(rendered, a.gameText.Translate(line.Text))
	}
	a.eventText = strings.Join(rendered, "\n")
	a.eventLabel = a.text(msgEndingPrompt)
	a.statusLine = a.eventLabel
}

// advanceEnding 翻到下一頁；翻完就讓 ECL 往下跑。
func (a *app) advanceEnding() error {
	if a.endingPage+1 < len(a.endingPages) {
		a.endingPage++
		a.showEndingPage()
		return nil
	}
	a.endingActive, a.endingPages, a.endingPage = false, nil, 0
	a.eventText, a.eventLabel = "", ""
	a.cellEventPending = false
	return a.continueInitialSearch(nil)
}
