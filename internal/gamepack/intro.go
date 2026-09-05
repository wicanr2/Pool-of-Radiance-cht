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
	// SpriteBlock／ApproachDistance／PortraitBody 是 `SETUP MONSTER` 的三個
	// operand（spec 117）：第一個是 `SPRIT<區號>.DAX` 的區塊（第一人稱視野裡
	// 走過來的那個人形），第二個是接近距離，第三個是 `BODY<區號>.DAX` 的區塊。
	SpriteBlock      uint8
	ApproachDistance uint8
	PortraitBody     uint8
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
	// Platinum 是 ECL 的 active-character 視窗 `6BC3h` 那一格。原版的賭場
	// 腳本（ecl3/0 `A513h`）把它印成 `'YOU HAVE' n 'PP.'`，港務長
	// （`A1E5h`）拿它扣一枚白金當船資，所以它是**選到的那個人身上的白金**。
	Platinum uint16
}

// CharacterPlatinumAddress 是 ECL 讀寫白金的位址。
const CharacterPlatinumAddress = 0x6BC3

// CharacterWindow 是 `0Ah LOAD CHARACTER` 選到的那個人的資料來源與去處。
//
// 原版的 `5CF0h` 是**指標**，指到那個人的 285-byte 記錄本身（spec 021），
// 所以腳本對 `6BC3h` 的加減是直接改在那個人身上的——ecl3/0 `A1F0h` 扣掉的
// 一枚白金、`A62Fh` 賭場寫回的餘額都是這樣留下來的。remake 的記憶體是
// 一張 map，沒有這種疊合，所以要用「選進來時抄進去、換人時抄回來」補上；
// 抄回來的那一半就是 `CommitPlatinum`。
type CharacterWindow interface {
	Character(index int) (InitialCharacter, bool)
	CommitPlatinum(index int, platinum uint16)
}

// CharacterBinding 把一個 window 綁在一個 session 上，並記住現在視窗裡是誰。
//
// 記住是誰是必要的：視窗裡的值要在**換人之前**抄回去，否則腳本改過的白金
// 會被下一次投影蓋掉。
type CharacterBinding struct {
	window  CharacterWindow
	current int
}

// NewCharacterBinding 建一個綁定；current 從 -1 開始表示視窗還沒有人。
func NewCharacterBinding(window CharacterWindow) *CharacterBinding {
	return &CharacterBinding{window: window, current: -1}
}

// Projector 是給 `0Ah LOAD CHARACTER` 用的投影器。
func (b *CharacterBinding) Projector() eclvm.CharacterProjector {
	return func(selected eclvm.CharacterSelection, memory map[uint16]uint16,
		strings map[uint16]string) error {
		return b.project(int(selected.Index), memory, strings)
	}
}

// Select 是 `39h WHO` 挑完人之後要做的事：原版寫 `5CF0h`（spec 090），
// 而視窗照的就是那個指標。少了這一步，WHO 後面的 `COMPARE @6BB8`、
// `SUBTRACT @6BC3` 讀到的還是上一個人的值。
func (b *CharacterBinding) Select(machine *eclvm.Machine, index int) error {
	if machine == nil {
		return fmt.Errorf("Pool ECL machine is nil")
	}
	if index < 0 || index > 0x7F {
		return fmt.Errorf("Pool character index %d is outside 0..127", index)
	}
	return b.project(index, machine.Memory, machine.Strings)
}

// Flush 把視窗現在的值抄回目前那個人。腳本跑完一段就該叫一次：
// 有些腳本改完就不再選人，只靠換人時抄回來會漏掉。
func (b *CharacterBinding) Flush(machine *eclvm.Machine) {
	if b == nil || machine == nil || b.current < 0 {
		return
	}
	b.window.CommitPlatinum(b.current, machine.Memory[CharacterPlatinumAddress])
}

func (b *CharacterBinding) project(index int, memory map[uint16]uint16,
	strings map[uint16]string) error {
	// 先抄回去再讀新的：同一個人再選一次時，腳本剛改過的值才不會被
	// 舊值蓋掉（`ADD 128` 之後那次重選就是這個情形）。
	if b.current >= 0 {
		b.window.CommitPlatinum(b.current, memory[CharacterPlatinumAddress])
	}
	character, ok := b.window.Character(index)
	if !ok {
		// The DOS handler leaves DS:5CF0/5CF2 unchanged when the linked-list
		// walk reaches nil, so the prior projection must remain intact.
		return nil
	}
	strings[0x6B00] = character.Name
	memory[0x6C00] = 1
	memory[0x6BB8] = uint16(character.ControlMorale)
	memory[CharacterPlatinumAddress] = character.Platinum
	b.current = index
	return nil
}

// staticCharacterWindow 是一份不會變的名單：投影得出去，寫不回來。
// 掃描與測試用的 session 走這一條。
type staticCharacterWindow []InitialCharacter

func (w staticCharacterWindow) Character(index int) (InitialCharacter, bool) {
	if index < 0 || index >= len(w) {
		return InitialCharacter{}, false
	}
	return w[index], true
}

func (w staticCharacterWindow) CommitPlatinum(int, uint16) {}

