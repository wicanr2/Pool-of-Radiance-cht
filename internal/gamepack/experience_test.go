package gamepack_test

import (
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// 四個有進程的職業，門檻與 AD&D 玩家手冊逐項相同。職業索引沿用
// 角色記錄 `+96h` 起的順序（0 牧師、2 戰士、5 法師、6 賊）。
func TestExperienceThresholdsMatchTheOriginalProgressions(t *testing.T) {
	table, err := gamepack.ReadDOSExperienceTable(dosZIP)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	for class, want := range map[int][]uint32{
		0: {1501, 3001, 6001, 13001, 27501},                       // 牧師
		2: {2001, 4001, 8001, 18001, 35001, 70001, 125001},        // 戰士
		5: {2501, 5001, 10001, 22501, 40001},                      // 法師
		6: {1251, 2501, 5001, 10001, 20001, 42501, 70001, 110001}, // 賊
	} {
		for index, threshold := range want {
			level := index + 2
			got, ok := table.RequiredExperience(class, level)
			if !ok {
				t.Fatalf("class %d level %d is unreachable, want %d", class, level, threshold)
			}
			if got != threshold {
				t.Fatalf("class %d level %d needs %d experience, want %d", class, level, got, threshold)
			}
		}
		// 下一級必須是上限哨兵，否則表比原版長。
		if _, ok := table.RequiredExperience(class, len(want)+2); ok {
			t.Fatalf("class %d can advance past level %d", class, len(want)+1)
		}
	}
}

// 上限直接從哨兵讀出來，與《光芒之池》公認的等級上限相同。
func TestClassLevelCaps(t *testing.T) {
	table, err := gamepack.ReadDOSExperienceTable(dosZIP)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	for class, want := range map[int]int{0: 6, 2: 8, 5: 6, 6: 9} {
		if got := table.MaxLevel(class); got != want {
			t.Fatalf("class %d caps at level %d, want %d", class, got, want)
		}
	}
}

// 索引 1／3／4／7 一整列都是哨兵：它們沒有自己的經驗進程。
// spec 063 說那四個索引在原版證據命名之前不得取名，這裡只記錄事實。
func TestUnnamedClassIndexesHaveNoProgression(t *testing.T) {
	table, err := gamepack.ReadDOSExperienceTable(dosZIP)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	for _, class := range []int{1, 3, 4, 7} {
		if got := table.MaxLevel(class); got != 1 {
			t.Fatalf("class index %d reaches level %d; it was expected to have no progression", class, got)
		}
	}
}

// 壞掉的表要失敗即關閉。
func TestParseExperienceTableRejectsTheWrongSize(t *testing.T) {
	if _, err := gamepack.ParseExperienceTable(make([]byte, 100)); err == nil {
		t.Fatal("a short experience table was accepted")
	}
}
