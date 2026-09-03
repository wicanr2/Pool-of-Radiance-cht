package main

import (
	"path/filepath"
	"testing"
)

// 整包掃描：26 個有地圖的 ECL 區塊、26624 次入口執行，一個錯誤都沒有。
// 這一條擋的是「VM 在別的區域炸掉」。
func TestWorldCellSweepRunsEveryCellWithoutError(t *testing.T) {
	result, err := sweep(filepath.Join("..", "..", "Pool of Radiance (1988).zip"))
	if err != nil {
		t.Skip("original DOS ZIP is intentionally not tracked")
	}
	if len(result.Blocks) != 26 {
		t.Errorf("掃到 %d 個區塊，預期 26", len(result.Blocks))
	}
	if result.TotalCells != 26624 {
		t.Errorf("掃到 %d 格，預期 26624", result.TotalCells)
	}
	for _, block := range result.Blocks {
		for message, count := range block.Errors {
			t.Errorf("ecl%d/%d 有錯誤 ×%d：%s",
				block.Archive, block.BlockID, count, message)
		}
	}
	if result.TotalErrors != 0 {
		t.Errorf("錯誤 %d 次，預期 0", result.TotalErrors)
	}
}
