package main

import (
	"fmt"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	"github.com/wicanr2/golden-box-remake-engine/geometry"
	"github.com/wicanr2/golden-box-remake-engine/graphics"
	"github.com/wicanr2/golden-box-remake-engine/viewport"
)

// FirstPersonInsetSize 是第一人稱內框的原生邊長（spec 047）。
const FirstPersonInsetSize = 88

// composeFirstPersonInset 把三段背景、牆片與牆後補層組成一張 88×88 的索引圖。
//
// 畫面與對拍走同一條路：畫的時候只是把這張圖放大兩倍貼上去，測試則拿它直接
// 與原版畫面逐格比。兩邊各寫一份的話，改了畫的那一份而對拍還是綠的——
// 那種綠沒有意義。
// wallSymbolTransparent 是 8×8 牆面符號裡「不要畫、讓背景透出來」的色號
// （spec 126）。
//
// 證據是逐格的：把它畫出來的時候，兩張原版基準圖的差異**全部**是
// 「remake 是 D、原版是那一區的背景」——地平線以下是棕 6、以上是天空 B、
// 城門那邊是灰 7。跳過之後兩張都是 **7744/7744 逐格相同**。
// 原版的截圖調色盤裡也根本沒有洋紅。
//
// **原版是用什麼機制不畫的還沒讀**：`Put8x8Symbol`（overlay-35 `015Dh`）
// 在 `[bp+8]` 非零時走 overlay-36 entry 8（`08F5h`），那一支用
// `[bp+0Eh] & 1` 開遮罩模式並看圖片記錄 `+13h`／`+15h` 的遠指標——
// 遮罩色是哪一個值沒有讀出來，`0Dh` 是從像素反推的。
const wallSymbolTransparent = 0x0D

func composeFirstPersonInset(grid geometry.Grid, piece graphics.PieceSet,
	spawn gamepack.Spawn, band0 graphics.Picture) (graphics.Picture, error) {
	fill, err := poolFirstPersonStageFill()
	if err != nil {
		return graphics.Picture{}, err
	}
	picture := graphics.Picture{
		WidthUnits: FirstPersonInsetSize / 8, HeightUnits: FirstPersonInsetSize, ItemCount: 1,
		Pixels: make([]uint8, FirstPersonInsetSize*FirstPersonInsetSize),
	}
	paint := func(rectangles []viewport.BackgroundRect, onlyBlack bool) {
		for _, rectangle := range rectangles {
			for row := 0; row < rectangle.Height; row++ {
				for column := 0; column < rectangle.Width; column++ {
					x, y := rectangle.X-24+column, rectangle.Y-24+row
					if x < 0 || x >= FirstPersonInsetSize || y < 0 || y >= FirstPersonInsetSize {
						continue
					}
					if onlyBlack && picture.Pixels[y*FirstPersonInsetSize+x] != 0 {
						continue
					}
					picture.Pixels[y*FirstPersonInsetSize+x] = rectangle.PaletteIndex
				}
			}
		}
	}
	paint(fill.Backdrop, false)
	stamps, err := initialWallStamps(grid, piece, spawn, band0)
	if err != nil {
		return graphics.Picture{}, err
	}
	for _, stamp := range stamps {
		for row := 0; row < stamp.Picture.Height(); row++ {
			for column := 0; column < stamp.Picture.Width(); column++ {
				value, ok := stamp.Picture.Pixel(int(stamp.Item), column, row)
				if !ok || value == 16 || value == wallSymbolTransparent {
					continue
				}
				x, y := stamp.Column*8+column, stamp.Row*8+row
				if x < 0 || x >= FirstPersonInsetSize || y < 0 || y >= FirstPersonInsetSize {
					continue
				}
				picture.Pixels[y*FirstPersonInsetSize+x] = value
			}
		}
	}
	// **PostWall 只蓋黑的。** 它存在的理由是「牆片合成之後，斜邊素材的黑底會
	// 重新蓋住頂部角落」（spec 047）——所以要復原的是那些**變成黑色**的格子，
	// 不是整條 row。整條蓋會把真的牆片一起洗掉：城門那一格 `(0,4)` 左上角
	// 有一個 120 格的灰色三角（原版 EGA 7），整條蓋會變成天空，
	// 對拍從 97.6% 掉到 97.0%。
	paint(fill.PostWall, true)
	return picture, nil
}

// firstPersonInsetImage 把上面那張圖轉成現行主題色盤的貼圖。
func (a *app) firstPersonInsetImage() (*graphics.Picture, error) {
	if a.initialMap == nil || a.initialWalls == nil {
		return nil, fmt.Errorf("initial map or wall art is not loaded")
	}
	picture, err := composeFirstPersonInset(a.initialMap.Grid, *a.initialWalls, a.spawn, a.symbolBand0)
	if err != nil {
		return nil, err
	}
	return &picture, nil
}
