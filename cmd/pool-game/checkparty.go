package main

import (
	"fmt"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	"github.com/wicanr2/golden-box-remake-engine/ecl"
	"github.com/wicanr2/golden-box-remake-engine/eclvm"
)

// `1Eh CHECKPARTY`（spec 092）。

// checkPartyEvent 找出結果裡的 `1Eh`。
func checkPartyEvent(result eclvm.Result) (eclvm.Event, bool) {
	for _, event := range result.Events {
		if event.Opcode == gamepack.CheckPartyOpcode {
			return event, true
		}
	}
	return eclvm.Event{}, false
}

// applyCheckParty 算好四個值寫回去，然後讓 ECL 繼續。
func (a *app) applyCheckParty(event eclvm.Event) error {
	instruction, err := a.eclInstruction(event.PC)
	if err != nil {
		return fmt.Errorf("decode Pool CHECKPARTY at %d: %w", event.PC, err)
	}
	if len(instruction.Operands) != gamepack.CheckPartyOperands {
		return fmt.Errorf("Pool CHECKPARTY has %d operands, want %d",
			len(instruction.Operands), gamepack.CheckPartyOperands)
	}
	memory := a.eventSession.Machine().Memory
	selector := instruction.Operands[0]
	address, addressErr := ecl.WordAddress(selector)
	mode, err := gamepack.CheckPartyMode(addressErr != nil, address)
	if err != nil {
		return err
	}
	var stats gamepack.CheckPartyStats
	switch mode {
	case gamepack.CheckPartyEffectMode:
		wanted, err := ecl.NumericValue(instruction.Operands[1], memory)
		if err != nil {
			return fmt.Errorf("Pool CHECKPARTY effect operand: %w", err)
		}
		effects := make([][]uint8, 0, len(a.state.Party))
		for _, member := range a.state.Party {
			effects = append(effects, member.Effects)
		}
		stats = gamepack.CheckPartyStats{Minimum: gamepack.CheckPartyInitialMinimum}
		stats.Flag = gamepack.CheckPartyEffectPresent(effects, uint8(wanted))
	case gamepack.CheckPartyFieldMovement:
		values := make([]uint8, 0, len(a.state.Party))
		for _, member := range a.state.Party {
			// 統計的是**算完裝備之後**的移動力，也就是原版記錄 `+11Ch`
			//（spec 079）；建角基礎值沒有把盔甲與負重算進去。
			_, armor, movement, err := partyCombatStats(member)
			if err != nil {
				return err
			}
			_, movement, err = a.memberDefenceStats(member, armor, movement)
			if err != nil {
				return err
			}
			values = append(values, movement)
		}
		stats = gamepack.CheckPartySummary(values)
	default:
		return fmt.Errorf("Pool CHECKPARTY selector %#04x reads record +79h, "+
			"a field whose meaning is not closed (spec 092)", address)
	}
	for index, value := range []uint8{stats.Minimum, stats.Maximum, stats.Average, stats.Flag} {
		destination, err := ecl.WordAddress(instruction.Operands[2+index])
		if err != nil {
			return fmt.Errorf("Pool CHECKPARTY result operand %d: %w", index+3, err)
		}
		memory[destination] = uint16(value)
	}
	return a.continueInitialSearch(nil)
}
