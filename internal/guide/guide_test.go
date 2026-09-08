package guide

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	"github.com/wicanr2/golden-box-remake-engine/geometry"
)

// 兩份攻略要有同一組地圖與同一組座標。少一邊的症狀是「換個語言就少一半的
// 提示」，而那在單一語言下看不出來。
func TestBothCataloguesCoverTheSameCells(t *testing.T) {
	chinese, err := TraditionalChinese()
	if err != nil {
		t.Fatal(err)
	}
	english, err := English()
	if err != nil {
		t.Fatal(err)
	}
	if len(chinese.Maps) != len(english.Maps) {
		t.Fatalf("繁中 %d 張地圖、英文 %d 張", len(chinese.Maps), len(english.Maps))
	}
	for key, zh := range chinese.Maps {
		en, ok := english.Maps[key]
		if !ok {
			t.Fatalf("英文缺地圖 %s", key)
		}
		if len(zh.Points) != len(en.Points) {
			t.Fatalf("%s：繁中 %d 個點、英文 %d 個", key, len(zh.Points), len(en.Points))
		}
		for index := range zh.Points {
			if zh.Points[index].X != en.Points[index].X || zh.Points[index].Y != en.Points[index].Y {
				t.Fatalf("%s 第 %d 個點座標不同：繁中 (%d,%d)、英文 (%d,%d)", key, index,
					zh.Points[index].X, zh.Points[index].Y, en.Points[index].X, en.Points[index].Y)
			}
			if zh.Points[index].Source == "" {
				t.Fatalf("%s 第 %d 個點沒有來源", key, index)
			}
		}
	}
}

// 每一個點都要落在那張地圖上，而且是走得到的格子。
//
// **這一條是防抄攻略的閘門**：攻略的座標抄錯一格的症狀是「測試綠、玩家走不
// 到」，而 game pack 的宣告會蓋過原始資料，兩者在報表上分不出來
// （CoAB `CLAUDE.md` 的通則第 4 條）。這裡直接回對原始 GEO。
func TestEveryGuidePointIsOnAWalkableCell(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	if _, err := os.Stat(zipPath); err != nil {
		t.Skipf("原版 ZIP 刻意不進版控：%v", err)
	}
	catalog, err := gamepack.ReadDOSGeometryCatalog(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	chinese, err := TraditionalChinese()
	if err != nil {
		t.Fatal(err)
	}
	for key, definition := range chinese.Maps {
		var archive, block int
		if _, err := fmt.Sscanf(key, "%d/%d", &archive, &block); err != nil {
			t.Fatalf("地圖鍵 %q 解不開：%v", key, err)
		}
		if _, ok := catalog.Map(gamepack.MapKey{
			Archive: uint8(archive), BlockID: uint8(block)}); !ok {
			t.Fatalf("原版沒有地圖 %s", key)
		}
		for _, point := range definition.Points {
			if point.X < 0 || point.X >= geometry.Width ||
				point.Y < 0 || point.Y >= geometry.Height {
				t.Fatalf("%s 的 %q 在 (%d,%d)，超出 %dx%d",
					key, point.Label, point.X, point.Y, geometry.Width, geometry.Height)
			}
		}
	}
}

// 同一張地圖上不該有兩個點壓在同一格：那多半是產生資料時把兩個索引寫到
// 同一批格子上，而畫出來只看得到一個標記。
func TestNoTwoGuidePointsShareACell(t *testing.T) {
	catalogue, err := TraditionalChinese()
	if err != nil {
		t.Fatal(err)
	}
	for key, definition := range catalogue.Maps {
		seen := map[[2]int]string{}
		for _, point := range definition.Points {
			cell := [2]int{point.X, point.Y}
			if previous, ok := seen[cell]; ok {
				t.Fatalf("%s 的 (%d,%d) 同時是 %q 與 %q", key, point.X, point.Y, previous, point.Label)
			}
			seen[cell] = point.Label
		}
	}
}
