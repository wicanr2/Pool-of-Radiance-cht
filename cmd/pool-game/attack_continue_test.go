package main

// 殺了目標、還有剩的攻擊次數時回合繼續（spec 160，issue #109）。盤面是獸人家那一場
// （orcHomeGearFixture）；玩家從按鍵進去，電腦從 foeTurn。

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// keepOnly 把 keep 以外的每一格移出盤面（兩邊都算）。
func keepOnly(state *tacticalState, keep ...uint8) {
	for index := 1; index < len(state.Roster); index++ {
		kept := false
		for _, want := range keep {
			kept = kept || uint8(index) == want
		}
		if !kept {
			state.Roster[index].FootprintClass = 0
			state.Scores[index] = 0
		}
	}
}

// freeCellsAround 列出 anchor 周圍 distance（切比雪夫距離）那一圈、地面與 anchor 同一種而且
// 沒人站的格子。
func freeCellsAround(t *testing.T, state *tacticalState, anchor uint8, distance int) [][2]uint8 {
	t.Helper()
	here := state.Roster[anchor]
	floor, err := state.Grid.TerrainAt(int(here.X), int(here.Y))
	if err != nil {
		t.Fatal(err)
	}
	var cells [][2]uint8
	for dy := -distance; dy <= distance; dy++ {
		for dx := -distance; dx <= distance; dx++ {
			if max(abs(dx), abs(dy)) != distance {
				continue
			}
			x, y := int(here.X)+dx, int(here.Y)+dy
			if x < 0 || y < 0 {
				continue
			}
			if terrain, err := state.Grid.TerrainAt(x, y); err != nil || terrain != floor {
				continue
			}
			taken := false
			for index := 1; index < len(state.Roster); index++ {
				cell := state.Roster[index]
				if cell.FootprintClass != 0 && int(cell.X) == x && int(cell.Y) == y {
					taken = true
				}
			}
			if !taken {
				cells = append(cells, [2]uint8{uint8(x), uint8(y)})
			}
		}
	}
	return cells
}

// placeAt 把 who 擺到 anchor 周圍 distance 那一圈、從 anchor 追得到而且距離剛好是 distance
// 的一格（skip 是已經用掉、不要再擺的格子）。
func placeAt(t *testing.T, state *tacticalState, who, anchor uint8, distance int) {
	t.Helper()
	saved := state.Roster[who]
	for _, cell := range freeCellsAround(t, state, anchor, distance) {
		state.Roster[who].X, state.Roster[who].Y = cell[0], cell[1]
		if got, ok := state.tacticalRange(anchor, who); ok && got == distance {
			return
		}
	}
	state.Roster[who] = saved
	t.Fatalf("no cell %d away from %d for %d", distance, anchor, who)
}

// continueFixture：第一個隊員對上 7 號頭目與另一隻獸人，其餘的人全部移出盤面；每一下都命中、
// 傷害擲最大（d20 一律 20）。
func continueFixture(t *testing.T) (*app, *tacticalState, uint8, uint8, uint8) {
	t.Helper()
	application, _ := orcHomeGearFixture(t)
	application.roller = fixedRoller{value: 20}
	state := application.tactical
	hero, other := uint8(0), uint8(0)
	for index := 1; index < len(state.Roster); index++ {
		switch {
		case hero == 0 && state.Friendly[index] && state.PartySlot[index] >= 0:
			hero = uint8(index)
		case other == 0 && !state.Friendly[index] && uint8(index) != missileLeader &&
			state.Roster[index].FootprintClass != 0:
			other = uint8(index)
		}
	}
	if hero == 0 || other == 0 {
		t.Fatal("orc home board lacks a party member or a second orc")
	}
	keepOnly(state, hero, missileLeader, other)
	state.swingsLeft = nil
	return application, state, hero, missileLeader, other
}

