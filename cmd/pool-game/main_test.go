package main

import (
	"errors"
	"image/color"
	"math/rand"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/creation"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
	"github.com/wicanr2/golden-box-remake-engine/graphics"
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

func TestSuneTempleCureUsesCurrentCharacterAndPersists(t *testing.T) {
	character := poolsave.Character{Name: "HERO", RaceID: "dwarf", GenderID: "male", ClassID: "fighter", AlignmentID: "lawful-good", Gold: 100, MaxHP: 12, CurrentHP: 2, PortraitHead: 1, PortraitBody: 1, IconSize: 1}
	application := &app{
		roller:       fixedTempleRoller(6),
		state:        poolsave.State{Schema: poolsave.Schema, PooledGold: 400, CharacterLibrary: []poolsave.Character{character}, Party: []poolsave.Character{character}},
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
	if application.state.Party[0].Gold != 0 || application.state.PooledGold != 400 || application.state.Party[0].CurrentHP != 8 || saved.Party[0].CurrentHP != 8 || !strings.Contains(application.eventText, "HERO is cured") {
		t.Fatalf("state=%+v saved=%+v text=%q", application.state, saved, application.eventText)
	}
}

func TestSuneTempleFailedSaveRollsBackCure(t *testing.T) {
	character := poolsave.Character{Name: "HERO", RaceID: "dwarf", GenderID: "male", ClassID: "fighter", AlignmentID: "lawful-good", Gold: 0, MaxHP: 12, CurrentHP: 2, PortraitHead: 1, PortraitBody: 1, IconSize: 1}
	application := &app{roller: fixedTempleRoller(6), state: poolsave.State{Schema: poolsave.Schema, PooledGold: 100, CharacterLibrary: []poolsave.Character{character}, Party: []poolsave.Character{character}}, templeActive: true, templeStage: templeConfirm, templeService: 0, cellMenuOptions: []string{"YES", "NO"}}
	application.saveState = func(poolsave.State) error { return errors.New("disk full") }
	if err := application.selectSuneTempleOption(); err == nil {
		t.Fatal("save failure was swallowed")
	}
	if application.state.PooledGold != 100 || application.state.Party[0].CurrentHP != 2 || application.state.CharacterLibrary[0].CurrentHP != 2 {
		t.Fatalf("failed save did not roll back: %+v", application.state)
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
	if err := press(application, ebiten.KeyH); err != nil || application.flow.IconHead != 1 {
		t.Fatalf("icon HEAD: %d err=%v", application.flow.IconHead, err)
	}
	if err := press(application, ebiten.KeyW); err != nil || application.flow.IconWeapon != 1 {
		t.Fatalf("icon WEAPON: %d err=%v", application.flow.IconWeapon, err)
	}
	if err := press(application, ebiten.KeyP); err != nil || application.flow.IconPart != 1 {
		t.Fatalf("icon PART: %d err=%v", application.flow.IconPart, err)
	}
	before := application.flow.IconColors[1][0]
	if err := press(application, ebiten.KeyDigit1); err != nil || application.flow.IconColors[1][0] != (before+1)&0x0F {
		t.Fatalf("icon COLOR-1: %d err=%v", application.flow.IconColors[1][0], err)
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
	if err := press(application, ebiten.KeyB); err != nil {
		t.Fatal(err)
	}
	if application.mode != modeMenu || application.statusLine == "" {
		t.Fatalf("empty-party Begin mode=%d status=%q", application.mode, application.statusLine)
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
	hero := poolsave.Character{Name: "HERO", RaceID: "dwarf", GenderID: "male", ClassID: "fighter", AlignmentID: "lawful-good", Gold: 100, MaxHP: 12, CurrentHP: 2, PortraitHead: 1, PortraitBody: 1, IconSize: 1}
	state := poolsave.NewState()
	state.CharacterLibrary, state.Party = []poolsave.Character{hero}, []poolsave.Character{hero}
	application := &app{mode: modeMenu, state: state, roller: fixedTempleRoller(6), initialMap: &initial, initialWalls: &walls, initialEvent: &event, spawn: gamepack.DOSInitialSpawn()}
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
	if err := press(application, ebiten.KeyArrowLeft); err != nil || application.spawn.Facing != 2 {
		t.Fatalf("turn facing=%d err=%v", application.spawn.Facing, err)
	}
	if err := press(application, ebiten.KeyArrowUp); err != nil || application.spawn.X != 1 || application.spawn.Y != 4 || application.cellEventPending {
		t.Fatalf("forward spawn=%+v err=%v", application.spawn, err)
	}
	if err := press(application, ebiten.KeyArrowLeft); err != nil || application.spawn.Facing != 1 {
		t.Fatalf("first north turn facing=%d err=%v", application.spawn.Facing, err)
	}
	if err := press(application, ebiten.KeyArrowLeft); err != nil || application.spawn.Facing != 0 {
		t.Fatalf("second north turn facing=%d err=%v", application.spawn.Facing, err)
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
	if application.state.Party[0].Gold != 0 || application.state.Party[0].CurrentHP != 8 || saved.Party[0].CurrentHP != 8 || !strings.Contains(application.eventText, "HERO is cured") {
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
	stamps, err := initialWallStamps(initial.Grid, piece, spawn)
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
