package gamepack_test

import (
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// 內建的法術表必須與 START.EXE 裡那張逐字相同。分開存是為了配中譯，
// 但只要有一條抄錯，畫面上就會出現原版沒有的法術名，而那看起來像翻譯問題。
func TestBuiltInSpellNamesMatchTheOriginalTable(t *testing.T) {
	names, err := gamepack.ReadDOSSpellNames(dosZIP)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	catalogue, err := gamepack.TraditionalChineseSpells()
	if err != nil {
		t.Fatal(err)
	}
	spells := catalogue.Spells()
	if len(spells) != len(names) {
		t.Fatalf("catalogue has %d spells, the original table has %d", len(spells), len(names))
	}
	for index, name := range names {
		if spells[index].Name != name {
			t.Fatalf("spell %d is %q in the catalogue and %q in START.EXE",
				index, spells[index].Name, name)
		}
	}
}

// 原版把兩類法術放在同一張表裡，用順序分組。分組界線與說明書下冊第六章的
// LEVEL 標題一致，每組的筆數固定。
func TestSpellGroupsMatchTheManualChapter(t *testing.T) {
	catalogue, err := gamepack.TraditionalChineseSpells()
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct {
		class gamepack.SpellClass
		level int
		count int
		first string
		last  string
	}{
		{gamepack.SpellClassCleric, 1, 8, "Bless", "Resist Cold"},
		{gamepack.SpellClassMagicUser, 1, 13, "Burning Hands", "Sleep"},
		{gamepack.SpellClassCleric, 2, 7, "Find Traps", "Spiritual Hammer"},
		{gamepack.SpellClassMagicUser, 2, 7, "Detect Invisibility", "Strength"},
		{gamepack.SpellClassCleric, 3, 9, "Animate Dead", "Bestow Curse"},
		{gamepack.SpellClassMagicUser, 3, 12, "Blink", "Restoration"},
	} {
		group := catalogue.ByClassAndLevel(item.class, item.level)
		if len(group) != item.count {
			t.Fatalf("%s level %d has %d spells, want %d", item.class, item.level, len(group), item.count)
		}
		if group[0].Name != item.first || group[len(group)-1].Name != item.last {
			t.Fatalf("%s level %d runs %q..%q, want %q..%q",
				item.class, item.level, group[0].Name, group[len(group)-1].Name, item.first, item.last)
		}
	}
}

// 每一條都要有中譯，而且中譯不得重複到讓兩條法術在畫面上分不出來。
// 同名不同級的（Dispel Magic、Hold Person…）由職業與等級區分，這裡只查
// 「同一職業同一等級之內不重名」。
func TestEverySpellHasADistinctTranslationWithinItsGroup(t *testing.T) {
	catalogue, err := gamepack.TraditionalChineseSpells()
	if err != nil {
		t.Fatal(err)
	}
	for _, class := range []gamepack.SpellClass{gamepack.SpellClassCleric, gamepack.SpellClassMagicUser} {
		for level := 1; level <= 3; level++ {
			seen := map[string]string{}
			for _, spell := range catalogue.ByClassAndLevel(class, level) {
				if previous, clash := seen[spell.Text]; clash {
					t.Fatalf("%s level %d: %q and %q share the translation %q",
						class, level, previous, spell.Name, spell.Text)
				}
				seen[spell.Text] = spell.Name
			}
		}
	}
}

// 壞掉的表要失敗即關閉：少一條會讓法術編號整批位移，而位移之後每一條都指錯。
func TestParseSpellCatalogueRejectsAShortTable(t *testing.T) {
	if _, err := gamepack.ParseSpellCatalogue([]byte(
		`{"schema":"pool-spell-names/1","locale":"zh-TW","entries":[]}`)); err == nil {
		t.Fatal("an empty spell catalogue was accepted")
	}
	if _, err := gamepack.ParseSpellCatalogue([]byte(
		`{"schema":"pool-spell-names/2","entries":[]}`)); err == nil {
		t.Fatal("a foreign schema was accepted")
	}
}

// 除了 Restoration，56 條都要有說明書的說明。少一條就是轉錄漏了，
// 而畫面上看起來只是「這個法術沒寫」。
func TestEverySpellCarriesTheManualDescription(t *testing.T) {
	catalogue, err := gamepack.TraditionalChineseSpells()
	if err != nil {
		t.Fatal(err)
	}
	missing := 0
	for _, spell := range catalogue.Spells() {
		if spell.Effect != "" {
			continue
		}
		if spell.Name != "Restoration" {
			t.Fatalf("%q has no manual description", spell.Name)
		}
		missing++
	}
	if missing != 1 {
		t.Fatalf("%d spells lack a description; only Restoration should", missing)
	}
}

// 說明書自己的〔原書如此〕註記要跟著條目走：比對法術名時要以遊戲的拼法
// 為準，說明書那六個拼錯的字不能反過來當標準。
func TestManualTyposAreRecordedAgainstTheGameSpelling(t *testing.T) {
	catalogue, err := gamepack.TraditionalChineseSpells()
	if err != nil {
		t.Fatal(err)
	}
	notes := map[string]bool{}
	for _, spell := range catalogue.Spells() {
		if spell.ManualNote != "" {
			notes[spell.Name] = true
		}
	}
	for _, name := range []string{
		"Spiritual Hammer", "Ray of Enfeeblement", "Bestow Curse",
		"Lightning Bolt", "Stinking Cloud", "Protection From Normal Missiles",
	} {
		if !notes[name] {
			t.Fatalf("%q lost the manual's original-spelling note", name)
		}
	}
}
