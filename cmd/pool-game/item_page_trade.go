package main

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	poolcharacter "github.com/wicanr2/Pool-of-Radiance-cht/internal/character"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/creation"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
	pooltreasure "github.com/wicanr2/Pool-of-Radiance-cht/internal/treasure"
)

// 物品頁剩下的三件事（spec 149，issue #105）：
//
//   - Trade（overlay-19 entry 13，`173Bh`）：挑一個隊員，entry 9（`274Fh`）的負重檢查過了
//     才把物品交過去，不過印 "Overloaded"。
//   - 穿戴效果 84h（overlay-12 entry 123）：陣營不合就卸下，受 `+3Dh ÷ 16` 點魔法傷害
//     （overlay-24 entry 19）。
//   - 戰鬥外 Use 的 "uses an item"（overlay-19 entry 8 `1B2Dh..1BB8h`）與 " Use" 的兩道門
//     （`0F8Ch..0FA1h`：記錄 `+10Dh` 非 0、`[4933h]+1CAh` 為 0）。

// 這一段訊息另開 `iota + 4400`（#105），在 init 登記進 messageKeys，重號直接 panic。
const (
	msgItemTrade messageID = iota + 4400
	msgItemTradeWhom
	msgItemTradeOverloaded
	msgItemUsesAnItem
	msgItemWearHurt
	msgItemWearHurtOne
	msgItemWearDown
	msgItemWearDying
	msgItemWearKilled
)

func init() {
	for id, key := range map[messageID]string{
		msgItemTrade:           "ui.itemTrade",
		msgItemTradeWhom:       "ui.itemTradeWhom",
		msgItemTradeOverloaded: "ui.itemTradeOverloaded",
		msgItemUsesAnItem:      "ui.itemUsesAnItem",
		msgItemWearHurt:        "ui.itemWearHurt",
		msgItemWearHurtOne:     "ui.itemWearHurtOne",
		msgItemWearDown:        "ui.itemWearDown",
		msgItemWearDying:       "ui.itemWearDying",
		msgItemWearKilled:      "ui.itemWearKilled",
	} {
		if existing, ok := messageKeys[id]; ok {
			panic(fmt.Sprintf("message id %d is already %q", id, existing))
		}
		messageKeys[id] = key
	}
}

const (
	// itemPageUsesNotice 是 entry 8 戰鬥外印完 "uses an item" 與物品名之後等的那一拍
	// （`1BB3h` overlay-37 entry 13），等完才交給 overlay-22 entry 5。
	itemPageUsesNotice itemPageStage = itemPageTarget + 1 + iota
	// itemPageTrade 是 overlay-25 entry 42（`2C81h`）的 "Trade with Whom?"。
	itemPageTrade
)

// antiMagicAddress 是 `[4933h]+1CAh` 的 ECL 位址（class 0：`6E00h + addr × 2 ≡ 1CAh`）。
// 唯一的腳本寫入端是 ECL8 block 16 `9BEEh` 的 SAVE，旁邊 `9C22h` 印 "YOU SENSE AN
// ANTI-MAGIC SHELL AROUND YOU."；區塊載入時 overlay-07 `025Ah` 清成 0（block_load.go）。
const antiMagicAddress = 0x49E5

// itemPageUsable 是 " Use" 接不接（`0F8Ch..0FBDh`）：隊伍選單開的（`DS:4954h` 為 0）不接；
// 其餘要記錄 `+10Dh` 非 0，而且 `[4933h]+1CAh` 為 0。
//
// `+10Dh` 在 remake 沒有自己的欄位：原版的寫入端都跟著 `+10Ch` 一起寫——扣血把狀態寫成
// 0／1 以外時寫 0（overlay-25 entry 28，spec 084）、逃離與轉化不死生物寫 0、建角與神殿的
// 石化解除寫 1 並把狀態寫 0——所以讀成「狀態不大於 1」（strong inference）。
func (a *app) itemPageUsable(slot int) bool {
	if a.equipment == nil || a.equipment.page.creationMenu || slot < 0 || slot >= len(a.state.Party) {
		return false
	}
	if a.state.Party[slot].Status > gamepack.AliveStateMax {
		return false
	}
	return a.eventMachine == nil || a.eventMachine.Memory[antiMagicAddress] == 0
}

// itemPageTradable 是 " Trade" 接不接（`0FF6h..101Ch`）：不在戰鬥中（物品頁一定不是），
// 而且與商店的 Sell 同一個角色條件（shopCanSell，spec 067）。
func (a *app) itemPageTradable(slot int) bool {
	return slot >= 0 && slot < len(a.state.Party) && shopCanSell(a.state.Party[slot])
}

