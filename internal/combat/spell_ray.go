package combat

import "fmt"

// 法術的射線（spec 098〈模式 8：射線〉，issue #78）。閃電束（overlay-22 `2B75h`）與
// 編號 3Ch（`2F02h`）都是「先 `287Ch` 打瞄準的那一格，再 `2919h` 從那一格往外拉
// 一條射線逐格打」。兩支都讀 overlay-22 的原始位元組（`967065cc…`）：
//
//	287Ch(X, Y, 傷害, 豁免類別, &擋住)                          retf 0Ch
//	  2889h  013Dh:004Dh(X, Y, &佔格者, &地形)  ; overlay-32 entry 9 = CellAt
//	  28A0h  地形 > 0 而且 DS:2758h[地形×4] == FFh → 擋住 = 1
//	  28BFh  佔格者 > 0 → 豁免 0100h:0043h(記錄, 類別, 0)、0100h:007Fh(記錄, 傷害, 2, 豁免)
//
//	2919h(長度, 傷害, 豁免類別, 加價)                           retf 8
//	  2927h  lastOcc = 瞄準那一格的佔格者（CellAt(DS:6CADh, 6CAEh)）
//	  293Eh  sign = 1；C = 施法者 X／Y（013Dh:006Bh／0070h）；加價旗標 = [bp+6]
//	  2968h  C == 瞄準那一格 → 直接返回
//	  297Dh  剩餘 = 長度 × 2（byte）
//	  298Ch  剩餘 == 0 → DS:677Eh = 0，返回
//	  2995h  走訪器 W：起點 T = DS:6CADh／6CAEh，終點 = T + (T − C) × sign × 剩餘
//	  2A0Eh  prev = W 目前
//	  2A2Ah  W 走一步；CellAt(W 目前)；以下任一成立就停：沒走動、有佔格者而且不是
//	         lastOcc、地形 0（盤面外）、DS:2758h[地形×4] > 1、W 成本 >= 剩餘
//	  2A82h  lastOcc = 這一格的佔格者；地形 0 → 剩餘 = 0
//	  2AC3h  287Ch(W 目前, 傷害, 類別, &擋住)
//	  2ACAh  擋住 → DS:6CADh／6CAEh = W 目前；另一支走訪器從這一格走到 C，
//	         加價旗標非 0 而且那一支的成本 <= 8 → W 成本 += 8；sign = −sign、
//	         加價旗標 = 0、lastOcc = 0（反彈：下一段從牆那一格往施法者那邊拉）
//	  2B44h  W 成本 < 剩餘 → 剩餘 −= W 成本，否則剩餘 = 0
//	  2B58h  沒擋住而且剩餘 != 0 → 回 2A0Eh（同一支 W 繼續走）；否則回 298Ch
//
// W 的成本是從這一段起點累積的，每停一次就從剩餘裡整筆扣一次——停得越多，
// 射線越短。這是原版的算法，照抄。

// spellRayGuard 只是讓迴圈一定有終點：剩餘每一輪都會減少，除非走訪器一步都沒動，
// 那在原版裡要起點等於終點才會發生，而 `2968h` 已經擋掉那一種。
const spellRayGuard = 1024

// SpellRayWallClass 是 `28B1h` 的 `cmp byte [di+2758h], 0FFh`：這一類地形把射線擋回來。
const SpellRayWallClass = 0xFF

// spellRayPassClass 是 `2A73h` 的 `cmp byte [di+2758h], 1 / ja`：大於它就停。
const spellRayPassClass = 1

// spellRaySurchargeReach 與 spellRaySurcharge 是 `2B25h`／`2B2Bh`：第一次反彈時
// 牆離施法者的走訪成本不超過 8，這一段的成本再加 8。
const (
	spellRaySurchargeReach = 8
	spellRaySurcharge      = 8
)

// SpellRay 是 `2919h` 要的東西。
type SpellRay struct {
	// CasterX、CasterY 是施法者的格子（`013Dh:006Bh`／`0070h` 讀 `DS:5CF0h`）。
	CasterX, CasterY int
	// TargetX、TargetY 是 `DS:6CADh`／`6CAEh`：瞄準的那一格。
	TargetX, TargetY int
	// Length 是 `[bp+0Ch]`：閃電束 8、編號 3Ch 3。
	Length uint8
	// Surcharge 是 `[bp+6]`：閃電束 1、編號 3Ch 0。
	Surcharge bool
}

