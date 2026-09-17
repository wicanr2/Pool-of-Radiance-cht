package main

// 作弊選單（spec 141）：鎖 HP、一擊斃命、穿牆。原版沒有，預設關；打開過一次就在存檔記
// `CheatsUsed`，畫面上一直標著。開過作弊的路線不算原版驗收，也不算 #5 的收據。
//
// app 上的欄位：`helpPage` 是 F1 說明頁的第幾頁（0 起算），`cheatOpen` 是選單開著，
// `cheatRestores` 是鎖 HP 寫回的次數。

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/font"
)

// cheatRestoreCounts 是鎖 HP 寫回的次數：戰術盤上（以隊員計）、戰鬥外（以隊員計）、
// 從死亡拉回來。測試與探針讀它，看鎖定到底補了幾次。
type cheatRestoreCounts struct {
	inCombat    int
	outOfCombat int
	revived     int
	revivedAt   []string
}

// cheatMenuAvailable 是 F6 開得了選單的時候：冒險模式（含戰鬥），而且沒有別的
// 覆蓋層蓋在上面。
func (a *app) cheatMenuAvailable() bool {
	return a.mode == modeAdventure && a.introDone && !a.help && !a.panelOpen() && !a.campOpen
}

// cheatInput 處理作弊選單開著時的按鍵；選單開著時吃掉所有按鍵。
func (a *app) cheatInput() (bool, error) {
	if !a.cheatOpen {
		if a.justPressed(ebiten.KeyF6) && a.cheatMenuAvailable() {
			a.cheatOpen = true
			return true, nil
		}
		return false, nil
	}
	switch {
	case a.justPressed(ebiten.KeyEscape), a.justPressed(ebiten.KeyF6):
		a.cheatOpen = false
	case a.justPressed(ebiten.KeyL):
		a.setCheat(&a.state.Cheats.LockHP, msgCheatLockHPName)
	case a.justPressed(ebiten.KeyO):
		a.setCheat(&a.state.Cheats.OneHitKill, msgCheatOneHitKillName)
	case a.justPressed(ebiten.KeyW):
		a.setCheat(&a.state.Cheats.WalkThroughWalls, msgCheatWalkThroughWallsName)
	}
	return true, nil
}

// setCheat 切換一個開關；打開時寫 `CheatsUsed`，那個標記不會再清掉。
func (a *app) setCheat(flag *bool, name messageID) {
	*flag = !*flag
	if *flag {
		a.state.CheatsUsed = true
	}
	state := msgHouseRuleOff
	if *flag {
		state = msgHouseRuleOn
	}
	a.statusLine = fmt.Sprintf(a.text(msgCheatToggled), a.text(name), a.text(state))
}

// applyCheatLockHP 在每一次 `Update()` 結束時補 HP（spec 141〈鎖 HP〉）。
func (a *app) applyCheatLockHP() {
	if !a.state.Cheats.LockHP || a.gameOver {
		return
	}
	restorePartyHitPoints(a, &a.cheatRestores)
}

