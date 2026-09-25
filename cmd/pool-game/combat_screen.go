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

// 地形圖塊的分段（spec 131、spec 060）。地圖裡存的是格位類別碼，類別表第四個
// 欄位 `PresentationCode` 才是圖塊序號。
//
// 哪一組圖塊由誰選，是 overlay-10 `12E5h` 決定的（exact）：`DS:495Bh` 等於 1
// 時載 `DungCom`（`12F7h` 的字串在 `12CDh`，序號 0..18h），否則載 `WildCom`
// （`12D5h`，0..21h）；兩邊之後都載 `RandCom`（`12DDh`，從 22h 起）。
// 序號與檔案的對應（`149Ah` 的兩個參數是起點與終點）是 strong inference：
// 三段正好蓋滿三個檔的 item 數，一個不多不少。所以同一個序號 16h 在室內是
// 地城的地板、在室外是野外的平地——分段看的是戰場模式，不是類別碼。
const combatRandomTileBase = 0x22

// combatTerrainTile 取一格要鋪的圖塊。取不到就回 nil，由呼叫端退回色塊。
func (a *app) combatTerrainTile(code uint8) *ebiten.Image {
	state := a.tactical
	if state == nil || a.loadCombatTiles == nil || int(code) >= len(state.Classes) {
		return nil
	}
	presentation := int(state.Classes[code].PresentationCode)
	name, item := assets.DungeonCombatTiles, presentation
	if state.Grid.Outdoor {
		name = assets.WildernessCombatTiles
	}
	if presentation >= combatRandomTileBase {
		name, item = assets.RandomCombatTiles, presentation-combatRandomTileBase
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
// `Cast ` 的 `es:[di+108h]+1` 是 runtime +1「這一回合還能施法」（spec 096〈entry 4〉），
// 受過傷、沉默或咳嗽就清 0——接上了（castingDisrupted）。`Turn ` 的 `+11h` 是
// 「這一場轉過了」（`Undead.Tried`，overlay-13 `11AAh` 寫）。
// **還沒接的一道**：`Cast ` 的 `ds:4933h+1CAh`（ECL `@49E5`，這一版恆為 0）。它只會讓
// 指令**多出現**，不會少，所以不會發生「原版有而 remake 沒有」。
func (a *app) combatCommandBar() string {
	segments := a.combatCommands
	if len(segments) == 0 {
		return ""
	}
	line := ""
	for _, segment := range segments {
		if a.combatSegmentShown(segment.Key) {
			line += a.gameText.Translate(segment.Text)
		}
	}
	return line
}

// combatSegmentShown 是指令列那一段接不接上（spec 129 的表）。按鍵也看同一份：
// 原版的指令迴圈只收列上有的字母。
func (a *app) combatSegmentShown(key gamepack.CombatCommandKey) bool {
	member, isParty := a.combatMoverCharacter()
	switch key {
	case gamepack.CombatCommandUse:
		// 記錄 `+0C7h` 大於 0：身上有東西才給 Use。
		return isParty && len(member.Inventory) > 0
	case gamepack.CombatCommandCast:
		// 記錄 `+17h` 起 21 格任何一格非零：記著任何一條法術。
		if !isParty || !hasMemorisedSpell(member.Memorised) {
			return false
		}
		// overlay-08 `072Fh`：runtime +1 為 0 就不接。
		return a.tactical == nil || !a.tactical.castingDisrupted(int(a.tactical.Mover))
	case gamepack.CombatCommandTurn:
		// `076Dh` 記錄 `+96h`（牧師等級）大於 0；`077Dh` runtime `+11h` 為 0。
		if !isParty || clericLevel(member) == 0 {
			return false
		}
		return a.tactical == nil || !a.tactical.Undead.Tried[int(a.tactical.Mover)]
	}
	return true
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
	// 原版這一欄畫的是**行動者自己的記錄**，不分玩家或怪物（#62）：overlay-25
	// entry 5（`09E8h`）拿傳進來的記錄遠指標逐行畫，第一行由 `1865h` 從記錄取
	// 名字；overlay-13 `0C30h` 那個呼叫端只看記錄 `+10Dh`（還在戰場上）就畫。
	// 怪物記錄與角色記錄同一份版面，所以輪到怪物時印的是怪物名。
	name := a.text(msgCombatFoe)
	if member, ok := a.combatMoverCharacter(); ok {
		name = strings.TrimSpace(member.Name)
	} else if monster, ok := a.stagedMonsterFor(int(state.Mover), state.Friendly); ok &&
		strings.TrimSpace(monster.Record.Name) != "" {
		name = strings.TrimSpace(a.monsterText.Translate(monster.Record.Name))
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