// SpellRayCell 是盤面：`CellAt` 的佔格者與地形，以及地形類別表。
// 佔格者要是**當下的**——前一格打死的人不再佔格，由呼叫端每次重建。
type SpellRayCell func(x, y int) (occupant uint8, terrain uint8, err error)

// StrikeSpellRayCell 是 `287Ch`：回報那一格擋不擋射線，有人站著就呼叫 hit。
func StrikeSpellRayCell(classes CellClasses, cellAt SpellRayCell, x, y int,
	hit func(occupant uint8) error) (bool, error) {
	occupant, terrain, err := cellAt(x, y)
	if err != nil {
		return false, err
	}
	blocked := false
	if terrain > 0 {
		class, err := CellClassAt(classes, terrain)
		if err != nil {
			return false, err
		}
		blocked = class.EntryThreshold == SpellRayWallClass
	}
	if occupant > 0 {
		if err := hit(occupant); err != nil {
			return false, err
		}
	}
	return blocked, nil
}

// TraceSpellRay 是 `2919h`：沿射線逐格呼叫 `287Ch`，hit 會被叫一次或多次
// （反彈回來可能再打到同一個人，那是原版的行為）。
func TraceSpellRay(ray SpellRay, classes CellClasses, cellAt SpellRayCell,
	hit func(occupant uint8) error) error {
	cx, cy := ray.CasterX, ray.CasterY
	tx, ty := ray.TargetX, ray.TargetY
	lastOcc, terrain, err := cellAt(tx, ty)
	if err != nil {
		return err
	}
	if cx == tx && cy == ty {
		return nil
	}
	sign := 1
	surcharge := ray.Surcharge
	remaining := uint8(uint16(ray.Length) << 1)
	var occupant uint8
	for guard := 0; remaining != 0; guard++ {
		if guard >= spellRayGuard {
			return fmt.Errorf("Pool spell ray from (%d,%d) did not terminate", tx, ty)
		}
		// `29A6h..2A01h`：終點用 word 算（cbw 之後 imul），截成 16 位元。
		goalX := int(int16(tx + (tx-cx)*sign*int(remaining)))
		goalY := int(int16(ty + (ty-cy)*sign*int(remaining)))
		walker := NewStepWalker(tx, ty, goalX, goalY)
		for {
			if walker.StartX != walker.GoalX || walker.StartY != walker.GoalY {
				for {
					moved := walker.Step()
					occupant, terrain, err = cellAt(walker.X, walker.Y)
					if err != nil {
						return err
					}
					if !moved || (occupant > 0 && occupant != lastOcc) || terrain == 0 {
						break
					}
					class, err := CellClassAt(classes, terrain)
					if err != nil {
						return err
					}
					if class.EntryThreshold > spellRayPassClass || walker.Cost >= remaining {
						break
					}
				}
			}
			lastOcc = occupant
			if terrain == 0 {
				remaining = 0
			}
			blocked, err := StrikeSpellRayCell(classes, cellAt, walker.X, walker.Y, hit)
			if err != nil {
				return err
			}
			if blocked {
				// `2ACCh`：DS:6CADh／6CAEh 是 byte，下一段從牆那一格起。
				tx, ty = int(int8(walker.X)), int(int8(walker.Y))
				back := NewStepWalker(tx, ty, cx, cy)
				for back.Step() {
				}
				if surcharge && back.Cost <= spellRaySurchargeReach {
					walker.Cost += spellRaySurcharge
				}
				sign, surcharge, lastOcc = -sign, false, 0
			}
			if walker.Cost < remaining {
				remaining -= walker.Cost
			} else {
				remaining = 0
			}
			if blocked || remaining == 0 {
				break
			}
			guard++
			if guard >= spellRayGuard {
				return fmt.Errorf("Pool spell ray from (%d,%d) did not terminate", tx, ty)
			}
		}
	}
	return nil
}

// SignedCellAt 是 `287Ch`／`2919h` 呼叫 overlay-32 `04C0h` 的方式：座標以 byte 推進去、
// 以有號 byte 比界（`04C3h` 的 `cmp byte [bp+10h], 0 / jl`，withinTactical 同一把尺），
// 界外兩個輸出都是 0。
func (state TacticalState) SignedCellAt(x, y int) (occupant uint8, terrain uint8, err error) {
	return state.CellAt(uint8(x), uint8(y))
}