// restorePartyHitPoints 把隊員補回上限、倒地與死亡拉回正常，並計數。作弊選單的
// 作弊選單的鎖 HP 用這一支（spec 141）。
func restorePartyHitPoints(a *app, counts *cheatRestoreCounts) {
	if state := a.tactical; state != nil {
		for index := 1; index < len(state.Roster) && index < len(state.PartySlot); index++ {
			slot := state.PartySlot[index]
			if slot < 0 || slot >= len(a.state.Party) || index >= len(state.HitPoints) || index >= len(state.States) {
				continue
			}
			max := a.state.Party[slot].MaxHP
			if index < len(state.MaxHitPoints) && state.MaxHitPoints[index] > 0 {
				max = state.MaxHitPoints[index]
			}
			down := state.States[index] == 4 || state.States[index] == 5 || state.States[index] == 6
			if state.HitPoints[index] >= max && !down {
				continue
			}
			if state.States[index] == 6 {
				counts.revived++
				counts.revivedAt = append(counts.revivedAt, fmt.Sprintf("%+v round %d member %d", a.spawn, state.Round, slot))
			}
			state.HitPoints[index] = max
			if down {
				state.States[index] = 0
				if index < len(state.DyingCounters) {
					state.DyingCounters[index] = 0
				}
				if index < len(state.Footprint) && state.Roster[index].FootprintClass == 0 {
					state.Roster[index].FootprintClass = state.Footprint[index]
				}
			}
			counts.inCombat++
		}
		return
	}
	for index := range a.state.Party {
		member := &a.state.Party[index]
		down := member.Status == 4 || member.Status == 5 || member.Status == 6
		if member.CurrentHP >= member.MaxHP && !down {
			continue
		}
		if member.Status == 6 {
			counts.revived++
			counts.revivedAt = append(counts.revivedAt, fmt.Sprintf("%+v out of combat member %d", a.spawn, index))
		}
		member.CurrentHP = member.MaxHP
		if down {
			member.Status = 0
		}
		counts.outOfCombat++
	}
}

// cheatDamage 是一擊斃命（spec 141〈一擊斃命〉）：隊員造成的傷害大於 0 時，改成目標
// 剩下的 HP。擲骰照常，只改結果。
func (a *app) cheatDamage(state *tacticalState, attacker, target uint8, damage int) int {
	if !a.state.Cheats.OneHitKill || damage <= 0 || state == nil {
		return damage
	}
	if int(attacker) >= len(state.PartySlot) || state.PartySlot[attacker] < 0 {
		return damage
	}
	if int(target) >= len(state.HitPoints) || state.HitPoints[target] <= damage {
		return damage
	}
	return state.HitPoints[target]
}

// cheatMark 是畫面上的作弊標示：開著的開關，或都關著但開過（spec 141〈標示〉）。
func (a *app) cheatMark() string {
	var on []string
	if a.state.Cheats.LockHP {
		on = append(on, a.text(msgCheatLockHPName))
	}
	if a.state.Cheats.OneHitKill {
		on = append(on, a.text(msgCheatOneHitKillName))
	}
	if a.state.Cheats.WalkThroughWalls {
		on = append(on, a.text(msgCheatWalkThroughWallsName))
	}
	if len(on) != 0 {
		return fmt.Sprintf(a.text(msgCheatMark), strings.Join(on, a.text(msgCheatMarkSeparator)))
	}
	if a.state.CheatsUsed {
		return a.text(msgCheatUsedMark)
	}
	return ""
}

// drawCheatMenu 畫作弊選單。
func drawCheatMenu(screen *ebiten.Image, a *app, background, foreground, accent color.Color) {
	for y := 120; y < 304; y++ {
		for x := 96; x < 544; x++ {
			screen.Set(x, y, background)
		}
	}
	title := a.text(msgCheatMenuTitle)
	drawText(screen, title, (logicalWidth-font.MeasureString(uiFace, displayText(title)).Ceil())/2, 148, accent)
	state := func(on bool) string {
		if on {
			return a.text(msgHouseRuleOn)
		}
		return a.text(msgHouseRuleOff)
	}
	drawText(screen, fmt.Sprintf(a.text(msgCheatMenuLockHP), state(a.state.Cheats.LockHP)), 128, 184, foreground)
	drawText(screen, fmt.Sprintf(a.text(msgCheatMenuOneHitKill), state(a.state.Cheats.OneHitKill)), 128, 208, foreground)
	drawText(screen, fmt.Sprintf(a.text(msgCheatMenuWalkThroughWalls), state(a.state.Cheats.WalkThroughWalls)), 128, 232, foreground)
	drawText(screen, a.text(msgCheatMenuNote), 128, 264, accent)
	drawText(screen, a.text(msgCheatMenuClose), 128, 288, foreground)
}
