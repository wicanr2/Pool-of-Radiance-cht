package main

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 探索畫面的 `C)AST`（spec 119）。
//
// 原版走 overlay-15 entry 2（`0497h`）：挑人與挑法術是同一個介面，選中之後
// 交給 **overlay-22 entry 5**——與戰鬥中施法是同一支派發（spec 073），
// 所以這裡也共用 `spellOptionsFor` 與 `gamepack.SpellCaster`：
// **看得到的就是施得出來的**。
//
// 施完一條會回到挑人那一步，可以接著施下一條（原版 `054Ch` 跳回迴圈頂）。
// 選到沒記法術的人會說 `<名字> has no spells memorized`（`047Fh`），
// 那是正常回應，不是錯誤。

// 探索施法的三步。
const (
	fieldCastPickCaster = iota
	fieldCastPickSpell
	fieldCastPickTarget
)

// 版面：**選單畫在冒險畫面下方那個框裡，不是整頁**。
//
// 原版在開清單之前呼叫 `sub_1500(0, 16h, 26h, 11h, 1)` 與
// `sub_1638(16h, 26h, 11h, 1)` 開一個框（spec 119 的 `0497h`），冒險畫面
// 留在框外面。先前這一頁鋪滿整個畫面，一個六人隊伍只填得滿六行，
// 其餘九成是黑的；而且玩家看不到自己站在哪裡。
//
// 框是 `dialogueTop`..`dialogueBottom`（264..370），行距 16，放得下六行——
// 隊伍上限正好是六人。法術比六條多時照游標捲動。
const (
	fieldCastLeft     = 52
	fieldCastFirstRow = 282
	fieldCastPitch    = 16
	fieldCastLines    = 6
)

// 挑法術那一步是**整頁**，版面照原版量的（spec 134）。原版 native 座標乘二
// 就是這裡的邏輯座標；只有標題那一行往下讓，因為 remake 在框上緣有自己的
// 一行標題（原版沒有），其餘各行的絕對位置與原版相同。
const (
	spellPageLeft = 18 // 原版 native 9
	// 標題：原版在 native 14（logical 28），那裡被 remake 的標題佔著。
	spellPageTitleRow = 62
	spellPageRuleTop  = 70
	// 級別那一行與清單：原版 native 46／54，行距 8。
	spellPageLevelRow = 92
	spellPageFirstRow = 108
	spellPagePitch    = 16
	spellPageIndent   = 50 // 原版 native 25：清單比級別再縮 16 個像素
	// 清單塞得下幾條：原版第一條在 native 54、下一條橫條在 129。
	spellPageLines = 9
	// 清單與資訊之間那條橫條，以及底下三行：原版 native 129／150／158／166。
	spellPageRuleBottom  = 262
	spellPageNameRow     = 300
	spellPageCanRow      = 316
	spellPageCountRow    = 332
	spellPageCountIndent = 82 // 原版 native 41
)

// openFieldCast 開始探索施法。
func (a *app) openFieldCast() {
	if len(a.state.Party) == 0 {
		return
	}
	a.fieldCastOpen = true
	a.fieldCastStage = fieldCastPickCaster
	a.fieldCastCursor = 0
	a.fieldCastMessage = ""
}

func (a *app) closeFieldCast() {
	a.fieldCastOpen = false
	a.fieldCastOptions = nil
	a.fieldCastMessage = ""
}

// fieldCastInput 處理這一頁的鍵。回傳是否吃掉了這一次按鍵。
func (a *app) fieldCastInput() (bool, error) {
	if !a.fieldCastOpen {
		return false, nil
	}
	switch {
	case a.justPressed(ebiten.KeyEscape):
		switch a.fieldCastStage {
		case fieldCastPickTarget:
			a.fieldCastStage = fieldCastPickSpell
			a.fieldCastCursor = 0
		case fieldCastPickSpell:
			a.fieldCastStage = fieldCastPickCaster
			a.fieldCastCursor = 0
		default:
			a.closeFieldCast()
		}
	case a.justPressed(ebiten.KeyArrowUp):
		a.fieldCastCursor = (a.fieldCastCursor + a.fieldCastCount() - 1) % a.fieldCastCount()
	case a.justPressed(ebiten.KeyArrowDown):
		a.fieldCastCursor = (a.fieldCastCursor + 1) % a.fieldCastCount()
	case a.justPressed(ebiten.KeyEnter), a.justPressed(ebiten.KeySpace):
		return true, a.fieldCastAdvance()
	}
	return true, nil
}

