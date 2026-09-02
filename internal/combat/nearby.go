package combat

import "fmt"

// NearbyRequest 是 overlay-31 `0912h` 的 14 bytes 參數（spec 056）。
// Budget 對應 `arg_8`：原版把它當成成本變數的初值傳進 `sub_419`，
// 而 `sub_419` 用它算走訪上限、又把實際成本寫回同一個變數，
// 所以它同時是預算與輸出欄位（spec 057）。
type NearbyRequest struct {
	Map     TacticalGrid
	Classes CellClasses
	Cells   []CombatantCell
	Class   uint8
	Facing  uint8
	Budget  uint16
	BaseX   uint8
	BaseY   uint8
}

// NearbyCells 重現 overlay-31 `0912h`：把 mover 的四個佔格與場上每個 combatant
// 的四個佔格兩兩比對，先過朝向弧、再走直線追蹤，取成本最小的一組寫成一筆結果。
//
// 原版不跳過 mover 自己——自己的格對自己的格在朝向弧裡無條件成立，因此 mover
// 一定會出現在結果裡，由呼叫端的陣營篩選（overlay-25 entry 32）剔除。
// 收尾的排序是 `sub_2E`。
func NearbyCells(req NearbyRequest) ([]NearbyCell, error) {
	own := FootprintCells(req.Class, req.BaseX, req.BaseY)
	results := make([]NearbyCell, 0, len(req.Cells))

	for index := 1; index < len(req.Cells); index++ {
		other := req.Cells[index]
		if other.FootprintClass == 0 {
			continue
		}
		theirs := FootprintCells(other.FootprintClass, other.X, other.Y)

		found := false
		best := uint8(0)
		var bestOwn, bestTheirs FootprintCell
		for _, from := range own {
			if !from.Valid() {
				continue
			}
			for _, to := range theirs {
				if !to.Valid() {
					continue
				}
				inArc, err := FacingArcContains(from.X, from.Y, to.X, to.Y, req.Facing)
				if err != nil {
					return nil, err
				}
				if !inArc {
					continue
				}
				trace, err := TraceMovement(req.Map, req.Classes,
					int(from.X), int(from.Y), int(to.X), int(to.Y), req.Budget)
				if err != nil {
					return nil, err
				}
				if !trace.Complete {
					continue
				}
				if found && trace.Cost >= best {
					continue
				}
				found, best, bestOwn, bestTheirs = true, trace.Cost, from, to
			}
		}
		if !found {
			continue
		}

		facing := req.Facing
		if facing >= DirectionCount {
			resolved, ok, err := RequiredFacingBetween(bestOwn, bestTheirs)
			if err != nil {
				return nil, err
			}
			if !ok {
				return nil, fmt.Errorf("Pool nearby cell for combatant %d has no facing that reaches it", index)
			}
			facing = resolved
		}
		results = append(results, NearbyCell{
			CombatantIndex: uint8(index),
			Cost:           best,
			Facing:         facing,
		})
	}

	SortNearbyCells(results)
	return results, nil
}

// RequiredFacingBetween 重現 `0912h` 結果 `+2` 的求值：朝向未指定時自 0 起遞增
// 呼叫朝向弧判定，取第一個成立的方向。方向 8 恆真，所以搜尋一定會停。
func RequiredFacingBetween(from, to FootprintCell) (uint8, bool, error) {
	for direction := uint8(0); direction <= DirectionAny; direction++ {
		inArc, err := FacingArcContains(from.X, from.Y, to.X, to.Y, direction)
		if err != nil {
			return 0, false, err
		}
		if inArc {
			return direction, true, nil
		}
	}
	return 0, false, nil
}

// NearbyReactionRange 是 overlay-13 entry 6 呼叫 overlay-25 entry 32 時給的
// 距離參數（spec 059 第 1 步）：1 就是「鄰接」。
const NearbyReactionRange = 1

// OpposingNearbyAt 依 overlay-25 entry 32 的流程，回報站在 (baseX, baseY) 的
// mover 有哪些敵對 combatant 在 range 之內。朝向一律傳未指定，與原版一致。
func OpposingNearbyAt(state TacticalState, moverIndex uint8, baseX, baseY uint8, budget uint16,
	opposingSide uint8, sideOf func(uint8) (uint8, bool)) ([]uint8, error) {
	if int(moverIndex) >= len(state.Cells) || moverIndex == 0 {
		return nil, fmt.Errorf("Pool mover index %d is outside the combatant cell table", moverIndex)
	}
	cells, err := NearbyCells(NearbyRequest{
		Map:     state.Map,
		Classes: state.Classes,
		Cells:   state.Cells,
		Class:   state.Cells[moverIndex].FootprintClass,
		Facing:  DirectionUnset,
		Budget:  budget,
		BaseX:   baseX,
		BaseY:   baseY,
	})
	if err != nil {
		return nil, err
	}
	return SelectOpposingNearby(cells, opposingSide, sideOf)
}

// LeavingOpponentsAfterStep 重現 overlay-13 entry 6 的前三步：查一次移動前的
// 鄰接敵人，把 mover 的座標暫時沿 direction 推一格再查一次，復原之後取差集。
//
// 原版推的是位置表本身、不動佔用格陣列，因為鄰近查詢只讀位置表。
func LeavingOpponentsAfterStep(state TacticalState, moverIndex uint8, direction uint8,
	opposingSide uint8, sideOf func(uint8) (uint8, bool)) ([]uint8, error) {
	if int(moverIndex) >= len(state.Cells) || moverIndex == 0 {
		return nil, fmt.Errorf("Pool mover index %d is outside the combatant cell table", moverIndex)
	}
	step, err := DirectionStep(direction)
	if err != nil {
		return nil, err
	}
	mover := state.Cells[moverIndex]

	before, err := OpposingNearbyAt(state, moverIndex, mover.X, mover.Y, NearbyReactionRange, opposingSide, sideOf)
	if err != nil {
		return nil, err
	}
	if len(before) == 0 {
		return nil, nil
	}

	state.Cells[moverIndex].X = mover.X + uint8(step.X)
	state.Cells[moverIndex].Y = mover.Y + uint8(step.Y)
	after, err := OpposingNearbyAt(state, moverIndex,
		state.Cells[moverIndex].X, state.Cells[moverIndex].Y, NearbyReactionRange, opposingSide, sideOf)
	state.Cells[moverIndex] = mover
	if err != nil {
		return nil, err
	}
	return LeavingOpponents(before, after), nil
}
