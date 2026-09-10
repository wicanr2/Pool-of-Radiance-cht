package main

import (
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 判準是「畫出來有東西」，不是「取得到字模格」：ASCII 那條路對任何 <= 0xFF
// 的碼位都取得到格子，但格子可能整片空白——那在畫面上與缺字沒有分別。
func TestBlankTellsAnEmptyCellFromAnInkedOne(t *testing.T) {
	if !blank(nil) {
		t.Fatal("沒有字模格卻不算空白")
	}
	empty := image.NewAlpha(image.Rect(0, 0, 8, 15))
	if !blank(empty) {
		t.Fatal("整片 0 的字模格不算空白")
	}
	// 一個像素就夠——缺字的判準是「一點墨都沒有」。
	inked := image.NewAlpha(image.Rect(0, 0, 8, 15))
	inked.Pix[len(inked.Pix)/2] = 1
	if blank(inked) {
		t.Fatal("有一個非零像素卻被當成空白")
	}
}

// 空白與控制字元不算缺字，其餘都要問字型。少了這一條，報表會被一堆玩家
// 看不到的碼位灌滿。
func TestSkipRuneOnlySkipsBlanksAndControls(t *testing.T) {
	for _, r := range []rune{' ', 0x09, 0x3000, 0x0A, 0x0D, 0x00, 0x7F} {
		if !skipRune(r) {
			t.Errorf("%U 該跳過", r)
		}
	}
	for _, r := range []rune{'A', '0', '，', '光', '芒'} {
		if skipRune(r) {
			t.Errorf("%q 不該跳過——它會被畫出來", r)
		}
	}
}

// **只掃字串常值，不掃註解。** 註解裡的字玩家看不到，算進去就是假的缺字；
// 而這兩種在原始碼裡長得很像，所以要有反對照。
func TestSourceScanReadsLiteralsAndSkipsComments(t *testing.T) {
	root := t.TempDir()
	source := "package sample\n" +
		"// 註解裡的字：霤\n" +
		"const Greeting = \"畫面上的字\"\n"
	if err := os.WriteFile(filepath.Join(root, "sample.go"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	// 這幾個目錄不掃：`docs` 底下是文件、`workplace` 是產物，兩者都不會被畫。
	for _, skipped := range []string{"docs", "workplace"} {
		dir := filepath.Join(root, skipped)
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "x.go"),
			[]byte("package x\nconst A = \"鑀\"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	seen := []string{}
	if err := noteSourceStrings(root, func(text, label string) {
		seen = append(seen, text)
	}); err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(seen, " ")
	if !strings.Contains(joined, "畫面上的字") {
		t.Fatalf("字串常值沒被掃到：%v", seen)
	}
	if strings.Contains(joined, "霤") {
		t.Fatalf("註解被算進去了：%v", seen)
	}
	if strings.Contains(joined, "鑀") {
		t.Fatalf("docs／workplace 被掃了：%v", seen)
	}
}

// 語法錯的 .go 要當場說出來，不能默默跳過——跳過的症狀是「那個檔的字全部
// 沒被檢查」，而報表照樣說沒有缺字。
func TestSourceScanFailsOnUnparseableGo(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "broken.go"),
		[]byte("package broken\nfunc ("), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := noteSourceStrings(root, func(string, string) {}); err == nil {
		t.Fatal("語法錯的檔案沒有讓掃描失敗")
	}
}
