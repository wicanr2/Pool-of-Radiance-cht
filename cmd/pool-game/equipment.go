package main

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 裝備畫面。規則那一半（哪一件算裝備上、裝上去之後 THAC0 與傷害怎麼算）由
// spec 065 從 overlay-25 讀出來；選項與每一個選項的規則是原版 overlay-19 entry 6
// （spec 144／149，item_page.go），版面是 remake 的呈現，不宣稱與原版一致。
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
	// page 是物品選單的選項與確認步驟（item_page.go）。
	page itemPageState
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
	a.equipment.page = itemPageState{}
	a.equipment.message = ""
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

// itemCategory 是物品型別表的類別欄（`+0`：武器、甲、盾、護符戒指…）。
func (a *app) itemCategory(item poolsave.Item) (uint8, bool) {
	if a.itemTypes == nil || len(item.Raw) <= itemTypeOffset {
		return 0, false
	}
	entry, err := a.itemTypes.Entry(item.Raw[itemTypeOffset])
	if err != nil {
		return 0, false
	}
	return entry.Category(), true
}

func (a *app) equipmentInput() {
	state := a.equipment
	if handled, err := a.itemPageInput(); handled {
		if err != nil {
			state.message = err.Error()
		}
		return
	}
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
	listed := len(member.Inventory)
	if drawItemPageOverlay(screen, a, foreground, accent) {
		listed = 0
	}
	for offset := 0; offset < equipmentLineCount && offset < listed; offset++ {
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
	//
	// 沒有備妥武器時這一行只寫「徒手」。**原版此時不動傷害骰**（記錄裡的
	// 傷害欄留在建角時算出來的值），所以這裡也不另外算一組——那是 remake
	// 對照原版的結論，寫在這裡就好，不必印在玩家的畫面上。
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

	footer := a.itemPageFooter()
	if state.message != "" {
		footer = state.message
	}
	drawText(screen, footer, equipmentTextLeft, footerBaseline, foreground)
}
