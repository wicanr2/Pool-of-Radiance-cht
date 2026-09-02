package main

import (
	"math/rand"
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

// 只用按鍵走到真的一場戰鬥裡施出魔法飛彈，並且確認：記憶的那一格被用掉、
// 敵人真的掉血、傷害落在 spec 098 算出來的範圍。
//
// 不用戰場預覽——預覽的盤面上沒有敵人，測起來會 skip，而 skip 與通過在
// 報表上分不出來。
func TestCastMagicMissileInCombat(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	// 第 6 級法師：魔法飛彈 3 發，每發 1d4+1（spec 098）。
	levels := make([]uint8, gamepack.ClassThac0ClassCount)
	levels[gamepack.ClassSlotMagicUser] = 6
	party := make([]poolsave.Character, 0, 6)
	for index := 0; index < 6; index++ {
		member := poolsave.Character{Name: string(rune('A' + index)),
			RaceID: "human", GenderID: "male", ClassID: "magic-user",
			AlignmentID: "lawful-good", Abilities: [6]int{10, 18, 10, 10, 10, 10},
			MaxHP: 30, CurrentHP: 30, PortraitHead: 1, PortraitBody: 1, IconSize: 1,
			ClassLevels: append([]uint8(nil), levels...),
			Memorised:   make([]uint8, gamepack.MemorisedSpellSlots)}
		member.Memorised[0] = gamepack.SpellIDMagicMissile
		party = append(party, member)
	}
	application.state = poolsave.State{Schema: poolsave.Schema,
		CharacterLibrary: party, Party: party}
	application.saveState = func(poolsave.State) error { return nil }
	application.roller = diceRoller{random: rand.New(rand.NewSource(3))}
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if err := press(application, ebiten.KeyB); err != nil {
		t.Fatal(err)
	}
	for tick := 0; tick < 20000 && !application.introDone; tick++ {
		if application.introWaiting || application.tourPage >= 0 {
			if err := press(application, ebiten.KeyEnter); err != nil {
				t.Fatal(err)
			}
			continue
		}
		application.keys = scriptedKeys{}
		if err := application.Update(); err != nil {
			t.Fatal(err)
		}
	}
	random := rand.New(rand.NewSource(7))
	for step := 0; step < 4000 && !application.combatActive; step++ {
		var err error
		switch {
		case application.encounter != nil, application.cellWaitingMenu, application.cellEventPending:
			err = press(application, ebiten.KeyEnter)
		default:
			if random.Intn(3) == 0 {
				err = press(application, ebiten.KeyArrowRight)
			} else {
				err = press(application, ebiten.KeyArrowUp)
			}
		}
		if err != nil {
			t.Fatalf("step %d: %v", step, err)
		}
	}
	if !application.combatActive {
		t.Fatal("never reached combat")
	}
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	state := application.tactical
	if state == nil {
		t.Fatalf("tactical state absent: %q", application.statusLine)
	}
	// 輪到隊員動的時候按 C 施法。
	cast := false
	for tick := 0; tick < 4000 && !cast; tick++ {
		state = application.tactical
		if state == nil || state.Finished {
			break
		}
		if state.Prompt {
			if err := press(application, ebiten.KeyY); err != nil {
				t.Fatal(err)
			}
			continue
		}
		mover := state.Mover
		if mover == 0 || int(mover) >= len(state.PartySlot) || state.PartySlot[mover] < 0 {
			if err := press(application, ebiten.KeyEnter); err != nil {
				t.Fatal(err)
			}
			continue
		}
		target, found := state.nearestReachableOpposing(mover)
		if !found {
			if err := press(application, ebiten.KeyEnter); err != nil {
				t.Fatal(err)
			}
			continue
		}
		partyIndex := state.PartySlot[mover]
		before := state.HitPoints[target]
		if err := press(application, ebiten.KeyC); err != nil {
			t.Fatal(err)
		}
		if !application.castOpen {
			t.Fatalf("按 C 沒有開出施法清單（狀態列 %q）", application.statusLine)
		}
		if len(application.castOptions) != 1 ||
			application.castOptions[0].ID != gamepack.SpellIDMagicMissile {
			t.Fatalf("清單應該只有魔法飛彈一條，拿到 %+v", application.castOptions)
		}
		if err := press(application, ebiten.KeyEnter); err != nil {
			t.Fatal(err)
		}
		if application.state.Party[partyIndex].Memorised[0] != 0 {
			t.Fatal("施完之後那一格記憶沒有被用掉")
		}
		after := state.HitPoints[target]
		damage := before - after
		if damage <= 0 {
			t.Fatalf("敵人的血從 %d 變成 %d，法術沒有造成傷害（狀態列 %q）",
				before, after, application.statusLine)
		}
		// 3 發，每發 1d4+1 → 6..15。目標被打倒時血會被夾到 0，所以只查上界。
		if damage > 15 {
			t.Fatalf("第 6 級的魔法飛彈最多 15 點，造成了 %d 點", damage)
		}
		t.Logf("施出魔法飛彈：對 %d 造成 %d 點", target, damage)
		cast = true
	}
	if !cast {
		t.Fatal("整場都沒有機會施法")
	}
}

// 選好的法術要休息過才施得出來：法術畫面記下、紮營休息、才進得了施法清單。
func TestMemorisedSpellsNeedRestBeforeCasting(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	levels := make([]uint8, gamepack.ClassThac0ClassCount)
	levels[gamepack.ClassSlotMagicUser] = 6
	member := poolsave.Character{Name: "A", RaceID: "human", GenderID: "male",
		ClassID: "magic-user", AlignmentID: "lawful-good",
		Abilities: [6]int{10, 18, 10, 10, 10, 10}, MaxHP: 30, CurrentHP: 12,
		PortraitHead: 1, PortraitBody: 1, IconSize: 1,
		ClassLevels: append([]uint8(nil), levels...)}
	application.state = poolsave.State{Schema: poolsave.Schema,
		CharacterLibrary: []poolsave.Character{member}, Party: []poolsave.Character{member}}
	if err := application.openSpells(); err != nil {
		t.Fatal(err)
	}
	// 切到法師第 1 級那一頁，游標停在魔法飛彈上。
	for application.spells.group != 3 {
		press(application, ebiten.KeyTab)
	}
	found := false
	for index, spell := range application.spells.current() {
		if uint8(spell.Index+1) == gamepack.SpellIDMagicMissile {
			application.spells.cursor, found = index, true
			break
		}
	}
	if !found {
		t.Fatal("法師第 1 級那一頁找不到魔法飛彈")
	}
	press(application, ebiten.KeyDigit1)
	press(application, ebiten.KeyM)
	stored := application.state.Party[0].Memorised
	if len(stored) == 0 || stored[0]&0x7f != gamepack.SpellIDMagicMissile {
		t.Fatalf("沒有記下魔法飛彈：%v（狀態列 %q）", stored, application.statusLine)
	}
	if gamepack.MemorisedSpellIsReady(stored[0]) {
		t.Fatal("剛選好就變成可施展了，應該要先休息")
	}
	// 紮營休息：記完，而且整隊回滿。
	application.spellsOpen = false
	application.mode = modeAdventure
	press(application, ebiten.KeyE)
	if !application.campOpen {
		t.Fatalf("按 E 沒有開出紮營選單（狀態列 %q）", application.statusLine)
	}
	press(application, ebiten.KeyEnter) // 游標在「休息」上
	if application.campOpen {
		t.Fatal("休息完應該關掉紮營選單")
	}
	rested := application.state.Party[0]
	if !gamepack.MemorisedSpellIsReady(rested.Memorised[0]) {
		t.Fatalf("休息完應該記好了，還是 %#02x（狀態列 %q）",
			rested.Memorised[0], application.statusLine)
	}
	if rested.CurrentHP != rested.MaxHP {
		t.Errorf("休息完整隊要回滿，還是 %d/%d", rested.CurrentHP, rested.MaxHP)
	}
	if application.state.CharacterLibrary[0].CurrentHP != rested.MaxHP {
		t.Error("角色庫沒有跟著更新")
	}
}
