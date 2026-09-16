package combat

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// 五張表逐位元組對 `START.EXE` DS 2D0h..33Fh 的匯出（spec 061）。
func TestDeploymentTablesMatchTheDataSegment(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "docs", "audit", "ida-start-deployment-tables.json"))
	if err != nil {
		t.Fatal(err)
	}
	var export struct {
		DSOffset int    `json:"ds_offset"`
		Bytes    string `json:"bytes"`
	}
	if err := json.Unmarshal(raw, &export); err != nil {
		t.Fatal(err)
	}
	data, err := hex.DecodeString(export.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	if export.DSOffset != 0x2D0 || len(data) < 0x340-0x2D0 {
		t.Fatalf("export covers %04X+%d, want 02D0..033F", export.DSOffset, len(data))
	}
	at := func(offset int) byte { return data[offset-0x2D0] }
	for q := 0; q < 4; q++ {
		for f := 0; f < DeploymentsPerSet; f++ {
			if got := at(0x2D0 + q*4 + f); got != deploymentNeighbourDirections[q][f] {
				t.Errorf("2D0h[%d][%d] = %d, table %d", q, f, got, deploymentNeighbourDirections[q][f])
			}
			if got := at(0x2E0 + q*4 + f); got != deploymentSweepAxis[q][f] {
				t.Errorf("2E0h[%d][%d] = %d, table %d", q, f, got, deploymentSweepAxis[q][f])
			}
		}
		if got := at(0x2F0 + q); got != deploymentSweepRing[q] {
			t.Errorf("2F0h[%d] = %d, table %d", q, got, deploymentSweepRing[q])
		}
		for origin := 0; origin < 2; origin++ {
			if got := int8(at(0x2F4 + origin*4 + q)); got != deploymentOriginCol[origin][q] {
				t.Errorf("2F4h[%d][%d] = %d, table %d", origin, q, got, deploymentOriginCol[origin][q])
			}
			if got := int8(at(0x2FC + origin*4 + q)); got != deploymentOriginRow[origin][q] {
				t.Errorf("2FCh[%d][%d] = %d, table %d", origin, q, got, deploymentOriginRow[origin][q])
			}
		}
	}
	for set := 0; set < 5; set++ {
		for row := 0; row < DeploymentTemplateRows; row++ {
			lo, hi := int8(at(0x304+set*12+row*2)), int8(at(0x305+set*12+row*2))
			if lo != deploymentRowSpans[set][row][0] || hi != deploymentRowSpans[set][row][1] {
				t.Errorf("304h[%d][%d] = %d..%d, table %d..%d", set, row, lo, hi,
					deploymentRowSpans[set][row][0], deploymentRowSpans[set][row][1])
			}
		}
	}
}

// openBoard 是一張整面可站的盤，Probe 只看佔用。
func openBoard(t *testing.T, cells *[]CombatantCell, index uint8) DeploymentBoard {
	t.Helper()
	return DeploymentBoard{
		Probe: func(x, y int) (uint8, uint8, error) {
			if x < 0 || y < 0 || x > TacticalMaxX || y > TacticalMaxY {
				return 0, OffBoardDestinationClass, nil
			}
			for i, cell := range *cells {
				if i == 0 || uint8(i) == index || cell.FootprintClass == 0 {
					continue
				}
				if int(cell.X) == x && int(cell.Y) == y {
					return uint8(i), OpenGroundCellClass, nil
				}
			}
			return 0, OpenGroundCellClass, nil
		},
	}
}

// dosgolem 的收據（docs/audit/dosgolem-deployment-peek.json）：單人隊面向西
// （`6A0Dh` = 6）、遭遇距離 0、雙方各一人，原版把隊員放在 (28,13)、敵人放在
// (26,12)。同狀態下這裡要算出同一組座標。
func TestPlaceCombatantMatchesTheDosgolemReceipt(t *testing.T) {
	sides, err := DeploymentSides(6, 0, [2]int{1, 1})
	if err != nil {
		t.Fatal(err)
	}
	want := [2]DeploymentSide{{0, 0, 1, 3}, {0, 0, 1, 1}}
	if sides != want {
		t.Fatalf("sides = %+v, want %+v (45B2h..45B9h = 00 00 00 00 01 01 03 01)", sides, want)
	}
	templates, err := FillDeploymentTemplates(sides)
	if err != nil {
		t.Fatal(err)
	}
	classes := testCellClasses()
	cells := []CombatantCell{{}}
	// 室內、原地城格四面都有牆：奇數象限那一步的三個方向都擋住。
	walls := func(direction uint8, x, y int) (uint8, error) { return WallBlocking, nil }
	for index, side := range []uint8{0, 1} {
		board := openBoard(t, &cells, uint8(index+1))
		board.Wall = walls
		placement, err := PlaceCombatant(sides, &templates, side, board, classes)
		if err != nil {
			t.Fatal(err)
		}
		if !placement.Placed {
			t.Fatalf("combatant %d on side %d was not placed", index+1, side)
		}
		cells = append(cells, CombatantCell{X: uint8(placement.X), Y: uint8(placement.Y), FootprintClass: 1})
	}
	if cells[1].X != 28 || cells[1].Y != 13 {
		t.Errorf("party member at (%d,%d), original 5E85h says (28,13)", cells[1].X, cells[1].Y)
	}
	if cells[2].X != 26 || cells[2].Y != 12 {
		t.Errorf("foe at (%d,%d), original 5E85h says (26,12)", cells[2].X, cells[2].Y)
	}
}

