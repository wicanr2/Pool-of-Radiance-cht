package main

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// 法術一覽。這是查閱用的畫面，不是施法：記憶與施展還沒接，畫面上也不假裝
// 接好了——原版的法術書畫面另有版面，等反組譯讀到再做。
//
// 表本身是原版 START.EXE 裡那 56 筆（spec 068），順序即原版的順序。
const (
	spellTextLeft   = 48
	spellFirstLine  = 118
	spellLineHeight = 16
	spellLineCount  = 8
	spellColumns    = 64
)

// spellGroups 是六個分頁，順序照原版表的分組。
var spellGroups = []struct {
	Class gamepack.SpellClass
	Level int
}{
	{gamepack.SpellClassCleric, 1}, {gamepack.SpellClassCleric, 2}, {gamepack.SpellClassCleric, 3},
	{gamepack.SpellClassMagicUser, 1}, {gamepack.SpellClassMagicUser, 2}, {gamepack.SpellClassMagicUser, 3},
}

type spellState struct {
	catalogue *gamepack.SpellCatalogue
	group     int
	cursor    int
}

func (s *spellState) current() []gamepack.Spell {
	item := spellGroups[s.group]
	return s.catalogue.ByClassAndLevel(item.Class, item.Level)
}

func (a *app) openSpells() error {
	if a.spells == nil {
		catalogue, err := gamepack.TraditionalChineseSpells()
		if err != nil {
			return err
		}
		a.spells = &spellState{catalogue: catalogue}
	}
	a.spellsOpen = true
	return nil
}

func (a *app) spellsInput() {
	state := a.spells
	switch {
	case a.justPressed(ebiten.KeyEscape), a.justPressed(ebiten.KeyK):
		a.spellsOpen = false
	case a.justPressed(ebiten.KeyTab), a.justPressed(ebiten.KeyRight):
		state.group = (state.group + 1) % len(spellGroups)
		state.cursor = 0
	case a.justPressed(ebiten.KeyLeft):
		state.group = (state.group - 1 + len(spellGroups)) % len(spellGroups)
		state.cursor = 0
	case a.justPressed(ebiten.KeyDown):
		if group := state.current(); len(group) > 0 {
			state.cursor = (state.cursor + 1) % len(group)
		}
	case a.justPressed(ebiten.KeyUp):
		if group := state.current(); len(group) > 0 {
			state.cursor = (state.cursor - 1 + len(group)) % len(group)
		}
	}
}

// spellFacts 把參數表裡讀得出證據的三個欄位排成一行：持續、豁免、命中。
func (a *app) spellFacts(spell gamepack.Spell) string {
	id := spell.Index + 1
	if id < 1 || id >= len(a.spellParameters) {
		return ""
	}
	record := a.spellParameters[id]
	fixed, perLevel := record.Duration(0), record.Duration(1)-record.Duration(0)
	var duration string
	switch {
	case fixed > 0 && perLevel > 0:
		duration = fmt.Sprintf(a.text(msgSpellsDurationBoth), fixed, perLevel)
	case fixed > 0:
		duration = fmt.Sprintf(a.text(msgSpellsDurationFixed), fixed)
	case perLevel > 0:
		duration = fmt.Sprintf(a.text(msgSpellsDurationPerLevel), perLevel)
	default:
		duration = a.text(msgSpellsDurationUntilBroken)
	}
	save := a.text(msgSpellsSaveNone)
	if record.AllowsSavingThrow() {
		save = a.text(msgSpellsSaveSpell)
		if record.SaveCategory() == gamepack.SaveParalyzation {
			save = a.text(msgSpellsSavePoison)
		}
	}
	// 中文用全形空白分隔，英文用兩個半形——這是排版，不是可翻譯的字串。
	separator := "  "
	if a.language == languageTraditionalChinese {
		separator = "　"
	}
	line := duration + separator + save
	if record.RequiresAttackRoll() {
		line += separator + a.text(msgSpellsTouch)
	}
	return line
}

func drawSpells(screen *ebiten.Image, a *app, background, foreground, accent color.Color) {
	state := a.spells
	for y := 40; y < 372; y++ {
		for x := 32; x < 608; x++ {
			screen.Set(x, y, background)
		}
	}
	drawText(screen, a.text(msgSpellsTitle), 268, 62, accent)

	current := spellGroups[state.group]
	className := a.text(msgSpellsCleric)
	if current.Class == gamepack.SpellClassMagicUser {
		className = a.text(msgSpellsMagicUser)
	}
	drawText(screen, fmt.Sprintf(a.text(msgSpellsGroup), className, current.Level),
		spellTextLeft, 92, accent)

	group := state.current()
	// 一頁放不下十三條，捲動時讓游標留在畫面內。
	first := state.cursor - spellLineCount/2
	if first < 0 {
		first = 0
	}
	if first+spellLineCount > len(group) {
		first = len(group) - spellLineCount
	}
	if first < 0 {
		first = 0
	}
	for offset := 0; offset < spellLineCount && first+offset < len(group); offset++ {
		spell := group[first+offset]
		cursor, ink := " ", foreground
		if first+offset == state.cursor {
			cursor, ink = ">", accent
		}
		drawText(screen, fmt.Sprintf("%s%-34s %s", cursor, spell.Name, spell.Text),
			spellTextLeft, spellFirstLine+offset*spellLineHeight, ink)
	}

	// 說明來自說明書下冊第六章，是「說明書寫的行為」，不是反組譯出來的規則。
	if state.cursor < len(group) {
		effect := group[state.cursor].Effect
		if effect == "" {
			effect = a.text(msgSpellsNoEffect)
		}
		for index, line := range wrapDisplay(effect, spellColumns) {
			if index >= 4 {
				break
			}
			drawText(screen, line, spellTextLeft, 258+index*spellLineHeight, foreground)
		}
	}
	// 這一行的三件事全部來自原版的參數表（spec 074），不是說明書。
	if state.cursor < len(group) {
		drawText(screen, a.spellFacts(group[state.cursor]), spellTextLeft, 316, foreground)
	}
	drawText(screen, fmt.Sprintf(a.text(msgSpellsCount), len(group)), spellTextLeft, 336, foreground)
	drawText(screen, a.text(msgSpellsFooter), spellTextLeft, 356, accent)
}
