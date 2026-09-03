package main

import (
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/assets"
	poolcharacter "github.com/wicanr2/Pool-of-Radiance-cht/internal/character"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/creation"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gametext"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/temple"
	pooltreasure "github.com/wicanr2/Pool-of-Radiance-cht/internal/treasure"
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

type stagedMonster struct {
	Spawn  eclvm.MonsterSpawn
	Record gamepack.MonsterRecord
}

const (
	modeTitle screenMode = iota
	modeMenu
	modeCreation
	modeAdventure
)

const (
	treasureMain treasureStage = iota
	treasureView
	treasureTake
	treasureItems
	treasureCharacter
	treasureMoneyCurrency
	treasureMoneyCharacter
	treasureMoneyAmount
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
	tacticalPreview  bool
	language         language
	gameText         *gametext.Catalogue
	tactical         *tacticalState
	journal          *journalState
	itemTypes        *gamepack.ItemTypeTable
	spellParameters  []gamepack.SpellParameters
	encounter        *encounterState
	spells           *spellState
	spellsOpen       bool
	shop             *shopState
	shopActive       bool
	equipment        *equipmentState
	equipmentOpen    bool
	journalOpen      bool
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
	eclCatalog       gamepack.ECLCatalog
	eclArchive       uint8
	initialWalls     *graphics.PieceSet
	loadPieceSlots   func(archive uint8, selectors [3]uint8) (graphics.PieceSet, error)
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
	// programManaging 為真時，隊伍管理畫面是 `38h PROGRAM` 從地圖上開的，
	// 離開時要回地圖並讓 ECL 繼續，不是重新開始冒險。
	programManaging bool
	// programAsking 為真時選單正在問「要不要開隊伍管理」（`38h` 的值 9）。
	programAsking bool
	// programExitsBlock 為真時，關掉隊伍管理之後這個 ECL block 就結束，
	// 不是從原地繼續——值 9 的結尾是呼叫 EXIT 的 handler。
	programExitsBlock bool
	// savingThrows 是 DS:41E6h 那張表（spec 075），2Eh DAMAGE 擲豁免要用。
	savingThrows *gamepack.SavingThrowTable
	// trainParty 是隊伍管理畫面上選中的成員，訓練指令對他生效。
	trainParty int
	// spellMember 是法術畫面上選中的成員，記憶指令對他生效。
	spellMember int
	// 紮營選單（原版 overlay-20）。
	campOpen   bool
	campCursor int
	// 戰鬥中的施法清單（spec 098）。
	castOpen    bool
	castOptions []castOption
	castCursor  int
	// 選目標那一步（原版 overlay-13 的 `Next Prev Manual`）。
	castTargeting       bool
	castTargetingAttack bool
	castTargets      []uint8
	castTargetCursor int
	castPending      castOption
	// levelUpTables 是生命骰、體質加成與職業分類遮罩（spec 097），訓練要用。
	levelUpTables gamepack.LevelUpTables
	// experienceTable 是昇級門檻（spec 071）。
	experienceTable gamepack.ExperienceTable
	// spellSlotTables 是牧師與法師的可記憶數表（spec 072），記憶法術要用。
	spellSlotTables gamepack.SpellSlotTables
	// spellCaster 收著逐支讀過的算法與那批純泛型的（spec 098）。
	spellCaster *gamepack.SpellCaster
	// parlay 是進行中的交涉選單（spec 086）。
	parlay *parlayState
	// eclInput 是進行中的 ECL 輸入列（spec 087）。
	eclInput *eclInputState
	// whoPending 為真時選單正在等玩家挑人（`39h WHO`，spec 090）。
	whoPending bool
	// eclClock 是 `34h ECL CLOCK` 推的那七格（spec 093）。
	eclClock gamepack.ECLClock
	// currentCharacter 是原版 `DS:5CF0h` 那個「目前角色」的索引。
	// `39h` 設定它，`28h ROB` 的範圍 0 讀它。
	currentCharacter int
	loadTreasure     func(archive, block uint8) ([]gamepack.TreasureItemRecord, error)
	loadMonster      func(archive, block uint8) (gamepack.MonsterRecord, error)
	combatActive     bool
	combatMonsters   []stagedMonster
	treasureActive   bool
	treasureStage    treasureStage
	treasureItems    []gamepack.TreasureItemRecord
	treasureSelected int
	treasureCurrency int
	treasureAmount   string
}

// defaultStatePath 是存檔的預設位置。
//
// 用作業系統的使用者設定目錄，不用工作目錄：發行包裡的 AppImage 與 .app 是
// 唯讀的，寫在旁邊會直接失敗，而那個失敗要到玩家按下存檔才會出現。
// 取不到設定目錄時退回工作目錄——那是開發時的行為，也是最後的退路。
func defaultStatePath() string {
	directory, err := os.UserConfigDir()
	if err != nil {
		return filepath.Join("saves", "pool-remake-state.json")
	}
	return filepath.Join(directory, "pool-of-radiance-remake", "state.json")
}

func newApp(zipPath, statePath string) (*app, error) {
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
	eclCatalog, err := gamepack.ReadDOSECLCatalog(zipPath)
	if err != nil {
		return nil, err
	}
	application.eclCatalog = eclCatalog
	application.eclArchive = 3
	initialWalls, err := gamepack.ReadDOSPieceSet(zipPath, 3, 1, 0)
	if err != nil {
		return nil, fmt.Errorf("load DOS initial wall set: %w", err)
	}
	application.initialWalls = &initialWalls
	application.loadPieceSlots = func(archive uint8, selectors [3]uint8) (graphics.PieceSet, error) {
		return gamepack.ReadDOSPieceSlots(zipPath, archive, selectors)
	}
	itemTypes, err := gamepack.ReadDOSItemTypeTable(zipPath)
	if err != nil {
		return nil, err
	}
	initialEvent, err := gamepack.ReadDOSInitialEvent(zipPath)
	if err != nil {
		return nil, fmt.Errorf("load DOS initial event: %w", err)
	}
	savingThrows, err := gamepack.ReadDOSSavingThrowTable(zipPath)
	if err != nil {
		return nil, err
	}
	application.initialEvent = &initialEvent
	application.itemTypes = itemTypes
	application.savingThrows = savingThrows
	levelUpTables, err := gamepack.ReadDOSLevelUpTables(zipPath)
	if err != nil {
		return nil, err
	}
	application.levelUpTables = levelUpTables
	experienceTable, err := gamepack.ReadDOSExperienceTable(zipPath)
	if err != nil {
		return nil, err
	}
	application.experienceTable = experienceTable
	spellSlotTables, err := gamepack.ReadDOSSpellSlotTableSet(zipPath)
	if err != nil {
		return nil, err
	}
	application.spellSlotTables = spellSlotTables
	spellCaster, err := gamepack.ReadDOSSpellCaster(zipPath)
	if err != nil {
		return nil, err
	}
	application.spellCaster = spellCaster
	spellParameters, err := gamepack.ReadDOSSpellParameters(zipPath)
	if err != nil {
		return nil, err
	}
	application.spellParameters = spellParameters
	application.saveState = func(state poolsave.State) error { return poolsave.WriteAtomic(statePath, state) }
	application.loadState = func() (poolsave.State, error) { return poolsave.Read(statePath) }
	application.loadTreasure = func(archive, block uint8) ([]gamepack.TreasureItemRecord, error) {
		return gamepack.ReadDOSTreasureItemBlock(zipPath, archive, block)
	}
	application.loadMonster = func(archive, block uint8) (gamepack.MonsterRecord, error) {
		return gamepack.ReadDOSMonsterRecord(zipPath, archive, block)
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
	if a.justPressed(ebiten.KeyF5) && a.mode == modeAdventure {
		a.tacticalPreview = !a.tacticalPreview
		if a.tacticalPreview {
			if err := a.enterTacticalPreview(); err != nil {
				a.tacticalPreview = false
				a.statusLine = err.Error()
			}
		}
	}
	if a.journalOpen {
		a.journalInput()
		return nil
	}
	if a.shopActive && a.shop != nil {
		if err := a.shopInput(); err != nil {
			a.statusLine = err.Error()
		}
		return nil
	}
	if a.equipmentOpen {
		a.equipmentInput()
		return nil
	}
	if a.spellsOpen {
		a.spellsInput()
		return nil
	}
	if a.campOpen {
		return a.campInput()
	}
	if a.mode == modeAdventure && !a.help && a.tactical == nil && a.justPressed(ebiten.KeyE) {
		a.openCamp()
		return nil
	}
	if a.mode == modeAdventure && !a.help && a.justPressed(ebiten.KeyK) {
		if err := a.openSpells(); err != nil {
			a.statusLine = err.Error()
		}
		return nil
	}
	if a.mode == modeAdventure && !a.help && a.justPressed(ebiten.KeyI) {
		a.openEquipment()
		return nil
	}
	if a.mode == modeAdventure && !a.help && a.justPressed(ebiten.KeyJ) {
		if err := a.openJournal(); err != nil {
			a.statusLine = err.Error()
		}
		return nil
	}
	if a.tacticalPreview && a.mode == modeAdventure && !a.help {
		if err := a.tacticalInput(); err != nil {
			a.statusLine = err.Error()
		}
		return nil
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
		if a.programManaging {
			// 1-6 挑人、T 訓練（spec 097）。原版把訓練掛在這個畫面的 `T` 上。
			for index, key := range []ebiten.Key{ebiten.KeyDigit1, ebiten.KeyDigit2,
				ebiten.KeyDigit3, ebiten.KeyDigit4, ebiten.KeyDigit5, ebiten.KeyDigit6} {
				if index < len(a.state.Party) && a.justPressed(key) {
					a.trainParty = index
					a.statusLine = strings.TrimSpace(a.state.Party[index].Name)
					return nil
				}
			}
			if a.justPressed(ebiten.KeyT) {
				if len(a.state.Party) == 0 {
					a.statusLine = a.text(msgTrainNeedsMember)
					return nil
				}
				if a.trainParty >= len(a.state.Party) {
					a.trainParty = 0
				}
				line, err := a.trainMember(a.trainParty)
				if err != nil {
					return err
				}
				a.statusLine = line
				return nil
			}
			// `38h PROGRAM` 開的隊伍管理：B 或 ESC 回地圖，ECL 從原地繼續。
			// 這裡不能走下面那條「開始冒險」——那會把開場整個重跑一次。
			if a.justPressed(ebiten.KeyB) || a.justPressed(ebiten.KeyEscape) {
				if len(a.state.Party) == 0 {
					a.statusLine = a.text(msgProgramNeedsParty)
					return nil
				}
				return a.closePartyManagement()
			}
		}
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
			a.templeActive, a.combatActive = false, false
			a.combatMonsters = nil
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
				if err := a.configureEventSession(session); err != nil {
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
		// 輸入列吃掉整個影格：ESC 與方向鍵在打字的時候不該有別的意思。
		if handled, err := a.eclInputUpdate(); handled {
			return err
		}
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
				if a.combatActive {
					if a.justPressed(ebiten.KeyEnter) && !a.tacticalPreview {
						if err := a.enterTacticalPreview(); err != nil {
							a.statusLine = err.Error()
							return nil
						}
						a.tacticalPreview = true
						a.statusLine = "Tactical combat entered; deployment and party combat stats are provisional."
						return nil
					}
					if !a.tacticalPreview {
						a.statusLine = "A real Pool encounter is staged; press ENTER to enter tactical combat."
					}
					return nil
				}
				if a.treasureActive && a.cellWaitingMenu && len(a.cellMenuOptions) != 0 {
					if a.treasureStage == treasureMoneyAmount {
						if a.justPressed(ebiten.KeyEscape) {
							a.enterTreasureMain()
							return nil
						}
						if a.justPressed(ebiten.KeyBackspace) && len(a.treasureAmount) != 0 {
							a.treasureAmount = a.treasureAmount[:len(a.treasureAmount)-1]
						}
						for _, entered := range a.inputChars() {
							if entered >= '0' && entered <= '9' && len(a.treasureAmount) < 10 {
								a.treasureAmount += string(entered)
							}
						}
						a.eventLabel = "AMOUNT " + a.treasureAmount + "   ENTER=TAKE   ESC=CANCEL"
						if a.justPressed(ebiten.KeyEnter) {
							return a.takeTreasureMoney()
						}
						return nil
					}
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
					if a.encounter != nil {
						return a.selectEncounterOption()
					}
					if a.templeActive {
						return a.selectSuneTempleOption()
					}
					if a.parlay != nil {
						return a.selectParlayOption()
					}
					if a.whoPending {
						return a.selectWhoOption()
					}
					if a.programAsking {
						return a.selectProgramOption()
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
				a.spawn.Facing = uint8((int(a.spawn.Facing) + 3) % 4)
				a.statusLine = "Turned left; Pool event dispatch remains pending."
				return nil
			}
			if a.justPressed(ebiten.KeyArrowRight) {
				a.spawn.Facing = uint8((int(a.spawn.Facing) + 1) % 4)
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
	// 0 北、1 東、2 南、3 西（spec 076）。
	dx, dy := 0, 0
	switch a.spawn.Facing {
	case 0:
		dy = -1
	case 1:
		dx = 1
	case 2:
		dy = 1
	case 3:
		dx = -1
	default:
		a.statusLine = "Face a cardinal direction before moving forward."
		return nil
	}
	if !a.initialMap.Grid.CanMoveDungeonWrapped(int(a.spawn.X), int(a.spawn.Y), a.spawn.Direction()) {
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
		if err := a.syncArchiveFromEventMachine(); err != nil {
			return result, err
		}
		if result.Exited || result.WaitingForMenu || len(result.Events) != 1 {
			return result, nil
		}
		event := result.Events[0]
		if len(event.Arguments) != len(event.ArgumentsValid) {
			return result, fmt.Errorf("Pool resource event 0x%02X has mismatched argument validity", event.Opcode)
		}
		consumed, err := a.applyTransitionResource(event)
		if err != nil {
			return result, err
		}
		if !consumed {
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

func (a *app) applyTransitionResource(event eclvm.Event) (bool, error) {
	if len(event.Arguments) != len(event.ArgumentsValid) {
		return false, fmt.Errorf("Pool resource event 0x%02X has mismatched argument validity", event.Opcode)
	}
	allValid := true
	for _, valid := range event.ArgumentsValid {
		allValid = allValid && valid
	}
	if !allValid || len(event.Arguments) != 3 {
		if event.Opcode == 0x21 || event.Opcode == 0x37 {
			return false, fmt.Errorf("Pool resource event 0x%02X has invalid arguments %v/%v", event.Opcode, event.Arguments, event.ArgumentsValid)
		}
		return false, nil
	}
	switch event.Opcode {
	case 0x21:
		if reflect.DeepEqual(event.Arguments, []uint16{0xFF, 0xFF, 0x7F}) {
			return true, nil
		}
		if event.Arguments[0] > 0xFF {
			return false, fmt.Errorf("Pool LOAD FILES has invalid arguments %v/%v", event.Arguments, event.ArgumentsValid)
		}
		key := gamepack.MapKey{Archive: a.spawn.Map.Archive, BlockID: uint8(event.Arguments[0])}
		loaded, ok := a.geometryCatalog.Map(key)
		if !ok {
			return false, fmt.Errorf("Pool LOAD FILES requested absent GEO%d block %d", key.Archive, key.BlockID)
		}
		a.initialMap = &loaded
		a.spawn.Map = key
		return true, nil
	case 0x37:
		if reflect.DeepEqual(event.Arguments, []uint16{127, 127, 127}) {
			return true, nil
		}
		selectors := [3]uint8{}
		for index, value := range event.Arguments {
			if value > 0xFF || value == 0xFF {
				return false, fmt.Errorf("Pool partial LOAD PIECES %v is not READY", event.Arguments)
			}
			selectors[index] = uint8(value)
		}
		if a.loadPieceSlots == nil {
			return false, fmt.Errorf("Pool LOAD PIECES loader is not configured")
		}
		loaded, err := a.loadPieceSlots(a.spawn.Map.Archive, selectors)
		if err != nil {
			return false, err
		}
		a.initialWalls = &loaded
		return true, nil
	default:
		return false, nil
	}
}

func (a *app) configureEventSession(session *eclvm.BlockSession) error {
	if session == nil {
		return fmt.Errorf("Pool ECL session is nil")
	}
	return session.SetBlockCatalogResolver(func(_, _ uint16, memory map[uint16]uint16) (map[uint16][]byte, error) {
		selector := memory[0x6E12]
		if selector == 0 || selector == uint16(a.eclArchive) {
			return nil, nil
		}
		if selector > 8 {
			return nil, fmt.Errorf("Pool ECL archive selector 0x%X is outside 1..8", selector)
		}
		archive, ok := a.eclCatalog.Archive(uint8(selector))
		if !ok {
			return nil, fmt.Errorf("Pool ECL archive %d is unavailable", selector)
		}
		return archive.Blocks, nil
	})
}

func (a *app) syncArchiveFromEventMachine() error {
	if a.eventMachine == nil {
		return nil
	}
	selector := a.eventMachine.Memory[0x6E12]
	if selector == 0 || selector == uint16(a.eclArchive) {
		return nil
	}
	if selector > 8 {
		return fmt.Errorf("Pool ECL archive selector 0x%X is outside 1..8", selector)
	}
	if _, ok := a.eclCatalog.Archive(uint8(selector)); !ok {
		return fmt.Errorf("Pool ECL archive %d is unavailable", selector)
	}
	a.eclArchive = uint8(selector)
	a.spawn.Map.Archive = uint8(selector)
	return nil
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
			// 商店與戰利品走同一條邊界，先分辨再分派：分不出來的話，
			// 走進商店會把整櫃存貨當成免費戰利品發下去。
			if a.isShopBoundary(result) {
				return a.enterShop(result.TreasureRequests)
			}
			return a.enterTreasure(result.TreasureRequests)
		}
		if result.CombatRequested && len(result.MonsterSpawns) != 0 {
			return a.enterCombatStaging(result.MonsterSpawns)
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
			a.finishCellBlock()
			return nil
		}
		if a.isSuneTempleBoundary(result) {
			return a.enterSuneTemple()
		}
		if event, ok := encounterEvent(result); ok {
			return a.enterEncounter(event)
		}
		if event, ok := programEvent(result); ok {
			return a.enterProgram(event)
		}
		if event, ok := damageEvent(result); ok {
			return a.applyDamageEvent(event)
		}
		if event, ok := partyQueryEvent(result); ok {
			return a.applyPartyQuery(event)
		}
		if event, ok := parlayEvent(result); ok {
			return a.enterParlay(event)
		}
		if event, ok := eclInputEvent(result); ok {
			return a.enterECLInput(event)
		}
		if event, ok := robEvent(result); ok {
			return a.applyRob(event)
		}
		if event, ok := protectionEvent(result); ok {
			return a.applyProtection(event)
		}
		if event, ok := whoEvent(result); ok {
			return a.enterWho(event)
		}
		if event, ok := addNPCEvent(result); ok {
			return a.applyAddNPC(event)
		}
		if event, ok := checkPartyEvent(result); ok {
			return a.applyCheckParty(event)
		}
		if event, ok := eclClockEvent(result); ok {
			return a.applyECLClock(event)
		}
		if event, ok := spellSearchEvent(result); ok {
			return a.applySpellSearch(event)
		}
		return a.pauseAppliedCellResult(result)
	}
	return fmt.Errorf("Pool SearchLocation exceeded presentation boundary limit")
}

func (a *app) enterCombatStaging(spawns []eclvm.MonsterSpawn) error {
	if a.loadMonster == nil {
		return fmt.Errorf("Pool monster loader is not configured")
	}
	archive := a.eclArchive
	if archive == 0 {
		archive = a.spawn.Map.Archive
	}
	staged := make([]stagedMonster, 0, len(spawns))
	labels := make([]string, 0, len(spawns))
	for _, spawn := range spawns {
		if spawn.Count == 0 {
			return fmt.Errorf("Pool monster %d has zero encounter count", spawn.MonsterID)
		}
		record, err := a.loadMonster(archive, spawn.MonsterID)
		if err != nil {
			return fmt.Errorf("load Pool monster archive %d block %d: %w", archive, spawn.MonsterID, err)
		}
		staged = append(staged, stagedMonster{Spawn: spawn, Record: record})
		labels = append(labels, fmt.Sprintf("%s ×%d", record.Name, spawn.Count))
	}
	a.combatActive = true
	a.combatMonsters = staged
	a.cellEventPending, a.cellWaitingMenu = true, false
	a.cellMenuOptions, a.cellMenuCursor = nil, 0
	a.eventText = "Encounter: " + strings.Join(labels, " / ")
	a.eventLabel = "TACTICAL COMBAT PENDING"
	a.statusLine = "Original monster records loaded; press ENTER to enter tactical combat."
	return nil
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
	pooled := [7]uint32{}
	for _, request := range requests {
		for currency, amount := range request.Amounts {
			if uint64(pooled[currency])+uint64(amount) > uint64(^uint32(0)) {
				return fmt.Errorf("Pool treasure %s overflows uint32", pooltreasure.Names[currency])
			}
			pooled[currency] += uint32(amount)
		}
		if request.ItemBlock > 0xFF {
			return fmt.Errorf("Pool treasure item block 0x%X exceeds byte range", request.ItemBlock)
		}
		if request.ItemBlock != 0 {
			items, err := a.loadTreasure(a.spawn.Map.Archive, uint8(request.ItemBlock))
			if err != nil {
				return err
			}
			loaded = append(loaded, items...)
		}
	}
	a.state.PooledMoney = pooled
	a.treasureActive, a.treasureStage = true, treasureMain
	a.treasureItems, a.treasureSelected, a.treasureCurrency, a.treasureAmount = loaded, 0, 0, ""
	a.cellEventPending, a.cellWaitingMenu = true, true
	a.enterTreasureMain()
	a.statusLine = fmt.Sprintf("Original Pool treasure service: %d item(s), seven money pools ready.", len(loaded))
	return nil
}

func (a *app) enterTreasureMain() {
	a.treasureStage = treasureMain
	a.cellMenuOptions, a.cellMenuCursor = []string{"View", "Take", "Pool", "Share", "Exit"}, 0
	a.eventText = "The party has found treasure!"
	a.eventLabel = a.cellMenuLabel()
}

func (a *app) selectTreasureOption() error {
	switch a.treasureStage {
	case treasureMain:
		switch a.cellMenuOptions[a.cellMenuCursor] {
		case "View":
			a.treasureStage = treasureView
			names := make([]string, 0, 7+len(a.treasureItems))
			for currency, amount := range a.state.PooledMoney {
				if amount != 0 {
					names = append(names, fmt.Sprintf("%s %d", pooltreasure.Names[currency], amount))
				}
			}
			for index := range a.treasureItems {
				names = append(names, a.treasureItems[index].Name)
			}
			if len(names) == 0 {
				names = append(names, "Nothing")
			}
			a.eventText = strings.Join(names, " / ")
			a.cellMenuOptions, a.cellMenuCursor = []string{"Return"}, 0
			a.eventLabel = a.cellMenuLabel()
			return nil
		case "Take":
			hasMoney := a.hasPooledMoney()
			if !hasMoney && len(a.treasureItems) == 0 {
				a.eventText = "There is no treasure left."
				return nil
			}
			if hasMoney && len(a.treasureItems) != 0 {
				a.treasureStage = treasureTake
				a.cellMenuOptions, a.cellMenuCursor = []string{"Money", "Items", "Exit"}, 0
				a.eventText = "Take what?"
				a.eventLabel = a.cellMenuLabel()
				return nil
			}
			if hasMoney {
				return a.enterTreasureMoneyCurrencies()
			}
			return a.enterTreasureItems()
		case "Pool":
			return a.applyTreasureMoneyService(pooltreasure.PoolMoney, "Party money pooled.")
		case "Share":
			return a.applyTreasureMoneyService(pooltreasure.ShareMoney, "Pooled money shared.")
		case "Exit":
			if len(a.treasureItems) == 0 && !a.hasPooledMoney() {
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
	case treasureTake:
		switch a.cellMenuOptions[a.cellMenuCursor] {
		case "Money":
			return a.enterTreasureMoneyCurrencies()
		case "Items":
			return a.enterTreasureItems()
		default:
			a.enterTreasureMain()
			return nil
		}
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
	case treasureMoneyCurrency:
		if a.cellMenuCursor == len(a.cellMenuOptions)-1 {
			a.enterTreasureMain()
			return nil
		}
		a.treasureCurrency = a.currencyForMenuIndex(a.cellMenuCursor)
		a.treasureStage = treasureMoneyCharacter
		a.cellMenuOptions = a.cellMenuOptions[:0]
		for _, member := range a.state.Party {
			a.cellMenuOptions = append(a.cellMenuOptions, member.Name)
		}
		a.cellMenuOptions = append(a.cellMenuOptions, "Cancel")
		a.cellMenuCursor = 0
		a.eventText = "Who will take " + pooltreasure.Names[a.treasureCurrency] + "?"
		a.eventLabel = a.cellMenuLabel()
		return nil
	case treasureMoneyCharacter:
		if a.cellMenuCursor == len(a.state.Party) {
			return a.enterTreasureMoneyCurrencies()
		}
		a.treasureSelected = a.cellMenuCursor
		a.treasureStage, a.treasureAmount = treasureMoneyAmount, ""
		a.cellMenuOptions = []string{"Amount"}
		a.eventText = fmt.Sprintf("How many %s? Available %d.", pooltreasure.Names[a.treasureCurrency], a.state.PooledMoney[a.treasureCurrency])
		a.eventLabel = "AMOUNT   ENTER=TAKE   ESC=CANCEL"
		return nil
	case treasureConfirmExit:
		if a.cellMenuCursor == 0 {
			return a.exitTreasure()
		}
		a.enterTreasureMain()
		return nil
	}
	return fmt.Errorf("unknown Pool treasure stage %d", a.treasureStage)
}

func (a *app) hasPooledMoney() bool {
	return a.state.PooledMoney != ([7]uint32{})
}

func (a *app) enterTreasureItems() error {
	a.treasureStage = treasureItems
	return a.rebuildTreasureItemMenu()
}

func (a *app) enterTreasureMoneyCurrencies() error {
	a.treasureStage = treasureMoneyCurrency
	a.cellMenuOptions = a.cellMenuOptions[:0]
	for currency, amount := range a.state.PooledMoney {
		if amount != 0 {
			a.cellMenuOptions = append(a.cellMenuOptions, fmt.Sprintf("%s %d", pooltreasure.Names[currency], amount))
		}
	}
	a.cellMenuOptions = append(a.cellMenuOptions, "Exit")
	a.cellMenuCursor = 0
	a.eventText = "Take: Money"
	a.eventLabel = a.cellMenuLabel()
	return nil
}

func (a *app) currencyForMenuIndex(menuIndex int) int {
	for currency, amount := range a.state.PooledMoney {
		if amount == 0 {
			continue
		}
		if menuIndex == 0 {
			return currency
		}
		menuIndex--
	}
	return -1
}

func (a *app) takeTreasureMoney() error {
	amount, err := strconv.ParseUint(a.treasureAmount, 10, 32)
	if err != nil || amount == 0 {
		a.statusLine = "Enter a positive whole-number amount."
		return nil
	}
	next := cloneSaveState(a.state)
	if err := pooltreasure.TakeMoney(&next, a.treasureSelected, a.treasureCurrency, uint32(amount)); err != nil {
		a.statusLine = err.Error()
		return nil
	}
	if a.saveState == nil {
		return fmt.Errorf("Pool save writer is not configured")
	}
	if err := a.saveState(next); err != nil {
		return err
	}
	a.state = next
	a.statusLine = fmt.Sprintf("%s takes %d %s.", a.state.Party[a.treasureSelected].Name, amount, pooltreasure.Names[a.treasureCurrency])
	return a.enterTreasureMoneyCurrencies()
}

func (a *app) applyTreasureMoneyService(service func(*poolsave.State) error, message string) error {
	next := cloneSaveState(a.state)
	if err := service(&next); err != nil {
		a.statusLine = err.Error()
		return nil
	}
	if a.saveState == nil {
		return fmt.Errorf("Pool save writer is not configured")
	}
	if err := a.saveState(next); err != nil {
		return err
	}
	a.state = next
	a.statusLine = message
	a.enterTreasureMain()
	return nil
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
	eclArchive := a.eclArchive
	if eclArchive == 0 {
		eclArchive = a.spawn.Map.Archive
	}
	next.Campaign = &poolsave.Campaign{
		MapArchive: a.spawn.Map.Archive, MapBlock: a.spawn.Map.BlockID,
		ECLArchive: eclArchive,
		X:          a.spawn.X, Y: a.spawn.Y, Facing: a.spawn.Facing, Session: snapshot,
	}
	return next, nil
}

func (a *app) restoreCampaign(loaded poolsave.State) error {
	if loaded.Campaign == nil {
		return fmt.Errorf("Pool save has no campaign")
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
	archive, ok := a.eclCatalog.Archive(campaign.ECLArchive)
	if !ok {
		return fmt.Errorf("Pool save ECL archive %d is unavailable", campaign.ECLArchive)
	}
	session, err := gamepack.NewDOSECLArchiveSession(archive, campaign.Session.Current, uint16(0x9900+campaign.Session.Machine.PC), characters...)
	if err != nil {
		return fmt.Errorf("rebuild Pool ECL session: %w", err)
	}
	if err := session.Restore(campaign.Session); err != nil {
		return fmt.Errorf("restore Pool ECL session: %w", err)
	}
	if err := a.configureEventSession(session); err != nil {
		return fmt.Errorf("configure restored Pool ECL session: %w", err)
	}
	// Commit only after every catalog and snapshot check succeeds.
	a.state = cloneSaveState(loaded)
	a.spawn = gamepack.Spawn{Map: key, X: campaign.X, Y: campaign.Y, Facing: campaign.Facing}
	a.initialMap = &geometryMap
	a.eclArchive = campaign.ECLArchive
	a.eventSession, a.eventMachine = session, session.Machine()
	a.introWaiting, a.introDone = false, true
	a.tourActive, a.tourStep, a.tourPage, a.tourDelay = false, -1, -1, 0
	a.cellEventPending, a.cellWaitingMenu = false, false
	a.templeActive, a.treasureActive, a.combatActive = false, false, false
	a.combatMonsters = nil
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
	a.statusLine = fmt.Sprintf("Temple character %d/%d: %s (HP %d/%d, %d GP; pool %d GP).", index+1, len(a.state.Party), name, a.state.Party[index].CurrentHP, a.state.Party[index].MaxHP, a.state.Party[index].Money[3], a.state.PooledMoney[3])
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

// finishCellBlock 收掉這一格的 ECL：`00h EXIT` 走完之後畫面回到移動狀態。
// `38h PROGRAM` 的值 9 也走這裡——原版在那一支的結尾就是呼叫 EXIT 的
// handler（spec 081）。
func (a *app) finishCellBlock() {
	a.cellEventPending, a.cellWaitingMenu = false, false
	a.templeActive = false
	a.cellMenuOptions, a.cellMenuCursor = nil, 0
	a.eventText, a.eventLabel = "", ""
	a.statusLine = "Moved using original GEO data; per-turn and SearchLocation returned normally."
}

func (a *app) pauseInitialCellResult(result eclvm.Result) error {
	a.applyCellECLResult(result)
	return a.pauseAppliedCellResult(result)
}

// pauseAppliedCellResult 是同一件事，但**不再套用一次**。`33h PRINT RETURN`
// 之後文字框的內容會隨套用次數改變，所以同一個 result 只能套一次；先前
// 每則訊息蓋掉上一則，套兩次看不出差別，這個重複因此一直沒被發現。
func (a *app) pauseAppliedCellResult(result eclvm.Result) error {
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
		// 選項的比對仍以原文進行（見 cellMenuOptions 的使用點），這裡只換顯示。
		option = a.gameText.Translate(option)
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
	// 文字框（spec 082）。`RunUntilEvent` 一遇到事件就返回，所以每個 result
	// 通常只帶一則——文字框的狀態因此要跨 result 留著，不能每次重建。
	//
	//   - `3Dh CLEAR BOX` 清掉整個框。
	//   - `33h PRINT RETURN` 換行。
	//   - 有文字的事件：**上一則以換行收尾就接上去，否則這是新的一頁**。
	//
	// 最後那條是刻意的：原版靠什麼在兩頁之間清框還沒讀出來（市政廳那幾個
	// block 根本沒有 `3Dh`），而逐頁取代已經對過原版（spec 015／016 的市政廳
	// 流程逐頁比對文字）。沒有 `33h` 的地方維持驗過的行為，有 `33h` 的地方
	// 那一行才真的接得起來。
	for _, event := range result.Events {
		switch event.Opcode {
		case gamepack.ClearBoxOpcode:
			a.eventText = ""
		case gamepack.PrintReturnOpcode:
			a.eventText += "\n"
		default:
			if event.Text == "" {
				continue
			}
			line := a.gameText.Translate(event.Text)
			if strings.HasSuffix(a.eventText, "\n") {
				a.eventText += line
			} else {
				a.eventText = line
			}
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
	if rolled.Gold < 0 || rolled.Gold > int(^uint16(0)) {
		return fmt.Errorf("Pool character gold %d is outside uint16", rolled.Gold)
	}
	money := [7]uint16{}
	money[3] = uint16(rolled.Gold)
	character := poolsave.Character{
		Name: a.flow.Name, RaceID: a.flow.SelectedRace().ID, GenderID: a.flow.SelectedGender().ID,
		ClassID: a.flow.SelectedClass().ID, AlignmentID: a.flow.SelectedAlignment().ID,
		Age: rolled.Age, Abilities: rolled.Abilities, ExceptionalStrength: rolled.ExceptionalStrength,
		Money: money, MaxHP: rolled.HP, CurrentHP: rolled.HP, RawHP: rolled.RawHP,
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
		drawText(screen, a.text(msgTitleHint), 264, 382, accent)
	} else if a.mode == modeMenu {
		drawFrame(screen, foreground, accent)
		drawText(screen, a.text(msgMenuTitle), 224, 54, accent)
		drawText(screen, a.text(msgMenuCreate), 176, 112, foreground)
		drawText(screen, a.text(msgMenuAdd), 176, 140, foreground)
		drawText(screen, a.text(msgMenuLoad), 176, 168, foreground)
		begin := a.text(msgMenuBegin)
		if a.programManaging {
			begin = a.text(msgProgramReturn)
		}
		drawText(screen, begin, 176, 196, foreground)
		if a.programManaging {
			drawText(screen, a.text(msgTrainCommand), 176, 214, foreground)
		}
		drawText(screen, fmt.Sprintf(a.text(msgMenuCounts), len(a.state.CharacterLibrary), len(a.state.Party)), 176, 230, accent)
		for index, member := range a.state.Party {
			// 隊伍管理時標出訓練指令要作用在誰身上。
			marker := " "
			if a.programManaging && index == a.trainParty {
				marker = ">"
			}
			drawText(screen, fmt.Sprintf("%s%d  %s", marker, index+1, member.Name),
				176, 254+index*18, foreground)
		}
		if a.statusLine != "" {
			drawText(screen, a.statusLine, 72, 350, foreground)
		}
	} else if a.mode == modeCreation {
		drawCreation(screen, a, foreground, accent)
	} else if a.tacticalPreview {
		drawTactical(screen, a, foreground, accent)
	} else {
		drawAdventure(screen, a, foreground, accent)
	}
	// 基線 386：倚天字型的 ascent 是 14，畫在 390 會被 drawFrame 的下框
	// （y 388..391）切掉字腳。
	drawText(screen, a.text(msgFooter), 16, 386, foreground)
	if a.help {
		drawHelp(screen, background, foreground, accent)
	}
	if a.journalOpen && a.journal != nil {
		drawJournal(screen, a, background, foreground, accent)
	}
	if a.equipmentOpen && a.equipment != nil {
		drawEquipment(screen, a, background, foreground, accent)
	}
	if a.shopActive && a.shop != nil {
		drawShop(screen, a, background, foreground, accent)
	}
	if a.spellsOpen && a.spells != nil {
		drawSpells(screen, a, background, foreground, accent)
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
	stageFill, err := poolFirstPersonStageFill()
	if err != nil {
		drawText(screen, "FIRST-PERSON STAGE ERROR", 72, 180, accent)
		return
	}
	drawPoolStageRects(screen, stageFill.Backdrop, viewLeft, viewTop)
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
	drawPoolStageRects(screen, stageFill.PostWall, viewLeft, viewTop)
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
		drawDialogue(screen, a.gameText.Translate(message), a.gameText.Translate(label), foreground, accent)
		dialogueVisible = true
	} else if a.tourActive && a.tourPage >= 0 && a.initialEvent != nil && a.tourStep >= 0 && a.tourStep < len(a.initialEvent.Tour) {
		step := a.initialEvent.Tour[a.tourStep]
		if a.tourPage < len(step.Messages) {
			drawDialogue(screen, a.gameText.Translate(step.Messages[a.tourPage]),
				a.gameText.Translate(a.initialEvent.ContinueLabel), foreground, accent)
			dialogueVisible = true
		}
	} else if a.cellEventPending && a.eventText != "" {
		drawDialogue(screen, a.eventText, a.gameText.Translate(a.eventLabel), foreground, accent)
		dialogueVisible = true
	}
	if a.statusLine != "" && !dialogueVisible {
		drawText(screen, a.statusLine, 42, 342, foreground)
	}
	drawCamp(screen, a, foreground, accent)
}

// drawCamp 畫紮營選單。原版的選單列是 `Rest daYs Hours Mins Inc Dec Exit`，
// 挑時間那一段還沒接，所以這裡只有三項。
func drawCamp(screen *ebiten.Image, a *app, foreground, accent color.Color) {
	if !a.campOpen {
		return
	}
	drawText(screen, a.text(msgCampTitle), 260, 120, accent)
	for index, label := range a.campOptionLabels() {
		cursor, ink := " ", foreground
		if index == a.campCursor {
			cursor, ink = ">", accent
		}
		drawText(screen, cursor+label, 244, 150+index*18, ink)
	}
	drawText(screen, a.campPendingLine(), 100, 230, foreground)
}

func poolFirstPersonStageFill() (viewport.StageInsetFill, error) {
	background := viewport.Background{SkyPalette: 1, Rects: []viewport.BackgroundRect{
		{X: 24, Y: 24, Width: 88, Height: 44, PaletteIndex: 1},
		{X: 24, Y: 68, Width: 88, Height: 0, PaletteIndex: 0},
		{X: 24, Y: 68, Width: 88, Height: 44, PaletteIndex: 8},
	}}
	return viewport.FillBackgroundToStageInset(background, viewport.StageInset{X: 24, Y: 24, Width: 88, Height: 88, WallTop: 40})
}

func drawPoolStageRects(screen *ebiten.Image, rectangles []viewport.BackgroundRect, viewLeft, viewTop int) {
	for _, rectangle := range rectangles {
		shade := graphics.EGA16[rectangle.PaletteIndex]
		target := poolStageScreenRect(rectangle, viewLeft, viewTop)
		for y := target.Min.Y; y < target.Max.Y; y++ {
			for x := target.Min.X; x < target.Max.X; x++ {
				screen.Set(x, y, shade)
			}
		}
	}
}

func poolStageScreenRect(rectangle viewport.BackgroundRect, viewLeft, viewTop int) image.Rectangle {
	left := viewLeft + (rectangle.X-24)*2
	top := viewTop + (rectangle.Y-24)*2
	return image.Rect(left, top, left+rectangle.Width*2, top+rectangle.Height*2)
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
	for index, line := range wrapDisplay(message, 68) {
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
	view, err := viewport.TraverseWallViewWrapped(grid, uint8(spawn.Direction()), int(spawn.X), int(spawn.Y))
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
		drawText(screen, a.text(msgCharacterSheet), 230, 42, accent)
		if a.rolled == nil {
			drawText(screen, a.text(msgRolling), 40, 82, foreground)
			return
		}
		value := a.rolled
		gender := a.optionText(a.flow.SelectedGender().ID, a.flow.SelectedGender().Label)
		race := a.optionText(a.flow.SelectedRace().ID, a.flow.SelectedRace().Label)
		class := a.optionText(a.flow.SelectedClass().ID, a.flow.SelectedClass().Label)
		drawText(screen, fmt.Sprintf("%s  %s  %s", gender, race, class), 48, 82, foreground)
		drawText(screen, fmt.Sprintf(a.text(msgAge), value.Age), 48, 110, foreground)
		for index := range value.Abilities {
			extra := ""
			if index == 0 && value.ExceptionalStrength != 0 {
				extra = fmt.Sprintf("/%02d", value.ExceptionalStrength)
			}
			drawText(screen, fmt.Sprintf("%s %2d%s", a.abilityName(index), value.Abilities[index], extra),
				48+(index/3)*180, 150+(index%3)*28, foreground)
		}
		drawText(screen, fmt.Sprintf(a.text(msgGoldAndHP), value.Gold, value.HP, value.HP), 48, 252, foreground)
		drawText(screen, a.text(msgKeepCharacter), 48, 302, accent)
		if a.statusLine != "" {
			drawText(screen, a.statusLine, 48, 334, foreground)
		}
		return
	}
	if a.flow.Stage == creation.StageName {
		drawText(screen, a.text(msgCharacterName), 160, 128, accent)
		drawText(screen, a.nameInput+"_", 160, 164, foreground)
		drawText(screen, a.text(msgNameRule), 160, 214, foreground)
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
		drawText(screen, a.hint("icon"), 48, 332, foreground)
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
	title := map[creation.Stage]messageID{
		creation.StageRace: msgStageRace, creation.StageGender: msgStageGender,
		creation.StageClass: msgStageClass, creation.StageAlignment: msgStageAlignment,
	}[a.flow.Stage]
	drawText(screen, a.text(title), 250, 42, accent)
	ids := a.flow.OptionIDs()
	for index, option := range a.flow.Options() {
		prefix := "  "
		ink := foreground
		if index == a.cursor {
			prefix, ink = "> ", accent
		}
		if index < len(ids) {
			option = a.optionText(ids[index], option)
		}
		drawText(screen, prefix+option, 128, 82+index*22, ink)
	}
	drawText(screen, a.hint(stageName(a.flow.Stage)), 32, 346, foreground)
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
		"J: open the adventurer's journal (-lang zh)",
		"I: ready or unready a party member's items",
		"K: browse the original spell list",
	}
	for index, line := range lines {
		drawText(screen, line, 104, 120+index*30, foreground)
	}
}

func drawText(screen *ebiten.Image, value string, x, y int, ink color.Color) {
	text.Draw(screen, strings.ToUpper(displayText(value)), uiFace, x, y, ink)
}

func (a *app) Layout(_, _ int) (int, int) { return logicalWidth, logicalHeight }

func main() {
	zipPath := flag.String("zip", "Pool of Radiance (1988).zip", "DOS source ZIP used as local asset source")
	langFlag := flag.String("lang", "auto", "UI language: en, zh, or auto (zh when an ETen font is supplied)")
	etenFont := flag.String("eten-font", "", "ETen STDFONT.15 path; the 16x15 Han glyphs the Chinese UI needs")
	etenSymbol := flag.String("eten-symbol-font", "", "optional ETen SPCFONT.15 path for full-width punctuation")
	etenASCII := flag.String("eten-ascii-font", "", "optional ETen ASCFONT.15 path; defaults to ascfont.15 beside -eten-font")
	savePath := flag.String("save", defaultStatePath(), "remake save file; defaults to the OS user config directory")
	flag.Parse()
	uiLanguage, face, err := resolveUILanguage(*langFlag, *etenFont, *etenSymbol, *etenASCII)
	if err != nil {
		log.Fatal(err)
	}
	uiFace = face
	catalogue, err := gameTextFor(uiLanguage)
	if err != nil {
		log.Fatal(err)
	}
	game, err := newApp(*zipPath, *savePath)
	if err != nil {
		log.Fatal(err)
	}
	game.language, game.gameText = uiLanguage, catalogue
	ebiten.SetWindowSize(960, 600)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowTitle("Pool of Radiance Remake")
	if err := ebiten.RunGame(game); err != nil && err != ebiten.Termination {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
