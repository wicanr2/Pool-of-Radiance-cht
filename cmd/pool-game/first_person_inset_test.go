package main

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	"github.com/wicanr2/golden-box-remake-engine/graphics"
)

// firstPersonMatchFloor 是目前對得上的比例。
//
// 這不是「差不多就好」的門檻，是**現況的地板**：兩處還沒收掉的差異
// （遠處牆頂多出一條洋紅、正前方那道門洞沒畫）合計約 4%。收掉任何一處都
// 應該把這個數字往上調，掉下來就是有東西壞了。
const firstPersonMatchFloor = 96.0

// 拿原版走到 GEO3/0 (14,1) 朝西的畫面當 oracle，逐格比第一人稱內框。
//
// 這條測試回答的是「差多少」，不是「像不像」——先前 WORKLIST 寫的
// 「牆片縮在框底一條、上面四分之三是純色」是看舊截圖得出的，量過就知道
// 不成立：透視、幾何與絕大部分像素本來就對得上。
func TestFirstPersonInsetMatchesTheDOSShot(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	catalog, err := gamepack.ReadDOSGeometryCatalog(zipPath)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	spawn := gamepack.Spawn{Map: gamepack.MapKey{Archive: 3, BlockID: 0}, X: 14, Y: 1, Facing: 3}
	initial, ok := catalog.Map(spawn.Map)
	if !ok {
		t.Fatal("GEO3 block 0 不在")
	}
	piece, err := gamepack.ReadDOSPieceSet(zipPath, 3, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	inset, err := composeFirstPersonInset(initial.Grid, piece, spawn)
	if err != nil {
		t.Fatal(err)
	}
	want, err := cropDOSShot(filepath.Join("..", "..", "docs", "reference", "original-dos",
		"adventure", "02-first-person-14-1-west.png"), 24, 24, FirstPersonInsetSize, FirstPersonInsetSize)
	if err != nil {
		t.Skipf("原版截圖不可用: %v", err)
	}
	same := 0
	for index := range want {
		if inset.Pixels[index] == want[index] {
			same++
		}
	}
	ratio := float64(same) * 100 / float64(len(want))
	t.Logf("第一人稱內框逐格相同 %d/%d（%.1f%%）", same, len(want), ratio)
	if ratio < firstPersonMatchFloor {
		t.Fatalf("逐格相同 %.1f%%，低於現況地板 %.1f%%", ratio, firstPersonMatchFloor)
	}
}

// cropDOSShot 從 640×400 的原版截圖裁一塊，換算回 320×200 的 EGA 索引。
func cropDOSShot(path string, x, y, width, height int) ([]uint8, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	shot, err := png.Decode(file)
	if err != nil {
		return nil, err
	}
	scale := shot.Bounds().Dx() / 320
	if scale < 1 {
		return nil, fmt.Errorf("截圖寬 %d，換算不出 320 的倍數", shot.Bounds().Dx())
	}
	pixels := make([]uint8, width*height)
	for row := 0; row < height; row++ {
		for column := 0; column < width; column++ {
			point := image.Pt(shot.Bounds().Min.X+(x+column)*scale, shot.Bounds().Min.Y+(y+row)*scale)
			red, green, blue, _ := shot.At(point.X, point.Y).RGBA()
			found := false
			for candidate, colour := range graphics.EGA16 {
				if uint32(colour.R) == red>>8 && uint32(colour.G) == green>>8 && uint32(colour.B) == blue>>8 {
					pixels[row*width+column] = uint8(candidate)
					found = true
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("(%d,%d) 的顏色不在 EGA 16 色裡", x+column, y+row)
			}
		}
	}
	return pixels, nil
}
