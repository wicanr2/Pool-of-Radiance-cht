package main

import (
	"fmt"
	"math/rand"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/golden-box-remake-engine/geometry"
)

// 六個種子各走 6000 步，量「走得到多少內容」並且擋住硬失敗。
//
// 遊戲總共有 29 個有文字的 ECL 區塊（docs/audit/dos-ecl-text-inventory.json）。
// 隨機走路走不到大部分——主線要有目的地才走得到——所以這一條**不是**進度指標，
// 它擋的是兩件事：走著走著炸掉，以及原本走得到的地方變成走不到。
func TestRandomWalkReachesKnownContentWithoutFailing(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	maps := map[string]int{}
	blocks := map[int]int{}
	fights := 0
	failures := 0
	for _, seed := range []int64{3, 7, 11, 29, 41, 55} {
		application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
		if err != nil {
			t.Skipf("no zip: %v", err)
		}
		application.roller = diceRoller{random: rand.New(rand.NewSource(seed))}
		party := make([]poolsave.Character, 0, 6)
		for index := 0; index < 6; index++ {
			party = append(party, poolsave.Character{Name: string(rune('A' + index)),
				RaceID: "dwarf", GenderID: "male", ClassID: "fighter",
				AlignmentID: "lawful-good", Abilities: [6]int{18, 10, 10, 16, 10, 10},
				MaxHP: 60, CurrentHP: 60, PortraitHead: 1, PortraitBody: 1, IconSize: 1})
		}
		application.state = poolsave.State{Schema: poolsave.Schema,
			CharacterLibrary: party, Party: party}
		application.saveState = func(poolsave.State) error { return nil }
		press(application, ebiten.KeyEnter)
		press(application, ebiten.KeyB)
		for tick := 0; tick < 20000 && !application.introDone; tick++ {
			if application.introWaiting || application.tourPage >= 0 {
				press(application, ebiten.KeyEnter)
				continue
			}
			application.keys = scriptedKeys{}
			application.Update()
		}
		keys := []ebiten.Key{ebiten.KeyArrowUp, ebiten.KeyArrowLeft,
			ebiten.KeyArrowRight, ebiten.KeyArrowDown}
		random := rand.New(rand.NewSource(seed))
		for step := 0; step < 6000; step++ {
			if application.initialMap != nil {
				maps[fmt.Sprintf("GEO%d/%d", application.spawn.Map.Archive,
					application.spawn.Map.BlockID)]++
			}
			if application.eventSession != nil {
				blocks[int(application.eventSession.CurrentBlockID())]++
			}
			busy := application.encounter != nil || application.cellWaitingMenu ||
				application.cellEventPending || application.combatActive ||
				application.shopActive || application.treasureActive ||
				application.templeActive || application.tactical != nil
			if busy {
				if application.tactical != nil {
					fights++
					if application.tactical.Prompt {
						if err := press(application, ebiten.KeyY); err != nil {
							t.Logf("種子 %d 第 %d 步硬失敗：%v", seed, step, err)
							break
						}
						continue
					}
				}
				if application.cellWaitingMenu && len(application.cellMenuOptions) > 1 {
					for k := random.Intn(len(application.cellMenuOptions)); k > 0; k-- {
						if err := press(application, ebiten.KeyArrowDown); err != nil {
							t.Logf("種子 %d 第 %d 步硬失敗：%v", seed, step, err)
							break
						}
					}
				}
				if err := press(application, ebiten.KeyEnter); err != nil {
					t.Errorf("種子 %d 第 %d 步硬失敗：%v", seed, step, err)
					failures++
					break
				}
				continue
			}
			if err := press(application, keys[random.Intn(len(keys))]); err != nil {
				t.Errorf("種子 %d 第 %d 步硬失敗：%v", seed, step, err)
				failures++
				break
			}
		}
	}
	mapNames := make([]string, 0, len(maps))
	for name := range maps {
		mapNames = append(mapNames, name)
	}
	sort.Strings(mapNames)
	blockIDs := make([]int, 0, len(blocks))
	for id := range blocks {
		blockIDs = append(blockIDs, id)
	}
	sort.Ints(blockIDs)
	t.Logf("走到的地圖 %d 張：%v", len(maps), mapNames)
	t.Logf("走到的 ECL block %d 個：%v", len(blocks), blockIDs)
	t.Logf("戰鬥 tick 數：%d", fights)
	if failures != 0 {
		t.Fatalf("六個種子走完出現 %d 次硬失敗", failures)
	}
	// 走得到的下限。少於這個數代表移動或轉場退步了。
	if len(maps) < 2 {
		t.Errorf("只走到 %d 張地圖，先前量到 2 張", len(maps))
	}
	if len(blocks) < 4 {
		t.Errorf("只走到 %d 個 ECL block，先前量到 4 個", len(blocks))
	}
	if fights == 0 {
		t.Error("整輪都沒有打到架，戰鬥沒有被走到")
	}
}

