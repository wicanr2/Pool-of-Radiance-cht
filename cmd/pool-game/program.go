package main

import (
	"fmt"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	"github.com/wicanr2/golden-box-remake-engine/ecl"
	"github.com/wicanr2/golden-box-remake-engine/eclvm"
)

// `38h PROGRAM`（spec 081）。四個呼叫點，三個值：0、8、9。
//
// 值 0 直接開城裡的隊伍管理畫面（overlay-16 entry 1 加 overlay-25 entry 37），
// 開完 ECL 從原地繼續——這一支讀得完整，所以接上。
//
// 值 9 多了兩件還沒讀出來的東西：問句的字串（`35DBh` 拿 `4948h` 當參數）與
// 「答完之後讓 block 結束」的路徑（原版是近呼叫 `00h EXIT` 的 handler）。
// 兩件都得靠猜才寫得出來，所以維持硬失敗——**跳過它與正確處理它在報表上
// 分不出來**。

// programEvent 找出結果裡的 `38h`。
func programEvent(result eclvm.Result) (eclvm.Event, bool) {
	for _, event := range result.Events {
		if event.Opcode == gamepack.ProgramOpcode {
			return event, true
		}
	}
	return eclvm.Event{}, false
}

// programSelector 解出運算元 1 的值。三個呼叫點的運算元都是位元組字面值，
// 所以這裡直接讀值，不必走記憶體。
func (a *app) programSelector(event eclvm.Event) (uint16, error) {
	if a.eventSession == nil {
		return 0, fmt.Errorf("Pool PROGRAM has no ECL session")
	}
	archive, ok := a.eclCatalog.Archive(a.eclArchive)
	if !ok {
		return 0, fmt.Errorf("Pool ECL archive %d is absent", a.eclArchive)
	}
	block, ok := archive.Blocks[a.eventSession.CurrentBlockID()]
	if !ok {
		return 0, fmt.Errorf("Pool ECL block %d is absent from archive %d",
			a.eventSession.CurrentBlockID(), a.eclArchive)
	}
	if len(block) < 2 {
		return 0, fmt.Errorf("Pool ECL block %d is shorter than its two-byte prefix",
			a.eventSession.CurrentBlockID())
	}
	instruction, err := ecl.DecodeInstruction(block[2:], event.PC)
	if err != nil {
		return 0, fmt.Errorf("decode Pool PROGRAM at %d: %w", event.PC, err)
	}
	if len(instruction.Operands) != gamepack.ProgramOperands {
		return 0, fmt.Errorf("Pool PROGRAM has %d operands, want %d",
			len(instruction.Operands), gamepack.ProgramOperands)
	}
	value, err := ecl.NumericValue(instruction.Operands[0], a.eventSession.Machine().Memory)
	if err != nil {
		return 0, fmt.Errorf("Pool PROGRAM selector: %w", err)
	}
	return value, nil
}

// enterProgram 依運算元分派。
func (a *app) enterProgram(event eclvm.Event) error {
	selector, err := a.programSelector(event)
	if err != nil {
		return err
	}
	switch selector {
	case gamepack.ProgramPartyManagement:
		a.openPartyManagement()
		return nil
	case gamepack.ProgramCamp:
		// 旅店的過夜（spec 081）：ECL 自己問過「要不要花一枚白金住下」也
		// 收過錢了，這裡直接開紮營畫面——選法術與休息就是原版那條鏈
		//（overlay-15 entry 1 → overlay-25 entry 37）在做的事。
		// 收掉畫面之後這個 block 結束，與原版那一支結尾呼叫 EXIT 一樣。
		a.campFromProgram = true
		a.programExitsBlock = true
		a.openCamp()
		if !a.campOpen {
			// 隊伍是空的，開不起來；照原版的收尾直接結束 block。
			a.campFromProgram, a.programExitsBlock = false, false
			a.finishCellBlock()
		}
		return nil
	case gamepack.ProgramEnding:
		// 值 8 是結局過場（overlay-18 entry 1，spec 108）：打贏泰倫斯拉克斯
		// 之後 `ECL5/7` 的 `A82Ah` 會推它。台詞與 `FINAL5.DAX` 那五層圖
		// 都在 `ending.go` 接好了。
		return a.enterEnding()
	default:
		// 其餘的值原版就是直接返回，什麼都不做（spec 081 的 `3167h`）。
		return a.continueInitialSearch(nil)
	}
}

// closeProgramCamp 是旅店那一次紮營收掉之後的收尾：原版那一支的結尾呼叫的
// 正是 `00h EXIT` 的 handler，所以走同一條路。
func (a *app) closeProgramCamp() error {
	if a.programExitsBlock {
		a.programExitsBlock = false
		a.finishCellBlock()
		return nil
	}
	return a.continueInitialSearch(nil)
}

// openPartyManagement 把畫面切到隊伍管理，並記住這是從地圖上進來的。
// 從標題進來的那一次會重跑開場，從這裡進來的不能——隊伍已經在圖上了。
func (a *app) openPartyManagement() {
	a.programManaging = true
	a.mode = modeMenu
	a.cellEventPending, a.cellWaitingMenu = false, false
	a.cellMenuOptions, a.cellMenuCursor = nil, 0
	a.eventText, a.eventLabel = "", ""
	a.statusLine = a.text(msgProgramManaging)
}

// closePartyManagement 回到地圖。值 0 讓 ECL 從 `38h` 的下一條繼續；
// 值 9 的結尾是呼叫 EXIT 的 handler，所以那一支直接結束這個 block。
func (a *app) closePartyManagement() error {
	a.programManaging = false
	a.mode = modeAdventure
	if a.programExitsBlock {
		a.programExitsBlock = false
		a.finishCellBlock()
		return nil
	}
	return a.continueInitialSearch(nil)
}
