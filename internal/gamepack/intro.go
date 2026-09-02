package gamepack

import (
	"archive/zip"
	"fmt"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/character"
	"github.com/wicanr2/golden-box-remake-engine/combat/ability"
	"github.com/wicanr2/golden-box-remake-engine/ecl"
	"github.com/wicanr2/golden-box-remake-engine/eclvm"
	"github.com/wicanr2/golden-box-remake-engine/geometry"
)

// InitialEvent is the first player-visible ECL3/block 0 event reached by a
// newly created party. Text stays sourced from the user's original ZIP.
type InitialEvent struct {
	TriggerAddress uint16
	TriggerLimit   uint16
	EntryAddress   uint16
	HandlerAddress uint16
	Position       Spawn
	MonsterID      uint16
	Message        string
	ContinueLabel  string
	Tour           []TourStep
	ScriptBlock    []byte
	ScriptBlocks   map[uint16][]byte
}

// InitialCharacter is the narrow active-character projection required by the
// currently READY Pool ECL path. Full DOS character records remain game data.
type InitialCharacter struct {
	Name                string
	ClassID             string
	Abilities           [6]int
	ExceptionalStrength int
	CurrentHP           int
	ControlMorale       uint8
}

// TourStep is one original scripted position frame. Most steps only move the
// view; the six non-zero selectors pause for one or two text pages.
type TourStep struct {
	Position Spawn
	Selector uint8
	Messages []string
}

