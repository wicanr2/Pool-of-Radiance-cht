package gamepack_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// 整張表逐格釘住。這一百個數字就是 AD&D 一版的轉變矩陣：正數是 1d20 要擲到的
// 點數（轉變），0 與負數是自動成功並且直接摧毀，99 是這一級動不了。
//
// 逐格釘住的理由：`欄 × 10 + 列` 這個取法只要 stride 讀錯一格，整張表都會位移
// 而且**仍然看起來像一張合理的表**——右下角照樣比左上角好。
func TestTurnUndeadTableMatchesTheOriginal(t *testing.T) {
	table, err := gamepack.ReadDOSTurnUndeadTable(filepath.Join("..", "..", "Pool of Radiance (1988).zip"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	want := [gamepack.TurnUndeadColumns + 1][gamepack.TurnUndeadRows + 1]int8{
		1:  {1: 10, 7, 4, 1, 1, 0, 0, -1, -1, -1},
		2:  {1: 13, 10, 7, 1, 1, 0, 0, 0, -1, -1},
		3:  {1: 16, 13, 10, 4, 1, 1, 0, 0, 0, -1},
		4:  {1: 19, 16, 13, 7, 4, 1, 1, 0, 0, -1},
		5:  {1: 20, 19, 16, 10, 7, 4, 1, 1, 0, 0},
		6:  {1: 99, 20, 19, 13, 10, 7, 4, 1, 1, 0},
		7:  {1: 99, 99, 20, 16, 13, 10, 7, 4, 1, 0},
		8:  {1: 99, 99, 99, 20, 16, 13, 10, 7, 4, 1},
		9:  {1: 99, 99, 99, 99, 20, 16, 13, 10, 7, 1},
		10: {1: 99, 99, 99, 99, 99, 20, 16, 13, 10, 4},
	}
	for column := 1; column <= gamepack.TurnUndeadColumns; column++ {
		for row := 1; row <= gamepack.TurnUndeadRows; row++ {
			if got := table[column][row]; got != want[column][row] {
				t.Errorf("欄 %d 列 %d 是 %d，原版是 %d", column, row, got, want[column][row])
			}
		}
	}
}

// 牧師等級怎麼壓成列（overlay-13 `11CFh`）。
func TestTurnUndeadRowFoldsClericLevels(t *testing.T) {
	for level := 1; level <= 8; level++ {
		if got := gamepack.TurnUndeadRow(level); got != level {
			t.Errorf("等級 %d 落在第 %d 列，應該是第 %d 列", level, got, level)
		}
	}
	for level := 9; level <= 13; level++ {
		if got := gamepack.TurnUndeadRow(level); got != 9 {
			t.Errorf("等級 %d 落在第 %d 列，9..13 應該併成第 9 列", level, got)
		}
	}
	for _, level := range []int{14, 20, 99} {
		if got := gamepack.TurnUndeadRow(level); got != 10 {
			t.Errorf("等級 %d 落在第 %d 列，14 以上應該是第 10 列", level, got)
		}
	}
}

// 判定的三種結果。門檻的**正負號**就是「轉變」與「摧毀」的分界，
// 這是原版把 AD&D 表上的 `T` 與 `D` 塞進同一個 signed byte 的做法。
func TestTurnUndeadOutcomeSplitsOnTheSign(t *testing.T) {
	table, err := gamepack.ReadDOSTurnUndeadTable(filepath.Join("..", "..", "Pool of Radiance (1988).zip"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	// 第 1 級牧師對骷髏（欄 1）：門檻 10，擲 9 不成、擲 10 成。
	if got := table.Outcome(1, 1, 9); got != gamepack.TurnFails {
		t.Errorf("1 級對骷髏擲 9 是 %v，應該失敗", got)
	}
	if got := table.Outcome(1, 1, 10); got != gamepack.TurnTurns {
		t.Errorf("1 級對骷髏擲 10 是 %v，應該轉變", got)
	}
	// 第 8 級對骷髏：門檻 −1，怎麼擲都摧毀。
	if got := table.Outcome(8, 1, 1); got != gamepack.TurnDestroys {
		t.Errorf("8 級對骷髏擲 1 是 %v，應該摧毀", got)
	}
	// 第 6 級對骷髏：門檻 0，自動成功但只算摧毀。
	if got := table.Outcome(6, 1, 1); got != gamepack.TurnDestroys {
		t.Errorf("6 級對骷髏擲 1 是 %v，應該摧毀", got)
	}
	// 第 1 級對吸血鬼（欄 10）：99，1d20 到不了。
	if got := table.Outcome(1, 10, 20); got != gamepack.TurnFails {
		t.Errorf("1 級對吸血鬼擲 20 是 %v，應該打不到", got)
	}
	// 欄 0（不是不死生物）一律打不到。
	if got := table.Outcome(10, 0, 20); got != gamepack.TurnFails {
		t.Errorf("非不死生物是 %v，應該打不到", got)
	}
}

// `+76h` 只有不死生物非零，而且值就是轉變表的欄位。
func TestUndeadTurnColumnIsSetOnlyForUndead(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	want := map[string]uint8{
		"GIANT SKELETON": 8,
		"VAMPIRE":        10,
		"ORC":            0,
		"KOBOLD":         0,
		"OGRE":           0,
	}
	seen := map[string]bool{}
	for archive := uint8(1); archive <= 8; archive++ {
		for id := 0; id < 256; id++ {
			record, err := gamepack.ReadDOSMonsterRecord(zipPath, archive, uint8(id))
			if err != nil {
				continue
			}
			name := strings.TrimSpace(record.Name)
			column, interesting := want[name]
			if !interesting {
				continue
			}
			if got := record.Raw[gamepack.UndeadTurnColumnOffset]; got != column {
				t.Errorf("%s 的 +%02Xh 是 %d，應該是 %d",
					name, gamepack.UndeadTurnColumnOffset, got, column)
			}
			seen[name] = true
		}
	}
	if len(seen) == 0 {
		t.Skip("original DOS ZIP is intentionally not tracked")
	}
	for name := range want {
		if !seen[name] {
			t.Errorf("原版裡找不到 %s，這一則的正對照失效了", name)
		}
	}
}
