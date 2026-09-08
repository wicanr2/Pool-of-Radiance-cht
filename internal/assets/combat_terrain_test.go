package assets_test

import (
	"os"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/assets"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

const terrainZip = "../../Pool of Radiance (1988).zip"

func skipWithoutTerrainZip(t *testing.T) {
	t.Helper()
	if _, err := os.Stat(terrainZip); err != nil {
		t.Skip("原版 ZIP 不在，跳過")
	}
}

func TestCombatTerrainTileSetsAreOneBoardCell(t *testing.T) {
	skipWithoutTerrainZip(t)
	for name, want := range assets.CombatTerrainItemCounts {
		picture, err := assets.ReadCombatTerrainTiles(terrainZip, name)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if picture.Width() != 24 || picture.Height() != 24 {
			t.Fatalf("%s 是 %dx%d，戰場一格是 24x24", name, picture.Width(), picture.Height())
		}
		if picture.ItemCount != want {
			t.Fatalf("%s 有 %d 個 item，原版是 %d", name, picture.ItemCount, want)
		}
	}
}

// 格位類別表第四個欄位是圖塊序號：把它當索引去取圖塊，每一筆都要取得到。
// 地城組的 `PresentationCode` 落在 0..18h 與 22h..27h 兩段，前段對 DUNGCOM、
// 後段對 RANDCOM；野外組落在 0..21h，對 WILDCOM。
func TestPresentationCodesIndexTheTileSets(t *testing.T) {
	skipWithoutTerrainZip(t)
	classes := gamepack.OriginalCombatCellClassTable()
	dungeon, err := assets.ReadCombatTerrainTiles(terrainZip, assets.DungeonCombatTiles)
	if err != nil {
		t.Fatal(err)
	}
	random, err := assets.ReadCombatTerrainTiles(terrainZip, assets.RandomCombatTiles)
	if err != nil {
		t.Fatal(err)
	}
	wilderness, err := assets.ReadCombatTerrainTiles(terrainZip, assets.WildernessCombatTiles)
	if err != nil {
		t.Fatal(err)
	}
	for code := 1; code < 32; code++ {
		presentation := int(classes[code].PresentationCode)
		switch {
		case presentation < int(dungeon.ItemCount):
		case presentation >= 0x22 && presentation-0x22 < int(random.ItemCount):
		default:
			t.Fatalf("室內類別 %02Xh 的圖塊序號 %02Xh 兩個圖塊集都取不到", code, presentation)
		}
	}
	for code := 32; code < len(classes); code++ {
		presentation := int(classes[code].PresentationCode)
		if presentation >= int(wilderness.ItemCount) {
			t.Fatalf("野外類別 %02Xh 的圖塊序號 %02Xh 超出 WILDCOM 的 %d 個",
				code, presentation, wilderness.ItemCount)
		}
	}
}

// 通不過的圖塊集要當場失敗，不能默默拿別的檔頂上。
func TestCombatTerrainRejectsUnknownSet(t *testing.T) {
	if _, err := assets.ReadCombatTerrainTiles(terrainZip, "CBODY.DAX"); err == nil {
		t.Fatal("CBODY.DAX 不是地形圖塊集，應該被擋下來")
	}
}
