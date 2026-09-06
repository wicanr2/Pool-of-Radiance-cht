package main

import (
	"errors"
	"image"
	"image/color"
	"math/rand"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/creation"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
	"github.com/wicanr2/golden-box-remake-engine/eclvm"
	"github.com/wicanr2/golden-box-remake-engine/graphics"
	"github.com/wicanr2/golden-box-remake-engine/viewport"
)

type scriptedKeys map[ebiten.Key]bool

type fixedTempleRoller int

func (value fixedTempleRoller) Roll(count, sides int) int { return int(value) }

func (keys scriptedKeys) JustPressed(key ebiten.Key) bool {
	pressed := keys[key]
	delete(keys, key)
	return pressed
}

type scriptedTextKeys struct {
	scriptedKeys
	chars []rune
}

func (keys *scriptedTextKeys) JustPressed(key ebiten.Key) bool {
	return keys.scriptedKeys.JustPressed(key)
}
func (keys *scriptedTextKeys) Chars() []rune { result := keys.chars; keys.chars = nil; return result }

// scriptedChars 餵一串字元給輸入列，按鍵一律沒按。
func scriptedChars(text string) *scriptedTextKeys {
	return &scriptedTextKeys{scriptedKeys: scriptedKeys{}, chars: []rune(text)}
}

func press(application *app, key ebiten.Key) error {
	application.keys = scriptedKeys{key: true}
	return application.Update()
}

func TestKeysDriveTitleToOriginalCharacterSheet(t *testing.T) {
	application := &app{
		mode:   modeTitle,
		flow:   creation.NewFlow(),
		roller: diceRoller{random: rand.New(rand.NewSource(1))},
	}
	for _, key := range []ebiten.Key{
		ebiten.KeyEnter, // title -> menu
		ebiten.KeyC,     // menu -> race
		ebiten.KeyEnter, // Dwarf
		ebiten.KeyEnter, // Male
		ebiten.KeyEnter, // Fighter
		ebiten.KeyEnter, // Lawful Good
	} {
		if err := press(application, key); err != nil {
			t.Fatal(err)
		}
	}
	// The roll page generates on its first update, matching the real frame path.
	application.keys = scriptedKeys{}
	if err := application.Update(); err != nil {
		t.Fatal(err)
	}
	if application.mode != modeCreation || application.flow.Stage != creation.StageRoll || application.rolled == nil {
		t.Fatalf("normal key path stopped at mode=%d stage=%d result=%v", application.mode, application.flow.Stage, application.rolled)
	}
}

func TestGlobalHelpThemeAndQuitKeys(t *testing.T) {
	application := &app{mode: modeMenu}
	if err := press(application, ebiten.KeyF2); err != nil || !application.modern {
		t.Fatalf("F2 theme: modern=%t err=%v", application.modern, err)
	}
	if err := press(application, ebiten.KeyF1); err != nil || !application.help {
		t.Fatalf("F1 help: help=%t err=%v", application.help, err)
	}
	application.help = false
	if err := press(application, ebiten.KeyF10); !errors.Is(err, ebiten.Termination) {
		t.Fatalf("F10=%v", err)
	}
}

func TestF10AndLoadRoundTripStableCampaignSession(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	character := poolsave.Character{Name: "HERO", RaceID: "dwarf", GenderID: "male", ClassID: "fighter", AlignmentID: "lawful-good", Abilities: [6]int{14, 10, 10, 13, 10, 10}, MaxHP: 8, CurrentHP: 8, PortraitHead: 1, PortraitBody: 1, IconSize: 1}
	application.state = poolsave.State{Schema: poolsave.Schema, CharacterLibrary: []poolsave.Character{character}, Party: []poolsave.Character{character}}
	session, err := gamepack.NewInitialEventSession(*application.initialEvent, gamepack.InitialCharacter{Name: "HERO", ClassID: "fighter", Abilities: character.Abilities, CurrentHP: 8})
	if err != nil {
		t.Fatal(err)
	}
	application.mode = modeAdventure
	application.eventSession, application.eventMachine = session, session.Machine()
	application.spawn = gamepack.Spawn{Map: gamepack.MapKey{Archive: 3, BlockID: 0}, X: 5, Y: 5, Facing: 2}
	application.eventMachine.Memory[0x4AC1] = 4
	application.eventMachine.Memory[0x4AB1] = 0
	application.eventMachine.Memory[0x4A96] = 0
	var saved poolsave.State
	application.saveState = func(state poolsave.State) error { saved = cloneSaveState(state); return nil }
	if err := press(application, ebiten.KeyF10); !errors.Is(err, ebiten.Termination) {
		t.Fatalf("F10 campaign save=%v", err)
	}
	if saved.Campaign == nil || saved.Campaign.X != 5 || saved.Campaign.Session.Machine.Memory == nil {
		t.Fatalf("saved campaign=%+v", saved.Campaign)
	}

	restored, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	restored.mode = modeMenu
	restored.loadState = func() (poolsave.State, error) { return cloneSaveState(saved), nil }
	if err := press(restored, ebiten.KeyL); err != nil {
		t.Fatal(err)
	}
	if restored.mode != modeAdventure || restored.spawn.X != 5 || restored.spawn.Y != 5 || restored.spawn.Facing != 2 || restored.introWaiting || restored.eventMachine.Memory[0x4AC1] != 4 || restored.eventSession.CurrentBlockID() != saved.Campaign.Session.Current {
		t.Fatalf("restored mode=%d spawn=%+v intro=%v block=%d flags=%d", restored.mode, restored.spawn, restored.introWaiting, restored.eventSession.CurrentBlockID(), restored.eventMachine.Memory[0x4AC1])
	}
}

func TestF10AndLoadRoundTripECL2SlumsNamespace(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	archive, ok := application.eclCatalog.Archive(2)
	if !ok {
		t.Fatal("ECL2 archive is absent")
	}
	session, err := gamepack.NewDOSECLArchiveSession(archive, 20, 0xB69C)
	if err != nil {
		t.Fatal(err)
	}
	application.mode, application.introDone = modeAdventure, true
	application.eventSession, application.eventMachine = session, session.Machine()
	application.eclArchive = 2
	application.spawn = gamepack.Spawn{Map: gamepack.MapKey{Archive: 2, BlockID: 20}, X: 3, Y: 4, Facing: 2}
	application.eventMachine.Memory[0x4ABB] = 24
	var saved poolsave.State
	application.saveState = func(state poolsave.State) error { saved = cloneSaveState(state); return state.Validate() }
	if err := press(application, ebiten.KeyF10); !errors.Is(err, ebiten.Termination) {
		t.Fatalf("save ECL2 campaign: %v", err)
	}
	if saved.Campaign == nil || saved.Campaign.ECLArchive != 2 || saved.Campaign.Session.Current != 20 {
		t.Fatalf("saved ECL2 campaign=%+v", saved.Campaign)
	}
	restored, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	restored.mode = modeMenu
	restored.loadState = func() (poolsave.State, error) { return cloneSaveState(saved), nil }
	if err := press(restored, ebiten.KeyL); err != nil {
		t.Fatal(err)
	}
	if restored.eclArchive != 2 || restored.eventSession.CurrentBlockID() != 20 || restored.spawn.Map != (gamepack.MapKey{Archive: 2, BlockID: 20}) || restored.eventMachine.Memory[0x4ABB] != 24 {
		t.Fatalf("restored ECL archive/block/map/4ABB=%d/%d/%+v/%d", restored.eclArchive, restored.eventSession.CurrentBlockID(), restored.spawn.Map, restored.eventMachine.Memory[0x4ABB])
	}
}

