package main

// 本檔用真實 corpus 釘住兩件事：一共有幾處要玩家打字，以及每一處解出來的答案。
// 密語提示（spec 141）顯示的就是這裡的 `answer`，錯的答案在遊戲裡長得跟對的一樣。

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestPasswordSitesAgainstRealCorpus(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	if _, err := os.Stat(zipPath); err != nil {
		t.Skip("original DOS ZIP is intentionally not tracked")
	}
	result, err := audit(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	if result.Sites != 20 || result.WithAnswer != 19 {
		t.Fatalf("盤點到 %d 處輸入、%d 處有答案（2026-09-18 量到 20／19）", result.Sites, result.WithAnswer)
	}
	// 主鍵是「檔名 + 位址」：`9E80h` 在 ecl4 與 ecl5 各有一處，只用位址會撞號。
	want := map[string]string{
		"ecl2.dax@9F1C": "HARASH",          // 石像鬼的「今天的口令」
		"ecl2.dax@9F6B": "TYRANTHRAXUS",    // 「誰派你來的」
		"ecl2.dax@AF9F": "OHLO",            //
		"ecl4.dax@9E80": "SAMOSUD／SHESTNI", // 依 4A26h 二選一，兩個都列
		"ecl4.dax@A382": "LUX",             // 索寇要塞的亡魂
		"ecl5.dax@AA56": "RHODIA",          // 城門口令
		"ecl7.dax@A4A1": "NOKNOK",          // 矮人符文
		"ecl8.dax@A0C9": "SAVIOR",          // 部族的友誼暗語
		"ecl5.dax@9E08": "",                // 巨人：打什麼都錯，本來就沒有答案
	}
	found := map[string]string{}
	for _, entry := range result.Entries {
		key := fmt.Sprintf("%s@%04X", entry.File, entry.Address)
		if _, ok := want[key]; ok {
			found[key] = entry.Answer
		}
	}
	for key, answer := range want {
		got, ok := found[key]
		if !ok {
			t.Errorf("%s 這一處不在盤點裡", key)
			continue
		}
		if got != answer {
			t.Errorf("%s 解出 %q，原版比的是 %q", key, got, answer)
		}
	}
}
