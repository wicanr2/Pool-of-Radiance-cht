package gamepack_test

import (
	"archive/zip"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

func readMember(t *testing.T, name string) []byte {
	t.Helper()
	archive, err := zip.OpenReader(dosZIP)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	defer archive.Close()
	for _, candidate := range archive.File {
		if strings.EqualFold(filepath.Base(candidate.Name), name) {
			reader, err := candidate.Open()
			if err != nil {
				t.Fatal(err)
			}
			defer reader.Close()
			raw, err := io.ReadAll(reader)
			if err != nil {
				t.Fatal(err)
			}
			return raw
		}
	}
	t.Fatalf("the DOS ZIP has no %s", name)
	return nil
}

// 每一份 `.spc` 都要解得開，而且節點數與檔案大小相符。
func TestEveryPremadeEffectListParses(t *testing.T) {
	for name, want := range map[string]int{
		"chrdatd1.spc": 1, "chrdatd2.spc": 1, "chrdatd3.spc": 2,
		"chrdatd4.spc": 2, "chrdatd5.spc": 1, "chrdatd6.spc": 1, "chrdate6.spc": 1,
	} {
		nodes, err := gamepack.ParseEffectList(readMember(t, name))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if len(nodes) != want {
			t.Fatalf("%s has %d nodes, want %d", name, len(nodes), want)
		}
	}
}

// 串列裡除了最後一個節點以外，遠指標都不是零——那是 spec 059 的 `+5` next，
// 也是「9 bytes 是 5 bytes 內容加一個遠指標」這個讀法的證據。
// 讀法若錯位，這個規律不會成立。
func TestEffectNodesChainThroughTheSavedFarPointer(t *testing.T) {
	for _, name := range []string{"chrdatd3.spc", "chrdatd4.spc"} {
		nodes, err := gamepack.ParseEffectList(readMember(t, name))
		if err != nil {
			t.Fatal(err)
		}
		if len(nodes) < 2 {
			t.Fatalf("%s has %d nodes", name, len(nodes))
		}
		for index, node := range nodes {
			last := index == len(nodes)-1
			if last && node.SavedNext != 0 {
				t.Fatalf("%s node %d is last but points to %#08x", name, index, node.SavedNext)
			}
			if !last && node.SavedNext == 0 {
				t.Fatalf("%s node %d is not last but has a null next", name, index)
			}
		}
	}
}

// 大小不是 9 的倍數就失敗即關閉：錯位之後每個代碼都會指到別的效果。
func TestParseEffectListRejectsARaggedFile(t *testing.T) {
	if _, err := gamepack.ParseEffectList(make([]byte, 10)); err == nil {
		t.Fatal("a ragged effect list was accepted")
	}
	if _, err := gamepack.ParseEffectList(nil); err != nil {
		t.Fatalf("an empty effect list was rejected: %v", err)
	}
}

// 效果代碼與已裝備的魔法物品完全對得上，這是「`.spc` 是效果串列」的實證：
// 掃過全部 20 個預設人物，代碼 3Dh 只出現在戴著 Ring of Fire Resistance
//（物品型別 45h）的四個人身上，26h 只出現在戴著 Gauntlets of Ogre Power
//（型別 3Fh）的三個人身上，沒有反例。
//
// 這條同時是負對照：若讀法錯位，代碼會落在別的 byte 上，這種一對一不會成立。
func TestEffectCodesTrackTheReadiedMagicItems(t *testing.T) {
	archive, err := zip.OpenReader(dosZIP)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	defer archive.Close()

	stems := map[string]bool{}
	for _, candidate := range archive.File {
		base := strings.ToLower(filepath.Base(candidate.Name))
		if strings.HasSuffix(base, ".sav") {
			stems[strings.TrimSuffix(base, ".sav")] = true
		}
	}
	if len(stems) < 20 {
		t.Fatalf("the ZIP has %d pre-made characters; the sample is too small to test", len(stems))
	}

	readAny := func(name string) []byte {
		for _, candidate := range archive.File {
			if strings.EqualFold(filepath.Base(candidate.Name), name) {
				reader, err := candidate.Open()
				if err != nil {
					t.Fatal(err)
				}
				defer reader.Close()
				raw, err := io.ReadAll(reader)
				if err != nil {
					t.Fatal(err)
				}
				return raw
			}
		}
		return nil
	}

	for code, itemType := range map[uint8]uint8{0x3D: 0x45, 0x26: 0x3F} {
		withCode, withItem := map[string]bool{}, map[string]bool{}
		for stem := range stems {
			for _, node := range mustParse(t, readAny(stem+".spc")) {
				if node.Code == code {
					withCode[stem] = true
				}
			}
			raw := readAny(stem + ".itm")
			for offset := 0; offset+63 <= len(raw); offset += 63 {
				record := raw[offset : offset+63]
				if record[0x34] != 0 && record[0x2e] == itemType {
					withItem[stem] = true
				}
			}
		}
		if len(withCode) == 0 {
			t.Fatalf("code %#02x never appears; the scan is broken, not the correlation", code)
		}
		for stem := range withCode {
			if !withItem[stem] {
				t.Fatalf("%s has effect %#02x but no readied item of type %#02x", stem, code, itemType)
			}
		}
		for stem := range withItem {
			if !withCode[stem] {
				t.Fatalf("%s wears item type %#02x but has no effect %#02x", stem, itemType, code)
			}
		}
	}
}

func mustParse(t *testing.T, raw []byte) []gamepack.EffectNode {
	t.Helper()
	nodes, err := gamepack.ParseEffectList(raw)
	if err != nil {
		t.Fatal(err)
	}
	return nodes
}
