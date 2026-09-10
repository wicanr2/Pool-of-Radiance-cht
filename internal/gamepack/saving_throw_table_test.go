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

// 玩家角色的攻擊次數編碼：戰士 7 級以上是 3（每兩回合三次），其餘一律 2
// （每回合一次）。出處 overlay-23 `007Ch..009Dh`（spec 072）。
func TestPlayerAttackRateTurnsOverAtFighterSeven(t *testing.T) {
	rate := func(fighter, cleric, mage, thief uint8) uint8 {
		var levels [gamepack.ClassThac0ClassCount]uint8
		levels[gamepack.ClassSlotFighter] = fighter
		levels[gamepack.ClassSlotCleric] = cleric
		levels[gamepack.ClassSlotMagicUser] = mage
		levels[gamepack.ClassSlotThief] = thief
		return gamepack.PlayerAttackRate(levels)
	}
	for _, one := range []struct {
		fighter uint8
		want    uint8
		why     string
	}{
		{0, 2, "非戰士"},
		{1, 2, "戰士一級"},
		{6, 2, "戰士六級還沒到"},
		{7, 3, "戰士七級開始 3/2"},
		{9, 3, "更高等級維持 3"},
	} {
		if got := rate(one.fighter, 0, 0, 0); got != one.want {
			t.Errorf("%s（戰士 %d 級）得到 %d，預期 %d", one.why, one.fighter, got, one.want)
		}
	}
	// **只看戰士那一格**：別的職業再高也不會多打一下。
	if got := rate(0, 12, 12, 12); got != 2 {
		t.Errorf("牧師／法師／賊各 12 級卻得到 %d，預期 2", got)
	}
	// 多職業裡只要戰士那一格到 7 就算。
	if got := rate(7, 9, 0, 0); got != 3 {
		t.Errorf("戰士七級兼牧師九級得到 %d，預期 3", got)
	}
}
