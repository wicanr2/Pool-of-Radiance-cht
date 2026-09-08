package journal_test

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/journal"
)

// 附錄那七節進遊戲時**重排過**：欄位對齊、過寬的表頭改用書上自己給的縮寫、
// 跨頁重印的小標題只留一份。所以它們不能像線索與傳言那樣逐字比
//（`TestTranscriptionAndGameCorpusAgree` 跳過附錄）。
//
// 這一則守的是同一件事的另一半：**排版可以變，字不可以掉也不可以改**。
// 轉錄裡每一格的文字都要在遊戲語料的那一節裡找得到。
func TestAppendixKeepsEveryCellFromTheTranscription(t *testing.T) {
	corpus, err := journal.TraditionalChinese()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(manualBookPath("journal-vol1.md"))
	if err != nil {
		t.Fatal(err)
	}
	sections := appendixSections(string(raw))
	if len(sections) != 7 {
		t.Fatalf("轉錄裡抓到 %d 節附錄，書上是 7 節", len(sections))
	}
	checked := 0
	for _, entry := range corpus.Entries(journal.Appendix) {
		source, ok := sections[entry.ID]
		if !ok {
			t.Errorf("語料有附錄 %s，轉錄裡找不到", entry.ID)
			continue
		}
		if entry.Title == "" {
			t.Errorf("附錄 %s 沒有標題", entry.ID)
		}
		packed := squeeze(entry.Text)
		for _, cell := range appendixCells(source) {
			if squeezedCell := squeeze(cell); squeezedCell != "" &&
				!strings.Contains(packed, squeezedCell) &&
				!shortenedHeadings[cell] {
				t.Errorf("附錄 %s 少了一格：%q", entry.ID, cell)
			}
			checked++
		}
	}
	if checked < 200 {
		t.Fatalf("只比到 %d 格，抽取顯然壞了", checked)
	}
}

// shortenedHeadings 是**唯一允許不逐字出現**的那幾格：過寬的表頭。
// 縮寫寫在 `cmd/pool-journal-corpus/appendix.go` 的 `appendixColumnNames`，
// 附錄 3 只是把括號與修飾語拿掉，附錄 7 用的是那一節正文自己寫的中文對照。
// 表頭以外的每一格都必須逐字在。
var shortenedHeadings = map[string]bool{
	"價值（單位：黃金）": true, "携帶後所能移動的最大步伐": true,
	"Name": true, "Damage vs. Man Sized": true,
	"Damage vs. Larger Than Man Sized": true, "Number of Hands": true,
	"Class": true, "種類": true, "防禦力": true,
}

var appendixStart = regexp.MustCompile(`(?m)^#### ([0-9]+)\. (.+?)\s*$`)
var appendixStop = regexp.MustCompile(`(?m)^(?:#{3,4} )`)

// appendixSections 把轉錄切成七節。換頁（`## p.N`）不算結束——附錄本來就跨頁。
func appendixSections(text string) map[string]string {
	out := map[string]string{}
	for _, span := range appendixStart.FindAllStringSubmatchIndex(text, -1) {
		id := text[span[2]:span[3]]
		end := len(text)
		if stop := appendixStop.FindStringIndex(text[span[1]:]); stop != nil {
			end = span[1] + stop[0]
		}
		out[id] = text[span[1]:end]
	}
	return out
}

// appendixCells 抽出一節裡表格的每一格，分隔列與空格丟掉。
func appendixCells(section string) []string {
	var out []string
	for _, line := range strings.Split(section, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "|") {
			continue
		}
		cells := strings.Split(strings.Trim(trimmed, "|"), "|")
		rule := true
		for _, cell := range cells {
			if strings.Trim(strings.TrimSpace(cell), "-:") != "" {
				rule = false
				break
			}
		}
		if rule {
			continue
		}
		for _, cell := range cells {
			out = append(out, strings.TrimSpace(cell))
		}
	}
	return out
}

// squeeze 拿掉空白與 Markdown 的記號，只留字本身。
func squeeze(value string) string {
	return strings.Map(func(symbol rune) rune {
		switch symbol {
		case ' ', '\t', '\n', '\r', '*', '>', '|':
			return -1
		}
		return symbol
	}, value)
}
