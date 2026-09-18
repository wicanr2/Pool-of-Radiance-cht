package gamepack

import (
	"slices"
	"strings"

	"github.com/wicanr2/golden-box-remake-engine/ecl"
)

// InputAnswer 找出一次 `10h INPUT STRING`（spec 087）之後拿來比對的字面，也就是
// 玩家該打進去的密語。原版有兩種寫法：
//
//	ecl7/23  A4A1 INPUT STRING #6 6E79h
//	         A4CA COMPARE 6E79h "NOKNOK"        ← 直接比字面
//
//	ecl4/21  9E80 INPUT STRING #7 982Ch
//	         9E8D SAVE "SAMOSUD" 9890h          ← 先存進字串變數
//	         9E9A SAVE "SHESTNI" 9890h          ← 依 4A26h 二選一
//	         9EA6 COMPARE 982Ch 9890h           ← 比的是變數
//
// 所以掃的時候順便記下 `09h SAVE <字面> → <位址>`，遇到比對變數的 `COMPARE`
// 就把該位址收到的字面全部拿出來。同一個位址被寫過兩次時兩個都列——玩家當下
// 需要哪一個，靜態分不出來，不猜。
//
// 掃描**跟著控制流走**：碰到 `01h GOTO` 就跳過去，碰到 `00h EXIT`／`13h RETURN`
// 就停。直線往下讀會走進不相干的下一段——巨人的「打什麼都錯」那一處
// （ecl5/block 5 `9E08h`，後面接 `RANDOM` 挑一句罵人的話再 `GOTO`）因此撿到隔壁
// 那一處的 `TYRANTHRAXUS`，而那個假答案在遊戲裡長得跟真答案一模一樣。
//
// decode 要用 Pool 自己的指令表（`PoolCommandTable`）；退回 engine 底稿時
// `34h` 的運算元個數不同，掃過它之後的每一條都會錯位。只往後掃 scanLimit 條，
// 找不到就回空字串。
func InputAnswer(decode func(offset int) (ecl.Instruction, error), pc int, address uint16) string {
	const scanLimit = 64
	saved := map[uint16][]string{}
	// conditional 記「這一條是不是 `IF`」；下一圈的 skippable 就是它，代表那一條會不會被跳過。
	conditional := false
	for offset, scanned := pc, 0; scanned < scanLimit; scanned++ {
		instruction, err := decode(offset)
		if err != nil {
			return ""
		}
		operands := instruction.Operands
		skippable := conditional
		conditional = instruction.Command.Opcode >= FirstIfOpcode &&
			instruction.Command.Opcode <= LastIfOpcode
		switch {
		case instruction.Command.Opcode == SaveOpcode && len(operands) == 2 && ecl.IsText(operands[0]):
			destination, err := ecl.WordAddress(operands[1])
			if err != nil {
				break
			}
			value, err := ecl.TextValue(operands[0], nil)
			if err != nil {
				break
			}
			if value = strings.TrimSpace(value); value != "" && !slices.Contains(saved[destination], value) {
				saved[destination] = append(saved[destination], value)
			}
		case instruction.Command.Opcode == CompareOpcode && len(operands) == 2:
			// 兩個運算元哪一個是輸入的變數不固定：索寇要塞寫成
			// `COMPARE 982Ch "LUX"`，而石像鬼口令寫成 `COMPARE "HARASH" 9836h`。
			// 只認一種寫法會漏掉一半以上的密語，而漏掉長得像「這一處沒有答案」。
			for _, order := range [2][2]int{{0, 1}, {1, 0}} {
				compared, err := ecl.WordAddress(operands[order[0]])
				if err != nil || compared != address {
					continue
				}
				other := operands[order[1]]
				if ecl.IsText(other) && other.Code == 0x80 {
					if answer, err := ecl.TextValue(other, nil); err == nil {
						if answer = strings.TrimSpace(answer); answer != "" {
							return answer
						}
					}
				}
				if source, err := ecl.WordAddress(other); err == nil {
					if answers := saved[source]; len(answers) != 0 {
						return strings.Join(answers, "／")
					}
				}
			}
		}
		switch instruction.Command.Opcode {
		case ExitOpcode, ReturnOpcode:
			if skippable {
				break
			}
			return ""
		case GotoOpcode:
			if skippable {
				// 緊接在 `IF` 後面的 `GOTO` 只有一半的時候會跳；另一半是直接
				// 往下走，而答案就在那一半（ecl7/23 `A4C6h` 之後才是
				// `COMPARE 6E79h "NOKNOK"`）。跳過去會把答案掃丟。
				break
			}
			if len(operands) != 1 || !operands[0].WordSet ||
				int(operands[0].Word) < PoolCodeAddressBase {
				return ""
			}
			// 跳回自己或往回跳都可能是迴圈（「猜錯再來一次」）；掃描次數的上限
			// 讓它終止，不必另外判。
			offset = int(operands[0].Word) - PoolCodeAddressBase
			continue
		}
		if instruction.Next <= offset {
			return ""
		}
		offset = instruction.Next
	}
	return ""
}
