package gamepack

// 用物品放出去的法術怎麼算射程（spec 098〈物品的射程〉，issue #85）。
//
// overlay-22 entry 3（`071Dh`，`retf 2`，參數是法術編號）是射程：
//
//	0723  80 3E B3 6C 00 / 75 3A     DS:6CB3h 非 0（用物品）→ 0764h
//	072E  9A D4 00 0A 01             否則等級 = overlay-25 entry 36（26F8h）
//	0748  F7 EA                      射程 = +2 + +3 × 等級
//	0782  8A 85 97 31 / B9 06 00 / F7 E9    用物品：射程 = +2 + +3 × 6
//	0792  結果是 0 而 +6 非 0 → 1；FFh → 1
//
// 用物品那一支乘的是常數 6，**不看**法術屬於誰：物品效果（參數表 +0 是 3）的法術
// 在施法者等級那一邊是 12 級（26F8h，CasterLevelFor），射程卻照 6 級算。呼叫端是
// overlay-13 `1E09h`（AI 挑目標，`1EF2h`）與玩家瞄準那一步，兩邊都在 overlay-19
// `1BBDh` 立起 `DS:6CB3h` 之後。
const ItemRangeLevel = 6

// ItemRange 是 `0764h..078Fh`：用物品放這條法術時的射程。
func (p SpellParameters) ItemRange() int { return p.Range(ItemRangeLevel) }
