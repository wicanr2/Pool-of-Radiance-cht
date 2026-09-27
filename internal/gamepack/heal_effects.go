package gamepack

// 治療常式與解病鏈（spec 155，issue #113）。輸入：overlay-24（`e878166e…`）entry 21 `175Dh`、overlay-22
// （`967065cc…`）`225Bh` 與編號 58 `2E02h`；`objdump -D -b binary -m i8086 -M intel`（`coab-go-test:20260729`）。
// 皆 exact。

// NoHealingEffectCode 是 `32h`：entry 21 在 `1796h..17ABh` 問它，身上有就不治療（旗標為 0 時）；
// 解病鏈 `22C6h` 摘它（連帶 `39h`）。
const NoHealingEffectCode uint8 = 0x32

// noHealingCompanion 是解病鏈 `22E6h` 跟著 `32h` 摘的 `39h`。
const noHealingCompanion uint8 = 0x39

// CureDiseaseChain 是 overlay-22 `225Bh`（`677Dh = 1`，摘節點不重掛）：
//
//	2272h  entry 15(記錄, 22h)                       ; 有就摘、印 "is Cured"，結果 = 1
//	228Ah  entry 15(記錄, 2Bh)：有 → 結果 = 1、entry 2 摘 2Ch、摘 1Fh
//	22CEh  entry 15(記錄, 32h)：有 → 結果 = 1、entry 2 摘 39h
//
// entry 15 與 entry 2 都摘最早掛上的那一個。回傳串列與結果。
func CureDiseaseChain(list EffectList) (EffectList, bool) {
	cured := false
	if list.Has(DiseaseEffectCode) {
		list, cured = list.Remove(DiseaseEffectCode), true
	}
	if list.Has(DiseaseWeakeningEffectCode) {
		list, cured = list.Remove(DiseaseWeakeningEffectCode), true
		list = list.Remove(DiseaseWastingEffectCode).Remove(HelplessEffectCode)
	}
	if list.Has(NoHealingEffectCode) {
		list, cured = list.Remove(NoHealingEffectCode), true
		list = list.Remove(noHealingCompanion)
	}
	return list, cured
}

// healableStates 是 entry 21 `1770h` 的集合常數（`CS:173Dh` 的 32 bytes，首 byte 33h）：{0, 1, 4, 5}。
func healableState(state uint8) bool {
	return state == 0 || state == AnimatedState || state == UnconsciousState || state == DyingState
}

// HealResult 是 entry 21 做完的結果。
type HealResult struct {
	HitPoints int
	State     uint8
	// Healed 是 entry 21 的回傳值（`183Bh` 設 1）：呼叫端（編號 58 的 `2E67h`）據此印 "is Healed"。
	Healed bool
	// StandsUp 為真時 `1825h..1838h` 以模式 1 叫 `4Eh` 的常式（entry 22）——只在戰鬥外。
	StandsUp bool
}

// HealByEntry21 是 overlay-24 entry 21（`175Dh`，`f(記錄, 量, 旗標)`）以旗標 0 被叫的那一條——法術的三個
// 呼叫端（overlay-22 `1074h`、`2E62h`、`2FA7h`）都推 `B0 00 50`：
//
//	1767h  +10Ch 不在 {0, 1, 4, 5} → 回 0
//	1796h  旗標 0 → 身上有 32h 就回 0
//	17B9h  +11Bh += 量（byte），無號大於 +32h 就墊回 +32h
//	17F4h  +10Dh == 0（倒著）：狀態 5 → 4；狀態 4 而不在戰鬥中 → entry 1(4Eh, 記錄, NULL, 1)
//	183Bh  回 1
func HealByEntry21(list EffectList, state uint8, down bool, hitPoints, maximum, amount int,
	inCombat bool) HealResult {
	result := HealResult{HitPoints: hitPoints, State: state}
	if !healableState(state) || list.Has(NoHealingEffectCode) {
		return result
	}
	value := uint8(hitPoints) + uint8(amount)
	if value > uint8(maximum) {
		value = uint8(maximum)
	}
	result.HitPoints, result.Healed = int(value), true
	if down {
		if result.State == DyingState {
			result.State = UnconsciousState
		}
		result.StandsUp = result.State == UnconsciousState && !inCombat
	}
	return result
}
