package main

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/treasure"
)

// 玩家戰鬥指令列的 T）urn 與 U）se（spec 129〈T 與 U〉、issue #75）。
//
// overlay-08 entry 4 的指令迴圈（`0375h` 起，`0381h` 取鍵）：
//
//	03E9  'U' → overlay-19 entry 6（0EFBh）物品選單(&結果)，overlay-13 entry 8（D29h），
//	            結果為 0 → overlay-25 entry 41（2BC1h）
//	0427  'T' → overlay-13 entry 12（116Ah）(行動者)，再 overlay-25 entry 34（266Dh），
//	            回傳值寫進結果（這一個行動用掉了）
//
// 兩支都與 AI 共用：T 是 AI entry 2（foe_turn_undead.go）呼叫的同一支 116Ah，
// U 最後落到 AI entry 3 交出去的同一支 overlay-19 entry 8（1A86h，foe_items.go）。
// 按鍵只在指令列上有那一段時才收（combatSegmentShown）。
//
// 物品選單在戰鬥中（`DS:4954h` == 5）的選項是 Ready、Use、Drop、Halve、Join
// （`0F79h..1119h`；Trade 與 Sell、Id 戰鬥中不接）。Use 在這裡，其餘四項與卷軸在
// combat_item_menu.go（spec 144）。
//
// Use 那一條（overlay-19 `12B0h..1325h`）：
//
//	12B7  物品 +34h（穿戴中）== 0 → 印 "Must be Readied"（`0EC6h`），留在選單
//	12DE  是卷軸（overlay-22 entry 6）→ 交給 entry 8
//	12EA  否則 +3Dh > 0 而且 +3Eh < 80h → 交給 entry 8；都不是就什麼也不做
//	1307  entry 8(物品, &結果)
//	1318  結果非 0 → 離開選單（行動用掉了）
//
// " Use" 本身只在 `0FBFh` runtime +2 非 0（沉默、咳嗽清掉）時才接進選項。

// 這一段訊息另開 `iota + 1740`（#75），在 init 登記進 messageKeys，重號直接 panic。
const (
	msgCombatItemsTitle messageID = iota + 1740
	msgCombatItemsFooter
	msgCombatItemsExitOnly
	msgItemMustBeReadied
)

func init() {
	for id, key := range map[messageID]string{
		msgCombatItemsTitle:    "ui.combatItemsTitle",
		msgCombatItemsFooter:   "ui.combatItemsFooter",
		msgCombatItemsExitOnly: "ui.combatItemsExitOnly",
		msgItemMustBeReadied:   "ui.itemMustBeReadied",
	} {
		if existing, ok := messageKeys[id]; ok {
			panic(fmt.Sprintf("message id %d is already %q", id, existing))
		}
		messageKeys[id] = key
	}
}

// 物品選單在右側資訊欄的位置：第四行（武器）底下起，到說明那一段（combatNoteLine）
// 之前放得下五行，多的捲動。版面是 remake 的呈現，原版的選單版面沒量。
const (
	combatItemsTitleLine = combatInfoLine4 + 18
	combatItemsLines     = 5
)

// combatItemMenu 是戰鬥中開著的物品選單（overlay-19 entry 6）。
type combatItemMenu struct {
	cursor int
	// stage 是目前在等哪一種輸入（combat_item_menu.go）。
	stage combatItemStage
	// pending 是正在確認丟掉、或正在挑卷軸那一件的索引。
	pending int
	// scroll 與 scrollCursor 是 overlay-19 entry 12 列出來的那幾行。
	scroll       []scrollOption
	scrollCursor int
}

// combatItemUse 是正在瞄準、還沒放出去的那一件（overlay-19 entry 8 把 `DS:6CB3h`
// 立成 1 的那一段）。瞄準收尾時由 finishCombatItem 接手，放棄時由 abortCombatItem。
type combatItemUse struct {
	slot  int
	item  int
	spell uint8
	// scroll 為真時用完抹掉那一行（overlay-22 entry 7），否則照 +3Ch 記帳。
	scroll bool
	// keepTurn 是參數表 `+0Bh` 為 0 的法術：entry 8 `1BF4h` 不呼叫 entry 34（spec 144）。
	keepTurn bool
}

// combatCommandInput 是指令迴圈裡 T 與 U 那兩支，外加開著的物品選單。
// 回傳 true 代表這一影格的按鍵用在這裡。
func (a *app) combatCommandInput(state *tacticalState) (bool, error) {
	if a.combatItems != nil {
		return true, a.combatItemInput(state)
	}
	if state.Mover == 0 || state.Moving {
		return false, nil
	}
	if a.justPressed(ebiten.KeyT) && a.combatSegmentShown(gamepack.CombatCommandTurn) {
		return true, a.playerTurnUndead(state)
	}
	if a.justPressed(ebiten.KeyU) && a.combatSegmentShown(gamepack.CombatCommandUse) {
		a.combatItems = &combatItemMenu{}
		return true, nil
	}
	return false, nil
}