// ReadDOSInitialEvent closes Spec 010's first page and Spec 011's complete
// 34-step guided tour. It deliberately does not assign a post-tour movement
// policy; that belongs to the still-pending GEO consumer slice.
func ReadDOSInitialEvent(zipPath string) (InitialEvent, error) {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return InitialEvent{}, fmt.Errorf("open DOS ZIP: %w", err)
	}
	defer zr.Close()
	member, err := uniqueMember(zr.File, "ECL3.DAX")
	if err != nil {
		return InitialEvent{}, err
	}
	blocks, err := readDAXBlocks(member)
	if err != nil {
		return InitialEvent{}, fmt.Errorf("ECL3.DAX: %w", err)
	}
	block, ok := blocks[0]
	if !ok {
		return InitialEvent{}, fmt.Errorf("ECL3.DAX has no block 0")
	}
	if len(block) < 2 {
		return InitialEvent{}, fmt.Errorf("ECL3.DAX block 0 has %d bytes, want prefix and payload", len(block))
	}
	payload := block[2:]
	decode := func(offset int, opcode byte) (ecl.Instruction, error) {
		instruction, err := ecl.DecodeInstruction(payload, offset)
		if err != nil {
			return ecl.Instruction{}, err
		}
		if instruction.Command.Opcode != opcode {
			return ecl.Instruction{}, fmt.Errorf("ECL3/block0 offset 0x%X opcode 0x%02X, want 0x%02X", offset, instruction.Command.Opcode, opcode)
		}
		return instruction, nil
	}
	compare, err := decode(576, 0x03)
	if err != nil {
		return InitialEvent{}, err
	}
	if len(compare.Operands) != 2 || compare.Operands[0].Word != 0x4AC5 || compare.Operands[1].Low != 1 {
		return InitialEvent{}, fmt.Errorf("initial event trigger operands changed")
	}
	if _, err := decode(582, 0x18); err != nil {
		return InitialEvent{}, err
	}
	jump, err := decode(583, 0x01)
	if err != nil {
		return InitialEvent{}, err
	}
	if len(jump.Operands) != 1 || jump.Operands[0].Word != 0xB06E {
		return InitialEvent{}, fmt.Errorf("initial event handler jump changed")
	}
	positionOffsets := []int{6030, 6036, 6042}
	wantAddresses := []uint16{0xC04B, 0xC04C, 0xC04D}
	values := [3]uint8{}
	for index, offset := range positionOffsets {
		instruction, err := decode(offset, 0x09)
		if err != nil {
			return InitialEvent{}, err
		}
		if len(instruction.Operands) != 2 || instruction.Operands[1].Word != wantAddresses[index] {
			return InitialEvent{}, fmt.Errorf("position SAVE at 0x%X changed", offset)
		}
		values[index] = instruction.Operands[0].Low
	}
	monster, err := decode(6058, 0x0C)
	if err != nil {
		return InitialEvent{}, err
	}
	if len(monster.Operands) != 3 || monster.Operands[0].Low != 12 {
		return InitialEvent{}, fmt.Errorf("Rolf setup changed")
	}
	message, err := decode(6070, 0x12)
	if err != nil {
		return InitialEvent{}, err
	}
	if len(message.Operands) != 1 || len(message.Operands[0].Packed) == 0 {
		return InitialEvent{}, fmt.Errorf("Rolf greeting is absent")
	}
	menu, err := ecl.DecodeMenuRecord(block, 5660)
	if err != nil {
		return InitialEvent{}, err
	}
	if len(menu.OptionTexts) != 1 {
		return InitialEvent{}, fmt.Errorf("continue menu has %d options", len(menu.OptionTexts))
	}
	getTableChecks := []struct {
		offset int
		table  uint16
		out    uint16
	}{
		{0xB145 - 0x9900, 0xB5B1, 0x6E7A},
		{0xB14F - 0x9900, 0xB5D3, 0x6E7B},
		{0xB159 - 0x9900, 0xB5F5, 0xC04D},
		{0xB163 - 0x9900, 0xB58F, 0x6E7C},
	}
	for _, check := range getTableChecks {
		instruction, err := decode(check.offset, 0x2A)
		if err != nil {
			return InitialEvent{}, err
		}
		if len(instruction.Operands) != 3 || instruction.Operands[0].Word != check.table || instruction.Operands[1].Word != 0x6E79 || instruction.Operands[2].Word != check.out {
			return InitialEvent{}, fmt.Errorf("tour GETTABLE at 0x%X changed", check.offset)
		}
	}
	targets, after, err := ecl.BranchTargetsAtBase(block, 0xB171-0x9900, 0x9900)
	if err != nil {
		return InitialEvent{}, err
	}
	wantTargets := []int{0xB1C1, 0xB1C2, 0xB257, 0xB2F3, 0xB36A, 0xB400, 0xB48F}
	if after != 0xB18C-0x9900 || len(targets) != len(wantTargets) {
		return InitialEvent{}, fmt.Errorf("tour ON GOSUB shape changed")
	}
	for index, target := range targets {
		if target+0x9900 != wantTargets[index] {
			return InitialEvent{}, fmt.Errorf("tour selector %d target changed", index)
		}
	}
	if _, err := decode(0xAE85-0x9900, 0x00); err != nil {
		return InitialEvent{}, fmt.Errorf("tour EXIT changed: %w", err)
	}
	const stepCount = 34
	table := func(address int) ([]byte, error) {
		start := address - 0x9900
		if start < 0 || start+stepCount > len(payload) {
			return nil, fmt.Errorf("tour table 0x%04X is outside payload", address)
		}
		return append([]byte(nil), payload[start:start+stepCount]...), nil
	}
	selectors, err := table(0xB58F)
	if err != nil {
		return InitialEvent{}, err
	}
	xs, err := table(0xB5B1)
	if err != nil {
		return InitialEvent{}, err
	}
	ys, err := table(0xB5D3)
	if err != nil {
		return InitialEvent{}, err
	}
	facings, err := table(0xB5F5)
	if err != nil {
		return InitialEvent{}, err
	}
	pageOffsets := map[uint8][]int{
		1: {0xB1C2 - 0x9900},
		2: {0xB257 - 0x9900},
		3: {0xB2F3 - 0x9900},
		4: {0xB36A - 0x9900},
		5: {0xB400 - 0x9900},
		6: {0xB48F - 0x9900, 0xB51C - 0x9900},
	}
	tour := make([]TourStep, stepCount)
	for index := range tour {
		selector := selectors[index]
		if selector > 6 {
			return InitialEvent{}, fmt.Errorf("tour step %d selector %d is outside 0..6", index, selector)
		}
		step := TourStep{Position: Spawn{Map: MapKey{Archive: 3, BlockID: 0}, X: xs[index], Y: ys[index], Facing: facings[index]}, Selector: selector}
		for _, offset := range pageOffsets[selector] {
			page, err := decode(offset, 0x12)
			if err != nil {
				return InitialEvent{}, err
			}
			if len(page.Operands) != 1 || len(page.Operands[0].Packed) == 0 {
				return InitialEvent{}, fmt.Errorf("tour page at 0x%X is absent", offset)
			}
			step.Messages = append(step.Messages, ecl.DecodePackedText(page.Operands[0].Packed))
		}
		tour[index] = step
	}
	scriptBlocks := make(map[uint16][]byte, len(blocks))
	for id, data := range blocks {
		scriptBlocks[uint16(id)] = append([]byte(nil), data...)
	}
	return InitialEvent{
		TriggerAddress: 0x4AC5,
		TriggerLimit:   1,
		EntryAddress:   0x9AF2,
		HandlerAddress: 0xB06E,
		Position: Spawn{
			Map:    MapKey{Archive: 3, BlockID: 0},
			X:      values[0],
			Y:      values[1],
			Facing: values[2],
		},
		MonsterID:     uint16(monster.Operands[0].Low),
		Message:       ecl.DecodePackedText(message.Operands[0].Packed),
		ContinueLabel: menu.OptionTexts[0],
		Tour:          tour,
		ScriptBlock:   append([]byte(nil), block...),
		ScriptBlocks:  scriptBlocks,
	}, nil
}

