package main

// 遭遇距離的最小重現（spec 078 的 `0489h`）：貧民窟 (14,4) 面向西，dosgolem 讀到
// `45B3h`／`45B5h` 都是 0，也就是 `+582h` 是 0——面前就是牆。同一格面向東是開
// 的，走兩步就是 2。`@49E6` 為 0 時不走、固定 2。

import (
	"path/filepath"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

func TestEncounterDistanceWalksUntilTheWall(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	archive, ok := application.eclCatalog.Archive(2)
	if !ok {
		t.Fatal("ECL2 archive is absent")
	}
	session, err := gamepack.NewDOSECLArchiveSession(archive, 20, 0x9914)
	if err != nil {
		t.Fatal(err)
	}
	geoMap, ok := application.geometryCatalog.Map(gamepack.MapKey{Archive: 2, BlockID: 20})
	if !ok {
		t.Fatal("GEO2/20 is absent")
	}
	application.eventSession, application.eventMachine = session, session.Machine()
	application.initialMap = &geoMap
	application.eventMachine.Memory[encounterWalkFlagAddress] = 1

	// dosgolem 的那一格：面向西撞牆。
	application.spawn = gamepack.Spawn{Map: geoMap.Key, X: 14, Y: 4, Facing: 3}
	if got := application.encounterStartDistance(2); got != 0 {
		t.Fatalf("(14,4) facing W: distance %d, dosgolem read 0", got)
	}
	// 同一格找一個開闊的朝向，走滿兩步；牆值直接用 GEO 對照。
	for facing := uint8(0); facing < 4; facing++ {
		application.spawn.Facing = facing
		want := 0
		x, y := 14, 4
		for want < 2 {
			wall, _ := geoMap.Grid.Wall(x, y, int(facing)*2)
			if wall != 0 {
				break
			}
			want++
			switch facing {
			case 0:
				y--
			case 1:
				x++
			case 2:
				y++
			case 3:
				x--
			}
		}
		if got := application.encounterStartDistance(2); got != want {
			t.Errorf("(14,4) facing %d: distance %d, GEO walk says %d", facing, got, want)
		}
		if got := application.encounterStartDistance(1); got > 1 {
			t.Errorf("facing %d: limit 1 not applied, got %d", facing, got)
		}
	}
	// `@49E6` 為 0：不走，固定 2。
	application.eventMachine.Memory[encounterWalkFlagAddress] = 0
	application.spawn.Facing = 3
	if got := application.encounterStartDistance(2); got != 2 {
		t.Fatalf("walk flag 0: distance %d, want 2", got)
	}
}
