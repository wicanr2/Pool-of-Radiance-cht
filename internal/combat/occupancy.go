package combat

import "fmt"

// NearbyCell 是原版 DS:6674h 結構的一筆，共三個 byte，順序與原版一致
// （spec 056）：+0 combatant 索引、+1 成本、+2 可及級數。
type NearbyCell struct {
	CombatantIndex uint8
	Cost           uint8
	ReachClass     uint8
}

// SelectOpposingNearby 重現 overlay-25 entry 32（`246Dh..2591h`）的篩選與輸出：
// 只留下 combatant record +10Eh 等於 mover 對立值者，命中者依原順序往前壓縮，
// 匯出的是 combatant index 陣列。
//
// sideOf 由呼叫端提供，對應原版經 DS:6517h far pointer 表讀取 record +10Eh；
// 查不到的 index 一律失敗即關閉，不當成不相符默默略過。
func SelectOpposingNearby(cells []NearbyCell, opposingSide uint8, sideOf func(combatantIndex uint8) (uint8, bool)) ([]uint8, error) {
	if sideOf == nil {
		return nil, fmt.Errorf("Pool nearby selection needs a combatant side lookup")
	}
	selected := make([]uint8, 0, len(cells))
	for position, cell := range cells {
		side, ok := sideOf(cell.CombatantIndex)
		if !ok {
			return nil, fmt.Errorf("Pool nearby cell %d references combatant index %d, which has no record",
				position+1, cell.CombatantIndex)
		}
		if side == opposingSide {
			selected = append(selected, cell.CombatantIndex)
		}
	}
	return selected, nil
}

// CompactOpposingNearby 重現同一函式對 DS:6674h 結構本身的原地壓縮：命中者依序
// 搬到表首，筆數改為命中數。回傳的切片與輸入共用底層陣列，與原版一致。
func CompactOpposingNearby(cells []NearbyCell, opposingSide uint8, sideOf func(combatantIndex uint8) (uint8, bool)) ([]NearbyCell, error) {
	if sideOf == nil {
		return nil, fmt.Errorf("Pool nearby compaction needs a combatant side lookup")
	}
	kept := 0
	for position, cell := range cells {
		side, ok := sideOf(cell.CombatantIndex)
		if !ok {
			return nil, fmt.Errorf("Pool nearby cell %d references combatant index %d, which has no record",
				position+1, cell.CombatantIndex)
		}
		if side != opposingSide {
			continue
		}
		cells[kept] = cell
		kept++
	}
	return cells[:kept], nil
}

// FootprintClassCount 是 DS:2860h 佔格偏移表的列數，FootprintSlots 是每列的槽數。
// 兩者都由原始資料決定，不得自行擴充（spec 056 READY 契約 1）。
const (
	FootprintClassCount = 4
	FootprintSlots      = 4
)

// footprintTable 逐位元組照抄 START.EXE 的 DS:2860h..287Fh，
// 見 docs/audit/ida-ds-footprint-offset-table.json。列為體型類別 1..4，
// 每列四組 (分量 A, 分量 B) 有號偏移，分量 A 為負代表該列到此為止。
var footprintTable = [FootprintClassCount][FootprintSlots * 2]byte{
	{0x00, 0x00, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
	{0x00, 0x00, 0x00, 0x01, 0xFF, 0xFF, 0xFF, 0xFF},
	{0x00, 0x00, 0x01, 0x00, 0xFF, 0xFF, 0xFF, 0xFF},
	{0x00, 0x00, 0x01, 0x00, 0x00, 0x01, 0x01, 0x01},
}

// FootprintOffset 是佔格偏移表的一組偏移。分量 A 對應 combatant record +0，
// 分量 B 對應 +1；兩者何者為 X 尚未閉合（spec 056）。
type FootprintOffset struct {
	A int8
	B int8
}

// FootprintOffsets 回傳體型類別的有效偏移，順序與原始表相同。
// 類別 0 在原版是「查詢直接失敗」，此處與超出表範圍的類別一樣視為錯誤。
func FootprintOffsets(class uint8) ([]FootprintOffset, error) {
	if class == 0 || int(class) > FootprintClassCount {
		return nil, fmt.Errorf("Pool footprint class %d is outside the original table (1..%d)",
			class, FootprintClassCount)
	}
	row := footprintTable[class-1]
	offsets := make([]FootprintOffset, 0, FootprintSlots)
	for slot := 0; slot < FootprintSlots; slot++ {
		a := int8(row[slot*2])
		if a < 0 {
			break
		}
		offsets = append(offsets, FootprintOffset{A: a, B: int8(row[slot*2+1])})
	}
	return offsets, nil
}

// FootprintCell 是展開到基準座標之後的一格。原版以「分量 A 的最高位元被設起」
// 表示無效，且無效時不會寫分量 B，因此這裡也只用分量 A 判定。
type FootprintCell struct {
	A uint8
	B uint8
}

// Valid 重現原版的 `cmp ...,0 / jl`：分量 A 以有號位元組看為負即無效。
// 這同時涵蓋「表已終止」與「加總後越過 127」兩種情形，與原版一致。
func (cell FootprintCell) Valid() bool { return int8(cell.A) >= 0 }

const footprintInvalid = 0xFF

// FootprintCells 重現 overlay-31 `0912h` 的展開：固定四個槽，逐槽查表，
// 有效者把偏移以位元組運算加到基準座標，無效者寫 0FFh 佔位而不是略過——
// 槽的位置本身要保留（spec 056 READY 契約 2）。
//
// class 為 0 時四個槽全部無效，對應原版「體型類別 0 的 combatant 不參與」。
func FootprintCells(class uint8, baseA, baseB uint8) [FootprintSlots]FootprintCell {
	var cells [FootprintSlots]FootprintCell
	for slot := range cells {
		cells[slot] = FootprintCell{A: footprintInvalid, B: footprintInvalid}
	}
	if class == 0 || int(class) > FootprintClassCount {
		return cells
	}
	row := footprintTable[class-1]
	for slot := 0; slot < FootprintSlots; slot++ {
		a := int8(row[slot*2])
		if a < 0 {
			continue
		}
		cells[slot] = FootprintCell{
			A: baseA + uint8(a),
			B: baseB + uint8(int8(row[slot*2+1])),
		}
	}
	return cells
}