// exploreStep 是「往哪一格走」的一步：先轉到 direction，再往前。
type exploreStep struct{ facing uint8 }

// exploreDeltas 是四個朝向的位移，0 北、1 東、2 南、3 西（spec 076）。
var exploreDeltas = [4][2]int{{0, -1}, {1, 0}, {0, 1}, {-1, 0}}

// explorePlan 從目前這一格廣度優先找到最近的一格「還沒踩過的」，
// 回傳走過去要的朝向序列。走不到就回 nil。
//
// 用的是原始 GEO 的 CanMoveDungeonWrapped，跟遊戲自己判斷能不能走同一支，
// 所以這條路徑不會宣告出資料裡沒有的通路。
func explorePlan(app *app, visited, avoid map[[3]int]bool, rotate int) []exploreStep {
	type node struct{ x, y int }
	start := node{int(app.spawn.X), int(app.spawn.Y)}
	from := map[node]node{start: start}
	via := map[node]uint8{}
	queue := []node{start}
	for len(queue) != 0 {
		current := queue[0]
		queue = queue[1:]
		key := [3]int{int(app.spawn.Map.Archive), int(app.spawn.Map.BlockID), current.y*100 + current.x}
		if current != start && !visited[key] && !avoid[key] {
			steps := []exploreStep{}
			for cursor := current; cursor != start; cursor = from[cursor] {
				steps = append([]exploreStep{{facing: via[cursor]}}, steps...)
			}
			return steps
		}
		for step := 0; step < 4; step++ {
			// rotate 讓每一趟從不同的方向先展開，見 planToCells 的說明。
			facing := (step + rotate) % 4
			if !app.initialMap.Grid.CanMoveDungeonWrapped(current.x, current.y, facing*2) {
				continue
			}
			next := node{
				x: geometry.WrapCoordinate(current.x+exploreDeltas[facing][0], geometry.Width),
				y: geometry.WrapCoordinate(current.y+exploreDeltas[facing][1], geometry.Height),
			}
			if _, seen := from[next]; seen {
				continue
			}
			from[next], via[next] = current, uint8(facing)
			queue = append(queue, next)
		}
	}
	return nil
}

// treasureMenuChoice 挑寶物那一串選單要停在哪一項：認得出 Exit 就選 Exit，
// 認不出而第一項是 Yes 就選 Yes，都不是才選最後一項。
//
// 「一律選最後一項」會卡死：Exit 之後還有一句 Yes／No 的確認，最後一項是
// **No**，於是選單原地重開，一直繞。
func treasureMenuChoice(options []string) int {
	for index, option := range options {
		if strings.EqualFold(option, "Exit") {
			return index
		}
	}
	if len(options) != 0 && strings.EqualFold(options[0], "Yes") {
		return 0
	}
	return len(options) - 1
}

// tacticalPilot 是探索用的「自己人怎麼打」：先試瞄準，打不到就往最近的敵人
// 走一步再試，走不動或走夠了就結束回合。
//
// **這不是原版的演算法**，只是讓探索不會停在打不完的架上。只按 Enter 的話
// 全隊都不出手，怪物也殺不完，那一場永遠結束不了——探索看起來像走不動，
// 實際上是卡在戰鬥裡。
type tacticalPilot struct {
	mover uint8
	round int
	tried bool
	moves int
	// goal 與 distance 是這一個回合的目標與步數表。每一 tick 重算一次
	// 廣度優先會讓探索的時間全花在重算上——盤面在同一個回合裡不會變到
	// 需要重算的程度。
	goal     uint8
	distance map[int]int
}

