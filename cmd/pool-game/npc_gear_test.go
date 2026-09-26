package main

// 隊伍裡的 NPC 開打時也跑 overlay-25 entry 7（#97，spec 147〈NPC〉）：overlay-10 `1380h`
// 沿 `5CF4h` 從隊員開始逐筆呼叫（`13A7h`），前後沒有分支；NPC 的物品是 ADD NPC
// 從 MONnITM 載進來的（overlay-17 entry 9 `1244h` → `0E90h`，spec 142）。

import (
	"path/filepath"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
	"github.com/wicanr2/golden-box-remake-engine/eclvm"
)

// ECL3/b0 `A046h` 是 `ADD NPC 6Bh`（DIRTEN，spec 091 的七個呼叫點之一）。
func TestAddNPCLoadsTheMonsterItemChain(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	archive, ok := application.eclCatalog.Archive(3)
	if !ok {
		t.Fatal("ECL3 archive is absent")
	}
	session, err := gamepack.NewDOSECLArchiveSession(archive, 0, 0x9914)
	if err != nil {
		t.Fatal(err)
	}
	application.eventSession, application.eventMachine = session, session.Machine()
	application.eclArchive = 3
	application.spawn = gamepack.Spawn{Map: gamepack.MapKey{Archive: 3, BlockID: 0}, X: 0, Y: 4, Facing: 3}
	if err := application.configureEventSession(session); err != nil {
		t.Fatal(err)
	}
	application.saveState = func(poolsave.State) error { return nil }
	// 事件的 PC 是 payload 內的位移（映射基準 9900h）。
	const pc = 0xA046 - 0x9900
	instruction, err := application.eclInstruction(pc)
	if err != nil || len(instruction.Operands) != gamepack.AddNPCOperands ||
		instruction.Command.Opcode != gamepack.AddNPCOpcode {
		t.Fatalf("ECL3/b0 @A046h is %+v (%v), want ADD NPC", instruction, err)
	}
	if err := application.applyAddNPC(eclvm.Event{PC: pc, Opcode: gamepack.AddNPCOpcode}); err != nil {
		t.Fatal(err)
	}
	if len(application.state.Party) != 1 || !application.state.Party[0].NPC {
		t.Fatalf("party after ADD NPC: %+v", application.state.Party)
	}
	want, err := gamepack.ReadDOSMonsterItems(zipPath, 3, 0x6B)
	if err != nil {
		t.Fatal(err)
	}
	got := application.state.Party[0].Inventory
	if len(want) == 0 || len(got) != len(want) {
		t.Fatalf("NPC carries %d items, MON3ITM block 6Bh has %d", len(got), len(want))
	}
	// 順序照檔案（`1178h..11F7h` 逐節接在尾端），內容從 `+2Eh` 起原封不動。
	for index := range want {
		if string(got[index].Raw[0x2E:]) != string(want[index][0x2E:]) {
			t.Errorf("item %d differs from MON3ITM: % X vs % X", index, got[index].Raw[0x2E:], want[index][0x2E:])
		}
	}
}

