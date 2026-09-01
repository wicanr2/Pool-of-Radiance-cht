package gamepack

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/wicanr2/golden-box-remake-engine/eclvm"
)

func TestInitialCharacterProjectorSelectsAndPreservesPreviousCharacter(t *testing.T) {
	project := initialCharacterProjector([]InitialCharacter{{Name: "ALICE", ControlMorale: 0x12}, {Name: "NPC", ControlMorale: 0x80}})
	memory := map[uint16]uint16{}
	stringsMemory := map[uint16]string{}
	if err := project(eclvm.CharacterSelection{Index: 1}, memory, stringsMemory); err != nil {
		t.Fatal(err)
	}
	if stringsMemory[0x6B00] != "NPC" || memory[0x6C00] != 1 || memory[0x6BB8] != 0x80 {
		t.Fatalf("selected projection memory=%v strings=%v", memory, stringsMemory)
	}
	if err := project(eclvm.CharacterSelection{Index: 7}, memory, stringsMemory); err != nil {
		t.Fatal(err)
	}
	if stringsMemory[0x6B00] != "NPC" || memory[0x6C00] != 1 || memory[0x6BB8] != 0x80 {
		t.Fatalf("missing selector changed prior projection: memory=%v strings=%v", memory, stringsMemory)
	}
}

func TestInitialCharacterProjectorLeavesEmptyPartyUnprojected(t *testing.T) {
	memory := map[uint16]uint16{}
	stringsMemory := map[uint16]string{}
	if err := initialCharacterProjector(nil)(eclvm.CharacterSelection{Index: 0}, memory, stringsMemory); err != nil {
		t.Fatal(err)
	}
	if memory[0x6C00] != 0 || stringsMemory[0x6B00] != "" {
		t.Fatalf("empty party projection memory=%v strings=%v", memory, stringsMemory)
	}
}

func TestInitialPartyStrengthProjectionMatchesDOSClassAnchors(t *testing.T) {
	tests := []struct {
		name    string
		value   InitialCharacter
		fields  [5]uint8
		contrib uint8
	}{
		{name: "FEM fighter", value: InitialCharacter{ClassID: "fighter", Abilities: [6]int{14, 15, 16, 13, 15, 13}, CurrentHP: 7}, fields: [5]uint8{0, 0, 40, 50, 7}, contrib: 1},
		{name: "HMU magic-user", value: InitialCharacter{ClassID: "magic-user", Abilities: [6]int{17, 15, 14, 14, 13, 14}, CurrentHP: 2}, fields: [5]uint8{0, 1, 41, 50, 2}, contrib: 2},
		{name: "HTH thief", value: InitialCharacter{ClassID: "thief", Abilities: [6]int{13, 16, 12, 16, 14, 13}, CurrentHP: 4}, fields: [5]uint8{0, 0, 40, 52, 4}, contrib: 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			record, err := initialPartyStrengthRecord(test.value)
			if err != nil {
				t.Fatal(err)
			}
			got := [5]uint8{record.Field96, record.Field9B, record.Field110, record.Field111, record.Field11B}
			if got != test.fields || record.Contribution() != test.contrib {
				t.Fatalf("fields=%v contribution=%d, want %v/%d", got, record.Contribution(), test.fields, test.contrib)
			}
		})
	}
	resolver := initialPartyStrengthResolver([]InitialCharacter{tests[0].value, tests[1].value, tests[2].value})
	if got, err := resolver(); err != nil || got != 3 {
		t.Fatalf("party strength=%d err=%v, want 3", got, err)
	}
}

func TestInitialPartyStrengthProjectionFailsClosedOnUnknownCharacterShape(t *testing.T) {
	for _, value := range []InitialCharacter{
		{ClassID: "unknown", Abilities: [6]int{10, 10, 10, 10, 10, 10}, CurrentHP: 1},
		{ClassID: "fighter", Abilities: [6]int{26, 10, 10, 10, 10, 10}, CurrentHP: 1},
		{ClassID: "fighter", Abilities: [6]int{18, 10, 10, 10, 10, 10}, ExceptionalStrength: 101, CurrentHP: 1},
	} {
		if _, err := initialPartyStrengthRecord(value); err == nil {
			t.Fatalf("invalid projection accepted: %+v", value)
		}
	}
}

