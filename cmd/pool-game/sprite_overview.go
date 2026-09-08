package main

import (
	"image/color"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/golden-box-remake-engine/graphics"
)

// `F4` 的素材總覽。
//
// 這一頁把 remake 目前**畫得出來**的幾類圖形排在同一個畫面上：肖像、戰鬥
// 造形、第一人稱的牆面圖塊、外框的三個符號。它是遊戲畫面，不是素材匯出——
// 圖形仍然只從玩家自己的原版 ZIP 讀出來，repo 不含也不產生任何素材檔
// （`cmd/pool-combat-icon-audit` 那一行「never exports source art」是同一條界線）。
//
// 用途是一眼看出「哪一類圖已經接上、畫成什麼樣子」。

const (
	// 每一區的標籤基線與圖的上緣。**不畫自己的標題**：外框上緣那一行已經
	// 是標題列，再畫一行會直接疊在上面。
	spritePortraitTop = 62
	spriteIconLeft    = 472
	spriteWallTop     = 216
	spriteFrameTop    = 288
	spriteLabelLeft   = 24
	spriteRowLeft     = 24
	// 總覽要載入的肖像頭數。原版有 14 個頭，這裡取前四個當樣本。
	spriteHeadSamples = 4
)

// openSpriteOverview 打開總覽，並把樣本圖載進來。
func (a *app) openSpriteOverview() {
	a.spriteOpen = true
	if len(a.spritePortraits) == 0 && a.loadPortrait != nil {
		for head := uint8(1); head <= spriteHeadSamples; head++ {
			portrait, err := a.loadPortrait(head, head)
			if err != nil {
				continue
			}
			a.spritePortraits = append(a.spritePortraits, portrait)
		}
	}
	// 戰鬥造形要自己載一組樣本：`iconReady`／`iconAction` 是建角那一步留下的，
	// 冒險畫面按 F4 時是空的。
	if len(a.spriteIcons) == 0 && a.loadIcon != nil {
		// 配色用建角那一步的預設六組（`flow.go` 的 `IconColors`）。
		// 全零會畫出一組沒有顏色的灰影，看起來像圖壞了。
		colours := [6][2]uint8{{1, 9}, {2, 10}, {3, 11}, {4, 12}, {6, 14}, {7, 15}}
		for _, action := range []bool{false, true} {
			icon, err := a.loadIcon(1, 1, 1, action, colours)
			if err != nil {
				continue
			}
			a.spriteIcons = append(a.spriteIcons, icon)
		}
	}
}

// spriteOverviewInput 處理這一頁的鍵。
func (a *app) spriteOverviewInput() (bool, error) {
	if !a.spriteOpen {
		return false, nil
	}
	if a.justPressed(ebiten.KeyEscape) || a.justPressed(ebiten.KeyF4) {
		a.spriteOpen = false
	}
	return true, nil
}

// drawSpriteOverview 畫那四區。
func drawSpriteOverview(screen *ebiten.Image, a *app, background, foreground, accent color.Color) {
	panel := ebiten.NewImage(logicalWidth-2*guidePanelInset, guidePanelBottom-guidePanelTop)
	panel.Fill(background)
	screen.DrawImage(panel, &ebiten.DrawImageOptions{
		GeoM: translated(guidePanelInset, guidePanelTop)})
	palette := a.artPalette()
	drawText(screen, a.text(msgSpritePortraits), spriteLabelLeft, spritePortraitTop, foreground)
	for index, portrait := range a.spritePortraits {
		if portrait == nil {
			continue
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(spriteRowLeft+index*104), float64(spritePortraitTop+4))
		screen.DrawImage(portrait, op)
	}

	drawText(screen, a.text(msgSpriteIcons), spriteIconLeft, spritePortraitTop, foreground)
	for index, icon := range a.spriteIcons {
		if icon == nil {
			continue
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(3, 3)
		op.GeoM.Translate(float64(spriteIconLeft+index*72), float64(spritePortraitTop+12))
		screen.DrawImage(icon, op)
	}

	drawText(screen, a.text(msgSpriteWalls), spriteLabelLeft, spriteWallTop, foreground)
	drawPieceRow(screen, a, a.initialWalls, spriteRowLeft, spriteWallTop+12, palette)

	drawText(screen, a.text(msgSpriteFrame), spriteLabelLeft, spriteFrameTop, foreground)
	for index, item := range []int{frameCornerItem, frameHorizontalItem, frameVerticalItem} {
		a.drawSymbol(screen, item, spriteRowLeft+index*40, spriteFrameTop+12, 3, palette)
	}
	drawText(screen, a.text(msgSpriteFooter), 0, footerBaseline, accent)
}

// drawPieceRow 畫一組牆面圖塊的前幾個。每個 item 是 8×8，畫成三倍。
func drawPieceRow(screen *ebiten.Image, a *app, pieces *graphics.PieceSet,
	left, top int, palette [16]color.RGBA) {
	if pieces == nil || len(pieces.Symbols) == 0 {
		drawText(screen, a.text(msgSpriteNoWalls), left, top+12, palette[15])
		return
	}
	ids := make([]int, 0, len(pieces.Symbols))
	for id := range pieces.Symbols {
		ids = append(ids, int(id))
	}
	sort.Ints(ids)
	picture := pieces.Symbols[uint8(ids[0])]
	width := int(picture.WidthUnits) * 8
	if width == 0 {
		return
	}
	const items, scale = 12, 3
	for item := 0; item < items; item++ {
		drawPictureItem(screen, picture, item, left+item*(8*scale+4), top, scale, palette)
	}
}

// drawPictureItem 畫一張 8x8D 圖片裡的第 item 個 8×8 格。做法與 `drawSymbol`
// 相同：每個 item 是連續的八列，每一列取前八個像素。
func drawPictureItem(screen *ebiten.Image, picture graphics.Picture,
	item, left, top, scale int, palette [16]color.RGBA) {
	width := int(picture.WidthUnits) * 8
	if width == 0 {
		return
	}
	base := item * width * frameTileSize
	for y := 0; y < frameTileSize; y++ {
		for x := 0; x < frameTileSize && x < width; x++ {
			index := base + y*width + x
			if index >= len(picture.Pixels) {
				return
			}
			shade := palette[picture.Pixels[index]&0x0f]
			for dy := 0; dy < scale; dy++ {
				for dx := 0; dx < scale; dx++ {
					screen.Set(left+x*scale+dx, top+y*scale+dy, shade)
				}
			}
		}
	}
}