// HERO（MON3CHA block 6Dh，雇傭兵表最高那一階）：穿著長劍 +1（26h）、盾（49h）、
// 帶甲 +1（3Ah），另帶一把沒拿在手上的短弓（2Bh）。開打時 entry 7 從記錄出發重算：
//
//	`+110h`：+2Dh 42，武器那一支（entry 1）加 +1 與 18/xx 的力量 → 44
//	`+111h`：樣板 53（檔案裡的殘值），`+0A9h` 50 起算，盾 1、甲 5+1 → 61（背後 AC 56）
//	`+115h..`：長劍 1d10，+1 加力量 +1 → 1d10+2
//	`+11Ch`：`+72h` 12，帶甲重 151..399 的那一段寫死成 9（`0240h`）
//
// 這些數字是 gamepack 那一支算的（`RecomputeMonsterCombatFields`，spec 147 的獸人家收據
// 逐位元組對過），這裡驗的是**NPC 上了戰場也走它**：修好之前盤面讀的是樣板，THAC0
// 是 `+2Dh + 力量` 43、AC 53、1d8、腳程 12。
func TestNPCGearIsRecomputedWhenTheBattleStarts(t *testing.T) {
	application, _ := orcHomeGearFixture(t)
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	record, err := application.loadMonster(3, 0x6D)
	if err != nil {
		t.Fatal(err)
	}
	items, err := application.npcItems(3, 0x6D)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 4 {
		t.Fatalf("HERO carries %d items, want 4", len(items))
	}
	hero := poolsave.Character{Name: "HERO", NPC: true, Record: append([]byte(nil), record.Raw[:]...),
		MaxHP: int(record.MaxHitPoints()), CurrentHP: int(record.CurrentHitPoints()), Inventory: items}
	// 同一場獸人家，第五個隊員換成 HERO（夾具的隊伍與角色庫共用底層陣列，先複製）。
	application.state.Party = append([]poolsave.Character(nil), application.state.Party...)
	application.state.Party[len(application.state.Party)-1] = hero
	application.tactical = nil
	if err := application.enterTacticalPreview(); err != nil {
		t.Fatal(err)
	}
	state := application.tactical
	index := npcBoardIndex(t, state, len(application.state.Party)-1)
	if state.THAC0[index] != 44 || state.ArmorClass[index] != 61 || state.BaseMovement[index] != 9 {
		t.Errorf("HERO on the board: THAC0 %d AC %d move %d; want 44 61 9",
			state.THAC0[index], state.ArmorClass[index], state.BaseMovement[index])
	}
	if state.AttackForms[index][0] != (combat.DamageDice{Count: 1, Sides: 10, Bonus: 2}) ||
		state.Damage[index] != state.AttackForms[index][0] {
		t.Errorf("HERO form 1 %+v damage %+v, want 1d10+2", state.AttackForms[index][0], state.Damage[index])
	}
	// 負對照：只讀記錄（#97 之前的做法）是另一組數字。
	template := &tacticalState{
		BaseMovement: make([]uint8, 2), HitPoints: make([]int, 2), THAC0: make([]uint8, 2),
		ArmorClass: make([]int, 2), Damage: make([]combat.DamageDice, 2),
		AttackForms: make([][gamepack.MonsterAttackSlots]combat.DamageDice, 2),
		AttackRates: make([][gamepack.MonsterAttackSlots]uint8, 2),
	}
	if err := applyNPCCombatStats(template, 1, hero); err != nil {
		t.Fatal(err)
	}
	if template.THAC0[1] != 43 || template.ArmorClass[1] != 53 || template.BaseMovement[1] != 12 ||
		template.AttackForms[1][0].Count != 1 || template.AttackForms[1][0].Sides != 8 {
		t.Errorf("record-only HERO: THAC0 %d AC %d move %d form %+v; want 43 53 12 1d8",
			template.THAC0[1], template.ArmorClass[1], template.BaseMovement[1], template.AttackForms[1][0])
	}
	// 攻擊次數 entry 7 不動，照記錄。
	if rate, _ := record.BaseAttackRate(1); state.AttackRates[index][0] != rate {
		t.Errorf("HERO attack rate %d, record %d", state.AttackRates[index][0], rate)
	}
	if state.AttackRange[index] != 1 {
		t.Errorf("HERO reach %d with a long sword in hand, want 1", state.AttackRange[index])
	}
	// 與 gamepack 那一支同一份結果：盤面沒有第二套算法。
	raws := make([][]byte, 0, len(items))
	for _, item := range items {
		raws = append(raws, item.Raw)
	}
	types, err := gamepack.ReadDOSItemTypeTable(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := gamepack.RecomputeMonsterCombatFields(record, raws, types)
	if err != nil {
		t.Fatal(err)
	}
	if state.THAC0[index] != runtime.Raw[gamepack.CurrentThac0Offset] ||
		state.ArmorClass[index] != int(runtime.Raw[gamepack.InternalArmourClassOffset]) ||
		state.BaseMovement[index] != runtime.Raw[gamepack.CurrentMovementOffset] {
		t.Errorf("board and entry 7 disagree")
	}

	// 戰鬥中換裝一樣重算（`1469h..1485h`）：放下長劍、拿起短弓，射程變成弓的。
	slot := len(application.state.Party) - 1
	member := &application.state.Party[slot]
	for i := range member.Inventory {
		switch member.Inventory[i].Raw[itemTypeOffset] {
		case 0x26:
			member.Inventory[i].Raw[itemReadyOffset] = 0
		case 0x2B:
			member.Inventory[i].Raw[itemReadyOffset] = 1
		}
	}
	state.Mover = uint8(index)
	if err := application.afterCombatItemChange(state, slot); err != nil {
		t.Fatal(err)
	}
	bow, err := application.itemTypes.Entry(0x2B)
	if err != nil {
		t.Fatal(err)
	}
	if state.AttackRange[index] != bow.AttackRange() || state.AttackRange[index] <= 1 {
		t.Errorf("HERO reach %d after readying the short bow, want %d", state.AttackRange[index], bow.AttackRange())
	}
	if state.AttackForms[index][0] == (combat.DamageDice{Count: 1, Sides: 10, Bonus: 2}) {
		t.Error("HERO still hits with the long sword after putting it away")
	}
}

// npcBoardIndex 把隊伍索引換成盤面索引。
func npcBoardIndex(t *testing.T, state *tacticalState, slot int) int {
	t.Helper()
	for index, partySlot := range state.PartySlot {
		if partySlot == slot {
			return index
		}
	}
	t.Fatalf("party slot %d is not on the board", slot)
	return 0
}
