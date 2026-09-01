package gamepack

import (
	"encoding/hex"
	"testing"
)

const originalCombatCellClassHex = "0100ff00ff000200ff000201ff000202ff00020301000004ff000205ff000206ff00020701000008ff0002090100000aff00020b0100000cff00020d0100000eff00020f01000010ff000211ff000212ff000213ff000214ff0002150100001601000017ff000218010000220100002301000024010000250100002601000027ff000200ff000201ff000202ff00020301000004010000050100000601000007ff000008ff0000090100000a0100000b0100000c0100000d0100000e0100000f0100001001000011ff000012ff000013010000140100001501000016ff000017ff000018010000190100001a0100001bff00001cff00021dff00021eff00021f01000020ff000221"

func TestParseCombatCellClassTableOriginalSTARTEXEBytes(t *testing.T) {
	raw, err := hex.DecodeString(originalCombatCellClassHex)
	if err != nil {
		t.Fatalf("decode fixed Pool table: %v", err)
	}
	records, err := ParseCombatCellClassTable(raw)
	if err != nil {
		t.Fatalf("ParseCombatCellClassTable: %v", err)
	}
	tests := []struct {
		index int
		want  CombatCellClass
	}{
		{index: 0, want: CombatCellClass{EntryThreshold: 1, PathByte2: 0xFF}},
		{index: 1, want: CombatCellClass{EntryThreshold: 0xFF, PathByte2: 2}},
		{index: 5, want: CombatCellClass{EntryThreshold: 1, PresentationCode: 4}},
		{index: 32, want: CombatCellClass{EntryThreshold: 0xFF, PathByte2: 2}},
		{index: 65, want: CombatCellClass{EntryThreshold: 0xFF, PathByte2: 2, PresentationCode: 0x21}},
	}
	for _, test := range tests {
		if got := records[test.index]; got != test.want {
			t.Fatalf("record %d=%+v want %+v", test.index, got, test.want)
		}
	}
}

func TestParseCombatCellClassTableRejectsWrongShape(t *testing.T) {
	for _, size := range []int{0, CombatCellClassCount*CombatCellClassRecordSize - 1, CombatCellClassCount*CombatCellClassRecordSize + 1} {
		if _, err := ParseCombatCellClassTable(make([]byte, size)); err == nil {
			t.Fatalf("ParseCombatCellClassTable accepted %d bytes", size)
		}
	}
}
