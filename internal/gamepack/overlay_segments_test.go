package gamepack

import "testing"

// stub segment 的正對照：四條各自獨立解出來的呼叫，查表得到的 overlay 與
// 內容相符（spec 109）。抄錯一個數字的症狀是「這個 far call 查到別顆
// overlay」，而那種錯會產生自洽但完全錯的結論。
func TestOverlayStubSegmentsResolveKnownFarCalls(t *testing.T) {
	segments, err := ReadDOSOverlayStubSegments(poolZipPath())
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	if len(segments) != 38 {
		t.Fatalf("算出 %d 顆 overlay，原版是 38 顆", len(segments))
	}
	for _, want := range []struct {
		segment uint16
		overlay int
		note    string
	}{
		{0x0100, 24, "掛效果／擲骰／豁免"},
		{0x010A, 25, "施法者等級與陣營"},
		{0x0138, 31, "鄰近查詢"},
		{0x013D, 32, "佔格展開"},
		{0x018E, 36, "圖片"},
		{0x0198, 37, "文字框"},
	} {
		if got := segments[want.overlay]; got != want.segment {
			t.Errorf("overlay-%d（%s）的 stub segment 是 %04X，spec 109 寫的是 %04X",
				want.overlay, want.note, got, want.segment)
		}
	}
	// 每顆的值都不同——同一個 segment 對到兩顆的話，查表就沒有意義了。
	seen := make(map[uint16]int, len(segments))
	for index, segment := range segments {
		if other, ok := seen[segment]; ok {
			t.Errorf("overlay-%d 與 overlay-%d 的 stub segment 都是 %04X",
				other, index, segment)
		}
		seen[segment] = index
	}
}
