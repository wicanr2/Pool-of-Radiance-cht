package gamepack

import (
	"archive/zip"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

// 法術名稱表。與物品型別表不同，這一張是編譯進 START.EXE 的初始化資料：
// 從檔案位移 41052 起，每筆 41 bytes，第一個 byte 是長度，其餘是名稱，
// 尾端補零。共 56 筆，最後一筆 `Restoration` 起於 43307。
//
// 名稱之後的位元組全是零，所以**職業與等級不在這張表裡**；分組是由條目順序
// 推出來的，並與說明書下冊第六章逐條核對過：六個分組的界線正好落在說明書的
// LEVEL 標題上，組內順序也逐條相同。
//
// 說明書那一份有幾個原書拼錯（SPIRITOAL、KEY OF ENFEEBLEMENT、BESTOW URSE、
// LIGHTNENG、Stinking Clud），因此比對要以遊戲的拼法為準，不能反過來。
const (
	// SpellNameTableOffset 是表在 START.EXE 裡的檔案位移。
	SpellNameTableOffset = 41052
	// SpellNameEntrySize 是每筆的 byte 數。
	SpellNameEntrySize = 41
	// SpellNameCount 是表的筆數。
	SpellNameCount = 56
)

//go:embed spell_names.zh-TW.json
var spellNamesJSON []byte

// SpellClass 是施法職業。原版把兩類法術放在同一張表裡，用順序分組。
type SpellClass string

const (
	SpellClassCleric     SpellClass = "cleric"
	SpellClassMagicUser  SpellClass = "magic-user"
)

// Spell 是一條法術。Index 是表裡的位置，也是原版用來指涉它的編號。
type Spell struct {
	Index int        `json:"index"`
	Name  string     `json:"name"`
	Text  string     `json:"text"`
	Class SpellClass `json:"class"`
	Level int        `json:"level"`
}

type spellFile struct {
	Schema  string  `json:"schema"`
	Locale  string  `json:"locale"`
	Source  string  `json:"source"`
	Entries []Spell `json:"entries"`
}

// SpellCatalogue 是內建的法術表：遊戲自己的名稱順序，配上說明書的中譯。
type SpellCatalogue struct {
	locale  string
	entries []Spell
}

// ParseSpellCatalogue 讀入一份法術表。
func ParseSpellCatalogue(raw []byte) (*SpellCatalogue, error) {
	var decoded spellFile
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, fmt.Errorf("Pool spell catalogue: %w", err)
	}
	if decoded.Schema != "pool-spell-names/1" {
		return nil, fmt.Errorf("Pool spell catalogue schema %q is not pool-spell-names/1", decoded.Schema)
	}
	if len(decoded.Entries) != SpellNameCount {
		return nil, fmt.Errorf("Pool spell catalogue has %d entries, the original table has %d",
			len(decoded.Entries), SpellNameCount)
	}
	for position, entry := range decoded.Entries {
		if entry.Index != position {
			return nil, fmt.Errorf("Pool spell entry %d claims index %d", position, entry.Index)
		}
		if strings.TrimSpace(entry.Name) == "" || strings.TrimSpace(entry.Text) == "" {
			return nil, fmt.Errorf("Pool spell %d has an empty name or translation", position)
		}
		if entry.Class != SpellClassCleric && entry.Class != SpellClassMagicUser {
			return nil, fmt.Errorf("Pool spell %d has unknown class %q", position, entry.Class)
		}
		if entry.Level < 1 || entry.Level > 3 {
			return nil, fmt.Errorf("Pool spell %d has level %d outside 1..3", position, entry.Level)
		}
	}
	return &SpellCatalogue{locale: decoded.Locale, entries: decoded.Entries}, nil
}

// TraditionalChineseSpells 回傳內建的法術表。
func TraditionalChineseSpells() (*SpellCatalogue, error) { return ParseSpellCatalogue(spellNamesJSON) }

// Locale 是這份表的語言標記。
func (c *SpellCatalogue) Locale() string {
	if c == nil {
		return ""
	}
	return c.locale
}

// Spells 回傳全部條目，順序即原版表的順序。
func (c *SpellCatalogue) Spells() []Spell {
	if c == nil {
		return nil
	}
	return c.entries
}

// Spell 依編號取一條。
func (c *SpellCatalogue) Spell(index int) (Spell, error) {
	if c == nil || index < 0 || index >= len(c.entries) {
		return Spell{}, fmt.Errorf("Pool spell index %d is outside 0..%d", index, SpellNameCount-1)
	}
	return c.entries[index], nil
}

// ByClassAndLevel 取出某職業某等級的法術。原版把兩類放在同一張表裡，
// 施法選單要照職業與等級分頁，所以這個切法是接線時最常用的。
func (c *SpellCatalogue) ByClassAndLevel(class SpellClass, level int) []Spell {
	if c == nil {
		return nil
	}
	var out []Spell
	for _, entry := range c.entries {
		if entry.Class == class && entry.Level == level {
			out = append(out, entry)
		}
	}
	return out
}

// ReadDOSSpellNames 直接從 START.EXE 解出名稱表，用來核對內建的那一份。
func ReadDOSSpellNames(zipPath string) ([]string, error) {
	archive, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("open DOS ZIP: %w", err)
	}
	defer archive.Close()

	var member *zip.File
	for _, candidate := range archive.File {
		if strings.EqualFold(filepath.Base(candidate.Name), "START.EXE") {
			if member != nil {
				return nil, fmt.Errorf("DOS ZIP has duplicate START.EXE")
			}
			member = candidate
		}
	}
	if member == nil {
		return nil, fmt.Errorf("DOS ZIP has no START.EXE")
	}
	reader, err := member.Open()
	if err != nil {
		return nil, fmt.Errorf("open START.EXE: %w", err)
	}
	defer reader.Close()
	raw, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read START.EXE: %w", err)
	}
	end := SpellNameTableOffset + SpellNameCount*SpellNameEntrySize
	if len(raw) < end {
		return nil, fmt.Errorf("START.EXE is %d bytes, the spell table needs %d", len(raw), end)
	}
	names := make([]string, 0, SpellNameCount)
	for index := 0; index < SpellNameCount; index++ {
		entry := raw[SpellNameTableOffset+index*SpellNameEntrySize:][:SpellNameEntrySize]
		length := int(entry[0])
		if length < 1 || length > SpellNameEntrySize-1 {
			return nil, fmt.Errorf("Pool spell name %d has length %d", index, length)
		}
		names = append(names, string(entry[1:1+length]))
	}
	return names, nil
}