// InitialEventPassthrough 讓盤點工具讀到同一份清單，不必自己再抄一次。
func InitialEventPassthrough() map[byte]bool { return initialEventPassthrough() }

func initialEventPassthrough() map[byte]bool {
	return map[byte]bool{
		0x0C: true, // SETUP MONSTER
		0x0D: true, // APPROACH
		0x0E: true, // PICTURE
		0x21: true, // LOAD FILES / title resource boundary
		0x24: true, // Pool service / encounter boundary
		0x29: true, // ENCOUNTER MENU（spec 078；前端一定要接，不能讓它靜靜跳過）
		0x2D: true, // CALL
		0x31: true, // SPRITE OFF
		0x37: true, // LOAD PIECES / title resource boundary
		0x0F: true, // INPUT NUMBER（spec 087）
		0x10: true, // INPUT STRING（spec 087）
		0x1E: true, // CHECKPARTY（spec 092；隊伍統計）
		0x22: true, // PARTY SURPRISE（spec 085）
		0x23: true, // SURPRISE（spec 085）
		0x28: true, // ROB（spec 088；前端要接，接不到隊伍不會掉東西）
		0x2C: true, // PARLAY（spec 086；前端要接，接不到會靜靜跳過一段交涉）
		0x2E: true, // DAMAGE（spec 084；前端要接，接不到會靜靜不扣血）
		0x32: true, // FIND ITEM（spec 085）
		0x33: true, // PRINT RETURN（spec 082；文字框換行）
		0x38: true, // PROGRAM（spec 081；前端要接，值 9 仍硬失敗）
		0x36: true, // ADD NPC（spec 091；把 NPC 加進隊伍）
		0x39: true, // WHO（spec 090；挑一個隊伍成員當「目前角色」）
		0x3C: true, // PROTECTION（spec 089；印一列數字）
		0x3D: true, // CLEAR BOX（spec 082；清掉文字框）
		0x3A: true, // DELAY
	}
}

