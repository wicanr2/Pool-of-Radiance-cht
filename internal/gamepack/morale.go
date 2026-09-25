package gamepack

// 士氣與逃跑的規則（spec 096〈entry 8〉〈逃跑〉，issue #74）。這一檔只放算式，
// 盤面、骰子與訊息在 cmd/pool-game/foe_flee.go。
//
// 來源一律是 DOS overlay（`workplace/ovr/overlay-*.bin`，SHA-256 對
// `docs/audit/dos-ovr-manifest.json`）：
//
//   - overlay-09 entry 8（`10DDh`）：士氣的兩關與失敗後的三條路。
//   - overlay-13 entry 25（`285Dh`）：敵方整體還剩幾成生命（`DS:6D22h`）。
//   - overlay-13 entry 26（`28E7h`）：對面跑得最快的那一個。
//   - overlay-13 entry 7（`0C6Ch`）：踏出盤面時逃不逃得掉。
//   - overlay-09 `08C5h..091Fh`：逃跑時的基準方向。

// MoraleCheckedBit 是記錄 `+84h` 的位元 7：立著才做士氣判定（`113Eh`）。
const MoraleCheckedBit = 0x80

// MoraleCeiling 是士氣值的上限（`115Dh`）：`(+84h & 7Fh) × 2` 超過 66h 就當成 0，
// 也就是第一關一定過不了、直接進第二關。
const MoraleCeiling = 0x66

// MoraleOffset 與 IntelligenceOffset 是 285-byte 記錄裡的兩格。
const (
	MoraleOffset       = 0x84
	IntelligenceOffset = 0x11
)

// SurrenderIntelligence 是投降的門檻（`1259h`：記錄 `+11h` 大於 5 才投降）。
const SurrenderIntelligence = 5

// 效果代碼：士氣那一段問的是群組 11h（`01h 02h 0Bh`，spec 112）。
const (
	MoraleBoostEffectCode = 0x01 // 士氣 +5（overlay-12 entry 5 `010Fh`，byte 會繞回）
	MoraleDropEffectCode  = 0x02 // 士氣 −5，不低於 0（entry 6 `0121h`）
)

// MoraleValue 是 `113Eh..1164h`：checked 為否代表這一隻不做士氣判定。
func MoraleValue(raw uint8) (value uint8, checked bool) {
	if raw&MoraleCheckedBit == 0 {
		return 0, false
	}
	value = (raw &^ MoraleCheckedBit) * 2
	if value > MoraleCeiling {
		value = 0
	}
	return value, true
}

// AdjustMorale 是群組 11h 對 `DS:6783h` 的兩支常式，依群組順序（`01h` 再 `02h`）。
// 每個代碼只算一次（overlay-24 `014Dh` 找到第一個節點就派發）。`0Bh`（魅惑）的
// 套用端在節點已經套過（`+3` 位元 5）時直接返回，所以對士氣沒有作用。
func AdjustMorale(value uint8, boost, drop bool) uint8 {
	if boost {
		value += 5
	}
	if drop {
		if value < 5 {
			value = 0
		} else {
			value -= 5
		}
	}
	return value
}

// SideMorale 是 overlay-13 entry 25（`285Dh`）：敵方（`+10Eh == 1`）在場的目前
// 生命加總 × 20 ÷ 全部的生命上限加總 × 5，存成一個 byte。乘 20 之後只留低 16 位
// （`28D6h` 的 `xor dx, dx`）。上限加總為 0 時原版不寫，回 ok 為否。
func SideMorale(current, maximum uint16) (uint8, bool) {
	if maximum == 0 {
		return 0, false
	}
	return uint8(uint16(current*20) / maximum * 5), true
}

// PartyMoraleCap 是戰鬥佈置時對隊伍 `+58Ch`（ECL `@6DC6`）的上限（overlay-10
// `1F7Fh..1F8Bh`：大於 100 就寫成 100）。
const PartyMoraleCap = 100

// MoraleOutcome 是 entry 8 的結果。
type MoraleOutcome int

const (
	// MoraleHolds：過關，或這一隻不做士氣判定。
	MoraleHolds MoraleOutcome = iota
	// MoraleForcedFlee：被轉變（runtime `+10h`）的，印 `is forced to flee`（`10FFh`）。
	MoraleForcedFlee
	// MoraleFlees：兩關都沒過，而對面沒有比自己快（`1220h`），印 `flees in panic`。
	MoraleFlees
	// MoraleSurrenders：兩關都沒過、對面比較快、智力大於 5（`1259h`）。
	MoraleSurrenders
	// MoraleCornered：兩關都沒過、對面比較快、智力不到。什麼都不做，照常行動。
	MoraleCornered
)

