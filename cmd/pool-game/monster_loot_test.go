package main

// 怪物的物品鏈（spec 142，issue #76）：真的 MONnITM 從 staging 一路進盤面、在 entry 3 被
// 用掉、戰後照 overlay-05 entry 2 進戰利品選單。盤面與選單全部從 `Update()` 送鍵。

import (
	"math/rand"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
	"github.com/wicanr2/golden-box-remake-engine/eclvm"
)

// stageSlumsFight 在貧民窟（ECL2 block 20）排一場 spawns，按 ENTER 開打。
func stageSlumsFight(t *testing.T, spawns []eclvm.MonsterSpawn) *app {
	t.Helper()
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	archive, ok := application.eclCatalog.Archive(2)
	if !ok {
		t.Fatal("ECL2 archive is absent")
	}
	session, err := gamepack.NewDOSECLArchiveSession(archive, 20, 0x9914)
	if err != nil {
		t.Fatal(err)
	}
	geoMap, ok := application.geometryCatalog.Map(gamepack.MapKey{Archive: 2, BlockID: 20})
	if !ok {
		t.Fatal("GEO2/20 is absent")
	}
	application.eventSession, application.eventMachine = session, session.Machine()
	application.eclArchive = 2
	application.initialMap = &geoMap
	application.spawn = gamepack.Spawn{Map: geoMap.Key, X: 4, Y: 4, Facing: 0}
	application.introDone, application.mode = true, modeAdventure
	if err := application.configureEventSession(session); err != nil {
		t.Fatal(err)
	}
	application.saveState = func(poolsave.State) error { return nil }
	application.roller = diceRoller{random: rand.New(rand.NewSource(3))}
	party := make([]poolsave.Character, 0, 4)
	for index := 0; index < 4; index++ {
		party = append(party, poolsave.Character{Name: string(rune('B' + index)), RaceID: "dwarf",
			GenderID: "male", ClassID: "fighter", AlignmentID: "lawful-good",
			Abilities: [6]int{15, 10, 10, 13, 10, 10}, MaxHP: 30, CurrentHP: 30,
			PortraitHead: 1, PortraitBody: 1, IconSize: 1})
	}
	application.state = poolsave.State{Schema: poolsave.Schema, CharacterLibrary: party, Party: party}
	if err := application.enterCombatStaging(spawns); err != nil {
		t.Fatal(err)
	}
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if application.tactical == nil {
		t.Fatal("no tactical state")
	}
	return application
}

// foeIndexes 是盤面上敵方那幾格，照 roster 順序。
func foeIndexes(state *tacticalState) []int {
	var foes []int
	for index := 1; index < len(state.Roster); index++ {
		if !state.Friendly[index] {
			foes = append(foes, index)
		}
	}
	return foes
}

// LEVEL 3 MU（mon2/94）身上那支杖（值 35000、穿戴中、67 次）在 entry 3 被挑中：
// 印 "USES AN ITEM"、次數減一，杖還在身上。
func TestStagedMagicUserUsesItsWand(t *testing.T) {
	application := stageSlumsFight(t, []eclvm.MonsterSpawn{{MonsterID: 94, Count: 1, IconBlock: 4}})
	state := application.tactical
	foes := foeIndexes(state)
	if len(foes) != 1 {
		t.Fatalf("foes %v, want one", foes)
	}
	wizard := foes[0]
	items := state.FoeItems[wizard]
	if len(items) != 1 || !strings.HasPrefix(items[0].Name, "Wand") {
		t.Fatalf("the magic-user carries %+v, want its wand", items)
	}
	charges := items[0].Raw[gamepack.AIItemChargesOffset]
	used := ""
	for tick := 0; tick < 400 && used == "" && application.tactical != nil; tick++ {
		key := ebiten.KeyEnter
		if state.Prompt {
			key = ebiten.KeyN
		}
		moverBefore := state.Mover
		if err := press(application, key); err != nil {
			t.Fatalf("tick %d: %v", tick, err)
		}
		if int(moverBefore) == wizard && strings.Contains(state.FoeLog, "USES AN ITEM") {
			used = state.FoeLog
		}
	}
	if used == "" {
		t.Fatal("the magic-user never used its wand")
	}
	left := state.FoeItems[wizard]
	if len(left) != 1 || left[0].Raw[gamepack.AIItemChargesOffset] != charges-1 {
		t.Fatalf("after %q the wand is %+v, want %d charges", used, left, charges-1)
	}
	t.Logf("%s; %d charges left", used, charges-1)
}

