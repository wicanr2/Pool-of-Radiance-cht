package main

import (
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/font"

	"github.com/wicanr2/golden-box-remake-engine/geometry"
)

// 冒險畫面最下面那一列指令（spec 119）。
//
// 字串就是原版的：`DS:04CAh` 是 `Area Cast View Encamp Search Look`，
// `DS:04F3h` 是**少了 `Area`** 的 `Cast View Encamp Search Look`，野外用後者
// ——說明書 p.21 寫的「在月之海沿岸的陸地上行動時，此指令完全沒有作用」
// 在位元組上就是這樣落地的。
var (
	adventureCommands          = []string{"AREA", "CAST", "VIEW", "ENCAMP", "SEARCH", "LOOK"}
	wildernessAdventureCommands = adventureCommands[1:]
)

const (
	// SearchWhileWalkingBit 是 `[4937h]+594h` 的第 0 位：邊走邊搜
	//（overlay-14 `0AA9h` 的 `xor ax, 1`）。
	SearchWhileWalkingBit = 1
	// LookOnceBit 是同一個欄位的第 1 位：這一次要搜
	//（overlay-14 `0AC4h` 的 `or ax, 2`）。
	LookOnceBit = 2
)

// adventureCommandList 依現在在哪一種地圖回傳那一列。
func (a *app) adventureCommandList() []string {
	if a.inWilderness() {
		return wildernessAdventureCommands
	}
	return adventureCommands
}

// searchIndicator 是狀態列後面那一段。說明書 p.20 的狀態列範例是
// `15,4 N 12:33 SEARCH`——邊走邊搜開著的時候才有。
//
// 紮營的時候接的是另一個字：原版在 overlay-25 `2931h`／`2939h` 放著
// ` camping` 與 ` search` 兩個（spec 135），時鐘行後面接哪一個看狀態。
func (a *app) searchIndicator() string {
	if a.campOpen {
		return " " + a.text(msgCampIndicator)
	}
	if a.searchFlags&SearchWhileWalkingBit == 0 {
		return ""
	}
	return " SEARCH"
}

// freeMovementActive 是「那一列指令看得到、按得動」的條件。
//
// 原版導覽還在跑的時候最下面是 `PRESS <ENTER>/<RETURN> TO CONTINUE`
//（基準圖 `03`），自由移動之後才換成指令列（基準圖 `06`）。
func (a *app) freeMovementActive() bool {
	return a.mode == modeAdventure && a.introDone && !a.introWaiting && !a.tourActive &&
		!a.cellEventPending && !a.cellWaitingMenu && !a.endingActive &&
		a.tactical == nil && !a.tacticalPreview && a.eclInput == nil
}

// adventureCommandInput 處理那一列的六個鍵。回傳是否吃掉了這一次按鍵。
//
// 分派照 overlay-14 `09CAh`：`A` 翻 `DS:6A0Ah`（平面圖開關）、`S` 翻
// `+594h` 的第 0 位、`L` 設第 1 位並把 ECL 的 PC 指到 entry 1
//（`ds:494Eh = ds:4946h`，那就是 spec 016 的 SearchLocation）。
func (a *app) adventureCommandInput() (bool, error) {
	if a.help || !a.freeMovementActive() {
		return false, nil
	}
	switch {
	case a.justPressed(ebiten.KeyA) && !a.inWilderness():
		a.areaMapOpen = !a.areaMapOpen
		return true, nil
	case a.justPressed(ebiten.KeyC):
		// 原版 `0A84h`：狀態正常才進 overlay-15 entry 2（spec 119）。
		// remake 沒有「選定角色」那個全域，所以直接開挑人那一步。
		a.openFieldCast()
		return true, nil
	case a.justPressed(ebiten.KeyV):
		// 原版 `0A8Fh`：`V` 走 overlay-19 entry 5（spec 119），那一頁就是
		// 人物資料頁（spec 130）。
		a.openViewSheet()
		return true, nil
	case a.justPressed(ebiten.KeyS):
		a.searchFlags ^= SearchWhileWalkingBit
		return true, nil
	case a.justPressed(ebiten.KeyL):
		a.searchFlags |= LookOnceBit
		if a.eventSession == nil || a.initialMap == nil {
			return true, nil
		}
		if err := a.beginInitialSearch(); err != nil {
			a.statusLine = err.Error()
		}
		a.searchFlags &^= LookOnceBit
		return true, nil
	}
	return false, nil
}

