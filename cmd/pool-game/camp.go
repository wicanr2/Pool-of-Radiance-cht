package main

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/creation"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 紮營（原版的 overlay-20）。畫面上的字串是 `Rest Time:`、
// `Rest daYs Hours Mins Inc Dec Exit`、`The Whole Party Is Healed`、
// `has memorized`、`Stop Resting?  The Party is rudely interrupted!`。
//
// 休息做兩件事：把選好但還沒記完的法術記完（`0945h` 的 `subb $80h`），
// 以及治好整隊。
//
// 挑時間那一段照原版接了（spec 114）：天／時／分三欄，Y／H／M 選欄、
// I／D 增減，分鐘一次五分；進位與夾限由 `gamepack.RestDuration` 處理。
// 休息的效果也照原版與說明書 p.29：**每二十四小時每人回一點生命力**，
// 而法術要記完得休息夠久（各法術等級的總和，單位是小時）。
//
// **還沒接**：休息被打斷。原版的順序是「打斷 → 休息回傳 1 → 呼叫端跑 ECL
// 的紮營入口」，貧民區跑到的是城市守衛那一段（ECL3／block 0 entry 3）；
// overlay-20 自己印的 `The Party is rudely interrupted!` 只是其中一條路。
// 缺的是判定用的那兩個參數從哪裡來，見 spec 114 的 OPEN。

// 紮營的選單樹照原版分層（spec 135）。每一層就是最下面那一列指令換一組，
// 上一層的畫面不動。
type campStage int

const (
	// campStageMenu 是 `CAMP: SAVE VIEW MAGIC REST ALTER EXIT`。
	campStageMenu campStage = iota
	// campStageRest 是 `REST  DAYS HOURS MINS  INC DEC  EXIT`。
	campStageRest
	// campStageAlter 是 `Alter: ORDER DROP SPEED ICON PICS EXIT`。
	campStageAlter
	// campStageOrderSelect／campStageOrderPlace 是 `Party Order:` 的兩步，
	// 原版的兩個字串就是 `Select Exit` 與 `Place Exit`（`DS:0598h` 與
	// `DS:05C1h`，每筆 41 bytes）。
	campStageOrderSelect
	campStageOrderPlace
	// campStageSpeed 是 `Game Speed: FASTER SLOWER EXIT`。
	campStageSpeed
	// campStagePics 是 `MONSTERS ON/OFF PORTRAITS ON/OFF EXIT`。
	campStagePics
	// campStageQuitConfirm 是 SAVE 存完之後那一句 `Quit TO DOS`。
	campStageQuitConfirm
	// campStageDropConfirm 是 ALTER→DROP 的 ` Drop from party? `。
	campStageDropConfirm
	// campStageIcon 是 ALTER→ICON：對目前角色開戰鬥造形編輯器
	//（原版 overlay-16 entry 4，與建角走到那一步是同一支）。
	campStageIcon
)

// campSpeedFastest／campSpeedSlowest 是遊戲速度的兩端。原版把值放在
// `ds:4943h`，畫面上寫 `(0=fastest 9=slowest)`，而 `Faster` 只在值大於 0 時
// 列出來、`Slower` 只在小於 9 時列出來（overlay-15 `1AD9h`／`1AF1h`）。
const (
	campSpeedFastest = 0
	campSpeedSlowest = 9
)

// openCamp 進紮營。
func (a *app) openCamp() {
	if len(a.state.Party) == 0 {
		a.statusLine = a.text(msgCampNeedsParty)
		return
	}
	a.campOpen, a.campStage = true, campStageMenu
	a.campMessage = a.text(msgCampMakesCamp)
	if a.campMember >= len(a.state.Party) {
		a.campMember = 0
	}
}

// closeCamp 離開紮營。
func (a *app) closeCamp() {
	a.campOpen, a.campStage, a.campMessage = false, campStageMenu, ""
	a.campOrderPick = -1
}

