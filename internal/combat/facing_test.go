package combat

import "testing"

// DS:274Ah／DS:2753h 的九項，逐項對回 docs/audit/ida-ds-direction-step-table.json。
func TestDirectionStepsMatchOriginalTable(t *testing.T) {
	want := []FootprintOffset{
		{X: 0, Y: -1}, {X: 1, Y: -1}, {X: 1, Y: 0}, {X: 1, Y: 1},
		{X: 0, Y: 1}, {X: -1, Y: 1}, {X: -1, Y: 0}, {X: -1, Y: -1},
		{X: 0, Y: 0},
	}
	for direction, expected := range want {
		got, err := DirectionStep(uint8(direction))
		if err != nil {
			t.Fatalf("direction %d: %v", direction, err)
		}
		if got != expected {
			t.Fatalf("direction %d step %+v, want %+v", direction, got, expected)
		}
	}
	if _, err := DirectionStep(DirectionAny + 1); err == nil {
		t.Fatal("direction outside the original table accepted")
	}
}

// 原版進入時以有號位元組檢查兩組座標，超界一律回 false 而不是錯誤。
func TestFacingArcRejectsCoordinatesOutsideTheBoard(t *testing.T) {
	cases := [][4]uint8{
		{TacticalMaxX + 1, 10, 10, 10},
		{10, TacticalMaxY + 1, 10, 10},
		{10, 10, TacticalMaxX + 1, 10},
		{10, 10, 10, TacticalMaxY + 1},
		{0x80, 10, 10, 10},
	}
	for _, c := range cases {
		inside, err := FacingArcContains(c[0], c[1], c[2], c[3], DirectionAny)
		if err != nil {
			t.Fatalf("%v: %v", c, err)
		}
		if inside {
			t.Fatalf("%v accepted although a coordinate is outside the board", c)
		}
	}
}

// 起點本身與弧的頂點在原版是無條件成立，與 direction 無關。
func TestFacingArcAlwaysContainsOriginAndApex(t *testing.T) {
	for direction := uint8(0); direction <= DirectionAny; direction++ {
		inside, err := FacingArcContains(25, 12, 25, 12, direction)
		if err != nil {
			t.Fatal(err)
		}
		if !inside {
			t.Fatalf("direction %d excluded its own cell", direction)
		}
		step, err := DirectionStep(direction)
		if err != nil {
			t.Fatal(err)
		}
		apexX := uint8(25 + int(step.X))
		apexY := uint8(12 + int(step.Y))
		inside, err = FacingArcContains(25, 12, apexX, apexY, direction)
		if err != nil {
			t.Fatal(err)
		}
		if !inside {
			t.Fatalf("direction %d excluded its apex (%d,%d)", direction, apexX, apexY)
		}
	}
}

// 四個正向是以頂點為頂的 90 度錐形。
func TestFacingArcAxisDirectionsAreCones(t *testing.T) {
	cases := []struct {
		direction uint8
		toX, toY  uint8
		want      bool
		reason    string
	}{
		{0, 25, 5, true, "正前方"},
		{0, 27, 9, true, "錐形邊界上"},
		{0, 28, 9, false, "超出錐形邊界"},
		{0, 25, 20, false, "正後方"},
		{2, 40, 12, true, "正右方"},
		{2, 30, 16, true, "錐形邊界上"},
		{2, 30, 17, false, "超出錐形邊界"},
		{4, 25, 20, true, "正下方"},
		{4, 25, 5, false, "反向"},
		{6, 10, 12, true, "正左方"},
		{6, 40, 12, false, "反向"},
	}
	for _, c := range cases {
		inside, err := FacingArcContains(25, 12, c.toX, c.toY, c.direction)
		if err != nil {
			t.Fatal(err)
		}
		if inside != c.want {
			t.Fatalf("direction %d to (%d,%d) = %v, want %v (%s)",
				c.direction, c.toX, c.toY, inside, c.want, c.reason)
		}
	}
}

// 四個對角是以頂點為原點的象限：兩個分量同號即成立。
func TestFacingArcDiagonalDirectionsAreQuadrants(t *testing.T) {
	cases := []struct {
		direction uint8
		toX, toY  uint8
		want      bool
	}{
		{1, 40, 2, true},
		{1, 40, 12, false},
		{1, 10, 2, false},
		{3, 40, 20, true},
		{3, 10, 20, false},
		{5, 10, 20, true},
		{5, 40, 20, false},
		{7, 10, 2, true},
		{7, 40, 2, false},
	}
	for _, c := range cases {
		inside, err := FacingArcContains(25, 12, c.toX, c.toY, c.direction)
		if err != nil {
			t.Fatal(err)
		}
		if inside != c.want {
			t.Fatalf("direction %d to (%d,%d) = %v, want %v", c.direction, c.toX, c.toY, inside, c.want)
		}
	}
}

