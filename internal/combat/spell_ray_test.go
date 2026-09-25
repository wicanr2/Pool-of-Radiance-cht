package combat

import (
	"reflect"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// rayBoard 是一張合成盤面：地形 5 是空地（類別 1），地形 6 是牆（類別 FFh），
// 地形 7 是擋線但不反彈的東西（類別 2）；people 是 (x, y) → 佔格者。
type rayBoard struct {
	walls, stops map[[2]int]bool
	people       map[[2]int]uint8
}

const (
	rayFloor = 5
	rayWall  = 6
	rayStop  = 7
)

func (board rayBoard) classes() CellClasses {
	var classes CellClasses
	classes[rayFloor] = gamepack.CombatCellClass{EntryThreshold: 1}
	classes[rayWall] = gamepack.CombatCellClass{EntryThreshold: SpellRayWallClass}
	classes[rayStop] = gamepack.CombatCellClass{EntryThreshold: 2}
	return classes
}

func (board rayBoard) cellAt(x, y int) (uint8, uint8, error) {
	if int8(x) < 0 || int8(x) > TacticalMaxX || int8(y) < 0 || int8(y) > TacticalMaxY {
		return 0, 0, nil
	}
	cell := [2]int{x, y}
	terrain := uint8(rayFloor)
	if board.walls[cell] {
		terrain = rayWall
	}
	if board.stops[cell] {
		terrain = rayStop
	}
	return board.people[cell], terrain, nil
}

func (board rayBoard) trace(t *testing.T, ray SpellRay) []uint8 {
	t.Helper()
	var hits []uint8
	err := TraceSpellRay(ray, board.classes(), board.cellAt, func(occupant uint8) error {
		hits = append(hits, occupant)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return hits
}

// 射線從瞄準的那一格往外拉：長度 8（剩餘 16 半格），沿線停在每一個新的人身上，
// 每停一次從剩餘整筆扣掉這一段累積的成本（`2B44h`），所以遠處那一個打不到。
func TestSpellRayWalksOutwardAndRunsOutOfLength(t *testing.T) {
	board := rayBoard{people: map[[2]int]uint8{
		{8, 5}: 2, {11, 5}: 3, {20, 5}: 4, {8, 7}: 5,
	}}
	hits := board.trace(t, SpellRay{CasterX: 5, CasterY: 5, TargetX: 8, TargetY: 5, Length: 8, Surcharge: true})
	// (9,5) 2、(10,5) 4、(11,5) 6 停在 3 號：剩 10；(12,5) 8、(13,5) 10 停：剩 0。
	if want := []uint8{3}; !reflect.DeepEqual(hits, want) {
		t.Fatalf("hits %v, want %v", hits, want)
	}
}

// 牆（類別 FFh）把射線擋回來：從牆那一格往施法者那邊再拉一段（`2ACCh`），
// lastOcc 清掉，所以瞄準的那一個會再挨一次。牆後的人打不到。
func TestSpellRayBouncesOffAWall(t *testing.T) {
	board := rayBoard{
		walls:  map[[2]int]bool{{11, 5}: true},
		people: map[[2]int]uint8{{8, 5}: 2, {13, 5}: 3},
	}
	hits := board.trace(t, SpellRay{CasterX: 5, CasterY: 5, TargetX: 8, TargetY: 5, Length: 8, Surcharge: true})
	// 去程 (11,5) 成本 6 撞牆：剩 10，牆離施法者 12 > 8 不加價。回程 (10,5) 2、(9,5) 4、
	// (8,5) 6 停在 2 號：剩 4；(7,5) 8 停，剩 0。
	if want := []uint8{2}; !reflect.DeepEqual(hits, want) {
		t.Fatalf("hits %v, want %v", hits, want)
	}
}

// 牆離施法者不超過 8 半格：第一次反彈那一段多扣 8（`2B1Fh..2B2Bh`），回程短了一截，
// 打不回施法者；不加價（編號 3Ch 的 `[bp+6]` 是 0）時同一條線會回頭打到施法者自己。
func TestSpellRaySurchargeShortensTheRebound(t *testing.T) {
	board := rayBoard{
		walls:  map[[2]int]bool{{9, 5}: true},
		people: map[[2]int]uint8{{5, 5}: 1, {7, 5}: 2},
	}
	// 去程 (8,5) 2、(9,5) 4 撞牆：牆到施法者 8 → 加價，成本 12，剩 16 − 12 = 4。
	// 回程 (8,5) 2、(7,5) 4 停在 2 號：成本 4 不小於 4，剩 0。
	surcharged := board.trace(t, SpellRay{CasterX: 5, CasterY: 5, TargetX: 7, TargetY: 5, Length: 8, Surcharge: true})
	if want := []uint8{2}; !reflect.DeepEqual(surcharged, want) {
		t.Fatalf("surcharged hits %v, want %v", surcharged, want)
	}
	// 不加價：剩 12；回程 (7,5) 4 停在 2 號剩 8，(6,5) 6、(5,5) 8 停在施法者：剩 0。
	plain := board.trace(t, SpellRay{CasterX: 5, CasterY: 5, TargetX: 7, TargetY: 5, Length: 8})
	if want := []uint8{2, 1}; !reflect.DeepEqual(plain, want) {
		t.Fatalf("plain hits %v, want %v", plain, want)
	}
}

// 類別 2 的地形讓走訪器停一下但不反彈（`2A73h` 是 `> 1` 就停，`28B1h` 只有 FFh
// 才算擋住）：停那一次照樣從剩餘扣掉累積成本，然後同一支走訪器繼續往前。
func TestSpellRayPausesOnAnObstacleWithoutBouncing(t *testing.T) {
	board := rayBoard{
		stops:  map[[2]int]bool{{10, 5}: true},
		people: map[[2]int]uint8{{8, 5}: 2, {12, 5}: 3, {14, 5}: 4},
	}
	// (10,5) 4 停：剩 12；(11,5) 6、(12,5) 8 停在 3 號：剩 4；(13,5) 10 停：剩 0。
	hits := board.trace(t, SpellRay{CasterX: 5, CasterY: 5, TargetX: 8, TargetY: 5, Length: 8, Surcharge: true})
	if want := []uint8{3}; !reflect.DeepEqual(hits, want) {
		t.Fatalf("hits %v, want %v", hits, want)
	}
}

// 盤面外（地形 0）直接把剩餘清成 0（`2A88h`）。
func TestSpellRayStopsAtTheEdgeOfTheBoard(t *testing.T) {
	board := rayBoard{people: map[[2]int]uint8{{47, 5}: 2, {49, 5}: 3}}
	hits := board.trace(t, SpellRay{CasterX: 45, CasterY: 5, TargetX: 47, TargetY: 5, Length: 8})
	if want := []uint8{3}; !reflect.DeepEqual(hits, want) {
		t.Fatalf("hits %v, want %v", hits, want)
	}
	// 施法者就站在瞄準那一格：`2968h` 直接返回。
	if hits := board.trace(t, SpellRay{CasterX: 47, CasterY: 5, TargetX: 47, TargetY: 5, Length: 8}); len(hits) != 0 {
		t.Fatalf("a ray aimed at the caster's own cell hit %v", hits)
	}
}
