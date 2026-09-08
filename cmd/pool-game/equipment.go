package main

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 裝備畫面。規則那一半（哪一件算裝備上、裝上去之後 THAC0 與傷害怎麼算）由
// spec 065 從 overlay-25 讀出來；這個畫面本身是 remake 的呈現，原版的 ITEMS
// 選單還沒反組譯，所以版面不宣稱與原版一致。
//
// 版面沿用手冊那一頁量過的數字：行距 16、內文左界 48。
const (
	equipmentTextLeft   = 48
	equipmentFirstLine  = 118
	equipmentLineHeight = 16
	equipmentLineCount  = 13
)

type equipmentState struct {
	member  int
	item    int
	message string
}

func (a *app) openEquipment() {
	if len(a.state.Party) == 0 {
		a.statusLine = a.text(msgEquipmentEmptyParty)
		return
	}
	if a.equipment == nil {
		a.equipment = &equipmentState{}
	}
	a.equipment.clamp(a.state.Party)
	a.equipmentOpen = true
}

func (s *equipmentState) clamp(party []poolsave.Character) {
	if len(party) == 0 {
		s.member, s.item = 0, 0
		return
	}
	s.member = (s.member%len(party) + len(party)) % len(party)
	count := len(party[s.member].Inventory)
	if count == 0 {
		s.item = 0
		return
	}
	s.item = (s.item%count + count) % count
}

// toggleReady 把選中的物品裝上或卸下。原版的角色記錄只有一個武器槽
//（`+0CCh`），所以裝上一件的同時要把別件卸下——兩件同時掛著會讓
// readiedWeapon 依順序挑，畫面上看起來像隨機換武器。
func (a *app) toggleReady() {
	state := a.equipment
	party := a.state.Party
	if len(party) == 0 || state.member >= len(party) {
		return
	}
	inventory := party[state.member].Inventory
	if state.item >= len(inventory) {
		return
	}
	if len(inventory[state.item].Raw) <= itemReadyOffset {
		state.message = a.text(msgEquipmentUnreadyable)
		return
	}
	wasReady := inventory[state.item].Raw[itemReadyOffset] != 0
	for index := range inventory {
		if len(inventory[index].Raw) > itemReadyOffset {
			inventory[index].Raw[itemReadyOffset] = 0
		}
	}
	if !wasReady {
		inventory[state.item].Raw[itemReadyOffset] = 1
	}
	state.message = ""
}

func (a *app) equipmentInput() {
	state := a.equipment
	switch {
	case a.justPressed(ebiten.KeyEscape), a.justPressed(ebiten.KeyI):
		a.equipmentOpen = false
	case a.justPressed(ebiten.KeyTab), a.justPressed(ebiten.KeyRight):
		state.member++
		state.item = 0
		state.message = ""
		state.clamp(a.state.Party)
	case a.justPressed(ebiten.KeyLeft):
		state.member--
		state.item = 0
		state.message = ""
		state.clamp(a.state.Party)
	case a.justPressed(ebiten.KeyDown):
		state.item++
		state.clamp(a.state.Party)
	case a.justPressed(ebiten.KeyUp):
		state.item--
		state.clamp(a.state.Party)
	case a.justPressed(ebiten.KeyEnter):
		a.toggleReady()
	}
}

func drawEquipment(screen *ebiten.Image, a *app, background, foreground, accent color.Color) {
	state := a.equipment
	for y := 40; y < 372; y++ {
		for x := 32; x < 608; x++ {
			screen.Set(x, y, background)
		}
	}
	drawText(screen, a.text(msgEquipmentTitle), 268, 62, accent)
	party := a.state.Party
	if len(party) == 0 || state.member >= len(party) {
		drawText(screen, a.text(msgEquipmentEmptyParty), equipmentTextLeft, equipmentFirstLine, foreground)
		return
	}
	member := party[state.member]
	drawText(screen, fmt.Sprintf("%d/%d  %s", state.member+1, len(party), member.Name),
		equipmentTextLeft, 92, accent)

	if len(member.Inventory) == 0 {
		drawText(screen, a.text(msgEquipmentNoItems), equipmentTextLeft, equipmentFirstLine, foreground)
	}
	for offset := 0; offset < equipmentLineCount && offset < len(member.Inventory); offset++ {
		item := member.Inventory[offset]
		marker := "  "
		if len(item.Raw) > itemReadyOffset && item.Raw[itemReadyOffset] != 0 {
			marker = a.text(msgEquipmentReadyMark)
		}
		cursor := " "
		if offset == state.item {
			cursor = ">"
		}
		ink := foreground
		if offset == state.item {
			ink = accent
		}
		drawText(screen, fmt.Sprintf("%s%s%s", cursor, marker, item.Name),
			equipmentTextLeft, equipmentFirstLine+offset*equipmentLineHeight, ink)
	}

	// 裝備上之後的戰鬥數值直接畫出來：看不到數字就分不出「裝備沒生效」與
	// 「這把武器本來就這麼弱」。
	line := a.text(msgEquipmentUnarmed)
	if weapon, ok := a.readiedWeapon(member); ok {
		baseThac0, _, _, err := partyCombatStats(member)
		if err != nil {
			line = err.Error()
		} else {
			stats, err := a.weaponCombatStats(weapon, member, baseThac0)
			if err != nil {
				line = err.Error()
			} else {
				line = fmt.Sprintf(a.text(msgEquipmentStats),
					60-int(stats.Thac0Internal), stats.DamageCount, stats.DamageSides, stats.DamageBonus)
			}
		}
	}
	// AC 與腳程同理：裝了盔甲卻看不到數字動，就分不出「規則沒接上」與
	// 「這件盔甲本來就沒比較好」。同一行放得下，就不另外佔一行——下面
	// 那行是 footer，中間沒有空間。
	if armor, movement, err := a.memberDefenceStats(member, creationArmorClassInternal, creationBaseMovement); err != nil {
		line = err.Error()
	} else {
		line += fmt.Sprintf(a.text(msgEquipmentDefence), 60-armor, movement)
	}
	drawText(screen, line, equipmentTextLeft, 336, accent)

	footer := a.text(msgEquipmentFooter)
	if state.message != "" {
		footer = state.message
	}
	drawText(screen, footer, equipmentTextLeft, footerBaseline, foreground)
}
