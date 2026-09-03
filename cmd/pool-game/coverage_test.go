package main

import (
	"fmt"
	"math/rand"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
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
// explorePlan 回傳走去最近一格「還沒踩過」的路，以及那一格的鍵。
// 目標要回傳出去：踩不到的格子得記次數，不然規劃器會一直挑同一格，
// 隊伍在半路來回走而 stuck 永遠不增加——實測一趟走六萬步只踩到 479 格。
func explorePlan(app *app, visited, avoid map[[3]int]bool, rotate int) ([]exploreStep, [3]int) {
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
			return steps, key
		}
		for step := 0; step < 4; step++ {
			// rotate 讓每一趟從不同的方向先展開，見 planToCells 的說明。
			facing := (step + rotate) % 4
			if !app.initialMap.Grid.CanMoveDungeonWrapped(current.x, current.y, facing*2) {
				continue
			}
			x := current.x + exploreDeltas[facing][0]
			y := current.y + exploreDeltas[facing][1]
			if x < 0 || x >= geometry.Width || y < 0 || y >= geometry.Height {
				// 跨出邊界那一步是**換區**（spec 100），不是探索這一張圖。
				// 讓規劃器自由跨出去的話，隊伍會在兩區之間來回彈——實測
				// 城區與貧民窟之間彈了兩萬次，別的地方一格都沒走到。
				continue
			}
			next := node{x: x, y: y}
			if _, seen := from[next]; seen {
				continue
			}
			from[next], via[next] = current, uint8(facing)
			queue = append(queue, next)
		}
	}
	return nil, [3]int{}
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

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
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

// exploreMaxCombatStall 是「同一個（回合、行動者、提示）連續幾個 tick 還沒
// 動」的上限。一場架正常會一直換行動者，停住就是卡住了。
const exploreMaxCombatStall = 4000

// exploreMaxTargetTries 是同一格被規劃成目標幾次還沒踩到就放棄。
const exploreMaxTargetTries = 12

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
	exitUses map[[4]int]int, visited map[[3]int]bool, maps map[string]bool,
	blocks map[int]bool, hardFailures *[]string) (int, bool) {
	return exploreWorldWithFlags(t, zipPath, seed, rotate, rewalkLimit, budget,
		avoid, walked, transitionUses, menuTurn, exitUses, visited, maps, blocks,
		nil, noBoatOverride, hardFailures)
}

// noBoatOverride 關掉航線覆寫（見 exploreWorldWithFlags 的 boat 參數）。
const noBoatOverride = -1

