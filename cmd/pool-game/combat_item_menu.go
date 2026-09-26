package main

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
	pooltreasure "github.com/wicanr2/Pool-of-Radiance-cht/internal/treasure"
)

// 戰鬥中物品選單的 Ready、Drop、Halve、Join 與卷軸（spec 144，issue #84）。
//
// overlay-19 entry 6（`0EFBh`）的按鍵分派，`[bp-35h]` 是選中的物品：
//
//	129C  'R' → entry 7（14CAh）裝上／卸下
//	12B0  'U' → entry 8（1A86h）用（combat_commands.go）
//	1351  'D' → entry 20（0D72h）放手檢查 → "Your <物品> will be gone forever"、
//	            "Drop It? "，Y → overlay-25 entry 17（156Ah）摘掉
//	1412  'H' → entry 14（17F2h）分半
//	1422  'J' → entry 15（18A2h）合併
//	1469  之後一律 overlay-25 entry 7（0BBEh）重算，回到 `0F2Fh` 選單迴圈
//
// **這四支都不動結果（`[bp+6]`）**，而選單迴圈只在結果非 0（`0F42h`）或身上沒東西
// （`0F51h`）時離開，所以它們在戰鬥中都不用掉行動：選單留著，換完武器還能接著
// 按 U 或 ESC 回指令列再攻擊。重算在同一步做完，THAC0、傷害與射程當場換成新武器的。

// 這一段訊息另開 `iota + 1860`（#84），在 init 登記進 messageKeys，重號直接 panic。
const (
	msgCombatItemReady messageID = iota + 1860
	msgCombatItemDrop
	msgCombatItemHalve
	msgCombatItemJoin
	msgItemCursed
	msgItemWrongClass
	msgItemAlreadyUsing
	msgItemHandsFull
	msgItemCannotHalve
	msgItemMustBeUnreadied
	msgItemDropPrompt
	msgScrollTitle
	msgScrollFooter
	msgScrollScribeMark
)

func init() {
	for id, key := range map[messageID]string{
		msgCombatItemReady:     "ui.combatItemReady",
		msgCombatItemDrop:      "ui.combatItemDrop",
		msgCombatItemHalve:     "ui.combatItemHalve",
		msgCombatItemJoin:      "ui.combatItemJoin",
		msgItemCursed:          "ui.itemCursed",
		msgItemWrongClass:      "ui.itemWrongClass",
		msgItemAlreadyUsing:    "ui.itemAlreadyUsing",
		msgItemHandsFull:       "ui.itemHandsFull",
		msgItemCannotHalve:     "ui.itemCannotHalve",
		msgItemMustBeUnreadied: "ui.itemMustBeUnreadied",
		msgItemDropPrompt:      "ui.itemDropPrompt",
		msgScrollTitle:         "ui.scrollTitle",
		msgScrollFooter:        "ui.scrollFooter",
		msgScrollScribeMark:    "ui.scrollScribeMark",
	} {
		if existing, ok := messageKeys[id]; ok {
			panic(fmt.Sprintf("message id %d is already %q", id, existing))
		}
		messageKeys[id] = key
	}
}

// combatItemStage 是物品選單目前在等哪一種輸入。
type combatItemStage uint8

const (
	combatItemPicking combatItemStage = iota
	// combatItemScribeConfirm 是 entry 20 的 "is it Okay to lose it?"。
	combatItemScribeConfirm
	// combatItemDropConfirm 是 "Drop It? "（`13D9h` 的 overlay-26 entry 6）。
	combatItemDropConfirm
	// combatItemScroll 是 overlay-19 entry 12（`297Bh`）挑卷軸上的一行。
	combatItemScroll
)

// scrollOption 是卷軸上列得出來的一行。
type scrollOption struct {
	spell    uint8
	scribing bool
}

// combatItemMenuInput 是物品選單裡 U 以外的鍵。回傳 true 代表這一影格的按鍵用在這裡。
func (a *app) combatItemMenuInput(state *tacticalState, slot int) (bool, error) {
	menu := a.combatItems
	switch menu.stage {
	case combatItemScribeConfirm, combatItemDropConfirm:
		return true, a.combatItemConfirmInput(state, slot)
	case combatItemScroll:
		return true, a.scrollPickInput(state, slot)
	}
	switch {
	case a.justPressed(ebiten.KeyR):
		return true, a.readyCombatItem(state, slot, menu.cursor)
	case a.justPressed(ebiten.KeyD):
		a.startCombatDrop(state, slot, menu.cursor)
		return true, nil
	case a.justPressed(ebiten.KeyH):
		if len(a.state.Party[slot].Inventory) < gamepack.HalveCountLimit {
			return true, a.halveCombatItem(state, slot, menu.cursor)
		}
		return true, nil
	case a.justPressed(ebiten.KeyJ):
		return true, a.joinCombatItem(state, slot, menu.cursor)
	}
	return false, nil
}

