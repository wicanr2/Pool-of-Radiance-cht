package gametext

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed monsters.zh-TW.json
var traditionalChineseMonstersJSON []byte

// MonsterEntry 是一個怪物名的譯法。Source 是 `MONnCHA` 記錄裡的名字，原版
// 一律大寫而且會補到固定長度，所以比對前兩邊都 TrimSpace。
//
// Basis 是證據等級（`exact`／`strong inference`），Note 說明來源。戰鬥畫面
// 顯示的是玩家看得到的字，所以每一條都要說得出依據——說明書怪物一覽表、
// 遊戲內文字用過的譯法，或由已定名的成分組出來的。
type MonsterEntry struct {
	Source string `json:"source"`
	Text   string `json:"text"`
	Basis  string `json:"basis"`
	Note   string `json:"note,omitempty"`
}

type monsterFile struct {
	Schema  string         `json:"schema"`
	Locale  string         `json:"locale"`
	Entries []MonsterEntry `json:"entries"`
}

// MonsterCatalogue 是一份語言的怪物名表。
type MonsterCatalogue struct {
	locale  string
	entries map[string]MonsterEntry
}

// ParseMonsters 讀入一份怪物名表。重複、空原文、空譯文與沒有證據等級都失敗
// 即關閉：那幾種都會讓某個怪物在戰鬥畫面上默默維持英文，而畫面上分不出是
// 漏翻還是打錯字。
func ParseMonsters(raw []byte) (*MonsterCatalogue, error) {
	var decoded monsterFile
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, fmt.Errorf("Pool monster name catalogue: %w", err)
	}
	if decoded.Schema != "pool-monster-names/1" {
		return nil, fmt.Errorf("Pool monster name catalogue schema %q is not pool-monster-names/1", decoded.Schema)
	}
	catalogue := &MonsterCatalogue{
		locale:  decoded.Locale,
		entries: make(map[string]MonsterEntry, len(decoded.Entries)),
	}
	for index, entry := range decoded.Entries {
		key := strings.TrimSpace(entry.Source)
		if key == "" {
			return nil, fmt.Errorf("Pool monster entry %d has an empty source", index)
		}
		if strings.TrimSpace(entry.Text) == "" {
			return nil, fmt.Errorf("Pool monster entry %d (%q) has an empty translation", index, entry.Source)
		}
		switch entry.Basis {
		case "exact", "strong inference":
		default:
			return nil, fmt.Errorf("Pool monster entry %d (%q) has basis %q, want exact or strong inference",
				index, entry.Source, entry.Basis)
		}
		if _, exists := catalogue.entries[key]; exists {
			return nil, fmt.Errorf("Pool monster entry %d repeats source %q", index, entry.Source)
		}
		catalogue.entries[key] = entry
	}
	return catalogue, nil
}

// TraditionalChineseMonsters 回傳內建的繁中怪物名表。
func TraditionalChineseMonsters() (*MonsterCatalogue, error) {
	return ParseMonsters(traditionalChineseMonstersJSON)
}

// Locale 是這份表的語言標記。
func (c *MonsterCatalogue) Locale() string {
	if c == nil {
		return ""
	}
	return c.locale
}

// Size 是表裡的條目數。
func (c *MonsterCatalogue) Size() int {
	if c == nil {
		return 0
	}
	return len(c.entries)
}

// Translate 把一個 `MONnCHA` 名字換成譯名；沒有譯名就原樣回傳（TrimSpace 後）。
// 空的 Catalogue 也能安全呼叫。
func (c *MonsterCatalogue) Translate(name string) string {
	trimmed := strings.TrimSpace(name)
	if c == nil {
		return trimmed
	}
	if entry, ok := c.entries[trimmed]; ok {
		return entry.Text
	}
	return trimmed
}

// Entry 取出一條，供稽核工具查證據等級。
func (c *MonsterCatalogue) Entry(name string) (MonsterEntry, bool) {
	if c == nil {
		return MonsterEntry{}, false
	}
	entry, ok := c.entries[strings.TrimSpace(name)]
	return entry, ok
}

// Sources 列出表裡所有原名，供核對工具使用。
func (c *MonsterCatalogue) Sources() []string {
	if c == nil {
		return nil
	}
	sources := make([]string, 0, len(c.entries))
	for source := range c.entries {
		sources = append(sources, source)
	}
	return sources
}
