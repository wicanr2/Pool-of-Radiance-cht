package combat

import "fmt"

// 目的格類別的兩個特殊值（spec 058）。
const (
	// DefaultDestinationClass 是 overlay-32 `0CB9h` 進入時寫下的初值。
	// 四個佔格都沒有更新它時，回報的就是這個值。
	DefaultDestinationClass = 0x17
	// OffBoardDestinationClass 代表目的格在盤面外；它一旦出現就蓋掉其他類別。
	OffBoardDestinationClass = 0x00
	// StickyDestinationClass 是原版另外特別處理的類別，同樣一出現就蓋掉其他值。
	StickyDestinationClass = 0x1E
)

// CombatantCell 是 DS:5E89h 起的一筆戰場位置，4 bytes。索引 1-based。
type CombatantCell struct {
	X              uint8 // +0
	Y              uint8 // +1
	Field2         uint8 // +2，語意未定
	FootprintClass uint8 // +3，0 代表該筆不參與
}

// TacticalState 蒐集戰術層讀寫的幾塊原版 DS 資料。
//
// Map 是 DS:6674h 遠指標指到的地圖；Occupancy 是 DS:6039h 起、50 寬的佔用格
// 陣列，值是 combatant 索引，0 表示空；Cells 對應 DS:5E89h 起的位置表，
// 索引 1-based，因此第 0 筆保留不用。
type TacticalState struct {
	Map       TacticalGrid
	Occupancy []uint8
	Cells     []CombatantCell
	Classes   CellClasses
}

// CellAt 重現 overlay-32 `04C0h`：盤面外時兩個輸出都是 0，否則回報該格的
// 佔用者索引與地形類別。
func (state TacticalState) CellAt(x, y uint8) (occupant uint8, class uint8, err error) {
	if !withinTactical(x, y) {
		return 0, 0, nil
	}
	class, err = state.Map.TerrainAt(int(x), int(y))
	if err != nil {
		return 0, 0, err
	}
	index := int(y)*TacticalRowStride + int(x)
	if index >= len(state.Occupancy) {
		return 0, 0, fmt.Errorf("Pool occupancy grid has no cell (%d,%d)", x, y)
	}
	return state.Occupancy[index], class, nil
}

// ProbeDestination 重現 overlay-32 `0CB9h`：把 mover 的每一個佔格都往 direction
// 推一格，看看落在哪裡。
//
// 回傳的 target 是撞到的 combatant 索引（撞到自己不算），class 是所有目的格裡
// 最「難進」的那一個——盤面外與 StickyDestinationClass 一出現就定案，其餘則比
// 地形表的 EntryThreshold 取大者。mover 的體型類別不在表內時四個槽全部略過，
// 於是 class 維持 DefaultDestinationClass，與原版一致。
//
// 兩個回傳值正是 ResolveMovementProbe 需要的 attackTargetID 與目的格類別。
func ProbeDestination(state TacticalState, moverIndex uint8, direction uint8) (target uint8, class uint8, err error) {
	if int(moverIndex) >= len(state.Cells) || moverIndex == 0 {
		return 0, 0, fmt.Errorf("Pool mover index %d is outside the combatant cell table", moverIndex)
	}
	step, err := DirectionStep(direction)
	if err != nil {
		return 0, 0, err
	}
	mover := state.Cells[moverIndex]
	class = DefaultDestinationClass
	best := uint8(1)

	offsets, err := FootprintOffsets(mover.FootprintClass)
	if err != nil {
		return 0, class, nil
	}
	for _, offset := range offsets {
		x := mover.X + uint8(offset.X) + uint8(step.X)
		y := mover.Y + uint8(offset.Y) + uint8(step.Y)
		occupant, cellClass, err := state.CellAt(x, y)
		if err != nil {
			return 0, 0, err
		}
		if occupant == moverIndex {
			occupant = 0
		}
		if occupant > 0 {
			target = occupant
		}
		if cellClass == OffBoardDestinationClass || class == OffBoardDestinationClass {
			class = OffBoardDestinationClass
			continue
		}
		if cellClass == StickyDestinationClass || class == StickyDestinationClass {
			class = StickyDestinationClass
			continue
		}
		record, err := CellClassAt(state.Classes, cellClass)
		if err != nil {
			return 0, 0, err
		}
		if record.EntryThreshold < best {
			continue
		}
		best = record.EntryThreshold
		class = cellClass
	}
	return target, class, nil
}

// ResolveDestination 把 ProbeDestination 的兩個 byte 接到 ResolveMovementProbe：
// 類別 0 代表目的格在盤面外，原版此時不是擋住而是問玩家要不要離開戰鬥，
// 所以這裡分成獨立的一個結果，不混進 MovementBlocked。
func ResolveDestination(state TacticalState, moverIndex uint8, direction uint8, budget uint8) (MovementProbeAction, bool, error) {
	target, class, err := ProbeDestination(state, moverIndex, direction)
	if err != nil {
		return MovementBlocked, false, err
	}
	if target != 0 {
		return MovementAttack, false, nil
	}
	if class == OffBoardDestinationClass {
		return MovementBlocked, true, nil
	}
	cellClass, err := CellClassAt(state.Classes, class)
	if err != nil {
		return MovementBlocked, false, err
	}
	return ResolveMovementProbe(budget, 0, cellClass.EntryThreshold), false, nil
}
