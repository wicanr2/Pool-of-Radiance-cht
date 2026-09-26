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

// 探索與營地中的物品頁（spec 149，issue #91）。
//
// 原版的物品頁與戰鬥中的是同一支 overlay-19 entry 6（`0EFBh`），差別只在
// `DS:4954h`：選項照 spec 144 那張表組，戰鬥外多了 Trade，`+07h` 為 0 的物品法術
// 問 "Use it?"，而且用完物品不離開選單（`130Ah..1314h` 把結果清 0）。所以這裡與
// combat_item_menu.go 共用 Ready、Halve、Join 的規則與訊息，只換掉「戰場」那一層。
//
// 裝上與卸下都經 overlay-24 entry 1 把物品 `+3Eh` 當效果碼派發（穿戴效果，
// `gamepack.ApplyWearEffect`）；戰鬥中的 R 也走同一支 wearItem。

// 這一段訊息另開 `iota + 3490`（#91），在 init 登記進 messageKeys，重號直接 panic。
const (
	msgItemStronger messageID = iota + 3490
	msgItemNeedsGiantStrength
	msgItemCombatOnlyPrompt
	msgItemPickTarget
	msgItemPageTargetFooter
)

func init() {
	for id, key := range map[messageID]string{
		msgItemStronger:           "ui.itemStronger",
		msgItemNeedsGiantStrength: "ui.itemNeedsGiantStrength",
		msgItemCombatOnlyPrompt:   "ui.itemCombatOnlyPrompt",
		msgItemPickTarget:         "ui.itemPickTarget",
		msgItemPageTargetFooter:   "ui.itemPageTargetFooter",
	} {
		if existing, ok := messageKeys[id]; ok {
			panic(fmt.Sprintf("message id %d is already %q", id, existing))
		}
		messageKeys[id] = key
	}
}

// itemPageStage 是物品頁目前在等哪一種輸入。
type itemPageStage uint8

const (
	itemPagePicking itemPageStage = iota
	// itemPageScribeConfirm 是 entry 20 的 "is it Okay to lose it?"（Drop 之前）。
	itemPageScribeConfirm
	// itemPageDropConfirm 是 "Drop It? "（`13D9h`）。
	itemPageDropConfirm
	// itemPageScroll 是 overlay-19 entry 12（`297Bh`）挑卷軸上的一行。
	itemPageScroll
	// itemPageCombatOnly 是 overlay-22 entry 5 戰鬥外的 "Use it? "（`0BE9h`）。
	itemPageCombatOnly
	// itemPageTarget 是戰鬥外放物品法術時挑對象（與 C)AST 的挑對象同一種，spec 119）。
	itemPageTarget
)

// itemPageState 是 equipmentState 裡物品選單那一半。
type itemPageState struct {
	stage   itemPageStage
	pending int
	// scroll 與 scrollCursor 是卷軸上列得出來的行。
	scroll       []scrollOption
	scrollCursor int
	// spell 與 fromScroll 是正要放的物品法術（挑對象、"Use it?" 兩步用）。
	spell      uint8
	fromScroll bool
	target     int
	// creationMenu 是從隊伍選單（原版 overlay-16，`DS:4954h` 在 `0167h` 設成 0）開的：
	// " Use" 只在 4954h 是 2、3、4（或戰鬥中）才接（`0FA3h..0FBDh`），這裡不接。
	creationMenu bool
}

// wearItem 是 overlay-19 `1528h..153Fh`／`1650h..1667h`：`+3Eh` 大於 7Fh 的物品裝上或
// 卸下之後，把它當效果碼交給 overlay-24 entry 1。state 為 nil 是戰鬥外，串列在角色身上；
// 戰鬥中串列在那一格（combat_effects.go 進場時搬過去的同一條）。回傳要印的那一句。
func (a *app) wearItem(slot int, raw []byte, mode gamepack.WearMode, state *tacticalState, cell int) string {
	if !gamepack.HasWearEffect(raw) || slot < 0 || slot >= len(a.state.Party) {
		return ""
	}
	member := &a.state.Party[slot]
	inCombat := state != nil && cell > 0 && cell < len(state.Effects)
	list := combatEffects(member.Effects)
	if inCombat {
		list = state.Effects[cell]
	}
	result := gamepack.ApplyWearEffect(list, raw, mode,
		uint8(member.Abilities[gamepack.AbilityStrength]), uint8(member.ExceptionalStrength))
	line := ""
	if result.Stronger {
		member.Abilities[gamepack.AbilityStrength] = int(result.Strength)
		member.ExceptionalStrength = int(result.Percentile)
		line = fmt.Sprintf(a.text(msgItemStronger), strings.TrimSpace(member.Name))
	}
	if result.Refused {
		// overlay-12 `3167h`：常式自己把 `+34h` 寫回 0。
		raw[gamepack.ItemReadiedOffset] = 0
		line = a.text(msgItemNeedsGiantStrength)
	}
	// 摘節點一律經 overlay-24 entry 2：`+4` 立著就先收尾（力量還原在這裡）。收尾會改
	// 剩下那幾個力量節點的快照，所以寫回串列要在它之後。
	for _, node := range result.Removed {
		a.expiredEffectTeardown(slot, node, result.List)
	}
	if inCombat {
		state.Effects[cell] = result.List
	}
	member.Effects = storedEffects(result.List)
	syncTrainedLibraryCharacter(&a.state, *member)
	return line
}

