package main

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/assets"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 戰鬥畫面的原版版面（spec 129）：左邊 7×7 格的戰場、右邊三行資訊、
// 框外一列指令。指令那一列是**算出來的**——原版逐段判斷要不要接上去，
// 照抄八項會讓玩家看到按不動的鍵。

// combatInternalArmorBase 是內部護甲值的基準：記錄裡存 `60 − AC`。
const combatInternalArmorBase = 60

// combatBoardCellRect 是視窗裡第 (column,row) 格在畫面上的範圍。
func combatBoardCellRect(column, row int) (left, top int) {
	return combatBoardLeft + column*combatBoardCell, combatBoardTop + row*combatBoardCell
}

// fillCombatCell 填一格。格與格之間留一像素，跟盤面那一版一樣看得出格線。
func fillCombatCell(screen *ebiten.Image, column, row int, ink color.Color) {
	left, top := combatBoardCellRect(column, row)
	for y := top; y < top+combatBoardCell-1; y++ {
		for x := left; x < left+combatBoardCell-1; x++ {
			screen.Set(x, y, ink)
		}
	}
}

// boardSprite 取一格的戰鬥造形，載一次就留著。
func (a *app) boardSprite(index int) *ebiten.Image {
	state := a.tactical
	if state == nil || index >= len(state.Icons) || a.loadIcon == nil {
		return nil
	}
	choice := state.Icons[index]
	if !choice.Valid {
		return nil
	}
	if icon, ok := a.boardIcons[choice]; ok {
		return icon
	}
	if a.boardIcons == nil {
		a.boardIcons = map[boardIcon]*ebiten.Image{}
	}
	icon, err := a.loadIcon(choice.Head, choice.Body, choice.Size, false, choice.Colours)
	if err != nil {
		// 載不出來也記著，不然每一影格都重試一次。
		a.boardIcons[choice] = nil
		return nil
	}
	a.boardIcons[choice] = icon
	return icon
}

// 地形圖塊的分段（spec 131）。地圖裡存的是格位類別碼，類別表第四個欄位
// `PresentationCode` 才是圖塊集裡的序號。
//
// 66 筆類別分兩段：**0..31 是室內、32..65 是野外**。這一刀是從序號範圍推的
// ——前段的序號正好蓋滿 DUNGCOM 的 `00h..18h`（25 個）加 RANDCOM 的
// `22h..27h`（6 個），後段正好蓋滿 WILDCOM 的 `00h..21h`（34 個），三個檔
// 一個 item 都不多不少。哪一段由誰選還沒從程式碼讀出來。
const (
	combatIndoorClassLimit = 32
	// combatRandomTileBase 是隨機遭遇那一組在序號空間裡的起點。
	combatRandomTileBase = 0x22
)

// combatTerrainTile 取一格要鋪的圖塊。取不到就回 nil，由呼叫端退回色塊。
func (a *app) combatTerrainTile(code uint8) *ebiten.Image {
	state := a.tactical
	if state == nil || a.loadCombatTiles == nil || int(code) >= len(state.Classes) {
		return nil
	}
	presentation := int(state.Classes[code].PresentationCode)
	name := assets.WildernessCombatTiles
	item := presentation
	if int(code) < combatIndoorClassLimit {
		name = assets.DungeonCombatTiles
		if presentation >= combatRandomTileBase {
			name, item = assets.RandomCombatTiles, presentation-combatRandomTileBase
		}
	}
	tiles, ok := a.combatTiles[name]
	if !ok {
		if a.combatTiles == nil {
			a.combatTiles = map[string][]*ebiten.Image{}
		}
		loaded, err := a.loadCombatTiles(name)
		if err != nil {
			// 載不出來也記著，不然每一影格都重試一次。
			a.combatTiles[name] = nil
			return nil
		}
		a.combatTiles[name] = loaded
		tiles = loaded
	}
	if item < 0 || item >= len(tiles) {
		return nil
	}
	return tiles[item]
}

// outlineCombatCell 在一格外圍畫一圈框。
func outlineCombatCell(screen *ebiten.Image, column, row int, ink color.Color) {
	left, top := combatBoardCellRect(column, row)
	for x := left; x < left+combatBoardCell-1; x++ {
		screen.Set(x, top, ink)
		screen.Set(x, top+combatBoardCell-2, ink)
	}
	for y := top; y < top+combatBoardCell-1; y++ {
		screen.Set(left, y, ink)
		screen.Set(left+combatBoardCell-2, y, ink)
	}
}

// combatCommandBar 依原版的條件組出最下面那一列（spec 129 的表）。
//
// **兩道閘還沒接**：`Cast ` 的 `es:[di+108h]+1` 與 `ds:4933h+1CAh`、
// `Turn ` 的 `es:[di+108h]+11h`，那三個欄位的語意還沒解出來（spec 129 標 DRAFT）。
// 現在接的是解出來的那幾道；沒接的那幾道只會讓指令**多出現**，不會少，
// 所以不會發生「原版有而 remake 沒有」。
func (a *app) combatCommandBar() string {
	segments := a.combatCommands
	if len(segments) == 0 {
		return ""
	}
	member, isParty := a.combatMoverCharacter()
	line := ""
	for _, segment := range segments {
		switch segment.Key {
		case gamepack.CombatCommandUse:
			// 記錄 `+0C7h` 大於 0：身上有東西才給 Use。
			if !isParty || len(member.Inventory) == 0 {
				continue
			}
		case gamepack.CombatCommandCast:
			// 記錄 `+17h` 起 21 格任何一格非零：記著任何一條法術。
			if !isParty || !hasMemorisedSpell(member.Memorised) {
				continue
			}
		case gamepack.CombatCommandTurn:
			// 記錄 `+96h` 大於 0：牧師等級。
			if !isParty || clericLevel(member) == 0 {
				continue
			}
		}
		line += a.gameText.Translate(segment.Text)
	}
	return line
}

