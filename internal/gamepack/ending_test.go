package gamepack

import "testing"

// 結局過場的十三行字，逐字對原版的 overlay-18。
func TestEndingScriptMatchesTheOriginalOverlay(t *testing.T) {
	script, err := ReadDOSEndingScript(poolZipPath())
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	want := []string{
		"Mortally wounded, the dragon roars!",
		"The spirit of Tyranthraxus flares up",
		"from the dragon's body.",
		`"Fools, you have but slain the body`,
		`I possessed.  I cannot be defeated!"`,
		"With the power of the pool of radiance",
		"which I moved here, to my lair,",
		"I will still rule, by possessing one",
		`of you!"`,
		`"No Lord Bane!  I can still rule here!`,
		"I have not failed.  Do not call me back",
		`through the pool!"`,
		"Noooo...",
	}
	if len(script.Lines) != len(want) {
		t.Fatalf("讀出 %d 行，原版是 %d 行", len(script.Lines), len(want))
	}
	for index, line := range script.Lines {
		if line.Text != want[index] {
			t.Errorf("第 %d 行（%04Xh）是 %q，原版是 %q",
				index, line.Offset, line.Text, want[index])
		}
		if line.Row < 0x11 || line.Row > 0x16 {
			t.Errorf("第 %d 行畫在第 %d 列，原版只用 11h..16h", index, line.Row)
		}
	}
	// 圖是 `FINAL5.DAX` 的五個區塊，依原版載入的順序。
	if got := script.PictureBlocks; len(got) != 5 ||
		got[0] != 1 || got[1] != 3 || got[2] != 4 || got[3] != 5 || got[4] != 6 {
		t.Errorf("圖序是 %v，原版是 [1 3 4 5 6]", got)
	}
}

// 分頁是列號算出來的：列回到第一列就是新的一頁。原版是三頁。
func TestEndingScriptSplitsIntoOriginalPages(t *testing.T) {
	script, err := ReadDOSEndingScript(poolZipPath())
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	pages := script.Pages()
	if len(pages) != 3 {
		t.Fatalf("切成 %d 頁，原版是 3 頁", len(pages))
	}
	for index, want := range []int{3, 6, 4} {
		if len(pages[index]) != want {
			t.Errorf("第 %d 頁有 %d 行，原版是 %d 行", index+1, len(pages[index]), want)
		}
	}
	if pages[0][0].Text != "Mortally wounded, the dragon roars!" {
		t.Errorf("第一頁第一行是 %q", pages[0][0].Text)
	}
	if last := pages[2][len(pages[2])-1]; last.Text != "Noooo..." {
		t.Errorf("最後一行是 %q", last.Text)
	}
}
