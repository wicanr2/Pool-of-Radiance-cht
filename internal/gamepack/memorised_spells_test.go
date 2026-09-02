package gamepack_test

import (
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// 三個預設施法者的記憶法術逐格解出來，而且**每一條都是自己職業的法術**。
// 這是 1-based 編號與陣列位置兩件事同時成立的證據：任一件讀錯，
// 牧師會拿到巫術、或編號整批位移一格而變成別的法術。
func TestPremadeCastersMemoriseTheirOwnClassSpells(t *testing.T) {
	catalogue, err := gamepack.TraditionalChineseSpells()
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct {
		file  string
		class gamepack.SpellClass
		want  []string
	}{
		{"chrdatd3.sav", gamepack.SpellClassCleric, []string{
			"Cure Light Wounds", "Cure Light Wounds", "Cure Light Wounds",
			"Cure Light Wounds", "Cure Light Wounds",
			"Hold Person", "Hold Person", "Hold Person", "Hold Person", "Hold Person",
			"Cure Disease", "Dispel Magic", "Prayer"}},
		{"chrdatd5.sav", gamepack.SpellClassMagicUser, []string{
			"Magic Missile", "Magic Missile", "Magic Missile",
			"Stinking Cloud", "Stinking Cloud", "Fireball", "Fireball"}},
		{"chrdatd6.sav", gamepack.SpellClassMagicUser, []string{
			"Magic Missile", "Magic Missile", "Magic Missile", "Magic Missile",
			"Invisibility", "Stinking Cloud", "Fireball", "Fireball"}},
	} {
		record := readMember(t, item.file)
		memorised, err := gamepack.MemorisedSpells(record)
		if err != nil {
			t.Fatalf("%s: %v", item.file, err)
		}
		if len(memorised) != len(item.want) {
			t.Fatalf("%s has %d memorised spells, want %d", item.file, len(memorised), len(item.want))
		}
		for index, entry := range memorised {
			spell, err := catalogue.SpellByID(entry.ID)
			if err != nil {
				t.Fatalf("%s slot %d: %v", item.file, entry.Slot, err)
			}
			if spell.Name != item.want[index] {
				t.Fatalf("%s slot %d is %q, want %q", item.file, entry.Slot, spell.Name, item.want[index])
			}
			if spell.Class != item.class {
				t.Fatalf("%s memorised a %s spell (%q)", item.file, spell.Class, spell.Name)
			}
			if entry.Flagged {
				t.Fatalf("%s slot %d has bit 7 set in a rested save", item.file, entry.Slot)
			}
		}
	}
}

// 非施法者的格子全是零。
func TestPremadeFighterHasNoMemorisedSpells(t *testing.T) {
	memorised, err := gamepack.MemorisedSpells(readMember(t, "chrdatb1.sav"))
	if err != nil {
		t.Fatal(err)
	}
	if len(memorised) != 0 {
		t.Fatalf("the pre-made fighter has %d memorised spells", len(memorised))
	}
}

// 編號是 1-based：1 是名稱表的第一筆，0 是空格而不是第一筆。
func TestSpellIDsAreOneBased(t *testing.T) {
	catalogue, err := gamepack.TraditionalChineseSpells()
	if err != nil {
		t.Fatal(err)
	}
	first, err := catalogue.SpellByID(1)
	if err != nil {
		t.Fatal(err)
	}
	if first.Name != "Bless" || first.Index != 0 {
		t.Fatalf("spell id 1 is %q at index %d, want Bless at 0", first.Name, first.Index)
	}
	last, err := catalogue.SpellByID(uint8(gamepack.SpellNameCount))
	if err != nil {
		t.Fatal(err)
	}
	if last.Name != "Restoration" {
		t.Fatalf("spell id %d is %q, want Restoration", gamepack.SpellNameCount, last.Name)
	}
	if _, err := catalogue.SpellByID(0); err == nil {
		t.Fatal("id 0 was accepted as a spell")
	}
	if _, err := catalogue.SpellByID(uint8(gamepack.SpellNameCount) + 1); err == nil {
		t.Fatal("an id past the table was accepted")
	}
}

// 超出表的編號要失敗即關閉：位移一格之後每一條都指錯，而畫面上看起來
// 只是「法術名怪怪的」。
func TestMemorisedSpellsRejectAnIDOutsideTheTable(t *testing.T) {
	record := make([]byte, 285)
	record[gamepack.MemorisedSpellOffset] = uint8(gamepack.SpellNameCount) + 1
	if _, err := gamepack.MemorisedSpells(record); err == nil {
		t.Fatal("an out-of-range spell id was accepted")
	}
	if _, err := gamepack.MemorisedSpells(make([]byte, 10)); err == nil {
		t.Fatal("a truncated record was accepted")
	}
}
