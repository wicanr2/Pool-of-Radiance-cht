package main

import (
	"strings"
	"unicode"
)

// 附錄那七節是**表格**，不是段落。段落那條路（`paragraphs`）會把沒有空行
// 隔開的表格列接成一長串，所以附錄另走這裡。
//
// 排版是 remake 自己的決定：原書是紙本的橫表，遊戲畫面只有 640×400、
// 字是等寬的倚天點陣（半形 8 像素、漢字 16 像素），一行放不下 103 個半形位。
// 這裡把每一欄縮到放得下，欄內放不下的字自動折行；**欄名的縮寫寫在
// `appendixColumnNames`，每一條都註明出處**，不自行改寫語意。

// appendixColumns 是一行放得下的半形位數。與 `journalColumns` 同一個值——
// 手冊畫面就是用它換行的，這裡多排一次只是為了讓欄位對得齊。
const appendixColumns = 64

// appendixColumnNames 覆寫過寬的表頭。鍵是附錄編號。
//
//   - 附錄 3 的表頭「價值（單位：黃金）」與「携帶後所能移動的最大步伐」
//     只是把括號與修飾語拿掉，語意不變。
//   - 附錄 7 原書照排英文表，**中文欄名取自那一節正文自己寫的對照**
//     （「名稱／對人之傷害力／對比人還大的怪物之傷害力／單、雙手持握／等級」），
//     不是另外翻的。
var appendixColumnNames = map[string][]string{
	"3": {"種類", "價值", "防禦力", "最大步伐"},
	"7": {"名稱", "對人", "對大型", "持握", "等級"},
}

// appendixText 把一節附錄排成手冊畫面直接畫得出來的行。
func appendixText(id string, lines []string) string {
	var out []string
	var table [][]string
	// 小標題出現幾次。原書一節跨頁時會把它再印一次，見 dropRepeatedHeadings。
	headings := map[string]int{}
	flushTable := func() {
		if len(table) == 0 {
			return
		}
		out = append(out, renderTable(id, table)...)
		table = nil
	}
	var paragraph strings.Builder
	flushParagraph := func() {
		if paragraph.Len() > 0 {
			out = append(out, paragraph.String())
			paragraph.Reset()
		}
	}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "|") {
			flushParagraph()
			cells := splitRow(trimmed)
			if isRule(cells) {
				continue
			}
			table = append(table, cells)
			continue
		}
		flushTable()
		if trimmed == "" {
			flushParagraph()
			continue
		}
		// `**A：牧師**` 這種小標題自成一段。接在別的字後面的話，附錄 2 的
		// 「A：牧師」會跟版面說明黏成一句。星號是 Markdown 的記號，
		// 畫面上不解析，所以這裡就拿掉。
		if strings.HasPrefix(trimmed, "**") && strings.HasSuffix(trimmed, "**") {
			flushParagraph()
			heading := strings.Trim(trimmed, "*")
			headings[heading]++
			out = append(out, heading)
			continue
		}
		paragraph.WriteString(strings.TrimSpace(strings.TrimPrefix(trimmed, ">")))
	}
	flushParagraph()
	flushTable()
	return strings.Join(dropRepeatedHeadings(out, headings), "\n")
}

// dropRepeatedHeadings 拿掉重複的小標題，**留最後一次**。
//
// 原書一節跨頁時會把小標題再印一次（附錄 2 的「A：牧師」在 p.48 結尾與
// p.49 開頭各一次），轉錄照實記了兩份；頁碼在遊戲裡不存在，兩份接在一起
// 就變成同一句連講兩遍。留最後一次是因為斷頁重印的目的正是讓**表格上方**
// 有標題——留第一次的話，表格會變成沒有標題。
func dropRepeatedHeadings(lines []string, headings map[string]int) []string {
	seen := map[string]int{}
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if count := headings[line]; count > 1 {
			seen[line]++
			if seen[line] < count {
				continue
			}
		}
		out = append(out, line)
	}
	return out
}

// splitRow 切一列 markdown 表格。
func splitRow(line string) []string {
	cells := strings.Split(strings.Trim(line, "|"), "|")
	for index := range cells {
		cells[index] = strings.TrimSpace(cells[index])
	}
	return cells
}

// isRule 認出 `|---|---:|` 那一列。
func isRule(cells []string) bool {
	for _, cell := range cells {
		if cell == "" || strings.Trim(cell, "-: ") != "" {
			return false
		}
	}
	return len(cells) > 0
}

