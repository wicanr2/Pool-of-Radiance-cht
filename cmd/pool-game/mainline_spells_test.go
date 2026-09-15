package main

// 玩家策略層第四條（issue #22）：法術。說明書 p.13 建議的隊伍有牧師與法師，
// p.44 說「多多使用催眠（Sleep）」。這裡照正常按鍵記法術：`K` 開法術頁、`1..6`
// 選人、TAB 換分頁、↓ 移游標、`M` 記、ESC 關；記完要休息過才施得出來
// （spec 110：第 7 位是「待記完」，休息時清掉）。

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// spellPlan 是每個法術組（`SpellSlotCounts` 的第一維：0 牧師、1 法師）要記的
// 第 1 級法術：牧師全記輕傷治療，法師全記催眠。
var spellPlan = [gamepack.SpellSlotGroups]uint8{gamepack.SpellIDCureLightWound, gamepack.SpellIDSleep}

// memoriseSpells 把每個人每一個空的第 1 級格子都記上計畫裡的法術。回傳記了幾條；
// 大於 0 就要休息一次才生效。
func (d *mainlineDriver) memoriseSpells() int {
	d.t.Helper()
	a := d.a
	if a.tactical != nil || a.campOpen || a.spellsOpen {
		d.fatalf("memoriseSpells: not on the adventure screen")
	}
	d.step(ebiten.KeyK)
	if !a.spellsOpen || a.spells == nil {
		d.fatalf("memoriseSpells: K did not open the spells screen")
	}
	memberKeys := []ebiten.Key{ebiten.KeyDigit1, ebiten.KeyDigit2, ebiten.KeyDigit3,
		ebiten.KeyDigit4, ebiten.KeyDigit5, ebiten.KeyDigit6}
	total := 0
	for index := range a.state.Party {
		d.step(memberKeys[index])
		maxima, used, ok := a.spellMemberSlots(index)
		if !ok {
			continue
		}
		for group, id := range spellPlan {
			free := int(maxima[group][0]) - int(used[group][0])
			if free <= 0 {
				continue
			}
			// 分頁順序照 spellGroups：牧師 1..3 級、法師 1..3 級。
			page := group * gamepack.SpellSlotLevels
			for guard := 0; guard < len(spellGroups) && a.spells.group != page; guard++ {
				d.step(ebiten.KeyTab)
			}
			list := a.spells.current()
			target := -1
			for position, spell := range list {
				if uint8(spell.Index+1) == id {
					target = position
				}
			}
			if target < 0 {
				d.fatalf("memoriseSpells: spell %d is not on page %d", id, page)
			}
			for guard := 0; guard < len(list) && a.spells.cursor != target; guard++ {
				d.step(ebiten.KeyDown)
			}
			for slot := 0; slot < free; slot++ {
				before := gamepack.MemorisedCounts(a.state.Party[index].Memorised, a.spellParameters)
				d.step(ebiten.KeyM)
				after := gamepack.MemorisedCounts(a.state.Party[index].Memorised, a.spellParameters)
				if after[group][0] != before[group][0]+1 {
					d.fatalf("memoriseSpells: %s could not memorise spell %d (%s)",
						a.state.Party[index].Name, id, a.statusLine)
				}
				total++
			}
		}
	}
	d.step(ebiten.KeyEscape)
	if a.spellsOpen {
		d.fatalf("memoriseSpells: the spells screen did not close")
	}
	return total
}

// pendingMemorisation 說有沒有人記了法術還沒休息過。
func pendingMemorisation(a *app) bool {
	for _, member := range a.state.Party {
		if gamepack.PendingMemorisationTime(member.Memorised, a.spellParameters) > 0 {
			return true
		}
	}
	return false
}

// sleepReady 說還有沒有人記著催眠術可以施。沒有法師的隊伍永遠回 true。
func sleepReady(a *app) bool {
	casters := false
	for _, member := range a.state.Party {
		if member.Status != 0 {
			continue
		}
		levels := memberClassLevels(member)
		if levels[gamepack.ClassSlotMagicUser] == 0 {
			continue
		}
		casters = true
		for _, option := range a.spellOptionsFor(member) {
			if option.ID == gamepack.SpellIDSleep {
				return true
			}
		}
	}
	return !casters
}