// readyMemberItem 是 overlay-19 entry 7（`14CAh`）加上穿戴效果，回傳要印的那一句。
// 戰鬥中 `16F3h` 對電腦接手的人不印 "Your hands are full!"——那一條由呼叫端判。
func (a *app) readyMemberItem(slot, index int, state *tacticalState, cell int) (gamepack.ReadyOutcome, string, error) {
	if a.itemTypes == nil || slot < 0 || slot >= len(a.state.Party) {
		return gamepack.ReadyDone, "", nil
	}
	member := &a.state.Party[slot]
	if index < 0 || index >= len(member.Inventory) {
		return gamepack.ReadyDone, "", nil
	}
	raws := make([][]byte, len(member.Inventory))
	for i := range member.Inventory {
		raws[i] = member.Inventory[i].Raw
		if len(raws[i]) <= gamepack.ItemEffectOffset {
			return gamepack.ReadyDone, a.text(msgEquipmentUnreadyable), nil
		}
	}
	result, err := gamepack.ReadyItem(raws, index, a.itemTypes, memberClassUseMask(*member))
	if err != nil {
		return result.Outcome, "", err
	}
	switch result.Outcome {
	case gamepack.ReadyDone:
		return result.Outcome, a.wearItem(slot, raws[index], gamepack.WearOn, state, cell), nil
	case gamepack.UnreadyDone:
		return result.Outcome, a.wearItem(slot, raws[index], gamepack.WearOff, state, cell), nil
	case gamepack.ReadyCursed:
		return result.Outcome, a.text(msgItemCursed), nil
	case gamepack.ReadyWrongClass:
		return result.Outcome, a.text(msgItemWrongClass), nil
	case gamepack.ReadyAlreadyUsing:
		// `16B9h` 是 overlay-25 entry 1（`0441h`）以四個 0 叫：只把那一件的名字重組進
		// 物品 `+0`（不加 Yes／No 欄、不印到畫面），接著 `16C4h` 印 "already using "
		// 加上那個名字。remake 的 Item.Name 在拿到、鑑定、抹卷軸時就已經照同一支重組過
		// （identifyName），所以直接用它；原版在這裡沒有另外印一段前綴。
		blocker := member.Inventory[result.Blocker]
		return result.Outcome, fmt.Sprintf(a.text(msgItemAlreadyUsing), strings.TrimSpace(blocker.Name)), nil
	case gamepack.ReadyHandsFull:
		return result.Outcome, a.text(msgItemHandsFull), nil
	}
	return result.Outcome, "", nil
}

// halveMemberItem 是 'H'（overlay-19 entry 14）：回傳 false 代表 "Can't halve that"。
func (a *app) halveMemberItem(slot, index int) bool {
	member := &a.state.Party[slot]
	item := member.Inventory[index]
	split, ok := gamepack.HalveItem(item.Raw)
	if !ok {
		return false
	}
	inventory := make([]poolsave.Item, 0, len(member.Inventory)+1)
	inventory = append(inventory, member.Inventory[:index+1]...)
	inventory = append(inventory, poolsave.Item{Name: item.Name, Raw: split})
	member.Inventory = append(inventory, member.Inventory[index+1:]...)
	return true
}

// joinMemberItems 是 'J'（overlay-19 entry 15），回傳游標前面被拿掉了幾件。
func (a *app) joinMemberItems(slot, index, cursor int) int {
	member := &a.state.Party[slot]
	raws := make([][]byte, len(member.Inventory))
	for i := range member.Inventory {
		raws[i] = member.Inventory[i].Raw
		if len(raws[i]) <= gamepack.ItemEffectOffset {
			return 0
		}
	}
	removed := gamepack.JoinItems(raws, index)
	before := 0
	for at := len(removed) - 1; at >= 0; at-- {
		gone := removed[at]
		member.Inventory = append(member.Inventory[:gone], member.Inventory[gone+1:]...)
		if gone < cursor {
			before++
		}
	}
	return before
}