// fieldCastCount 是目前這一步有幾個選項。
func (a *app) fieldCastCount() int {
	switch a.fieldCastStage {
	case fieldCastPickSpell:
		if len(a.fieldCastOptions) == 0 {
			return 1
		}
		return len(a.fieldCastOptions)
	default:
		if len(a.state.Party) == 0 {
			return 1
		}
		return len(a.state.Party)
	}
}

// fieldCastAdvance 走下一步。
func (a *app) fieldCastAdvance() error {
	switch a.fieldCastStage {
	case fieldCastPickCaster:
		if a.fieldCastCursor >= len(a.state.Party) {
			return nil
		}
		a.fieldCastCaster = a.fieldCastCursor
		member := a.state.Party[a.fieldCastCaster]
		options := a.spellOptionsFor(member)
		if len(options) == 0 {
			// 原版 `052Ah`：把名字接上 `has no spells memorized` 印出來。
			a.fieldCastMessage = fmt.Sprintf(a.text(msgFieldCastNoSpells),
				strings.TrimSpace(member.Name))
			return nil
		}
		a.fieldCastOptions = options
		a.fieldCastStage, a.fieldCastCursor = fieldCastPickSpell, 0
		a.fieldCastMessage = ""
	case fieldCastPickSpell:
		if a.fieldCastCursor >= len(a.fieldCastOptions) {
			return nil
		}
		a.fieldCastSpell = a.fieldCastCursor
		a.fieldCastStage, a.fieldCastCursor = fieldCastPickTarget, a.fieldCastCaster
	case fieldCastPickTarget:
		if a.fieldCastCursor >= len(a.state.Party) {
			return nil
		}
		return a.resolveFieldCast(a.fieldCastCursor)
	}
	return nil
}

// resolveFieldCast 把選中的法術施在目標身上。
//
// 戰鬥外沒有戰術格，所以只結算**作用在人身上**的那幾種：治療、把生命值墊到
// 下限、拿掉效果、還一級能量吸取、能力值加成。其餘（傷害、範圍、睡眠…）
// 原版在戰鬥外也沒有目標可打，這裡照實說一句，不假裝施出去了。
func (a *app) resolveFieldCast(target int) error {
	if len(a.fieldCastOptions) == 0 {
		return nil
	}
	option := a.fieldCastChosen()
	caster := &a.state.Party[a.fieldCastCaster]
	if option.Slot >= len(caster.Memorised) ||
		caster.Memorised[option.Slot]&0x7f != option.ID {
		a.fieldCastMessage = a.text(msgCastNothingReady)
		a.fieldCastStage, a.fieldCastCursor = fieldCastPickCaster, 0
		return nil
	}
	levels := memberClassLevels(*caster)
	casterLevel := gamepack.CasterLevelFor(a.spellParameters[option.ID],
		int(levels[gamepack.ClassSlotCleric]), int(levels[gamepack.ClassSlotMagicUser]), false)
	effect, err := a.spellCaster.Cast(option.ID, a.spellParameters, casterLevel, a.roller)
	if err != nil {
		return err
	}
	subject := &a.state.Party[target]
	applied := applyFieldEffect(subject, effect)
	// 記憶那一格用掉了——原版施完把槽位清成 FFFFh 再回迴圈頂。
	caster.Memorised[option.Slot] = 0
	syncTrainedLibraryCharacter(&a.state, *caster)
	syncTrainedLibraryCharacter(&a.state, *subject)
	if applied {
		a.fieldCastMessage = fmt.Sprintf(a.text(msgFieldCastDone),
			strings.TrimSpace(caster.Name), option.Label, strings.TrimSpace(subject.Name))
	} else {
		a.fieldCastMessage = fmt.Sprintf(a.text(msgFieldCastCombatOnly), option.Label)
	}
	a.fieldCastStage, a.fieldCastCursor = fieldCastPickCaster, 0
	a.fieldCastOptions = nil
	return nil
}

