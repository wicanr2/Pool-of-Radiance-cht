// Package journal 提供遊戲內可查的《探險者手冊》條目。
//
// 遊戲文字會說「成為線索報導 46」，玩家接著要翻手冊去讀那一條。原版把手冊
// 印成紙本，玩家要離開螢幕才讀得到；remake 把同一批條目放進遊戲內，編號沿用
// 說明書自己的編號，不自創。
//
// 內容來自軟體世界代理當年的官方繁中說明書上冊，轉錄在
// docs/reference/manual/journal-vol1.md，由 cmd/pool-journal-corpus 切成本套件
// 的 JSON。譯文權利屬軟體世界與當年譯者。
package journal

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed zh-TW.json
var traditionalChineseJSON []byte

// Kind 是手冊裡的四種條目，對應說明書的第五、六、四章與書末的附錄。
type Kind string

const (
	Clue         Kind = "clue"         // 第五章 探險者線索提示，編號 1..58
	Rumour       Kind = "rumour"       // 第六章 酒店傳言，編號 1..23
	Proclamation Kind = "proclamation" // 第四章 新菲蘭城議會公告，字號為羅馬數字
	// Appendix 是書末的七節附錄（p.48–54）：金錢換算、法術表、裝備、
	// 昇級經驗、對抗不死、各等級可用裝備武器、武器一覽。它們是**規則表**，
	// 原版要玩家翻紙本才查得到。
	Appendix Kind = "appendix"
)

// Kinds 是顯示順序：遊戲最常引用線索報導，其次是傳言，公告只在市政廳出現，
// 附錄是查規則用的，放最後。
var Kinds = []Kind{Clue, Rumour, Proclamation, Appendix}

// Entry 是一條手冊條目。ID 是說明書印出的編號本身（線索與傳言是十進位數字，
// 公告是羅馬數字），因為遊戲畫面說的就是那個編號。
type Entry struct {
	Kind Kind   `json:"kind"`
	ID   string `json:"id"`
	// Title 只有附錄有：書上那一節的標題（「金錢換算方法」）。
	Title string `json:"title,omitempty"`
	Page string `json:"page,omitempty"`
	Pic  string `json:"pic,omitempty"`
	Text string `json:"text"`
}

type file struct {
	Schema  string  `json:"schema"`
	Locale  string  `json:"locale"`
	Source  string  `json:"source"`
	Entries []Entry `json:"entries"`
}

// Corpus 是一份語言的手冊。
type Corpus struct {
	locale  string
	byKind  map[Kind][]Entry
	byIndex map[Kind]map[string]int
}

// 說明書自己的條目數。數量對不上就失敗即關閉：少一條的症狀是玩家查不到那一條，
// 而畫面上看起來像遊戲沒有這個功能，不像資料缺了一筆。
var wantCounts = map[Kind]int{Clue: 58, Rumour: 23, Proclamation: 18, Appendix: 7}

// Parse 讀入一份手冊。
func Parse(raw []byte) (*Corpus, error) {
	var decoded file
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, fmt.Errorf("Pool journal corpus: %w", err)
	}
	if decoded.Schema != "pool-journal/1" {
		return nil, fmt.Errorf("Pool journal corpus schema %q is not pool-journal/1", decoded.Schema)
	}
	corpus := &Corpus{
		locale:  decoded.Locale,
		byKind:  make(map[Kind][]Entry, len(Kinds)),
		byIndex: make(map[Kind]map[string]int, len(Kinds)),
	}
	for index, entry := range decoded.Entries {
		if _, known := wantCounts[entry.Kind]; !known {
			return nil, fmt.Errorf("Pool journal entry %d has unknown kind %q", index, entry.Kind)
		}
		if strings.TrimSpace(entry.ID) == "" {
			return nil, fmt.Errorf("Pool journal entry %d has an empty id", index)
		}
		if strings.TrimSpace(entry.Text) == "" {
			return nil, fmt.Errorf("Pool journal entry %s %s has empty text", entry.Kind, entry.ID)
		}
		if corpus.byIndex[entry.Kind] == nil {
			corpus.byIndex[entry.Kind] = map[string]int{}
		}
		if _, exists := corpus.byIndex[entry.Kind][entry.ID]; exists {
			return nil, fmt.Errorf("Pool journal repeats %s %s", entry.Kind, entry.ID)
		}
		corpus.byIndex[entry.Kind][entry.ID] = len(corpus.byKind[entry.Kind])
		corpus.byKind[entry.Kind] = append(corpus.byKind[entry.Kind], entry)
	}
	for kind, want := range wantCounts {
		if got := len(corpus.byKind[kind]); got != want {
			return nil, fmt.Errorf("Pool journal has %d %s entries, the manual prints %d", got, kind, want)
		}
	}
	return corpus, nil
}

// TraditionalChinese 回傳內建的繁中手冊。
func TraditionalChinese() (*Corpus, error) { return Parse(traditionalChineseJSON) }

// Locale 是這份手冊的語言標記。
func (c *Corpus) Locale() string {
	if c == nil {
		return ""
	}
	return c.locale
}

// Entries 回傳某一章的全部條目，順序即說明書的編號順序。
func (c *Corpus) Entries(kind Kind) []Entry {
	if c == nil {
		return nil
	}
	return c.byKind[kind]
}

// Lookup 依章別與編號取一條；遊戲文字報出的編號就是這個鍵。
func (c *Corpus) Lookup(kind Kind, id string) (Entry, bool) {
	if c == nil {
		return Entry{}, false
	}
	index, ok := c.byIndex[kind][id]
	if !ok {
		return Entry{}, false
	}
	return c.byKind[kind][index], true
}

// Size 是全部條目數。
func (c *Corpus) Size() int {
	if c == nil {
		return 0
	}
	total := 0
	for _, entries := range c.byKind {
		total += len(entries)
	}
	return total
}
