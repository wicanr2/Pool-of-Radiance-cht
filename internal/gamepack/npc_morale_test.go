package gamepack

import (
	"path/filepath"
	"testing"
)

// 八個 ADD NPC 呼叫點的運算元從原版 ECL 重新解一次，與 NPCMoraleSources 對上
// （#74）。正對照：ecl3/11 `9F1Ch` 讀到 `36 01 79 6E`（ADD NPC @6E79 …），位址基準對了
// 才讀得到。
func TestNPCMoraleSourcesMatchTheOriginalECL(t *testing.T) {
	catalog, err := ReadDOSECLCatalog(filepath.Join("..", "..", "Pool of Radiance (1988).zip"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	read := func(archive uint8, block uint16, address uint16, count int) []byte {
		t.Helper()
		selected, ok := catalog.Archive(archive)
		if !ok {
			t.Fatalf("ECL archive %d is absent", archive)
		}
		raw, ok := selected.Blocks[block]
		if !ok {
			t.Fatalf("ECL%d block %d is absent", archive, block)
		}
		// 區塊原始資料前兩個 byte 是長度標頭，payload 從第 2 個 byte 起，映射基準 9900h。
		offset := int(address) - 0x9900 + 2
		if offset < 0 || offset+count > len(raw) {
			t.Fatalf("ECL%d/%d address %04X outside %d bytes", archive, block, address, len(raw))
		}
		return raw[offset : offset+count]
	}
	if got := read(3, 11, 0x9F1C, 4); got[0] != 0x36 || got[2] != 0x79 || got[3] != 0x6E {
		t.Fatalf("control: ecl3/11 9F1Ch = % X, want 36 .. 79 6E", got)
	}
	table := read(3, 11, 0xA170, 8)
	for tier, want := range NPCMercenaryTiers {
		if table[tier] != want {
			t.Fatalf("mercenary tier %d: ECL has %02X, table has %02X", tier, table[tier], want)
		}
	}
	sites := []struct {
		archive uint8
		block   uint16
		address uint16
		source  NPCMoraleSource
	}{
		{2, 15, 0xA9F4, NPCMoraleSource{2, 25, 0}},
		{3, 0, 0xA046, NPCMoraleSource{3, 107, 99}},
		{4, 10, 0xAE81, NPCMoraleSource{4, 24, 99}},
		{4, 2, 0xA5B8, NPCMoraleSource{4, 27, 100}},
		{5, 7, 0xA321, NPCMoraleSource{5, 88, 100}},
		{8, 13, 0xA8D8, NPCMoraleSource{8, 104, 100}},
		{8, 13, 0xB2D1, NPCMoraleSource{8, 104, 100}},
	}
	known := map[NPCMoraleSource]bool{}
	for _, source := range NPCMoraleSources() {
		known[source] = true
	}
	for _, site := range sites {
		// 立即值運算元是 `00 值`：36 00 id 00 morale。
		got := read(site.archive, site.block, site.address, 5)
		if got[0] != 0x36 || got[1] != 0 || got[3] != 0 {
			t.Fatalf("ECL%d/%d %04X = % X, want an ADD NPC with two immediates", site.archive, site.block, site.address, got)
		}
		if got[2] != site.source.Block || got[4] != site.source.Morale {
			t.Fatalf("ECL%d/%d %04X: ADD NPC %d %d, table says %+v", site.archive, site.block, site.address, got[2], got[4], site.source)
		}
		if !known[site.source] {
			t.Fatalf("%+v is missing from NPCMoraleSources", site.source)
		}
	}
	if NPCMoraleByte(99) != 0xB1 || NPCMoraleByte(0) != 0x80 {
		t.Fatalf("NPCMoraleByte: 99 → %02X, 0 → %02X", NPCMoraleByte(99), NPCMoraleByte(0))
	}
}
