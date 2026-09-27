package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// sweepFixture 是獸人家那一格、同一支五人隊，對面換成四隻 GOBLIN GUARD（`+73h` 0）。
// 第一名隊員設成戰士 level 級，三隻哥布林圍在他身邊。
func sweepFixture(t *testing.T, level uint8) (*app, uint8, []uint8) {
	t.Helper()
	walk, err := os.ReadFile(filepath.Join("..", "..", "docs", "audit", "dosgolem-deployment-peek-orc-home.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fight struct {
		PartyCell struct{ X, Y, Facing uint8 } `json:"party_cell"`
	}
	if err := json.Unmarshal(walk, &fight); err != nil {
		t.Fatal(err)
	}
	a := newDeploymentFixture(t, dosZIPForTests, fight.PartyCell.X, fight.PartyCell.Y, fight.PartyCell.Facing,
		[][3]uint8{{0, 4, 4}}, true)
	a.state.Party[0].ClassLevels = []uint8{0, 0, level, 0, 0, 0, 0, 0}
	state := a.tactical
	mover := uint8(npcBoardIndex(t, state, 0))
	var goblins []uint8
	for index := 1; index < len(state.Roster); index++ {
		if !state.Friendly[index] {
			if state.HitDice[index] != 0 {
				t.Fatalf("goblin %d has +73h %d", index, state.HitDice[index])
			}
			goblins = append(goblins, uint8(index))
		}
	}
	if len(goblins) != 4 {
		t.Fatalf("%d goblins on the board", len(goblins))
	}
	// 三隻搬到隔壁：八個方向依序試，距離量得到 1 才算數。
	here := state.Roster[mover]
	placed := 0
	for _, step := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, 1}, {-1, -1}, {1, -1}, {-1, 1}} {
		if placed == 3 {
			break
		}
		x, y := int(here.X)+step[0], int(here.Y)+step[1]
		taken := false
		for index := range state.Roster {
			if index != 0 && int(state.Roster[index].X) == x && int(state.Roster[index].Y) == y && state.Roster[index].FootprintClass != 0 {
				taken = true
			}
		}
		if taken || x < 0 || y < 0 {
			continue
		}
		goblin := goblins[placed]
		saved := state.Roster[goblin]
		state.Roster[goblin].X, state.Roster[goblin].Y = uint8(x), uint8(y)
		if distance, ok := state.tacticalRange(mover, goblin); !ok || distance != 1 {
			state.Roster[goblin] = saved
			continue
		}
		placed++
	}
	if placed != 3 {
		t.Fatalf("only %d goblins fit next to the fighter", placed)
	}
	state.Mover = mover
	return a, mover, goblins[:3]
}

func sweepNoticeCount(a *app, state *tacticalState) int {
	count := 0
	want := state.say(msgStatusSweeps)
	for _, notice := range state.Notices {
		if strings.Contains(notice.Text, want) {
			count++
		}
	}
	return count
}

// 戰士 3 級、一回合一下、身邊三隻不到一個生命骰的哥布林：印 "sweeps"，三隻各砍一下
// （overlay-13 entry 10 `0E8Ch`，spec 154）。同一回合再出手就掃不了（`17D0h` 清掉 runtime `+5`）。
func TestFighterSweepsGoblinsBelowOneHitDie(t *testing.T) {
	a, mover, goblins := sweepFixture(t, 3)
	state := a.tactical
	before := state.Activity.PartyAttacks
	if err := a.resolveTacticalAttack(state, goblins[1]); err != nil {
		t.Fatal(err)
	}
	if got := state.Activity.PartyAttacks - before; got != 3 {
		t.Fatalf("fighter 3 struck %d times, want one swing at each of the three goblins", got)
	}
	if sweepNoticeCount(a, state) != 1 {
		t.Fatalf("notices %+v, want one %q", state.Notices, state.say(msgStatusSweeps))
	}
	if limit := a.sweepLimit(state, mover); limit != 0 {
		t.Fatalf("runtime +5 after striking is %d, want 0", limit)
	}
	// 同一回合第二次出手照一般攻擊打。
	before = state.Activity.PartyAttacks
	state.Notices = nil
	if err := a.resolveTacticalAttack(state, goblins[0]); err != nil {
		t.Fatal(err)
	}
	if got := state.Activity.PartyAttacks - before; got != 1 || sweepNoticeCount(a, state) != 0 {
		t.Fatalf("second attack in the same round: %d attacks, %d sweep notices", got, sweepNoticeCount(a, state))
	}
	// 下一回合 runtime `+5` 從 `+6Bh` 抄回來。
	state.Round++
	if limit := a.sweepLimit(state, mover); limit != 3 {
		t.Fatalf("runtime +5 next round %d, want 3", limit)
	}
}

// 負對照：一級戰士的上限 1 不比攻擊次數多，照一般攻擊打；戰士 2 級只砍兩隻。
func TestSweepNeedsMoreLimitThanAttacks(t *testing.T) {
	a, _, goblins := sweepFixture(t, 1)
	state := a.tactical
	before := state.Activity.PartyAttacks
	if err := a.resolveTacticalAttack(state, goblins[0]); err != nil {
		t.Fatal(err)
	}
	if got := state.Activity.PartyAttacks - before; got != 1 || sweepNoticeCount(a, state) != 0 {
		t.Fatalf("fighter 1: %d attacks, %d sweep notices; want an ordinary attack", got, sweepNoticeCount(a, state))
	}
	a, _, goblins = sweepFixture(t, 2)
	state = a.tactical
	before = state.Activity.PartyAttacks
	if err := a.resolveTacticalAttack(state, goblins[2]); err != nil {
		t.Fatal(err)
	}
	if got := state.Activity.PartyAttacks - before; got != 2 || sweepNoticeCount(a, state) != 1 {
		t.Fatalf("fighter 2: %d attacks, %d sweep notices; want two", got, sweepNoticeCount(a, state))
	}
}