// fieldCastChosen 是挑目標那一步之前選中的法術。
func (a *app) fieldCastChosen() castOption {
	if a.fieldCastSpell < len(a.fieldCastOptions) {
		return a.fieldCastOptions[a.fieldCastSpell]
	}
	return a.fieldCastOptions[0]
}

// applyFieldEffect 套用戰鬥外算得出來的那幾種效果，回傳有沒有真的作用。
func applyFieldEffect(member *poolsave.Character, effect gamepack.CastEffect) bool {
	applied := false
	if effect.Heal > 0 {
		before := member.CurrentHP
		member.CurrentHP += effect.Heal
		if member.CurrentHP > member.MaxHP {
			member.CurrentHP = member.MaxHP
		}
		applied = applied || member.CurrentHP != before
	}
	if effect.MinimumHitPoints > 0 && member.CurrentHP < effect.MinimumHitPoints {
		member.CurrentHP = effect.MinimumHitPoints
		applied = true
	}
	for _, code := range effect.RemoveEffects {
		for index, value := range member.Effects {
			if value == code {
				member.Effects = append(member.Effects[:index], member.Effects[index+1:]...)
				applied = true
				break
			}
		}
	}
	return applied
}

// fieldCastRows 是這一步要列的東西。挑法術時是法術，其餘兩步是隊伍。
func (a *app) fieldCastRows() []string {
	if a.fieldCastStage == fieldCastPickSpell {
		rows := make([]string, 0, len(a.fieldCastOptions))
		for _, option := range a.fieldCastOptions {
			rows = append(rows, option.Label)
		}
		return rows
	}
	return a.partyPickerRows()
}

// fieldCastWindow 是列表捲動之後的起點：游標永遠留在看得見的範圍裡。
func fieldCastWindow(cursor, count, lines int) int {
	if count <= lines {
		return 0
	}
	top := cursor - lines/2
	if top < 0 {
		top = 0
	}
	if top > count-lines {
		top = count - lines
	}
	return top
}

// drawPickerInFrame 在下方那個框裡列一份選單。挑施法者、挑法術、`V` 挑人
// 走的是同一支——三處都是「冒險畫面上疊一個小選單」，不是換一頁。
func drawPickerInFrame(screen *ebiten.Image, a *app, rows []string, cursor int,
	message, title string, foreground, accent color.Color) {
	drawDialogueFrame(screen, accent)
	lines := fieldCastLines
	// 有話要說就讓出最後一行——訊息比第六個選項重要（「這名角色沒有記憶
	// 法術」正是玩家按下去之後最需要看到的那一句）。
	if message != "" {
		lines--
	}
	top := fieldCastWindow(cursor, len(rows), lines)
	for offset := 0; offset < lines && top+offset < len(rows); offset++ {
		index := top + offset
		mark, ink := "  ", foreground
		if index == cursor {
			mark, ink = "> ", accent
		}
		drawText(screen, mark+rows[index], fieldCastLeft,
			fieldCastFirstRow+offset*fieldCastPitch, ink)
	}
	if message != "" {
		drawText(screen, message, fieldCastLeft,
			fieldCastFirstRow+(fieldCastLines-1)*fieldCastPitch, foreground)
	}
	drawText(screen, title+"　"+a.text(msgFieldCastFooter), fieldCastLeft, footerBaseline, accent)
}

