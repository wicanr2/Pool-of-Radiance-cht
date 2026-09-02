package gamepack

// ECL 直譯器把每個運算元拆成三個平行陣列存（overlay-07 entry 2，code `0000h`）。
// 取運算元的常式對每個運算元 `i`（1 起算）寫三個地方，然後 PC 前進；
// 型別是 1、2、3 時才多讀一個高位元組。
//
// `0045h:0025h` 只回傳低位元組，所以需要 16 位值的 opcode 會自己把高低兩個
// 陣列合起來——`29h ENCOUNTER MENU` 的運算元 4（結果要寫進哪個 ECL 變數）
// 就是這樣讀的（spec 078）。
const (
	// ECLProgramCounterAddress 是 ECL 的位元組 PC。
	ECLProgramCounterAddress = 0x494e
	// ECLOperandTypeBase 是型別陣列的基底，運算元 i 在 `+i`。
	ECLOperandTypeBase = 0x6dcb
	// ECLOperandHighBase 是高位元組陣列的基底。
	ECLOperandHighBase = 0x6e0b
	// ECLOperandLowBase 是低位元組陣列的基底。
	ECLOperandLowBase = 0x6e4b

	// ECLOperandWideTypeMax 是「後面還有高位元組」的最大型別碼；
	// 型別 1..3 才是 16 位的值。
	ECLOperandWideTypeMax = 3

	// EncounterMenuOpcode 是遭遇選單。
	EncounterMenuOpcode = 0x29
	// EncounterMenuOperands 是它吃幾個運算元。
	EncounterMenuOperands = 14
	// EncounterMenuResultOperand 是「結果寫進哪個 ECL 變數」那個運算元的序號。
	EncounterMenuResultOperand = 4
	// EncounterMenuOutcomeFirstOperand 是五格結果表的第一個運算元序號；
	// 表用選單的選擇當索引。
	EncounterMenuOutcomeFirstOperand = 5
	// EncounterMenuOutcomeCount 是結果表的格數。
	EncounterMenuOutcomeCount = 5
)

// EncounterMenuChoices 是選單的四個選項，順序即畫面順序。第四項依情境在
// PARLAY 與 ADVANCE 之間切換（overlay-03 `2333h`）。
var EncounterMenuChoices = []string{"COMBAT", "WAIT", "FLEE", "PARLAY"}

// EncounterMenuAdvanceChoices 是第四項換成 ADVANCE 的那一版。
var EncounterMenuAdvanceChoices = []string{"COMBAT", "WAIT", "FLEE", "ADVANCE"}
