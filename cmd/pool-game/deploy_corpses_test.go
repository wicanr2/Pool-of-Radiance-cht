package main

// 部署時就不在場的隊員（spec 061、spec 060〈生成之後的寫入者〉，issue #69）：原版一樣擺上去、
// 用掉一格樣板，體型改 0，登記成屍體（overlay-10 `1D3Ah..1E1Bh`）。盤面是獸人家那一場。

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

func TestDownedPartyMemberIsDeployedAsACorpse(t *testing.T) {
	application, _ := orcHomeGearFixture(t)
	const downed = 1
	application.state.Party = append([]poolsave.Character(nil), application.state.Party...)
	application.state.Party[downed].Status = combat.DyingState
	application.state.Party[downed].CurrentHP = 0
	application.tactical = nil
	if err := application.enterTacticalPreview(); err != nil {
		t.Fatal(err)
	}
	state := application.tactical
	index := npcBoardIndex(t, state, downed)
	cell := state.Roster[index]
	if cell.FootprintClass != 0 || state.Footprint[index] != 1 {
		t.Fatalf("the dying member is on the board with footprint %d (remembered %d); want 0 and 1",
			cell.FootprintClass, state.Footprint[index])
	}
	if state.States[index] != combat.DyingState || state.HitPoints[index] != 0 {
		t.Fatalf("the dying member enters with state %d HP %d; want 5 and 0", state.States[index], state.HitPoints[index])
	}
	if got := state.corpseAt(int(cell.X), int(cell.Y)); int(got) != index {
		t.Fatalf("the corpse table has %d at (%d,%d); want the dying member %d", got, cell.X, cell.Y, index)
	}
	// 用掉一格樣板：沒有別人站在它那一格（`14CFh` 放完就把樣板格清掉，後面的人往後排）。
	for other := 1; other < len(state.Roster); other++ {
		if other != index && state.Roster[other].FootprintClass != 0 &&
			state.Roster[other].X == cell.X && state.Roster[other].Y == cell.Y {
			t.Fatalf("combatant %d was deployed onto the corpse at (%d,%d)", other, cell.X, cell.Y)
		}
	}
	// 陣型第一腿的上限只數在場的（overlay-25 entry 31 `2440h`）：我方四人。
	if counts := state.sideCounts(); counts.Party != 4 {
		t.Fatalf("%d allies stand on the board, want 4", counts.Party)
	}

	// 回合收尾沿同一條串列推進倒地計時（overlay-08 `08C6h`）：從按鍵走完第一回合。
	round := state.Round
	for guard := 0; guard < 5000 && application.tactical == state && state.Round == round; guard++ {
		if err := press(application, ebiten.KeyEnter); err != nil {
			t.Fatal(err)
		}
	}
	if state.Round == round {
		t.Fatalf("the first round never ended (status %q)", state.Status)
	}
	if state.DyingCounters[index] != 1 || state.States[index] != combat.DyingState {
		t.Fatalf("after one round the dying member has counter %d state %d; want 1 and 5",
			state.DyingCounters[index], state.States[index])
	}
}