// finishWithLoot 把兩隻狗頭人首領與一個 LEVEL 3 MU 打完：第一隻投降、第二隻逃掉、
// 法師倒下，然後在 "Continue Battle:" 按 N。價值 0 的物品擲 Roll(1, 10) 一律給 1。
func finishWithLoot(t *testing.T, noItems bool) (*app, gamepack.MonsterRecord, gamepack.MonsterRecord) {
	t.Helper()
	application := stageSlumsFight(t, []eclvm.MonsterSpawn{
		{MonsterID: 1, Count: 2, IconBlock: 4}, {MonsterID: 94, Count: 1, IconBlock: 4}})
	state := application.tactical
	foes := foeIndexes(state)
	if len(foes) != 3 {
		t.Fatalf("foes %v, want three", foes)
	}
	// 同一條 LOAD MONSTER 的第二隻，物品順序是反的（overlay-03 `0684h`）。
	first, second := state.FoeItems[foes[0]], state.FoeItems[foes[1]]
	if len(first) != 5 || len(second) != 5 ||
		first[0].Raw[gamepack.ItemTypeOffset] != second[4].Raw[gamepack.ItemTypeOffset] ||
		first[4].Raw[gamepack.ItemTypeOffset] != second[0].Raw[gamepack.ItemTypeOffset] {
		t.Fatalf("copy order: first %v second %v", itemTypes(first), itemTypes(second))
	}
	state.leaveBoard(uint8(foes[0]), gamepack.SurrenderedState)
	state.leaveBoard(uint8(foes[1]), gamepack.FledState)
	state.leaveBoard(uint8(foes[2]), gamepack.SurrenderedState)
	state.Prompt = true
	application.roller = fixedRoller{1}
	if noItems {
		application.eventMachine.Memory[monsterLootNoItemsAddress] = 1
	}
	if err := press(application, ebiten.KeyN); err != nil {
		t.Fatal(err)
	}
	kobold := application.combatMonstersRecordFor(t, 2, 1)
	wizard := application.combatMonstersRecordFor(t, 2, 94)
	return application, kobold, wizard
}

func (a *app) combatMonstersRecordFor(t *testing.T, archive, block uint8) gamepack.MonsterRecord {
	t.Helper()
	record, err := a.loadMonster(archive, block)
	if err != nil {
		t.Fatal(err)
	}
	return record
}

func itemTypes(items []poolsave.Item) []uint8 {
	types := make([]uint8, 0, len(items))
	for _, item := range items {
		types = append(types, item.Raw[gamepack.ItemTypeOffset])
	}
	return types
}

func monsterMoney(record gamepack.MonsterRecord) [7]uint32 {
	var money [7]uint32
	for currency := range money {
		offset := gamepack.MonsterMoneyOffset + currency*2
		money[currency] = uint32(record.Raw[offset]) | uint32(record.Raw[offset+1])<<8
	}
	return money
}

