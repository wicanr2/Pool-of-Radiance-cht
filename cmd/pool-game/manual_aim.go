package main

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
)

// 瞄準時的 Manual 格子游標（spec 127）。
//
// 原版的瞄準列是 `Aim: Next Prev Manual Center Exit`（overlay-13 `2B96h`
// 把 `2A58h` 與 `2A6Ah` 接起來），`M` 進 overlay-13 `2DD3h` 的格子游標。
//
// **離開的條件是一個集合，不是字串。** `2E31h` 呼叫的 `05BB:08D4h` 是
// Turbo Pascal 的**集合成員測試**（`element in set`）——它把元素號碼除以 8
// 當 byte 索引、餘數當位元，`test es:[bx+di], al` 之後結果留在旗標裡，
// 不在 AL 裡。`cs:2D80h` 那 32 個位元組因此是位元圖不是 Pascal 字串，
// 解出來是 **{掃描碼 0, Return, 'E', 'T'}**——四個離開鍵。
//
// 先前把 `05BB:08D4h` 讀成 `Pos`、把 `cs:2D80h` 讀成長度 1 的字串 `" "`，
// 兩種極性都會矛盾（游標的按鍵初值就是空白）。**那個矛盾本身就是讀錯的訊號。**

// manualAimExitKeys 是原版那個集合裡的四個鍵。
//
// Return 與 `T` 是「就選這一格」，`E` 與掃描碼 0 是取消——**所以外層
// `36CDh` 那個看起來多出來的 `T` 與 Return 是同一件事**，不是選項列漏了一項。
// 掃描碼 0 在 remake 這邊對應 Esc（原版是「按了一個沒有 ASCII 的鍵」）。
var manualAimConfirmKeys = []ebiten.Key{ebiten.KeyEnter, ebiten.KeyT}

var manualAimCancelKeys = []ebiten.Key{ebiten.KeyE, ebiten.KeyEscape}

// beginManualAim 進 Manual：游標停在目前挑到的那一格（原版 `2DEBh` 用
// `013D:006Bh`／`0070h` 取目前目標的 X／Y），方向初值 8 ＝ 原地。
func (a *app) beginManualAim() bool {
	state := a.tactical
	if state == nil || len(a.castTargets) == 0 || a.castTargetCursor >= len(a.castTargets) {
		return false
	}
	target := a.castTargets[a.castTargetCursor]
	if int(target) >= len(state.Roster) {
		return false
	}
	a.castManual = true
	a.castManualX, a.castManualY = int(state.Roster[target].X), int(state.Roster[target].Y)
	return true
}

// manualAimInput 走 Manual 的迴圈。回傳 true 代表這一格按鍵已經被它吃掉。
func (a *app) manualAimInput() (bool, error) {
	if !a.castManual {
		return false, nil
	}
	for _, key := range manualAimCancelKeys {
		if a.justPressed(key) {
			// 取消：原版 `3272h` 先 `013D:0043h` 還原那一格，再把回傳旗標清 0。
			// remake 沒有那一層畫面狀態，回到 Next／Prev 就好。
			a.castManual = false
			return true, nil
		}
	}
	for _, key := range manualAimConfirmKeys {
		if a.justPressed(key) {
			return true, a.confirmManualAim()
		}
	}
	// 八個方向鍵與戰術移動同一組（spec 053）。原版 `3217h` 起逐個比
	// `H I M Q P O K G`，對到方向 0..7；其餘按鍵設方向 8（原地），
	// 也就是「什麼都不動」。
	for direction, key := range tacticalStepKeys {
		if !a.justPressed(key) {
			continue
		}
		step, err := combat.DirectionStep(uint8(direction))
		if err != nil {
			return true, nil
		}
		x, y := a.castManualX+int(int8(step.X)), a.castManualY+int(int8(step.Y))
		if x < 0 || x > combat.TacticalMaxX || y < 0 || y > combat.TacticalMaxY {
			return true, nil
		}
		a.castManualX, a.castManualY = x, y
		return true, nil
	}
	return true, nil
}

// confirmManualAim 是原版 `2C17h` 那一步：游標停的那一格上要真的有人，
// 才收成目標。沒有人就什麼都不做，游標留著——原版是把回傳旗標清成 0。
func (a *app) confirmManualAim() error {
	state := a.tactical
	if state == nil {
		a.castManual = false
		return nil
	}
	for index := 1; index < len(state.Roster); index++ {
		cell := state.Roster[index]
		if cell.FootprintClass == 0 {
			continue
		}
		if int(cell.X) != a.castManualX || int(cell.Y) != a.castManualY {
			continue
		}
		a.castManual, a.castTargeting = false, false
		if a.castTargetingAttack {
			a.castTargetingAttack = false
			return a.resolveAimedAttack(uint8(index))
		}
		return a.finishCast(a.castPending, uint8(index), true)
	}
	return nil
}
