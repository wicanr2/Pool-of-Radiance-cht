package gamepack

import (
	"path/filepath"
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
	for _, opcode := range []byte{0x0C, 0x0D, 0x0E, 0x2D, 0x31, 0x3A} {
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
