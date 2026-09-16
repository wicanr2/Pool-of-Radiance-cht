package main

// 古托井井底那一場（#37，spec 137「古托井與索寇要塞的門」）：ECL8/29 `9DDBh..9DE9h`
// 三條 LOAD MONSTER 是 32×1、57×5、1×9，MON8CHA 那三筆記錄的 `+2Dh`／`+111h` 給的
// THAC0 與 AC，remake 擺上盤面的 combatant 要逐格相同——蜥蜴人 AC 4、THAC0 16 是
// 原版資料，不是 remake 讀錯。位址基準先拿 `9A41h` 那條已解出的 GETTABLE 對過。

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/golden-box-remake-engine/eclvm"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// eclBlockBytes 讀 ECL 區塊在 9900h 基準位址上的 n 個位元組（區塊前兩個位元組是長度標頭）。
func eclBlockBytes(t *testing.T, archive gamepack.ECLArchive, block uint16, address, n int) []byte {
	t.Helper()
	data, ok := archive.Blocks[block]
	if !ok {
		t.Fatalf("ECL%d has no block %d", archive.Number, block)
	}
	offset := address - 0x9900 + 2
	if offset < 0 || offset+n > len(data) {
		t.Fatalf("ECL%d/%d: %04X+%d is outside the block (%d bytes)", archive.Number, block, address, n, len(data))
	}
	return data[offset : offset+n]
}

