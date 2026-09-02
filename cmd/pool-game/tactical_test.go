package main

import (
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/golden-box-remake-engine/geometry"
)

// GEO 的牆面值是牆的樣式編號，與戰術層的 0／1／3 還沒對上，
// 所以任何非零都當成擋路；門因此暫時畫成實牆。
func TestGeoWallProbeTreatsAnyWallTypeAsBlocking(t *testing.T) {
	var grid geometry.Grid
	grid.Cells[5][4].WallDirections = [4]uint8{0, 3, 0, 1}
	probe := geoWallProbe(grid)

	got, err := probe(combat.WallDirectionEast, 4, 5)
	if err != nil {
		t.Fatal(err)
	}
	if got != combat.WallBlocking {
		t.Fatalf("east wall type 3 mapped to %d, want %d", got, combat.WallBlocking)
	}
	if got, _ = probe(combat.WallDirectionWest, 4, 5); got != combat.WallBlocking {
		t.Fatalf("west wall type 1 mapped to %d", got)
	}
	if got, _ = probe(combat.WallDirectionNorth, 4, 5); got != combat.WallOpen {
		t.Fatalf("an absent wall mapped to %d, want %d", got, combat.WallOpen)
	}
}

// 生成器要能吃下這個 probe 並鋪滿整張盤面。
func TestGeoWallProbeDrivesTheGenerator(t *testing.T) {
	var grid geometry.Grid
	built, err := combat.GenerateIndoorTacticalGrid(8, 8, geoWallProbe(grid))
	if err != nil {
		t.Fatal(err)
	}
	for index, code := range built.Terrain {
		if code == combat.UnpaintedCellClass {
			t.Fatalf("cell %d was left unpainted", index)
		}
	}
}
