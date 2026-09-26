package gamepack

// 物品的穿戴效果（spec 149，issue #91）。
//
// overlay-19 entry 7（Ready，`14CAh`）在 `14D4h` 看物品 `+3Eh` 是不是大於 7Fh；是的話
// 裝上（`1642h` 寫 `+34h = 1` 之後）與卸下（`151Ah` 寫 `+34h = 0` 之後）各呼叫一次
// overlay-24 entry 1（`0000h`），把 `+3Eh` 當成效果碼，物品本身當成「節點」傳進去：
//
//	1650..1667  entry 1(代碼 = +3Eh, 角色 = DS:5CF0h, 節點 = 物品, 模式 0)   ; 裝上
//	1528..153F  entry 1(代碼 = +3Eh, 角色 = DS:5CF0h, 節點 = 物品, 模式 1)   ; 卸下
//
// entry 1 查 `DS:678Ah` 的表交給那個碼自己的處理常式（spec 112）。80h..8Bh 這十二個碼
// 的常式在 overlay-12 entry 121..125，逐條讀在 spec 149。這一層只動效果串列與力量，
// 誰按了什麼鍵、訊息怎麼印在 cmd/pool-game。

const (
	// WearEffectThreshold 是 overlay-19 `14D4h` 的 `cmp byte es:[di+3Eh], 7Fh; ja`：
	// `+3Eh` 大於它才是穿戴效果碼（小於等於它的是卷軸第三行或 0）。
	WearEffectThreshold = 0x7f
	// ItemGrantedEffectOffset 是物品 `+3Dh`：80h 那一組常式掛上的效果碼
	// （overlay-12 `2F0Ch`／`2F27h` 的 `mov al, es:[di+3Dh]`）。
	ItemGrantedEffectOffset = 0x3d

	// wearGrantedLevel 是 80h 那一組掛節點時的等級（`2F2Fh` 的 `mov al, 0Ch`）。
	wearGrantedLevel = 0x0c

	// wearGiantStrength 與 wearGiantPercentile 是 83h（食人魔之力手套）要的 18/00
	// （overlay-12 `3027h` 的 `mov al, 12h` 與 `302Ah` 的 `mov al, 64h`）。
	wearGiantStrength   = 0x12
	wearGiantPercentile = 0x64
	// wearRequiredStrength 是 87h 的門檻：記錄 `+10h`（力量）小於 13h 就卸下
	// （`315Dh` 的 `cmp byte es:[di+10h], 13h; jae`）。
	wearRequiredStrength = 0x13
	// wearRemovedByCode89 是 89h 卸下時摘掉的效果碼（`3195h` 的 `mov al, 17h`）。
	wearRemovedByCode89 = 0x17
)

// WearMode 是 overlay-24 entry 1 的模式：0 套用（裝上）、1 收尾（卸下）。
type WearMode uint8

const (
	// WearOn 是裝上（overlay-19 `1664h` 推 0）。
	WearOn WearMode = iota
	// WearOff 是卸下（overlay-19 `153Ch` 推 1）。
	WearOff
)

// WearResult 是一次穿戴效果的結果。
type WearResult struct {
	// List 是處理之後的效果串列。
	List EffectList
	// Removed 是從串列摘掉的節點。原版摘節點一律走 overlay-24 entry 2（`0028h`），
	// 節點 `+4` 立著就先以模式 1 叫一次那個碼的處理常式（力量效果的還原就在那裡），
	// 所以呼叫端要對每一個照 `NeedsTeardown` 做收尾，再把 List 寫回去。
	Removed []EffectNode
	// Strength 與 Percentile 是處理之後的力量；只有 Stronger 為真時與傳進來的不同。
	Strength, Percentile uint8
	// Stronger 是 83h 裝上而力量真的往上調了：印 "is stronger"（`2F5Ch`）。
	Stronger bool
	// Refused 是 87h 的「力量不足」：常式把物品 `+34h` 寫回 0，印
	// "Must have Giant Strength"（`3128h`）。
	Refused bool
	// Known 是這個碼的處理常式有讀、有實作。沒有實作的碼（84h 的陣營限制）什麼也不做。
	Known bool
}

// HasWearEffect 是 overlay-19 `14D4h..14E1h` 的判斷。
func HasWearEffect(item []byte) bool {
	return len(item) > ItemEffectOffset && item[ItemEffectOffset] > WearEffectThreshold
}

// wearGrantsItemEffect 是共用 overlay-12 entry 121（`2EECh`）的八個碼
// （`DS:678Ah` 那張表，`docs/audit/dos-effect-handlers.json`）。
func wearGrantsItemEffect(code uint8) bool {
	switch code {
	case 0x80, 0x81, 0x82, 0x85, 0x86, 0x88, 0x8a, 0x8b:
		return true
	}
	return false
}

