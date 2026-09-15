package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 圖書館那一段的最小重現（診斷用，直接進圖，不是主線收據）：一級隊伍站在
// 圖書館北緣 (11,0)，由賊撬開閂住的北門、在兩間書架按 L）OOK 搜出五本書，
// 六個槽變 FEh、`4A01` 不動（沒踩地形 3）。**然後出不去**：站上北門格轉北撞門，
// `ecl2/15 99F8h` 召出幽靈（7 HD，2030 XP，一級隊伍打不動也逃不掉）。這一條把
// spec 137「圖書館一級可」那個假設推翻——書拿得到，交不出去。
func TestLibraryBooksSummonTheSpectreOnTheWayOut(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	archive, ok := application.eclCatalog.Archive(2)
	if !ok {
		t.Fatal("ECL2 archive is absent")
	}
	session, err := gamepack.NewDOSECLArchiveSession(archive, 15, 0x9914)
	if err != nil {
		t.Fatal(err)
	}
	geoMap, ok := application.geometryCatalog.Map(gamepack.MapKey{Archive: 2, BlockID: 15})
	if !ok {
		t.Fatal("GEO2/15 is absent")
	}
	application.eventSession, application.eventMachine = session, session.Machine()
	application.eclArchive = 2
	application.initialMap = &geoMap
	application.spawn = gamepack.Spawn{Map: geoMap.Key, X: 11, Y: 0, Facing: 2}
	application.introDone, application.mode = true, modeAdventure
	if err := application.configureEventSession(session); err != nil {
		t.Fatal(err)
	}
	application.saveState = func(poolsave.State) error { return nil }
	party := []poolsave.Character{}
	for index, spec := range manualPartyBuild {
		member := poolsave.Character{Name: string(spec.name), RaceID: "human", GenderID: "male",
			ClassID: spec.classID, AlignmentID: "lawful-good",
			Abilities: [6]int{16, 15, 15, 15, 15, 10}, MaxHP: 8, CurrentHP: 8,
			PortraitHead: 1, PortraitBody: 1, IconSize: 1}
		if index == 4 {
			member.RaceID = "half-elf"
		}
		if err := application.fillThiefSkills(&member); err != nil {
			t.Fatal(err)
		}
		party = append(party, member)
	}
	application.state.Party, application.state.CharacterLibrary = party, party
	driver := &mainlineDriver{t: t, a: application, pilot: &tacticalPilot{},
		step: func(key ebiten.Key) {
			if err := press(application, key); err != nil {
				t.Fatal(err)
			}
		}}
	t.Logf("thief open locks=%d", openLockSkill(party[4]))
	bits := driver.libraryBooks()
	if bits&(1|2|4|8|16) != 1|2|4|8|16 {
		t.Fatalf("books 4A2F=%02X, want the five commission books", bits)
	}
	for slot := uint16(4); slot <= 8; slot++ {
		if got := application.eventMachine.Memory[0x4AA6+slot]; got != uint16(gamepack.CityHallSlotPending) {
			t.Fatalf("slot %d = %02X, want FE", slot, got)
		}
	}
	if got := application.eventMachine.Memory[0x4A01]; got != 0 {
		t.Fatalf("4A01=%d after the library, want 0 (the ticket flag must stay clear)", got)
	}
	text := driver.bumpLibraryNorthDoorFromInside()
	if !strings.Contains(text, "SPECTRE") {
		t.Fatalf("bumping the north door with the books did not summon the spectre: %q", text)
	}
	t.Logf("spectre: %q", text)
	t.Logf("log: %v", driver.log)
}
