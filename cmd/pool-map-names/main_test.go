package main

import (
	"path/filepath"
	"testing"
)

// 這一支的價值在於「文字與 NEWECL 在同一條控制流上」。釘住幾個**跨封存檔
// 互相印證**的配對：同一個目的地被兩個不相干的腳本指到，兩邊的提示語講同一
// 個地方，才不是關鍵字碰巧命中（spec 055 的警告）。
func TestMapNamesPairsTextWithItsDestination(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	result, err := collect(zipPath, 14)
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	if len(result.FailedBlocks) != 0 {
		t.Errorf("有 %d 個 block 追不完：%v", len(result.FailedBlocks), result.FailedBlocks[:1])
	}
	near := func(destination int, want string) bool {
		for _, item := range result.Transitions {
			if item.Destination != destination {
				continue
			}
			for _, text := range item.Before {
				if text.Text == want {
					return true
				}
			}
		}
		return false
	}
	for _, testCase := range []struct {
		name        string
		destination int
		text        string
	}{
		// 區塊 0 是第一張圖（GEO3/0）。原版自己叫它「費蘭的文明區」。
		{"區塊 0 ＝ 費蘭文明區（ecl5/7 那條）", 0,
			"FINALLY YOU ENTER THE CIVILIZED AREA OF PHLAN."},
		{"區塊 0 ＝ 搭船回費蘭（ecl4/21 那條）", 0,
			"DO YOU WANT TO TAKE A BOAT BACK TO PHLAN?"},
		// 區塊 28 是海盜據點，區塊 25 是它外面的野外。
		{"區塊 28 ＝ 據點", 28, "THE RIDERS ESCORT YOU INTO THE OUTPOST."},
		{"區塊 25 ＝ 據點外的野外", 25, "YOU ESCAPE FROM THE OUTPOST."},
	} {
		if !near(testCase.destination, testCase.text) {
			t.Errorf("%s：NEWECL %d 附近找不到「%s」",
				testCase.name, testCase.destination, testCase.text)
		}
	}
}

// **`Graph.Edges` 只有分支邊，沒有循序邊。** 只用它回走的話 73 個 NEWECL
// 一段文字都收不到——而那看起來像「原版沒有提示語」，不像工具有洞。
// 這一條守住補的那些循序邊。
func TestBackwardWalkNeedsSequentialEdges(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	result, err := collect(zipPath, 14)
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	withText := 0
	for _, item := range result.Transitions {
		if len(item.Before) != 0 {
			withText++
		}
	}
	if withText == 0 {
		t.Fatal("一個換圖點都沒收到文字——循序邊多半又漏了")
	}
	t.Logf("%d / %d 個換圖點收得到前置文字", withText, len(result.Transitions))
}