// hasMemorisedSpell 是原版那個掃描：任何一格非零就算。
func hasMemorisedSpell(memorised []uint8) bool {
	for _, value := range memorised {
		if value != 0 {
			return true
		}
	}
	return false
}

func clericLevel(member poolsave.Character) int {
	if int(gamepack.ClassSlotCleric) >= len(member.ClassLevels) {
		return 0
	}
	return int(member.ClassLevels[gamepack.ClassSlotCleric])
}

// combatMoverCharacter 取行動者的隊員記錄。行動的是怪物就回 false。
func (a *app) combatMoverCharacter() (poolsave.Character, bool) {
	if a.tactical == nil {
		return poolsave.Character{}, false
	}
	index, ok := a.moverPartyIndex(a.tactical.Mover)
	if !ok {
		return poolsave.Character{}, false
	}
	return a.state.Party[index], true
}

// drawCombatInfo 畫右側那三行：行動者的名字、HITPOINTS、AC（spec 129）。
func drawCombatInfo(screen *ebiten.Image, a *app, foreground, accent color.Color) {
	state := a.tactical
	if state == nil {
		return
	}
	name := a.text(msgCombatFoe)
	if member, ok := a.combatMoverCharacter(); ok {
		name = strings.TrimSpace(member.Name)
	}
	drawText(screen, name, combatInfoLeft, combatInfoLine1, accent)
	mover := int(state.Mover)
	if mover < len(state.HitPoints) {
		drawText(screen, fmt.Sprintf("%s %d", a.text(msgCombatHitPoints),
			state.HitPoints[mover]), combatInfoLeft, combatInfoLine2, foreground)
	}
	if member, ok := a.combatMoverCharacter(); ok {
		if weapon, has := a.readiedWeapon(member); has {
			drawText(screen, strings.TrimSpace(weapon.Name),
				combatInfoLeft, combatInfoLine4, foreground)
		}
	}
	if mover < len(state.ArmorClass) {
		// `state.ArmorClass` 存的是**內部值** `60 − AC`（命中判定用的那個，
		// 見 `MonsterRecord.ArmorClass`）。畫面上要顯示的是 AC 本身，
		// 所以再減一次；直接印內部值會變成「AC 50」。
		drawText(screen, fmt.Sprintf("%s %d", a.text(msgCombatArmorClass),
			combatInternalArmorBase-state.ArmorClass[mover]),
			combatInfoLeft, combatInfoLine3, foreground)
	}
}

// drawCombatBoard 畫 7×7 的視窗。視窗左上角是戰術地圖 record 的 `+2`／`+3`
// （`combat.ViewportOrigin`），與原版同一個來源。
func drawCombatBoard(screen *ebiten.Image, a *app, foreground color.Color) {
	state := a.tactical
	if state == nil {
		return
	}
	skin := a.currentTheme()
	floor, wall := skin.tacticalFloor, skin.tacticalWall
	occupancy, err := combat.RebuildOccupancy(state.Roster)
	if err != nil {
		drawText(screen, "OCCUPANCY ERROR", combatBoardLeft, combatBoardTop+16, foreground)
		return
	}
	partyInk := color.RGBA{85, 255, 85, 255}
	foeInk := color.RGBA{255, 85, 85, 255}
	for row := 0; row < combat.ViewportTileSpan; row++ {
		for column := 0; column < combat.ViewportTileSpan; column++ {
			mapX := int(state.Viewport.X) + column
			mapY := int(state.Viewport.Y) + row
			if mapX < 0 || mapX >= combat.TacticalMapWidth ||
				mapY < 0 || mapY >= combat.TacticalMapHeight {
				continue
			}
			offset := mapY*combat.TacticalRowStride + mapX
			code := state.Grid.Terrain[offset]
			if code == combat.UnpaintedCellClass {
				continue
			}
			// 先鋪地形。原版每一格是 `*COM.DAX` 的一個 24×24 圖塊
			// （spec 131）；載不出來才退回色塊，那時牆與地板仍分得開。
			if tile := a.combatTerrainTile(code); tile != nil {
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Scale(2, 2)
				left, top := combatBoardCellRect(column, row)
				op.GeoM.Translate(float64(left), float64(top))
				screen.DrawImage(tile, op)
			} else {
				ink := floor
				if int(code) < len(state.Classes) &&
					state.Classes[code].EntryThreshold == 0xFF {
					ink = wall
				}
				fillCombatCell(screen, column, row, ink)
			}
			index := occupancy[offset]
			if index == 0 {
				continue
			}
			// 有人就畫**戰鬥造形**（24×24，正好一格；spec 129）。載不出來
			// 才退回色塊——那時畫面上還是看得出誰站哪裡。
			if icon := a.boardSprite(int(index)); icon != nil {
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Scale(2, 2)
				left, top := combatBoardCellRect(column, row)
				op.GeoM.Translate(float64(left), float64(top))
				screen.DrawImage(icon, op)
			} else {
				mark := foeInk
				if int(index) < len(state.Friendly) && state.Friendly[index] {
					mark = partyInk
				}
				fillCombatCell(screen, column, row, mark)
			}
			// 輪到誰就框起來，原版那一格有一圈白框。
			if uint8(index) == state.Mover {
				outlineCombatCell(screen, column, row, color.RGBA{255, 255, 255, 255})
			}
		}
	}
}
