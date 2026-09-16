package main

// dosgolem 收據（docs/audit/dosgolem-deployment-peek-orc-home.json）：五人隊撬開 (3,4) 北面
// 的門踏進獸人的家（ecl2/20 地形 9，(3,3) 面向北），`LOAD MONSTER 15 1 5／14 3 5／4 20 4`
// 二十四隻，距離 0。原版 `5E85h`：隊員五格、獸人放上二十隻（四隻放不下被摘掉），
// `45B2h..45BAh = 00 00 00 00 03 0C 00 02 01`。同狀態的 remake 要逐格相同。

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/golden-box-remake-engine/eclvm"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

func TestDeploymentMatchesTheOrcHomeReceipt(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	archive, ok := application.eclCatalog.Archive(2)
	if !ok {
		t.Fatal("ECL2 archive is absent")
	}
	session, err := gamepack.NewDOSECLArchiveSession(archive, 20, 0x9914)
	if err != nil {
		t.Fatal(err)
	}
	geoMap, ok := application.geometryCatalog.Map(gamepack.MapKey{Archive: 2, BlockID: 20})
	if !ok {
		t.Fatal("GEO2/20 is absent")
	}
	application.eventSession, application.eventMachine = session, session.Machine()
	application.eclArchive = 2
	application.initialMap = &geoMap
	application.spawn = gamepack.Spawn{Map: geoMap.Key, X: 3, Y: 3, Facing: 0}
	application.introDone, application.mode = true, modeAdventure
	if err := application.configureEventSession(session); err != nil {
		t.Fatal(err)
	}
	application.saveState = func(poolsave.State) error { return nil }
	application.eventMachine.Memory[encounterWalkFlagAddress] = 1
	application.eventMachine.Memory[encounterDistanceAddress] = 0

	party := make([]poolsave.Character, 0, 5)
	for index, hp := range []int{9, 6, 12, 6, 9} {
		party = append(party, poolsave.Character{Name: string(rune('B' + index)), RaceID: "dwarf",
			GenderID: "male", ClassID: "fighter", AlignmentID: "lawful-good",
			Abilities: [6]int{15, 10, 10, 13, 10, 10}, MaxHP: hp, CurrentHP: hp,
			PortraitHead: 1, PortraitBody: 1, IconSize: 1})
	}
	application.state = poolsave.State{Schema: poolsave.Schema, CharacterLibrary: party, Party: party}
	spawns := []eclvm.MonsterSpawn{{MonsterID: 15, Count: 1, IconBlock: 5}, {MonsterID: 14, Count: 3, IconBlock: 5}, {MonsterID: 4, Count: 20, IconBlock: 4}}
	if err := application.enterCombatStaging(spawns); err != nil {
		t.Fatal(err)
	}
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	state := application.tactical
	if state == nil {
		t.Fatalf("no tactical state: %q", application.eventText)
	}
	want := [][2]uint8{{27, 13}, {28, 13}, {26, 13}, {28, 14}, {29, 14},
		{27, 12}, {26, 12}, {28, 12}, {25, 12}, {24, 12}, {23, 12}, {26, 11}, {25, 11}, {27, 11}, {24, 11},
		{23, 11}, {22, 11}, {22, 12}, {21, 11}, {21, 12}, {20, 11}, {20, 12}, {19, 11}, {19, 12}, {18, 11}}
	if len(state.Roster) != len(want)+1 {
		t.Errorf("roster has %d entries, receipt has %d (original dropped 4 of 24 orcs)", len(state.Roster)-1, len(want))
	}
	for index, cell := range want {
		if index+1 >= len(state.Roster) {
			break
		}
		got := state.Roster[index+1]
		if got.X != cell[0] || got.Y != cell[1] {
			t.Errorf("combatant %d at (%d,%d), original 5E85h says (%d,%d)", index+1, got.X, got.Y, cell[0], cell[1])
		}
	}
	snapshot := func(round int) string {
		line := fmt.Sprintf("round %d:", round)
		for index := 1; index < len(state.Roster); index++ {
			cell := state.Roster[index]
			line += fmt.Sprintf(" %d:(%d,%d,%d)", index, cell.X, cell.Y, cell.FootprintClass)
		}
		return line
	}
	t.Log(snapshot(state.Round))
	// 隊員一律結束回合（原版那邊是 GUARD），看獸人怎麼走；原版兩批 d,g 就全滅了。
	seen := state.Round
	for tick := 0; tick < 6000 && application.tactical == state && !state.Finished && seen < 4; tick++ {
		key := ebiten.KeyEnter
		if state.Prompt {
			key = ebiten.KeyY
		}
		if err := press(application, key); err != nil {
			t.Fatal(err)
		}
		if state.Round != seen {
			seen = state.Round
			t.Log(snapshot(seen))
		}
	}
}