// renderTable 把一張表排成等寬對齊的幾行。
func renderTable(id string, rows [][]string) []string {
	count := 0
	for _, row := range rows {
		if len(row) > count {
			count = len(row)
		}
	}
	if count == 0 {
		return nil
	}
	if names, ok := appendixColumnNames[id]; ok && len(rows) > 0 && len(names) == count {
		rows = append([][]string{names}, rows[1:]...)
	}
	widths := make([]int, count)
	for _, row := range rows {
		for index, cell := range row {
			if width := displayWidth(cell); width > widths[index] {
				widths[index] = width
			}
		}
	}
	shrinkToFit(widths, rows)

	out := make([]string, 0, len(rows))
	for _, row := range rows {
		// 一列可能因為某一欄折行而佔好幾行；沒有內容的欄留空白。
		wrapped := make([][]string, count)
		height := 1
		for index := 0; index < count; index++ {
			cell := ""
			if index < len(row) {
				cell = row[index]
			}
			wrapped[index] = wrapToWidth(cell, widths[index])
			if len(wrapped[index]) > height {
				height = len(wrapped[index])
			}
		}
		for line := 0; line < height; line++ {
			var builder strings.Builder
			for index := 0; index < count; index++ {
				piece := ""
				if line < len(wrapped[index]) {
					piece = wrapped[index][line]
				}
				builder.WriteString(piece)
				if index == count-1 {
					continue
				}
				builder.WriteString(strings.Repeat(" ",
					widths[index]-displayWidth(piece)+1))
			}
			out = append(out, strings.TrimRight(builder.String(), " "))
		}
	}
	return out
}

// shrinkToFit 把總寬壓進一行。從最寬的那一欄開始扣，扣到放得下為止；
// 欄名本身的寬度是下限，扣到看不出欄名就不是排版而是刪字了。
func shrinkToFit(widths []int, rows [][]string) {
	floors := make([]int, len(widths))
	for index := range widths {
		floors[index] = 4
		if len(rows) > 0 && index < len(rows[0]) {
			if width := displayWidth(rows[0][index]); width > floors[index] {
				floors[index] = width
			}
		}
		if floors[index] > widths[index] {
			floors[index] = widths[index]
		}
	}
	for total(widths) > appendixColumns {
		widest, at := 0, -1
		for index, width := range widths {
			// 並列最寬時取**右邊**那一欄。左邊第一欄通常是名稱，折了就會
			// 變成「魔術師（Magic-User」加下一行一個「）」，中間還夾著同一列
			// 其他欄的字；右邊的欄多半是可以斷句的長描述。
			if width > floors[index] && width >= widest {
				widest, at = width, index
			}
		}
		if at < 0 {
			return
		}
		widths[at]--
	}
}

// total 是這些欄加上欄間一個空白之後的總寬。
func total(widths []int) int {
	sum := len(widths) - 1
	for _, width := range widths {
		sum += width
	}
	return sum
}

// displayWidth 是畫出來佔幾個半形位。倚天字型的漢字正好是半形的兩倍寬。
func displayWidth(value string) int {
	width := 0
	for _, symbol := range value {
		width += runeWidth(symbol)
	}
	return width
}

func runeWidth(symbol rune) int {
	if symbol < 0x80 {
		return 1
	}
	// 半形片假名與拉丁補充之外的東亞字元都是兩格。這批文字只有 CJK 與
	// 全形標點，所以用「不是 ASCII 就是兩格」就夠，不必拖一張表進來。
	if unicode.Is(unicode.Hiragana, symbol) || unicode.Is(unicode.Katakana, symbol) ||
		unicode.Is(unicode.Han, symbol) || unicode.Is(unicode.Hangul, symbol) {
		return 2
	}
	if symbol >= 0x3000 && symbol <= 0x303f || symbol >= 0xff00 && symbol <= 0xff60 {
		return 2
	}
	return 2
}

// wrapToWidth 把一格的內容折成幾行。折點優先落在頓號、斜線與空白上——
// 附錄 2 的法術欄就是用頓號分隔的長串。
func wrapToWidth(value string, width int) []string {
	if value == "" {
		return []string{""}
	}
	var out []string
	var line strings.Builder
	lineWidth := 0
	breakAt, breakWidth := -1, 0
	for _, symbol := range value {
		symbolWidth := runeWidth(symbol)
		if lineWidth+symbolWidth > width && lineWidth > 0 {
			text := line.String()
			if breakAt > 0 && breakAt < len(text) {
				out = append(out, strings.TrimRight(text[:breakAt], " "))
				rest := strings.TrimLeft(text[breakAt:], " ")
				line.Reset()
				line.WriteString(rest)
				lineWidth = displayWidth(rest)
			} else {
				out = append(out, text)
				line.Reset()
				lineWidth = 0
			}
			breakAt, breakWidth = -1, 0
		}
		line.WriteRune(symbol)
		lineWidth += symbolWidth
		switch symbol {
		case '、', '／', '/', ' ', '，', ',':
			breakAt, breakWidth = line.Len(), lineWidth
		}
	}
	_ = breakWidth
	if line.Len() > 0 {
		out = append(out, line.String())
	}
	return out
}