// 兩次攻擊的戰士（編碼 4）撞上第一隻、一下就殺掉：回合沒結束，還在移動裡；撞第二隻只打
// 剩下的那一下，打完回合才結束。
func TestPlayerFighterKeepsTheSwingLeftAfterAKill(t *testing.T) {
	application, state, hero, first, second := continueFixture(t)
	state.setSingleAttackForm(int(hero), combat.DamageDice{Count: 1, Sides: 8})
	state.AttackRates[hero] = [gamepack.MonsterAttackSlots]uint8{4, 0}
	state.THAC0[hero] = 1
	application.state.Party[state.PartySlot[hero]].Inventory = nil
	state.HitPoints[first], state.HitPoints[second] = 1, 100
	placeAt(t, state, first, hero, 1)
	placeAt(t, state, second, hero, 1)
	state.Prompt, state.Mover, state.Moving = false, hero, false
	state.Budgets[hero] = state.BaseMovement[hero] * 2

	bump := func(target uint8) {
		t.Helper()
		drainCombatNotices(t, application)
		here, there := state.Roster[hero], state.Roster[target]
		direction, err := combat.RequiredFacing(here.X, here.Y, there.X, there.Y, combat.DirectionAny)
		if err != nil {
			t.Fatal(err)
		}
		if err := press(application, tacticalStepKeypad[direction]); err != nil {
			t.Fatal(err)
		}
	}
	bump(first)
	if state.HitPoints[first] > 0 || state.Roster[first].FootprintClass != 0 {
		t.Fatalf("first orc HP %d, footprint %d; want it down", state.HitPoints[first], state.Roster[first].FootprintClass)
	}
	if state.Mover != hero || !state.Moving || state.Activity.PartyAttacks != 1 {
		t.Fatalf("after the kill: mover %d moving %v attacks %d; want %d still moving after one attack",
			state.Mover, state.Moving, state.Activity.PartyAttacks, hero)
	}
	if got := state.swingsLeft[hero]; got != [2]uint8{1, 0} {
		t.Fatalf("+113h/+114h left %v, want [1 0]", got)
	}
	bump(second)
	if got := state.HitPoints[second]; got != 100-8 {
		t.Fatalf("second orc HP %d, want one 1d8 hit (92)", got)
	}
	if state.Mover == hero || state.Activity.PartyAttacks != 2 {
		t.Fatalf("after the last swing: mover %d, attacks %d; want the turn over after two attacks",
			state.Mover, state.Activity.PartyAttacks)
	}
}

// 短弓一回合兩發：第一發殺了近的那一隻，A、Enter 再瞄，預設停在剩下那一隻，射剩下的一發。
func TestPlayerArcherShootsTheSecondArrowAtTheNextTarget(t *testing.T) {
	application, state, hero, first, second := continueFixture(t)
	slot := state.PartySlot[hero]
	bow, arrows := leaderItem(t, state, 0x2b), leaderItem(t, state, gamepack.ItemTypeArrow)
	bow.Raw[gamepack.ItemReadiedOffset], arrows.Raw[gamepack.ItemReadiedOffset] = 1, 1
	arrows.Raw[gamepack.ItemCountOffset] = 5
	application.state.Party[slot].Inventory = []poolsave.Item{bow, arrows}
	if err := application.applyPartyGearStats(state, int(hero), application.state.Party[slot]); err != nil {
		t.Fatal(err)
	}
	state.THAC0[hero] = 1
	state.HitPoints[first], state.HitPoints[second] = 1, 100
	placeAt(t, state, first, hero, 2)
	placeAt(t, state, second, hero, 3)
	state.Prompt, state.Mover, state.Moving = false, hero, false

	shoot := func() {
		t.Helper()
		drainCombatNotices(t, application)
		for _, key := range []ebiten.Key{ebiten.KeyA, ebiten.KeyEnter} {
			if err := press(application, key); err != nil {
				t.Fatal(err)
			}
		}
	}
	shoot()
	if state.Roster[first].FootprintClass != 0 || state.Mover != hero {
		t.Fatalf("first shot: near orc footprint %d, mover %d; want it down and %d still up",
			state.Roster[first].FootprintClass, state.Mover, hero)
	}
	if count, _ := countOf(application.state.Party[slot].Inventory, gamepack.ItemTypeArrow); count != 4 {
		t.Fatalf("after the kill %d arrows left, want 5 − 1 = 4", count)
	}
	hp := state.HitPoints[second]
	shoot()
	if count, _ := countOf(application.state.Party[slot].Inventory, gamepack.ItemTypeArrow); count != 3 {
		t.Fatalf("after the second shot %d arrows left, want 3", count)
	}
	if state.HitPoints[second] >= hp || state.Mover == hero {
		t.Fatalf("second shot: far orc HP %d → %d, mover %d; want it hit and the turn over",
			hp, state.HitPoints[second], state.Mover)
	}
	if state.Activity.PartyAttacks != 2 {
		t.Fatalf("%d attacks, want 2", state.Activity.PartyAttacks)
	}
}

