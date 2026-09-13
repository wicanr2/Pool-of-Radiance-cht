package main

import (
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// spec 089：PROTECTION 從指定位置起讀，輸出值加一，並在第一個零停止。
func TestProtectionRowAddsOneAndStopsAtZero(t *testing.T) {
	memory := map[uint16]uint16{0x5000: 1, 0x5001: 4, 0x5002: 0, 0x5003: 9}
	got := formatProtectionRow(func(address uint16) uint16 { return memory[address] }, 0x5000)
	if got != "2 5" {
		t.Fatalf("PROTECTION row = %q，應為 %q", got, "2 5")
	}
}

// 損壞資料沒有零結尾時仍須受 remake 的 64 格安全上限約束。
func TestProtectionRowStopsAtTheSafetyLimit(t *testing.T) {
	reads := 0
	got := formatProtectionRow(func(uint16) uint16 {
		reads++
		return 1
	}, 0x5000)
	if reads != gamepack.ProtectionMaxEntries {
		t.Fatalf("讀了 %d 格，應為 %d", reads, gamepack.ProtectionMaxEntries)
	}
	if got == "" {
		t.Fatal("非零列不應是空字串")
	}
}
