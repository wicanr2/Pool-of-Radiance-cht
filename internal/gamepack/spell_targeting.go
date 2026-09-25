package gamepack

// 施法怎麼收目標（spec 098〈收目標：overlay-13 `20AEh`〉，issue #73）。
//
// overlay-22 entry 5（`0C14h`）在戰鬥中經 `DS:6A78h` 呼叫 overlay-13 entry 18
// （`20AEh(法術, 旗標, &結果)`），它依參數表 `+6` 的低四位決定怎麼把目標收進
// `DS:6B85h` 那張表（筆數在 `DS:6B88h`，範圍旗標在 `DS:677Eh`）。**玩家與 AI 走
// 同一支**，差別只在 `1E09h` 挑一個目標時是玩家瞄（`352Ch`）還是 AI 擲骰
// （`1ECCh` 起）。
//
//	20BBh  DS:6B88h = 0；DS:677Eh = 0；DS:6CADh／6CAEh = 施法者的 X／Y
//	20F0h  模式 = +6 & 0Fh
//	20FDh  0：表 = [施法者]
//	211Ah  0Fh：1E09h(法術, 旗標, 0) 挑一個；挑到的是人 → 表 = [他]，
//	            是空格 → 以那一格為中心收範圍，預算 (+6 & 0Fh) >> 4（恆為 0）
//	220Fh  8..0Eh：1E09h(法術, 旗標, 1) 挑一點，以它為中心、預算 +6 & 7 收範圍
//	22BEh  其餘：要 (模式 & 3) + 1 個，逐個 1E09h(法術, 旗標, 0)：
//	            挑不到（Exit）→ 要的數量減一；
//	            挑到重複的 → 玩家印 "Already been targeted" 再挑一次，AI 直接減一；
//	            挑到新的 → 收進表，要的數量減一
//	23C1h  DS:6B88h = 收到幾個；0 個就回報失敗
//
// `1E09h` 的第三個引數交給 `352Ch` 當 `[bp+0Eh]`，一路傳到 Manual 游標
// （`2DD3h` 的 `30DEh`）：**為 0 時空格子不能選，為 1 時可以**——所以範圍法術
// 瞄的是一個點，單體法術瞄的是一個人。

// SpellTargetKind 是 `20AEh` 那四條路。
type SpellTargetKind uint8

const (
	// SpellTargetKindSelf 是模式 0：表就是施法者自己，不挑（`20FDh`）。
	SpellTargetKindSelf SpellTargetKind = iota
	// SpellTargetKindOne 是模式 0Fh：挑一個（`211Ah`）。
	SpellTargetKindOne
	// SpellTargetKindArea 是模式 8..0Eh：挑一個點再收範圍（`220Fh`）。
	SpellTargetKindArea
	// SpellTargetKindCount 是其餘模式：逐個挑 (模式 & 3) + 1 個（`22BEh`）。
	SpellTargetKindCount
)

// SpellTargetPlan 是一條法術照 `20AEh` 要怎麼收目標。
type SpellTargetPlan struct {
	Kind SpellTargetKind
	// Count 是 SpellTargetKindCount 要挑幾個：`22CDh` 的 `and al, 3; inc ax`。
	Count int
	// AreaBudget 是 SpellTargetKindArea 以中心點收人的預算：`2241h` 的
	// `and al, 7`，交給 `0912h` 當上限。與 AI 判斷擠不擠用的 `+0Fh` 是兩個欄位。
	AreaBudget int
	// PointAim 為真代表瞄準時可以選空格子（`1E09h` 的第三個引數是 1）。
	PointAim bool
}

// spellTargetAreaMask 是 `2241h` 的 `and al, 7`。
const spellTargetAreaMask = 0x07

// TargetPlan 照 overlay-13 `20AEh` 把參數表 `+6` 換成收目標的方式。
func (p SpellParameters) TargetPlan() SpellTargetPlan {
	raw := p.Raw[spellParameterTargeting]
	mode := raw & SpellTargetModeMask
	switch {
	case mode == 0:
		return SpellTargetPlan{Kind: SpellTargetKindSelf}
	case mode == 0x0F:
		// `217Dh..2184h` 把 `+6 & 0Fh` 再右移四位當預算，結果恆為 0：
		// 挑到空格時只收那一格上的人（也就是沒有人）。
		return SpellTargetPlan{Kind: SpellTargetKindOne}
	case mode >= 0x08 && mode <= 0x0E:
		return SpellTargetPlan{Kind: SpellTargetKindArea,
			AreaBudget: int(raw & spellTargetAreaMask), PointAim: true}
	default:
		return SpellTargetPlan{Kind: SpellTargetKindCount, Count: int(mode&3) + 1}
	}
}

// FireballOutdoorAreaBudget 說火球術（`262Eh`）的範圍要不要改收：`2661h`
// `cmp word [4933h]+1CCh, 0 / jne 26E0h`——ECL `@49E6` 為 0 時以雲心重收一次、
// 預算 FireballAreaBudget（2）；非 0 時留著 `20AEh` 收好的那張表（預算 `+6 & 7`，
// 火球術是 3）。回傳 0 代表不重收。
func FireballOutdoorAreaBudget(id uint8, walkFlag uint8) int {
	if id != SpellIDFireball && id != SpellIDFireballAlt {
		return 0
	}
	if walkFlag != 0 {
		return 0
	}
	return FireballAreaBudget
}