// partyPickerRows 是隊伍清單那幾行：名字加現有／最大生命值。
func (a *app) partyPickerRows() []string {
	rows := make([]string, 0, len(a.state.Party))
	for _, member := range a.state.Party {
		rows = append(rows, fmt.Sprintf("%s  %d/%d",
			strings.TrimSpace(member.Name), member.CurrentHP, member.MaxHP))
	}
	return rows
}

// drawFieldCast 畫下方那個框：框內是選項，框外那一列是標題與鍵位。
func drawFieldCast(screen *ebiten.Image, a *app, background, foreground, accent color.Color) {
	if a.fieldCastStage == fieldCastPickSpell {
		drawSpellPage(screen, a, background, foreground, accent)
		return
	}
	title := a.text(msgFieldCastPickCaster)
	if a.fieldCastStage == fieldCastPickTarget {
		title = a.text(msgFieldCastPickTarget)
	}
	drawPickerInFrame(screen, a, a.fieldCastRows(), a.fieldCastCursor,
		a.fieldCastMessage, title, foreground, accent)
}

// drawSpellPage 畫挑法術那一頁：標題、一條橫線、縮排的清單、再一條橫線、
// 底下「還能記幾條」，指令列在框外（spec 134）。
//
// **原版這一頁是整頁，不是疊在冒險畫面上的小框**——挑人那一步才是小框，
// 而挑人本來就是 remake 自己加的（原版對「目前角色」施法，spec 119）。
func drawSpellPage(screen *ebiten.Image, a *app, background, foreground, accent color.Color) {
	panel := ebiten.NewImage(logicalWidth-2*guidePanelInset, guidePanelBottom-guidePanelTop)
	panel.Fill(background)
	screen.DrawImage(panel, &ebiten.DrawImageOptions{
		GeoM: translated(guidePanelInset, guidePanelTop)})

	caster := ""
	if a.fieldCastCaster < len(a.state.Party) {
		caster = strings.TrimSpace(a.state.Party[a.fieldCastCaster].Name)
	}
	drawText(screen, fmt.Sprintf(a.text(msgSpellPageTitle), caster),
		spellPageLeft, spellPageTitleRow, accent)
	drawSpellPageRule(screen, spellPageRuleTop, accent)

	drawText(screen, a.text(msgSpellPageLevel), spellPageLeft, spellPageLevelRow, foreground)
	top := fieldCastWindow(a.fieldCastCursor, len(a.fieldCastOptions), spellPageLines)
	for offset := 0; offset < spellPageLines && top+offset < len(a.fieldCastOptions); offset++ {
		index := top + offset
		mark, ink := "  ", foreground
		if index == a.fieldCastCursor {
			mark, ink = "> ", accent
		}
		drawText(screen, mark+a.fieldCastOptions[index].Label,
			spellPageIndent, spellPageFirstRow+offset*spellPagePitch, ink)
	}

	drawSpellPageRule(screen, spellPageRuleBottom, accent)
	drawText(screen, caster, spellPageLeft, spellPageNameRow, foreground)
	if a.fieldCastMessage != "" {
		drawText(screen, a.fieldCastMessage, spellPageLeft, spellPageCanRow, foreground)
	} else {
		drawText(screen, a.text(msgSpellPageCanCast), spellPageLeft, spellPageCanRow, foreground)
		drawText(screen, fmt.Sprintf(a.text(msgSpellPageCount), len(a.fieldCastOptions)),
			spellPageCountIndent, spellPageCountRow, foreground)
	}
	drawText(screen, a.text(msgFieldCastFooter), spellPageLeft, footerBaseline, accent)
}

// drawSpellPageRule 畫一條橫線。原版那兩條是繩索圖塊，remake 這一頁在面板
// 上，畫繩索會與外框的繩索重疊成一團，所以用一條線——標為
// layout-reconstructed，見 spec 134。
func drawSpellPageRule(screen *ebiten.Image, y int, ink color.Color) {
	for x := spellPageLeft; x < logicalWidth-spellPageLeft; x++ {
		screen.Set(x, y, ink)
	}
}
