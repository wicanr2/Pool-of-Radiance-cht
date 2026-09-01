package gamepack

import "fmt"

const (
	CombatCellClassCount      = 66
	CombatCellClassRecordSize = 4
)

// CombatCellClass preserves one Pool DS:2758h record. EntryThreshold is
// consumed directly by player movement. PathByte1 and PathByte2 participate
// in the original route-selection function but remain numerically named until
// their producer/consumer contracts are closed. PresentationCode is passed to
// the tactical tile drawing service.
type CombatCellClass struct {
	EntryThreshold   uint8
	PathByte1        uint8
	PathByte2        uint8
	PresentationCode uint8
}

// ParseCombatCellClassTable decodes the exact 66×4 initialized table exported
// from START.EXE DS:2758h. It rejects truncated, extended, or shape-shifted
// input instead of silently accepting another Gold Box version's table.
func ParseCombatCellClassTable(raw []byte) ([CombatCellClassCount]CombatCellClass, error) {
	var records [CombatCellClassCount]CombatCellClass
	want := CombatCellClassCount * CombatCellClassRecordSize
	if len(raw) != want {
		return records, fmt.Errorf("Pool combat cell class table has %d bytes, want %d", len(raw), want)
	}
	for index := range records {
		offset := index * CombatCellClassRecordSize
		records[index] = CombatCellClass{
			EntryThreshold:   raw[offset],
			PathByte1:        raw[offset+1],
			PathByte2:        raw[offset+2],
			PresentationCode: raw[offset+3],
		}
	}
	return records, nil
}
