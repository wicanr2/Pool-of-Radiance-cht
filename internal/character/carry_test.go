package character

import (
	"encoding/binary"
	"testing"
)

func weightedItem(weight uint16, quantity byte) []byte {
	raw := make([]byte, ItemRecordSize)
	binary.LittleEndian.PutUint16(raw[0x37:0x39], weight)
	raw[0x39] = quantity
	return raw
}

func TestCarryCapacityOriginalTable(t *testing.T) {
	tests := []struct{ strength, exceptional, want int }{
		{3, 0, 1150}, {5, 0, 1250}, {7, 0, 1350}, {10, 0, 1500},
		{13, 0, 1600}, {15, 0, 1700}, {16, 0, 1850}, {17, 0, 2000},
		{18, 0, 2250}, {18, 51, 2750}, {18, 100, 4500}, {19, 0, 5500},
		{20, 0, 6500}, {21, 0, 7500}, {25, 0, 16500},
	}
	for _, test := range tests {
		got, err := CarryCapacity(test.strength, test.exceptional)
		if err != nil || got != test.want {
			t.Errorf("CarryCapacity(%d,%d)=%d,%v want %d", test.strength, test.exceptional, got, err, test.want)
		}
	}
	if _, err := CarryCapacity(18, 101); err == nil {
		t.Fatal("invalid exceptional strength accepted")
	}
}

func TestReceiveItemUsesPreInsertionCountAndWeight(t *testing.T) {
	item := weightedItem(100, 2)
	if load, err := ItemLoad(item); err != nil || load != 200 {
		t.Fatalf("load=%d err=%v", load, err)
	}
	inventory := make([][]byte, 15)
	for index := range inventory {
		inventory[index] = weightedItem(1, 0)
	}
	if ok, err := CanReceiveItem(10, 0, inventory, item); err != nil || !ok {
		t.Fatalf("15 existing items should accept item 16: ok=%v err=%v", ok, err)
	}
	inventory = append(inventory, weightedItem(1, 0))
	if ok, err := CanReceiveItem(21, 0, inventory, item); err != nil || ok {
		t.Fatalf("16 existing items should reject: ok=%v err=%v", ok, err)
	}
	if ok, err := CanReceiveItem(3, 0, [][]byte{weightedItem(1000, 0)}, weightedItem(150, 0)); err != nil || !ok {
		t.Fatalf("capacity boundary should accept: ok=%v err=%v", ok, err)
	}
	if ok, err := CanReceiveItem(3, 0, [][]byte{weightedItem(1000, 0)}, weightedItem(151, 0)); err != nil || ok {
		t.Fatalf("over capacity should reject: ok=%v err=%v", ok, err)
	}
}