// NewInitialEventSession owns all ECL3 blocks and follows original NEWECL
// transitions without resetting shared memory or the random stream.
func NewInitialEventSession(event InitialEvent, characters ...InitialCharacter) (*eclvm.BlockSession, error) {
	blocks := event.ScriptBlocks
	if len(blocks) == 0 && len(event.ScriptBlock) != 0 {
		blocks = map[uint16][]byte{0: event.ScriptBlock}
	}
	if len(blocks) == 0 {
		return nil, fmt.Errorf("initial event has no ECL blocks")
	}
	session, err := eclvm.NewBlockSession(blocks, 0, 0x9900, int(event.HandlerAddress)-0x9900, 5, initialEventPassthrough(), 1)
	if err != nil {
		return nil, err
	}
	if err := session.SetTransitionEntries(0, 4); err != nil {
		return nil, err
	}
	session.Machine().SetCharacterProjector(initialCharacterProjector(characters))
	session.Machine().SetPartyStrengthResolver(initialPartyStrengthResolver(characters))
	return session, nil
}

func initialPartyStrengthResolver(characters []InitialCharacter) eclvm.PartyStrengthResolver {
	snapshot := append([]InitialCharacter(nil), characters...)
	return func() (uint8, error) {
		records := make([]character.PartyStrengthRecord, len(snapshot))
		for index, value := range snapshot {
			record, err := initialPartyStrengthRecord(value)
			if err != nil {
				return 0, fmt.Errorf("Pool party member %d: %w", index, err)
			}
			records[index] = record
		}
		return character.PartyStrength(records), nil
	}
}

func initialPartyStrengthRecord(value InitialCharacter) (character.PartyStrengthRecord, error) {
	if value.CurrentHP < 0 || value.CurrentHP > 0xFF {
		return character.PartyStrengthRecord{}, fmt.Errorf("current HP %d is outside byte range", value.CurrentHP)
	}
	strengthIndex, ok := ability.StrengthIndex(value.Abilities[0], value.ExceptionalStrength)
	if !ok {
		return character.PartyStrengthRecord{}, fmt.Errorf("strength %d/%d has no original table index", value.Abilities[0], value.ExceptionalStrength)
	}
	if value.Abilities[3] < 0 || value.Abilities[3] > 0xFF {
		return character.PartyStrengthRecord{}, fmt.Errorf("dexterity %d is outside byte range", value.Abilities[3])
	}
	cleric, magicUser, ok := initialCasterLevels(value.ClassID)
	if !ok {
		return character.PartyStrengthRecord{}, fmt.Errorf("class %q is outside the Pool creation catalog", value.ClassID)
	}
	storedAttack := 40 + ability.StrengthHitAdjustment(strengthIndex)
	storedArmor := 50 + ability.DexterityDefenceAdjustment(value.Abilities[3])
	if storedAttack < 0 || storedAttack > 0xFF || storedArmor < 0 || storedArmor > 0xFF {
		return character.PartyStrengthRecord{}, fmt.Errorf("derived attack/armor %d/%d is outside byte range", storedAttack, storedArmor)
	}
	return character.PartyStrengthRecord{
		Field96: uint8(cleric), Field9B: uint8(magicUser), Field110: uint8(storedAttack),
		Field111: uint8(storedArmor), Field11B: uint8(value.CurrentHP),
	}, nil
}

func initialCasterLevels(classID string) (cleric, magicUser int, ok bool) {
	switch classID {
	case "cleric":
		return 1, 0, true
	case "fighter", "thief":
		return 0, 0, true
	case "magic-user":
		return 0, 1, true
	case "cleric-fighter":
		return 1, 0, true
	case "cleric-fighter-magic-user", "cleric-magic-user":
		return 1, 1, true
	case "fighter-magic-user", "fighter-magic-user-thief", "magic-user-thief":
		return 0, 1, true
	case "fighter-thief":
		return 0, 0, true
	default:
		return 0, 0, false
	}
}

func initialCharacterProjector(characters []InitialCharacter) eclvm.CharacterProjector {
	snapshot := append([]InitialCharacter(nil), characters...)
	return func(selected eclvm.CharacterSelection, memory map[uint16]uint16, strings map[uint16]string) error {
		index := int(selected.Index)
		if index < 0 || index >= len(snapshot) {
			// The DOS handler leaves DS:5CF0/5CF2 unchanged when the linked-list
			// walk reaches nil, so the prior projection must remain intact.
			return nil
		}
		character := snapshot[index]
		strings[0x6B00] = character.Name
		memory[0x6C00] = 1
		memory[0x6BB8] = uint16(character.ControlMorale)
		return nil
	}
}

