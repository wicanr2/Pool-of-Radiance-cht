// Command pool-journal-corpus 把轉錄好的《探險者手冊》上冊切成遊戲內可查的條目。
//
// 遊戲文字會說「成為線索報導 46」，玩家接著要翻手冊去讀那一條。手冊的轉錄是
// 一份連續的 Markdown，人讀沒問題，但遊戲要的是「編號 → 全文」。本工具做的就是
// 這個轉換，輸出 internal/journal/zh-TW.json。
//
// 條目的切法與編號規則見 spec 064；轉錄本身（來源、簡繁規則、逐章的數量）
// 在 spec 054。
//
// 三種條目在原書裡的形狀不同，因此各有自己的辨識方式：
//
//	線索報導  `### 線索報導 N` 標題，內文到下一個標題為止
//	酒店傳言  `**傳言 N**：…` 一段就是一條
//	議會公告  `**公告字號 X**` 之後的段落，X 是羅馬數字
//
// 數量是失敗即關閉的閘門：說明書第五章 58 條、第六章 23 條、第四章 18 則。
// 少一條的症狀是玩家查不到那一條，而畫面上看起來像遊戲沒有這個功能。
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	wantClues         = 58
	wantRumours       = 23
	wantProclamations = 18
	wantAppendices    = 7
)

// entry 是一條手冊條目。ID 用說明書自己的編號，不自創。
type entry struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
	// Title 只有附錄有：書上那一節的標題（「金錢換算方法」）。
	Title string `json:"title,omitempty"`
	Page string `json:"page,omitempty"`
	Pic  string `json:"pic,omitempty"`
	Text string `json:"text"`
}

type corpus struct {
	Schema  string  `json:"schema"`
	Locale  string  `json:"locale"`
	Source  string  `json:"source"`
	Entries []entry `json:"entries"`
}

var (
	pageHeading   = regexp.MustCompile(`^## p\.([0-9]+)(?: · [^（]*)?（(Pic[0-9]+(?:-(?:left|right))?)）\s*$`)
	clueHeading   = regexp.MustCompile(`^### 線索報導 ([0-9]+)\s*$`)
	otherHeading  = regexp.MustCompile(`^#{2,6} `)
	rumourLine    = regexp.MustCompile(`^\*\*傳言 ([0-9]+)\*\*[：:]\s*(.*)$`)
	proclamHeader = regexp.MustCompile(`^\*\*公告字號 ([IVXLCDM]+)\*\*\s*$`)
	// 附錄那七節的標題是 `#### 1. 金錢換算方法`，編號就是書上的編號。
	appendixHeading = regexp.MustCompile(`^#### ([0-9]+)\. (.+?)\s*$`)
)

func main() {
	source := flag.String("source", "docs/reference/manual/journal-vol1.md", "轉錄好的上冊 Markdown")
	flag.Parse()

	raw, err := os.ReadFile(*source)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	entries, err := extract(string(raw))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	out, err := json.MarshalIndent(corpus{
		Schema:  "pool-journal/1",
		Locale:  "zh-TW",
		Source:  *source,
		Entries: entries,
	}, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Stdout.Write(append(out, '\n'))
}

// collector 累積一條正在讀的多段條目。
type collector struct {
	kind  string
	id    string
	page  string
	pic   string
	title string
	lines []string
}

// paragraphs 把轉錄時的硬換行還原成段落。原書一條可以跨頁，頁首被拿掉之後
// 兩邊會各留一個空行，若照字面當成分段，一句話會被切成兩段。
//
// 段內接合不補空白：中文本來就不用空白，而轉錄的硬換行都落在中文之間。
func (c *collector) paragraphs() string {
	var out []string
	var current strings.Builder
	closeParagraph := func() {
		if current.Len() > 0 {
			out = append(out, current.String())
			current.Reset()
		}
	}
	for _, line := range c.lines {
		trimmed := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), ">"))
		if trimmed == "" {
			closeParagraph()
			continue
		}
		current.WriteString(trimmed)
	}
	closeParagraph()
	return strings.Join(out, "\n")
}

