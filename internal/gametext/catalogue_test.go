package gametext

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

func TestTraditionalChineseCatalogueLoads(t *testing.T) {
	catalogue, err := TraditionalChinese()
	if err != nil {
		t.Fatal(err)
	}
	if catalogue.Locale() != "zh-TW" || catalogue.Size() == 0 {
		t.Fatalf("locale %q size %d", catalogue.Locale(), catalogue.Size())
	}
}

// 沒翻的句子原樣回傳，nil 表也一樣——「沒載入」與「還沒翻」走同一條路徑。
func TestTranslateFallsBackToTheSource(t *testing.T) {
	catalogue, err := TraditionalChinese()
	if err != nil {
		t.Fatal(err)
	}
	if got := catalogue.Translate("NOT IN THE CATALOGUE"); got != "NOT IN THE CATALOGUE" {
		t.Fatalf("untranslated source became %q", got)
	}
	var absent *Catalogue
	if got := absent.Translate("ANY"); got != "ANY" {
		t.Fatalf("nil catalogue returned %q", got)
	}
	if absent.Size() != 0 || absent.Locale() != "" || absent.Sources() != nil {
		t.Fatal("nil catalogue reported contents")
	}
}

func TestParseRejectsBrokenCatalogues(t *testing.T) {
	for name, raw := range map[string]string{
		"wrong schema":  `{"schema":"other/1","locale":"zh-TW","entries":[]}`,
		"empty source":  `{"schema":"pool-game-text/1","entries":[{"source":" ","text":"X"}]}`,
		"empty text":    `{"schema":"pool-game-text/1","entries":[{"source":"A","text":""}]}`,
		"repeat source": `{"schema":"pool-game-text/1","entries":[{"source":"A","text":"X"},{"source":"A","text":"Y"}]}`,
		"not json":      `{`,
	} {
		if _, err := Parse([]byte(raw)); err == nil {
			t.Fatalf("%s was accepted", name)
		}
	}
}

// 每一條原文都必須真的出現在原版資料裡。抄錯一個字的症狀是「畫面上那句沒翻」，
// 而漏翻與抄錯在畫面上分不出來，所以拿盤點檔逐條核對。
//
// 原版的字有兩個來源：ECL 的 6-bit packed 文字（盤點檔）與 **overlay 內嵌的
// 短字串**。結局過場那十三行屬於後者（overlay-18，spec 108），所以兩邊都認。
func TestEverySourceExistsInTheOriginalInventory(t *testing.T) {
	path := filepath.Join("..", "..", "docs", "audit", "dos-ecl-text-inventory.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("text inventory is not present: %v", err)
	}
	var inventory struct {
		Entries []struct {
			Source string `json:"source"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(raw, &inventory); err != nil {
		t.Fatal(err)
	}
	known := make(map[string]bool, len(inventory.Entries))
	for _, entry := range inventory.Entries {
		known[entry.Source] = true
	}
	catalogue, err := TraditionalChinese()
	if err != nil {
		t.Fatal(err)
	}
	if script, err := gamepack.ReadDOSEndingScript(
		filepath.Join("..", "..", "Pool of Radiance (1988).zip")); err == nil {
		for _, line := range script.Lines {
			known[line.Text] = true
		}
	}
	for _, source := range catalogue.Sources() {
		if !known[source] {
			t.Fatalf("catalogue source is not in the original data: %q", source)
		}
	}
}
