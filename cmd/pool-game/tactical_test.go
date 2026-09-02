package main

import (
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/golden-box-remake-engine/geometry"
)

// 沒有牆 → 0；有牆而該方向的 detail 為 0 → 1；有牆且 detail 非 0 → 3。
func TestGeoWallProbeMapsGeoDataToTheThreeOriginalValues(t *testing.T) {
	var grid geometry.Grid
	grid.Cells[5][4].WallDirections = [4]uint8{0, 3, 0, 1}
	grid.Cells[5][4].DetailDirections = [4]uint8{0, 2, 0, 0}
	probe := geoWallProbe(grid, 5)

	got, err := probe(combat.WallDirectionNorth, 4, 5)
	if err != nil {
		t.Fatal(err)
	}
	if got != combat.WallOpen {
		t.Fatalf("an absent wall mapped to %d, want %d", got, combat.WallOpen)
	}
	if got, _ = probe(combat.WallDirectionWest, 4, 5); got != combat.WallBlocking {
		t.Fatalf("a wall with no detail mapped to %d, want %d", got, combat.WallBlocking)
	}
	if got, _ = probe(combat.WallDirectionEast, 4, 5); got != combat.WallAlternate {
		t.Fatalf("a wall with detail mapped to %d, want %d", got, combat.WallAlternate)
	}
}

// 原版不取模：界外一律是牆，只有隊伍那一列的東西向例外。
func TestGeoWallProbeTreatsOutsideTheGridAsWall(t *testing.T) {
	var grid geometry.Grid
	probe := geoWallProbe(grid, 5)

	if got, _ := probe(combat.WallDirectionNorth, -1, 5); got != combat.WallBlocking {
		t.Fatalf("north of the grid mapped to %d", got)
	}
	if got, _ := probe(combat.WallDirectionEast, -1, 5); got != combat.WallOpen {
		t.Fatalf("east on the party row mapped to %d, want open", got)
	}
	if got, _ := probe(combat.WallDirectionEast, -1, 7); got != combat.WallBlocking {
		t.Fatalf("east off the party row mapped to %d, want blocking", got)
	}
}

// 生成器要能吃下這個 probe 並鋪滿整張盤面。
func TestGeoWallProbeDrivesTheGenerator(t *testing.T) {
	var grid geometry.Grid
	built, err := combat.GenerateIndoorTacticalGrid(8, 8, geoWallProbe(grid, 8))
	if err != nil {
		t.Fatal(err)
	}
	for index, code := range built.Terrain {
		if code == combat.UnpaintedCellClass {
			t.Fatalf("cell %d was left unpainted", index)
		}
	}
}
