package gamepack

import "fmt"

// 結局過場的圖（spec 108）。overlay-18 entry 1 在第一頁台詞之後，
// 依序把 `FINAL5.DAX` 的五個區塊畫進 `DS:4980h` 那個 168×168 的圖片緩衝。
//
// 位置就在繪製常式的引數裡，**X 與 Y 都是 8 像素為單位**：
// overlay-36 `0D9Fh` 的 X 與來源／目標的 `+2`（寬，單位同樣是 8 像素）比大小，
// Y 則在 `0F1Fh` 先 `shl 3` 才用。三張小圖因此正好落在 120×120 那張場景的
// 下緣裡——單位讀錯的話它們會掉到畫面外。
//
// 後三張畫不畫由 `[4937h] + 67Ch` 決定，那是**隊伍人數**（spec 084、091）。
// 所以隊伍愈多人，結局畫面上的人也愈多。
const (
	// EndingSceneWidth 與 EndingSceneHeight 是兩張底圖的尺寸。
	EndingSceneWidth  = 120
	EndingSceneHeight = 120
	// EndingPositionUnit 是引數的單位：X 與 Y 都乘 8 才是像素。
	EndingPositionUnit = 8
)

// EndingLayer 是結局畫面上的一層。
type EndingLayer struct {
	// Block 是 `FINAL5.DAX` 的區塊編號。
	Block uint8
	// X、Y 是像素座標，已由引數的 8 像素單位換算過。
	X, Y int
	// MinimumPartySize 是這一層要出現所需的隊伍人數；0 代表一定畫。
	MinimumPartySize int
}

// EndingLayers 是原版的五層，順序就是原版畫的順序。
//
// 引數逐一照抄 overlay-18 `036Fh`、`03E4h`、`043Eh`、`0499h`、`04F5h`：
// (0,0)、(0,0)、(0,11)、(13,9)、(0,9)，乘 8 之後是下面的像素座標。
func EndingLayers() []EndingLayer {
	return []EndingLayer{
		{Block: 1, X: 0, Y: 0},
		{Block: 3, X: 0, Y: 0},
		{Block: 4, X: 0, Y: 11 * EndingPositionUnit, MinimumPartySize: 2},
		{Block: 5, X: 13 * EndingPositionUnit, Y: 9 * EndingPositionUnit, MinimumPartySize: 3},
		{Block: 6, X: 0, Y: 9 * EndingPositionUnit, MinimumPartySize: 4},
	}
}

// VisibleEndingLayers 依隊伍人數挑出要畫的那幾層。
//
// 原版的比較是 `>`：`cmp word ptr es:[di+67Ch], 1 / jbe 跳過`，
// 所以人數要**大於** 1 才畫第三層。這裡把它寫成「人數 >= 2」，同一回事。
func VisibleEndingLayers(partySize int) []EndingLayer {
	layers := EndingLayers()
	visible := make([]EndingLayer, 0, len(layers))
	for _, layer := range layers {
		if partySize < layer.MinimumPartySize {
			continue
		}
		visible = append(visible, layer)
	}
	return visible
}

// EndingLayerBlocks 是五層用到的區塊編號，載入時照這份取。
func EndingLayerBlocks() []uint8 {
	seen := map[uint8]bool{}
	var blocks []uint8
	for _, layer := range EndingLayers() {
		if seen[layer.Block] {
			continue
		}
		seen[layer.Block] = true
		blocks = append(blocks, layer.Block)
	}
	return blocks
}

// CheckEndingLayerFits 是失敗即關閉的邊界檢查：每一層都要完整落在 120×120 的
// 場景裡。位置的單位若讀錯（例如把 8 像素當成 1 像素），這裡會擋下來。
func CheckEndingLayerFits(layer EndingLayer, width, height int) error {
	if layer.X < 0 || layer.Y < 0 ||
		layer.X+width > EndingSceneWidth || layer.Y+height > EndingSceneHeight {
		return fmt.Errorf("Pool ending block %d is %dx%d at (%d,%d), outside the %dx%d scene",
			layer.Block, width, height, layer.X, layer.Y, EndingSceneWidth, EndingSceneHeight)
	}
	return nil
}
