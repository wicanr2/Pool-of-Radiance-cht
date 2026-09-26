package main

// 射擊與彈藥（spec 151，issue #98）落到戰場上：玩家從按鍵瞄準、電腦從 foeTurn，
// 兩邊都走原版的武器槽規則。盤面是獸人家那一場（orcHomeGearFixture），物品是原版
// MON2 block 14 那三隻獸人頭目身上的東西。

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// missileLeader 是收據 7 號：MON2 block 14 的獸人頭目，23h 與短弓 2Bh 都沒穿、箭（7 支）穿著。
const missileLeader uint8 = 7

// clearFoesExcept 把 keep 以外的敵方移出盤面。
func clearFoesExcept(state *tacticalState, keep uint8) {
	for index := 1; index < len(state.Roster); index++ {
		if !state.Friendly[index] && uint8(index) != keep {
			state.Roster[index].FootprintClass = 0
		}
	}
}

// placeForShot 把 move 擺到 around 周圍的一格：從 observer 看出去貼身沒有敵人，而射程 reach
// 之內有（直線追蹤走得到）。地面要跟 move 原本那一格同一種。
func placeForShot(t *testing.T, state *tacticalState, move, around, observer uint8, reach int) {
	t.Helper()
	floor, err := state.Grid.TerrainAt(int(state.Roster[move].X), int(state.Roster[move].Y))
	if err != nil {
		t.Fatal(err)
	}
	anchor := state.Roster[around]
	for _, offset := range [][2]int{{0, -3}, {0, 3}, {3, 0}, {-3, 0}, {3, 3}, {-3, -3}, {3, -3}, {-3, 3},
		{0, -2}, {0, 2}, {2, 0}, {-2, 0}, {2, 2}, {-2, -2}, {2, -2}, {-2, 2}} {
		x, y := int(anchor.X)+offset[0], int(anchor.Y)+offset[1]
		if x < 0 || y < 0 {
			continue
		}
		if terrain, err := state.Grid.TerrainAt(x, y); err != nil || terrain != floor {
			continue
		}
		occupied := false
		for index := 1; index < len(state.Roster); index++ {
			cell := state.Roster[index]
			if uint8(index) != move && cell.FootprintClass != 0 && int(cell.X) == x && int(cell.Y) == y {
				occupied = true
			}
		}
		if occupied {
			continue
		}
		saved := state.Roster[move]
		state.Roster[move].X, state.Roster[move].Y = uint8(x), uint8(y)
		snapshot, err := state.tacticalSnapshot()
		if err != nil {
			t.Fatal(err)
		}
		side, _ := state.sideOf(observer)
		here := state.Roster[observer]
		adjacent, err := combat.OpposingNearbyAt(snapshot, observer, here.X, here.Y, 1, 1-side, state.sideOf)
		if err != nil {
			t.Fatal(err)
		}
		inRange, err := combat.OpposingNearbyAt(snapshot, observer, here.X, here.Y, uint16(reach), 1-side, state.sideOf)
		if err != nil {
			t.Fatal(err)
		}
		if len(adjacent) == 0 && len(inRange) != 0 {
			return
		}
		state.Roster[move] = saved
	}
	t.Fatalf("no cell around %d gives %d a clear shot", around, observer)
}

// leaderItem 取 7 號頭目身上型別是 itemType 的那一件（複製一份）。
func leaderItem(t *testing.T, state *tacticalState, itemType uint8) poolsave.Item {
	t.Helper()
	for _, item := range state.FoeItems[int(missileLeader)] {
		if item.Raw[gamepack.ItemTypeOffset] == itemType {
			return poolsave.Item{Name: item.Name, Raw: append([]byte(nil), item.Raw...)}
		}
	}
	t.Fatalf("orc leader carries no item of type %#02x", itemType)
	return poolsave.Item{}
}

func countOf(items []poolsave.Item, itemType uint8) (int, bool) {
	for _, item := range items {
		if item.Raw[gamepack.ItemTypeOffset] == itemType {
			return int(item.Raw[gamepack.ItemCountOffset]), true
		}
	}
	return 0, false
}

