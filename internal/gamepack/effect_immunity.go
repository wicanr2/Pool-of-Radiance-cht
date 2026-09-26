package gamepack

// 掛效果之前的免疫檢查：overlay-24 entry 20（`1656h`）的群組 9（spec 112〈群組 9：免疫〉，
// issue #86）。
//
//	166Fh  DS:6775h = 要掛的碼
//	1675h  02E2h(9, 目標)                 ; 群組 9
//	1682h  6775h == 0 → 印 "is Unaffected"、不掛
//
// 群組 9 的處理常式全部經 overlay-12 `0000h` 動 `6775h`：
//
//	0000h  f(碼)：碼 == 0 或 6775h == 碼 → 6776h = 0、6775h = 0      ; retf 2
//
// 所以「對某個碼免疫」就是 `0000h(那個碼)`，「全部免疫」是 `0000h(0)`。來源是
// overlay-12（SHA-256 `d1b05743…`），位元組逐條讀過。

// immunityGroup 是群組 9，照 `02E2h` 的呼叫順序。
var immunityGroup = [...]uint8{0x69, 0x6a, 0x6b, 0x6c, 0x6d, 0x6e, 0x6f, 0x70, 0x7c, 0x7d}

// 被點名的四個碼。
const (
	poisonedEffectCode = 0x37 // "Poisoned"（overlay-15 的名稱鏈）
)

// Immunity damage flags 是 `DS:6777h` 裡這一組讀的三個位元。`08BCh` 在 `[bp+0Ch]`（傷害）
// 為 0 時把 `6777h` 寫 0（`08DBh`），所以祝福、詛咒、急速、緩速、魅惑掛效果時三個都是 0。
const (
	ImmunityFireFlag  = 0x01 // `70h` 的 `2A1Dh`：`and al, 1`
	ImmunityColdFlag  = 0x02 // `6Eh` 的 `29E1h`：`and al, 2`
	ImmunityMagicFlag = 0x08 // 魔法抗性 `2910h` 的 `2949h`：`and al, 8`
)

// SpellEffectImmunity 是一次 entry 20 的群組 9：回 true 代表 `6775h` 被清成 0，印
// "is Unaffected"、不掛。casterLevel 是 `010Ah:00D4h(DS:6779h)`（overlay-25 `26F8h`，
// 這一支法術的施法者等級）；roll 是 overlay-24 entry 8（`0DE5h`，`0100h:0048h`）。
// 每個代碼只問一次、看最早掛上的那一個節點（`014Dh`）；處理常式不讀節點。
func SpellEffectImmunity(target EffectList, code uint8, damageFlags uint8, casterLevel int,
	roll func(count, sides int) int) bool {
	pending := code
	clear := func(which uint8) {
		// `0000h`：`0003h` 參數為 0 直接清；否則 `6775h` 要等於它。
		if which == 0 || pending == which {
			pending = 0
		}
	}
	for _, carried := range immunityGroup {
		if !target.Has(carried) {
			continue
		}
		switch carried {
		case 0x69:
			// entry 99 `296Eh`：`B0 32 50 / E8 98 FF` → 2910h(50)。
			magicResistance(50, pending, damageFlags, casterLevel, roll, clear)
		case 0x6a:
			// entry 100 `297Eh`：2910h(0Fh)。
			magicResistance(15, pending, damageFlags, casterLevel, roll, clear)
		case 0x6b:
			// entry 101 `298Eh`：0100h:0048h(1, 64h) 不大於 5Ah → 0000h(35h)、0000h(0Bh)。
			if roll(1, 100) <= 90 {
				clear(SleepEffectCode)
				clear(CharmPersonEffectCode)
			}
		case 0x6c:
			// entry 102 `29B4h`：0000h(0Bh)、0000h(35h)。
			clear(CharmPersonEffectCode)
			clear(SleepEffectCode)
		case 0x6d:
			// entry 103 `29CBh`：0000h(34h)。
			clear(HoldPersonEffectCode)
		case 0x6e:
			// entry 104 `29DBh`：`6777h` and 2 → 0000h(0)。
			if damageFlags&ImmunityColdFlag != 0 {
				clear(0)
			}
		case 0x6f:
			// entry 105 `29F4h`：0000h(37h)、0000h(34h)；`6788h` 為 0 時 `6774h` = 64h
			// （豁免骰，entry 20 在這之前已經拿到豁免結果，不影響這一次）。
			clear(poisonedEffectCode)
			clear(HoldPersonEffectCode)
		case 0x70:
			// entry 106 `2A17h`：`6777h` and 1 → 0000h(0)。
			if damageFlags&ImmunityFireFlag != 0 {
				clear(0)
			}
		case 0x7c:
			// entry 118 `2DF9h`：0100h:0048h(1, 64h) 不大於 1Eh → 0000h(0Bh)、0000h(35h)。
			if roll(1, 100) <= 30 {
				clear(CharmPersonEffectCode)
				clear(SleepEffectCode)
			}
		case 0x7d:
			// entry 119 `2E1Fh`：0Bh、35h、34h、37h 四個；`6774h` 同 6Fh。
			clear(CharmPersonEffectCode)
			clear(SleepEffectCode)
			clear(HoldPersonEffectCode)
			clear(poisonedEffectCode)
		}
	}
	return pending == 0
}

// magicResistance 是 overlay-12 `2910h(百分比)`：
//
//	2916h  x = 010Ah:00D4h(DS:6779h)                 ; 施法者等級
//	2929h  門檻 = byte(百分比 − (11 − x) × 5)
//	293Fh  6775h != 0 或 6777h and 8 → 擲 1d100，不大於門檻（無號）→ 0000h(0)
//
// 門檻是 byte、比較是無號（`ja`），所以施法者等級低到把門檻減成負數時，門檻繞成
// 很大的數、一定抗得掉——例如 15% 那一支對 7 級以下施法者一定成立。照搬。
func magicResistance(percent int, pending, damageFlags uint8, casterLevel int,
	roll func(count, sides int) int, clear func(uint8)) {
	threshold := uint8(percent - (11-casterLevel)*5)
	if pending == 0 && damageFlags&ImmunityMagicFlag == 0 {
		return
	}
	if uint8(roll(1, 100)) <= threshold {
		clear(0)
	}
}
