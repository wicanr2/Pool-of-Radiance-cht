package gamepack

// 模式 0Ah 那四支（祝福、詛咒、急速、緩速）掛上去的效果，以及它們在戰鬥裡實際改的數值
// （spec 098〈模式 0Ah：效果怎麼掛上去〉、spec 112〈群組 10／18〉，issue #81）。
//
// 掛效果的是 overlay-22 `08BCh` 的 `0A08h..0A5Ah`：參數表 `+0Ah` 非 0 就呼叫
// `0100h:0084h`（overlay-24 entry 20，`1656h`），推的是
// (目標, 碼, 持續 = `07C7h`(法術), 等級, `[bp+0Eh]`, 豁免規則, 豁免結果, 訊息)。
// 效果真正作用的地方不在施法那一側，而是各個戰鬥計算當下問效果系統的那幾個群組：
//
//	群組 10／16（overlay-24 entry 6 `0CE9h`／`0CF6h`）攻擊者與目標身上、改命中骰 `DS:6780h`
//	群組 18（overlay-13 `0055h`／`016Fh`／`0DB2h`）每回合初始化時改攻擊次數與移動的工作值 `DS:6778h`
//
// 每個群組對每個代碼只問一次 `014Dh`，而 `014Dh` 找到的是**最早掛上**的那一個節點
// （overlay-25 entry 27 的線性搜尋），所以同一個碼掛兩個節點不會疊加。

const (
	// BlessEffectCode 是祝福術的參數表 `+0Ah`（`01h`，處理常式 overlay-12 entry 5 `010Fh`）。
	BlessEffectCode uint8 = 0x01
	// CurseEffectCode 是詛咒術的參數表 `+0Ah`（`02h`，overlay-12 entry 6 `0121h`）。
	CurseEffectCode uint8 = 0x02
	// ImmobileEffectCode 是 `3Ah`（overlay-12 entry 53 `145Ah`）：把移動清成 0。
	// 四支法術都不掛它；放在這裡是因為群組 18 的三個碼要一起照順序套。
	ImmobileEffectCode uint8 = 0x3a

	// EffectHasteAgedBit 是節點 `+3` 的位元 4：`27h` 的處理常式（`0C70h..0C89h`）
	// 第一次被問到時立起它，同時讓那個人老一歲（記錄 `+30h` 加一、印 "ages"）。
	EffectHasteAgedBit = 0x10
)

// 命中骰的調整（群組 10／16）在 hit_roll_effects.go，掛效果前的免疫（群組 9）在
// effect_immunity.go（issue #86）。

// RoundRateEffects 記一個人在回合初始化時身上帶著群組 18 的哪幾個碼。
type RoundRateEffects uint8

const (
	// RoundRateHasted 是 `27h`：`0CB4h..0CBBh` 把 `DS:6778h` 左移一位。
	RoundRateHasted RoundRateEffects = 1 << iota
	// RoundRateSlowed 是 `2Ah`：`10CDh..10D8h` 把 `DS:6778h` 除以 2（`idiv`，值是零延伸的）。
	RoundRateSlowed
	// RoundRateImmobile 是 `3Ah`：移動那一次（`DS:677Bh` 非 0）把 `DS:6778h` 清成 0。
	RoundRateImmobile
)

// RoundRateEffectsOf 照群組 18 的順序（`27h 2Ah 3Ah`）看一條串列帶著哪幾個。
func RoundRateEffectsOf(list EffectList) RoundRateEffects {
	var flags RoundRateEffects
	if list.Has(HasteEffectCode) {
		flags |= RoundRateHasted
	}
	if list.Has(SlowEffectCode) {
		flags |= RoundRateSlowed
	}
	if list.Has(ImmobileEffectCode) {
		flags |= RoundRateImmobile
	}
	return flags
}

// AttackRateAfterEffects 是攻擊次數編碼在群組 18 之後的值。原版每回合初始化時
// 先把記錄 `+0A2h`（第二槽走 `0D29h`：`+0A1h` 或遠程武器的射速）放進 `DS:6778h`、
// `DS:677Bh` 清 0，派發群組 18，再交給 `0E58h` 依相位換算成這一相位的次數
// （overlay-13 `0045h..0068h`、`0DA2h..0DC5h`）。全部是 byte 運算。
func AttackRateAfterEffects(rate uint8, flags RoundRateEffects) uint8 {
	if flags&RoundRateHasted != 0 {
		rate <<= 1
	}
	if flags&RoundRateSlowed != 0 {
		rate /= 2
	}
	return rate
}

// MovementAfterEffects 是移動預算在群組 18 之後的值（overlay-13 `0150h..0182h`：
// `DS:677Bh = 1`、`DS:6778h` ＝ 夾過、乘過 2 的移動，派發群組 18，結果寫回 runtime `+6`）。
func MovementAfterEffects(budget uint8, flags RoundRateEffects) uint8 {
	if flags&RoundRateHasted != 0 {
		budget <<= 1
	}
	if flags&RoundRateSlowed != 0 {
		budget /= 2
	}
	if flags&RoundRateImmobile != 0 {
		budget = 0
	}
	return budget
}

// MarkHasteAged 是 `27h` 處理常式開頭那一段：最早掛上的那個 `27h` 節點的 `+3`
// 位元 4 還沒立就立起來，回 true 代表這一下要老一歲（`0C80h` `05 10 00`、`0CB0h`
// `26 FF 45 30`）。沒有 `27h` 或早就立過回 false。
func MarkHasteAged(list EffectList) (EffectList, bool) {
	index, ok := list.IndexOf(HasteEffectCode)
	if !ok || list[index].Payload[effectNodeLevelOffset]&EffectHasteAgedBit != 0 {
		return list, false
	}
	result := append(EffectList(nil), list...)
	result[index].Payload[effectNodeLevelOffset] += EffectHasteAgedBit
	return result, true
}

// ApplySpellEffectNode 是 overlay-24 entry 20 過了免疫與豁免之後的那一段（`16B8h..1716h`）：
//
//	16C7h  找最早掛上的同碼節點
//	16D3h  它有計時而且新的持續比它長 → 摘掉
//	16E6h  新的持續是 0（不計時）→ 摘掉
//	1716h  無論如何都用 entry 10 在尾端掛一個新的
//
// 所以同一個碼可以掛兩個節點（舊的剩得比新的多時舊的留著），但每個群組只問一次
// 最早的那一個，修正不會疊加。
func ApplySpellEffectNode(list EffectList, node EffectNode) EffectList {
	if index, ok := list.IndexOf(node.Code); ok {
		old, fresh := list[index].Duration(), node.Duration()
		if (old > 0 && fresh > old) || fresh == 0 {
			list = list.RemoveAt(index)
		}
	}
	return list.Append(node)
}