// playerTurnUndead 是 `0427h`：不先挑目標，直接進 116Ah——挑不到也照樣擲骰、
// 立 runtime +11h、用掉這個行動（AI 那一側才先問 1352h）。
func (a *app) playerTurnUndead(state *tacticalState) error {
	if a.turnUndeadTable == nil {
		return nil
	}
	mover := state.Mover
	caster, ok := a.foeSpellcasterFor(state, mover)
	if !ok {
		return nil
	}
	opposing, candidates, err := a.turnUndeadCandidates(state, mover)
	if err != nil {
		return err
	}
	a.turnUndead(state, mover, caster.levels[0], opposing, candidates)
	if state.Finished {
		return a.finishCombat(state.Outcome)
	}
	return nil
}

// combatItemsUsable 是 `0F97h..0FCCh` 接不接 " Use"：`[4933h]+1CAh`（這一版恆為 0）、
// 戰鬥中 runtime +2 非 0——沉默（15h）與咳嗽（1Eh）清它，與 AI 的 entry 3 同一道。
func (state *tacticalState) combatItemsUsable(index int) bool {
	return !state.hasEffect(index, silenceEffectCode) &&
		!state.hasEffect(index, gamepack.StinkingCloudEffectCode)
}

// combatItemInput 處理物品選單的按鍵：上下挑、U 或 Enter 用、ESC 離開。
func (a *app) combatItemInput(state *tacticalState) error {
	slot, ok := a.moverPartyIndex(state.Mover)
	if !ok || len(a.state.Party[slot].Inventory) == 0 {
		a.combatItems = nil
		return nil
	}
	menu := a.combatItems
	count := len(a.state.Party[slot].Inventory)
	menu.cursor = (menu.cursor%count + count) % count
	if handled, err := a.combatItemMenuInput(state, slot); handled || err != nil {
		return err
	}
	switch {
	case a.justPressed(ebiten.KeyEscape):
		a.combatItems = nil
	case a.justPressed(ebiten.KeyArrowUp):
		menu.cursor = (menu.cursor + count - 1) % count
	case a.justPressed(ebiten.KeyArrowDown):
		menu.cursor = (menu.cursor + 1) % count
	case a.justPressed(ebiten.KeyU), a.justPressed(ebiten.KeyEnter):
		if !state.combatItemsUsable(int(state.Mover)) {
			return nil
		}
		return a.useCombatItem(state, slot, menu.cursor)
	}
	return nil
}

// useCombatItem 是 `12B0h` 的 Use 加上 entry 8 的前半：挑出物品上的法術、印一句，
// 然後與施法同一支 overlay-22 entry 5 瞄準（玩家的 `+10Fh` 是 0，所以是玩家自己瞄）。
func (a *app) useCombatItem(state *tacticalState, slot, index int) error {
	member := a.state.Party[slot]
	item := member.Inventory[index]
	if len(item.Raw) <= gamepack.AIItemGuardOffset {
		return nil
	}
	if item.Raw[gamepack.ItemReadiedOffset] == 0 {
		a.tacticalStatus(state, a.text(msgItemMustBeReadied))
		return nil
	}
	scroll, known := a.itemIsScroll(item)
	if !known {
		return nil
	}
	if scroll {
		// 卷軸走 entry 8 的 `1AA9h`：overlay-19 entry 12 讓玩家從卷軸上挑一條，放完由
		// overlay-22 entry 7（3243h）抹掉那一條（combat_item_menu.go）。
		category, _ := a.itemCategory(item)
		a.openScrollPick(state, slot, index, category)
		return nil
	}
	// `12EAh` 與 entry 8 的 `1ACEh`／`1B23h` 與 AI 的 entry 3 是同一組過濾與換算
	// （+3Eh < 80h、+3Dh 非 0、大於 38h 減 17h；穿戴中上面已經看過）。
	spell, ok := gamepack.AIItemSpell(item.Raw, false)
	if !ok {
		return nil
	}
	// 參數表 `+0Bh` 為 0 的也放得出去：overlay-22 entry 5 在戰鬥中（`0C2Ah`）不看
	// 它，只有 entry 8 `1BF4h` 看，而那裡只決定要不要呼叫 entry 34（spec 144）。
	return a.castFromItem(state, slot, index, spell, false)
}

// itemIsScroll 是 overlay-22 entry 6（31F6h）：物品型別表的類別落在卷軸那一段。
// 型別表不在就回 known = false（失敗即關閉）。
func (a *app) itemIsScroll(item poolsave.Item) (scroll, known bool) {
	category, ok := a.itemCategory(item)
	if !ok {
		return false, false
	}
	return category >= treasure.ScrollCategoryFirst && category <= treasure.ScrollCategoryLast, true
}