// campMemberName 是目前角色的名字。
//
// **原版的紮營有「目前角色」**（`ds:5CF0h`／`5CF2h` 那個遠指標）：紮營那一列
// 的數字鍵 1..6 就是換他，而 `VIEW`、`DROP`、`ICON` 都對他生效——與 `C)AST`
// 不挑人是同一個機制（spec 119）。
func (a *app) campMemberName() string {
	if a.campMember < 0 || a.campMember >= len(a.state.Party) {
		return ""
	}
	return strings.TrimSpace(a.state.Party[a.campMember].Name)
}

// campSelectMember 是那一列的數字鍵。原版的選單元件（overlay-26 entry 3）
// 收到 1..6 時回一個旗標，呼叫端就把目前角色換成那一個。
func (a *app) campSelectMember() bool {
	for index, key := range []ebiten.Key{ebiten.KeyDigit1, ebiten.KeyDigit2,
		ebiten.KeyDigit3, ebiten.KeyDigit4, ebiten.KeyDigit5, ebiten.KeyDigit6} {
		if !a.justPressed(key) {
			continue
		}
		if index >= len(a.state.Party) {
			return true
		}
		a.campMember = index
		a.campMessage = fmt.Sprintf(a.text(msgCampCurrentMember), a.campMemberName())
		return true
	}
	return false
}

// campInput 處理紮營那兩列的按鍵。
//
// **兩層的字母是分開的**：第一層的 `M` 是 MAGIC，第二層的 `M` 是 MINS——
// 原版就是靠分層讓同一個鍵在兩處有不同意思。排時間那一層另外照
// overlay-20 `06E0h` 把方向鍵對應過去：上＝I、下＝D、左右換欄。
func (a *app) campInput() error {
	switch a.campStage {
	case campStageRest:
		return a.campRestInput()
	case campStageAlter:
		return a.campAlterInput()
	case campStageOrderSelect, campStageOrderPlace:
		return a.campOrderInput()
	case campStageSpeed:
		return a.campSpeedInput()
	case campStagePics:
		return a.campPicsInput()
	case campStageQuitConfirm:
		return a.campQuitConfirmInput()
	case campStageDropConfirm:
		return a.campDropConfirmInput()
	case campStageIcon:
		return a.iconMenuInput()
	}
	if a.campSelectMember() {
		return nil
	}
	switch {
	case a.justPressed(ebiten.KeyEscape), a.justPressed(ebiten.KeyE):
		a.closeCamp()
	case a.justPressed(ebiten.KeyR):
		a.campStage = campStageRest
		a.campMessage = ""
	case a.justPressed(ebiten.KeyV):
		// 原版走 overlay-19 entry 5，也就是人物資料頁（spec 119／130）。
		a.openViewSheet()
	case a.justPressed(ebiten.KeyM):
		return a.openSpells()
	case a.justPressed(ebiten.KeyA):
		a.campStage = campStageAlter
		a.campMessage = ""
	case a.justPressed(ebiten.KeyS):
		// 原版先存檔（overlay-17 entry 11），存完才問 `Quit TO DOS`。
		if err := a.saveCurrentGame(); err != nil {
			return err
		}
		a.campStage = campStageQuitConfirm
		a.campMessage = a.text(msgCampQuitToDOS)
	}
	return nil
}

// campQuitConfirmInput 是存完之後那一句 `Quit TO DOS`（overlay-15 `1E38h`）。
// 原版按 `Y` 就 `call far 26Bh:0` 回 DOS，其餘回紮營。
func (a *app) campQuitConfirmInput() error {
	switch {
	case a.justPressed(ebiten.KeyY):
		return a.exitToDOS()
	case a.justPressed(ebiten.KeyN), a.justPressed(ebiten.KeyEscape),
		a.justPressed(ebiten.KeyEnter):
		a.campStage = campStageMenu
		a.campMessage = a.text(msgCampSaved)
	}
	return nil
}