func extract(text string) ([]entry, error) {
	var (
		entries []entry
		open      *collector
		page      string
		pic       string
		skipBlank bool
	)
	flush := func() {
		if open == nil {
			return
		}
		body := open.paragraphs()
		if open.kind == "appendix" {
			body = appendixText(open.id, open.lines)
		}
		if body != "" {
			entries = append(entries, entry{
				Kind: open.kind, ID: open.id, Page: open.page, Pic: open.pic,
				Title: open.title, Text: body,
			})
		}
		open = nil
	}

	for _, line := range strings.Split(text, "\n") {
		// 頁首只記錄目前頁碼，不進條目內文——一條條目常常跨頁。
		// 連同頁首前後的空行一起吃掉，否則跨頁的那一句會被當成兩段。
		if match := pageHeading.FindStringSubmatch(line); match != nil {
			page, pic = match[1], match[2]
			if open != nil {
				open.lines = trimTrailingBlank(open.lines)
			}
			skipBlank = true
			continue
		}
		if skipBlank {
			if strings.TrimSpace(line) == "" {
				continue
			}
			skipBlank = false
		}
		if match := clueHeading.FindStringSubmatch(line); match != nil {
			flush()
			open = &collector{kind: "clue", id: match[1], page: page, pic: pic}
			continue
		}
		if match := appendixHeading.FindStringSubmatch(line); match != nil {
			flush()
			open = &collector{kind: "appendix", id: match[1], page: page, pic: pic,
				title: match[2]}
			continue
		}
		if match := proclamHeader.FindStringSubmatch(line); match != nil {
			flush()
			open = &collector{kind: "proclamation", id: match[1], page: page, pic: pic}
			continue
		}
		if match := rumourLine.FindStringSubmatch(line); match != nil {
			flush()
			open = &collector{kind: "rumour", id: match[1], page: page, pic: pic}
			open.lines = append(open.lines, match[2])
			continue
		}
		// 別的標題結束目前條目：第五章結束時下一個標題就是第六章的章名。
		if otherHeading.MatchString(line) {
			flush()
			continue
		}
		if open != nil {
			open.lines = append(open.lines, line)
		}
	}
	flush()

	if err := checkCounts(entries); err != nil {
		return nil, err
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Kind != entries[j].Kind {
			return entries[i].Kind < entries[j].Kind
		}
		return lessID(entries[i], entries[j])
	})
	return entries, nil
}

// lessID 讓線索與傳言依數字排序，公告依羅馬數字的值排序。
func trimTrailingBlank(lines []string) []string {
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func lessID(a, b entry) bool {
	if a.Kind == "proclamation" {
		return roman(a.ID) < roman(b.ID)
	}
	left, _ := strconv.Atoi(a.ID)
	right, _ := strconv.Atoi(b.ID)
	return left < right
}

func roman(value string) int {
	digits := map[byte]int{'I': 1, 'V': 5, 'X': 10, 'L': 50, 'C': 100, 'D': 500, 'M': 1000}
	total := 0
	for index := 0; index < len(value); index++ {
		current := digits[value[index]]
		if index+1 < len(value) && current < digits[value[index+1]] {
			total -= current
			continue
		}
		total += current
	}
	return total
}

func checkCounts(entries []entry) error {
	seen := map[string]map[string]bool{}
	for _, item := range entries {
		if seen[item.Kind] == nil {
			seen[item.Kind] = map[string]bool{}
		}
		if seen[item.Kind][item.ID] {
			return fmt.Errorf("%s %s 出現兩次", item.Kind, item.ID)
		}
		seen[item.Kind][item.ID] = true
	}
	for kind, want := range map[string]int{
		"clue": wantClues, "rumour": wantRumours, "proclamation": wantProclamations,
		"appendix": wantAppendices,
	} {
		if got := len(seen[kind]); got != want {
			return fmt.Errorf("%s 取出 %d 條，說明書是 %d 條", kind, got, want)
		}
	}
	// 線索與傳言的編號必須連續：缺號代表某一條的標題沒被認出來。
	for kind, want := range map[string]int{"clue": wantClues, "rumour": wantRumours} {
		for number := 1; number <= want; number++ {
			if !seen[kind][strconv.Itoa(number)] {
				return fmt.Errorf("%s 缺編號 %d", kind, number)
			}
		}
	}
	return nil
}
