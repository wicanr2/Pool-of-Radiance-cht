package main

import (
	"math/rand"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// newNormalSlumWanderingEncounter 從標題與 B)EGIN ADVENTURING 走完羅夫導覽，
// 再由 (0,4) 朝西進貧民窟。它只從 Update() 送玩家按鍵；沒有直接切地圖、座標、
// ECL entry、旗標或遭遇結果（spec 136）。
func newNormalSlumWanderingEncounter(t *testing.T, eclSeed, diceSeed int64) *app {
	t.Helper()
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	// ECL RANDOM 與 SURPRISE 是兩條獨立骰流；測試把兩者都明示固定，
	// 才能重播同一條玩家路徑與同一種選單分支（spec 136）。
	application.roller = diceRoller{random: rand.New(rand.NewSource(diceSeed))}
	application.eclSeed = eclSeed
	party := make([]poolsave.Character, 0, 6)
	for index := 0; index < 6; index++ {
		party = append(party, poolsave.Character{Name: string(rune('A' + index)),
			RaceID: "dwarf", GenderID: "male", ClassID: "fighter", AlignmentID: "lawful-good",
			Abilities: [6]int{16, 10, 10, 13, 10, 10}, MaxHP: 8, CurrentHP: 8,
			PortraitHead: 1, PortraitBody: 1, IconSize: 1})
	}
	application.state = poolsave.State{Schema: poolsave.Schema,
		CharacterLibrary: party, Party: party}
	application.saveState = func(poolsave.State) error { return nil }

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
	if !application.introDone {
		t.Fatal("羅夫導覽沒有結束")
	}
	if application.spawn.X != 0 || application.spawn.Y != 4 || application.spawn.Facing != 3 {
		t.Fatalf("導覽落點是 %+v，不是 (0,4) 朝西", application.spawn)
	}

	for step := 0; step < 32 && application.encounter == nil; step++ {
		key := ebiten.KeyArrowUp
		if application.cellEventPending || application.cellWaitingMenu {
			key = ebiten.KeyEnter
		}
		if err := press(application, key); err != nil {
			t.Fatalf("進貧民窟第 %d 個輸入：%v", step, err)
		}
	}
	if application.encounter == nil {
		t.Fatalf("固定 seed 從 (0,4) 往西沒有排出走路遭遇；最後在 %+v：%q",
			application.spawn, application.eventText)
	}
	want := []string{"COMBAT", "WAIT", "FLEE", "PARLAY"}
	if strings.Join(application.cellMenuOptions, "|") != strings.Join(want, "|") {
		t.Fatalf("走路遭遇選單是 %v，不是 %v", application.cellMenuOptions, want)
	}
	if application.eclArchive != 2 || application.eventSession.CurrentBlockID() != 20 {
		t.Fatalf("遭遇不在貧民窟 ECL2/20：archive=%d block=%d",
			application.eclArchive, application.eventSession.CurrentBlockID())
	}
	return application
}

func TestNormalKeysReachAllSlumWanderingEncounterChoices(t *testing.T) {
	t.Run("COMBAT", func(t *testing.T) {
		application := newNormalSlumWanderingEncounter(t, 1, 1)
		resultAddress := application.encounter.resultAddress
		if err := selectMenuOption(t, application, "COMBAT"); err != nil {
			t.Fatal(err)
		}
		for guard := 0; guard < 8 && !application.combatActive; guard++ {
			if !application.cellEventPending && !application.cellWaitingMenu {
				break
			}
			if err := press(application, ebiten.KeyEnter); err != nil {
				t.Fatal(err)
			}
		}
		if !application.combatActive || application.encounter != nil {
			t.Fatalf("COMBAT 後 combat=%v encounter=%v pending=%v menu=%v result@%04X=%d archive=%d block=%d pc=%04X text=%q",
				application.combatActive, application.encounter != nil,
				application.cellEventPending, application.cellWaitingMenu,
				resultAddress, application.eventMachine.Memory[resultAddress],
				application.eclArchive, application.eventSession.CurrentBlockID(),
				0x9900+application.eventMachine.PC, application.eventText)
		}
		if got := application.eventMachine.Memory[resultAddress]; got != 1 {
			t.Fatalf("COMBAT 寫回 @%04X=%d，不是 1", resultAddress, got)
		}
		t.Logf("COMBAT：@%04X=1，建立 %d 群怪物", resultAddress, len(application.combatMonsters))
	})

	t.Run("WAIT", func(t *testing.T) {
		application := newNormalSlumWanderingEncounter(t, 1, 2)
		resultAddress := application.encounter.resultAddress
		if err := selectMenuOption(t, application, "WAIT"); err != nil {
			t.Fatal(err)
		}
		if application.encounter != nil || application.combatActive ||
			!strings.Contains(application.eventText, gamepack.EncounterMessageFlee) {
			t.Fatalf("WAIT 沒有走到本次突襲表的怪物撤退出口：encounter=%v combat=%v text=%q",
				application.encounter != nil, application.combatActive, application.eventText)
		}
		if got := application.eventMachine.Memory[resultAddress]; got != 0 {
			t.Fatalf("WAIT 寫回 @%04X=%d，不是這張表的 0", resultAddress, got)
		}
		t.Logf("WAIT：@%04X=0，%q", resultAddress, application.eventText)
	})

	t.Run("FLEE", func(t *testing.T) {
		application := newNormalSlumWanderingEncounter(t, 1, 1)
		resultAddress := application.encounter.resultAddress
		if err := selectMenuOption(t, application, "FLEE"); err != nil {
			t.Fatal(err)
		}
		if application.encounter != nil || application.combatActive {
			t.Fatalf("FLEE 後 encounter=%v combat=%v text=%q",
				application.encounter != nil, application.combatActive, application.eventText)
		}
		safe := map[[2]uint8]bool{{15, 4}: true, {14, 6}: true, {11, 5}: true, {11, 2}: true, {9, 2}: true}
		if !safe[[2]uint8{application.spawn.X, application.spawn.Y}] {
			t.Fatalf("FLEE 落在 %+v，不是 spec 136 的五個安全點之一", application.spawn)
		}
		if got := application.eventMachine.Memory[resultAddress]; got != 2 {
			t.Fatalf("FLEE 寫回 @%04X=%d，不是 2", resultAddress, got)
		}
		t.Logf("FLEE：@%04X=2，落點 (%d,%d)", resultAddress, application.spawn.X, application.spawn.Y)
	})

	t.Run("PARLAY", func(t *testing.T) {
		application := newNormalSlumWanderingEncounter(t, 1, 2)
		resultAddress := application.encounter.resultAddress
		for attempt := 0; attempt < 4 && application.encounter != nil; attempt++ {
			if err := selectMenuOption(t, application, "PARLAY"); err != nil {
				t.Fatal(err)
			}
		}
		if application.encounter != nil || application.combatActive ||
			!strings.Contains(application.eventText, "CONVERSING WITH") {
			t.Fatalf("PARLAY 沒有進交談：encounter=%v combat=%v text=%q",
				application.encounter != nil, application.combatActive, application.eventText)
		}
		if got := application.eventMachine.Memory[resultAddress]; got != 3 {
			t.Fatalf("PARLAY 寫回 @%04X=%d，不是 3", resultAddress, got)
		}
		t.Logf("PARLAY：@%04X=3，%q", resultAddress, application.eventText)
	})
}
