package combat

import "fmt"

// NearbyCell 是原版 DS:6674h 鄰近格位表的一筆，共三個 byte。第三個 byte 是
// combatant index，前兩個 byte 的語意尚未閉合（spec 056），因此保留 raw。
type NearbyCell struct {
	First          uint8
	Second         uint8
	CombatantIndex uint8
}

// SelectOpposingNearby 重現 overlay-25 entry 32（`2465h..2591h`）的篩選與輸出：
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

// CompactOpposingNearby 重現同一函式對 DS:6674h 表本身的原地壓縮：命中者依序
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
