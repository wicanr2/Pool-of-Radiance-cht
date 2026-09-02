package main

import (
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// 法術一覽那一行的三件事來自原版參數表（spec 074），不是說明書。抽五條性質
// 不同的：固定回合、每級回合、兩者都有、可豁免、須擲中。
func TestSpellFactsComeFromTheOriginalParameters(t *testing.T) {
	parameters, err := gamepack.ReadDOSSpellParameters(dosZIPForTests)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	catalogue, err := gamepack.TraditionalChineseSpells()
	if err != nil {
		t.Fatal(err)
	}
	a := &app{language: languageTraditionalChinese, spellParameters: parameters}
	for _, item := range []struct {
		id   int
		want string
	}{
		{1, "射程 6 格　持續 6 回合　不可豁免"},
		{6, "射程 1 格　每級 3 回合　不可豁免"},
		{23, "射程 6 格　持續 4 回合（每級 +1）　可豁免（法術）"},
		{4, "射程 1 格　持續到解除　不可豁免　須擲中"},
		{34, "射程 3 格　每級 1 回合　可豁免（毒）"},
	} {
		spell, err := catalogue.SpellByID(uint8(item.id))
		if err != nil {
			t.Fatal(err)
		}
		if got := a.spellFacts(spell); got != item.want {
			t.Fatalf("spell %d (%s) reads %q, want %q", item.id, spell.Name, got, item.want)
		}
	}
}

// 這一行要塞得進畫面：左邊界 x=48、右邊界 x=608，一個半形 8 像素，
// 所以最多 70 欄。每一條都量過，不能有一條爆版。
func TestSpellFactsFitTheScreen(t *testing.T) {
	parameters, err := gamepack.ReadDOSSpellParameters(dosZIPForTests)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	catalogue, err := gamepack.TraditionalChineseSpells()
	if err != nil {
		t.Fatal(err)
	}
	for _, language := range []language{languageEnglish, languageTraditionalChinese} {
		a := &app{language: language, spellParameters: parameters}
		for id := 1; id <= gamepack.SpellNameCount; id++ {
			spell, err := catalogue.SpellByID(uint8(id))
			if err != nil {
				t.Fatal(err)
			}
			line := a.spellFacts(spell)
			width := 0
			for _, r := range line {
				width += runeWidth(r)
			}
			if width > 70 {
				t.Fatalf("spell %d (%s) reads %q, %d columns wide", id, spell.Name, line, width)
			}
		}
	}
}
