// pool-game 是 remake 的遊戲本體：Ebiten 視窗、玩家輸入、畫面，以及與共用
// engine 和 game pack 的接線。玩法規則本身不寫在這裡，寫在 internal/ 與
// JSON game pack 裡。
//
// 這支就是 spec 005 說的那一支 remake executable。
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
	"golang.org/x/image/font"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/assets"
	poolcharacter "github.com/wicanr2/Pool-of-Radiance-cht/internal/character"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/creation"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/etenfont"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gametext"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/guide"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/music"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/temple"
	pooltreasure "github.com/wicanr2/Pool-of-Radiance-cht/internal/treasure"
	"github.com/wicanr2/golden-box-remake-engine/ecl"
	"github.com/wicanr2/golden-box-remake-engine/eclvm"
	"github.com/wicanr2/golden-box-remake-engine/geometry"
	"github.com/wicanr2/golden-box-remake-engine/graphics"
	"github.com/wicanr2/golden-box-remake-engine/viewport"
)

const (
	logicalWidth  = 640
	logicalHeight = 400
	// campSpeedDelayMilliseconds 是原版一拍的長度：`Delay(GameSpeed × 225)`
	//（overlay-37 entry 13 `0C83h` 把 `ds:4943h` 乘上 0E1h 再交給 resident 的
	// 延遲常式）。ECL 分派器、戰鬥、法術效果、休息都呼叫同一支，所以那就是
	// 遊戲裡「等一下」的統一單位（spec 135）。
	//
	// 這一條取代了原本寫死的九個影格——那時原版的 DELAY 還沒讀出來，
	// 註解自承是 approximation。現在讀出來了，就照它算。
	campSpeedDelayMilliseconds = 225
	// campSpeedDefault 是遊戲速度的初始值，原版在 overlay-11 `03BEh`
	// 寫 `mov byte ptr ds:4943h, 4`。
	campSpeedDefault = 4
)

