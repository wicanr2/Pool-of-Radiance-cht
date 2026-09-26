package gamepack

// 致病術（法術 40）掛的 `22h` 到期之後那一串（spec 098〈#99：收尾〉，issue #99）。輸入同
// save_damage_effects.go；exact。
//
// `22h`（overlay-12 entry 32 `0BE3h`）自己不做事：以同一個模式、同一個節點依序叫 `2Bh` 與
// `2Ch` 的常式（`0BF9h`、`0C11h` 都是 `9A 25 00 00 01`，overlay-24 entry 1）。兩支不看模式，
// 開頭都是 overlay-12 `0021h(記錄, 自己的碼, 節點 +3, 持續)`：
//
//	0027h  DS:677Dh != 0 → 回 0（不重掛）
//	004Ch  否則用 entry 10 掛一個新節點（碼, 持續, 節點 +3, 有收尾 1），回 1
//
// `677Dh` 只有治療那幾條路（overlay-22 解病術 `2265h`..`22F4h`、overlay-04、緩毒術的 `16h`
// 常式 `07C4h`..`07DDh`）在摘節點的前後立起又清掉，所以自然到期時一定重掛——這是一個會自己
// 續命的計時器，直到被治好。remake 的治療路徑摘節點時不跑收尾，等同於 `677Dh` 立著。
//
//	2Bh  entry 42 `10EDh`：重掛（持續 3Ch）；記錄 `+10h`（力量）大於 3 → 印 "is weakened"、
//	     力量減一；否則身上沒有 `1Fh` 就掛一個（持續 0、`+3` FFh、不收尾）
//	2Ch  entry 43 `1177h`：重掛（持續 0Ah）；`+11Bh`（生命值）大於 1 → `DS:6777h = 0`、
//	     entry 19 打 1 點（規則 0、沒豁免）；否則同上掛 `1Fh`

const (
	// DiseaseEffectCode 是致病術的參數表 `+0Ah`（`22h`）。
	DiseaseEffectCode uint8 = 0x22
	// DiseaseWeakeningEffectCode 是 `2Bh`：每 3Ch 減一點力量。
	DiseaseWeakeningEffectCode uint8 = 0x2b
	// DiseaseWastingEffectCode 是 `2Ch`：每 0Ah 扣一點生命值。
	DiseaseWastingEffectCode uint8 = 0x2c
	// HelplessEffectCode 是 `1Fh`：群組 7（反應攻擊否決，spec 059）的四個碼之一；解病術也拿它
	// （CureDiseaseEffectCodes）。
	HelplessEffectCode uint8 = 0x1f

	diseaseWeakeningDuration = 0x3c // `1104h` `B8 3C 00`
	diseaseWastingDuration   = 0x0a // `118Eh` `B8 0A 00`
	diseaseStrengthFloor     = 3    // `1113h` `26 80 7D 10 03 / 76 29`
	diseaseHitPointFloor     = 1    // `119Dh` `26 80 BD 1B 01 01 / 76 2F`
)

// DiseaseTeardown 是一個節點到期時那一串做完的結果。
type DiseaseTeardown struct {
	// Effects 是掛完新節點之後的串列（接在尾端，entry 10）。
	Effects EffectList
	// Strength 與 HitPoints 是改過之後的值；Weakened 為真時原版印 "is weakened"（`10E1h`）。
	Strength  uint8
	HitPoints int
	Weakened  bool
	// Wasted 為真時這一下扣了 1 點（entry 19）。
	Wasted bool
}

// DiseaseTeardownOf 是 `22h`／`2Bh`／`2Ch` 的收尾（模式 1）。node 是到期的那一個，list 是摘掉它
// 之後的串列。不是這三個碼的原樣回傳。
func DiseaseTeardownOf(node EffectNode, list EffectList, strength uint8, hitPoints int) DiseaseTeardown {
	result := DiseaseTeardown{Effects: list, Strength: strength, HitPoints: hitPoints}
	level := node.Payload[effectNodeLevelOffset]
	switch node.Code {
	case DiseaseEffectCode:
		result.weaken(level)
		result.waste(level)
	case DiseaseWeakeningEffectCode:
		result.weaken(level)
	case DiseaseWastingEffectCode:
		result.waste(level)
	}
	return result
}

func (result *DiseaseTeardown) weaken(level uint8) {
	result.Effects = result.Effects.Append(NewEffectNode(DiseaseWeakeningEffectCode,
		diseaseWeakeningDuration, level, true))
	if result.Strength > diseaseStrengthFloor {
		result.Strength--
		result.Weakened = true
		return
	}
	result.helpless()
}

func (result *DiseaseTeardown) waste(level uint8) {
	result.Effects = result.Effects.Append(NewEffectNode(DiseaseWastingEffectCode,
		diseaseWastingDuration, level, true))
	if result.HitPoints > diseaseHitPointFloor {
		result.HitPoints--
		result.Wasted = true
		return
	}
	result.helpless()
}

// helpless 是 `1143h..116Ch`（`11D4h..11FDh`）：`010Ah:00A7h` 問有沒有 `1Fh`，沒有才
// `0100h:0052h(記錄, 1Fh, 0, FFh, 0)`。
func (result *DiseaseTeardown) helpless() {
	if result.Effects.Has(HelplessEffectCode) {
		return
	}
	result.Effects = result.Effects.Append(NewEffectNode(HelplessEffectCode, 0, EffectUndispellable, false))
}

// IsDiseaseEffect 回答這個碼是不是這一串的。
func IsDiseaseEffect(code uint8) bool {
	return code == DiseaseEffectCode || code == DiseaseWeakeningEffectCode || code == DiseaseWastingEffectCode
}