// memberClassUseMask 是記錄 `+0B0h`：玩家建的角色由職業等級算（overlay-16 `0D3Fh`），
// NPC 讀它自己帶的記錄。
func memberClassUseMask(member poolsave.Character) uint8 {
	if member.NPC && len(member.Record) > 0xb0 {
		return member.Record[0xb0]
	}
	return gamepack.ClassUseMask(memberClassLevels(member))
}

// readyCombatItem 是 'R'（overlay-19 entry 7）。
func (a *app) readyCombatItem(state *tacticalState, slot, index int) error {
	if a.itemTypes == nil {
		return nil
	}
	member := &a.state.Party[slot]
	raws := make([][]byte, len(member.Inventory))
	for i := range member.Inventory {
		raws[i] = member.Inventory[i].Raw
		if len(raws[i]) <= gamepack.ItemEffectOffset {
			return nil
		}
	}
	result, err := gamepack.ReadyItem(raws, index, a.itemTypes, memberClassUseMask(*member))
	if err != nil {
		return err
	}
	switch result.Outcome {
	case gamepack.ReadyCursed:
		a.tacticalStatus(state, a.text(msgItemCursed))
	case gamepack.ReadyWrongClass:
		a.tacticalStatus(state, a.text(msgItemWrongClass))
	case gamepack.ReadyAlreadyUsing:
		a.tacticalStatus(state, fmt.Sprintf(a.text(msgItemAlreadyUsing),
			strings.TrimSpace(member.Inventory[result.Blocker].Name)))
	case gamepack.ReadyHandsFull:
		// `16F3h`：戰鬥中由電腦接手（`+10Fh` 非 0）的不印。
		if !state.aiDriven(state.Mover) {
			a.tacticalStatus(state, a.text(msgItemHandsFull))
		}
	}
	return a.afterCombatItemChange(state, slot)
}

// startCombatDrop 是 'D'：先過 entry 20（`0D72h`，spec 067〈放手檢查〉），再問要不要丟。
func (a *app) startCombatDrop(state *tacticalState, slot, index int) {
	item := a.state.Party[slot].Inventory[index]
	if pooltreasure.SellNeedsUnready(item.Raw) {
		a.tacticalStatus(state, a.text(msgItemMustBeUnreadied))
		return
	}
	a.combatItems.pending = index
	if category, ok := a.itemCategory(item); ok && pooltreasure.SellNeedsScribeConfirm(item.Raw, category) {
		a.combatItems.stage = combatItemScribeConfirm
		a.tacticalStatus(state, fmt.Sprintf(a.text(msgShopSellScribe),
			strings.TrimSpace(a.state.Party[slot].Name)))
		return
	}
	a.askCombatDrop(state, slot)
}

func (a *app) askCombatDrop(state *tacticalState, slot int) {
	a.combatItems.stage = combatItemDropConfirm
	name := strings.TrimSpace(a.state.Party[slot].Inventory[a.combatItems.pending].Name)
	a.tacticalStatus(state, fmt.Sprintf(a.text(msgItemDropPrompt), name))
}

// combatItemConfirmInput 是兩個 Y／N：entry 20 只認 Y 才放手，"Drop It? " 也只認 Y
// （`13DEh` 的 `cmp al, 59h`）。
func (a *app) combatItemConfirmInput(state *tacticalState, slot int) error {
	menu := a.combatItems
	switch {
	case a.justPressed(ebiten.KeyY):
		if menu.stage == combatItemScribeConfirm {
			a.askCombatDrop(state, slot)
			return nil
		}
		member := &a.state.Party[slot]
		if menu.pending < len(member.Inventory) {
			member.Inventory = append(member.Inventory[:menu.pending], member.Inventory[menu.pending+1:]...)
		}
		menu.stage = combatItemPicking
		a.tacticalStatus(state, "")
		return a.afterCombatItemChange(state, slot)
	case a.justPressed(ebiten.KeyN), a.justPressed(ebiten.KeyEscape), a.justPressed(ebiten.KeyEnter):
		menu.stage = combatItemPicking
		a.tacticalStatus(state, "")
	}
	return nil
}