// SetCharacterWindow 換掉 session 的角色來源，讓腳本看到的是活的隊伍資料，
// 並把綁定交回呼叫端——`39h WHO` 與收尾的 Flush 都要用同一個。
func SetCharacterWindow(session *eclvm.BlockSession, window CharacterWindow) (*CharacterBinding, error) {
	if session == nil || session.Machine() == nil {
		return nil, fmt.Errorf("Pool ECL session is nil")
	}
	if window == nil {
		return nil, fmt.Errorf("Pool character window is nil")
	}
	binding := NewCharacterBinding(window)
	session.Machine().SetCharacterProjector(binding.Projector())
	return binding, nil
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
		MonsterID:        uint16(monster.Operands[0].Low),
		SpriteBlock:      monster.Operands[0].Low,
		ApproachDistance: monster.Operands[1].Low,
		PortraitBody:     monster.Operands[2].Low,
		Message:       ecl.DecodePackedText(message.Operands[0].Packed),
		ContinueLabel: menu.OptionTexts[0],
		Tour:          tour,
		ScriptBlock:   append([]byte(nil), block...),
		ScriptBlocks:  scriptBlocks,
	}, nil
}

// InitialEventPassthrough 讓盤點工具讀到同一份清單，不必自己再抄一次。
func InitialEventPassthrough() map[byte]bool { return initialEventPassthrough() }

// CellEventPassthrough 是格子事件那條路要讓前端看到的 opcode。
// 掃描工具與前端共用同一份，兩邊看到的邊界才會一致。
func CellEventPassthrough() map[byte]bool { return initialEventPassthrough() }

// NewCellSweepSession 為任何一個 ECL 區塊建一個乾淨的 session，供整包掃描用。
//
// 這**不是**正常遊玩：變數都從 0 開始，沒有走過主線，所以它只回答
// 「每一格的入口跑不跑得動、停在哪一種邊界」，不回答「玩家走得到嗎」。
func NewCellSweepSession(archive ECLArchive, blockID uint16,
	characters ...InitialCharacter) (*eclvm.BlockSession, error) {
	if len(archive.Blocks) == 0 {
		return nil, fmt.Errorf("Pool ECL archive %d has no blocks", archive.Number)
	}
	if _, ok := archive.Blocks[blockID]; !ok {
		return nil, fmt.Errorf("Pool ECL archive %d has no block %d", archive.Number, blockID)
	}
	session, err := eclvm.NewBlockSession(archive.Blocks, blockID, 0x9900, 0, 5,
		initialEventPassthrough(), 1)
	if err != nil {
		return nil, err
	}
	// 解碼用 Pool 自己量出來的指令表，不是共用 engine 那張二手的（spec 093）。
	session.Machine().SetCommands(PoolCommandTable())
	// 沒有投影器的話 `1Ch LOAD CHARACTER` 會直接報錯，而正常遊玩那條路是有的
	// （`NewInitialEventSession` 會接）。掃描要量的是腳本，不是缺投影器。
	session.Machine().SetCharacterProjector(initialCharacterProjector(characters))
	session.Machine().SetPartyStrengthResolver(initialPartyStrengthResolver(characters))
	return session, nil
}

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
		0x38: true, // PROGRAM（spec 081；前端要接，值 0 與值 9 都接上了）
		0x34: true, // ECL CLOCK（spec 093；推進遊戲時鐘）
		0x36: true, // ADD NPC（spec 091；把 NPC 加進隊伍）
		0x39: true, // WHO（spec 090；挑一個隊伍成員當「目前角色」）
		0x3B: true, // SPELL（spec 094；找隊上誰記了某個法術）
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
	session.Machine().SetCommands(PoolCommandTable())
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
	window := staticCharacterWindow(append([]InitialCharacter(nil), characters...))
	return NewCharacterBinding(window).Projector()
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

// RolfPortraitHeadBlock 是 Rolf 半身像的 HEAD 區塊。
//
// **head 選擇子不在 ECL 裡**：原版是 `SETUP MONSTER` 存下 body 區塊，head 則來自
// 那個結構的 `+5C2h`（overlay-7 `0699h` 把它當第一個參數傳給 overlay-29 entry 9
// 的 `HEAD` 檔）。整包 38 顆 overlay 與 `START.EXE` 逐位元組掃過 `5C2h` 這個位移，
// 唯一的寫入是 overlay-7 `0233h` 在 ECL block 初始化時寫 `0FFh`——所以真正的
// producer 一定是用別的定址形式（算出來的偏移或整塊搬移）寫進去的，那是掃描面的
// 洞，不是「沒有 producer」。
//
// 這個 8 因此是**從原版畫面量出來的**：`docs/reference/original-dos/adventure/`
// 的 `03-rolf-approach.png` 裁下 `(24,24)` 起的 88×88，上半 88×40 與
// `HEAD3.DAX` 區塊 8 逐格 100% 相同，下半 88×48 與 `BODY3.DAX` 區塊 9
// 逐格 100% 相同（後者也正好等於 ECL operand）。負對照：八個 `PIC*.DAX`
// 的每一張 88×88 都比不到 60%，所以那張圖不可能來自 `PIC` 那條路。
const RolfPortraitHeadBlock = 8
