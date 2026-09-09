package main

import (
	"fmt"
	"strings"

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

// openCamp 開紮營選單。
func (a *app) openCamp() {
	if len(a.state.Party) == 0 {
		a.statusLine = a.text(msgCampNeedsParty)
		return
	}
	a.campOpen, a.campCursor = true, 0
}

// campOptions 是紮營選單的項目，順序照原版的 `Rest ... Exit`。
func (a *app) campOptionLabels() []string {
	return []string{a.text(msgCampRest), a.text(msgCampMemorise), a.text(msgCampExit)}
}

// campInput 處理紮營選單的按鍵。
//
// 原版的選單列是 `Rest daYs Hours Mins Inc Dec Exit`（overlay-20 `069Fh`），
// 而 `06E0h` 的迴圈把方向鍵也對應過去：上＝I、下＝D、左右換欄。這裡照它接，
// 另外保留 remake 自己的上下選單游標——原版的「記憶法術」是紮營選單的另一項，
// 不是這一列的按鍵。
func (a *app) campInput() error {
	options := a.campOptionLabels()
	switch {
	case a.justPressed(ebiten.KeyEscape), a.justPressed(ebiten.KeyE):
		a.campOpen = false
	case a.justPressed(ebiten.KeyY):
		a.restField = gamepack.RestFieldDays
	case a.justPressed(ebiten.KeyH):
		a.restField = gamepack.RestFieldHours
	case a.justPressed(ebiten.KeyM):
		a.restField = gamepack.RestFieldMinutes
	case a.justPressed(ebiten.KeyI):
		a.restDuration = a.restDuration.Increase(a.restField)
	case a.justPressed(ebiten.KeyD):
		a.restDuration = a.restDuration.Decrease(a.restField)
	case a.justPressed(ebiten.KeyArrowLeft):
		a.restField = a.restField.PreviousField()
	case a.justPressed(ebiten.KeyArrowRight):
		a.restField = a.restField.NextField()
	case a.justPressed(ebiten.KeyR):
		a.restParty()
	case a.justPressed(ebiten.KeyArrowUp):
		a.campCursor = (a.campCursor + len(options) - 1) % len(options)
	case a.justPressed(ebiten.KeyArrowDown):
		a.campCursor = (a.campCursor + 1) % len(options)
	case a.justPressed(ebiten.KeyEnter), a.justPressed(ebiten.KeySpace):
		switch a.campCursor {
		case 0:
			a.restParty()
		case 1:
			a.campOpen = false
			return a.openSpells()
		default:
			a.campOpen = false
		}
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
	a.campOpen = false
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

// campPendingLine 是給玩家看的一行：誰還有幾條沒記完。
func (a *app) campPendingLine() string {
	parts := make([]string, 0, len(a.state.Party))
	for _, member := range a.state.Party {
		pending := 0
		for _, value := range member.Memorised {
			if value != 0 && !gamepack.MemorisedSpellIsReady(value) {
				pending++
			}
		}
		if pending > 0 {
			parts = append(parts, fmt.Sprintf("%s %d", strings.TrimSpace(member.Name), pending))
		}
	}
	if len(parts) == 0 {
		return a.text(msgCampNothingPending)
	}
	return fmt.Sprintf(a.text(msgCampPending), strings.Join(parts, "  "))
}

// 讓 poolsave 這個 import 在只用到型別時也成立。
var _ poolsave.Character
