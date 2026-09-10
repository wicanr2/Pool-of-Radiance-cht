package main

import (
	"testing"

	"github.com/wicanr2/golden-box-remake-engine/graphics"
)

// 造一張整片同一個色號的 320×200，用來看它被貼到哪一半。
func solidBlock(colour uint8) graphics.Picture {
	picture := graphics.Picture{
		WidthUnits:  titleBlockWidth / 8,
		HeightUnits: titleBlockHeight,
		ItemCount:   1,
	}
	picture.Pixels = make([]uint8, titleBlockWidth*titleBlockHeight)
	for index := range picture.Pixels {
		picture.Pixels[index] = colour
	}
	return picture
}

// 標題是兩個區塊左右並排，**區塊 1 在左、區塊 2 在右**（spec 001）。
// 順序反了畫面就是左右顛倒的，而那在檔案大小與尺寸上看不出來。
func TestTitleAtlasPutsBlockOneOnTheLeft(t *testing.T) {
	pictures := map[uint8]graphics.Picture{1: solidBlock(1), 2: solidBlock(2)}
	atlas, rendered, err := composeTitleAtlas(pictures)
	if err != nil {
		t.Fatal(err)
	}
	if got := atlas.Bounds().Dx(); got != titleBlockWidth*2 {
		t.Fatalf("整張寬 %d，該是 %d", got, titleBlockWidth*2)
	}
	if got := atlas.Bounds().Dy(); got != titleBlockHeight {
		t.Fatalf("整張高 %d，該是 %d", got, titleBlockHeight)
	}
	if len(rendered) != len(titleBlocks) {
		t.Fatalf("回傳 %d 張，該是 %d 張", len(rendered), len(titleBlocks))
	}
	left, right := graphics.EGA16[1], graphics.EGA16[2]
	// 取兩半的中心點——貼歪一格看不出來，貼錯一半看得出來。
	if got := atlas.RGBAAt(titleBlockWidth/2, titleBlockHeight/2); got != left {
		t.Fatalf("左半是 %v，該是區塊 1 的 %v", got, left)
	}
	if got := atlas.RGBAAt(titleBlockWidth+titleBlockWidth/2, titleBlockHeight/2); got != right {
		t.Fatalf("右半是 %v，該是區塊 2 的 %v", got, right)
	}
	// 反對照：把兩張對調，同一組斷言必須失敗——否則上面兩個點證明不了順序。
	swapped, _, err := composeTitleAtlas(map[uint8]graphics.Picture{1: solidBlock(2), 2: solidBlock(1)})
	if err != nil {
		t.Fatal(err)
	}
	if swapped.RGBAAt(titleBlockWidth/2, titleBlockHeight/2) == left {
		t.Fatal("兩張對調之後左半沒變，這組斷言測不到順序")
	}
}

// 缺區塊要當場說出來。少一塊的整張圖是半黑的，看起來像「原版就長這樣」。
func TestTitleAtlasRefusesAMissingBlock(t *testing.T) {
	if _, _, err := composeTitleAtlas(map[uint8]graphics.Picture{1: solidBlock(1)}); err == nil {
		t.Fatal("缺了區塊 2 卻沒報錯")
	}
}