// 六人隊面向北、敵方八隻在距離 2：第一腿上限 3，所以第一列放三個
// （原點、右一、左一），其餘在第二列從中間往外排；敵方在 dy = −2 那一格。
// 這是從位元組推出來的形狀，dosgolem 六人隊的收據還沒拍（spec 061）。
func TestPlaceCombatantSweepsFromTheMiddleOutward(t *testing.T) {
	sides, err := DeploymentSides(0, 2, [2]int{6, 8})
	if err != nil {
		t.Fatal(err)
	}
	if sides[1].DY != -2 || sides[1].DX != 0 || sides[1].Quadrant != 2 {
		t.Fatalf("foe side = %+v", sides[1])
	}
	templates, err := FillDeploymentTemplates(sides)
	if err != nil {
		t.Fatal(err)
	}
	classes := testCellClasses()
	cells := []CombatantCell{{}}
	var got [][2]int
	for i := 0; i < 6; i++ {
		placement, err := PlaceCombatant(sides, &templates, 0, openBoard(t, &cells, uint8(len(cells))), classes)
		if err != nil {
			t.Fatal(err)
		}
		if !placement.Placed {
			t.Fatalf("member %d not placed", i+1)
		}
		cells = append(cells, CombatantCell{X: uint8(placement.X), Y: uint8(placement.Y), FootprintClass: 1})
		got = append(got, [2]int{placement.X, placement.Y})
	}
	want := [][2]int{{27, 13}, {28, 13}, {26, 13}, {28, 14}, {29, 14}, {27, 14}}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("member %d at %v, want %v (all: %v)", i+1, got[i], want[i], got)
		}
	}
	var foes [][2]int
	for i := 0; i < 8; i++ {
		placement, err := PlaceCombatant(sides, &templates, 1, openBoard(t, &cells, uint8(len(cells))), classes)
		if err != nil {
			t.Fatal(err)
		}
		if !placement.Placed {
			t.Fatalf("foe %d not placed", i+1)
		}
		cells = append(cells, CombatantCell{X: uint8(placement.X), Y: uint8(placement.Y), FootprintClass: 1})
		foes = append(foes, [2]int{placement.X, placement.Y})
	}
	// 敵方象限 2：原點 (5,2)、沿 W／E 掃、換腿往 NW；投影 X = 22 + 5×(−2) + col、
	// Y = 10 − 10 + row。
	wantFoes := [][2]int{{17, 2}, {16, 2}, {18, 2}, {15, 2}, {16, 1}, {15, 1}, {17, 1}, {14, 1}}
	for i := range wantFoes {
		if foes[i] != wantFoes[i] {
			t.Errorf("foe %d at %v, want %v (all: %v)", i+1, foes[i], wantFoes[i], foes)
		}
	}
}

// 樣板整塊被佔滿之後換陣型：有牆的鄰格跳過，沒牆的那一格接著放。
func TestPlaceCombatantRelocatesWhenTheBlockIsFull(t *testing.T) {
	sides, err := DeploymentSides(0, 0, [2]int{40, 0})
	if err != nil {
		t.Fatal(err)
	}
	templates, err := FillDeploymentTemplates(sides)
	if err != nil {
		t.Fatal(err)
	}
	classes := testCellClasses()
	cells := []CombatantCell{{}}
	// 面向北：陣型 1..3 的方向是 S、W、E；只讓 W 沒牆。
	walls := func(direction uint8, x, y int) (uint8, error) {
		if direction == 6 || direction == 2 {
			return WallOpen, nil
		}
		return WallBlocking, nil
	}
	relocated := 0
	for i := 0; i < 30; i++ {
		board := openBoard(t, &cells, uint8(len(cells)))
		board.Wall = walls
		placement, err := PlaceCombatant(sides, &templates, 0, board, classes)
		if err != nil {
			t.Fatal(err)
		}
		if !placement.Placed {
			t.Fatalf("member %d not placed", i+1)
		}
		if placement.Formation != 0 {
			relocated++
			if placement.Formation != 2 {
				t.Errorf("member %d went to formation %d, want 2 (W is the open neighbour)", i+1, placement.Formation)
			}
		}
		cells = append(cells, CombatantCell{X: uint8(placement.X), Y: uint8(placement.Y), FootprintClass: 1})
	}
	// 象限 0 的樣板只有 8 + 8 + 7 = 23 格。
	if relocated != 30-23 {
		t.Errorf("%d members relocated, want %d", relocated, 30-23)
	}
}
