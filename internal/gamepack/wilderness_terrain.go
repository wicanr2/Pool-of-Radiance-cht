package gamepack

import "fmt"

// 野外戰場挑背景用的大地圖地形表（spec 060）。
//
// overlay-10 `1255h` 讀 `[4933h]+186h`／`+188h`（也就是 ECL `@49C3`／`@49C4`，
// 野外座標）當 X／Y，依 `DS:495Bh` 的模式把 X 加 0／13／26，再以
// `y × 2Ch + x` 查 `DS:35E2h`，結果存進 `DS:45BCh` 交給四支細節建構器。
//
// 表寬 44 是 `12A9h` 的 `mov dx, 2Ch` 直接寫死的（exact）。高 36 是從資料推的
// （strong inference）：第 34、35 列整列都是 `4Eh`，第 36 列起的
// `00 0D 1A` 正好是三個模式的 X 偏移，已經是下一份資料。野外座標 Y 最大 33，
// 36 列蓋得住。
const (
	WildernessTerrainTableOffset = 0x35E2
	WildernessTerrainWidth       = 0x2C
	WildernessTerrainHeight      = 36
	WildernessTerrainSize        = WildernessTerrainWidth * WildernessTerrainHeight
)

// 戰場模式 `DS:495Bh`（overlay-03 `3653h..36DBh`，exact）。主迴圈每一圈先寫 1，
// 目前的 ECL 區塊（`DS:82A2h`）是 25／26／27 **而且** `@49E6` 為 0（站在野外
// 地形上，不是區域圖）時改成 2／3／4。overlay-10 `12E5h` 以「等於 1」分岔：
// 1 走室內 `0820h`，其餘走室外 `1255h`。
const (
	CombatAreaIndoor         uint8 = 1
	CombatAreaWildernessWest uint8 = 2 // ECL 區塊 25
	CombatAreaWildernessMid  uint8 = 3 // ECL 區塊 26
	CombatAreaWildernessEast uint8 = 4 // ECL 區塊 27
)

// wildernessSheetOffsets 是 `1289h..129Bh` 對 X 加的偏移：模式 3 加 0Dh、
// 模式 4 加 1Ah，模式 2 不加。
var wildernessSheetOffsets = map[uint8]int{
	CombatAreaWildernessWest: 0,
	CombatAreaWildernessMid:  0x0D,
	CombatAreaWildernessEast: 0x1A,
}

// CombatAreaMode 重現 overlay-03 `3653h..36DBh` 對 `DS:495Bh` 的寫入。
// block 是 `DS:82A2h`，walkFlag 是 ECL `@49E6`（`[4933h]+1CCh`）。
func CombatAreaMode(block uint16, walkFlag uint16) uint8 {
	if walkFlag != 0 {
		return CombatAreaIndoor
	}
	switch block {
	case 0x19:
		return CombatAreaWildernessWest
	case 0x1A:
		return CombatAreaWildernessMid
	case 0x1B:
		return CombatAreaWildernessEast
	}
	return CombatAreaIndoor
}

// WildernessTerrainTable 是 `DS:35E2h` 那張 44×36 的地形碼表。
type WildernessTerrainTable [WildernessTerrainSize]uint8

// ParseWildernessTerrainTable 只接受剛好一整張表的長度。
func ParseWildernessTerrainTable(raw []byte) (WildernessTerrainTable, error) {
	var table WildernessTerrainTable
	if len(raw) != WildernessTerrainSize {
		return table, fmt.Errorf("Pool wilderness terrain table has %d bytes, want %d", len(raw), WildernessTerrainSize)
	}
	copy(table[:], raw)
	return table, nil
}

// ReadDOSWildernessTerrainTable 從 DOS ZIP 的 START.EXE 讀出這張表。
func ReadDOSWildernessTerrainTable(zipPath string) (WildernessTerrainTable, error) {
	raw, err := readStartExecutable(zipPath)
	if err != nil {
		return WildernessTerrainTable{}, err
	}
	start := WildernessTerrainTableOffset + startDataSegmentFileDelta
	end := start + WildernessTerrainSize
	if len(raw) < end {
		return WildernessTerrainTable{}, fmt.Errorf("START.EXE is %d bytes, the wilderness terrain table needs %d", len(raw), end)
	}
	return ParseWildernessTerrainTable(raw[start:end])
}

// Background 重現 `1271h..12B6h`：由野外座標與模式查出 `DS:45BCh`。
//
// 原版的 X／Y 是 byte、經 `cbw` 當有號數，查表時不做邊界檢查；這裡出界就
// 報錯，不去讀表外的位元組。
func (t WildernessTerrainTable) Background(mode uint8, x, y int) (uint8, error) {
	offset, ok := wildernessSheetOffsets[mode]
	if !ok {
		return 0, fmt.Errorf("Pool combat area mode %d has no wilderness sheet", mode)
	}
	// 偏移加在 byte 上，之後才 `cbw`。
	x = int(int8(uint8(x) + uint8(offset)))
	y = int(int8(uint8(y)))
	if x < 0 || x >= WildernessTerrainWidth || y < 0 || y >= WildernessTerrainHeight {
		return 0, fmt.Errorf("Pool wilderness position (%d,%d) is outside the %d×%d terrain table",
			x, y, WildernessTerrainWidth, WildernessTerrainHeight)
	}
	return t[y*WildernessTerrainWidth+x], nil
}
