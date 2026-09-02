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
// **時間還沒接**：原版讓玩家挑天／時／分，而且會被打斷（`Stop Resting?`）。
// 這裡先做「休息到記完為止」，需要多久算得出來（各法術等級的總和），
// 但不模擬時間流逝，也沒有遭遇打斷。原版挑時間那一段的界面還沒讀。

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
func (a *app) campInput() error {
	options := a.campOptionLabels()
	switch {
	case a.justPressed(ebiten.KeyEscape):
		a.campOpen = false
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

// restParty 休息：把待記的法術記完，並治好整隊。
func (a *app) restParty() {
	hours := 0
	memorised := 0
	for index := range a.state.Party {
		member := &a.state.Party[index]
		hours += gamepack.PendingMemorisationTime(member.Memorised, a.spellParameters)
		memorised += gamepack.CompletePendingMemorisation(member.Memorised)
		// 「The Whole Party Is Healed」：休息完整隊回滿。
		member.CurrentHP = member.MaxHP
		syncTrainedLibraryCharacter(&a.state, *member)
	}
	a.campOpen = false
	if memorised == 0 {
		a.statusLine = a.text(msgCampHealedOnly)
		return
	}
	a.statusLine = fmt.Sprintf(a.text(msgCampRested), memorised, hours)
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
