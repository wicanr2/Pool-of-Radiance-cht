package main

import (
	"path/filepath"
	"testing"
)

// 三張野外圖的地點表。數字是量到的，變了要嘛是解碼退步，要嘛是表位址讀錯。
func TestWildernessSheetsDecode(t *testing.T) {
	result, err := build(filepath.Join("..", "..", "Pool of Radiance (1988).zip"))
	if err != nil {
		t.Skip("original DOS ZIP is intentionally not tracked")
	}
	want := map[int]struct{ rows, places int }{
		25: {8, 23},
		26: {9, 14},
		27: {5, 9},
	}
	for _, sheet := range result.Sheets {
		expected, ok := want[sheet.BlockID]
		if !ok {
			t.Errorf("多了一張圖 ecl%d/%d", sheet.Archive, sheet.BlockID)
			continue
		}
		if sheet.Rows != expected.rows || len(sheet.Places) != expected.places {
			t.Errorf("ecl%d/%d 有 %d 列 %d 個地點，預期 %d 列 %d 個",
				sheet.Archive, sheet.BlockID, sheet.Rows, len(sheet.Places),
				expected.rows, expected.places)
		}
	}
}

// 碼頭那三條航線的登陸座標，每一個都要在地點表裡找得到。
// 這是這份解碼的語意交叉核對：座標是從別的區塊（ecl3/0 的碼頭）讀出來的。
func TestBoatLandingsAreWildernessPlaces(t *testing.T) {
	result, err := build(filepath.Join("..", "..", "Pool of Radiance (1988).zip"))
	if err != nil {
		t.Skip("original DOS ZIP is intentionally not tracked")
	}
	places := map[[3]int]int{}
	for _, sheet := range result.Sheets {
		for _, item := range sheet.Places {
			places[[3]int{sheet.BlockID, item.X, item.Y}] = item.LocationID
		}
	}
	for _, landing := range []struct {
		name              string
		block, x, y, want int
	}{
		// ecl3/0 `9C04h`／`9C19h`／`9C2Eh`：選完航線之後設 49C3／49C4 再 NEWECL。
		{"EAST", 26, 7, 29, 7},
		{"WEST", 26, 13, 27, 3},
		{"BAY", 27, 9, 29, 3},
	} {
		got, ok := places[[3]int{landing.block, landing.x, landing.y}]
		if !ok {
			t.Errorf("%s 的登陸點 ecl?/%d (%d,%d) 不在地點表裡",
				landing.name, landing.block, landing.x, landing.y)
			continue
		}
		if got != landing.want {
			t.Errorf("%s 的登陸點是地點 %d，預期 %d", landing.name, got, landing.want)
		}
	}
}
