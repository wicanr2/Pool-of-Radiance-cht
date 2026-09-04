package main

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	"github.com/wicanr2/golden-box-remake-engine/ecl"
	"github.com/wicanr2/golden-box-remake-engine/eclvm"
)

// `0Fh INPUT NUMBER` 與 `10h INPUT STRING`（spec 087）：向玩家要一個數字或
// 一段字串，寫進**運算元 2** 指的 ECL 變數。提示句由前一則 `12h PRINT` 給，
// 兩條 opcode 自己不帶文字。

// eclInputState 是進行中的一次輸入。
type eclInputState struct {
	// numeric 為真是 `0Fh`，否則是 `10h`。
	numeric bool
	// address 是運算元 2 的位址。
	address uint16
	// buffer 是目前打進去的內容。
	buffer string
}

// eclInputEvent 找出結果裡的 `0Fh` 或 `10h`。
func eclInputEvent(result eclvm.Result) (eclvm.Event, bool) {
	for _, event := range result.Events {
		switch event.Opcode {
		case gamepack.InputNumberOpcode, gamepack.InputStringOpcode:
			return event, true
		}
	}
	return eclvm.Event{}, false
}

// enterECLInput 解出寫回位址，然後把輸入列擺出來。
func (a *app) enterECLInput(event eclvm.Event) error {
	instruction, err := a.eclInstruction(event.PC)
	if err != nil {
		return fmt.Errorf("decode Pool input at %d: %w", event.PC, err)
	}
	if len(instruction.Operands) != gamepack.InputOperands {
		return fmt.Errorf("Pool input 0x%02X has %d operands, want %d",
			event.Opcode, len(instruction.Operands), gamepack.InputOperands)
	}
	// 位址在運算元 2；原版取的是 `6E0Dh`／`6E4Dh` 那一組，也就是第二個運算元。
	address, err := ecl.WordAddress(instruction.Operands[gamepack.InputDestinationOperand-1])
	if err != nil {
		return fmt.Errorf("Pool input destination operand: %w", err)
	}
	a.eclInput = &eclInputState{numeric: event.Opcode == gamepack.InputNumberOpcode, address: address}
	a.cellEventPending, a.cellWaitingMenu = true, false
	a.cellMenuOptions, a.cellMenuCursor = nil, 0
	// 原版要玩家翻說明書才知道要打什麼字（索寇要塞的亡魂、野外的密碼門）。
	// 那些字是遊戲內 NPC 說過的，但隔了很多格，忘了就過不去。答案本來就寫在
	// 原版資料的 `03h COMPARE` 裡，直接附在問句後面。
	// **這是 remake 的擴充，不是原版行為**（原版沒有這個括號）。
	if answer := a.eclInputAnswer(instruction.Next, address); answer != "" {
		a.eventText = strings.TrimRight(a.eventText, " ") + "（" + answer + "）"
	}
	a.eventLabel = a.eclInputLabel()
	a.statusLine = a.text(msgEclInputPrompt)
	return nil
}

// eclInputAnswer 找出這次輸入之後拿來比對的字面。原版有兩種寫法：
//
//	ecl7/23  A4A1 INPUT STRING #6 6E79h
//	         A4CA COMPARE 6E79h "NOKNOK"        ← 直接比字面
//
//	ecl4/21  9E80 INPUT STRING #7 982Ch
//	         9E8D SAVE "SAMOSUD" 9890h          ← 先存進字串變數
//	         9E9A SAVE "SHESTNI" 9890h          ← 依 4A26h 二選一
//	         9EA6 COMPARE 982Ch 9890h           ← 比的是變數
//
// 所以掃的時候順便記下 `09h SAVE <字面> → <位址>`，遇到比對變數的
// `COMPARE` 就把該位址收到的字面全部拿出來。同一個位址被寫過兩次時兩個
// 都列——玩家當下需要哪一個，靜態分不出來，不猜。
//
// 只往後掃固定步數，找不到就不顯示。
func (a *app) eclInputAnswer(pc int, address uint16) string {
	const scanLimit = 64
	saved := map[uint16][]string{}
	for offset, scanned := pc, 0; scanned < scanLimit; scanned++ {
		instruction, err := a.eclInstruction(offset)
		if err != nil {
			return ""
		}
		operands := instruction.Operands
		switch {
		case instruction.Command.Opcode == gamepack.SaveOpcode && len(operands) == 2 &&
			ecl.IsText(operands[0]):
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
		case instruction.Command.Opcode == gamepack.CompareOpcode && len(operands) == 2:
			compared, err := ecl.WordAddress(operands[0])
			if err != nil || compared != address {
				break
			}
			if ecl.IsText(operands[1]) && operands[1].Code == 0x80 {
				if answer, err := ecl.TextValue(operands[1], nil); err == nil {
					if answer = strings.TrimSpace(answer); answer != "" {
						return answer
					}
				}
			}
			if source, err := ecl.WordAddress(operands[1]); err == nil {
				if answers := saved[source]; len(answers) != 0 {
					return strings.Join(answers, "／")
				}
			}
		}
		if instruction.Next <= offset {
			return ""
		}
		offset = instruction.Next
	}
	return ""
}

// eclInputLabel 是輸入列本身。
func (a *app) eclInputLabel() string {
	if a.eclInput == nil {
		return ""
	}
	return a.eclInput.buffer + "_"
}

// eclInputUpdate 收鍵盤。回傳 true 代表這一影格由輸入列吃掉了。
func (a *app) eclInputUpdate() (bool, error) {
	state := a.eclInput
	if state == nil {
		return false, nil
	}
	if a.justPressed(ebiten.KeyBackspace) && len(state.buffer) != 0 {
		state.buffer = state.buffer[:len(state.buffer)-1]
	}
	for _, entered := range a.inputChars() {
		if len(state.buffer) >= gamepack.InputStringMaxLength {
			break
		}
		if state.numeric {
			if entered >= '0' && entered <= '9' {
				state.buffer += string(entered)
			}
			continue
		}
		if entered >= 0x20 && entered <= 0x7E {
			state.buffer += strings.ToUpper(string(entered))
		}
	}
	a.eventLabel = a.eclInputLabel()
	if a.justPressed(ebiten.KeyEnter) {
		if err := a.commitECLInput(); err != nil {
			return true, err
		}
		return true, a.continueInitialSearch(nil)
	}
	return true, nil
}

// commitECLInput 把打好的內容寫進 ECL 變數並收掉輸入列。
func (a *app) commitECLInput() error {
	state := a.eclInput
	if state == nil {
		return fmt.Errorf("Pool input has no state")
	}
	if a.eventMachine == nil {
		return fmt.Errorf("Pool input has no ECL machine")
	}
	if state.numeric {
		value := 0
		if state.buffer != "" {
			parsed, err := strconv.Atoi(state.buffer)
			if err != nil {
				return fmt.Errorf("Pool input number %q: %w", state.buffer, err)
			}
			value = parsed
		}
		a.eventMachine.Memory[state.address] = uint16(value)
	} else {
		text := state.buffer
		if text == "" {
			// 原版對空字串塞一個空白（overlay-03 `09AFh` 複製 `cs:95Eh`）。
			text = gamepack.InputStringEmptyReplacement
		}
		a.eventMachine.Strings[state.address] = text
	}
	a.eclInput = nil
	a.cellEventPending, a.cellWaitingMenu = false, false
	a.eventLabel = ""
	return nil
}
