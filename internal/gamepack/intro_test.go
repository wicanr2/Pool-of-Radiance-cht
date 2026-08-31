package gamepack

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestReadDOSInitialEventMatchesSpec010(t *testing.T) {
	event, err := ReadDOSInitialEvent(filepath.Join("..", "..", "Pool of Radiance (1988).zip"))
	if err != nil {
		t.Fatal(err)
	}
	if event.TriggerAddress != 0x4AC5 || event.TriggerLimit != 1 || event.EntryAddress != 0x9AF2 || event.HandlerAddress != 0xB06E {
		t.Fatalf("event control identity=%+v", event)
	}
	wantPosition := Spawn{Map: MapKey{Archive: 3, BlockID: 0}, X: 15, Y: 1, Facing: 3}
	if event.Position != wantPosition || event.MonsterID != 12 {
		t.Fatalf("position=%+v monster=%d", event.Position, event.MonsterID)
	}
	if !strings.Contains(event.Message, "ROLF") || !strings.Contains(event.Message, "PHLAN") {
		t.Fatalf("greeting lacks semantic anchors: %q", event.Message)
	}
	if event.ContinueLabel != "PRESS <RETURN> OR BUTTON TO CONTINUE" {
		t.Fatalf("continue label=%q", event.ContinueLabel)
	}
}
