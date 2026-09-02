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

	// ProgramOpcode 是 `38h PROGRAM`（spec 081）：用一個運算元選一支子程式。
	ProgramOpcode = 0x38
	// ProgramOperands 是它吃幾個運算元。
	ProgramOperands = 1
	// ProgramPartyManagement（值 0）直接開城裡的隊伍管理畫面
	// （overlay-16 entry 1 加 overlay-25 entry 37）。
	ProgramPartyManagement = 0
	// ProgramUnusedEight（值 8）走 overlay-18 entry 1；全遊戲沒有呼叫點。
	ProgramUnusedEight = 8
	// ProgramAskThenManage（值 9）先問一句，答應才開同一個畫面，然後結束
	// 這個 block（overlay-03 `312Ah`）。
	ProgramAskThenManage = 9

	// ParlayOpcode 是 `2Ch PARLAY`（spec 086）：五種語氣的交涉選單。
	ParlayOpcode = 0x2c
	// ParlayOperands 是它吃幾個運算元。
	ParlayOperands = 6
	// ParlayOutcomeCount 是結果表的格數，等於選項數。
	ParlayOutcomeCount = 5
	// ParlayResultOperand 是「結果寫進哪個 ECL 變數」那個運算元的序號。
	ParlayResultOperand = 6

	// PrintReturnOpcode 是 `33h PRINT RETURN`（spec 082）：文字框換行。
	PrintReturnOpcode = 0x33
	// ClearBoxOpcode 是 `3Dh CLEAR BOX`（spec 082）：清掉文字框。
	ClearBoxOpcode = 0x3D
)

// EncounterMenuChoices 是選單的四個選項，順序即畫面順序。第四項依情境在
// PARLAY 與 ADVANCE 之間切換（overlay-03 `2333h`）。
var EncounterMenuChoices = []string{"COMBAT", "WAIT", "FLEE", "PARLAY"}

// ParlayChoices 是交涉選單的五種語氣，順序即畫面順序，也就是結果表
// 運算元 1..5 的索引。字面值來自 overlay-03 `2785h` 的
// `~HAUGHTY ~SLY ~NICE ~MEEK ~ABUSIVE`（`~` 標的是熱鍵字母，與 spec 078
// 的選單同一套寫法）。
var ParlayChoices = []string{"HAUGHTY", "SLY", "NICE", "MEEK", "ABUSIVE"}

// EncounterMenuAdvanceChoices 是第四項換成 ADVANCE 的那一版。
var EncounterMenuAdvanceChoices = []string{"COMBAT", "WAIT", "FLEE", "ADVANCE"}
