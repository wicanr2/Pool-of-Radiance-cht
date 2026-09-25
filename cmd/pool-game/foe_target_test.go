package main

import (
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// sequenceRoller 依序回 values，用完之後一律回最大面。
type sequenceRoller struct {
	values []int
	asked  []int
}

func (roller *sequenceRoller) Roll(_, sides int) int {
	roller.asked = append(roller.asked, sides)
	if len(roller.values) == 0 {
		return sides
	}
	value := roller.values[0]
	roller.values = roller.values[1:]
	if value > sides {
		return sides
	}
	return value
}

// foeTargetBoard：敵人（3）在 (11,10)；隊員 1、2 在 (10,10)、(12,10)，兩邊都貼著。
func foeTargetBoard(t *testing.T, foeX uint8) *tacticalState {
	t.Helper()
	state := newRoundState(3)
	cellCount := combat.TacticalRowStride * (combat.TacticalMaxY + 1)
	const openTerrain = 5
	terrain := make([]uint8, cellCount)
	for index := range terrain {
		terrain[index] = openTerrain
	}
	state.Grid = combat.TacticalGrid{Terrain: terrain}
	state.Classes[openTerrain] = gamepack.CombatCellClass{EntryThreshold: 1}
	state.Friendly[1], state.Friendly[2] = true, true
	state.Roster[1] = combat.CombatantCell{X: 10, Y: 10, FootprintClass: 1}
	state.Roster[2] = combat.CombatantCell{X: 12, Y: 10, FootprintClass: 1}
	state.Roster[3] = combat.CombatantCell{X: foeX, Y: 10, FootprintClass: 1}
	state.HitPoints = []int{0, 10, 10, 10}
	state.THAC0 = []uint8{0, 40, 40, 40}
	state.ArmorClass = []int{0, 50, 50, 50}
	for index := 1; index <= 3; index++ {
		state.setSingleAttackForm(index, combat.DamageDice{Count: 1, Sides: 8})
		state.Scores[index] = 5
		state.Budgets[index] = 24
	}
	state.FoeTargets = make([]uint8, 4)
	state.Mover = 3
	return state
}

// sidesRoller 在擲 sides 面時回 value，其餘一律回最大面，並記下每一擲的面數。
type sidesRoller struct {
	sides, value int
	asked        []int
}

func (roller *sidesRoller) Roll(_, sides int) int {
	roller.asked = append(roller.asked, sides)
	if sides == roller.sides {
		return roller.value
	}
	return sides
}

// 搆得到兩個人：打誰由 `骰(1, n)` 決定（overlay-09 `0D97h`，#65），不是固定打第一個。
// 原版骰流收據裡 d20 前面緊接一擲 d1 的有 23 次——名單只有一個人時也擲。
func TestFoeAttacksTheRolledOneOfTheReachable(t *testing.T) {
	hit := map[int]bool{}
	for pick := 1; pick <= 2; pick++ {
		state := foeTargetBoard(t, 11)
		// 先給牠一個追擊目標，讓 `37B8h` 那一擲不發生，這裡只量 `0D97h`。
		state.setFoeTarget(3, 1)
		roller := &sidesRoller{sides: 2, value: pick}
		a := &app{roller: roller, tactical: state}
		if err := a.foeTurn(state); err != nil {
			t.Fatal(err)
		}
		d20 := -1
		for index, sides := range roller.asked {
			if sides == 20 {
				d20 = index
				break
			}
		}
		if d20 < 1 || roller.asked[d20-1] != 2 {
			t.Fatalf("pick %d: rolls %v, want a d2 over the two reachable right before the d20", pick, roller.asked)
		}
		for index := 1; index <= 2; index++ {
			if state.HitPoints[index] < 10 {
				hit[index] = true
			}
		}
	}
	if !hit[1] || !hit[2] {
		t.Fatalf("both picks hit the same member: %v", hit)
	}
}

// 重挑時過不了 `1087h` 的劃掉重擲（`37B8h`）：1 號身上有 47h，擲到 1 就劃掉再擲。
func TestFoeRepickStrikesAVetoedCandidate(t *testing.T) {
	state := foeTargetBoard(t, 20)
	state.addEffect(1, vetoEffectAlways, 10, 1)
	targets, err := state.foeTargetCandidates(3, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 2 {
		t.Fatalf("fixture: %d candidates, want 2", len(targets))
	}
	vetoedSlot := 1
	if targets[1] == 1 {
		vetoedSlot = 2
	}
	roller := &sequenceRoller{values: []int{vetoedSlot, vetoedSlot, 3 - vetoedSlot}}
	a := &app{roller: roller, tactical: state}
	if err := a.foeTurn(state); err != nil {
		t.Fatal(err)
	}
	if got, ok := state.foeTarget(3); !ok || got != 2 {
		t.Fatalf("the foe chases %d (ok=%t), want member 2 after striking the vetoed one", got, ok)
	}
}

// 追擊目標被否決了就不沿用（`37B8h` 的 `3809h`）。
func TestFoeDropsAStickyTargetThatIsVetoed(t *testing.T) {
	state := foeTargetBoard(t, 20)
	state.setFoeTarget(3, 1)
	if got, ok := state.foeTarget(3); !ok || got != 1 {
		t.Fatalf("fixture: sticky target %d ok=%t", got, ok)
	}
	state.addEffect(1, vetoEffectAlways, 10, 1)
	if got, ok := state.foeTarget(3); ok {
		t.Fatalf("a vetoed sticky target %d was kept", got)
	}
}

// `19h`：被打的一方身上有，而當下行動者沒有 18h 就否決（overlay-12 `0927h`，#65）。
func TestVetoEffect19hNeedsTheActorToCarry18h(t *testing.T) {
	state := foeTargetBoard(t, 20)
	state.addEffect(1, vetoEffectProtected, 10, 1)
	if !state.attackVetoed(1, 3) {
		t.Fatal("19h on the target did not veto an actor without 18h")
	}
	state.addEffect(3, vetoEffectProtectedKey, 10, 1)
	if state.attackVetoed(1, 3) {
		t.Fatal("19h still vetoed after the actor gained 18h")
	}
}

// `7Eh`：出手的一方身上有，看目標裝備中的物品（overlay-12 `2E50h`，#65）。
func TestVetoEffect7EhReadsTheTargetsReadiedItems(t *testing.T) {
	item := func(ready bool, at int, value, byte31 byte) []byte {
		raw := make([]byte, 0x40)
		if ready {
			raw[itemReadyOffset] = 1
		}
		raw[at] = value
		if at != 0x31 {
			raw[0x31] = byte31
		}
		return raw
	}
	cases := []struct {
		name string
		raw  []byte
		veto bool
	}{
		{"76h at +2Fh", item(true, 0x2F, 0x76, 0), true},
		{"76h at +31h", item(true, 0x31, 0x76, 0), true},
		{"76h but not readied", item(false, 0x2F, 0x76, 0), false},
		{"98h with +31h FCh", item(true, 0x30, 0x98, 0xFC), true},
		{"98h without +31h FCh", item(true, 0x30, 0x98, 0x00), false},
		{"76h at +2Eh is outside i = 1..3", item(true, 0x2E, 0x76, 0), false},
	}
	for _, c := range cases {
		state := foeTargetBoard(t, 20)
		state.addEffect(3, vetoEffectTargetsItems, 10, 1)
		raw := c.raw
		state.ItemsOf = func(index int) [][]byte {
			if index == 1 {
				return [][]byte{raw}
			}
			return nil
		}
		if got := state.attackVetoed(1, 3); got != c.veto {
			t.Errorf("%s: vetoed=%t, want %t", c.name, got, c.veto)
		}
	}
}
