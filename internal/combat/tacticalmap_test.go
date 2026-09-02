package combat

import "testing"

// 原版配置 4E9h bytes：7 個標頭 byte 加 50×25 格。
func TestTacticalMapSizeMatchesTheOriginalAllocation(t *testing.T) {
	if TacticalMapSize != 0x4E9 {
		t.Fatalf("map size %d (0x%X), want 0x4E9", TacticalMapSize, TacticalMapSize)
	}
	if TacticalMapCellCount != 0x4E2 {
		t.Fatalf("cell count %d (0x%X), want 0x4E2", TacticalMapCellCount, TacticalMapCellCount)
	}
	if TacticalMapWidth != 50 || TacticalMapHeight != 25 {
		t.Fatalf("grid is %d×%d, want 50×25", TacticalMapWidth, TacticalMapHeight)
	}
}

func TestOutdoorGridIsEntirelyOpenGround(t *testing.T) {
	grid := NewOutdoorTacticalGrid()
	if grid.IgnoreTerrain {
		t.Fatal("the outdoor grid skipped terrain checks; the original writes 0 to +6")
	}
	if len(grid.Terrain) != TacticalMapCellCount {
		t.Fatalf("grid has %d cells, want %d", len(grid.Terrain), TacticalMapCellCount)
	}
	for index, code := range grid.Terrain {
		if code != OpenGroundCellClass {
			t.Fatalf("cell %d is class %d, want %d", index, code, OpenGroundCellClass)
		}
	}
}

// 整面平坦戰場上，走得完就是走得完。
func TestOutdoorGridLetsAFullTraceThrough(t *testing.T) {
	var classes CellClasses
	classes[OpenGroundCellClass] = cellClassOpenGround()
	result, err := TraceMovement(NewOutdoorTacticalGrid(), classes, 0, 0, 6, 3, 10)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Complete {
		t.Fatalf("trace stopped at (%d,%d)", result.X, result.Y)
	}
}

func TestTacticalMapHeaderMatchesTheOriginalWrites(t *testing.T) {
	header := NewTacticalMapHeader()
	if header != (TacticalMapHeader{Field4: 0, Field5: 1, Field6: 0}) {
		t.Fatalf("header %+v", header)
	}
}

// 視窗是 13×5，外層 Y、內層 X，兩層都由負值遞增。
func TestIndoorWindowCellsFollowTheOriginalLoopOrder(t *testing.T) {
	cells := IndoorWindowCells()
	if len(cells) != 13*5 {
		t.Fatalf("window has %d cells, want 65", len(cells))
	}
	if cells[0] != [2]int{-6, -2} {
		t.Fatalf("first cell %v, want [-6 -2]", cells[0])
	}
	if cells[12] != [2]int{6, -2} {
		t.Fatalf("cell 12 %v, want [6 -2]", cells[12])
	}
	if cells[len(cells)-1] != [2]int{6, 2} {
		t.Fatalf("last cell %v, want [6 2]", cells[len(cells)-1])
	}
}