// exploreMaxCombatSteps 是一個角色一回合最多走幾步。原版有移動額度擋著，
// 這裡另外加一個上限，免得額度算法出錯時無限走下去。
const exploreMaxCombatSteps = 12

func (pilot *tacticalPilot) key(app *app) ebiten.Key {
	state := app.tactical
	if state.Prompt {
		// 敵方清光之後那一次問的是「還要不要繼續打」（spec 062）。答 Y
		// 會再開一輪，於是那一場永遠結束不了——開打與收工共用同一個旗標。
		if state.sideCounts().Foes == 0 {
			return ebiten.KeyN
		}
		return ebiten.KeyY
	}
	if app.castTargeting {
		return ebiten.KeyEnter
	}
	if state.Mover == 0 || int(state.Mover) >= len(state.Friendly) ||
		!state.Friendly[state.Mover] {
		return ebiten.KeyEnter
	}
	if pilot.mover != state.Mover || pilot.round != state.Round {
		pilot.mover, pilot.round = state.Mover, state.Round
		pilot.tried, pilot.moves = false, 0
		pilot.goal, pilot.distance = 0, nil
	}
	if !pilot.tried {
		pilot.tried = true
		if pilot.adjacentToFoe(state) {
			return ebiten.KeyA
		}
	}
	if pilot.moves >= exploreMaxCombatSteps {
		return ebiten.KeyEnter
	}
	if pilot.distance == nil || state.Roster[pilot.goal].FootprintClass == 0 {
		target, ok := state.nearestReachableOpposing(state.Mover)
		if !ok {
			return ebiten.KeyEnter
		}
		pilot.goal = target
		pilot.distance = tacticalStepDistances(state.Grid, state.Classes,
			state.Roster[target].X, state.Roster[target].Y)
	}
	goal := state.Roster[pilot.goal]
	distance := pilot.distance
	here := state.Roster[state.Mover]
	best, bestDistance := -1, tacticalDistanceAt(distance, here.X, here.Y, goal)
	for direction := uint8(0); direction < combat.DirectionCount; direction++ {
		step, err := combat.DirectionStep(direction)
		if err != nil {
			continue
		}
		nextX := int(here.X) + int(step.X)
		nextY := int(here.Y) + int(step.Y)
		if nextX < 0 || nextY < 0 || nextX > combat.TacticalMaxX || nextY > combat.TacticalMaxY {
			continue
		}
		candidate := tacticalDistanceAt(distance, uint8(nextX), uint8(nextY), goal)
		if candidate < bestDistance {
			best, bestDistance = int(direction), candidate
		}
	}
	if best < 0 {
		return ebiten.KeyEnter
	}
	pilot.moves++
	// 走完只有站到敵人旁邊才值得再試一次瞄準。每走一步都按一次 A 會讓
	// 一場架的 tick 數翻倍，探索的預算就全花在打不到的瞄準上。
	pilot.tried = !pilot.adjacentToFoe(state)
	return tacticalStepKeys[best]
}

// adjacentToFoe 說目前這一格旁邊有沒有站著還在場的敵人。
func (pilot *tacticalPilot) adjacentToFoe(state *tacticalState) bool {
	here := state.Roster[state.Mover]
	for index := 1; index < len(state.Roster); index++ {
		if index == int(state.Mover) || state.Roster[index].FootprintClass == 0 {
			continue
		}
		if state.Friendly[index] == state.Friendly[state.Mover] {
			continue
		}
		if chebyshev(here.X, here.Y, state.Roster[index].X, state.Roster[index].Y) <= 1 {
			return true
		}
	}
	return false
}