// DirectionAny 與原版的 0FFh 一樣恆真，因此八個方向的聯集覆蓋整個盤面。
func TestFacingArcDirectionsCoverTheWholeBoard(t *testing.T) {
	const fromX, fromY = 25, 12
	for toX := uint8(0); toX <= TacticalMaxX; toX++ {
		for toY := uint8(0); toY <= TacticalMaxY; toY++ {
			covered := false
			for direction := uint8(0); direction < DirectionCount; direction++ {
				inside, err := FacingArcContains(fromX, fromY, toX, toY, direction)
				if err != nil {
					t.Fatal(err)
				}
				if inside {
					covered = true
					break
				}
			}
			if !covered {
				t.Fatalf("(%d,%d) is in none of the eight arcs", toX, toY)
			}
			inside, err := FacingArcContains(fromX, fromY, toX, toY, DirectionUnset)
			if err != nil {
				t.Fatal(err)
			}
			if !inside {
				t.Fatalf("(%d,%d) rejected by the unset direction", toX, toY)
			}
		}
	}
}

// 指定的方向小於 8 時原版直接沿用，否則自 0 起取第一個成立的方向。
func TestRequiredFacingUsesExplicitDirectionOrSearchesFromZero(t *testing.T) {
	got, err := RequiredFacing(25, 12, 40, 20, 3)
	if err != nil {
		t.Fatal(err)
	}
	if got != 3 {
		t.Fatalf("explicit direction became %d", got)
	}
	got, err = RequiredFacing(25, 12, 25, 5, DirectionUnset)
	if err != nil {
		t.Fatal(err)
	}
	if got != 0 {
		t.Fatalf("search returned %d, want 0", got)
	}
	// 正東方的目標不落在方向 1 的象限內：方向 1 的頂點是 (26,11)，目標在頂點下方。
	got, err = RequiredFacing(25, 12, 40, 12, DirectionUnset)
	if err != nil {
		t.Fatal(err)
	}
	if got != 2 {
		t.Fatalf("search returned %d, want 2", got)
	}
	got, err = RequiredFacing(25, 12, 40, 2, DirectionUnset)
	if err != nil {
		t.Fatal(err)
	}
	if got != 1 {
		t.Fatalf("search returned %d, want 1", got)
	}
}

// 座標超界時每個方向都不成立，原版會落進未定義的回傳值，這裡失敗即關閉。
func TestRequiredFacingFailsClosedOutsideTheBoard(t *testing.T) {
	if _, err := RequiredFacing(25, 12, TacticalMaxX+1, 12, DirectionUnset); err == nil {
		t.Fatal("a target outside the board produced a facing")
	}
}

// AdvanceTacticalCoordinate 自帶一份方向位移表，與 DS:274Ah／DS:2753h 解出的
// directionSteps 是同一份原始資料。兩處必須逐項相同，否則其中一份已經飄掉。
func TestAdvanceTacticalCoordinateAgreesWithTheDirectionTable(t *testing.T) {
	const baseX, baseY = 20, 12
	for direction := uint8(0); direction < DirectionCount; direction++ {
		step, err := DirectionStep(direction)
		if err != nil {
			t.Fatal(err)
		}
		gotX, gotY, err := AdvanceTacticalCoordinate(baseX, baseY, direction)
		if err != nil {
			t.Fatal(err)
		}
		wantX := uint8(baseX + int(step.X))
		wantY := uint8(baseY + int(step.Y))
		if gotX != wantX || gotY != wantY {
			t.Fatalf("direction %d moved to (%d,%d), the direction table says (%d,%d)",
				direction, gotX, gotY, wantX, wantY)
		}
	}
}

// 盤面外的座標要當場失敗，不是回報「八個方向都不通」。
//
// `FacingArcContains` 的界限檢查對 `DirectionAny` 一樣生效，所以出界時九個
// 候選一個都不成立。先前那條註解寫「DirectionAny 恆真，搜尋一定會停」，
// 在出界這一組輸入下是假的。
func TestRequiredFacingRejectsCellsOffTheBoard(t *testing.T) {
	if _, err := RequiredFacing(35, TacticalMaxY+1, 19, 11, DirectionAny); err == nil {
		t.Fatal("盤面外的起點沒有失敗")
	}
	if _, err := RequiredFacing(19, 11, TacticalMaxX+1, 11, DirectionAny); err == nil {
		t.Fatal("盤面外的目標沒有失敗")
	}
	// 正對照：同一組輸入搬回盤面內就要成功。
	if _, err := RequiredFacing(35, TacticalMaxY, 19, 11, DirectionAny); err != nil {
		t.Fatalf("盤面內的座標反而失敗：%v", err)
	}
}