// speedDelayTicks 是目前速度下「等一拍」有幾個 60 Hz 影格。
// 速度 0（最快）就是不等。
func (a *app) speedDelayTicks() int {
	return int(a.gameSpeed) * campSpeedDelayMilliseconds * 60 / 1000
}

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
	// A）ppraise 的兩層：先挑寶石或珠寶，再對估好價的那一件選 Sell／Keep。
	templeAppraise
	templeAppraiseOffer
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
	mode            screenMode
	title           *ebiten.Image
	flow            creation.Flow
	cursor          int
	rolled          *creation.RolledCharacter
	roller          creation.Roller
	help            bool
	tacticalPreview bool
	language        language
	gameText        *gametext.Catalogue
	// monsterText 是 `MONnCHA` 名字的譯名表。戰鬥與遭遇畫面顯示的是
	// 玩家看得到的字，沒有這一份就會在中文畫面上冒出 `SPECTRE ×2`。
	monsterText     *gametext.MonsterCatalogue
	tactical        *tacticalState
	journal         *journalState
	itemTypes       *gamepack.ItemTypeTable
	spellParameters []gamepack.SpellParameters
	encounter       *encounterState
	spells          *spellState
	spellsOpen      bool
	shop            *shopState
	shopActive      bool
	equipment       *equipmentState
	equipmentOpen   bool
	journalOpen     bool
	// 遊戲內攻略（`F3`，guide.go）。guideFull 是攤開全圖的劇透模式，
	// guideSpoilerWarned 讓第一次按 `V` 只出警告不攤開。
	guide              *guide.Catalogue
	guideOpen          bool
	guideFull          bool
	guideSpoilerWarned bool
	guideExplored      map[guideCellKey]bool
	modern          bool
	statusLine      string
	// screenStatePath 是 `-screen-state` 指定的檔案；自動截圖用它等畫面，
	// 不用猜時間（screen_state.go）。空字串代表不寫。
	screenStatePath string
	screenStateLast string
	keys            keySource
	nameInput       string
	portrait        *ebiten.Image
	iconReady       *ebiten.Image
	iconAction      *ebiten.Image
	// iconReadyOld／iconActionOld 是**進編輯器那一刻**的造形。原版的編輯器
	// 同時畫 OLD 與 NEW 兩組 `READY`／`ACTION`（基準畫面 `25-k`），改了什麼
	// 一眼就比得出來；沒有 OLD 那一組就只剩「現在長這樣」，看不出改了什麼。
	iconReadyOld    *ebiten.Image
	iconActionOld   *ebiten.Image
	loadPortrait    func(head, body uint8) (*ebiten.Image, error)
	loadIcon        func(head, body, size uint8, action bool, colors [6][2]uint8) (*ebiten.Image, error)
	loadCombatTiles func(name string) ([]*ebiten.Image, error)
	state           poolsave.State
	// iconMenu 是 combat icon editor 的巢狀選單狀態（spec 003 第 7..10 步）。
	iconMenu        iconMenuState
	// reloadTitle 換主題時把標題圖用新色盤重畫。
	reloadTitle     func() error
	saveState       func(poolsave.State) error
	loadState       func() (poolsave.State, error)
	// exportDOSCharacter 在建角完成時寫出原版格式的三個檔（spec 003 第 11 步）。
	// 測試把它換掉就不用碰檔案系統。
	exportDOSCharacter func(poolsave.Character) error
	initialMap         *gamepack.GeometryMap
	geometryCatalog    gamepack.GeometryCatalog
	eclCatalog         gamepack.ECLCatalog
	eclArchive         uint8
	initialWalls       *graphics.PieceSet
	loadPieceSlots     func(archive uint8, selectors [3]uint8) (graphics.PieceSet, error)
	initialEvent       *gamepack.InitialEvent
	spawn              gamepack.Spawn
	introWaiting       bool
	introDone          bool
	tourActive         bool
	tourStep           int
	tourPage           int
	tourDelay          int
	eventMachine       *eclvm.Machine
	eventSession       *eclvm.BlockSession
	eventText          string
	eventLabel         string
	cellEventPending   bool
	cellWaitingMenu    bool
	// cellMovedByScript 記「這一步是腳本自己用 `CALL C01Eh` 走掉的」。
	cellMovedByScript bool
	cellMenuOptions   []string
	// door 是目前擋在前面的那一道鎖住的門（spec 122）。**不走
	// cellMenuOptions**：那一條的選擇要餵回 ECL VM，這一個不用。
	door *doorMenu
	// characterBinding 是 ECL 的 active-character 視窗與隊伍之間的來回。
	characterBinding *gamepack.CharacterBinding
	cellMenuCursor   int
	templeActive     bool
	templeStage      templeStage
	appraiseKind     appraiseKind
	appraiseValue    int
	templeParty      int
	templeService    int
	// musicPlayer 是可選的配樂輸出（spec 128）。nil 代表沒有音訊資產——
	// 可散布的發行包本來就不帶，所有方法對 nil 安全。
	musicPlayer *music.Player
	// stingPrompt／stingBuffer／stingUnlocked 是原版的除錯碼（`J` 再輸入
	// `STING`，overlay-16 `049Ah`），見 training_gate.go。
	stingPrompt   bool
	stingBuffer   string
	stingUnlocked bool
	// programManaging 為真時，隊伍管理畫面是 `38h PROGRAM` 從地圖上開的，
	// 離開時要回地圖並讓 ECL 繼續，不是重新開始冒險。
	programManaging bool
	// campFromProgram 為真時，紮營畫面是 `38h PROGRAM` 的值 9 從旅店開的
	// （spec 081）：收掉畫面之後這個 ECL block 就結束。
	campFromProgram bool
	// programExitsBlock 為真時，關掉隊伍管理之後這個 ECL block 就結束，
	// 不是從原地繼續——值 9 的結尾是呼叫 EXIT 的 handler。
	programExitsBlock bool
	// savingThrows 是 DS:41E6h 那張表（spec 075），2Eh DAMAGE 擲豁免要用。
	savingThrows *gamepack.SavingThrowTable
	// menuMember 是人物管理選擇項上選中的成員；D、M、T、V、R 五個指令
	// 都對他生效。1..6 選人（說明書 p.8 起的那幾項都要先指定對象）。
	menuMember int
	// menuDropPending 是 D）ROP 的再確認畫面。
	menuDropPending bool
	// cellWaitedOnce／cellTextSticky 是「這一格的事件等過一次了」與
	// 「腳本結束了但文字還留在框裡」（見 `pauseAppliedCellResult`）。
	cellWaitedOnce bool
	cellTextSticky bool
	// spellMember 是法術畫面上選中的成員，記憶指令對他生效。
	spellMember int
	// 紮營（原版 overlay-15 的畫面 ＋ overlay-20 的時間，spec 135）。
	// 那不是彈出選單，是把視野換成營火、指令列換成另一列。
	campOpen  bool
	campStage campStage
	// campMember 是紮營的「目前角色」（原版 `ds:5CF0h`），數字鍵 1..6 換他，
	// `VIEW`／`DROP`／`ICON` 都對他生效。
	campMember int
	// campOrderPick 是 `Party Order` 選好、還沒放下的那一個；-1 是還沒選。
	campOrderPick int
	// campFlowBackup 是 `ALTER → ICON` 借用 `flow` 之前的樣子。
	campFlowBackup creation.Flow
	// gameSpeed 是 `ALTER → SPEED` 的值（原版 `ds:4943h`，0 最快 9 最慢）。
	// **remake 目前沒有逐字顯示，所以這個值還沒有作用**（spec 135 的 OPEN）。
	gameSpeed uint8
	// monsterPicsHidden／portraitsHidden 是 `ALTER → PICS` 的兩個開關
	//（原版 `ds:4957h`／`ds:4956h`）。**存的是「關掉了沒有」**，因為兩個
	// 開關預設都是開的，而零值要對應預設——存成 `showX` 的話每一份測試用的
	// `app{}` 都會變成「圖片關掉」，而那不是任何人選的。
	monsterPicsHidden bool
	portraitsHidden   bool
	// campMessage 是對話框那一行（`The party makes camp...`）。
	campMessage string
	// campFire 是營火動畫的兩張（`PIC<區號>.DAX` 區塊 29）。
	campFire     []*ebiten.Image
	campFireTick int
	loadCampFire func(archive uint8) ([]*ebiten.Image, error)
	// 紮營要玩家挑的休息時間（spec 114）。原版的欄位是天／時／分，
	// 分鐘一次五分；`restField` 是目前選中的那一欄。
	restDuration gamepack.RestDuration
	// timeRadix 是逐位的進位上限（`DS:35D4h`）；gameTime 是遊戲時鐘本身，
	// 狀態列的 `HH:MM` 就是它（spec 118）。
	timeRadix gamepack.TimeRadix
	gameTime  gamepack.GameTime
	restField    gamepack.RestField
	// 戰鬥中的施法清單（spec 098）。
	castOpen    bool
	castOptions []castOption
	castCursor  int
	// 選目標那一步（原版 overlay-13 的 `Next Prev Manual`）。
	castTargeting       bool
	castTargetingAttack bool
	castTargets         []uint8
	castTargetCursor    int
	// castManual 與 castManualX／Y 是瞄準時的 Manual 格子游標（spec 127）。
	castManual           bool
	castManualX, castManualY int
	castPending         castOption
	// levelUpTables 是生命骰、體質加成與職業分類遮罩（spec 097），訓練要用。
	levelUpTables gamepack.LevelUpTables
	// thiefSkillTables 是賊技能的基礎、種族與敏捷三張表（spec 095），建角要用。
	thiefSkillTables gamepack.ThiefSkillTables
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
	// 結局過場（spec 108）：`38h PROGRAM` 的值 8 進來，一頁一頁按 ENTER。
	endingScript     gamepack.EndingScript
	// `F4` 的素材總覽（sprite_overview.go）。
	spriteOpen      bool
	spritePortraits []*ebiten.Image
	spriteIcons     []*ebiten.Image
	spriteMonsters  [][2]*ebiten.Image
	spriteEffects   [][2]*ebiten.Image
	// journalCue 是文字框現在引用的手冊條目，按繼續那一下就翻過去；
	// journalCueDone 記這一格已經翻過哪幾則，免得讀完回來又彈一次。
	journalCues    []journalCue
	journalCueDone map[journalCue]bool
	// boardIcons 是戰場上每一種造形載好的圖，載一次就留著。
	boardIcons map[boardIcon]*ebiten.Image
	// combatTiles 是戰場的地形圖塊，一組（DUNGCOM／WILDCOM／RANDCOM）載一次。
	combatTiles map[string][]*ebiten.Image
	spritePage      int
	// loadMonsterSprite 讀戰場上的怪物圖形（`COMSPR.DAX`）。
	loadMonsterSprite func(block uint8, action bool) (*ebiten.Image, error)
	// 探索畫面的 `V)IEW`（spec 119，view_sheet.go）。
	viewSheetOpen     bool
	viewSheetShown    bool
	viewSheetCursor   int
	viewSheetPortrait *ebiten.Image
	// 探索畫面的施法（spec 119 的 `C)AST`，field_cast.go）。
	fieldCastOpen    bool
	fieldCastStage   int
	fieldCastCursor  int
	fieldCastCaster  int
	fieldCastSpell   int
	fieldCastOptions []castOption
	fieldCastMessage string
	// combatCommands 是戰鬥指令列那六段原版字串（spec 129）。哪幾段接上去
	// 由 `combatCommandBar` 依角色算。
	combatCommands []gamepack.CombatCommandSegment
	// continuePrompt 是「按鍵繼續」那一列的原文（overlay-03 `118Ah`）。
	// 選項數是 1 的選單，原版畫的是它，不是腳本給的那一條（spec 082）。
	continuePrompt string
	// eclSessionArchive 是 ECL session 目前握著哪一份 archive 的區塊。
	// 與 `eclArchive` 分開：後者是地圖與素材命名用的鏡像，會先一步更新。
	eclSessionArchive uint8
	// searchFlags 是 `[4937h]+594h`：第 0 位邊走邊搜、第 1 位這一次要搜
	// （spec 119）。areaMapOpen 是 `DS:6A0Ah` 的平面圖開關。
	searchFlags uint16
	areaMapOpen bool
	// symbolBand0 是第 0 帶（`01h..2Dh`）的全域 8×8 符號集（spec 120）。
	symbolBand0 graphics.Picture
	// symbolBand4 是第 4 帶（`100h..11Eh`）。畫面外框那圈繩索就在裡面
	// （`114h`／`115h`／`116h`，spec 123）。
	symbolBand4 graphics.Picture
	endingActive      bool
	endingPages      [][]gamepack.EndingLine
	endingPage       int
	// endingScene 是疊好的結局畫面（spec 108）；隊伍人數決定疊幾層。
	endingScene *ebiten.Image
	// loadEndingScene 由 newApp 注入，測試不必碰檔案系統。
	loadEndingScene func(partySize int) (*ebiten.Image, error)
	// loadNPCPortrait 疊 APPROACH 的 NPC 半身像（spec 117）；同樣由 newApp 注入。
	loadNPCPortrait func(archive, head, body uint8) (*ebiten.Image, error)
	// npcPortrait 是現在要蓋在第一人稱框上的半身像；沒有就畫視野。
	npcPortrait *ebiten.Image
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
	application := &app{
		mode:   modeTitle,
		flow:   creation.NewFlow(),
		roller: diceRoller{random: rand.New(rand.NewSource(time.Now().UnixNano()))},
		keys:   ebitenKeys{},
		state:  poolsave.NewState(),
		campOrderPick: -1,
		gameSpeed:     campSpeedDefault,
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
	application.eclSessionArchive = 3
	initialWalls, err := gamepack.ReadDOSPieceSet(zipPath, 3, 1, 0)
	if err != nil {
		return nil, fmt.Errorf("load DOS initial wall set: %w", err)
	}
	band0, band4, err := gamepack.ReadDOSGlobalSymbolBands(zipPath)
	if err != nil {
		return nil, fmt.Errorf("load DOS global 8x8 symbol bands: %w", err)
	}
	application.symbolBand0 = band0
	application.symbolBand4 = band4
	application.initialWalls = &initialWalls
	application.loadPieceSlots = func(archive uint8, selectors [3]uint8) (graphics.PieceSet, error) {
		previous := graphics.PieceSet{}
		if application.initialWalls != nil {
			previous = *application.initialWalls
		}
		return gamepack.ReadDOSPieceSlots(zipPath, archive, selectors, previous)
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
	thiefSkillTables, err := gamepack.ReadDOSThiefSkillTables(zipPath)
	if err != nil {
		return nil, err
	}
	application.thiefSkillTables = thiefSkillTables
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
	// 戰鬥指令列的六段。讀不到就讓它是空的——那一列不畫，不要拿寫死的
	// 八項頂替：頂替出來的畫面看起來是對的，而玩家按下去沒有反應。
	if segments, err := gamepack.ReadDOSCombatCommands(zipPath); err == nil {
		application.combatCommands = segments
	}
	// 「按鍵繼續」那一列。讀不到就留空，顯示端退回腳本給的那一條——
	// 寫法會與原版不同，但玩家至少看得到提示。
	if prompt, err := gamepack.DOSContinuePrompt(zipPath, gamepack.ContinuePromptKeyboard); err == nil {
		application.continuePrompt = prompt
	}
	endingScript, err := gamepack.ReadDOSEndingScript(zipPath)
	if err != nil {
		return nil, err
	}
	application.endingScript = endingScript
	spellParameters, err := gamepack.ReadDOSSpellParameters(zipPath)
	if err != nil {
		return nil, err
	}
	application.spellParameters = spellParameters
	timeRadix, err := gamepack.ReadDOSTimeRadix(zipPath)
	if err != nil {
		return nil, err
	}
	application.restDuration = gamepack.NewRestDuration(timeRadix)
	application.timeRadix = timeRadix
	application.restField = gamepack.RestFieldMinutes
	application.saveState = func(state poolsave.State) error { return poolsave.WriteAtomic(statePath, state) }
	application.exportDOSCharacter = func(member poolsave.Character) error {
		files, err := application.buildDOSCharacterFiles(member)
		if err != nil {
			return err
		}
		return writeDOSCharacterFiles(filepath.Join(filepath.Dir(statePath), dosExportDir), member.Name, files)
	}
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
		rendered, err := composed.RGBA(0, application.artPalette())
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
		rendered, err := picture.RGBA(0, application.artPalette())
		if err != nil {
			return nil, err
		}
		return ebiten.NewImageFromImage(rendered), nil
	}
	application.loadCombatTiles = func(name string) ([]*ebiten.Image, error) {
		picture, err := assets.ReadCombatTerrainTiles(zipPath, name)
		if err != nil {
			return nil, err
		}
		tiles := make([]*ebiten.Image, 0, int(picture.ItemCount))
		for item := 0; item < int(picture.ItemCount); item++ {
			rendered, err := picture.RGBA(item, application.artPalette())
			if err != nil {
				return nil, err
			}
			tiles = append(tiles, ebiten.NewImageFromImage(rendered))
		}
		return tiles, nil
	}
	application.loadMonsterSprite = func(block uint8, action bool) (*ebiten.Image, error) {
		picture, err := assets.ReadMonsterSprite(zipPath, block, action)
		if err != nil {
			return nil, err
		}
		rendered, err := picture.RGBA(0, application.artPalette())
		if err != nil {
			return nil, err
		}
		return ebiten.NewImageFromImage(rendered), nil
	}
	application.loadEndingScene = func(partySize int) (*ebiten.Image, error) {
		pictures, err := assets.ReadEndingPictures(zipPath, gamepack.EndingLayerBlocks())
		if err != nil {
			return nil, err
		}
		scene, err := assets.ComposeEndingScene(pictures, gamepack.VisibleEndingLayers(partySize))
		if err != nil {
			return nil, err
		}
		rendered, err := scene.RGBA(0, application.artPalette())
		if err != nil {
			return nil, err
		}
		return ebiten.NewImageFromImage(rendered), nil
	}
	application.loadCampFire = func(archive uint8) ([]*ebiten.Image, error) {
		frames, err := assets.ReadCampFire(zipPath, archive)
		if err != nil {
			return nil, err
		}
		result := make([]*ebiten.Image, 0, len(frames))
		for _, frame := range frames {
			rendered, err := frame.RGBA(0, application.artPalette())
			if err != nil {
				return nil, err
			}
			result = append(result, ebiten.NewImageFromImage(rendered))
		}
		return result, nil
	}
	application.loadNPCPortrait = func(archive, head, body uint8) (*ebiten.Image, error) {
		picture, err := assets.ReadNPCPortrait(zipPath, archive, head, body)
		if err != nil {
			return nil, err
		}
		rendered, err := picture.RGBA(0, application.artPalette())
		if err != nil {
			return nil, err
		}
		return ebiten.NewImageFromImage(rendered), nil
	}
	application.reloadTitle = func() error {
		rendered, err := pictures[1].RGBA(0, application.artPalette())
		if err != nil {
			return err
		}
		application.title = ebiten.NewImageFromImage(rendered)
		return nil
	}
	// 標題圖走與換主題同一條路，開場與 F2 之後才不會是兩套算法。
	if err := application.reloadTitle(); err != nil {
		return nil, err
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

// rememberOldIcons 把現在這一組記成 OLD。**要在 reloadIcons 之後叫**：
// 進編輯器那一步會先把 flow 的造形欄位設好再重載，那個結果才是「進來時的
// 樣子」。編輯途中不會再叫，所以 OLD 整段編輯期間都不動。
func (a *app) rememberOldIcons() {
	a.iconReadyOld, a.iconActionOld = a.iconReady, a.iconAction
}

func (a *app) Update() error {
	// 這一格的畫面識別字（screen_state.go）。放在最前面：上一格處理完的
	// 結果就是這一格玩家看到的東西，而腳本要等的正是那個。
	a.publishScreenName()
	// 配樂跟著畫面狀態走（spec 128）。沒有音訊資產時 musicPlayer 是 nil，
	// 這一行什麼都不做。
	a.updateMusic()
	if a.justPressed(ebiten.KeyF10) {
		if a.saveState != nil {
			state, err := a.stateForSave()
			if err != nil {
				// 存不了就說為什麼，不要把視窗收掉：`Update` 回傳非
				// Termination 的 error 時 ebiten 會直接結束，玩家在對話
				// 中按一下 F10 遊戲就沒了。
				a.statusLine = err.Error()
				return nil
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
		if err := a.switchTheme(); err != nil {
			a.statusLine = err.Error()
		}
	}
	// 攻略頁開著的時候由它先吃鍵，`F3` 與 ESC 才關得掉。
	if handled, err := a.guideInput(); handled {
		return err
	}
	// 探索施法那一頁同理。
	if handled, err := a.fieldCastInput(); handled {
		return err
	}
	if handled, err := a.viewSheetInput(); handled {
		return err
	}
	if handled, err := a.spriteOverviewInput(); handled {
		return err
	}
	if a.justPressed(ebiten.KeyF4) && a.mode == modeAdventure && a.introDone {
		a.openSpriteOverview()
		return nil
	}
	if a.justPressed(ebiten.KeyF3) && a.mode == modeAdventure && a.introDone {
		// 站著的那一格當然走過了。只在「移動之後」記的話，剛進城還沒走
		// 就開攻略會看到一張全空的圖，看起來像攻略沒資料。
		a.rememberGuideCell()
		a.guideOpen, a.guideFull = true, false
		a.help = false
		return nil
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
		// 營火是動畫，由這裡推著走——原版是選單元件的等鍵迴圈在推
		//（overlay-26 `02A0h`）。
		a.campFireTick++
		if err := a.campInput(); err != nil {
			return err
		}
		// 旅店開的那一次收掉畫面就結束 block。「記憶法術」會把紮營關掉再開
		// 法術一覽，那還在同一段裡，所以要等法術一覽也關掉。
		if a.campFromProgram && !a.campOpen && !a.spellsOpen {
			a.campFromProgram = false
			return a.closeProgramCamp()
		}
		return nil
	}
	if a.campFromProgram && !a.campOpen && !a.spellsOpen {
		a.campFromProgram = false
		return a.closeProgramCamp()
	}
	if handled, err := a.adventureCommandInput(); handled {
		return err
	}
	if a.mode == modeAdventure && !a.help && a.tactical == nil && a.justPressed(ebiten.KeyE) {
		a.openCamp()
		return nil
	}
	// 這三個地圖上的快捷鍵在戰術地圖上是**移動鍵**：K 是方向 6、I 是方向 1
	// （spec 053 的 H I M Q P O K G）。少了 `a.tactical == nil`，隊伍往那兩個
	// 方向走會變成開法術書或開裝備，那一場架就再也結束不了——實測索寇要塞的
	// 遭遇會把整趟探索的預算吃光。E（紮營）本來就擋著，這裡照它接。
	if a.mode == modeAdventure && !a.help && a.tactical == nil && a.justPressed(ebiten.KeyK) {
		if err := a.openSpells(); err != nil {
			a.statusLine = err.Error()
		}
		return nil
	}
	if a.mode == modeAdventure && !a.help && a.tactical == nil && a.justPressed(ebiten.KeyI) {
		a.openEquipment()
		return nil
	}
	// 手冊統一由 `J` 開，**ENTER 一下都不碰它**。
	//
	// 原本是「文字報了手冊編號，按繼續那一下就翻過去」——而那一下 ENTER 同時
	// 是門選單的確定鍵、格子事件的繼續鍵。一個鍵三種意思，玩家按下去會發生
	// 什麼取決於看不見的狀態；鎖住的門就是這樣變成按了沒反應（spec 122／132）。
	//
	// 文字框裡的提示跟著改寫成 `J`（`journalCuePrompt`），所以玩家看得到按哪一個。
	// 門選單開著時不開手冊：門是 modal 的，那時候每一個鍵都歸它。
	if a.mode == modeAdventure && !a.help && a.tactical == nil && a.door == nil &&
		a.justPressed(ebiten.KeyJ) {
		// 有引用就直接翻到那一則，沒有才開在上次停的地方。
		if cue, ok := a.takeJournalCue(); ok {
			return a.openJournalAt(cue)
		}
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
		// `J` 的除錯碼在原版就是掛在這個畫面上，兩種進法都吃得到。
		if a.stingInput() {
			return nil
		}
		if a.justPressed(ebiten.KeyJ) {
			a.stingPrompt, a.stingBuffer = true, ""
			return nil
		}
		if a.programManaging {
			// 1-6 挑人、T 訓練（spec 097）。原版把訓練掛在這個畫面的 `T` 上。
			for index, key := range []ebiten.Key{ebiten.KeyDigit1, ebiten.KeyDigit2,
				ebiten.KeyDigit3, ebiten.KeyDigit4, ebiten.KeyDigit5, ebiten.KeyDigit6} {
				if index < len(a.state.Party) && a.justPressed(key) {
					a.menuMember = index
					a.statusLine = strings.TrimSpace(a.state.Party[index].Name)
					return nil
				}
			}
			if a.justPressed(ebiten.KeyT) {
				if len(a.state.Party) == 0 {
					a.statusLine = a.text(msgTrainNeedsMember)
					return nil
				}
				if a.menuMember >= len(a.state.Party) {
					a.menuMember = 0
				}
				line, err := a.trainMember(a.menuMember)
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
		if a.menuDropPending {
			return a.partyMenuDropConfirm()
		}
		// 1..6 指定 D、M、T、V、R 要對誰生效。
		for index, key := range []ebiten.Key{ebiten.KeyDigit1, ebiten.KeyDigit2,
			ebiten.KeyDigit3, ebiten.KeyDigit4, ebiten.KeyDigit5, ebiten.KeyDigit6} {
			if index < len(a.state.Party) && a.justPressed(key) {
				a.menuMember = index
				a.statusLine = strings.TrimSpace(a.state.Party[index].Name)
				return nil
			}
		}
		if a.programManaging {
			// `38h PROGRAM` 開的隊伍管理：B 或 ESC 回地圖，ECL 從原地繼續。
			// 這裡不能走「開始冒險」——那會把開場整個重跑一次。
			if a.justPressed(ebiten.KeyB) || a.justPressed(ebiten.KeyEscape) {
				if len(a.state.Party) == 0 {
					a.statusLine = a.text(msgProgramNeedsParty)
					return nil
				}
				return a.closePartyManagement()
			}
		}
		return a.partyMenuCommand()
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
				// **導覽走完不留字。** 原版走完就是自由移動，最下面換成指令列。
				// 這裡本來留著一句開發用的英文狀態（「movement policy remains
				// pending」），那是給自己看的，卻是玩家導覽結束後第一眼看到的東西。
				a.statusLine = ""
				return nil
			}
			step := a.initialEvent.Tour[a.tourStep]
			a.spawn = step.Position
			a.tourDelay = a.speedDelayTicks()
			if len(step.Messages) != 0 {
				a.tourPage = 0
			}
		}
		if a.introDone {
			if a.cellEventPending {
				if a.endingActive {
					if a.justPressed(ebiten.KeyEnter) || a.justPressed(ebiten.KeySpace) {
						return a.advanceEnding()
					}
					return nil
				}
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
					// **這一下 ENTER 只用來回答眼前的問題**：遭遇、神殿、交涉、
					// `WHO`、格子選單、或是把事件按過去。翻手冊改由 `J` 負責
					// （上面那一段），所以這裡不必再排除它們。
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
			// 撞到鎖住的門之後，那個選單要吃得到按鍵（spec 122）。
			//
			// **這一段的位置就是修正本身。** 原本它在上面 `cellEventPending`
			// 那個區塊**裡面**，而撞門的時候 `cellEventPending` 是 false
			// ——選單畫得出來、狀態列也印著 `Locked. BASH EXIT`，但方向鍵與
			// ENTER 全都走到下面的轉向與前進去了，門根本開不了。既有的門測試
			// 都直接呼叫 `resolveDoorMenu`，繞過輸入層，所以那一排綠燈證明的
			// 是規則對，不是玩家按得動。
			//
			// 排在 `cellEventPending` **後面**也是修正的一部分：`a.door` 是
			// 持久狀態（同一道門再撞一次要接續 BASH／PICK 額度），撞不開的時候
			// 它留著。放在前面的話，門還在而腳本排出遭遇時，`[COMBAT WAIT FLEE
			// PARLAY]` 的方向鍵會被門吃掉、ENTER 又跑回去撞門，兩層選單一起卡死。
			if a.door != nil {
				return a.doorMenuInput(
					a.justPressed(ebiten.KeyArrowLeft) || a.justPressed(ebiten.KeyArrowUp),
					a.justPressed(ebiten.KeyArrowRight) || a.justPressed(ebiten.KeyArrowDown),
					a.justPressed(ebiten.KeyEnter) || a.justPressed(ebiten.KeySpace))
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

// moveInitialDungeonForward 往面對的方向走一格。
//
// 三份規格在這一條路上交會：座標的 16×16 繞回是 spec 012（原版 overlay-07
// entry 27 的 cardinal wrapper，remake 這一側是 `geometry.WrapCoordinate`）、
// 「走得過去嗎」是 spec 014 的 GEO walk（牆與門的細節由共用 engine 的
// `CanMoveDungeonWrapped` 判），走完之後跑哪一個 ECL 入口是 spec 022
// （入口 0 每格、入口 1 搜尋，順序由 `DS:4944`..`494C` 那五個 header 決定）。
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
	// 野外的一步是**兩件事一起發生**：引擎照一般規則在載入的 GEO 上走一格，
	// 而那一區的 ECL 入口 0 同時把野外座標 `49C3`／`49C4` 往前推一格
	// （spec 105）。前端要做的只是「先把方向交給 ECL、事後把算出來的野外
	// 座標收回去」，不是換一條移動路徑——野外那三個區塊都有自己的 GEO
	// （進去時 `LOAD FILES 6, 6, 0`），也照樣用 `CALL @C018` 重畫牆。
	wilderness := a.inWildernessOverland()
	if wilderness {
		a.eventMachine.Memory[wildernessFacing] = wildernessFacingIndex(a.spawn.Facing)
		a.eventMachine.Memory[wildernessRefuse] = 0
	}
	if !a.initialMap.Grid.CanMoveDungeonWrapped(int(a.spawn.X), int(a.spawn.Y), a.spawn.Direction()) {
		// 鎖住的門會出 `Bash`／`Pick`／`Knock`／`Exit` 的選單（spec 122）；
		// 實牆就只是擋著。
		if a.beginDoorMenu(int(a.spawn.X), int(a.spawn.Y), a.spawn.Direction()) {
			return nil
		}
		a.statusLine = "A wall blocks the way."
		return nil
	}
	// **走得過去，上一道門的選單就過期了。**
	//
	// `a.door` 是持久狀態（同一道門再撞一次要接續 BASH／PICK 額度），但那個
	// 「同一道門」的前提是**中間沒有走成任何一步**。少了這一行，門會跟著隊伍
	// 跨到別的格：玩家在那裡觸發腳本事件，而門選單還開著吃掉方向鍵與 ENTER，
	// 於是兩個 UI 疊在一起誰都動不了。原版沒有這個狀態——撞門是當下的一次性
	// 互動，走成了就結束（spec 122）。
	a.door = nil
	if a.eventMachine != nil {
		a.cellMovedByScript = false
		a.setMapExitFlag(dx, dy)
		result, err := gamepack.RunInitialSessionCellEntry(a.eventSession, a.initialMap.Grid, a.spawn)
		if err != nil {
			return fmt.Errorf("dispatch Pool initial cell: %w", err)
		}
		result, err = a.consumeInitialTransitionResources(result)
		if err != nil {
			return err
		}
		// **往文字框寫字不是停頓點**（spec 082）。每走一步跑的入口 0 與
		// SearchLocation 走的是同一套：`12h PRINTCLEAR` 換頁、`11h PRINT`
		// 接著印，兩者之間沒有等待指令，要玩家按一下的是腳本自己放的單選項
		// 選單。這條路徑先前每一則都停一次，同一段話因此被切成好幾幀，而且
		// 每停一次就多吃一個按鍵。
		for boundaries := 0; boundaries < 64 && presentationBoundary(result); boundaries++ {
			if result.Events[0].Opcode == gamepack.PrintClearOpcode && result.Events[0].Text == "" {
				a.eventText = ""
			}
			a.applyCellECLResult(result)
			if result.Exited {
				// 印完就結束的那一種：字留在框裡，這一步照走。停在這裡的話，
				// 空的 `12h PRINTCLEAR` 會變成一個沒有內容的 pending——畫面
				// 上什麼都沒有，方向鍵卻按不動。
				a.cellTextSticky = a.eventText != ""
				break
			}
			var err error
			result, err = a.eventSession.RunUntilEvent(4096, nil, true)
			if err != nil {
				return fmt.Errorf("continue Pool cell entry presentation: %w", err)
			}
			// 換區塊會帶著要載入的資源，每一輪都要收——只在進迴圈前收一次的話，
			// 迴圈裡換過去的那一份就漏了（`consumeInitialSearch` 是每一輪收）。
			if result, err = a.consumeInitialTransitionResources(result); err != nil {
				return err
			}
		}
		if !result.Exited || result.WaitingForMenu ||
			(len(result.Events) != 0 && !presentationBoundary(result)) {
			return a.pauseInitialCellResult(result)
		}
		if wilderness {
			if a.eventMachine.Memory[wildernessRefuse] == 255 {
				a.statusLine = "The wilderness step was refused by the original script."
				return a.beginInitialSearch()
			}
			a.eventMachine.Memory[wildernessX] = a.eventMachine.Memory[wildernessNextX]
			a.eventMachine.Memory[wildernessY] = a.eventMachine.Memory[wildernessNextY]
		}
		// 腳本自己叫過 `CALL C01Eh` 就已經走過那一步了，不能再走一次。
		if !a.cellMovedByScript {
			a.spawn.X = uint8(geometry.WrapCoordinate(int(a.spawn.X)+dx, geometry.Width))
			a.spawn.Y = uint8(geometry.WrapCoordinate(int(a.spawn.Y)+dy, geometry.Height))
			a.advanceGameMinute()
		}
		return a.beginInitialSearch()
	}
	a.spawn.X = uint8(geometry.WrapCoordinate(int(a.spawn.X)+dx, geometry.Width))
	a.spawn.Y = uint8(geometry.WrapCoordinate(int(a.spawn.Y)+dy, geometry.Height))
	a.advanceGameMinute()
	a.rememberGuideCell()
	a.statusLine = "Moved using original GEO data; cell ECL returned normally."
	return nil
}

// wildernessFacingIndex 把四方位的朝向換成那張八支 `ON GOSUB` 的索引。
//
// `26h ON GOSUB` 的索引**從 0 起算**（引擎 `machine.go` 的 `targets[index]`），
// 而八支的順序照 `branch_targets` 讀出來是：
//
//	0 `9A9Dh` Y−1（北）      1 `9AA7h` Y−1 X+1（東北）
//	2 `9AB0h` X+1（東）      3 `9ABAh` X+1 Y+1（東南）
//	4 `9AC3h` Y+1（南）      5 `9ACDh` Y+1 X−1（西南）
//	6 `9AD6h` X−1（西）      7 `9A94h` X−1 Y−1（西北）
//
// 所以四方位是 0、2、4、6，也就是朝向乘二。用 1、3、5、7 的話每一步都會
// 斜著走——實測往東走一步，X 與 Y 會同時加一。
func wildernessFacingIndex(facing uint8) uint16 {
	return uint16(facing%4) * 2
}

// consumeInitialTransitionResources 把腳本停下來的那些邊界一個一個吃掉。
//
// `2Dh CALL` 的五個有動作的選擇子與其餘「沒有動作、直接繼續」的那些，
// 由 spec 104 逐條讀出來——沒有動作不代表可以停在那裡，仍然要續跑。
func (a *app) consumeInitialTransitionResources(result eclvm.Result) (eclvm.Result, error) {
	// 上限只是防呆。`2Dh CALL` 也走這條路之後，一次移動可以連續吃掉十幾個
	// 邊界（樞紐那幾張圖每一格都有 CALL），8 太小會誤報成硬失敗。
	for boundary := 0; boundary < 64; boundary++ {
		if err := a.syncArchiveFromEventMachine(); err != nil {
			return result, err
		}
		if result.WaitingForMenu {
			// **停在選單之前發生的資源事件仍然要套用。**
			//
			// 一段腳本可以先換地圖再問問題，而 `RunUntilEvent` 會把兩者放進
			// 同一個 result：`ecl8/29 AEAFh` 是
			// `LOAD FILES #32` → `LOAD PIECES` → `GOTO A1CCh` → 選單。
			// 先前這裡看到 `WaitingForMenu` 就整個返回，那兩個資源事件因此
			// 被丟掉——地圖沒換，隊伍留在原本那一張圖上，而腳本以為已經換了。
			//
			// 症狀是**無限迴圈而不是報錯**：`AE87h` 把 `4A10h` 設回 1、
			// 跳到井的問句，答完 `A23Fh` 再清成 0，下一步又被設回 1，
			// 井的問句就一直跳。探索器在那裡答了四千次出不來。
			//
			// 這裡只套用、不續跑：續跑要等玩家回答。
			for _, event := range result.Events {
				if len(event.Arguments) != len(event.ArgumentsValid) {
					return result, fmt.Errorf(
						"Pool resource event 0x%02X has mismatched argument validity", event.Opcode)
				}
				if _, err := a.applyTransitionResource(event); err != nil {
					return result, err
				}
			}
			return result, nil
		}
		if result.Exited || len(result.Events) != 1 {
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
			return result, fmt.Errorf("continue Pool transition resource 0x%02X from ecl%d/%d: %w",
				event.Opcode, a.eclArchive, a.eventSession.CurrentBlockID(), err)
		}
		// **寫入要累積，不能跟著 result 一起換掉。** 呼叫端只會套用最後回傳
		// 的那一個 result，所以被吃掉的那幾段裡的 `SAVE` 就這樣消失了——
		// 而腳本的傳送正是寫在資源邊界之間：斯托亞諾夫城門付過路費那一支
		// （`ecl2/9 AC4Bh`）在 `PICTURE 255` 與 `CALL 2C90h` 中間寫
		// `C04B`／`C04C`，隊伍因此站在原地不動，記憶體裡卻已經是新座標。
		next.Writes = append(append([]eclvm.Write{}, result.Writes...), next.Writes...)
		result = next
	}
	detail := ""
	if len(result.Events) == 1 {
		detail = fmt.Sprintf("，最後停在 opcode %02X @%04X",
			result.Events[0].Opcode, result.Events[0].PC+0x9900)
		if selector, ok := a.scriptCallSelector(result.Events[0]); ok {
			detail += fmt.Sprintf(" 選擇子 %04X", selector)
		}
	}
	return result, fmt.Errorf("Pool transition resource boundary limit exceeded%s", detail)
}

func (a *app) applyTransitionResource(event eclvm.Event) (bool, error) {
	if len(event.Arguments) != len(event.ArgumentsValid) {
		return false, fmt.Errorf("Pool resource event 0x%02X has mismatched argument validity", event.Opcode)
	}
	if event.Opcode == 0x2D {
		// `2Dh CALL` 一律吃掉再往下跑。原版的分派只認五個選擇子，其餘什麼
		// 都不做——讓它停在這裡的話，樞紐那幾張圖每一格都有一個 CALL，
		// 隊伍一步都走不動（ecl6/25、ecl7/26、ecl8/27、ecl8/29 是每一格）。
		if selector, ok := a.scriptCallSelector(event); ok {
			a.applyScriptCall(selector)
		}
		return true, nil
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
		// 第一欄是 `FFh` 或 `7Fh` 就不載地圖：原版的 handler 在這兩個值上
		// 直接跳過 GEO loader（spec 043）。原本只認 `{FF,FF,7F}` 這**一組**，
		// 於是索寇要塞的 `LOAD FILES FFh,2,FFh` 會被讀成「載入 block 255」
		// 然後硬失敗——那條路正是清完要塞、開出其他航線的主線。
		if event.Arguments[0] == 0xFF || event.Arguments[0] == 0x7F {
			return true, nil
		}
		if event.Arguments[0] > 0xFF {
			return false, fmt.Errorf("Pool LOAD FILES has invalid arguments %v/%v", event.Arguments, event.ArgumentsValid)
		}
		// 用**區塊編號**查地圖，不用目前的 archive：編號在八個 GEO 檔裡全域
		// 唯一（`TestGeometryBlockIDsAreGloballyUnique` 釘住這條性質），
		// 而 `21h` 只帶編號（spec 043：第二欄整支沒有 consumer）。
		// 追 archive 會落後——`ecl7` block 26 要 `LOAD FILES 5`，
		// 當下 archive 是 7，而 block 5 在 GEO5。
		loaded, ok := a.geometryCatalog.MapByBlock(uint8(event.Arguments[0]))
		if !ok {
			return false, fmt.Errorf("Pool LOAD FILES requested absent GEO block %d", event.Arguments[0])
		}
		a.initialMap = &loaded
		a.spawn.Map = loaded.Key
		return true, nil
	case 0x37:
		if reflect.DeepEqual(event.Arguments, []uint16{127, 127, 127}) {
			return true, nil
		}
		selectors := [3]uint8{}
		for index, value := range event.Arguments {
			// `FFh` 是「這一格不換」的哨兵（spec 043），交給 loader 沿用。
			if value > 0xFF {
				return false, fmt.Errorf("Pool LOAD PIECES %v exceeds byte range", event.Arguments)
			}
			selectors[index] = uint8(value)
		}
		if a.loadPieceSlots == nil {
			return false, fmt.Errorf("Pool LOAD PIECES loader is not configured")
		}
		// 用 ECL 的 archive，不是 GEO 的。`LOAD PIECES` 是腳本要的資源，
		// 而腳本所在的 archive 與它載進來的地圖不一定同號——野外那幾張圖
		// 就是（ecl7/26 載的是 GEO5 的區塊）。`WALLDEF5.DAX` 只有 1 與 24
		// 兩塊，拿 GEO 的號去找 selector 3 一定落空。
		archive := a.eclArchive
		if archive == 0 {
			archive = a.spawn.Map.Archive
		}
		loaded, err := a.loadPieceSlots(archive, selectors)
		if err != nil {
			return false, fmt.Errorf("Pool LOAD PIECES archive %d %v: %w",
				archive, selectors, err)
		}
		a.initialWalls = &loaded
		return true, nil
	default:
		return false, nil
	}
}

// partyWindow 讓 ECL 的 active-character 視窗看到活的隊伍，而不是建 session
// 當下的快照。賭場贏來的白金、船資扣掉的白金都要留在隊伍身上（spec 102）。
type partyWindow struct{ app *app }

func (w partyWindow) Character(index int) (gamepack.InitialCharacter, bool) {
	if w.app == nil || index < 0 || index >= len(w.app.state.Party) {
		return gamepack.InitialCharacter{}, false
	}
	character := w.app.state.Party[index]
	return gamepack.InitialCharacter{
		Name: character.Name, ClassID: character.ClassID, Abilities: character.Abilities,
		ExceptionalStrength: character.ExceptionalStrength, CurrentHP: character.CurrentHP,
		Platinum: character.Money[pooltreasure.Platinum],
	}, true
}

func (w partyWindow) CommitPlatinum(index int, platinum uint16) {
	if w.app == nil || index < 0 || index >= len(w.app.state.Party) {
		return
	}
	w.app.state.Party[index].Money[pooltreasure.Platinum] = platinum
	name := w.app.state.Party[index].Name
	for library := range w.app.state.CharacterLibrary {
		if w.app.state.CharacterLibrary[library].Name == name {
			w.app.state.CharacterLibrary[library].Money[pooltreasure.Platinum] = platinum
		}
	}
}

func (a *app) configureEventSession(session *eclvm.BlockSession) error {
	if session == nil {
		return fmt.Errorf("Pool ECL session is nil")
	}
	binding, err := gamepack.SetCharacterWindow(session, partyWindow{app: a})
	if err != nil {
		return err
	}
	a.characterBinding = binding
	// 接上一個 session 的時候，它握著的就是呼叫端剛設好的那一份 archive。
	// （`restoreCampaign` 是先接再設 `eclArchive`，所以它自己再設一次。）
	a.eclSessionArchive = a.eclArchive
	// 比較的對象是 **session 手上真的是哪一份**（`eclSessionArchive`），
	// 不是 `eclArchive`。後者是給地圖與素材命名用的鏡像，
	// `syncArchiveFromEventMachine` 可能在 `NEWECL` 真的執行之前就先更新
	// 它——那時 resolver 會看到「選擇子等於現行值」而不換 catalog，
	// session 卻還握著上一個 archive 的區塊，於是換到目的區塊時報
	// 「target block 0x1A is unavailable」。
	return session.SetBlockCatalogResolver(func(_, _ uint16, memory map[uint16]uint16) (map[uint16][]byte, error) {
		selector := memory[0x6E12]
		if selector == 0 || selector == uint16(a.eclSessionArchive) {
			return nil, nil
		}
		if selector > 8 {
			return nil, fmt.Errorf("Pool ECL archive selector 0x%X is outside 1..8", selector)
		}
		archive, ok := a.eclCatalog.Archive(uint8(selector))
		if !ok {
			return nil, fmt.Errorf("Pool ECL archive %d is unavailable", selector)
		}
		a.eclSessionArchive = uint8(selector)
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
	// **不要**順手把 `spawn.Map.Archive` 也設成它。地圖的 archive 由
	// `LOAD FILES` 的區塊編號決定（spec 043），ECL 的 archive 是另一回事；
	// 兩者混在一起會生出 GEO1/21 這種不存在的組合。
	return nil
}

func (a *app) beginInitialSearch() error {
	// 新的一格，等待次數從頭算（見 `pauseAppliedCellResult`）。
	a.cellWaitedOnce, a.cellTextSticky = false, false
	// 命令列的兩個搜尋旗標是 ECL 變數，腳本自己會讀（spec 136）。
	if a.eventSession != nil {
		if machine := a.eventSession.Machine(); machine != nil {
			machine.Memory[searchFlagsAddress] = a.searchFlags
		}
	}
	result, err := gamepack.RunInitialSessionSearchEntry(a.eventSession, a.initialMap.Grid, a.spawn)
	if err != nil {
		return fmt.Errorf("start Pool SearchLocation: %w", err)
	}
	return a.consumeInitialSearch(result)
}

// beginCampInterruption 把紮營被打斷之後那一棒交給 ECL 的入口 3（spec 114）。
//
// 走的是搜尋那條路的同一套消費流程——入口 3 產生的事件（印字、水平選單）
// 和搜尋一模一樣，所以文字框、等待與選擇都不必另外寫一份。
func (a *app) beginCampInterruption() error {
	if a.eventSession == nil {
		return nil
	}
	a.cellWaitedOnce, a.cellTextSticky = false, false
	result, err := gamepack.RunInitialSessionCampEntry(a.eventSession, a.initialMap.Grid, a.spawn)
	if err != nil {
		return fmt.Errorf("start Pool camp interruption: %w", err)
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
		// `11h PRINT` **不是停頓點**：它接著印，與前面那一頁是同一頁
		// （spec 082）。原版走到市政廳外的第二段就是
		// `12h PRINTCLEAR` ＋ `11h PRINT` 一次顯示成四行——remake 先前在
		// 每一個 `11h` 都停一次，玩家因此要多按幾次 Return，而中間那一幀
		// 是原版沒有的（`docs/audit/dos-parity-sample.md` 的市政廳那一段）。
		// **往文字框寫字不是停頓點。** `12h PRINTCLEAR` 換一頁、`11h PRINT`
		// 接著印，兩者之間都沒有等待指令（spec 082）。
		//
		// 原版要玩家按一下的時候，**腳本自己會放一個單選項的選單**：市政廳外
		// 第一段之後的 `GOSUB 0xAF1C` 就是 `HORIZONTAL MENU`，唯一的選項是
		// 字串 `PRESS <RETURN> OR BUTTON TO CONTINUE`（overlay-03 `11B1h` 在
		// 選項數是 1 時把它換成 `PRESS <ENTER>/<RETURN> TO CONTINUE` 再畫）。
		// 所以「要不要等」是腳本決定的，不是每一頁自動加的——remake 先前在
		// 每一則文字都停一次，同一段話因此被切成好幾幀。
		// 但**同一則文字可以帶著它的選單**：蘇恩神殿的 `DO YOU SEEK HEALING?`
		// 就是 `11h PRINT` 與 `YES NO` 一起回來的。這種要停——不停的話這裡
		// 會用 `nil` 選擇繼續跑，等於替玩家按了第一個選項。
		if presentationBoundary(result) {
			if result.Events[0].Opcode == 0x12 && result.Events[0].Text == "" {
				a.eventText = ""
			}
			if result.Exited {
				// 印完就結束的那一種：字留在框裡，回自由移動。
				a.finishCellBlockKeepingText()
				return nil
			}
			next, err := a.eventSession.RunUntilEvent(4096, nil, true)
			if err != nil {
				return fmt.Errorf("continue Pool SearchLocation presentation: %w", err)
			}
			result = next
			continue
		}
		if result.Exited && !result.WaitingForMenu && len(result.Events) == 0 {
			// **腳本跑完時文字要留在框裡。** 原版走到市政廳外，第二段印完之後
			// 底下換成指令列、方向鍵就走得動了，而那幾行字還在框裡
			//（dosgolem 實測，見 `docs/audit/dos-parity-sample.md`）。
			// remake 先前在這裡把框清空，玩家等於少看一段。
			if a.eventText != "" {
				a.finishCellBlockKeepingText()
				return nil
			}
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

// presentationBoundary 說這一次停頓只是往文字框寫字（spec 082）。
//
// `11h PRINT` 接續、`12h PRINTCLEAR` 換頁、`0Eh` 清框，三者與前後之間都沒有
// 等待指令，所以拿到這種 result 要接著跑，不要停下來等玩家——原版要玩家按
// 一下的地方，腳本自己會放一個只有一個選項的 `HORIZONTAL MENU`。
//
// **帶著選單的那一則不算**：蘇恩神殿的 `DO YOU SEEK HEALING?` 就是 `11h`
// 與 `YES NO` 一起回來的，接著跑等於用 `nil` 選擇替玩家按了第一個選項。
//
// 每走一步的入口 0 與 SearchLocation 兩條路徑共用這個判準。分成兩份寫過，
// 結果只有一條改對，另一條照樣每則停一次。
func presentationBoundary(result eclvm.Result) bool {
	if result.WaitingForMenu || len(result.Events) != 1 {
		return false
	}
	// 帶著要前端接手的請求就不是「只寫字」：開戰、怪物設定與寶物都得停下來
	// 交出去，接著跑等於把它們靜靜跳過。
	//
	// **換區塊與挑角色不在這一列**：`NEWECL` 由 session 自己 `switchTo` 完成，
	// `CharacterSelections`／`PartyStrengthRequests` 由 VM 的 resolver 當場答完，
	// 到前端時都只是紀錄。把它們也當成「要接手」的話，市政廳那一帶的空
	// `12h PRINTCLEAR` 會變成沒有內容的 pending，方向鍵按不動。
	if result.CombatRequested || result.MonsterSetup != nil || len(result.TreasureRequests) != 0 {
		return false
	}
	switch result.Events[0].Opcode {
	case gamepack.PrintOpcode, gamepack.PrintClearOpcode,
		gamepack.PictureOpcode, gamepack.ApproachOpcode:
		return true
	}
	return false
}

// enterCombatStaging 接 `24h COMBAT` 排出來的遭遇：把每一群怪物的 285-byte
// 記錄讀進來（spec 048 的 MON*CHA 與 staging），而那條指令同時分派戰鬥、
// 神殿與戰後服務三種去向（spec 036）。
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
		// 數量 0 的先跳過。貧民窟的隨機遭遇把數量放在 `DS:9808h`，那個值由
		// **入口 1**（走一步之後跑的那一支）在 `9AA5h` 用 `1Dh PARTYSTRENGTH`
		// 之後 ÷3 ×2 +5 算出來，紮營被打斷的入口 3 則是 `GOTO @9B68` 跳進
		// 同一段（spec 136）。換區時原版先跑入口 0（spec 025），而入口 0 不碰
		// 這個值，所以那一步讀到的還是 0。報錯會讓玩家從起點往西走一步就掛掉。
		//
		// OPEN：原版的 `0Bh LOAD MONSTER` 對數量 0 到底怎麼處置還沒讀，
		// 這裡先取「不放這種怪」——那是唯一不會比崩潰更糟的選擇。
		if spawn.Count == 0 {
			continue
		}
		record, err := a.loadMonster(archive, spawn.MonsterID)
		if err != nil {
			return fmt.Errorf("load Pool monster archive %d block %d: %w", archive, spawn.MonsterID, err)
		}
		staged = append(staged, stagedMonster{Spawn: spawn, Record: record})
		labels = append(labels, fmt.Sprintf("%s ×%d", a.monsterText.Translate(record.Name), spawn.Count))
	}
	if len(staged) == 0 {
		// 一隻都沒放成，就不要開戰鬥；開了會是一場沒有敵人的架。
		a.statusLine = "Original monster descriptors staged no monsters."
		return nil
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
			// **原版比較完就把它清成 0。** overlay-03 的 `24h` handler
			// `18A2h..18B4h` 比較 runtime state `es:[di+5C4h] == 1`、清成 0，
			// 才呼叫 overlay-04 entry 1（spec 017，IDA 逐值分派的 exact 證據）。
			// 那是一次性的服務票，不是「來過神殿」的長期標記——不清的話，
			// 進過一次神殿之後，**每一個怪物已清空的 `24h COMBAT` 都會被當成
			// 神殿**：ECL3/block 0 有八個 `24h`，實測城區 (8,4)、(1,1)、(3,1)
			// 都會把玩家拉進神殿介面，而那三格沒有神殿。
			a.eventMachine.Memory[0x6DE2] = 0
			return true
		}
	}
	return false
}

// enterTreasure 接 `27h TREASURE`：八欄的請求（七種幣別加物品）由 spec 032
// 讀出來，而戰後那張選單與「拿走一件就要從物品鏈摘掉」的邊界在 spec 034。
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
			switch {
			case errors.Is(err, gamepack.ErrTreasureBlockAbsent):
				// 那個編號在這個 ITEM 檔裡沒有。多半是腳本讀到還沒被主線
				// 設起來的變數（探索器走到主線之前的格子時會這樣）。
				// 原版不可能在這裡崩潰，所以當成「這一堆沒有物品」繼續。
				//
				// OPEN：原版的 `TREASURE` 對認不得的 item block 怎麼處置
				// 還沒讀，也還沒確認該用哪一個 archive 去找 ITEM 檔。
				a.statusLine = err.Error()
			case err != nil:
				return err
			default:
				loaded = append(loaded, items...)
			}
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
	// 戰鬥中存不了檔。原版的 SAVE 在營地，戰鬥畫面根本沒有那個入口；而這裡
	// 的 Campaign 只存 ECL session 與座標，`tactical` 與 `combatMonsters`
	// 都不在裡面——存了讀回來怪物會整批消失，等於免費脫離戰鬥，而且 ECL
	// session 停在「戰鬥進行中」那一點，兩邊對不起來。
	//
	// 這一條要排在對話那一條前面：戰術地圖開著的時候 `cellEventPending`
	// 仍然是 true（戰鬥掛在格子事件底下），排後面的話玩家會看到「先把對話
	// 讀完」，而畫面上根本沒有對話。
	if a.combatActive || a.tactical != nil {
		return poolsave.State{}, fmt.Errorf("finish the current Pool battle before saving")
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
		X:          a.spawn.X, Y: a.spawn.Y, Facing: a.spawn.Facing,
		Clock:      a.gameTime, Session: snapshot,
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
	a.eclSessionArchive = campaign.ECLArchive
	a.gameTime = campaign.Clock
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

// templeHealServiceIDs 是 H）EAL 底下九項的順序，照原版選單。名稱不在這裡
// 寫死——由 `temple.Services` 取，兩份表就不會漂開（spec 115）。
var templeHealServiceIDs = []string{
	"cure-blindness", "cure-disease", "cure-light-wounds", "cure-serious-wounds",
	"cure-critical-wounds", "neutralize-poison", "raise-dead", "remove-curse",
	"stone-to-flesh",
}

var templeHealOptions = buildTempleHealOptions()

func buildTempleHealOptions() []string {
	options := make([]string, 0, len(templeHealServiceIDs)+1)
	for _, id := range templeHealServiceIDs {
		service, ok := temple.ServiceByID(id)
		if !ok {
			panic("Pool temple service " + id + " is missing from temple.Services")
		}
		options = append(options, service.Name)
	}
	return append(options, "Exit")
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
		case 3:
			a.enterTempleAppraise()
			return nil
		default:
			a.statusLine = "This temple service remains fail-closed until its DOS rules are READY."
			return nil
		}
	case templeAppraise:
		switch a.cellMenuCursor {
		case 0:
			return a.offerAppraise(appraiseGem)
		case 1:
			return a.offerAppraise(appraiseJewel)
		default:
			a.enterTempleMain()
			return nil
		}
	case templeAppraiseOffer:
		a.resolveAppraise(a.cellMenuCursor == 1)
		return nil
	case templeHeal:
		if a.cellMenuCursor == len(templeHealOptions)-1 {
			a.enterTempleMain()
			return nil
		}
		if a.cellMenuCursor < 0 || a.cellMenuCursor >= len(templeHealServiceIDs) {
			a.statusLine = "This temple service is not on the original menu."
			return nil
		}
		a.templeService = a.cellMenuCursor
		service, _ := temple.ServiceByID(templeHealServiceIDs[a.templeService])
		// 沒有那個毛病時原版先印 `is not …` 再問要不要照做，錢照收。
		// 這裡把那一句放進事件文字，選 YES 才會付錢。
		if !service.Applies(a.state.Party[a.templeParty]) && service.Refusal != "" {
			a.statusLine = a.state.Party[a.templeParty].Name + " " + service.Refusal
		} else {
			a.statusLine = "Confirm the original temple service price."
		}
		a.templeStage = templeConfirm
		a.cellMenuOptions = []string{"YES", "NO"}
		a.cellMenuCursor = 0
		a.eventText = fmt.Sprintf("%d gold pieces.\npay for cure", service.Cost)
		a.eventLabel = a.cellMenuLabel()
		return nil
	case templeConfirm:
		if a.cellMenuCursor != 0 {
			a.enterTempleHeal()
			return nil
		}
		before := a.state
		before.Party = append([]poolsave.Character(nil), a.state.Party...)
		before.CharacterLibrary = append([]poolsave.Character(nil), a.state.CharacterLibrary...)
		id := templeHealServiceIDs[a.templeService]
		result, err := temple.Serve(&a.state, a.templeParty, id, a.roller)
		if err != nil {
			a.enterTempleHeal()
			if errors.Is(err, temple.ErrNotEnoughMoney) {
				a.eventText = "Not enough money."
				a.statusLine = "The cure was not purchased; no money or HP changed."
				return nil
			}
			// 沒有那個毛病：原版不收錢也不做事，只留那一句。
			a.eventText = err.Error()
			a.statusLine = "Nothing was purchased; no money changed."
			return nil
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
		if result.Healed > 0 {
			a.statusLine = fmt.Sprintf("Paid %d GP from %s; restored %d HP.",
				result.Cost, result.PaidFrom, result.Healed)
		} else {
			a.statusLine = fmt.Sprintf("Paid %d GP from %s.", result.Cost, result.PaidFrom)
		}
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
// finishCellBlockKeepingText 是「腳本結束，但文字留在框裡」——原版一格事件
// 等過一次之後就是這樣：玩家可以直接走開，那幾行字留到下一件事把框換掉。
func (a *app) finishCellBlockKeepingText() {
	text := a.eventText
	// 手冊提示跟著文字一起留下來：那幾則引用還在框裡寫著，按下去要翻得到
	//（spec 132）。
	cues, done := a.journalCues, a.journalCueDone
	a.finishCellBlock()
	a.eventText, a.cellTextSticky = text, true
	a.journalCues, a.journalCueDone = cues, done
	a.statusLine = ""
}

func (a *app) finishCellBlock() {
	// 腳本這一段跑完了，把 active-character 視窗裡的值抄回隊伍：原版的視窗
	// 就是那個人的記錄，腳本改的是本尊（spec 021）。
	if a.characterBinding != nil {
		a.characterBinding.Flush(a.eventMachine)
	}
	a.cellEventPending, a.cellWaitingMenu = false, false
	a.cellTextSticky = false
	a.templeActive = false
	a.cellMenuOptions, a.cellMenuCursor = nil, 0
	a.eventText, a.eventLabel = "", ""
	a.journalCues, a.journalCueDone = nil, nil
	a.rememberGuideCell()
	a.statusLine = "Moved using original GEO data; per-turn and SearchLocation returned normally."
}

// textOnlyPause 說這一次停頓除了往文字框寫字之外什麼都沒做。
func textOnlyPause(result eclvm.Result) bool {
	if result.WaitingForMenu || result.CombatRequested || result.NewECLBlockID != nil ||
		len(result.TreasureRequests) != 0 || result.MonsterSetup != nil {
		return false
	}
	if len(result.Events) == 0 {
		return false
	}
	for _, event := range result.Events {
		switch event.Opcode {
		case gamepack.PrintOpcode, gamepack.PrintClearOpcode,
			gamepack.PrintReturnOpcode, gamepack.ClearBoxOpcode:
		default:
			return false
		}
	}
	return true
}

func (a *app) pauseInitialCellResult(result eclvm.Result) error {
	a.applyCellECLResult(result)
	return a.pauseAppliedCellResult(result)
}

// pauseAppliedCellResult 是同一件事，但**不再套用一次**。`33h PRINT RETURN`
// 之後文字框的內容會隨套用次數改變，所以同一個 result 只能套一次；先前
// 每則訊息蓋掉上一則，套兩次看不出差別，這個重複因此一直沒被發現。
func (a *app) pauseAppliedCellResult(result eclvm.Result) error {
	a.cellWaitedOnce = true
	a.templeActive = false
	a.cellEventPending = true
	// **新的一則事件不是上一格留在框裡的字。** `cellTextSticky` 是
	// 「腳本跑完了但文字還留著」的狀態（`finishCellBlockKeepingText`），
	// 那時 ENTER 拿去翻手冊。忘了清的話，下一格的選單開起來時它還亮著，
	// 而翻手冊那一段在輸入分派的更前面——ENTER 全被它吃掉，選單就永遠
	// 答不了：實測探索器在市政廳的守衛與索寇要塞的碼頭上各答了上萬次，
	// 一次都沒送進 VM。
	a.cellTextSticky = false
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
	// **選項只有一個的時候原版不畫那個選項。** overlay-03 `11B1h` 改畫自己
	// 那一條「按鍵繼續」（spec 082）——所以市政廳外、文書官辦公室、鬼魂那幾
	// 段雖然腳本各寫各的（`PRESS <RETURN> OR BUTTON TO CONTINUE`、
	// `HIT <RETURN> …`、`PRESS BUTTON OR …` 共六種），玩家看到的都是同一句。
	if len(a.cellMenuOptions) == 1 {
		return a.gameText.Translate(a.continueLabel(a.cellMenuOptions[0]))
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

// continueLabel 是「按鍵繼續」那一列的原文。讀不到原版字串時退回呼叫端手上
// 那一條，別讓提示整條消失。
func (a *app) continueLabel(fallback string) string {
	if a.continuePrompt != "" {
		return a.continuePrompt
	}
	if fallback != "" {
		return fallback
	}
	return "RETURN"
}

// mapExitFlagAddress 是「隊伍正要走出這一區」的 ECL 變數（spec 100）。
//
// **原版的寫入點還沒找到**，所以「什麼時候該是 1」是 strong inference：
// 三個使用點都在地圖邊上、都通往離開這一區，貧民窟那一支還用朝向挑鄰居
// ——只有「往哪一邊走出去」需要那個分支。攻略也是這樣寫的：菲蘭分成幾區，
// 區與區之間靠邊界上的城門相接，走過去就到隔壁區。
//
// 三個地方讀它，讀到非零就離開這一區：城區的西門（`ecl3` block 0 `993Ah`
// → 貧民窟）、貧民窟的邊界（`ecl2` block 20 `9934h`，**用朝向挑鄰居**）、
// 索寇要塞的碼頭（`ecl4` block 21 `9918h` → 回菲蘭的船）。
//
// 座標本身是繞回去的（overlay-30 `0358h` 在查牆之前把 X／Y 夾回 0..15，
// spec 099），所以繞回之後那一格看起來合法——引擎另外記下「這一步本來會
// 走出去」才說得通。攻略也是這樣寫的：菲蘭分成幾區，**區與區之間靠邊界上的
// 城門相接，走過去就到隔壁區**。
//
// **原版的寫入點還沒找到**（沒有任何 ECL 寫它，執行檔裡也沒有直接定址它），
// 所以語意是 strong inference，不是 exact。
const mapExitFlagAddress = 0x6DD5

// 野外地圖（ECL block 25、26、27，spec 105）。
const (
	// wildernessX／wildernessY 是隊伍在野外的位置。X 的範圍 2..15、
	// Y 的範圍 8..33，所以它們不是格子座標。
	wildernessX = 0x49C3
	wildernessY = 0x49C4
	// wildernessNextX／wildernessNextY 是入口 0 算出來的「這一步要去哪」。
	wildernessNextX = 0x00FB
	wildernessNextY = 0x00FC
	// wildernessFacing 是那張八支 `ON GOSUB` 的索引（ecl7/26 `9A18h` 的
	// 運算元就是這個位址），順序為北、東北、東、東南、南、西南、西、西北。
	wildernessFacing = 0x033D
	// wildernessRefuse 由 ECL 寫 255 表示「這一步不給走」。三處都是這個
	// 意思：撞到不可通行表、Y 到北緣、跨圖的例外座標。
	wildernessRefuse = 0x6DC9
	// searchFlagsAddress 是 `[4937h]+594h` 的 ECL 位址，也就是命令列那兩個
	// 搜尋旗標（`command_bar.go` 的 `searchFlags`）。腳本會讀它——貧民窟走路
	// 遭遇在 `9B40h` 比對 `@6DCA == 1`，相等就把那一擲加四（spec 136）。
	// 所以這個值不投影進去，邊走邊搜就不會像原版那樣把怪引出來。
	searchFlagsAddress = 0x6DCA
	// wildernessArea 是 255 就代表隊伍在**區域圖**裡，不在野外地形上。
	//
	// 野外那三個區塊各有兩種身分：`LOAD FILES 4,4,0` 載的是地形（在上面走，
	// 野外座標跟著動），`LOAD FILES 25,2,255` 載的是區域圖，進去之前
	// `ecl6/25 A46Ch` 會寫 `4A9E = 255`，而入口 0 的第一行就是
	// `COMPARE @4A9E, 255 → EXIT`——**整段野外的每步處理不跑**。
	// 所以在區域圖裡不能再做野外那一套：`00FBh`／`00FCh` 是上一次野外移動
	// 留下的舊值，照抄回 `49C3`／`49C4` 沒有意義（spec 105）。
	wildernessArea = 0x4A9E
)

// wildernessBlocks 是三張野外圖的 ECL 區塊編號（spec 105）。
var wildernessBlocks = map[uint16]bool{25: true, 26: true, 27: true}

// inWilderness 說目前的腳本區塊是不是野外圖。
func (a *app) inWilderness() bool {
	return a.eventSession != nil && wildernessBlocks[a.eventSession.CurrentBlockID()]
}

// inWildernessOverland 說隊伍是不是真的站在野外地形上——野外區塊的另一種
// 身分（區域圖）不算。
func (a *app) inWildernessOverland() bool {
	return a.inWilderness() && a.eventMachine != nil &&
		a.eventMachine.Memory[wildernessArea] != 255
}

// mapExitCommitCall 是 `2Dh CALL C01Eh` 的選擇子。
//
// `2Dh` 的處理常式（overlay-03 `3026h`）拿運算元的值減 `7FFFh` 之後分派；
// `C01Eh` 那一支（`30FAh`）呼叫 overlay-07 `1A17h`，而那一支做的事很明確：
//
//	1a1a  al = DS:6A0Dh                ; 朝向
//	1a1f  0（北）→ Y > 0 就 Y--，否則 Y = 15
//	1a38  2（東）→ X < 15 就 X++，否則 X = 0
//	1a51  4（南）→ Y < 15 就 Y++，否則 Y = 0
//	1a6a  6（西）→ X > 0 就 X--，否則 X = 15
//	1a81  重算地形與牆的暫存
//
// 就是**把這一步走掉，而且在邊界繞回去**。三個讀 `6DD5h` 的分支後面都緊跟著
// 它，所以走出這一區的那一步是由 ECL 自己叫這一支完成的，不是引擎默默做的。
const mapExitCommitCall = 0xC01E

// setMapExitFlag 在跑格子入口 0 之前，把「這一步會不會走出這一區」寫進去。
//
// 座標本身是繞回去的（overlay-30 `0358h` 在查牆之前把 X／Y 夾回 0..15，
// spec 099），所以繞回之後那一格看起來合法——引擎另外記下「這一步本來會
// 走出去」才說得通。
// 邊界是**繞回去**，與原版相同（spec 125）。
//
// 提交這一步的是 ECL 的 `2Dh CALL C01Eh`——選擇子指到 overlay-07 entry 27
// （`1A17h`），內容逐朝向寫死「還沒到邊就加減一，到邊了就跳到對邊」。
// 這裡的 `WrapCoordinate` 與它逐條相同。
//
// 前面那一支 overlay-14 `06AEh`（上鍵）看起來像在「夾」座標，其實**那四個
// 寫入一律等於原值**（X 是 0 往西算出 −1、夾回 0）——它真正的作用是把 ECL 的
// `@6DD5` 立成 1，讓那一格的腳本決定要不要換圖（spec 100）。
// **把前置當成提交、照著改成夾，野外那三張圖的貼圖就會壞。**

func (a *app) setMapExitFlag(dx, dy int) {
	if a.eventMachine == nil {
		return
	}
	leaving := uint16(0)
	if x := int(a.spawn.X) + dx; x < 0 || x >= geometry.Width {
		leaving = 1
	}
	if y := int(a.spawn.Y) + dy; y < 0 || y >= geometry.Height {
		leaving = 1
	}
	a.eventMachine.Memory[mapExitFlagAddress] = leaving
}

// applyMapExitCommit 走 `2Dh CALL C01Eh` 那一步：依朝向移動一格、邊界繞回，
// 並把離開旗標清掉。
//
// 選擇子要看**運算元本身的字**，不是它指到的值：共用 VM 會把那個運算元當成
// 記憶體參照解出來（實測拿到的是 0），原版的處理常式比的是字本身。
func (a *app) applyMapExitCommit(result eclvm.Result) {
	if a.eventMachine == nil {
		return
	}
	for _, event := range result.Events {
		if event.Opcode != 0x2D {
			continue
		}
		instruction, err := a.eclInstruction(event.PC)
		if err != nil || len(instruction.Operands) == 0 {
			continue
		}
		selector, err := ecl.WordAddress(instruction.Operands[0])
		if err != nil || selector != mapExitCommitCall {
			continue
		}
		a.applyScriptCall(selector)
	}
}

// applyScriptCall 執行 `2Dh CALL` 的一個選擇子。
//
// 原版的分派在 overlay-03 `3026h`：把運算元減 `7FFFh` 之後只比五個值，
// **其餘一律什麼都不做**（`3126h` 直接返回）。所以認不出來的選擇子不是
// 「還沒接」，是原版本來就沒有動作——不能讓它擋住移動。
//
// 五個有動作的：`8000h`／`8001h`（overlay-07 `00A2h`）、`2C90h`（重算地形
// 暫存）、`BA03h`（音效）、`C018h`（重算牆的暫存）、`C01Eh`（依朝向走一格、
// 邊界繞回）。前四個在 remake 這邊每次查地圖時本來就重算，所以只有
// `C01Eh` 需要動作。
func (a *app) applyScriptCall(selector uint16) {
	if selector != mapExitCommitCall {
		return
	}
	switch a.spawn.Facing {
	case 0:
		a.spawn.Y = uint8(geometry.WrapCoordinate(int(a.spawn.Y)-1, geometry.Height))
	case 1:
		a.spawn.X = uint8(geometry.WrapCoordinate(int(a.spawn.X)+1, geometry.Width))
	case 2:
		a.spawn.Y = uint8(geometry.WrapCoordinate(int(a.spawn.Y)+1, geometry.Height))
	case 3:
		a.spawn.X = uint8(geometry.WrapCoordinate(int(a.spawn.X)-1, geometry.Width))
	}
	// 旗標只對「這一步」有效。不清的話換區之後的入口 0 還看得到 1，
	// 於是又換一次區——實測會在城區與貧民窟之間換到邊界上限。
	if a.eventMachine != nil {
		a.eventMachine.Memory[mapExitFlagAddress] = 0
	}
	a.cellMovedByScript = true
}

// scriptCallSelector 取出 `2Dh CALL` 的選擇子。共用 VM 會把那個運算元當成
// 記憶體參照解出來（實測拿到 0），原版比的是**字本身**，所以要回頭讀指令。
func (a *app) scriptCallSelector(event eclvm.Event) (uint16, bool) {
	instruction, err := a.eclInstruction(event.PC)
	if err != nil || len(instruction.Operands) == 0 {
		return 0, false
	}
	selector, err := ecl.WordAddress(instruction.Operands[0])
	if err != nil {
		return 0, false
	}
	return selector, true
}

// joinPrintedText 把 `11h PRINT` 接在目前這一頁後面。
//
// 原版的兩段之間**沒有空白**：量過 ecl3/0 的 `AC22`（`…IN YOUR JOURNAL YOU
// NOTE`，末字元 `E`）接 `AC9B`（`PROCLAMATIONS LXIV…`，首字元 `P`），以及
// ecl7/23 的 `A4A7`（`DO YOU REALLY MEAN`）接 `A4BD`（`?`），四段前後都不帶
// 空格。直接串接會得到 `YOU NOTEPROCLAMATIONS`，所以分隔要由呈現層補。
//
// 補的規則是「一個空格，新片段以標點開頭時不補」。**原版實際怎麼排版還沒有
// 畫面證據**——它也可能是換行——所以這是 `layout-reconstructed`，見 spec 082。
func joinPrintedText(page, line string) string {
	if page == "" || strings.HasSuffix(page, "\n") || strings.HasSuffix(page, " ") {
		return page + line
	}
	if runes := []rune(line); len(runes) > 0 && strings.ContainsRune("?!.,;:)］」』、。！？，：；", runes[0]) {
		return page + line
	}
	return page + " " + line
}

func (a *app) applyCellECLResult(result eclvm.Result) {
	a.applyMapExitCommit(result)
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
	//   - `11h PRINT` **接著印**，不清框——原版靠它把一句話拼起來。
	//   - `12h PRINTCLEAR` 是新的一頁；上一則以換行收尾時才續行。
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
		case gamepack.PrintOpcode:
			if event.Text == "" {
				continue
			}
			a.eventText = joinPrintedText(a.eventText, a.gameText.Translate(event.Text))
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
	a.updateJournalCue()
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
			a.tourDelay = a.speedDelayTicks()
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
		a.statusLine = "" // 同上：導覽走完不留開發用的字
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
			// 原版的資料頁右上角就有肖像（spec 130 的第 16 幀），用的是
			// 預設那一張——玩家要到後面（`StagePortrait`）才換得動它。
			// 那一步本來就從 1／1 起算，這裡先填同一組，不然是 0／0，
			// 載不出圖，資料頁右上角就空著。
			if a.flow.PortraitHead == 0 {
				a.flow.PortraitHead, a.flow.PortraitBody = 1, 1
			}
			if err := a.reloadPortrait(); err != nil {
				a.statusLine = err.Error()
			}
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
			a.statusLine = a.text(msgPortraitAccepted)
			a.resetIconMenu()
			if err := a.reloadIcons(); err != nil {
				return err
			}
			a.rememberOldIcons()
			return nil
		}
		return nil
	}
	if a.flow.Stage == creation.StageIcon {
		return a.iconMenuInput()
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

// beginAdventuring 是 B）EGIN ADVENTURING：「離開人物管理選擇項，開始冒險」
//（說明書 p.10）。
func (a *app) beginAdventuring() error {
	if len(a.state.Party) == 0 {
		a.statusLine = a.text(msgMenuNeedsOneCharacter)
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

// loadSavedGame 是 L）OAD SAVED GAME：「叫出以前存下的遊戲進度」（說明書 p.9）。
func (a *app) loadSavedGame() error {
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
	return nil
}

func (a *app) finishCharacter() error {
	if a.rolled == nil {
		return fmt.Errorf("Pool character confirmation has no rolled character")
	}
	for _, existing := range a.state.CharacterLibrary {
		if existing.Name == a.flow.Name {
			a.statusLine = a.text(msgMenuNameTaken)
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
	// 法術書照原版的建角規則填（spec 109）：牧師會第 1 級的全部神術，
	// 法師會寫死的四條。少了這一步，一級法師記得起火球術。
	character.Spellbook = gamepack.NewCharacterSpellbook(memberClassLevels(character),
		character.Abilities[gamepack.AbilityWisdom], a.spellSlotTables, a.spellParameters)
	// 賊技能照原版的建角規則填（spec 095）：三張表加一條算式，敏捷那一段
	// 套的是第 0 列——建角那一刻記錄裡還沒有能力值，原版讀到的就是 0。
	if err := a.fillThiefSkills(&character); err != nil {
		return err
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
	a.statusLine = fmt.Sprintf(a.text(msgMenuSavedToLibrary), character.Name)
	// 原版在同一個時機寫出 `<NAME>.CHA`／`.SPC`（spec 003 第 11 步）。
	// remake 的真相是自己的 JSON，所以匯出失敗只回報，不把角色收回去。
	if a.exportDOSCharacter != nil {
		if err := a.exportDOSCharacter(character); err != nil {
			a.statusLine = err.Error()
		}
	}
	return nil
}

func (a *app) addFirstLibraryCharacter() error {
	if len(a.state.Party) >= 6 {
		a.statusLine = a.text(msgMenuPartyFull)
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
		a.statusLine = fmt.Sprintf(a.text(msgMenuAddedToParty), candidate.Name)
		return nil
	}
	a.statusLine = a.text(msgMenuNoSpareCharacter)
	return nil
}

func (a *app) Draw(screen *ebiten.Image) {
	skin := a.currentTheme()
	background, foreground, accent := skin.background, skin.foreground, skin.accent
	screen.Fill(background)
	if a.mode == modeTitle {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(2, 2)
		screen.DrawImage(a.title, op)
		// 提示放在標題美術中間那一條黑帶：原版那一張在 native y 146..157
		// 是純黑（12 列，dosgolem 逐列量的），換算成邏輯座標是 292..315。
		// 原本畫在 382，正好壓在下面那條藍帶的第二行版權字上——兩段文字
		// 疊在一起，看起來像字型壞掉。
		hint := a.text(msgTitleHint)
		hintWidth := font.MeasureString(uiFace, displayText(hint)).Ceil()
		drawText(screen, hint, (logicalWidth-hintWidth)/2, 308, accent)
	} else if a.mode == modeMenu {
		a.drawFrame(screen, foreground, accent)
		drawText(screen, a.text(msgMenuTitle), 224, 54, accent)
		// 左欄是指令，右欄是隊伍。十一項一路排下來會撞到狀態列，
		// 而原版的畫面本來就是指令在左、名單在右。
		for index, entry := range a.visiblePartyMenuEntries() {
			label := a.text(entry.message)
			if entry.key == ebiten.KeyB && a.programManaging {
				label = a.text(msgProgramReturn)
			}
			drawText(screen, label, 112, partyMenuFirstLine+index*partyMenuLineHeight, foreground)
		}
		drawText(screen, fmt.Sprintf(a.text(msgMenuCounts), len(a.state.CharacterLibrary), len(a.state.Party)), 112, 320, accent)
		for index, member := range a.state.Party {
			// 標出 D、M、T、V、R 會作用在誰身上。
			marker := " "
			if index == a.menuMember {
				marker = ">"
			}
			drawText(screen, fmt.Sprintf("%s%d  %s", marker, index+1, member.Name),
				396, partyMenuFirstLine+index*partyMenuLineHeight, foreground)
		}
		if len(a.state.Party) > 1 {
			drawText(screen, a.text(msgMenuSelectHint), 396, 320, accent)
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
	// 底部那一列畫在 `footerBaseline`（398），也就是繩索框**外面**——
	// 原版每一頁的指令列都在那裡（native 192..198）。
	switch {
	case a.mode == modeTitle:
		// 標題那一張是原版的整幅美術，底下那條藍帶裡就是原版的版權文字。
		// 再疊一列 F-key 提示會直接壓在上面——標題畫面也按不到那幾個鍵，
		// 畫它只是把原版的畫面弄髒。`msgTitleHint` 那一句留著，那是要按的。
	case a.mode == modeCreation &&
		(a.flow.Stage == creation.StageRoll ||
			a.flow.Stage == creation.StagePortrait ||
			a.flow.Stage == creation.StageIconConfirm):
		// 這兩頁最下面那一列都是原版自己的：資料頁是
		// `KEEP THIS CHARACTER? YES NO`（spec 130），肖像編輯器是
		// `HEAD BODY KEEP`（基準畫面 `24-Return`）——兩者都由 `drawCreation`
		// 畫。兩邊都畫會疊成一團。
	case a.panelOpen():
		// 手冊、裝備、法術、商店、紮營那幾頁自己有一列鍵盤提示，而且它們
		// 開著的時候 `Update` 提早返回、指令列的鍵按不到。畫它只會從面板
		// 底下露出半個字（裝備頁左下角原本會冒出一個 `A`）。
	case a.freeMovementActive():
		// 自由移動時最下面那一列是原版的指令列（spec 119）；F-key 提示移到
		// F1 說明頁，不是拿掉。導覽還在跑的時候原版那一列是「按 Return 繼續」，
		// 所以那時不畫指令列。
		drawCommandBar(screen, a, foreground, accent)
	case a.tacticalPreview:
		// 戰術盤面那一頁自己用掉這一列（移動鍵與回合鍵的提示），
		// 而且它第一行右邊就寫著 `F5 返回`。兩邊都畫會疊在一起。
	case a.dialogueVisible():
		// 對話框自己在框內畫「按 RETURN 繼續」，基線只差四個像素。
		// 兩邊都畫的話兩行字會疊成一團——導覽那一張截圖就是這樣。
	default:
		drawText(screen, a.text(msgFooter), 20, footerBaseline, foreground)
	}
	if a.help {
		drawHelp(screen, a, background, foreground, accent, a.adventureProvenanceLines())
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
	if a.fieldCastOpen {
		drawFieldCast(screen, a, background, foreground, accent)
	}
	if a.viewSheetOpen {
		drawViewSheet(screen, a, background, foreground, accent)
	}
	if a.spriteOpen {
		drawSpriteOverview(screen, a, background, foreground, accent)
	}
	if a.campOpen && a.campStage == campStageIcon {
		// 原版的造形編輯器是整頁（overlay-16 entry 4），不是疊在紮營畫面上的
		// 小框——與建角走到那一步看到的是同一頁（spec 135）。
		panel := ebiten.NewImage(logicalWidth-2*guidePanelInset, guidePanelBottom-spellPagePanelTop)
		panel.Fill(background)
		screen.DrawImage(panel, &ebiten.DrawImageOptions{
			GeoM: translated(guidePanelInset, spellPagePanelTop)})
		drawIconEditor(screen, a, foreground, accent)
	}
	// 攻略疊在最上層：它是覆蓋層，不是另一個模式。
	if a.guideOpen {
		drawGuide(screen, a, background, foreground, accent)
	}
}

func drawAdventure(screen *ebiten.Image, a *app, foreground, accent color.Color) {
	a.drawFrame(screen, foreground, accent)
	// 原版的冒險畫面上面沒有標題列，這裡本來留著一句開發用的英文
	// （`INITIAL DOS FIRST-PERSON VIEW`）。那是給自己看的，卻是玩家
	// 整趟冒險每一格都看得到的東西。
	if a.initialMap == nil || a.initialWalls == nil {
		drawText(screen, "INITIAL MAP OR WALL ART IS NOT LOADED", 150, 190, foreground)
		return
	}
	viewLeft, viewTop := 48, 70
	// 第一人稱那一框在原版也是同一圈繩索圍起來的（spec 123）：外框
	// native (16,16)..(119,119)、內部 88×88 從 (24,24) 起，也就是**內容外面
	// 一圈 tile**。這裡照同樣的關係圍在 remake 的 176×176 視野外面。
	//
	// 視野在 `(48,70)`，所以連繩索在內這一框佔 `(32,54)..(240,278)`——
	// **一框的下緣是 278，不是視野的 262**。對話框接在 278 底下
	//（`dialogueTop`）；先前照 262 排，上框線會從繩索中間切過去。
	// 位置仍比原版低（原版視野在 native `(24,24)`，這裡是 `(24,35)`，
	// 差在 remake 上面多一列標題），那屬於整體版面，不在這一項裡。
	a.drawRopeBox(screen, viewLeft-frameTileSize*2, viewTop-frameTileSize*2, 13, 13)
	// 結局過場時那一格畫的是結局的圖，不是第一人稱視野（spec 108）。
	if a.endingActive && a.endingScene != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(2, 2)
		op.GeoM.Translate(float64(viewLeft), float64(viewTop))
		screen.DrawImage(a.endingScene, op)
		a.showDialogue(screen, a.eventText, a.gameText.Translate(a.eventLabel), foreground, accent)
		return
	}
	// 紮營的時候原版把那一框換成營火（spec 135）。與 APPROACH 一樣只換左邊
	// 那一框，右邊的隊伍面板與時鐘照畫。
	if fire := a.campFireImage(); fire != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(2, 2)
		op.GeoM.Translate(float64(viewLeft), float64(viewTop))
		screen.DrawImage(fire, op)
		drawPartyPanel(screen, a, foreground, accent)
		drawCamp(screen, a, foreground, accent)
		return
	}
	// APPROACH 的時候原版把半身像整個蓋在那一框上，不是畫視野（spec 117）。
	if portrait := a.approachPortrait(); portrait != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(2, 2)
		op.GeoM.Translate(float64(viewLeft), float64(viewTop))
		screen.DrawImage(portrait, op)
		// **半身像只蓋左邊那一框，右邊那一塊照畫。** 原版在羅夫講話的時候
		// 隊伍面板與座標時鐘都還在（dosgolem 的 `31-b`：`NAME AC HP`、
		// `HERO 10 8`、`15,1 W 00:00`）。這裡本來提早返回，把整個右半邊留白，
		// 而那是玩家第一次看到的畫面。
		drawPartyPanel(screen, a, foreground, accent)
		if a.initialEvent != nil {
			message, label := a.initialEvent.Message, a.continueLabel(a.initialEvent.ContinueLabel)
			if a.eventText != "" {
				message = a.eventText
			}
			if a.eventLabel != "" {
				label = a.eventLabel
			}
			a.showDialogue(screen, a.gameText.Translate(message), a.gameText.Translate(label), foreground, accent)
		}
		return
	}
	if a.areaMapOpen {
		// 平面圖走第一人稱那一條的同一套路：組成 88×88 的索引圖，過主題色盤，
		// 再放大兩倍貼上去。圖塊有色盤索引，直接畫到畫面上會繞過那一層。
		if area, err := a.areaMapImage(); err == nil {
			if rendered, err := area.RGBA(0, a.artPalette()); err == nil {
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Scale(2, 2)
				op.GeoM.Translate(float64(viewLeft), float64(viewTop))
				screen.DrawImage(ebiten.NewImageFromImage(rendered), op)
				drawPartyPanel(screen, a, foreground, accent)
				return
			}
		}
		// 圖塊沒載到就退回畫線那一版，總比整框空白好。
		drawAreaMap(screen, a, viewLeft, viewTop)
		drawPartyPanel(screen, a, foreground, accent)
		return
	}
	inset, err := a.firstPersonInsetImage()
	if err != nil {
		drawText(screen, "FIRST-PERSON STAGE ERROR", 72, 180, accent)
		return
	}
	rendered, err := inset.RGBA(0, a.artPalette())
	if err != nil {
		drawText(screen, "WALL VIEW ERROR", 72, 180, accent)
	} else {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(2, 2)
		op.GeoM.Translate(float64(viewLeft), float64(viewTop))
		screen.DrawImage(ebiten.NewImageFromImage(rendered), op)
	}
	// 右邊那一塊是原版的隊伍面板加狀態列，不是除錯文字；出處與現況那幾列
	// 移到 F1 的說明頁（`adventureProvenanceLines`），畫面上留給玩家看得到的
	// 東西。右欄的最後一列不能低於 262——對話框的上緣在 `dialogueTop`（264）。
	drawPartyPanel(screen, a, foreground, accent)
	dialogueVisible := a.dialogueVisible()
	if a.introWaiting && a.initialEvent != nil {
		message, label := a.initialEvent.Message, a.continueLabel(a.initialEvent.ContinueLabel)
		if a.eventText != "" {
			message = a.eventText
		}
		if a.eventLabel != "" {
			label = a.eventLabel
		}
		a.showDialogue(screen, a.gameText.Translate(message), a.gameText.Translate(label), foreground, accent)
	} else if a.tourActive && a.tourPage >= 0 && a.initialEvent != nil && a.tourStep >= 0 && a.tourStep < len(a.initialEvent.Tour) {
		step := a.initialEvent.Tour[a.tourStep]
		if a.tourPage < len(step.Messages) {
			a.showDialogue(screen, a.gameText.Translate(step.Messages[a.tourPage]),
				a.gameText.Translate(a.continueLabel(a.initialEvent.ContinueLabel)), foreground, accent)
		}
	} else if a.cellEventPending && a.eventText != "" {
		a.showDialogue(screen, a.eventText, a.gameText.Translate(a.eventLabel), foreground, accent)
	} else if a.cellTextSticky && a.eventText != "" {
		// 腳本結束了，字還留著。**這一種沒有「按 RETURN 繼續」**——原版那時
		// 底下印的是指令列，玩家可以直接走開。
		a.showDialogue(screen, a.eventText, "", foreground, accent)
	}
	if a.statusLine != "" && !dialogueVisible {
		// 畫面只有 640 寬，從 42 起算放得下 74 個字；超過就截掉，
		// 不要讓字流出框外（自由移動那一張截圖抓到過）。
		line := a.statusLine
		if len(line) > adventureStatusLineLimit {
			line = line[:adventureStatusLineLimit-1] + "…"
		}
		drawText(screen, line, 42, 342, foreground)
	}
	drawCamp(screen, a, foreground, accent)
}

// drawCamp 畫紮營時對話框那一區的字（spec 135）。
//
// **紮營不再是彈出選單**：視野那一框換成營火、指令列換成
// `CAMP: SAVE VIEW MAGIC REST ALTER EXIT`，這裡只剩對話框裡的兩行——
// 第一層是 `The party makes camp...`（overlay-15 `1E03h`），
// 排時間那一層是 `Rest Time:`（overlay-20 `05A0h`）。
// 選中的那一欄在原版是換色（`05D8h` 把顏色從 0Ah 改成 0Fh），這裡用強調色。
//
// 原版的兩行在 native 136..142 與 144..150，也就是對話框的第一、二行。
// drawIconEditor 畫戰鬥造形編輯器那一頁。建角走到那一步用它，紮營的
// `ALTER → ICON` 也用它——原版兩處都是 overlay-16 entry 4 同一支（spec 135）。
func drawIconEditor(screen *ebiten.Image, a *app, foreground, accent color.Color) {
	drawText(screen, a.text(msgIconTitle), 216, 52, accent)
	drawText(screen, a.iconMenuPath(), 48, 84, accent)
	for index, option := range a.iconMenuOptions() {
		prefix, ink := "  ", foreground
		if index == a.iconMenu.cursor {
			prefix, ink = "> ", accent
		}
		drawText(screen, prefix+a.iconOptionLabel(option.label), 48, 116+index*22, ink)
	}
	drawIconPreviews(screen, a, accent)
	size := a.iconOptionLabel("LARGE")
	if a.flow.IconSize == 1 {
		size = a.iconOptionLabel("SMALL")
	}
	drawText(screen, fmt.Sprintf(a.text(msgIconSummary),
		a.flow.IconHead, a.flow.IconWeapon, size), 340, 300, foreground)
}

// drawIconPreviews 畫**兩組四格**：上面 OLD（進編輯器時的樣子）、下面 NEW
// （現在的樣子），每一組各有 `READY` 與 `ACTION`，改了什麼一眼就比得出來。
//
// 編輯器（基準 `25-k`）與確認那一頁（基準 `26-e`）畫的是同一組四格，差別
// 只在框外那一列——所以這一段抽出來共用，兩邊各自畫自己的底下那一行。
func drawIconPreviews(screen *ebiten.Image, a *app, accent color.Color) {
	for row, group := range [2]struct {
		label string
		icons [2]*ebiten.Image
	}{
		{a.text(msgIconOld), [2]*ebiten.Image{a.iconReadyOld, a.iconActionOld}},
		{a.text(msgIconNew), [2]*ebiten.Image{a.iconReady, a.iconAction}},
	} {
		// 放大倍率從 4 降到 3：原版的造形在 320×200 上約佔畫面高度的一成，
		// 640×400 等比是 2 倍，4 倍等於放大成兩倍大而兩組排不下。3 倍是
		// 「排得開」與「看得清楚」之間的折衷。
		//
		// 行距 132 與底下那些偏移都是量出來的：造形 3 倍放大之後約 68 像素
		// 高，而 `drawText` 的 y 是**基線**、字往上長約 16 像素——所以下一組
		// 的標籤基線要比上一組的圖底再低 16 以上，否則字會壓在腳上。
		top := 40 + row*132
		drawText(screen, group.label, 404, top, accent)
		drawText(screen, a.text(msgIconReady), 356, top+22, accent)
		drawText(screen, a.text(msgIconAction), 472, top+22, accent)
		for index, icon := range group.icons {
			if icon == nil {
				continue
			}
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(3, 3)
			op.GeoM.Translate(float64(348+index*116), float64(top+42))
			screen.DrawImage(icon, op)
		}
	}
}

func drawCamp(screen *ebiten.Image, a *app, foreground, accent color.Color) {
	if !a.campOpen {
		return
	}
	drawDialogueFrame(screen, accent)
	switch a.campStage {
	case campStageRest:
		// 原版那一層只有這一行（native 136..142）。第二行是空的——
		// 「還要休息幾小時才記得完」原版沒有印，玩家自己算。
		drawText(screen, a.campRestTimeLine(), dialogueLeft, dialogueFirstRow, accent)
		return
	case campStageSpeed:
		// `Game Speed = <n> (0=fastest 9=slowest)`（overlay-15 `1A00h` ＋ 值
		// ＋ `1A0Eh`），與那一列指令是分開的兩塊。
		drawText(screen, a.campSpeedLine(), dialogueLeft, dialogueFirstRow, accent)
		return
	}
	if a.campMessage != "" {
		// `The party makes camp...` 在原版是第二行（native 144..150）。
		drawText(screen, a.campMessage, dialogueLeft, dialogueFirstRow+dialoguePitch, foreground)
	}
}

// campFireTicksPerFrame 是營火多久換一張。
//
// 原版的換張門檻在選單元件的等鍵迴圈裡算（overlay-26 `02A0h` 一帶）：
// `門檻 = 那一張的延遲 ÷ 7` 個 BIOS tick，而延遲就是 PIC 每張前面那 4 bytes
// （spec 117 只說「4 bytes 前綴」，沒說是什麼）。**營火兩張的延遲都是 2**，
// 2÷7 ＝ 0，所以是「每過一個 BIOS tick 就換」——18.2 Hz。remake 跑 60 fps，
// 三個影格 50 ms 最接近那一個 tick 的 54.9 ms。
//
// 除法而不是乘法是從值反推的：同一個容器裡的船（`PIC3.DAX` 區塊 41）延遲是
// 20／15，除以 7 是兩個 tick（0.11 秒，船在晃），乘以 7 會變成七秒一格。
const campFireTicksPerFrame = 3

// campFireImage 是紮營時蓋在視野那一框上的營火（spec 135）。
func (a *app) campFireImage() *ebiten.Image {
	if !a.campOpen {
		return nil
	}
	if len(a.campFire) != 0 {
		return a.campFire[a.campFireTick/campFireTicksPerFrame%len(a.campFire)]
	}
	if a.loadCampFire == nil {
		return nil
	}
	frames, err := a.loadCampFire(uint8(a.spawn.Map.Archive))
	if err != nil || len(frames) == 0 {
		// 載不出來就留著視野；不要因為缺一張圖就讓紮營進不去。
		a.loadCampFire = nil
		return nil
	}
	a.campFire = frames
	return a.campFire[a.campFireTick/campFireTicksPerFrame%len(a.campFire)]
}

// clearCampFire 換主題時要重畫一次（配色是換主題換的）。
func (a *app) clearCampFire() { a.campFire = nil }

// poolFirstPersonStageFill 是第一人稱視野的三段背景。
//
// **幾何量過了**（2026-09-05，`docs/reference/original-dos/adventure/`）：
// 原版那一框的內部在 320×200 座標是 `(24,24)` 起的 88×88，與這裡的
// `StageInset` 逐格相同。
//
// **顏色也量過了**：碼頭那一張的天空是 EGA 11（`85,255,255` 青），
// 地面是 EGA 6（`170,85,0` 棕）。spec 047 原本寫的「保留既有 EGA blue／
// dark-gray」是接手時帶過來的佔位值，不是從原版讀的——對拍一看就差很多。
//
// **這兩個索引的來源仍未讀**：原版一定是逐區決定的（地城不會有青色天空），
// 而那份資料在哪還沒找到。所以這裡是「對著唯一畫得出來的那一張圖量出來的」，
// 不是「從原版資料讀出來的」；接第二張圖之前不要把它當通用值。
const (
	poolSkyPaletteIndex    = 11
	poolGroundPaletteIndex = 6
)

func poolFirstPersonStageFill() (viewport.StageInsetFill, error) {
	// 三段的邊界照共用 engine 的 `BuildBackground`（它是從原版
	// `Draw3dWorldBackground` 解出來的）：天空 44 列、中間**兩列黑**、
	// 地面從第 70 列起 42 列。先前中間那一段寫 0 列、地面從 68 起，
	// 對拍時那兩列會多算成地面色。
	background := viewport.Background{SkyPalette: poolSkyPaletteIndex, Rects: []viewport.BackgroundRect{
		{X: 24, Y: 24, Width: 88, Height: 44, PaletteIndex: poolSkyPaletteIndex},
		{X: 24, Y: 68, Width: 88, Height: 2, PaletteIndex: 0},
		{X: 24, Y: 70, Width: 88, Height: 42, PaletteIndex: poolGroundPaletteIndex},
	}}
	return viewport.FillBackgroundToStageInset(background, viewport.StageInset{X: 24, Y: 24, Width: 88, Height: 88, WallTop: 40})
}

func drawPoolStageRects(screen *ebiten.Image, rectangles []viewport.BackgroundRect, viewLeft, viewTop int, palette [16]color.RGBA) {
	for _, rectangle := range rectangles {
		shade := palette[rectangle.PaletteIndex]
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

// 對話框的上緣停在第一人稱框**整個**下面（`dialogueTop`）。
//
// 原版的文字框在那一框下面，兩者不重疊——`03-rolf-approach.png` 的半身像
// 是整張看得見的。remake 原本把框畫在 `198`，剛好切掉視野下面 64 個像素，
// 症狀在只畫牆片時看起來像「牆片本來就矮」，換成半身像之後才明顯：Rolf 被
// 攔腰截斷。
//
// **視野的 176×176 不是那一框的全部。** 視野是 `(48,70)` 起的 176×176，
// 下緣 246；但外面還圍著一圈繩索（`drawRopeBox` 從 `(32,54)` 起 13×13 個
// tile，每個 16 像素），所以那一框真正的下緣在 **262**。照視野的下緣排會讓
// 對話框的上框線從繩索中間切過去——畫面上看起來是「視野框沒有下緣」，
// 而不是「有東西蓋住它」，所以不容易發現。
//
// 264 也接近原版：原版那一幕（`31-b`）的文字框上緣繩索在 native 129..134
//（logical 258..269），下緣在 185..190（logical 370..381）。
//
// 下緣停在 370：底部那一列說明文字的字頂在 372（基線 386、ascent 14），
// 畫到 372 以下框線就會壓在字上。264..370 這 106 像素放五列訊息：首列基線
// 282（離上框線兩像素，不然中文字的頂會壓在框線上），末列字底 346。
const (
	dialogueTop      = 264
	dialogueBottom   = 370
	dialogueLines    = 5
	dialogueFirstRow = 282
	dialogueLeft     = 52
	dialoguePitch    = 16
)

// dialogueVisible 說這一格會不會畫出對話框。
//
// **功能鍵列要靠它讓位**：原版在導覽跑的時候，最下面那一列只有
// 「按 RETURN 繼續」，沒有功能鍵列。
//（`footerBaseline` 移到框外之後兩條不再相疊，但這一條的理由本來就是對版面，
// 不是閃避。）
// panelOpen 回報「有一頁面板蓋在冒險畫面上」。這幾個在 `Update` 裡都會
// 提早返回，所以指令列那一列的鍵此時按不到。
func (a *app) panelOpen() bool {
	return a.journalOpen || a.equipmentOpen || a.spellsOpen || a.shopActive ||
		a.guideOpen || a.fieldCastOpen || a.viewSheetOpen ||
		a.spriteOpen
}

func (a *app) dialogueVisible() bool {
	switch {
	case a.introWaiting && a.initialEvent != nil:
		return true
	case a.tourActive && a.tourPage >= 0 && a.initialEvent != nil &&
		a.tourStep >= 0 && a.tourStep < len(a.initialEvent.Tour):
		return a.tourPage < len(a.initialEvent.Tour[a.tourStep].Messages)
	case a.cellEventPending && a.eventText != "":
		return true
	case a.cellTextSticky && a.eventText != "":
		return true
	}
	return false
}

// showDialogue 畫對話框，但**面板蓋上來時整段不畫**。
//
// 框本身會被面板蓋住，框外那一列提示（`footerBaseline`）卻不會——只擋框、
// 不擋提示的話，冒險畫面的「按 RETURN 繼續」會與手冊自己的頁尾疊在同一行，
// 看起來像字型壞了。
func (a *app) showDialogue(screen *ebiten.Image, message, label string, foreground, accent color.Color) {
	if a.panelOpen() {
		return
	}
	drawDialogue(screen, message, label, a.journalCuePrompt(), foreground, accent)
}

// drawDialogueFrame 只畫那個框。探索施法的選單也用它——原版在開清單之前
// 呼叫 `sub_1500`／`sub_1638` 開一個框（spec 119 的 `0497h`），冒險畫面
// 留在框外面，不是換成整頁選單。
func drawDialogueFrame(screen *ebiten.Image, accent color.Color) {
	panel := ebiten.NewImage(560, dialogueBottom-dialogueTop)
	panel.Fill(color.RGBA{0, 0, 0, 255})
	screen.DrawImage(panel, &ebiten.DrawImageOptions{GeoM: translated(40, dialogueTop)})
	for x := 40; x < 600; x++ {
		screen.Set(x, dialogueTop, accent)
		screen.Set(x, dialogueBottom, accent)
	}
	for y := dialogueTop; y <= dialogueBottom; y++ {
		screen.Set(40, y, accent)
		screen.Set(599, y, accent)
	}
}

func drawDialogue(screen *ebiten.Image, message, label, cue string, foreground, accent color.Color) {
	drawDialogueFrame(screen, accent)
	lines := wrapDisplay(message, 68)
	for index, line := range lines {
		if index >= dialogueLines {
			break
		}
		drawText(screen, line, dialogueLeft, dialogueFirstRow+index*dialoguePitch, foreground)
	}
	// 手冊提示畫在**框裡**最後一行，不是框外那一列（spec 132）。框外那一列
	// 是原版的指令列與「按 RETURN 繼續」的位置——提示放那裡會把它擠掉，
	// 而原版走到市政廳外時那一列印的是 `AREA CAST VIEW ENCAMP SEARCH LOOK`。
	if cue != "" {
		row := len(lines)
		if row >= dialogueLines {
			row = dialogueLines - 1
		}
		drawText(screen, cue, dialogueLeft, dialogueFirstRow+row*dialoguePitch, accent)
	}
	// 提示畫在**繩索框外面**那一列，跟原版同一個位置——原版導覽那一幕
	// （`docs/audit/dos-parity-sample.md` 的 31-b）文字框裡只有台詞，
	// `PRESS <ENTER>/<RETURN> TO CONTINUE` 在框下緣底下那一條。
	// 畫在框裡的話，中文台詞只有兩行時框內會空一大塊，而框外整條是空的。
	drawText(screen, label, dialogueLeft, footerBaseline, accent)
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

func initialWallStamps(grid geometry.Grid, piece graphics.PieceSet, spawn gamepack.Spawn,
	band0 graphics.Picture) ([]graphics.WallStamp, error) {
	view, err := viewport.TraverseWallViewWrapped(grid, uint8(spawn.Direction()), int(spawn.X), int(spawn.Y))
	if err != nil {
		return nil, err
	}
	return resolveWallStamps(piece, view, band0), nil
}

func drawCreation(screen *ebiten.Image, a *app, foreground, accent color.Color) {
	a.drawFrame(screen, foreground, accent)
	if a.flow.Stage == creation.StageRoll {
		if a.rolled == nil {
			drawText(screen, a.text(msgRolling), sheetLeft, sheetLine1, foreground)
			return
		}
		drawCharacterSheet(screen, a, foreground, accent)
		drawText(screen, a.text(msgKeepCharacter), 0, footerBaseline, accent)
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
		// 原版是**在整張人物資料頁上**換頭與身體，不是另開一頁：底下框外那一
		// 列是 `HEAD BODY KEEP`，右上角的肖像跟著選擇換（基準畫面
		// `24-Return`）。`drawCharacterSheet` 畫的肖像取自 `a.portrait`，
		// 而換頭換身體改的就是它，所以這裡不必自己再貼一張。
		drawCharacterSheet(screen, a, foreground, accent)
		drawText(screen, a.text(msgPortraitTitle), 0, footerBaseline, accent)
		return
	}
	if a.flow.Stage == creation.StageIcon {
		drawIconEditor(screen, a, foreground, accent)
		return
	}
	if a.flow.Stage == creation.StageIconConfirm {
		// 原版**保留那四格**，只把框外那一列換成問句（基準畫面 `26-e`：
		// `IS THIS ICON OK? YES NO`）。先前這一頁只有問句，四格整組不見。
		drawIconPreviews(screen, a, accent)
		drawText(screen, a.text(msgIconConfirm)+"  "+a.text(msgIconConfirmYes)+
			"  "+a.text(msgIconConfirmNo), 0, footerBaseline, accent)
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

func drawHelp(screen *ebiten.Image, a *app, background, foreground, accent color.Color, provenance []string) {
	for y := 54; y < 340; y++ {
		for x := 72; x < 568; x++ {
			screen.Set(x, y, background)
		}
	}
	drawText(screen, a.text(msgHelpTitle), 292, 80, accent)
	// 每一行一句，換行分隔——翻譯要換行數或併行時不用改程式。
	lines := strings.Split(a.text(msgHelpKeys), "\n")
	for index, line := range lines {
		drawText(screen, line, 104, 100+index*22, foreground)
	}
	for index, line := range adventureCommandHelp() {
		drawText(screen, line, 104, 100+(len(lines)+index)*22, foreground)
	}
	base := len(lines) + len(adventureCommandHelp()) + 1
	for index, line := range provenance {
		drawText(screen, line, 104, 100+(base+index)*22, accent)
	}
}

// drawnText 是測試用的接縫：非 nil 時每畫一段字就記一筆（內容與位置）。
//
// 離線讀不到畫面像素——`(*ebiten.Image).At` 在遊戲迴圈外會 panic，所以
// 「這一塊到底有沒有畫」在測試裡只能靠這條路問。正常執行時它是 nil，
// 一次 nil 比較而已。
var drawnText func(value string, x, y int)

// uiShadowFace 是 `uiFace` 的「外圈」面（見 `drawText`）。載不出倚天字型時
// 是 nil，那時就只畫字身。
var uiShadowFace *etenfont.Face

func drawText(screen *ebiten.Image, value string, x, y int, ink color.Color) {
	if drawnText != nil {
		drawnText(value, x, y)
	}
	display := strings.ToUpper(displayText(value))
	// 先把加厚的那一圈用同色系暗一階畫上去，再用主色畫字身。厚度從顏色拿，
	// 不從解析度拿——直接把字身膨脹一格會同時吃掉字內的縫隙（見
	// `etenfont.rasterGlyph`）。
	if uiShadowFace != nil {
		if dim, ok := dimmerInk(ink); ok {
			text.Draw(screen, display, uiShadowFace, x, y, dim)
		}
	}
	text.Draw(screen, display, uiFace, x, y, ink)
}

// dimmerInk 是同一個色相暗一階的那一色。EGA 的十六色就是「暗八色 ＋ 亮八色」，
// 亮色的索引正好是暗色加八，所以配對是查表得來的，不是調出來的。
func dimmerInk(ink color.Color) (color.Color, bool) {
	r, g, b, a := ink.RGBA()
	if a == 0 {
		return nil, false
	}
	for index := 8; index < len(graphics.EGA16); index++ {
		candidate := graphics.EGA16[index]
		cr, cg, cb, _ := candidate.RGBA()
		if cr == r && cg == g && cb == b {
			return graphics.EGA16[index-8], true
		}
	}
	return nil, false
}

// footerBaseline 是畫面最底下那一列文字的基線。
//
// **在繩索框外面，跟原版一樣。** 原版每一頁的指令／提示列都畫在 native
// y 192..198，也就是外框下緣（184..191）**下面**那一條——`CHOOSE A FUNCTION`、
// `KEEP THIS CHARACTER? YES NO`、`AREA CAST VIEW ENCAMP SEARCH LOOK` 都是
//（用 dosgolem 逐列量的，`docs/audit/dos-parity-sample.md`）。
//
// 換算到 remake 的 640×400：外框下緣那一列 tile 佔 368..383（tile row 23，
// spec 123），底下 384..399 整條是空的。倚天字型 ascent 14／descent 1，
// 所以 398 的字剛好落在 384..399——**那一條就是原版留給這一列的位置**。
//
// 先前是 366（框內）。那時的註解說「比它大的基線會畫進框裡」——在那個版面下
// 沒錯，但真正的解法是整列移到框外，不是把字往上擠。
const footerBaseline = 398

func (a *app) Layout(_, _ int) (int, int) { return logicalWidth, logicalHeight }

func main() {
	zipPath := flag.String("zip", "Pool of Radiance (1988).zip", "DOS source ZIP used as local asset source")
	langFlag := flag.String("lang", "auto", "UI language: en, zh, or auto (zh when an ETen font is supplied)")
	etenFont := flag.String("eten-font", "", "ETen STDFONT.15 path; the 16x15 Han glyphs the Chinese UI needs")
	etenSymbol := flag.String("eten-symbol-font", "", "optional ETen SPCFONT.15 path for full-width punctuation")
	etenASCII := flag.String("eten-ascii-font", "", "optional ETen ASCFONT.15 path; defaults to ascfont.15 beside -eten-font")
	savePath := flag.String("save", defaultStatePath(), "remake save file; defaults to the OS user config directory")
	// 骰子種子。預設跟著時間跑（每一局不一樣），給值就固定——
	// 對拍截圖要的是「同一份程式碼拍出同一張圖」，擲值每次不同的話
	// 連 HP 都會變，雜湊就永遠對不上，那份清冊也就證不了東西。
	diceSeed := flag.Int64("dice-seed", 0, "fixed dice seed; 0 keeps the time-based seed")
	// 配樂目錄。空字串時找執行檔旁邊的 music/；那個目錄只有本機的 full-local
	// 發行包才有，可散布的包不帶音訊（spec 128）。
	musicDir := flag.String("music-dir", "", "directory holding the OGG music; defaults to music/ beside the executable")
	// 原版（C64，實跑量過）只有標題有音樂；地圖與戰鬥是靜的。full 會連那兩處
	// 也放，那是 remake 自己加的（spec 128）。
	musicMode := flag.String("music-mode", "original", "music cues: original (title only, as measured) or full")
	// 自動截圖用：每次畫面換了就把識別字寫進這個檔（screen_state.go）。
	// 空字串（預設）什麼都不寫。
	screenState := flag.String("screen-state", "", "write the current screen identifier to this file; used by the capture scripts")
	flag.Parse()
	uiLanguage, face, err := resolveUILanguage(*langFlag, *etenFont, *etenSymbol, *etenASCII)
	if err != nil {
		log.Fatal(err)
	}
	uiFace = face
	if eten, ok := face.(*etenfont.Face); ok {
		uiShadowFace = eten.ShadowFace()
	}
	catalogue, err := gameTextFor(uiLanguage)
	if err != nil {
		log.Fatal(err)
	}
	monsters, err := monsterTextFor(uiLanguage)
	if err != nil {
		log.Fatal(err)
	}
	game, err := newApp(*zipPath, *savePath)
	if err != nil {
		log.Fatal(err)
	}
	if *diceSeed != 0 {
		game.roller = diceRoller{random: rand.New(rand.NewSource(*diceSeed))}
	}
	game.screenStatePath = defaultScreenStatePath(*screenState)
	game.language, game.gameText, game.monsterText = uiLanguage, catalogue, monsters
	// 遊戲內攻略（`F3`）。載不進來就讓它是 nil——那時 F3 會說「這張地圖還沒有
	// 建過攻略點」，不是把遊戲收掉。
	if guideCatalogue, err := guideFor(uiLanguage); err == nil {
		game.guide = guideCatalogue
	} else {
		fmt.Fprintln(os.Stderr, "guide:", err)
	}
	dir := *musicDir
	if dir == "" {
		dir = defaultMusicDir()
	}
	if dir != "" && !audioDeviceLikelyAvailable() {
		fmt.Fprintln(os.Stderr, "music: 找不到音訊裝置，這一次不放音樂")
		dir = ""
	}
	// 音樂開不起來不該讓遊戲開不起來：報一行就繼續，安靜地跑。
	if player, err := music.NewPlayer(dir); err != nil {
		fmt.Fprintln(os.Stderr, "music:", err)
	} else {
		switch music.Mode(*musicMode) {
		case music.ModeOriginal, music.ModeFull:
			player.SetMode(music.Mode(*musicMode))
		default:
			log.Fatalf("-music-mode 只能是 original 或 full，收到 %q", *musicMode)
		}
		game.musicPlayer = player
		defer player.Close()
	}
	// **整數倍**：邏輯畫布 640×400，視窗是它的兩倍。原版的字是 8×15／16×15
	// 的點陣，非整數倍放大會把某些像素行複製、某些丟掉——筆畫密的漢字
	//（「鈕」「繼」那種）看起來就像糊在一起。先前是 960×600（1.5 倍），
	// 那一列「按 <RETURN> 或按鈕繼續」在截圖上疊成一團，而同一個字型在
	// 邏輯解析度下畫出來是清楚的。
	ebiten.SetWindowSize(1280, 800)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowTitle("Pool of Radiance Remake")
	if err := ebiten.RunGame(game); err != nil && err != ebiten.Termination {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
