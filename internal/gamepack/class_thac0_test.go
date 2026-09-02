package gamepack

import "testing"

// 表逐位元組照抄 DS:3C16h；這裡固化三組獨立的進程，抓「抄錯一列」。
func TestOriginalClassThac0TableMatchesTheOriginalBytes(t *testing.T) {
	table := OriginalClassThac0Table()
	for _, test := range []struct {
		class int
		typed [ClassThac0MaxLevel]int
	}{
		{0, [10]int{20, 20, 20, 18, 18, 18, 16, 16, 16, 14}},
		{2, [10]int{20, 19, 18, 17, 16, 15, 14, 13, 12, 11}},
		{5, [10]int{20, 20, 20, 20, 20, 19, 19, 19, 19, 19}},
		{6, [10]int{20, 20, 20, 20, 19, 19, 19, 19, 16, 16}},
	} {
		for level := 1; level <= ClassThac0MaxLevel; level++ {
			got := 60 - int(table[test.class][level])
			if got != test.typed[level-1] {
				t.Fatalf("class %d level %d THAC0 %d, want %d", test.class, level, got, test.typed[level-1])
			}
		}
	}
	// 三組相同的列：1 與 0、3／4 與 2、7 與 0。
	for _, pair := range [][2]int{{1, 0}, {3, 2}, {4, 2}, {7, 0}} {
		if table[pair[0]] != table[pair[1]] {
			t.Fatalf("class %d and %d rows differ; the original bytes are identical", pair[0], pair[1])
		}
	}
}

// 多職業取 internal 最大者，也就是 typed 最小、最好的那一個。
func TestBaseThac0InternalTakesTheBestClass(t *testing.T) {
	var levels [ClassThac0ClassCount]uint8
	levels[5] = 10 // 法師 10 級，typed 19
	levels[2] = 4  // 戰士 4 級，typed 17
	got, err := BaseThac0Internal(levels)
	if err != nil {
		t.Fatal(err)
	}
	if 60-int(got) != 17 {
		t.Fatalf("THAC0 %d, want the fighter's 17", 60-int(got))
	}
}

// 全部等級 0 的記錄回 0，與原版由 0 起算一致。
func TestBaseThac0InternalStartsAtZero(t *testing.T) {
	var levels [ClassThac0ClassCount]uint8
	got, err := BaseThac0Internal(levels)
	if err != nil || got != 0 {
		t.Fatalf("got %d err %v, want 0", got, err)
	}
}

func TestBaseThac0InternalRejectsALevelOutsideTheTable(t *testing.T) {
	var levels [ClassThac0ClassCount]uint8
	levels[2] = ClassThac0MaxLevel + 1
	if _, err := BaseThac0Internal(levels); err == nil {
		t.Fatal("a level past the table was accepted")
	}
}
