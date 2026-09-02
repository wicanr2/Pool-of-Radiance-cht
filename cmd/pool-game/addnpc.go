package main

import (
	"fmt"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
	"github.com/wicanr2/golden-box-remake-engine/ecl"
	"github.com/wicanr2/golden-box-remake-engine/eclvm"
)

// `36h ADD NPC`（spec 091）：依編號從 `MON<n>CHA.DAX` 讀一筆 285-byte 記錄，
// 加進隊伍。編號 18h 那一位站到對面。

// addNPCEvent 找出結果裡的 `36h`。
func addNPCEvent(result eclvm.Result) (eclvm.Event, bool) {
	for _, event := range result.Events {
		if event.Opcode == gamepack.AddNPCOpcode {
			return event, true
		}
	}
	return eclvm.Event{}, false
}

// applyAddNPC 把 NPC 加進隊伍，然後讓 ECL 繼續。
func (a *app) applyAddNPC(event eclvm.Event) error {
	instruction, err := a.eclInstruction(event.PC)
	if err != nil {
		return fmt.Errorf("decode Pool ADD NPC at %d: %w", event.PC, err)
	}
	if len(instruction.Operands) != gamepack.AddNPCOperands {
		return fmt.Errorf("Pool ADD NPC has %d operands, want %d",
			len(instruction.Operands), gamepack.AddNPCOperands)
	}
	memory := a.eventSession.Machine().Memory
	id, err := ecl.NumericValue(instruction.Operands[0], memory)
	if err != nil {
		return fmt.Errorf("Pool ADD NPC id operand: %w", err)
	}
	if id > 0xFF {
		return fmt.Errorf("Pool ADD NPC id %d exceeds one byte", id)
	}
	// 隊伍滿了就不加（原版在 overlay-17 entry 9 擋「人數大於 7」）。
	if len(a.state.Party) >= poolsave.PartyMaximum {
		return a.continueInitialSearch(nil)
	}
	if a.loadMonster == nil {
		return fmt.Errorf("Pool monster loader is not configured")
	}
	archive := a.eclArchive
	if archive == 0 {
		archive = a.spawn.Map.Archive
	}
	record, err := a.loadMonster(archive, uint8(id))
	if err != nil {
		return fmt.Errorf("load Pool NPC archive %d block %d: %w", archive, id, err)
	}
	member := poolsave.Character{
		Name:      record.Name,
		NPC:       true,
		Side:      gamepack.AddNPCSide(uint8(id)),
		Record:    append([]byte(nil), record.Raw[:]...),
		MaxHP:     int(record.MaxHitPoints()),
		CurrentHP: int(record.CurrentHitPoints()),
	}
	if member.CurrentHP > member.MaxHP {
		member.CurrentHP = member.MaxHP
	}
	a.state.Party = append(a.state.Party, member)
	a.eventText = fmt.Sprintf(a.text(msgNPCJoined), member.Name)
	return a.continueInitialSearch(nil)
}
