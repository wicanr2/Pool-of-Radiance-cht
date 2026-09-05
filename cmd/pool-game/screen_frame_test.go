package main

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// 畫面外框的三個符號要與原版截圖上的那一圈逐格相同（spec 123）。
//
// 這一條不是「渲染出來的畫面像不像」，而是**素材與位置的對照**：把
// `8X8D1.DAX` 區塊 202 的 item 20／21／22 拿去對截圖上四個角、上下兩列、
// 左右兩欄的實際像素。素材對得上、位置也對得上，畫出來才可能對。
func TestScreenFrameSymbolsMatchTheDOSShot(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	_, band4, err := gamepack.ReadDOSGlobalSymbolBands(zipPath)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	width := int(band4.WidthUnits) * 8
	if width != frameTileSize || int(band4.ItemCount) <= frameHorizontalItem {
		t.Fatalf("第 4 帶是 %d 寬、%d item，接不上外框", width, band4.ItemCount)
	}
	symbol := func(item int) []uint8 {
		base := item * width * frameTileSize
		return band4.Pixels[base : base+width*frameTileSize]
	}
	shot := filepath.Join("..", "..", "docs", "reference", "original-dos", "adventure",
		"06-free-move-0-4-west.png")
	// 320×200 的 tile 網格：外框是第 0／23 列、第 0／39 欄。
	const lastColumn = 39
	cases := []struct {
		name         string
		item         int
		column, row  int
	}{
		{"左上角", frameCornerItem, 0, 0},
		{"右上角", frameCornerItem, lastColumn, 0},
		{"左下角", frameCornerItem, 0, frameBottomTileRow},
		{"右下角", frameCornerItem, lastColumn, frameBottomTileRow},
		{"上緣第 1 格", frameHorizontalItem, 1, 0},
		{"上緣第 20 格", frameHorizontalItem, 20, 0},
		{"下緣第 20 格", frameHorizontalItem, 20, frameBottomTileRow},
		{"左緣第 1 列", frameVerticalItem, 0, 1},
		{"左緣第 12 列", frameVerticalItem, 0, 12},
		{"右緣第 12 列", frameVerticalItem, lastColumn, 12},
	}
	for _, testCase := range cases {
		want, err := cropDOSShot(shot,
			testCase.column*frameTileSize, testCase.row*frameTileSize,
			frameTileSize, frameTileSize)
		if err != nil {
			t.Skipf("原版截圖不可用: %v", err)
		}
		got := symbol(testCase.item)
		if diff := firstDifference(got, want); diff >= 0 {
			t.Errorf("%s（item %#x）第 %d 格不同：素材 %d、截圖 %d\n%s",
				testCase.name, testCase.item, diff, got[diff], want[diff],
				sideBySide(got, want))
		}
	}
	// 最後一列 tile（y=192..199）在原版是空的——外框停在第 23 列。
	blank, err := cropDOSShot(shot, 0, (frameBottomTileRow+1)*frameTileSize,
		frameTileSize, frameTileSize)
	if err != nil {
		t.Skipf("原版截圖不可用: %v", err)
	}
	for index, value := range blank {
		if value != 0 {
			t.Fatalf("最後一列 tile 第 %d 格是 %d，原版那一列應該是空的", index, value)
		}
	}
}

func firstDifference(got, want []uint8) int {
	for index := range want {
		if index >= len(got) || got[index] != want[index] {
			return index
		}
	}
	return -1
}

func sideBySide(got, want []uint8) string {
	out := "  素材             截圖\n"
	for row := 0; row < frameTileSize; row++ {
		left, right := "", ""
		for column := 0; column < frameTileSize; column++ {
			left += fmt.Sprintf("%X", got[row*frameTileSize+column])
			right += fmt.Sprintf("%X", want[row*frameTileSize+column])
		}
		out += "  " + left + "         " + right + "\n"
	}
	return out
}
