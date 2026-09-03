package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// 整包掃描：26 個有地圖的 ECL 區塊、26624 次入口執行。這一條擋的是
// 「VM 在別的區域炸掉」——目前唯一的錯誤是 LOAD CHARACTER 缺投影器。
func TestWorldCellSweepHasOnlyTheKnownGap(t *testing.T) {
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
			if !strings.Contains(message, "LOAD CHARACTER") {
				t.Errorf("ecl%d/%d 有沒見過的錯誤 ×%d：%s",
					block.Archive, block.BlockID, count, message)
			}
		}
	}
	// 量到的數字，不是目標。變多代表退步，變少就把它調下來。
	if result.TotalErrors > 67 {
		t.Errorf("錯誤 %d 次，之前量到 67 次", result.TotalErrors)
	}
}
