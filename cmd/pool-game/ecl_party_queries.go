package main

import (
	"fmt"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/creation"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	"github.com/wicanr2/golden-box-remake-engine/ecl"
	"github.com/wicanr2/golden-box-remake-engine/eclvm"
)

// 三條直接對隊伍發問的 opcode（spec 085）：讀隊伍、算一個值、寫回 ECL 變數，
// 不需要任何畫面。

// partyQueryEvent 找出結果裡的 `32h`／`22h`／`23h`。
func partyQueryEvent(result eclvm.Result) (eclvm.Event, bool) {
	for _, event := range result.Events {
		switch event.Opcode {
		case gamepack.FindItemOpcode, gamepack.PartySurpriseOpcode, gamepack.SurpriseOpcode:
			return event, true
		}
	}
	return eclvm.Event{}, false
}

// eclInstruction 解出目前 block 裡某個位址的那一條指令。
func (a *app) eclInstruction(pc int) (ecl.Instruction, error) {
	if a.eventSession == nil {
		return ecl.Instruction{}, fmt.Errorf("Pool ECL session is absent")
	}
	archive, ok := a.eclCatalog.Archive(a.eclArchive)
	if !ok {
		return ecl.Instruction{}, fmt.Errorf("Pool ECL archive %d is absent", a.eclArchive)
	}
	block, ok := archive.Blocks[a.eventSession.CurrentBlockID()]
	if !ok {
		return ecl.Instruction{}, fmt.Errorf("Pool ECL block %d is absent from archive %d",
			a.eventSession.CurrentBlockID(), a.eclArchive)
	}
	if len(block) < 2 {
		return ecl.Instruction{}, fmt.Errorf("Pool ECL block %d is shorter than its two-byte prefix",
			a.eventSession.CurrentBlockID())
	}
	return ecl.DecodeInstruction(block[2:], pc)
}

// applyPartyQuery 依 opcode 分派，做完讓 ECL 繼續。
func (a *app) applyPartyQuery(event eclvm.Event) error {
	instruction, err := a.eclInstruction(event.PC)
	if err != nil {
		return fmt.Errorf("decode Pool party query at %d: %w", event.PC, err)
	}
	memory := a.eventSession.Machine().Memory
	switch event.Opcode {
	case gamepack.FindItemOpcode:
		if len(instruction.Operands) != gamepack.FindItemOperands {
			return fmt.Errorf("Pool FIND ITEM has %d operands, want %d",
				len(instruction.Operands), gamepack.FindItemOperands)
		}
		wanted, err := ecl.NumericValue(instruction.Operands[0], memory)
		if err != nil {
			return fmt.Errorf("Pool FIND ITEM operand: %w", err)
		}
		// 原版先把 `6D3Eh` 起六個位元組清成 0，再設「沒找到」。
		for offset := 0; offset < gamepack.FindItemClearedBytes; offset++ {
			memory[uint16(gamepack.FindItemFoundAddress+offset)] = 0
		}
		if a.partyCarriesItemType(uint8(wanted)) {
			memory[gamepack.FindItemFoundAddress] = 1
		} else {
			memory[gamepack.FindItemMissingAddress] = 1
		}
	case gamepack.PartySurpriseOpcode:
		if len(instruction.Operands) != gamepack.PartySurpriseOperands {
			return fmt.Errorf("Pool PARTY SURPRISE has %d operands, want %d",
				len(instruction.Operands), gamepack.PartySurpriseOperands)
		}
		codes, err := a.partyClassCodes()
		if err != nil {
			return err
		}
		alert, err := ecl.WordAddress(instruction.Operands[0])
		if err != nil {
			return fmt.Errorf("Pool PARTY SURPRISE operand 1: %w", err)
		}
		second, err := ecl.WordAddress(instruction.Operands[1])
		if err != nil {
			return fmt.Errorf("Pool PARTY SURPRISE operand 2: %w", err)
		}
		memory[alert] = uint16(gamepack.PartySurpriseAlert(codes))
		memory[second] = 0
	case gamepack.SurpriseOpcode:
		if len(instruction.Operands) != gamepack.SurpriseOperands {
			return fmt.Errorf("Pool SURPRISE has %d operands, want %d",
				len(instruction.Operands), gamepack.SurpriseOperands)
		}
		var values [gamepack.SurpriseOperands]uint8
		for index, operand := range instruction.Operands {
			value, err := ecl.NumericValue(operand, memory)
			if err != nil {
				return fmt.Errorf("Pool SURPRISE operand %d: %w", index+1, err)
			}
			values[index] = uint8(value)
		}
		memory[gamepack.SurpriseResultAddress] = uint16(gamepack.SurpriseOutcome(values,
			a.rollDice(1, gamepack.SurpriseDie), a.rollDice(1, gamepack.SurpriseDie)))
	default:
		return fmt.Errorf("Pool party query 0x%02X has no handler", event.Opcode)
	}
	return a.continueInitialSearch(nil)
}

// partyCarriesItemType 走隊伍與每個人的物品鏈，找型別索引相同的那一件。
// 原版找到就停（`28BEh` 之後不再往下走）。
func (a *app) partyCarriesItemType(wanted uint8) bool {
	for _, member := range a.state.Party {
		for _, item := range member.Inventory {
			if len(item.Raw) > itemTypeOffset && item.Raw[itemTypeOffset] == wanted {
				return true
			}
		}
	}
	return false
}

// partyClassCodes 取出每個人的職業碼（原版記錄的 `+2Fh`）。
func (a *app) partyClassCodes() ([]uint8, error) {
	codes := make([]uint8, 0, len(a.state.Party))
	for _, member := range a.state.Party {
		code, ok := creation.ClassDOSCode(member.ClassID)
		if !ok {
			return nil, fmt.Errorf("Pool character %q has unknown class %q", member.Name, member.ClassID)
		}
		codes = append(codes, code)
	}
	return codes, nil
}
