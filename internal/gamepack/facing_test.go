package gamepack

import (
	"path/filepath"
	"testing"
)

// Pool 的朝向是 0 北、1 東、2 南、3 西（spec 076）。證據是導覽自己走出來的：
// Gold Box 的導覽一步一步往前，所以每一步的位移應該等於上一步的朝向。
// 換成別的對應馬上會出現不符，所以這是可以被推翻的預測。
func TestTourMovementProvesTheFacingConvention(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	event, err := ReadDOSInitialEvent(zipPath)
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	delta := [4][2]int{{0, -1}, {1, 0}, {0, 1}, {-1, 0}}
	agree := 0
	previous := event.Position
	for index, step := range event.Tour {
		dx := int(step.Position.X) - int(previous.X)
		dy := int(step.Position.Y) - int(previous.Y)
		if dx == 0 && dy == 0 {
			previous = step.Position
			continue
		}
		want := delta[previous.Facing]
		if dx != want[0] || dy != want[1] {
			t.Fatalf("tour step %d moved (%+d,%+d) while facing %d, want (%+d,%+d)",
				index, dx, dy, previous.Facing, want[0], want[1])
		}
		agree++
		previous = step.Position
	}
	if agree != 20 {
		t.Fatalf("%d tour steps moved, want 20", agree)
	}
}

// 原版 35 個位置的朝向只出現 0..3，一次都沒有 4..7。
func TestOriginalPositionsNeverFaceBeyondThree(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	event, err := ReadDOSInitialEvent(zipPath)
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	positions := append([]Spawn{event.Position}, func() []Spawn {
		out := make([]Spawn, 0, len(event.Tour))
		for _, step := range event.Tour {
			out = append(out, step.Position)
		}
		return out
	}()...)
	if len(positions) != 35 {
		t.Fatalf("the original data has %d positions, want 35", len(positions))
	}
	for index, position := range positions {
		if position.Facing > 3 {
			t.Fatalf("position %d faces %d", index, position.Facing)
		}
	}
}

// 換算給共用 engine 的方向是乘 2；engine 服務兩個作品，換算留在 Pool 這一側。
func TestSpawnDirectionDoublesTheFacing(t *testing.T) {
	for facing, want := range map[uint8]int{0: 0, 1: 2, 2: 4, 3: 6} {
		if got := (Spawn{Facing: facing}).Direction(); got != want {
			t.Fatalf("facing %d maps to direction %d, want %d", facing, got, want)
		}
	}
}
