package main

// 跟著隊伍打的怪物（spec 061〈跟著隊伍的非隊員〉，#83）：怪物記錄 `+10Eh` 是 0 的只有 MON4 block 70
// 的 EFREETI，瓦海登墳場（ECL4 block 10 `B161h`）與吸血鬼、殭屍一起載進來。原版部署 `1CEEh` 逐筆
// 拿 `+10Eh` 挑樣板，所以它站在隊伍那一邊、由 AI 走；戰後 overlay-05 entry 2 只算 `+10Eh == 1` 的
// 經驗值；B）ANDAGE 只包 runtime `+13h == 0`（隊員）的。

import (
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/golden-box-remake-engine/eclvm"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// valhingenFixture 是 ECL4 block 10 的盤面：五名戰士、吸血鬼（23）、三隻 106、EFREETI（70），
// 從 Update() 按 ENTER 開打。
func valhingenFixture(t *testing.T) *app {
	t.Helper()
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	archive, ok := application.eclCatalog.Archive(4)
	if !ok {
		t.Fatal("ECL4 archive is absent")
	}
	session, err := gamepack.NewDOSECLArchiveSession(archive, 10, 0x9914)
	if err != nil {
		t.Fatal(err)
	}
	geoMap, ok := application.geometryCatalog.Map(gamepack.MapKey{Archive: 4, BlockID: 10})
	if !ok {
		t.Fatal("GEO4/10 is absent")
	}
	application.eventSession, application.eventMachine = session, session.Machine()
	application.eclArchive = 4
	application.initialMap = &geoMap
	application.spawn = gamepack.Spawn{Map: geoMap.Key, X: 7, Y: 7, Facing: 0}
	application.introDone, application.mode = true, modeAdventure
	if err := application.configureEventSession(session); err != nil {
		t.Fatal(err)
	}
	application.saveState = func(poolsave.State) error { return nil }
	application.eventMachine.Memory[encounterWalkFlagAddress] = 1
	application.eventMachine.Memory[encounterDistanceAddress] = 0
	party := make([]poolsave.Character, 0, 5)
	for index := 0; index < 5; index++ {
		party = append(party, poolsave.Character{Name: string(rune('B' + index)), RaceID: "dwarf",
			GenderID: "male", ClassID: "fighter", AlignmentID: "lawful-good",
			Abilities: [6]int{15, 10, 10, 13, 10, 10}, MaxHP: 9, CurrentHP: 9,
			PortraitHead: 1, PortraitBody: 1, IconSize: 1})
	}
	application.state = poolsave.State{Schema: poolsave.Schema, CharacterLibrary: party, Party: party}
	spawns := []eclvm.MonsterSpawn{
		{MonsterID: 23, Count: 1, IconBlock: 1},
		{MonsterID: 106, Count: 3, IconBlock: 2},
		{MonsterID: 70, Count: 1, IconBlock: 3},
	}
	if err := application.enterCombatStaging(spawns); err != nil {
		t.Fatal(err)
	}
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if application.tactical == nil {
		t.Fatal("no tactical state")
	}
	return application
}

func TestEfreetiFightsOnThePartySide(t *testing.T) {
	application := valhingenFixture(t)
	state := application.tactical
	var names []string
	efreeti := -1
	for index := 1; index < len(state.Roster); index++ {
		if state.PartySlot[index] >= 0 {
			if !state.Friendly[index] {
				t.Fatalf("party member %d deployed on the far side", index)
			}
			continue
		}
		monster, ok := application.stagedMonsterFor(index, state.PartySlot, state.Friendly)
		if !ok {
			t.Fatalf("board cell %d maps to no staged monster", index)
		}
		names = append(names, monster.Record.Name)
		if monster.Record.Name == "EFREETI" {
			efreeti = index
			continue
		}
		if state.Friendly[index] {
			t.Fatalf("%s (%d) stands with the party", monster.Record.Name, index)
		}
	}
	// 怪物照 LOAD MONSTER 的順序、一筆對一格：跟著隊伍的那一隻不會把後面的對應推歪。
	if len(names) != 5 || names[0] != "VAMPIRE" || names[4] != "EFREETI" {
		t.Fatalf("monster cells map to %v", names)
	}
	if !state.Friendly[efreeti] || !state.aiDrives(efreeti) {
		t.Fatalf("EFREETI friendly %v ai %v; want the party side under AI", state.Friendly[efreeti], state.aiDrives(efreeti))
	}
	if counts := state.sideCounts(); counts.Party != 6 || counts.Foes != 4 {
		t.Fatalf("side counts %+v, want 6 with the party and 4 against", counts)
	}
	// 它追的是對面：挑目標的名單裡沒有隊員。
	candidates, err := state.foeTargetCandidates(uint8(efreeti), 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, candidate := range candidates {
		if state.Friendly[candidate] {
			t.Fatalf("EFREETI's target list has %d on its own side", candidate)
		}
	}

	// 倒地了也不包紮（`100Fh` 的 `+13h`）。
	state.States[efreeti] = combat.DyingState
	if index, ok := state.bandageTarget(); ok && index == efreeti {
		t.Fatal("B)ANDAGE picked the EFREETI")
	}

	// 經驗值只算對面：吸血鬼加三隻 106，EFREETI 不算。
	want := uint32(0)
	for _, monster := range application.combatMonsters {
		if monster.Record.Name == "EFREETI" {
			continue
		}
		want += monster.Record.ExperienceValue(int(monster.Record.MaxHitPoints())) * uint32(monster.Spawn.Count)
	}
	eligible := []bool{true, true, true, true, true}
	if share := application.awardCombatExperienceWithLoot(nil, 0, eligible, 0); share != want/5 {
		t.Fatalf("each share is %d, want %d (the EFREETI is not worth anything)", share, want/5)
	}
}
