package main

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// AI 代打的回合走到一半交還玩家（#123，spec 167）。
//
// 原版在 AI 的回合裡有三個地方問鍵盤，都是 overlay-09 entry 7（`0FC8h`）：
//
//	entry 1 `001Ch`  回合一開頭（擲戰術模式之前）
//	entry 1 `01ACh`  換完武器（`13D5h`）之後、挑目標與 entry 5 之前
//	`07E8h` `0843h`  每一次走一步之前（接近與逃跑兩條都經過這一支）
//
// entry 7 有鍵才讀一個（`0FD2h` `512h:2FAh`），讀完清緩衝區（`10B2h` `26Bh:0F6h`）。SPACE
// （`103Eh`）把串列上 `+84h < 80h` 而且 `+10Ch != 1` 的記錄 `+10Fh` 清 0；清完這一位自己的
// `+10Fh` 是 0，就把 runtime `+3`（先攻分數）寫成 14h（`109Ah`）並回 1。三個呼叫端拿到 1 都
// 直接收工、不叫 entry 34（`0843h` 跳 `0B36h`；entry 5 的 `0BD8h` 看到 `+3 == 14h` 收工；
// entry 1 的 `0106h`／`01B2h` 跳 `01FDh`），所以分數還是 14h。先攻選取（overlay-08 `0124h`）
// 挑最大的分數，14h 是上限，於是**下一個被選的就是同一位**；overlay-08 entry 3 開頭
// （`0231h..0248h`）把 14h 改成 13h，這時 `+10Fh` 已經是 0，走的是玩家指令迴圈（`0307h`）。
// 腳程（runtime `+6`）在這中間沒人動：走過的步數扣掉了，剩下的交給玩家。
//
// remake 的 AI 回合原本在一個影格裡跑完，玩家沒有機會在中途按鍵。會被 SPACE 收回的那一種
// 行動者（隊員、不是 NPC、`+84h < 80h`）在遊戲速度不是 0 時改成在 goroutine 裡跑 foeTurn：
// 每到上面三個點就停一個影格，下一個影格帶著那一影格的按鍵回來。其他行動者（怪物、NPC）SPACE 收不回來，
// entry 7 對它們永遠回 0，照舊在同一個影格跑完。停的長度（一個影格）不是原版的——原版
// 一步之間停多久取決於 overlay-13 entry 5 的重畫（`27Fh:0000h`，沒讀），屬停拍長度。

// handBackInitiative 是 `109Ah` 寫進 runtime `+3` 的 14h；`resumedInitiative` 是 overlay-08
// entry 3 `0248h` 把它改成的 13h。
const (
	handBackInitiative = 0x14
	resumedInitiative  = 0x13
)

// foeRun 是一個跨影格執行中的 AI 回合。goroutine 與主迴圈輪流執行，由兩個不帶緩衝的
// channel 交棒，所以同一時間只有一邊碰遊戲狀態。
type foeRun struct {
	state  *tacticalState
	mover  uint8
	resume chan bool
	yield  chan struct{}
	done   chan error
}

// handableMover 是 SPACE 收得回來的 AI 行動者：entry 7 `1059h..106Fh` 清的是 `+84h < 80h`、
// `+10Ch != 1` 的記錄（releaseQuick 同一個判準），而這一位現在由 AI 走。
func (a *app) handableMover(state *tacticalState, mover uint8) bool {
	if mover == 0 || !state.aiDrives(int(mover)) || int(mover) >= len(state.PartySlot) {
		return false
	}
	slot := state.PartySlot[mover]
	if slot < 0 || slot >= len(a.state.Party) {
		return false
	}
	member := a.state.Party[slot]
	return !member.NPC && memberMoraleRaw(member) <= moraleHighBit
}

