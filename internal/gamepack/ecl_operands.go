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
	// ProgramCamp（值 9）是**旅店的過夜**。全遊戲只有一處推它：
	// `ecl3/0` 的 `A1ADh`，就在旅店收掉一枚白金之後（`A1B4h WHO
	// 'WHO WILL PAY?'`）。派發鏈的第一支是 overlay-15 entry 1——
	// overlay-15 整支都在動記憶法術陣列（spec 070）——回傳非 0 才接
	// overlay-25 entry 37。所以值 9 是「選法術＋休息」，不是隊伍管理。
	ProgramCamp = 9
	// ProgramEnding（值 8）是結局過場（overlay-18 entry 1，spec 108）。
	// 唯一的呼叫點是 `ECL5/7` 的 `A82Ah`——打贏泰倫斯拉克斯之後。
	ProgramEnding = 8

	// AddNPCOpcode 是 `36h ADD NPC`（spec 091）。
	AddNPCOpcode = 0x36
	// AddNPCOperands 是它吃幾個運算元。
	AddNPCOperands = 2
	// AddNPCHostileID 是唯一會站到對面的 NPC 編號（overlay-03 `2F2Ah`
	// 只比這一個值）。
	AddNPCHostileID = 0x18

	// WhoOpcode 是 `39h WHO`（spec 083／090）：讓玩家挑一個隊伍成員，
	// 挑到的那個存進「目前角色」槽（原版的 `DS:5CF0h`）。
	WhoOpcode = 0x39
	// WhoOperands 是它吃幾個運算元。運算元 1 被取出來但沒看到被讀。
	WhoOperands = 1

	// ProtectionOpcode 是 `3Ch PROTECTION`（spec 089）：把運算元 1 位址起的
	// 連續 ECL 變數印成一列數字。
	ProtectionOpcode = 0x3c
	// ProtectionOperands 是它吃幾個運算元。
	ProtectionOperands = 1
	// ProtectionMaxEntries 是 remake 這一側的安全上限。原版沒有上限，
	// 靠「遇到 0 就停」收尾；記憶體是 map 的話讀不到的位址回 0，一樣會停，
	// 但留一個上限免得資料壞掉時無限印。
	ProtectionMaxEntries = 64

	// InputNumberOpcode 是 `0Fh INPUT NUMBER`（spec 087）。
	InputNumberOpcode = 0x0f
	// InputStringOpcode 是 `10h INPUT STRING`（spec 087）。
	InputStringOpcode = 0x10
	// InputOperands 是兩條輸入 opcode 各吃幾個運算元。
	InputOperands = 2
	// InputDestinationOperand 是「寫進哪個 ECL 變數」那個運算元的序號。
	InputDestinationOperand = 2
	// InputStringMaxLength 是 `10h` 收的最長字元數（`0991h` 推的 28h）。
	InputStringMaxLength = 0x28
	// InputStringEmptyReplacement 是空字串會被換成什麼（`09AFh` 的 `cs:95Eh`
	// 是長度 1 的一個空白）。
	InputStringEmptyReplacement = " "

	// ParlayOpcode 是 `2Ch PARLAY`（spec 086）：五種語氣的交涉選單。
	ParlayOpcode = 0x2c
	// ParlayOperands 是它吃幾個運算元。
	ParlayOperands = 6
	// ParlayOutcomeCount 是結果表的格數，等於選項數。
	ParlayOutcomeCount = 5
	// ParlayResultOperand 是「結果寫進哪個 ECL 變數」那個運算元的序號。
	ParlayResultOperand = 6

	// SaveOpcode 是 `09h SAVE`：把運算元 1 的值寫進運算元 2 指的位址。
	// 索寇要塞的密碼先用它把字面存進字串變數，再拿變數去比對。
	SaveOpcode = 0x09
	// CompareOpcode 是 `03h COMPARE`：把兩個運算元比一比，結果供 `16h IF =`
	// 之類使用。輸入密碼那一段用它拿玩家打的字對原版寫死的字面。
	CompareOpcode = 0x03
	// PrintOpcode 是 `11h PRINT`（spec 082）：**接著印**，不清框。
	// 原版靠它把一句話拼起來，例如 ecl7/23 的密碼確認框是
	// `12h "DO YOU REALLY MEAN"` ＋ `11h <玩家打的字>` ＋ `11h "?"`。
	PrintOpcode = 0x11
	// PrintReturnOpcode 是 `33h PRINT RETURN`（spec 082）：文字框換行。
	PrintReturnOpcode = 0x33
	// ClearBoxOpcode 是 `3Dh CLEAR BOX`（spec 082）：清掉文字框。
	ClearBoxOpcode = 0x3D
	// PrintClearOpcode 是 `12h PRINTCLEAR`（spec 082）：清框再印，也就是
	// 新的一頁。
	PrintClearOpcode = 0x12
	// ApproachOpcode 是 `0Dh APPROACH`（spec 117）：把接近距離減一再重畫
	// 那張半身像。**只是重畫，不等玩家**——overlay-03 `07E1h` 做完就返回。
	ApproachOpcode = 0x0D
	// PictureOpcode 是 `0Eh PICTURE`（spec 117）：換一張圖，同樣不等玩家
	// （overlay-03 `0822h`）。
	PictureOpcode = 0x0E
)

// EncounterMenuChoices 是選單的四個選項，順序即畫面順序。第四項依情境在
// PARLAY 與 ADVANCE 之間切換（overlay-03 `2333h`）。
var EncounterMenuChoices = []string{"COMBAT", "WAIT", "FLEE", "PARLAY"}

// AddNPCSide 是 `36h` 寫進記錄 `+10Eh` 的陣營：只有編號 18h 得到 1，
// 其餘都是 0（overlay-03 `2F2Ah` 的單一比較）。
func AddNPCSide(id uint8) uint8 {
	if id == AddNPCHostileID {
		return 1
	}
	return 0
}

// ParlayChoices 是交涉選單的五種語氣，順序即畫面順序，也就是結果表
// 運算元 1..5 的索引。字面值來自 overlay-03 `2785h` 的
// `~HAUGHTY ~SLY ~NICE ~MEEK ~ABUSIVE`（`~` 標的是熱鍵字母，與 spec 078
// 的選單同一套寫法）。
var ParlayChoices = []string{"HAUGHTY", "SLY", "NICE", "MEEK", "ABUSIVE"}

// EncounterMenuAdvanceChoices 是第四項換成 ADVANCE 的那一版。
var EncounterMenuAdvanceChoices = []string{"COMBAT", "WAIT", "FLEE", "ADVANCE"}