// planToCells 找到最近的一格目標並回傳走過去的朝向序列，走不到就回 nil。
// wanted 收的是「這一格是不是要去的」。
func planToCells(app *app, rotate int, wanted func(x, y int) bool) []exploreStep {
	type node struct{ x, y int }
	start := node{int(app.spawn.X), int(app.spawn.Y)}
	from := map[node]node{start: start}
	via := map[node]uint8{}
	queue := []node{start}
	for len(queue) != 0 {
		current := queue[0]
		queue = queue[1:]
		if current != start && wanted(current.x, current.y) {
			steps := []exploreStep{}
			for cursor := current; cursor != start; cursor = from[cursor] {
				steps = append([]exploreStep{{facing: via[cursor]}}, steps...)
			}
			return steps
		}
		for step := 0; step < 4; step++ {
			// rotate 讓每一趟從不同的方向先展開。牆是**單向**的（走得過去
			// 不代表走得回來），所以固定順序每一趟都會走進同一個死角；
			// 換個順序就會換一條路。
			facing := (step + rotate) % 4
			if !app.initialMap.Grid.CanMoveDungeonWrapped(current.x, current.y, facing*2) {
				continue
			}
			next := node{
				x: geometry.WrapCoordinate(current.x+exploreDeltas[facing][0], geometry.Width),
				y: geometry.WrapCoordinate(current.y+exploreDeltas[facing][1], geometry.Height),
			}
			if _, seen := from[next]; seen {
				continue
			}
			from[next], via[next] = current, uint8(facing)
			queue = append(queue, next)
		}
	}
	return nil
}

// sokalKeepPassword 是鬼魂在索寇要塞說出來的通關密語（ecl4 block 21 `AD42h`）。
const sokalKeepPassword = "SAMOSUD"

// exploreMaxTransitionHops 是「這一張走完了，回頭走另一個換圖點」最多做幾次。
// 沒有上限的話，所有圖都走完之後兩張圖之間會一直來回。
const exploreMaxTransitionHops = 60

// 有目的地走：把每一張圖上「走得到的格子」逐格踩過，換圖就換到新圖上繼續。
//
// 隨機走路量的是「不會炸掉」，這一條量的是**世界有多少走得到**——兩個不同的
// 問題。主線要能破關，第一件事是玩家真的走得到那些地方。
// exploreWorld 走一趟：從新的一局開始，把目前這一張圖上還沒踩過而且不在
// avoid 裡的格子逐格踩完，踩完了才走一個換圖點，換圖之後在新圖上繼續。
//
// avoid 收的是**已知的換圖點**。少了它，第一趟會在起點附近就踩到碼頭上船，
// 之後困在索寇要塞回不來——起始圖 226 格只走了 31 格就再也沒機會走完。
// 逐趟把已知的換圖點擋掉，下一趟就會先把這一張走完再換圖。
func exploreWorld(t *testing.T, zipPath string, seed int64, rotate, rewalkLimit, budget int,
	avoid, walked map[[3]int]bool, transitionUses, menuTurn map[[3]int]int,
	visited map[[3]int]bool, maps map[string]bool, blocks map[int]bool) (int, bool) {
	return exploreWorldWithFlags(t, zipPath, seed, rotate, rewalkLimit, budget,
		avoid, walked, transitionUses, menuTurn, visited, maps, blocks, nil)
}

