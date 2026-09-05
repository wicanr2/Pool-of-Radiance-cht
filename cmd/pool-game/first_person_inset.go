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
func composeFirstPersonInset(grid geometry.Grid, piece graphics.PieceSet,
	spawn gamepack.Spawn) (graphics.Picture, error) {
	fill, err := poolFirstPersonStageFill()
	if err != nil {
		return graphics.Picture{}, err
	}
	picture := graphics.Picture{
		WidthUnits: FirstPersonInsetSize / 8, HeightUnits: FirstPersonInsetSize, ItemCount: 1,
		Pixels: make([]uint8, FirstPersonInsetSize*FirstPersonInsetSize),
	}
	paint := func(rectangles []viewport.BackgroundRect) {
		for _, rectangle := range rectangles {
			for row := 0; row < rectangle.Height; row++ {
				for column := 0; column < rectangle.Width; column++ {
					x, y := rectangle.X-24+column, rectangle.Y-24+row
					if x < 0 || x >= FirstPersonInsetSize || y < 0 || y >= FirstPersonInsetSize {
						continue
					}
					picture.Pixels[y*FirstPersonInsetSize+x] = rectangle.PaletteIndex
				}
			}
		}
	}
	paint(fill.Backdrop)
	stamps, err := initialWallStamps(grid, piece, spawn)
	if err != nil {
		return graphics.Picture{}, err
	}
	for _, stamp := range stamps {
		for row := 0; row < stamp.Picture.Height(); row++ {
			for column := 0; column < stamp.Picture.Width(); column++ {
				value, ok := stamp.Picture.Pixel(int(stamp.Item), column, row)
				if !ok || value == 16 {
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
	paint(fill.PostWall)
	return picture, nil
}

// firstPersonInsetImage 把上面那張圖轉成現行主題色盤的貼圖。
func (a *app) firstPersonInsetImage() (*graphics.Picture, error) {
	if a.initialMap == nil || a.initialWalls == nil {
		return nil, fmt.Errorf("initial map or wall art is not loaded")
	}
	picture, err := composeFirstPersonInset(a.initialMap.Grid, *a.initialWalls, a.spawn)
	if err != nil {
		return nil, err
	}
	return &picture, nil
}
