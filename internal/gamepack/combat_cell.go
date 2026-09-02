package gamepack

import (
	"encoding/hex"
	"fmt"
)

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

// originalCombatCellClassHex 是 START.EXE 的 DS:2758h 起 66×4 bytes，
// 逐位元組照抄。這是本表在 repo 裡的唯一一份；戰鬥層與畫面都由
// OriginalCombatCellClassTable 取得，不另建副本。
const originalCombatCellClassHex = "0100ff00ff000200ff000201ff000202ff00020301000004ff000205ff000206ff00020701000008ff0002090100000aff00020b0100000cff00020d0100000eff00020f01000010ff000211ff000212ff000213ff000214ff0002150100001601000017ff000218010000220100002301000024010000250100002601000027ff000200ff000201ff000202ff00020301000004010000050100000601000007ff000008ff0000090100000a0100000b0100000c0100000d0100000e0100000f0100001001000011ff000012ff000013010000140100001501000016ff000017ff000018010000190100001a0100001bff00001cff00021dff00021eff00021f01000020ff000221"

// OriginalCombatCellClassTable 解出原版的 66 筆格位類別表。
// 表的內容是編譯進 START.EXE 的初始化資料，不隨存檔或關卡改變，
// 因此每次呼叫都回傳同一份值的複本。
func OriginalCombatCellClassTable() [CombatCellClassCount]CombatCellClass {
	raw, err := hex.DecodeString(originalCombatCellClassHex)
	if err != nil {
		panic("Pool combat cell class table is not valid hex: " + err.Error())
	}
	records, err := ParseCombatCellClassTable(raw)
	if err != nil {
		panic("Pool combat cell class table does not parse: " + err.Error())
	}
	return records
}
