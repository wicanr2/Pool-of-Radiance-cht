package main

// 怪物開打時的 entry 7 重算（#93，spec 147）落到戰場上：獸人家那一場 dosgolem 讀到的
// 二十隻怪物，THAC0、AC、腳程、第一種形態的傷害骰逐隻對上；拿短弓的那一隻獸人頭目
// 射程是弓的射程，而且在敵方回合（foeTurn）從射程外直接射人，不先走過去。

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
)

type monsterRecomputeReceipt struct {
	Generator string `json:"generator"`
	Records   []struct {
		Index  int    `json:"index"`
		Name   string `json:"name"`
		THAC0  uint8  `json:"thac0_110h"`
		AC     uint8  `json:"ac_111h"`
		Dice   string `json:"runtime_dice_114h_11Ah"`
		Move   uint8  `json:"move_11Ch"`
		Weapon string `json:"weapon_0CCh"`
	} `json:"records"`
}

func orcHomeGearFixture(t *testing.T) (*app, monsterRecomputeReceipt) {
	t.Helper()
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	if _, err := os.Stat(zipPath); err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join("..", "..", "docs", "audit", "dosgolem-monster-recompute-runtime.json"))
	if err != nil {
		t.Fatal(err)
	}
	var receipt monsterRecomputeReceipt
	if err := json.Unmarshal(raw, &receipt); err != nil {
		t.Fatal(err)
	}
	if receipt.Generator != "dosgolem" || len(receipt.Records) != 20 {
		t.Fatalf("receipt from %q with %d records", receipt.Generator, len(receipt.Records))
	}
	walk, err := os.ReadFile(filepath.Join("..", "..", "docs", "audit", "dosgolem-deployment-peek-orc-home.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fight struct {
		PartyCell struct{ X, Y, Facing uint8 } `json:"party_cell"`
		Spawns    [][3]uint8                   `json:"spawns"`
	}
	if err := json.Unmarshal(walk, &fight); err != nil {
		t.Fatal(err)
	}
	return newDeploymentFixture(t, zipPath, fight.PartyCell.X, fight.PartyCell.Y, fight.PartyCell.Facing, fight.Spawns, false), receipt
}

func TestMonsterGearMatchesTheOrcHomeRuntimeReceipt(t *testing.T) {
	application, receipt := orcHomeGearFixture(t)
	state := application.tactical
	armed := 0
	for _, want := range receipt.Records {
		index := want.Index
		if index >= len(state.Roster) || state.Friendly[index] {
			t.Fatalf("receipt combatant %d (%s) is not a foe on the remake board", index, want.Name)
		}
		dice, err := hex.DecodeString(want.Dice)
		if err != nil {
			t.Fatal(err)
		}
		// `+114h..+11Ah`：+1／+3／+5 是第一種形態的顆數、面數、加值。
		wantForm := combat.DamageDice{Count: dice[1], Sides: dice[3], Bonus: int8(dice[5])}
		wantRange := 1
		if want.Weapon != "00000000" {
			armed++
			// 短弓（型別 2Bh）：型別表 `+0Ch` 是射程加一。
			entry, err := application.itemTypes.Entry(0x2B)
			if err != nil {
				t.Fatal(err)
			}
			wantRange = entry.AttackRange()
		}
		if state.THAC0[index] != want.THAC0 || state.ArmorClass[index] != int(want.AC) ||
			state.BaseMovement[index] != want.Move || state.AttackForms[index][0] != wantForm ||
			state.AttackRange[index] != wantRange {
			t.Errorf("combatant %d %s: remake THAC0 %d AC %d move %d form %+v range %d; runtime %d %d %d %+v %d",
				index, want.Name, state.THAC0[index], state.ArmorClass[index], state.BaseMovement[index],
				state.AttackForms[index][0], state.AttackRange[index],
				want.THAC0, want.AC, want.Move, wantForm, wantRange)
		}
	}
	if armed != 1 {
		t.Fatalf("%d armed leaders in the receipt, want 1", armed)
	}
}

