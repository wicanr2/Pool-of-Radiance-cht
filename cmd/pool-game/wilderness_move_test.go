package main

import "testing"

// 野外那張八支 `26h ON GOSUB` 的索引從 0 起算，順序是北、東北、東、東南、
// 南、西南、西、西北（ecl7/26 `9A18h` 的 branch_targets），所以四方位是
// 0、2、4、6。用 1、3、5、7 的話每一步都會斜著走。
func TestWildernessFacingIndexPicksTheCardinalArms(t *testing.T) {
	for facing, want := range map[uint8]uint16{0: 0, 1: 2, 2: 4, 3: 6} {
		if got := wildernessFacingIndex(facing); got != want {
			t.Errorf("朝向 %d 的索引是 %d，要 %d", facing, got, want)
		}
	}
}
