package gamepack_test

import (
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// 兩張表的第 2..6 級與 AD&D 逐項相同。
func TestSpellSlotTablesMatchTheOriginalProgressions(t *testing.T) {
	cleric, magicUser, err := gamepack.ReadDOSSpellSlotTables(dosZIP)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	for level, want := range map[int]gamepack.SpellSlots{
		2: {2, 0, 0}, 3: {2, 1, 0}, 4: {3, 2, 0}, 5: {3, 3, 1}, 6: {3, 3, 2},
	} {
		if cleric[level] != want {
			t.Fatalf("cleric level %d has slots %v, want %v", level, cleric[level], want)
		}
	}
	for level, want := range map[int]gamepack.SpellSlots{
		2: {2, 0, 0}, 3: {2, 1, 0}, 4: {3, 2, 0}, 5: {4, 2, 1}, 6: {4, 2, 2},
	} {
		if magicUser[level] != want {
			t.Fatalf("magic-user level %d has slots %v, want %v", level, magicUser[level], want)
		}
	}
}

// 表算出來的格數要與原版預設人物記錄裡**已經算好的**那三個 byte 相同。
// 這是整條規則的端對端驗證：表位置、+1 偏移、睿智加成、記錄欄位，
// 任一處錯了這一條就不會過。
func TestComputedSlotsMatchThePremadeRecords(t *testing.T) {
	cleric, magicUser, err := gamepack.ReadDOSSpellSlotTables(dosZIP)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	for _, item := range []struct {
		file  string
		class gamepack.SpellClass
		level int
	}{
		{"chrdatd3.sav", gamepack.SpellClassCleric, 6},
		{"chrdatd5.sav", gamepack.SpellClassMagicUser, 6},
		{"chrdatd6.sav", gamepack.SpellClassMagicUser, 6},
	} {
		record := readMember(t, item.file)
		stored, err := gamepack.RecordSpellSlots(record, item.class)
		if err != nil {
			t.Fatal(err)
		}
		computed := magicUser[item.level]
		if item.class == gamepack.SpellClassCleric {
			wisdom, err := gamepack.RecordWisdom(record)
			if err != nil {
				t.Fatal(err)
			}
			computed = gamepack.WisdomBonusSlots(wisdom, cleric[item.level])
		}
		if stored != computed {
			t.Fatalf("%s stores slots %v but the tables compute %v", item.file, stored, computed)
		}
	}
}

// 睿智加成逐段各加一次，而且只加在本來就有格子的等級上。
// 少了那個守衛，一個第 2 級的牧師會憑空得到第三級法術的格子。
func TestWisdomBonusOnlyRaisesLevelsThatAlreadyHaveSlots(t *testing.T) {
	base := gamepack.SpellSlots{3, 3, 2}
	for wisdom, want := range map[int]gamepack.SpellSlots{
		12: {3, 3, 2}, 13: {4, 3, 2}, 14: {5, 3, 2}, 15: {5, 4, 2},
		16: {5, 5, 2}, 17: {5, 5, 3}, 18: {5, 5, 3},
	} {
		if got := gamepack.WisdomBonusSlots(wisdom, base); got != want {
			t.Fatalf("wisdom %d gives %v, want %v", wisdom, got, want)
		}
	}
	// 第 2 級牧師只有第一級的格子：高睿智不得憑空給出第二、三級的格子。
	low := gamepack.SpellSlots{2, 0, 0}
	if got := gamepack.WisdomBonusSlots(18, low); got != (gamepack.SpellSlots{4, 0, 0}) {
		t.Fatalf("wisdom 18 on %v gives %v, want [4 0 0]", low, got)
	}
}

// 解析結果只涵蓋第 2..6 級，其餘留零。overlay-23 對等級 1 直接跳過
//（`cmp [bp-3], 1; jle`），檔案裡那一列是 FFh 佔位；第 1 級的格數由別處寫入，
// 本規格不涵蓋。呼叫端要自己擋住第 1 級——把這裡的零當答案，
// 一級施法者會一個法術都記不了。
func TestSlotTableDoesNotCoverLevelOne(t *testing.T) {
	cleric, magicUser, err := gamepack.ReadDOSSpellSlotTables(dosZIP)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	for _, table := range [][]gamepack.SpellSlots{cleric, magicUser} {
		if table[1] != (gamepack.SpellSlots{}) {
			t.Fatalf("level 1 row is %v, want zeros", table[1])
		}
		if table[2] == (gamepack.SpellSlots{}) {
			t.Fatal("level 2 is zero; the table did not load")
		}
	}
}
