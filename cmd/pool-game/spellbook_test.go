package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 法術書的閘門（spec 110）：書上沒有的記不起來。
//
// 少了這一條，一級法師記得起火球術——而畫面上那看起來像「法術系統做好了」，
// 不像少了一道閘門。
func TestAFirstLevelMageCannotMemoriseSpellsOutsideTheSpellbook(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	member := poolsave.Character{Name: "TARRY", RaceID: "human", GenderID: "female",
		ClassID: "magic-user", AlignmentID: "lawful-good",
		Abilities: [6]int{10, 10, 10, 18, 10, 10}, MaxHP: 4, CurrentHP: 4}
	application.state = poolsave.State{Schema: poolsave.Schema,
		CharacterLibrary: []poolsave.Character{member}, Party: []poolsave.Character{member}}
	application.saveState = func(poolsave.State) error { return nil }
	if err := application.openSpells(); err != nil {
		t.Fatal(err)
	}
	application.spellMember = 0
	application.ensureSpellbook(0)
	book := application.state.Party[0].Spellbook
	if len(book) != 4 {
		t.Fatalf("新法師的書上有 %d 條，原版寫死四條：%v", len(book), book)
	}

	// 火球術（47）不在書上，記不起來。
	if got := memoriseByID(t, application, 47); got {
		t.Fatal("一級法師把火球術記起來了")
	}
	if !strings.Contains(application.statusLine, application.text(msgSpellsNotInBook)) {
		t.Errorf("拒絕的理由是 %q，應該說書上沒有", application.statusLine)
	}
	// 催眠術（21）在起手四條裡，記得起來。
	if got := memoriseByID(t, application, 21); !got {
		t.Fatalf("起手就會的催眠術記不起來：%q", application.statusLine)
	}
}

// 昇級給的額度可以拿去學一條新的，學完就記得起來。
func TestTrainingLetsAMageLearnOneMoreSpell(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	member := poolsave.Character{Name: "TARRY", RaceID: "human", GenderID: "female",
		ClassID: "magic-user", AlignmentID: "lawful-good",
		Abilities: [6]int{10, 10, 10, 18, 10, 10}, MaxHP: 4, CurrentHP: 4,
		Spellbook: []uint8{11, 18, 19, 21}, SpellsToLearn: 1}
	application.state = poolsave.State{Schema: poolsave.Schema,
		CharacterLibrary: []poolsave.Character{member}, Party: []poolsave.Character{member}}
	application.saveState = func(poolsave.State) error { return nil }
	if err := application.openSpells(); err != nil {
		t.Fatal(err)
	}
	application.spellMember = 0

	// 第 3 級的火球術學不起來：一級法師沒有第 3 級的格子。
	learnByID(t, application, 47)
	if gamepack.SpellbookKnows(application.state.Party[0].Spellbook, 47) {
		t.Fatal("一級法師學會了火球術")
	}
	// 第 1 級的魔法飛彈（15）學得起來，額度用掉一次。
	learnByID(t, application, 15)
	if !gamepack.SpellbookKnows(application.state.Party[0].Spellbook, 15) {
		t.Fatalf("魔法飛彈沒學起來：%q", application.statusLine)
	}
	if got := application.state.Party[0].SpellsToLearn; got != 0 {
		t.Errorf("學完之後還剩 %d 次額度，應該是 0", got)
	}
	// 額度用完就不能再學。
	learnByID(t, application, 9)
	if gamepack.SpellbookKnows(application.state.Party[0].Spellbook, 9) {
		t.Fatal("額度用完還學得到")
	}
	// 學到的那一條記得起來。
	if got := memoriseByID(t, application, 15); !got {
		t.Fatalf("學會的魔法飛彈記不起來：%q", application.statusLine)
	}
}

// pointSpellCursor 把法術一覽的游標移到某個編號那一條。
func pointSpellCursor(t *testing.T, application *app, id uint8) {
	t.Helper()
	for group := range spellGroups {
		application.spells.group = group
		list := application.spells.current()
		for cursor, spell := range list {
			if uint8(spell.Index+1) == id {
				application.spells.cursor = cursor
				return
			}
		}
	}
	t.Fatalf("法術一覽上找不到編號 %d", id)
}

func memoriseByID(t *testing.T, application *app, id uint8) bool {
	t.Helper()
	pointSpellCursor(t, application, id)
	application.memoriseHighlightedSpell()
	return gamepack.SearchMemorisedSpell(application.state.Party[0].Memorised, id) !=
		gamepack.SpellSearchNotFound
}

func learnByID(t *testing.T, application *app, id uint8) {
	t.Helper()
	pointSpellCursor(t, application, id)
	application.learnHighlightedSpell()
}

// L 這一鍵真的接在法術一覽上，不是只有函式存在。
func TestTheSpellScreenBindsLToLearning(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	member := poolsave.Character{Name: "TARRY", ClassID: "magic-user",
		Abilities: [6]int{10, 10, 10, 18, 10, 10}, MaxHP: 4, CurrentHP: 4,
		Spellbook: []uint8{11, 18, 19, 21}, SpellsToLearn: 1}
	application.state = poolsave.State{Schema: poolsave.Schema, Party: []poolsave.Character{member}}
	application.saveState = func(poolsave.State) error { return nil }
	if err := application.openSpells(); err != nil {
		t.Fatal(err)
	}
	application.spellMember = 0
	pointSpellCursor(t, application, 15)
	application.keys = scriptedKeys{ebiten.KeyL: true}
	application.spellsInput()
	if !gamepack.SpellbookKnows(application.state.Party[0].Spellbook, 15) {
		t.Fatalf("按 L 沒有學會：%q", application.statusLine)
	}
}
