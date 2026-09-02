// Package gametext 把原版 ECL 的敘述文字換成譯文。
//
// 原版 block 一個位元組都不改：譯文放在本套件的 JSON 裡，以「原文整句」為鍵。
// 用整句而不是指令位址當鍵，是因為同一句話常常出現在好幾個 block（`ecl3/0`
// 與 `ecl3/11` 就有共用的句子），以位址為鍵會逼人把同一句翻好幾次，
// 而且改一處忘另一處時畫面會一半中文一半英文。
package gametext

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed zh-TW.json
var traditionalChineseJSON []byte

// Entry 是一條譯文。Source 必須與原版解出來的字串完全相同——
// 對不上就翻不到，而翻不到在畫面上看起來像漏翻，不像打錯字，所以有測試
// 拿 `docs/audit/dos-ecl-text-inventory.json` 逐條核對。
type Entry struct {
	Source string `json:"source"`
	Text   string `json:"text"`
	Note   string `json:"note,omitempty"`
}

type file struct {
	Schema  string  `json:"schema"`
	Locale  string  `json:"locale"`
	Entries []Entry `json:"entries"`
}

// Catalogue 是一份語言的譯文表。
type Catalogue struct {
	locale  string
	entries map[string]string
}

// Parse 讀入一份譯文表。重複的原文、空的原文或空的譯文都失敗即關閉：
// 那三種都會讓某一句話默默維持英文，而漏翻在畫面上分不出是哪一種原因。
func Parse(raw []byte) (*Catalogue, error) {
	var decoded file
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, fmt.Errorf("Pool game text catalogue: %w", err)
	}
	if decoded.Schema != "pool-game-text/1" {
		return nil, fmt.Errorf("Pool game text catalogue schema %q is not pool-game-text/1", decoded.Schema)
	}
	catalogue := &Catalogue{locale: decoded.Locale, entries: make(map[string]string, len(decoded.Entries))}
	for index, entry := range decoded.Entries {
		if strings.TrimSpace(entry.Source) == "" {
			return nil, fmt.Errorf("Pool game text entry %d has an empty source", index)
		}
		if strings.TrimSpace(entry.Text) == "" {
			return nil, fmt.Errorf("Pool game text entry %d (%q) has an empty translation", index, entry.Source)
		}
		if _, exists := catalogue.entries[entry.Source]; exists {
			return nil, fmt.Errorf("Pool game text entry %d repeats source %q", index, entry.Source)
		}
		catalogue.entries[entry.Source] = entry.Text
	}
	return catalogue, nil
}

// TraditionalChinese 回傳內建的繁中譯文表。
func TraditionalChinese() (*Catalogue, error) { return Parse(traditionalChineseJSON) }

// Locale 是這份表的語言標記。
func (c *Catalogue) Locale() string {
	if c == nil {
		return ""
	}
	return c.locale
}

// Size 是表裡的條目數。
func (c *Catalogue) Size() int {
	if c == nil {
		return 0
	}
	return len(c.entries)
}

// Translate 把一句原文換成譯文；沒有譯文就原樣回傳。
// 空的 Catalogue 也能安全呼叫，讓「沒載入譯文」與「這句還沒翻」走同一條路徑。
func (c *Catalogue) Translate(source string) string {
	if c == nil {
		return source
	}
	if text, ok := c.entries[source]; ok {
		return text
	}
	return source
}

// Sources 列出表裡所有原文，供核對工具使用。
func (c *Catalogue) Sources() []string {
	if c == nil {
		return nil
	}
	sources := make([]string, 0, len(c.entries))
	for source := range c.entries {
		sources = append(sources, source)
	}
	return sources
}
