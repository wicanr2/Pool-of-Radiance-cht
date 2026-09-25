package gamepack

// 敵方 AI（與交給電腦的隊員）用身上的魔法物品：overlay-09 entry 3（`03E3h`）挑哪一件、
// overlay-19 entry 8（`1A86h`）用掉之後怎麼記（spec 096〈entry 3〉，issue #71）。
//
// 物品記錄是 63 bytes（spec 065）。這裡讀的五格：
//
//	+2Eh  型別索引（查物品型別表 `DS:54E0h + 型別 × 16`，`+0` 是類別）
//	+34h  穿戴中（非零才算）
//	+39h  數量（大於 1 時用掉的是一個）
//	+3Ch  次數（0 代表不會用完）
//	+3Dh  法術編號（大於 38h 時減 17h）
//	+3Eh  第 7 位立著的不用

const (
	// AIItemChargesOffset 是物品的使用次數（`+3Ch`）。
	AIItemChargesOffset = 0x3C
	// AIItemSpellOffset 是物品上的法術編號（`+3Dh`）。
	AIItemSpellOffset = 0x3D
	// AIItemGuardOffset 是 entry 3 另外看的一格（`+3Eh`），第 7 位立著就跳過。
	// 語意還沒對上（spec 096〈還沒讀〉），照碼接。
	AIItemGuardOffset = 0x3E
	// aiItemSpellFold 與 aiItemSpellShift：`+3Dh` 大於 38h 時減 17h
	// （overlay-09 `04C7h`、overlay-19 `1B23h` 兩處同形）。
	aiItemSpellFold  = 0x38
	aiItemSpellShift = 0x17
	// AIItemFirstThreshold 是 entry 3 的門檻初值（`03F5h`：`[bp-2] = 7`），
	// 每一輪減一，與 entry 4 同一套（spec 096〈entry 4〉）。
	AIItemFirstThreshold = 7
)

// AIItemSpell 是 overlay-09 entry 3 對一件物品的四道過濾與編號換算（`049Eh..04D5h`）：
//
//	049E  是卷軸（overlay-22 entry 6，類別 0Bh..0Dh） → 跳過
//	04AD  +3Eh >= 80h                                 → 跳過
//	04B7  +34h == 0（沒穿戴）                         → 跳過
//	04C1  +3Dh == 0                                   → 跳過
//	04C7  +3Dh > 38h                                  → 減 17h
//
// scroll 由呼叫端依物品型別表判（卷軸的範圍常數在 `treasure` 那一層）。
func AIItemSpell(raw []byte, scroll bool) (uint8, bool) {
	if scroll || len(raw) <= AIItemGuardOffset {
		return 0, false
	}
	if raw[AIItemGuardOffset] >= 0x80 || raw[ItemReadiedOffset] == 0 {
		return 0, false
	}
	spell := raw[AIItemSpellOffset]
	if spell == 0 {
		return 0, false
	}
	if spell > aiItemSpellFold {
		spell -= aiItemSpellShift
	}
	return spell, true
}

// AIItemCandidate 是物品串列裡的一件：它在串列裡的位置與換算好的法術編號。
// 過不了 AIItemSpell 的不放進來。
type AIItemCandidate struct {
	Index int
	Spell uint8
}

// ChooseAIItem 是 entry 3 的挑選迴圈（`0440h..0516h`）：擲好的次數有幾輪就跑幾輪，
// 每一輪從物品串列的頭（記錄 `+C8h`）依序問 `02EAh(記錄, 法術, 門檻)`，第一件成立的
// 就是它；門檻每一輪減一。**整支不擲骰**——次數骰在呼叫端、閘門之前就擲了（`03FFh`）。
//
// 挑到之後後面幾輪不再掃（`0473h`：已經挑到就直接到 `050Bh`），所以回傳的是第一件。
func ChooseAIItem(candidates []AIItemCandidate, rounds int,
	accept func(spell, threshold uint8) bool) (AIItemCandidate, bool) {
	threshold := uint8(AIItemFirstThreshold)
	for round := 1; round <= rounds; round++ {
		for _, candidate := range candidates {
			if accept(candidate.Spell, threshold) {
				return candidate, true
			}
		}
		threshold--
	}
	return AIItemCandidate{}, false
}

// SpendAIItemUse 是 overlay-19 entry 8 用完之後的記帳（`1C3Fh..1C7Bh`，非卷軸那一支）：
//
//	1C42  +3Ch == 0 → 不動（不會用完）
//	1C4C  +39h > 1  → 數量減一
//	1C5F  否則       → +3Ch 減一；減到 0 → overlay-25 entry 17（`156Ah`）把它從串列摘掉
//
// 回傳 true 代表這一件要從身上拿掉。
func SpendAIItemUse(raw []byte) bool {
	if len(raw) <= AIItemChargesOffset || raw[AIItemChargesOffset] == 0 {
		return false
	}
	if raw[ItemCountOffset] > 1 {
		raw[ItemCountOffset]--
		return false
	}
	raw[AIItemChargesOffset]--
	return raw[AIItemChargesOffset] == 0
}
