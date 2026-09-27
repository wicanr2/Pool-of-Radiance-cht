package gamepack

// 怪物接近之前的效果群組 0Eh 其餘三個碼（spec 161，issue #82）。吐息 `58h` 在 breath.go。
//
// 群組 0Eh 只有 overlay-09 entry 5 開場（`0B66h`）一個呼叫端。overlay-24 entry 3（`02E2h`）
// 依 `53h 54h 58h 79h` 的順序對每個碼叫一次 `014Dh`：身上有那個碼就以最早掛上的節點叫它的
// 處理常式一次（`0177h..0185h`、`02C3h..02D9h`）。三支處理常式都在 overlay-12
// （SHA-256 `d1b05743…`），`retf 0Ah`（記錄、節點、模式），目標都是自己 runtime `+0Ah`
// 那一個（`37B8h` 剛挑好的追擊目標）。

const (
	// GazeStoneEffectCode 是 `53h`：BASILISK、MEDUSA 的石化凝視（entry 76 `1CC5h`）。
	GazeStoneEffectCode uint8 = 0x53
	// GazeCharmEffectCode 是 `54h`：VAMPIRE 的魅惑凝視（entry 77 `1E87h`）。
	GazeCharmEffectCode uint8 = 0x54
	// AcidSpitEffectCode 是 `79h`：AHNKHEG 的噴酸（entry 115 `2C32h`）。
	AcidSpitEffectCode uint8 = 0x79
	// GazeReflectableEffectCode 是 `1D47h..1D5Eh` 問的 `7Fh`：凝視者自己身上有它，才看目標
	// 手上有沒有鏡子。BASILISK（MON2／MON5 block 26）與 MEDUSA（MON5 block 49）都帶著。
	GazeReflectableEffectCode uint8 = 0x7f
	// AcidBiteEffectCode 是 `2D38h` 跟著 `79h` 一起摘掉的 `50h`（群組 2），AHNKHEG 帶著。
	AcidBiteEffectCode uint8 = 0x50

	// MirrorNameWord 是物品名稱字詞表的 `76h`（"Mirror"）。`1D9Ah..1DB3h` 比物品
	// `+2Fh`／`+30h`／`+31h` 三格。
	MirrorNameWord = 0x76

	// StonedState 是 `1E57h` 推給 `005Ah` 的狀態 7（神殿的 Stone to Flesh 收的就是它）。
	StonedState uint8 = 7
	// GazeStoneSaveCategory 是 `1E40h` 的 `B0 01`：類別 1（石化／變形）、修正 0。
	GazeStoneSaveCategory SaveCategory = 1

	// GazeCharmSpell 是 `1F7Eh` 寫進 `DS:6779h` 的 0Ah（魅惑人類）。
	GazeCharmSpell = 0x0a
	// GazeCharmLevel 是 `1F9Fh` 的 `05 0C 00`：節點等級 = (自己 `+10Eh` << 7) + 0Ch。
	GazeCharmLevel = 0x0c
	// GazeCharmSaveCategory／GazeCharmSaveModifier 是 `1FB9h..1FBEh` 的 `B0 04`／`B0 FE`。
	GazeCharmSaveCategory SaveCategory = 4
	GazeCharmSaveModifier              = -2

	// AcidSpitPercent 是 `2C58h` 的 `3C 19 / 76 03`：Roll(1, 100) 不大於 25 才噴。
	AcidSpitPercent = 25
	// AcidSpitReach 是 `2C72h` 的 `3C 04 / 72 03`：距離小於 4 才噴。
	AcidSpitReach = 4
	// AcidSpitDiceCount／AcidSpitDiceSides 是 `2CF6h..2CFCh` 的 overlay-24 entry 9(8, 4)。
	AcidSpitDiceCount = 8
	AcidSpitDiceSides = 4
	// AcidSpitSaveCategory 是 `2D0Dh` 的 `B0 03`（吐息），`2D02h` 的 `B0 02` 是處置規則（減半）。
	AcidSpitSaveCategory SaveCategory = 3
	AcidSpitSaveRule                  = SaveRuleHalves
)

// ApproachEffectCodes 是 entry 3 對群組 0Eh 的呼叫順序（spec 112 的群組表）。
var ApproachEffectCodes = [...]uint8{GazeStoneEffectCode, GazeCharmEffectCode, BreathEffectCode, AcidSpitEffectCode}

// MirrorReadied 是 `1D61h..1E35h` 走目標的物品串列（記錄 `+0C8h`，`+2Ah` 是下一個）：
// 裝備中（`+34h` 非 0）而且三個名稱字詞有一個是 `76h`，就反射回去。
func MirrorReadied(items [][]byte) bool {
	for _, raw := range items {
		if len(raw) <= ItemReadiedOffset || raw[ItemReadiedOffset] == 0 {
			continue
		}
		for word := 0; word < 3; word++ {
			if raw[ItemNameWordOffset+word] == MirrorNameWord {
				return true
			}
		}
	}
	return false
}

// AcidSpitAfterUse 是 `2D1Eh..2D41h`：overlay-24 entry 2 摘掉這一次的 `79h` 節點（014Dh 交的是
// 最早的那一個），再以空節點叫 entry 2 摘第一個 `50h`（`0041h..0072h` 沿串列找碼）。兩個節點的
// `+4` 都是 0（MON6SPC block 65 的 `0000FF00`），不跑收尾。
func AcidSpitAfterUse(list EffectList) EffectList {
	for _, code := range []uint8{AcidSpitEffectCode, AcidBiteEffectCode} {
		if index, ok := list.IndexOf(code); ok {
			list = list.RemoveAt(index)
		}
	}
	return list
}