// startItemPageTrade 是 'T'（`1328h..1341h`）：先過 entry 20（`0D72h`）的放手檢查，過了才進
// entry 13。
func (a *app) startItemPageTrade(slot, index int) {
	state := a.equipment
	item := a.state.Party[slot].Inventory[index]
	state.message = ""
	if pooltreasure.SellNeedsUnready(item.Raw) {
		state.message = a.text(msgItemMustBeUnreadied)
		return
	}
	state.page.pending, state.page.trading = index, true
	if category, ok := a.itemCategory(item); ok && pooltreasure.SellNeedsScribeConfirm(item.Raw, category) {
		state.page.stage = itemPageScribeConfirm
		state.message = fmt.Sprintf(a.text(msgShopSellScribe), strings.TrimSpace(a.state.Party[slot].Name))
		return
	}
	a.askItemPageTrade(slot)
}

// askItemPageTrade 是 entry 13 的前半：`1757h` "Trade with Whom?"，overlay-25 entry 42
// 從 `DS:467Ch` 那一個人起挑（entry 5 `0AD9h` 開頁時設成自己，每一次交易之後設成對方）。
func (a *app) askItemPageTrade(slot int) {
	state := a.equipment
	page := &state.page
	page.trading = false
	page.stage = itemPageTrade
	page.target = slot
	if page.tradeTarget > 0 && page.tradeTarget-1 < len(a.state.Party) {
		page.target = page.tradeTarget - 1
	}
	state.message = a.text(msgItemTradeWhom)
}

// itemPageTradeInput 是 entry 42 的選擇：上下換人、Enter 選、ESC 放棄（回傳 NULL，
// `1774h` 什麼也不做）。整條隊伍都選得到，包括自己。
func (a *app) itemPageTradeInput(slot int) {
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
		page.stage, state.message = itemPagePicking, ""
		page.tradeTarget = page.target + 1
		a.tradeMemberItem(slot, page.pending, page.target)
	}
}

// tradeMemberItem 是 entry 13 的後半（`1782h..17D6h`）：entry 9 說超重就以 overlay-25
// entry 19 印 "Overloaded"；否則 entry 18 把物品複製一份接在對方串列最後、entry 17
// 從自己身上拿掉、entry 7 重算對方。對方是自己時等於把它搬到最後一格。
func (a *app) tradeMemberItem(slot, index, target int) {
	state := a.equipment
	party := a.state.Party
	if index < 0 || index >= len(party[slot].Inventory) || target < 0 || target >= len(party) {
		return
	}
	item := party[slot].Inventory[index]
	recipient := party[target]
	raws := make([][]byte, len(recipient.Inventory))
	for i := range recipient.Inventory {
		raws[i] = recipient.Inventory[i].Raw
	}
	overloaded, err := poolcharacter.ReceiveOverloaded(recipient.Abilities[gamepack.AbilityStrength],
		recipient.ExceptionalStrength, raws, recipient.Money, item.Raw)
	if err != nil {
		state.message = err.Error()
		return
	}
	if overloaded {
		state.message = a.text(msgItemTradeOverloaded)
		return
	}
	copied := poolsave.Item{Name: item.Name, Raw: append([]byte(nil), item.Raw...)}
	party[target].Inventory = append(party[target].Inventory, copied)
	member := &party[slot]
	member.Inventory = append(member.Inventory[:index:index], member.Inventory[index+1:]...)
	syncTrainedLibraryCharacter(&a.state, party[target])
	a.afterItemPageChange(slot)
}

// useNoticeOnPage 是 entry 8 戰鬥外那一段（`1B2Dh..1BB8h`）：entry 20(記錄, "uses an item",
// 0Ah, 0) 印名字與那一句（不停），`1B94h` entry 1 以旗標 (1, 16h, 0, 1) 在第 22 列欄 1 印
// 物品名（前面沒有 "Item:"），`1BB3h` 等一拍，`1BB8h` entry 21 清掉，才進 overlay-22 entry 5。
// 那一拍是遊戲速度的 225 ms 倍；速度為 0 就不等，直接往下。
func (a *app) useNoticeOnPage(slot, index int, spell uint8) {
	state := a.equipment
	member := a.state.Party[slot]
	state.message = fmt.Sprintf(a.text(msgItemUsesAnItem), strings.TrimSpace(member.Name),
		strings.TrimSpace(member.Inventory[index].Name))
	ticks := a.speedDelayTicks()
	if ticks <= 0 {
		state.message = ""
		a.beginItemPageSpell(slot, index, spell, false)
		return
	}
	state.page.stage, state.page.pending, state.page.spell, state.page.ticks = itemPageUsesNotice, index, spell, ticks
}

// itemPageUsesNoticeInput 在那一拍裡不收鍵；拍子數完才交給 entry 5。
func (a *app) itemPageUsesNoticeInput(slot int) {
	page := &a.equipment.page
	if page.ticks > 1 {
		page.ticks--
		return
	}
	page.stage, page.ticks, a.equipment.message = itemPagePicking, 0, ""
	a.beginItemPageSpell(slot, page.pending, page.spell, false)
}

