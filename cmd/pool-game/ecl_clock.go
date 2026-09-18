package main

import (
	"fmt"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	"github.com/wicanr2/golden-box-remake-engine/ecl"
	"github.com/wicanr2/golden-box-remake-engine/eclvm"
)

// `34h ECL CLOCK`（spec 093）。

// eclClockEvent 找出結果裡的 `34h`。
func eclClockEvent(result eclvm.Result) (eclvm.Event, bool) {
	for _, event := range result.Events {
		if event.Opcode == gamepack.ECLClockOpcode {
			return event, true
		}
	}
	return eclvm.Event{}, false
}

// eclClockDepthLimit 是連續的 `34h` 最多疊幾層。
//
// `applyECLClock` 推完時鐘就讓 ECL 繼續跑，而繼續跑之後可能又停在下一個 `34h`
// ——腳本在迴圈裡推時鐘的話（遊牧營地 ECL7/17 `9D37h` 就會），這條路是
// **互相遞迴**的：`applyECLClock → continueInitialSearch → consumeInitialSearch
// → applyECLClock`。沒有上限的話結果不是報錯，是 `fatal error: stack overflow`
// ——那個訊息只給得出 Go 的呼叫堆疊，不會說是哪一段腳本、哪一格。
//
// 上限給寬：正常的腳本一格推幾次而已，512 層離「一格推不完」還很遠，
// 而 512 層 Go 堆疊離爆掉也很遠。
const eclClockDepthLimit = 512

// applyECLClock 把時鐘往前推，然後讓 ECL 繼續。
func (a *app) applyECLClock(event eclvm.Event) error {
	a.eclClockDepth++
	defer func() { a.eclClockDepth-- }()
	if a.eclClockDepth > eclClockDepthLimit {
		block := -1
		if a.eventSession != nil {
			block = int(a.eventSession.CurrentBlockID())
		}
		return fmt.Errorf("Pool ECL CLOCK 連續停了 %d 次還沒跑完：ECL%d/%d PC %d 於 %+v",
			a.eclClockDepth, a.eclArchive, block, event.PC, a.spawn)
	}
	instruction, err := a.eclInstruction(event.PC)
	if err != nil {
		return fmt.Errorf("decode Pool ECL CLOCK at %d: %w", event.PC, err)
	}
	if len(instruction.Operands) == 0 {
		return fmt.Errorf("Pool ECL CLOCK at %d has no operands", event.PC)
	}
	count, err := ecl.NumericValue(instruction.Operands[0], a.eventSession.Machine().Memory)
	if err != nil {
		return fmt.Errorf("Pool ECL CLOCK operand: %w", err)
	}
	// 進位那一支（overlay-20 `02B1h`）還沒讀，所以只加不進位；
	// 時鐘目前沒有任何取用點，加錯進位看不出來也影響不到玩家路徑。
	a.eclClock = gamepack.AdvanceECLClock(a.eclClock,
		gamepack.ECLClockAdvancedField, int(count), nil)
	return a.continueInitialSearch(nil)
}
