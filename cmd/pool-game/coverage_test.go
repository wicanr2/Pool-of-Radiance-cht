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

// holdMainlineExits 在主線鎖亮著的時候，把這一張圖上所有會換圖的格子放進
// avoid，並記在 heldBack 裡等鎖熄了再放回去。
//
// 兩種來源都要：`transitionUses` 是踩過才知道的，`boundaryExitKeys` 才擋得住
// 第一次遇到的。
func holdMainlineExits(app *app, avoid, heldBack map[[3]int]bool, transitionUses map[[3]int]int) {
	archive, block := int(app.spawn.Map.Archive), int(app.spawn.Map.BlockID)
	hold := func(key [3]int) {
		if !avoid[key] {
			avoid[key], heldBack[key] = true, true
		}
	}
	for key := range transitionUses {
		if key[0] == archive && key[1] == block {
			hold(key)
		}
	}
	for _, key := range boundaryExitKeys(app) {
		hold(key)
	}
	// 城區還要多擋一層：鎖住時走到的**任何**事件格都可能把 `4A01` 寫回 1
	// （競技場 ECL3/11 `9CACh` 就是，實測 `4A01 255→1 於 GEO3/0 (7,2)`），
	// 前功盡棄。鎖住這段時間的任務只有一件——走到港務長，所以除了他門口的
	// 那兩格，城區的事件格整批不去。
	if archive == cityArchive && block == cityBlock && app.initialMap != nil {
		for y := 0; y < geometry.Height; y++ {
			for x := 0; x < geometry.Width; x++ {
				if x == harbourMasterCell[0] &&
					(y == harbourMasterCell[1] || y == harbourApproachCell[1]) {
					continue
				}
				cell, ok := app.initialMap.Grid.Cell(x, y)
				if !ok || cell.Terrain&0x7F == 0 {
					continue
				}
				hold([3]int{archive, block, y*100 + x})
			}
		}
	}
}

// 城區與港務長的位置。`harbourApproachCell` 是站的那一格，往北一步就是
// 港務長；兩格都不能被鎖住時的「事件格整批不去」擋掉。
const (
	cityArchive = 3
	cityBlock   = 0
)

var (
	harbourMasterCell   = [2]int{11, 1}
	harbourApproachCell = [2]int{11, 2}
	// dockCell 是碼頭那一格；踩上去就會開航線選單。
	dockCell = [2]int{15, 1}
)

// boundaryExitKeys 是這一張 GEO 邊界上「往外沒有牆」的格子。
//
// 換圖不是在按下前進的那一 tick 發生，是那一格的事件用 `LOAD FILES` 做掉的，
// 所以 explorePlan 的「跨出邊界那一步不走」擋不住換圖——隊伍只要**踩到**
// 這種格子就換走了。主線鎖住時得把它們整批從目標清單裡濾掉；靠
// `transitionUses` 收不到，那是踩過才知道的，第一次遇到的一定擋不住。
func boundaryExitKeys(app *app) [][3]int {
	archive, block := int(app.spawn.Map.Archive), int(app.spawn.Map.BlockID)
	var keys [][3]int
	for y := 0; y < geometry.Height; y++ {
		for x := 0; x < geometry.Width; x++ {
			if x != 0 && y != 0 && x != geometry.Width-1 && y != geometry.Height-1 {
				continue
			}
			for facing := 0; facing < 4; facing++ {
				nx, ny := x+exploreDeltas[facing][0], y+exploreDeltas[facing][1]
				if nx >= 0 && nx < geometry.Width && ny >= 0 && ny < geometry.Height {
					continue
				}
				if !app.initialMap.Grid.CanMoveDungeonWrapped(x, y, facing*2) {
					continue
				}
				keys = append(keys, [3]int{archive, block, y*100 + x})
				break
			}
		}
	}
	return keys
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
// planToCellsAvoiding 是 planToCells，但**路上**不踏進 skip 說要避開的格子。
//
// 主線那一段需要它：城區的地點格會觸發自己的腳本，而競技場（ECL3/11
// `9CACh`）踩到就把 `4A01` 寫回 1（spec 102），去港務長的路上經過就前功盡棄。
func planToCellsAvoiding(app *app, rotate int, wanted func(x, y int) bool,
	skip func(x, y int) bool) []exploreStep {
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
			if skip != nil && !wanted(next.x, next.y) && skip(next.x, next.y) {
				continue
			}
			from[next], via[next] = current, uint8(facing)
			queue = append(queue, next)
		}
	}
	return nil
}

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

