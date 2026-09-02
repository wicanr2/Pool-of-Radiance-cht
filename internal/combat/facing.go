package combat

import "fmt"

// 戰術格座標的合法範圍，取自 overlay-31 `0579h` 進入時的四道有號位元組界限
// （spec 056）。X 軸較寬，Y 軸較窄。
const (
	TacticalMaxX = 0x31
	TacticalMaxY = 0x18
)

// DirectionCount 是 DS:274Ah／DS:2753h 兩張表的方向數，DirectionAny 是表的第九項，
// 代表「不指定方向」。原版另以 0FFh 表示同一件事。
const (
	DirectionCount = 8
	DirectionAny   = 8
	DirectionUnset = 0xFF
)

// directionSteps 逐位元組照抄 DS:274Ah（X 位移）與 DS:2753h（Y 位移），
// 見 docs/audit/ida-ds-direction-step-table.json。0 為上，順時針到 7。
var directionSteps = [DirectionCount + 1]FootprintOffset{
	{X: 0, Y: -1},
	{X: 1, Y: -1},
	{X: 1, Y: 0},
	{X: 1, Y: 1},
	{X: 0, Y: 1},
	{X: -1, Y: 1},
	{X: -1, Y: 0},
	{X: -1, Y: -1},
	{X: 0, Y: 0},
}

// DirectionStep 回傳方向的單格位移。
func DirectionStep(direction uint8) (FootprintOffset, error) {
	if int(direction) > DirectionAny {
		return FootprintOffset{}, fmt.Errorf("Pool facing direction %d is outside the original table (0..%d)",
			direction, DirectionAny)
	}
	return directionSteps[direction], nil
}

func withinTactical(x, y uint8) bool {
	return int8(x) >= 0 && int8(x) <= TacticalMaxX && int8(y) >= 0 && int8(y) <= TacticalMaxY
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

// FacingArcContains 重現 overlay-31 `0579h`：由 (fromX, fromY) 朝 direction 看出去，
// (toX, toY) 是否落在攻擊弧內。
//
// 原版的流程是：先以有號位元組檢查兩組座標都在盤面內，否則直接回 false；
// 把 0FFh 的 direction 換成 DirectionAny；以 direction 的單格位移求出弧的頂點；
// 頂點或起點本身即為目標時無條件成立；其餘依 direction 分支——
// 四個正向是以頂點為頂的 90 度錐形，四個對角是以頂點為原點的象限，
// DirectionAny 恆真。
//
// direction 超出 0..8 在原版會讀到未初始化的回傳值，這裡改為失敗即關閉。
func FacingArcContains(fromX, fromY, toX, toY uint8, direction uint8) (bool, error) {
	if !withinTactical(fromX, fromY) || !withinTactical(toX, toY) {
		return false, nil
	}
	if direction == DirectionUnset {
		direction = DirectionAny
	}
	step, err := DirectionStep(direction)
	if err != nil {
		return false, err
	}
	apexX := int8(fromX + uint8(step.X))
	apexY := int8(fromY + uint8(step.Y))
	if fromX == toX && fromY == toY {
		return true, nil
	}
	if apexX == int8(toX) && apexY == int8(toY) {
		return true, nil
	}

	// u 為目標在 X 軸上超出頂點的量，v 為目標在頂點「上方」的量。
	u := int(int8(toX)) - int(apexX)
	v := int(apexY) - int(int8(toY))
	switch direction {
	case 0:
		return v >= abs(u), nil
	case 1:
		return u >= 0 && v >= 0, nil
	case 2:
		return u >= abs(v), nil
	case 3:
		return u >= 0 && v <= 0, nil
	case 4:
		return -v >= abs(u), nil
	case 5:
		return u <= 0 && v <= 0, nil
	case 6:
		return -u >= abs(v), nil
	case 7:
		return u <= 0 && v >= 0, nil
	default:
		return true, nil
	}
}

// RequiredFacing 重現 overlay-31 `0912h` 決定結果第三個 byte 的方式：指定的方向
// 小於 DirectionAny 時直接沿用，否則自 0 起遞增取第一個成立的方向。
// DirectionAny 恆真，因此搜尋一定會停。
func RequiredFacing(fromX, fromY, toX, toY uint8, direction uint8) (uint8, error) {
	if direction < DirectionAny {
		return direction, nil
	}
	for candidate := uint8(0); candidate <= DirectionAny; candidate++ {
		inside, err := FacingArcContains(fromX, fromY, toX, toY, candidate)
		if err != nil {
			return 0, err
		}
		if inside {
			return candidate, nil
		}
	}
	return 0, fmt.Errorf("Pool facing search from (%d,%d) to (%d,%d) exhausted every direction",
		fromX, fromY, toX, toY)
}
