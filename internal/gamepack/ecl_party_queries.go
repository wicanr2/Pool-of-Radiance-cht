package gamepack

// 三條直接對隊伍發問、把答案寫回 ECL 變數的 opcode（spec 085）。
// 它們的共同點是不需要 UI：讀隊伍、算一個值、寫進固定或運算元指定的位址。
const (
	// FindItemOpcode 是 `32h FIND ITEM`：隊上有沒有某個型別的物品。
	FindItemOpcode = 0x32
	// FindItemOperands 是它吃幾個運算元。
	FindItemOperands = 1
	// FindItemFoundAddress 與 FindItemMissingAddress 是它寫的兩個 ECL 變數。
	// 原版先把 `6D3Eh` 起六個位元組清成 0，再把「沒找到」設成 1。
	FindItemFoundAddress   = 0x6d3e
	FindItemMissingAddress = 0x6d3f
	// FindItemClearedBytes 是被清掉的位元組數。
	FindItemClearedBytes = 6

	// PartySurpriseOpcode 是 `22h PARTY SURPRISE`。
	PartySurpriseOpcode = 0x22
	// PartySurpriseOperands 是它吃幾個運算元。
	PartySurpriseOperands = 2

	// SurpriseOpcode 是 `23h SURPRISE`。
	SurpriseOpcode = 0x23
	// SurpriseOperands 是它吃幾個運算元。
	SurpriseOperands = 4
	// SurpriseResultAddress 是它寫結果的固定位址。
	SurpriseResultAddress = 0x6dcb
	// SurpriseDie 是兩次判定用的骰子面數。
	SurpriseDie = 6
	// SurpriseBias 是兩個門檻共用的常數（`17DAh` 與 `17EEh` 的 `add 2`）。
	SurpriseBias = 2

	// CharacterClassCodeOffset 是角色記錄裡的職業碼（`+2Fh`）。
	CharacterClassCodeOffset = 0x2f
)

// SurpriseAlertClassCodes 是 `22h` 認得的職業碼。原版比對的是 4 與 0Ah，
// 而《光芒之池》的建角表根本沒有這兩個碼——它們是遊俠與牧師／遊俠，
// 只會由 `36h ADD NPC` 帶進隊伍。所以自己建的隊伍一律得到 0，那是正確結果，
// 不是「還沒接」。
var SurpriseAlertClassCodes = [...]uint8{4, 0x0a}

// PartySurpriseAlert 是 `22h` 寫進運算元 1 的值：隊上有沒有那兩個職業碼。
// 運算元 2 一律是 0（原版的第二個累加器從頭到尾沒被寫過）。
func PartySurpriseAlert(classCodes []uint8) uint8 {
	for _, code := range classCodes {
		for _, alert := range SurpriseAlertClassCodes {
			if code == alert {
				return 1
			}
		}
	}
	return 0
}

// 突襲結果碼。原版把它寫進 `DS:6DCBh`，由 ECL 自己分支。
const (
	// SurpriseNeither 是兩邊都沒被突襲。
	SurpriseNeither uint8 = 0
	// SurpriseParty 是隊伍被突襲。
	SurpriseParty uint8 = 1
	// SurpriseFoes 是對方被突襲。
	SurpriseFoes uint8 = 2
)

// SurpriseOutcome 重現 overlay-03 `1798h` 的判定。
//
// 兩個門檻的配對是原版的樣子，不是筆誤：隊伍門檻用運算元 4 減運算元 1，
// 對方門檻用運算元 2 減運算元 3，兩者都再加 2。
//
// 結果碼 3 在原版裡到不了：`1830h` 寫下 3 之後直接落進 `183Ah`，而那一段
// 在同一個條件下會把它改成 2。這裡照抄那個行為——「兩邊都被突襲」在原版
// 就是當成「對方被突襲」處理的。
func SurpriseOutcome(operands [SurpriseOperands]uint8, partyRoll, foeRoll int) uint8 {
	partyTarget := int(int8(operands[3])) + SurpriseBias - int(int8(operands[0]))
	foeTarget := int(int8(operands[1])) + SurpriseBias - int(int8(operands[2]))
	result := SurpriseNeither
	if partyRoll <= partyTarget {
		result = SurpriseParty
	}
	if foeRoll <= foeTarget {
		result = SurpriseFoes
	}
	return result
}