// campAlterInput 是 `Alter: ORDER DROP SPEED ICON PICS EXIT`
//（前綴 overlay-15 `1BE0h`，那一列是 `DS:056Eh`）。
func (a *app) campAlterInput() error {
	if a.campSelectMember() {
		return nil
	}
	switch {
	case a.justPressed(ebiten.KeyEscape), a.justPressed(ebiten.KeyE):
		a.campStage = campStageMenu
		a.campMessage = ""
	case a.justPressed(ebiten.KeyO):
		a.campStage, a.campOrderPick = campStageOrderSelect, -1
		a.campMessage = ""
	case a.justPressed(ebiten.KeyD):
		// 原版丟的是**目前角色**，不另外挑人（同 `C)AST`，spec 119）。
		// 先印一句 `<名字> will be gone`，再問 ` Drop from party? `。
		if a.campMemberName() == "" {
			return nil
		}
		a.campStage = campStageDropConfirm
		a.campMessage = a.campMemberName() + a.text(msgCampDropWillBeGone)
	case a.justPressed(ebiten.KeyS):
		a.campStage = campStageSpeed
		a.campMessage = ""
	case a.justPressed(ebiten.KeyP):
		a.campStage = campStagePics
		a.campMessage = ""
	case a.justPressed(ebiten.KeyI):
		return a.beginCampIconEdit()
	}
	return nil
}

// beginCampIconEdit 對目前角色開戰鬥造形編輯器（原版 overlay-16 entry 4）。
//
// 編輯器本身讀寫的是 `creation.Flow`，因為建角那一步就是這樣接的。這裡把
// 角色身上的四個欄位**載回 flow**、編完再寫回去，離開時把 flow 還原成進來
// 之前的樣子——冒險途中的 flow 本來就不該有內容，借用它不能留下痕跡。
//
// 種族也要一起載回去：`UsesIconSizeMenu` 看的是 flow 的種族，載錯的話
// 小種族的 `SIZE` 那一項會消失（或反過來冒出來）。
func (a *app) beginCampIconEdit() error {
	if a.campMember < 0 || a.campMember >= len(a.state.Party) {
		return nil
	}
	member := a.state.Party[a.campMember]
	a.campFlowBackup = a.flow
	a.flow = creation.NewFlow()
	if _, index, ok := findRaceIndex(member.RaceID); ok {
		a.flow.RaceIndex = index
	}
	a.flow.IconHead, a.flow.IconWeapon = member.IconHead, member.IconWeapon
	a.flow.IconSize, a.flow.IconColors = member.IconSize, member.IconColors
	a.flow.Stage = creation.StageIcon
	a.resetIconMenu()
	a.campStage = campStageIcon
	a.campMessage = ""
	// 載不出造形的圖就只是右邊那兩格空著，選單照樣能用——不要因為缺一張圖
	// 就讓玩家進不了這一層。
	if err := a.reloadIcons(); err != nil {
		a.statusLine = err.Error()
	}
	return nil
}

// finishCampIconEdit 把編好的造形寫回角色，並還原 flow。
func (a *app) finishCampIconEdit() error {
	if a.campMember >= 0 && a.campMember < len(a.state.Party) {
		member := &a.state.Party[a.campMember]
		member.IconHead, member.IconWeapon = a.flow.IconHead, a.flow.IconWeapon
		member.IconSize, member.IconColors = a.flow.IconSize, a.flow.IconColors
		syncTrainedLibraryCharacter(&a.state, *member)
	}
	a.flow = a.campFlowBackup
	a.campFlowBackup = creation.Flow{}
	a.resetIconMenu()
	a.campStage = campStageAlter
	a.campMessage = ""
	if err := a.reloadIcons(); err != nil {
		a.statusLine = err.Error()
	}
	return a.persistState(a.text(msgCampIconSaved))
}

// campDropConfirmInput 是 ` Drop from party? `（overlay-15 `1863h`）。
func (a *app) campDropConfirmInput() error {
	switch {
	case a.justPressed(ebiten.KeyY):
		name := a.campMemberName()
		// 原版依角色記錄 `+10Dh` 換收尾那一句：非 0 是
		// ` bids you farewell`，0 是 ` is dumped in a ditch`。那個欄位的
		// 語意還沒逐位元組確認，這裡讀成「還活著」（推論）。
		alive := a.campMember < len(a.state.Party) &&
			a.state.Party[a.campMember].CurrentHP > 0
		if err := a.dropMember(a.campMember); err != nil {
			return err
		}
		if a.campMember >= len(a.state.Party) {
			a.campMember = 0
		}
		farewell := a.text(msgCampDropDitch)
		if alive {
			farewell = a.text(msgCampDropFarewell)
		}
		a.campStage = campStageAlter
		a.campMessage = name + farewell
		if len(a.state.Party) == 0 {
			a.closeCamp()
		}
	case a.justPressed(ebiten.KeyN), a.justPressed(ebiten.KeyEscape),
		a.justPressed(ebiten.KeyEnter):
		// 答否是第三句：`<名字> Breathes A sigh of relief`
		//（overlay-15 `189Ch`，`19DAh` 那一支就是 `cmp al, 'Y'` 沒中時跳去的）。
		a.campStage = campStageAlter
		a.campMessage = a.campMemberName() + a.text(msgCampDropRelief)
	}
	return nil
}