// 電腦那一側：兩次攻擊的獸人一下殺了貼身的隊員，entry 1 再進 entry 5——重挑目標、往前一步、
// 對另一個隊員打剩下的那一下。
func TestFoeKeepsTheSwingLeftAfterAKill(t *testing.T) {
	application, state, hero, orc, _ := continueFixture(t)
	var ally uint8
	for index := 1; index < len(state.Roster); index++ {
		if state.Friendly[index] && uint8(index) != hero && state.PartySlot[index] >= 0 {
			ally = uint8(index)
			break
		}
	}
	if ally == 0 {
		t.Fatal("orc home board has one party member only")
	}
	state.Roster[ally].FootprintClass = state.Roster[hero].FootprintClass
	keepOnly(state, hero, ally, orc)
	state.FoeItems[int(orc)] = nil
	if err := application.storeCombatItems(state, int(orc), -1, nil); err != nil {
		t.Fatal(err)
	}
	state.AttackRates[orc] = [gamepack.MonsterAttackSlots]uint8{4, 0}
	state.THAC0[orc] = 1
	state.HitPoints[hero], state.HitPoints[ally] = 1, 100
	placeAt(t, state, hero, orc, 1)
	placeAt(t, state, ally, orc, 2)
	state.Prompt, state.Mover = false, orc
	state.FoeTargets[orc] = hero
	state.Budgets[orc] = state.BaseMovement[orc] * 2
	start := state.Roster[orc]
	if err := application.foeTurn(state); err != nil {
		t.Fatal(err)
	}
	if state.HitPoints[hero] > 0 {
		t.Fatalf("adjacent hero HP %d, want down", state.HitPoints[hero])
	}
	// 兩下都命中：第一下殺了隊員，剩下的那一下打在兩格外的另一個隊員身上。
	if state.Activity.FoeAttacks != 2 || state.Activity.FoeHits != 2 || state.HitPoints[ally] >= 100 {
		t.Fatalf("%d attacks, %d hits, ally HP %d (log %q); want the second swing on the ally",
			state.Activity.FoeAttacks, state.Activity.FoeHits, state.HitPoints[ally], state.FoeLog)
	}
	if state.Roster[orc] == start {
		t.Fatal("the orc hit the ally two cells away without stepping")
	}
	if left := state.swingsLeft[orc]; left != [2]uint8{} {
		t.Fatalf("+113h/+114h left %v after the turn, want none", left)
	}
}

