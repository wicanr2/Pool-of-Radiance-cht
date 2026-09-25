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

// 模式 0Ah 的四支（祝福、詛咒、急速、緩速）瞄準時照 `220Fh` 挑一點、以預算 2 收人，
// 收到的表**不分敵我**；分邊是處理常式自己再走一次表（spec 098〈模式 0Ah：分邊〉）。
// 兩支分邊常式都是 overlay-22 內的 near call：
//
//	0F35h(邊, 訊息)  祝福 0FF5h 推施法者 +10Eh、詛咒 1026h 推 23F5h(施法者)
//	  0F4Eh  DS:677Eh = 1
//	  0F59h  for i := 1 to DS:6B88h：
//	  0F77h    表[i] 的 +10Eh 不等於「邊」→ 表[i] 清成 nil
//	  0F81h    等於，而且 DS:6779h == 1（祝福）、DS:4954h == 5（戰鬥中）：
//	  0FA5h      010Ah:00C0h(表[i], 1)（overlay-25 entry 32，旁邊一步內的對面人數）
//	           非 0 → 表[i] 清成 nil
//	  0FE1h  08BCh(法術, 0, 0, 0, 0, 訊息)
//	2724h(效果碼, 邊, 訊息)  急速 2858h 推 2Ah 與施法者 +10Eh、緩速 2BCDh 推 27h 與 23F5h(施法者)
//	  273Dh  DS:677Eh = 1；額度 = 26F8h(法術)（施法者等級）
//	  2775h  表[i] 的 +10Eh 等於「邊」而且額度 > 0：額度減一；
//	  279Fh    0100h:006Bh(表[i], 效果碼) 非 0（身上已經有）→ 表[i] 清成 nil
//	  27BFh  否則 表[i] 清成 nil
//	  27F2h  08BCh(法術, 0, 0, 0, 0, 訊息)；之後逐個留下的 0100h:002Fh(表[i], 12h)

// SpellSideRoutine 是模式 0Ah 那四支走哪一支分邊常式。
type SpellSideRoutine uint8

const (
	// SpellSideNone 是不分邊（其他法術）。
	SpellSideNone SpellSideRoutine = iota
	// SpellSideBless 是 overlay-22 `0F35h`：只留同一邊，祝福另外剔掉貼身有敵人的。
	SpellSideBless
	// SpellSideQuota 是 overlay-22 `2724h`：只留同一邊、額度是施法者等級、
	// 身上已經有那個效果的剔掉（額度照扣）。
	SpellSideQuota
)

// SpellSideFilter 是一支模式 0Ah 法術的分邊方式。
type SpellSideFilter struct {
	Routine SpellSideRoutine
	// CasterSide 為真代表留施法者自己那一邊（推 `+10Eh`），為假代表留對面
	// （推 overlay-25 entry 30 `23F5h(施法者)`，spec 096：`+10Eh` 為 0 回 1，否則回 0）。
	CasterSide bool
	// SkipEngaged 是 `0F81h..0FACh`：只有祝福（`DS:6779h == 1`）會剔掉旁邊一步內
	// 有對面的人。
	SkipEngaged bool
	// EffectCode 是 `2724h` 查「已經有」的效果碼（急速 2Ah、緩速 27h）。
	EffectCode uint8
}

// SpellSideFilterFor 說這支法術的處理常式怎麼分邊；不是那四支回 false。
func SpellSideFilterFor(id uint8) (SpellSideFilter, bool) {
	switch id {
	case SpellIDBless:
		// 0FF5h：`C4 3E F0 5C 26 8A 85 0E 01 50` 推施法者的 +10Eh。
		return SpellSideFilter{Routine: SpellSideBless, CasterSide: true, SkipEngaged: true}, true
	case SpellIDCurse:
		// 1026h：`9A B6 00 0A 01 50` 推 23F5h(施法者)。
		return SpellSideFilter{Routine: SpellSideBless}, true
	case SpellIDHaste:
		// 2858h：`B0 2A 50` 再推施法者的 +10Eh。
		return SpellSideFilter{Routine: SpellSideQuota, CasterSide: true, EffectCode: HasteEffectCode}, true
	case SpellIDSlow:
		// 2BCDh：`B0 27 50` 再推 23F5h(施法者)。
		return SpellSideFilter{Routine: SpellSideQuota, EffectCode: SlowEffectCode}, true
	}
	return SpellSideFilter{}, false
}

// SpellSideQuery 是分邊常式要問盤面的三件事，由呼叫端（戰場）提供。
type SpellSideQuery struct {
	// Side 是那一格的 `+10Eh`；查不到回 false，那一格一律剔掉（失敗即關閉）。
	Side func(index uint8) (uint8, bool)
	// Engaged 是 overlay-25 entry 32 `(那一格, 1)` 回非 0：旁邊一步內有對面的人。
	Engaged func(index uint8) (bool, error)
	// HasEffect 是 `0100h:006Bh(那一格, 碼)`。
	HasEffect func(index uint8, code uint8) bool
}

// FilterSpellSide 照 `0F35h`／`2724h` 把 `20AEh` 收好的表剔成處理常式真正作用的那幾個。
// 順序照原表；被清成 nil 的直接拿掉（`08BCh` 走表時會跳過 nil 那一格）。
// casterSide 是施法者的 `+10Eh`，casterLevel 是 `26F8h` 的等級（`2724h` 的額度）。
func FilterSpellSide(filter SpellSideFilter, list []uint8, casterSide uint8, casterLevel int,
	query SpellSideQuery) ([]uint8, error) {
	side := casterSide
	if !filter.CasterSide {
		// 23F5h：`+10Eh` 為 0 回 1，否則回 0。
		side = 0
		if casterSide == 0 {
			side = 1
		}
	}
	quota := casterLevel
	kept := make([]uint8, 0, len(list))
	for _, index := range list {
		own, ok := query.Side(index)
		if !ok || own != side {
			continue
		}
		switch filter.Routine {
		case SpellSideBless:
			if filter.SkipEngaged && query.Engaged != nil {
				engaged, err := query.Engaged(index)
				if err != nil {
					return nil, err
				}
				if engaged {
					continue
				}
			}
		case SpellSideQuota:
			// 額度是 byte，`277Fh` 的 `cmp [bp-2Ah], 0 / jbe` 用完就剔。
			if quota <= 0 {
				continue
			}
			quota--
			if query.HasEffect != nil && query.HasEffect(index, filter.EffectCode) {
				continue
			}
		}
		kept = append(kept, index)
	}
	return kept, nil
}