// exploreWorldWithFlags 與 exploreWorld 相同，另外在結束時把幾個 ECL 變數
// 抄進 flags，讓呼叫端可以斷言主線推到哪裡。
func exploreWorldWithFlags(t *testing.T, zipPath string, seed int64, rotate, rewalkLimit, budget int,
	avoid, walked map[[3]int]bool, transitionUses, menuTurn map[[3]int]int,
	visited map[[3]int]bool, maps map[string]bool, blocks map[int]bool,
	flags map[uint16]uint16) (int, bool) {
	t.Helper()
	application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		return 0, false
	}
	application.roller = diceRoller{random: rand.New(rand.NewSource(seed))}
	party := make([]poolsave.Character, 0, 6)
	for index := 0; index < 6; index++ {
		party = append(party, poolsave.Character{Name: string(rune('A' + index)),
			RaceID: "dwarf", GenderID: "male", ClassID: "fighter",
			AlignmentID: "lawful-good", Abilities: [6]int{18, 10, 10, 16, 10, 10},
			MaxHP: 60, CurrentHP: 60, PortraitHead: 1, PortraitBody: 1, IconSize: 1})
	}
	application.state = poolsave.State{Schema: poolsave.Schema,
		CharacterLibrary: party, Party: party}
	application.saveState = func(poolsave.State) error { return nil }
	press(application, ebiten.KeyEnter)
	press(application, ebiten.KeyB)
	for tick := 0; tick < 20000 && !application.introDone; tick++ {
		if application.introWaiting || application.tourPage >= 0 {
			press(application, ebiten.KeyEnter)
			continue
		}
		application.keys = scriptedKeys{}
		application.Update()
	}

	var plan []exploreStep
	pilot := &tacticalPilot{}
	stuck, hops, moved := 0, 0, 0
	// walked 由呼叫端給：量覆蓋率時直接傳 visited（跨趟累積，不重做已經走過
	// 的路），要重走找出口時傳一份自己的。
	rewalks := map[[2]int]int{}
	lastMap, lastCell := application.spawn.Map, [2]int{-1, -1}
	for step := 0; step < budget; step++ {
		if application.spawn.Map != lastMap {
			// 換圖不是在「按下前進」那一 tick 發生的：格子事件先跑，
			// LOAD FILES 是在事件那一段做掉的。所以要跨 tick 比對，
			// 記下**換圖前站的那一格**。少了這個，avoid 永遠是空的，
			// 逐趟走就只剩「重開一局」的效果。
			if lastCell[0] >= 0 {
				key := [3]int{int(lastMap.Archive), int(lastMap.BlockID),
					lastCell[1]*100 + lastCell[0]}
				transitionUses[key]++
				avoid[key] = true
			}
			lastMap = application.spawn.Map
		}
		lastCell = [2]int{int(application.spawn.X), int(application.spawn.Y)}
		if application.initialMap != nil {
			maps[fmt.Sprintf("GEO%d/%d", application.spawn.Map.Archive,
				application.spawn.Map.BlockID)] = true
			key := [3]int{int(application.spawn.Map.Archive),
				int(application.spawn.Map.BlockID),
				int(application.spawn.Y)*100 + int(application.spawn.X)}
			visited[key], walked[key] = true, true
		}
		if application.eventSession != nil {
			blocks[int(application.eventSession.CurrentBlockID())] = true
		}
		busy := application.encounter != nil || application.cellWaitingMenu ||
			application.cellEventPending || application.combatActive ||
			application.shopActive || application.treasureActive ||
			application.templeActive || application.tactical != nil ||
			application.programAsking || application.parlay != nil ||
			application.whoPending || application.mode != modeAdventure
		if application.programManaging {
			// 地圖上的隊伍管理畫面吃掉方向鍵。原版按 B 回地圖。
			plan = nil
			if err := press(application, ebiten.KeyB); err != nil {
				t.Fatalf("第 %d 步硬失敗：%v", step, err)
			}
			continue
		}
		if application.treasureActive && len(application.cellMenuOptions) != 0 {
			// 寶物選單停在 View 上，一直按 Enter 只會一直看，出不去。
			plan = nil
			key := ebiten.KeyEnter
			if want := treasureMenuChoice(application.cellMenuOptions); want != application.cellMenuCursor {
				key = ebiten.KeyArrowRight
			}
			if err := press(application, key); err != nil {
				t.Fatalf("第 %d 步硬失敗：%v", step, err)
			}
			continue
		}
		if application.eclInput != nil {
			// 索寇要塞的不死者會問通關密語（`INPUT STRING`，spec 087）。
			// 密語是鬼魂在遊戲裡說的：「TO PASS MY GUARDS ON THE WAY OUT,
			// SPEAK THE WORD 'SAMOSUD'」。不打進去就出不了那張圖。
			plan = nil
			if application.eclInput.buffer == "" && !application.eclInput.numeric {
				application.keys = scriptedChars(sokalKeepPassword)
				if err := application.Update(); err != nil {
					t.Fatalf("第 %d 步硬失敗：%v", step, err)
				}
				continue
			}
			if err := press(application, ebiten.KeyEnter); err != nil {
				t.Fatalf("第 %d 步硬失敗：%v", step, err)
			}
			continue
		}
		if busy {
			plan = nil
			if application.tactical != nil {
				if err := press(application, pilot.key(application)); err != nil {
					t.Fatalf("第 %d 步硬失敗：%v", step, err)
				}
				continue
			}
			if application.cellWaitingMenu && len(application.cellMenuOptions) > 1 {
				// 每一格的選單輪流選不同的選項。一律停在第 0 項的話，
				// 「要不要離開這裡」這種問句永遠答同一個答案。
				key := [3]int{int(application.spawn.Map.Archive),
					int(application.spawn.Map.BlockID),
					int(application.spawn.Y)*100 + int(application.spawn.X)}
				want := menuTurn[key] % len(application.cellMenuOptions)
				if application.cellMenuCursor != want {
					if err := press(application, ebiten.KeyArrowRight); err != nil {
						t.Fatalf("第 %d 步硬失敗：%v", step, err)
					}
					continue
				}
				menuTurn[key]++
			}
			if err := press(application, ebiten.KeyEnter); err != nil {
				t.Fatalf("第 %d 步硬失敗：%v", step, err)
			}
			continue
		}
		if len(plan) == 0 {
			plan = explorePlan(application, walked, avoid, rotate)
			if len(plan) == 0 && hops < exploreMaxTransitionHops {
				// 這一張踩完了：走一個用得最少的換圖點。
				here := [2]int{int(application.spawn.Map.Archive),
					int(application.spawn.Map.BlockID)}
				fewest := -1
				for key, count := range transitionUses {
					if key[0] != here[0] || key[1] != here[1] {
						continue
					}
					if fewest < 0 || count < fewest {
						fewest = count
					}
				}
				if fewest >= 0 {
					plan = planToCells(application, rotate, func(x, y int) bool {
						return transitionUses[[3]int{here[0], here[1], y*100 + x}] == fewest
					})
					if len(plan) != 0 {
						hops++
					}
				}
			}
			if len(plan) == 0 {
				here := [2]int{int(application.spawn.Map.Archive),
					int(application.spawn.Map.BlockID)}
				if rewalks[here] < rewalkLimit {
					rewalks[here]++
					for key := range walked {
						if key[0] == here[0] && key[1] == here[1] {
							delete(walked, key)
						}
					}
					continue
				}
				stuck++
				if stuck > 3 {
					break
				}
				continue
			}
			stuck = 0
		}
		want := plan[0]
		if application.spawn.Facing != want.facing {
			key := ebiten.KeyArrowRight
			if (int(want.facing)-int(application.spawn.Facing)+4)%4 == 3 {
				key = ebiten.KeyArrowLeft
			}
			if err := press(application, key); err != nil {
				t.Fatalf("第 %d 步硬失敗：%v", step, err)
			}
			continue
		}
		before := application.spawn
		if err := press(application, ebiten.KeyArrowUp); err != nil {
			t.Fatalf("第 %d 步硬失敗：%v", step, err)
		}
		if application.spawn.Map != before.Map ||
			(application.spawn.X == before.X && application.spawn.Y == before.Y) {
			plan = nil
			continue
		}
		moved++
		plan = plan[1:]
	}
	if flags != nil && application.eventMachine != nil {
		for _, address := range []uint16{0x4A21, 0x4AA7, 0x4AC4, 0x6E12} {
			flags[address] = application.eventMachine.Memory[address]
		}
	}
	return moved, true
}