// runFoeTurn 取代直接呼叫 foeTurn：收得回來的行動者走 goroutine，其餘照舊。
func (a *app) runFoeTurn(state *tacticalState) error {
	// 速度 0 時原版的 Delay 是 0（`speedDelayTicks`），整個回合在按鍵之間就跑完了，
	// 只有回合開頭那一次問得到鍵——同一個影格跑完是同一件事。
	if !a.handableMover(state, state.Mover) || a.speedDelayTicks() == 0 {
		return a.foeTurn(state)
	}
	run := &foeRun{state: state, mover: state.Mover,
		resume: make(chan bool), yield: make(chan struct{}), done: make(chan error, 1)}
	a.foeRun = run
	go func() { run.done <- a.foeTurn(state) }()
	return a.awaitFoeRun()
}

// awaitFoeRun 等 goroutine 停在下一個問鍵的點，或整個回合跑完。
func (a *app) awaitFoeRun() error {
	run := a.foeRun
	select {
	case <-run.yield:
		return nil
	case err := <-run.done:
		a.foeRun = nil
		return err
	}
}

// resumeFoeRun 是停著的那一個點在這一影格問鍵（entry 7），把答案交回去，再等下一個點。
// typeahead 是停拍時按下、留在緩衝區的 SPACE（tacticalInput）。
func (a *app) resumeFoeRun(state *tacticalState, typeahead bool) error {
	run := a.foeRun
	if run.state != state {
		// 盤面換了（不會發生在正常路徑上）：放掉這一個回合，goroutine 停在原地。
		a.foeRun = nil
		return nil
	}
	run.resume <- a.foeKeyCheck(state, run.mover, typeahead)
	if err := a.awaitFoeRun(); err != nil {
		return err
	}
	if a.foeRun == nil && state.Finished {
		return a.finishCombat(state.Outcome)
	}
	return nil
}

// foeKeyCheck 是 entry 7：`2` 切換 Magic On／Off（`0FF4h`），SPACE 收回全隊（`103Eh`），
// 收完這一位不再由 AI 走就回 true（`1087h..109Fh`）。一次只讀一個鍵（`0FDEh`）。
func (a *app) foeKeyCheck(state *tacticalState, mover uint8, typeahead bool) bool {
	switch {
	case a.justPressed(ebiten.KeyDigit2):
		a.toggleMagic(state)
	case typeahead || a.justPressed(ebiten.KeySpace):
		if a.releaseQuick(state) {
			state.Status = state.say(msgStatusQuickOff)
		}
		return !state.aiDrives(int(mover))
	}
	return false
}

// foeCheckpoint 在 goroutine 裡：停一個影格，回主迴圈問到的答案。同步跑的回合（怪物、NPC）
// 沒有 foeRun，entry 7 對它們一定回 0。
func (a *app) foeCheckpoint(state *tacticalState, mover uint8) bool {
	run := a.foeRun
	if run == nil || run.state != state || run.mover != mover {
		return false
	}
	run.yield <- struct{}{}
	return <-run.resume
}

// handBackFoeTurn 是交還的那一刻：戰術模式寫回（runtime `+15h` 原版擲完就寫，`00B1h`）、
// 走過的步數記帳，分數寫 14h、不叫 entry 34，接著照常重選（overlay-08 `0124h`）。
func (a *app) handBackFoeTurn(state *tacticalState, mover uint8, mode, steps int) {
	state.setTacticMode(mover, mode)
	state.Activity.FoeSteps += steps
	if int(mover) < len(state.Scores) {
		state.Scores[mover] = handBackInitiative
	}
	state.Moving = false
	state.selectActor(a.rollDice)
}

// handBackAtTurnStart 是 entry 1 `001Ch` 那一次：entry 7 回 1 之後照樣擲戰術模式
// （`0045h..00B1h`），士氣與後面的一切都跳過（`00B5h`、`0106h`）。
func (a *app) handBackAtTurnStart(state *tacticalState, mover uint8) {
	a.handBackFoeTurn(state, mover, state.tacticMode(mover, a.rollDice), 0)
}
