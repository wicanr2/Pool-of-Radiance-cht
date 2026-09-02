package main

import (
	"fmt"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	"github.com/wicanr2/golden-box-remake-engine/eclvm"
)

// `39h WHO`（spec 090）：讓玩家挑一個隊伍成員，挑到的存進「目前角色」。
//
// 原版把挑到的角色指標寫進 `DS:5CF0h`（傳的是那個槽的位址，是出參），
// 後面的 opcode 再從那裡讀——`28h ROB` 的範圍 0、`36h ADD NPC` 寫欄位、
// `38h PROGRAM` 也會搬它。remake 這一側用 currentCharacter 這個索引。

// whoEvent 找出結果裡的 `39h`。
func whoEvent(result eclvm.Result) (eclvm.Event, bool) {
	for _, event := range result.Events {
		if event.Opcode == gamepack.WhoOpcode {
			return event, true
		}
	}
	return eclvm.Event{}, false
}

// enterWho 把隊伍擺成選單。
func (a *app) enterWho(event eclvm.Event) error {
	instruction, err := a.eclInstruction(event.PC)
	if err != nil {
		return fmt.Errorf("decode Pool WHO at %d: %w", event.PC, err)
	}
	if len(instruction.Operands) != gamepack.WhoOperands {
		return fmt.Errorf("Pool WHO has %d operands, want %d",
			len(instruction.Operands), gamepack.WhoOperands)
	}
	if len(a.state.Party) == 0 {
		return fmt.Errorf("Pool WHO has no party to choose from")
	}
	a.whoPending = true
	a.cellEventPending, a.cellWaitingMenu = true, true
	a.cellMenuOptions = a.cellMenuOptions[:0]
	for _, member := range a.state.Party {
		a.cellMenuOptions = append(a.cellMenuOptions, member.Name)
	}
	a.cellMenuCursor = 0
	a.eventLabel = a.cellMenuLabel()
	a.statusLine = a.text(msgWhoPrompt)
	return nil
}

// selectWhoOption 記下挑到的人，然後讓 ECL 繼續。
func (a *app) selectWhoOption() error {
	if err := a.resolveWhoChoice(); err != nil {
		return err
	}
	return a.continueInitialSearch(nil)
}

// resolveWhoChoice 只做「記下來並收掉選單」那一半，好讓它單獨測得到。
func (a *app) resolveWhoChoice() error {
	if !a.whoPending {
		return fmt.Errorf("Pool WHO has no pending choice")
	}
	if a.cellMenuCursor < 0 || a.cellMenuCursor >= len(a.state.Party) {
		return fmt.Errorf("Pool WHO choice %d is outside the party", a.cellMenuCursor)
	}
	a.currentCharacter = a.cellMenuCursor
	a.whoPending = false
	a.cellEventPending, a.cellWaitingMenu = false, false
	a.cellMenuOptions, a.cellMenuCursor = nil, 0
	return nil
}
