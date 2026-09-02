package gamepack

// `34h ECL CLOCK`（spec 093）：把遊戲時鐘往前推。
//
// 時鐘是七個 word（overlay-20 entry 2 從 `[4933h] + 6E00h` 起搬七個進區域
// 變數再寫回去）。推進的方式是「對指定的那一格加一，然後跑一次進位」，
// 重複 count 次——不是直接加 count，因為每加一次都要讓進位往上帶。
const (
	// ECLClockOpcode 是這條 opcode。
	ECLClockOpcode = 0x34
	// ECLClockFields 是時鐘的欄位數。
	ECLClockFields = 7
	// ECLClockAdvancedField 是 `34h` 推進的那一格。呼叫端固定傳 1
	//（overlay-03 `2E38h` 推的常數）。
	ECLClockAdvancedField = 1
)

// ECLClock 是七個欄位。語意（回合／輪／小時…）還沒讀出來，所以不取名。
type ECLClock [ECLClockFields]uint16

// AdvanceECLClock 對第 field 格加 count 次一，每次之後由 carry 帶進位。
// carry 是 overlay-20 `02B1h` 的那一支，語意還沒讀，所以由呼叫端傳進來；
// 傳 nil 就只加不進位。
func AdvanceECLClock(clock ECLClock, field int, count int, carry func(*ECLClock)) ECLClock {
	if field < 0 || field >= ECLClockFields {
		return clock
	}
	for step := 0; step < count; step++ {
		clock[field]++
		if carry != nil {
			carry(&clock)
		}
	}
	return clock
}
