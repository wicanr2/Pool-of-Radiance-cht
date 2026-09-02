package gamepack_test

import (
	"os"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// 七名預設人物記錄裡已經算好的 `+6Dh` 五格，要能從表加職業等級重算出來。
// 這是對著原始位元組的驗證：表讀錯、索引算錯、或取最小值那一步漏掉，
// 任何一個都會讓某一列對不上。
func TestSavingThrowTargetsMatchThePremadeRecords(t *testing.T) {
	table, err := gamepack.ReadDOSSavingThrowTable(dosZIP)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	for _, name := range []string{
		"chrdatd1", "chrdatd2", "chrdatd3", "chrdatd4", "chrdatd5", "chrdatd6", "chrdatd7",
	} {
		record, err := os.ReadFile("../../workplace/oracle/dos/" + name + ".sav")
		if err != nil {
			t.Skipf("original character records unavailable: %v", err)
		}
		var levels [gamepack.SavingThrowTableClasses]uint8
		copy(levels[:], record[gamepack.ClassLevelsOffset:])
		got, err := table.TargetsForLevels(levels)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		want, err := gamepack.SavingThrowTargets(record)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if got != want {
			t.Fatalf("%s levels %v give %v, the record stores %v", name, levels, got, want)
		}
	}
}

// 表的三個代表列與規則書相同。少了這一條，上面那個測試只證明「我算的和
// 原版算的一樣」，證明不了兩邊都對。
func TestSavingThrowTableMatchesTheRulebook(t *testing.T) {
	table, err := gamepack.ReadDOSSavingThrowTable(dosZIP)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	for _, item := range []struct {
		slot  int
		level uint8
		want  [gamepack.SavingThrowCategories]uint8
		note  string
	}{
		{gamepack.ClassSlotCleric, 1, [5]uint8{10, 13, 14, 16, 15}, "牧師 1..3"},
		{gamepack.ClassSlotFighter, 1, [5]uint8{14, 15, 16, 17, 17}, "戰士 1..2"},
		{gamepack.ClassSlotMagicUser, 1, [5]uint8{14, 13, 11, 15, 12}, "法師 1..5"},
		{gamepack.ClassSlotThief, 1, [5]uint8{13, 12, 14, 16, 15}, "賊 1..4"},
		// 賊第 9 級落進下一個職業的第 0 列，那是刻意的接續。
		{gamepack.ClassSlotThief, 9, [5]uint8{11, 10, 10, 14, 11}, "賊 9..12"},
	} {
		var levels [gamepack.SavingThrowTableClasses]uint8
		levels[item.slot] = item.level
		got, err := table.TargetsForLevels(levels)
		if err != nil {
			t.Fatalf("%s: %v", item.note, err)
		}
		if got != item.want {
			t.Fatalf("%s gives %v, the rulebook says %v", item.note, got, item.want)
		}
	}
}
