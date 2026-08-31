package main

import (
	"errors"
	"flag"
	"fmt"
	"image/color"
	"log"
	"math/rand"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font/basicfont"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/assets"
	poolcharacter "github.com/wicanr2/Pool-of-Radiance-cht/internal/character"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/creation"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/temple"
	"github.com/wicanr2/golden-box-remake-engine/eclvm"
	"github.com/wicanr2/golden-box-remake-engine/geometry"
	"github.com/wicanr2/golden-box-remake-engine/graphics"
	"github.com/wicanr2/golden-box-remake-engine/viewport"
)

const (
	logicalWidth  = 640
	logicalHeight = 400
	// Spec 011 permits a deterministic approximation for the original DELAY.
	// Nine 60 Hz updates make each non-dialogue tour frame visible (~150 ms).
	tourStepDelayTicks = 9
)

type screenMode uint8

type templeStage uint8
type treasureStage uint8

const (
	modeTitle screenMode = iota
	modeMenu
	modeCreation
	modeAdventure
)

const (
	treasureMain treasureStage = iota
	treasureView
	treasureItems
	treasureCharacter
	treasureConfirmExit
)

const (
	templeMain templeStage = iota
	templeHeal
	templeConfirm
)

type diceRoller struct{ random *rand.Rand }

func (roller diceRoller) Roll(count, sides int) int {
	total := 0
	for index := 0; index < count; index++ {
		total += roller.random.Intn(sides) + 1
	}
	return total
}

type keySource interface{ JustPressed(ebiten.Key) bool }
type ebitenKeys struct{}

func (ebitenKeys) JustPressed(key ebiten.Key) bool { return inpututil.IsKeyJustPressed(key) }
func (ebitenKeys) Chars() []rune                   { return ebiten.AppendInputChars(nil) }

type app struct {
	mode             screenMode
	title            *ebiten.Image
	flow             creation.Flow
	cursor           int
	rolled           *creation.RolledCharacter
	roller           creation.Roller
	help             bool
	modern           bool
	statusLine       string
	keys             keySource
	nameInput        string
	portrait         *ebiten.Image
	iconReady        *ebiten.Image
	iconAction       *ebiten.Image
	loadPortrait     func(head, body uint8) (*ebiten.Image, error)
	loadIcon         func(head, body, size uint8, action bool, colors [6][2]uint8) (*ebiten.Image, error)
	state            poolsave.State
	saveState        func(poolsave.State) error
	loadState        func() (poolsave.State, error)
	initialMap       *gamepack.GeometryMap
	geometryCatalog  gamepack.GeometryCatalog
	initialWalls     *graphics.PieceSet
	initialEvent     *gamepack.InitialEvent
	spawn            gamepack.Spawn
	introWaiting     bool
	introDone        bool
	tourActive       bool
	tourStep         int
	tourPage         int
	tourDelay        int
	eventMachine     *eclvm.Machine
	eventSession     *eclvm.BlockSession
	eventText        string
	eventLabel       string
	cellEventPending bool
	cellWaitingMenu  bool
	cellMenuOptions  []string
	cellMenuCursor   int
	templeActive     bool
	templeStage      templeStage
	templeParty      int
	templeService    int
	loadTreasure     func(archive, block uint8) ([]gamepack.TreasureItemRecord, error)
	treasureActive   bool
	treasureStage    treasureStage
	treasureItems    []gamepack.TreasureItemRecord
	treasureSelected int
}

func newApp(zipPath string) (*app, error) {
	pictures, err := assets.ReadTitlePictures(zipPath)
	if err != nil {
		return nil, err
	}
	rendered, err := pictures[1].RGBA(0, graphics.EGA16)
	if err != nil {
		return nil, err
	}
	application := &app{
		mode:   modeTitle,
		title:  ebiten.NewImageFromImage(rendered),
		flow:   creation.NewFlow(),
		roller: diceRoller{random: rand.New(rand.NewSource(time.Now().UnixNano()))},
		keys:   ebitenKeys{},
		state:  poolsave.NewState(),
	}
	catalog, err := gamepack.ReadDOSGeometryCatalog(zipPath)
	if err != nil {
		return nil, err
	}
	application.spawn = gamepack.DOSInitialSpawn()
	initialMap, ok := catalog.Map(application.spawn.Map)
	if !ok {
		return nil, fmt.Errorf("DOS initial map GEO%d block %d is absent", application.spawn.Map.Archive, application.spawn.Map.BlockID)
	}
	application.initialMap = &initialMap
	application.geometryCatalog = catalog
	initialWalls, err := gamepack.ReadDOSPieceSet(zipPath, 3, 1, 0)
	if err != nil {
		return nil, fmt.Errorf("load DOS initial wall set: %w", err)
	}
	application.initialWalls = &initialWalls
	initialEvent, err := gamepack.ReadDOSInitialEvent(zipPath)
	if err != nil {
		return nil, fmt.Errorf("load DOS initial event: %w", err)
	}
	application.initialEvent = &initialEvent
	const statePath = "saves/pool-remake-state.json"
	application.saveState = func(state poolsave.State) error { return poolsave.WriteAtomic(statePath, state) }
	application.loadState = func() (poolsave.State, error) { return poolsave.Read(statePath) }
	application.loadTreasure = func(archive, block uint8) ([]gamepack.TreasureItemRecord, error) {
		return gamepack.ReadDOSTreasureItemBlock(zipPath, archive, block)
	}
	application.loadPortrait = func(head, body uint8) (*ebiten.Image, error) {
		parts, err := assets.ReadCreationPortraitParts(zipPath, head, body)
		if err != nil {
			return nil, err
		}
		composed, err := assets.ComposeCreationPortrait(parts)
		if err != nil {
			return nil, err
		}
		rendered, err := composed.RGBA(0, graphics.EGA16)
		if err != nil {
			return nil, err
		}
		return ebiten.NewImageFromImage(rendered), nil
	}
	application.loadIcon = func(head, body, size uint8, action bool, colors [6][2]uint8) (*ebiten.Image, error) {
		picture, err := assets.ReadCustomizedCombatIcon(zipPath, assets.CombatIconSelection{Head: head, Body: body, Size: size}, action, colors)
		if err != nil {
			return nil, err
		}
		rendered, err := picture.RGBA(0, graphics.EGA16)
		if err != nil {
			return nil, err
		}
		return ebiten.NewImageFromImage(rendered), nil
	}
	return application, nil
}

func (a *app) justPressed(key ebiten.Key) bool {
	return a.keys != nil && a.keys.JustPressed(key)
}

func (a *app) inputChars() []rune {
	if source, ok := a.keys.(interface{ Chars() []rune }); ok {
		return source.Chars()
	}
	return nil
}

func (a *app) reloadPortrait() error {
	if a.loadPortrait == nil {
		return fmt.Errorf("portrait loader is not configured")
	}
	portrait, err := a.loadPortrait(a.flow.PortraitHead, a.flow.PortraitBody)
	if err != nil {
		return err
	}
	a.portrait = portrait
	return nil
}

func (a *app) reloadIcons() error {
	if a.loadIcon == nil {
		return fmt.Errorf("combat icon loader is not configured")
	}
	ready, err := a.loadIcon(a.flow.IconHead, a.flow.IconWeapon, a.flow.IconSize, false, a.flow.IconColors)
	if err != nil {
		return err
	}
	action, err := a.loadIcon(a.flow.IconHead, a.flow.IconWeapon, a.flow.IconSize, true, a.flow.IconColors)
	if err != nil {
		return err
	}
	a.iconReady, a.iconAction = ready, action
	return nil
}

