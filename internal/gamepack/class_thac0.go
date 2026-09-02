package gamepack

import (
	"encoding/hex"
	"fmt"
)

const (
	// ClassThac0ClassCount 是 DS:3C16h 表的列數，與 record +96h 起的
	// 每職業等級陣列同長度（spec 063）。
	ClassThac0ClassCount = 8
	// ClassThac0RowSize 是每列的 byte 數。表折疊成 1-based：等級 1..10 用
	// index 1..10，index 0 空著，所以列長是 11 而不是 10。
	ClassThac0RowSize = 11
	// ClassThac0MaxLevel 是表能查到的最高等級。
	ClassThac0MaxLevel = ClassThac0RowSize - 1
)

// ParseClassThac0Table 解出 DS:3C16h 起的 8×11 bytes。長度不符就失敗即關閉，
// 不接受別的 Gold Box 版本的表。
func ParseClassThac0Table(raw []byte) ([ClassThac0ClassCount][ClassThac0RowSize]uint8, error) {
	var table [ClassThac0ClassCount][ClassThac0RowSize]uint8
	want := ClassThac0ClassCount * ClassThac0RowSize
	if len(raw) != want {
		return table, fmt.Errorf("Pool class THAC0 table has %d bytes, want %d", len(raw), want)
	}
	for class := range table {
		copy(table[class][:], raw[class*ClassThac0RowSize:(class+1)*ClassThac0RowSize])
	}
	return table, nil
}

// originalClassThac0Hex 是 START.EXE 的 DS:3C16h 起 88 bytes，逐位元組照抄。
// 值是 internal encoding，typed THAC0 為 60 減它（spec 049）。
const originalClassThac0Hex = "282828282a2a2a2c2c2c2e282828282a2a2a2c2c2c2e2828292a2b2c2d2e2f30312828292a2b2c2d2e2f30312828292a2b2c2d2e2f303128282828282829292929292828282828292929292c2c282828282a2a2a2c2c2c2e"

// OriginalClassThac0Table 解出原版的職業／等級 THAC0 表。內容是編譯進
// START.EXE 的初始化資料，不隨存檔改變。
func OriginalClassThac0Table() [ClassThac0ClassCount][ClassThac0RowSize]uint8 {
	raw, err := hex.DecodeString(originalClassThac0Hex)
	if err != nil {
		panic("Pool class THAC0 table is not valid hex: " + err.Error())
	}
	table, err := ParseClassThac0Table(raw)
	if err != nil {
		panic("Pool class THAC0 table is malformed: " + err.Error())
	}
	return table
}

// BaseThac0Internal 重現 overlay-16 `0CC6h..0D4Fh`：從 0 起算，對每個等級大於 0
// 的職業查表，取 internal 最大者——internal 越大 typed THAC0 越小，所以這就是
// 「多職業取最好的那一個」。levels 依 record +96h 起的順序，索引即職業。
//
// 等級超出表的範圍時失敗即關閉；原版沒有這條路徑（等級上限由別處管），
// 這裡不猜它會讀到什麼。
func BaseThac0Internal(levels [ClassThac0ClassCount]uint8) (uint8, error) {
	table := OriginalClassThac0Table()
	var best uint8
	for class, level := range levels {
		if level == 0 {
			continue
		}
		if int(level) > ClassThac0MaxLevel {
			return 0, fmt.Errorf("Pool class %d level %d is outside the THAC0 table", class, level)
		}
		if value := table[class][level]; value > best {
			best = value
		}
	}
	return best, nil
}
