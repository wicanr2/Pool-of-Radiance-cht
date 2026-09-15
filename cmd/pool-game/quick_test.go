package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// Q）UICK（spec 139，issue #27）：按 Q 的人這一回合立刻由 AI 走，旗標留在角色身上；
// 頂層的 Q 不再是東南方向——方向要先按 M。
func TestQuickHandsTheTurnToTheAI(t *testing.T) {
	state := newFoeTurnState(6, 5, 4, 5, 6)
	state.HitPoints = []int{0, 40, 30}
	state.THAC0 = []uint8{0, 40, 40}
	state.ArmorClass = []int{0, 50, 50}
	state.setSingleAttackForm(1, combat.DamageDice{Count: 1, Sides: 8})
	state.setSingleAttackForm(2, combat.DamageDice{Count: 1, Sides: 8})
	state.PartySlot = []int{-1, 0, -1}
	state.AIDriven = aiDriven(state.PartySlot)
	state.FoeTargets = make([]uint8, 3)
	state.Mover = 1
	application := &app{mode: modeAdventure, tacticalPreview: true, roller: fixedRoller{20},
		tactical: state, language: languageEnglish}
	application.state.Party = []poolsave.Character{{Name: "A"}}
	if err := press(application, ebiten.KeyQ); err != nil {
		t.Fatal(err)
	}
	if !application.state.Party[0].Quick {
		t.Fatal("Q did not set the character's quick flag (+10Fh)")
	}
	if !state.AIDriven[1] {
		t.Fatal("Q did not hand the roster slot to the AI")
	}
	// AI 走了這一回合：站在敵人旁邊，必中（d20 固定 20）就打到了。
	if state.HitPoints[2] >= 30 {
		t.Fatalf("the AI did not act for the quick character: foe hp %d, status %q", state.HitPoints[2], state.Status)
	}
	// SPACE 收回（要在 AI 接手下一回合之前生效）。
	state.Mover, state.Prompt = 1, false
	if err := press(application, ebiten.KeySpace); err != nil {
		t.Fatal(err)
	}
	if application.state.Party[0].Quick || state.AIDriven[1] {
		t.Fatal("SPACE did not take the character back")
	}
}

// 方向鍵要先按 M：頂層的 Q 不會把隊員往東南搬。
func TestDirectionKeysNeedMoveFirst(t *testing.T) {
	state := newFoeTurnState(10, 5, 4, 5, 6)
	state.PartySlot = []int{-1, 0, -1}
	state.AIDriven = aiDriven(state.PartySlot)
	state.FoeTargets = make([]uint8, 3)
	state.Mover = 1
	application := &app{mode: modeAdventure, tacticalPreview: true, roller: fixedRoller{1},
		tactical: state, language: languageEnglish}
	application.state.Party = []poolsave.Character{{Name: "A", NPC: true}}
	before := state.Roster[1]
	if err := press(application, ebiten.KeyH); err != nil {
		t.Fatal(err)
	}
	if state.Roster[1] != before {
		t.Fatal("a direction key moved the character without M first")
	}
	if err := press(application, ebiten.KeyM); err != nil {
		t.Fatal(err)
	}
	if !state.Moving {
		t.Fatal("M did not enter move mode")
	}
	if err := press(application, ebiten.KeyH); err != nil {
		t.Fatal(err)
	}
	if state.Roster[1].Y != before.Y-1 {
		t.Fatalf("H after M should step north: %+v → %+v", before, state.Roster[1])
	}
}

// 收據：一場自然戰鬥全隊按 Q 交給電腦、之後只按 ENTER，打完經驗值有發。
func TestQuickCombatEarnsExperience(t *testing.T) {
	party := make([]poolsave.Character, 0, 4)
	for index := 0; index < 4; index++ {
		party = append(party, poolsave.Character{Name: string(rune('A' + index)), RaceID: "dwarf",
			GenderID: "male", ClassID: "fighter", AlignmentID: "lawful-good",
			Abilities: [6]int{18, 10, 10, 16, 10, 10}, MaxHP: 40, CurrentHP: 40,
			PortraitHead: 1, PortraitBody: 1, IconSize: 1})
	}
	application := newGameAtFirstCombat(t, party)
	before := make([]uint32, len(application.state.Party))
	for index, member := range application.state.Party {
		before[index] = member.Experience
	}
	for tick := 0; tick < 40000; tick++ {
		state := application.tactical
		if state == nil || state.Finished {
			break
		}
		key := ebiten.KeyEnter
		switch {
		case state.Prompt && state.sideCounts().Foes == 0:
			key = ebiten.KeyN
		case state.Prompt:
			key = ebiten.KeyY
		case state.Mover != 0 && int(state.Mover) < len(state.PartySlot) &&
			state.PartySlot[state.Mover] >= 0 && !state.aiDrives(int(state.Mover)):
			key = ebiten.KeyQ
		}
		if err := press(application, key); err != nil {
			t.Fatalf("combat tick %d: %v", tick, err)
		}
	}
	if application.tactical != nil {
		t.Fatalf("quick combat never ended: round %d, status %q", application.tactical.Round, application.tactical.Status)
	}
	if application.gameOver {
		t.Fatalf("the quick party was destroyed: %q", application.eventText)
	}
	gained := false
	for index, member := range application.state.Party {
		if member.Experience > before[index] {
			gained = true
		}
		if !member.Quick {
			t.Fatalf("%s lost the quick flag after the fight; the original keeps +10Fh until SPACE", member.Name)
		}
	}
	if !gained {
		t.Fatal("a fight won on QUICK awarded no experience")
	}
}
