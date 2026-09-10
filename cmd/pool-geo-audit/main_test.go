package main

import (
	"testing"

	"github.com/wicanr2/golden-box-remake-engine/geometry"
)

// GEO 容器只有 `GEO1.DAX`..`GEO8.DAX` 八個。名字判斷寬一格，盤點就會把別的
// 容器算進來；窄一格則會整包漏掉——兩種錯誤在報告上都只是數字不一樣。
func TestOnlyTheEightGEOArchivesCount(t *testing.T) {
	for _, name := range []string{"GEO1.DAX", "GEO8.DAX", "GEO5.DAX"} {
		if !isGEOArchive(name) {
			t.Errorf("%s 該算 GEO 容器", name)
		}
	}
	for _, name := range []string{
		"GEO0.DAX",  // 沒有第 0 區
		"GEO9.DAX",  // 也沒有第 9 區
		"GEO10.DAX", // 長度不對
		"GEOX.DAX",
		"CHEAD.DAX",
		"GEO1.DAT",
		"GEO1",
	} {
		if isGEOArchive(name) {
			t.Errorf("%s 不該算 GEO 容器", name)
		}
	}
}

// 一塊全 0 的 GEO：沒有牆，所以牆面數是 0，而且**繞回的邊比不繞回的多**
// ——多出來的正好是四邊各一排。這一條釘住「三種移動語意不是同一個數」。
func TestMeasureCountsAnEmptyBlock(t *testing.T) {
	data := make([]byte, geometry.BlockSize)
	row := measure("GEO1.DAX", 3, data)
	if row.Error != "" {
		t.Fatalf("全 0 的區塊解不開：%s", row.Error)
	}
	if row.WallSides != 0 {
		t.Fatalf("沒有牆卻數出 %d 面", row.WallSides)
	}
	cells := geometry.Width * geometry.Height
	if row.TerrainCounts[0] != cells {
		t.Fatalf("地形 0 數出 %d 格，該是 %d", row.TerrainCounts[0], cells)
	}
	edges := cells * 4
	if row.WrappedMoveEdges != edges {
		t.Fatalf("繞回的邊有 %d 條，全空的圖該是 %d", row.WrappedMoveEdges, edges)
	}
	if row.BoundedMoveEdges != edges-4*geometry.Width {
		t.Fatalf("不繞回的邊有 %d 條，該是 %d", row.BoundedMoveEdges, edges-4*geometry.Width)
	}
}

// 立一面牆，牆面數就要跟著多——反對照配上面那條全 0 的。
func TestMeasureSeesAWall(t *testing.T) {
	data := make([]byte, geometry.BlockSize)
	// payload 從 +2 起；第一個平面的高四位是北面。
	data[2] = 0x10
	row := measure("GEO1.DAX", 3, data)
	if row.Error != "" {
		t.Fatalf("解不開：%s", row.Error)
	}
	if row.WallSides != 1 {
		t.Fatalf("立了一面牆卻數出 %d 面", row.WallSides)
	}
	// **牆是雙向的**：從這一格往北走不過去，從北邊那一格往南也走不過去，
	// 所以一面牆扣掉的是兩條邊。牆面數（1）與可走邊數（−2）本來就不同調。
	if row.WrappedMoveEdges != geometry.Width*geometry.Height*4-2 {
		t.Fatalf("繞回的邊有 %d 條，一面牆該扣兩條", row.WrappedMoveEdges)
	}
}

// 大小不對的區塊要留下原因，不能被當成「這一塊沒有牆」——那兩種在報告上
// 都是一排 0。
func TestMeasureRecordsAnUndecodableBlock(t *testing.T) {
	row := measure("GEO1.DAX", 3, make([]byte, 10))
	if row.Error == "" {
		t.Fatal("10 bytes 的區塊沒有留下原因")
	}
	if row.WallSides != 0 || len(row.TerrainCounts) != 0 {
		t.Fatalf("解不開卻填了統計：%+v", row)
	}
}
