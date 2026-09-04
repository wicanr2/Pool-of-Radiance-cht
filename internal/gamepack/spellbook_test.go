package gamepack_test

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

var spellbookZIP = filepath.Join("..", "..", "Pool of Radiance (1988).zip")

// 二十個預設人物檔逐格核對法術書（spec 110）。
//
// 這一則是這個欄位的全部證據：位置沒有反組譯過的讀寫可以直接引用——原版
// 一律寫成「記錄基底加編號再取 `+32h`」，而 `+32h` 本身是最大生命值，
// 所以位元組搜尋分不出兩者。分得出來的是資料：二十個檔、四種職業、
// 三個等級層，零反例。
func TestPregeneratedSpellbooksAreClassAndLevelConsistent(t *testing.T) {
	parameters, err := gamepack.ReadDOSSpellParameters(spellbookZIP)
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	names, err := filepath.Glob("../../workplace/oracle/dos/chrdat*.sav")
	if err != nil || len(names) == 0 {
		t.Skipf("original character records unavailable: %v", err)
	}
	sort.Strings(names)
	casters := 0
	for _, path := range names {
		record, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		known, err := gamepack.RecordSpellbook(record)
		if err != nil {
			t.Fatal(err)
		}
		if len(known) == 0 {
			continue
		}
		casters++
		sources := map[gamepack.SpellSource]bool{}
		for _, id := range known {
			if value := record[gamepack.SpellbookRecordBase+int(id)]; value != gamepack.SpellbookKnown {
				t.Errorf("%s 的第 %d 條是 %d，法術書只放 1", filepath.Base(path), id, value)
			}
			if int(id) >= len(parameters) {
				t.Fatalf("%s 會編號 %d，超出參數表", filepath.Base(path), id)
			}
			sources[parameters[id].Source()] = true
		}
		// 一個人只會一種來源的法術。位置或編號的基底讀錯的話，
		// 牧師與法師的法術就會混在同一本書裡。
		if len(sources) != 1 {
			t.Errorf("%s 的書裡混了 %d 種來源：%v", filepath.Base(path), len(sources), sources)
		}
	}
	if casters == 0 {
		t.Fatal("二十個預設檔裡一個施法者都沒讀到；欄位一定讀錯了")
	}
	t.Logf("%d 個檔，其中 %d 個有法術書", len(names), casters)
}

// 牧師的書＝「那一級有格子就全部會」；法師的書＝起手四條加上後來學的。
// 這一則拿 ALFRED（牧師 6）與 TARRY／CARRY（法師 6）當樣本。
func TestPregeneratedSpellbooksMatchTheTwoRules(t *testing.T) {
	parameters, err := gamepack.ReadDOSSpellParameters(spellbookZIP)
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	tables, err := gamepack.ReadDOSSpellSlotTableSet(spellbookZIP)
	if err != nil {
		t.Fatal(err)
	}
	alfred, err := os.ReadFile("../../workplace/oracle/dos/chrdatd3.sav")
	if err != nil {
		t.Skipf("original character records unavailable: %v", err)
	}
	known, err := gamepack.RecordSpellbook(alfred)
	if err != nil {
		t.Fatal(err)
	}
	// ALFRED 是牧師 6：第 1..3 級都有格子，所以三級的神術一條不缺。
	var levels [8]uint8
	levels[gamepack.ClassSlotCleric] = 6
	want := gamepack.RefreshClericSpellbook(nil, levels, int(alfred[0x12]), tables, parameters)
	if len(want) != len(known) {
		t.Fatalf("ALFRED 書裡 %d 條，「有格子就全會」算出 %d 條", len(known), len(want))
	}
	for index := range want {
		if want[index] != known[index] {
			t.Fatalf("第 %d 條：算出 %d，記錄裡是 %d", index, want[index], known[index])
		}
	}

	// 起手四條在兩名法師的書裡都在。
	for _, name := range []string{"chrdatd5", "chrdatd6", "chrdatb5", "chrdatb6"} {
		record, err := os.ReadFile("../../workplace/oracle/dos/" + name + ".sav")
		if err != nil {
			t.Fatal(err)
		}
		book, err := gamepack.RecordSpellbook(record)
		if err != nil {
			t.Fatal(err)
		}
		var mage [8]uint8
		mage[gamepack.ClassSlotMagicUser] = 1
		for _, id := range gamepack.NewCharacterSpellbook(mage, 10, tables, parameters) {
			if !gamepack.SpellbookKnows(book, id) {
				t.Errorf("%s 的書裡沒有起手的第 %d 條", name, id)
			}
		}
	}
}

// 建角當下：牧師只會第 1 級的神術（那時只有第 1 級有格子），法師會四條。
func TestNewCharacterSpellbookFollowsTheCreationRules(t *testing.T) {
	parameters, err := gamepack.ReadDOSSpellParameters(spellbookZIP)
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	tables, err := gamepack.ReadDOSSpellSlotTableSet(spellbookZIP)
	if err != nil {
		t.Fatal(err)
	}
	var cleric [8]uint8
	cleric[gamepack.ClassSlotCleric] = 1
	book := gamepack.NewCharacterSpellbook(cleric, 10, tables, parameters)
	if len(book) == 0 {
		t.Fatal("新牧師一條神術都不會")
	}
	for _, id := range book {
		entry := parameters[id]
		if entry.Source() != 0 || entry.Level() != 1 {
			t.Errorf("新牧師會第 %d 條（來源 %d、等級 %d），只該有第 1 級神術",
				id, entry.Source(), entry.Level())
		}
	}

	var mage [8]uint8
	mage[gamepack.ClassSlotMagicUser] = 1
	book = gamepack.NewCharacterSpellbook(mage, 10, tables, parameters)
	if got := len(book); got != 4 {
		t.Fatalf("新法師會 %d 條，原版寫死四條", got)
	}
	for _, id := range book {
		if entry := parameters[id]; entry.Source() != 1 || entry.Level() != 1 {
			t.Errorf("新法師會第 %d 條（來源 %d、等級 %d）", id, entry.Source(), entry.Level())
		}
	}
}
