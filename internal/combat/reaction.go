package combat

// 效果串列的走訪常數，取自 overlay-25 `21DCh`（spec 059）。
const (
	// EffectListHeadOffset 是 combatant record 裡效果串列頭的位移。
	EffectListHeadOffset = 0x7F
	// EffectNodeCodeOffset 是節點的效果代碼位移。
	EffectNodeCodeOffset = 0x00
	// EffectNodeNextOffset 是節點的下一個指標位移。
	EffectNodeNextOffset = 0x05
)

// DisablingEffectCodes 逐位元組照抄 DS:2880h..2883h：overlay-25 entry 6 就是
// 逐一問這四個代碼在不在效果串列裡。折疊基底是 287Fh，索引 1..4。
var DisablingEffectCodes = [4]uint8{0x33, 0x34, 0x35, 0x1F}

// HasEffect 重現 overlay-25 `21DCh`：沿效果串列找指定代碼。
// codes 是呼叫端由 combatant record 走訪出來的代碼序列，順序要與原版一致。
func HasEffect(codes []uint8, wanted uint8) bool {
	for _, code := range codes {
		if code == wanted {
			return true
		}
	}
	return false
}

// IsReactionDisabled 重現 overlay-25 entry 6（`0B79h`）：四個代碼任一命中即為真。
// 原版把它當成反應攻擊的否決條件——為真就不打。
func IsReactionDisabled(codes []uint8) bool {
	for _, code := range DisablingEffectCodes {
		if HasEffect(codes, code) {
			return true
		}
	}
	return false
}

// ReactionFacingWindow 是 overlay-13 entry 6 為反應攻擊試的朝向數量。
// 原版由目標目前的朝向起算 +6 到 +10，取模 8 之後就是「目前朝向的前後兩格」。
const ReactionFacingWindow = 5

// ReactionFacings 依原版順序列出要試的朝向：`(base+6)%8` 到 `(base+10)%8`。
func ReactionFacings(base uint8) [ReactionFacingWindow]uint8 {
	var facings [ReactionFacingWindow]uint8
	for index := range facings {
		facings[index] = (base + uint8(6+index)) % DirectionCount
	}
	return facings
}

// ReactionAttackSlot 是 overlay-13 entry 6 選出的攻擊槽與更新後的 phase 計數。
type ReactionAttackSlot struct {
	Slot        int
	PhaseCounts [2]uint8
}

// SelectReactionAttackSlot 重現 overlay-13 entry 6 的槽位選擇：
// 主攻擊 base rate（record `+A1h`）非零時預設用第一槽，否則用第二槽；
// 接著依序看兩個 phase 計數（`+113h`、`+114h`），計數大於 0 的槽會覆蓋預設，
// 後看到的贏；最後若選中的槽計數是 0，就補成 1。
//
// phaseCounts 的索引 0、1 分別對應原版的槽 1、2。
func SelectReactionAttackSlot(primaryBaseRate uint8, phaseCounts [2]uint8) ReactionAttackSlot {
	slot := 2
	if primaryBaseRate != 0 {
		slot = 1
	}
	for candidate := 1; candidate <= 2; candidate++ {
		if phaseCounts[candidate-1] > 0 {
			slot = candidate
		}
	}
	if phaseCounts[slot-1] == 0 {
		phaseCounts[slot-1] = 1
	}
	return ReactionAttackSlot{Slot: slot, PhaseCounts: phaseCounts}
}

// LeavingOpponents 重現 overlay-13 entry 6 的差集：把移動前鄰接、移動後仍鄰接的
// 對手去掉，剩下的就是這一步會脫離的對手。原版以 0 覆蓋而不是壓縮陣列，
// 因此這裡也保留原始順序、只回報留下來的那些。
func LeavingOpponents(before, after []uint8) []uint8 {
	leaving := make([]uint8, 0, len(before))
	for _, candidate := range before {
		still := false
		for _, other := range after {
			if other == candidate {
				still = true
				break
			}
		}
		if !still {
			leaving = append(leaving, candidate)
		}
	}
	return leaving
}
