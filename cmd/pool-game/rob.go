package main

import (
	"encoding/binary"
	"fmt"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
	"github.com/wicanr2/golden-box-remake-engine/ecl"
	"github.com/wicanr2/golden-box-remake-engine/eclvm"
)

// `28h ROB`（spec 088）：按比例拿走錢，並逐件試著偷走物品。

// robEvent 找出結果裡的 `28h`。
func robEvent(result eclvm.Result) (eclvm.Event, bool) {
	for _, event := range result.Events {
		if event.Opcode == gamepack.RobOpcode {
			return event, true
		}
	}
	return eclvm.Event{}, false
}

// applyRob 套用一條 `28h`，然後讓 ECL 繼續。
func (a *app) applyRob(event eclvm.Event) error {
	instruction, err := a.eclInstruction(event.PC)
	if err != nil {
		return fmt.Errorf("decode Pool ROB at %d: %w", event.PC, err)
	}
	if len(instruction.Operands) != gamepack.RobOperands {
		return fmt.Errorf("Pool ROB has %d operands, want %d",
			len(instruction.Operands), gamepack.RobOperands)
	}
	memory := a.eventSession.Machine().Memory
	scope, err := ecl.NumericValue(instruction.Operands[gamepack.RobScopeOperand-1], memory)
	if err != nil {
		return fmt.Errorf("Pool ROB scope operand: %w", err)
	}
	percent, err := ecl.NumericValue(instruction.Operands[gamepack.RobPercentOperand-1], memory)
	if err != nil {
		return fmt.Errorf("Pool ROB percent operand: %w", err)
	}
	// 範圍 0 只對「目前角色」下手，那是 `39h WHO` 挑的那一個。
	targets := []int{a.currentCharacter}
	if scope != 0 {
		targets = targets[:0]
		for index := range a.state.Party {
			targets = append(targets, index)
		}
	}
	for _, index := range targets {
		if index >= len(a.state.Party) {
			continue
		}
		a.robOne(index, int(percent))
	}
	a.eventText = a.text(msgRobbed)
	return a.continueInitialSearch(nil)
}

// robOne 對一個人下手：先扣錢，再逐件試著偷走物品。
func (a *app) robOne(index, percent int) {
	member := &a.state.Party[index]
	member.Money = gamepack.RobMoney(member.Money, percent)
	// 成功率會被重的物品一路壓低，而且壓下去不會回復（原版就地改參數）。
	chance := percent
	kept := member.Inventory[:0]
	for _, item := range member.Inventory {
		weight := 0
		if len(item.Raw) > gamepack.ItemWeightOffset+1 {
			weight = int(binary.LittleEndian.Uint16(item.Raw[gamepack.ItemWeightOffset:]))
		}
		chance = gamepack.RobChanceAfterWeight(chance, weight)
		if gamepack.RobTakesItem(a.rollDice(1, gamepack.RobItemDie), chance) {
			continue
		}
		kept = append(kept, item)
	}
	member.Inventory = kept
	a.syncLibraryCharacter(*member)
}

// syncLibraryCharacter 把隊伍成員的變動同步回角色庫，兩邊分開會讓存檔
// 讀回來時錢與物品又長回來。
func (a *app) syncLibraryCharacter(member poolsave.Character) {
	for index := range a.state.CharacterLibrary {
		if a.state.CharacterLibrary[index].Name == member.Name {
			a.state.CharacterLibrary[index].Money = member.Money
			a.state.CharacterLibrary[index].Inventory = member.Inventory
		}
	}
}