// memberAlignment 是記錄 `+0A0h`：NPC 讀它自己的記錄，玩家建的角色取建角的編號
// （creation.Alignments 的順序就是那個值，與 rememberPartySaveRecord 同）。
func memberAlignment(member poolsave.Character) (uint8, bool) {
	if member.NPC && len(member.Record) > recordAlignmentOffset {
		return member.Record[recordAlignmentOffset], true
	}
	for value, choice := range creation.Alignments {
		if choice.ID == member.AlignmentID {
			return uint8(value), true
		}
	}
	return 0, false
}

// alignedWear 是 84h 戴上時 overlay-12 entry 123 那一段：陣營不合就把 `+34h` 寫回 0，
// 再經 overlay-24 entry 19（`133Ah`）受 `+3Dh ÷ 16` 點魔法傷害（`DS:6777h = 8`、規則 0、
// 沒豁免）。回傳處理過的串列（群組 6 可能動它）與要印的那一句；沒卸下就原樣回傳。
func (a *app) alignedWear(slot int, raw []byte, mode gamepack.WearMode, list gamepack.EffectList,
	state *tacticalState, cell int) (gamepack.EffectList, string) {
	member := &a.state.Party[slot]
	alignment, known := memberAlignment(*member)
	if !known {
		return list, ""
	}
	refused, damage := gamepack.AlignedWearDamage(raw, mode, alignment)
	if !refused {
		return list, ""
	}
	raw[gamepack.ItemReadiedOffset] = 0
	// `1351h`：傷害進 `DS:6776h` 之後先派發受傷那一個的群組 6（`DS:6779h` 是 0）。
	outcome := gamepack.SpellDamageEffects{Effects: list, DamageFlags: gamepack.AlignedWearDamageFlags,
		Roll: a.rollDice}.Apply(damage)
	list, damage = outcome.Effects, outcome.Damage
	if damage <= 0 {
		// `137Ah`：傷害為 0 整段跳過，什麼也不印。
		return list, ""
	}
	name := strings.TrimSpace(member.Name)
	// `1384h..13DCh` 與 `14A8h..14D8h`："takes N points of damage " 或 "takes 1 point of
	// damage "，`6777h` 只有 8 這一位 → 接 "from Magic"；overlay-25 entry 26 戰鬥外以
	// entry 20(記錄, 字串, 0Ah, 1) 印出並停一拍。
	line := fmt.Sprintf(a.text(msgItemWearHurt), name, damage)
	if damage == 1 {
		line = fmt.Sprintf(a.text(msgItemWearHurtOne), name)
	}
	var next uint8
	if state != nil && cell > 0 && cell < len(state.HitPoints) {
		next = a.woundWearer(state, cell, damage)
	} else {
		result := gamepack.ApplyDamage(member.CurrentHP, member.Status, damage)
		member.CurrentHP, member.Status = result.HitPoints, result.State
		next = result.State
	}
	if next > gamepack.AliveStateMax {
		line += "  " + a.wearDownLine(name, next)
	}
	return list, line
}

// wearDownLine 是 entry 19 的 `1560h..15F6h`：`+10Dh` 變成 0 就印 "Goes Down"；狀態 5
// 接 ", and is Dying"；狀態在 `1310h` 那個集合（6、7、8）改成 "is killed"。
func (a *app) wearDownLine(name string, status uint8) string {
	switch {
	case status >= gamepack.DeadState && status <= 8:
		return fmt.Sprintf(a.text(msgItemWearKilled), name)
	case status == gamepack.DyingState:
		return fmt.Sprintf(a.text(msgItemWearDying), name)
	}
	return fmt.Sprintf(a.text(msgItemWearDown), name)
}

// woundWearer 是戰鬥中那一份：生命值在盤面那一格。扣法與 overlay-25 entry 28（`2266h`，
// gamepack.ApplyDamage）相同；受傷打斷施法（`1500h..155Fh`）與倒下離場照其他傷害那一套。
func (a *app) woundWearer(state *tacticalState, cell int, damage int) uint8 {
	target := uint8(cell)
	current := uint8(0)
	if cell < len(state.States) {
		current = state.States[cell]
	}
	result := gamepack.ApplyDamage(state.HitPoints[cell], current, damage)
	state.HitPoints[cell] = result.HitPoints
	a.woundCombatant(state, target, damage)
	defer a.announceLostSpells(state)
	if result.Downed {
		state.rememberFootprint(cell)
		state.Roster[cell].FootprintClass = 0
		state.Scores[cell] = 0
		state.States[cell] = result.State
	}
	return result.State
}
