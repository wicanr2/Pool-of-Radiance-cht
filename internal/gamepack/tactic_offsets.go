package gamepack

import "fmt"

// 怪物的戰術模式（spec 096）。模式存在戰鬥子結構的 `+15h`，值 1..6；
// 每個模式是一列五個**相對方向偏移**，`方向 = (基準方向 + 偏移) mod 8`。
const (
	// TacticOffsetTableAddress 是那張表在資料段的位址。
	TacticOffsetTableAddress = 0x02AC
	// TacticModes 是擲骰擲得到的模式數（`0A16h` 也是照這個循環）。
	TacticModes = 6
	// TacticSteps 是每個模式試幾個方向。呼叫端的迴圈是「步 = 1..5」，
	// 所以模式 m 用到的是 `TacticOffsetTableAddress + m*5 + 1` 起的五個 byte。
	TacticSteps = 5
	// TacticDirections 是方向的模數（八方位）。
	TacticDirections = 8
)

// TacticOffsets 是六個模式各自的五個方向偏移，索引 0 對應模式 1。
type TacticOffsets [TacticModes][TacticSteps]uint8

// ReadDOSTacticOffsets 從 `START.EXE` 的資料段取出那張表。
func ReadDOSTacticOffsets(zipPath string) (TacticOffsets, error) {
	executable, err := readStartExecutable(zipPath)
	if err != nil {
		return TacticOffsets{}, err
	}
	var offsets TacticOffsets
	for mode := 1; mode <= TacticModes; mode++ {
		for step := 1; step <= TacticSteps; step++ {
			at := TacticOffsetTableAddress + mode*TacticSteps + step + startDataSegmentFileDelta
			if at < 0 || at >= len(executable) {
				return TacticOffsets{}, fmt.Errorf(
					"Pool tactic offset for mode %d step %d is outside START.EXE", mode, step)
			}
			value := executable[at]
			if value > TacticDirections {
				return TacticOffsets{}, fmt.Errorf(
					"Pool tactic offset for mode %d step %d is %#02x, want 0..8", mode, step, value)
			}
			offsets[mode-1][step-1] = value
		}
	}
	return offsets, nil
}

// Direction 算出模式 m（1 起算）第 step 步（1 起算）要試的方向。
func (offsets TacticOffsets) Direction(mode, step int, base uint8) (uint8, error) {
	if mode < 1 || mode > TacticModes {
		return 0, fmt.Errorf("Pool tactic mode %d is outside 1..%d", mode, TacticModes)
	}
	if step < 1 || step > TacticSteps {
		return 0, fmt.Errorf("Pool tactic step %d is outside 1..%d", step, TacticSteps)
	}
	value := int(base) + int(offsets[mode-1][step-1])
	return uint8(value % TacticDirections), nil
}

// NextTacticMode 是模式試完之後換的下一個（`0A16h` 的 `(模式 mod 6) + 1`）。
func NextTacticMode(mode int) int {
	if mode < 1 || mode > TacticModes {
		return 1
	}
	return mode%TacticModes + 1
}
