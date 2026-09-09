package main

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// 探索畫面的 `V)IEW`（spec 119）。原版走 overlay-19 entry 5（`0AB6h`），
// 開出來的就是人物資料頁（spec 130）——與建角那一頁同一個版面，
// 所以這裡共用 `drawSheetFor`。
//
// 先挑人再看：原版有「選定角色」那個全域（`ds:5CF0h`），remake 沒有，
// 所以多一步挑人；挑完的那一頁與原版逐格同版面。

// openViewSheet 開始看隊員資料。
func (a *app) openViewSheet() {
	if len(a.state.Party) == 0 {
		return
	}
	a.viewSheetOpen, a.viewSheetCursor, a.viewSheetShown = true, 0, false
}

// viewSheetInput 處理這一頁的鍵。
func (a *app) viewSheetInput() (bool, error) {
	if !a.viewSheetOpen {
		return false, nil
	}
	switch {
	case a.justPressed(ebiten.KeyEscape):
		if a.viewSheetShown {
			a.viewSheetShown = false
			return true, nil
		}
		a.viewSheetOpen = false
	case a.justPressed(ebiten.KeyArrowUp):
		if !a.viewSheetShown && len(a.state.Party) > 0 {
			a.viewSheetCursor = (a.viewSheetCursor + len(a.state.Party) - 1) % len(a.state.Party)
		}
	case a.justPressed(ebiten.KeyArrowDown):
		if !a.viewSheetShown && len(a.state.Party) > 0 {
			a.viewSheetCursor = (a.viewSheetCursor + 1) % len(a.state.Party)
		}
	case a.justPressed(ebiten.KeyEnter), a.justPressed(ebiten.KeySpace):
		if !a.viewSheetShown && a.viewSheetCursor < len(a.state.Party) {
			a.viewSheetShown = true
			// 那個人自己的肖像。載不出來就讓它是 nil——右上角空著，
			// 其餘欄位照樣看得到。
			a.viewSheetPortrait = nil
			if a.loadPortrait != nil {
				member := a.state.Party[a.viewSheetCursor]
				if portrait, err := a.loadPortrait(member.PortraitHead,
					member.PortraitBody); err == nil {
					a.viewSheetPortrait = portrait
				}
			}
		}
	}
	return true, nil
}

// drawViewSheet 畫這一頁：先是隊員清單，選完之後是那個人的資料頁。
func drawViewSheet(screen *ebiten.Image, a *app, background, foreground, accent color.Color) {
	// **挑人那一步不換頁**：它是冒險畫面上疊一個小選單，與 `C` 的施法選單
	// 同一支（`drawPickerInFrame`）。資料頁本身才是整頁——那一頁原版就是
	// 整頁（spec 130）。
	if !a.viewSheetShown {
		drawPickerInFrame(screen, a, a.partyPickerRows(), a.viewSheetCursor,
			"", a.text(msgViewSheetPick), foreground, accent)
		return
	}
	if a.viewSheetCursor >= len(a.state.Party) {
		return
	}
	panel := ebiten.NewImage(logicalWidth-2*guidePanelInset, guidePanelBottom-guidePanelTop)
	panel.Fill(background)
	screen.DrawImage(panel, &ebiten.DrawImageOptions{
		GeoM: translated(guidePanelInset, guidePanelTop)})
	member := a.state.Party[a.viewSheetCursor]
	// **名字不畫在頁面裡。** 原版那一頁沒有名字那一行（建角當下還沒命名），
	// 硬加一行會壓到外框上緣的標題。名字放在框外那一列。
	drawSheetFor(screen, a, member, a.viewSheetPortrait, foreground, accent)
	drawText(screen, fmt.Sprintf("%s   %s", strings.TrimSpace(member.Name),
		a.text(msgViewSheetFooter)), 0, footerBaseline, accent)
}
