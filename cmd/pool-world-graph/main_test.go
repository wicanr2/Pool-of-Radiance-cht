package main

import (
	"path/filepath"
	"strconv"
	"testing"
)

// 這一條把 spec 101 的圖釘住。世界的接法變了要嘛是解碼退步，要嘛是原始
// 資料換了——兩種都該讓測試紅。
func TestWorldGraphShape(t *testing.T) {
	result, err := build(filepath.Join("..", "..", "Pool of Radiance (1988).zip"))
	if err != nil {
		t.Skip("original DOS ZIP is intentionally not tracked")
	}
	if len(result.ECLBlocks) != 29 || len(result.GEOBlocks) != 29 {
		t.Fatalf("區塊數 ECL %d GEO %d，預期各 29",
			len(result.ECLBlocks), len(result.GEOBlocks))
	}
	targets := map[string][]string{}
	for _, block := range result.ECLBlocks {
		key := blockKey(block.Archive, block.BlockID)
		targets[key] = block.NewECL
	}
	// 26 是接得最廣的那一個：一個區塊接十一個。
	if got := len(targets["7/26"]); got != 11 {
		t.Errorf("ecl7/26 接到 %d 個區塊，預期 11：%v", got, targets["7/26"])
	}
	// 城區只從碼頭那一段接得到 21/26/27，其餘是室內場景與貧民窟。
	want := []string{"8", "11", "20", "21", "26", "27"}
	if got := targets["3/0"]; !sameStrings(got, want) {
		t.Errorf("ecl3/0 接到 %v，預期 %v", got, want)
	}
	exits := map[string][]geoExit{}
	for _, block := range result.GEOBlocks {
		exits[blockKey(block.Archive, block.BlockID)] = block.Exits
	}
	// 城區起點 (0,4) 往西那一個出口，是這一張圖上唯一站得到的邊界出口。
	city := exits["3/0"]
	if len(city) != 28 {
		t.Fatalf("geo3/0 邊界出口 %d 個，預期 28", len(city))
	}
	var start geoExit
	for _, exit := range city {
		if exit.X == 0 && exit.Y == 4 && exit.Facing == "W" {
			start = exit
		}
	}
	if start.Facing == "" {
		t.Fatal("geo3/0 沒有 (0,4)W 這個出口")
	}
	same := 0
	for _, exit := range city {
		if exit.Component == start.Component {
			same++
		}
	}
	if same != 1 {
		t.Errorf("geo3/0 與起點同一個連通元件的出口有 %d 個，預期 1", same)
	}
}

func blockKey(archive, block int) string {
	return strconv.Itoa(archive) + "/" + strconv.Itoa(block)
}

func sameStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for index := range got {
		if got[index] != want[index] {
			return false
		}
	}
	return true
}
