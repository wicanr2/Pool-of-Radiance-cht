package main

import (
	"path/filepath"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// 盤點跑得起來，而且「已處理」那一半真的來自兩個來源，不是抄一份會過期的常數。
func TestFrontierCountsEveryUnhandledSite(t *testing.T) {
	result, err := scan(filepath.Join("..", "..", "Pool of Radiance (1988).zip"))
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	if result.BlockTotal == 0 {
		t.Fatal("no ECL blocks were scanned")
	}
	total := 0
	for _, row := range result.Unhandled {
		if len(row.Sites) == 0 {
			t.Fatalf("%s is listed with no call site", row.Opcode)
		}
		total += len(row.Sites)
	}
	if total != result.SiteTotal {
		t.Fatalf("site total %d, rows add up to %d", result.SiteTotal, total)
	}
	// 已經接上的不准出現在待辦裡。29h 與 38h 是前端接的，走 passthrough。
	for _, row := range result.Unhandled {
		switch row.Opcode {
		case "0x29", "0x38", "0x37", "0x21", "0x2D":
			t.Fatalf("%s is already wired but still listed as unhandled", row.Opcode)
		}
	}
}

// passthrough 清單是遊戲行為的一部分：多宣告一個等於讓那條 opcode 靜靜跳過。
func TestPassthroughStaysExplicit(t *testing.T) {
	passthrough := gamepack.InitialEventPassthrough()
	for _, code := range []byte{0x29, 0x38} {
		if !passthrough[code] {
			t.Fatalf("opcode 0x%02X must be passed through so the front end sees it", code)
		}
	}
	// 39h WHO 還沒接：宣告成 passthrough 而前端不處理，等於靜靜跳過。
	if passthrough[0x39] {
		t.Fatal("opcode 0x39 WHO is passed through but has no front end")
	}
}
