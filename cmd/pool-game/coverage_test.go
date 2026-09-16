package main

// 本檔的跨 archive 世界巡遊釘住 spec 107 的 FF sentinel 交接。

import (
	"fmt"
	"math/rand"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
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
		application.eclSeed = 1
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

// explorerCanTraverse 是測試導覽器的規劃邊。原始 GEO 把鎖門與閒門
// 標成不可直接通行，但玩家走向它會先進入 BASH／PICK／KNOCK 選單。
// 規劃器只將這兩類「可嘗試的門」納入路徑；門有沒有真正打開仍由
// Update() 的正常輸入分派與角色能力決定，這裡不改地圖狀態。
func explorerCanTraverse(app *app, x, y, facing int) bool {
	if app.initialMap.Grid.CanMoveDungeonWrapped(x, y, facing*2) {
		return true
	}
	// 目前只有 Stojanow 這組已完成任務卻回不去的地圖有直接證據：
	// GEO7/22 與 GEO7/23 各被靜態門邊切成 5／22 個連通元件，而腳本要求
	// 23→22→26 才能返回樞紐。全圖開放這個假設會讓貧民窟探索器誤走
	// 西側邊界，所以在其他地圖仍只規劃原始 GEO 當下可通行的邊。
	// 貧民窟（GEO2/20）也算：清貧民窟要的 25 場裡，14 個固定事件多半在
	// 鎖著的屋子裡（`ecl2/20` 入口 1 依地形碼 1..20 分派），隨機遭遇又在
	// `4A80 ≥ 15` 之後停發——只走開著的路只踩得到 66 格、16 場。
	slums := app.spawn.Map == (gamepack.MapKey{Archive: 2, BlockID: 20})
	if !slums && (app.spawn.Map.Archive != 7 ||
		(app.spawn.Map.BlockID != 22 && app.spawn.Map.BlockID != 23)) {
		return false
	}
	flags, door := app.initialMap.Grid.WallDoorFlagsWrapped(x, y, facing*2)
	return door && (flags == gamepack.DoorLocked || flags == gamepack.DoorBarred)
}

// explorePlan 從目前這一格廣度優先找到最近的一格「還沒踩過的」，
// 回傳走過去要的朝向序列。走不到就回 nil。
//
// 用的是原始 GEO 的 CanMoveDungeonWrapped，跟遊戲自己判斷能不能走同一支，
// 所以這條路徑不會宣告出資料裡沒有的通路。
// explorePlan 回傳走去最近一格「還沒踩過」的路，以及那一格的鍵。
// 目標要回傳出去：踩不到的格子得記次數，不然規劃器會一直挑同一格，
// 隊伍在半路來回走而 stuck 永遠不增加——實測一趟走六萬步只踩到 479 格。
// banPass 為真時 `avoid` 的格子**連路過都不行**（換區的格子踩到就換走）。
// 主線鎖亮著的時候要關掉：那段期間 `avoid` 裡塞的是暫時擋起來的換圖點
// （見 holdMainlineExits），連路過都禁的話索寇要塞那一張就走不動了
// ——實測只踩到 19 格就宣告走不動，走到的地圖從 7 張掉到 2 張。
func explorePlan(app *app, visited, avoid map[[3]int]bool, banPass bool, rotate int) ([]exploreStep, [3]int) {
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
			if !explorerCanTraverse(app, current.x, current.y, facing) {
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
			// avoid 的格子**連路過都不行**。先前只把它們排除在目標之外，
			// 但邊界換區的格子是「踩到就換走」（spec 100 的 `LOAD FILES`
			// 由格子事件做掉），所以路過等於換走——實測 GEO1/18 五次離開
			// 全部是這樣發生的，`chooseAreaExit` 一次都沒輪到。
			if banPass && avoid[[3]int{int(app.spawn.Map.Archive),
				int(app.spawn.Map.BlockID), y*100 + x}] {
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
	// civilisedPhlanBoatCell 是區域樞紐裡的回程船。連續主線實跑已從
	// GEO5/5 (8,0) 經 TAKE BOAT 正常回到 GEO3/0；它是玩家要交差時的
	// 明確目的地，不是直接改寫 spawn 的傳送接縫。
	civilisedPhlanBoatCell = [2]int{8, 0}
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

// pendingCommission 說同一個 campaign 是否有已完成、尚未回市政廳交差的槽。
// 只讀原版腳本自己寫出的 FEh；探索器不製造或改寫委任旗標。
func pendingCommission(application *app) bool {
	if application.eventMachine == nil {
		return false
	}
	for address := uint16(0x4AA6); address <= 0x4ABF; address++ {
		if application.eventMachine.Memory[address] == uint16(gamepack.CityHallSlotPending) {
			return true
		}
	}
	return false
}

// cityHallPlan 在有待交差委任時，從城區走進市政廳，再走到職員 (5,5)。
// 路線全部由目前 GEO 的牆資料規劃，最後一步也經 Update()，不改 spawn 或 ECL PC。
func cityHallPlan(application *app, rotate int) []exploreStep {
	if !pendingCommission(application) {
		return nil
	}
	// `4A01 > 0` 會從 `9BA8h` 跳過 reward scan 到 `A7FDh`；區域腳本尚未
	// 把這個共用工作格清回 0 時，不能只因委任槽是 FE 就搶先回 clerk。
	// 重入時還要先踩職員外側 (4,5)，把 `4A06` 清零，再往東踏入職員格。
	insideClerk := application.eclArchive == 3 && application.eventSession != nil &&
		application.eventSession.CurrentBlockID() == 8
	if application.eventMachine == nil {
		return nil
	}
	// `4A01 == FF` 是已清除的索寇船票。費蘭接受實話後
	//（`4AA7 >= FE`、`4A26 == FF`），正常下一步是回到職員處交差。
	// 把所有非零值都視為有效船票會把探索器送回港務長、再次購買索寇船票；
	// 因此只在這個亡魂已結案的狀態允許進市政廳，真正有效的船票
	//（`4A01 == 1`）仍然擋住。
	clearedSokalTicket := application.eventMachine.Memory[0x4A01] == 0xFF &&
		application.eventMachine.Memory[0x4AA7] >= uint16(gamepack.CityHallSlotPending) &&
		application.eventMachine.Memory[0x4A26] == 0xFF
	if application.eventMachine.Memory[0x4A01] != 0 && !clearedSokalTicket && !insideClerk {
		return nil
	}
	// City Hall 共用 GEO3/0 的地圖；進門後只有 ECL block 換成 8，spawn.Map
	// 仍是 block 0。只看地圖鍵會把已進門的隊伍又帶回入口，形成 (3,4)／
	// (4,4) 往返。
	if insideClerk {
		if application.eventMachine.Memory[0x4A06] != 0 {
			return planToCells(application, 0, func(x, y int) bool { return x == 4 && y == 5 })
		}
		if int(application.spawn.X) == 4 && int(application.spawn.Y) == 5 {
			return []exploreStep{{facing: 1}}
		}
		// 踏到職員格之後只需把文字／獎賞事件翻完；再送方向鍵會離開
		// 職員，形成 (5,5)／(5,6) 往返。
		if int(application.spawn.X) == 5 && int(application.spawn.Y) == 5 {
			return nil
		}
		// 與職員重入的真實腳本相同：由 (4,5) 往東踏進 (5,5)。
		return planToCells(application, 0, func(x, y int) bool { return x == 5 && y == 5 })
	}
	switch application.spawn.Map {
	case (gamepack.MapKey{Archive: 2, BlockID: 20}):
		// 野外回程船會把隊伍送到貧民窟（ECL block 20），不是直接送進
		// City Hall。沿 GEO 的牆走到東側可通行邊界，再由正常前進鍵回城。
		isEastExit := func(x, y int) bool {
			return x == geometry.Width-1 &&
				application.initialMap.Grid.CanMoveDungeonWrapped(x, y, 2)
		}
		plan := planToCells(application, rotate, isEastExit)
		if isEastExit(int(application.spawn.X), int(application.spawn.Y)) {
			plan = nil
		}
		return append(plan, exploreStep{facing: 1})
	case (gamepack.MapKey{Archive: 3, BlockID: 0}):
		plan := planToCells(application, rotate, func(x, y int) bool { return x == 3 && y == 4 })
		if int(application.spawn.X) == 3 && int(application.spawn.Y) == 4 {
			plan = nil
		}
		return append(plan, exploreStep{facing: 1})
	}
	return nil
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

// planToCells 找到最近的一格目標並回傳走過去的朝向序列，走不到就回 nil。
// wanted 收的是「這一格是不是要去的」。
// planToCellsAvoiding 是 planToCells，但**路上**不踏進 skip 說要避開的格子。
//
// 主線那一段需要它：城區的地點格會觸發自己的腳本，而競技場（ECL3/11
// `9CACh`）踩到就把 `4A01` 寫回 1（spec 102），去港務長的路上經過就前功盡棄。
func planToCellsAvoiding(app *app, rotate int, wanted func(x, y int) bool,
	skip func(x, y int) bool) []exploreStep {
	return planToCellsAvoidingWrap(app, rotate, wanted, skip, true)
}

// planToCellsWithoutWrapping 用在「先走到邊界、再刻意跨出去」的出口計畫。
// GEO 的座標資料雖以 16×16 環狀取格，玩家從邊界往外走會先換區；規劃器若
// 把另一側當成一步可達，送出的那一步其實會提早離開目前地圖。
func planToCellsWithoutWrapping(app *app, rotate int, wanted func(x, y int) bool,
	skip func(x, y int) bool) []exploreStep {
	return planToCellsAvoidingWrap(app, rotate, wanted, skip, false)
}

func planToCellsAvoidingWrap(app *app, rotate int, wanted func(x, y int) bool,
	skip func(x, y int) bool, wrap bool) []exploreStep {
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
			if !explorerCanTraverse(app, current.x, current.y, facing) {
				continue
			}
			nextX := current.x + exploreDeltas[facing][0]
			nextY := current.y + exploreDeltas[facing][1]
			if !wrap && (nextX < 0 || nextX >= geometry.Width || nextY < 0 || nextY >= geometry.Height) {
				continue
			}
			next := node{x: geometry.WrapCoordinate(nextX, geometry.Width),
				y: geometry.WrapCoordinate(nextY, geometry.Height)}
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
			if !explorerCanTraverse(app, current.x, current.y, facing) {
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

// exploreMaxDoorTurns 是**連續**按門選單的上限；走到路上那一步就歸零。
// 一道門最多五下就關掉（游標移到 EXIT 再 ENTER，選項最多四項），所以連續
// 四十下還關不掉一定是壞了。
//
// 計連續不計總數：一趟兩千步可能撞上幾十道門，用總數當門檻會在正常的一趟
// 裡誤判，而那種誤判看起來會像「門關不掉」——剛好蓋掉真正要抓的東西。
const exploreMaxDoorTurns = 40

// exploreMaxTargetTries 是同一格被規劃成目標幾次還沒踩到就放棄。
const exploreMaxTargetTries = 12

// exploreMaxWildernessSteps 是一趟在野外最多亂走幾步。野外沒有「這一格踩過
// 了」可用（位置在 `DS:49C3h`／`DS:49C4h`，不是 GEO 格子，spec 105），
// 所以隨機走不會自己停；沒有上限的話一趟就把整包預算花在那裡。
const exploreMaxWildernessSteps = 8000

// explorerDoorKey 是同一趟探索裡一道門的穩定身分。選過的動作要跟著門，
// 不能只跟著選單游標；BASH 失敗時選項可能仍留在原位。
func explorerDoorKey(a *app) [5]int {
	return [5]int{int(a.spawn.Map.Archive), int(a.spawn.Map.BlockID),
		a.door.X, a.door.Y, a.door.Direction}
}

// explorerDoorChoice 依原版選單的玩家動作順序挑尚未試過的一項。成功開門時
// 選單會自己消失；全部可用方法都失敗後才選 EXIT。
func explorerDoorChoice(options []string, tried map[string]bool) string {
	for _, candidate := range []string{doorOptionBash, doorOptionPick, doorOptionKnock} {
		if tried[candidate] {
			continue
		}
		for _, option := range options {
			if option == candidate {
				return candidate
			}
		}
	}
	return doorOptionExit
}

func TestExplorerTriesDoorActionsBeforeExit(t *testing.T) {
	options := []string{doorOptionBash, doorOptionPick, doorOptionKnock, doorOptionExit}
	tried := map[string]bool{}
	for _, want := range options {
		if got := explorerDoorChoice(options, tried); got != want {
			t.Fatalf("已試 %v 時選到 %q，預期 %q", tried, got, want)
		}
		tried[want] = true
	}
}

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
	avoid, walked map[[3]int]bool, transitionUses map[[3]int]int, menuTurn map[string]int,
	exitUses map[[4]int]int, visited map[[3]int]bool, maps map[string]bool,
	blocks map[int]bool, hardFailures *[]string) (int, bool) {
	return exploreWorldWithFlags(t, zipPath, seed, rotate, rewalkLimit, budget,
		avoid, walked, transitionUses, menuTurn, exitUses, visited, maps, blocks,
		nil, noBoatOverride, hardFailures, nil, nil, nil, nil, false)
}

// noBoatOverride 關掉航線覆寫（見 exploreWorldWithFlags 的 boat 參數）。
const noBoatOverride = -1

// explorerCellKey 是「哪一張圖的哪一格」。
func explorerCellKey(a *app) string {
	return fmt.Sprintf("%d/%d/%d,%d", a.spawn.Map.Archive, a.spawn.Map.BlockID,
		a.spawn.X, a.spawn.Y)
}

// explorerMenuKey 是「哪一格的哪一組選項」。
//
// 同一格會冒出好幾種選單（要不要離開、去哪個地點、YES／NO 確認），
// 共用一個計數就會互相把指標推走。分開之後每一組選項都輪得完。
func explorerMenuKey(a *app, options []string) string {
	return explorerCellKey(a) + "|" + strings.Join(options, "|")
}

// exploreWorldWithFlags 與 exploreWorld 相同，另外在結束時把幾個 ECL 變數
// 抄進 flags，讓呼叫端可以斷言主線推到哪裡。
func exploreWorldWithFlags(t *testing.T, zipPath string, seed int64, rotate, rewalkLimit, budget int,
	avoid, walked map[[3]int]bool, transitionUses map[[3]int]int, menuTurn map[string]int,
	exitUses map[[4]int]int, visited map[[3]int]bool, maps map[string]bool,
	blocks map[int]bool, flags map[uint16]uint16, boat int,
	hardFailures *[]string, overrides map[uint16]uint16, existing *app,
	holdMap *gamepack.MapKey, stop func(*app) bool, deferHandIn bool) (int, bool) {
	t.Helper()
	application := existing
	if application == nil {
		var err error
		application, err = newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
		if err != nil {
			return 0, false
		}
		application.roller = diceRoller{random: rand.New(rand.NewSource(seed))}
		application.eclSeed = 1
		party := make([]poolsave.Character, 0, 6)
		for index := 0; index < 6; index++ {
			member := poolsave.Character{Name: string(rune('A' + index)),
				RaceID: "dwarf", GenderID: "male", ClassID: "fighter",
				AlignmentID: "lawful-good", Abilities: [6]int{18, 10, 10, 16, 10, 10},
				// 白金給足：船資是一枚白金（spec 090），身上沒有的話港務長那
				// 一段永遠停在「你的白金不夠」，探索器就永遠出不了海。
				MaxHP: 60, CurrentHP: 60, PortraitHead: 1, PortraitBody: 1, IconSize: 1,
				Money: [7]uint16{4: 20}}
			if index == 0 {
				member.ClassLevels = make([]uint8, gamepack.ClassThac0ClassCount)
				member.ClassLevels[gamepack.ClassSlotFighter] = 1
				member.ClassLevels[gamepack.ThiefClassSlotIndex] = 1
				member.ThiefSkills = make([]uint8, gamepack.ThiefSkillCount)
			}
			if index == 1 {
				member.Memorised = []uint8{gamepack.KnockSpellID}
			}
			party = append(party, member)
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
	// doorTurns 是**連續**按了幾下門選單，走到路上那一步就歸零。
	doorTurns := 0
	// 每一道門各自記下試過的 BASH／PICK／KNOCK；失敗仍留在選單上的
	// BASH 不可每一 tick 重試，否則後兩種方法永遠輪不到。
	doorAttempts := map[[5]int]map[string]bool{}
	// 沒有選單的格子事件也要有看門狗。只有 menuStall 的時候，一格只要停在
	// 「pending 但按 Enter 什麼都不變」，整趟的預算就靜靜地被吃掉，而報表
	// 只寫「走完預算」——看不出是哪一格，也分不出「走得慢」與「卡住」。
	eventStallKey, eventStall := [4]int{-1, -1, -1, -1}, 0
	tries := map[[3]int]int{}
	// approached 記「這一格從這個方向走進去過了」。見 spec 102。
	approached := map[[4]int]bool{}
	var exit *areaExit
	exitContext := [2]int{-1, -1}
	var failures []string
	// walked 由呼叫端給：量覆蓋率時直接傳 visited（跨趟累積，不重做已經走過
	// 的路），要重走找出口時傳一份自己的。
	rewalks := map[[2]int]int{}
	// heldBack 是「暫時擋住的換圖格」。清掉船票之後不能離開索寇要塞
	//（見 mustSettleSokalGhost），但規劃器會把換圖格當成一般的沒踩過的格子
	// 走上去，所以那段期間先把它們塞進 avoid，了結之後再拿回來。
	heldBack := map[[3]int]bool{}
	lastMap, lastCell := application.spawn.Map, [2]int{-1, -1}
	if int(lastMap.Archive) == cityArchive && int(lastMap.BlockID) == cityBlock {
		avoid[[3]int{cityArchive, cityBlock, dockCell[1]*100 + dockCell[0]}] = true
		if deferHandIn {
			avoid[[3]int{cityArchive, cityBlock, 4*100 + 4}] = true
		}
	}
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
	// 野外的目標追蹤（wilderness_explore_test.go）。兩種走法與跨圖的列由
	// 種子決定：單一種走法量到的區塊不一樣，聯集才是覆蓋面。
	wild := newWildernessWalk(seed%2 == 0, int(seed%32))
	// 完成委託後先回文明區交差。東航線的下船點 (9,29) 同時是原版
	// 地點 3「回文明區的船」（spec 105）；剛下船時必須先離開一格，
	// 再以正常走路踏回去，才會重新觸發地點腳本。
	returnWild := newWildernessWalk(true, int(seed%32))
	returnDetour := newWildernessWalk(false, int(seed%32))
	// dockTried 同理：票拿到手之後主動走一次碼頭，每個鎖住週期一次。
	dockTried := false
	// routeBought 只在五項航線選單真的按下選擇後成立；走到港務長門口不等於
	// 已買票，否則會拿 `4AC4` 的舊目的地再次登船。
	routeBought := false
	// 同一堆寶物只嘗試 Share 一次；若還有不能整除或無法收入的餘額，
	// 正常玩家路徑是 Exit 後確認留下，不能永遠重按 Share。
	treasureShared := false
walk:
	for step := 0; step < budget; step++ {
		if stop != nil && stop(application) {
			reason = "指定的主線狀態已達成"
			break
		}
		if application.eventMachine != nil {
			now := [2]uint16{application.eventMachine.Memory[0x4AA7],
				application.eventMachine.Memory[0x4A01]}
			if now != quest {
				if now[1] != quest[1] {
					block := -1
					if application.eventSession != nil {
						block = int(application.eventSession.CurrentBlockID())
					}
					if application.spawn.Map.BlockID == 21 ||
						application.eclArchive == cityArchive && block == cityBlock {
						t.Logf("4A01 %d→%d 於 GEO%d/%d (%d,%d) ECL block %d",
							quest[1], now[1], application.spawn.Map.Archive,
							application.spawn.Map.BlockID, application.spawn.X,
							application.spawn.Y, block)
					}
				}
				quest = now
				for key := range approached {
					delete(approached, key)
				}
			}
		}
		if !application.treasureActive {
			treasureShared = false
		}
		if application.spawn.Map != lastMap {
			// plan／exit 都是以上一張 GEO 的牆與座標算出的；LOAD FILES
			// 換圖後若繼續送它的下一步，會在新圖重新規劃前誤踩另一個事件格，
			// 甚至立刻彈回原圖。換圖本身已由正常 Update() 完成，這裡只清除
			// 測試導覽器的過期導航狀態。
			plan, exit = nil, nil
			currentBlock := -1
			if application.eventSession != nil {
				currentBlock = int(application.eventSession.CurrentBlockID())
			}
			t.Logf("換圖 GEO%d/%d → GEO%d/%d 位置 (%d,%d) 朝向 %d",
				lastMap.Archive, lastMap.BlockID,
				application.spawn.Map.Archive, application.spawn.Map.BlockID,
				application.spawn.X, application.spawn.Y, application.spawn.Facing)
			if application.eventMachine != nil {
				t.Logf("  換圖狀態 ECL%d/%d 4AA7=%d 4A01=%d 4A26=%d pending=%v",
					application.eclArchive, currentBlock,
					application.eventMachine.Memory[0x4AA7],
					application.eventMachine.Memory[0x4A01],
					application.eventMachine.Memory[0x4A26], pendingCommission(application))
			}
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
			// 同一趟航程可能先從城區自然探索到 ECL4/21，開出
			// 4AA7=FE，再依 Sokal 航線重新抵達同一張圖。第二次
			// 抵達時，這張圖的腳本閘門已改變；不能沿用第一段
			// 探索留下的 walked，否則會把回程碼頭當成唯一出口，
			// 立刻彈回城區而沒有重跑新的事件。
			if application.spawn.Map == (gamepack.MapKey{Archive: 4, BlockID: 21}) &&
				application.eventMachine != nil &&
				application.eventMachine.Memory[0x4AA7] == uint16(gamepack.CityHallSlotPending) {
				for key := range walked {
					if key[0] == 4 && key[1] == 21 {
						delete(walked, key)
					}
				}
			}
			// 每次由外地回到城區，都是一趟新的航程：先重新向港務長選目的地，
			// 再走碼頭。這兩個布林只記探索器已送過哪些正常按鍵，不代表遊戲狀態。
			if int(application.spawn.Map.Archive) == cityArchive &&
				int(application.spawn.Map.BlockID) == cityBlock {
				harbourTried, dockTried, routeBought = false, false, false
				// 一般探索不能把碼頭當成普通事件格；只有五項航線選單真的
				// 買票後，下面的專用碼頭計畫才會放行這一格。
				avoid[[3]int{cityArchive, cityBlock, dockCell[1]*100 + dockCell[0]}] = true
				if deferHandIn {
					// 同一趟航程先把可達的委任做完，再一起回報。職員入口
					// 是 (4,4)；暫緩期間只擋這一格，不改任何 ECL 狀態。
					avoid[[3]int{cityArchive, cityBlock, 4*100 + 4}] = true
				}
			}
			// 踏上新的一張圖就先把邊界出口格擋起來。
			//
			// 不擋的話探索器是**走路走出去的，不是挑出口挑出去的**：規劃器
			// 逐格踩的時候會先踩到某一個邊界格、當場換走，而
			// `chooseAreaExit`（照 `exitUses` 輪流挑八個方向）只在「這一張
			// 踩完了」之後才跑。實測 GEO1/18 每一趟都只用 (15,4) 往東那一個
			// 出口，北邊那兩個一次都沒試過——而北邊那一支就是缺的區塊 9
			// （`99D4 ON GOTO @C04D` 第 0 支），後面還掛著 6→3→{4,5}→7。
			//
			// 擋住的是**走路踩上去**；`chooseAreaExit` 與它的走位不看 avoid，
			// 所以刻意離開這條路不受影響。
			for _, key := range boundaryExitKeys(application) {
				avoid[key] = true
			}
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
		// overrides 是同一種治具：把主線旗標直接寫進去，看那些區域自己
		// 走不走得動。與 boat 一樣，**這不是玩得到**。
		if application.eventMachine != nil {
			for address, value := range overrides {
				application.eventMachine.Memory[address] = value
			}
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
		// 撞到鎖住的門要把那個選單處理掉。**門選單是 modal 的**：
		// `a.door != nil` 的時候方向鍵與 ENTER 都歸它（`main.go` 的分派排在
		// `cellEventPending` 之後、轉向與前進之前），治具不處理就會在原地
		// 一直按方向鍵移門選單的游標，位置永遠不動。
		//
		// 每道門依序試 BASH、PICK、KNOCK；缺少的選項跳過，三種都失敗才 EXIT。
		// 動作仍由 Update() 的真實輸入分派接收，不直接呼叫處理函式。
		if application.door != nil && !application.cellEventPending {
			spin["門"]++
			doorTurns++
			if doorTurns > exploreMaxDoorTurns {
				failures = append(failures, fmt.Sprintf(
					"門選單關不掉：GEO%d/%d (%d,%d) 選項 %v 游標 %d 狀態列 %q",
					application.spawn.Map.Archive, application.spawn.Map.BlockID,
					application.spawn.X, application.spawn.Y,
					application.door.Options, application.door.Cursor,
					application.statusLine))
				reason = "門選單關不掉"
				break walk
			}
			key := explorerDoorKey(application)
			tried := doorAttempts[key]
			if tried == nil {
				tried = map[string]bool{}
				doorAttempts[key] = tried
			}
			want := explorerDoorChoice(application.door.Options, tried)
			if application.door.Options[application.door.Cursor] != want {
				if err := press(application, ebiten.KeyArrowRight); err != nil {
					failures = append(failures, fmt.Sprintf("第 %d 步：%v", step, err))
					break walk
				}
				continue
			}
			tried[want] = true
			if err := press(application, ebiten.KeyEnter); err != nil {
				failures = append(failures, fmt.Sprintf("第 %d 步：%v", step, err))
				break walk
			}
			continue
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
				// **地圖與地形碼要一起印。** 腳本分派看的是 `C04Fh`
				// （由 `initialMap` 那一張的地形碼投影進去的），而這裡印的
				// 位置來自 `spawn`。兩者對不起來的時候，只印座標會把人帶去
				// 查錯的腳本——井那一條就是這樣繞了一圈（spec 041）。
				loaded, terrain := "無", -1
				if application.initialMap != nil {
					loaded = fmt.Sprintf("GEO%d/%d",
						application.initialMap.Key.Archive, application.initialMap.Key.BlockID)
					cell := application.initialMap.Grid.CellWrapped(
						int(application.spawn.X), int(application.spawn.Y))
					terrain = int(cell.Terrain & 0x7F)
				}
				code, gate, answer, at := -1, -1, -1, -1
				if application.eventMachine != nil {
					code = int(application.eventMachine.Memory[0xC04F] & 0x7F)
					gate = int(application.eventMachine.Memory[0x4A10])
					answer = int(application.eventMachine.Memory[0x9802])
					at = 0x9900 + application.eventMachine.PC
				}
				failures = append(failures, fmt.Sprintf(
					"格子選單卡住：GEO%d/%d (%d,%d) 游標 %d／%v 標籤 %q 文字 %q 狀態列 %q "+
						"block %d 這一格答過 %d 次 載入的地圖 %s 地形碼 %d C04F %d 4A10 %d 9802 %d PC $%04X",
					application.spawn.Map.Archive, application.spawn.Map.BlockID,
					application.spawn.X, application.spawn.Y,
					application.cellMenuCursor, application.cellMenuOptions,
					application.eventLabel, application.eventText,
					application.statusLine, block,
					menuTurn[explorerMenuKey(application, application.cellMenuOptions)],
					loaded, terrain, code, gate, answer, at))
				reason = "格子選單卡住"
				traceApp = application
				break walk
			}
		case application.encounter != nil:
			spin["遭遇"]++
		case application.combatActive:
			spin["戰鬥"]++
		case application.cellEventPending:
			spin["格子事件"]++
			eventKey := [4]int{int(application.spawn.Map.Archive),
				int(application.spawn.Map.BlockID),
				int(application.spawn.Y)*100 + int(application.spawn.X),
				len(application.eventText)}
			if eventKey != eventStallKey {
				eventStallKey, eventStall = eventKey, 0
			}
			eventStall++
			if eventStall > exploreMaxCombatStall {
				block := -1
				if application.eventSession != nil {
					block = int(application.eventSession.CurrentBlockID())
				}
				at := -1
				if application.eventMachine != nil {
					at = 0x9900 + application.eventMachine.PC
				}
				failures = append(failures, fmt.Sprintf(
					"格子事件卡住：GEO%d/%d (%d,%d) 標籤 %q 文字 %q 狀態列 %q block %d PC $%04X",
					application.spawn.Map.Archive, application.spawn.Map.BlockID,
					application.spawn.X, application.spawn.Y,
					application.eventLabel, application.eventText,
					application.statusLine, block, at))
				reason = "格子事件卡住"
				traceApp = application
				break walk
			}
		case application.mode != modeAdventure:
			spin[fmt.Sprintf("模式 %v", application.mode)]++
		default:
			spin["走路"]++
			menuStall, eventStall, doorTurns = 0, 0, 0
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
			want := treasureMenuChoice(application.cellMenuOptions)
			if !treasureShared && application.state.PooledMoney != ([7]uint32{}) {
				for index, option := range application.cellMenuOptions {
					if strings.EqualFold(option, "Share") {
						want = index
						break
					}
				}
			}
			if want != application.cellMenuCursor {
				key = ebiten.KeyArrowRight
			}
			if key == ebiten.KeyEnter &&
				strings.EqualFold(application.cellMenuOptions[application.cellMenuCursor], "Share") {
				treasureShared = true
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
				inputKey := explorerCellKey(application)
				word := eclPasswords[menuTurn[inputKey]%len(eclPasswords)]
				// 答案就寫在問句的括號裡（`eclInputAnswer` 從原版資料解出來，
				// 玩家看得到），治具照著打——這也是玩家實際會做的事，比自己
				// 猜準。多個候選（同一個變數被 `SAVE` 兩次）就輪流試。
				//
				// 這一步很重要：ecl7/23 的密碼門答錯一次會扣血、
				// `A564 OR 4A51h #64` 設旗標然後 EXIT，入口 `A3E4` 檢查同一個
				// bit 就 EXIT——**那一格只有一次機會**，輪流猜等於把它用掉。
				switch hints := answerHints(application.eventText); {
				case len(hints) != 0:
					word = hints[menuTurn[inputKey]%len(hints)]
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
				menuTurn[inputKey]++
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
				// 計數的鍵是「格子 ＋ 這一組選項」，不是只有格子。
				// 只用格子的話，同一格上的 YES／NO、地點選單與密碼輸入
				// 共用一個計數，彼此把指標推走，選項就不是逐一輪完而是跳著選。
				//
				// **這一步本身沒有讓覆蓋變多**（改前改後都是 17 個 ECL 區塊、
				// 18 張地圖，A／B 各跑一次量過）。留著是因為「每一組選項各自
				// 輪完」才是這個治具想做的事；覆蓋要再往上得靠別的。
				optionKey := explorerMenuKey(application, application.cellMenuOptions)
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
				want := menuTurn[optionKey] % len(application.cellMenuOptions)
				// 主線不殺 NPC：有 LEAVE 可選的 ATTACK 選單（算命的老婦人、
				// 沒貨的店家）一律不選 ATTACK——殺老婦人會把 `4A0B` 設成 FFh，
				// 之後貧民窟的隨機遭遇人數 +5、屋內休息也會被打斷；打店家
				// 招來 21 名衛兵。有 SPEAK 就講話，否則走人。
				if flags != nil {
					attack, leave, speak := -1, -1, -1
					for index, option := range application.cellMenuOptions {
						switch strings.ToUpper(option) {
						case "ATTACK":
							attack = index
						case "LEAVE":
							leave = index
						case "SPEAK":
							speak = index
						}
					}
					if attack >= 0 && leave >= 0 {
						want = leave
						if speak >= 0 {
							want = speak
						}
					}
				}
				// 剛打完密碼的那一次是確認框，答 NO 只會跳回去重打。
				if confirmInput[key] {
					want = 0
				}
				// flags 非 nil 那一條要推主線，所以 YES／NO 先答 YES
				//（「要不要拿走裝備」答 NO 就推不動要塞那一段）。
				//
				// **但不能一直答 YES。** 有些 YES／NO 是雙向的梯子：
				// `ecl8/29` 的井答 YES 下去（`AE87h` 把 `4A10h` 設 1、
				// `AEAFh LOAD FILES #32`），到下面再答 YES 就爬回來
				// （`A23Fh` 把 `4A10h` 清 0、`A26Ah LOAD FILES #29`）。
				// 一律 YES 的話探索器就在兩層之間上上下下，永遠出不來——
				// 那正是先前「答了四千次」的成因，**遊戲沒有壞，是治具的
				// 答題策略壞了**（指令級追蹤只花六個位址就看出來：
				// `AF71 AF80 AF86 AF87 AE87 AE8D`，答完 `4A10h` 就變 1）。
				//
				// 答過幾次之後改回輪流，玩家也是這樣——下去看過了就不會再下去。
				if flags != nil && strings.EqualFold(application.cellMenuOptions[0], "YES") &&
					menuTurn[optionKey] < 4 {
					want = 0
				}
				// 索寇要塞上層的舊樓梯是雙向入口：第一次正常探索要下到
				// GEO8/30，完成下層委任後由 `mainlineSokalExitPlan` 帶回
				// 上層，這時同一格仍會再次顯示 YES／NO。玩家不會在剛爬
				// 回來後又立刻下樓；固定選 NO 才能讓上層繼續走向荒野
				// 出口。這只針對原文樓梯問句，不影響其他 YES／NO 地點。
				if application.initialMap != nil &&
					application.initialMap.Key == (gamepack.MapKey{Archive: 8, BlockID: 16}) &&
					application.eventMachine != nil &&
					application.eventMachine.Memory[0x4AA7] == 0xFF &&
					strings.Contains(strings.ToUpper(application.eventText), "STAIRS") {
					for index, option := range application.cellMenuOptions {
						if strings.EqualFold(option, "NO") {
							want = index
						}
					}
				}
				// 搭船抵達索寇要塞時，隊伍正站在回程碼頭，ECL4/21
				// 會先問「DO YOU WANT TO TAKE A BOAT BACK TO PHLAN?」。
				// 第一次必須答 NO 才能從碼頭走進要塞；探索完回到同一格
				// 時讓輪替策略答 YES，正常回城交件。
				if application.eventSession != nil &&
					application.eventSession.CurrentBlockID() == 21 &&
					application.eventMachine != nil &&
					application.eventMachine.Memory[0x4AA7] == uint16(gamepack.CityHallSlotPending) &&
					application.eventMachine.Memory[0x4A01] == 1 &&
					menuTurn["\x00sokal-dock-stayed"] == 0 &&
					application.spawn.Map == (gamepack.MapKey{Archive: 4, BlockID: 21}) &&
					application.spawn.X == 15 && application.spawn.Y == 1 {
					for index, option := range application.cellMenuOptions {
						if strings.EqualFold(option, "NO") {
							want = index
							menuTurn["\x00sokal-dock-stayed"] = 1
						}
					}
				}
				// 斯托揚諾河的密室不是一個單次 YES／NO 事件。原版先讓
				// 玩家搜尋，再在石板前顯示 GO BACK／MOVE ON／THROW A ROCK；
				// `THROW A ROCK` 會把 4A4D 的搜尋狀態推進，最後才由
				// ecl7/23 AC35/AC53 設 4A52 bit 2 與委任槽 13。通用輪替
				// 若先選 GO BACK 或 MOVE ON 只會離開／重置房間，永遠不會
				// 抵達結案分支。
				if application.eventSession != nil &&
					application.eventSession.CurrentBlockID() == 23 &&
					application.eventMachine.Memory[0x4AB3] < 0xFE {
					for index, option := range application.cellMenuOptions {
						if strings.EqualFold(option, "THROW A ROCK") {
							want = index
						}
					}
				}
				// 同一條路上要走到亡魂那一段：登陸的遭遇選「交涉」，Ferran
				// 問話必須先說謊、再說實話（spec 102）。第一次說謊把船票
				// `4A01` 清成 255，但不完成 `4A26`／`4AA7`；探索器留在要塞
				// 再觸發一次，說實話才完成亡魂委託並保留已清掉的船票。
				if flags != nil {
					// 港務長的完整航線選單：SOKAL 是回索寇要塞（已經走過），
					// NONE 是不上船，所以在 EAST／WEST／BAY 之間輪流挑。
					if len(application.cellMenuOptions) == 5 &&
						strings.EqualFold(application.cellMenuOptions[0], "SOKAL") {
						// City Hall 的 Sokal 委任仍是 FE 時，第一項 SOKAL
						// 才是這筆委任的正常目的地；若誤選 EAST，會落到
						// 荒野的其他地點，永遠不會進入 ECL4/21。
						// 結案後才輪流使用 EAST／WEST／BAY。
						if application.eventMachine.Memory[0x4AA7] == uint16(gamepack.CityHallSlotPending) {
							want = 0
						} else {
							want = 1 + menuTurn[optionKey]%3
						}
						// 公告進度 3 時，下一個窄目標是 Kuto's Well 的 Norris。
						// BAY 航線接 ECL7/26 城外樞紐，再由城區入口通往 Podal Plaza；選擇
						// 仍經港務長的正常五項選單，不寫船票或目的地旗標。
						if application.eventMachine.Memory[0x4AA7] != uint16(gamepack.CityHallSlotPending) &&
							application.eventMachine.Memory[0x4AC1] == 3 &&
							application.eventMachine.Memory[0x4AA6] < 0xFE {
							want = 3
						}
						if application.eventMachine.Memory[0x4AA7] != uint16(gamepack.CityHallSlotPending) &&
							application.eventMachine.Memory[0x4AC1] == 2 &&
							application.eventMachine.Memory[0x4AB0] < 0xFE {
							want = 3
						}
						// BAY 航線在荒野圖 26 的 (13,27) 落地；波多廣場的
						// 西城緣與墓園入口都在相鄰地點（spec 105）。
						if application.eventMachine.Memory[0x4AA7] != uint16(gamepack.CityHallSlotPending) &&
							application.eventMachine.Memory[0x4AC1] >= 4 &&
							application.eventMachine.Memory[0x4AB1] < 0xFE {
							want = 3
						}
						if application.cellMenuCursor == want {
							routeBought = true
						}
					}
					for index, option := range application.cellMenuOptions {
						if strings.EqualFold(option, "Parlay") {
							want = index
						}
						if !strings.EqualFold(option, "TELL THE TRUTH?") &&
							!strings.EqualFold(option, "LIE?") {
							continue
						}
						chooseLie := application.eventMachine.Memory[0x4A01] == 1
						if strings.EqualFold(option, "LIE?") != chooseLie {
							continue
						}
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
				// 帶封印的箱子不要拆。這不是憑感覺挑的：`ecl4/2` 的
				// `A99C ON GOTO` 兩條分支寫得很清楚——`OPEN IT`（索引 0）走
				// `A9AF SAVE 128 → 4AC8h`，`TAKE IT UNOPENED`（索引 1）走
				// `A9A8 SAVE 1 → 4AC8h`；而 `ecl3/0` 的 `AB17 COMPARE 4AC8h, 1`
				// 才會走到 `AB8A SAVE 254 → 4AB8h`，也就是卡德納那一條委任
				//（槽 18，spec 041）。拆了箱子那一條就再也交不了差，而輪流選
				// 會有一半的趟數把它拆掉。
				for index, option := range application.cellMenuOptions {
					if strings.EqualFold(option, "TAKE IT UNOPENED") {
						want = index
					}
				}
				// Norris the Gray 在 Kuto's Well 地下包圍隊伍時只給 SURRENDER／FIGHT。
				// City Hall 明確委託是除掉他，且原始 ecl8/29 的 FIGHT 分支才接
				// LOAD MONSTER 32/57/1、COMBAT 與槽 0 的 FEh producer（spec 041）。
				if application.spawn.Map.BlockID == 32 &&
					application.eventMachine.Memory[0x4AA6] != 0xFF {
					for index, option := range application.cellMenuOptions {
						if strings.EqualFold(option, "FIGHT") &&
							len(application.cellMenuOptions) == 2 &&
							strings.EqualFold(application.cellMenuOptions[0], "SURRENDER") {
							want = index
						}
					}
				}
				if (application.eventMachine.Memory[0x4AC1] == 2 &&
					application.eventMachine.Memory[0x4AB0] < 0xFE ||
					application.eventMachine.Memory[0x4AC1] == 3 &&
						application.eventMachine.Memory[0x4AA6] < 0xFE) &&
					len(application.cellMenuOptions) == 3 &&
					strings.EqualFold(application.cellMenuOptions[0], "NORTH") &&
					strings.EqualFold(application.cellMenuOptions[1], "SOUTH") {
					want = 0
				}
				if application.eventMachine.Memory[0x4AC1] == 3 &&
					application.eventMachine.Memory[0x4AA6] < 0xFE &&
					len(application.cellMenuOptions) == 3 &&
					strings.EqualFold(application.cellMenuOptions[0], "CITY") &&
					strings.EqualFold(application.cellMenuOptions[1], "GRAVEYARD") {
					want = 0
				}
				if pendingCommission(application) &&
					len(application.cellMenuOptions) == 3 &&
					strings.EqualFold(application.cellMenuOptions[0], "CITY") &&
					strings.EqualFold(application.cellMenuOptions[1], "GRAVEYARD") {
					want = 0
				}
				if application.eventMachine.Memory[0x4AC1] >= 4 &&
					application.eventMachine.Memory[0x4AB1] < 0xFE &&
					len(application.cellMenuOptions) == 3 &&
					strings.EqualFold(application.cellMenuOptions[0], "CITY") &&
					strings.EqualFold(application.cellMenuOptions[1], "GRAVEYARD") {
					want = 1
				}
				// 槽 11 在 City Hall 結案後是 FFh，而且這一槽依原版通知表不增加
				// 4AC1h。若交回通用輪替又選 GRAVEYARD，墓園完成分支會把它重寫
				// 成 FEh，製造一個原本已結案的重複委託。結案後由北門進 CITY，
				// 繼續探索同一公告進度解鎖的下一區。
				if application.eventMachine.Memory[0x4AB1] == 0xFF &&
					len(application.cellMenuOptions) == 3 &&
					strings.EqualFold(application.cellMenuOptions[0], "CITY") &&
					strings.EqualFold(application.cellMenuOptions[1], "GRAVEYARD") {
					want = 0
				}
				// 墓園 terrain 25 的黑色大理石墓穴以 SANCTIFY 把
				// `4A43` 寫成 FBh，下一次正常踏入才接到吸血鬼事件
				//（ECL4/10 `AF0Eh`、`B06Eh`、`B102h`）。選 EXAMINE
				// 只會離開，通用輪替會徒增整張圖的探索趟數。
				if application.eventMachine.Memory[0x4AC1] >= 4 &&
					application.eventMachine.Memory[0x4AB1] < 0xFE &&
					len(application.cellMenuOptions) == 3 &&
					strings.EqualFold(application.cellMenuOptions[0], "EXAMINE") &&
					strings.EqualFold(application.cellMenuOptions[1], "SANCTIFY") &&
					strings.EqualFold(application.cellMenuOptions[2], "OVERTURN") {
					want = 1
				}
				// 公告進度 2 的波多廣場任務要求查明拍賣品。入口的三條
				// `STRIDE`／`DISGUISE`／`SNEAK` 分支裡，原版攻略與 ECL
				// 拍賣完成分支都指向偽裝進場；通用輪替若先選 STRIDE，會
				// 把整趟變成清街怪而不是委任路徑（spec 041）。
				for index, option := range application.cellMenuOptions {
					if strings.EqualFold(option, "DISGUISE PARTY AS MONSTERS.") {
						want = index
					}
				}
				// 神殿估價後的 Sell／Keep 若選 Keep，珠寶仍留在身上，下一輪
				// Appraise 會再回到同一件物品。探索器要把這個有限流程收掉，
				// 所以估價完成就選 Sell；這仍是玩家選單的正常按鍵路徑。
				if len(application.cellMenuOptions) == 2 &&
					strings.EqualFold(application.cellMenuOptions[0], "SELL") &&
					strings.EqualFold(application.cellMenuOptions[1], "KEEP") {
					want = 0
				}
				// 在野外地形上遇到「要不要進去」就進去。進區域圖再走出來會把
				// 野外的位移重擲一次（`ecl6/25 A472h RANDOM 3`，spec 105），
				// 那是換口袋的唯一方法——不進去就只走得到落點那一個口袋。
				// 只認進入／調查區域的選項，所以不會誤選登陸點的 `TAKE BOAT`
				//（那是回城）。野外 26 的原文是 `INVESTIGATE`，漏掉它就會
				// 永遠拒絕西區唯一能重擲位移的正常玩家事件（spec 105）。
				if application.inWildernessOverland() {
					for index, option := range application.cellMenuOptions {
						if strings.EqualFold(option, "ENTER") ||
							strings.EqualFold(option, "ENTER CAVE") ||
							strings.EqualFold(option, "INVESTIGATE") {
							want = index
						}
					}
				}
				// 一旦已有待交委任，回文明區優先於繼續探索。這會在區域
				// 樞紐與荒野回程點選玩家可見的 TAKE BOAT，不改寫地圖或旗標。
				if !deferHandIn && pendingCommission(application) {
					for index, option := range application.cellMenuOptions {
						// 從金字塔回到區域樞紐時仍站在入口事件格；先選
						// GO BACK 留在外面，否則通用輪替會再次 ENTER。
						if application.spawn.Map == (gamepack.MapKey{Archive: 5, BlockID: 5}) &&
							strings.EqualFold(option, "GO BACK") {
							want = index
						}
						if strings.EqualFold(option, "TAKE BOAT") {
							want = index
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
				menuTurn[optionKey]++
				delete(confirmInput, key)
			}
			if err := press(application, ebiten.KeyEnter); err != nil {
				failures = append(failures, fmt.Sprintf("第 %d 步：%v", step, err))
				break walk
			}
			continue
		}
		if application.inWildernessOverland() {
			// 野外的位置存在 `DS:49C3h`／`DS:49C4h`，不在 GEO 格子上
			//（spec 105），但**有地點表**，而且野外座標與 GEO 格子是同步走
			// 的——所以規劃得起來。細節見 wilderness_explore_test.go。
			//
			// 仍然要有上限：野外沒有「這一格踩過了」可用，走不到目標也不會
			// 自己停，一趟就會把整包預算吃在那裡（實測連帶把整個 package 的
			// 10 分鐘 timeout 用完）。
			plan, exit = nil, nil
			spin["野外"]++
			if spin["野外"] > exploreMaxWildernessSteps {
				reason = "野外走到上限"
				break
			}
			key := ebiten.KeyArrowUp
			aimed := false
			want, ok := uint8(0), false
			here := [2]int{int(application.eventMachine.Memory[wildernessX]),
				int(application.eventMachine.Memory[wildernessY])}
			if pendingCommission(application) && application.eventMachine.Memory[0x4A01] == 1 {
				// 新船票落地時仍站在回程船的地點格，staging 要先正常走
				// 一步才把 4A01 清成 0。先走一個可通行方向，下一步再由
				// wildernessReturnFacing 原路踏回船格，讓 TAKE BOAT 事件出現。
				for facing := uint8(0); facing < 4; facing++ {
					if wildernessCanLeave(application, here, facing) {
						want, ok = facing, true
						break
					}
				}
			}
			// 公告進度 5 之後，尚未結案的三個野外委任各有原版地點表
			// 入口（spec 105）：蜥蜴人是圖 27 的 (11,8)，狗頭人是圖 27
			// 的 (6,15)，遊牧營地與斯托揚諾河分別是圖 26 的 (12,11)
			// 與 (6,16)。先以目前航線口袋能走得到的目標為優先；走不到
			// 時交回下方的跨圖／進出區域規劃，下一趟會以新的正常船路
			// 位移再試，不能直接寫旗標代替玩家走路。
			if !ok && !mustSettleSokalGhost(application) &&
				application.eventSession.CurrentBlockID() == 27 {
				target := [2]int{}
				haveTarget := false
				switch {
				case application.eventMachine.Memory[0x4AB5] < 0xFE:
					target, haveTarget = [2]int{11, 8}, true // lizardmen
				case application.eventMachine.Memory[0x4AB6] < 0xFE:
					target, haveTarget = [2]int{6, 15}, true // kobolds
				}
				if haveTarget {
					if route := wildernessRoute(application, here, target, nil); len(route) != 0 {
						want, ok = route[0], true
					}
				}
			}
			if !ok && !mustSettleSokalGhost(application) &&
				application.eventSession.CurrentBlockID() == 26 {
				target := [2]int{}
				haveTarget := false
				switch {
				case application.eventMachine.Memory[0x4AB7] < 0xFE:
					target, haveTarget = [2]int{12, 11}, true // nomads
				case application.eventMachine.Memory[0x4AB3] < 0xFE:
					target, haveTarget = [2]int{6, 16}, true // Stojanow River
				}
				if haveTarget {
					if route := wildernessRoute(application, here, target, nil); len(route) != 0 {
						want, ok = route[0], true
					}
				}
			}
			if !ok && application.eventMachine.Memory[0x4AC1] == 2 &&
				application.eventMachine.Memory[0x4AB0] < 0xFE &&
				application.eventSession.CurrentBlockID() == 26 {
				// 公告進度 2 的下一項是波多廣場。荒野圖 26 的 (11,28)
				// 是原版西城緣入口（spec 105），由那裡選北入口進廣場。
				if route := wildernessRoute(application, here, [2]int{11, 28}, nil); len(route) != 0 {
					want, ok = route[0], true
				}
			}
			if !ok && application.eventMachine.Memory[0x4AC1] == 3 &&
				application.eventMachine.Memory[0x4AA6] < 0xFE &&
				application.eventSession.CurrentBlockID() == 26 {
				if route := wildernessRoute(application, here, [2]int{11, 28}, nil); len(route) != 0 {
					want, ok = route[0], true
				}
			}
			if !ok && application.eventMachine.Memory[0x4AC1] >= 4 &&
				application.eventMachine.Memory[0x4AB1] < 0xFE &&
				application.eventSession.CurrentBlockID() == 26 {
				// 公告進度 4 解鎖瓦海登墳場委託（spec 038）。荒野圖 26
				// 的 (13,26) 是原版地點 2 的墓園入口（spec 105）；只規劃
				// 正常走路，抵達後仍由玩家選單決定是否進入。
				if route := wildernessRoute(application, here, [2]int{13, 26}, nil); len(route) != 0 {
					want, ok = route[0], true
				}
			}
			if !ok && !deferHandIn && pendingCommission(application) &&
				application.eventMachine.Memory[0x4A01] == 0 {
				want, ok = wildernessReturnFacing(application, zipPath, returnWild)
				if !ok {
					// 目前的平鋪迷宮若沒有通往東側邊界的路，先走最近的
					// 玩家可見地點並正常進出，讓原版腳本重擲野外位移；
					// 否則只會在同一個封閉口袋隨機漫步。
					want, ok = wildernessNextFacing(application, zipPath, returnDetour)
				}
			} else if !ok {
				want, ok = wildernessNextFacing(application, zipPath, wild)
				if !ok && flags != nil {
					// 連續主線已走完這趟航線的可達地點時，沿正常船路回 Phlan
					// 再選下一條航線；留在荒野亂走只會重踩同一座平鋪迷宮。
					want, ok = wildernessReturnFacing(application, zipPath, returnWild)
					if !ok {
						// WEST／BAY 的落點可能是孤立口袋，原版必須繼續走動等
						// `INVESTIGATE`／`ENTER` 隨機事件，進出區域後才會重擲
						// 位移。沒有可規劃路線時交回下面的有界正常按鍵漫步。
						ok = false
					}
				}
			}
			if ok {
				key, aimed = wildernessTurnKey(application.spawn.Facing, want), true
			}
			if !aimed {
				switch application.roller.Roll(1, 6) {
				case 1:
					key = ebiten.KeyArrowRight
				case 2:
					key = ebiten.KeyArrowLeft
				}
			}
			if err := press(application, key); err != nil {
				failures = append(failures, fmt.Sprintf("第 %d 步：%v", step, err))
				break walk
			}
			continue
		}
		// 這一張踩完之後才刻意走出去：站到邊界那一格、轉向外面、往前一步。
		// 規劃器本身不跨邊界（見 explorePlan），所以換區一定經過這一段。
		if exit != nil && application.eventSession != nil &&
			exitContext != [2]int{int(application.eclArchive), int(application.eventSession.CurrentBlockID())} {
			// C01Eh 可以先把座標繞到同一張 GEO 的對邊，再由 NEWECL 切到
			// 區域樞紐。ECL 上下文已換就代表出口分派成功；若只比 Map，
			// 探索器會把同一個出口當成尚未完成而反覆走。
			exit, plan = nil, nil
		}
		if exit != nil {
			at := [2]int{int(application.spawn.X), int(application.spawn.Y)}
			switch {
			case at == exit.cell && application.spawn.Facing == exit.facing:
				before := application.spawn
				if err := press(application, ebiten.KeyArrowUp); err != nil {
					failures = append(failures, fmt.Sprintf("第 %d 步：%v", step, err))
					break walk
				}
				// 邊界前進可能先被原版外部 CALL／隨機遭遇攔下；C01Eh 甚至會
				// 先把座標繞到對邊，下一個 RETURN 才跑到 NEWECL。只要仍是
				// 同一張圖，出口就仍是同一個玩家目標。舊治具不分結果就清掉
				// exit，八次停頓後把 hop 額度耗光，後續整輪不再嘗試離開。
				if application.spawn.Map != before.Map {
					exit = nil
				}
				plan = nil
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
				// 走去出口的路上也不能路過換區的格子。少了這一條，
				// `chooseAreaExit` 挑到的出口等於白挑——實測 GEO1/18 三次都
				// 挑中 (4,0) 朝北（那一支就是缺的區塊 9），三次都在半路被
				// 別的格子換走，落點是 GEO8/29 的 (1,11)。
				plan = planToCellsWithoutWrapping(application, rotate,
					func(x, y int) bool {
						return x == target.cell[0] && y == target.cell[1]
					},
					func(x, y int) bool {
						// 主線指定的 ECL1/18 西出口已有 DOS `99D4h`
						// 分支證據；舊探索趟留下的 avoid 不能把必經格子封死。
						// 沿途事件仍由 Update() 正常處理，這裡只重算導航。
						if application.spawn.Map == (gamepack.MapKey{Archive: 1, BlockID: 18}) &&
							target.facing == 3 {
							return false
						}
						return !heldHere && avoid[[3]int{
							int(application.spawn.Map.Archive),
							int(application.spawn.Map.BlockID), y*100 + x}]
					})
				if len(plan) == 0 {
					exit = nil
				}
			}
		}
		if len(plan) == 0 && exit == nil {
			if sokal := mainlineSokalExitPlan(application, rotate); len(sokal) != 0 {
				plan = sokal
				spin["索寇出口"]++
				if spin["索寇出口"] <= 8 {
					cell := application.initialMap.Grid.CellWrapped(int(application.spawn.X), int(application.spawn.Y))
					t.Logf("索寇出口計畫 #%d：GEO%d/%d (%d,%d) 朝向 %d 地形 %d C04F=%d，步數 %d",
						spin["索寇出口"], application.spawn.Map.Archive, application.spawn.Map.BlockID,
						application.spawn.X, application.spawn.Y, application.spawn.Facing,
						cell.Terrain&0x7F, application.eventMachine.Memory[0xC04F]&0x7F, len(plan))
				}
				continue
			}
			// 索寇亡魂的兩段狀態還沒結算時，港務長優先於市政廳。
			// `4AA7 == FE` 也長得像一筆待交委託，但 `4A01 == FF` 代表
			// 先前的船票已清掉；此時先踩 clerk office 只會跳過 reward scan，
			// 並讓下面的港務長計畫永遠拿不到執行機會（spec 102）。
			if !heldHere && !deferHandIn {
				mainline := cityHallPlan(application, rotate)
				if len(mainline) != 0 {
					plan = mainline
					spin["市政廳交差"]++
					if spin["市政廳交差"] <= 12 {
						block := -1
						if application.eventSession != nil {
							block = int(application.eventSession.CurrentBlockID())
						}
						t.Logf("市政廳計畫 #%d：GEO%d/%d (%d,%d) 朝向 %d ECL%d/%d 步數 %d",
							spin["市政廳交差"], application.spawn.Map.Archive,
							application.spawn.Map.BlockID, application.spawn.X, application.spawn.Y,
							application.spawn.Facing, application.eclArchive, block, len(mainline))
					}
					continue
				}
			}
			// 在 Stojanow 完成委任後，GEO5/5 的 (8,0) 會開出
			// TAKE BOAT／STAY，前者正常返回文明區。通用探索若仍以「最少
			// 使用的轉場」優先，會先走向其他區域並在樞紐間循環；有待交
			// 委任時改以玩家已知的回程船為目標，移動與選擇仍全部經 Update()。
			if !heldHere && !deferHandIn && pendingCommission(application) &&
				application.spawn.Map == (gamepack.MapKey{Archive: 5, BlockID: 5}) {
				plan = planToCells(application, rotate, func(x, y int) bool {
					return x == civilisedPhlanBoatCell[0] && y == civilisedPhlanBoatCell[1]
				})
				if len(plan) != 0 {
					spin["搭回程船交差"]++
					continue
				}
			}
			settling := heldHere
			// 航線開放後，每次回到城區都先問港務長再去碼頭。鎖住的那一趟
			// 尤其不能先踩其他地點：城區有好幾個地點會把
			// `4A01` 寫回 1（市政廳職員 ECL3/8 `9BACh` 只在 0 時寫，
			// 競技場 ECL3/11 `9CACh` 進到那一支就寫），先踩到就前功盡棄。
			//
			// 鎖的評估已經提到每一步（見迴圈上方），所以回到城區時手上的
			// 舊計畫會在鎖亮的那一刻被丟掉，不會照著走過去踩競技場。
			if int(application.spawn.Map.Archive) == cityArchive &&
				int(application.spawn.Map.BlockID) == cityBlock && !harbourTried &&
				application.eventMachine != nil &&
				application.eventMachine.Memory[0x4AA7] >= 254 &&
				application.eventMachine.Memory[0x4A01] != 1 {
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
							if avoid[[3]int{cityArchive, cityBlock, y*100 + x}] {
								return true
							}
							cell, ok := application.initialMap.Grid.Cell(x, y)
							return ok && cell.Terrain&0x7F != 0
						})
					if len(plan) != 0 {
						plan = append(plan, exploreStep{facing: 0})
					}
				}
				if len(plan) != 0 {
					spin["找港務長"]++
					t.Logf("港務長計畫 #%d：GEO%d/%d (%d,%d) 朝向 %d，首步 %d，步數 %d，4A01=%d",
						spin["找港務長"], application.spawn.Map.Archive,
						application.spawn.Map.BlockID, application.spawn.X, application.spawn.Y,
						application.spawn.Facing, plan[0].facing, len(plan),
						application.eventMachine.Memory[0x4A01])
					continue
				}
			}
			// 票拿到手之後主動走去碼頭。港務長那一段做完時 `4A01 == 1`
			// 而且 `4AA7 >= 254`，碼頭 (15,1) 的選單才會從「唯一的船是
			// 索寇要塞」變成五個目的地（spec 099）。
			//
			// 探索器自己走不過去：碼頭是換圖點，第一次用完就進了 avoid，
			// 規劃器再也不挑它。路上一樣繞開別的事件格，免得又踩到競技場。
			// 航線尚未解鎖時，碼頭本來就只有前往索寇要塞的一條船，不必先向
			// 港務長挑目的地；解鎖之後才要求 routeBought，避免沿用舊目的地。
			initialSokalRoute := application.eventMachine != nil &&
				application.eventMachine.Memory[0x4AA7] < 254 &&
				application.eventMachine.Memory[0x4A01] == 1
			unlockedRoute := application.eventMachine != nil && routeBought &&
				application.eventMachine.Memory[0x4AA7] >= 254 &&
				application.eventMachine.Memory[0x4A01] == 1
			if !settling && (initialSokalRoute || unlockedRoute) &&
				int(application.spawn.Map.Archive) == cityArchive &&
				int(application.spawn.Map.BlockID) == cityBlock && !dockTried &&
				application.eventMachine != nil {
				dockTried = true
				plan = planToCellsAvoiding(application, rotate,
					func(x, y int) bool { return x == dockCell[0] && y == dockCell[1] },
					func(x, y int) bool {
						if x == dockCell[0] && y == dockCell[1] {
							return false
						}
						if avoid[[3]int{cityArchive, cityBlock, y*100 + x}] {
							return true
						}
						cell, ok := application.initialMap.Grid.Cell(x, y)
						return ok && cell.Terrain&0x7F != 0
					})
				if len(plan) == 0 && initialSokalRoute {
					// 第一次去索寇以前，城市裡若有地點格橫在最短路上，可以正常
					// 踩過去處理；此時尚無已解鎖航線目的地可被它覆寫。
					plan = planToCellsAvoiding(application, rotate,
						func(x, y int) bool { return x == dockCell[0] && y == dockCell[1] },
						func(x, y int) bool {
							if x == dockCell[0] && y == dockCell[1] {
								return false
							}
							return avoid[[3]int{cityArchive, cityBlock, y*100 + x}]
						})
				}
				if len(plan) != 0 {
					spin["走去碼頭"]++
					continue
				}
			}
			if len(plan) == 0 {
				if graveyard := mainlineGraveyardVampirePlan(application, rotate); len(graveyard) != 0 {
					plan = graveyard
					spin["墓園吸血鬼"]++
					continue
				}
			}
			leaving := hops < exploreMaxTransitionHops && !settling &&
				(holdMap == nil || application.spawn.Map != *holdMap)
			if len(plan) == 0 && leaving {
				if auction := mainlinePodalAuctionPlan(application, rotate); len(auction) != 0 {
					plan = auction
					spin["波多拍賣"]++
					continue
				}
			}
			// 已知委任出口是主線目標，不是「整張圖探索完才試」的 fallback。
			// 若先跑通用 explorePlan，波多廣場會在數百個地點朝向之間耗盡
			// 本趟預算，甚至先踩到另一個換圖點；出口本身仍全部經正常按鍵。
			if len(plan) == 0 && !settling && (leaving || pendingCommission(application)) {
				if next, ok := mainlineCommissionExit(application); ok {
					spin["委任出口"]++
					if spin["委任出口"] <= 8 {
						t.Logf("委任出口 #%d：GEO%d/%d (%d,%d) ECL%d/%d → (%d,%d) 朝向 %d",
							spin["委任出口"], application.spawn.Map.Archive,
							application.spawn.Map.BlockID, application.spawn.X, application.spawn.Y,
							application.eclArchive, application.eventSession.CurrentBlockID(),
							next.cell[0], next.cell[1], next.facing)
					}
					exitUses[exitKey(application, next)]++
					exit, hops = &next, hops+1
					exitContext = [2]int{int(application.eclArchive), int(application.eventSession.CurrentBlockID())}
					continue
				}
			}
			var target [3]int
			plan, target = explorePlan(application, walked, avoid, !heldHere, rotate)
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
						// 走去地點的路上也不能路過 avoid 的格子——換區的
						// 格子踩到就換走，`explorePlan` 已經擋了，這一條
						// 漏掉就等於白擋。
						plan = planToCellsAvoiding(application, rotate,
							func(x, y int) bool { return x == from[0] && y == from[1] },
							func(x, y int) bool {
								return !heldHere && avoid[[3]int{
									int(application.spawn.Map.Archive),
									int(application.spawn.Map.BlockID), y*100 + x}]
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
				next, ok := chooseAreaExit(application, exitUses)
				if ok {
					// 選中就記一次。走不到那一格的出口不記的話，
					// 下一輪會挑到同一個，隊伍就卡在原地重選到 hop 用完。
					exitUses[exitKey(application, next)]++
					exit, hops = &next, hops+1
					exitContext = [2]int{int(application.eclArchive), int(application.eventSession.CurrentBlockID())}
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
		// 4AC8h 是「身上帶著卡德納的箱子」：`ecl4/2` 的 `TAKE IT UNOPENED`
		// 寫 1、`OPEN IT` 寫 128，而 `ecl3/0 AB17` 只認 1（槽 18，spec 041）。
		for _, address := range []uint16{0x4A21, 0x4AC4, 0x6E12, 0x4A01, 0x4AC5, 0x4AC8} {
			flags[address] = application.eventMachine.Memory[address]
		}
		// 二十六個委任槽（spec 041）也一起帶出來：它們是「玩家真的走到那個
		// 條件了沒有」唯一的直接答案。
		for address := uint16(0x4AA6); address <= 0x4ABF; address++ {
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
	menuTurn := map[string]int{}
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
	menuTurn := map[string]int{}
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
			visited, maps, blocks, flags, noBoatOverride, &hardFailures, nil, nil, nil, nil, false)
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
	t.Logf("走到的地圖：%d 張 %v；ECL block：%d 個 %v；僵局安全閥收場 %d 次",
		len(maps), sortedMapNames(maps), len(blocks), blockList, tacticalStalemateEndings)
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
	//  2. 航線剛開但亡魂尚未完成（`4A26 != FF`）時，不能先回市政廳或
	//     港務長；否則舊船票會被重新啟用。亡魂完成後（`4A26 == FF`）
	//     下一個正常玩家步驟是回市政廳交差，不再把隊伍鎖在港務長。
	//
	// 港務長賣出新票後先寫 1；實際搭船進目的區域時，blocks 25／26／27
	// 的 staging 會把這個共用工作格寫成 0。0 已是「票用完、繼續當地」；
	// 若把所有 `!= 1` 都視為待結算，就會在洞穴等區域錯誤封鎖全部出口。
	//
	// 亡魂還沒了結的第一種狀態與地圖無關；第二種只能在城區鎖起來。
	// 若隊伍還在貧民窟或外地就鎖住，會連回城的 transition 也一併被擋掉。
	return memory[0x4A01] == 255 && memory[0x4A26] != 255
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
				if !explorerCanTraverse(application, fromX, fromY, int(facing)) {
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

// mainlineCommissionExit 只為已由正常 City Hall 進度解鎖的委託選已知出口。
// GEO1/18 的 (15,11) 向東由本測試的正常 runtime trace 證實會進 ECL8/29；
// 其餘狀態交回資料導出的輪替策略。
func mainlineCommissionExit(application *app) (areaExit, bool) {
	if application.eventMachine == nil {
		return areaExit{}, false
	}
	if pendingCommission(application) {
		switch application.spawn.Map {
		case gamepack.MapKey{Archive: 1, BlockID: 31}:
			if application.eclArchive != 1 || application.eventSession == nil ||
				application.eventSession.CurrentBlockID() != 24 {
				return areaExit{}, false
			}
			// ECL1/24 `99B0h` 的 map-31 表把北支導到 ECL3/14；隊伍從
			// `(11,15)` 保持北向再走一步時，ECL3/14 `993Ch` 的方向 0
			// 分支接到 ECL7/26。這條兩段鏈避開東支會先進入的隨機遭遇。
			return areaExit{cell: [2]int{11, 0}, facing: 0}, true
		case gamepack.MapKey{Archive: 1, BlockID: 24}:
			return areaExit{cell: [2]int{15, 11}, facing: 1}, true
		}
	}
	if application.spawn.Map != (gamepack.MapKey{Archive: 1, BlockID: 18}) {
		return areaExit{}, false
	}
	progress := application.eventMachine.Memory[0x4AC1]
	if progress == 2 &&
		application.eventMachine.Memory[0x4AB0] == uint16(gamepack.CityHallSlotPending) {
		return areaExit{cell: [2]int{0, 11}, facing: 3}, true
	}
	if progress == 3 && application.eventMachine.Memory[0x4AA6] < 0xFE {
		return areaExit{cell: [2]int{15, 11}, facing: 1}, true
	}
	if progress == 3 &&
		application.eventMachine.Memory[0x4AA6] == uint16(gamepack.CityHallSlotPending) {
		return areaExit{cell: [2]int{0, 11}, facing: 3}, true
	}
	if progress >= 4 && application.eventMachine.Memory[0x4AB1] < 0xFE {
		return areaExit{cell: [2]int{0, 11}, facing: 3}, true
	}
	return areaExit{}, false
}

// mainlinePodalAuctionPlan 把已接到的波多委託導向原版 entry dispatcher 的
// terrain 1。ECL1/18 `9AC0h` 的第二支 edge 是拍賣流程 `A2A3h`，而真實
// GEO1/18 的 terrain-1 格是 (7,6)..(7,8)。路線與最後一步都經 Update()；
// 拍賣內的玩家選單仍由原 ECL 逐頁處理。
func mainlinePodalAuctionPlan(application *app, rotate int) []exploreStep {
	if application.eventMachine == nil || application.initialMap == nil ||
		application.eventMachine.Memory[0x4AC1] != 2 ||
		application.eventMachine.Memory[0x4AB0] >= 0xFE ||
		application.spawn.Map != (gamepack.MapKey{Archive: 1, BlockID: 18}) {
		return nil
	}
	return planToCells(application, rotate, func(x, y int) bool {
		cell, ok := application.initialMap.Grid.Cell(x, y)
		return ok && cell.Terrain&0x7F == 1
	})
}

// mainlineGraveyardVampirePlan 依原版 ECL4/10 的兩段吸血鬼狀態機規劃路線：
// terrain 25（`AEF8h`）先把墓穴聖化成 `4A43=FBh`，terrain 28
//（`B1E1h`）的第一次吸血鬼戰鬥再寫 `4A41=FAh`，最後回 terrain 25
// 才進 `B102h` 的委任戰鬥。這裡只規劃玩家實際可走的按鍵，不寫旗標。
func mainlineGraveyardVampirePlan(application *app, rotate int) []exploreStep {
	if application.eventMachine == nil || application.initialMap == nil ||
		application.eventMachine.Memory[0x4AC1] < 4 ||
		application.eventMachine.Memory[0x4AB1] >= 0xFE ||
		application.spawn.Map != (gamepack.MapKey{Archive: 4, BlockID: 10}) {
		return nil
	}
	targetTerrain := uint8(25)
	if application.eventMachine.Memory[0x4A43] > 0xFA &&
		application.eventMachine.Memory[0x4A41] < 0xFA {
		targetTerrain = 28
	}
	return planToCells(application, rotate, func(x, y int) bool {
		cell, ok := application.initialMap.Grid.Cell(x, y)
		return ok && cell.Terrain&0x7F == targetTerrain
	})
}

// mainlineSokalExitPlan 在索寇要塞下層完成亡魂委託後，走向 ECL8/16
// `9B00h`／`9B0Ch` 表列的「往上」地形與朝向。這些不是 GEO 邊界出口，
// chooseAreaExit 看不到；移動仍是從相鄰格經 Update() 踏進目標格。
func mainlineSokalExitPlan(application *app, rotate int) []exploreStep {
	if application.eventMachine == nil ||
		application.eventSession == nil || application.eventSession.CurrentBlockID() != 16 ||
		application.initialMap == nil {
		return nil
	}
	// 亡魂委託尚未結案時，隊伍必須留在 GEO8/16 上層，走樓梯進入
	// GEO8/30；把上層直接送回荒野會跳過 ECL8/16 的委託。結案後腳本
	// 才會把隊伍留在 GEO8/30 下層，該圖沒有邊界出口，必須沿同一組
	// 「往上」地形回到 GEO8/16，才能由正常 NEWECL 27 回到荒野。
	if application.initialMap.Key == (gamepack.MapKey{Archive: 8, BlockID: 16}) {
		if application.eventMachine.Memory[0x4AA7] != uint16(gamepack.CityHallSlotPending) {
			return nil
		}
		// 讓一般 cell-menu 路由處理樓梯的 YES；這裡只負責完成後的下層出口。
		return nil
	}
	if application.initialMap.Key != (gamepack.MapKey{Archive: 8, BlockID: 30}) ||
		application.eventMachine.Memory[0x4AA7] == uint16(gamepack.CityHallSlotPending) {
		return nil
	}
	directions := map[uint8]uint8{17: 0, 18: 1, 19: 0, 20: 0, 21: 2, 22: 3, 23: 0}
	here := application.initialMap.Grid.CellWrapped(int(application.spawn.X), int(application.spawn.Y))
	if facing, ok := directions[here.Terrain&0x7F]; ok {
		return []exploreStep{{facing: facing}}
	}
	var best []exploreStep
	for y := 0; y < geometry.Height; y++ {
		for x := 0; x < geometry.Width; x++ {
			cell, ok := application.initialMap.Grid.Cell(x, y)
			if !ok {
				continue
			}
			facing, ok := directions[cell.Terrain&0x7F]
			if !ok {
				continue
			}
			from := [2]int{x - exploreDeltas[facing][0], y - exploreDeltas[facing][1]}
			if from[0] < 0 || from[0] >= geometry.Width || from[1] < 0 || from[1] >= geometry.Height ||
				!application.initialMap.Grid.CanMoveDungeonWrapped(from[0], from[1], int(facing)*2) {
				continue
			}
			route := planToCells(application, rotate, func(candidateX, candidateY int) bool {
				return candidateX == from[0] && candidateY == from[1]
			})
			if len(route) == 0 && from != [2]int{int(application.spawn.X), int(application.spawn.Y)} {
				continue
			}
			// 第一個同向步驟踏上表列地形；下一個步驟才讓 per-turn 入口
			// 以「目前地形＋目前朝向」命中出口分派。
			route = append(route, exploreStep{facing: facing}, exploreStep{facing: facing})
			if best == nil || len(route) < len(best) {
				best = route
			}
		}
	}
	return best
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
// tourOverrides 是巡迴用的主線治具：委任進度直接寫成 9（`DS:4AC1h`，
// spec 024 的 `ON GOSUB` 認 1..9），讓城內那幾區的閘門先開起來。
// 與 `4AC4h` 的航線覆寫同一個性質——**這不是玩得到**。
// tourWorldStates 是巡迴要走過的三種主線狀態。每一種開的區域不一樣，
// **量的是三者的聯集**——單獨一種都不夠：
//
//   - 什麼都不設：一開始的城區，走得到 12 個區塊。
//   - 委任進度 9（`DS:4AC1h`，spec 024 的 `ON GOSUB` 認 1..9）：
//     多開 16 與 17。
//   - 26 個完成槽（`4AA6h..4ABFh`，spec 041）全設 `FEh`：多開 22 與 23，
//     但已完成的區域反而不再給內容，所以它單獨量出來只有 10 個。
//
// 這些都是**治具**，與 `4AC4h` 的航線覆寫同一個性質：證明的是「那些區域的
// 腳本在完整的前端底下跑得動」，不是玩家走得到。
func tourWorldStates() []map[uint16]uint16 {
	completed := map[uint16]uint16{}
	for address := uint16(0x4AA6); address <= 0x4ABF; address++ {
		completed[address] = 0xFE
	}
	// `4A8C = 255` 是「接下拯救比凡特家繼承人的委任」（`ecl3/8 ACFAh`，
	// 印出 'THE HEIR TO THE HOUSE OF BIVANT MUST BE RESCUED.' 之後才寫）。
	// 野外圖 25 的 (12,31) 海盜基地要 `4A8C == 255` 且 `4AA9 == 0` 才會給
	// `ENTER` 選單（`ecl6/25 9D87h`／`9D92h`），而那是 EAST 航線那個口袋裡
	// 唯一進得去的地方——不進去就沒有機會重擲野外位移（spec 105）。
	// `4A77 = 4` 是「近塔已清、遠塔還在」。`ecl2/9` 的 A444／A457 讀這個值的
	// 第 3 位與第 2 位湊出 `@6E7A`，只有 `== 2` 那一支才演
	// 'MONSTERS ARE CHARGING TOWARD YOU FROM THE FAR TOWER.'——而那一場遭遇
	// 的非戰鬥結局是**區塊 6 唯一的入口**（`A57E ON GOTO` 兩支之外的
	// fall-through → `A58E` 擺位置 → `A5A0 NEWECL 6`）。後面還掛著 3、4、5、7。
	return []map[uint16]uint16{nil, {0x4AC1: 9}, completed,
		{0x4A8C: 255, 0x4A77: 4}}
}

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
	// menuTurn **跨趟共用**：探索器對同一格的選單是「第幾次來就選第幾項」，
	// 每一趟重新歸零的話，每一趟都只選得到第 0 項。樞紐圖的地點是選單選的
	// （`geo6/25` 只有 62 格卻分成 42 個互不相連的區塊），不換選項就永遠
	// 只進得去同一個地點。
	menuTurn := map[string]int{}
	// exitUses **跨趟共用**。每一趟重新建的話，`chooseAreaExit` 每一趟都從
	// 同一個出口開始輪——實測 GEO4/2 六次都挑北邊那兩個，而缺的區塊 15 在
	// **東**邊（`ecl4/2 996Dh` 的 `ON GOTO @C04D` 第 1 支：`6E12 = 2`、
	// `NEWECL 15`）。共用之後八個方向才輪得完。
	exitUses := map[[4]int]int{}
	for pass, destination := range []int{0, 1, 2, 3, 1, 2, 3, 1, 2, 3,
		1, 2, 3, 1, 2, 3, 1, 2, 3, 1, 2, 3,
		1, 2, 3, 1, 2, 3, 1, 2, 3, 1, 2, 3} {
		seed := int64(13 + pass*7 + destination)
		avoid := map[[3]int]bool{}
		transitionUses := map[[3]int]int{}
		states := tourWorldStates()
		_, reachable := exploreWorldWithFlags(t, zipPath, seed, 0, 1, 200000,
			avoid, map[[3]int]bool{}, transitionUses, menuTurn, exitUses,
			visited, maps, blocks, nil, destination, &hardFailures,
			states[pass%len(states)], nil, nil, nil, false)
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
	// **門檻不跟著實測值走**：三種主線狀態的聯集現在量得到 17 個，門檻留在 15
	// ——把門檻頂到實測值等於再做一次單一亂數對齊的快照，下一個會改變抽籤
	// 次數的修改又會紅。
	if len(blocks) < 15 {
		t.Errorf("只走到 %d 個 ECL block：%v", len(blocks), blockIDs)
	}
}

// 委任的前半段：**玩家自己走得到那個條件嗎。**
//
// spec 041 的另外兩支測試各驗一半——producer 端驗「條件成立時腳本會不會把槽
// 寫成 `FEh`」，交差端驗「`FEh` 之後職員會做什麼」。中間那一段（從開場走過去、
// 在那張圖上把條件滿足掉）只有貧民窟（`TestTwentyFiveRealSlumsWins…`）與索寇
// 要塞（`TestSokalKeepOpensTheOtherBoatRoutes`）是整條連著跑的。
//
// 這一支量的就是那一段：**完全不給主線旗標**，讓探索器從開場自己走，
// 走完看有幾個委任槽被寫成 `FEh`。與世界巡迴那一支的差別正在這裡——
// 那一支會把 26 槽全部預設成 `FEh` 去解鎖內容，所以它量不到這件事。
func TestPlayingTheWorldCompletesCommissionsOnItsOwn(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	completed := map[uint16]bool{}
	visited := map[[3]int]bool{}
	maps := map[string]bool{}
	blocks := map[int]bool{}
	menuTurn := map[string]int{}
	exitUses := map[[4]int]int{}
	var hardFailures []string
	ok := false
	var carry map[uint16]uint16
	chestInHand := false
	incrementsProgress := progressIncrementSlots(t, zipPath)
	// 兩個階段。第一階段從乾淨的開場走；第二階段從「把第一階段打出來的委任
	// 交差完」那個狀態走。
	//
	// **交差是解鎖的關鍵，不是收尾。** 探索器會把委任打完，但它不會走進市政廳
	// 交差，所以 `4AC1h`（公告進度）永遠是 0——而下一批區域正是靠那個數字開的。
	// 第二階段帶的是這些趟**自己打出來的**成果經過交差之後的狀態：該槽變成
	// `FFh`、十個會加一的槽各讓 `4AC1h` 加一。那條交差鏈由
	// TestEveryCommissionHandsInAtCityHall 逐槽驗過，所以把兩段接起來是有
	// 根據的，不是憑空給旗標。
	//
	// 試過直接帶 `FEh`（做完但沒交差）：**更差**——委任 3→1 條、地圖 13→7 張。
	// `FEh` 是「做完還沒回報」，港務長的選單與各區出口會走到另一條路上。
	for stage, boats := range [][]int{
		{0, 1, 2, 3, 0, 1, 2, 3, 0, 1, 2, 3},
		{0, 1, 2, 3, 0, 1, 2, 3},
	} {
		for pass, boat := range boats {
			flags := map[uint16]uint16{}
			_, reachable := exploreWorldWithFlags(t, zipPath,
				int64(29+stage*101+pass*11+boat), pass%4, 1, 120000,
				map[[3]int]bool{}, map[[3]int]bool{}, map[[3]int]int{},
				menuTurn, exitUses, visited, maps, blocks, flags, boat,
				&hardFailures, carry, nil, nil, nil, false)
			if !reachable {
				t.Skip("original DOS ZIP is intentionally not tracked")
			}
			ok = true
			for address := uint16(0x4AA6); address <= 0x4ABF; address++ {
				if flags[address] >= 0xFE {
					completed[address] = true
				}
			}
			// 箱子完好就帶著它進下一趟。它是隊伍身上的東西，與委任狀態
			// 同一類，跨趟不該憑空消失。卡德納那一條要的是「先在 GEO4/2
			// 拿到完好的箱子，再帶著它站上城區 (0,4) 的 gateway」——兩步
			// 隔著整張地圖，湊在同一趟的機率不高，而每一趟的起點正是 (0,4)。
			if flags[0x4AC8] == 1 {
				chestInHand = true
				if carry != nil {
					carry[0x4AC8] = 1
				}
			}
		}
		if stage != 0 {
			continue
		}
		carry = map[uint16]uint16{}
		progress := uint16(0)
		for address := range completed {
			carry[address] = uint16(gamepack.CityHallSlotAcknowledged)
			if incrementsProgress[int(address-0x4AA6)] {
				progress++
			}
		}
		carry[0x4AC1] = progress
		if chestInHand {
			carry[0x4AC8] = 1
		}
		t.Logf("第一階段打完 %d 條，交差之後 4AC1h = %d", len(completed), progress)
	}
	if !ok {
		t.Skip("original DOS ZIP is intentionally not tracked")
	}
	slots := make([]int, 0, len(completed))
	for address := range completed {
		slots = append(slots, int(address-0x4AA6))
	}
	sort.Ints(slots)
	t.Logf("玩家自己走完的委任 %d 條：%v", len(slots), slots)
	t.Logf("順帶走到的地圖 %d 張、ECL block %d 個", len(maps), len(blocks))
	// 量到的下限，不是目標。現在量得到五條：0（諾里斯）、1（索寇要塞）、
	// 11（瓦海登墳場）、18（卡德納的付款）、23（巴恩神殿）。
	// **少於這個數代表玩家走得到的主線退步了**，而那是「測試綠、玩家卡關」
	// 這一類缺陷唯一擋得住的地方。這一支抓到過兩個：NPC 入隊沒帶職業
	// （一條→三條），以及治具在雙向梯子上一律答 YES 上上下下（三條→五條）。
	if len(slots) < 5 {
		t.Errorf("只走完 %d 條委任：%v", len(slots), slots)
	}
	if len(hardFailures) != 0 {
		t.Errorf("出現 %d 次硬失敗", len(hardFailures))
	}
}

// traceApp 留下最後一個卡在格子選單的 app，讓追蹤探針接手。**只在測試裡用。**
var traceApp *app

// progressIncrementSlots 是「交差會讓 `4AC1h` 加一」的那十個槽。
// 名單解自原版 ECL3/block 8 的 `9D63h` ON GOSUB，不是抄的。
func progressIncrementSlots(t *testing.T, zipPath string) map[int]bool {
	t.Helper()
	application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	archive, ok := application.eclCatalog.Archive(3)
	if !ok {
		t.Fatal("ECL3 archive is absent")
	}
	slots, err := gamepack.ReadCityHallNotifications(archive)
	if err != nil {
		t.Fatal(err)
	}
	out := map[int]bool{}
	for _, slot := range slots {
		if slot.IncrementsProgress {
			out[slot.Index] = true
		}
	}
	return out
}
