package main

import (
	"fmt"


	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	"github.com/wicanr2/golden-box-remake-engine/geometry"
	"github.com/wicanr2/golden-box-remake-engine/graphics"
)

// 平面圖用第 4 帶的 8×8 圖塊，十六種牆組合各一張（spec 119／120）。
//
// **原版不畫線。** overlay-30 offset 0 的迴圈每一格算一個編號
// `0104h + 北1 + 東2 + 南4 + 西8`，再把那張圖塊貼上去——所以「牆兩個像素厚」
// 是美術，不是線寬參數。照著畫線的話每一格的最後一個像素都會對不上。
const (
	areaMapSymbolBand = 4
	// 第 4 帶（`0100h..011Eh`）的前二十張是平面圖在用：
	// `0100h`..`0103h` 是隊伍箭頭的四個方向，`0104h`..`0113h` 是牆的十六種組合。
	areaMapArrowBase  = 0x0100 - 0x0100
	areaMapSymbolBase = 0x0104 - 0x0100
)

// composeAreaMap 把 `A)REA` 那一框組成一張 88×88 的索引像素圖。
//
// 走的是 composeFirstPersonInset 的同一條路：先組成原版尺寸的索引圖，
// 再由呼叫端過主題色盤。直接畫到畫面上的話色盤那一層就繞過去了。
// symbols 收第 4 帶本身，不是整個 PieceSet：第 4 帶與第 0 帶一樣是開機時由
// overlay-11 載的全域符號集，**不跟著 `37h LOAD PIECES` 換**（spec 120），
// 所以它不在 `LOAD PIECES` 那份 PieceSet 裡，拿 PieceSet 去查一定落空。
func composeAreaMap(grid geometry.Grid, symbols graphics.Picture,
	spawn gamepack.Spawn) (graphics.Picture, error) {
	if symbols.ItemCount == 0 {
		return graphics.Picture{}, fmt.Errorf("平面圖要的第 %d 帶圖塊沒有載入", areaMapSymbolBand)
	}

	picture := graphics.Picture{
		WidthUnits: FirstPersonInsetSize / 8, HeightUnits: FirstPersonInsetSize, ItemCount: 1,
		Pixels: make([]uint8, FirstPersonInsetSize*FirstPersonInsetSize),
	}
	originX := areaMapOrigin(int(spawn.X), geometry.Width)
	originY := areaMapOrigin(int(spawn.Y), geometry.Height)

	for row := 0; row < areaMapWindow; row++ {
		for column := 0; column < areaMapWindow; column++ {
			cell := grid.CellWrapped(originX+column, originY+row)
			item := areaMapSymbolBase
			for index, weight := range []int{1, 2, 4, 8} {
				if cell.WallDirections[index] != 0 {
					item += weight
				}
			}
			if item >= int(symbols.ItemCount) {
				continue
			}
			blitSymbol(&picture, symbols, item, column*8, row*8)
		}
	}

	// 箭頭最後畫，疊在牆上面——原版也是先跑完整個迴圈才畫它。
	//
	// 圖塊編號是 `0100h + 朝向 / 2`。原版的朝向存 0／2／4／6（八方向裡的四個
	// 正向，`0634h` 那條用 `(朝向+6) mod 8` 算左方就是這個值域），而 remake
	// 存 0..3（spec 076），所以那個除以二在這裡已經做掉了：直接就是索引。
	arrow := areaMapArrowBase + int(spawn.Facing&3)
	if arrow < int(symbols.ItemCount) {
		blitSymbol(&picture, symbols, arrow,
			(int(spawn.X)-originX)*8, (int(spawn.Y)-originY)*8)
	}
	return picture, nil
}

// blitSymbol 把一張 8×8 圖塊貼到索引圖上。透明的兩個值與第一人稱那一側同一套。
func blitSymbol(target *graphics.Picture, symbols graphics.Picture, item, left, top int) {
	for row := 0; row < symbols.Height(); row++ {
		for column := 0; column < symbols.Width(); column++ {
			value, ok := symbols.Pixel(item, column, row)
			if !ok || value == 16 || value == wallSymbolTransparent {
				continue
			}
			x, y := left+column, top+row
			if x < 0 || x >= FirstPersonInsetSize || y < 0 || y >= FirstPersonInsetSize {
				continue
			}
			target.Pixels[y*FirstPersonInsetSize+x] = value
		}
	}
}

// areaMapImage 是 firstPersonInsetImage 的平面圖版本。
func (a *app) areaMapImage() (*graphics.Picture, error) {
	if a.initialMap == nil {
		return nil, fmt.Errorf("initial map is not loaded")
	}
	picture, err := composeAreaMap(a.initialMap.Grid, a.symbolBand4, a.spawn)
	if err != nil {
		return nil, err
	}
	return &picture, nil
}
