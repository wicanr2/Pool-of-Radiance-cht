package main

import (
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 只用按鍵在法術畫面上把法術記給一個牧師，記滿了就記不進去，忘掉一格又記得進去。
func TestMemoriseFromTheSpellScreen(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	// 第 1 級的牧師：建角寫下第一級一格，沒有睿智加成（spec 072）。
	cleric := poolsave.Character{Name: "A", RaceID: "human", GenderID: "male",
		ClassID: "cleric", AlignmentID: "lawful-good",
		Abilities: [6]int{10, 10, 18, 10, 10, 10}, MaxHP: 8, CurrentHP: 8,
		PortraitHead: 1, PortraitBody: 1, IconSize: 1}
	application.state = poolsave.State{Schema: poolsave.Schema,
		CharacterLibrary: []poolsave.Character{cleric}, Party: []poolsave.Character{cleric}}
	if err := application.openSpells(); err != nil {
		t.Fatal(err)
	}
	application.mode = modeMenu
	maxima, _, ok := application.spellMemberSlots(0)
	if !ok {
		t.Fatal("算不出可記憶數")
	}
	// 第 1 級牧師是 1/0/0：睿智 18 也拿不到加成，加成在「等級大於 1」的分支裡。
	if maxima[gamepack.SpellSlotGroupCleric] != [gamepack.SpellSlotLevels]int{1, 0, 0} {
		t.Fatalf("第 1 級牧師的可記憶數應該是 1/0/0，算出 %v",
			maxima[gamepack.SpellSlotGroupCleric])
	}
	// 游標停在牧師第 1 級的第一條，按 M 記下去。
	press(application, ebiten.KeyDigit1)
	press(application, ebiten.KeyM)
	member := application.state.Party[0]
	memorised := 0
	for _, value := range member.Memorised {
		if value != 0 {
			memorised++
		}
	}
	if memorised != 1 {
		t.Fatalf("應該記下一個，記了 %d 個（狀態列 %q）", memorised, application.statusLine)
	}
	// 只有一格，第二個記不進去。
	press(application, ebiten.KeyM)
	memorised = 0
	for _, value := range application.state.Party[0].Memorised {
		if value != 0 {
			memorised++
		}
	}
	if memorised != 1 {
		t.Fatalf("只有一格，卻記了 %d 個", memorised)
	}
	// 忘掉之後又記得進去。
	press(application, ebiten.KeyF)
	for _, value := range application.state.Party[0].Memorised {
		if value != 0 {
			t.Fatalf("按 F 之後不該還記著 %d", value)
		}
	}
	press(application, ebiten.KeyM)
	memorised = 0
	for _, value := range application.state.Party[0].Memorised {
		if value != 0 {
			memorised++
		}
	}
	if memorised != 1 {
		t.Fatalf("忘掉之後應該記得回去，記了 %d 個", memorised)
	}
	// 角色庫要跟著同步。
	if len(application.state.CharacterLibrary[0].Memorised) == 0 {
		t.Error("角色庫沒有跟著更新")
	}
}
