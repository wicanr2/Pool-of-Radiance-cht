package gamepack

import "fmt"

// `1Eh CHECKPARTY`（spec 092）：對整隊算一個統計，把最小值、最大值、平均與
// 一個旗標寫進運算元 3..6。模式由運算元 1 決定。
const (
	// CheckPartyOpcode 是這條 opcode。
	CheckPartyOpcode = 0x1e
	// CheckPartyOperands 是它吃幾個運算元。
	CheckPartyOperands = 6

	// CheckPartyEffectMode 是運算元 1 為字面值 0 時的模式：問隊上有沒有人
	// 掛著運算元 2 那個效果碼。
	CheckPartyEffectMode = 0

	// CheckPartyFieldSeventyNine 是運算元 1 指到 `DS:6BA7h` 時要統計的欄位
	//（記錄 `+79h`）。那個欄位的語意還沒讀出來。
	CheckPartyFieldSeventyNine = 0x6ba7
	// CheckPartyFieldMovement 是運算元 1 指到 `DS:6C1Bh` 時要統計的欄位
	//（記錄 `+11Ch`，spec 079 的移動力）。
	CheckPartyFieldMovement = 0x6c1b

	// CheckPartyInitialMinimum 是最小值的初始值（`15A8h` 的 FFh）。
	CheckPartyInitialMinimum = 0xff
)

// CheckPartyStats 是要寫回運算元 3..6 的四個值。
type CheckPartyStats struct {
	Minimum uint8
	Maximum uint8
	Average uint8
	Flag    uint8
}

// CheckPartySummary 統計一組值：最小、最大、平均（總和整除人數）。
// 初始值照原版——最小值從 FFh 開始、最大值從 0 開始，所以空隊伍會原樣寫回去。
func CheckPartySummary(values []uint8) CheckPartyStats {
	stats := CheckPartyStats{Minimum: CheckPartyInitialMinimum}
	total, count := 0, 0
	for _, value := range values {
		if value < stats.Minimum {
			stats.Minimum = value
		}
		if value > stats.Maximum {
			stats.Maximum = value
		}
		total += int(value)
		count++
	}
	if count != 0 {
		stats.Average = uint8(total / count)
	}
	return stats
}

// CheckPartyEffectPresent 是效果模式的結果：隊上任何一個人的效果串列
//（spec 069，記錄 `+7Fh` 起、節點 `+0` 是效果碼、`+5` 是下一個）帶著那個碼
// 就是 1。原版找到第一個就停。
func CheckPartyEffectPresent(effects [][]uint8, wanted uint8) uint8 {
	for _, member := range effects {
		for _, code := range member {
			if code == wanted {
				return 1
			}
		}
	}
	return 0
}

// CheckPartyMode 依運算元 1 的位址（或字面值 0）說出這是哪一種模式。
// 讀不到的位址回錯誤——猜一個欄位去統計，結果會是一個看起來合理的數字。
func CheckPartyMode(literal bool, address uint16) (uint16, error) {
	if literal {
		return CheckPartyEffectMode, nil
	}
	switch address {
	case CheckPartyFieldSeventyNine, CheckPartyFieldMovement:
		return address, nil
	}
	return 0, fmt.Errorf("Pool CHECKPARTY selector %#04x is not one of the two known fields", address)
}