// 玩家那一側，全部從按鍵進去：A 瞄準、Enter 射。短弓一回合兩發（型別表 `+05h` = 4），
// 每射一發少一支箭；只剩一支時只射一發，用完那一件從物品鏈拿掉；沒有箭就沒有 Target，
// 按 Enter 不打、回合也不算用掉。拿著弓撞上去是 "Not with that weapon"。
func TestPlayerArrowsAreSpentAndRunOut(t *testing.T) {
	application, _ := orcHomeGearFixture(t)
	application.roller = fixedRoller{value: 1} // d20 一律擲 1：全部落空，目標不會先倒下
	state := application.tactical
	archer := uint8(0)
	for index := 1; index < len(state.Roster); index++ {
		if state.Friendly[index] && state.PartySlot[index] >= 0 {
			archer = uint8(index)
			break
		}
	}
	slot := state.PartySlot[archer]
	bow, arrows := leaderItem(t, state, 0x2b), leaderItem(t, state, gamepack.ItemTypeArrow)
	bow.Raw[gamepack.ItemReadiedOffset], arrows.Raw[gamepack.ItemReadiedOffset] = 1, 1
	arrows.Raw[gamepack.ItemCountOffset] = 3
	application.state.Party[slot].Inventory = []poolsave.Item{bow, arrows}
	if err := application.applyPartyGearStats(state, int(archer), application.state.Party[slot]); err != nil {
		t.Fatal(err)
	}
	clearFoesExcept(state, missileLeader)
	placeForShot(t, state, missileLeader, archer, archer, state.attackRangeOf(archer))
	foeHP := state.HitPoints[missileLeader]

	shoot := func() {
		t.Helper()
		state.Prompt, state.Mover = false, archer
		for _, key := range []ebiten.Key{ebiten.KeyA, ebiten.KeyEnter} {
			if err := press(application, key); err != nil {
				t.Fatal(err)
			}
		}
	}
	shoot()
	if count, ok := countOf(application.state.Party[slot].Inventory, gamepack.ItemTypeArrow); !ok || count != 1 {
		t.Fatalf("after one volley the archer has %d arrows (present %v), want 3 − 2 = 1", count, ok)
	}
	shoot()
	if _, ok := countOf(application.state.Party[slot].Inventory, gamepack.ItemTypeArrow); ok {
		t.Fatal("the last arrow was shot but the arrow record is still carried")
	}
	if len(application.state.Party[slot].Inventory) != 1 {
		t.Fatalf("inventory has %d items, want only the bow", len(application.state.Party[slot].Inventory))
	}
	attacks := state.Activity.PartyAttacks
	shoot()
	if state.Activity.PartyAttacks != attacks || state.Mover != archer {
		t.Fatalf("no arrows: %d attacks, mover %d; want no shot and the turn kept", state.Activity.PartyAttacks-attacks, state.Mover)
	}
	if want := state.say(msgAimNoTarget, missileLeader); state.Status != want {
		t.Fatalf("status %q, want %q", state.Status, want)
	}
	if state.HitPoints[missileLeader] != foeHP {
		t.Fatalf("foe HP %d → %d with every d20 a 1", foeHP, state.HitPoints[missileLeader])
	}

	// 撞上去：把頭目擺到弓手旁邊，往它那個方向走一步。
	here := state.Roster[archer]
	moved := false
	for direction := uint8(0); direction < combat.DirectionCount && !moved; direction++ {
		x, y, err := combat.AdvanceTacticalCoordinate(here.X, here.Y, direction)
		if err != nil {
			continue
		}
		free := true
		for index := 1; index < len(state.Roster); index++ {
			cell := state.Roster[index]
			if uint8(index) != missileLeader && cell.FootprintClass != 0 && cell.X == x && cell.Y == y {
				free = false
			}
		}
		if terrain, err := state.Grid.TerrainAt(int(x), int(y)); err != nil || !free || terrain == 0 {
			continue
		}
		state.Roster[missileLeader].X, state.Roster[missileLeader].Y = x, y
		state.Prompt, state.Mover, state.Moving = false, archer, false
		state.Budgets[archer] = state.BaseMovement[archer] * 2
		if err := press(application, tacticalStepKeypad[direction]); err != nil {
			t.Fatal(err)
		}
		moved = true
	}
	if !moved {
		t.Fatal("no free cell next to the archer")
	}
	if want := state.say(msgStatusNotWithWeapon); state.Status != want || state.Activity.PartyAttacks != attacks {
		t.Fatalf("bumping with a bow: status %q, %d attacks; want %q and no attack",
			state.Status, state.Activity.PartyAttacks-attacks, want)
	}
}