// drawCommandBar 畫那一列，可按的那個字母用強調色。
//
// **高亮的是「大寫的那個字母」，不是第一個字母。** 原版的字型只有大寫字模，
// 所以字串本身是混合大小寫、顯示出來全大寫，而大小寫決定顏色：
// `Area Cast View Encamp Search Look` 的大寫剛好都在字首，但紮營那一列的
// `Rest   daYs Hours Mins   Inc Dec   Exit` 不是——`daYs` 高亮的是第三個字母
//（原版 native 量到的就是格 7、8 綠、格 9 白、格 10 綠，spec 135）。
func drawCommandBar(screen *ebiten.Image, a *app, foreground, accent color.Color) {
	// 基線是 `footerBaseline`：外框下緣那一列 tile（邏輯 y 368..383，
	// spec 123）**下面**那一條，與原版同一個位置。
	// 從畫面最左邊起，跟原版一樣（量到的起點是 native x 0..1，
	// 也就是邏輯 0..2）。那一列在框外面，不受框的內縮限制。
	x := 0
	if prefix := a.commandBarPrefix(); prefix != "" {
		drawText(screen, prefix, x, footerBaseline, a.commandPrefixInk(accent))
		x += font.MeasureString(uiFace, displayText(prefix)).Ceil()
	}
	for _, command := range a.commandBarList() {
		label := a.commandLabel(command)
		if label == "" {
			continue
		}
		before, key, after := splitCommandKey(label)
		// **推進用整串的寬度，不是三段各自 Ceil 的和。** 每一段各自向上取整
		// 會多出一兩個像素，一列六個指令累積起來就把整列往右推——對拍上是
		// 十個像素的無聲退步。
		beforeWidth := font.MeasureString(uiFace, displayText(before)).Ceil()
		keyWidth := font.MeasureString(uiFace, displayText(key)).Ceil()
		if before != "" {
			drawText(screen, before, x, footerBaseline, foreground)
		}
		drawText(screen, key, x+beforeWidth, footerBaseline, accent)
		if after != "" {
			drawText(screen, after, x+beforeWidth+keyWidth, footerBaseline, foreground)
		}
		// 間距一格。寬度要量出來——中文字是兩格寬，用 len() 會算錯。
		x += font.MeasureString(uiFace, displayText(label)).Ceil() + commandGlyphWidth
	}
}

// splitCommandKey 把一個標籤切成「鍵之前／鍵／鍵之後」三段。
//
// **按 rune 切，不是按位元組**：中文標籤從位元組中間切下去會切在 UTF-8 的
// 中間，畫出來是亂碼。找的是第一個大寫 ASCII 字母；沒有（純中文標籤）就退回
// 第一個字元，那與舊行為相同。
func splitCommandKey(label string) (before, key, after string) {
	runes := []rune(label)
	index := 0
	for position, value := range runes {
		if value >= 'A' && value <= 'Z' {
			index = position
			break
		}
	}
	return string(runes[:index]), string(runes[index : index+1]), string(runes[index+1:])
}

// commandLabel 取這一個指令在目前語言下的字樣。
//
// 英文就是原版的位元組（`DS:04CAh`）。繁中取自軟體世界說明書 p.21–p.31 對
// 每一個指令的說明用詞：AREA 是「平面圖」（「便會出現一幅本地區的平面全圖」）、
// CAST 施法、VIEW 檢視、ENCAMP 紮營、SEARCH 邊走邊搜、LOOK 搜尋。
// **鍵名留在最前面**，因為原版就是首字母選擇，換成純中文會讓玩家不知道按什麼。
func (a *app) commandLabel(command string) string {
	id, ok := commandMessages[command]
	if !ok {
		return command
	}
	if label := a.text(id); label != "" {
		return label
	}
	return command
}

var commandMessages = map[string]messageID{
	"AREA":   msgCommandArea,
	"CAST":   msgCommandCast,
	"VIEW":   msgCommandView,
	"ENCAMP": msgCommandEncamp,
	"SEARCH": msgCommandSearch,
	"LOOK":   msgCommandLook,
	"SAVE":   msgCampCommandSave,
	"MAGIC":  msgCampCommandMagic,
	"REST":   msgCampCommandRest,
	"ALTER":  msgCampCommandAlter,
	"EXIT":   msgCampCommandExit,
	"DAYS":   msgCampCommandDays,
	"HOURS":  msgCampCommandHours,
	"MINS":   msgCampCommandMins,
	"INC":    msgCampCommandInc,
	"DEC":    msgCampCommandDec,
}

