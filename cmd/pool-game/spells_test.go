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
		{1, "持續 6 回合　不可豁免"},
		{6, "每級持續 3 回合　不可豁免"},
		{23, "持續 4 回合，每級再加 1　可豁免（對法術）"},
		{4, "持續到解除為止　不可豁免　須擲中才生效"},
		{34, "每級持續 1 回合　可豁免（對毒）"},
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