// ApplyWearEffect 是 overlay-24 entry 1 交給 80h..8Bh 那幾支常式之後的結果：
//
//	80h 81h 82h 85h 86h 88h 8Ah 8Bh → entry 121（2EECh）
//	    模式 0：entry 10 掛上（代碼 = 物品 +3Dh、持續 0、等級 0Ch、不收尾），
//	            再以模式 0、節點 NULL 叫一次那個碼的常式（見 spec 149〈立即派發〉）
//	    模式 1：entry 2（代碼 = 物品 +3Dh、節點 NULL）→ 摘掉最早的那一個
//	83h → entry 122（2F68h），食人魔之力手套
//	    模式 0：entry 18（1158h）以 18/00 調力量；調上去了印 "is stronger"；
//	            再以 entry 10 掛 26h（持續 0、等級 = entry 18 回傳的快照、要收尾）
//	    模式 1：現在正好 18/00 → 摘第一個 26h；否則摘第一個快照解回 18/00 的 26h
//	87h → entry 124（3141h）：模式 0 而力量小於 19 → 物品 +34h = 0，"Must have Giant Strength"
//	89h → entry 125（3186h）：模式 1 → entry 2 摘第一個 17h；模式 0 什麼也不做
//	84h → entry 123（30B1h）：陣營不合就卸下並受傷——傷害那一段（overlay-24 entry 19）
//	      沒接，這裡回 Known = false。
//
// strength／percentile 是角色記錄 `+10h`／`+16h`。
func ApplyWearEffect(list EffectList, item []byte, mode WearMode, strength, percentile uint8) WearResult {
	result := WearResult{List: list, Strength: strength, Percentile: percentile}
	if !HasWearEffect(item) {
		return result
	}
	code := item[ItemEffectOffset]
	switch {
	case wearGrantsItemEffect(code):
		result.Known = true
		granted := item[ItemGrantedEffectOffset]
		if mode == WearOn {
			result.List = list.Append(NewEffectNode(granted, 0, wearGrantedLevel, false))
			return result
		}
		result.List, result.Removed = removeFirstEffect(list, func(node EffectNode) bool {
			return node.Code == granted
		})
	case code == 0x83:
		result.Known = true
		if mode == WearOn {
			result.List, result.Strength, result.Percentile, result.Stronger = ApplyStrengthEffect(
				list, GiantStrengthEffectCode, 0, strength, percentile,
				wearGiantStrength, wearGiantPercentile)
			return result
		}
		// `2F8Dh..2FA4h`：現在是不是正好 18/00。
		atGiant := strength == wearGiantStrength && percentile == wearGiantPercentile
		result.List, result.Removed = removeFirstEffect(list, func(node EffectNode) bool {
			if node.Code != GiantStrengthEffectCode {
				return false
			}
			// `2FD4h..2FE1h` 比的是節點 `+0`（代碼 26h）是不是小於 80h，恆成立——
			// 所以 18/00 時摘的是第一個 26h，不管它是不是這雙手套掛的。照位元組做。
			if atGiant {
				return true
			}
			value, snapshotPercentile := decodeStrengthSnapshot(node.Payload[effectNodeLevelOffset])
			return value == wearGiantStrength && snapshotPercentile == wearGiantPercentile
		})
	case code == 0x87:
		result.Known = true
		if mode == WearOn && strength < wearRequiredStrength {
			result.Refused = true
		}
	case code == 0x89:
		result.Known = true
		if mode == WearOff {
			result.List, result.Removed = removeFirstEffect(list, func(node EffectNode) bool {
				return node.Code == wearRemovedByCode89
			})
		}
	}
	return result
}

// removeFirstEffect 摘掉第一個符合的節點（overlay-24 entry 2 的 `0050h..0072h` 走訪，
// 或 83h 常式自己的走訪），回傳剩下的串列與摘掉的那一個。
func removeFirstEffect(list EffectList, match func(EffectNode) bool) (EffectList, []EffectNode) {
	for index, node := range list {
		if match(node) {
			return list.RemoveAt(index), []EffectNode{node}
		}
	}
	return list, nil
}

// ItemUsableOutsideCombat 是 overlay-22 entry 5 `0C3Fh` 的 `cmp byte [di+319Bh], 0`：
// 參數表 `+07h` 為 0 的法術戰鬥外放不出去。物品放的會問 "Use it?"，答 Y 照樣記帳
// （`0D17h` 結果 = 1）但不放（`0D1Fh`）。spec 144〈探索中（非戰鬥）的差異〉。
func ItemUsableOutsideCombat(p SpellParameters) bool {
	return p.Raw[spellParameterArea] != 0
}
