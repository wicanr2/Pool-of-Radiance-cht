package gamepack

// 建角依種族掛的效果（spec 145，issue #88）。
//
// overlay-16（SHA-256 `a142d8a8…`）`078Ah..0905h`：記錄 `+2Eh` 寫入種族之後依它分支，
// 每個碼推一次 overlay-24 entry 10（`9A 52 00 00 01`，`0E54h`）：
//
//	push 記錄；B0 碼 50；31 C0 50（持續 0）；B0 FF 50（+3 = FFh）；B0 00 50（+4 = 0）
//
// entry 10 把 `[bp+0Ah]` 寫 `+1`、`[bp+08h]` 寫 `+3`、`[bp+06h]` 寫 `+4`（`0EDCh..0EF6h`），
// 所以每個節點都是 `碼 00 00 FF 00`：持續 0（永久，spec 069）、`+3 = FFh`（解除魔法跳過，
// spec 098 的 `23BEh`）、不需收尾。人類（7）與其他值只寫記錄 `+0C0h = 2`，不掛效果。

// RaceEffectCodes 回傳原版記錄 `+2Eh` 為 race 時建角掛的碼，照 overlay-16 的推入順序。
func RaceEffectCodes(race uint8) []uint8 {
	switch race {
	case 1: // 矮人 `07D9h`：cmp al,1
		return []uint8{0x5a, 0x61, 0x1a, 0x2f}
	case 2: // 精靈 `08B0h`：cmp al,2
		return []uint8{0x6b}
	case 3: // 侏儒 `0845h`：cmp al,3
		return []uint8{0x61, 0x12, 0x2f, 0x30}
	case 4: // 半精靈 `08D6h`：cmp al,4
		return []uint8{0x7c}
	case 5: // 半身人 `079Bh`：cmp al,5
		return []uint8{0x5a, 0x61}
	}
	return nil
}

// RaceEffectNode 是建角掛上的一個節點：`NewEffectNode(碼, 0, FFh, false)`。
func RaceEffectNode(code uint8) EffectNode {
	return NewEffectNode(code, 0, EffectUndispellable, false)
}

// RaceEffects 是 race 建角時的整條串列（沒有就是 nil）。
func RaceEffects(race uint8) EffectList {
	codes := RaceEffectCodes(race)
	if len(codes) == 0 {
		return nil
	}
	list := make(EffectList, 0, len(codes))
	for _, code := range codes {
		list = append(list, RaceEffectNode(code))
	}
	return list
}
