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

// 節點 `+3` 打包了四件事（spec 112）：低四位是下效果者的等級、
// 位元 5 已套用、位元 6 原本的陣營、位元 7 施法者的陣營。
func TestEffectNodePacksFourThingsIntoOneByte(t *testing.T) {
	node := gamepack.NewEffectNode(0x0B, 300, 6, true)
	if node.Duration() != 300 {
		t.Fatalf("持續是 %d", node.Duration())
	}
	if node.CasterLevel() != 6 {
		t.Fatalf("等級是 %d", node.CasterLevel())
	}
	if !node.NeedsTeardown() {
		t.Fatal("收尾旗標沒立起來")
	}
	if node.Applied() {
		t.Fatal("剛掛上不該是已套用")
	}
	// 套用：立位元 5，同時把目前的陣營記進位元 6。
	if !node.MarkApplied(1) {
		t.Fatal("第一次套用應該成立")
	}
	if !node.Applied() || node.OriginalSide() != 1 {
		t.Fatalf("套用之後 %+v", node)
	}
	if node.MarkApplied(0) {
		t.Fatal("套過的不該重複套")
	}
	// 等級仍讀得出來——四件事共用一個 byte，互相不能踩到。
	if node.CasterLevel() != 6 {
		t.Fatalf("套用之後等級變成 %d", node.CasterLevel())
	}
	node.SetDuration(1)
	if node.Duration() != 1 {
		t.Fatalf("改持續之後是 %d", node.Duration())
	}
}

// `+3` 整個是 `0FFh` 代表解除魔法解不掉。那與「等級 15」不同——
// 只看低四位會把它讀成 15 級，然後解得掉。
func TestUndispellableIsTheWholeByteNotTheLowNibble(t *testing.T) {
	node := gamepack.NewEffectNode(0x3D, 0, gamepack.EffectUndispellable, false)
	if !node.Undispellable() {
		t.Fatal("0FFh 應該是解不掉")
	}
	if node.CasterLevel() != 0 {
		t.Fatalf("解不掉的節點不該回等級 %d", node.CasterLevel())
	}
	level15 := gamepack.NewEffectNode(0x3D, 0, 15, false)
	if level15.Undispellable() {
		t.Fatal("等級 15 不是解不掉")
	}
}

// 新的接在尾端，線性搜尋找到的是最早掛上的那一個。
func TestEffectListAppendsAtTheTail(t *testing.T) {
	var list gamepack.EffectList
	list = list.Append(gamepack.NewEffectNode(0x34, 10, 3, false))
	list = list.Append(gamepack.NewEffectNode(0x35, 20, 5, false))
	list = list.Append(gamepack.NewEffectNode(0x34, 30, 7, false))
	if len(list) != 3 || list[2].Duration() != 30 {
		t.Fatalf("串列是 %+v", list)
	}
	index, ok := list.IndexOf(0x34)
	if !ok || index != 0 || list[index].CasterLevel() != 3 {
		t.Fatalf("找到的是第 %d 個（%v）", index, ok)
	}
	list = list.Remove(0x34)
	if len(list) != 2 || list[0].Code != 0x35 {
		t.Fatalf("拿掉之後是 %+v", list)
	}
	if !list.Has(0x34) {
		t.Fatal("第二個 34h 應該還在——Remove 只拿掉一個")
	}
}

// 解除魔法的成功率不對稱：高一級加 5、低一級只扣 2。
func TestDispelChanceIsAsymmetric(t *testing.T) {
	for _, item := range [3][3]int{{6, 6, 50}, {8, 6, 60}, {6, 8, 46}} {
		if got := gamepack.DispelChance(item[0], item[1]); got != item[2] {
			t.Fatalf("施法者 %d 對效果 %d 是 %d，預期 %d", item[0], item[1], got, item[2])
		}
	}
}

// 每一個節點各擲一次；`0FFh` 的跳過不擲。
func TestDispelRollsOncePerNode(t *testing.T) {
	list := gamepack.EffectList{
		gamepack.NewEffectNode(0x34, 10, 3, false),
		gamepack.NewEffectNode(0x3D, 10, gamepack.EffectUndispellable, false),
		gamepack.NewEffectNode(0x35, 10, 3, false),
	}
	rolls := 0
	// 必成：骰 1 一定小於等於任何成功率。
	after, removed := list.Dispel(6, func() int { rolls++; return 1 })
	if rolls != 2 {
		t.Fatalf("擲了 %d 次，預期 2 次（解不掉的那個不擲）", rolls)
	}
	if removed != 2 || len(after) != 1 || after[0].Code != 0x3D {
		t.Fatalf("剩下 %+v，拿掉 %d 個", after, removed)
	}
	// 必敗：骰 100 大於施法者 6 對效果 3 的 65。
	_, removed = list.Dispel(6, func() int { return 100 })
	if removed != 0 {
		t.Fatalf("必敗卻拿掉了 %d 個", removed)
	}
}
