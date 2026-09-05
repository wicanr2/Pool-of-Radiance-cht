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
	// `SETUP MONSTER 12,2,9` 的三個 operand（spec 117）：SPRIT 區塊、接近距離、
	// BODY 區塊。第三個要是 9——半身像的下半就是 `BODY3.DAX` 區塊 9。
	if event.SpriteBlock != 12 || event.ApproachDistance != 2 || event.PortraitBody != 9 {
		t.Fatalf("SETUP MONSTER operands=%d,%d,%d，預期 12,2,9",
			event.SpriteBlock, event.ApproachDistance, event.PortraitBody)
	}
	if !strings.Contains(event.Message, "ROLF") || !strings.Contains(event.Message, "PHLAN") {
		t.Fatalf("greeting lacks semantic anchors: %q", event.Message)
	}
	if event.ContinueLabel != "PRESS <RETURN> OR BUTTON TO CONTINUE" {
		t.Fatalf("continue label=%q", event.ContinueLabel)
	}
	if len(event.Tour) != 34 {
		t.Fatalf("tour steps=%d, want 34", len(event.Tour))
	}
	stops := map[int]struct {
		position Spawn
		selector uint8
		pages    int
		anchor   string
	}{
		5:  {Spawn{Map: MapKey{Archive: 3, BlockID: 0}, X: 11, Y: 2, Facing: 2}, 1, 1, "TEMPLE OF TYR"},
		7:  {Spawn{Map: MapKey{Archive: 3, BlockID: 0}, X: 11, Y: 2, Facing: 0}, 2, 1, "PASSENGER DOCK"},
		20: {Spawn{Map: MapKey{Archive: 3, BlockID: 0}, X: 5, Y: 2, Facing: 1}, 3, 1, "TRAINING SCHOOLS"},
		25: {Spawn{Map: MapKey{Archive: 3, BlockID: 0}, X: 4, Y: 3, Facing: 2}, 4, 1, "CITY HALL"},
		32: {Spawn{Map: MapKey{Archive: 3, BlockID: 0}, X: 1, Y: 4, Facing: 3}, 5, 1, "SUNE'S TEMPLE"},
		33: {Spawn{Map: MapKey{Archive: 3, BlockID: 0}, X: 0, Y: 4, Facing: 3}, 6, 2, "OLD CITY"},
	}
	for index, step := range event.Tour {
		want, isStop := stops[index]
		if !isStop {
			if step.Selector != 0 || len(step.Messages) != 0 {
				t.Fatalf("tour step %d unexpectedly pauses: %+v", index, step)
			}
			continue
		}
		if step.Position != want.position || step.Selector != want.selector || len(step.Messages) != want.pages || !strings.Contains(step.Messages[0], want.anchor) {
			t.Fatalf("tour stop %d=%+v", index, step)
		}
	}
	if !strings.Contains(event.Tour[33].Messages[1], "ON YOUR OWN NOW") {
		t.Fatalf("tour final page lacks completion anchor")
	}
}
