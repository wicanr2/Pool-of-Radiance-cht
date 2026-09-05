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

// 站在雲裡的判定是「佔格裡只要有一格是雲」——原版 `013Dh:007Fh` 的
// 三條短路規則裡，`1Eh` 排在 0 之後、權重比較之前（spec 121）。
func TestStandingInCloudTakesAnySingleCell(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		terrain []uint8
		want    bool
	}{
		{"四格都不是雲", []uint8{1, 2, 3, 4}, false},
		{"只有一格是雲也算", []uint8{1, CloudTerrain, 3, 4}, true},
		{"四格都是雲", []uint8{CloudTerrain, CloudTerrain, CloudTerrain, CloudTerrain}, true},
		{"沒有佔格就不算", nil, false},
		// 0 在原版是「直接收工」的那一條，但它排在雲**前面**，所以
		// 「有 0 也有雲」的時候原版回 0。這裡只回答「是不是雲」，用不到
		// 那一段；把這個案例釘住是為了讓將來補權重時記得順序有意義。
		{"有 0 也有雲", []uint8{0, CloudTerrain}, true},
	} {
		if got := StandingInCloud(testCase.terrain); got != testCase.want {
			t.Errorf("%s：得到 %v，預期 %v", testCase.name, got, testCase.want)
		}
	}
}

// AC 每結算一次變差 2 點，最差停在內部值 32h（顯示 AC 10）。
// 內部值越大 AC 越好（顯示 AC ＝ 60 − 內部值，spec 049），所以「變差」是減。
func TestStinkingCloudArmourClassWorsensToTheFloor(t *testing.T) {
	for _, testCase := range []struct{ from, want int }{
		{0x3C, 0x3A}, // 顯示 AC 0 → 2
		{0x36, 0x34}, // 顯示 AC 6 → 8
		{0x35, 0x33}, // 35h > 34h，照樣減 2
		{0x34, 0x32}, // 不大於 34h 就一路壓到底
		{0x33, 0x32},
		{0x32, 0x32}, // 已經最差就不動
		{0x20, 0x32}, // 比下限還差的（原版不會出現）也被拉回 32h
	} {
		if got := StinkingCloudArmourClass(testCase.from); got != testCase.want {
			t.Errorf("內部 AC %#x 結算後是 %#x，預期 %#x", testCase.from, got, testCase.want)
		}
	}
}
