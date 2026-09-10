package main

import (
	"image"
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

	"ORDER":         msgCampCommandOrder,
	"DROP":          msgCampCommandDrop,
	"SPEED":         msgCampCommandSpeed,
	"ICON":          msgCampCommandIcon,
	"PICS":          msgCampCommandPics,
	"SELECT":        msgCampCommandSelect,
	"PLACE":         msgCampCommandPlace,
	"FASTER":        msgCampCommandFaster,
	"SLOWER":        msgCampCommandSlower,
	"MONSTERS-ON":   msgCampCommandMonstersOn,
	"MONSTERS-OFF":  msgCampCommandMonstersOff,
	"PORTRAITS-ON":  msgCampCommandPortraitsOn,
	"PORTRAITS-OFF": msgCampCommandPortraitsOff,
}

// 紮營的每一列指令（spec 135）。原版的字串分別是：
//
//	`Save View Magic Rest Alter Exit`          `DS:051Bh`
//	`Rest   daYs Hours Mins   Inc Dec   Exit`  overlay-20 `0698h`
//	`Order Drop Speed Icon Pics Exit`          `DS:056Eh`
//	`Select Exit` ／ `Place Exit`              `DS:0598h`／`DS:05C1h`
//	` Faster` ` Slower` ` Exit`                overlay-15 `1A25h`…
//	`Monsters on/off` `Portraits on/off` `Exit` overlay-15 `1BE8h`…
//
// 前綴分別在 overlay-15 `1E31h`（`Camp: `）、`1BE0h`（`Alter: `）、
// `170Eh`（`Party Order: `）、`1A3Bh`（`Game Speed:`）。
var (
	campCommands      = []string{"SAVE", "VIEW", "MAGIC", "REST", "ALTER", "EXIT"}
	campRestCommands  = []string{"REST", "DAYS", "HOURS", "MINS", "INC", "DEC", "EXIT"}
	campAlterCommands = []string{"ORDER", "DROP", "SPEED", "ICON", "PICS", "EXIT"}
)

// commandBarList 是現在該畫哪一列。
func (a *app) commandBarList() []string {
	if !a.campOpen {
		return a.adventureCommandList()
	}
	switch a.campStage {
	case campStageRest:
		return campRestCommands
	case campStageAlter:
		return campAlterCommands
	case campStageOrderSelect:
		return []string{"SELECT", "EXIT"}
	case campStageOrderPlace:
		return []string{"PLACE", "EXIT"}
	case campStageSpeed:
		// 原版把這一列組出來：值是 0 就不列 `Faster`、是 9 就不列 `Slower`
		//（overlay-15 `1AD9h`／`1AF1h` 的兩個比較）。
		row := make([]string, 0, 3)
		if a.gameSpeed > campSpeedFastest {
			row = append(row, "FASTER")
		}
		if a.gameSpeed < campSpeedSlowest {
			row = append(row, "SLOWER")
		}
		return append(row, "EXIT")
	case campStagePics:
		monsters, portraits := "MONSTERS-ON", "PORTRAITS-ON"
		if a.monsterPicsHidden {
			monsters = "MONSTERS-OFF"
		}
		if a.portraitsHidden {
			portraits = "PORTRAITS-OFF"
		}
		return []string{monsters, portraits, "EXIT"}
	case campStageQuitConfirm, campStageDropConfirm:
		// 問句那兩層原版沒有指令列，只有對話框裡的問句。
		return nil
	case campStageIcon:
		// 造形編輯器是整頁，自己帶選單。
		return nil
	}
	return campCommands
}

// commandBarPrefix 是那一列前面的字。排時間與 PICS 兩層原版沒有前綴——
// `Rest` 與 `Monsters` 都是從畫面最左邊直接排起。
func (a *app) commandBarPrefix() string {
	if !a.campOpen {
		return ""
	}
	switch a.campStage {
	case campStageMenu:
		return a.text(msgCampCommandPrefix)
	case campStageAlter:
		return a.text(msgCampAlterPrefix)
	case campStageOrderSelect, campStageOrderPlace:
		return a.text(msgCampOrderPrefix)
	case campStageSpeed:
		return a.text(msgCampSpeedPrefix)
	}
	return ""
}

