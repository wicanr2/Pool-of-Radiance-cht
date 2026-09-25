package main

// 敵方走位對原版，一個動作一個動作對，而且骰照原版的骰流餵（spec 096）：三筆收據
// （docs/audit/dosgolem-deployment-peek-{orc-home,guards,alarm}.json）裡每一幀位置表的變動，
// 就是原版某一隻在那一次行動走到了哪一格；`dice` 是同一場裡 Turbo Pascal `Random(n)` 的
// 每一次呼叫（dosgolem `-trace-call 5BB:C94`）。一隻怪的一個回合在骰流裡長這樣：
//
//	[d4 沿不沿用模式] [d8 (d4|d2) 重擲模式] d7 d7 [d(n) 挑目標…] 走 [d(n) 卡住第二次重挑…] [d(n) d20 傷害]
//
// n 是對面還站著的人數。這裡對每一個原版走過的動作，把 remake 的盤擺成動作前那一幀、
// 模式與目標照骰流設、卡住重挑的骰照原版餵，要求 `foeTurn` 停在原版停的那一格。模式與
// 沿用的目標跨動作追蹤（remake 的 `TacticModes`／`FoeTargets`），骰流的形狀對不上追蹤
// 的狀態時退回「模式與目標各試一遍」並另外計數。d7 那一對是 overlay-09 entry 3（用物品）
// 與 entry 4（挑法術）的次數骰，remake 由 foeCastPhase 照原版的位置擲（spec 096），這裡
// 從骰流原樣餵進去；d100 是選下一個行動者的決勝骰（spec 052），不進走位。

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	// Dice 的每一筆是 [批, 指令步數, 骰面, 結果, 呼叫端]。
	Dice []diceRoll `json:"dice"`
}

type diceRoll struct {
	Step        uint64
	Sides, Roll int
}

func (roll *diceRoll) UnmarshalJSON(raw []byte) error {
	var fields []json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return err
	}
	if len(fields) != 5 {
		return fmt.Errorf("dice entry has %d fields, want 5", len(fields))
	}
	if err := json.Unmarshal(fields[1], &roll.Step); err != nil {
		return err
	}
	if err := json.Unmarshal(fields[2], &roll.Sides); err != nil {
		return err
	}
	return json.Unmarshal(fields[3], &roll.Roll)
}

// scriptedRoller 照劇本回骰：骰面要對得上，對不上或劇本用完就記一筆、回 1。
type scriptedRoller struct {
	script   []diceRoll
	next     int
	mismatch []string
}

func (roller *scriptedRoller) Roll(count, sides int) int {
	if count != 1 || roller.next >= len(roller.script) || roller.script[roller.next].Sides != sides {
		if sides != 100 {
			roller.mismatch = append(roller.mismatch, fmt.Sprintf("asked %dd%d at script %d/%d", count, sides, roller.next, len(roller.script)))
		}
		return count
	}
	value := roller.script[roller.next].Roll
	roller.next++
	return value
}

// originalAction 是原版某一隻連續走的一段：從哪一格出發、每一步落在哪、出發前的盤。
type originalAction struct {
	mover  uint8
	from   [2]uint8
	path   [][2]uint8
	before [][4]uint8
	// prevStep 是動作前最後一幀的指令步數，stepAt 是每一步那一幀的。
	prevStep uint64
	stepAt   []uint64
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
			current.stepAt = append(current.stepAt, receipt.Actions[i].Step)
			continue
		}
		actions = append(actions, originalAction{mover: mover, from: from, path: [][2]uint8{to}, before: before,
			prevStep: receipt.Actions[i-1].Step, stepAt: []uint64{receipt.Actions[i].Step}})
		current = &actions[len(actions)-1]
	}
	return actions
}