// itemCasterLevel 是物品放法術時的施法者等級：entry 8 在呼叫 overlay-22 entry 5 之前
// 把 `DS:6CB3h` 立成 1（`1BBDh`），`26F8h` 看到它就把牧師／法師的法術當成 6 級、
// 物品效果（參數表 +0 == 2）12 級。戰鬥中也一樣。
func itemCasterLevel(params gamepack.SpellParameters, levels [2]int) int {
	return gamepack.CasterLevelFor(params, levels[0], levels[1], true)
}

// finishCombatItem 是瞄準收好之後 entry 8 的後半：overlay-22 entry 5 放出去、戰鬥中
// 參數表 +0Bh 非 0 就 entry 34 結束行動（`1BF4h`），接著記帳（`1C0Eh`，spendFoeItem）。
// 回傳 false 代表現在不是在用物品。
func (a *app) finishCombatItem(option castOption, targets spellTargets) (bool, error) {
	use := a.combatItem
	if use == nil {
		return false, nil
	}
	a.combatItem = nil
	state := a.tactical
	if state == nil || use.slot >= len(a.state.Party) {
		return true, nil
	}
	member := &a.state.Party[use.slot]
	levels := memberClassLevels(*member)
	casting := spellCasting{
		name:      member.Name,
		member:    member,
		partySlot: use.slot,
		level: itemCasterLevel(a.spellParameters[option.ID],
			[2]int{int(levels[gamepack.ClassSlotCleric]), int(levels[gamepack.ClassSlotMagicUser])}),
		consume:  a.itemSpender(use.slot, use.item),
		keepTurn: use.keepTurn,
	}
	if use.scroll {
		casting.consume = a.scrollEraser(use.slot, use.item, use.spell)
	}
	if err := a.castSpell(state, casting, option, targets); err != nil {
		return true, err
	}
	if state.Finished {
		return true, a.finishCombat(state.Outcome)
	}
	return true, nil
}

// abortCombatItem 是瞄準時放棄（overlay-22 `0EE7h..0F0Bh`）碰上物品的那一支：
// `0EFBh` 看 `DS:6CB3h` 非 0 就不清記憶；回到 entry 8 之後 `1BF4h` 照樣 entry 34、
// 結果 1，所以這一件還是記帳（卷軸抹掉那一行）。回傳 handled 為 false 代表現在不是
// 在用物品。
//
// 參數表 `+0Bh` 為 0 的沒有 entry 34：結果留著瞄準的回傳 0，不記帳，物品選單的迴圈
// 在戰鬥中接著轉（`1318h` 結果 0 → 重畫、留在選單），這個行動也沒用掉——keep 為真。
func (a *app) abortCombatItem() (handled, keep bool) {
	use := a.combatItem
	if use == nil {
		return false, false
	}
	a.combatItem = nil
	if use.keepTurn {
		a.combatItems = &combatItemMenu{cursor: use.item}
		return true, true
	}
	if use.scroll {
		a.scrollEraser(use.slot, use.item, use.spell)()
	} else {
		a.itemSpender(use.slot, use.item)()
	}
	return true, false
}

// itemSpender 包一次 spendFoeItem：同一件只記一次帳。
func (a *app) itemSpender(slot, item int) func() {
	spent := false
	return func() {
		if spent {
			return
		}
		spent = true
		a.spendFoeItem(slot, item)
	}
}

// drawCombatItems 把物品選單畫在盤面右邊，選項列畫在指令列那條基線上。
func drawCombatItems(screen *ebiten.Image, a *app, foreground, accent color.Color) {
	state := a.tactical
	if a.combatItems == nil || state == nil {
		return
	}
	slot, ok := a.moverPartyIndex(state.Mover)
	if !ok {
		return
	}
	if a.combatItems.stage == combatItemScroll {
		drawScrollOptions(screen, a, foreground, accent)
		drawText(screen, a.combatItemFooter(state, slot), 0, footerBaseline, accent)
		return
	}
	drawText(screen, a.text(msgCombatItemsTitle), combatInfoLeft, combatItemsTitleLine, accent)
	inventory := a.state.Party[slot].Inventory
	first := 0
	if a.combatItems.cursor >= combatItemsLines {
		first = a.combatItems.cursor - combatItemsLines + 1
	}
	for index := first; index < len(inventory) && index < first+combatItemsLines; index++ {
		item := inventory[index]
		cursor, ink := " ", foreground
		if index == a.combatItems.cursor {
			cursor, ink = ">", accent
		}
		marker := "  "
		if len(item.Raw) > gamepack.ItemReadiedOffset && item.Raw[gamepack.ItemReadiedOffset] != 0 {
			marker = a.text(msgEquipmentReadyMark)
		}
		drawText(screen, cursor+marker+strings.TrimSpace(item.Name),
			combatInfoLeft, combatItemsTitleLine+18+(index-first)*18, ink)
	}
	drawText(screen, a.combatItemFooter(state, slot), 0, footerBaseline, accent)
}
