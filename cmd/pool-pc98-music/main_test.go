package main

import "testing"

// 增益除以 2 之後超出 16 位元的樣本要被夾住並計數（spec 169〈增益〉）：
// 工具靠這個數字在削波時失敗。
func TestScaleCountsClippedSamples(t *testing.T) {
	out, clipped := scale([]int32{0, 2, -2, 65534, 65536, -65536, -65538})
	want := []int16{0, 1, -1, 32767, 32767, -32768, -32768}
	for index := range want {
		if out[index] != want[index] {
			t.Errorf("第 %d 個樣本是 %d，要 %d", index, out[index], want[index])
		}
	}
	if clipped != 2 {
		t.Errorf("削波 %d 個，要 2（65536 與 −65538）", clipped)
	}
}

// 不是 MZ 執行檔就停：版面偵測在任意位元組上可能碰巧找到一組看起來合理的表。
func TestLoadImageRejectsNonExecutables(t *testing.T) {
	if _, err := loadImage(make([]byte, 64)); err == nil {
		t.Fatal("全零的檔案被當成執行檔")
	}
	raw := make([]byte, 64)
	copy(raw, "MZ")
	raw[8] = 2 // 標頭 2 段 = 32 位元組
	image, err := loadImage(raw)
	if err != nil || len(image) != 32 {
		t.Fatalf("映像 %d 位元組、err=%v，要 32", len(image), err)
	}
}

func TestLevelsOfSilenceAndFullScale(t *testing.T) {
	if peak, rms := levels(make([]int16, 10)); peak != -120 || rms != -120 {
		t.Errorf("靜音是 %v／%v dBFS", peak, rms)
	}
	if peak, _ := levels([]int16{-32768}); peak != 0 {
		t.Errorf("滿刻度峰值是 %v dBFS", peak)
	}
}
