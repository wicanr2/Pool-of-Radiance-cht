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
// 時鐘是走出來的：一步一分（`advanceGameMinute`），小時與分鐘就是那個逐位
// 數的第 3 位與分的十位／個位。
func (a *app) adventureStatusLine() string {
	return fmt.Sprintf("%d, %d %s %02d:%02d%s", a.spawn.X, a.spawn.Y,
		facingLetter(a.spawn.Facing),
		a.gameTime[gamepack.TimeDigitHour], a.gameTime.Minutes(), a.searchIndicator())
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
	// `AC` 與 `HP` 留原文：說明書自己在中文行文裡就是這樣用的
	//（p.19「欄位為 NAME／AC／HP」、p.22「裝甲防護力（AC）」、
	// 詞彙表「Hit Points 生命力（H、P）」），換成漢字反而與手冊對不上。
	drawText(screen, a.text(msgPartyPanelName), partyPanelLeft, partyPanelHeaderRow, accent)
	drawTextRight(screen, a.text(msgPartyPanelAC), partyPanelACRight, partyPanelHeaderRow, accent)
	drawTextRight(screen, a.text(msgPartyPanelHP), partyPanelHPRight, partyPanelHeaderRow, accent)
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
	// 戰場開著的時候，把「還是暫定的那幾項」放進來。它原本畫在戰鬥畫面的
	// 資訊欄，而那是 remake 對自己的狀態說明——玩家在戰場上不需要知道。
	if a.tactical != nil {
		lines = append(lines, a.text(msgTacticalProvisional))
	}
	return lines
}

// advanceGameMinute 走一步加一分鐘（spec 118）。
//
// 量原版量出來的：`workplace/oracle/screens` 那一輪從導覽結束的 `(0,4)` 起走，
// 狀態列的時鐘是 `00:00 → 00:01 → 00:02`，一步一分。**轉向、被牆擋下來的
// 那一步、以及換圖那一步都不加**——換圖那一步（`(0,4)` 往西進貧民窟）之後
// 時鐘還是 `00:00`，下一步才變 `00:01`。
//
// 進位用原版的逐位上限表（`DS:35D4h`），與紮營共用同一份。
func (a *app) advanceGameMinute() {
	a.advanceGameTime(1)
}

// advanceGameTime 是 **overlay-20 entry 2**（offset `0`）：世界時鐘往前走，
// 身上的效果跟著往到期靠近。原版把兩件事寫在同一支裡，所以這裡也不拆成
// 兩條路——拆開的話，任何一個推時鐘的地方漏掉遞減，效果就會永遠不過期。
//
// 一次只推一格（走路一分、休息一刻五分）：`Normalise` 照原版每一位只進位
// 一次，靠的就是「每加一次就正規化一次」。
func (a *app) advanceGameTime(minutes int) {
	a.gameTime[gamepack.TimeDigitMinuteOnes] += minutes
	a.gameTime, _ = a.gameTime.Normalise(a.timeRadix)
	a.advancePartyEffects(minutes)
}
