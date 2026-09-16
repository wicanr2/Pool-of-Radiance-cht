package main

// 敵方走位對原版，一個動作一個動作對（spec 096）：三筆收據（docs/audit/
// dosgolem-deployment-peek-{orc-home,guards,alarm}.json）裡每一幀位置表的變動，就是原版
// 某一隻在那一次行動走到了哪一格。誰先動、追誰是擲骰（先攻、`37B8h`），骰流對不上；
// 但「給定同一張盤、同一個目標、同一個戰術模式，原版走出來的那一格 remake 的走法能不能
// 走到」不吃骰。這裡對每一個原版走過的動作，把 remake 的盤擺成前一幀、目標與模式各試一遍
// （五個隊員 × 六個模式），要求至少有一組讓 `foeTurn` 停在原版停的那一格。走不到的列出來，
// 那才是規則差。

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/golden-box-remake-engine/eclvm"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

type dosgolemWalkReceipt struct {
	PartyCell struct {
		X, Y   uint8
		Facing uint8
	} `json:"party_cell"`
	Spawns  [][3]uint8 `json:"spawns"`
	Actions []struct {
		Batch int        `json:"round_batch"`
		Step  uint64     `json:"step"`
		Table [][4]uint8 `json:"table"`
	} `json:"actions"`
}

// originalAction 是原版某一隻連續走的一段：從哪一格出發、每一步落在哪、出發前的盤。
type originalAction struct {
	mover  uint8
	from   [2]uint8
	path   [][2]uint8
	before [][4]uint8
}

// groupActions 把逐步快照（一筆一步）合成動作：同一隻連續動就是同一個動作，
// 中間夾了別人的動作或死亡就斷開。
func groupActions(receipt dosgolemWalkReceipt) []originalAction {
	var actions []originalAction
	var current *originalAction
	for i := 1; i < len(receipt.Actions); i++ {
		before, after := receipt.Actions[i-1].Table, receipt.Actions[i].Table
		var mover uint8
		var from, to [2]uint8
		moved := 0
		for k := range after {
			if k >= len(before) {
				break
			}
			if before[k][1] != after[k][1] || before[k][2] != after[k][2] {
				moved++
				mover = after[k][0]
				from = [2]uint8{before[k][1], before[k][2]}
				to = [2]uint8{after[k][1], after[k][2]}
			}
		}
		if moved != 1 || mover <= 5 {
			current = nil // 死亡或多人同時變動：斷開
			continue
		}
		if current != nil && current.mover == mover {
			current.path = append(current.path, to)
			continue
		}
		actions = append(actions, originalAction{mover: mover, from: from, path: [][2]uint8{to}, before: before})
		current = &actions[len(actions)-1]
	}
	return actions
}

