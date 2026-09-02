package main

import (
	"path/filepath"
	"testing"
)

func TestPrintableRunsKeepsOffsetsAndDropsShortNoise(t *testing.T) {
	data := []byte{0x00, 'O', 'R', 'C', 0x00, 'K', 'O', 'B', 'O', 'L', 'D', 0x00}
	runs := printableRuns(data, 4)
	if len(runs) != 1 {
		t.Fatalf("got %+v, want only the long run", runs)
	}
	if runs[0].Offset != 5 || runs[0].Text != "KOBOLD" || runs[0].Encoding != "plain" {
		t.Fatalf("run %+v", runs[0])
	}
}

// 詞界比對：LIS 不能命中 LISTEN，但 LIS RIVER 要命中。
func TestContainsWordRespectsWordBoundaries(t *testing.T) {
	if containsWord("stand and listen", "lis") {
		t.Fatal("LIS matched inside LISTEN")
	}
	if !containsWord("the lis river", "lis") {
		t.Fatal("LIS did not match as its own word")
	}
	if !containsWord("sokal keep", "sokal keep") {
		t.Fatal("a two-word term did not match")
	}
}

// 6-bit 解碼會把圖形也解成有字母的字串；虛詞是它們幾乎不會有的東西。
func TestReadableSeparatesProseFromDecodedNoise(t *testing.T) {
	if !readable("THE HARBOR MASTER TELLS YOU BOATS LEAVE FOR THE WEST") {
		t.Fatal("real prose was rejected")
	}
	if readable(`8C#5HD!P1RU"S!8E"T2UC4$8UHEBBT8EBSQ[(HE2UHAC8FS5XAC?0`) {
		t.Fatal("decoded noise was accepted as prose")
	}
}

// 每個 term 的拼法至少有一個，而且變體不得與正名重複。
func TestTermSpellingsAreDistinct(t *testing.T) {
	for _, term := range terms {
		spellings := term.spellings()
		if len(spellings) == 0 || spellings[0] != term.English {
			t.Fatalf("%q spellings %v", term.English, spellings)
		}
		seen := map[string]bool{}
		for _, spelling := range spellings {
			if seen[spelling] {
				t.Fatalf("%q repeats spelling %q", term.English, spelling)
			}
			seen[spelling] = true
		}
	}
}

// 真檔回歸：原版文字裡確實出現的九個專名要有可讀的命中，
// 說明書獨有的八個要是零——兩邊都固定住，才分得開「原版沒有」與「掃描面有洞」。
func TestAuditFindsTheNamesTheOriginalTextActuallyShows(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	result, err := audit(zipPath)
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	if result.PackedStrings == 0 {
		t.Fatal("no packed text was decoded; the scan would report false zeros")
	}
	present := map[string]bool{
		"Phlan": true, "Braccio": true, "Valjevo": true, "Urslingen": true,
		"Sokal": true, "Kobold": true, "Magic-User": true, "Thief": true, "Yarash": true,
	}
	absent := map[string]bool{
		"Sembia": true, "Thentia": true, "Mulmaster": true, "Lis": true,
		"Tesh": true, "Stormy Bay": true, "Twilight": true, "Yulash": true,
	}
	for _, row := range result.Terms {
		switch {
		case present[row.English] && row.Readable == 0:
			t.Fatalf("%q is in the original text but the audit found none", row.English)
		case absent[row.English] && row.Readable != 0:
			t.Fatalf("%q was reported %d times; it is manual-only", row.English, row.Readable)
		}
	}
}
