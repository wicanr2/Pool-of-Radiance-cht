package main

import (
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// 戰利品選單的 " Detect Exit"（overlay-05 `0E85h`，spec 150〈Detect〉，issue #111）。
//
// 出現條件（exact，overlay-05 SHA-256 `900ea1b8…`）：
//
//	0EA9  9A 66 00 D9 00           ; [bp-1] 有錢、[bp-2] 有物品（每一圈重算）
//	0EE0  80 7E FE 00 / 74 53      ; 沒有物品 → 後綴照舊 " Exit"
//	0EE6..0F37  i = 0..14h：[5CF0h]+17h+i 等於 05h 或 0Bh 就記下那個編號並停
//	              （26 80 7D 17 05、26 80 7D 17 0B：比的是整個位元組，第 7 位立著的
//	              ——還沒記完的——不算）
//	0F40  後綴 = " Detect Exit"（0DFBh）
//
// 按 D（`102Ah`，`3C 44`）：`00E2h:0039h(編號, 0, 0, &[bp-106h])` = overlay-22 entry 5
// （`0C14h`），與探索施法同一支，只是第三個參數 0 不印「誰施了什麼」（`0D23h`）。
// 那一支在 `0E8Dh` 以 overlay-25 entry 16（`9A 70 00 0A 01`）把法術從記憶清掉，再派發
// `DS:6A78h + 編號 × 4`：05h／0Bh 都是 `10D1h`，只呼叫 `08BCh` 把效果碼 05h 掛上去
// （spec 073／074）。效果 05h 之後怎麼改變物品的顯示還沒讀（spec 150 的 DRAFT 表）。

// detectMagicSpells 是 `0F00h`／`0F13h` 比的兩個編號：牧師與法師的 Detect Magic。
var detectMagicSpells = [2]uint8{0x05, 0x0B}

// treasureDetectOption 是目前角色記著、可以按 D 施的那一條；沒有物品或沒記就回 false。
func (a *app) treasureDetectOption() (castOption, bool) {
	if len(a.treasureItems) == 0 || a.currentCharacter < 0 || a.currentCharacter >= len(a.state.Party) {
		return castOption{}, false
	}
	member := a.state.Party[a.currentCharacter]
	// spellOptionsFor 依陣列順序列出記完的法術，與 `0EE6h` 由小到大掃、找到就停同一個次序。
	for _, option := range a.spellOptionsFor(member) {
		if option.ID == detectMagicSpells[0] || option.ID == detectMagicSpells[1] {
			return option, true
		}
	}
	return castOption{}, false
}

// treasureDetect 是按 D：目前角色把那一條施出去（效果掛在自己身上），記憶清掉，
// 回到選單頂端重畫（`1156h` 跳回 `0E9Fh`）。
func (a *app) treasureDetect() error {
	option, ok := a.treasureDetectOption()
	if !ok {
		return nil
	}
	caster := &a.state.Party[a.currentCharacter]
	levels := memberClassLevels(*caster)
	casterLevel := gamepack.CasterLevelFor(a.spellParameters[option.ID],
		int(levels[gamepack.ClassSlotCleric]), int(levels[gamepack.ClassSlotMagicUser]), false)
	effect, err := a.spellCaster.Cast(option.ID, a.spellParameters, casterLevel, a.roller)
	if err != nil {
		return err
	}
	message, _ := a.campSpellEffect(a.currentCharacter, option, effect, casterLevel, a.currentCharacter)
	caster = &a.state.Party[a.currentCharacter]
	caster.Memorised[option.Slot] = 0
	syncTrainedLibraryCharacter(&a.state, *caster)
	a.enterTreasureMain()
	a.statusLine = message
	return nil
}