func TestRealBlock8GraveyardStrengthGateUsesInlineResolver(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	event, err := ReadDOSInitialEvent(zipPath)
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	for _, test := range []struct {
		name, anchor string
		hp           int
		want         uint8
	}{
		{name: "below threshold", hp: 175, want: 18, anchor: "ON THE MATTER OF COMMISSION"},
		{name: "at threshold", hp: 185, want: 19, anchor: "VALHINGEN GRAVEYARD"},
	} {
		t.Run(test.name, func(t *testing.T) {
			member := InitialCharacter{ClassID: "fighter", Abilities: [6]int{14, 10, 10, 13, 10, 10}, CurrentHP: test.hp}
			fixture := event
			fixture.HandlerAddress = 0xA592
			fixture.ScriptBlock = nil
			fixture.ScriptBlocks = map[uint16][]byte{0: event.ScriptBlocks[8]}
			session, err := NewInitialEventSession(fixture, member)
			if err != nil {
				t.Fatal(err)
			}
			machine := session.Machine()
			machine.Memory[0x4AC1], machine.Memory[0x4AB1], machine.Memory[0x4A96] = 4, 0, 0
			result, err := machine.RunUntilEvent(128, nil, true)
			if err != nil {
				t.Fatal(err)
			}
			if machine.Memory[0x6E79] != uint16(test.want) || len(result.PartyStrengthRequests) != 1 || result.PartyStrengthRequests[0].Value != test.want || len(result.Events) != 1 || !strings.Contains(result.Events[0].Text, test.anchor) {
				t.Fatalf("strength=%d requests=%v events=%v", machine.Memory[0x6E79], result.PartyStrengthRequests, result.Events)
			}
		})
	}
}

