package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"

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

// 紮營有兩層，原版就是兩列指令（spec 135）。
type campStage int

const (
	// campStageMenu 是 `CAMP: SAVE VIEW MAGIC REST ALTER EXIT`。
	campStageMenu campStage = iota
	// campStageRest 是按下 `REST` 之後的
	// `REST  DAYS HOURS MINS  INC DEC  EXIT`。
	campStageRest
)

// openCamp 進紮營。
func (a *app) openCamp() {
	if len(a.state.Party) == 0 {
		a.statusLine = a.text(msgCampNeedsParty)
		return
	}
	a.campOpen, a.campStage = true, campStageMenu
	a.campMessage = a.text(msgCampMakesCamp)
}

// closeCamp 離開紮營。
func (a *app) closeCamp() {
	a.campOpen, a.campStage, a.campMessage = false, campStageMenu, ""
}

// campInput 處理紮營那兩列的按鍵。
//
// **兩層的字母是分開的**：第一層的 `M` 是 MAGIC，第二層的 `M` 是 MINS——
// 原版就是靠分層讓同一個鍵在兩處有不同意思。排時間那一層另外照
// overlay-20 `06E0h` 把方向鍵對應過去：上＝I、下＝D、左右換欄。
func (a *app) campInput() error {
	if a.campStage == campStageRest {
		return a.campRestInput()
	}
	switch {
	case a.justPressed(ebiten.KeyEscape), a.justPressed(ebiten.KeyE):
		a.closeCamp()
	case a.justPressed(ebiten.KeyR):
		a.campStage = campStageRest
		a.campMessage = ""
	case a.justPressed(ebiten.KeyV):
		a.openViewSheet()
	case a.justPressed(ebiten.KeyM):
		return a.openSpells()
	case a.justPressed(ebiten.KeyS), a.justPressed(ebiten.KeyA):
		// `SAVE` 與 `ALTER` 按下去做什麼還沒讀出來（spec 135 的 OPEN）。
		// **仍然列在那一列上**——不列的話玩家看到的指令列就與原版不同，
		// 而那正是這一頁要修的東西。
		a.campMessage = a.text(msgCampCommandUnread)
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
