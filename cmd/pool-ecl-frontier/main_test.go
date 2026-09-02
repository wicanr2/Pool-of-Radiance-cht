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

// passthrough 清單是遊戲行為的一部分：**多宣告一個等於讓那條 opcode 靜靜
// 跳過**，而跳過與正確處理在報表上分不出來。所以這裡把整份清單釘死——
// 新增一條而沒有在 cmd/pool-game 接上前端，這個測試就會紅。
func TestPassthroughStaysExplicit(t *testing.T) {
	want := map[byte]string{
		0x0C: "SETUP MONSTER", 0x0D: "APPROACH", 0x0E: "PICTURE",
		0x0F: "INPUT NUMBER", 0x10: "INPUT STRING", 0x1E: "CHECKPARTY",
		0x21: "LOAD FILES", 0x22: "PARTY SURPRISE", 0x23: "SURPRISE",
		0x24: "service boundary", 0x28: "ROB", 0x29: "ENCOUNTER MENU",
		0x2C: "PARLAY", 0x2D: "CALL", 0x2E: "DAMAGE", 0x31: "SPRITE OFF",
		0x32: "FIND ITEM", 0x33: "PRINT RETURN", 0x34: "ECL CLOCK",
		0x37: "LOAD PIECES",
		0x36: "ADD NPC", 0x38: "PROGRAM", 0x39: "WHO", 0x3A: "DELAY",
		0x3B: "SPELL", 0x3C: "PROTECTION",
		0x3D: "CLEAR BOX",
	}
	got := gamepack.InitialEventPassthrough()
	for code, name := range want {
		if !got[code] {
			t.Fatalf("opcode 0x%02X (%s) is no longer passed through", code, name)
		}
	}
	for code := range got {
		if _, ok := want[code]; !ok {
			t.Fatalf("opcode 0x%02X was added to the passthrough list; wire a front end for it "+
				"and add it here, or it will be silently skipped", code)
		}
	}
}
