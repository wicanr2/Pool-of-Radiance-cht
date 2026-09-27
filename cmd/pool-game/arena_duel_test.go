package main

// 競技場決鬥（ECL `CALL 8000h`，spec 150〈競技場〉，issue #111）：從 `ecl3/11 9CA5h` 的
// `CLEARMONSTERS → CALL 8000h → COMBAT` 開始跑，戰鬥與頁面都從 `Update()` 送鍵。

import (
	"slices"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/creation"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// arenaApp 停在競技場 `WHO WILL DUEL?` 之後、`COMPARE @6BB8` 通過的那一段，
// 目前角色是第二個人，身上帶兩件東西與一個效果。
func arenaApp(t *testing.T) *app {
	t.Helper()
	application := scriptApp(t, 3, 11, 0x9CA5)
	application.currentCharacter = 1
	champion := &application.state.Party[1]
	champion.Inventory = []poolsave.Item{
		{Name: "FIRST", Raw: make([]byte, gamepack.MonsterItemRecordSize)},
		{Name: "SECOND", Raw: make([]byte, gamepack.MonsterItemRecordSize)},
	}
	champion.Effects = poolsave.PermanentEffects(0x01)
	champion.IconHead, champion.IconWeapon = 3, 7
	return application
}

func TestArenaDuelCopiesTheCurrentCharacterAsRolf(t *testing.T) {
	application := arenaApp(t)
	if err := application.continueInitialSearch(nil); err != nil {
		t.Fatal(err)
	}
	if !application.duel || !application.arenaCopy || !application.combatActive {
		t.Fatalf("duel=%v copy=%v combat=%v after CALL 8000h", application.duel,
			application.arenaCopy, application.combatActive)
	}
	if got := application.eventMachine.Memory[duelArenaAddress]; got != 1 {
		t.Fatalf("@6DE6 = %d, want 1 (1ABEh)", got)
	}
	if len(application.state.Party) != 5 {
		t.Fatalf("party size %d, want the copy at the tail", len(application.state.Party))
	}
	copied := application.state.Party[4]
	champion := application.state.Party[1]
	if copied.Name != "ROLF" || copied.Side != 1 || !copied.Quick || len(copied.Effects) != 0 {
		t.Fatalf("copy %q side=%d quick=%v effects=%v", copied.Name, copied.Side, copied.Quick, copied.Effects)
	}
	if copied.CurrentHP != champion.CurrentHP || copied.ClassID != champion.ClassID ||
		copied.Abilities != champion.Abilities {
		t.Fatalf("copy %+v does not carry the champion's record", copied)
	}
	names := []string{}
	for _, item := range copied.Inventory {
		names = append(names, item.Name)
	}
	if !slices.Equal(names, []string{"SECOND", "FIRST"}) {
		t.Fatalf("copied items %v, want the reversed chain (1C81h)", names)
	}
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	state := application.tactical
	if state == nil {
		t.Fatal("no tactical board")
	}
	// 不上場的隊員也在串列上、用掉一格樣板，只是體型 0（overlay-10 `1D50h`，spec 061）。
	deployed, foes, standing := []int{}, []int{}, 0
	for index := 1; index < len(state.PartySlot); index++ {
		slot := state.PartySlot[index]
		if state.Roster[index].FootprintClass == 0 {
			continue
		}
		standing++
		switch {
		case slot >= 0 && state.Friendly[index]:
			deployed = append(deployed, slot)
		case slot >= 0:
			foes = append(foes, slot)
			if !state.AIDriven[index] {
				t.Fatal("the copy is not driven by the AI (+10Fh = 1)")
			}
			if state.Icons[index].Head != 3 || state.Icons[index].Body != 7 {
				t.Fatalf("copy icon %+v, want the champion's", state.Icons[index])
			}
			if state.Morale.Raw[index] != 0xB2 {
				t.Fatalf("copy morale %02X, want B2h (1BC6h)", state.Morale.Raw[index])
			}
		}
	}
	if !slices.Equal(deployed, []int{1}) || !slices.Equal(foes, []int{4}) || standing != 2 {
		t.Fatalf("board: party %v, opposing %v, %d standing", deployed, foes, standing)
	}
	winByKeys(t, application)
	report := application.postCombat
	if report == nil || report.title() != msgPostCombatDuelWon {
		t.Fatalf("report %+v, want the won duel", report)
	}
	if len(application.state.Party) != 4 || application.arenaCopy {
		t.Fatalf("copy still in the party (%d members)", len(application.state.Party))
	}
	// entry 2 `0006h`：最高職業等級 × 100，不除人數。
	if report.share != 100 {
		t.Fatalf("duelist share %d, want level 1 × 100", report.share)
	}
	code, _ := creation.ClassDOSCode(champion.ClassID)
	for index, member := range application.state.Party {
		want := uint32(0)
		if index == 1 {
			want = gamepack.ExperienceShare(100, code, champion.Abilities)
		}
		if member.Experience != want {
			t.Fatalf("member %d gained %d XP, want %d", index, member.Experience, want)
		}
	}
	texts := drawnPostCombatTexts(application)
	if !slices.Contains(texts, "You have won the duel.") || !slices.Contains(texts, "The duelist receives 100") {
		t.Fatalf("duel page draws %q", texts)
	}
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if err := selectMenuOption(t, application, "Exit"); err != nil {
		t.Fatal(err)
	}
	if application.treasureActive || application.duel || application.eventMachine.Memory[duelArenaAddress] != 0 {
		t.Fatalf("after the menu: treasure=%v duel=%v @6DE6=%d", application.treasureActive,
			application.duel, application.eventMachine.Memory[duelArenaAddress])
	}
}

// 兩邊都交給 AI 打到收場：複製品真的會出手，而不論誰贏，打完隊伍裡都沒有它。
func TestArenaDuelCopyFightsBack(t *testing.T) {
	application := arenaApp(t)
	application.state.Party[1].Quick = true
	if err := application.continueInitialSearch(nil); err != nil {
		t.Fatal(err)
	}
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	championHP := application.state.Party[1].CurrentHP
	foeSwings := 0
	for frame := 0; frame < 20000 && application.tactical != nil; frame++ {
		state := application.tactical
		foeSwings = state.Activity.FoeAttacks
		if state.Prompt {
			if err := press(application, ebiten.KeyN); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := press(application, ebiten.KeyEnter); err != nil {
			t.Fatal(err)
		}
	}
	if application.tactical != nil {
		t.Fatal("the duel never ended")
	}
	if foeSwings == 0 {
		t.Fatal("the copy never swung")
	}
	if len(application.state.Party) != 4 || application.arenaCopy {
		t.Fatalf("copy still in the party (%d members)", len(application.state.Party))
	}
	report := application.postCombat
	if report == nil || !report.duel {
		t.Fatalf("report %+v, want a duel page", report)
	}
	if report.title() == msgPostCombatDuelLost && application.state.Party[1].CurrentHP == championHP {
		t.Fatal("the champion lost without taking a hit")
	}
	if application.gameOver {
		t.Fatal("a duel ended the game")
	}
	t.Logf("duel ended: %v, copy swung %d time(s)", report.title() == msgPostCombatDuelWon, foeSwings)
}