// 投降的與倒下的交出錢與物品，逃掉的不算（`0079h`）；物品插在串列頭，所以最後收的
// 法師那支杖排第一。選單從 Update() 走 Take → Items → 杖 → 第一個隊員。
func TestMonsterLootEntersTheTreasureMenu(t *testing.T) {
	application, kobold, wizard := finishWithLoot(t, false)
	if !application.treasureActive || application.tactical != nil {
		t.Fatalf("treasure %v tactical %v after the win", application.treasureActive, application.tactical != nil)
	}
	want := monsterMoney(kobold)
	for currency, amount := range monsterMoney(wizard) {
		want[currency] += amount
	}
	if application.state.PooledMoney != want {
		t.Fatalf("pooled %v, want %v (one kobold leader and the magic-user)", application.state.PooledMoney, want)
	}
	loot := application.treasureItems
	if len(loot) != 6 {
		t.Fatalf("%d items on the pile, want the wand and five kobold items", len(loot))
	}
	if !strings.HasPrefix(loot[0].Name, "Wand") {
		t.Fatalf("head of the pile is %q, want the wand collected last", loot[0].Name)
	}
	wantTypes := []byte{0x34, 0x3B, 0x25, 0x49, 0x2C}
	for index, itemType := range wantTypes {
		if got := loot[index+1].Raw[gamepack.ItemTypeOffset]; got != itemType {
			t.Fatalf("pile item %d type %02X, want %02X", index+1, got, itemType)
		}
	}
	for _, item := range loot {
		if item.Raw[gamepack.ItemReadiedOffset] != 0 {
			t.Fatalf("%q is still readied on the pile", item.Name)
		}
	}
	if application.eventMachine.Memory[monsterLootNoItemsAddress] != 0 {
		t.Fatal("@6DE3 was not cleared after the fight")
	}
	if err := selectMenuOption(t, application, "Take"); err != nil {
		t.Fatal(err)
	}
	if err := selectMenuOption(t, application, "Items"); err != nil {
		t.Fatal(err)
	}
	if err := press(application, ebiten.KeyEnter); err != nil { // 杖
		t.Fatal(err)
	}
	if err := press(application, ebiten.KeyEnter); err != nil { // 第一個隊員
		t.Fatal(err)
	}
	inventory := application.state.Party[0].Inventory
	if len(inventory) != 1 || !strings.HasPrefix(inventory[0].Name, "Wand") {
		t.Fatalf("the first member carries %+v, want the wand", inventory)
	}
	if err := application.state.Validate(); err != nil {
		t.Fatalf("the taken wand does not save: %v", err)
	}
	if len(application.treasureItems) != 5 {
		t.Fatalf("%d items left on the pile, want 5", len(application.treasureItems))
	}
}

// ECL @6DE3 == 1（`[4937h]+5C6h`，`00F7h`）：錢照收，物品整段不收。
func TestMonsterLootSkipsItemsWhenTheScriptSaysSo(t *testing.T) {
	application, kobold, wizard := finishWithLoot(t, true)
	if !application.treasureActive || len(application.treasureItems) != 0 {
		t.Fatalf("treasure %v with %d items, want money only", application.treasureActive,
			len(application.treasureItems))
	}
	want := monsterMoney(kobold)
	for currency, amount := range monsterMoney(wizard) {
		want[currency] += amount
	}
	if application.state.PooledMoney != want {
		t.Fatalf("pooled %v, want %v", application.state.PooledMoney, want)
	}
	if application.eventMachine.Memory[monsterLootNoItemsAddress] != 0 {
		t.Fatal("@6DE3 was not cleared after the fight")
	}
}

// leaveTreasureMenu 是走路駕駛用的：戰利品選單開著就挑 Exit、確認答 Yes，直到選單
// 關掉為止（離開之後腳本若再開一個也一併離開）。回傳有沒有動過。
func leaveTreasureMenu(a *app) bool {
	acted := false
	for guard := 0; guard < 64 && a.treasureActive; guard++ {
		acted = true
		want := "Exit"
		switch a.treasureStage {
		case treasureConfirmExit:
			want = "Yes"
		case treasureMain:
		default:
			press(a, ebiten.KeyEscape)
			continue
		}
		for turn := 0; turn < 12 && len(a.cellMenuOptions) != 0 &&
			a.cellMenuOptions[a.cellMenuCursor] != want; turn++ {
			press(a, ebiten.KeyArrowRight)
		}
		press(a, ebiten.KeyEnter)
	}
	return acted
}

// leaveCombatLoot 是打完之後玩家對怪物的戰利品選單按 Exit、有確認就答 Yes，把東西
// 留在原地（spec 142）。只離開這一個選單：離開之後戰後腳本若又開了一個（委任獎賞
// 那種 `TREASURE`），留給呼叫端。
func leaveCombatLoot(t *testing.T, application *app) {
	t.Helper()
	if !application.treasureActive {
		return
	}
	if err := selectMenuOption(t, application, "Exit"); err != nil {
		t.Fatalf("leave the monster loot: %v", err)
	}
	if application.treasureActive && application.treasureStage == treasureConfirmExit {
		if err := selectMenuOption(t, application, "Yes"); err != nil {
			t.Fatalf("leave the monster loot: %v", err)
		}
	}
}
