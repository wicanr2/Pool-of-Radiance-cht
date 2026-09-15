package main

import (
	"path/filepath"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

func TestInspectGEO7PasswordDoorGeometry(t *testing.T) {
	catalog, err := gamepack.ReadDOSGeometryCatalog(filepath.Join("..", "..", "Pool of Radiance (1988).zip"))
	if err != nil {
		t.Skip(err)
	}
	area, ok := catalog.Map(gamepack.MapKey{Archive: 7, BlockID: 23})
	if !ok {
		t.Fatal("GEO7/23 is absent")
	}
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			cell := area.Grid.Cells[y][x]
			terrain := cell.Terrain & 0x7f
			if terrain != 22 && terrain != 25 {
				continue
			}
			t.Logf("terrain %d cell (%d,%d): walls=%v details=%v", terrain, x, y,
				cell.WallDirections, cell.DetailDirections)
			for direction := 0; direction < 8; direction += 2 {
				flags, door := area.Grid.WallDoorFlagsWrapped(x, y, direction)
				t.Logf("  direction %d: passable=%t door=%t flags=%d", direction,
					area.Grid.CanMoveDungeonWrapped(x, y, direction), door, flags)
			}
		}
	}
}