// NewInitialEventMachine executes the original Pool bytecode through the
// shared VM. Only the six external effects observed on the READY Rolf path
// are acknowledged; their title-specific effects remain frontend work.
func NewInitialEventMachine(event InitialEvent) (*eclvm.Machine, error) {
	session, err := NewInitialEventSession(event)
	if err != nil {
		return nil, err
	}
	return session.Machine(), nil
}

// RunInitialSessionCellEntry is RunInitialCellEntry for the cross-block
// session used by normal gameplay.
func RunInitialSessionCellEntry(session *eclvm.BlockSession, grid geometry.Grid, position Spawn) (eclvm.Result, error) {
	if session == nil || session.Machine() == nil {
		return eclvm.Result{}, fmt.Errorf("initial ECL session is nil")
	}
	projectInitialPosition(session.Machine(), grid, position)
	if err := session.SetEntry(0); err != nil {
		return eclvm.Result{}, err
	}
	return session.RunUntilEvent(4096, nil, true)
}

func projectInitialPosition(machine *eclvm.Machine, grid geometry.Grid, position Spawn) {
	cell := grid.CellWrapped(int(position.X), int(position.Y))
	machine.Memory[0xC04B] = uint16(position.X)
	machine.Memory[0xC04C] = uint16(position.Y)
	machine.Memory[0xC04D] = uint16(position.Facing)
	wall, ok := grid.WallWrapped(int(position.X), int(position.Y), position.Direction())
	if !ok {
		wall = 0
	}
	machine.Memory[0xC04E] = uint16(wall)
	machine.Memory[0xC04F] = uint16(cell.Terrain)
}

// RunInitialSessionSearchEntry starts command-set entry one in the current
// block after projecting the post-move position, and follows any internal
// NEWECL transitions.
func RunInitialSessionSearchEntry(session *eclvm.BlockSession, grid geometry.Grid, position Spawn) (eclvm.Result, error) {
	if session == nil || session.Machine() == nil {
		return eclvm.Result{}, fmt.Errorf("initial ECL session is nil")
	}
	projectInitialPosition(session.Machine(), grid, position)
	if err := session.SetEntry(1); err != nil {
		return eclvm.Result{}, err
	}
	return session.RunUntilEvent(4096, nil, true)
}

// RunInitialCellEntry projects the live first-person registers and executes
// ECL3/block0 lifecycle entry zero in the same session used by the Rolf event.
func RunInitialCellEntry(machine *eclvm.Machine, grid geometry.Grid, position Spawn) (eclvm.Result, error) {
	if machine == nil {
		return eclvm.Result{}, fmt.Errorf("initial ECL machine is nil")
	}
	cell := grid.CellWrapped(int(position.X), int(position.Y))
	machine.Memory[0xC04B] = uint16(position.X)
	machine.Memory[0xC04C] = uint16(position.Y)
	machine.Memory[0xC04D] = uint16(position.Facing)
	wall, ok := grid.WallWrapped(int(position.X), int(position.Y), position.Direction())
	if !ok {
		wall = 0
	}
	machine.Memory[0xC04E] = uint16(wall)
	machine.Memory[0xC04F] = uint16(cell.Terrain)
	if err := machine.SetPC(0x9914 - 0x9900); err != nil {
		return eclvm.Result{}, err
	}
	return machine.RunUntilEvent(4096, nil, true)
}

// RunInitialSearchEntry starts command-set entry one after the per-turn entry
// has completed. The five-entry role is supported by the Pool block shape and
// cross-title executable evidence; Pool executable confirmation remains a
// separate evidence task.
func RunInitialSearchEntry(machine *eclvm.Machine) (eclvm.Result, error) {
	if machine == nil {
		return eclvm.Result{}, fmt.Errorf("initial ECL machine is nil")
	}
	if err := machine.SetPC(0x99EB - 0x9900); err != nil {
		return eclvm.Result{}, err
	}
	return machine.RunUntilEvent(4096, nil, true)
}
