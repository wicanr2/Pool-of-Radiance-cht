package main

import (
	"fmt"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	"github.com/wicanr2/golden-box-remake-engine/ecl"
	"github.com/wicanr2/golden-box-remake-engine/eclvm"
)

// `2Ch PARLAY`（spec 086）：五種語氣的交涉選單，形狀與 `29h ENCOUNTER MENU`
// 一樣——運算元 1..5 是用選擇當索引的結果表，運算元 6 說結果寫進哪個 ECL
// 變數。分支由 ECL 自己做，remake 只要把碼寫對。

// parlayState 是一次交涉。
type parlayState struct {
	// outcomes 是運算元 1..5 的值。
	outcomes [gamepack.ParlayOutcomeCount]uint16
	// resultAddress 是運算元 6 指的位址。
	resultAddress uint16
}

// parlayEvent 找出結果裡的 `2Ch`。
func parlayEvent(result eclvm.Result) (eclvm.Event, bool) {
	for _, event := range result.Events {
		if event.Opcode == gamepack.ParlayOpcode {
			return event, true
		}
	}
	return eclvm.Event{}, false
}

// enterParlay 解出結果表與寫回位址，然後把選單擺出來。
func (a *app) enterParlay(event eclvm.Event) error {
	instruction, err := a.eclInstruction(event.PC)
	if err != nil {
		return fmt.Errorf("decode Pool PARLAY at %d: %w", event.PC, err)
	}
	if len(instruction.Operands) != gamepack.ParlayOperands {
		return fmt.Errorf("Pool PARLAY has %d operands, want %d",
			len(instruction.Operands), gamepack.ParlayOperands)
	}
	state := &parlayState{}
	memory := a.eventSession.Machine().Memory
	for index := 0; index < gamepack.ParlayOutcomeCount; index++ {
		value, err := ecl.NumericValue(instruction.Operands[index], memory)
		if err != nil {
			return fmt.Errorf("Pool PARLAY outcome %d: %w", index+1, err)
		}
		state.outcomes[index] = value
	}
	// 位址要用 WordAddress：NumericValue 會把變數的**內容**當成位址。
	address, err := ecl.WordAddress(instruction.Operands[gamepack.ParlayResultOperand-1])
	if err != nil {
		return fmt.Errorf("Pool PARLAY result operand: %w", err)
	}
	state.resultAddress = address
	a.parlay = state
	a.showParlayMenu()
	return nil
}

// parlayMenuMessages 是五種語氣的顯示字串，順序與 gamepack.ParlayChoices
// 相同。這五個字來自 overlay-03 的程式碼字串而不是 ECL 文字，所以歸 UI
// 訊息表管，不進 internal/gametext 的原文對照（那一份只收 ECL 文字）。
var parlayMenuMessages = [gamepack.ParlayOutcomeCount]messageID{
	msgParlayHaughty, msgParlaySly, msgParlayNice, msgParlayMeek, msgParlayAbusive,
}

// showParlayMenu 把五種語氣擺上選單。
func (a *app) showParlayMenu() {
	a.cellEventPending, a.cellWaitingMenu = true, true
	a.cellMenuOptions = a.cellMenuOptions[:0]
	for _, item := range parlayMenuMessages {
		a.cellMenuOptions = append(a.cellMenuOptions, a.text(item))
	}
	a.cellMenuCursor = 0
	a.eventLabel = a.cellMenuLabel()
	a.statusLine = a.text(msgParlayPrompt)
}

// selectParlayOption 把選到的那一格結果碼寫回去，然後讓 ECL 繼續。
func (a *app) selectParlayOption() error {
	if err := a.resolveParlayChoice(); err != nil {
		return err
	}
	return a.continueInitialSearch(nil)
}

// resolveParlayChoice 只做「寫回結果碼並收掉選單」那一半，好讓它單獨測得到。
func (a *app) resolveParlayChoice() error {
	state := a.parlay
	if state == nil {
		return fmt.Errorf("Pool parlay has no state")
	}
	choice := a.cellMenuCursor
	if choice < 0 || choice >= gamepack.ParlayOutcomeCount {
		return fmt.Errorf("Pool parlay choice %d is outside the outcome table", choice)
	}
	if a.eventMachine == nil {
		return fmt.Errorf("Pool parlay has no ECL machine")
	}
	a.eventMachine.Memory[state.resultAddress] = state.outcomes[choice]
	a.parlay = nil
	a.cellEventPending, a.cellWaitingMenu = false, false
	a.cellMenuOptions, a.cellMenuCursor = nil, 0
	return nil
}
