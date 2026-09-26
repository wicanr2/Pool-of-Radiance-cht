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

// ---- 只掛效果的那一批（`08BCh` 的通用路，issue #89）----
//
// 參數表 `+0Ah` 非 0 而處理常式沒有自己的算法的那幾十支，全部走 overlay-22 `08BCh` 的
// `090Ch..0A62h`：對 `20AEh` 收好的表逐格擲豁免（`096Bh`）、`+2` 是 FFh 的先擲一次命中
// （`0997h..09DBh`）、持續 `07C7h`、然後 `0A5Ah` 呼叫 overlay-24 entry 20（spec 098
// 〈只掛效果的那一批〉）。效果真正作用在各群組被問到的時候；這一段收的是那幾個群組裡
// 屬於這批法術的碼（群組 6／11／12 各自只接到下面列的那幾個，其餘寫在 spec 112〈OPEN〉）。

// 這一批法術掛的碼裡，要在群組 6／11／12 裡作用的。
const (
	// FriendsEffectCode 是友誼術（`0Eh`）的參數表 `+0Ah`：處理常式 overlay-12 entry 16
	// `05E0h` 只做 `記錄 +15h = 節點 +3`——收尾時把魅力還原成施法前的值。
	FriendsEffectCode uint8 = 0x0e
	// ShieldEffectCode 是護盾術（`13h`）的參數表 `+0Ah`：overlay-12 entry 19 `065Eh`。
	ShieldEffectCode uint8 = 0x11
	// BlindnessEffectCode 是致盲（`26h`）的參數表 `+0Ah`：overlay-12 entry 31 `0BBEh`。
	BlindnessEffectCode uint8 = 0x21
	// BestowCurseEffectCode 是降咒（`2Ch`）的參數表 `+0Ah`：overlay-12 entry 34 `0C2Dh`。
	BestowCurseEffectCode uint8 = 0x24
	// PoisonEffectCode 是中毒（`37h`）。緩毒術 `1873h` 先問它，沒中毒就整支不做。
	PoisonEffectCode uint8 = 0x37
	// ShieldArmourClassFloor 是 `065Eh` 的 `cmp es:[di+111h], 39h / jae` 之後
	// `mov es:[di+111h], 39h`：AC 內部值墊到 57，也就是 AC 3。
	ShieldArmourClassFloor = 0x39
)

// Magnitude 是節點 `+3` 整個 byte，不拆位元。等級覆寫推進來的值原樣存在這裡
// （友誼術存施法前的魅力、鏡影術存影像數、祈禱術存 `(邊 << 4) + 等級`）。
func (node EffectNode) Magnitude() uint8 { return node.Payload[effectNodeLevelOffset] }

// `07C7h` 的六個特例（`07D2h..086Eh`，其餘走 `+4 + +5 × 26F8h(法術)`）。
const (
	durationSpellCauseDisease = 0x28 // Roll(1, 6) × 10
	durationSpellSpeedy       = 0x39 // Roll(5, 4)
	durationSpellParalyze     = 0x3d // Roll(5, 4)
	durationSpellGiantStr     = 0x3b // Roll(1, 4) × 10 + 40
	durationSpellInvisible    = 0x3f // 戰鬥中 Roll(2, 10) × 10，否則 (Roll(1, 10) + 10) × 10
	durationSpellReading      = 0x43 // 固定 5A0h
	// durationReadingRounds 是 `086Eh` `C7 46 FC A0 05`。
	durationReadingRounds = 0x5a0
)

// SpellEffectDuration 是 overlay-22 `07C7h(法術)`：`08BCh` 在 `0A35h` 逐格呼叫一次，
// 結果當成節點的持續。casterLevel 是 `26F8h` 的施法者等級（`0879h`，**不是**等級覆寫）。
// inCombat 是 `DS:4954h == 5`，只有 `3Fh` 看它。擲骰走 overlay-24 entry 8
// （`9A 48 00 00 01`），順序照原版：每一格各擲一次。
func SpellEffectDuration(id uint8, parameters SpellParameters, casterLevel int, inCombat bool,
	roll func(count, sides int) int) int {
	switch id {
	case durationSpellCauseDisease:
		return roll(1, 6) * 10
	case durationSpellSpeedy, durationSpellParalyze:
		return roll(5, 4)
	case durationSpellGiantStr:
		return roll(1, 4)*10 + 40
	case durationSpellInvisible:
		if inCombat {
			return roll(2, 10) * 10
		}
		return (roll(1, 10) + 10) * 10
	case durationSpellReading:
		return durationReadingRounds
	}
	return parameters.Duration(casterLevel)
}

// HitCheckArmourClass 是群組 11 對「被打的那一個」的 AC 內部值（`+111h`）做的事。
//
// 兩個呼叫端同形：近戰 overlay-13 `1587h..1595h`、碰觸法術 overlay-22 `099Eh..09B2h`，都是
// 先 `010Ah:0043h(目標)`（overlay-25 entry 7 `0BBEh`，整份重算戰鬥數值，spec 063），再派發
// 群組 11（`21h 11h 08h 09h 2Dh 2Eh 1Eh`），然後才擲命中。所以這裡的調整每一次出手都是從
// 重算過的值起算，不會累加。
//
//	21h  entry 31 `0BBEh`：`+111h`／`+112h` 各減 4（`6780h`／`6774h` 的寫入在這個時點
//	     會被 entry 6 的 `0CC9h` 與 entry 7 的 `0D71h` 蓋掉，看不到）
//	11h  entry 19 `065Eh`：`+111h` 小於 39h 就寫成 39h
//
// `08h 09h 2Dh 2Eh` 只寫 `6774h`／`6780h`，同理看不到；`1Eh`（雲）的 AC 那一段由臭雲術的
// 每回合結算處理（spec 121），這裡不重複。
func HitCheckArmourClass(list EffectList, armourClass int) int {
	if list.Has(BlindnessEffectCode) {
		armourClass -= 4
	}
	if list.Has(ShieldEffectCode) && armourClass < ShieldArmourClassFloor {
		armourClass = ShieldArmourClassFloor
	}
	return armourClass
}

// SaveRollAfterEffects 是群組 12（overlay-24 entry 7 `0DB2h`）只帶串列與邊的簡式：類別、體質、
// 行動者陣營與傷害種類都當成不知道。完整的輸入見 SaveRollEffects（save_damage_effects.go）。
//
//	11h  entry 19 `0675h` `FE 06 74 67`：+1
//	21h  entry 31 `80 2E 74 67 04`：−4
//	24h  entry 34 `80 2E 74 67 04`：−4
//	31h  entry 46 `12C1h`：節點 `+3` 位元 4 等於 side → `FE 06 74 67` +1，否則 `FE 0E 74 67` −1
func SaveRollAfterEffects(list EffectList, value int, side uint8,
	areaNode func(code uint8) (EffectNode, bool)) int {
	return int(SaveRollEffects{Effects: list, Category: 0xff, Side: side, AreaNode: areaNode}.
		Apply(uint8(value)))
}

// SpellDamageAfterEffects 是群組 6（overlay-24 entry 19 `1351h`）只帶法術編號的簡式：傷害種類 0、
// 不是範圍、不擲骰（鏡影不作用）。完整的輸入見 SpellDamageEffects。
func SpellDamageAfterEffects(list EffectList, spell uint8, damage int) int {
	return SpellDamageEffects{Effects: list, Spell: spell}.Apply(damage).Damage
}
