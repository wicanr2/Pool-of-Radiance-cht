package main

// 隊員踏出盤面逃走（spec 150〈隊伍逃走〉，issue #111）：`ecl4/10 A5DBh` 的
// `CLEARMONSTERS → LOAD MONSTER → TREASURE → COMBAT`，全部從 `Update()` 送鍵。

import (
	"slices"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// partyMoverTurn 按 ENTER 讓敵方把回合走完，停在輪到隊員的那一刻。
func partyMoverTurn(t *testing.T, application *app) uint8 {
	t.Helper()
	for guard := 0; guard < 2000; guard++ {
		state := application.tactical
		if state == nil {
			t.Fatal("the fight ended before a party member moved")
		}
		if mover := state.Mover; mover != 0 && state.Friendly[mover] && state.PartySlot[mover] >= 0 &&
			!state.AIDriven[mover] && len(state.Notices) == 0 {
			return mover
		}
		if err := press(application, ebiten.KeyEnter); err != nil {
			t.Fatal(err)
		}
	}
	t.Fatal("no party member got a turn")
	return 0
}

func TestPartyMemberFleesOffTheBoard(t *testing.T) {
	application := scriptApp(t, 4, 10, 0xA5DB)
	if err := application.continueInitialSearch(nil); err != nil {
		t.Fatal(err)
	}
	pending := len(application.pendingTreasure)
	pool := application.state.PooledMoney
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	mover := partyMoverTurn(t, application)
	state := application.tactical
	runner := state.PartySlot[mover]
	runnerName := application.state.Party[runner].Name
	// 其餘的人倒在地上：`04ADh` 在沒有人站著時才算逃走。
	for index := 1; index < len(state.Roster); index++ {
		if state.Friendly[index] && index != int(mover) {
			state.Roster[index].FootprintClass = 0
			state.States[index], state.HitPoints[index] = gamepack.DyingState, 0
		}
		if !state.Friendly[index] {
			state.BaseMovement[index] = 1 // 比逃跑的人慢：overlay-13 `0CB3h` 直接逃掉
		}
	}
	state.BaseMovement[mover] = 12
	state.Roster[mover].X, state.Roster[mover].Y = 0, 0
	// 往西（K）踏出盤面：先答 N，人留在原地；再答 Y。
	for _, key := range []ebiten.Key{ebiten.KeyM, ebiten.KeyK} {
		if err := press(application, key); err != nil {
			t.Fatal(err)
		}
	}
	if !state.FleePrompt {
		t.Fatalf("stepping off the board did not ask Flee: (status %q)", state.Status)
	}
	if err := press(application, ebiten.KeyN); err != nil {
		t.Fatal(err)
	}
	if state.FleePrompt || state.Roster[mover].FootprintClass == 0 {
		t.Fatal("answering N moved the runner off the board")
	}
	for _, key := range []ebiten.Key{ebiten.KeyK, ebiten.KeyY} {
		if err := press(application, key); err != nil {
			t.Fatal(err)
		}
	}
	if state.States[mover] != gamepack.FledState || state.Roster[mover].FootprintClass != 0 {
		t.Fatalf("runner state %d footprint %d, want fled (3) and off the board",
			state.States[mover], state.Roster[mover].FootprintClass)
	}
	for guard := 0; guard < 2000 && application.tactical != nil; guard++ {
		if application.tactical.Prompt {
			t.Fatal("the fight asked to continue with nobody on the board")
		}
		if err := press(application, ebiten.KeyEnter); err != nil {
			t.Fatal(err)
		}
	}
	if application.tactical != nil || application.gameOver {
		t.Fatalf("tactical=%v gameOver=%v; fleeing is not a wipe", application.tactical != nil, application.gameOver)
	}
	report := application.postCombat
	if report == nil || report.title() != msgPostCombatFled || report.shownShare() != 0 {
		t.Fatalf("report %+v, want the fled page", report)
	}
	if len(application.state.Party) != 1 || application.state.Party[0].Name != runnerName ||
		application.state.Party[0].Status != 0 {
		t.Fatalf("party after fleeing: %+v, want only %q back at status 0", application.state.Party, runnerName)
	}
	if got := application.eventMachine.Memory[0x6DC7]; got != partyFledResultCode {
		t.Fatalf("@6DC7 = %02X, want 81h (068Ch)", got)
	}
	texts := drawnPostCombatTexts(application)
	if !slices.Contains(texts, "The party has fled.") || !slices.Contains(texts, "Each character receives 0") {
		t.Fatalf("fled page draws %q", texts)
	}
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	// entry 2 沒跑：怪物身上的錢與物品不進來，TREASURE 寫的那一份照舊在。
	if !application.treasureMenuShown() || len(application.treasureItems) != pending ||
		application.state.PooledMoney != pool {
		t.Fatalf("menu shown=%v items=%d/%d pool %v/%v", application.treasureMenuShown(),
			len(application.treasureItems), pending, application.state.PooledMoney, pool)
	}
}
