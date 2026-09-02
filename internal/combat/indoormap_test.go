package combat

import (
	"reflect"
	"testing"
)

func classAt(paints []IndoorPaint, subA, subB int) (uint8, bool) {
	var found uint8
	var ok bool
	for _, paint := range paints {
		if paint.SubA == subA && paint.SubB == subB {
			found, ok = paint.Class, true
		}
	}
	return found, ok
}

func TestPaintWestBandFloorsFirst(t *testing.T) {
	paints, err := PaintWestBand(WallOpen)
	if err != nil {
		t.Fatal(err)
	}
	if len(paints) != 18 {
		t.Fatalf("open west painted %d cells, want 18", len(paints))
	}
	for _, paint := range paints {
		if paint.Class != IndoorFloorBuilderClass {
			t.Fatalf("open west painted class %02Xh", paint.Class)
		}
	}
}

// 西牆是一條斜線：每一列的牆格跟著 subA 往右挪一格。
func TestPaintWestBandDrawsADiagonal(t *testing.T) {
	paints, err := PaintWestBand(WallBlocking)
	if err != nil {
		t.Fatal(err)
	}
	for subA := 2; subA <= 4; subA++ {
		if class, ok := classAt(paints, subA, subA); !ok || class != 0x03 {
			t.Fatalf("row %d centre class %02Xh ok=%v, want 03h", subA, class, ok)
		}
		if class, _ := classAt(paints, subA, subA-1); class != 0x04 {
			t.Fatalf("row %d left class %02Xh, want 04h", subA, class)
		}
		if class, _ := classAt(paints, subA, subA+1); class != 0x0D {
			t.Fatalf("row %d right class %02Xh, want 0Dh", subA, class)
		}
	}
}

func TestPaintWestBandAlternateTouchesTwoCells(t *testing.T) {
	paints, err := PaintWestBand(WallAlternate)
	if err != nil {
		t.Fatal(err)
	}
	if class, _ := classAt(paints, 2, 1); class != 0x08 {
		t.Fatalf("class %02Xh at (2,1), want 08h", class)
	}
	if class, _ := classAt(paints, 4, 5); class != 0x00 {
		t.Fatalf("class %02Xh at (4,5), want 00h", class)
	}
}

// 北帶只分「等於 1」與其他，WallAlternate 走地板那一支。
func TestPaintNorthBandOnlySplitsOnBlocking(t *testing.T) {
	blocking, err := PaintNorthBand(WallBlocking)
	if err != nil {
		t.Fatal(err)
	}
	want := []IndoorPaint{
		{SubA: 0, SubB: 3, Class: 0x05},
		{SubA: 0, SubB: 4, Class: 0x05},
		{SubA: 1, SubB: 3, Class: 0x0A},
		{SubA: 1, SubB: 4, Class: 0x0A},
	}
	if !reflect.DeepEqual(blocking, want) {
		t.Fatalf("blocking north %+v", blocking)
	}
	for _, value := range []uint8{WallOpen, WallAlternate} {
		paints, err := PaintNorthBand(value)
		if err != nil {
			t.Fatal(err)
		}
		for _, paint := range paints {
			if paint.Class != IndoorFloorBuilderClass {
				t.Fatalf("north %d painted class %02Xh, want floor", value, paint.Class)
			}
		}
	}
}

func TestPaintNorthWestCornerBranches(t *testing.T) {
	cases := []struct {
		north, west uint8
		open        bool
		want        [4]uint8 // 依 (0,1) (0,2) (1,1) (1,2)
	}{
		{WallOpen, WallOpen, false, [4]uint8{0x16, 0x16, 0x16, 0x16}},
		{WallOpen, WallAlternate, true, [4]uint8{0x0D, 0x16, 0x14, 0x16}},
		{WallOpen, WallBlocking, true, [4]uint8{0x00, 0x16, 0x01, 0x0D}},
		{WallOpen, WallBlocking, false, [4]uint8{0x0D, 0x16, 0x03, 0x0D}},
		{WallAlternate, WallOpen, true, [4]uint8{0x0F, 0x11, 0x10, 0x17}},
		{WallBlocking, WallOpen, false, [4]uint8{0x05, 0x05, 0x0A, 0x0A}},
		{WallBlocking, WallBlocking, true, [4]uint8{0x12, 0x05, 0x01, 0x06}},
	}
	for _, test := range cases {
		paints, err := PaintNorthWestCorner(test.north, test.west, test.open)
		if err != nil {
			t.Fatal(err)
		}
		got := [4]uint8{paints[0].Class, paints[1].Class, paints[2].Class, paints[3].Class}
		if got != test.want {
			t.Fatalf("north=%d west=%d open=%v gave %v, want %v",
				test.north, test.west, test.open, got, test.want)
		}
	}
}