// campOrderInput 是 `Party Order:` 的兩步：先 `Select` 挑一個人，
// 再 `Place` 指定他要排到第幾位。兩步都用數字鍵。
func (a *app) campOrderInput() error {
	if a.justPressed(ebiten.KeyEscape) || a.justPressed(ebiten.KeyE) {
		a.campStage, a.campOrderPick = campStageAlter, -1
		a.campMessage = ""
		return nil
	}
	for index, key := range []ebiten.Key{ebiten.KeyDigit1, ebiten.KeyDigit2,
		ebiten.KeyDigit3, ebiten.KeyDigit4, ebiten.KeyDigit5, ebiten.KeyDigit6} {
		if !a.justPressed(key) || index >= len(a.state.Party) {
			continue
		}
		if a.campStage == campStageOrderSelect {
			a.campOrderPick = index
			a.campStage = campStageOrderPlace
			a.campMessage = fmt.Sprintf(a.text(msgCampOrderPicked),
				strings.TrimSpace(a.state.Party[index].Name))
			return nil
		}
		a.moveMember(a.campOrderPick, index)
		a.campStage, a.campOrderPick = campStageOrderSelect, -1
		return a.persistState(a.text(msgCampOrderMoved))
	}
	return nil
}

// moveMember 把隊伍第 from 位的人挪到第 to 位，其餘依序往後遞補。
func (a *app) moveMember(from, to int) {
	if from < 0 || from >= len(a.state.Party) || to < 0 || to >= len(a.state.Party) ||
		from == to {
		return
	}
	member := a.state.Party[from]
	rest := append(a.state.Party[:from:from], a.state.Party[from+1:]...)
	moved := make([]poolsave.Character, 0, len(a.state.Party))
	moved = append(moved, rest[:to]...)
	moved = append(moved, member)
	moved = append(moved, rest[to:]...)
	a.state.Party = moved
	a.campMember = to
}

// campSpeedInput 是 `Game Speed: FASTER SLOWER EXIT`。原版把值放在
// `ds:4943h`，`0=fastest 9=slowest`。
func (a *app) campSpeedInput() error {
	switch {
	case a.justPressed(ebiten.KeyEscape), a.justPressed(ebiten.KeyE):
		a.campStage = campStageAlter
		a.campMessage = ""
	case a.justPressed(ebiten.KeyF):
		if a.gameSpeed > campSpeedFastest {
			a.gameSpeed--
		}
	case a.justPressed(ebiten.KeyS):
		if a.gameSpeed < campSpeedSlowest {
			a.gameSpeed++
		}
	}
	return nil
}

// campSpeedLine 是那一層上面那一行：`Game Speed = <n> (0=fastest 9=slowest)`
//（overlay-15 `1A00h` ＋ 值 ＋ `1A0Eh`）。
func (a *app) campSpeedLine() string {
	return fmt.Sprintf(a.text(msgCampSpeedLine), a.gameSpeed)
}

// campPicsInput 是 `MONSTERS ON/OFF PORTRAITS ON/OFF EXIT`：兩個開關，
// 原版存在 `ds:4957h`（怪物圖）與 `ds:4956h`（半身像）。
func (a *app) campPicsInput() error {
	switch {
	case a.justPressed(ebiten.KeyEscape), a.justPressed(ebiten.KeyE):
		a.campStage = campStageAlter
		a.campMessage = ""
	case a.justPressed(ebiten.KeyM):
		a.monsterPicsHidden = !a.monsterPicsHidden
	case a.justPressed(ebiten.KeyP):
		a.portraitsHidden = !a.portraitsHidden
	}
	return nil
}