// 兩隻獸人頭目放在同一格、面對同一批隊員。敵方回合接近之前先跑 overlay-09 entry 9
// （`13D5h`，spec 151）重挑武器：7 號（MON2 block 14）兩件武器都沒穿、箭穿著，身邊沒人
// 就把短弓穿上，在原地射人（一次攻擊、零步）；6 號（block 15）釘頭錘與短弓都穿著，
// 三隻手超過兩隻，entry 9 把弓卸下，拿釘頭錘往前走。
func TestArmedOrcLeaderShootsFromOutsideMeleeReach(t *testing.T) {
	for _, testCase := range []struct {
		mover   uint8
		shoots  bool
		comment string
	}{
		{7, true, "unworn short bow readied by entry 9"},
		{6, false, "mace and short bow both worn: entry 9 puts the bow away"},
	} {
		application, _ := orcHomeGearFixture(t)
		state := application.tactical
		mover := testCase.mover
		// 把行動者擺到離隊員兩、三格、近戰搆不到而弓搆得到的地方（先挑一個隊員，往它的八個方向找）。
		var victim uint8
		for index := 1; index < len(state.Roster); index++ {
			if state.Friendly[index] && state.Roster[index].FootprintClass != 0 {
				victim = uint8(index)
				break
			}
		}
		if victim == 0 {
			t.Fatal("no party member on the board")
		}
		// 其他怪物全部移出盤面，只留行動者，距離判斷才不會被擋。
		for index := 1; index < len(state.Roster); index++ {
			if !state.Friendly[index] && uint8(index) != mover {
				state.Roster[index].FootprintClass = 0
			}
		}
		placed := false
		here := state.Roster[victim]
		for _, offset := range [][2]int{{0, -3}, {0, 3}, {3, 0}, {-3, 0}, {3, 3}, {-3, -3}, {3, -3}, {-3, 3}, {0, -2}, {0, 2}, {2, 0}, {-2, 0}} {
			x, y := int(here.X)+offset[0], int(here.Y)+offset[1]
			if x < 0 || y < 0 {
				continue
			}
			nearest := 99
			for index := 1; index < len(state.Roster); index++ {
				if state.Friendly[index] && state.Roster[index].FootprintClass != 0 {
					if d := chebyshev(uint8(x), uint8(y), state.Roster[index].X, state.Roster[index].Y); d < nearest {
						nearest = d
					}
				}
			}
			// 只挑與行動者原本站的那一格同一種地面的格子（不擺進牆裡）。
			floor, err := state.Grid.TerrainAt(int(state.Roster[mover].X), int(state.Roster[mover].Y))
			if err != nil {
				t.Fatal(err)
			}
			if terrain, err := state.Grid.TerrainAt(x, y); err != nil || terrain != floor || nearest < 2 {
				continue
			}
			state.Roster[mover].X, state.Roster[mover].Y = uint8(x), uint8(y)
			snapshot, err := state.tacticalSnapshot()
			if err != nil {
				t.Fatal(err)
			}
			side, _ := state.sideOf(mover)
			inReach, err := combat.OpposingNearbyAt(snapshot, mover, uint8(x), uint8(y), 1, 1-side, state.sideOf)
			if err != nil {
				t.Fatal(err)
			}
			// 弓搆得到（直線追蹤沒被牆擋），近戰搆不到：兩個情形用同一格。
			inBowRange, err := combat.OpposingNearbyAt(snapshot, mover, uint8(x), uint8(y),
				uint16(state.AttackRange[6]), 1-side, state.sideOf)
			if err != nil {
				t.Fatal(err)
			}
			if len(inReach) == 0 && len(inBowRange) != 0 {
				placed = true
				break
			}
		}
		if !placed {
			t.Fatalf("%s: no cell three squares from the party is free of melee reach", testCase.comment)
		}
		state.Mover = mover
		state.FoeTargets[mover] = victim
		state.Budgets[mover] = state.BaseMovement[mover] * 2
		before := state.Roster[mover]
		activity := state.Activity
		if err := application.foeTurn(state); err != nil {
			t.Fatal(err)
		}
		attacked := state.Activity.FoeAttacks - activity.FoeAttacks
		moved := state.Roster[mover].X != before.X || state.Roster[mover].Y != before.Y
		if testCase.shoots && (attacked != 1 || moved) {
			t.Errorf("%s: %d attacks, moved %v (log %q); want one shot from where it stood",
				testCase.comment, attacked, moved, state.FoeLog)
		}
		// 沒拿武器的要先走（走到相鄰之後可能同一回合就出手，所以不看攻擊次數）。
		if !testCase.shoots && !moved {
			t.Errorf("%s: %d attacks, moved %v (log %q); want it to close in first",
				testCase.comment, attacked, moved, state.FoeLog)
		}
	}
}