func TestRealSlumsCombatStagesMonsterRecords(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	archive, ok := application.eclCatalog.Archive(2)
	if !ok {
		t.Fatal("ECL2 archive is absent")
	}
	session, err := gamepack.NewDOSECLArchiveSession(archive, 20, 0x9E5D)
	if err != nil {
		t.Fatal(err)
	}
	application.mode, application.introDone = modeAdventure, true
	application.eventSession, application.eventMachine = session, session.Machine()
	application.eclArchive = 2
	application.spawn = gamepack.Spawn{Map: gamepack.MapKey{Archive: 2, BlockID: 20}, X: 3, Y: 4, Facing: 2}
	result, err := session.RunUntilEvent(16, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := application.consumeInitialSearch(result); err != nil {
		t.Fatal(err)
	}
	if !application.combatActive || !application.cellEventPending || application.cellWaitingMenu || len(application.combatMonsters) != 2 {
		t.Fatalf("combat active/pending/menu/monsters=%t/%t/%t/%+v", application.combatActive, application.cellEventPending, application.cellWaitingMenu, application.combatMonsters)
	}
	if application.combatMonsters[0].Record.Name != "ORC" || application.combatMonsters[0].Spawn.Count != 1 || application.combatMonsters[0].Spawn.IconBlock != 4 || application.combatMonsters[1].Record.Name != "ORC" || application.combatMonsters[1].Spawn.Count != 3 || !strings.Contains(application.eventText, "ORC ×1 / ORC ×3") {
		t.Fatalf("staged combat=%+v text=%q", application.combatMonsters, application.eventText)
	}
	if got := uint16(0x9900 + session.Machine().PC); got != 0x9E6D || session.Machine().Memory[0x4ABB] != 0 {
		t.Fatalf("combat staging advanced PC/state to 0x%04X / %d", got, session.Machine().Memory[0x4ABB])
	}
	// ENTER 進的是戰術戰鬥，不是把 ECL 推過去；PC 要留在 COMBAT 邊界後的那一條，
	// 由戰鬥結果決定續不續跑（spec 046 契約 5）。
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if !application.tacticalPreview || application.tactical == nil {
		t.Fatal("ENTER did not enter tactical combat")
	}
	if got := uint16(0x9900 + session.Machine().PC); got != 0x9E6D {
		t.Fatalf("entering combat advanced the ECL PC to 0x%04X", got)
	}
}

// 打贏真實的 Slums 遭遇之後，停在 COMBAT 邊界的 ECL session 要真的往下跑。
// 這是整條垂直鏈裡唯一沒有被單元測試涵蓋的一段：真的 archive、真的 session、
// 真的怪物記錄，直到戰後腳本繼續為止。
func TestWinningTheRealSlumsCombatResumesTheECLScript(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	archive, ok := application.eclCatalog.Archive(2)
	if !ok {
		t.Fatal("ECL2 archive is absent")
	}
	session, err := gamepack.NewDOSECLArchiveSession(archive, 20, 0x9E5D)
	if err != nil {
		t.Fatal(err)
	}
	member := poolsave.Character{
		Name: "HERO", RaceID: "dwarf", GenderID: "male", ClassID: "fighter",
		AlignmentID: "lawful-good", MaxHP: 12, CurrentHP: 12,
	}
	application.mode, application.introDone = modeAdventure, true
	application.eventSession, application.eventMachine = session, session.Machine()
	application.eclArchive = 2
	application.state.Party = []poolsave.Character{member}
	application.spawn = gamepack.Spawn{Map: gamepack.MapKey{Archive: 2, BlockID: 20}, X: 3, Y: 4, Facing: 2}
	result, err := session.RunUntilEvent(16, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := application.consumeInitialSearch(result); err != nil {
		t.Fatal(err)
	}
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	state := application.tactical
	if state == nil {
		t.Fatal("ENTER did not enter tactical combat")
	}

	// 把場上的敵人全部打倒——這裡直接改盤面而不是模擬走位，因為要測的是
	// 「贏了之後會不會續跑」，不是走位本身（走位有自己的測試）。
	for index := 1; index < len(state.Roster); index++ {
		if !state.Friendly[index] {
			state.Roster[index].FootprintClass = 0
			state.Scores[index] = 0
			state.States[index] = combat.DyingState
		}
	}
	state.Mover, state.Prompt = 0, false
	state.endRound(application.rollDice)
	if !state.Prompt {
		t.Fatal("clearing the foes did not raise the continue prompt")
	}
	if got := uint16(0x9900 + session.Machine().PC); got != 0x9E6D {
		t.Fatalf("the prompt already advanced the ECL PC to 0x%04X", got)
	}

	// 答 N 結束戰鬥，戰後腳本才續跑。
	if err := press(application, ebiten.KeyN); err != nil {
		t.Fatal(err)
	}
	if application.tacticalPreview || application.tactical != nil {
		t.Fatal("the tactical screen stayed open after the battle ended")
	}
	if application.combatActive {
		t.Fatal("the encounter is still staged after a win")
	}
	if got := uint16(0x9900 + session.Machine().PC); got == 0x9E6D {
		t.Fatal("the post-combat script did not run after the win")
	}
}

// 對話或服務進行中按 F10：不寫檔，把理由寫進狀態列，而且**不能結束程式**。
// `Update` 回傳非 Termination 的 error 時 ebiten 會直接收掉視窗——玩家在
// 對話中按一下 F10 遊戲就沒了。
func TestF10RejectsTransientCampaignWithoutWriting(t *testing.T) {
	application := &app{mode: modeAdventure, cellEventPending: true}
	called := false
	application.saveState = func(poolsave.State) error { called = true; return nil }
	if err := press(application, ebiten.KeyF10); err != nil || called {
		t.Fatalf("transient F10 err=%v called=%v", err, called)
	}
	if application.statusLine == "" {
		t.Fatal("transient F10 refused to save without saying why")
	}
}

func TestSuneTempleCureUsesCurrentCharacterAndPersists(t *testing.T) {
	character := poolsave.Character{Name: "HERO", RaceID: "dwarf", GenderID: "male", ClassID: "fighter", AlignmentID: "lawful-good", Money: [7]uint16{3: 100}, MaxHP: 12, CurrentHP: 2, PortraitHead: 1, PortraitBody: 1, IconSize: 1}
	application := &app{
		roller:       fixedTempleRoller(6),
		state:        poolsave.State{Schema: poolsave.Schema, PooledMoney: [7]uint32{3: 400}, CharacterLibrary: []poolsave.Character{character}, Party: []poolsave.Character{character}},
		templeActive: true, cellEventPending: true, cellWaitingMenu: true,
	}
	var saved poolsave.State
	application.saveState = func(state poolsave.State) error { saved = state; return nil }
	application.enterTempleMain()
	if err := application.selectSuneTempleOption(); err != nil || application.templeStage != templeHeal || len(application.cellMenuOptions) != 10 {
		t.Fatalf("enter Heal stage=%d options=%v err=%v", application.templeStage, application.cellMenuOptions, err)
	}
	application.cellMenuCursor = 2
	if err := application.selectSuneTempleOption(); err != nil || application.templeStage != templeConfirm || !strings.Contains(application.eventText, "100 gold pieces") {
		t.Fatalf("confirm stage=%d text=%q err=%v", application.templeStage, application.eventText, err)
	}
	if err := application.selectSuneTempleOption(); err != nil {
		t.Fatal(err)
	}
	if application.state.Party[0].Money[3] != 0 || application.state.PooledMoney[3] != 400 || application.state.Party[0].CurrentHP != 8 || saved.Party[0].CurrentHP != 8 || !strings.Contains(application.eventText, "HERO is cured") {
		t.Fatalf("state=%+v saved=%+v text=%q", application.state, saved, application.eventText)
	}
}

// templeService 是 H）EAL 選單上的位置（spec 115 的九項），不是
// WoundServices 的索引——第 2 項才是治療輕傷。
func TestSuneTempleFailedSaveRollsBackCure(t *testing.T) {
	character := poolsave.Character{Name: "HERO", RaceID: "dwarf", GenderID: "male", ClassID: "fighter", AlignmentID: "lawful-good", MaxHP: 12, CurrentHP: 2, PortraitHead: 1, PortraitBody: 1, IconSize: 1}
	application := &app{roller: fixedTempleRoller(6), state: poolsave.State{Schema: poolsave.Schema, PooledMoney: [7]uint32{3: 100}, CharacterLibrary: []poolsave.Character{character}, Party: []poolsave.Character{character}}, templeActive: true, templeStage: templeConfirm, templeService: 2, cellMenuOptions: []string{"YES", "NO"}}
	application.saveState = func(poolsave.State) error { return errors.New("disk full") }
	if err := application.selectSuneTempleOption(); err == nil {
		t.Fatal("save failure was swallowed")
	}
	if application.state.PooledMoney[3] != 100 || application.state.Party[0].CurrentHP != 2 || application.state.CharacterLibrary[0].CurrentHP != 2 {
		t.Fatalf("failed save did not roll back: %+v", application.state)
	}
}

func treasureRecord(name string, weight uint16) gamepack.TreasureItemRecord {
	var record gamepack.TreasureItemRecord
	record.Name = name
	record.Raw[0] = byte(len(name))
	copy(record.Raw[1:], name)
	record.Raw[0x37] = byte(weight)
	record.Raw[0x38] = byte(weight >> 8)
	return record
}

func TestTreasureTakePersistsBeforeRemovingPendingItem(t *testing.T) {
	character := poolsave.Character{Name: "HERO", RaceID: "dwarf", GenderID: "male", ClassID: "fighter", AlignmentID: "lawful-good", Abilities: [6]int{10, 10, 10, 10, 10, 10}, MaxHP: 8, CurrentHP: 8, PortraitHead: 1, PortraitBody: 1, IconSize: 1}
	record := treasureRecord("Two-Handed Sword +1", 250)
	application := &app{
		state:          poolsave.State{Schema: poolsave.Schema, CharacterLibrary: []poolsave.Character{character}, Party: []poolsave.Character{character}},
		treasureActive: true, treasureStage: treasureCharacter, treasureItems: []gamepack.TreasureItemRecord{record}, treasureSelected: 0,
	}
	var saved poolsave.State
	application.saveState = func(state poolsave.State) error { saved = state; return nil }
	if err := application.giveTreasureItem(0); err != nil {
		t.Fatal(err)
	}
	if len(application.treasureItems) != 0 || len(application.state.Party[0].Inventory) != 1 || len(application.state.CharacterLibrary[0].Inventory) != 1 || len(saved.Party[0].Inventory) != 1 {
		t.Fatalf("state=%+v saved=%+v pending=%v", application.state, saved, application.treasureItems)
	}
	if got := application.state.Party[0].Inventory[0]; got.Name != record.Name || !reflect.DeepEqual(got.Raw, record.Raw[:]) {
		t.Fatalf("inventory item=%+v", got)
	}
}

func TestGraveyardTreasureRequestEntersFiveItemService(t *testing.T) {
	items := []gamepack.TreasureItemRecord{
		treasureRecord("Scroll 1", 10), treasureRecord("Scroll 2", 10),
		treasureRecord("Scroll 3", 10), treasureRecord("Scroll 4", 10),
		treasureRecord("Two-Handed Sword +1 +3 vs. Undead", 250),
	}
	application := &app{spawn: gamepack.Spawn{Map: gamepack.MapKey{Archive: 3}}}
	application.loadTreasure = func(archive, block uint8) ([]gamepack.TreasureItemRecord, error) {
		if archive != 3 || block != 0x33 {
			t.Fatalf("load identity ITEM%d/%02X", archive, block)
		}
		return append([]gamepack.TreasureItemRecord(nil), items...), nil
	}
	if err := application.enterTreasure([]eclvm.TreasureRequest{{ItemBlock: 0x33}}); err != nil {
		t.Fatal(err)
	}
	if !application.treasureActive || application.treasureStage != treasureMain || !application.cellEventPending || !application.cellWaitingMenu || len(application.treasureItems) != 5 || !reflect.DeepEqual(application.cellMenuOptions, []string{"View", "Take", "Pool", "Share", "Exit"}) {
		t.Fatalf("treasure service=%+v options=%v", application, application.cellMenuOptions)
	}
	application.cellMenuCursor = 4
	if err := application.selectTreasureOption(); err != nil || application.treasureStage != treasureConfirmExit || !strings.Contains(application.eventText, "still treasure") {
		t.Fatalf("leave confirmation stage=%d text=%q err=%v", application.treasureStage, application.eventText, err)
	}
}

func TestSlumsLoadPiecesResourceReplacesAllThreeWallSlots(t *testing.T) {
	before := graphics.PieceSet{SetID: 1, Selector: 9}
	want := graphics.PieceSet{SetID: 1, WallDefs: make([]graphics.WallDef, 3)}
	application := &app{spawn: gamepack.Spawn{Map: gamepack.MapKey{Archive: 2, BlockID: 20}}, initialWalls: &before}
	var seen [3]uint8
	application.loadPieceSlots = func(archive uint8, selectors [3]uint8) (graphics.PieceSet, error) {
		if archive != 2 {
			t.Fatalf("LOAD PIECES archive=%d", archive)
		}
		seen = selectors
		return want, nil
	}
	event := eclvm.Event{Opcode: 0x37, Arguments: []uint16{2, 4, 1}, ArgumentsValid: []bool{true, true, true}}
	consumed, err := application.applyTransitionResource(event)
	if err != nil || !consumed || !reflect.DeepEqual(*application.initialWalls, want) {
		t.Fatalf("consumed=%v err=%v walls=%+v", consumed, err, application.initialWalls)
	}
	if seen != ([3]uint8{2, 4, 1}) {
		t.Fatalf("LOAD PIECES selectors=%v", seen)
	}
	// `FFh` 是「這一格不換」的哨兵（spec 043 的 handler 只對非 FFh 的 selector
	// 呼叫 LoadWallSet），要原封不動交給 loader，由它沿用上一份的同一格。
	event.Arguments[1] = 0xFF
	if consumed, err := application.applyTransitionResource(event); err != nil || !consumed {
		t.Fatalf("partial consumed=%v err=%v", consumed, err)
	}
	if seen != ([3]uint8{2, 0xFF, 1}) {
		t.Fatalf("FFh 沒有傳到 loader：%v", seen)
	}
}

func TestRealNewPhlanControllerCrossesFromECL3ToSlumsECL2(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	archive, ok := application.eclCatalog.Archive(3)
	if !ok {
		t.Fatal("ECL3 archive is absent")
	}
	session, err := gamepack.NewDOSECLArchiveSession(archive, 0, 0x9955)
	if err != nil {
		t.Fatal(err)
	}
	application.eventSession, application.eventMachine = session, session.Machine()
	application.eclArchive = 3
	application.spawn = gamepack.Spawn{Map: gamepack.MapKey{Archive: 3, BlockID: 0}, X: 0, Y: 4, Facing: 3}
	if err := application.configureEventSession(session); err != nil {
		t.Fatal(err)
	}
	result, err := session.RunUntilEvent(4096, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Events) != 1 || result.Events[0].Opcode != 0x21 || !reflect.DeepEqual(result.Events[0].Arguments, []uint16{0xFF, 0xFF, 0x7F}) {
		t.Fatalf("first controller boundary=%+v", result)
	}
	if _, err := application.consumeInitialTransitionResources(result); err != nil {
		t.Fatal(err)
	}
	if application.eclArchive != 2 || session.CurrentBlockID() != 20 || application.spawn.Map != (gamepack.MapKey{Archive: 2, BlockID: 20}) {
		t.Fatalf("archive/block/map=%d/%d/%+v", application.eclArchive, session.CurrentBlockID(), application.spawn.Map)
	}
	if application.eventMachine.Memory[0x6E12] != 2 || application.initialMap == nil || application.initialMap.Key != (gamepack.MapKey{Archive: 2, BlockID: 20}) {
		t.Fatalf("selector/map=%d/%+v", application.eventMachine.Memory[0x6E12], application.initialMap)
	}
	if application.initialWalls == nil || application.initialWalls.SetID != 1 || !reflect.DeepEqual(application.initialWalls.SymbolBlockIDs, []uint8{2, 4, 1}) || len(application.initialWalls.WallDefs) != 3 {
		t.Fatalf("Slums wall slots=%+v", application.initialWalls)
	}
}

func TestMoneyTreasureNormalMenuTakePoolAndShare(t *testing.T) {
	hero := poolsave.Character{Name: "HERO", RaceID: "human", GenderID: "male", ClassID: "fighter", AlignmentID: "lawful-good", Abilities: [6]int{18, 10, 10, 10, 10, 10}, Money: [7]uint16{3: 3}, MaxHP: 8, CurrentHP: 8, PortraitHead: 1, PortraitBody: 1, IconSize: 1}
	application := &app{spawn: gamepack.Spawn{Map: gamepack.MapKey{Archive: 3}}, state: poolsave.State{Schema: poolsave.Schema, CharacterLibrary: []poolsave.Character{hero}, Party: []poolsave.Character{hero}}}
	application.loadTreasure = func(archive, block uint8) ([]gamepack.TreasureItemRecord, error) { return nil, nil }
	application.saveState = func(state poolsave.State) error { return state.Validate() }
	if err := application.enterTreasure([]eclvm.TreasureRequest{{Amounts: [7]uint16{0: 2, 3: 7, 6: 1}}}); err != nil {
		t.Fatal(err)
	}
	if application.state.PooledMoney != ([7]uint32{0: 2, 3: 7, 6: 1}) {
		t.Fatalf("initial pools=%v", application.state.PooledMoney)
	}
	application.cellMenuCursor = 2
	if err := application.selectTreasureOption(); err != nil || application.state.Party[0].Money[3] != 0 || application.state.PooledMoney[3] != 10 {
		t.Fatalf("pool err=%v state=%+v", err, application.state)
	}
	application.cellMenuCursor = 3
	if err := application.selectTreasureOption(); err != nil || application.state.PooledMoney != ([7]uint32{}) || application.state.Party[0].Money != ([7]uint16{0: 2, 3: 10, 6: 1}) {
		t.Fatalf("share err=%v state=%+v", err, application.state)
	}
	application.state.PooledMoney[3] = 9
	if err := application.enterTreasureMoneyCurrencies(); err != nil {
		t.Fatal(err)
	}
	application.cellMenuCursor = 0
	if err := application.selectTreasureOption(); err != nil || application.treasureStage != treasureMoneyCharacter {
		t.Fatalf("currency err=%v stage=%d", err, application.treasureStage)
	}
	application.cellMenuCursor = 0
	if err := application.selectTreasureOption(); err != nil || application.treasureStage != treasureMoneyAmount {
		t.Fatalf("character err=%v stage=%d", err, application.treasureStage)
	}
	application.treasureAmount = "4"
	if err := application.takeTreasureMoney(); err != nil || application.state.PooledMoney[3] != 5 || application.state.Party[0].Money[3] != 14 {
		t.Fatalf("take err=%v state=%+v", err, application.state)
	}
}

func TestRealGraveyardTreasureBytesEnterFiveItemService(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	event, err := gamepack.ReadDOSInitialEvent(zipPath)
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	fixture := event
	fixture.HandlerAddress = 0xA780
	fixture.ScriptBlock = nil
	fixture.ScriptBlocks = map[uint16][]byte{0: event.ScriptBlocks[8]}
	session, err := gamepack.NewInitialEventSession(fixture)
	if err != nil {
		t.Fatal(err)
	}
	result, err := session.Machine().RunUntilEvent(8, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	application := &app{eventSession: session, eventMachine: session.Machine(), spawn: gamepack.Spawn{Map: gamepack.MapKey{Archive: 3}}}
	application.loadTreasure = func(archive, block uint8) ([]gamepack.TreasureItemRecord, error) {
		return gamepack.ReadDOSTreasureItemBlock(zipPath, archive, block)
	}
	if err := application.consumeInitialSearch(result); err != nil {
		t.Fatal(err)
	}
	if !application.treasureActive || len(application.treasureItems) != 5 || application.treasureItems[4].Name != "Two-Handed Sword +1 +3 vs. Undead" || len(result.Events) != 1 || result.Events[0].Opcode != 0x24 {
		t.Fatalf("result=%+v active=%v items=%v", result, application.treasureActive, application.treasureItems)
	}
}

func TestTreasureTakeOverloadAndSaveFailureKeepPendingItem(t *testing.T) {
	character := poolsave.Character{Name: "HERO", RaceID: "dwarf", GenderID: "male", ClassID: "fighter", AlignmentID: "lawful-good", Abilities: [6]int{3, 10, 10, 10, 10, 10}, MaxHP: 8, CurrentHP: 8, PortraitHead: 1, PortraitBody: 1, IconSize: 1}
	heavy := treasureRecord("Heavy", 1151)
	application := &app{state: poolsave.State{Schema: poolsave.Schema, CharacterLibrary: []poolsave.Character{character}, Party: []poolsave.Character{character}}, treasureStage: treasureCharacter, treasureItems: []gamepack.TreasureItemRecord{heavy}}
	called := false
	application.saveState = func(poolsave.State) error { called = true; return nil }
	if err := application.giveTreasureItem(0); err != nil || called || len(application.treasureItems) != 1 || len(application.state.Party[0].Inventory) != 0 || application.eventText != "OverLoaded" {
		t.Fatalf("overload err=%v called=%v pending=%d state=%+v text=%q", err, called, len(application.treasureItems), application.state, application.eventText)
	}
	light := treasureRecord("Light", 1)
	application.treasureItems = []gamepack.TreasureItemRecord{light}
	application.saveState = func(poolsave.State) error { return errors.New("disk full") }
	if err := application.giveTreasureItem(0); err == nil || len(application.treasureItems) != 1 || len(application.state.Party[0].Inventory) != 0 {
		t.Fatalf("save rollback err=%v pending=%d state=%+v", err, len(application.treasureItems), application.state)
	}
}

func TestKeysContinueThroughNameAndOriginalPortraitEditor(t *testing.T) {
	application := &app{
		mode:   modeCreation,
		flow:   creation.Flow{Stage: creation.StageRoll},
		rolled: &creation.RolledCharacter{},
		loadPortrait: func(head, body uint8) (*ebiten.Image, error) {
			image := ebiten.NewImage(88, 88)
			image.Fill(color.RGBA{uint8(head), uint8(body), 0, 255})
			return image, nil
		},
		loadIcon: func(head, body, size uint8, action bool, colors [6][2]uint8) (*ebiten.Image, error) {
			image := ebiten.NewImage(24, 24)
			image.Fill(color.RGBA{head, body, size, 255})
			return image, nil
		},
	}
	if err := press(application, ebiten.KeyEnter); err != nil || application.flow.Stage != creation.StageName {
		t.Fatalf("roll accept: stage=%d err=%v", application.flow.Stage, err)
	}
	application.keys = &scriptedTextKeys{scriptedKeys: scriptedKeys{}, chars: []rune("hero")}
	if err := application.Update(); err != nil {
		t.Fatal(err)
	}
	if application.nameInput != "HERO" {
		t.Fatalf("name input = %q", application.nameInput)
	}
	if err := press(application, ebiten.KeyEnter); err != nil || application.flow.Stage != creation.StagePortrait || application.portrait == nil {
		t.Fatalf("name accept: stage=%d portrait=%v err=%v", application.flow.Stage, application.portrait, err)
	}
	if err := press(application, ebiten.KeyH); err != nil || application.flow.PortraitHead != 2 {
		t.Fatalf("HEAD: %d err=%v", application.flow.PortraitHead, err)
	}
	if err := press(application, ebiten.KeyB); err != nil || application.flow.PortraitBody != 2 {
		t.Fatalf("BODY: %d err=%v", application.flow.PortraitBody, err)
	}
	if err := press(application, ebiten.KeyK); err != nil || application.flow.Stage != creation.StageIcon {
		t.Fatalf("KEEP: stage=%d err=%v", application.flow.Stage, err)
	}
	if application.iconReady == nil || application.iconAction == nil {
		t.Fatal("combat icon previews were not loaded")
	}
	// combat icon editor 是巢狀選單（spec 003 第 7..10 步）：
	// PARTS → HEAD → NEXT 一次，KEEP 收下，EXIT 回頂層。
	for _, item := range []struct {
		key  ebiten.Key
		what string
	}{{ebiten.KeyP, "PARTS"}, {ebiten.KeyH, "HEAD"}, {ebiten.KeyN, "NEXT"}} {
		if err := press(application, item.key); err != nil {
			t.Fatalf("icon %s: %v", item.what, err)
		}
	}
	if application.flow.IconHead != 1 {
		t.Fatalf("icon HEAD: %d", application.flow.IconHead)
	}
	// EXIT 退回進來時的值，KEEP 才算數——兩個出口的差別就在這裡。
	if err := press(application, ebiten.KeyE); err != nil || application.flow.IconHead != 0 {
		t.Fatalf("icon HEAD EXIT: %d err=%v", application.flow.IconHead, err)
	}
	for _, item := range []struct {
		key  ebiten.Key
		what string
	}{{ebiten.KeyH, "HEAD"}, {ebiten.KeyN, "NEXT"}, {ebiten.KeyK, "KEEP"}, {ebiten.KeyE, "PARTS EXIT"}} {
		if err := press(application, item.key); err != nil {
			t.Fatalf("icon %s: %v", item.what, err)
		}
	}
	if application.flow.IconHead != 1 {
		t.Fatalf("icon HEAD KEEP: %d", application.flow.IconHead)
	}
	// COLOR-1 的 BODY：畫面上第二列，記錄裡是第 0 格。
	before := application.flow.IconColors[0][0]
	for _, item := range []struct {
		key  ebiten.Key
		what string
	}{{ebiten.KeyDigit1, "COLOR-1"}, {ebiten.KeyB, "BODY"}, {ebiten.KeyN, "NEXT"}, {ebiten.KeyK, "KEEP"}} {
		if err := press(application, item.key); err != nil {
			t.Fatalf("icon %s: %v", item.what, err)
		}
	}
	if application.flow.IconColors[0][0] != (before+1)&0x0F {
		t.Fatalf("icon COLOR-1: %d", application.flow.IconColors[0][0])
	}
}

func TestIconConfirmationSavesLibraryThenAddBuildsParty(t *testing.T) {
	saves := 0
	application := &app{
		mode: modeCreation,
		flow: creation.Flow{Stage: creation.StageIconConfirm, Name: "HERO", PortraitHead: 1, PortraitBody: 1, IconSize: 1,
			IconColors: [6][2]uint8{{1, 9}, {2, 10}, {3, 11}, {4, 12}, {6, 14}, {7, 15}}},
		rolled:    &creation.RolledCharacter{Age: 53, Abilities: [6]int{16, 16, 11, 12, 12, 13}, Gold: 140, HP: 6, RawHP: 6},
		state:     poolsave.NewState(),
		saveState: func(state poolsave.State) error { saves++; return state.Validate() },
	}
	if err := press(application, ebiten.KeyY); err != nil {
		t.Fatal(err)
	}
	if application.mode != modeMenu || len(application.state.CharacterLibrary) != 1 || len(application.state.Party) != 0 {
		t.Fatalf("after finish mode=%d state=%+v", application.mode, application.state)
	}
	if got := application.state.CharacterLibrary[0]; got.Name != "HERO" || got.RaceID != "dwarf" || got.IconSize != 1 {
		t.Fatalf("saved character=%+v", got)
	}
	if err := press(application, ebiten.KeyA); err != nil {
		t.Fatal(err)
	}
	if len(application.state.Party) != 1 || application.state.Party[0].Name != "HERO" {
		t.Fatalf("party=%+v", application.state.Party)
	}
	if saves != 2 {
		t.Fatalf("save calls=%d, want 2", saves)
	}
}

func TestLoadSavedGameUsesVersionedStateSeam(t *testing.T) {
	want := poolsave.NewState()
	application := &app{mode: modeMenu, state: poolsave.NewState(), loadState: func() (poolsave.State, error) { return want, nil }}
	if err := press(application, ebiten.KeyL); err != nil {
		t.Fatal(err)
	}
	if application.state.Schema != poolsave.Schema {
		t.Fatalf("loaded schema=%q", application.state.Schema)
	}
}

func TestBeginAdventureRequiresPartyAndRunsSpec010FirstEvent(t *testing.T) {
	initial := gamepack.GeometryMap{Key: gamepack.MapKey{Archive: 3, BlockID: 0}}
	event := gamepack.InitialEvent{
		Position:      gamepack.Spawn{Map: gamepack.MapKey{Archive: 3, BlockID: 0}, X: 15, Y: 1, Facing: 3},
		MonsterID:     12,
		Message:       "ROLF GREETS THE PARTY IN PHLAN.",
		ContinueLabel: "PRESS <RETURN> OR BUTTON TO CONTINUE",
		Tour: []gamepack.TourStep{
			{Position: gamepack.Spawn{Map: gamepack.MapKey{Archive: 3, BlockID: 0}, X: 14, Y: 1, Facing: 3}},
			{Position: gamepack.Spawn{Map: gamepack.MapKey{Archive: 3, BlockID: 0}, X: 0, Y: 4, Facing: 3}, Selector: 6, Messages: []string{"OLD CITY GATE", "ON YOUR OWN NOW"}},
		},
	}
	walls := graphics.PieceSet{}
	application := &app{
		mode:         modeMenu,
		state:        poolsave.NewState(),
		spawn:        gamepack.DOSInitialSpawn(),
		initialMap:   &initial,
		initialWalls: &walls,
		initialEvent: &event,
	}
	// 隊伍是空的時候 B）EGIN 根本不在選單上（說明書 p.9），所以按下去
	// 沒有反應——這不是缺一句提示，是原版就沒有那一列。
	for _, entry := range application.visiblePartyMenuEntries() {
		if entry.key == ebiten.KeyB {
			t.Fatal("空隊伍不該列出 B）EGIN ADVENTURING")
		}
	}
	if err := press(application, ebiten.KeyB); err != nil {
		t.Fatal(err)
	}
	if application.mode != modeMenu {
		t.Fatalf("empty-party Begin mode=%d", application.mode)
	}
	application.state.Party = []poolsave.Character{{Name: "HERO"}}
	if err := press(application, ebiten.KeyB); err != nil {
		t.Fatal(err)
	}
	if application.mode != modeAdventure || application.spawn != event.Position || !application.introWaiting {
		t.Fatalf("Begin mode=%d spawn=%+v", application.mode, application.spawn)
	}
	if err := press(application, ebiten.KeyEnter); err != nil || application.introWaiting || !application.tourActive || application.introDone || application.mode != modeAdventure {
		t.Fatalf("Return gate mode=%d waiting=%v active=%v done=%v err=%v", application.mode, application.introWaiting, application.tourActive, application.introDone, err)
	}
	if err := application.Update(); err != nil || application.tourStep != 0 || application.spawn != event.Tour[0].Position {
		t.Fatalf("first tour frame step=%d spawn=%+v err=%v", application.tourStep, application.spawn, err)
	}
	for tick := 0; tick <= tourStepDelayTicks; tick++ {
		if err := application.Update(); err != nil {
			t.Fatal(err)
		}
	}
	if application.tourStep != 1 || application.tourPage != 0 || application.spawn != event.Tour[1].Position {
		t.Fatalf("final stop step=%d page=%d spawn=%+v", application.tourStep, application.tourPage, application.spawn)
	}
	if err := press(application, ebiten.KeyEnter); err != nil || application.tourPage != 1 {
		t.Fatalf("second final page=%d err=%v", application.tourPage, err)
	}
	if err := press(application, ebiten.KeyEnter); err != nil || application.tourPage != -1 {
		t.Fatalf("final Return page=%d err=%v", application.tourPage, err)
	}
	if err := application.Update(); err != nil || application.tourActive || !application.introDone {
		t.Fatalf("tour EXIT active=%v done=%v err=%v", application.tourActive, application.introDone, err)
	}
	if err := press(application, ebiten.KeyEscape); err != nil || application.mode != modeMenu {
		t.Fatalf("adventure ESC mode=%d err=%v", application.mode, err)
	}
}

func TestRealInitialAdventureUsesSharedVMToRolfExit(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	event, err := gamepack.ReadDOSInitialEvent(zipPath)
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	catalog, err := gamepack.ReadDOSGeometryCatalog(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	initial, ok := catalog.Map(event.Position.Map)
	if !ok {
		t.Fatal("initial GEO map is absent")
	}
	walls := graphics.PieceSet{}
	hero := poolsave.Character{Name: "HERO", RaceID: "dwarf", GenderID: "male", ClassID: "fighter", AlignmentID: "lawful-good", Money: [7]uint16{3: 100}, MaxHP: 12, CurrentHP: 2, PortraitHead: 1, PortraitBody: 1, IconSize: 1}
	state := poolsave.NewState()
	state.CharacterLibrary, state.Party = []poolsave.Character{hero}, []poolsave.Character{hero}
	application := &app{mode: modeMenu, state: state, roller: fixedTempleRoller(6), initialMap: &initial, geometryCatalog: catalog, initialWalls: &walls, initialEvent: &event, spawn: gamepack.DOSInitialSpawn()}
	var saved poolsave.State
	application.saveState = func(state poolsave.State) error { saved = state; return nil }
	if err := press(application, ebiten.KeyB); err != nil {
		t.Fatal(err)
	}
	if application.eventMachine == nil || !application.introWaiting || !strings.Contains(application.eventText, "GREETINGS, COURAGEOUS ONES") {
		t.Fatalf("initial VM state waiting=%v text=%q", application.introWaiting, application.eventText)
	}
	for tick := 0; tick < 2000 && !application.introDone; tick++ {
		if application.introWaiting {
			err = press(application, ebiten.KeyEnter)
		} else {
			err = application.Update()
		}
		if err != nil {
			t.Fatalf("tick %d: %v", tick, err)
		}
	}
	if !application.introDone || application.tourActive || application.tourStep != 33 || application.spawn.X != 0 || application.spawn.Y != 4 || application.spawn.Facing != 3 {
		t.Fatalf("final done=%v active=%v spawn=%+v step=%d", application.introDone, application.tourActive, application.spawn, application.tourStep)
	}
	// 導覽在 (0,4) 結束時朝西（3）。左轉兩次到東（1）才能往 +x 走。
	if err := press(application, ebiten.KeyArrowLeft); err != nil || application.spawn.Facing != 2 {
		t.Fatalf("turn facing=%d err=%v", application.spawn.Facing, err)
	}
	if err := press(application, ebiten.KeyArrowLeft); err != nil || application.spawn.Facing != 1 {
		t.Fatalf("second turn facing=%d err=%v", application.spawn.Facing, err)
	}
	if err := press(application, ebiten.KeyArrowUp); err != nil || application.spawn.X != 1 || application.spawn.Y != 4 || application.cellEventPending {
		t.Fatalf("forward spawn=%+v err=%v", application.spawn, err)
	}
	if err := press(application, ebiten.KeyArrowLeft); err != nil || application.spawn.Facing != 0 {
		t.Fatalf("north turn facing=%d err=%v", application.spawn.Facing, err)
	}
	if err := press(application, ebiten.KeyArrowUp); err != nil {
		t.Fatal(err)
	}
	if application.spawn.X != 1 || application.spawn.Y != 3 || !application.cellEventPending || !strings.Contains(application.eventText, "PRIESTESS JOY OF SUNE") {
		t.Fatalf("Sune event spawn=%+v pending=%v text=%q", application.spawn, application.cellEventPending, application.eventText)
	}
	if err := press(application, ebiten.KeyEnter); err != nil || !strings.Contains(application.eventText, "DO YOU SEEK HEALING") {
		t.Fatalf("Sune question pending=%v text=%q err=%v", application.cellEventPending, application.eventText, err)
	}
	if err := press(application, ebiten.KeyEnter); err != nil || !application.cellWaitingMenu || !reflect.DeepEqual(application.cellMenuOptions, []string{"YES", "NO"}) || application.cellMenuCursor != 0 {
		t.Fatalf("Sune menu waiting=%v options=%v cursor=%d label=%q err=%v", application.cellWaitingMenu, application.cellMenuOptions, application.cellMenuCursor, application.eventLabel, err)
	}
	if err := press(application, ebiten.KeyArrowRight); err != nil || application.cellMenuCursor != 1 || !strings.Contains(application.eventLabel, "> NO") {
		t.Fatalf("Sune NO selection cursor=%d label=%q err=%v", application.cellMenuCursor, application.eventLabel, err)
	}
	if err := press(application, ebiten.KeyArrowLeft); err != nil || application.cellMenuCursor != 0 {
		t.Fatalf("Sune YES reselection cursor=%d err=%v", application.cellMenuCursor, err)
	}
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if !application.templeActive || !application.cellEventPending || !application.cellWaitingMenu || !reflect.DeepEqual(application.cellMenuOptions, []string{"Heal", "View", "Pool", "Appraise", "Exit"}) || !strings.Contains(application.eventText, "HERO, how can we help you?") {
		t.Fatalf("temple active=%v pending=%v waiting=%v options=%v text=%q", application.templeActive, application.cellEventPending, application.cellWaitingMenu, application.cellMenuOptions, application.eventText)
	}
	if application.eventMachine.Memory[0x6DE2] != 1 {
		t.Fatalf("temple flag=%d, want 1", application.eventMachine.Memory[0x6DE2])
	}
	if err := press(application, ebiten.KeyEnter); err != nil || application.templeStage != templeHeal || !reflect.DeepEqual(application.cellMenuOptions, templeHealOptions) {
		t.Fatalf("temple Heal stage=%d options=%v err=%v", application.templeStage, application.cellMenuOptions, err)
	}
	if err := press(application, ebiten.KeyArrowRight); err != nil {
		t.Fatal(err)
	}
	if err := press(application, ebiten.KeyArrowRight); err != nil || application.cellMenuCursor != 2 {
		t.Fatalf("Cure Light cursor=%d err=%v", application.cellMenuCursor, err)
	}
	if err := press(application, ebiten.KeyEnter); err != nil || application.templeStage != templeConfirm || !strings.Contains(application.eventText, "100 gold pieces") {
		t.Fatalf("Cure Light confirm stage=%d text=%q err=%v", application.templeStage, application.eventText, err)
	}
	if err := press(application, ebiten.KeyEnter); err != nil || application.templeStage != templeHeal {
		t.Fatalf("Cure Light purchase stage=%d err=%v", application.templeStage, err)
	}
	if application.state.Party[0].Money[3] != 0 || application.state.Party[0].CurrentHP != 8 || saved.Party[0].CurrentHP != 8 || !strings.Contains(application.eventText, "HERO is cured") {
		t.Fatalf("Cure Light state=%+v saved=%+v text=%q", application.state, saved, application.eventText)
	}
	if err := press(application, ebiten.KeyArrowLeft); err != nil || application.cellMenuCursor != 9 || !strings.Contains(application.eventLabel, "> Exit") {
		t.Fatalf("Heal Exit cursor=%d label=%q err=%v", application.cellMenuCursor, application.eventLabel, err)
	}
	if err := press(application, ebiten.KeyEnter); err != nil || application.templeStage != templeMain || application.cellMenuCursor != 0 {
		t.Fatalf("return temple main stage=%d cursor=%d err=%v", application.templeStage, application.cellMenuCursor, err)
	}
	if err := press(application, ebiten.KeyArrowLeft); err != nil || application.cellMenuCursor != 4 || !strings.Contains(application.eventLabel, "> Exit") {
		t.Fatalf("temple Exit selection cursor=%d label=%q err=%v", application.cellMenuCursor, application.eventLabel, err)
	}
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if application.templeActive || application.cellEventPending || application.cellWaitingMenu || application.eventMachine.Memory[0x6DE1] != 0xFF || application.spawn.X != 1 || application.spawn.Y != 3 || application.spawn.Facing != 0 {
		t.Fatalf("temple exit active=%v pending=%v waiting=%v flag=%04X spawn=%+v", application.templeActive, application.cellEventPending, application.cellWaitingMenu, application.eventMachine.Memory[0x6DE1], application.spawn)
	}
	// 北（0）右轉兩次是南（2）。
	for turn := 0; turn < 2; turn++ {
		if err := press(application, ebiten.KeyArrowRight); err != nil {
			t.Fatal(err)
		}
	}
	if application.spawn.Facing != 2 {
		t.Fatalf("City Hall south facing=%d", application.spawn.Facing)
	}
	if err := press(application, ebiten.KeyArrowUp); err != nil || application.spawn.X != 1 || application.spawn.Y != 4 {
		t.Fatalf("City Hall south step spawn=%+v err=%v", application.spawn, err)
	}
	// 南（2）左轉一次是東（1）。
	if err := press(application, ebiten.KeyArrowLeft); err != nil || application.spawn.Facing != 1 {
		t.Fatalf("City Hall east facing=%d err=%v", application.spawn.Facing, err)
	}
	for wantX := uint8(2); wantX <= 3; wantX++ {
		if err := press(application, ebiten.KeyArrowUp); err != nil {
			t.Fatalf("move toward City Hall x=%d: %v", wantX, err)
		}
		if application.spawn.X != wantX || application.spawn.Y != 4 {
			t.Fatalf("City Hall route spawn=%+v want=(%d,4)", application.spawn, wantX)
		}
	}
	if !application.cellEventPending || !strings.Contains(application.eventText, "OUTSIDE THE CITY HALL") {
		t.Fatalf("City Hall pending=%v text=%q", application.cellEventPending, application.eventText)
	}
	cityHallText := application.eventText
	if err := press(application, ebiten.KeyEnter); err != nil || !application.cellWaitingMenu || application.eventText != cityHallText {
		t.Fatalf("City Hall continue menu waiting=%v text=%q err=%v", application.cellWaitingMenu, application.eventText, err)
	}
	if err := press(application, ebiten.KeyEnter); err != nil || application.cellWaitingMenu || application.eventText != "PROCLAMATIONS ARE POSTED ON THE WALLS, IN YOUR JOURNAL YOU NOTE" {
		t.Fatalf("City Hall proclamation intro waiting=%v text=%q err=%v", application.cellWaitingMenu, application.eventText, err)
	}
	// 布告清單是 `AC9B PRINT`（`11h`），接在 `AC22 PRINTCLEAR` 那一句後面——
	// 兩者之間沒有任何等待玩家的指令，所以原版是一句話，不是兩頁。
	// remake 先前把 `11h` 也當成取代，第二段就把第一段蓋掉了（spec 082）。
	if err := press(application, ebiten.KeyEnter); err != nil ||
		application.eventText != "PROCLAMATIONS ARE POSTED ON THE WALLS, IN YOUR JOURNAL YOU NOTE PROCLAMATIONS LXIV, LXXVIII, CIX, AND LIX." {
		t.Fatalf("City Hall proclamation list text=%q err=%v", application.eventText, err)
	}
	if err := press(application, ebiten.KeyEnter); err != nil || application.cellEventPending || application.cellWaitingMenu {
		t.Fatalf("City Hall return to movement pending=%v waiting=%v text=%q err=%v", application.cellEventPending, application.cellWaitingMenu, application.eventText, err)
	}
	if err := press(application, ebiten.KeyArrowUp); err != nil {
		t.Fatalf("attempt City Hall doorway: %v", err)
	}
	if application.spawn.Map != (gamepack.MapKey{Archive: 3, BlockID: 0}) || application.spawn.X != 4 || application.spawn.Y != 4 || application.spawn.Facing != 1 || application.eventSession.CurrentBlockID() != 8 || application.cellEventPending || application.cellWaitingMenu {
		t.Fatalf("City Hall doorway spawn=%+v pending=%v waiting=%v text=%q script_block=%d", application.spawn, application.cellEventPending, application.cellWaitingMenu, application.eventText, application.eventSession.CurrentBlockID())
	}
	// 東（1）右轉一次是南（2）。
	if err := press(application, ebiten.KeyArrowRight); err != nil || application.spawn.Facing != 2 {
		t.Fatalf("turn toward clerk corridor facing=%d err=%v", application.spawn.Facing, err)
	}
	if err := press(application, ebiten.KeyArrowUp); err != nil || application.spawn.X != 4 || application.spawn.Y != 5 || !application.cellEventPending || !strings.Contains(application.eventText, "OUTSIDE THE CLERK'S OFFICE") {
		t.Fatalf("clerk outside spawn=%+v pending=%v text=%q err=%v", application.spawn, application.cellEventPending, application.eventText, err)
	}
	if err := press(application, ebiten.KeyEnter); err != nil || application.cellEventPending {
		t.Fatalf("leave clerk outside boundary pending=%v text=%q err=%v", application.cellEventPending, application.eventText, err)
	}
	// 南（2）左轉一次是東（1）。
	if err := press(application, ebiten.KeyArrowLeft); err != nil || application.spawn.Facing != 1 {
		t.Fatalf("turn into clerk office facing=%d err=%v", application.spawn.Facing, err)
	}
	if err := press(application, ebiten.KeyArrowUp); err != nil || application.spawn.X != 5 || application.spawn.Y != 5 || !application.cellEventPending || !strings.Contains(application.eventText, "COUNCIL CLERK BEGINS LOOKING") {
		t.Fatalf("clerk entry spawn=%+v pending=%v text=%q err=%v", application.spawn, application.cellEventPending, application.eventText, err)
	}
	if application.eventMachine.Memory[0x4A01] != 1 || application.eventMachine.Memory[0x4A06] != 1 {
		t.Fatalf("clerk entry flags 4A01=%d 4A06=%d", application.eventMachine.Memory[0x4A01], application.eventMachine.Memory[0x4A06])
	}
	clerkEntryText := application.eventText
	if err := press(application, ebiten.KeyEnter); err != nil || !application.cellWaitingMenu || application.eventText != clerkEntryText {
		t.Fatalf("clerk entry continue menu waiting=%v text=%q err=%v", application.cellWaitingMenu, application.eventText, err)
	}
	if err := press(application, ebiten.KeyEnter); err != nil || application.cellWaitingMenu || application.eventText != "THE CLERK SHUFFLES THROUGH HER PAPERS. 'ON THE MATTER OF COMMISSION,' SHE SAYS, 'I CAN OFFER THE FOLLOWING: '" {
		t.Fatalf("clerk commission boundary waiting=%v text=%q err=%v", application.cellWaitingMenu, application.eventText, err)
	}
	previous := application.eventText
	for index, want := range []string{
		"THE SLUMS IMMEDIATELY TO OUR WEST NEED TO BE CLEARED OF MONSTERS.'",
		"SOKAL KEEP ON THORN ISLAND MUST BE CLEARED.'",
		"THE COUNCIL IS OFFERING A REWARD FOR BOOKS, MAPS, TOMES, ETC. WHICH PROVIDE USEFUL INFORMATION ABOUT PHLAN BEFORE THE FALL.  THE REWARD IS TIED TO THE VALUE OF THE INFORMATION.'",
		"'THESE ARE ALL OF THE COMMISSIONS CURRENTLY AVAILABLE.'",
	} {
		if err := press(application, ebiten.KeyEnter); err != nil || !application.cellWaitingMenu || application.eventText != previous {
			t.Fatalf("clerk commission menu %d waiting=%v text=%q want previous=%q err=%v", index, application.cellWaitingMenu, application.eventText, previous, err)
		}
		if err := press(application, ebiten.KeyEnter); err != nil || application.cellWaitingMenu || application.eventText != want {
			t.Fatalf("clerk commission page %d text=%q want=%q err=%v", index, application.eventText, want, err)
		}
		previous = want
	}
	if err := press(application, ebiten.KeyEnter); err != nil || application.cellEventPending || application.cellWaitingMenu {
		t.Fatalf("clerk EXIT pending=%v waiting=%v text=%q err=%v", application.cellEventPending, application.cellWaitingMenu, application.eventText, err)
	}
	if application.spawn.X != 5 || application.spawn.Y != 5 || application.eventSession.CurrentBlockID() != 8 {
		t.Fatalf("clerk EXIT spawn=%+v script_block=%d", application.spawn, application.eventSession.CurrentBlockID())
	}
}

func TestWrapASCIIUsesStableLineWidth(t *testing.T) {
	lines := wrapASCII("ONE TWO THREE FOUR FIVE", 9)
	want := []string{"ONE TWO", "THREE", "FOUR FIVE"}
	if !reflect.DeepEqual(lines, want) {
		t.Fatalf("lines=%q want=%q", lines, want)
	}
}

func TestInitialDOSFirstPersonViewResolvesOriginalWallStamps(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	catalog, err := gamepack.ReadDOSGeometryCatalog(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	spawn := gamepack.DOSInitialSpawn()
	initial, ok := catalog.Map(spawn.Map)
	if !ok {
		t.Fatal("initial GEO map is absent")
	}
	piece, err := gamepack.ReadDOSPieceSet(zipPath, 3, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	band0, _, err := gamepack.ReadDOSGlobalSymbolBands(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	stamps, err := initialWallStamps(initial.Grid, piece, spawn, band0)
	if err != nil {
		t.Fatal(err)
	}
	if len(stamps) == 0 {
		t.Fatal("initial first-person view resolved no wall stamps")
	}
	for _, stamp := range stamps {
		if stamp.Row < 0 || stamp.Row > 10 || stamp.Column < 0 || stamp.Column > 10 {
			t.Fatalf("unclipped stamp=%+v", stamp)
		}
	}
	t.Logf("initial DOS first-person view resolved %d visible 8x8 wall stamps", len(stamps))
}

func TestPoolFirstPersonUsesSharedStageInsetFill(t *testing.T) {
	fill, err := poolFirstPersonStageFill()
	if err != nil {
		t.Fatal(err)
	}
	// 兩個色索引是量原版量出來的（spec 047 的來源節、
	// `docs/reference/original-dos/adventure/02-first-person-14-1-west.png`）：
	// 天空 EGA 11 青、地面 EGA 6 棕。
	wantPostWall := []viewport.BackgroundRect{{X: 24, Y: 24, Width: 88, Height: 16, PaletteIndex: poolSkyPaletteIndex}}
	if len(fill.Backdrop) != 3 || !reflect.DeepEqual(fill.PostWall, wantPostWall) {
		t.Fatalf("Pool stage fill=%+v, want three bands and post-wall=%+v", fill, wantPostWall)
	}
	if fill.Backdrop[0].PaletteIndex != poolSkyPaletteIndex ||
		fill.Backdrop[2].PaletteIndex != poolGroundPaletteIndex ||
		fill.Backdrop[2].Y+fill.Backdrop[2].Height != 112 {
		t.Fatalf("Pool stage palettes/bounds changed: %+v", fill.Backdrop)
	}
}

func TestPoolPostWallLayerMapsOverTheRenderedTopCorners(t *testing.T) {
	fill, err := poolFirstPersonStageFill()
	if err != nil {
		t.Fatal(err)
	}
	if len(fill.PostWall) != 1 {
		t.Fatalf("Pool post-wall layers=%v", fill.PostWall)
	}
	got := poolStageScreenRect(fill.PostWall[0], 48, 86)
	want := image.Rect(48, 86, 224, 118)
	if got != want {
		t.Fatalf("Pool top fill screen rect=%v, want %v", got, want)
	}
}

// 全域熱鍵（J 手冊、I 裝備、K 法術）只在冒險畫面生效，而且**不能在別的畫面
// 把按鍵吃掉**：`justPressed` 讀一次就消費，條件寫成
// `justPressed(K) && mode == adventure` 會讓建角的 portrait editor 收不到 KEEP。
func TestGlobalHotkeysDoNotConsumeKeysOutsideAdventure(t *testing.T) {
	for _, key := range []ebiten.Key{ebiten.KeyJ, ebiten.KeyI, ebiten.KeyK} {
		keys := scriptedKeys{key: true}
		application := &app{mode: modeCreation, keys: keys}
		if err := application.Update(); err != nil {
			t.Fatalf("key %v: %v", key, err)
		}
		if application.journalOpen || application.equipmentOpen || application.spellsOpen {
			t.Fatalf("key %v opened an overlay outside the adventure screen", key)
		}
		if !keys[key] {
			t.Fatalf("key %v was consumed on the creation screen", key)
		}
	}
}

// 同一頁裡的多則訊息要拼起來，`33h PRINT RETURN` 是頁內換行、
// `3Dh CLEAR BOX` 清掉已經拼好的部分（spec 082）。
// 先前每則訊息蓋掉上一則，一頁印三行只看得到最後一行。
func TestCellTextBuildsOnePageFromEveryMessage(t *testing.T) {
	application := &app{}
	// 每個 result 一則：RunUntilEvent 一遇到事件就返回，所以狀態要跨 result。
	apply := func(events ...eclvm.Event) {
		application.applyCellECLResult(eclvm.Result{Events: events})
	}
	apply(eclvm.Event{Opcode: 0x12, Text: "FIRST LINE"})
	apply(eclvm.Event{Opcode: gamepack.PrintReturnOpcode})
	apply(eclvm.Event{Opcode: 0x12, Text: "SECOND LINE"})
	if application.eventText != "FIRST LINE\nSECOND LINE" {
		t.Fatalf("page is %q", application.eventText)
	}
	// 沒有換行收尾的下一則是新的一頁，不是接在後面。
	apply(eclvm.Event{Opcode: 0x12, Text: "NEW PAGE"})
	if application.eventText != "NEW PAGE" {
		t.Fatalf("page is %q", application.eventText)
	}
	// CLEAR BOX 清掉整個框。
	apply(eclvm.Event{Opcode: gamepack.ClearBoxOpcode})
	if application.eventText != "" {
		t.Fatalf("CLEAR BOX left %q", application.eventText)
	}
	apply(eclvm.Event{Opcode: 0x12, Text: "AFTER CLEAR"})
	// 沒有文字的事件不動文字框。
	apply(eclvm.Event{Opcode: 0x0E})
	if application.eventText != "AFTER CLEAR" {
		t.Fatalf("an empty event changed the page to %q", application.eventText)
	}
}

// `2Eh DAMAGE` 的兩條界線：赤字大於 9 直接死透，剛好歸零是不省人事
// （spec 084）。這兩條錯了，一場戰鬥的死傷名單就跟原版不同。
func TestDamageStateBoundaries(t *testing.T) {
	for _, item := range []struct {
		hp, damage int
		state      uint8
		wantHP     int
		wantState  uint8
		note       string
	}{
		{hp: 20, damage: 5, state: 0, wantHP: 15, wantState: 0, note: "扣血還活著"},
		{hp: 5, damage: 5, state: 0, wantHP: 0, wantState: 4, note: "剛好歸零"},
		{hp: 5, damage: 5, state: 1, wantHP: 0, wantState: 6, note: "狀態 1 歸零就死透"},
		{hp: 5, damage: 14, state: 0, wantHP: 0, wantState: 5, note: "赤字 9 是瀕死"},
		{hp: 5, damage: 15, state: 0, wantHP: 0, wantState: 6, note: "赤字 10 死透"},
	} {
		got := gamepack.ApplyDamage(item.hp, item.state, item.damage)
		if got.HitPoints != item.wantHP || got.State != item.wantState {
			t.Fatalf("%s: %d 點血吃 %d 傷害得到 %+v，預期 %d/%d",
				item.note, item.hp, item.damage, got, item.wantHP, item.wantState)
		}
		if downed := item.wantState > gamepack.AliveStateMax; got.Downed != downed {
			t.Fatalf("%s: Downed=%v，預期 %v", item.note, got.Downed, downed)
		}
	}
}

// 旗標的四個位元各自管什麼（spec 084）。bit 7 沒設整條不做事，
// 少判這一個會讓不該扣血的地方扣血。
//
// 低五位是**豁免修正**、運算元 5 的低三位才是**類別**：呼叫端
// （overlay-03 `2BCEh..2BD6h`）先推 `運算元5 & 7` 再推 `旗標 & 1Fh`，
// 而豁免常式（overlay-24 entry 7、`0D61h`）把第一個參數加進 d20、
// 第二個拿去索引 `record[+6Dh + 類別]`。
func TestDamageRequestFlags(t *testing.T) {
	quiet := gamepack.NewDamageRequest([gamepack.DamageOperands]uint16{0x00, 1, 6, 0, 0})
	if quiet.Applies() {
		t.Fatal("bit 7 沒設卻要動手")
	}
	// 低五位 0Ah＝修正 +10；運算元 5 的 0Ch & 7 ＝ 類別 4（法術）。
	// 10 這個值先前被當成類別，於是治具走到野外後面的區域時會失敗即關閉。
	party := gamepack.NewDamageRequest([gamepack.DamageOperands]uint16{0x80 | 0x40 | 0x0a, 2, 8, 3, 0x0c})
	if !party.Applies() || !party.WholeParty() || !party.AllowsSave() {
		t.Fatalf("%+v", party)
	}
	if party.SaveModifier() != 10 || party.SaveCategory != 4 {
		t.Fatalf("modifier=%d category=%d", party.SaveModifier(), party.SaveCategory)
	}
	if party.DiceCount != 2 || party.DiceSides != 8 || party.Bonus != 3 {
		t.Fatalf("%+v", party)
	}
	noSave := gamepack.NewDamageRequest([gamepack.DamageOperands]uint16{0x80 | 0x20, 1, 4, 0, 0})
	if noSave.AllowsSave() || noSave.WholeParty() {
		t.Fatalf("%+v", noSave)
	}
}

// `2Ch PARLAY`：選到第 N 個語氣就把運算元 N 的結果碼寫進運算元 6 指的變數
// （spec 086）。挑錯一格的症狀是交涉走錯分支，而畫面上分不出來。
func TestParlayWritesTheChosenOutcome(t *testing.T) {
	application := &app{eventMachine: &eclvm.Machine{Memory: map[uint16]uint16{}}}
	application.parlay = &parlayState{
		outcomes:      [gamepack.ParlayOutcomeCount]uint16{11, 22, 33, 44, 55},
		resultAddress: 0x1234,
	}
	application.cellMenuCursor = 2 // NICE
	if err := application.resolveParlayChoice(); err != nil {
		t.Fatal(err)
	}
	if got := application.eventMachine.Memory[0x1234]; got != 33 {
		t.Fatalf("parlay wrote %d, want the third outcome 33", got)
	}
	if application.parlay != nil || application.cellWaitingMenu {
		t.Fatal("the parlay menu is still up after a choice")
	}
}

// 五個顯示字串與結果表同長且順序相同。少一個或錯位就會選到別格。
func TestParlayMenuHasFiveOptionsInOrder(t *testing.T) {
	if len(gamepack.ParlayChoices) != gamepack.ParlayOutcomeCount {
		t.Fatalf("%d 個選項，結果表 %d 格", len(gamepack.ParlayChoices), gamepack.ParlayOutcomeCount)
	}
	if len(parlayMenuMessages) != gamepack.ParlayOutcomeCount {
		t.Fatalf("顯示字串 %d 個", len(parlayMenuMessages))
	}
	application := &app{}
	for index, item := range parlayMenuMessages {
		if got := application.text(item); got != gamepack.ParlayChoices[index] {
			t.Fatalf("第 %d 個顯示 %q，原文是 %q", index, got, gamepack.ParlayChoices[index])
		}
	}
}

// 兩條輸入 opcode 把打好的內容寫進運算元 2 指的變數（spec 087）：
// 數字寫 Memory，字串寫 Strings，而空字串會被換成一個空白——那是原版的
// 行為，少了它「什麼都沒打」與「打了一個空白」在 ECL 比對時結果不同。
func TestECLInputWritesTheDestination(t *testing.T) {
	newInputApp := func(numeric bool, buffer string) *app {
		return &app{
			eventMachine: &eclvm.Machine{Memory: map[uint16]uint16{}, Strings: map[uint16]string{}},
			eclInput:     &eclInputState{numeric: numeric, address: 0x4321, buffer: buffer},
		}
	}
	number := newInputApp(true, "137")
	if err := number.commitECLInput(); err != nil {
		t.Fatal(err)
	}
	if got := number.eventMachine.Memory[0x4321]; got != 137 {
		t.Fatalf("number wrote %d", got)
	}
	text := newInputApp(false, "MANTOR")
	if err := text.commitECLInput(); err != nil {
		t.Fatal(err)
	}
	if got := text.eventMachine.Strings[0x4321]; got != "MANTOR" {
		t.Fatalf("string wrote %q", got)
	}
	empty := newInputApp(false, "")
	if err := empty.commitECLInput(); err != nil {
		t.Fatal(err)
	}
	if got := empty.eventMachine.Strings[0x4321]; got != gamepack.InputStringEmptyReplacement {
		t.Fatalf("empty input wrote %q, want a single space", got)
	}
	// 沒打數字就是 0，不是錯誤。
	blank := newInputApp(true, "")
	if err := blank.commitECLInput(); err != nil {
		t.Fatal(err)
	}
	if got := blank.eventMachine.Memory[0x4321]; got != 0 {
		t.Fatalf("blank number wrote %d", got)
	}
	if blank.eclInput != nil || blank.cellEventPending {
		t.Fatal("the input line is still up after ENTER")
	}
}

// 數字輸入只收數字，字串輸入收可見字元並轉大寫，兩者都在上限處停下來。
func TestECLInputFiltersWhatItAccepts(t *testing.T) {
	application := &app{eclInput: &eclInputState{numeric: true}}
	application.keys = scriptedChars("12a3")
	if _, err := application.eclInputUpdate(); err != nil {
		t.Fatal(err)
	}
	if application.eclInput.buffer != "123" {
		t.Fatalf("numeric buffer %q", application.eclInput.buffer)
	}
	application = &app{eclInput: &eclInputState{numeric: false}}
	application.keys = scriptedChars("man tor")
	if _, err := application.eclInputUpdate(); err != nil {
		t.Fatal(err)
	}
	if application.eclInput.buffer != "MAN TOR" {
		t.Fatalf("string buffer %q", application.eclInput.buffer)
	}
	application = &app{eclInput: &eclInputState{numeric: false,
		buffer: strings.Repeat("X", gamepack.InputStringMaxLength)}}
	application.keys = scriptedChars("Y")
	if _, err := application.eclInputUpdate(); err != nil {
		t.Fatal(err)
	}
	if len(application.eclInput.buffer) != gamepack.InputStringMaxLength {
		t.Fatalf("buffer grew past the limit to %d", len(application.eclInput.buffer))
	}
}

// `39h WHO` 挑到的人要留下來給後面的 opcode 用（spec 090）：`28h ROB` 的
// 範圍 0 打的就是「目前角色」，挑錯人的症狀是別人掉了東西。
func TestWhoSetsTheCurrentCharacter(t *testing.T) {
	application := &app{state: poolsave.State{Party: []poolsave.Character{
		{Name: "A"}, {Name: "B"}, {Name: "C"},
	}}}
	application.whoPending = true
	application.cellWaitingMenu = true
	application.cellMenuOptions = []string{"A", "B", "C"}
	application.cellMenuCursor = 1
	if err := application.resolveWhoChoice(); err != nil {
		t.Fatal(err)
	}
	if application.currentCharacter != 1 {
		t.Fatalf("current character is %d, want 1", application.currentCharacter)
	}
	if application.whoPending || application.cellWaitingMenu {
		t.Fatal("the WHO menu is still up after a choice")
	}
	// 沒有等待中的選擇時不能悄悄成功。
	if err := application.resolveWhoChoice(); err == nil {
		t.Fatal("resolving a WHO choice that was never asked for succeeded")
	}
}

// NPC 加進隊伍之後，戰鬥數值要讀它自己的記錄，不是套建角那一組欄位
// （spec 091）。硬套會得到一個「一級戰士」，與原版差很多。
func TestNPCCombatStatsComeFromItsOwnRecord(t *testing.T) {
	state := &tacticalState{
		BaseMovement: make([]uint8, 2),
		HitPoints:    make([]int, 2),
		THAC0:        make([]uint8, 2),
		ArmorClass:   make([]int, 2),
		Damage:       make([]combat.DamageDice, 2),
		AttackForms:  make([][gamepack.MonsterAttackSlots]combat.DamageDice, 2),
		AttackRates:  make([][gamepack.MonsterAttackSlots]uint8, 2),
	}
	record := make([]byte, poolsave.NPCRecordSize)
	record[0x11C] = 9  // 移動力
	record[0x11B] = 33 // 目前生命值
	record[0x110] = 60 - 14
	record[0x111] = 60 - 3
	// 傷害骰讀的是來源欄位 `+0A2h/+0A4h/+0A6h`（spec 051）：形態 1 是 2d6+1。
	record[0xA1] = 2
	record[0xA3] = 2
	record[0xA5] = 6
	record[0xA7] = 1
	member := poolsave.Character{Name: "NPC", NPC: true, Record: record}
	if err := applyNPCCombatStats(state, 1, member); err != nil {
		t.Fatal(err)
	}
	if state.BaseMovement[1] != 9 || state.HitPoints[1] != 33 {
		t.Fatalf("movement=%d hp=%d", state.BaseMovement[1], state.HitPoints[1])
	}
	if state.THAC0[1] != 60-14 || state.ArmorClass[1] != 60-3 {
		t.Fatalf("thac0=%d ac=%d", state.THAC0[1], state.ArmorClass[1])
	}
	if state.Damage[1] != (combat.DamageDice{Count: 2, Sides: 6, Bonus: 1}) {
		t.Fatalf("damage=%+v", state.Damage[1])
	}
	if state.AttackForms[1][0] != (combat.DamageDice{Count: 2, Sides: 6, Bonus: 1}) ||
		state.AttackRates[1][0] != 2 {
		t.Fatalf("形態 1 是 %+v 次數 %d", state.AttackForms[1][0], state.AttackRates[1][0])
	}
	// 記錄長度不對就要失敗，不能靜靜用零值打。
	if err := applyNPCCombatStats(state, 1, poolsave.Character{Name: "X", NPC: true}); err == nil {
		t.Fatal("an NPC without a record was accepted")
	}
}

// 只有編號 18h 的 NPC 站到對面（spec 091）。
func TestAddNPCSideOnlyFlipsForOneID(t *testing.T) {
	if gamepack.AddNPCSide(gamepack.AddNPCHostileID) != 1 {
		t.Fatal("編號 18h 應該站到對面")
	}
	for _, id := range []uint8{0, 0x17, 0x19, 0x68, 0x6B, 0xFF} {
		if gamepack.AddNPCSide(id) != 0 {
			t.Fatalf("編號 %#x 不該站到對面", id)
		}
	}
}

// 存檔的隊伍上限是八，但玩家自己建的只佔得了六格。
func TestPartyAllowsTwoNPCsBeyondTheSixPlayers(t *testing.T) {
	state := poolsave.NewState()
	newPlayer := func(name string) poolsave.Character {
		return poolsave.Character{Name: name, MaxHP: 8, CurrentHP: 8,
			PortraitHead: 1, PortraitBody: 1, IconSize: 1}
	}
	newNPC := func(name string) poolsave.Character {
		return poolsave.Character{Name: name, NPC: true, MaxHP: 8, CurrentHP: 8,
			Record: make([]byte, poolsave.NPCRecordSize)}
	}
	for index := 0; index < 7; index++ {
		state.CharacterLibrary = append(state.CharacterLibrary, newPlayer(string(rune('A'+index))))
	}
	for index := 0; index < 6; index++ {
		state.Party = append(state.Party, newPlayer(string(rune('A'+index))))
	}
	state.Party = append(state.Party, newNPC("NPC1"), newNPC("NPC2"))
	if err := state.Validate(); err != nil {
		t.Fatalf("六名玩家加兩個 NPC 應該過：%v", err)
	}
	state.Party = append(state.Party, newNPC("NPC3"))
	if err := state.Validate(); err == nil {
		t.Fatal("九個人不該過")
	}
	state.Party = state.Party[:6]
	state.Party = append(state.Party, newPlayer("G"))
	if err := state.Validate(); err == nil {
		t.Fatal("七名玩家角色不該過")
	}
}

// `eclArchive` 這個鏡像會比 session 早一步更新——`syncArchiveFromEventMachine`
// 在腳本已經把 `6E12h` 寫成新值、但 `NEWECL` 還沒執行的那個空檔就會設它。
// 換 catalog 的判斷若拿它當基準，那一刻會看到「選擇子等於現行值」而不換，
// session 卻還握著上一個 archive 的區塊，於是換過去就報
// `ECL session target block ... is unavailable`。
//
// 這一則把那個空檔造出來：接好 session（此時它握著 ECL3）之後，先把鏡像撥到
// 2，再讓 ECL3/0 的控制器跑 `SAVE 2,6E12h → NEWECL 20`。修好之前這裡會紅。
func TestArchiveSwapFollowsTheSessionNotTheMirror(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	archive, ok := application.eclCatalog.Archive(3)
	if !ok {
		t.Fatal("ECL3 archive is absent")
	}
	session, err := gamepack.NewDOSECLArchiveSession(archive, 0, 0x9955)
	if err != nil {
		t.Fatal(err)
	}
	application.eventSession, application.eventMachine = session, session.Machine()
	application.eclArchive = 3
	application.spawn = gamepack.Spawn{Map: gamepack.MapKey{Archive: 3, BlockID: 0}, X: 0, Y: 4, Facing: 3}
	if err := application.configureEventSession(session); err != nil {
		t.Fatal(err)
	}
	// 鏡像先跑到前面去。
	application.eclArchive = 2
	result, err := session.RunUntilEvent(4096, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := application.consumeInitialTransitionResources(result); err != nil {
		t.Fatalf("鏡像領先時換不過去：%v", err)
	}
	if session.CurrentBlockID() != 20 || application.eclSessionArchive != 2 {
		t.Fatalf("session block=%d archive=%d，預期 20／2",
			session.CurrentBlockID(), application.eclSessionArchive)
	}
}

// NPC 入隊時要把職業從記錄帶出來。
//
// **這是實跑抓到的**：探索器走到某一格讓 WARRIOR 入隊之後，
// `Pool character "WARRIOR" has unknown class ""` 讓整局停在那裡。NPC 沒有
// 經過建角流程，職業只存在記錄的 `+2Fh`（複合職業碼）與 `+96h` 起的八個
// 等級裡；不帶出來的話 `partyClassCodes`（ECL 的隊伍查詢）與
// `partyClassLevels`（THAC0 與豁免）都會報錯——症狀不是顯示錯，是玩不下去。
func TestAddNPCTakesItsClassFromTheRecord(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	// 原版的記錄照讀，不做代用品——要驗的正是「記錄裡的碼查得到職業」。
	// 八個封存檔全掃：只驗一筆的話，查不到的那一筆正好沒被挑到就看不出來。
	var sample poolsave.Character
	records := 0
	for archive := uint8(1); archive <= 8; archive++ {
		for id := 0; id < 64; id++ {
			record, err := application.loadMonster(archive, uint8(id))
			if err != nil {
				continue
			}
			records++
			raw := record.Raw[:]
			code := raw[gamepack.ClassCodeOffset]
			classID, ok := creation.ClassIDForDOSCode(code)
			if !ok {
				t.Errorf("mon%d/%d %q 的 +2Fh=%d 查不到職業", archive, id, record.Name, code)
				continue
			}
			if back, ok := creation.ClassDOSCode(classID); !ok || back != code {
				t.Errorf("mon%d/%d 的職業 %q 反查回來是 %d，原本是 %d",
					archive, id, classID, back, code)
			}
			if sample.Record != nil {
				continue
			}
			levels, err := gamepack.ClassLevels(raw)
			if err != nil {
				t.Fatalf("mon%d/%d 的 +96h 取不出等級：%v", archive, id, err)
			}
			sample = poolsave.Character{Name: record.Name, NPC: true, ClassID: classID,
				Record: append([]byte(nil), raw...),
				ClassLevels: append([]uint8(nil), levels[:]...)}
		}
	}
	if records == 0 {
		t.Skip("讀不到原版的 NPC 記錄")
	}
	t.Logf("掃過 %d 筆記錄，`+2Fh` 全部查得到職業", records)

	// 兩個先前會炸的消費端都要過。
	application.state.Party = []poolsave.Character{sample}
	code := sample.Record[gamepack.ClassCodeOffset]
	codes, err := application.partyClassCodes()
	if err != nil {
		t.Fatalf("隊伍職業碼：%v", err)
	}
	if len(codes) != 1 || codes[0] != code {
		t.Errorf("查出來的職業碼是 %v，記錄裡是 %d", codes, code)
	}
	if _, err := partyClassLevels(sample); err != nil {
		t.Fatalf("職業等級：%v", err)
	}
	// 等級全零的記錄不能帶進 ClassLevels：帶了 `partyClassLevels` 會回一組
	// 全零，而不是退回「沒訓練過就照第 1 級算」。
	blank := poolsave.Character{Name: "Z", NPC: true, ClassID: sample.ClassID,
		Record: make([]byte, len(sample.Record))}
	levelsForBlank, err := partyClassLevels(blank)
	if err != nil {
		t.Fatalf("沒有等級的 NPC：%v", err)
	}
	sum := 0
	for _, level := range levelsForBlank {
		sum += int(level)
	}
	if sum == 0 {
		t.Error("沒有等級的 NPC 應該退回第 1 級，不是一組全零")
	}
	// 負對照：職業空著又沒有等級就要報錯，不能靜靜當成 0。
	if _, err := partyClassLevels(poolsave.Character{Name: "X"}); err == nil {
		t.Error("沒有職業也沒有等級的角色被接受了")
	}
}