func (a *app) Update() error {
	if a.justPressed(ebiten.KeyF10) {
		if a.saveState != nil {
			state, err := a.stateForSave()
			if err != nil {
				return err
			}
			if err := a.saveState(state); err != nil {
				return err
			}
			a.state = state
		}
		return ebiten.Termination
	}
	if a.justPressed(ebiten.KeyF1) {
		a.help = !a.help
	}
	if a.justPressed(ebiten.KeyF2) {
		a.modern = !a.modern
	}
	if a.help {
		if a.justPressed(ebiten.KeyEscape) {
			a.help = false
		}
		return nil
	}
	switch a.mode {
	case modeTitle:
		if a.justPressed(ebiten.KeyEnter) || a.justPressed(ebiten.KeySpace) {
			a.mode = modeMenu
		}
	case modeMenu:
		if a.justPressed(ebiten.KeyB) {
			if len(a.state.Party) == 0 {
				a.statusLine = "Add at least one character before beginning adventure."
				return nil
			}
			if a.initialMap == nil || a.initialWalls == nil || a.initialEvent == nil {
				return fmt.Errorf("Pool initial adventure data is not configured")
			}
			a.spawn = a.initialEvent.Position
			a.introWaiting, a.introDone = true, false
			a.tourActive, a.tourStep, a.tourPage, a.tourDelay = false, -1, -1, 0
			a.eventMachine, a.eventSession, a.eventText, a.eventLabel = nil, nil, "", ""
			a.templeActive = false
			if len(a.initialEvent.ScriptBlock) != 0 {
				characters := make([]gamepack.InitialCharacter, len(a.state.Party))
				for index, character := range a.state.Party {
					characters[index] = gamepack.InitialCharacter{
						Name: character.Name, ClassID: character.ClassID, Abilities: character.Abilities,
						ExceptionalStrength: character.ExceptionalStrength, CurrentHP: character.CurrentHP,
					}
				}
				session, err := gamepack.NewInitialEventSession(*a.initialEvent, characters...)
				if err != nil {
					return err
				}
				machine := session.Machine()
				a.eventSession = session
				a.eventMachine = machine
				result, runErr := machine.Run(2000, nil, true)
				if runErr != nil {
					return fmt.Errorf("start Pool initial ECL: %w", runErr)
				}
				a.applyECLResult(result)
				if !a.introWaiting {
					return fmt.Errorf("Pool initial ECL did not reach its Return menu")
				}
			}
			a.mode = modeAdventure
			a.statusLine = "Original first Rolf event loaded; movement remains disabled."
			return nil
		}
		if a.justPressed(ebiten.KeyC) || a.justPressed(ebiten.KeyEnter) {
			a.flow, a.cursor, a.rolled = creation.NewFlow(), 0, nil
			a.mode = modeCreation
			return nil
		}
		if a.justPressed(ebiten.KeyA) {
			return a.addFirstLibraryCharacter()
		}
		if a.justPressed(ebiten.KeyL) {
			if a.loadState == nil {
				a.statusLine = "No save loader is configured."
				return nil
			}
			loaded, err := a.loadState()
			if err != nil {
				a.statusLine = err.Error()
				return nil
			}
			if loaded.Campaign != nil {
				if err := a.restoreCampaign(loaded); err != nil {
					a.statusLine = err.Error()
					return nil
				}
				return nil
			}
			a.state = loaded
			a.statusLine = fmt.Sprintf("Loaded %d library / %d party characters; no campaign was saved.", len(loaded.CharacterLibrary), len(loaded.Party))
		}
	case modeCreation:
		return a.updateCreation()
	case modeAdventure:
		if a.justPressed(ebiten.KeyEscape) {
			a.mode = modeMenu
			a.statusLine = "Returned from the initial event."
			return nil
		}
		if a.eventMachine != nil && a.introWaiting && (a.justPressed(ebiten.KeyEnter) || a.justPressed(ebiten.KeySpace)) {
			selection := uint16(0)
			a.introWaiting = false
			if !a.tourActive {
				a.tourActive, a.tourStep = true, -1
			}
			return a.runECLUntilBoundary(&selection)
		}
		if a.introWaiting && (a.justPressed(ebiten.KeyEnter) || a.justPressed(ebiten.KeySpace)) {
			a.introWaiting, a.tourActive = false, true
			a.tourStep, a.tourPage, a.tourDelay = -1, -1, 0
			a.statusLine = "Running the original 34-step Rolf tour; movement remains disabled."
			return nil
		}
		if a.tourActive {
			if a.eventMachine != nil {
				if a.introWaiting {
					return nil
				}
				if a.tourDelay > 0 {
					a.tourDelay--
					return nil
				}
				return a.runECLUntilBoundary(nil)
			}
			if a.initialEvent == nil {
				return fmt.Errorf("Pool initial tour is not configured")
			}
			if a.tourPage >= 0 {
				if a.justPressed(ebiten.KeyEnter) || a.justPressed(ebiten.KeySpace) {
					step := a.initialEvent.Tour[a.tourStep]
					if a.tourPage+1 < len(step.Messages) {
						a.tourPage++
					} else {
						a.tourPage, a.tourDelay = -1, 0
					}
				}
				return nil
			}
			if a.tourDelay > 0 {
				a.tourDelay--
				return nil
			}
			a.tourStep++
			if a.tourStep >= len(a.initialEvent.Tour) {
				a.tourActive, a.introDone = false, true
				a.statusLine = "Rolf tour reached ECL EXIT at (0,4), facing 3; player movement policy remains pending."
				return nil
			}
			step := a.initialEvent.Tour[a.tourStep]
			a.spawn = step.Position
			a.tourDelay = tourStepDelayTicks
			if len(step.Messages) != 0 {
				a.tourPage = 0
			}
		}
		if a.introDone {
			if a.cellEventPending {
				if a.treasureActive && a.cellWaitingMenu && len(a.cellMenuOptions) != 0 {
					if a.justPressed(ebiten.KeyArrowLeft) || a.justPressed(ebiten.KeyArrowUp) {
						a.cellMenuCursor = (a.cellMenuCursor + len(a.cellMenuOptions) - 1) % len(a.cellMenuOptions)
						a.eventLabel = a.cellMenuLabel()
						return nil
					}
					if a.justPressed(ebiten.KeyArrowRight) || a.justPressed(ebiten.KeyArrowDown) {
						a.cellMenuCursor = (a.cellMenuCursor + 1) % len(a.cellMenuOptions)
						a.eventLabel = a.cellMenuLabel()
						return nil
					}
					if a.justPressed(ebiten.KeyEnter) || a.justPressed(ebiten.KeySpace) {
						return a.selectTreasureOption()
					}
					a.statusLine = "A Pool treasure menu is active; choose an option."
					return nil
				}
				if a.templeActive && a.templeStage != templeConfirm {
					for index, key := range []ebiten.Key{ebiten.KeyDigit1, ebiten.KeyDigit2, ebiten.KeyDigit3, ebiten.KeyDigit4, ebiten.KeyDigit5, ebiten.KeyDigit6} {
						if index < len(a.state.Party) && a.justPressed(key) {
							a.selectTempleParty(index)
							return nil
						}
					}
				}
				if a.cellWaitingMenu && len(a.cellMenuOptions) != 0 {
					if a.justPressed(ebiten.KeyArrowLeft) || a.justPressed(ebiten.KeyArrowUp) {
						a.cellMenuCursor = (a.cellMenuCursor + len(a.cellMenuOptions) - 1) % len(a.cellMenuOptions)
						a.eventLabel = a.cellMenuLabel()
						return nil
					}
					if a.justPressed(ebiten.KeyArrowRight) || a.justPressed(ebiten.KeyArrowDown) {
						a.cellMenuCursor = (a.cellMenuCursor + 1) % len(a.cellMenuOptions)
						a.eventLabel = a.cellMenuLabel()
						return nil
					}
				}
				if a.justPressed(ebiten.KeyEnter) || a.justPressed(ebiten.KeySpace) {
					if a.templeActive {
						return a.selectSuneTempleOption()
					}
					var selection *uint16
					if a.cellWaitingMenu {
						value := uint16(a.cellMenuCursor)
						selection = &value
					}
					a.cellEventPending, a.cellWaitingMenu = false, false
					a.cellMenuOptions, a.cellMenuCursor = nil, 0
					return a.continueInitialSearch(selection)
				}
				a.statusLine = "A Pool cell event is active; press ENTER to continue."
				return nil
			}
			if a.justPressed(ebiten.KeyArrowLeft) {
				a.spawn.Facing = uint8((int(a.spawn.Facing) + 7) % 8)
				a.statusLine = "Turned left; Pool event dispatch remains pending."
				return nil
			}
			if a.justPressed(ebiten.KeyArrowRight) {
				a.spawn.Facing = uint8((int(a.spawn.Facing) + 1) % 8)
				a.statusLine = "Turned right; Pool event dispatch remains pending."
				return nil
			}
			if a.justPressed(ebiten.KeyArrowUp) {
				return a.moveInitialDungeonForward()
			}
		}
	}
	return nil
}

