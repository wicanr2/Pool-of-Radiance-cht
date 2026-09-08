package main

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/guide"
	"github.com/wicanr2/golden-box-remake-engine/geometry"
)

// 遊戲內攻略（`F3`）。做法與 CoAB 的 `guide_overlay.go` 同一套：
// 平面圖上標出這張圖的地點，右邊列出離隊伍最近的幾個。
//
// **預設只顯示走過的格子。** 一打開就把整張圖的答案攤開，等於替玩家把遊戲玩完了；
// 想要全部就按 `V`，第一次會先問一次。CoAB 的做法也是這樣。
//
// 座標的來源見 `internal/guide`：一律出自原始 GEO，不出自任何攻略。

// guideFor 依語言取內建的攻略。
func guideFor(language language) (*guide.Catalogue, error) {
	if language == languageTraditionalChinese {
		return guide.TraditionalChinese()
	}
	return guide.English()
}

// 攻略頁的版面。底部那一列走 `footerBaseline`（框外），所以面板本身停在框內。
const (
	// 外框的左右直列佔 0..15 與 624..639，所以面板從 16 起鋪。
	guidePanelInset = 16
	// 上緣留給外框那一行標題（基線 36，字佔 22..37）。
	guidePanelTop    = 48
	guidePanelBottom = 366
	guideListTop     = 112
	guideListBottom  = 352
)

// guideCellKey 是「走過哪些格」的鍵。
type guideCellKey struct {
	Map  string
	X, Y int
}

// rememberGuideCell 記下隊伍現在站的格子。移動之後叫。
func (a *app) rememberGuideCell() {
	if a.initialMap == nil {
		return
	}
	if a.guideExplored == nil {
		a.guideExplored = map[guideCellKey]bool{}
	}
	a.guideExplored[guideCellKey{
		Map: guide.Key(a.spawn.Map.Archive, a.spawn.Map.BlockID),
		X:   int(a.spawn.X), Y: int(a.spawn.Y),
	}] = true
}

func (a *app) guideCellSeen(mapKey string, x, y int) bool {
	return a.guideExplored[guideCellKey{Map: mapKey, X: x, Y: y}]
}

// currentGuideMap 取出這張圖的攻略。
func (a *app) currentGuideMap() (guide.Map, bool) {
	if a.guide == nil || a.initialMap == nil {
		return guide.Map{}, false
	}
	return a.guide.Map(a.spawn.Map.Archive, a.spawn.Map.BlockID)
}

// guideInput 處理攻略頁自己的鍵。回傳是否吃掉了這一次按鍵。
func (a *app) guideInput() (bool, error) {
	if !a.guideOpen {
		return false, nil
	}
	switch {
	case a.justPressed(ebiten.KeyEscape), a.justPressed(ebiten.KeyF3):
		a.guideOpen, a.guideFull = false, false
	case a.justPressed(ebiten.KeyV):
		// 第一次要先看到警告；再按一次才真的攤開。
		if !a.guideFull && !a.guideSpoilerWarned {
			a.guideSpoilerWarned = true
			return true, nil
		}
		a.guideFull = !a.guideFull
	}
	return true, nil
}