// campRestInput 是排時間那一層。
func (a *app) campRestInput() error {
	switch {
	case a.justPressed(ebiten.KeyEscape), a.justPressed(ebiten.KeyE):
		a.campStage = campStageMenu
		a.campMessage = a.text(msgCampMakesCamp)
	case a.justPressed(ebiten.KeyY):
		a.restField = gamepack.RestFieldDays
	case a.justPressed(ebiten.KeyH):
		a.restField = gamepack.RestFieldHours
	case a.justPressed(ebiten.KeyM):
		a.restField = gamepack.RestFieldMinutes
	case a.justPressed(ebiten.KeyI), a.justPressed(ebiten.KeyArrowUp):
		a.restDuration = a.restDuration.Increase(a.restField)
	case a.justPressed(ebiten.KeyD), a.justPressed(ebiten.KeyArrowDown):
		a.restDuration = a.restDuration.Decrease(a.restField)
	case a.justPressed(ebiten.KeyArrowLeft):
		a.restField = a.restField.PreviousField()
	case a.justPressed(ebiten.KeyArrowRight):
		a.restField = a.restField.NextField()
	case a.justPressed(ebiten.KeyR):
		a.restParty()
	}
	return nil
}

// campRestTimeLine 是畫面上那一列休息時間。
func (a *app) campRestTimeLine() string {
	return fmt.Sprintf(a.text(msgCampRestTime),
		a.restDuration.Days(), a.restDuration.Hours(), a.restDuration.Minutes())
}

// restParty 休息選好的那段時間。
//
// 兩件事都跟時間長短有關，不是按一下就全好：
//   - **法術**：每個人身上還沒記完的要花「各法術等級的總和」小時
//     （overlay-20 entry 15 每小時把記錄 `+2Ch` 減一，歸零才算記完）。
//   - **生命力**：每滿二十四小時每人回一點（`0830h` 的 288 刻，
//     說明書 p.29 也是這樣寫）。原版的 `The Whole Party Is Healed`
//     就印在那一刻。
func (a *app) restParty() {
	// 打斷的兩個參數目前是 0／0，也就是**永遠不會被打斷**——這是已知缺口，
	// 不是與原版一致。原版在貧民區排兩小時，第五分鐘就被城市守衛趕起來
	//（`YOU ARE ROUSTED BY THE CITY WATCH…`，ECL3／block 0 entry 3）。
	// 全 36 顆 overlay 裡只有 overlay-07 `0244h` 寫這兩個欄位，而且是清成 0，
	// 所以一定還有第三個 writer 沒找到（spec 114 的 OPEN）。
	outcome := gamepack.SimulateRest(a.restDuration, gamepack.RestInterruption{}, a.rollDice)
	ticks := outcome.Ticks
	restedHours := ticks / gamepack.RestTicksPerHour
	healed := gamepack.RestHealing(ticks)

	memorised, needed := 0, 0
	for index := range a.state.Party {
		member := &a.state.Party[index]
		pending := gamepack.PendingMemorisationTime(member.Memorised, a.spellParameters)
		if pending > needed {
			needed = pending
		}
		if pending > 0 && restedHours >= pending {
			memorised += gamepack.CompletePendingMemorisation(member.Memorised)
		}
		if healed > 0 {
			member.CurrentHP += healed
			if member.CurrentHP > member.MaxHP {
				member.CurrentHP = member.MaxHP
			}
		}
		syncTrainedLibraryCharacter(&a.state, *member)
	}
	a.closeCamp()
	switch {
	case memorised > 0:
		a.statusLine = fmt.Sprintf(a.text(msgCampRested), memorised, restedHours)
	case needed > 0:
		a.statusLine = a.text(msgCampRestTooShort)
	case healed > 0:
		a.statusLine = fmt.Sprintf(a.text(msgCampHealedBy), healed)
	default:
		a.statusLine = a.text(msgCampHealedOnly)
	}
}


// 讓 poolsave 這個 import 在只用到型別時也成立。
var _ poolsave.Character