func TestRealBlock8GraveyardTreasurePrecedesCombat(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	event, err := ReadDOSInitialEvent(zipPath)
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	fixture := event
	fixture.HandlerAddress = 0xA780
	fixture.ScriptBlock = nil
	fixture.ScriptBlocks = map[uint16][]byte{0: event.ScriptBlocks[8]}
	session, err := NewInitialEventSession(fixture)
	if err != nil {
		t.Fatal(err)
	}
	result, err := session.Machine().RunUntilEvent(8, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	want := eclvm.TreasureRequest{ItemBlock: 0x33}
	if len(result.TreasureRequests) != 1 || result.TreasureRequests[0] != want || len(result.Events) != 1 || result.Events[0].Opcode != 0x24 {
		t.Fatalf("result=%+v, want graveyard TREASURE then COMBAT", result)
	}
}

func TestRealSlumsEncounterCarriesMonsterRosterToCombatBoundary(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	archive, err := ReadDOSECLArchive(zipPath, 2)
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	session, err := NewDOSECLArchiveSession(archive, 20, 0x9E5D)
	if err != nil {
		t.Fatal(err)
	}
	result, err := session.RunUntilEvent(16, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	want := []eclvm.MonsterSpawn{{MonsterID: 13, Count: 1, IconBlock: 4}, {MonsterID: 4, Count: 3, IconBlock: 4}}
	if !result.CombatRequested || !result.MonstersCleared || !reflect.DeepEqual(result.MonsterSpawns, want) {
		t.Fatalf("Slums combat boundary=%+v, want roster=%+v", result, want)
	}
	if got := uint16(0x9900 + session.Machine().PC); got != 0x9E6D {
		t.Fatalf("continuation PC=0x%04X, want 0x9E6D", got)
	}
	if session.Machine().Memory[0x4ABB] != 0 {
		t.Fatalf("combat request prematurely changed Slums progress to %d", session.Machine().Memory[0x4ABB])
	}
}

func TestRealBlock8GraveyardRewardAccumulatorSlots(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	event, err := ReadDOSInitialEvent(zipPath)
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	wants := [7]eclvm.TreasureRequest{
		{Amounts: [7]uint16{0, 0, 0, 1, 0, 0, 0}, ItemBlock: 0xFF},
		{Amounts: [7]uint16{0, 0, 0, 0, 1, 0, 0}, ItemBlock: 0xFF},
		{Amounts: [7]uint16{0, 0, 0, 0, 0, 1, 0}, ItemBlock: 0xFF},
		{Amounts: [7]uint16{0, 0, 0, 0, 0, 0, 1}, ItemBlock: 0xFF},
		{Amounts: [7]uint16{0, 0, 0, 0, 1, 0, 0}, ItemBlock: 0xFF},
		{Amounts: [7]uint16{0, 0, 0, 0, 0, 1, 0}, ItemBlock: 0xFF},
		{Amounts: [7]uint16{0, 0, 0, 0, 0, 0, 1}, ItemBlock: 0xFF},
	}
	for slot := range wants {
		t.Run(string(rune('0'+slot)), func(t *testing.T) {
			fixture := event
			fixture.HandlerAddress = 0x9C34
			fixture.ScriptBlock = nil
			fixture.ScriptBlocks = map[uint16][]byte{0: event.ScriptBlocks[8]}
			session, err := NewInitialEventSession(fixture, InitialCharacter{ClassID: "fighter", Abilities: [6]int{14, 10, 10, 13, 10, 10}, CurrentHP: 8})
			if err != nil {
				t.Fatal(err)
			}
			machine := session.Machine()
			machine.Memory[0x4A39+uint16(slot)] = 1
			var pages []string
			var treasures []eclvm.TreasureRequest
			for boundary := 0; boundary < 80; boundary++ {
				result, err := machine.RunUntilEvent(4096, nil, true)
				if err != nil {
					t.Fatalf("boundary %d: %v", boundary, err)
				}
				for _, event := range result.Events {
					if event.Text != "" {
						pages = append(pages, event.Text)
					}
				}
				treasures = append(treasures, result.TreasureRequests...)
				if result.WaitingForMenu {
					result, err = machine.RunUntilEvent(4096, []uint16{0}, true)
					if err != nil {
						t.Fatalf("boundary %d menu: %v", boundary, err)
					}
					for _, event := range result.Events {
						if event.Text != "" {
							pages = append(pages, event.Text)
						}
					}
					treasures = append(treasures, result.TreasureRequests...)
				}
				if result.Exited {
					break
				}
			}
			joined := strings.Join(pages, "\n")
			if !strings.Contains(joined, "ELIMINATED SOME UNDEAD FROM THE GRAVEYARD") || !strings.Contains(joined, "HERE IS YOUR REWARD") {
				t.Fatalf("slot %d missing graveyard reward pages: %q", slot, pages)
			}
			if len(treasures) != 1 || treasures[0] != wants[slot] {
				t.Fatalf("slot %d treasures=%+v, want %+v", slot, treasures, wants[slot])
			}
			if machine.Memory[0x4A8F+uint16(slot)] != 1 {
				t.Fatalf("slot %d acknowledgement was not updated", slot)
			}
			if machine.Memory[0x4AC1] != 0 {
				t.Fatalf("slot %d changed later proclamation progress to %d", slot, machine.Memory[0x4AC1])
			}
		})
	}
}

// This is the first second-title consumer of the shared VM core. It executes
// the original bytes rather than replaying the typed TourStep projection.
func TestSharedVMRunsRealRolfTourToExit(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	event, err := ReadDOSInitialEvent(zipPath)
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	machine, err := NewInitialEventMachine(event)
	if err != nil {
		t.Fatal(err)
	}

	pages := make([]string, 0, 8)
	writes := map[uint16]uint16{}
	opcodes := map[byte]bool{}
	result, err := machine.Run(2000, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	for run := 0; ; run++ {
		for _, event := range result.Events {
			opcodes[event.Opcode] = true
			if event.Text != "" {
				pages = append(pages, event.Text)
			}
		}
		for _, write := range result.Writes {
			writes[write.Address] = write.Value
		}
		if result.Exited {
			break
		}
		if !result.WaitingForMenu {
			t.Fatalf("run %d stopped without menu or EXIT: %+v", run, result)
		}
		if run > 12 {
			t.Fatal("Rolf tour did not terminate")
		}
		result, err = machine.Run(4000, []uint16{0}, true)
		if err != nil {
			t.Fatal(err)
		}
	}
	joined := strings.Join(pages, "\n")
	for _, anchor := range []string{"GREETINGS, COURAGEOUS ONES", "TEMPLE OF TYR", "PASSENGER DOCK", "TRAINING", "CITY HALL", "OLD CITY", "ON YOUR OWN"} {
		if !strings.Contains(joined, anchor) {
			t.Errorf("missing page anchor %q", anchor)
		}
	}
	if writes[0xC04B] != 0 || writes[0xC04C] != 4 || writes[0xC04D] != 3 {
		t.Fatalf("final position writes=(%d,%d,%d)", writes[0xC04B], writes[0xC04C], writes[0xC04D])
	}
	for _, opcode := range []byte{0x0D, 0x0E, 0x2D, 0x31, 0x3A} {
		if !opcodes[opcode] {
			t.Errorf("declared passthrough opcode 0x%02X was not exercised", opcode)
		}
	}
	catalog, err := ReadDOSGeometryCatalog(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	initial, ok := catalog.Map(MapKey{Archive: 3, BlockID: 0})
	if !ok {
		t.Fatal("initial map absent")
	}
	cellResult, err := RunInitialCellEntry(machine, initial.Grid, Spawn{Map: initial.Key, X: 1, Y: 4, Facing: 2})
	if err != nil {
		t.Fatal(err)
	}
	if !cellResult.Exited || cellResult.WaitingForMenu || cellResult.PC+0x9900 != 0x997E || cellResult.Steps != 15 || len(cellResult.Events) != 0 {
		t.Fatalf("first moved cell result: pc=%04X exited=%v waiting=%v steps=%d events=%v menus=%v", cellResult.PC+0x9900, cellResult.Exited, cellResult.WaitingForMenu, cellResult.Steps, cellResult.Events, cellResult.Menus)
	}
}

func TestCityHallCommissionSelectsOriginalProclamation(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	event, err := ReadDOSInitialEvent(zipPath)
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	catalog, err := ReadDOSGeometryCatalog(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	initial, ok := catalog.Map(MapKey{Archive: 3, BlockID: 0})
	if !ok {
		t.Fatal("initial map absent")
	}
	want := []string{"CI.", "CXXVI AND CX.", "CXXXIV.", "CLIV.", "CXIV.", "CCIV.", "CXXIX.", "CCI.", "CXIV."}
	for commission, numeral := range want {
		session, err := NewInitialEventSession(event)
		if err != nil {
			t.Fatal(err)
		}
		session.Machine().Memory[0x4AC5] = 1
		session.Machine().Memory[0x4AC1] = uint16(commission + 1)
		entry, err := RunInitialSessionCellEntry(session, initial.Grid, Spawn{Map: initial.Key, X: 2, Y: 4, Facing: 2})
		if err != nil || !entry.Exited || entry.WaitingForMenu || len(entry.Events) != 0 {
			t.Fatalf("commission %d old-cell entry=%+v err=%v", commission+1, entry, err)
		}
		result, err := RunInitialSessionSearchEntry(session, initial.Grid, Spawn{Map: initial.Key, X: 3, Y: 4, Facing: 2})
		if err != nil {
			t.Fatalf("commission %d entry: %v", commission+1, err)
		}
		var pages []string
		for boundary := 0; boundary < 12; boundary++ {
			for _, event := range result.Events {
				if event.Text != "" {
					pages = append(pages, event.Text)
				}
			}
			if result.Exited {
				break
			}
			var choices []uint16
			if result.WaitingForMenu {
				choices = []uint16{0}
			}
			result, err = session.RunUntilEvent(4096, choices, true)
			if err != nil {
				t.Fatalf("commission %d boundary %d: %v", commission+1, boundary, err)
			}
		}
		joined := strings.Join(pages, "\n")
		if !result.Exited || !strings.Contains(joined, "PROCLAMATIONS ARE POSTED ON THE WALLS") || !strings.Contains(joined, "PROCLAMATION\n"+numeral) {
			t.Fatalf("commission %d exited=%v pages=%q, want numeral %q", commission+1, result.Exited, pages, numeral)
		}
	}
}
