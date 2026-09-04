package main

import (
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gametext"
)

// dosMonsterNames 掃出 `MON1CHA`..`MON8CHA` 全部記錄的名字。原版把同一種怪物
// 放在好幾個 archive（`KOBOLD` 在 mon2、mon7 都有），這裡只要名字的集合。
func dosMonsterNames(t *testing.T) []string {
	t.Helper()
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	seen := map[string]bool{}
	for archive := 1; archive <= 8; archive++ {
		for id := 0; id < 256; id++ {
			record, err := gamepack.ReadDOSMonsterRecord(zipPath, uint8(archive), uint8(id))
			if err != nil {
				continue
			}
			if name := strings.TrimSpace(record.Name); name != "" {
				seen[name] = true
			}
		}
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// 戰鬥與遭遇畫面顯示的是 `MONnCHA` 的名字。少一條，中文畫面上就會冒出
// 一行英文——而那在畫面上看起來像漏翻，不像資料缺了一列。
func TestEveryDOSMonsterNameHasATranslation(t *testing.T) {
	names := dosMonsterNames(t)
	if len(names) == 0 {
		t.Skip("original DOS ZIP is intentionally not tracked")
	}
	catalogue, err := gametext.TraditionalChineseMonsters()
	if err != nil {
		t.Fatal(err)
	}
	var missing []string
	for _, name := range names {
		if _, ok := catalogue.Entry(name); !ok {
			missing = append(missing, name)
		}
	}
	if len(missing) != 0 {
		t.Errorf("%d 個怪物名沒有譯名：%v", len(missing), missing)
	}
	t.Logf("原版怪物名 %d 種，全部有譯名", len(names))
}

// 反過來也要成立：表裡不該有對不到任何原版記錄的條目。多出來的條目是
// 改錯字或抄錯名，而那一條永遠不會被用到，畫面上看不出來。
func TestMonsterNameCatalogueHasNoStrayEntries(t *testing.T) {
	names := dosMonsterNames(t)
	if len(names) == 0 {
		t.Skip("original DOS ZIP is intentionally not tracked")
	}
	known := map[string]bool{}
	for _, name := range names {
		known[name] = true
	}
	catalogue, err := gametext.TraditionalChineseMonsters()
	if err != nil {
		t.Fatal(err)
	}
	var stray []string
	for _, source := range catalogue.Sources() {
		if !known[source] {
			stray = append(stray, source)
		}
	}
	sort.Strings(stray)
	if len(stray) != 0 {
		t.Errorf("%d 個條目對不到原版記錄：%v", len(stray), stray)
	}
}

// 每一條都要說得出依據。`exact` 是說明書原書譯名或遊戲內文字用過的譯法，
// `strong inference` 是由已定名的成分組出來、或 AD&D 通行譯名。
func TestMonsterNameCatalogueRecordsItsEvidence(t *testing.T) {
	catalogue, err := gametext.TraditionalChineseMonsters()
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, source := range catalogue.Sources() {
		entry, ok := catalogue.Entry(source)
		if !ok {
			t.Fatalf("%q 列在 Sources 卻取不出來", source)
		}
		if strings.TrimSpace(entry.Note) == "" {
			t.Errorf("%q（%s）沒有寫來源", source, entry.Text)
		}
		counts[entry.Basis]++
	}
	if counts["exact"] == 0 || counts["strong inference"] == 0 {
		t.Errorf("證據等級分布看起來不對：%v", counts)
	}
	t.Logf("證據等級：exact %d、strong inference %d", counts["exact"], counts["strong inference"])
}