func TestPaintNorthEastCornerBranches(t *testing.T) {
	cases := []struct {
		north, east, aboveEast, rightNorth uint8
		open                               bool
		want                               [4]uint8 // 依 (0,5) (0,6) (1,5) (1,6)
	}{
		{WallOpen, WallOpen, WallOpen, WallOpen, true, [4]uint8{0x16, 0x16, 0x16, 0x16}},
		{WallOpen, WallOpen, WallBlocking, WallBlocking, false, [4]uint8{0x04, 0x0B, 0x16, 0x0C}},
		{WallOpen, WallBlocking, WallBlocking, WallOpen, false, [4]uint8{0x04, 0x03, 0x16, 0x04}},
		{WallOpen, WallOpen, WallAlternate, WallBlocking, false, [4]uint8{0x16, 0x18, 0x16, 0x0C}},
		{WallBlocking, WallOpen, WallOpen, WallOpen, true, [4]uint8{0x05, 0x11, 0x0A, 0x17}},
		{WallBlocking, WallOpen, WallOpen, WallOpen, false, [4]uint8{0x05, 0x13, 0x0A, 0x17}},
		{WallAlternate, WallBlocking, WallOpen, WallOpen, false, [4]uint8{0x0F, 0x09, 0x10, 0x0E}},
	}
	for _, test := range cases {
		paints, err := PaintNorthEastCorner(test.north, test.east, test.aboveEast, test.rightNorth, test.open)
		if err != nil {
			t.Fatal(err)
		}
		got := [4]uint8{paints[0].Class, paints[1].Class, paints[2].Class, paints[3].Class}
		if got != test.want {
			t.Fatalf("north=%d east=%d aboveEast=%d rightNorth=%d open=%v gave %v, want %v",
				test.north, test.east, test.aboveEast, test.rightNorth, test.open, got, test.want)
		}
	}
}

// 原版只產生 0、1、3；其他值會讓區域變數維持未初始化，所以這裡失敗即關閉。
func TestBuildersRejectWallValuesOutsideTheOriginalSet(t *testing.T) {
	if _, err := PaintWestBand(2); err == nil {
		t.Fatal("west band accepted wall value 2")
	}
	if _, err := PaintNorthBand(9); err == nil {
		t.Fatal("north band accepted wall value 9")
	}
	if _, err := PaintNorthWestCorner(0, 2, true); err == nil {
		t.Fatal("north-west corner accepted wall value 2")
	}
	if _, err := PaintNorthEastCorner(0, 0, 0, 4, true); err == nil {
		t.Fatal("north-east corner accepted wall value 4")
	}
}

// 四支建構器落筆的區域不重疊：西帶 subA 2..4，其餘三支在 subA 0..1 的
// 不同 subB 區間。
func TestIndoorBuildersCoverDisjointSubCells(t *testing.T) {
	west, err := PaintWestBand(WallBlocking)
	if err != nil {
		t.Fatal(err)
	}
	north, err := PaintNorthBand(WallBlocking)
	if err != nil {
		t.Fatal(err)
	}
	nw, err := PaintNorthWestCorner(WallBlocking, WallBlocking, true)
	if err != nil {
		t.Fatal(err)
	}
	ne, err := PaintNorthEastCorner(WallBlocking, WallBlocking, WallBlocking, WallBlocking, true)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[[2]int]string{}
	for name, paints := range map[string][]IndoorPaint{
		"north": north, "north-west": nw, "north-east": ne,
	} {
		for _, paint := range paints {
			key := [2]int{paint.SubA, paint.SubB}
			if other, clash := seen[key]; clash {
				t.Fatalf("%s and %s both paint sub-cell %v", name, other, key)
			}
			seen[key] = name
		}
	}
	for _, paint := range west {
		if paint.SubA < 2 {
			t.Fatalf("the west band reached into the north builders' rows: %+v", paint)
		}
	}
}

func openProbe(uint8, int, int) (uint8, error) { return WallOpen, nil }

// 一整片沒有牆的地城，生成出來的戰術格都是地板。
func TestGenerateIndoorTacticalGridOnOpenGround(t *testing.T) {
	grid, err := GenerateIndoorTacticalGrid(10, 10, openProbe)
	if err != nil {
		t.Fatal(err)
	}
	if len(grid.Terrain) != TacticalMapCellCount {
		t.Fatalf("grid has %d cells", len(grid.Terrain))
	}
	painted := 0
	for _, code := range grid.Terrain {
		switch code {
		case UnpaintedCellClass:
		case OpenGroundCellClass:
			painted++
		default:
			t.Fatalf("open ground produced class %02Xh", code)
		}
	}
	if painted == 0 {
		t.Fatal("nothing was painted")
	}
}

// 有牆時牆面類別要真的出現，而且能被格位類別表判成擋路。
func TestGenerateIndoorTacticalGridPaintsWalls(t *testing.T) {
	probe := func(direction uint8, x, y int) (uint8, error) {
		if direction == WallDirectionNorth {
			return WallBlocking, nil
		}
		return WallOpen, nil
	}
	grid, err := GenerateIndoorTacticalGrid(10, 10, probe)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[uint8]int{}
	for _, code := range grid.Terrain {
		seen[code]++
	}
	if seen[StoredCellClass(0x05)] == 0 && seen[StoredCellClass(0x0A)] == 0 {
		t.Fatalf("no wall classes appeared: %v", seen)
	}
}

func TestGenerateIndoorTacticalGridNeedsAProbe(t *testing.T) {
	if _, err := GenerateIndoorTacticalGrid(10, 10, nil); err == nil {
		t.Fatal("a nil probe was accepted")
	}
}

// 牆面查詢是雙向的：只有鄰格那一側有牆時也要算數。
func TestWallBetweenChecksBothSides(t *testing.T) {
	probe := func(direction uint8, x, y int) (uint8, error) {
		if direction == WallDirectionEast && x == 4 {
			return WallBlocking, nil
		}
		return WallOpen, nil
	}
	got, err := WallBetween(probe, WallDirectionWest, 5, 5)
	if err != nil {
		t.Fatal(err)
	}
	if got != WallBlocking {
		t.Fatalf("wall between (5,5) and its west neighbour = %d, want %d", got, WallBlocking)
	}
}