func (a *app) moveInitialDungeonForward() error {
	if a.initialMap == nil {
		return fmt.Errorf("Pool initial map is not configured")
	}
	dx, dy := 0, 0
	switch a.spawn.Facing {
	case 0:
		dy = -1
	case 2:
		dx = 1
	case 4:
		dy = 1
	case 6:
		dx = -1
	default:
		a.statusLine = "Face a cardinal direction before moving forward."
		return nil
	}
	if !a.initialMap.Grid.CanMoveDungeonWrapped(int(a.spawn.X), int(a.spawn.Y), int(a.spawn.Facing)) {
		a.statusLine = "A wall or locked door blocks the way."
		return nil
	}
	if a.eventMachine != nil {
		result, err := gamepack.RunInitialSessionCellEntry(a.eventSession, a.initialMap.Grid, a.spawn)
		if err != nil {
			return fmt.Errorf("dispatch Pool initial cell: %w", err)
		}
		result, err = a.consumeInitialTransitionResources(result)
		if err != nil {
			return err
		}
		if !result.Exited || result.WaitingForMenu || len(result.Events) != 0 {
			return a.pauseInitialCellResult(result)
		}
		a.spawn.X = uint8(geometry.WrapCoordinate(int(a.spawn.X)+dx, geometry.Width))
		a.spawn.Y = uint8(geometry.WrapCoordinate(int(a.spawn.Y)+dy, geometry.Height))
		return a.beginInitialSearch()
	}
	a.spawn.X = uint8(geometry.WrapCoordinate(int(a.spawn.X)+dx, geometry.Width))
	a.spawn.Y = uint8(geometry.WrapCoordinate(int(a.spawn.Y)+dy, geometry.Height))
	a.statusLine = "Moved using original GEO data; cell ECL returned normally."
	return nil
}

func (a *app) consumeInitialTransitionResources(result eclvm.Result) (eclvm.Result, error) {
	for boundary := 0; boundary < 8; boundary++ {
		if result.Exited || result.WaitingForMenu || len(result.Events) != 1 {
			return result, nil
		}
		event := result.Events[0]
		if len(event.Arguments) != len(event.ArgumentsValid) {
			return result, fmt.Errorf("Pool resource event 0x%02X has mismatched argument validity", event.Opcode)
		}
		allValid := true
		for _, valid := range event.ArgumentsValid {
			allValid = allValid && valid
		}
		switch event.Opcode {
		case 0x21:
			if !allValid || len(event.Arguments) != 3 || event.Arguments[0] > 0xFF {
				return result, fmt.Errorf("Pool LOAD FILES has invalid arguments %v/%v", event.Arguments, event.ArgumentsValid)
			}
			key := gamepack.MapKey{Archive: a.spawn.Map.Archive, BlockID: uint8(event.Arguments[0])}
			loaded, ok := a.geometryCatalog.Map(key)
			if !ok {
				return result, fmt.Errorf("Pool LOAD FILES requested absent GEO%d block %d", key.Archive, key.BlockID)
			}
			a.initialMap = &loaded
			a.spawn.Map = key
		case 0x37:
			if !allValid || !reflect.DeepEqual(event.Arguments, []uint16{127, 127, 127}) {
				return result, fmt.Errorf("Pool LOAD PIECES %v is not READY", event.Arguments)
			}
		default:
			return result, nil
		}
		next, err := a.eventSession.RunUntilEvent(4096, nil, true)
		if err != nil {
			return result, fmt.Errorf("continue Pool transition resource 0x%02X: %w", event.Opcode, err)
		}
		result = next
	}
	return result, fmt.Errorf("Pool transition resource boundary limit exceeded")
}

func (a *app) beginInitialSearch() error {
	result, err := gamepack.RunInitialSessionSearchEntry(a.eventSession, a.initialMap.Grid, a.spawn)
	if err != nil {
		return fmt.Errorf("start Pool SearchLocation: %w", err)
	}
	return a.consumeInitialSearch(result)
}

func (a *app) continueInitialSearch(selection *uint16) error {
	var selections []uint16
	if selection != nil {
		selections = []uint16{*selection}
	}
	result, err := a.eventSession.RunUntilEvent(4096, selections, true)
	if err != nil {
		return fmt.Errorf("continue Pool SearchLocation: %w", err)
	}
	return a.consumeInitialSearch(result)
}

func (a *app) consumeInitialSearch(result eclvm.Result) error {
	for boundaries := 0; boundaries < 64; boundaries++ {
		var err error
		result, err = a.consumeInitialTransitionResources(result)
		if err != nil {
			return err
		}
		a.applyCellECLResult(result)
		if len(result.TreasureRequests) != 0 {
			return a.enterTreasure(result.TreasureRequests)
		}
		presentationOnly := len(result.Events) == 1 && ((result.Events[0].Opcode == 0x12 && result.Events[0].Text == "") || result.Events[0].Opcode == 0x0E)
		if presentationOnly {
			if result.Events[0].Opcode == 0x12 {
				a.eventText = ""
			}
			next, err := a.eventSession.RunUntilEvent(4096, nil, true)
			if err != nil {
				return fmt.Errorf("continue Pool SearchLocation presentation: %w", err)
			}
			result = next
			continue
		}
		if result.Exited && !result.WaitingForMenu && len(result.Events) == 0 {
			a.cellEventPending, a.cellWaitingMenu = false, false
			a.templeActive = false
			a.cellMenuOptions, a.cellMenuCursor = nil, 0
			a.eventText, a.eventLabel = "", ""
			a.statusLine = "Moved using original GEO data; per-turn and SearchLocation returned normally."
			return nil
		}
		if a.isSuneTempleBoundary(result) {
			return a.enterSuneTemple()
		}
		return a.pauseInitialCellResult(result)
	}
	return fmt.Errorf("Pool SearchLocation exceeded presentation boundary limit")
}

func (a *app) isSuneTempleBoundary(result eclvm.Result) bool {
	if !result.MonstersCleared || a.eventMachine == nil || a.eventMachine.Memory[0x6DE2] != 1 {
		return false
	}
	for _, event := range result.Events {
		if event.Opcode == 0x24 {
			return true
		}
	}
	return false
}

func (a *app) enterTreasure(requests []eclvm.TreasureRequest) error {
	if a.loadTreasure == nil {
		return fmt.Errorf("Pool treasure loader is not configured")
	}
	loaded := make([]gamepack.TreasureItemRecord, 0)
	for _, request := range requests {
		if request.Amounts != ([7]uint16{}) {
			return fmt.Errorf("Pool money treasure %v is not READY", request.Amounts)
		}
		if request.ItemBlock > 0xFF {
			return fmt.Errorf("Pool treasure item block 0x%X exceeds byte range", request.ItemBlock)
		}
		items, err := a.loadTreasure(a.spawn.Map.Archive, uint8(request.ItemBlock))
		if err != nil {
			return err
		}
		loaded = append(loaded, items...)
	}
	a.treasureActive, a.treasureStage = true, treasureMain
	a.treasureItems, a.treasureSelected = loaded, 0
	a.cellEventPending, a.cellWaitingMenu = true, true
	a.cellMenuOptions, a.cellMenuCursor = []string{"View", "Take", "Exit"}, 0
	a.eventText = "The party has found treasure!"
	a.eventLabel = a.cellMenuLabel()
	a.statusLine = fmt.Sprintf("Original Pool treasure service: %d item(s).", len(loaded))
	return nil
}

func (a *app) enterTreasureMain() {
	a.treasureStage = treasureMain
	a.cellMenuOptions, a.cellMenuCursor = []string{"View", "Take", "Exit"}, 0
	a.eventText = "The party has found treasure!"
	a.eventLabel = a.cellMenuLabel()
}

