package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	"github.com/wicanr2/golden-box-remake-engine/ecl"
	"github.com/wicanr2/golden-box-remake-engine/eclvm"
)

// `3Ch PROTECTION`（spec 089）：從運算元 1 的位址起讀連續的 ECL 變數，
// 每個非零的值印成一格（印的是**值加一**），碰到 0 就停，最後換行。

// protectionEvent 找出結果裡的 `3Ch`。
func protectionEvent(result eclvm.Result) (eclvm.Event, bool) {
	for _, event := range result.Events {
		if event.Opcode == gamepack.ProtectionOpcode {
			return event, true
		}
	}
	return eclvm.Event{}, false
}

// applyProtection 印出那一列，然後讓 ECL 繼續。
func (a *app) applyProtection(event eclvm.Event) error {
	instruction, err := a.eclInstruction(event.PC)
	if err != nil {
		return fmt.Errorf("decode Pool PROTECTION at %d: %w", event.PC, err)
	}
	if len(instruction.Operands) != gamepack.ProtectionOperands {
		return fmt.Errorf("Pool PROTECTION has %d operands, want %d",
			len(instruction.Operands), gamepack.ProtectionOperands)
	}
	// 用的是運算元自己的位址，不是它的內容。
	address, err := ecl.WordAddress(instruction.Operands[0])
	if err != nil {
		return fmt.Errorf("Pool PROTECTION operand: %w", err)
	}
	memory := a.eventSession.Machine().Memory
	parts := make([]string, 0, 8)
	for offset := 0; offset < gamepack.ProtectionMaxEntries; offset++ {
		value := memory[address+uint16(offset)]
		if value == 0 {
			break
		}
		// 原版印的是讀到的值加一（`324Fh` 的 `inc ax`）。
		parts = append(parts, strconv.Itoa(int(value)+1))
	}
	if len(parts) != 0 {
		line := strings.Join(parts, " ")
		if a.eventText != "" && !strings.HasSuffix(a.eventText, "\n") {
			a.eventText += "\n"
		}
		a.eventText += line
	}
	// 印完換行（`32BAh` 的 `inc [5D83h]`）。
	a.eventText += "\n"
	return a.continueInitialSearch(nil)
}
