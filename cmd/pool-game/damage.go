package main

import (
	"fmt"
	"strings"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
	"github.com/wicanr2/golden-box-remake-engine/ecl"
	"github.com/wicanr2/golden-box-remake-engine/eclvm"
)

// `2Eh DAMAGE`（spec 084）：ECL 直接對隊伍造成傷害，全遊戲 47 個呼叫點。
// 規則本身在 internal/gamepack；這裡負責解運算元、挑目標、擲骰與擲豁免，
// 再把結果寫回隊伍。

// damageEvent 找出結果裡的 `2Eh`。
func damageEvent(result eclvm.Result) (eclvm.Event, bool) {
	for _, event := range result.Events {
		if event.Opcode == gamepack.DamageOpcode {
			return event, true
		}
	}
	return eclvm.Event{}, false
}

// damageRequest 解出五個運算元。它們在原始資料裡是位元組字面值或變數，
// 所以照 NumericValue 取值。
func (a *app) damageRequest(event eclvm.Event) (gamepack.DamageRequest, error) {
	if a.eventSession == nil {
		return gamepack.DamageRequest{}, fmt.Errorf("Pool DAMAGE has no ECL session")
	}
	archive, ok := a.eclCatalog.Archive(a.eclArchive)
	if !ok {
		return gamepack.DamageRequest{}, fmt.Errorf("Pool ECL archive %d is absent", a.eclArchive)
	}
	block, ok := archive.Blocks[a.eventSession.CurrentBlockID()]
	if !ok {
		return gamepack.DamageRequest{}, fmt.Errorf("Pool ECL block %d is absent from archive %d",
			a.eventSession.CurrentBlockID(), a.eclArchive)
	}
	if len(block) < 2 {
		return gamepack.DamageRequest{}, fmt.Errorf("Pool ECL block %d is shorter than its two-byte prefix",
			a.eventSession.CurrentBlockID())
	}
	instruction, err := ecl.DecodeInstruction(block[2:], event.PC)
	if err != nil {
		return gamepack.DamageRequest{}, fmt.Errorf("decode Pool DAMAGE at %d: %w", event.PC, err)
	}
	if len(instruction.Operands) != gamepack.DamageOperands {
		return gamepack.DamageRequest{}, fmt.Errorf("Pool DAMAGE has %d operands, want %d",
			len(instruction.Operands), gamepack.DamageOperands)
	}
	var values [gamepack.DamageOperands]uint16
	for index, operand := range instruction.Operands {
		value, err := ecl.NumericValue(operand, a.eventSession.Machine().Memory)
		if err != nil {
			return gamepack.DamageRequest{}, fmt.Errorf("Pool DAMAGE operand %d: %w", index+1, err)
		}
		values[index] = value
	}
	return gamepack.NewDamageRequest(values), nil
}

// applyDamageEvent 套用一條 `2Eh`，然後讓 ECL 繼續。
func (a *app) applyDamageEvent(event eclvm.Event) error {
	request, err := a.damageRequest(event)
	if err != nil {
		return err
	}
	if !request.Applies() {
		// bit 7 沒設的話原版整條跳過（`2B89h`）。
		return a.continueInitialSearch(nil)
	}
	if len(a.state.Party) == 0 {
		return fmt.Errorf("Pool DAMAGE has no party to damage")
	}
	// 傷害只擲一次：全隊模式下每個人吃的是同一個數字（`2B47h` 在挑目標之前）。
	damage := a.rollDice(request.DiceCount, request.DiceSides) + request.Bonus
	targets := make([]int, 0, len(a.state.Party))
	if request.WholeParty() {
		for index := range a.state.Party {
			targets = append(targets, index)
		}
	} else {
		targets = append(targets, a.rollDice(1, len(a.state.Party))-1)
	}
	lines := make([]string, 0, len(targets))
	for _, index := range targets {
		line, err := a.damageOne(index, request, damage)
		if err != nil {
			return err
		}
		if line != "" {
			lines = append(lines, line)
		}
	}
	if len(lines) != 0 {
		a.eventText = strings.Join(lines, "\n")
	}
	return a.continueInitialSearch(nil)
}

// damageOne 對一個人擲豁免、套用傷害並回報一行訊息。
func (a *app) damageOne(index int, request gamepack.DamageRequest, damage int) (string, error) {
	member := a.state.Party[index]
	if member.Status > gamepack.AliveStateMax {
		// 已經倒下的人不再吃傷害（`2958h` 對狀態 6 直接返回）。
		return "", nil
	}
	if request.AllowsSave() {
		saved, err := a.savingThrowFor(member, request.SaveCategory(), request.SaveModifier)
		if err != nil {
			return "", err
		}
		if saved {
			return fmt.Sprintf(a.text(msgDamageSaved), member.Name), nil
		}
	}
	outcome := gamepack.ApplyDamage(member.CurrentHP, member.Status, damage)
	a.state.Party[index].CurrentHP = outcome.HitPoints
	a.state.Party[index].Status = outcome.State
	for library := range a.state.CharacterLibrary {
		if a.state.CharacterLibrary[library].Name == member.Name {
			a.state.CharacterLibrary[library].CurrentHP = outcome.HitPoints
			a.state.CharacterLibrary[library].Status = outcome.State
		}
	}
	if outcome.Downed {
		return fmt.Sprintf(a.text(msgDamageDies), member.Name), nil
	}
	return fmt.Sprintf(a.text(msgDamageHit), member.Name, damage), nil
}

// savingThrowFor 依 spec 075 判定：目標值由職業等級查 `DS:41E6h` 的表算出來
// （spec 075 的 `0253h`），修正欄位 `+101h` remake 目前一律是 0。
func (a *app) savingThrowFor(member poolsave.Character, category, modifier int) (bool, error) {
	if a.savingThrows == nil {
		return false, fmt.Errorf("Pool saving throw table is not configured")
	}
	levels, err := partyClassLevels(member)
	if err != nil {
		return false, err
	}
	targets, err := a.savingThrows.TargetsForLevels(levels)
	if err != nil {
		return false, err
	}
	if category >= gamepack.SavingThrowCategories {
		return false, fmt.Errorf("Pool DAMAGE saving throw category %d is outside 0..%d",
			category, gamepack.SavingThrowCategories-1)
	}
	roll := a.rollDice(1, gamepack.SavingThrowDie)
	if roll == 1 {
		return false, nil
	}
	if roll == gamepack.SavingThrowDie {
		return true, nil
	}
	return int(targets[category]) <= roll+modifier, nil
}
