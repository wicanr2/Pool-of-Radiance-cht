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

const (
	fieldCastLeft  = 48
	fieldCastTop   = 96
	fieldCastPitch = 20
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

// drawFieldCast 畫這一頁：一列選項加一行訊息。
func drawFieldCast(screen *ebiten.Image, a *app, background, foreground, accent color.Color) {
	panel := ebiten.NewImage(logicalWidth-2*guidePanelInset, guidePanelBottom-guidePanelTop)
	panel.Fill(background)
	screen.DrawImage(panel, &ebiten.DrawImageOptions{
		GeoM: translated(guidePanelInset, guidePanelTop)})

	title := a.text(msgFieldCastPickCaster)
	switch a.fieldCastStage {
	case fieldCastPickSpell:
		title = a.text(msgFieldCastPickSpell)
	case fieldCastPickTarget:
		title = a.text(msgFieldCastPickTarget)
	}
	drawText(screen, title, fieldCastLeft, 78, accent)

	line := fieldCastTop
	if a.fieldCastStage == fieldCastPickSpell {
		for index, option := range a.fieldCastOptions {
			mark, ink := "  ", foreground
			if index == a.fieldCastCursor {
				mark, ink = "> ", accent
			}
			drawText(screen, mark+option.Label, fieldCastLeft, line, ink)
			line += fieldCastPitch
		}
	} else {
		for index, member := range a.state.Party {
			mark, ink := "  ", foreground
			if index == a.fieldCastCursor {
				mark, ink = "> ", accent
			}
			drawText(screen, fmt.Sprintf("%s%s  %d/%d", mark,
				strings.TrimSpace(member.Name), member.CurrentHP, member.MaxHP),
				fieldCastLeft, line, ink)
			line += fieldCastPitch
		}
	}
	if a.fieldCastMessage != "" {
		drawText(screen, a.fieldCastMessage, fieldCastLeft, guidePanelBottom-24, foreground)
	}
	drawText(screen, a.text(msgFieldCastFooter), 0, footerBaseline, accent)
}
