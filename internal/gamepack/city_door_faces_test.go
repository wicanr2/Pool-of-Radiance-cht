package gamepack

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// 城區那十一扇牆型 9 的門各自通到哪一家店（spec 102，#39）。
//
// 晚上（`49C9 >= 14`）站在街上面對牆型 9 會問
// 「THE DOOR IS LOCKED. DO YOU WANT TO BREAK IN?」（`ecl3/0 9920h`）。
// 那個問句面對的是哪幾棟，靠兩份原版資料就答得完，不必實拍：
//
//  1. GEO3/0 的牆面（哪幾格的哪一面是牆型 9）；
//  2. 門後那一格的 `terrain & 0x7F`，也就是 `ecl3/0 99EBh` 那張 28 項
//     `ON GOTO` 的索引（spec 102 的索引表）。
//
// 答案是**三家店，一家民宅都沒有**：雜貨店五扇、武具店四扇、銀器店兩扇。
// 索引 22（武具店）在圖上有五格，但只有四格有門——第五格從店內側相鄰，
// 街上沒有對著它的門面。
//
// 證據等級 `exact`：兩邊都是原版位元組，沒有推論。
func TestCityDoorFacesMapToTheThreeShops(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	if _, err := os.Stat(zipPath); err != nil {
		t.Skip("original DOS ZIP is intentionally not tracked")
	}
	catalog, err := ReadDOSGeometryCatalog(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	city, ok := catalog.MapByBlock(0)
	if !ok {
		t.Fatal("GEO3/0 is absent")
	}
	// 街上那一側的門面 → 門後那一格的地點索引（spec 102 的表：
	// 19 雜貨店、22 武具店、23 銀器店）。
	want := map[string]int{
		"(14,8)東":  19,
		"(11,9)南":  23,
		"(12,9)南":  19,
		"(13,9)北":  22,
		"(10,10)西": 19,
		"(7,11)東":  22,
		"(10,11)東": 19,
		"(8,12)南":  22,
		"(9,12)北":  19,
		"(10,12)東": 22,
		"(10,12)南": 23,
	}
	facing := [4]string{"北", "東", "南", "西"}
	step := [4][2]int{{0, -1}, {1, 0}, {0, 1}, {-1, 0}}
	got := map[string]int{}
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			for f := 0; f < 4; f++ {
				wall, ok := city.Grid.WallWrapped(x, y, Spawn{Facing: uint8(f)}.Direction())
				if !ok || wall != 9 {
					continue
				}
				// 門有兩面；問句是站在街上（地點索引 0）時才出的那一面。
				if city.Grid.CellWrapped(x, y).Terrain&0x7F != 0 {
					continue
				}
				nx := (x + step[f][0] + 16) % 16
				ny := (y + step[f][1] + 16) % 16
				got[fmt.Sprintf("(%d,%d)%s", x, y, facing[f])] =
					int(city.Grid.CellWrapped(nx, ny).Terrain & 0x7F)
			}
		}
	}
	if len(got) != len(want) {
		keys := make([]string, 0, len(got))
		for key := range got {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		t.Fatalf("街上那一側的牆型 9 門面有 %d 個，預期 %d 個：%v", len(got), len(want), keys)
	}
	for face, index := range want {
		if got[face] != index {
			t.Errorf("%s 的門後是地點索引 %d，預期 %d", face, got[face], index)
		}
	}
}
