package gamepack

import (
	"path/filepath"
	"strings"
	"testing"
)

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
}
