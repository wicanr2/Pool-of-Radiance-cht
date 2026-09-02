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
	spellLineCount  = 13
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
	for offset := 0; offset < spellLineCount && offset < len(group); offset++ {
		spell := group[offset]
		cursor, ink := " ", foreground
		if offset == state.cursor {
			cursor, ink = ">", accent
		}
		drawText(screen, fmt.Sprintf("%s%-34s %s", cursor, spell.Name, spell.Text),
			spellTextLeft, spellFirstLine+offset*spellLineHeight, ink)
	}
	drawText(screen, fmt.Sprintf(a.text(msgSpellsCount), len(group)), spellTextLeft, 336, foreground)
	drawText(screen, a.text(msgSpellsFooter), spellTextLeft, 356, accent)
}
