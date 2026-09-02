package main

import (
	"fmt"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	"github.com/wicanr2/golden-box-remake-engine/ecl"
	"github.com/wicanr2/golden-box-remake-engine/eclvm"
)

// `3Bh SPELL`（spec 094）：找隊上誰記了某個法術，把槽位編號與第幾個人
// 寫進運算元 2 與 3。

// spellSearchEvent 找出結果裡的 `3Bh`。
func spellSearchEvent(result eclvm.Result) (eclvm.Event, bool) {
	for _, event := range result.Events {
		if event.Opcode == gamepack.SpellSearchOpcode {
			return event, true
		}
	}
	return eclvm.Event{}, false
}

// applySpellSearch 找完寫回去，然後讓 ECL 繼續。
func (a *app) applySpellSearch(event eclvm.Event) error {
	instruction, err := a.eclInstruction(event.PC)
	if err != nil {
		return fmt.Errorf("decode Pool SPELL at %d: %w", event.PC, err)
	}
	if len(instruction.Operands) != gamepack.SpellSearchOperands {
		return fmt.Errorf("Pool SPELL has %d operands, want %d",
			len(instruction.Operands), gamepack.SpellSearchOperands)
	}
	memory := a.eventSession.Machine().Memory
	wanted, err := ecl.NumericValue(instruction.Operands[0], memory)
	if err != nil {
		return fmt.Errorf("Pool SPELL id operand: %w", err)
	}
	slotAddress, err := ecl.WordAddress(instruction.Operands[1])
	if err != nil {
		return fmt.Errorf("Pool SPELL slot operand: %w", err)
	}
	memberAddress, err := ecl.WordAddress(instruction.Operands[2])
	if err != nil {
		return fmt.Errorf("Pool SPELL member operand: %w", err)
	}
	// **原版只掃得到第一個人**：內層掃完就把「找到了」的旗標設成 1，
	// 外層因此在第一輪之後就退出（spec 094）。照抄那個範圍。
	slot := gamepack.SpellSearchNotFound
	if len(a.state.Party) != 0 {
		slot = gamepack.SearchMemorisedSpell(a.state.Party[0].Memorised, uint8(wanted))
	}
	members := 0
	if len(a.state.Party) != 0 {
		members = 1
	}
	memory[slotAddress] = uint16(slot)
	memory[memberAddress] = uint16(members)
	return a.continueInitialSearch(nil)
}