// halveCombatItem 是 'H'（overlay-19 entry 14）。
func (a *app) halveCombatItem(state *tacticalState, slot, index int) error {
	member := &a.state.Party[slot]
	item := member.Inventory[index]
	split, ok := gamepack.HalveItem(item.Raw)
	if !ok {
		a.tacticalStatus(state, a.text(msgItemCannotHalve))
		return nil
	}
	inventory := make([]poolsave.Item, 0, len(member.Inventory)+1)
	inventory = append(inventory, member.Inventory[:index+1]...)
	inventory = append(inventory, poolsave.Item{Name: item.Name, Raw: split})
	member.Inventory = append(inventory, member.Inventory[index+1:]...)
	return a.afterCombatItemChange(state, slot)
}

// joinCombatItem 是 'J'（overlay-19 entry 15）。
func (a *app) joinCombatItem(state *tacticalState, slot, index int) error {
	member := &a.state.Party[slot]
	raws := make([][]byte, len(member.Inventory))
	for i := range member.Inventory {
		raws[i] = member.Inventory[i].Raw
		if len(raws[i]) <= gamepack.ItemEffectOffset {
			return nil
		}
	}
	removed := gamepack.JoinItems(raws, index)
	for at := len(removed) - 1; at >= 0; at-- {
		gone := removed[at]
		member.Inventory = append(member.Inventory[:gone], member.Inventory[gone+1:]...)
		if gone < a.combatItems.cursor {
			a.combatItems.cursor--
		}
	}
	return a.afterCombatItemChange(state, slot)
}

// afterCombatItemChange 是 `1469h..1485h`：overlay-25 entry 7 重算這個人，回到選單迴圈；
// 身上沒東西了選單就收起來（`0F51h`）。
func (a *app) afterCombatItemChange(state *tacticalState, slot int) error {
	member := a.state.Party[slot]
	syncTrainedLibraryCharacter(&a.state, member)
	if len(member.Inventory) == 0 {
		a.combatItems = nil
	} else if a.combatItems != nil && a.combatItems.cursor >= len(member.Inventory) {
		a.combatItems.cursor = len(member.Inventory) - 1
	}
	if member.NPC {
		// NPC 的戰鬥數值直接讀它帶的記錄（applyNPCCombatStats），不走裝備那一條。
		return nil
	}
	return a.applyPartyGearStats(state, int(state.Mover), member)
}

// aiDriven 是記錄 `+10Fh`：這一格由電腦接手。
func (state *tacticalState) aiDriven(index uint8) bool {
	return int(index) < len(state.AIDriven) && state.AIDriven[index]
}

// openScrollPick 是 entry 8 的卷軸那一支（`1AA9h..1AC6h`）：overlay-19 entry 12
// 列出卷軸上讀得到的行讓玩家挑。讀不出來（`+35h` 還藏著）或一行都沒有時 entry 12
// 回 0，entry 8 結果 0，戰鬥中留在物品選單，什麼也不印（`1AF1h`）。
func (a *app) openScrollPick(state *tacticalState, slot, index int, category uint8) {
	member := &a.state.Party[slot]
	item := &member.Inventory[index]
	levels := memberClassLevels(*member)
	readable, revealed := gamepack.ScrollReadable(item.Raw, category,
		state.hasEffect(int(state.Mover), gamepack.ReadMagicEffectCode),
		levels[gamepack.ClassSlotCleric])
	if revealed {
		if name, err := a.identifyName(*item); err == nil {
			item.Name = name
		}
		syncTrainedLibraryCharacter(&a.state, *member)
	}
	if !readable {
		return
	}
	spells, scribing := gamepack.ScrollSpells(item.Raw)
	if len(spells) == 0 {
		return
	}
	options := make([]scrollOption, len(spells))
	for i := range spells {
		options[i] = scrollOption{spell: spells[i], scribing: scribing[i]}
	}
	a.combatItems.stage = combatItemScroll
	a.combatItems.pending = index
	a.combatItems.scroll = options
	a.combatItems.scrollCursor = 0
}

// scrollPickInput 是 overlay-22 entry 1（`008Ah`）的挑選：上下挑、Enter 施、ESC 回選單。
func (a *app) scrollPickInput(state *tacticalState, slot int) error {
	menu := a.combatItems
	count := len(menu.scroll)
	switch {
	case a.justPressed(ebiten.KeyEscape):
		menu.stage, menu.scroll = combatItemPicking, nil
	case a.justPressed(ebiten.KeyArrowUp):
		menu.scrollCursor = (menu.scrollCursor + count - 1) % count
	case a.justPressed(ebiten.KeyArrowDown):
		menu.scrollCursor = (menu.scrollCursor + 1) % count
	case a.justPressed(ebiten.KeyEnter), a.justPressed(ebiten.KeySpace):
		spell := menu.scroll[menu.scrollCursor].spell
		index := menu.pending
		menu.stage, menu.scroll = combatItemPicking, nil
		return a.castFromItem(state, slot, index, spell, true)
	}
	return nil
}

