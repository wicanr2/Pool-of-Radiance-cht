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
// **就地睡，不走去找床**：打完一場常常只剩兩個人站著，走去找床的路上再撞一場
// 就是全滅（實測第 14 場之後；#24 接上入口 2 之後再試一次，四場架就死了）。
//
// 入口 2 接上之後（#24）這一條在貧民窟街上會睡不滿：街上是 24／24（每兩小時
// 擲一次、24% 會被城衛隊趕起來），而回一點生命力要連續睡滿二十四小時
// （`RestTicksPerHeal` 288 刻），十二次檢定全過的機率只有四%。屋內才是 0／0。
// **那是原版的規則，不是缺口**——原版玩家清出一間屋子或去旅店睡（城區的
// `4A07 != 0` 才寫 0／0，hypothesis：那是旅店的房間）。駕駛要挑地方睡是
// 玩家策略層的事，歸 #22。
func (d *mainlineDriver) restUntilHealed() {
	t, application, step := d.t, d.a, d.step
	t.Helper()
	// 找床要走路，而走路每一步都會問「受傷了沒」——不擋就無限遞迴。
	if d.resting {
		return
	}
	d.resting = true
	defer func() { d.resting = false }()
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
	// 這一區會不會打擾？入口 2 只寫變數，跑它沒有副作用（#24）。會打擾就
	// **不要睡**：貧民窟街上被打斷不是選單而是一場隨機遭遇（`ecl2/20` 入口 3
	// `9A49h`：`SAVE 200 @4A1F`、`PARTYSTRENGTH`、`GOTO 9B68h` 排一場架），
	// 傷兵在那裡連睡八次就是全滅。原版要睡得進屋、或去旅店（城區 `4A07 != 0`
	// 才寫 0／0，hypothesis）。挑地方睡是玩家策略層的事，見 #22。
	if err := application.runRestEntry(); err == nil && application.restInterruption().Period != 0 {
		d.note("restUntilHealed: %+v interrupts rest (%d／%d); not sleeping here",
			application.spawn, application.restInterruption().Period, application.restInterruption().Threshold)
		return
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
		// 量牆的那幾條（`tolerateDefeat`）要的是「這一場打得贏嗎」，睡不滿是
		// 原版在街上的常態（見檔頭），記一行就往下走；主線探針還是要當錯。
		if d.tolerateDefeat {
			d.note("rested but still hurt at %+v: %s", application.spawn, application.statusLine)
			return
		}
		d.fatalf("still hurt or unmemorised after resting at %+v (status %q)", application.spawn, application.statusLine)
	}
}
