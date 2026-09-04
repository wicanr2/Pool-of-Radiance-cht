package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// GEO7/23 (1,1) 是密碼門。世界巡迴掃 102 趟只有 `seed 106、destination 2`
// 這一趟走得到它，走到就卡：告示牌跑完問 [YES NO]，答 NO 之後告示牌從頭再跑
// 一次，`cellWaitingMenu` 整段都沒有變回 false。
//
// 這一則把那一趟單獨拉出來當可重跑的 pass/fail loop，不必等 22 趟的
// TestWorldTourReachesTheAreasBehindTheHarbour 跑完才知道有沒有踩到。
func TestCellMenuDoesNotStallAtTheGEO7PasswordDoor(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	visited := map[[3]int]bool{}
	var hardFailures []string
	_, reachable := exploreWorldWithFlags(t, zipPath, 106, 0, 1, 200000,
		map[[3]int]bool{}, map[[3]int]bool{}, map[[3]int]int{}, map[[3]int]int{},
		map[[4]int]int{}, visited, map[string]bool{}, map[int]bool{}, nil, 2,
		&hardFailures)
	if !reachable {
		t.Skip("original DOS ZIP is intentionally not tracked")
	}
	for _, failure := range hardFailures {
		if strings.Contains(failure, "選單") || strings.Contains(failure, "menu") {
			t.Errorf("走到密碼門就卡住：%s", failure)
		}
	}
	if len(hardFailures) > 0 {
		t.Logf("這一趟的硬失敗共 %d 筆：", len(hardFailures))
		for _, failure := range hardFailures {
			t.Logf("  %s", failure)
		}
	}
}