// eclPasswords 是遊戲裡會被 `10h INPUT STRING` 比對的字，逐一試。
//
// 索寇要塞有三個（ecl4 block 21 的 `9E8Dh SAMOSUD`、`9E9Ah SHESTNI`、
// `A388h`／`AA84h LUX`），野外那一區還有一個（ecl7 block 23 `A4CAh NOKNOK`）。
// 玩家從手札與遊戲裡的對話知道這些字；探索器只認一個的話，
// 「說 LUX 才會出現」的那個亡魂永遠碰不到。
var eclPasswords = []string{"SAMOSUD", "LUX", "SHESTNI", "NOKNOK"}

// knownECLPasswords 是 `answerHints` 解不出來時的退路。鍵是 (ECL archive, block)。
var knownECLPasswords = map[[2]int]string{
	{7, 23}: "NOKNOK",
}

// answerHints 把問句尾巴那個括號拆成候選字。`enterECLInput` 會在問句後面
// 附上原版資料裡的答案，例如「…SAY...?（SAMOSUD／SHESTNI）」。
func answerHints(prompt string) []string {
	open := strings.LastIndex(prompt, "（")
	if open < 0 || !strings.HasSuffix(strings.TrimRight(prompt, "\n "), "）") {
		return nil
	}
	inner := prompt[open+len("（"):]
	inner = strings.TrimRight(strings.TrimRight(inner, "\n "), "）")
	var hints []string
	for _, candidate := range strings.Split(inner, "／") {
		if candidate = strings.TrimSpace(candidate); candidate != "" {
			hints = append(hints, candidate)
		}
	}
	return hints
}

// exploreMaxTransitionHops 是「這一張走完了，回頭走另一個換圖點」最多做幾次。
// 沒有上限的話，所有圖都走完之後兩張圖之間會一直來回。
const exploreMaxTransitionHops = 60

// exploreMaxCombatStall 是「同一個（回合、行動者、提示）連續幾個 tick 還沒
// 動」的上限。一場架正常會一直換行動者，停住就是卡住了。
const exploreMaxCombatStall = 4000

// exploreMaxTargetTries 是同一格被規劃成目標幾次還沒踩到就放棄。
const exploreMaxTargetTries = 12

// exploreMaxWildernessSteps 是一趟在野外最多亂走幾步。野外沒有「這一格踩過
// 了」可用（位置在 `DS:49C3h`／`DS:49C4h`，不是 GEO 格子，spec 105），
// 所以隨機走不會自己停；沒有上限的話一趟就把整包預算花在那裡。
const exploreMaxWildernessSteps = 3000

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
			// 白金給足：船資是一枚白金（spec 090），身上沒有的話港務長那
			// 一段永遠停在「你的白金不夠」，探索器就永遠出不了海。
			MaxHP: 60, CurrentHP: 60, PortraitHead: 1, PortraitBody: 1, IconSize: 1,
			Money: [7]uint16{4: 20}})
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
	// heldBack 是「暫時擋住的換圖格」。清掉船票之後不能離開索寇要塞
	//（見 mustSettleSokalGhost），但規劃器會把換圖格當成一般的沒踩過的格子
	// 走上去，所以那段期間先把它們塞進 avoid，了結之後再拿回來。
	heldBack := map[[3]int]bool{}
	lastMap, lastCell := application.spawn.Map, [2]int{-1, -1}
	// 主線旗標一動，走過的地點就要重走一次：地點腳本是旗標閘門的
	//（港務長 `9C4Bh` 比 `4A01`、`9C5Ch` 比 `4AA7`，spec 102），
	// 同一格在不同旗標下講不同的話。只記「這個方向走進去過」的話，
	// 要塞回來之後港務長就再也不會被問第二次。
	quest := [2]uint16{}
	// 每一種選單只印一次，用來確認某一支腳本到底有沒有被觸發。
	seenMenus := map[string]bool{}
	// 剛送出密碼的格子。原版是「輸入 → 印出你打的字 → [YES NO] 確認」，
	// 答 NO 就跳回去重新輸入（ecl7/23 的 `A4C6 GOTO A3FAh`）。治具若在這裡
	// 輪流答，永遠有一半機會回頭，就走不完這一格。
	confirmInput := map[[3]int]bool{}
	heldHere := false
	// harbourTried 讓「主線鎖住就先去港務長」每個鎖住期間只試一次。
	harbourTried := false
	// dockTried 同理：票拿到手之後主動走一次碼頭，每個鎖住週期一次。
	dockTried := false