func (a *app) selectTreasureOption() error {
	switch a.treasureStage {
	case treasureMain:
		switch a.cellMenuCursor {
		case 0:
			a.treasureStage = treasureView
			names := make([]string, len(a.treasureItems))
			for index := range a.treasureItems {
				names[index] = a.treasureItems[index].Name
			}
			a.eventText = strings.Join(names, " / ")
			a.cellMenuOptions, a.cellMenuCursor = []string{"Return"}, 0
			a.eventLabel = a.cellMenuLabel()
			return nil
		case 1:
			if len(a.treasureItems) == 0 {
				a.eventText = "There are no items left."
				return nil
			}
			a.treasureStage = treasureItems
			a.cellMenuOptions = make([]string, 0, len(a.treasureItems)+1)
			for _, item := range a.treasureItems {
				a.cellMenuOptions = append(a.cellMenuOptions, item.Name)
			}
			a.cellMenuOptions = append(a.cellMenuOptions, "Exit")
			a.cellMenuCursor = 0
			a.eventText = "Take: Items"
			a.eventLabel = a.cellMenuLabel()
			return nil
		case 2:
			if len(a.treasureItems) == 0 {
				return a.exitTreasure()
			}
			a.treasureStage = treasureConfirmExit
			a.cellMenuOptions, a.cellMenuCursor = []string{"Yes", "No"}, 0
			a.eventText = "There is still treasure left. Do you want to leave it?"
			a.eventLabel = a.cellMenuLabel()
			return nil
		}
	case treasureView:
		a.enterTreasureMain()
		return nil
	case treasureItems:
		if a.cellMenuCursor == len(a.treasureItems) {
			a.enterTreasureMain()
			return nil
		}
		a.treasureSelected = a.cellMenuCursor
		a.treasureStage = treasureCharacter
		a.cellMenuOptions = make([]string, 0, len(a.state.Party)+1)
		for _, member := range a.state.Party {
			a.cellMenuOptions = append(a.cellMenuOptions, member.Name)
		}
		a.cellMenuOptions = append(a.cellMenuOptions, "Cancel")
		a.cellMenuCursor = 0
		a.eventText = "Who will take " + a.treasureItems[a.treasureSelected].Name + "?"
		a.eventLabel = a.cellMenuLabel()
		return nil
	case treasureCharacter:
		if a.cellMenuCursor == len(a.state.Party) {
			a.treasureStage = treasureItems
			return a.rebuildTreasureItemMenu()
		}
		return a.giveTreasureItem(a.cellMenuCursor)
	case treasureConfirmExit:
		if a.cellMenuCursor == 0 {
			return a.exitTreasure()
		}
		a.enterTreasureMain()
		return nil
	}
	return fmt.Errorf("unknown Pool treasure stage %d", a.treasureStage)
}

func (a *app) rebuildTreasureItemMenu() error {
	if len(a.treasureItems) == 0 {
		a.enterTreasureMain()
		return nil
	}
	a.cellMenuOptions = a.cellMenuOptions[:0]
	for _, item := range a.treasureItems {
		a.cellMenuOptions = append(a.cellMenuOptions, item.Name)
	}
	a.cellMenuOptions = append(a.cellMenuOptions, "Exit")
	a.cellMenuCursor = 0
	a.eventText = "Take: Items"
	a.eventLabel = a.cellMenuLabel()
	return nil
}

