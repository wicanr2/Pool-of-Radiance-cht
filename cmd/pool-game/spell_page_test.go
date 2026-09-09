package main

import "testing"

// 挑法術那一頁的版面照原版量的（spec 134）。原版是 320×200，remake 的邏輯
// 畫布是它的兩倍。
//
// 兩種座標要分開看：
//
//   - **文字**的常數是基線，`drawText` 畫在 `Row-14 .. Row-1`。原版量到的是
//     文字的最後一列 native `n`，對應的基線是 `2n+2`。
//   - **繩索橫條**的常數是圖塊左上角，直接 `2n`：原版在 native 16 與 128
//     各鋪一列 8×8 的橫繩，畫出來是 17..22 與 129..134。
func TestSpellPageMatchesTheOriginalRows(t *testing.T) {
	texts := []struct {
		name   string
		native int
		got    int
	}{
		{"標題", 14, spellPageTitleRow},
		{"1ST LEVEL 那一行", 46, spellPageLevelRow},
		{"清單第一條", 54, spellPageFirstRow},
		{"角色名", 150, spellPageNameRow},
		{"還施得出來", 158, spellPageCanRow},
		{"條數", 166, spellPageCountRow},
	}
	for _, item := range texts {
		if want := item.native*2 + 2; item.got != want {
			t.Errorf("%s 的基線在 %d，原版 native 最後一列 %d 對應 %d",
				item.name, item.got, item.native, want)
		}
	}
	tiles := []struct {
		name   string
		native int
		got    int
	}{
		{"面板上緣", 8, spellPagePanelTop},
		{"標題與清單之間那條繩索", 16, spellPageRuleTop},
		{"清單與資訊之間那條繩索", 128, spellPageRuleBottom},
	}
	for _, item := range tiles {
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
	// 清單塞得下幾條還沒定案（spec 134 的 DRAFT）：原版那一幀只有八條，
	// 而八條之後、下一條繩索之前還有兩行的空間。這裡先取 9，唯一釘住的是
	// 最後一條不能壓到繩索。
	if spellPageLines != 9 {
		t.Errorf("清單 %d 行，目前取 9", spellPageLines)
	}
	if last := spellPageFirstRow + (spellPageLines-1)*spellPagePitch; last > spellPageRuleBottom {
		t.Errorf("第 %d 條的基線 %d 壓到繩索 %d", spellPageLines, last, spellPageRuleBottom)
	}
}