// castFromItem 是 entry 8 從 `1AF1h` 起：法術為 0 不做；處理常式沒讀過的放不出去。
// 卷軸不印 "uses an item"（`1B19h` 看 `DS:6CB3h` 為 0 直接跳到 `1BBDh`）。
func (a *app) castFromItem(state *tacticalState, slot, index int, spell uint8, scroll bool) error {
	if spell == 0 || int(spell) >= len(a.spellParameters) || a.spellCaster == nil ||
		!a.spellCaster.Implemented(spell) {
		return nil
	}
	a.combatItems = nil
	if !scroll {
		name := strings.TrimSpace(a.state.Party[slot].Inventory[index].Name)
		a.tacticalStatus(state, state.say(msgFoeUsesItem, state.Mover, name))
	}
	a.combatItem = &combatItemUse{slot: slot, item: index, spell: spell, scroll: scroll,
		keepTurn: a.spellParameters[spell].CampOnly()}
	return a.aimSpell(castOption{Slot: -1, ID: spell, Label: a.spellLabel(spell)}, false)
}

// scrollEraser 是 `1C1Dh..1C3Dh`：結果非 0 就以 overlay-22 entry 7 抹掉那一行，
// 名稱跟著 `+30h` 換（"With 2 Spells" → "With 1 Spell"），最後一行抹掉就拿掉卷軸。
func (a *app) scrollEraser(slot, index int, spell uint8) func() {
	done := false
	return func() {
		if done || slot >= len(a.state.Party) {
			return
		}
		done = true
		member := &a.state.Party[slot]
		if index >= len(member.Inventory) {
			return
		}
		item := &member.Inventory[index]
		if gamepack.EraseScrollSpell(item.Raw, spell) {
			member.Inventory = append(member.Inventory[:index], member.Inventory[index+1:]...)
		} else if name, err := a.identifyName(*item); err == nil {
			item.Name = name
		}
		syncTrainedLibraryCharacter(&a.state, *member)
	}
}

// keepCombatTurn 是指令迴圈以結果非 0 離開、卻沒有呼叫 entry 34 的那一種收尾：
// 分數不歸零，直接重選（與 beginCasting 同一個形狀）。
func (state *tacticalState) keepCombatTurn(roll func(count, sides int) int) {
	result := state.Status
	state.Moving = false
	state.selectActor(roll)
	if state.Mover == 0 {
		state.endRound(roll)
	}
	if !state.Prompt && !state.Finished {
		state.Status = result
	}
}

// combatItemFooter 是物品選單的選項列（`0F79h..1148h` 組起來的那一條）：Ready、
// Use（沉默、咳嗽時沒有）、Drop、Halve（身上不到 10h 件）、Join，最後 Exit。
func (a *app) combatItemFooter(state *tacticalState, slot int) string {
	menu := a.combatItems
	if menu != nil && menu.stage == combatItemScroll {
		return a.text(msgScrollFooter)
	}
	parts := []string{a.text(msgCombatItemReady)}
	if state.combatItemsUsable(int(state.Mover)) {
		parts = append(parts, a.text(msgCombatItemsFooter))
	}
	parts = append(parts, a.text(msgCombatItemDrop))
	if len(a.state.Party[slot].Inventory) < gamepack.HalveCountLimit {
		parts = append(parts, a.text(msgCombatItemHalve))
	}
	parts = append(parts, a.text(msgCombatItemJoin), a.text(msgCombatItemsExitOnly))
	// 分隔是空白，放不進字串表（空白字串會被當成缺字）：英文半形、中文全形。
	separator := "　"
	if a.language == languageEnglish {
		separator = " "
	}
	return strings.Join(parts, separator)
}

// drawScrollOptions 把卷軸上列得出來的行畫在物品清單那一塊（overlay-19 entry 12 的
// "Spells on Scroll"）；有人正要抄的那一行前面加 " *"（overlay-22 `03BDh`）。
func drawScrollOptions(screen *ebiten.Image, a *app, foreground, accent color.Color) {
	menu := a.combatItems
	drawText(screen, a.text(msgScrollTitle), combatInfoLeft, combatItemsTitleLine, accent)
	for index, option := range menu.scroll {
		cursor, ink := " ", foreground
		if index == menu.scrollCursor {
			cursor, ink = ">", accent
		}
		marker := "  "
		if option.scribing {
			marker = a.text(msgScrollScribeMark)
		}
		drawText(screen, cursor+marker+a.spellLabel(option.spell),
			combatInfoLeft, combatItemsTitleLine+18+index*18, ink)
	}
}
