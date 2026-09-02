package gamepack

import "testing"

func TestParseCombatCellClassTableOriginalSTARTEXEBytes(t *testing.T) {
	records := OriginalCombatCellClassTable()
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