// itemPageInput 是物品頁的選項鍵（R／Enter、U、D、H、J）與幾個確認步驟。回傳 true
// 代表這一影格的按鍵用在這裡。
func (a *app) itemPageInput() (bool, error) {
	state := a.equipment
	page := &state.page
	party := a.state.Party
	if state.member >= len(party) {
		return false, nil
	}
	slot := state.member
	switch page.stage {
	case itemPageScribeConfirm, itemPageDropConfirm:
		a.itemPageDropConfirmInput(slot)
		return true, nil
	case itemPageScroll:
		a.itemPageScrollInput(slot)
		return true, nil
	case itemPageCombatOnly:
		a.itemPageCombatOnlyInput(slot)
		return true, nil
	case itemPageTarget:
		return true, a.itemPageTargetInput(slot)
	}
	if len(party[slot].Inventory) == 0 || state.item >= len(party[slot].Inventory) {
		return false, nil
	}
	index := state.item
	switch {
	case a.justPressed(ebiten.KeyR), a.justPressed(ebiten.KeyEnter):
		_, line, err := a.readyMemberItem(slot, index, nil, 0)
		state.message = line
		a.afterItemPageChange(slot)
		return true, err
	case a.justPressed(ebiten.KeyU):
		if !page.creationMenu {
			a.useItemOnPage(slot, index)
		}
		return true, nil
	case a.justPressed(ebiten.KeyD):
		a.startItemPageDrop(slot, index)
		return true, nil
	case a.justPressed(ebiten.KeyH):
		if len(party[slot].Inventory) < gamepack.HalveCountLimit {
			state.message = ""
			if !a.halveMemberItem(slot, index) {
				state.message = a.text(msgItemCannotHalve)
			}
			a.afterItemPageChange(slot)
		}
		return true, nil
	case a.justPressed(ebiten.KeyJ):
		state.message = ""
		state.item -= a.joinMemberItems(slot, index, index)
		a.afterItemPageChange(slot)
		return true, nil
	}
	return false, nil
}

// afterItemPageChange 是 `1469h`：每一個選項做完都以 overlay-25 entry 7 重算這個人，
// 回到選單迴圈。remake 的重算在畫面上即時算（memberDefenceStats／weaponCombatStats），
// 這裡只把角色庫同步、把游標收回範圍內。
func (a *app) afterItemPageChange(slot int) {
	syncTrainedLibraryCharacter(&a.state, a.state.Party[slot])
	a.equipment.clamp(a.state.Party)
}

// startItemPageDrop 是 'D'：先過 entry 20（`0D72h`），再問要不要丟（同戰鬥中那一支）。
func (a *app) startItemPageDrop(slot, index int) {
	state := a.equipment
	item := a.state.Party[slot].Inventory[index]
	if pooltreasure.SellNeedsUnready(item.Raw) {
		state.message = a.text(msgItemMustBeUnreadied)
		return
	}
	state.page.pending = index
	if category, ok := a.itemCategory(item); ok && pooltreasure.SellNeedsScribeConfirm(item.Raw, category) {
		state.page.stage = itemPageScribeConfirm
		state.message = fmt.Sprintf(a.text(msgShopSellScribe), strings.TrimSpace(a.state.Party[slot].Name))
		return
	}
	a.askItemPageDrop(slot)
}

func (a *app) askItemPageDrop(slot int) {
	state := a.equipment
	state.page.stage = itemPageDropConfirm
	name := strings.TrimSpace(a.state.Party[slot].Inventory[state.page.pending].Name)
	state.message = fmt.Sprintf(a.text(msgItemDropPrompt), name)
}

// itemPageDropConfirmInput：兩個 Y／N 都只認 Y（`13DEh` 的 `cmp al, 59h`）。
func (a *app) itemPageDropConfirmInput(slot int) {
	state := a.equipment
	page := &state.page
	switch {
	case a.justPressed(ebiten.KeyY):
		if page.stage == itemPageScribeConfirm {
			a.askItemPageDrop(slot)
			return
		}
		member := &a.state.Party[slot]
		if page.pending < len(member.Inventory) {
			member.Inventory = append(member.Inventory[:page.pending], member.Inventory[page.pending+1:]...)
		}
		page.stage, state.message = itemPagePicking, ""
		a.afterItemPageChange(slot)
	case a.justPressed(ebiten.KeyN), a.justPressed(ebiten.KeyEscape), a.justPressed(ebiten.KeyEnter):
		page.stage, state.message = itemPagePicking, ""
	}
}

