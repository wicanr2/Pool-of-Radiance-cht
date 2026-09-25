package gamepack

// AI 施法（spec 096〈entry 4：AI 挑法術〉，issue #64）。
//
// overlay-09 entry 4（`053Eh`）是怪物與交給電腦（QUICK）的隊員「要不要放法術、
// 放哪一條」的那一支。它讀的是參數表（`DS:3194h`，spec 074）的三個欄位，以前
// spec 074 列為 DRAFT 的 `+0Bh`..`+0Dh` 就在這裡定了語意：
//
//	+0Bh（`319Fh`）overlay-13 `2450h`：為 0 印 "Camp Only Spell"，不施
//	+0Ch（`31A0h`）overlay-13 `24AAh`：除以 3 是施法時間，扣在先攻分數上
//	+0Dh（`31A1h`）overlay-09 `02FFh`：AI 的優先度，門檻從 7 往下降
//
// `+0Eh`（`31A2h`，spec 074 已讀）在 `02EAh` 與 overlay-13 `1E67h` 兩處都是
// 「為 0 就是打自己，不必找敵人」。
const (
	spellParameterCombatUse   = 0x0B
	spellParameterCastingTime = 0x0C
	spellParameterAIPriority  = 0x0D
	spellParameterTarget      = 0x0E

	// spellCastingTimeDivisor 是 overlay-13 `24B1h` 的 `mov cx, 3; idiv cx`。
	spellCastingTimeDivisor = 3
)

// CampOnly 說這條法術只能在營地施（`+0Bh` 為 0）。AI 挑到它時 overlay-13
// entry 19 印 "Camp Only Spell" 就返回，不施、也不算行動。
func (p SpellParameters) CampOnly() bool { return p.Raw[spellParameterCombatUse] == 0 }

// CastingCost 是施法時間（`+0Ch ÷ 3`）。0 就當場放出去；非 0 要先「開始施法」，
// 從先攻分數扣掉這麼多，輪到下一次才放（overlay-13 entry 19 `24AAh..2552h`）。
func (p SpellParameters) CastingCost() uint8 {
	return p.Raw[spellParameterCastingTime] / spellCastingTimeDivisor
}

// AIPriority 是 AI 挑法術的優先度（`+0Dh`）。overlay-09 `02F4h`：優先度小於這一輪的
// 門檻就不挑。六十七格裡 0 到 7 都有，0 的那幾格（偵測類、營地法術）AI 永遠不放。
func (p SpellParameters) AIPriority() uint8 { return p.Raw[spellParameterAIPriority] }

// TargetsCaster 說這條法術沒得挑時打施法者自己（`+0Eh` 為 0，spec 074）。
func (p SpellParameters) TargetsCaster() bool { return p.Raw[spellParameterTarget] == 0 }

const (
	// AISpellArrayOffset 是 AI 讀的法術陣列起點。overlay-09 `0557h..0590h` 從
	// 記錄 `+17h` 起讀 21 格（索引 0..14h），非零的依序收進清單；施完由 overlay-25
	// entry 16（`14ECh`）在同一個範圍裡清掉第一個等於那個編號的格子。怪物記錄的
	// 法術就從 `+18h` 起放（LEVEL 3 MU 是 `00 0F 15 22`），所以這 21 格是整個陣列，
	// 隊員的 `+1Fh` 那 13 格（spec 070）是它的後段。
	AISpellArrayOffset = 0x17
	// AISpellArraySlots 是那 21 格。
	AISpellArraySlots = 0x15
	// AISpellCureLightWounds 是治療輕傷。`02EAh` 與 overlay-13 `1E7Eh` 都為它
	// 另走 `1BF0h`：在身邊九格找受傷的自己人。
	AISpellCureLightWounds = 3

	// aiSpellTriesDie 是 `059Ah` 的 Roll(1, 7)：門檻最多往下降幾輪。
	aiSpellTriesDie = 7
	// aiSpellStartThreshold 是 `0596h` 的初值 7。
	aiSpellStartThreshold = 7
	// aiSpellPicksPerPass 是 `05EEh` 的內圈：每一輪擲三次（1..3，`cmp 4; jae`）。
	aiSpellPicksPerPass = 3
)

// AIDisablingEffects 是 `DS:2880h..2883h` 那四個效果碼（索引 1..4，折疊常數
// `287Fh`）：33h（迷蛇）、34h（定身）、35h（催眠）、1Fh。overlay-25 entry 6
// （`0B79h`）問目標身上有沒有其中之一，overlay-13 `1FC5h..2003h` 再看這條法術
// 掛的效果碼是不是其中之一——兩者都成立就不再對同一個目標放。
var AIDisablingEffects = [4]uint8{0x33, 0x34, 0x35, 0x1F}

// IsAIDisablingEffect 說這個效果碼在 AIDisablingEffects 裡。
func IsAIDisablingEffect(code uint8) bool {
	for _, value := range AIDisablingEffects {
		if code == value {
			return true
		}
	}
	return false
}

// ChooseAISpell 重現 overlay-09 entry 4 挑法術的那一段（`0592h..0640h`）：
//
//	0596  門檻 = 7
//	059A  次數 = Roll(1, 7)                 ; 不論有沒有法術都先擲
//	05AC  清單是空的 → 不施
//	05B5  記錄 +84h <= 7Fh 而且 DS:6D23h == 0 → 不施   ; allowed 是這兩個條件
//	05C7  對面沒人站著（DS:6772h[對立陣營]）→ 不施   ; 也折在 allowed 裡
//	05DC  第 k 輪（k = 1..次數）：
//	05EE    擲三次：號碼 = Roll(1, 清單長度)，accept(清單[號碼−1], 門檻) 成立就收下
//	063A    門檻 −1
//
// accept 是 `02EAh`（見 spec 096）。清單照陣列的順序排，空格不收。
func ChooseAISpell(list []uint8, allowed bool, roll func(count, sides int) int,
	accept func(id, threshold uint8) bool) uint8 {
	tries := roll(1, aiSpellTriesDie)
	if len(list) == 0 || !allowed {
		return 0
	}
	threshold := uint8(aiSpellStartThreshold)
	for pass := 1; pass <= tries; pass++ {
		for pick := 0; pick < aiSpellPicksPerPass; pick++ {
			id := list[roll(1, len(list))-1]
			if accept(id, threshold) {
				return id
			}
		}
		threshold--
	}
	return 0
}

// AISpellList 把法術陣列裡非零的格子依序收成清單（overlay-09 `0557h..0590h`
// 的 `cmp byte ptr es:[di+17h], 0; jbe`）。比的是整個 byte，所以第 7 位立著
// （還沒記完，spec 070）的格子也收進來——它會佔清單的一個號碼。
func AISpellList(array []uint8) []uint8 {
	list := make([]uint8, 0, len(array))
	for _, value := range array {
		if value != 0 {
			list = append(list, value)
		}
	}
	return list
}
