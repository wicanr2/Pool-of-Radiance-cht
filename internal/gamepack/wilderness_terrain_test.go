package gamepack

import (
	"path/filepath"
	"testing"
)

// `DS:495Bh` 的寫入（overlay-03 `3653h..36DBh`）：只有區塊 25／26／27 而且
// `@49E6` 為 0 才是野外模式。
func TestCombatAreaMode(t *testing.T) {
	cases := []struct {
		block, walk uint16
		want        uint8
	}{
		{0x19, 0, CombatAreaWildernessWest},
		{0x1A, 0, CombatAreaWildernessMid},
		{0x1B, 0, CombatAreaWildernessEast},
		{0x1B, 1, CombatAreaIndoor},
		{0x00, 0, CombatAreaIndoor},
		{0x15, 0, CombatAreaIndoor},
	}
	for _, c := range cases {
		if got := CombatAreaMode(c.block, c.walk); got != c.want {
			t.Errorf("block %d @49E6=%d → mode %d, want %d", c.block, c.walk, got, c.want)
		}
	}
}

func TestParseWildernessTerrainTableRejectsWrongSize(t *testing.T) {
	for _, size := range []int{0, WildernessTerrainSize - 1, WildernessTerrainSize + 1} {
		if _, err := ParseWildernessTerrainTable(make([]byte, size)); err == nil {
			t.Errorf("a %d-byte table was accepted", size)
		}
	}
}

// 真檔錨點：START.EXE 檔案位移 `35E2h + 30640` 起的 44×36 bytes。
func TestReadDOSWildernessTerrainTable(t *testing.T) {
	table, err := ReadDOSWildernessTerrainTable(filepath.Join("..", "..", "Pool of Radiance (1988).zip"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	if table[0] != 0x01 || table[8*WildernessTerrainWidth+10] != 0xC9 || table[35*WildernessTerrainWidth+43] != 0x4E {
		t.Fatalf("anchors = %02X %02X %02X, want 01 C9 4E", table[0],
			table[8*WildernessTerrainWidth+10], table[35*WildernessTerrainWidth+43])
	}
	// 船的登陸點：野外 27（模式 4）的 (9,29) → 表的 (35,29)。
	if got, err := table.Background(CombatAreaWildernessEast, 9, 29); err != nil || got != 0xEC {
		t.Fatalf("landing background = %02Xh, %v; want ECh", got, err)
	}
	if _, err := table.Background(CombatAreaIndoor, 9, 29); err == nil {
		t.Error("indoor mode has no wilderness sheet but Background accepted it")
	}
	if _, err := table.Background(CombatAreaWildernessEast, 20, 29); err == nil {
		t.Error("x 20 + 26 = 46 is past the 44-wide table but was accepted")
	}
}