// 紮營的兩列指令（spec 135）。原版的字串是 `DS:051Bh` 的
// `Save View Magic Rest Alter Exit` 與 overlay-20 `0698h` 的
// `Rest   daYs Hours Mins   Inc Dec   Exit`，前綴 `Camp: ` 在
// overlay-15 `1E31h`。
var (
	campCommands     = []string{"SAVE", "VIEW", "MAGIC", "REST", "ALTER", "EXIT"}
	campRestCommands = []string{"REST", "DAYS", "HOURS", "MINS", "INC", "DEC", "EXIT"}
)

// commandBarList 是現在該畫哪一列。
func (a *app) commandBarList() []string {
	if !a.campOpen {
		return a.adventureCommandList()
	}
	if a.campStage == campStageRest {
		return campRestCommands
	}
	return campCommands
}

// commandBarPrefix 是那一列前面的字。只有紮營的第一層有（`Camp: `）；
// 排時間那一層原版是從畫面最左邊直接排 `Rest`，沒有前綴。
func (a *app) commandBarPrefix() string {
	if !a.campOpen || a.campStage == campStageRest {
		return ""
	}
	return a.text(msgCampCommandPrefix)
}

// commandPrefixInk 是前綴的顏色。原版用第三種色（色號 13），與可按的字母
//（15）和其餘（10）都不同；remake 沒有第三個語意色，用強調色。
func (a *app) commandPrefixInk(accent color.Color) color.Color { return accent }

// commandGlyphWidth 是等寬字的一格。倚天與退路字型的半形都是這個寬度。
const commandGlyphWidth = 8

// drawAreaMap 畫 `A)REA` 的平面全圖：只畫牆，隊伍是一個箭頭。
//
// 說明書 p.21：「圖上僅顯示牆而不顯示門」，隊伍「以一個箭號表示，箭頭所指的
// 方向就是隊伍前進的方向」。格線用 GEO 的四個方向牆位元組，非零就畫一條邊。
func drawAreaMap(screen *ebiten.Image, a *app, foreground, accent color.Color,
	viewLeft, viewTop int) {
	const cell = 11 // 176 ÷ 16
	for y := 0; y < geometry.Height; y++ {
		for x := 0; x < geometry.Width; x++ {
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
	drawPartyArrow(screen, viewLeft+int(a.spawn.X)*cell, viewTop+int(a.spawn.Y)*cell,
		cell, a.spawn.Facing, accent)
}

// drawMapEdge 畫一格的一條邊。方向是原版的 0 北 2 東 4 南 6 西。
func drawMapEdge(screen *ebiten.Image, left, top, cell, direction int, ink color.Color) {
	for step := 0; step < cell; step++ {
		switch direction {
		case 0:
			screen.Set(left+step, top, ink)
		case 2:
			screen.Set(left+cell-1, top+step, ink)
		case 4:
			screen.Set(left+step, top+cell-1, ink)
		case 6:
			screen.Set(left, top+step, ink)
		}
	}
}

// drawPartyArrow 在隊伍那一格畫一個指著前進方向的箭頭。
func drawPartyArrow(screen *ebiten.Image, left, top, cell int, facing uint8, ink color.Color) {
	centre := cell / 2
	for step := 0; step < centre; step++ {
		for span := -step; span <= step; span++ {
			var x, y int
			switch facing & 3 {
			case 0: // 北
				x, y = left+centre+span, top+step
			case 1: // 東
				x, y = left+cell-1-step, top+centre+span
			case 2: // 南
				x, y = left+centre+span, top+cell-1-step
			default: // 西
				x, y = left+step, top+centre+span
			}
			screen.Set(x, y, ink)
		}
	}
}

// adventureCommandHelp 是說明頁那幾列，讓 F-key 提示不因為底下換成原版指令列
// 而消失。
func adventureCommandHelp() []string {
	return []string{
		"F1 HELP  F2 THEME  F5 TACTICAL  ESC BACK  F10 QUIT",
		"A)REA toggles the overhead map; S)EARCH walks while searching",
		"L)OOK searches this cell once; " + strings.ToUpper("e)ncamp opens camp"),
	}
}

// adventureStatusLineLimit 是狀態訊息那一列放得下幾個字。
// 畫面 640 寬、從 x=42 起算、一個字 8 像素：(640−42−16) ÷ 8 = 72。
const adventureStatusLineLimit = 72