// 電腦那一側：7 號頭目身邊沒人，entry 9 把短弓穿上、原地射人；只剩一支箭時只射一發，
// 用完那一件就拿掉。下一個回合沒有箭了，entry 9 改穿 23h、卸下弓，往前走。
func TestOrcLeaderSwitchesToMeleeWhenArrowsRunOut(t *testing.T) {
	application, _ := orcHomeGearFixture(t)
	application.roller = fixedRoller{value: 1}
	state := application.tactical
	mover := missileLeader
	var victim uint8
	for index := 1; index < len(state.Roster); index++ {
		if state.Friendly[index] && state.Roster[index].FootprintClass != 0 {
			victim = uint8(index)
			break
		}
	}
	items := state.FoeItems[int(mover)]
	for _, item := range items {
		if item.Raw[gamepack.ItemTypeOffset] == gamepack.ItemTypeArrow {
			item.Raw[gamepack.ItemCountOffset] = 1
		}
	}
	clearFoesExcept(state, mover)
	bowRange := application.weaponAttackRange(leaderItem(t, state, 0x2b))
	placeForShot(t, state, mover, victim, mover, bowRange)

	turn := func() (attacks int, moved bool) {
		t.Helper()
		state.Prompt, state.Mover = false, mover
		state.FoeTargets[mover] = victim
		state.Budgets[mover] = state.BaseMovement[mover] * 2
		before, activity := state.Roster[mover], state.Activity
		if err := application.foeTurn(state); err != nil {
			t.Fatal(err)
		}
		return state.Activity.FoeAttacks - activity.FoeAttacks,
			state.Roster[mover].X != before.X || state.Roster[mover].Y != before.Y
	}
	readied := func(itemType uint8) bool {
		for _, item := range state.FoeItems[int(mover)] {
			if item.Raw[gamepack.ItemTypeOffset] == itemType {
				return item.Raw[gamepack.ItemReadiedOffset] != 0
			}
		}
		return false
	}
	if attacks, moved := turn(); attacks != 1 || moved {
		t.Fatalf("first turn: %d attacks, moved %v (log %q); want one volley from where it stood", attacks, moved, state.FoeLog)
	}
	if !readied(0x2b) {
		t.Fatal("entry 9 did not ready the short bow")
	}
	if _, ok := countOf(state.FoeItems[int(mover)], gamepack.ItemTypeArrow); ok {
		t.Fatal("the only arrow was shot but the leader still carries it")
	}
	if _, moved := turn(); !moved {
		t.Fatalf("out of arrows: the leader stayed put (log %q); want it to close in", state.FoeLog)
	}
	if readied(0x2b) || !readied(0x23) || state.attackRangeOf(mover) != 1 {
		t.Fatalf("out of arrows: bow readied %v, 23h readied %v, reach %d; want the 23h in hand",
			readied(0x2b), readied(0x23), state.attackRangeOf(mover))
	}
}

// `29h` 防護普通飛彈：頭目的箭（`+32h` 0）從兩格外射來，每一發命中都被擋掉——不扣血、
// 印 "Avoids it"。拿掉那個效果的同一場（負對照）照樣受傷。
func TestProtectionFromNormalMissilesStopsTheLeadersArrows(t *testing.T) {
	for _, protected := range []bool{true, false} {
		application, _ := orcHomeGearFixture(t)
		application.roller = fixedRoller{value: 20} // d20 一律 20：每一發都命中
		state := application.tactical
		mover := missileLeader
		var victim uint8
		hp := map[int]int{}
		for index := 1; index < len(state.Roster); index++ {
			if !state.Friendly[index] {
				continue
			}
			if victim == 0 && state.Roster[index].FootprintClass != 0 {
				victim = uint8(index)
			}
			hp[index] = state.HitPoints[index]
			if protected {
				state.addEffect(index, gamepack.ProtectionFromNormalMissilesEffectCode, 10, 5)
			}
		}
		clearFoesExcept(state, mover)
		placeForShot(t, state, mover, victim, mover, application.weaponAttackRange(leaderItem(t, state, 0x2b)))
		state.Prompt, state.Mover = false, mover
		state.FoeTargets[mover] = victim
		state.Budgets[mover] = state.BaseMovement[mover] * 2
		activity := state.Activity
		if err := application.foeTurn(state); err != nil {
			t.Fatal(err)
		}
		if state.Activity.FoeAttacks-activity.FoeAttacks != 1 {
			t.Fatalf("protected %v: %d attacks (log %q)", protected, state.Activity.FoeAttacks-activity.FoeAttacks, state.FoeLog)
		}
		wounded := false
		for index, before := range hp {
			wounded = wounded || state.HitPoints[index] < before
		}
		avoided := strings.Contains(state.FoeLog, state.say(msgStatusAvoidsMissile, 0)[1:])
		if protected && (wounded || !avoided) {
			t.Errorf("protected: wounded %v, log %q; want every arrow avoided", wounded, state.FoeLog)
		}
		if !protected && !wounded {
			t.Errorf("unprotected: nobody was hurt (log %q)", state.FoeLog)
		}
	}
}
