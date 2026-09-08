package main

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

)

// 原版的畫面外框（spec 123）。
//
// 那圈紅色的繩索花紋不是畫出來的線，是三個 8×8 符號拼出來的，全部住在
// 全域第 4 帶（`8X8D1.DAX` 區塊 202，符號 `100h..11Eh`，spec 120）：
//
//	114h（item 20）四個角的結
//	115h（item 21）直的繩子——左右兩欄
//	116h（item 22）橫的繩子——上下兩列
//
// 位置是拿 `docs/reference/original-dos/adventure/` 那幾張原版截圖量的：
// 外框佔 320×200 的**第 0 列與第 23 列 tile、第 0 欄與第 39 欄 tile**，
// 也就是 y=0..7／184..191、x=0..7／312..319。最後一列（y=192..199）是空的。
// 五張截圖（空隊伍選單、有隊伍選單、Rolf、自由移動、貧民窟）逐格相同，
// 所以它是**整個遊戲共用的畫面框**，不是某一個畫面的裝飾。
const (
	// frameCornerItem 是四個角。
	frameCornerItem = 0x14
	// frameVerticalItem 是左右兩欄。
	frameVerticalItem = 0x15
	// frameHorizontalItem 是上下兩列。
	frameHorizontalItem = 0x16
	// frameTileSize 是一個符號的邊長。
	frameTileSize = 8
	// frameBottomTileRow 是外框下緣那一列 tile 的編號（0 起算）。
	// 200 ÷ 8 ＝ 25 列，原版用的是第 23 列——**最後一列是空的**。
	frameBottomTileRow = 23
)

// drawFrame 畫畫面外框。band 4 沒載進來（測試的假 app、或原版 ZIP 不在）
// 時退回原本那圈素色方框，畫面不會空掉。
func (a *app) drawFrame(screen *ebiten.Image, foreground, accent color.Color) {
	if a == nil || a.symbolBand4.ItemCount <= frameHorizontalItem {
		drawPlainFrame(screen, a.text(msgFrameTitle), foreground, accent)
		return
	}
	palette := a.artPalette()
	scale := logicalWidth / 320
	columns := logicalWidth / (frameTileSize * scale)
	put := func(item, column, row int) {
		a.drawSymbol(screen, item, column*frameTileSize*scale, row*frameTileSize*scale, scale, palette)
	}
	for column := 1; column < columns-1; column++ {
		put(frameHorizontalItem, column, 0)
		put(frameHorizontalItem, column, frameBottomTileRow)
	}
	for row := 1; row < frameBottomTileRow; row++ {
		put(frameVerticalItem, 0, row)
		put(frameVerticalItem, columns-1, row)
	}
	put(frameCornerItem, 0, 0)
	put(frameCornerItem, columns-1, 0)
	put(frameCornerItem, 0, frameBottomTileRow)
	put(frameCornerItem, columns-1, frameBottomTileRow)
	drawText(screen, a.text(msgFrameTitle), 20, 36, foreground)
}

// drawRopeBox 用同三個符號圍一個框。left／top 是外框左上角的**邏輯座標**，
// columns／rows 是含四邊在內的 tile 數。第一人稱那一框在原版就是這樣圍的
//（外框 (16,16)..(119,119)，內部 88×88 從 (24,24) 起）。
func (a *app) drawRopeBox(screen *ebiten.Image, left, top, columns, rows int) {
	if a == nil || a.symbolBand4.ItemCount <= frameHorizontalItem || columns < 2 || rows < 2 {
		return
	}
	palette := a.artPalette()
	scale := logicalWidth / 320
	step := frameTileSize * scale
	put := func(item, column, row int) {
		a.drawSymbol(screen, item, left+column*step, top+row*step, scale, palette)
	}
	for column := 1; column < columns-1; column++ {
		put(frameHorizontalItem, column, 0)
		put(frameHorizontalItem, column, rows-1)
	}
	for row := 1; row < rows-1; row++ {
		put(frameVerticalItem, 0, row)
		put(frameVerticalItem, columns-1, row)
	}
	put(frameCornerItem, 0, 0)
	put(frameCornerItem, columns-1, 0)
	put(frameCornerItem, 0, rows-1)
	put(frameCornerItem, columns-1, rows-1)
}

// drawSymbol 把第 4 帶的一個 8×8 符號畫上去，放大 scale 倍。
func (a *app) drawSymbol(screen *ebiten.Image, item, left, top, scale int, palette [16]color.RGBA) {
	picture := a.symbolBand4
	width := int(picture.WidthUnits) * 8
	if width == 0 {
		return
	}
	base := item * width * frameTileSize
	for y := 0; y < frameTileSize; y++ {
		for x := 0; x < width && x < frameTileSize; x++ {
			index := base + y*width + x
			if index >= len(picture.Pixels) {
				return
			}
			shade := palette[picture.Pixels[index]&0x0f]
			for dy := 0; dy < scale; dy++ {
				for dx := 0; dx < scale; dx++ {
					screen.Set(left+x*scale+dx, top+y*scale+dy, shade)
				}
			}
		}
	}
}

// drawPlainFrame 是沒有原版素材時的退路。
func drawPlainFrame(screen *ebiten.Image, title string, foreground, accent color.Color) {
	for inset := 8; inset < 12; inset++ {
		for x := inset; x < logicalWidth-inset; x++ {
			screen.Set(x, inset, accent)
			screen.Set(x, logicalHeight-inset-1, accent)
		}
		for y := inset; y < logicalHeight-inset; y++ {
			screen.Set(inset, y, accent)
			screen.Set(logicalWidth-inset-1, y, accent)
		}
	}
	drawText(screen, title, 18, 26, foreground)
}