walk:
	for step := 0; step < budget; step++ {
		if application.eventMachine != nil {
			now := [2]uint16{application.eventMachine.Memory[0x4AA7],
				application.eventMachine.Memory[0x4A01]}
			if now != quest {
				if now[1] != quest[1] {
					block := -1
					if application.eventSession != nil {
						block = int(application.eventSession.CurrentBlockID())
					}
					t.Logf("4A01 %d→%d 於 GEO%d/%d (%d,%d) ECL block %d",
						quest[1], now[1], application.spawn.Map.Archive,
						application.spawn.Map.BlockID, application.spawn.X,
						application.spawn.Y, block)
				}
				quest = now
				for key := range approached {
					delete(approached, key)
				}
			}
		}
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
		// 主線鎖每一步都看，不只在「手上沒有計畫」的時候。實測隊伍就是帶著
		// **鎖亮之前規劃好的路**走出城區的——鎖是在城區某一格亮起來的，那時
		// 計畫早就排好了，等它走完往往已經換圖。
		//
		// 只在**由暗轉亮**那一刻丟一次計畫。每一步都丟試過，更差：隊伍會在
		// 城區與貧民窟之間來回，走到的 block 從 5 掉到 4。
		if application.initialMap != nil {
			settling := mustSettleSokalGhost(application)
			if settling {
				holdMainlineExits(application, avoid, heldBack, transitionUses)
			} else {
				harbourTried = false
			}
			if settling != heldHere {
				heldHere = settling
				t.Logf("主線鎖 %v GEO%d/%d（4AA7=%d 4A01=%d 4A26=%d）", settling,
					application.spawn.Map.Archive, application.spawn.Map.BlockID,
					application.eventMachine.Memory[0x4AA7],
					application.eventMachine.Memory[0x4A01],
					application.eventMachine.Memory[0x4A26])
				if settling {
					plan, exit, harbourTried, dockTried = nil, nil, false, false
				} else {
					for key := range heldBack {
						delete(avoid, key)
						delete(heldBack, key)
					}
				}
			}
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
						"名冊 %d，施法選單 %v/%v，狀態 %q，是我方 %v，狀態列 %q）",
					application.spawn.Map.Archive, application.spawn.Map.BlockID,
					application.tactical.Round, application.tactical.Mover,
					application.tactical.Prompt, counts.Party, counts.Foes,
					len(application.tactical.Roster), application.castOpen,
					application.castTargeting, application.tactical.Status,
					application.tactical.Friendly[application.tactical.Mover],
					application.statusLine))
				for index := 1; index < len(application.tactical.Roster); index++ {
					cell := application.tactical.Roster[index]
					t.Logf("    名冊 %d：(%d,%d) 體型 %d 我方 %v 分數 %d 額度 %d",
						index, cell.X, cell.Y, cell.FootprintClass,
						application.tactical.Friendly[index],
						application.tactical.Scores[index],
						application.tactical.Budgets[index])
				}
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
				block := -1
				if application.eventSession != nil {
					block = int(application.eventSession.CurrentBlockID())
				}
				failures = append(failures, fmt.Sprintf(
					"格子選單卡住：GEO%d/%d (%d,%d) 游標 %d／%v 標籤 %q 文字 %q 狀態列 %q "+
						"block %d 這一格答過 %d 次",
					application.spawn.Map.Archive, application.spawn.Map.BlockID,
					application.spawn.X, application.spawn.Y,
					application.cellMenuCursor, application.cellMenuOptions,
					application.eventLabel, application.eventText,
					application.statusLine, block,
					menuTurn[[3]int{int(application.spawn.Map.Archive),
						int(application.spawn.Map.BlockID),
						int(application.spawn.Y)*100 + int(application.spawn.X)}]))
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
			application.campOpen || application.parlay != nil ||
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
		if application.campOpen {
			// 旅店的過夜（`38h PROGRAM` 值 9）開的是紮營畫面，ESC 收掉。
			plan = nil
			if err := press(application, ebiten.KeyEscape); err != nil {
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
				key := [3]int{int(application.spawn.Map.Archive),
					int(application.spawn.Map.BlockID),
					int(application.spawn.Y)*100 + int(application.spawn.X)}
				word := eclPasswords[menuTurn[key]%len(eclPasswords)]
				// 答案就寫在問句的括號裡（`eclInputAnswer` 從原版資料解出來，
				// 玩家看得到），治具照著打——這也是玩家實際會做的事，比自己
				// 猜準。多個候選（同一個變數被 `SAVE` 兩次）就輪流試。
				//
				// 這一步很重要：ecl7/23 的密碼門答錯一次會扣血、
				// `A564 OR 4A51h #64` 設旗標然後 EXIT，入口 `A3E4` 檢查同一個
				// bit 就 EXIT——**那一格只有一次機會**，輪流猜等於把它用掉。
				switch hints := answerHints(application.eventText); {
				case len(hints) != 0:
					word = hints[menuTurn[key]%len(hints)]
				case flags != nil && application.spawn.Map.BlockID == 21:
					// 提示解不出來時的退路。索寇要塞登陸那一格的亡魂只問一次
					// （問完 `SAVE 255 @4A13`），答錯就再也不出現。
					word = "LUX"
				default:
					if known, ok := knownECLPasswords[[2]int{
						int(application.spawn.Map.Archive),
						int(application.spawn.Map.BlockID)}]; ok {
						word = known
					}
				}
				menuTurn[key]++
				confirmInput[key] = true
				t.Logf("密碼輸入 GEO%d/%d (%d,%d)：問句 %q，送出 %q",
					application.spawn.Map.Archive, application.spawn.Map.BlockID,
					application.spawn.X, application.spawn.Y, application.eventText, word)
				application.keys = scriptedChars(word)
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
				if signature := strings.Join(application.cellMenuOptions, "|"); !seenMenus[signature] {
					seenMenus[signature] = true
					// 一併印文字框：問句是選單自己的 Prompt 還是前面幾條
					// `11h PRINT` 拼出來的，看得到才分得出來。
					t.Logf("選單 GEO%d/%d (%d,%d) 朝向 %d：%v／文字 %q",
						application.spawn.Map.Archive, application.spawn.Map.BlockID,
						application.spawn.X, application.spawn.Y,
						application.spawn.Facing, application.cellMenuOptions,
						application.eventText)
				}
				want := menuTurn[key] % len(application.cellMenuOptions)
				// 剛打完密碼的那一次是確認框，答 NO 只會跳回去重打。
				if confirmInput[key] {
					want = 0
				}
				// flags 非 nil 那一條要推主線，所以 YES／NO 一律答 YES
				//（「要不要拿走裝備」答 NO 就推不動要塞那一段）。
				if flags != nil && strings.EqualFold(application.cellMenuOptions[0], "YES") {
					want = 0
				}
				// 同一條路上要走到亡魂那一段：登陸的遭遇要選「交涉」，
				// 費蘭問話則**看船票在不在手上**：
				//
				//   - `4A01 == 1`（手上有票，港務長不再開口）→ 選「說謊」，
				//     `ABBDh` 會把 `4A01` 寫成 255，票就清掉了。
				//   - 否則 → 選「說實話」，費蘭才會給 SAMOSUD 並把
				//     `4AA7` 寫成 254（`ADAAh`），碼頭的其他航線才開。
				//
				// 說謊那一支**不寫 `4A26`**，所以亡魂還會再出現；說實話那一支
				// 才寫（`ADA4h`），寫完就不再出現。兩件事因此都做得到。
				if flags != nil {
					// 港務長的完整航線選單：SOKAL 是回索寇要塞（已經走過），
					// NONE 是不上船，所以在 EAST／WEST／BAY 之間輪流挑。
					if len(application.cellMenuOptions) == 5 &&
						strings.EqualFold(application.cellMenuOptions[0], "SOKAL") {
						want = 1 + menuTurn[key]%3
					}
					for index, option := range application.cellMenuOptions {
						if strings.EqualFold(option, "Parlay") {
							want = index
						}
						lie := strings.EqualFold(option, "LIE?")
						truth := strings.EqualFold(option, "TELL THE TRUTH?")
						if !lie && !truth {
							continue
						}
						holdsTicket := application.eventMachine != nil &&
							application.eventMachine.Memory[0x4A01] == 1
						if lie == holdsTicket {
							want = index
							t.Logf("費蘭選單 GEO%d/%d (%d,%d) 選 %s"+
								"（4A01=%d 4A26=%d 4AA7=%d）",
								application.spawn.Map.Archive,
								application.spawn.Map.BlockID,
								application.spawn.X, application.spawn.Y, option,
								application.eventMachine.Memory[0x4A01],
								application.eventMachine.Memory[0x4A26],
								application.eventMachine.Memory[0x4AA7])
						}
					}
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
				delete(confirmInput, key)
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
			//
			// 要有上限：野外沒有「這一格踩過了」可用，隨機亂走不會自己停，
			// 一趟就會把整包預算吃在那裡（實測連帶把整個 package 的
			// 10 分鐘 timeout 用完）。
			plan, exit = nil, nil
			spin["野外"]++
			if spin["野外"] > exploreMaxWildernessSteps {
				reason = "野外走到上限"
				break
			}
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
			settling := heldHere
			// 鎖住的時候，城區這一趟優先去問港務長。城區有好幾個地點會把
			// `4A01` 寫回 1（市政廳職員 ECL3/8 `9BACh` 只在 0 時寫，
			// 競技場 ECL3/11 `9CACh` 進到那一支就寫），先踩到就前功盡棄。
			//
			// 鎖的評估已經提到每一步（見迴圈上方），所以回到城區時手上的
			// 舊計畫會在鎖亮的那一刻被丟掉，不會照著走過去踩競技場。
			if settling && int(application.spawn.Map.Archive) == cityArchive &&
				int(application.spawn.Map.BlockID) == cityBlock && !harbourTried {
				harbourTried = true
				if int(application.spawn.X) == harbourApproachCell[0] &&
					int(application.spawn.Y) == harbourApproachCell[1] {
					plan = []exploreStep{{facing: 0}}
				} else {
					plan = planToCellsAvoiding(application, rotate,
						func(x, y int) bool {
							return x == harbourApproachCell[0] && y == harbourApproachCell[1]
						},
						func(x, y int) bool {
							cell, ok := application.initialMap.Grid.Cell(x, y)
							return ok && cell.Terrain&0x7F != 0
						})
					if len(plan) != 0 {
						plan = append(plan, exploreStep{facing: 0})
					}
				}
				if len(plan) != 0 {
					spin["找港務長"]++
					continue
				}
			}
			// 票拿到手之後主動走去碼頭。港務長那一段做完時 `4A01 == 1`
			// 而且 `4AA7 == 254`，碼頭 (15,1) 的選單才會從「唯一的船是
			// 索寇要塞」變成五個目的地（spec 099）。
			//
			// 探索器自己走不過去：碼頭是換圖點，第一次用完就進了 avoid，
			// 規劃器再也不挑它。路上一樣繞開別的事件格，免得又踩到競技場。
			if !settling && int(application.spawn.Map.Archive) == cityArchive &&
				int(application.spawn.Map.BlockID) == cityBlock && !dockTried &&
				application.eventMachine != nil &&
				application.eventMachine.Memory[0x4A01] == 1 &&
				application.eventMachine.Memory[0x4AA7] == 254 {
				dockTried = true
				plan = planToCellsAvoiding(application, rotate,
					func(x, y int) bool { return x == dockCell[0] && y == dockCell[1] },
					func(x, y int) bool {
						if x == dockCell[0] && y == dockCell[1] {
							return false
						}
						cell, ok := application.initialMap.Grid.Cell(x, y)
						return ok && cell.Terrain&0x7F != 0
					})
				if len(plan) != 0 {
					spin["走去碼頭"]++
					continue
				}
			}
			leaving := hops < exploreMaxTransitionHops && !settling
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
				if cell, facing, ok := chooseApproach(application, approached, avoid, rotate); ok {
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
			if len(plan) == 0 && leaving {
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
			if len(plan) == 0 && leaving {
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
		for _, failure := range failures {
			// 印 seed 才回得去：彙總那一層會把訊息去重，重現得靠這一行認出
			// 是哪一趟。少了它只知道「某一趟卡住」，得把 22 趟重跑一遍才找得到。
			t.Logf("  硬失敗（seed %d）：%s", seed, failure)
		}
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
	// 主線這一段要同時滿足兩件事才算推得動：`4AA7 = 254`（要塞清掉、
	// 航線開了）與 `4A01 != 1`（船票狀態被清掉，港務長才會再開口，
	// spec 102）。兩件事走的是要塞裡的不同路線，所以多試幾個種子。
	best := map[uint16]uint16{}
	for _, seed := range []int64{7, 11, 3, 29, 41, 53, 67, 71, 83, 97} {
		// 每一個種子都從乾淨的狀態開始：avoid 與換圖點的使用次數留著的話，
		// 第二輪一開始就被擋在碼頭外面。
		avoid = map[[3]int]bool{}
		transitionUses = map[[3]int]int{}
		// 重走上限拉高：要塞那一段有先後順序——先在裝備架拿到裝備、打贏
		// 那一場（`4A21 = 255`），鬼魂才肯說出 SAMOSUD（`4AA7 = 254`）。
		// 只走一遍的話，鬼魂那一格多半在拿到裝備之前就踩過了。
		_, reachable := exploreWorldWithFlags(t, zipPath, seed, 0, 8, 600000,
			avoid, map[[3]int]bool{}, transitionUses, menuTurn, map[[4]int]int{},
			visited, maps, blocks, flags, noBoatOverride, &hardFailures)
		if !reachable {
			t.Skip("original DOS ZIP is intentionally not tracked")
		}
		ok = true
		for address, value := range flags {
			if value != 0 {
				best[address] = value
			}
		}
		// 這一則要斷言的東西**全部**湊齊了才提早收工。只看旗標就 break 的話，
		// 「走得到幾張圖」會變成「第幾個種子先湊齊旗標」的函數——2026-09-05
		// 修掉 startRound 的一處亂數流之後，第一個種子就湊齊了旗標，於是
		// union 只剩那一趟的五張圖，看起來像覆蓋率退步，其實是提早收工。
		if best[0x4AA7] == 254 && best[0x4A01] != 1 &&
			len(maps) >= 7 && len(blocks) >= 7 &&
			(blocks[25] || blocks[26] || blocks[27]) {
			break
		}
	}
	if !ok {
		t.Skip("original DOS ZIP is intentionally not tracked")
	}
	for _, failure := range hardFailures {
		t.Logf("硬失敗：%s", failure)
	}
	blockList := make([]int, 0, len(blocks))
	for block := range blocks {
		blockList = append(blockList, block)
	}
	sort.Ints(blockList)
	t.Logf("走到的地圖：%d 張 %v；ECL block：%d 個 %v",
		len(maps), sortedMapNames(maps), len(blocks), blockList)
	t.Logf("旗標 4A21=%d（要塞的裝備與那一場架）4AA7=%d（碼頭航線）4AC4=%d 6E12=%d "+
		"4A01=%d 4AC5=%d 4ABA=%d",
		flags[0x4A21], flags[0x4AA7], flags[0x4AC4], flags[0x6E12],
		flags[0x4A01], flags[0x4AC5], flags[0x4ABA])
	t.Logf("跨種子推到的最好狀態：4AA7=%d 4A01=%d 4A21=%d 4AC4=%d",
		best[0x4AA7], best[0x4A01], best[0x4A21], best[0x4AC4])
	// **只釘住走得到的部分**：碼頭的船會把隊伍送到索寇要塞（ECL block 21）。
	// 那一段的旗標（拿裝備 `4A21h`、開航線 `4AA7h`）**推不推得到跟路線有關**
	// ——探索器是機器人，走到哪一格、答哪一個選項會隨著別處的修正而改變，
	// 釘住它只會在無關的改動上變紅。旗標印出來當觀察值，要推主線得靠有目的地
	// 的路線（WORKLIST 有這一條）。
	if !blocks[21] {
		t.Errorf("沒走到索寇要塞（ECL block 21），走到的是 %v", blocks)
	}
	// 走得到野外了（2026-09-04）：票拿到手之後主動走一次碼頭
	//（`走去碼頭` 那一支），航線選單就會把隊伍送出去。這兩個數字是**量到的
	// 下限**，不是目標——少於它代表主線那一段或碼頭那一步退步了。
	if len(maps) < 7 {
		t.Errorf("只走到 %d 張地圖，先前量到 7 張（%v）", len(maps), sortedMapNames(maps))
	}
	if len(blocks) < 7 {
		t.Errorf("只走到 %d 個 ECL block，先前量到 7 個（%v）", len(blocks), blocks)
	}
	// 野外是這一項的目的地：三張野外／樞紐圖（ECL block 25／26／27，spec 105）
	// 至少要碰到一張，才算「探索器走得到野外」。
	if !blocks[25] && !blocks[26] && !blocks[27] {
		t.Errorf("沒走到野外（ECL block 25／26／27），走到的是 %v", blocks)
	}
}

// sortedMapNames 把走過的地圖名排序後列出來，好讀。
func sortedMapNames(maps map[string]bool) []string {
	names := make([]string, 0, len(maps))
	for name := range maps {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// mustSettleSokalGhost 回答「現在還不能離開這一張圖嗎」。
//
// 費蘭那張選單的兩支各開一半的鎖（spec 102）：說謊寫 `4A01 = 255`（清掉
// 只能去索寇要塞的那張船票），說實話（或打贏那一場）寫 `4AA7 = 254`
// （開其他航線）並寫 `4A26 = 255`（亡魂不再出現）。兩件事要都成立，
// 港務長才會再開口並給出完整的目的地選單。
//
// 中間**不能回城區**：`4AA7` 還沒開到 254 時，港務長走的是「唯一的船是去
// 索寇要塞」那一支，`9DCEh` 會把 `4A01` 寫回 1，說謊那一步就白做了。
// 所以票已經清掉、亡魂還沒了結的這段期間，探索器不挑換圖點也不挑出口。
func mustSettleSokalGhost(application *app) bool {
	if application.eventMachine == nil {
		return false
	}
	memory := application.eventMachine.Memory
	// 兩種狀態都算「主線這一段還沒了結」，而且**與隊伍站在哪一張圖無關**：
	//
	//  1. 票清掉了但亡魂還沒了結（要塞那一段還沒做完）。
	//  2. 航線開了、票卻不在手上——這一趟就是要去找港務長。碼頭 (15,1) 就在
	//     港務長 (11,1) 旁邊，先上船的話 `4AC4 = 0` 又被載回索寇要塞，
	//     回來時入口 4 再把 `4AC4` 清成 0，繞不出去。
	//
	// 先前這支**依當前地圖分派**，於是同一組記憶體值在城區回 true、走進
	// 貧民窟就回 false。鎖一熄 `heldBack` 整批放回去，回城區又亮——實測
	// 「找港務長」一趟觸發 112 次，走到的 ECL block 反而從 5 掉到 4。
	return (memory[0x4A01] == 255 && memory[0x4A26] != 255) ||
		(memory[0x4AA7] == 254 && memory[0x4A01] != 1)
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
func chooseApproach(application *app, approached map[[4]int]bool,
	avoid map[[3]int]bool, rotate int) ([2]int, uint8, bool) {
	if application.initialMap == nil {
		return [2]int{}, 0, false
	}
	for y := 0; y < geometry.Height; y++ {
		for x := 0; x < geometry.Width; x++ {
			cell, ok := application.initialMap.Grid.Cell(x, y)
			if !ok || cell.Terrain&0x7F == 0 {
				continue
			}
			// 被擋住的格子也不當走進去的目標：主線那幾段要把換圖格
			// 暫時關掉（見 mustSettleSokalGhost），而換圖格自己多半
			// 也是地點格——碼頭 (15,1) 就是。
			if avoid[[3]int{int(application.spawn.Map.Archive),
				int(application.spawn.Map.BlockID), y*100 + x}] {
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
	// 趟數是**覆蓋面**，不是耐心：野外那幾段每一趟只走 36 步，落點由亂數決定，
	// 而任何會改變抽籤次數的修改（例如攻擊改成一次行動揮好幾下）都會讓同一個
	// 種子走去別的地方。趟數少的時候「走到幾個區塊」就變成單一亂數對齊的快照，
	// 一改就紅。多跑幾組種子才量得到真正的下限。
	for pass, destination := range []int{0, 1, 2, 3, 1, 2, 3, 1, 2, 3,
		1, 2, 3, 1, 2, 3, 1, 2, 3, 1, 2, 3,
		1, 2, 3, 1, 2, 3, 1, 2, 3, 1, 2, 3} {
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
	// **門檻不跟著實測值走**：現在量得到 12 個，門檻留在 11——把門檻頂到實測值
	// 等於再做一次單一亂數對齊的快照，下一個會改變抽籤次數的修改又會紅。
	if len(blocks) < 11 {
		t.Errorf("只走到 %d 個 ECL block：%v", len(blocks), blockIDs)
	}
}
