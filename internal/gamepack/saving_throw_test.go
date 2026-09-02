package gamepack_test

import (
	"os"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// 七名預設人物的 `+6Dh..+71h` 逐列與 AD&D 規則書的豁免表相同，五個
// 職業等級組合全中。順序也因此定下來：癱瘓／毒／死亡、石化／變形、
// 法杖／魔杖／權杖、吐息、法術。
func TestSavingThrowTargetsMatchTheOriginalTables(t *testing.T) {
	for _, item := range []struct {
		file string
		want [gamepack.SavingThrowCategories]uint8
		who  string
	}{
		{"chrdatd1.sav", [5]uint8{10, 11, 12, 12, 13}, "戰士 8"},
		{"chrdatd3.sav", [5]uint8{9, 12, 13, 15, 14}, "牧師 6"},
		{"chrdatd4.sav", [5]uint8{11, 10, 10, 14, 11}, "賊 9"},
		{"chrdatd5.sav", [5]uint8{13, 11, 9, 13, 10}, "法師 6"},
		{"chrdatd7.sav", [5]uint8{13, 14, 15, 16, 16}, "戰士 4"},
	} {
		record, err := os.ReadFile("../../workplace/oracle/dos/" + item.file)
		if err != nil {
			t.Skipf("original character records unavailable: %v", err)
		}
		targets, err := gamepack.SavingThrowTargets(record)
		if err != nil {
			t.Fatal(err)
		}
		if targets != item.want {
			t.Fatalf("%s (%s) saves are %v, want %v", item.file, item.who, targets, item.want)
		}
	}
}

// 自然 1 與自然 20 在加修正之前就決定結果。
func TestSavingThrowNaturalRollsBypassTheTarget(t *testing.T) {
	record := make([]byte, 285)
	for index := range gamepack.SavingThrowCategories {
		record[gamepack.SavingThrowOffset+index] = 20
	}
	record[gamepack.SavingThrowBonusOffset] = 10
	if made, err := gamepack.SavingThrow(record, gamepack.SaveSpell, 10, 1); err != nil || made {
		t.Fatalf("a natural 1 saved (%v, %v)", made, err)
	}
	for index := range gamepack.SavingThrowCategories {
		record[gamepack.SavingThrowOffset+index] = 21
	}
	record[gamepack.SavingThrowBonusOffset] = 0
	if made, err := gamepack.SavingThrow(record, gamepack.SaveSpell, 0, 20); err != nil || !made {
		t.Fatalf("a natural 20 failed (%v, %v)", made, err)
	}
}

// 目標值 ≤ 骰值加修正才成功，剛好等於也算成功。
func TestSavingThrowComparesAgainstTheAdjustedRoll(t *testing.T) {
	record := make([]byte, 285)
	record[gamepack.SavingThrowOffset+int(gamepack.SaveBreathWeapon)] = 15
	record[gamepack.SavingThrowBonusOffset] = 2
	for _, item := range []struct {
		roll     int
		modifier int
		want     bool
	}{
		{13, 0, true}, {12, 0, false}, {12, 1, true}, {10, 2, false},
	} {
		made, err := gamepack.SavingThrow(record, gamepack.SaveBreathWeapon, item.modifier, item.roll)
		if err != nil {
			t.Fatal(err)
		}
		if made != item.want {
			t.Fatalf("roll %d with %+d saved=%v, want %v", item.roll, item.modifier, made, item.want)
		}
	}
}

// 參數表的 `+7` 落在 0..4，而且只有兩個法術不是「對法術」的豁免：
// Stinking Cloud 是毒、編號 61 的癱瘓效果是癱瘓，兩個都歸在同一類。
func TestSpellSaveCategoriesAreAlmostAlwaysSpell(t *testing.T) {
	table, err := gamepack.ReadDOSSpellParameters(dosZIP)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	exceptions := map[int]bool{34: true, 61: true}
	for _, record := range table[1:] {
		category := record.SaveCategory()
		if category >= gamepack.SavingThrowCategories {
			t.Fatalf("spell %d has save category %d", record.SpellID, category)
		}
		want := gamepack.SaveSpell
		if exceptions[record.SpellID] {
			want = gamepack.SaveParalyzation
		}
		if category != want {
			t.Fatalf("spell %d has save category %d, want %d", record.SpellID, category, want)
		}
	}
}