// 有目的地走完整個世界：逐格踩，換圖就換到新圖上繼續，走完再開新的一局
// 把上一趟提早換掉的圖補完。
//
// 隨機走路量的是「不會炸掉」，走不到的地方它分不出來是「原作沒有」還是
// 「remake 走不進去」。這一條量的是**世界有多少走得到**——主線要能破關，
// 第一件事是玩家真的走得到那些地方。
func TestDirectedExplorationReachesMaps(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	avoid := map[[3]int]bool{}
	transitionUses := map[[3]int]int{}
	// menuTurn 跨趟保留：每一格的選單逐趟換一個答案。留在單趟裡的話每趟都
	// 從第 0 項開始，「要不要接任務」這種問句永遠是同一個答案。
	menuTurn := map[[3]int]int{}
	visited := map[[3]int]bool{}
	maps := map[string]bool{}
	blocks := map[int]bool{}
	total := 0
	for pass := 0; pass < 24; pass++ {
		moved, ok := exploreWorld(t, zipPath, int64(7+pass), pass%4, 0, 300000,
			avoid, visited, transitionUses, menuTurn, visited, maps, blocks)
		if !ok {
			t.Skip("original DOS ZIP is intentionally not tracked")
		}
		total += moved
	}

	mapNames := make([]string, 0, len(maps))
	for name := range maps {
		mapNames = append(mapNames, name)
	}
	sort.Strings(mapNames)
	blockIDs := make([]int, 0, len(blocks))
	for id := range blocks {
		blockIDs = append(blockIDs, id)
	}
	sort.Ints(blockIDs)
	perMap := map[string]int{}
	for key := range visited {
		perMap[fmt.Sprintf("GEO%d/%d", key[0], key[1])]++
	}
	t.Logf("走到的地圖 %d 張：%v", len(maps), mapNames)
	t.Logf("走到的 ECL block %d 個：%v", len(blockIDs), blockIDs)
	t.Logf("踩過的格子 %d 格，走了 %d 步", len(visited), total)
	for _, name := range mapNames {
		t.Logf("  %s 踩過 %d 格", name, perMap[name])
	}
	// 走得到的下限。這是**量到的數字**，不是目標。少於這個數代表移動、
	// 轉場或戰鬥退步了。
	if len(visited) < 445 {
		t.Errorf("只踩到 %d 格，先前量到 448 格（兩張圖都踩滿）", len(visited))
	}
	if len(blocks) < 4 {
		t.Errorf("只走到 %d 個 ECL block", len(blocks))
	}
}

