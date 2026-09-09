package main

import "testing"

// 挑法術那一頁的版面照原版量的（spec 134）。原版是 320×200，remake 的邏輯
// 畫布是它的兩倍，所以每一行的位置就是 native 座標乘二。
//
// **只有標題那一行例外**：原版在 native 14，那個位置在 remake 是框上緣自己
// 的標題列（原版沒有那一行）。
func TestSpellPageMatchesTheOriginalRows(t *testing.T) {
	cases := []struct {
		name   string
		native int
		got    int
	}{
		{"1ST LEVEL 那一行", 46, spellPageLevelRow},
		{"清單第一條", 54, spellPageFirstRow},
		{"清單與資訊之間那條橫線", 131, spellPageRuleBottom},
		{"角色名", 150, spellPageNameRow},
		{"還施得出來", 158, spellPageCanRow},
		{"條數", 166, spellPageCountRow},
	}
	for _, item := range cases {
		if want := item.native * 2; item.got != want {
			t.Errorf("%s 在 %d，原版 native %d ×2 是 %d", item.name, item.got, item.native, want)
		}
	}
	// 左界與縮排：原版 native 9（標題／級別／資訊）與 25（法術名）。
	if spellPageLeft != 18 {
		t.Errorf("左界 %d，原版 native 9 ×2 是 18", spellPageLeft)
	}
	if spellPageIndent != 50 {
		t.Errorf("清單縮排 %d，原版 native 25 ×2 是 50", spellPageIndent)
	}
	// 行距：原版是 8。
	if spellPagePitch != 16 {
		t.Errorf("行距 %d，原版 native 8 ×2 是 16", spellPagePitch)
	}
	// 清單塞得下幾條：原版第一條在 native 54、下一條橫線在 129，
	// (129 − 54) ÷ 8 ＝ 9 行。
	if spellPageLines != 9 {
		t.Errorf("清單 %d 行，原版塞得下 9 行", spellPageLines)
	}
	// 最後一條不能掉到橫線底下。
	if last := spellPageFirstRow + (spellPageLines-1)*spellPagePitch; last >= spellPageRuleBottom {
		t.Errorf("第 %d 條的基線 %d 壓到橫線 %d", spellPageLines, last, spellPageRuleBottom)
	}
}
