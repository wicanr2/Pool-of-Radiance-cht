package main

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/font"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 隊伍面板的欄位位置。原版把它放在第一人稱框右邊，`NAME` 靠左、`AC` 與 `HP`
// 靠右（`docs/reference/original-dos/adventure/02-first-person-14-1-west.png`）。
const (
	partyPanelLeft      = 310
	partyPanelACRight   = 540
	partyPanelHPRight   = 598
	partyPanelHeaderRow = 106
	partyPanelFirstRow  = 130
	partyPanelRowHeight = 22
	partyPanelStatusRow = 262
)

// partyPanelRow 是面板上的一列。
type partyPanelRow struct {
	Name        string
	ArmourClass int
	HitPoints   int
}

// partyPanelRows 依隊伍順序算出面板要顯示的三欄。
//
// AC 是檯面值（`60 − 內部值`，spec 080）；算不出來的（NPC 帶自己的記錄、
// 物品型別表沒載入）就留原本的建角基礎值，不要顯示一個假的 0。
func (a *app) partyPanelRows() []partyPanelRow {
	rows := make([]partyPanelRow, 0, len(a.state.Party))
	for _, member := range a.state.Party {
		row := partyPanelRow{
			Name:        member.Name,
			ArmourClass: gamepack.ArmourClassScale - creationArmorClassInternal,
			HitPoints:   member.CurrentHP,
		}
		if internal, _, err := a.memberDefenceStats(member,
			creationArmorClassInternal, creationBaseMovement); err == nil {
			row.ArmourClass = gamepack.ArmourClassScale - internal
		}
		rows = append(rows, row)
	}
	return rows
}

// adventureStatusLine 是原版第一人稱畫面底下那一列：`14, 1 W 00:00`。
//
// 時鐘目前一定是 `00:00`——remake 還沒有走路推進時間的那一段（原版是
// `ECL` 時鐘，spec 114 只接了紮營休息）。原版剛開場也是 `00:00`，所以這一列
// 現在對得上，但它不是「算出來的」。
func (a *app) adventureStatusLine() string {
	return fmt.Sprintf("%d, %d %s 00:00", a.spawn.X, a.spawn.Y, facingLetter(a.spawn.Facing))
}

// facingLetter 把 spec 076 的 0 北 1 東 2 南 3 西換成原版狀態列的字母。
func facingLetter(facing uint8) string {
	switch facing & 3 {
	case 0:
		return "N"
	case 1:
		return "E"
	case 2:
		return "S"
	default:
		return "W"
	}
}

// drawPartyPanel 畫第一人稱畫面右邊那一塊。
func drawPartyPanel(screen *ebiten.Image, a *app, foreground, accent color.Color) {
	drawText(screen, "NAME", partyPanelLeft, partyPanelHeaderRow, accent)
	drawTextRight(screen, "AC", partyPanelACRight, partyPanelHeaderRow, accent)
	drawTextRight(screen, "HP", partyPanelHPRight, partyPanelHeaderRow, accent)
	for index, row := range a.partyPanelRows() {
		if index >= poolsave.PartyMaximum {
			break
		}
		y := partyPanelFirstRow + index*partyPanelRowHeight
		drawText(screen, row.Name, partyPanelLeft, y, foreground)
		drawTextRight(screen, fmt.Sprintf("%d", row.ArmourClass), partyPanelACRight, y, foreground)
		drawTextRight(screen, fmt.Sprintf("%d", row.HitPoints), partyPanelHPRight, y, foreground)
	}
	drawText(screen, a.adventureStatusLine(), partyPanelLeft, partyPanelStatusRow, foreground)
}

// drawTextRight 讓一段文字的右緣落在 right 上。
func drawTextRight(screen *ebiten.Image, value string, right, y int, ink color.Color) {
	width := font.MeasureString(uiFace, displayText(value)).Ceil()
	drawText(screen, value, right-width, y, ink)
}

// adventureProvenanceLines 是「這一格的資料是哪來的、現在做到哪」那幾列。
//
// 它們原本畫在第一人稱畫面右邊，但原版那個位置是隊伍面板；留在畫面上等於
// 多一塊原版沒有的東西。移到 F1 之後資訊沒有少，畫面才對得上。
func (a *app) adventureProvenanceLines() []string {
	lines := []string{
		fmt.Sprintf("GEO%d BLOCK %d  X %d  Y %d  FACING %d",
			a.spawn.Map.Archive, a.spawn.Map.BlockID, a.spawn.X, a.spawn.Y, a.spawn.Facing),
		"GEO / WALL SOURCE: EXACT",
		"VIEW TRAVERSAL: STRONG INFERENCE",
	}
	if a.introDone {
		lines = append(lines, "GEO WALK: ENABLED / EVENTS PENDING")
	} else {
		lines = append(lines, "MOVE POLICY: PENDING / DISABLED")
	}
	if a.initialEvent != nil {
		lines = append(lines, fmt.Sprintf("FIRST EVENT: ROLF / MONSTER %d", a.initialEvent.MonsterID))
		if a.tourActive {
			lines = append(lines, fmt.Sprintf("TOUR STEP %02d / %02d",
				a.tourStep+1, len(a.initialEvent.Tour)))
		}
	} else {
		lines = append(lines, "FIRST EVENT: NOT LOADED")
	}
	return lines
}
