package main

import (
	"github.com/wicanr2/golden-box-remake-engine/graphics"
	"github.com/wicanr2/golden-box-remake-engine/viewport"
)

// 符號編號的五帶（spec 120）。原版的 `Put8x8Symbol`（overlay-35 `015Dh`）
// 逐帶比較，帶的起點在 `DS:2736h`。**決定去哪一組取圖的是編號自己落在哪一帶，
// 不是那一筆 WALLDEF 記錄**——共用 engine 的 `BuildWallLayout` 用記錄決定，
// 所以低號帶那些編號一律算成負索引被丟掉，遠處那道城門就整個不見。
var symbolBandBases = [5]int{0x01, 0x2E, 0x74, 0xBA, 0x100}

// structuralSymbolLimit 是「不畫」的編號上限：`1`、`2`、`3` 不畫。
//
// **這是量出來的，不是讀出來的**：兩張基準圖各掃一次，畫它們會讓
// `(14,1)` 從 98.6% 掉到 95.2%、`(0,4)` 從 97.0% 掉到 92.9%。
// 形狀上也說得通——`(0,4)` 那道門面的最上一列整列是 `01`（原版那裡是天空），
// 拱底那一列中間是 `03`（原版那裡是地面），所以 1..3 像是天／頂／地的標記，
// 背景已經畫過了。切點只能界定在 4..13 之間（這兩張圖裡 4..13 沒出現），
// 為什麼不畫也還沒讀。
const structuralSymbolLimit = 4

// wallSymbolBand 回傳編號屬於第幾帶。
func wallSymbolBand(id uint8) int {
	value := int(id)
	for band := len(symbolBandBases) - 1; band >= 0; band-- {
		if value >= symbolBandBases[band] {
			return band
		}
	}
	return 0
}

// resolveWallStamps 把一次視野走訪的所有 call 展成畫得出來的圖章。
// wallSymbolPicture 取某一帶的 8×8 圖塊集。第 0 帶另外傳進來——它與第 4 帶
// 一樣是開機時載的，不跟著 `37h LOAD PIECES` 換（spec 120）。
func wallSymbolPicture(piece graphics.PieceSet, band int, band0 graphics.Picture) (graphics.Picture, bool) {
	if band == 0 {
		return band0, band0.ItemCount > 0
	}
	for record, id := range piece.SymbolSetIDs {
		if int(id) != band || record >= len(piece.SymbolBlockIDs) {
			continue
		}
		picture, ok := piece.Symbols[piece.SymbolBlockIDs[record]]
		return picture, ok
	}
	return graphics.Picture{}, false
}

func resolveWallStamps(piece graphics.PieceSet, view viewport.WallView,
	band0 graphics.Picture) []graphics.WallStamp {
	pictureFor := func(band int) (graphics.Picture, bool) {
		return wallSymbolPicture(piece, band, band0)
	}
	var result []graphics.WallStamp
	for _, call := range view.Calls {
		raw, err := graphics.RawWallStamps(piece, call.WallType, call.Layout, call.RowStart, call.ColStart)
		if err != nil {
			continue
		}
		for _, stamp := range raw {
			if int(stamp.SymbolID) < structuralSymbolLimit {
				continue
			}
			band := wallSymbolBand(stamp.SymbolID)
			picture, ok := pictureFor(band)
			if !ok {
				continue
			}
			item := int(stamp.SymbolID) - symbolBandBases[band]
			if item < 0 || item >= int(picture.ItemCount) {
				continue
			}
			// 視野只有 11×11 格，落在框外的整片丟掉——它們是側牆延伸出去的
			// 部分，8 像素對齊所以不會只露一半。
			if stamp.Row < 0 || stamp.Row > 10 || stamp.Column < 0 || stamp.Column > 10 {
				continue
			}
			stamp.Picture, stamp.Item = picture, item
			result = append(result, stamp)
		}
	}
	return result
}