func (a *app) giveTreasureItem(partyIndex int) error {
	if partyIndex < 0 || partyIndex >= len(a.state.Party) || a.treasureSelected < 0 || a.treasureSelected >= len(a.treasureItems) {
		return fmt.Errorf("Pool treasure selection is outside party/item range")
	}
	member := a.state.Party[partyIndex]
	rawInventory := make([][]byte, len(member.Inventory))
	for index := range member.Inventory {
		rawInventory[index] = member.Inventory[index].Raw
	}
	record := a.treasureItems[a.treasureSelected]
	ok, err := poolcharacter.CanReceiveItem(member.Abilities[0], member.ExceptionalStrength, rawInventory, record.Raw[:])
	if err != nil {
		return err
	}
	if !ok {
		a.eventText = "OverLoaded"
		a.statusLine = member.Name + " cannot carry that item."
		return nil
	}
	next := cloneSaveState(a.state)
	item := poolsave.Item{Name: record.Name, Raw: append([]byte(nil), record.Raw[:]...)}
	next.Party[partyIndex].Inventory = append(next.Party[partyIndex].Inventory, item)
	found := false
	for index := range next.CharacterLibrary {
		if next.CharacterLibrary[index].Name == member.Name {
			next.CharacterLibrary[index].Inventory = append(next.CharacterLibrary[index].Inventory, item)
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("Pool party member %q is absent from character library", member.Name)
	}
	if a.saveState == nil {
		return fmt.Errorf("Pool save writer is not configured")
	}
	if err := a.saveState(next); err != nil {
		return fmt.Errorf("save Pool treasure transfer: %w", err)
	}
	a.state = next
	a.treasureItems = append(a.treasureItems[:a.treasureSelected], a.treasureItems[a.treasureSelected+1:]...)
	a.treasureStage = treasureItems
	a.statusLine = member.Name + " takes " + record.Name + "."
	return a.rebuildTreasureItemMenu()
}

func cloneSaveState(state poolsave.State) poolsave.State {
	cloneCharacters := func(values []poolsave.Character) []poolsave.Character {
		out := append([]poolsave.Character(nil), values...)
		for index := range out {
			out[index].Inventory = append([]poolsave.Item(nil), values[index].Inventory...)
			for item := range out[index].Inventory {
				out[index].Inventory[item].Raw = append([]byte(nil), values[index].Inventory[item].Raw...)
			}
		}
		return out
	}
	state.CharacterLibrary = cloneCharacters(state.CharacterLibrary)
	state.Party = cloneCharacters(state.Party)
	if state.Campaign != nil {
		campaign := *state.Campaign
		campaign.Session.TransitionEntries = append([]int(nil), campaign.Session.TransitionEntries...)
		campaign.Session.PendingEntries = append([]int(nil), campaign.Session.PendingEntries...)
		campaign.Session.Machine.Stack = append([]int(nil), campaign.Session.Machine.Stack...)
		campaign.Session.Machine.Memory = append([]eclvm.MemoryWord(nil), campaign.Session.Machine.Memory...)
		campaign.Session.Machine.Strings = append([]eclvm.StringWord(nil), campaign.Session.Machine.Strings...)
		state.Campaign = &campaign
	}
	return state
}

func (a *app) stateForSave() (poolsave.State, error) {
	next := cloneSaveState(a.state)
	if a.mode != modeAdventure {
		return next, nil
	}
	if a.eventSession == nil {
		return poolsave.State{}, fmt.Errorf("cannot save Pool campaign without an ECL session")
	}
	if a.introWaiting || a.tourActive || a.cellEventPending || a.cellWaitingMenu || a.templeActive || a.treasureActive {
		return poolsave.State{}, fmt.Errorf("finish the current Pool dialogue or service before saving")
	}
	snapshot, err := a.eventSession.Snapshot()
	if err != nil {
		return poolsave.State{}, fmt.Errorf("snapshot Pool ECL session: %w", err)
	}
	next.Campaign = &poolsave.Campaign{
		MapArchive: a.spawn.Map.Archive, MapBlock: a.spawn.Map.BlockID,
		X: a.spawn.X, Y: a.spawn.Y, Facing: a.spawn.Facing, Session: snapshot,
	}
	return next, nil
}

func (a *app) restoreCampaign(loaded poolsave.State) error {
	if loaded.Campaign == nil {
		return fmt.Errorf("Pool save has no campaign")
	}
	if a.initialEvent == nil {
		return fmt.Errorf("Pool campaign event catalog is not configured")
	}
	campaign := loaded.Campaign
	key := gamepack.MapKey{Archive: campaign.MapArchive, BlockID: campaign.MapBlock}
	geometryMap, ok := a.geometryCatalog.Map(key)
	if !ok {
		return fmt.Errorf("Pool save map GEO%d block %d is unavailable", key.Archive, key.BlockID)
	}
	characters := make([]gamepack.InitialCharacter, len(loaded.Party))
	for index, character := range loaded.Party {
		characters[index] = gamepack.InitialCharacter{
			Name: character.Name, ClassID: character.ClassID, Abilities: character.Abilities,
			ExceptionalStrength: character.ExceptionalStrength, CurrentHP: character.CurrentHP,
		}
	}
	session, err := gamepack.NewInitialEventSession(*a.initialEvent, characters...)
	if err != nil {
		return fmt.Errorf("rebuild Pool ECL session: %w", err)
	}
	if err := session.Restore(campaign.Session); err != nil {
		return fmt.Errorf("restore Pool ECL session: %w", err)
	}
	// Commit only after every catalog and snapshot check succeeds.
	a.state = cloneSaveState(loaded)
	a.spawn = gamepack.Spawn{Map: key, X: campaign.X, Y: campaign.Y, Facing: campaign.Facing}
	a.initialMap = &geometryMap
	a.eventSession, a.eventMachine = session, session.Machine()
	a.introWaiting, a.introDone = false, true
	a.tourActive, a.tourStep, a.tourPage, a.tourDelay = false, -1, -1, 0
	a.cellEventPending, a.cellWaitingMenu = false, false
	a.templeActive, a.treasureActive = false, false
	a.eventText, a.eventLabel, a.cellMenuOptions = "", "", nil
	a.mode = modeAdventure
	a.statusLine = fmt.Sprintf("Campaign restored at GEO%d block %d (%d,%d).", key.Archive, key.BlockID, campaign.X, campaign.Y)
	return nil
}

func (a *app) exitTreasure() error {
	a.treasureActive, a.treasureStage = false, treasureMain
	a.treasureItems, a.cellMenuOptions = nil, nil
	a.cellEventPending, a.cellWaitingMenu = false, false
	a.eventText, a.eventLabel = "", ""
	result, err := a.eventSession.RunUntilEvent(4096, nil, true)
	if err != nil {
		return fmt.Errorf("continue after Pool treasure service: %w", err)
	}
	return a.consumeInitialSearch(result)
}

func (a *app) enterSuneTemple() error {
	if len(a.state.Party) == 0 || strings.TrimSpace(a.state.Party[0].Name) == "" {
		return fmt.Errorf("Sune temple requires a named first party member")
	}
	a.templeActive = true
	a.templeStage, a.templeParty, a.templeService = templeMain, 0, 0
	a.cellEventPending, a.cellWaitingMenu = true, true
	a.cellMenuOptions = []string{"Heal", "View", "Pool", "Appraise", "Exit"}
	a.cellMenuCursor = 0
	a.eventText = strings.TrimSpace(a.state.Party[0].Name) + ", how can we help you?"
	a.eventLabel = a.cellMenuLabel()
	a.statusLine = "Original Sune temple menu; press 1-6 to select the current character."
	return nil
}

var templeHealOptions = []string{
	"Cure Blindness", "Cure Disease", "Cure Light Wounds", "Cure Serious Wounds",
	"Cure Critical Wounds", "Neutralize Poison", "Raise Dead", "Remove Curse",
	"Stone to Flesh", "Exit",
}

func (a *app) selectTempleParty(index int) {
	a.templeParty = index
	name := strings.TrimSpace(a.state.Party[index].Name)
	if a.templeStage == templeMain {
		a.eventText = name + ", how can we help you?"
	} else {
		a.eventText = "Choose a cure for " + name + "."
	}
	a.statusLine = fmt.Sprintf("Temple character %d/%d: %s (HP %d/%d, %d GP; pool %d GP).", index+1, len(a.state.Party), name, a.state.Party[index].CurrentHP, a.state.Party[index].MaxHP, a.state.Party[index].Gold, a.state.PooledGold)
}

func (a *app) enterTempleMain() {
	a.templeStage = templeMain
	a.cellMenuOptions = []string{"Heal", "View", "Pool", "Appraise", "Exit"}
	a.cellMenuCursor = 0
	a.eventLabel = a.cellMenuLabel()
	a.selectTempleParty(a.templeParty)
}

func (a *app) enterTempleHeal() {
	a.templeStage = templeHeal
	a.cellMenuOptions = append(a.cellMenuOptions[:0], templeHealOptions...)
	a.cellMenuCursor = 0
	a.eventLabel = a.cellMenuLabel()
	a.selectTempleParty(a.templeParty)
}

func (a *app) selectSuneTempleOption() error {
	switch a.templeStage {
	case templeMain:
		switch a.cellMenuCursor {
		case 0:
			a.enterTempleHeal()
			return nil
		case len(a.cellMenuOptions) - 1:
			return a.leaveSuneTemple()
		default:
			a.statusLine = "This temple service remains fail-closed until its DOS rules are READY."
			return nil
		}
	case templeHeal:
		if a.cellMenuCursor == len(templeHealOptions)-1 {
			a.enterTempleMain()
			return nil
		}
		if a.cellMenuCursor < 2 || a.cellMenuCursor > 4 {
			a.statusLine = "This status cure remains fail-closed until its DOS rules are READY."
			return nil
		}
		a.templeService = a.cellMenuCursor - 2
		service := temple.WoundServices[a.templeService]
		a.templeStage = templeConfirm
		a.cellMenuOptions = []string{"YES", "NO"}
		a.cellMenuCursor = 0
		a.eventText = fmt.Sprintf("%d gold pieces.\npay for cure", service.Cost)
		a.eventLabel = a.cellMenuLabel()
		a.statusLine = "Confirm the original temple cure price."
		return nil
	case templeConfirm:
		if a.cellMenuCursor != 0 {
			a.enterTempleHeal()
			return nil
		}
		before := a.state
		before.Party = append([]poolsave.Character(nil), a.state.Party...)
		before.CharacterLibrary = append([]poolsave.Character(nil), a.state.CharacterLibrary...)
		result, err := temple.CureWounds(&a.state, a.templeParty, a.templeService, a.roller)
		if err != nil {
			a.enterTempleHeal()
			if errors.Is(err, temple.ErrNotEnoughMoney) {
				a.eventText = "Not enough money."
				a.statusLine = "The cure was not purchased; no money or HP changed."
				return nil
			}
			return err
		}
		if a.saveState != nil {
			if err := a.saveState(a.state); err != nil {
				a.state = before
				return err
			}
		}
		name := a.state.Party[a.templeParty].Name
		a.enterTempleHeal()
		a.eventText = name + " is cured."
		a.statusLine = fmt.Sprintf("Paid %d GP from %s; restored %d HP.", result.Cost, result.PaidFrom, result.Healed)
		return nil
	default:
		return fmt.Errorf("unknown Pool temple stage %d", a.templeStage)
	}
}

func (a *app) leaveSuneTemple() error {
	a.templeActive = false
	a.templeStage, a.templeParty, a.templeService = templeMain, 0, 0
	a.cellEventPending, a.cellWaitingMenu = false, false
	a.cellMenuOptions, a.cellMenuCursor = nil, 0
	a.eventText, a.eventLabel = "", ""
	return a.continueInitialSearch(nil)
}

func (a *app) pauseInitialCellResult(result eclvm.Result) error {
	a.applyCellECLResult(result)
	a.templeActive = false
	a.cellEventPending = true
	a.cellWaitingMenu = result.WaitingForMenu
	if result.WaitingForMenu && len(result.Menus) != 0 {
		menu := result.Menus[len(result.Menus)-1]
		a.cellMenuOptions = append(a.cellMenuOptions[:0], menu.Options...)
		a.cellMenuCursor = 0
		if menu.Prompt != "" {
			a.eventText = menu.Prompt
		}
		a.eventLabel = a.cellMenuLabel()
	}
	if a.eventLabel == "" {
		a.eventLabel = "RETURN"
	}
	if a.eventText != "" {
		a.statusLine = "Original Pool cell text is waiting for RETURN."
	} else {
		a.statusLine = "A Pool external event boundary is pending implementation."
	}
	return nil
}

func (a *app) cellMenuLabel() string {
	if len(a.cellMenuOptions) == 0 {
		return "RETURN"
	}
	parts := make([]string, len(a.cellMenuOptions))
	for index, option := range a.cellMenuOptions {
		if index == a.cellMenuCursor {
			parts[index] = "> " + option
		} else {
			parts[index] = "  " + option
		}
	}
	return strings.Join(parts, "   ")
}

func (a *app) applyCellECLResult(result eclvm.Result) {
	for _, write := range result.Writes {
		switch write.Address {
		case 0xC04B:
			a.spawn.X = uint8(write.Value)
		case 0xC04C:
			a.spawn.Y = uint8(write.Value)
		case 0xC04D:
			a.spawn.Facing = uint8(write.Value)
		}
	}
	for _, event := range result.Events {
		if event.Text != "" {
			a.eventText = event.Text
		}
	}
}

func (a *app) applyECLResult(result eclvm.Result) {
	for _, write := range result.Writes {
		switch write.Address {
		case 0xC04B:
			a.spawn.X = uint8(write.Value)
		case 0xC04C:
			a.spawn.Y = uint8(write.Value)
		case 0xC04D:
			a.spawn.Facing = uint8(write.Value)
		}
	}
	for _, event := range result.Events {
		if event.Text != "" {
			a.eventText = event.Text
		}
		if event.Opcode == 0x3A {
			a.tourStep++
			a.tourDelay = tourStepDelayTicks
		}
	}
	if result.WaitingForMenu {
		a.introWaiting = true
		if len(result.Menus) != 0 && len(result.Menus[len(result.Menus)-1].Options) != 0 {
			a.eventLabel = result.Menus[len(result.Menus)-1].Options[0]
		}
	}
	if result.Exited {
		a.tourActive, a.introWaiting, a.introDone = false, false, true
		a.statusLine = "Rolf tour reached ECL EXIT at (0,4), facing 3; player movement policy remains pending."
	}
}

func (a *app) runECLUntilBoundary(selection *uint16) error {
	for step := 0; step < 4096; step++ {
		var selections []uint16
		if selection != nil {
			selections = []uint16{*selection}
			selection = nil
		}
		result, err := a.eventMachine.Run(1, selections, true)
		if err != nil {
			return fmt.Errorf("run Pool initial ECL: %w", err)
		}
		a.applyECLResult(result)
		if result.WaitingForMenu || result.Exited || a.tourDelay > 0 {
			return nil
		}
	}
	return fmt.Errorf("Pool initial ECL did not reach a bounded frontend boundary")
}

func (a *app) updateCreation() error {
	if a.justPressed(ebiten.KeyEscape) {
		if a.flow.Stage == creation.StageRace {
			a.mode, a.cursor, a.rolled = modeMenu, 0, nil
			return nil
		}
		a.flow.Back()
		if a.flow.Stage == creation.StageName {
			a.nameInput = a.flow.Name
		}
		if a.flow.Stage != creation.StagePortrait {
			a.portrait = nil
		}
		a.cursor = 0
		if a.flow.Stage <= creation.StageAlignment {
			a.rolled = nil
		}
		return nil
	}
	if a.flow.Stage == creation.StageRoll {
		if a.rolled == nil || a.justPressed(ebiten.KeyR) {
			rolled, err := a.flow.Roll(a.roller)
			if err != nil {
				return err
			}
			a.rolled = &rolled
		}
		if a.justPressed(ebiten.KeyEnter) || a.justPressed(ebiten.KeyY) {
			if err := a.flow.AcceptRoll(); err != nil {
				return err
			}
			a.nameInput, a.statusLine = "", ""
		}
		return nil
	}
	if a.flow.Stage == creation.StageName {
		if a.justPressed(ebiten.KeyBackspace) && len(a.nameInput) > 0 {
			a.nameInput = a.nameInput[:len(a.nameInput)-1]
		}
		for _, entered := range a.inputChars() {
			if entered >= 0x20 && entered <= 0x7E && len(a.nameInput) < 15 {
				a.nameInput += strings.ToUpper(string(entered))
			}
		}
		if a.justPressed(ebiten.KeyEnter) {
			if err := a.flow.SetName(a.nameInput); err != nil {
				a.statusLine = err.Error()
				return nil
			}
			a.statusLine = ""
			return a.reloadPortrait()
		}
		return nil
	}
	if a.flow.Stage == creation.StagePortrait {
		changed := false
		if a.justPressed(ebiten.KeyH) {
			if err := a.flow.NextPortraitHead(); err != nil {
				return err
			}
			changed = true
		}
		if a.justPressed(ebiten.KeyB) {
			if err := a.flow.NextPortraitBody(); err != nil {
				return err
			}
			changed = true
		}
		if changed {
			return a.reloadPortrait()
		}
		if a.justPressed(ebiten.KeyK) {
			if err := a.flow.KeepPortrait(); err != nil {
				return err
			}
			a.statusLine = "Portrait accepted."
			return a.reloadIcons()
		}
		return nil
	}
	if a.flow.Stage == creation.StageIcon {
		changed := false
		if a.justPressed(ebiten.KeyH) {
			if err := a.flow.NextIconHead(); err != nil {
				return err
			}
			changed = true
		}
		if a.justPressed(ebiten.KeyW) {
			if err := a.flow.NextIconWeapon(); err != nil {
				return err
			}
			changed = true
		}
		if a.justPressed(ebiten.KeyP) {
			if err := a.flow.SelectNextIconPart(); err != nil {
				return err
			}
		}
		if a.justPressed(ebiten.KeyDigit1) {
			if err := a.flow.NextIconColor(0); err != nil {
				return err
			}
			changed = true
		}
		if a.justPressed(ebiten.KeyDigit2) {
			if err := a.flow.NextIconColor(1); err != nil {
				return err
			}
			changed = true
		}
		if a.justPressed(ebiten.KeyS) {
			if err := a.flow.ToggleIconSize(); err != nil {
				return err
			}
			changed = true
		}
		if a.justPressed(ebiten.KeyEnter) || a.justPressed(ebiten.KeyE) {
			return a.flow.RequestIconConfirmation()
		}
		if changed {
			return a.reloadIcons()
		}
		return nil
	}
	if a.flow.Stage == creation.StageIconConfirm {
		if a.justPressed(ebiten.KeyN) {
			return a.flow.RejectIconConfirmation()
		}
		if a.justPressed(ebiten.KeyY) || a.justPressed(ebiten.KeyEnter) {
			return a.finishCharacter()
		}
		return nil
	}
	options := a.flow.Options()
	if a.justPressed(ebiten.KeyArrowUp) || a.justPressed(ebiten.KeyHome) {
		a.cursor = (a.cursor + len(options) - 1) % len(options)
	}
	if a.justPressed(ebiten.KeyArrowDown) || a.justPressed(ebiten.KeyEnd) {
		a.cursor = (a.cursor + 1) % len(options)
	}
	if a.justPressed(ebiten.KeyEnter) {
		if err := a.flow.Select(a.cursor); err != nil {
			return err
		}
		a.cursor = 0
	}
	return nil
}

func (a *app) finishCharacter() error {
	if a.rolled == nil {
		return fmt.Errorf("Pool character confirmation has no rolled character")
	}
	for _, existing := range a.state.CharacterLibrary {
		if existing.Name == a.flow.Name {
			a.statusLine = "A character with that name already exists."
			return nil
		}
	}
	rolled := a.rolled
	character := poolsave.Character{
		Name: a.flow.Name, RaceID: a.flow.SelectedRace().ID, GenderID: a.flow.SelectedGender().ID,
		ClassID: a.flow.SelectedClass().ID, AlignmentID: a.flow.SelectedAlignment().ID,
		Age: rolled.Age, Abilities: rolled.Abilities, ExceptionalStrength: rolled.ExceptionalStrength,
		Gold: rolled.Gold, MaxHP: rolled.HP, CurrentHP: rolled.HP, RawHP: rolled.RawHP,
		PortraitHead: a.flow.PortraitHead, PortraitBody: a.flow.PortraitBody,
		IconHead: a.flow.IconHead, IconWeapon: a.flow.IconWeapon, IconSize: a.flow.IconSize, IconColors: a.flow.IconColors,
	}
	a.state.CharacterLibrary = append(a.state.CharacterLibrary, character)
	if a.saveState != nil {
		if err := a.saveState(a.state); err != nil {
			a.state.CharacterLibrary = a.state.CharacterLibrary[:len(a.state.CharacterLibrary)-1]
			a.statusLine = err.Error()
			return nil
		}
	}
	a.mode, a.flow, a.cursor, a.rolled = modeMenu, creation.NewFlow(), 0, nil
	a.portrait, a.iconReady, a.iconAction = nil, nil, nil
	a.statusLine = character.Name + " saved to the character library."
	return nil
}

func (a *app) addFirstLibraryCharacter() error {
	if len(a.state.Party) >= 6 {
		a.statusLine = "The party already has six characters."
		return nil
	}
	for _, candidate := range a.state.CharacterLibrary {
		present := false
		for _, member := range a.state.Party {
			if member.Name == candidate.Name {
				present = true
				break
			}
		}
		if present {
			continue
		}
		a.state.Party = append(a.state.Party, candidate)
		if a.saveState != nil {
			if err := a.saveState(a.state); err != nil {
				a.state.Party = a.state.Party[:len(a.state.Party)-1]
				a.statusLine = err.Error()
				return nil
			}
		}
		a.statusLine = candidate.Name + " added to the party."
		return nil
	}
	a.statusLine = "No unassigned character is available."
	return nil
}

func (a *app) Draw(screen *ebiten.Image) {
	background, foreground, accent := color.RGBA{0, 0, 0, 255}, color.RGBA{170, 255, 255, 255}, color.RGBA{255, 255, 85, 255}
	if a.modern {
		background, foreground, accent = color.RGBA{16, 20, 30, 255}, color.RGBA{238, 232, 207, 255}, color.RGBA{255, 202, 72, 255}
	}
	screen.Fill(background)
	if a.mode == modeTitle {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(2, 2)
		screen.DrawImage(a.title, op)
		drawText(screen, "ENTER / SPACE", 264, 382, accent)
	} else if a.mode == modeMenu {
		drawFrame(screen, foreground, accent)
		drawText(screen, "PARTY CREATION MENU", 224, 54, accent)
		drawText(screen, "C  CREATE NEW CHARACTER", 176, 112, foreground)
		drawText(screen, "A  ADD CHARACTER TO PARTY", 176, 140, foreground)
		drawText(screen, "L  LOAD SAVED GAME", 176, 168, foreground)
		drawText(screen, "B  BEGIN ADVENTURING", 176, 196, foreground)
		drawText(screen, fmt.Sprintf("LIBRARY %d   PARTY %d/6", len(a.state.CharacterLibrary), len(a.state.Party)), 176, 230, accent)
		for index, member := range a.state.Party {
			drawText(screen, fmt.Sprintf("%d  %s", index+1, member.Name), 176, 254+index*18, foreground)
		}
		if a.statusLine != "" {
			drawText(screen, a.statusLine, 72, 350, foreground)
		}
	} else if a.mode == modeCreation {
		drawCreation(screen, a, foreground, accent)
	} else {
		drawAdventure(screen, a, foreground, accent)
	}
	drawText(screen, "F1 Help  F2 Theme  ESC Back  F10 Quit", 16, 390, foreground)
	if a.help {
		drawHelp(screen, background, foreground, accent)
	}
}

func drawAdventure(screen *ebiten.Image, a *app, foreground, accent color.Color) {
	drawFrame(screen, foreground, accent)
	drawText(screen, "INITIAL DOS FIRST-PERSON VIEW", 176, 52, accent)
	if a.initialMap == nil || a.initialWalls == nil {
		drawText(screen, "INITIAL MAP OR WALL ART IS NOT LOADED", 150, 190, foreground)
		return
	}
	viewLeft, viewTop := 48, 86
	for y := 0; y < 176; y++ {
		shade := color.RGBA{0, 0, 170, 255}
		if y >= 88 {
			shade = color.RGBA{85, 85, 85, 255}
		}
		for x := 0; x < 176; x++ {
			screen.Set(viewLeft+x, viewTop+y, shade)
		}
	}
	stamps, err := initialWallStamps(a.initialMap.Grid, *a.initialWalls, a.spawn)
	if err != nil {
		drawText(screen, "WALL VIEW ERROR", 72, 180, accent)
	} else {
		for _, stamp := range stamps {
			rgba, renderErr := stamp.Picture.RGBA(stamp.Item, graphics.EGA16)
			if renderErr != nil {
				continue
			}
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(2, 2)
			op.GeoM.Translate(float64(viewLeft+stamp.Column*16), float64(viewTop+stamp.Row*16))
			screen.DrawImage(ebiten.NewImageFromImage(rgba), op)
		}
	}
	drawText(screen, fmt.Sprintf("GEO%d BLOCK %d", a.spawn.Map.Archive, a.spawn.Map.BlockID), 310, 106, foreground)
	drawText(screen, fmt.Sprintf("X %d  Y %d  FACING %d", a.spawn.X, a.spawn.Y, a.spawn.Facing), 310, 136, foreground)
	drawText(screen, "GEO / WALL SOURCE: EXACT", 310, 184, accent)
	drawText(screen, "VIEW TRAVERSAL: STRONG INFERENCE", 310, 210, accent)
	moveStatus := "MOVE POLICY: PENDING / DISABLED"
	if a.introDone {
		moveStatus = "GEO WALK: ENABLED / EVENTS PENDING"
	}
	drawText(screen, moveStatus, 310, 246, foreground)
	if a.initialEvent != nil {
		drawText(screen, fmt.Sprintf("FIRST EVENT: ROLF / MONSTER %d", a.initialEvent.MonsterID), 310, 272, foreground)
	} else {
		drawText(screen, "FIRST EVENT: NOT LOADED", 310, 272, foreground)
	}
	drawText(screen, "ESC: PARTY CREATION MENU", 310, 308, foreground)
	if a.tourActive && a.initialEvent != nil {
		drawText(screen, fmt.Sprintf("TOUR STEP %02d / %02d", a.tourStep+1, len(a.initialEvent.Tour)), 310, 294, accent)
	}
	dialogueVisible := false
	if a.introWaiting && a.initialEvent != nil {
		message, label := a.initialEvent.Message, a.initialEvent.ContinueLabel
		if a.eventText != "" {
			message = a.eventText
		}
		if a.eventLabel != "" {
			label = a.eventLabel
		}
		drawDialogue(screen, message, label, foreground, accent)
		dialogueVisible = true
	} else if a.tourActive && a.tourPage >= 0 && a.initialEvent != nil && a.tourStep >= 0 && a.tourStep < len(a.initialEvent.Tour) {
		step := a.initialEvent.Tour[a.tourStep]
		if a.tourPage < len(step.Messages) {
			drawDialogue(screen, step.Messages[a.tourPage], a.initialEvent.ContinueLabel, foreground, accent)
			dialogueVisible = true
		}
	} else if a.cellEventPending && a.eventText != "" {
		drawDialogue(screen, a.eventText, a.eventLabel, foreground, accent)
		dialogueVisible = true
	}
	if a.statusLine != "" && !dialogueVisible {
		drawText(screen, a.statusLine, 42, 342, foreground)
	}
}

func drawDialogue(screen *ebiten.Image, message, label string, foreground, accent color.Color) {
	panel := ebiten.NewImage(560, 142)
	panel.Fill(color.RGBA{0, 0, 0, 255})
	screen.DrawImage(panel, &ebiten.DrawImageOptions{GeoM: translated(40, 198)})
	for x := 40; x < 600; x++ {
		screen.Set(x, 198, accent)
		screen.Set(x, 339, accent)
	}
	for y := 198; y <= 339; y++ {
		screen.Set(40, y, accent)
		screen.Set(599, y, accent)
	}
	for index, line := range wrapASCII(message, 74) {
		if index >= 6 {
			break
		}
		drawText(screen, line, 52, 218+index*16, foreground)
	}
	drawText(screen, label, 52, 326, accent)
}

func translated(x, y float64) ebiten.GeoM {
	var result ebiten.GeoM
	result.Translate(x, y)
	return result
}

func wrapASCII(value string, width int) []string {
	words := strings.Fields(value)
	if len(words) == 0 || width < 1 {
		return nil
	}
	lines := []string{words[0]}
	for _, word := range words[1:] {
		last := len(lines) - 1
		if len(lines[last])+1+len(word) <= width {
			lines[last] += " " + word
			continue
		}
		lines = append(lines, word)
	}
	return lines
}

func initialWallStamps(grid geometry.Grid, piece graphics.PieceSet, spawn gamepack.Spawn) ([]graphics.WallStamp, error) {
	view, err := viewport.TraverseWallViewWrapped(grid, spawn.Facing, int(spawn.X), int(spawn.Y))
	if err != nil {
		return nil, err
	}
	var result []graphics.WallStamp
	for _, call := range view.Calls {
		stamps, err := graphics.BuildWallLayout(piece, call.WallType, call.Layout, call.RowStart, call.ColStart)
		if err != nil {
			continue
		}
		for _, stamp := range stamps {
			if stamp.Row < 0 || stamp.Row > 10 || stamp.Column < 0 || stamp.Column > 10 {
				continue
			}
			result = append(result, stamp)
		}
	}
	return result, nil
}

func drawCreation(screen *ebiten.Image, a *app, foreground, accent color.Color) {
	drawFrame(screen, foreground, accent)
	if a.flow.Stage == creation.StageRoll {
		drawText(screen, "CHARACTER SHEET", 230, 42, accent)
		if a.rolled == nil {
			drawText(screen, "Rolling...", 40, 82, foreground)
			return
		}
		value := a.rolled
		drawText(screen, fmt.Sprintf("%s  %s  %s", a.flow.SelectedGender().Label, a.flow.SelectedRace().Label, a.flow.SelectedClass().Label), 48, 82, foreground)
		drawText(screen, fmt.Sprintf("AGE %d", value.Age), 48, 110, foreground)
		for index, name := range []string{"STR", "INT", "WIS", "DEX", "CON", "CHA"} {
			extra := ""
			if index == 0 && value.ExceptionalStrength != 0 {
				extra = fmt.Sprintf("/%02d", value.ExceptionalStrength)
			}
			drawText(screen, fmt.Sprintf("%-3s %2d%s", name, value.Abilities[index], extra), 48+(index/3)*180, 150+(index%3)*28, foreground)
		}
		drawText(screen, fmt.Sprintf("GOLD %d     HP %d/%d", value.Gold, value.HP, value.HP), 48, 252, foreground)
		drawText(screen, "KEEP THIS CHARACTER?  ENTER/Y = YES   R = REROLL", 48, 302, accent)
		if a.statusLine != "" {
			drawText(screen, a.statusLine, 48, 334, foreground)
		}
		return
	}
	if a.flow.Stage == creation.StageName {
		drawText(screen, "CHARACTER NAME:", 160, 128, accent)
		drawText(screen, a.nameInput+"_", 160, 164, foreground)
		drawText(screen, "1-15 CHARACTERS; ENTER ACCEPTS", 160, 214, foreground)
		if a.statusLine != "" {
			drawText(screen, a.statusLine, 48, 334, foreground)
		}
		return
	}
	if a.flow.Stage == creation.StagePortrait {
		drawText(screen, "HEAD / BODY / KEEP", 48, 72, accent)
		drawText(screen, fmt.Sprintf("H HEAD %02d/14", a.flow.PortraitHead), 48, 118, foreground)
		drawText(screen, fmt.Sprintf("B BODY %02d/12", a.flow.PortraitBody), 48, 150, foreground)
		drawText(screen, "K KEEP", 48, 182, foreground)
		if a.portrait != nil {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(2, 2)
			op.GeoM.Translate(448, 16)
			screen.DrawImage(a.portrait, op)
		}
		return
	}
	if a.flow.Stage == creation.StageIcon {
		drawText(screen, "COMBAT ICON EDITOR", 216, 72, accent)
		parts := []string{"BODY", "ARM", "LEG", "HAIR/FACE", "SHIELD", "WEAPON"}
		drawText(screen, fmt.Sprintf("H HEAD %02d/13   W WEAPON %02d/31", a.flow.IconHead, a.flow.IconWeapon), 48, 116, foreground)
		drawText(screen, fmt.Sprintf("P PART %-9s  1 COLOR-1 %X  2 COLOR-2 %X", parts[a.flow.IconPart], a.flow.IconColors[a.flow.IconPart][0], a.flow.IconColors[a.flow.IconPart][1]), 48, 146, foreground)
		size := "LARGE"
		if a.flow.IconSize == 1 {
			size = "SMALL"
		}
		drawText(screen, "S SIZE "+size, 48, 176, foreground)
		drawText(screen, "READY", 356, 118, accent)
		drawText(screen, "ACTION", 472, 118, accent)
		for index, icon := range []*ebiten.Image{a.iconReady, a.iconAction} {
			if icon == nil {
				continue
			}
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(4, 4)
			op.GeoM.Translate(float64(340+index*116), 142)
			screen.DrawImage(icon, op)
		}
		drawText(screen, "ALL DOS OPTIONS ARE KEPT; DIRECT KEYS GUIDE THIS FIRST SLICE.", 48, 298, foreground)
		drawText(screen, creation.HintFor("icon"), 48, 332, foreground)
		return
	}
	if a.flow.Stage == creation.StageIconConfirm {
		drawText(screen, "IS THIS ICON OK?", 230, 138, accent)
		drawText(screen, "Y / ENTER  YES", 230, 190, foreground)
		drawText(screen, "N          NO", 230, 222, foreground)
		if a.statusLine != "" {
			drawText(screen, a.statusLine, 72, 310, foreground)
		}
		return
	}
	title := map[creation.Stage]string{creation.StageRace: "PICK RACE", creation.StageGender: "PICK GENDER", creation.StageClass: "PICK CLASS", creation.StageAlignment: "PICK ALIGNMENT"}[a.flow.Stage]
	drawText(screen, title, 250, 42, accent)
	for index, option := range a.flow.Options() {
		prefix := "  "
		ink := foreground
		if index == a.cursor {
			prefix, ink = "> ", accent
		}
		drawText(screen, prefix+option, 128, 82+index*22, ink)
	}
	drawText(screen, creation.HintFor(stageName(a.flow.Stage)), 32, 346, foreground)
}

func stageName(stage creation.Stage) string {
	switch stage {
	case creation.StageRace:
		return "race"
	case creation.StageClass:
		return "class"
	case creation.StageAlignment:
		return "alignment"
	default:
		return ""
	}
}

func drawFrame(screen *ebiten.Image, foreground, accent color.Color) {
	for inset := 8; inset < 12; inset++ {
		for x := inset; x < logicalWidth-inset; x++ {
			screen.Set(x, inset, accent)
			screen.Set(x, logicalHeight-inset-1, accent)
		}
		for y := inset; y < logicalHeight-inset; y++ {
			screen.Set(inset, y, accent)
			screen.Set(logicalWidth-inset-1, y, accent)
		}
	}
	drawText(screen, "SSI GOLD BOX / POOL REMAKE", 18, 26, foreground)
}

func drawHelp(screen *ebiten.Image, background, foreground, accent color.Color) {
	for y := 54; y < 340; y++ {
		for x := 72; x < 568; x++ {
			screen.Set(x, y, background)
		}
	}
	drawText(screen, "HELP", 292, 80, accent)
	lines := []string{
		"UP/DOWN or HOME/END: choose an item",
		"ENTER: accept the selected item",
		"ESC: return to the previous screen",
		"R: reroll on the character sheet",
		"F2: switch original/modern presentation",
		"B: begin adventure after adding a party member",
		"F10: save the remake state and quit",
	}
	for index, line := range lines {
		drawText(screen, line, 104, 120+index*30, foreground)
	}
}

func drawText(screen *ebiten.Image, value string, x, y int, ink color.Color) {
	text.Draw(screen, strings.ToUpper(value), basicfont.Face7x13, x, y, ink)
}

func (a *app) Layout(_, _ int) (int, int) { return logicalWidth, logicalHeight }

func main() {
	zipPath := flag.String("zip", "Pool of Radiance (1988).zip", "DOS source ZIP used as local asset source")
	flag.Parse()
	game, err := newApp(*zipPath)
	if err != nil {
		log.Fatal(err)
	}
	ebiten.SetWindowSize(960, 600)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowTitle("Pool of Radiance Remake")
	if err := ebiten.RunGame(game); err != nil && err != ebiten.Termination {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
