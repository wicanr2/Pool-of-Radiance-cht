package assets

import (
	"archive/zip"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/wicanr2/golden-box-remake-engine/dax"
	"github.com/wicanr2/golden-box-remake-engine/graphics"
)

// 戰場的地形圖塊（spec 131）。
//
// 戰術地圖本身是生成的（spec 060），但**鋪在每一格上的圖不是**——那三個
// `*COM.DAX` 就是圖塊集，每個 item 24×24，正好一格：
//
//	DUNGCOM.DAX  25 個  地城戰鬥
//	WILDCOM.DAX  34 個  野外戰鬥
//	RANDCOM.DAX   6 個  隨機遭遇
//
// 選哪一個由格位類別表（`DS:2758h`）第四個欄位 `PresentationCode` 決定，
// 而地圖裡存的是類別碼；`PresentationCode` 就是圖塊集裡的 item 序號。
const (
	DungeonCombatTiles     = "DUNGCOM.DAX"
	WildernessCombatTiles  = "WILDCOM.DAX"
	RandomCombatTiles      = "RANDCOM.DAX"
	CombatTerrainTileBlock = 1
	// CombatTerrainTilePixels 是一格的邊長。與戰鬥造形同寬，所以造形直接
	// 蓋在圖塊上就對齊。
	CombatTerrainTilePixels = 24
)

// CombatTerrainItemCounts 是三個圖塊集各自的 item 數。讀出來不是這個數就
// 不是本作的檔——寧可失敗，也不要拿別一款金盒子的圖塊去鋪。
var CombatTerrainItemCounts = map[string]uint8{
	DungeonCombatTiles:    25,
	WildernessCombatTiles: 34,
	RandomCombatTiles:     6,
}

// ReadCombatTerrainTiles 讀一個圖塊集。
func ReadCombatTerrainTiles(zipPath, name string) (graphics.Picture, error) {
	want, ok := CombatTerrainItemCounts[strings.ToUpper(name)]
	if !ok {
		return graphics.Picture{}, fmt.Errorf("%s is not a Pool combat terrain tile set", name)
	}
	archive, err := zip.OpenReader(zipPath)
	if err != nil {
		return graphics.Picture{}, fmt.Errorf("open DOS ZIP: %w", err)
	}
	defer archive.Close()
	var member *zip.File
	for _, candidate := range archive.File {
		if strings.EqualFold(filepath.Base(candidate.Name), name) {
			member = candidate
			break
		}
	}
	if member == nil {
		return graphics.Picture{}, fmt.Errorf("DOS ZIP has no %s", name)
	}
	stream, err := member.Open()
	if err != nil {
		return graphics.Picture{}, fmt.Errorf("open %s: %w", name, err)
	}
	data, readErr := io.ReadAll(io.LimitReader(stream, 1<<20))
	closeErr := stream.Close()
	if readErr != nil {
		return graphics.Picture{}, fmt.Errorf("read %s: %w", name, readErr)
	}
	if closeErr != nil {
		return graphics.Picture{}, fmt.Errorf("close %s: %w", name, closeErr)
	}
	blocks, err := dax.Parse(data)
	if err != nil {
		return graphics.Picture{}, fmt.Errorf("parse %s: %w", name, err)
	}
	for _, block := range blocks {
		if block.Entry.ID != CombatTerrainTileBlock {
			continue
		}
		// **不遮罩**：地形是背景，沒有透明色；遮罩會讓色號 0 變成透明，
		// 而地城地板整格就是色號 0。
		picture, err := graphics.ParsePicture(block.Data, false, 0)
		if err != nil {
			return graphics.Picture{}, fmt.Errorf("%s block %d: %w", name, CombatTerrainTileBlock, err)
		}
		if picture.Width() != CombatTerrainTilePixels ||
			picture.Height() != CombatTerrainTilePixels || picture.ItemCount != want {
			return graphics.Picture{}, fmt.Errorf("%s block %d shape is %dx%dx%d, want %dx%dx%d",
				name, CombatTerrainTileBlock, picture.Width(), picture.Height(), picture.ItemCount,
				CombatTerrainTilePixels, CombatTerrainTilePixels, want)
		}
		return picture, nil
	}
	return graphics.Picture{}, fmt.Errorf("%s has no block %d", name, CombatTerrainTileBlock)
}
