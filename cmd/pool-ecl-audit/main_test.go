package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPoolCodeAddressBaseAgainstRealCorpus(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	if _, err := os.Stat(zipPath); err != nil {
		t.Skip("original DOS ZIP is intentionally not tracked")
	}
	correct, err := auditAtBase(zipPath, 0x9900)
	if err != nil {
		t.Fatal(err)
	}
	// 29／29 全部走得完（2026-09-04）：`20h NEWECL` 之後不再往下讀，
	// 而 `34h ECL CLOCK` 改吃 Pool 自己量出來的一個運算元（spec 093）。
	if correct.Blocks != 29 || correct.DecodedBlocks != 29 || correct.FailedBlocks != 0 || correct.Instructions != 16034 {
		t.Fatalf("0x9900 report=%+v", correct)
	}

	wrong, err := auditAtBase(zipPath, 0x9914)
	if err != nil {
		t.Fatal(err)
	}
	if wrong.DecodedBlocks != 3 || wrong.FailedBlocks != 26 {
		t.Fatalf("0x9914 negative control decoded=%d failed=%d", wrong.DecodedBlocks, wrong.FailedBlocks)
	}
	found := false
	for _, row := range wrong.BlockResults {
		if strings.EqualFold(filepath.Base(row.File), "ECL3.DAX") && row.BlockID == 0 &&
			strings.Contains(row.Error, "unknown opcode 0xAE at payload offset 215") {
			found = true
		}
	}
	if !found {
		t.Fatal("0x9914 negative control did not reproduce the ECL3/block 0 offset-215 false opcode")
	}
}
