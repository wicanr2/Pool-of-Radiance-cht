package gamepack

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestRealDOSGeometryCatalog(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	if _, err := os.Stat(zipPath); err != nil {
		t.Skip("original DOS ZIP is intentionally not tracked")
	}
	catalog, err := ReadDOSGeometryCatalog(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	wantKeys := []MapKey{
		{1, 18}, {1, 24}, {1, 31}, {2, 9}, {2, 15}, {2, 20},
		{3, 0}, {3, 14}, {4, 2}, {4, 10}, {4, 21},
		{5, 3}, {5, 4}, {5, 5}, {5, 6}, {5, 7},
		{6, 1}, {6, 25}, {6, 28}, {7, 17}, {7, 22}, {7, 23}, {7, 26},
		{8, 13}, {8, 16}, {8, 27}, {8, 29}, {8, 30}, {8, 32},
	}
	keys := catalog.Keys()
	if catalog.Len() != 29 || len(keys) != len(wantKeys) {
		t.Fatalf("catalog=%d keys=%d, want 29", catalog.Len(), len(keys))
	}
	for index := range wantKeys {
		if keys[index] != wantKeys[index] {
			t.Fatalf("key[%d]=%+v, want %+v", index, keys[index], wantKeys[index])
		}
		value, ok := catalog.Map(keys[index])
		if !ok || value.Key != keys[index] || value.Grid.BlockID != keys[index].BlockID {
			t.Fatalf("map[%+v]=%+v, found=%t", keys[index], value, ok)
		}
	}
	if _, ok := catalog.Map(MapKey{Archive: 1, BlockID: 0}); ok {
		t.Fatal("catalog invented an absent GEO1 block 0")
	}
	anchor, ok := catalog.Map(MapKey{Archive: 1, BlockID: 18})
	if !ok || anchor.Prefix != [2]uint8{0, 4} {
		t.Fatalf("GEO1 block 18 prefix=%v, found=%t; want [0 4]", anchor.Prefix, ok)
	}
	anchor.Grid.Cells[0][0].Terrain ^= 0xFF
	again, _ := catalog.Map(MapKey{Archive: 1, BlockID: 18})
	if again.Grid.Cells[0][0].Terrain == anchor.Grid.Cells[0][0].Terrain {
		t.Fatal("mutating a returned grid changed the catalog")
	}
	spawn := DOSInitialSpawn()
	if spawn.Map != (MapKey{Archive: 3, BlockID: 0}) || spawn.X != 15 || spawn.Y != 1 || spawn.Facing != 3 {
		t.Fatalf("initial spawn=%+v", spawn)
	}
	if _, ok := catalog.Map(spawn.Map); !ok {
		t.Fatalf("initial spawn map %+v is absent from the fixed DOS corpus", spawn.Map)
	}
}

func TestInitialMapRolfExitDungeonEdges(t *testing.T) {
	catalog, err := ReadDOSGeometryCatalog(filepath.Join("..", "..", "Pool of Radiance (1988).zip"))
	if err != nil {
		t.Skip(err)
	}
	grid, ok := catalog.Map(MapKey{Archive: 3, BlockID: 0})
	if !ok {
		t.Fatal("initial map absent")
	}
	want := map[int]bool{0: true, 2: true, 4: false, 6: true}
	for direction, passable := range want {
		if got := grid.Grid.CanMoveDungeonWrapped(0, 4, direction); got != passable {
			t.Errorf("direction %d passable=%v want %v", direction, got, passable)
		}
	}
}

func TestDOSGeometryCatalogFailsClosedWhenArchiveMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.zip")
	writeTestZIP(t, path, []string{"poolrad/GEO1.DAX"})
	if _, err := ReadDOSGeometryCatalog(path); err == nil {
		t.Fatal("missing GEO archives should fail closed")
	}
}

func TestDOSGeometryCatalogRejectsDuplicateArchive(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duplicate.zip")
	writeTestZIP(t, path, []string{"a/GEO1.DAX", "b/geo1.dax"})
	if _, err := ReadDOSGeometryCatalog(path); err == nil {
		t.Fatal("duplicate archive identity should fail closed")
	}
}

func writeTestZIP(t *testing.T, path string, names []string) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	for _, name := range names {
		member, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := member.Write([]byte{0, 0}); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

// 區塊編號在八個 GEO 檔裡全域唯一，所以編號本身就決定了 archive。
// `21h LOAD FILES` 只帶編號，這條性質是拿編號查地圖的前提。
func TestGeometryBlockIDsAreGloballyUnique(t *testing.T) {
	catalog, err := ReadDOSGeometryCatalog(poolZipPath())
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	owner := map[uint8]uint8{}
	for _, key := range catalog.Keys() {
		if previous, clash := owner[key.BlockID]; clash {
			t.Fatalf("block %d 同時在 GEO%d 與 GEO%d", key.BlockID, previous, key.Archive)
		}
		owner[key.BlockID] = key.Archive
	}
	if len(owner) != catalog.Len() {
		t.Fatalf("%d 個編號對 %d 張圖", len(owner), catalog.Len())
	}
	for blockID, archive := range owner {
		found, ok := catalog.MapByBlock(blockID)
		if !ok || found.Key.Archive != archive || found.Key.BlockID != blockID {
			t.Errorf("MapByBlock(%d) 找到 %v，要 GEO%d/%d", blockID, found.Key, archive, blockID)
		}
	}
}