// useItemOnPage 是 'U'（`12B0h`）加上 entry 8（`1A86h`）的前半：沒穿戴的印
// "Must be Readied"；卷軸讓玩家挑一行；其餘 `+3Dh` 非 0 而且 `+3Eh` < 80h 的才放。
func (a *app) useItemOnPage(slot, index int) {
	state := a.equipment
	member := &a.state.Party[slot]
	item := &member.Inventory[index]
	if len(item.Raw) <= gamepack.AIItemGuardOffset {
		return
	}
	state.message = ""
	if item.Raw[gamepack.ItemReadiedOffset] == 0 {
		state.message = a.text(msgItemMustBeReadied)
		return
	}
	scroll, known := a.itemIsScroll(*item)
	if !known {
		return
	}
	if !scroll {
		if spell, ok := gamepack.AIItemSpell(item.Raw, false); ok {
			a.beginItemPageSpell(slot, index, spell, false)
		}
		return
	}
	// 卷軸：overlay-22 entry 12（`0441h`）。閱讀魔法的效果在戰鬥外就是角色身上那一條。
	category, _ := a.itemCategory(*item)
	levels := memberClassLevels(*member)
	readable, revealed := gamepack.ScrollReadable(item.Raw, category,
		combatEffects(member.Effects).Has(gamepack.ReadMagicEffectCode),
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
	state.page.stage, state.page.pending = itemPageScroll, index
	state.page.scroll, state.page.scrollCursor = options, 0
}

// itemPageScrollInput 是 overlay-22 entry 1（`008Ah`）的挑選：上下挑、Enter 放、ESC 回選單。
func (a *app) itemPageScrollInput(slot int) {
	page := &a.equipment.page
	count := len(page.scroll)
	switch {
	case a.justPressed(ebiten.KeyEscape):
		page.stage, page.scroll = itemPagePicking, nil
	case a.justPressed(ebiten.KeyArrowUp):
		page.scrollCursor = (page.scrollCursor + count - 1) % count
	case a.justPressed(ebiten.KeyArrowDown):
		page.scrollCursor = (page.scrollCursor + 1) % count
	case a.justPressed(ebiten.KeyEnter), a.justPressed(ebiten.KeySpace):
		spell := page.scroll[page.scrollCursor].spell
		page.stage, page.scroll = itemPagePicking, nil
		a.beginItemPageSpell(slot, page.pending, spell, true)
	}
}

// beginItemPageSpell 是 overlay-22 entry 5 在戰鬥外的前段（`0C34h..0D1Fh`）：參數表
// `+07h` 為 0 的物品法術問 "Use it? "；其餘照 C)AST 戰鬥外那一條挑對象（spec 119）。
// 處理常式沒讀過的放不出去（與戰鬥中 castFromItem 同一道）。
func (a *app) beginItemPageSpell(slot, index int, spell uint8, scroll bool) {
	state := a.equipment
	if spell == 0 || int(spell) >= len(a.spellParameters) {
		return
	}
	state.page.pending, state.page.spell, state.page.fromScroll = index, spell, scroll
	if !gamepack.ItemUsableOutsideCombat(a.spellParameters[spell]) {
		state.page.stage = itemPageCombatOnly
		state.message = a.text(msgItemCombatOnlyPrompt)
		return
	}
	if a.spellCaster == nil || !a.spellCaster.Implemented(spell) {
		return
	}
	state.page.stage, state.page.target = itemPageTarget, slot
	state.message = fmt.Sprintf(a.text(msgItemPickTarget),
		strings.TrimSpace(a.state.Party[slot].Name), a.spellLabel(spell))
}

// itemPageCombatOnlyInput：Y 就把結果設 1（`0D17h`）——照樣記帳、不放出去（`0D1Fh`）。
func (a *app) itemPageCombatOnlyInput(slot int) {
	state := a.equipment
	switch {
	case a.justPressed(ebiten.KeyY):
		a.spendItemPageUse(slot)
		state.page.stage, state.message = itemPagePicking, ""
	case a.justPressed(ebiten.KeyN), a.justPressed(ebiten.KeyEscape), a.justPressed(ebiten.KeyEnter):
		state.page.stage, state.message = itemPagePicking, ""
	}
}

// itemPageTargetInput 挑對象：上下換人、Enter 放、ESC 放棄（瞄準回 0，不記帳）。
func (a *app) itemPageTargetInput(slot int) error {
	state := a.equipment
	page := &state.page
	count := len(a.state.Party)
	switch {
	case a.justPressed(ebiten.KeyEscape):
		page.stage, state.message = itemPagePicking, ""
	case a.justPressed(ebiten.KeyArrowUp):
		page.target = (page.target + count - 1) % count
	case a.justPressed(ebiten.KeyArrowDown):
		page.target = (page.target + 1) % count
	case a.justPressed(ebiten.KeyEnter), a.justPressed(ebiten.KeySpace):
		member := a.state.Party[slot]
		levels := memberClassLevels(member)
		level := itemCasterLevel(a.spellParameters[page.spell],
			[2]int{int(levels[gamepack.ClassSlotCleric]), int(levels[gamepack.ClassSlotMagicUser])})
		effect, err := a.spellCaster.Cast(page.spell, a.spellParameters, level, a.roller)
		if err != nil {
			return err
		}
		subject := &a.state.Party[page.target]
		applied := applyFieldEffect(subject, effect)
		syncTrainedLibraryCharacter(&a.state, *subject)
		label := a.spellLabel(page.spell)
		if applied {
			state.message = fmt.Sprintf(a.text(msgFieldCastDone),
				strings.TrimSpace(member.Name), label, strings.TrimSpace(subject.Name))
		} else {
			state.message = fmt.Sprintf(a.text(msgFieldCastCombatOnly), label)
		}
		a.spendItemPageUse(slot)
		page.stage = itemPagePicking
	}
	return nil
}

// spendItemPageUse 是 entry 8 的 `1C0Eh..1C7Bh`：結果非 0 就記帳——卷軸抹掉那一行
// （overlay-22 entry 7），其餘數量多於 1 減數量、否則減充能，減到 0 拿掉。
func (a *app) spendItemPageUse(slot int) {
	page := &a.equipment.page
	if page.fromScroll {
		a.scrollEraser(slot, page.pending, page.spell)()
	} else {
		a.spendFoeItem(slot, page.pending)
	}
	a.afterItemPageChange(slot)
}

// itemPageFooter 是選項列（`0F79h..1148h`）：Ready、Use（隊伍選單開的沒有）、Drop、
// Halve（身上不到 10h 件）、Join，最後 Exit。Trade 沒接（spec 149〈未閉合〉）。
func (a *app) itemPageFooter() string {
	state := a.equipment
	switch state.page.stage {
	case itemPageScroll:
		return a.text(msgScrollFooter)
	case itemPageTarget:
		return a.text(msgItemPageTargetFooter)
	}
	parts := []string{a.text(msgCombatItemReady)}
	if !state.page.creationMenu {
		parts = append(parts, a.text(msgCombatItemsFooter))
	}
	parts = append(parts, a.text(msgCombatItemDrop))
	if state.member < len(a.state.Party) && len(a.state.Party[state.member].Inventory) < gamepack.HalveCountLimit {
		parts = append(parts, a.text(msgCombatItemHalve))
	}
	parts = append(parts, a.text(msgCombatItemJoin), a.text(msgCombatItemsExitOnly))
	separator := "　"
	if a.language == languageEnglish {
		separator = " "
	}
	return strings.Join(parts, separator)
}

// drawItemPageOverlay 在清單那一塊畫卷軸的行或挑對象的隊伍；回傳 true 代表畫了、
// 清單就不必再畫。
func drawItemPageOverlay(screen *ebiten.Image, a *app, foreground, accent color.Color) bool {
	page := a.equipment.page
	var rows []string
	cursor := 0
	switch page.stage {
	case itemPageScroll:
		drawText(screen, a.text(msgScrollTitle), equipmentTextLeft, equipmentFirstLine, accent)
		for _, option := range page.scroll {
			marker := "  "
			if option.scribing {
				marker = a.text(msgScrollScribeMark)
			}
			rows = append(rows, marker+a.spellLabel(option.spell))
		}
		cursor = page.scrollCursor
	case itemPageTarget:
		rows = a.partyPickerRows()
		cursor = page.target
	default:
		return false
	}
	top := equipmentFirstLine + equipmentLineHeight
	if page.stage == itemPageTarget {
		top = equipmentFirstLine
	}
	for index, row := range rows {
		prefix, ink := " ", foreground
		if index == cursor {
			prefix, ink = ">", accent
		}
		drawText(screen, prefix+row, equipmentTextLeft, top+index*equipmentLineHeight, ink)
	}
	return true
}