func TestFoeWalkReproducesEveryOriginalAction(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	total, reproduced, searched := 0, 0, 0
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
			for _, action := range groupActions(receipt) {
				total++
				want := action.path[len(action.path)-1]
				alive := []uint8{}
				for _, entry := range action.before {
					if entry[0] >= 1 && entry[0] <= 5 && entry[3] != 0 {
						alive = append(alive, entry[0])
					}
				}
				tracked := 0
				if int(action.mover) < len(state.TacticModes) {
					tracked = int(state.TacticModes[action.mover])
				}
				stickyTarget, sticky := state.foeTarget(action.mover)
				plan := planTurn(receipt.Dice, action, len(alive), tracked)
				// 原版的候選名單是 `0912h` 依直線追蹤成本排的（spec 096 `37B8h`），擲出
				// 幾就是名單的第幾個；擺好動作前那一幀再算一次同一份名單。
				place(action.before)
				candidates := []uint8{}
				if side, ok := state.sideOf(action.mover); ok {
					var err error
					if candidates, err = state.foeTargetCandidates(action.mover, side); err != nil {
						t.Fatal(err)
					}
				}
				if len(candidates) != len(alive) {
					plan.notes = append(plan.notes, fmt.Sprintf("candidate list has %d entries but %d members stand", len(candidates), len(alive)))
				}
				targets := []uint8{}
				switch {
				case plan.pick > 0 && plan.pick <= len(candidates):
					targets = []uint8{candidates[plan.pick-1]}
				case plan.pick > 0:
					targets = []uint8{alive[plan.pick-1]}
					plan.notes = append(plan.notes, "pick beyond the candidate list: index order used")
				case sticky:
					targets = []uint8{stickyTarget}
				default:
					targets = alive
					plan.notes = append(plan.notes, "target not in the dice and none tracked: every standing member tried")
				}
				found := ""
				for _, candidate := range plan.modes {
					for _, target := range targets {
						if found != "" {
							break
						}
						place(action.before)
						state.Mover = action.mover
						state.Budgets[action.mover] = combat.InitialMovementBudgetBeforeEffects(state.BaseMovement[action.mover], false, 0)
						state.setFoeTarget(action.mover, target)
						state.setTacticMode(action.mover, tracked)
						script := append(modeScript(tracked, candidate.mode, candidate.keep), plan.sevens...)
						roller := &scriptedRoller{script: append(script, plan.rePicks...)}
						application.roller = roller
						if err := application.foeTurn(state); err != nil {
							t.Fatalf("foeTurn %d: %v", action.mover, err)
						}
						got := state.Roster[action.mover]
						if got.X == want[0] && got.Y == want[1] {
							found = fmt.Sprintf("mode %d%s target %d, %d re-picks%s", candidate.mode,
								map[bool]string{true: " (kept)", false: ""}[candidate.keep], target, len(plan.rePicks), roller.summary())
						}
					}
				}
				if found == "" {
					t.Logf("NOT reproduced: foe %d went (%d,%d)→%v in the original; dice give modes %v targets %v re-picks %v; notes %v",
						action.mover, action.from[0], action.from[1], action.path, plan.modes, targets, plan.rePicks, plan.notes)
					continue
				}
				if len(plan.modes) == 1 && len(targets) == 1 {
					reproduced++
				} else {
					searched++
				}
				t.Logf("foe %d (%d,%d)→%v reproduced with %s; notes %v", action.mover, action.from[0], action.from[1], action.path, found, plan.notes)
			}
		})
	}
	t.Logf("original foe actions reproduced: %d/%d straight from the dice stream, %d more where the dice leave the mode or target ambiguous", reproduced, total, searched)
	// 骰流接上之後三場 40 個動作全部走到原版那一格（playtest 補八 8e）。這條是地板：
	// 改走位規則不准讓它掉下去。
	if reproduced+searched < total {
		t.Errorf("only %d/%d original foe actions reproduced", reproduced+searched, total)
	}
}

// turnPlan 是從骰流讀出來的一個回合：模式的候選（骰流能定就一個）、開場挑到誰
// （alive 名單的第幾個，0 是沿用）、卡住重挑要餵給 remake 的骰。
type turnPlan struct {
	modes   []modeCandidate
	pick    int
	rePicks []diceRoll
	// sevens 是模式與挑目標之間那一對 d7（entry 3 與 entry 4 的次數骰）。
	sevens []diceRoll
	notes  []string
}

type modeCandidate struct {
	mode int
	keep bool
}

// modeScript 是讓 remake 的 `RollTacticMode` 從追蹤到的模式走到 want 要餵的骰：
// 追蹤的落在 1..4 時它先擲 d4 問沿不沿用（1 才重擲），重擲時 d8 擲 8 才走 d2+4。
func modeScript(tracked, want int, keep bool) []diceRoll {
	script := []diceRoll{}
	if tracked >= 1 && tracked <= 4 {
		if keep && tracked == want {
			return []diceRoll{{Sides: 4, Roll: 2}}
		}
		script = append(script, diceRoll{Sides: 4, Roll: 1})
	}
	if want >= 5 {
		return append(script, diceRoll{Sides: 8, Roll: 8}, diceRoll{Sides: 2, Roll: want - 4})
	}
	return append(script, diceRoll{Sides: 8, Roll: 1}, diceRoll{Sides: 4, Roll: want})
}