// drawGuide 畫攻略頁。
func drawGuide(screen *ebiten.Image, a *app, background, foreground, accent color.Color) {
	// 只鋪底，不另外圍一圈繩索——外框已經有一圈，再圍一圈會蓋掉上面那一行
	// 標題列，而且原版的覆蓋層也沒有雙框。
	// 鋪滿外框以內的整塊。**左邊要蓋到 x=16**：第一人稱那一框自己圍了一圈
	// 繩索，左邊那兩個角落 tile 在 32..47，蓋不到就會從面板旁邊露出兩塊黃色。
	panel := ebiten.NewImage(logicalWidth-2*guidePanelInset, guidePanelBottom-guidePanelTop)
	panel.Fill(background)
	screen.DrawImage(panel, &ebiten.DrawImageOptions{
		GeoM: translated(guidePanelInset, guidePanelTop)})

	definition, ok := a.currentGuideMap()
	title := a.text(msgGuideTitle)
	if ok && definition.Title != "" {
		title = fmt.Sprintf("%s：%s", a.text(msgGuideTitle), definition.Title)
	}
	drawText(screen, title, 72, 78, accent)

	if !ok {
		drawText(screen, a.text(msgGuideNoMap), 72, 120, foreground)
		drawText(screen, a.guideFooterText(), 72, footerBaseline, accent)
		return
	}

	const cell = 11
	viewLeft, viewTop := 72, 100
	mapKey := guide.Key(a.spawn.Map.Archive, a.spawn.Map.BlockID)
	for y := 0; y < geometry.Height; y++ {
		for x := 0; x < geometry.Width; x++ {
			if !a.guideFull && !a.guideCellSeen(mapKey, x, y) {
				continue
			}
			grid := a.initialMap.Grid.CellWrapped(x, y)
			left, top := viewLeft+x*cell, viewTop+y*cell
			for index, direction := range []int{0, 2, 4, 6} {
				if grid.WallDirections[index] == 0 {
					continue
				}
				drawMapEdge(screen, left, top, cell, direction, foreground)
			}
		}
	}
	// 地點標記逐格畫——一個地點常常佔好幾格，那是玩家要看的形狀。
	for _, point := range definition.Points {
		if !a.guideFull && !a.guideCellSeen(mapKey, point.X, point.Y) {
			continue
		}
		markGuideCell(screen, viewLeft+point.X*cell, viewTop+point.Y*cell, cell, accent)
	}
	drawPartyArrow(screen, viewLeft+int(a.spawn.X)*cell, viewTop+int(a.spawn.Y)*cell,
		cell, a.spawn.Facing, accent)

	// 右邊列出離隊伍最近的幾個地點，同一個地點只列一次。
	//
	// **列幾筆由高度算，不是寫死的。** 寫死的話字型換行高或面板改高度時，
	// 清單會直接溢出面板畫到框上——而那看起來像繪製壞掉，不像清單太長。
	const entryPitch = 40
	textLeft, line := viewLeft+geometry.Width*cell+24, guideListTop
	capacity := (guideListBottom - guideListTop) / entryPitch
	listed := 0
	for _, point := range definition.Labels(int(a.spawn.X), int(a.spawn.Y)) {
		if !a.guideFull && !a.guideCellSeen(mapKey, point.X, point.Y) {
			continue
		}
		if listed >= capacity {
			drawText(screen, "...", textLeft, line, foreground)
			break
		}
		drawText(screen, fmt.Sprintf("(%d,%d) %s", point.X, point.Y, point.Label),
			textLeft, line, accent)
		drawText(screen, point.Summary, textLeft, line+18, foreground)
		line += entryPitch
		listed++
	}
	if listed == 0 {
		drawText(screen, a.text(msgGuideNothingSeen), textLeft, guideListTop, foreground)
	}
	// 模式與底部提示併成同一列。分成兩列的話那一列會橫過去撞上右邊的清單
	// ——而「兩段字疊在一起」看起來像字型壞掉，不像版面沒排開。
	drawText(screen, a.guideFooterText(), 72, footerBaseline, accent)
}

// guideFooterText 是底部那一列：平常是操作提示，攤開或剛按過警告時換成狀態。
func (a *app) guideFooterText() string {
	switch {
	case a.guideFull:
		return a.text(msgGuideFullOn)
	case a.guideSpoilerWarned:
		return a.text(msgGuideSpoilerWarning)
	}
	return a.text(msgGuideFooter)
}

// markGuideCell 在一格中間點一個實心方塊。
func markGuideCell(screen *ebiten.Image, left, top, cell int, ink color.Color) {
	for y := top + cell/3; y < top+cell-cell/3; y++ {
		for x := left + cell/3; x < left+cell-cell/3; x++ {
			screen.Set(x, y, ink)
		}
	}
}