func TestNorrisFightCompositionAndNumbersComeFromTheOriginalData(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	archive, ok := application.eclCatalog.Archive(8)
	if !ok {
		t.Fatal("ECL8 archive is absent")
	}
	// 正對照：`9A41h GETTABLE @AFCEh @6E82h @9800h`（地形碼換分派索引那一條）。
	if got, want := eclBlockBytes(t, archive, 29, 0x9A41, 10), []byte{0x2A, 0x01, 0xCE, 0xAF, 0x01, 0x82, 0x6E, 0x01, 0x00, 0x98}; !bytes.Equal(got, want) {
		t.Fatalf("ECL8/29 9A41h reads % X, want % X (address base is off)", got, want)
	}
	// `AFCEh` 的表：地形 12 → 索引 15（`9D1Bh`，諾里斯）。
	if table := eclBlockBytes(t, archive, 29, 0xAFCE, 16); table[12] != 15 || table[1] != 2 || table[13] != 14 {
		t.Fatalf("AFCEh dispatch table is % X; want terrain 12 → 15, 1 → 2, 13 → 14", table)
	}
	// `9DDBh..9DF0h`：三條 `0Bh LOAD MONSTER`（運算元各是 code 0 的位元組）與 `24h COMBAT`
	//（Pool 的 opcode 表，`gamepack.PoolCommandTable`）。
	want := []byte{
		0x0B, 0x00, 32, 0x00, 1, 0x00, 7,
		0x0B, 0x00, 57, 0x00, 5, 0x00, 57,
		0x0B, 0x00, 1, 0x00, 9, 0x00, 1,
		0x24,
	}
	if got := eclBlockBytes(t, archive, 29, 0x9DDB, len(want)); !bytes.Equal(got, want) {
		t.Fatalf("ECL8/29 9DDBh reads % X, want % X", got, want)
	}

	type expect struct {
		id    uint8
		name  string
		base  uint8 // +2Dh
		thac0 int   // 表面值
		ac    int
	}
	expects := []expect{
		{32, "NORRIS THE GRAY", 45, 14, 7},
		{57, "LIZARDMAN", 44, 16, 4},
		{1, "KOBOLD LEADER", 41, 19, 7},
	}
	for _, e := range expects {
		record, err := gamepack.ReadDOSMonsterRecord(zipPath, 8, e.id)
		if err != nil {
			t.Fatal(err)
		}
		internal, err := record.CombatThac0Internal()
		if err != nil {
			t.Fatal(err)
		}
		if record.Name != e.name || record.Raw[0x2D] != e.base || 60-int(internal) != e.thac0 || record.ArmorClass() != e.ac {
			t.Fatalf("MON8CHA/%d is %q +2Dh=%d thac0=%d ac=%d, want %q %d %d %d",
				e.id, record.Name, record.Raw[0x2D], 60-int(internal), record.ArmorClass(), e.name, e.base, e.thac0, e.ac)
		}
	}

	// 擺盤：井底 GEO8/32 (10,3) 朝北，腳本那三條 LOAD MONSTER；每一格敵方的 THAC0 與 AC 對回記錄。
	session, err := gamepack.NewDOSECLArchiveSession(archive, 29, 0x9914)
	if err != nil {
		t.Fatal(err)
	}
	geoMap, ok := application.geometryCatalog.Map(gamepack.MapKey{Archive: 8, BlockID: 32})
	if !ok {
		t.Fatal("GEO8/32 is absent")
	}
	if cell, _ := geoMap.Grid.Cell(10, 3); cell.Terrain&0x7F != 12 {
		t.Fatalf("GEO8/32 (10,3) terrain is %02X, want low bits 12", cell.Terrain)
	}
	application.eventSession, application.eventMachine = session, session.Machine()
	application.eclArchive = 8
	application.initialMap = &geoMap
	application.spawn = gamepack.Spawn{Map: geoMap.Key, X: 10, Y: 3, Facing: 0}
	application.introDone, application.mode = true, modeAdventure
	if err := application.configureEventSession(session); err != nil {
		t.Fatal(err)
	}
	application.saveState = func(poolsave.State) error { return nil }
	application.eventMachine.Memory[encounterWalkFlagAddress] = 1
	application.eventMachine.Memory[encounterDistanceAddress] = 0
	party := make([]poolsave.Character, 0, 6)
	for index, hp := range []int{7, 7, 6, 4, 8, 9} {
		party = append(party, poolsave.Character{Name: string(rune('A' + index)), RaceID: "human",
			GenderID: "male", ClassID: "fighter", AlignmentID: "lawful-good",
			Abilities: [6]int{15, 10, 10, 13, 10, 10}, MaxHP: hp, CurrentHP: hp,
			PortraitHead: 1, PortraitBody: 1, IconSize: 1})
	}
	application.state = poolsave.State{Schema: poolsave.Schema, CharacterLibrary: party, Party: party}
	spawns := []eclvm.MonsterSpawn{{MonsterID: 32, Count: 1, IconBlock: 7}, {MonsterID: 57, Count: 5, IconBlock: 57}, {MonsterID: 1, Count: 9, IconBlock: 1}}
	if err := application.enterCombatStaging(spawns); err != nil {
		t.Fatal(err)
	}
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	state := application.tactical
	if state == nil {
		t.Fatal("no tactical state")
	}
	// 盤面的順序就是 LOAD MONSTER 的順序（隊員佔 1..6，敵方從 7 起）：諾里斯、五個蜥蜴人、
	// 九個狗頭人首領。每一格的 THAC0／AC／HP 對回記錄（HP 是 `+32h`，25／11／4）。
	order := []expect{expects[0]}
	for i := 0; i < 5; i++ {
		order = append(order, expects[1])
	}
	for i := 0; i < 9; i++ {
		order = append(order, expects[2])
	}
	hp := map[string]int{"NORRIS THE GRAY": 25, "LIZARDMAN": 11, "KOBOLD LEADER": 4}
	foes := 0
	for index := 1; index < len(state.Roster); index++ {
		if state.Friendly[index] || state.Roster[index].FootprintClass == 0 {
			continue
		}
		if foes >= len(order) {
			t.Fatalf("more than %d foes on the board", len(order))
		}
		e := order[foes]
		foes++
		if got := 60 - int(state.THAC0[index]); got != e.thac0 {
			t.Errorf("combatant %d (%s): THAC0 %d, record says %d", index, e.name, got, e.thac0)
		}
		if got := 60 - int(state.ArmorClass[index]); got != e.ac {
			t.Errorf("combatant %d (%s): AC %d, record says %d", index, e.name, got, e.ac)
		}
		if got := state.MaxHitPoints[index]; got != hp[e.name] {
			t.Errorf("combatant %d (%s): max HP %d, record says %d", index, e.name, got, hp[e.name])
		}
	}
	if foes != 15 {
		t.Fatalf("board has %d foes, want 15 (1 + 5 + 9; spec 061 deployment)", foes)
	}
	t.Logf("Norris's hall: 15 foes on the board; THAC0 14/16/19, AC 7/4/7, HP 25/11/4 all from MON8CHA")
}
