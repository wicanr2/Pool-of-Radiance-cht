package main

// 玩家策略層第一條（#22）：打完看 HP 決定要不要就地紮營，兩條主線探針與戰術的
// 最小重現共用。從 mainline_probe_test.go 搬出來，內容不變。

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// 玩家策略層第一條：打完看 HP。有人掉到一半以下、或昏迷（狀態 4）就停下
// 探索，找地方紮營休息到滿——每二十四小時回一點（spec 114）。
func partyHurt(a *app) bool {
	// 打到一半不算：戰鬥的生命值要等 `finishCombat` 才寫回隊伍，而續戰
	// 提示（`CONTINUE BATTLE`）還開著時按 E 開不了營。
	if a.tactical != nil {
		return false
	}
	for _, member := range a.state.Party {
		if member.Status == 4 || (member.Status == 0 && member.CurrentHP*2 < member.MaxHP) {
			return true
		}
	}
	// 催眠術用完也算：沒有催眠的一場架（13 名哥布林）就是全滅的那一場。
	// 原版玩家每打完一場就回去休息重記，這裡照做。
	return !sleepReady(a)
}

// restUntilHealed 就地紮營到全隊回滿：按鍵照原版紮營畫面（spec 135）：
// E 紮營、R 排時間、Y 選天、I 加一天、R 休息、ESC 收掉。天數是全隊缺最多
// 的那一位（每 24 小時回 1 點，spec 114）。
//
// 不再先走去屋內：打完一場常常只剩兩個人站著，走去找床的路上再撞一場
// 就是全滅（實測第 14 場之後）。原版在街上休息會被打斷（`ecl2/20` 入口 2
// 把打斷參數設成 24／24，屋內 0／0），remake 還沒跑那個入口（#24）；接上
// 之後被打斷的那一場由同一個駕駛打，打完再睡，這裡的迴圈就是為那一天留的。
func (d *mainlineDriver) restUntilHealed() {
	t, application, step := d.t, d.a, d.step
	t.Helper()
	restPilot := &tacticalPilot{}
	settle := func() {
		for guard := 0; guard < 400 && (application.cellEventPending || application.cellWaitingMenu ||
			application.tactical != nil); guard++ {
			if application.tactical != nil {
				step(restPilot.key(application))
				continue
			}
			step(ebiten.KeyEnter)
		}
		if application.gameOver {
			d.fatalf("the party was destroyed while resting: %q", application.eventText)
		}
	}
	for attempt := 0; attempt < 8 && (partyHurt(application) || pendingMemorisation(application)); attempt++ {
		settle()
		// 打過架用掉的法術格先補記，這一次休息順便記完。
		d.memoriseSpells()
		days := 0
		for _, member := range application.state.Party {
			if member.Status == 0 || member.Status == 4 {
				if missing := member.MaxHP - member.CurrentHP; missing > days {
					days = missing
				}
			}
		}
		if days == 0 && pendingMemorisation(application) {
			days = 1
		}
		before := application.gameTime
		step(ebiten.KeyE)
		if !application.campOpen {
			d.fatalf("E did not open the camp at %+v: %q (mode=%d help=%t tactical=%t event=%t menu=%t door=%t shop=%t program=%t text=%q)",
				application.spawn, application.statusLine, application.mode, application.help,
				application.tactical != nil, application.cellEventPending, application.cellWaitingMenu,
				application.door != nil, application.shopActive, application.campFromProgram, application.eventText)
		}
		step(ebiten.KeyR)
		step(ebiten.KeyY)
		for guard := 0; guard < 64 && !application.restDuration.IsZero(); guard++ {
			step(ebiten.KeyD)
		}
		for day := 0; day < days; day++ {
			step(ebiten.KeyI)
		}
		step(ebiten.KeyR)
		for guard := 0; guard < 8 && (application.campOpen || application.campFromProgram); guard++ {
			step(ebiten.KeyEscape)
		}
		settle()
		hp := []string{}
		for _, member := range application.state.Party {
			hp = append(hp, fmt.Sprintf("%s %d/%d st%d", strings.TrimSpace(member.Name), member.CurrentHP, member.MaxHP, member.Status))
		}
		t.Logf("rested %d days at %+v (clock %v → %v): %v", days, application.spawn, before, application.gameTime, hp)
	}
	if partyHurt(application) || pendingMemorisation(application) {
		d.fatalf("still hurt or unmemorised after resting at %+v (status %q)", application.spawn, application.statusLine)
	}
}
