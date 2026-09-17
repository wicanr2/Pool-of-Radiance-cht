package main

// HP 鎖定（#40，goal `docs/goals/issue-40-hp-locked-route-a.md`）：**診斷用**的測試治具，
// 讓主線探針不因戰力卡住，專門量路線、旗標、換圖與結局鏈。收據不能拿來證明
// 「打得贏」——那是 #22／#5 的事。
//
// 只寫我方：每個 tick（`Update()` 之後）把戰術盤上的隊員補滿 HP、倒地／昏迷／死亡
// 拉回 0、瀕死計數歸零、佔格類別從 `Footprint` 拿回來（同死靈術的復原形狀，
// `tactical.go`）；戰鬥外把 `state.Party` 補滿。**每一次寫回都計數**，死亡救回來
// 另外記位置：一個 tick 內從滿血打到死亡，表示每 tick 寫回補不到那一刻。
// 寫回本身是產品的 `restorePartyHitPoints`（作弊選單的鎖 HP 共用，spec 141）；治具只負責
// 每個 tick 叫它、跨 app 實例計數，不開作弊開關、不動旗標座標金錢經驗值、不改戰鬥結果。

import (
	"fmt"
	"strings"
	"testing"
)

// afterTick 是每次 `Update()` 之後的測試鉤子；nil 就什麼都不做。只給 HP 鎖定用。
var afterTick func(*app)

// runAfterTick 在呼叫 `Update()` 的測試 helper 裡叫。
func runAfterTick(a *app) {
	if afterTick != nil {
		afterTick(a)
	}
}

type hpLock struct {
	counts    cheatRestoreCounts // 寫回次數，形狀與作弊選單的鎖 HP 共用（spec 141）
	gameOvers int                // 鎖定下仍然看到全滅畫面的次數（每 tick 寫回沒擋住）
}

// install 掛上鉤子，回傳解除用的函式。
func (l *hpLock) install() func() {
	previous := afterTick
	afterTick = l.apply
	return func() { afterTick = previous }
}

// apply 呼叫產品的 `restorePartyHitPoints`（作弊選單的鎖 HP 也用它）。治具仍然
// 自己掛在 `afterTick`，不走作弊開關：它要跨 app 實例統計（讀檔之後是另一個
// app），而且不能讓探針的存檔多出 `CheatsUsed`。
func (l *hpLock) apply(a *app) {
	if a.gameOver {
		l.gameOvers++
		return
	}
	restorePartyHitPoints(a, &l.counts)
}

func (l *hpLock) line() string {
	return fmt.Sprintf("HP lock: in combat %d, out of combat %d, revived from death %d [%s], game-over screens %d",
		l.counts.inCombat, l.counts.outOfCombat, l.counts.revived, strings.Join(l.counts.revivedAt, "; "), l.gameOvers)
}

// 最小重現：同一個 seed 的貧民窟衛兵攔截（地形 13，30 獸人＋4 首領），不鎖全滅，
// 鎖了打贏。負對照先跑，證明這一場本來就打不過——否則「鎖了打贏」什麼也證明不了。
func TestHPLockTurnsTheSlumsGuardsWipeIntoAWin(t *testing.T) {
	const seed, guards = 136, 13
	unlocked := slumsWallFightWith(t, seed, guards, nil)
	if !unlocked.a.gameOver {
		t.Skipf("seed %d 沒鎖也打贏了衛兵攔截（%s），這個 seed 當不了負對照", seed, unlocked.tally.line())
	}
	lock := &hpLock{}
	uninstall := lock.install()
	defer uninstall()
	locked := slumsWallFightWith(t, seed, guards, nil)
	t.Log(lock.line())
	if locked.a.gameOver {
		t.Fatalf("鎖了 HP 還是全滅：%s；%s", locked.tally.line(), lock.line())
	}
	if lock.counts.inCombat == 0 {
		t.Fatalf("鎖定一次都沒寫回，這場贏跟鎖定無關：%s", locked.tally.line())
	}
	// 結果看戰後的結果碼，不看 `battleTally` 的推算：收場那一 tick 看不到結果時，它用
	// 「我方倒下次數 ≥ 人數」推 DEFEAT，而鎖定下同一個人會倒下再被拉回來好幾次。
	// `6DC7`：打贏 0、全滅 80h、被打退 81h（spec 137、overlay-05 `04ADh`）。
	if got := locked.a.eventMachine.Memory[0x6DC7]; got != 0 {
		t.Fatalf("鎖了之後結果碼 6DC7=%02X，打贏該是 0：%s", got, locked.tally.line())
	}
	if locked.a.tactical != nil || locked.a.combatActive {
		t.Fatalf("鎖了之後戰鬥沒有收場")
	}
}