func (roller *scriptedRoller) summary() string {
	if len(roller.mismatch) == 0 {
		return ""
	}
	return " (roller: " + strings.Join(roller.mismatch, ", ") + ")"
}

// planTurn 讀動作前的骰流：走之前最後一段是「[d4 沿用] [d8 (d4|d2)] d7 d7 [d(n)…]」。
// tracked 是 remake 追蹤到的這一隻的模式；骰流的形狀跟它對不上時（那一隻在沒走的
// 回合裡換過模式）以骰流為準，兩種讀法都說得通時兩個都列。
func planTurn(dice []diceRoll, action originalAction, alive, tracked int) turnPlan {
	var pre []diceRoll
	for _, roll := range dice {
		if roll.Step > action.prevStep && roll.Step <= action.stepAt[0] && roll.Sides != 100 {
			pre = append(pre, roll)
		}
	}
	plan := turnPlan{}
	i := len(pre)
	for i > 0 && pre[i-1].Sides == alive {
		i--
	}
	picks := pre[i:]
	if len(picks) > 0 {
		plan.pick = picks[len(picks)-1].Roll
	}
	for k := 1; k < len(action.stepAt); k++ {
		var last *diceRoll
		for j := range dice {
			if dice[j].Step > action.stepAt[k-1] && dice[j].Step <= action.stepAt[k] && dice[j].Sides == alive {
				last = &dice[j]
			}
		}
		if last != nil {
			plan.rePicks = append(plan.rePicks, *last)
		}
	}
	inKept := tracked >= 1 && tracked <= 4
	if i < 2 || pre[i-1].Sides != 7 || pre[i-2].Sides != 7 {
		plan.notes = append(plan.notes, fmt.Sprintf("no d7 pair before the walk (pre-walk dice %v): every mode tried", pre))
		for mode := 1; mode <= gamepack.TacticModes; mode++ {
			plan.modes = append(plan.modes, modeCandidate{mode: mode})
		}
		return plan
	}
	i -= 2
	plan.sevens = append([]diceRoll(nil), pre[i:i+2]...)
	fresh := func(at int) (int, bool) {
		// pre[at] 是 d8，pre[at+1] 是 d4 或 d2。
		if at < 0 || at+1 >= len(pre) || pre[at].Sides != 8 {
			return 0, false
		}
		if pre[at].Roll == 8 && pre[at+1].Sides == 2 {
			return pre[at+1].Roll + 4, true
		}
		if pre[at].Roll != 8 && pre[at+1].Sides == 4 {
			return pre[at+1].Roll, true
		}
		return 0, false
	}
	keepRoll := i >= 1 && pre[i-1].Sides == 4 && pre[i-1].Roll != 1
	freshMode, isFresh := fresh(i - 2)
	switch {
	case isFresh && i >= 3 && pre[i-3].Sides == 4 && pre[i-3].Roll == 1:
		// 沿用失敗（d4=1）再重擲：一定是重擲。
		plan.modes = []modeCandidate{{mode: freshMode}}
	case isFresh && !keepRoll:
		plan.modes = []modeCandidate{{mode: freshMode}}
	case keepRoll && !isFresh:
		if inKept {
			plan.modes = []modeCandidate{{mode: tracked, keep: true}}
		} else {
			plan.notes = append(plan.notes, "the dice kept a mode the remake never saw: 1..4 tried")
			for mode := 1; mode <= 4; mode++ {
				plan.modes = append(plan.modes, modeCandidate{mode: mode, keep: true})
			}
		}
	case keepRoll && isFresh:
		// 「d8 d4」既可能是重擲，也可能是別人的傷害骰接著這一隻的沿用骰。
		if inKept {
			plan.modes = []modeCandidate{{mode: tracked, keep: true}, {mode: freshMode}}
			plan.notes = append(plan.notes, "d8 before the keep roll may be damage: kept mode tried first")
		} else {
			plan.modes = []modeCandidate{{mode: freshMode}}
		}
	default:
		plan.notes = append(plan.notes, fmt.Sprintf("no mode block before the d7 pair (pre-walk dice %v): every mode tried", pre))
		for mode := 1; mode <= gamepack.TacticModes; mode++ {
			plan.modes = append(plan.modes, modeCandidate{mode: mode})
		}
	}
	return plan
}
