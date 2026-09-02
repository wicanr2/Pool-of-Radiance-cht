package main

import (
	"fmt"

	"github.com/wicanr2/golden-box-remake-engine/ecl"
	"github.com/wicanr2/golden-box-remake-engine/eclvm"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// 遭遇選單（`29h`，spec 078）。原版把玩家的選擇經過一張五格類型表，再依
// 「類型 × 選擇」把 0..3 其中一個結果碼寫進運算元 4 指的 ECL 變數；分支由
// ECL 自己做，所以這裡只要寫對。
type encounterState struct {
	// resultAddress 是運算元 4 給的 ECL 變數位址。
	resultAddress uint16
	// kinds 是運算元 5..9 的五格類型表。
	kinds [gamepack.EncounterMenuOutcomeCount]uint8
	// distance 是怪物與隊伍的距離，PARLAY／ADVANCE 會讓它減一。
	distance int
	// fleeThreshold／advanceThreshold 是運算元 13 與 14。
	fleeThreshold    int
	advanceThreshold int
	// slowest／fastest 是隊伍的移動力範圍（spec 079）。
	slowest int
	fastest int
	// message 是上一次選擇印出來的字串。
	message string
}

// encounterNumericOperands 是這裡真的當數值讀的運算元序號：距離上限、
// 五格類型表、兩個移動力門檻。
var encounterNumericOperands = []int{2, 5, 6, 7, 8, 9, 13, 14}

// encounterEvent 找出結果裡的 `29h`。
func encounterEvent(result eclvm.Result) (eclvm.Event, bool) {
	for _, event := range result.Events {
		if event.Opcode == gamepack.EncounterMenuOpcode {
			return event, true
		}
	}
	return eclvm.Event{}, false
}

// encounterResultAddress 解出運算元 4 的位址。引擎的事件只帶運算元 0 的位址，
// 所以這裡自己把目前這一塊的位元組解一次——位址不能用 NumericValue 取，
// 那會把變數的**內容**當成位址。
func (a *app) encounterResultAddress(event eclvm.Event) (uint16, error) {
	if a.eventSession == nil {
		return 0, fmt.Errorf("Pool encounter has no ECL session")
	}
	archive, ok := a.eclCatalog.Archive(a.eclArchive)
	if !ok {
		return 0, fmt.Errorf("Pool ECL archive %d is absent", a.eclArchive)
	}
	block, ok := archive.Blocks[a.eventSession.CurrentBlockID()]
	if !ok {
		return 0, fmt.Errorf("Pool ECL block %d is absent from archive %d", a.eventSession.CurrentBlockID(), a.eclArchive)
	}
	if len(block) < 2 {
		return 0, fmt.Errorf("Pool ECL block %d is shorter than its two-byte prefix", a.eventSession.CurrentBlockID())
	}
	instruction, err := ecl.DecodeInstruction(block[2:], event.PC)
	if err != nil {
		return 0, fmt.Errorf("decode Pool encounter at %d: %w", event.PC, err)
	}
	if len(instruction.Operands) != gamepack.EncounterMenuOperands {
		return 0, fmt.Errorf("Pool encounter has %d operands, want %d", len(instruction.Operands), gamepack.EncounterMenuOperands)
	}
	address, err := ecl.WordAddress(instruction.Operands[gamepack.EncounterMenuResultOperand-1])
	if err != nil {
		return 0, fmt.Errorf("Pool encounter result operand: %w", err)
	}
	return address, nil
}

// partyMovementRange 是隊伍最慢與最快的移動力（overlay-03 `1F87h` 的作用）。
func (a *app) partyMovementRange() (slowest int, fastest int, err error) {
	if len(a.state.Party) == 0 {
		return 0, 0, fmt.Errorf("Pool party is empty")
	}
	if a.itemTypes == nil {
		return 0, 0, fmt.Errorf("Pool item type table is not configured")
	}
	for index, member := range a.state.Party {
		items := make([][]byte, 0, len(member.Inventory))
		for _, item := range member.Inventory {
			items = append(items, item.Raw)
		}
		carried, err := gamepack.CarriedWeight(items, member.Money)
		if err != nil {
			return 0, 0, fmt.Errorf("party member %d: %w", index, err)
		}
		rate, err := gamepack.MovementRateFor(gamepack.BaseMovementRate,
			member.Abilities[0], member.ExceptionalStrength, carried, items, a.itemTypes)
		if err != nil {
			return 0, 0, fmt.Errorf("party member %d: %w", index, err)
		}
		if index == 0 || rate < slowest {
			slowest = rate
		}
		if index == 0 || rate > fastest {
			fastest = rate
		}
	}
	return slowest, fastest, nil
}

// enterEncounter 把 `29h` 的參數收齊並開啟選單。
func (a *app) enterEncounter(event eclvm.Event) error {
	if len(event.Arguments) != gamepack.EncounterMenuOperands || len(event.ArgumentsValid) != gamepack.EncounterMenuOperands {
		return fmt.Errorf("Pool encounter carries %d arguments, want %d", len(event.Arguments), gamepack.EncounterMenuOperands)
	}
	// 只要求真的會用到的那幾個運算元解得出數值。運算元 4 是位址（要從
	// 原始位元組取），1、3、10..12 這支常式不當數值用。
	for _, operand := range encounterNumericOperands {
		if !event.ArgumentsValid[operand-1] {
			return fmt.Errorf("Pool encounter operand %d did not resolve", operand)
		}
	}
	address, err := a.encounterResultAddress(event)
	if err != nil {
		return err
	}
	slowest, fastest, err := a.partyMovementRange()
	if err != nil {
		return err
	}
	state := &encounterState{
		resultAddress:    address,
		distance:         encounterStartDistance(int(event.Arguments[1])),
		fleeThreshold:    int(event.Arguments[12]),
		advanceThreshold: int(event.Arguments[13]),
		slowest:          slowest,
		fastest:          fastest,
	}
	for index := range state.kinds {
		state.kinds[index] = uint8(event.Arguments[gamepack.EncounterMenuOutcomeFirstOperand-1+index])
	}
	a.encounter = state
	a.showEncounterMenu()
	return nil
}

// encounterStartDistance 是怪物離隊伍幾格。remake 還沒有地圖上的怪物群，
// 所以走原版「沒有情境」那一支：`0489h` 直接給 2，再夾到運算元 2 的上限。
func encounterStartDistance(limit int) int {
	distance := 2
	if limit < distance {
		distance = limit
	}
	if distance < 0 {
		distance = 0
	}
	return distance
}

func (a *app) showEncounterMenu() {
	state := a.encounter
	a.cellEventPending, a.cellWaitingMenu = true, true
	a.cellMenuOptions = gamepack.EncounterMenuOptions(state.distance, false)
	a.cellMenuCursor = 0
	text := fmt.Sprintf(a.text(msgEncounterDistance), state.distance)
	if state.message != "" {
		text = state.message + "  " + text
	}
	a.eventText = text
	a.eventLabel = a.cellMenuLabel()
	a.statusLine = a.text(msgEncounterPrompt)
}

// selectEncounterOption 處理玩家在遭遇選單上的選擇。
func (a *app) selectEncounterOption() error {
	state := a.encounter
	choice := gamepack.EncounterChoiceIndex(a.cellMenuCursor, a.cellMenuOptions)
	if choice >= gamepack.EncounterMenuOutcomeCount {
		return fmt.Errorf("Pool encounter choice %d is outside the outcome table", choice)
	}
	outcome, err := gamepack.ResolveEncounterChoice(gamepack.EncounterInputs{
		Kind:             state.kinds[choice],
		Choice:           choice,
		Distance:         state.distance,
		SlowestMovement:  state.slowest,
		FastestMovement:  state.fastest,
		FleeThreshold:    state.fleeThreshold,
		AdvanceThreshold: state.advanceThreshold,
	})
	if err != nil {
		return err
	}
	if outcome.Approach && state.distance > 0 {
		state.distance--
	}
	state.message = outcome.Message
	if outcome.Repeat {
		a.showEncounterMenu()
		return nil
	}
	if !outcome.Store {
		return fmt.Errorf("Pool encounter produced neither a result nor a repeat")
	}
	if a.eventMachine == nil {
		return fmt.Errorf("Pool encounter has no ECL machine")
	}
	a.eventMachine.Memory[state.resultAddress] = outcome.ResultCode
	a.encounter = nil
	a.cellEventPending, a.cellWaitingMenu = false, false
	a.cellMenuOptions, a.cellMenuCursor = nil, 0
	if outcome.Message != "" {
		a.eventText = outcome.Message
	}
	return a.continueInitialSearch(nil)
}
