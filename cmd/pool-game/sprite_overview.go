package main

import (
	"image/color"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/assets"
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
	// 戰鬥特效：`COMSPR.DAX` 十三組，站立與動作各一張。
	if len(a.spriteEffects) == 0 && a.loadMonsterSprite != nil {
		for _, block := range assets.MonsterSpriteBlocks() {
			var pair [2]*ebiten.Image
			for index, action := range []bool{false, true} {
				if icon, err := a.loadMonsterSprite(block, action); err == nil {
					pair[index] = icon
				}
			}
			a.spriteEffects = append(a.spriteEffects, pair)
		}
	}
	// 戰場上的造形：`CBODY.DAX` 的身體逐個配同一個頭。
	// 怪物記錄與角色記錄同格式，所以走的是同一套（`ReadCombatIcon`）。
	if len(a.spriteMonsters) == 0 && a.loadIcon != nil {
		colours := [6][2]uint8{{1, 9}, {2, 10}, {3, 11}, {4, 12}, {6, 14}, {7, 15}}
		for body := uint8(0); body < 32; body++ {
			var pair [2]*ebiten.Image
			if icon, err := a.loadIcon(1, body, 1, false, colours); err == nil {
				pair[0] = icon
			}
			a.spriteMonsters = append(a.spriteMonsters, pair)
		}
	}
	// 戰場地形：三個 `*COM.DAX` 各一組（spec 131）。與戰鬥畫面共用同一份
	// 快取，所以在戰鬥裡看過之後這一頁不會再載一次。
	if a.loadCombatTiles != nil {
		for _, row := range spriteTerrainRows {
			if _, done := a.combatTiles[row.Name]; done {
				continue
			}
			if a.combatTiles == nil {
				a.combatTiles = map[string][]*ebiten.Image{}
			}
			tiles, err := a.loadCombatTiles(row.Name)
			if err != nil {
				a.combatTiles[row.Name] = nil
				continue
			}
			a.combatTiles[row.Name] = tiles
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

// spriteOverviewInput 處理這一頁的鍵。TAB 在「素材」與「怪物」兩頁之間切。
func (a *app) spriteOverviewInput() (bool, error) {
	if !a.spriteOpen {
		return false, nil
	}
	switch {
	case a.justPressed(ebiten.KeyEscape), a.justPressed(ebiten.KeyF4):
		a.spriteOpen = false
	case a.justPressed(ebiten.KeyTab):
		a.spritePage = (a.spritePage + 1) % spritePageCount
	}
	return true, nil
}

// 四頁：素材、戰場造形、戰鬥特效、戰場地形。
const (
	spritePageAssets = iota
	spritePageMonsters
	spritePageEffects
	spritePageTerrain
	spritePageCount
)

// drawEffectOverview 畫戰鬥特效（`COMSPR.DAX`）：箭、飛斧、石頭、閃光、
// 爆炸那一類**投射物與特效**，站立／動作各一張。它不是怪物——怪物走的是
// 下面那一頁的造形庫。
func drawEffectOverview(screen *ebiten.Image, a *app, foreground, accent color.Color) {
	drawText(screen, a.text(msgSpriteEffects), spriteLabelLeft, spritePortraitTop, accent)
	if len(a.spriteEffects) == 0 {
		drawText(screen, a.text(msgSpriteNoMonsters), spriteLabelLeft,
			spritePortraitTop+24, foreground)
		return
	}
	const columns, cell = 7, 80
	for index, pair := range a.spriteEffects {
		left := spriteRowLeft + (index%columns)*cell
		top := spritePortraitTop + 16 + (index/columns)*cell
		for offset, icon := range pair {
			if icon == nil {
				continue
			}
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(1.5, 1.5)
			op.GeoM.Translate(float64(left+offset*36), float64(top))
			screen.DrawImage(icon, op)
		}
	}
}

// 地形那一頁的版面。三組各一列標籤加圖塊，圖塊畫成 1.5 倍（36 像素）。
var spriteTerrainRows = []struct {
	Name    string
	Label   int
	Columns int
}{
	{assets.DungeonCombatTiles, 82, 15},
	{assets.WildernessCombatTiles, 176, 15},
	{assets.RandomCombatTiles, 320, 15},
}

// drawTerrainOverview 畫戰場的地形圖塊（spec 131）。三個 `*COM.DAX` 各一組：
// 地城 25 個、野外 34 個、隨機遭遇 6 個，每個 item 24×24 正好戰場一格。
//
// 地圖裡存的是格位類別碼，不是這裡的序號；中間隔著類別表第四個欄位
// `PresentationCode`。這一頁畫的是圖塊本身，照序號排。
func drawTerrainOverview(screen *ebiten.Image, a *app, foreground, accent color.Color) {
	drawText(screen, a.text(msgSpriteTerrain), spriteLabelLeft, spritePortraitTop, accent)
	for _, row := range spriteTerrainRows {
		tiles := a.combatTiles[row.Name]
		drawText(screen, row.Name, spriteLabelLeft, row.Label, foreground)
		if len(tiles) == 0 {
			continue
		}
		const cell = 38
		for index, tile := range tiles {
			if tile == nil {
				continue
			}
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(1.5, 1.5)
			op.GeoM.Translate(float64(spriteRowLeft+(index%row.Columns)*cell),
				float64(row.Label+8+(index/row.Columns)*cell))
			screen.DrawImage(tile, op)
		}
	}
}

// drawMonsterOverview 畫戰場上的造形庫（`CBODY.DAX` 的三十二種身體）。
//
// **怪物與玩家角色共用這一組**：怪物記錄裡的造形欄位（`+BDh`..`+C6h`）全是 0，
// 戰場上用哪一個由 ECL 的 `LOAD MONSTER` 第三個引數（`MonsterSpawn.IconBlock`）
// 指定。這裡畫的是玩家的預設配色；原版把哥布林那一類畫成紅色是換了配色，
// **配色從哪來還沒定位**。
func drawMonsterOverview(screen *ebiten.Image, a *app, foreground, accent color.Color) {
	drawText(screen, a.text(msgSpriteMonsters), spriteLabelLeft, spritePortraitTop, accent)
	if len(a.spriteMonsters) == 0 {
		drawText(screen, a.text(msgSpriteNoMonsters), spriteLabelLeft,
			spritePortraitTop+24, foreground)
		return
	}
	const columns, cellWidth, cellHeight = 8, 72, 60
	for index, pair := range a.spriteMonsters {
		icon := pair[0]
		if icon == nil {
			continue
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(1.5, 1.5)
		op.GeoM.Translate(float64(spriteRowLeft+(index%columns)*cellWidth),
			float64(spritePortraitTop+16+(index/columns)*cellHeight))
		screen.DrawImage(icon, op)
	}
}

// drawSpriteOverview 畫那四區。
func drawSpriteOverview(screen *ebiten.Image, a *app, background, foreground, accent color.Color) {
	panel := ebiten.NewImage(logicalWidth-2*guidePanelInset, guidePanelBottom-guidePanelTop)
	panel.Fill(background)
	screen.DrawImage(panel, &ebiten.DrawImageOptions{
		GeoM: translated(guidePanelInset, guidePanelTop)})
	switch a.spritePage {
	case spritePageMonsters:
		drawMonsterOverview(screen, a, foreground, accent)
		drawText(screen, a.text(msgSpriteFooter), 0, footerBaseline, accent)
		return
	case spritePageEffects:
		drawEffectOverview(screen, a, foreground, accent)
		drawText(screen, a.text(msgSpriteFooter), 0, footerBaseline, accent)
		return
	case spritePageTerrain:
		drawTerrainOverview(screen, a, foreground, accent)
		drawText(screen, a.text(msgSpriteFooter), 0, footerBaseline, accent)
		return
	}
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

