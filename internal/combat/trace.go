package combat

import "fmt"

// 每步的成本，取自 overlay-31 `02C4h` 對走訪器 +18h 的兩種累加
// （spec 057）。原版把成本記在半格單位上，所以直走是 2 而不是 1。
const (
	StraightStepCost = 2
	DiagonalStepCost = 3
)

// StepWalker 是 overlay-31 `0196h`（初始化）與 `02C4h`（前進一步）操作的走訪器，
// 對應原版 26 bytes 的 record。這是一支標準的 Bresenham：以較長的軸為主軸逐格前進，
// 誤差項非負時才在副軸上也走一格（即斜走）。
type StepWalker struct {
	StartX, StartY int // +0, +2
	GoalX, GoalY   int // +4, +6
	Error          int // +8
	DX, DY         int // +0Ah, +0Ch：兩軸的絕對差
	DiagonalDelta  int // +0Eh：斜走時的誤差增量
	StraightDelta  int // +10h：直走時的誤差增量
	SignX, SignY   int // +12h, +13h
	X, Y           int // +14h, +16h：目前位置
	Cost           uint8
	Direction      uint8 // +19h：這一步的方向索引
}

func sign(value int) int {
	switch {
	case value < 0:
		return -1
	case value > 0:
		return 1
	default:
		return 0
	}
}

// stepDirections 逐位元組照抄 DS:25D4h 的 3×3 表，索引為 (signY+1)*3 + (signX+1)。
// 值與 DS:274Ah／DS:2753h 的方向環一致，見 docs/audit/ida-ds-step-direction-table.json。
var stepDirections = [9]uint8{7, 0, 1, 6, 8, 2, 5, 4, 3}

// NewStepWalker 重現 overlay-31 `0196h`：算出兩軸差、正負號與誤差項初值，
// 並把目前位置設在起點；成本累加器歸零。
func NewStepWalker(startX, startY, goalX, goalY int) *StepWalker {
	walker := &StepWalker{
		StartX: startX, StartY: startY,
		GoalX: goalX, GoalY: goalY,
		X: startX, Y: startY,
	}
	walker.DX = abs(goalX - startX)
	walker.DY = abs(goalY - startY)
	walker.SignX = sign(goalX - startX)
	walker.SignY = sign(goalY - startY)
	major, minor := walker.DX, walker.DY
	if walker.DX <= walker.DY {
		major, minor = walker.DY, walker.DX
	}
	walker.Error = 2*minor - major
	walker.DiagonalDelta = 2 * (minor - major)
	walker.StraightDelta = 2 * minor
	return walker
}

// Step 重現 overlay-31 `02C4h`：走一格，回報是否真的走了。已經到終點時回 false
// 且不改變任何狀態（方向欄位除外，原版同樣會重算）。
func (walker *StepWalker) Step() bool {
	moved := false
	stepX, stepY := 0, 0
	if walker.DX > walker.DY {
		if walker.X != walker.GoalX {
			stepX = walker.SignX
			if walker.Error >= 0 {
				stepY = walker.SignY
				walker.Y += walker.SignY
				walker.Error += walker.DiagonalDelta
				walker.Cost += DiagonalStepCost
			} else {
				walker.Error += walker.StraightDelta
				walker.Cost += StraightStepCost
			}
			walker.X += walker.SignX
			moved = true
		}
	} else if walker.Y != walker.GoalY {
		stepY = walker.SignY
		if walker.Error >= 0 {
			stepX = walker.SignX
			walker.X += walker.SignX
			walker.Error += walker.DiagonalDelta
			walker.Cost += DiagonalStepCost
		} else {
			walker.Error += walker.StraightDelta
			walker.Cost += StraightStepCost
		}
		walker.Y += walker.SignY
		moved = true
	}
	walker.Direction = stepDirections[(sign(stepY)+1)*3+(sign(stepX)+1)]
	return moved
}

// TerrainRule 是 DS:2758h 地形表的一筆，四個欄位全數保留。
//
// EntryThreshold（+0）是 overlay-08 Move handler 的准入門檻，交給
// ResolveMovementProbe 與剩餘步數比較（spec 053）；Level（+1）與 Block（+2）
// 是 overlay-31 0419h 直線追蹤讀的兩欄（spec 057）；Field3（+3）的語意未定。
type TerrainRule struct {
	EntryThreshold uint8 // +0
	Level          uint8 // +1
	Block          uint8 // +2
	Field3         uint8 // +3
}

// TacticalGrid 是 overlay-31 `0419h` 收到的地圖：`+6` 非 0 時整段地形判定被跳過，
// 格子自 `+7` 起、列距 50。
type TacticalGrid struct {
	IgnoreTerrain bool
	Terrain       []uint8
}

// TacticalRowStride 是原版計算格子位址時用的列距（`arg_0 + y*32h + x + 7`）。
const TacticalRowStride = 0x32

// TerrainAt 依原版的定址取出一格的地形碼。
func (grid TacticalGrid) TerrainAt(x, y int) (uint8, error) {
	if x < 0 || x > TacticalMaxX || y < 0 || y > TacticalMaxY {
		return 0, fmt.Errorf("Pool tactical cell (%d,%d) is outside the board", x, y)
	}
	index := y*TacticalRowStride + x
	if index >= len(grid.Terrain) {
		return 0, fmt.Errorf("Pool tactical cell (%d,%d) is outside the supplied grid", x, y)
	}
	return grid.Terrain[index], nil
}

// TraceResult 是 TraceMovement 的結果，欄位對應原版寫回三個 var 參數的內容。
type TraceResult struct {
	X, Y     int
	Cost     uint8
	Complete bool
}

// TraceMovement 重現 overlay-31 `0419h`：由 (fromX, fromY) 沿直線走向
// (toX, toY)，每格先判地形再判預算，走得完回報 Complete。
//
// 原版的預算上限是 budget*2+1，因為成本記在半格單位上。地形判定是
// 「目的格的 Block 大於起點格的 Level 就擋住」；原版另外建了一支水平的
// 走訪器來取那個 Level，但它的兩個端點同高，逐步走訪不會改變它，
// 所以這裡直接用常數。
//
// 走不完時回傳停下來的格子與當下成本，Complete 為 false，與原版寫回
// 參數的行為一致。
func TraceMovement(grid TacticalGrid, rules []TerrainRule, fromX, fromY, toX, toY int, budget uint16) (TraceResult, error) {
	lookup := func(x, y int) (TerrainRule, error) {
		code, err := grid.TerrainAt(x, y)
		if err != nil {
			return TerrainRule{}, err
		}
		if int(code) >= len(rules) {
			return TerrainRule{}, fmt.Errorf("Pool terrain code %d at (%d,%d) has no rule", code, x, y)
		}
		return rules[code], nil
	}

	start, err := lookup(fromX, fromY)
	if err != nil {
		return TraceResult{}, err
	}
	limit := int(budget)*2 + 1
	walker := NewStepWalker(fromX, fromY, toX, toY)
	for {
		if !grid.IgnoreTerrain {
			here, err := lookup(walker.X, walker.Y)
			if err != nil {
				return TraceResult{}, err
			}
			if here.Block > start.Level {
				return TraceResult{X: walker.X, Y: walker.Y, Cost: walker.Cost}, nil
			}
		}
		if int(walker.Cost) > limit {
			return TraceResult{X: walker.X, Y: walker.Y, Cost: walker.Cost}, nil
		}
		if !walker.Step() {
			return TraceResult{X: walker.X, Y: walker.Y, Cost: walker.Cost, Complete: true}, nil
		}
	}
}
