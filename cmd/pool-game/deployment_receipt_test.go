package main

// dosgolem 收據（docs/audit/dosgolem-deployment-peek-goblins.json）：五人隊（預設擲骰的
// 矮人戰士）在貧民窟 (15,5) 面向東突襲四隻哥布林，距離 0。原版開打那一刻的位置表
// `5E85h` 是隊員 (26,12)(27,13)(25,11)(24,12)(25,13)、哥布林 (28,13)(27,12)(29,13)(28,12)；
// 之後每個隊員 DONE→GUARD，哥布林自己走了三輪。這裡把同一個狀態（同格、同朝向、
// 同人數、距離 0）直接擺出來——這是診斷，不是玩家路徑——部署要逐格相同；
// 哥布林走位印成同一份表對照（敵方 AI 仍是暫定的，那一段只印不斷言）。

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/golden-box-remake-engine/eclvm"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

func TestDeploymentMatchesTheGoblinReceipt(t *testing.T) {
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
	application.spawn = gamepack.Spawn{Map: geoMap.Key, X: 15, Y: 5, Facing: 1}
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

	goblin := uint8(0xFF)
	for id := uint8(0); id < 60; id++ {
		record, err := application.loadMonster(2, id)
		if err == nil && strings.HasPrefix(record.Name, "GOBLIN") {
			goblin = id
			break
		}
	}
	if goblin == 0xFF {
		t.Fatal("ECL2 has no GOBLIN monster record")
	}
	if err := application.enterCombatStaging([]eclvm.MonsterSpawn{{MonsterID: goblin, Count: 4, IconBlock: 4}}); err != nil {
		t.Fatal(err)
	}
	if !application.combatActive {
		t.Fatal("staging did not arm combat")
	}
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	state := application.tactical
	if state == nil {
		t.Fatalf("no tactical state: %q", application.eventText)
	}
	want := [][2]uint8{{26, 12}, {27, 13}, {25, 11}, {24, 12}, {25, 13}, {28, 13}, {27, 12}, {29, 13}, {28, 12}}
	if len(state.Roster) != len(want)+1 {
		t.Fatalf("roster has %d entries, receipt has %d", len(state.Roster)-1, len(want))
	}
	for index, cell := range want {
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
	// 隊員一律結束回合（原版那邊是 GUARD，remake 沒有守衛攻擊），看哥布林怎麼走。
	seen := state.Round
	for tick := 0; tick < 4000 && application.tactical == state && !state.Finished && seen < 4; tick++ {
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