// 主線第一段：清掉索寇要塞會把碼頭的其他航線開出來（spec 099）。
//
// 這一條與覆蓋率那一條問的不是同一件事。覆蓋率量「走得到多少」，這一條量
// **主線推得動嗎**：拿到裝備、打贏那一場、鬼魂說出 SAMOSUD，`DS:4AA7h`
// 才會被寫成 254，碼頭才會從「唯一的船是去索寇要塞的」變成五個選項。
func TestSokalKeepOpensTheOtherBoatRoutes(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	avoid := map[[3]int]bool{}
	transitionUses := map[[3]int]int{}
	menuTurn := map[[3]int]int{}
	visited := map[[3]int]bool{}
	maps := map[string]bool{}
	blocks := map[int]bool{}
	flags := map[uint16]uint16{}
	// 重走同一張圖是為了再踩到出口那一格：出口不會被記成「還沒踩過」，
	// 所以踩完一遍之後規劃器就不會再挑它，隊伍會困在那一張圖上。
	_, ok := exploreWorldWithFlags(t, zipPath, 7, 0, 6, 600000,
		avoid, map[[3]int]bool{}, transitionUses, menuTurn, visited, maps, blocks, flags)
	if !ok {
		t.Skip("original DOS ZIP is intentionally not tracked")
	}
	t.Logf("走到的地圖：%d 張；ECL block：%d 個", len(maps), len(blocks))
	t.Logf("旗標 4A21=%d（要塞的裝備與那一場架）4AA7=%d（碼頭航線）4AC4=%d 6E12=%d",
		flags[0x4A21], flags[0x4AA7], flags[0x4AC4], flags[0x6E12])
	if flags[0x4A21] != 255 {
		t.Errorf("要塞那一段沒推完：4A21=%d，要 255", flags[0x4A21])
	}
	if flags[0x4AA7] != 254 {
		t.Errorf("碼頭的航線沒開：4AA7=%d，要 254", flags[0x4AA7])
	}
}
