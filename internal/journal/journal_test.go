package journal_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/journal"
)

// 內建手冊要能載入，且三章的條目數與說明書印出的一致。
func TestTraditionalChineseCorpusLoads(t *testing.T) {
	corpus, err := journal.TraditionalChinese()
	if err != nil {
		t.Fatal(err)
	}
	if corpus.Locale() != "zh-TW" {
		t.Fatalf("locale %q", corpus.Locale())
	}
	for kind, want := range map[journal.Kind]int{
		journal.Clue: 58, journal.Rumour: 23, journal.Proclamation: 18,
		journal.Appendix: 7,
	} {
		if got := len(corpus.Entries(kind)); got != want {
			t.Fatalf("%s has %d entries, want %d", kind, got, want)
		}
	}
	if corpus.Size() != 106 {
		t.Fatalf("corpus size %d", corpus.Size())
	}
}

// 線索與傳言的編號必須是連續的 1..N：缺號代表切條時漏認一個標題，
// 而遊戲會直接報那個編號，查不到就是玩家看得見的洞。
func TestClueAndRumourNumbersAreContiguous(t *testing.T) {
	corpus, err := journal.TraditionalChinese()
	if err != nil {
		t.Fatal(err)
	}
	for kind, count := range map[journal.Kind]int{journal.Clue: 58, journal.Rumour: 23} {
		for number := 1; number <= count; number++ {
			if _, ok := corpus.Lookup(kind, strconv.Itoa(number)); !ok {
				t.Fatalf("%s %d is missing", kind, number)
			}
		}
	}
}

// 遊戲文字報出的每一個 ENTRY 編號都要真的查得到。這是這批工作的目的：
// 畫面說「成為線索報導 46」，玩家就必須能在遊戲內翻到第 46 條。
func TestEveryEntryNumberQuotedByTheGameExists(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "docs", "audit", "dos-ecl-text-inventory.json"))
	if err != nil {
		t.Skipf("inventory unavailable: %v", err)
	}
	var inventory struct {
		Entries []struct {
			Source string `json:"source"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(raw, &inventory); err != nil {
		t.Fatal(err)
	}
	corpus, err := journal.TraditionalChinese()
	if err != nil {
		t.Fatal(err)
	}
	quoted := regexp.MustCompile(`ENTRY ([0-9]+)`)
	seen := map[string]bool{}
	for _, item := range inventory.Entries {
		for _, match := range quoted.FindAllStringSubmatch(strings.ToUpper(item.Source), -1) {
			seen[match[1]] = true
		}
	}
	if len(seen) == 0 {
		t.Fatal("the inventory quotes no ENTRY numbers; the scan is broken, not the corpus")
	}
	for number := range seen {
		if _, ok := corpus.Lookup(journal.Clue, number); !ok {
			t.Fatalf("the game quotes ENTRY %s but the journal has no such clue", number)
		}
	}
}

// 遊戲派發的公告字號也要查得到。原版在市政廳只印字號，內容在手冊上。
func TestEveryProclamationNumberQuotedByTheGameExists(t *testing.T) {
	corpus, err := journal.TraditionalChinese()
	if err != nil {
		t.Fatal(err)
	}
	// 這九個字號由既有的 City Hall commission 測試以 VM 實跑取得，見 spec 054。
	for _, id := range []string{"CI", "CXXVI", "CX", "CXXXIV", "CLIV", "CXIV", "CCIV", "CXXIX", "CCI"} {
		if _, ok := corpus.Lookup(journal.Proclamation, id); !ok {
			t.Fatalf("the game dispatches proclamation %s but the journal has no such entry", id)
		}
	}
}

// 壞掉的手冊要失敗即關閉，不能安靜地少一章。
func TestParseRejectsAShortCorpus(t *testing.T) {
	if _, err := journal.Parse([]byte(`{"schema":"pool-journal/1","locale":"zh-TW","entries":[]}`)); err == nil {
		t.Fatal("an empty corpus was accepted")
	}
	if _, err := journal.Parse([]byte(`{"schema":"pool-journal/2","entries":[]}`)); err == nil {
		t.Fatal("a foreign schema was accepted")
	}
}
