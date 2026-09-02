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

// applyECLClock 把時鐘往前推，然後讓 ECL 繼續。
func (a *app) applyECLClock(event eclvm.Event) error {
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