// `0E09h`：出過手之後叫 entry 8。新 < 舊寫回新；新 ≥ 舊 × 2 留著 0D49h 寫的 `+0A1h`；射擊也
// 留著 `+0A1h`；其餘近戰寫回新。還沒出過手的不動（下一次出手照現數）。
func TestRecountSwingsFollowsTheConditionalWriteBack(t *testing.T) {
	application, state, hero, _, _ := continueFixture(t)
	slot := state.PartySlot[hero]
	bow, arrows := leaderItem(t, state, 0x2b), leaderItem(t, state, gamepack.ItemTypeArrow)
	bow.Raw[gamepack.ItemReadiedOffset], arrows.Raw[gamepack.ItemReadiedOffset] = 1, 1
	arrows.Raw[gamepack.ItemCountOffset] = 5
	for _, test := range []struct {
		name       string
		bow        bool
		rate, left uint8
		want       uint8
	}{
		{"fewer now", false, 2, 2, 1},                  // 0E18：新 1 < 舊 2
		{"twice as many keeps +0A1h", false, 4, 1, 4},  // 0E34：新 2 ≥ 舊 1 × 2
		{"melee in between writes back", false, 4, 2, 2}, // 0E41 之後：新 2
		{"missile keeps +0A1h", true, 3, 2, 3},         // 0E41：短弓一回合兩發，新 2、舊 2 → 留 3
	} {
		application.state.Party[slot].Inventory = nil
		if test.bow {
			application.state.Party[slot].Inventory = []poolsave.Item{bow, arrows}
		}
		if err := application.applyPartyGearStats(state, int(hero), application.state.Party[slot]); err != nil {
			t.Fatal(err)
		}
		state.setSingleAttackForm(int(hero), combat.DamageDice{Count: 1, Sides: 6})
		state.AttackRates[hero] = [gamepack.MonsterAttackSlots]uint8{test.rate, 0}
		state.swingsLeft = map[uint8][2]uint8{hero: {test.left, 0}}
		if err := application.recountSwings(state, hero); err != nil {
			t.Fatal(err)
		}
		if got := state.swingsLeft[hero][0]; got != test.want {
			t.Errorf("%s: +113h %d, want %d", test.name, got, test.want)
		}
	}
	state.swingsLeft = nil
	if err := application.recountSwings(state, hero); err != nil {
		t.Fatal(err)
	}
	if got, ok := state.swingsLeft[hero]; ok {
		t.Errorf("not attacked yet: counts %v stored", got)
	}
}

// 送鍵走一次 `0E34h`：編碼 4 的戰士（這一相位兩下）瞄準、一下殺了目標，剩 1；按 U 開物品選單再按
// ESC 關掉，選單回來叫 entry 8——新 2 ≥ 舊 1 × 2，`+113h` 留著 `+0A1h`，變成 4。
func TestItemMenuAfterAKillLeavesTheRawRate(t *testing.T) {
	application, state, hero, first, second := continueFixture(t)
	state.setSingleAttackForm(int(hero), combat.DamageDice{Count: 1, Sides: 8})
	state.AttackRates[hero] = [gamepack.MonsterAttackSlots]uint8{4, 0}
	state.THAC0[hero] = 1
	arrows := leaderItem(t, state, gamepack.ItemTypeArrow)
	arrows.Raw[gamepack.ItemReadiedOffset] = 0
	application.state.Party[state.PartySlot[hero]].Inventory = []poolsave.Item{arrows}
	state.HitPoints[first], state.HitPoints[second] = 1, 100
	placeAt(t, state, first, hero, 1)
	placeAt(t, state, second, hero, 3)
	state.Prompt, state.Mover, state.Moving = false, hero, false
	for _, key := range []ebiten.Key{ebiten.KeyA, ebiten.KeyEnter} {
		if err := press(application, key); err != nil {
			t.Fatal(err)
		}
	}
	if state.Roster[first].FootprintClass != 0 || state.Mover != hero || state.swingsLeft[hero] != [2]uint8{1, 0} {
		t.Fatalf("after the kill: footprint %d, mover %d, left %v; want the orc down, %d up, [1 0]",
			state.Roster[first].FootprintClass, state.Mover, state.swingsLeft[hero], hero)
	}
	drainCombatNotices(t, application)
	for _, key := range []ebiten.Key{ebiten.KeyU, ebiten.KeyEscape} {
		if err := press(application, key); err != nil {
			t.Fatal(err)
		}
	}
	if application.combatItems != nil {
		t.Fatal("the item menu is still open after ESC")
	}
	if got := state.swingsLeft[hero]; got != [2]uint8{4, 0} {
		t.Fatalf("after the item menu +113h/+114h %v, want [4 0] (+0A1h kept, 0E34h)", got)
	}
}
