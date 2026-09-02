package gamepack_test

import (
	"os"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// 七名預設人物的移動力逐一算得出來。這是整條管線的端對端驗證：基礎值、
// 盔甲分段、魔法盔甲的 +3、力量的重量寬容、負重分段、以及「只往下壓」，
// 任一處錯了就對不上。
func TestMovementRateMatchesThePremadeCharacters(t *testing.T) {
	types, err := gamepack.ReadDOSItemTypeTable(dosZIP)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	for _, item := range []struct {
		name string
		want int
		note string
	}{
		{"chrdatd1", 3, "戰士 8，Plate Mail +3，負重超過寬容 1024"},
		{"chrdatd2", 9, "盔甲重 450 有加值 → 6+3"},
		{"chrdatd3", 9, "同上"},
		{"chrdatd4", 12, "盔甲重 150 → 基礎值，負重在寬容內"},
		{"chrdatd5", 3, "法師，無盔甲，負重超過寬容 1024"},
		{"chrdatd6", 3, "同上"},
		{"chrdatd7", 12, "盔甲重 150 無加值，負重輕"},
	} {
		record, err := os.ReadFile("../../workplace/oracle/dos/" + item.name + ".sav")
		if err != nil {
			t.Skipf("original character records unavailable: %v", err)
		}
		raw, err := os.ReadFile("../../workplace/oracle/dos/" + item.name + ".itm")
		if err != nil {
			t.Skipf("original item records unavailable: %v", err)
		}
		items := make([][]byte, 0, len(raw)/63)
		for offset := 0; offset+63 <= len(raw); offset += 63 {
			items = append(items, raw[offset:offset+63])
		}
		rate, err := gamepack.MovementRate(record, items, types)
		if err != nil {
			t.Fatalf("%s: %v", item.name, err)
		}
		if rate != item.want {
			t.Fatalf("%s (%s) moves %d, want %d", item.name, item.note, rate, item.want)
		}
		// 原版記錄裡已經算好的那個 byte 就是答案。
		if int(record[gamepack.CurrentMovementOffset]) != item.want {
			t.Fatalf("%s stores %d in +11Ch, want %d", item.name, record[gamepack.CurrentMovementOffset], item.want)
		}
	}
}

// 力量的重量寬容分段。18/00 與 18/100 差很多，正是預設人物之間的差別來源。
func TestStrengthWeightAllowance(t *testing.T) {
	for _, item := range []struct {
		strength, exceptional, want int
	}{
		{3, 0, -350}, {5, 0, -250}, {7, 0, -150}, {10, 0, 0},
		{13, 0, 100}, {15, 0, 200}, {16, 0, 350},
		{18, 0, 750},   // 索引 18 → 500 + 250
		{18, 1, 1000},  // 索引 19
		{18, 100, 3000}, // 索引 23 → 2000 + 1000
	} {
		index, err := gamepack.StrengthTableIndex(item.strength, item.exceptional)
		if err != nil {
			t.Fatal(err)
		}
		if got := gamepack.StrengthWeightAllowance(int(index)); got != item.want {
			t.Fatalf("strength %d/%02d allows %d, want %d", item.strength, item.exceptional, got, item.want)
		}
	}
}

// 負重只會往下壓，不會把盔甲壓低的值調回去。
func TestEncumbranceOnlyLowersTheRate(t *testing.T) {
	if got := gamepack.EncumbranceMovementRate(100, 0, 6); got != 6 {
		t.Fatalf("a light load raised the rate to %d", got)
	}
	if got := gamepack.EncumbranceMovementRate(2000, 0, 12); got != 3 {
		t.Fatalf("a crushing load left the rate at %d", got)
	}
	if got := gamepack.EncumbranceMovementRate(600, 0, 6); got != 6 {
		t.Fatalf("the 9 bucket raised a 6 to %d", got)
	}
}
