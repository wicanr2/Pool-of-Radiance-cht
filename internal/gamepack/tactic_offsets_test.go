package gamepack

import "testing"

// 六列兩兩成鏡像：1 與 2 是「先往左試」與「先往右試」、3 與 4 是另一組、
// 5 一路往左掃、6 一路往右掃。這個對稱是「步是 1-based」的判準——用 0-based
// 讀出來的六列沒有任何對稱可言（spec 096）。
func TestTacticOffsetsAreMirrorPairs(t *testing.T) {
	offsets, err := ReadDOSTacticOffsets(poolZipPath())
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	want := TacticOffsets{
		{0x08, 0x07, 0x06, 0x01, 0x02},
		{0x08, 0x01, 0x02, 0x07, 0x06},
		{0x07, 0x01, 0x08, 0x06, 0x02},
		{0x01, 0x07, 0x08, 0x02, 0x06},
		{0x08, 0x07, 0x06, 0x05, 0x04},
		{0x08, 0x01, 0x02, 0x03, 0x04},
	}
	if offsets != want {
		t.Fatalf("表是 %v，原版是 %v", offsets, want)
	}
	// 鏡像：模式 1 與 2、3 與 4 的每一步都互為 8 的補數（0 對 0）。
	for _, pair := range [][2]int{{0, 1}, {2, 3}} {
		for step := 0; step < TacticSteps; step++ {
			left := int(offsets[pair[0]][step]) % TacticDirections
			right := int(offsets[pair[1]][step]) % TacticDirections
			if (left+right)%TacticDirections != 0 {
				t.Errorf("模式 %d 與 %d 的第 %d 步是 %d 與 %d，不成鏡像",
					pair[0]+1, pair[1]+1, step+1, left, right)
			}
		}
	}
}

// 方向的算法與模式的輪替。
func TestTacticDirectionAndModeCycle(t *testing.T) {
	offsets, err := ReadDOSTacticOffsets(poolZipPath())
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	// 模式 1 第一步的偏移是 8，mod 8 之後是 0——也就是照基準方向直走。
	if got, err := offsets.Direction(1, 1, 3); err != nil || got != 3 {
		t.Errorf("模式 1 第 1 步從基準 3 算出 %d（%v），應該還是 3", got, err)
	}
	// 第二步是 7，也就是往左一格。
	if got, err := offsets.Direction(1, 2, 0); err != nil || got != 7 {
		t.Errorf("模式 1 第 2 步從基準 0 算出 %d（%v），應該是 7", got, err)
	}
	// 鏡像的模式 2 第二步往右一格。
	if got, err := offsets.Direction(2, 2, 0); err != nil || got != 1 {
		t.Errorf("模式 2 第 2 步從基準 0 算出 %d（%v），應該是 1", got, err)
	}
	if _, err := offsets.Direction(7, 1, 0); err == nil {
		t.Error("模式 7 應該被擋下來")
	}
	if _, err := offsets.Direction(1, 6, 0); err == nil {
		t.Error("第 6 步應該被擋下來")
	}
	for mode, want := range map[int]int{1: 2, 5: 6, 6: 1} {
		if got := NextTacticMode(mode); got != want {
			t.Errorf("模式 %d 的下一個是 %d，應該是 %d", mode, got, want)
		}
	}
}