// commandPrefixInk 是前綴的顏色。原版用第三種色（色號 13），與可按的字母
//（15）和其餘（10）都不同；remake 沒有第三個語意色，用強調色。
func (a *app) commandPrefixInk(accent color.Color) color.Color { return accent }

// commandGlyphWidth 是等寬字的一格。倚天與退路字型的半形都是這個寬度。
const commandGlyphWidth = 8

// 平面圖的幾何（overlay-30 offset 0，2026-09-10 反組譯出來的，見 spec 119）。
//
// 原版一次只顯示 11×11 格，不是整張 16×16：視窗原點是
// `clamp(隊伍座標 - 5, 0, 5)`——盡量把隊伍擺中間，但不讓視窗掉出地圖外。
// 上限 5 是因為 5+10 剛好是最後一格，所以 11 格正好涵蓋到底。
const (
	areaMapSize   = 176
	areaMapWindow = 11
	areaMapCell   = areaMapSize / areaMapWindow
	areaMapMargin = areaMapWindow / 2
)

// areaMapOrigin 是視窗左上角那一格，`clamp(pos - 5, 0, 5)`。
//
// 原版那段是先減再夾兩次（`sub ax,5` 後 `jge`／`jle` 各夾一次），
// 這裡照同一個順序寫，語意才對得上：先往回退五格，退過頭就貼邊。
func areaMapOrigin(position, span int) int {
	origin := position - areaMapMargin
	if origin < 0 {
		origin = 0
	}
	if limit := span - areaMapWindow; origin > limit {
		origin = limit
	}
	return origin
}

// drawAreaMap 畫 `A)REA` 的平面全圖：可走區填滿，牆是另一層灰，隊伍是箭頭。
//
// 說明書 p.21 說「圖上僅顯示牆而不顯示門」，讀起來像線稿——**但原版畫的是
// 實心的兩層灰**（spec 119，逐像素量的）：深灰（EGA 8）鋪可走的地方佔
// 75.04%、淺灰（EGA 7）畫牆佔 24.47%，整框填滿沒有黑底，隊伍那個箭頭是白的。
//
// 顏色走 `artPalette` 而不是介面的 foreground／accent：平面圖是原版的索引像素
// 畫面，classic 主題拿到的就是真正的 EGA 灰，modern 主題拿到它自己那一套的
// 對應格。先前用介面前景色畫線，畫出來的是 `(170,255,255)`——那個顏色根本
// 不在 EGA 十六色裡。
//
// 格數與視窗規則照原版：11×11 格、一格 16 個邏輯像素（原版 native 8 的兩倍），
// 視窗原點見 `areaMapOrigin`。
func drawAreaMap(screen *ebiten.Image, a *app, viewLeft, viewTop int) {
	palette := a.artPalette()
	floor, wall, marker := palette[8], palette[7], palette[15]

	area := image.Rect(viewLeft, viewTop, viewLeft+areaMapSize, viewTop+areaMapSize)
	screen.SubImage(area).(*ebiten.Image).Fill(floor)

	originX := areaMapOrigin(int(a.spawn.X), geometry.Width)
	originY := areaMapOrigin(int(a.spawn.Y), geometry.Height)
	for row := 0; row < areaMapWindow; row++ {
		for column := 0; column < areaMapWindow; column++ {
			grid := a.initialMap.Grid.CellWrapped(originX+column, originY+row)
			left, top := viewLeft+column*areaMapCell, viewTop+row*areaMapCell
			for index, direction := range []int{0, 2, 4, 6} {
				if grid.WallDirections[index] == 0 {
					continue
				}
				drawMapEdge(screen, left, top, areaMapCell, direction, wall)
			}
		}
	}
	drawPartyArrow(screen, viewLeft+(int(a.spawn.X)-originX)*areaMapCell,
		viewTop+(int(a.spawn.Y)-originY)*areaMapCell, areaMapCell, a.spawn.Facing, marker)
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