// MoraleCheck 是 entry 8 讀的每一個值。
type MoraleCheck struct {
	Turned       bool  // runtime `+10h`
	Raw          uint8 // 記錄 `+84h`
	Boost, Drop  bool  // 身上有沒有效果 01h／02h
	HitPoints    int   // 記錄 `+11Bh`
	MaxHitPoints int   // 記錄 `+32h`
	SideMorale   uint8 // `DS:6D22h`
	PartyMorale  uint16
	FoeSide      bool  // 記錄 `+10Eh` 非 0
	Intelligence uint8 // 記錄 `+11h`
	OwnSpeed     int   // overlay-13 `0123h` ÷ 2
	FastestFoe   int   // overlay-13 entry 26 `28E7h`
}

// ResolveMorale 是 overlay-09 entry 8（`10DDh..128Eh`）的判斷，不擲骰。
func ResolveMorale(check MoraleCheck) MoraleOutcome {
	if check.Turned {
		return MoraleForcedFlee
	}
	value, checked := MoraleValue(check.Raw)
	if !checked {
		return MoraleHolds
	}
	// 第一關（`1169h..11ADh`）：士氣 >= 掉了幾成血。有號整數運算，往零取整。
	value = AdjustMorale(value, check.Boost, check.Drop)
	loss := 0
	if check.MaxHitPoints > 0 {
		loss = 100 - check.HitPoints*100/check.MaxHitPoints
	}
	if int(value) >= loss && value != 0 {
		return MoraleHolds
	}
	// 第二關（`11B0h..11F3h`）：敵方整體的生命成數對 100 − 隊伍 `+58Ch`，無號比較；
	// 還要自己在敵方那一邊（`+10Eh != 0`）。
	value = AdjustMorale(check.SideMorale, check.Boost, check.Drop)
	threshold := uint16(PartyMoraleCap) - check.PartyMorale
	if uint16(value) >= threshold && value != 0 && check.FoeSide {
		return MoraleHolds
	}
	// 失敗（`11F6h..1220h`）：對面最快的 ÷ 2 不大於自己的 ÷ 2 就逃。
	if check.FastestFoe <= check.OwnSpeed {
		return MoraleFlees
	}
	if check.Intelligence > SurrenderIntelligence {
		return MoraleSurrenders
	}
	return MoraleCornered
}

// FleeBaseDirection 是 `08DCh..091Fh`：朝向（`DS:6A0Dh`，八方向裡的 0／2／4／6）
// 減掉 ((朝向 + 2) mod 4) div 2，隊伍那一側（`+10Eh == 0`）再加 4，取 mod 8。
// partyFacing 是地圖上的 0..3，對到 `DS:6A0Dh` 是乘 2（overlay-10 `13F4h` 用
// `DS:6A0Dh ÷ 2` 查四格的部署朝向表，spec 059）。
func FleeBaseDirection(partyFacing uint8, partySide bool) uint8 {
	facing := int(partyFacing&3) * 2
	base := facing - ((facing+2)%4)/2
	if partySide {
		base += 4
	}
	return uint8(((base % 8) + 8) % 8)
}

// EscapeSucceeds 是 overlay-13 entry 7（`0C6Ch..0CD9h`）：對面一個都搆不到
// （`010Ah:00C0h(記錄, 0FFh)` 回 0）就逃掉；否則對面最快的比自己慢就逃掉，一樣快時
// 擲 `骰(1,2)` 擲出 1 才逃掉，比自己快就逃不掉。只有一樣快的那一支擲骰。
func EscapeSucceeds(opponents, ownSpeed, fastestFoe int, roll func(count, sides int) int) bool {
	if opponents == 0 {
		return true
	}
	if fastestFoe < ownSpeed {
		return true
	}
	if fastestFoe == ownSpeed {
		return roll(1, 2) == 1
	}
	return false
}

// EscapeStrippedEffects 是 overlay-24 `1004h` 在人離開盤面之後逐一摘掉的十五個
// 代碼（`DS:0C28h..0C36h`，START.EXE 檔案位移 `30640 + 0C28h`，`DS:2880h` 讀到
// `33 34 35 1F` 當正對照）。每個代碼摘第一個節點。
var EscapeStrippedEffects = [...]uint8{
	0x07, 0x0B, 0x1E, 0x1F, 0x20, 0x33, 0x34, 0x35, 0x36, 0x3A, 0x3B, 0x5F, 0x62, 0x89, 0x4A,
}

// MoraleClearedEffects 是 overlay-24 entry 14（`103Bh`）每次士氣判定開頭摘掉的兩個
// 代碼（`DS:0C38h`／`0C39h` = `4A 4B`）；逃跑成立時 `122Fh..1252h` 再摘一次。
var MoraleClearedEffects = [...]uint8{0x4A, 0x4B}

// FledState 與 SurrenderedState 是離開盤面之後的記錄 `+10Ch`（overlay-24 entry 11
// `0F00h` 的第二個參數）：逃掉是 3、投降是 4。只有 3 保留生命值（`0F73h`），
// 戰後的經驗值與戰利品跳過的也只有 3（overlay-05 `0079h`）。
const (
	FledState        = 3
	SurrenderedState = UnconsciousState
)