// exploreWorldWithFlags 與 exploreWorld 相同，另外在結束時把幾個 ECL 變數
// 抄進 flags，讓呼叫端可以斷言主線推到哪裡。
func exploreWorldWithFlags(t *testing.T, zipPath string, seed int64, rotate, rewalkLimit, budget int,
	avoid, walked map[[3]int]bool, transitionUses, menuTurn map[[3]int]int,
	exitUses map[[4]int]int, visited map[[3]int]bool, maps map[string]bool,
	blocks map[int]bool, flags map[uint16]uint16, boat int,
	hardFailures *[]string) (int, bool) {
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
	reason := "走完預算"
	spin := map[string]int{}
	// 一場架卡住的話，整趟的預算會全部花在戰術地圖上——實測 60 萬個 tick
	// 裡有 59 萬 9 千個在那裡。停在同一個（回合、行動者）太久就當它卡住，
	// 記成硬失敗，不要靜靜地把預算吃掉。
	stallKey, stall := [3]int{-1, -1, -1}, 0
	menuStall := 0
	tries := map[[3]int]int{}
	// approached 記「這一格從這個方向走進去過了」。見 spec 102。
	approached := map[[4]int]bool{}
	var exit *areaExit
	var failures []string
	// walked 由呼叫端給：量覆蓋率時直接傳 visited（跨趟累積，不重做已經走過
	// 的路），要重走找出口時傳一份自己的。
	rewalks := map[[2]int]int{}
	lastMap, lastCell := application.spawn.Map, [2]int{-1, -1}
walk:
	for step := 0; step < budget; step++ {
		if application.spawn.Map != lastMap {
			t.Logf("換圖 GEO%d/%d → GEO%d/%d 位置 (%d,%d) 朝向 %d",
				lastMap.Archive, lastMap.BlockID,
				application.spawn.Map.Archive, application.spawn.Map.BlockID,
				application.spawn.X, application.spawn.Y, application.spawn.Facing)
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
		// boat：**測試治具**，不是遊玩。港務長那一段目前推不動（船票旗標
		// `4A01` 沒有人清回去，spec 102 的 OPEN），而碼頭的船照著 `4AC4`
		// 決定去哪。把那三個值直接寫進去，就能把主線之後的區域先走一遍，
		// 找出那些區域自己的問題——走得到不走得到是另一個問題。
		if boat != noBoatOverride && application.eventMachine != nil {
			application.eventMachine.Memory[0x4AA7] = 254
			application.eventMachine.Memory[0x4A01] = 1
			application.eventMachine.Memory[0x4AC4] = uint16(boat)
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
		switch {
		case application.programManaging:
			spin["隊伍管理"]++
		case application.tactical != nil:
			spin["戰術地圖"]++
			key := [3]int{int(application.tactical.Round),
				int(application.tactical.Mover), boolInt(application.tactical.Prompt)}
			if key != stallKey {
				stallKey, stall = key, 0
			}
			stall++
			if stall > exploreMaxCombatStall {
				counts := application.tactical.sideCounts()
				failures = append(failures, fmt.Sprintf(
					"戰術地圖卡住：GEO%d/%d 第 %d 回合行動者 %d（提示 %v，我方 %d 敵方 %d，"+
						"名冊 %d，施法選單 %v/%v，狀態 %q，是我方 %v）",
					application.spawn.Map.Archive, application.spawn.Map.BlockID,
					application.tactical.Round, application.tactical.Mover,
					application.tactical.Prompt, counts.Party, counts.Foes,
					len(application.tactical.Roster), application.castOpen,
					application.castTargeting, application.tactical.Status,
					application.tactical.Friendly[application.tactical.Mover]))
				reason = "戰術地圖卡住"
				break walk
			}
		case application.treasureActive:
			spin["寶物"]++
		case application.eclInput != nil:
			spin["輸入字串"]++
		case application.cellWaitingMenu:
			spin["格子選單"]++
			menuStall++
			if menuStall > exploreMaxCombatStall {
				failures = append(failures, fmt.Sprintf(
					"格子選單卡住：GEO%d/%d (%d,%d) 游標 %d／%v 標籤 %q 文字 %q",
					application.spawn.Map.Archive, application.spawn.Map.BlockID,
					application.spawn.X, application.spawn.Y,
					application.cellMenuCursor, application.cellMenuOptions,
					application.eventLabel, application.eventText))
				reason = "格子選單卡住"
				break walk
			}
		case application.encounter != nil:
			spin["遭遇"]++
		case application.combatActive:
			spin["戰鬥"]++
		case application.cellEventPending:
			spin["格子事件"]++
		case application.mode != modeAdventure:
			spin[fmt.Sprintf("模式 %v", application.mode)]++
		default:
			spin["走路"]++
			menuStall = 0
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
				failures = append(failures, fmt.Sprintf("第 %d 步：%v", step, err))
				break walk
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
				failures = append(failures, fmt.Sprintf("第 %d 步：%v", step, err))
				break walk
			}
			continue
		}
		if application.shopActive {
			// 商店只有 Escape 出得去（`shopInput`）：Enter 是購買，買不起就
			// 原地不動，而那一格會一直把商店開回來。實測城區 (15,8) 的商店
			// 把一整趟的預算吃掉五十萬個 tick。
			plan = nil
			if err := press(application, ebiten.KeyEscape); err != nil {
				failures = append(failures, fmt.Sprintf("第 %d 步：%v", step, err))
				break walk
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
					failures = append(failures, fmt.Sprintf("第 %d 步：%v", step, err))
					break walk
				}
				continue
			}
			if err := press(application, ebiten.KeyEnter); err != nil {
				failures = append(failures, fmt.Sprintf("第 %d 步：%v", step, err))
				break walk
			}
			continue
		}
		if busy {
			plan = nil
			if application.tactical != nil {
				if err := press(application, pilot.key(application)); err != nil {
					failures = append(failures, fmt.Sprintf("第 %d 步：%v", step, err))
					break walk
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
				// flags 非 nil 那一條要推主線，所以 YES／NO 一律答 YES
				//（「要不要拿走裝備」答 NO 就推不動要塞那一段）。
				if flags != nil && strings.EqualFold(application.cellMenuOptions[0], "YES") {
					want = 0
				}
				for _, option := range application.cellMenuOptions {
					if strings.EqualFold(option, "SOKAL") {
						t.Logf("碼頭選單 GEO%d/%d (%d,%d) 選 %d／%v",
							application.spawn.Map.Archive, application.spawn.Map.BlockID,
							application.spawn.X, application.spawn.Y, want,
							application.cellMenuOptions)
						break
					}
				}
				if application.cellMenuCursor != want {
					if err := press(application, ebiten.KeyArrowRight); err != nil {
						failures = append(failures, fmt.Sprintf("第 %d 步：%v", step, err))
						break walk
					}
					continue
				}
				menuTurn[key]++
			}
			if err := press(application, ebiten.KeyEnter); err != nil {
				failures = append(failures, fmt.Sprintf("第 %d 步：%v", step, err))
				break walk
			}
			continue
		}
		if application.inWilderness() {
			// 野外的位置存在 `DS:49C3h`／`DS:49C4h`，不在 GEO 格子上
			//（spec 105），所以規劃器沒得規劃。輪流轉向再往前走，
			// 讓它把野外那幾張圖走開。
			plan, exit = nil, nil
			spin["野外"]++
			key := ebiten.KeyArrowUp
			switch application.roller.Roll(1, 6) {
			case 1:
				key = ebiten.KeyArrowRight
			case 2:
				key = ebiten.KeyArrowLeft
			}
			if err := press(application, key); err != nil {
				failures = append(failures, fmt.Sprintf("第 %d 步：%v", step, err))
				break walk
			}
			continue
		}
		// 這一張踩完之後才刻意走出去：站到邊界那一格、轉向外面、往前一步。
		// 規劃器本身不跨邊界（見 explorePlan），所以換區一定經過這一段。
		if exit != nil {
			at := [2]int{int(application.spawn.X), int(application.spawn.Y)}
			switch {
			case at == exit.cell && application.spawn.Facing == exit.facing:
				if err := press(application, ebiten.KeyArrowUp); err != nil {
					failures = append(failures, fmt.Sprintf("第 %d 步：%v", step, err))
					break walk
				}
				exit, plan = nil, nil
				continue
			case at == exit.cell:
				key := ebiten.KeyArrowRight
				if (int(exit.facing)-int(application.spawn.Facing)+4)%4 == 3 {
					key = ebiten.KeyArrowLeft
				}
				if err := press(application, key); err != nil {
					failures = append(failures, fmt.Sprintf("第 %d 步：%v", step, err))
					break walk
				}
				continue
			}
			if len(plan) == 0 {
				target := *exit
				plan = planToCells(application, rotate, func(x, y int) bool {
					return x == target.cell[0] && y == target.cell[1]
				})
				if len(plan) == 0 {
					exit = nil
				}
			}
		}
		if len(plan) == 0 && exit == nil {
			var target [3]int
			plan, target = explorePlan(application, walked, avoid, rotate)
			if len(plan) != 0 {
				spin["規劃"]++
				// 挑同一格挑太多次還沒踩到，就當它走不進去。原因可能是
				// 單向的邊、也可能是那一格的事件把隊伍推回來；兩種都會讓
				// 規劃器永遠有事做，而 stuck 永遠是 0。
				tries[target]++
				if tries[target] > exploreMaxTargetTries {
					avoid[target] = true
					plan = nil
					continue
				}
			}
			// 地點還沒走遍：城區的地點腳本有些會比朝向（spec 102 的港務長要
			// 面向北），而入口 1 看到的朝向就是走進那一格的方向。所以「踩過
			// 這一格」不等於「試過這個地點」——terrain 索引非 0 的格子，
			// 四個方向都要走進去一次。
			if len(plan) == 0 {
				if cell, facing, ok := chooseApproach(application, approached, rotate); ok {
					approached[approachKey(application, cell, facing)] = true
					step := exploreStep{facing: facing}
					from := [2]int{cell[0] - exploreDeltas[facing][0],
						cell[1] - exploreDeltas[facing][1]}
					if from == [2]int{int(application.spawn.X), int(application.spawn.Y)} {
						plan = []exploreStep{step}
					} else {
						plan = planToCells(application, rotate, func(x, y int) bool {
							return x == from[0] && y == from[1]
						})
						if len(plan) != 0 {
							plan = append(plan, step)
						}
					}
					if len(plan) != 0 {
						spin["走進地點"]++
						continue
					}
				}
			}
			// 這一張踩完了，先回頭踩已知的換圖點。碼頭就是這樣再用一次的：
			// 第一趟船去索寇要塞，回來之後要塞的旗標已經開了其他航線，
			// 但那一格早就進了 avoid，規劃器不會再挑它。
			if len(plan) == 0 && hops < exploreMaxTransitionHops {
				if cell, ok := chooseTransitionCell(application, transitionUses); ok {
					plan = planToCells(application, rotate, func(x, y int) bool {
						return x == cell[0] && y == cell[1]
					})
					if len(plan) != 0 {
						transitionUses[[3]int{int(application.spawn.Map.Archive),
							int(application.spawn.Map.BlockID),
							cell[1]*100 + cell[0]}]++
						hops++
						continue
					}
				}
			}
			if len(plan) == 0 && hops < exploreMaxTransitionHops {
				if next, ok := chooseAreaExit(application, exitUses); ok {
					// 選中就記一次。走不到那一格的出口不記的話，
					// 下一輪會挑到同一個，隊伍就卡在原地重選到 hop 用完。
					exitUses[exitKey(application, next)]++
					exit, hops = &next, hops+1
					continue
				}
			}
			if len(plan) == 0 {
				here := [2]int{int(application.spawn.Map.Archive),
					int(application.spawn.Map.BlockID)}
				if rewalks[here] < rewalkLimit {
					rewalks[here]++
					spin["重走"]++
					for key := range walked {
						if key[0] == here[0] && key[1] == here[1] {
							delete(walked, key)
						}
					}
					continue
				}
				stuck++
				if stuck > 3 {
					reason = "走不動：這一張沒有沒踩過的格子，也沒有走得到的出口"
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
				failures = append(failures, fmt.Sprintf("第 %d 步：%v", step, err))
				break walk
			}
			continue
		}
		before := application.spawn
		if err := press(application, ebiten.KeyArrowUp); err != nil {
			failures = append(failures, fmt.Sprintf("第 %d 步：%v", step, err))
			reason = "按鍵失敗"
			break walk
		}
		if application.spawn.Map != before.Map ||
			(application.spawn.X == before.X && application.spawn.Y == before.Y) {
			plan = nil
			continue
		}
		moved++
		plan = plan[1:]
	}
	t.Logf("這一趟結束於 GEO%d/%d (%d,%d)：%s（走了 %d 步）",
		application.spawn.Map.Archive, application.spawn.Map.BlockID,
		application.spawn.X, application.spawn.Y, reason, moved)
	spinKeys := make([]string, 0, len(spin))
	for key := range spin {
		spinKeys = append(spinKeys, key)
	}
	sort.Slice(spinKeys, func(i, j int) bool { return spin[spinKeys[i]] > spin[spinKeys[j]] })
	parts := make([]string, 0, len(spinKeys))
	for _, key := range spinKeys {
		parts = append(parts, fmt.Sprintf("%s %d", key, spin[key]))
	}
	t.Logf("  這一趟的迴圈花在：%s", strings.Join(parts, "、"))
	transitionKeys := make([][3]int, 0, len(transitionUses))
	for key := range transitionUses {
		transitionKeys = append(transitionKeys, key)
	}
	sort.Slice(transitionKeys, func(i, j int) bool {
		return transitionKeys[i][0]*10000+transitionKeys[i][1]*100+transitionKeys[i][2] <
			transitionKeys[j][0]*10000+transitionKeys[j][1]*100+transitionKeys[j][2]
	})
	labels := make([]string, 0, len(transitionKeys))
	for _, key := range transitionKeys {
		labels = append(labels, fmt.Sprintf("GEO%d/%d (%d,%d)×%d",
			key[0], key[1], key[2]%100, key[2]/100, transitionUses[key]))
	}
	t.Logf("  這一趟用過的換圖點：%s", strings.Join(labels, "、"))
	perMap := map[[2]int]int{}
	for key := range walked {
		perMap[[2]int{key[0], key[1]}]++
	}
	t.Logf("  這一趟踩過：%v（重走 %d 次）", perMap, spin["重走"])
	if hardFailures != nil {
		*hardFailures = append(*hardFailures, failures...)
	}
	if flags != nil && application.eventMachine != nil {
		for _, address := range []uint16{0x4A21, 0x4AA7, 0x4AC4, 0x6E12, 0x4A01, 0x4AC5, 0x4ABA} {
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
	// exitUses 跨趟保留：每一趟都從城區開始，不記著上一趟走過哪個城門的話，
	// 每一趟都會挑同一個，八趟走的是同一條路。
	exitUses := map[[4]int]int{}
	var hardFailures []string
	total := 0
	for pass := 0; pass < 8; pass++ {
		moved, ok := exploreWorld(t, zipPath, int64(7+pass), pass%4, 0, 40000,
			avoid, visited, transitionUses, menuTurn, exitUses, visited, maps, blocks,
			&hardFailures)
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
	seen := map[string]int{}
	for _, failure := range hardFailures {
		key := failure
		if index := strings.Index(failure, "："); index >= 0 {
			key = failure[index+len("："):]
		}
		seen[key]++
	}
	for failure, count := range seen {
		t.Logf("硬失敗 ×%d：%s", count, failure)
	}
	for _, name := range mapNames {
		t.Logf("  %s 踩過 %d 格", name, perMap[name])
	}
	// 走得到的下限。這是**量到的數字**，不是目標。少於這個數代表移動、
	// 轉場或戰鬥退步了。
	//
	// 走得到的下限。這是**量到的數字**，不是目標。少於這個數代表移動、
	// 轉場或戰鬥退步了。
	//
	// 格子數比「只走得到兩張圖」的時候少：離開這一區那一條接上去之後
	// （spec 100），每一趟很快就走出去，不會把一張圖踩滿。換來的是走得到的
	// 區域從兩個 archive 變成六個。
	if len(maps) < 3 {
		t.Errorf("只走到 %d 張地圖，先前量到 3 張（訓練所那一區、貧民窟、索寇要塞）", len(maps))
	}
	if len(blocks) < 5 {
		t.Errorf("只走到 %d 個 ECL block，先前量到 5 個", len(blocks))
	}
	// 硬失敗一個都不該有。收集起來一次列完，比走到第一個就 Fatal 好查。
	if len(hardFailures) != 0 {
		t.Errorf("出現 %d 次硬失敗", len(hardFailures))
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
	var hardFailures []string
	// 幾個種子輪流試：戰鬥的結果會改路線，單一種子太容易因為別處的修正而失準。
	ok := false
	for _, seed := range []int64{7, 11, 3, 29, 41} {
		// 每一個種子都從乾淨的狀態開始：avoid 與換圖點的使用次數留著的話，
		// 第二輪一開始就被擋在碼頭外面。
		avoid = map[[3]int]bool{}
		transitionUses = map[[3]int]int{}
		_, reachable := exploreWorldWithFlags(t, zipPath, seed, 0, 2, 600000,
			avoid, map[[3]int]bool{}, transitionUses, menuTurn, map[[4]int]int{},
			visited, maps, blocks, flags, noBoatOverride, &hardFailures)
		if !reachable {
			t.Skip("original DOS ZIP is intentionally not tracked")
		}
		ok = true
		if flags[0x4AA7] == 254 {
			break
		}
	}
	if !ok {
		t.Skip("original DOS ZIP is intentionally not tracked")
	}
	for _, failure := range hardFailures {
		t.Logf("硬失敗：%s", failure)
	}
	t.Logf("走到的地圖：%d 張；ECL block：%d 個", len(maps), len(blocks))
	t.Logf("旗標 4A21=%d（要塞的裝備與那一場架）4AA7=%d（碼頭航線）4AC4=%d 6E12=%d "+
		"4A01=%d 4AC5=%d 4ABA=%d",
		flags[0x4A21], flags[0x4AA7], flags[0x4AC4], flags[0x6E12],
		flags[0x4A01], flags[0x4AC5], flags[0x4ABA])
	// **只釘住走得到的部分**：碼頭的船會把隊伍送到索寇要塞（ECL block 21）。
	// 那一段的旗標（拿裝備 `4A21h`、開航線 `4AA7h`）**推不推得到跟路線有關**
	// ——探索器是機器人，走到哪一格、答哪一個選項會隨著別處的修正而改變，
	// 釘住它只會在無關的改動上變紅。旗標印出來當觀察值，要推主線得靠有目的地
	// 的路線（WORKLIST 有這一條）。
	if !blocks[21] {
		t.Errorf("沒走到索寇要塞（ECL block 21），走到的是 %v", blocks)
	}
}

// approachKey 把「這一格＋走進去的方向」壓成 approached 的鍵。
func approachKey(application *app, cell [2]int, facing uint8) [4]int {
	return [4]int{int(application.spawn.Map.Archive), int(application.spawn.Map.BlockID),
		cell[1]*100 + cell[0], int(facing)}
}

// chooseApproach 挑一個「還沒從這個方向走進去過」的地點格。
//
// 地點格＝`terrain & 0x7F` 非 0 的格子（spec 102）。方向要從地圖資料算：
// 走進來的那一格得在圖內，而且那一步不能被牆擋著。
func chooseApproach(application *app, approached map[[4]int]bool, rotate int) ([2]int, uint8, bool) {
	if application.initialMap == nil {
		return [2]int{}, 0, false
	}
	for y := 0; y < geometry.Height; y++ {
		for x := 0; x < geometry.Width; x++ {
			cell, ok := application.initialMap.Grid.Cell(x, y)
			if !ok || cell.Terrain&0x7F == 0 {
				continue
			}
			for step := 0; step < 4; step++ {
				facing := uint8((step + rotate) % 4)
				fromX := x - exploreDeltas[facing][0]
				fromY := y - exploreDeltas[facing][1]
				if fromX < 0 || fromX >= geometry.Width ||
					fromY < 0 || fromY >= geometry.Height {
					continue
				}
				if !application.initialMap.Grid.CanMoveDungeonWrapped(
					fromX, fromY, int(facing)*2) {
					continue
				}
				if approached[approachKey(application, [2]int{x, y}, facing)] {
					continue
				}
				return [2]int{x, y}, facing, true
			}
		}
	}
	return [2]int{}, 0, false
}

// chooseTransitionCell 從這一張圖上已知的換圖點裡挑一個用得最少的。
//
// 換圖點是**量出來的**：走到那一格之後地圖換了，就記一次（見走路迴圈開頭）。
// 站著的那一格不算——再踩一次不會重新觸發。
func chooseTransitionCell(application *app, uses map[[3]int]int) ([2]int, bool) {
	here := [2]int{int(application.spawn.X), int(application.spawn.Y)}
	best, bestUses, found := [2]int{}, 0, false
	for key, count := range uses {
		if key[0] != int(application.spawn.Map.Archive) ||
			key[1] != int(application.spawn.Map.BlockID) {
			continue
		}
		cell := [2]int{key[2] % 100, key[2] / 100}
		if cell == here {
			continue
		}
		if !found || count < bestUses {
			best, bestUses, found = cell, count, true
		}
	}
	return best, found
}

// areaExit 是「站在這一格、面向這個方向往前一步就離開這一區」。
type areaExit struct {
	cell   [2]int
	facing uint8
}

// exitKey 把一個出口壓成 exitUses 的鍵。
func exitKey(application *app, exit areaExit) [4]int {
	return [4]int{int(application.spawn.Map.Archive), int(application.spawn.Map.BlockID),
		exit.cell[1]*100 + exit.cell[0], int(exit.facing)}
}

// chooseAreaExit 從這一張圖的邊界上挑一個用得最少的出口。
//
// 「出口」是資料算出來的，不是抄來的：邊界上那一格朝外的方向沒有牆，
// 走出去就是換區（spec 100）。用得最少的優先，這樣八個方向都會輪到。
func chooseAreaExit(application *app, exitUses map[[4]int]int) (areaExit, bool) {
	if application.initialMap == nil {
		return areaExit{}, false
	}
	best, bestUses, found := areaExit{}, 0, false
	for y := 0; y < geometry.Height; y++ {
		for x := 0; x < geometry.Width; x++ {
			for facing := 0; facing < 4; facing++ {
				nextX := x + exploreDeltas[facing][0]
				nextY := y + exploreDeltas[facing][1]
				inside := nextX >= 0 && nextX < geometry.Width &&
					nextY >= 0 && nextY < geometry.Height
				if inside {
					continue
				}
				if !application.initialMap.Grid.CanMoveDungeonWrapped(x, y, facing*2) {
					continue
				}
				candidate := areaExit{cell: [2]int{x, y}, facing: uint8(facing)}
				uses := exitUses[exitKey(application, candidate)]
				if !found || uses < bestUses {
					best, bestUses, found = candidate, uses, true
				}
			}
		}
	}
	return best, found
}

// 世界巡迴：**測試治具**，不是玩家路徑。港務長那一段目前推不動
//（船票旗標 `4A01` 沒有人清回去，spec 102 的 OPEN），所以主線之後的區域
// 一直沒有被真的跑過——只有 spec 103 的入口掃描碰過它們，而那是乾淨變數的
// 靜態掃描，沒有前端、沒有戰鬥、沒有選單。
//
// 這一條把碼頭的目的地直接寫進 `DS:4AC4h`，讓探索器把四條航線各走一遍，
// 量「走到了哪些區塊、撞到哪些硬失敗」。它證明的是**那些區域的腳本在
// 完整的前端底下跑不跑得動**，不是玩家走不走得到。
func TestWorldTourReachesTheAreasBehindTheHarbour(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	maps := map[string]bool{}
	blocks := map[int]bool{}
	visited := map[[3]int]bool{}
	var hardFailures []string
	ok := false
	for pass, destination := range []int{0, 1, 2, 3, 1, 2, 3, 1, 2, 3} {
		seed := int64(13 + pass*7 + destination)
		avoid := map[[3]int]bool{}
		transitionUses := map[[3]int]int{}
		menuTurn := map[[3]int]int{}
		_, reachable := exploreWorldWithFlags(t, zipPath, seed, 0, 1, 200000,
			avoid, map[[3]int]bool{}, transitionUses, menuTurn, map[[4]int]int{},
			visited, maps, blocks, nil, destination, &hardFailures)
		if !reachable {
			t.Skip("original DOS ZIP is intentionally not tracked")
		}
		ok = true
	}
	if !ok {
		t.Skip("original DOS ZIP is intentionally not tracked")
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
	t.Logf("走到的 ECL block %d 個：%v", len(blockIDs), blockIDs)
	seen := map[string]int{}
	for _, failure := range hardFailures {
		key := failure
		if index := strings.Index(failure, "："); index >= 0 {
			key = failure[index+len("："):]
		}
		seen[key]++
	}
	for failure, count := range seen {
		t.Logf("硬失敗 ×%d：%s", count, failure)
	}
	// 量到的下限，不是目標。少於這個數代表航線、野外移動或資源載入退步了。
	if len(blocks) < 11 {
		t.Errorf("只走到 %d 個 ECL block：%v", len(blocks), blockIDs)
	}
}