func TestFoeWalkReproducesEveryOriginalAction(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	total, reproduced := 0, 0
	for _, name := range []string{"orc-home", "guards", "alarm"} {
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join("..", "..", "docs", "audit", "dosgolem-deployment-peek-"+name+".json"))
			if err != nil {
				t.Fatal(err)
			}
			var receipt dosgolemWalkReceipt
			if err := json.Unmarshal(raw, &receipt); err != nil {
				t.Fatal(err)
			}
			application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
			if err != nil {
				t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
			}
			archive, ok := application.eclCatalog.Archive(2)
			if !ok {
				t.Fatal("ECL2 archive is absent")
			}
			session, err := gamepack.NewDOSECLArchiveSession(archive, 20, 0x9914)
			if err != nil {
				t.Fatal(err)
			}
			geoMap, ok := application.geometryCatalog.Map(gamepack.MapKey{Archive: 2, BlockID: 20})
			if !ok {
				t.Fatal("GEO2/20 is absent")
			}
			application.eventSession, application.eventMachine = session, session.Machine()
			application.eclArchive = 2
			application.initialMap = &geoMap
			application.spawn = gamepack.Spawn{Map: geoMap.Key, X: receipt.PartyCell.X, Y: receipt.PartyCell.Y, Facing: receipt.PartyCell.Facing}
			application.introDone, application.mode = true, modeAdventure
			if err := application.configureEventSession(session); err != nil {
				t.Fatal(err)
			}
			application.saveState = func(poolsave.State) error { return nil }
			application.eventMachine.Memory[encounterWalkFlagAddress] = 1
			application.eventMachine.Memory[encounterDistanceAddress] = 0
			party := make([]poolsave.Character, 0, 5)
			for index, hp := range []int{9, 6, 12, 6, 9} {
				party = append(party, poolsave.Character{Name: string(rune('B' + index)), RaceID: "dwarf",
					GenderID: "male", ClassID: "fighter", AlignmentID: "lawful-good",
					Abilities: [6]int{15, 10, 10, 13, 10, 10}, MaxHP: hp, CurrentHP: hp,
					PortraitHead: 1, PortraitBody: 1, IconSize: 1})
			}
			application.state = poolsave.State{Schema: poolsave.Schema, CharacterLibrary: party, Party: party}
			spawns := make([]eclvm.MonsterSpawn, 0, len(receipt.Spawns))
			for _, spawn := range receipt.Spawns {
				spawns = append(spawns, eclvm.MonsterSpawn{MonsterID: spawn[0], Count: spawn[1], IconBlock: spawn[2]})
			}
			if err := application.enterCombatStaging(spawns); err != nil {
				t.Fatal(err)
			}
			if err := press(application, ebiten.KeyEnter); err != nil {
				t.Fatal(err)
			}
			state := application.tactical
			if state == nil {
				t.Fatal("no tactical state")
			}
			hitPoints := append([]int(nil), state.HitPoints...)
			states := append([]uint8(nil), state.States...)

			// 把盤擺成某一幀：座標與體型類別照原版的位置表。
			place := func(table [][4]uint8) {
				// 收據的每一筆是 (索引, X, Y, 體型類別)。
				for _, entry := range table {
					index := int(entry[0])
					if index <= 0 || index >= len(state.Roster) {
						continue
					}
					state.Roster[index].X, state.Roster[index].Y = entry[1], entry[2]
					state.Roster[index].FootprintClass = entry[3]
				}
				copy(state.HitPoints, hitPoints)
				copy(state.States, states)
				state.Finished, state.Prompt = false, false
			}
			partyIndices := []uint8{1, 2, 3, 4, 5}
			for _, action := range groupActions(receipt) {
				total++
				want := action.path[len(action.path)-1]
				found, tried := "", ""
				for _, target := range partyIndices {
					if found != "" {
						break
					}
					standing := false
					for _, entry := range action.before {
						if entry[0] == target && entry[3] != 0 {
							standing = true
						}
					}
					if !standing {
						continue
					}
					// 卡住第二次時的重挑目標是擲骰（`38A6h`），換幾個骰種子各試一遍。
					for attempt := 0; attempt < 36 && found == ""; attempt++ {
						mode := attempt%6 + 1
						application.roller = diceRoller{random: rand.New(rand.NewSource(int64(attempt/6 + 1)))}
						place(action.before)
						state.Mover = action.mover
						state.Budgets[action.mover] = combat.InitialMovementBudgetBeforeEffects(state.BaseMovement[action.mover], false, 0)
						state.setFoeTarget(action.mover, target)
						state.setTacticMode(action.mover, mode)
						if err := application.foeTurn(state); err != nil {
							t.Fatalf("foeTurn %d: %v", action.mover, err)
						}
						got := state.Roster[action.mover]
						if attempt == 0 {
							tried += fmt.Sprintf(" t%d→(%d,%d)/%s", target, got.X, got.Y, state.FoeLog)
						}
						if got.X == want[0] && got.Y == want[1] {
							found = fmt.Sprintf("target %d mode %d", target, mode)
						}
					}
				}
				if found == "" {
					t.Logf("NOT reproduced: foe %d went (%d,%d)→%v in the original; no target/mode/dice tried makes the remake stop there (mode 1:%s)",
						action.mover, action.from[0], action.from[1], action.path, tried)
					continue
				}
				reproduced++
				t.Logf("foe %d (%d,%d)→%v reproduced with %s", action.mover, action.from[0], action.from[1], action.path, found)
			}
		})
	}
	t.Logf("original foe actions reproduced: %d/%d", reproduced, total)
	// 2026-09-16 量到 32/40：沒對上的八個都是連續反向步（卡住→換模式→重挑目標）
	// 那一類，重挑是擲骰，要同骰流（#34）。這條是地板：改走位規則不准讓它掉下去。
	if reproduced < 32 {
		t.Errorf("only %d/%d original foe actions reproduced; the floor is 32", reproduced, total)
	}
	_ = hex.DecodeString
}
