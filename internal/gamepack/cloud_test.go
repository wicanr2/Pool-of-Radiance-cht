package gamepack

import "testing"

// board 是測試用的戰術盤：記下每一格的地形，並且可以標某幾格「有別的物件」。
type board struct {
	terrain  map[[2]int]uint8
	occupied map[[2]int]bool
}

func newBoard() *board {
	return &board{terrain: map[[2]int]uint8{}, occupied: map[[2]int]bool{}}
}

func (b *board) SetTerrain(x, y int, terrain uint8) { b.terrain[[2]int{x, y}] = terrain }
func (b *board) Occupied(x, y int) bool             { return b.occupied[[2]int{x, y}] }

// 一團雲是 2×2：雲心、東、東南、南。
//
// 方向索引取自 `DS:28A7h` 的 `[1..4]`（原版的迴圈就是 1..4），位移取自
// `DS:274Ah`／`DS:2753h`。多一格少一格都會讓收雲時還原錯格子。
func TestCloudCoversTheTwoByTwoBlock(t *testing.T) {
	cloud := Cloud{CentreX: 5, CentreY: 7}
	want := [CloudCells][2]int{{5, 7}, {6, 7}, {6, 8}, {5, 8}}
	for index := 0; index < CloudCells; index++ {
		x, y, err := cloud.CellAt(index)
		if err != nil {
			t.Fatal(err)
		}
		if [2]int{x, y} != want[index] {
			t.Fatalf("第 %d 格是 (%d,%d)，預期 %v", index, x, y, want[index])
		}
	}
	if _, _, err := cloud.CellAt(CloudCells); err == nil {
		t.Fatal("超出範圍的格子應該失敗")
	}
}

// 蓋上去寫 1Eh；收起來還原原地形，那一格有別的物件就寫 1Fh。
func TestCloudStampAndRemoveRestoreTerrain(t *testing.T) {
	table := newBoard()
	cloud := Cloud{Caster: 0, Index: 0, CentreX: 2, CentreY: 3,
		Covered:      [CloudCells]bool{true, true, false, true},
		SavedTerrain: [CloudCells]uint8{1, 2, 3, 4}}
	list := CloudList{}.Append(cloud)
	if err := cloud.Stamp(table); err != nil {
		t.Fatal(err)
	}
	// 蓋不住的那一格（索引 2）不該被寫。
	if _, marked := table.terrain[[2]int{3, 4}]; marked {
		t.Fatal("蓋不住的格子被寫了")
	}
	for _, cell := range [][2]int{{2, 3}, {3, 3}, {2, 4}} {
		if table.terrain[cell] != CloudTerrain {
			t.Fatalf("(%d,%d) 是 %d，預期 %d", cell[0], cell[1], table.terrain[cell], CloudTerrain)
		}
	}
	// 其中一格有別的物件。
	table.occupied[[2]int{3, 3}] = true
	list, err := list.RemoveAt(0, table)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("收完之後還剩 %d 團", len(list))
	}
	if got := table.terrain[[2]int{2, 3}]; got != 1 {
		t.Fatalf("(2,3) 還原成 %d，預期原地形 1", got)
	}
	if got := table.terrain[[2]int{3, 3}]; got != ObstacleTerrain {
		t.Fatalf("(3,3) 是 %d，那一格有別的物件，預期 %d", got, ObstacleTerrain)
	}
	if got := table.terrain[[2]int{2, 4}]; got != 4 {
		t.Fatalf("(2,4) 還原成 %d，預期原地形 4", got)
	}
}

// 兩團重疊時收掉一團，另一團的格子要重新蓋回去——不重蓋就會在中間開一個洞。
func TestRemovingOneCloudRepaintsTheOthers(t *testing.T) {
	table := newBoard()
	first := Cloud{Caster: 0, Index: 0, CentreX: 4, CentreY: 4,
		Covered: [CloudCells]bool{true, true, true, true}}
	second := Cloud{Caster: 1, Index: 0, CentreX: 5, CentreY: 4,
		Covered: [CloudCells]bool{true, true, true, true}}
	list := CloudList{}.Append(first).Append(second)
	for _, cloud := range list {
		if err := cloud.Stamp(table); err != nil {
			t.Fatal(err)
		}
	}
	// (5,4) 是兩團共用的那一格。
	list, err := list.RemoveAt(0, table)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Caster != 1 {
		t.Fatalf("剩下的是 %+v", list)
	}
	if got := table.terrain[[2]int{5, 4}]; got != CloudTerrain {
		t.Fatalf("共用的 (5,4) 收完之後是 %d，第二團還在，應該是 %d", got, CloudTerrain)
	}
	// 只屬於第一團的格子要還原。
	if got := table.terrain[[2]int{4, 4}]; got != 0 {
		t.Fatalf("(4,4) 是 %d，預期還原成 0", got)
	}
}

// 數同一個施法者身上有幾團——生新的一團時要用它當編號。
func TestCloudListCountsPerCaster(t *testing.T) {
	list := CloudList{}.
		Append(Cloud{Caster: 3, Index: 0}).
		Append(Cloud{Caster: 5, Index: 0}).
		Append(Cloud{Caster: 3, Index: 1})
	if got := list.CountFor(3); got != 2 {
		t.Fatalf("施法者 3 有 %d 團，預期 2", got)
	}
	if got := list.CountFor(9); got != 0 {
		t.Fatalf("沒放過雲的人有 %d 團", got)
	}
	if got := list.IndexOf(3, 1); got != 2 {
		t.Fatalf("找第 1 團找到位置 %d，預期 2", got)
	}
	if got := list.IndexOf(3, 7); got != -1 {
		t.Fatalf("找不存在的團回 %d，預期 -1", got)
	}
}
