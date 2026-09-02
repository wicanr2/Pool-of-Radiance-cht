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

// 地城格原點的斜向投影，X 也吃 dy 的位移。
func TestIndoorTacticalCellProjection(t *testing.T) {
	x, y, ok := IndoorTacticalCell(0, 0, 0, 0)
	if !ok || x != IndoorOriginX || y != IndoorOriginY {
		t.Fatalf("origin cell (%d,%d) ok=%v", x, y, ok)
	}
	x, _, _ = IndoorTacticalCell(1, 0, 0, 0)
	if x != IndoorOriginX+IndoorStepXPerColumn {
		t.Fatalf("one column right gave x=%d", x)
	}
	x, y, _ = IndoorTacticalCell(0, 1, 0, 0)
	if x != IndoorOriginX+IndoorStepXPerRow || y != IndoorOriginY+IndoorStepYPerRow {
		t.Fatalf("one row down gave (%d,%d); the projection is skewed, X must shift too", x, y)
	}
}

// 視窗的兩個角落算出來落在盤面外，原版整格不寫。
func TestIndoorTacticalCellDropsCellsOutsideTheBoard(t *testing.T) {
	if _, _, ok := IndoorTacticalCell(IndoorWindowMinX, IndoorWindowMinY, 0, 0); ok {
		t.Fatal("the far top-left dungeon cell landed on the board")
	}
	if _, _, ok := IndoorTacticalCell(IndoorWindowMaxX, IndoorWindowMaxY, 4, 5); ok {
		t.Fatal("the far bottom-right dungeon cell landed on the board")
	}
}

// 建構器填地板用 16h，存進地圖是 17h——與室外整面填的類別相同。
func TestStoredCellClassIsOneMoreThanTheBuilderClass(t *testing.T) {
	if got := StoredCellClass(IndoorFloorBuilderClass); got != OpenGroundCellClass {
		t.Fatalf("stored class %02Xh, want %02Xh", got, OpenGroundCellClass)
	}
	if got := StoredCellClass(0); got != 1 {
		t.Fatalf("stored class %d, want 1", got)
	}
}

// 走過整個視窗，落在盤面內的格子必須各自唯一，否則投影寫錯了。
func TestIndoorWindowProjectionDoesNotCollide(t *testing.T) {
	seen := map[[2]int]bool{}
	for _, cell := range IndoorWindowCells() {
		for subA := 2; subA <= 4; subA++ {
			for subB := 0; subB <= 5; subB++ {
				x, y, ok := IndoorTacticalCell(cell[0], cell[1], subA, subB)
				if !ok {
					continue
				}
				key := [2]int{x, y}
				if seen[key] {
					t.Fatalf("dungeon cell %v sub (%d,%d) reused tactical cell %v", cell, subA, subB, key)
				}
				seen[key] = true
			}
		}
	}
	if len(seen) == 0 {
		t.Fatal("the whole window fell outside the board")
	}
}
