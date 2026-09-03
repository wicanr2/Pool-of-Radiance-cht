package gamepack_test

import (
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

const dosZIP = "../../Pool of Radiance (1988).zip"

// 表的形狀由檔案大小決定：2,050 ＝ 2-byte 檔頭加 128 筆 × 16。
func TestReadDOSItemTypeTable(t *testing.T) {
	table, err := gamepack.ReadDOSItemTypeTable(dosZIP)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	if len(table.Entries) != gamepack.ItemTypeCount {
		t.Fatalf("table has %d entries", len(table.Entries))
	}
}

// 正對照：ITEM3.DAX/33h 的雙手劍記錄 `+2Eh` 是 26h，而表的第 26h 筆必須是
// AD&D 雙手劍的 1d10（中小型）／3d6（大型）。這一條把「poolrad/items 就是
// DS:54E0h 那張表」與「索引就是記錄的 +2Eh」一次釘住；對不上就是我們讀錯了
// 表或讀錯了欄位，而兩者在別的測試裡都看不出來。
func TestTreasureSwordIndexesTheTwoHandedSwordEntry(t *testing.T) {
	table, err := gamepack.ReadDOSItemTypeTable(dosZIP)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	records, err := gamepack.ReadDOSTreasureItemBlock(dosZIP, 3, 0x33)
	if err != nil {
		t.Fatal(err)
	}
	var sword *gamepack.TreasureItemRecord
	for index := range records {
		if records[index].Name == "Two-Handed Sword +1 +3 vs. Undead" {
			sword = &records[index]
		}
	}
	if sword == nil {
		t.Fatal("the graveyard treasure has no two-handed sword")
	}
	if got := sword.Raw[0x2e]; got != 0x26 {
		t.Fatalf("the sword's item type is %#02x, want 0x26", got)
	}
	if got := sword.Raw[0x32]; got != 1 {
		t.Fatalf("the sword's plus is %d, want 1 to match its name", got)
	}
	entry, err := table.Entry(sword.Raw[0x2e])
	if err != nil {
		t.Fatal(err)
	}
	if count, sides := entry.Damage(); count != 1 || sides != 10 {
		t.Fatalf("entry 0x26 does %dd%d against small targets, want 1d10", count, sides)
	}
	if count, sides := entry.LargeDamage(); count != 3 || sides != 6 {
		t.Fatalf("entry 0x26 does %dd%d against large targets, want 3d6", count, sides)
	}
}

// 壞掉的表要失敗即關閉，不能安靜地少幾筆。
func TestParseItemTypeTableRejectsTheWrongSize(t *testing.T) {
	if _, err := gamepack.ParseItemTypeTable(make([]byte, 2049)); err == nil {
		t.Fatal("a short table was accepted")
	}
	if _, err := gamepack.ParseItemTypeTable(nil); err == nil {
		t.Fatal("an empty table was accepted")
	}
}

// 索引超出表要報錯，不能回一筆空的：空的一筆在畫面上是「這把武器沒有傷害」，
// 看起來像規則錯，不像資料越界。
func TestItemTypeEntryRejectsAnIndexOutsideTheTable(t *testing.T) {
	table, err := gamepack.ParseItemTypeTable(make([]byte, 2050))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := table.Entry(gamepack.ItemTypeCount); err == nil {
		t.Fatal("an out-of-range item type was accepted")
	}
	if _, err := table.Entry(gamepack.ItemTypeCount - 1); err != nil {
		t.Fatalf("the last entry was rejected: %v", err)
	}
}

// 射程逐筆對原版的表：近戰武器存 0（＝相鄰），遠程的存「射程加一」。
func TestItemTypeAttackRange(t *testing.T) {
	table, err := gamepack.ReadDOSItemTypeTable(dosZIP)
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	for _, testCase := range []struct {
		itemType uint8
		raw      uint8
		want     int
	}{
		{0, 0, 1},  // 近戰：表裡 0，打相鄰
		{1, 0, 1},  // 近戰
		{2, 4, 3},  // 遠程：表裡 4 → 3 格
		{8, 1, 1},  // 表裡 1 也是相鄰（原版 dec 之後變 0，呼叫端墊成 1）
		{9, 6, 5},  // 表裡 6 → 5 格
		{21, 7, 6}, // 表裡 7 → 6 格
	} {
		entry, err := table.Entry(testCase.itemType)
		if err != nil {
			t.Fatalf("型別 %d: %v", testCase.itemType, err)
		}
		if entry.Raw[0x0c] != testCase.raw {
			t.Errorf("型別 %d 的 +0Ch 是 %d，原版是 %d",
				testCase.itemType, entry.Raw[0x0c], testCase.raw)
		}
		if got := entry.AttackRange(); got != testCase.want {
			t.Errorf("型別 %d 的射程算成 %d，預期 %d", testCase.itemType, got, testCase.want)
		}
	}
}
