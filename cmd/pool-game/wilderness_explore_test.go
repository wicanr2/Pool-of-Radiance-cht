package main

import (
	"path/filepath"
	"sort"
	"sync"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// 野外的走法：瞄地點表，而且照 GEO 的牆規劃路線。
//
// 野外的位置存在 `DS:49C3h`／`DS:49C4h`，不在 GEO 格子上（spec 105），先前
// 的做法因此是輪流轉向再往前走。那走得到「野外會動」，走不到任何一個**地點**
// ——三張圖上有腳本的格子加起來只有 46 個，散在 X 2..15、Y 8..33 這片範圍裡。
// 而缺的 ECL 區塊 1、13、19、28 的入口全部在那 46 格上（spec 101、105）。
//
// 規劃得起來的關鍵是**野外座標與 GEO 格子是同步走的**：spec 105 已經證明每
// 一步是「引擎在載入的 GEO 上走一格，ECL 同時把野外座標推一格」，牆擋住的
// 時候兩個都不動。所以兩者的差是常數，野外座標對到的 GEO 格子算得出來
// （模 16），可通行判定就直接用 `CanMoveDungeonWrapped`——和遊戲自己走路
// 用的是同一支。野外因此是一張**把 GEO 平鋪上去的迷宮**，可以做 BFS。
//
// 目標座標是**從原版資料讀的**（`gamepack.ReadDOSWildernessSheets` 解那四張
// 表），不是從攻略抄的。

// exploreMaxWildernessTargetSteps 是同一個野外目標最多花幾步。走不到就換下
// 一個：地形不可通行時 ECL 會寫 `6DC9 = 255` 拒絕，座標不動。
const exploreMaxWildernessTargetSteps = 120

// 野外座標的範圍（spec 105）。BFS 不出這個框——X 走到 2 或 15 就是跨圖，
// 那是另一件事。
const (
	wildernessMinX, wildernessMaxX = 2, 15
	wildernessMinY, wildernessMaxY = 3, 34
)

var (
	wildernessPlacesOnce sync.Once
	wildernessPlaces     map[int][][2]int
)

// wildernessTargetsFor 回傳一張野外圖上所有有腳本的座標。讀不出來就回空——
// 呼叫端會退回亂走，不會因此把整趟弄掉。
func wildernessTargetsFor(zipPath string, block int) [][2]int {
	wildernessPlacesOnce.Do(func() {
		wildernessPlaces = map[int][][2]int{}
		sheets, err := gamepack.ReadDOSWildernessSheets(zipPath)
		if err != nil {
			return
		}
		for _, sheet := range sheets {
			for _, place := range sheet.Places {
				wildernessPlaces[sheet.BlockID] = append(
					wildernessPlaces[sheet.BlockID], [2]int{place.X, place.Y})
			}
		}
	})
	return wildernessPlaces[block]
}

// wildernessCrossings 是每一張野外圖跨得出去的兩側：走到那個 X 再往那個方向
// 踏一步，入口 0 就會換到鄰圖（spec 105 的三張邊界表）。三張圖由西到東是
// 25 → 26 → 27。
//
// 順序有意義：**先跨到還沒去過的鄰圖，再走這一張的地點**。一張圖上隨便一個
// 地點都可能立刻 `NEWECL` 把隊伍帶走，先走地點就幾乎不會走到最西邊那一張
// ——而區塊 1、19、28 的入口全部只在圖 25 上。
var wildernessCrossings = map[int][]wildernessCrossing{
	27: {{X: wildernessMinX, Facing: 3, To: 26}},
	26: {{X: wildernessMinX, Facing: 3, To: 25}, {X: wildernessMaxX, Facing: 1, To: 27}},
	25: {{X: wildernessMaxX, Facing: 1, To: 26}},
}

type wildernessCrossing struct {
	X      int
	Facing uint8
	To     int
}

// wildernessWalk 是探索器在野外的狀態。walked 與 gaveUp 只在一趟裡累積：
// 不同的主線旗標會開不同的地點，跨趟記住反而會把後來才開的那些擋掉。
type wildernessWalk struct {
	walked, gaveUp map[[3]int]bool
	sheets         map[int]bool
	// route 是規劃好的一整條路線，at 是規劃時站的那一格。
	//
	// **要整條記住，不能每一步重算。** 重算的版本會每次挑「到任何一列的
	// 邊界最短的那一條」的第一步，而那個目標會隨著移動改變——實測從圖 26 的
	// (13,28) 出發，每一步都重挑，於是一路往西踩進 (11,28) 那個地點，
	// 被腳本送去區塊 18。繞開地點的規劃只有整條走完才算數。
	route  []uint8
	at     [2]int
	target [2]int
	block  int
	steps  int
	// crossFirst 決定「先跨圖還是先走地點」。兩種都要跑：先跨圖走得到最西邊
	// 那一張（區塊 1、19、28 的入口只在那裡），先走地點才走得到中間那張圖上
	// 的地點——實測兩種各自量到的區塊不一樣，聯集才是覆蓋面。
	crossFirst bool
	// rowPick 讓不同的趟從不同的列跨圖。跨圖之後野外座標與 GEO 格子的差會
	// 重算（`49C3` 被設成 14／3，GEO 只是往旁邊走一格），所以**從哪一列跨
	// 過去，決定了下一張圖是哪一座迷宮**。固定一列就只看得到一種。
	rowPick int
	// crossings 是這一趟跨過幾次圖。跨圖的列要一次換一個，不然三張圖會用
	// 同一條路線，等於只看得到一座迷宮。
	crossings int
	// stall 是「連續幾個 tick 位置沒動」。牆擋住的時候按前進不會有任何效果，
	// 而規劃器看不出差別——實測有一趟就這樣站在圖 26 的 (2,23) 對著西邊的牆
	// 按了七千多次。
	stall    int
	lastHere [2]int
	// reshuffles 是「這張圖能走的都走完了，就跨出去再跨回來」做過幾次。
	//
	// **它換不到另一座迷宮**：跨圖的位移推移是固定的（往東 +13、往西 −13），
	// 來回一趟剛好抵銷，位移回到原樣。上限從 6 拉到 24 實測走到的地點一格都
	// 沒多。留著是因為它讓隊伍換一列重新進來，落點會不一樣；真正決定迷宮的
	// 是**當初從哪一條航線下船**（spec 105）。
	reshuffles int
}

// wildernessMaxReshuffles 是一趟最多換幾座迷宮。沒有上限的話，地點全部走不到
// 的那幾張圖會把整趟的預算耗在來回跨圖上。
const wildernessMaxReshuffles = 6

// wildernessMaxStall 是位置沒動幾個 tick 就當這個目標走不到。轉向也要一個
// tick，所以不能太小。
const wildernessMaxStall = 8

func newWildernessWalk(crossFirst bool, rowPick int) *wildernessWalk {
	return &wildernessWalk{walked: map[[3]int]bool{},
		gaveUp: map[[3]int]bool{}, sheets: map[int]bool{}, block: -1,
		crossFirst: crossFirst, rowPick: rowPick}
}

// wildernessDeltas 是四方位的位移：0 北、1 東、2 南、3 西。
var wildernessDeltas = [4][2]int{{0, -1}, {1, 0}, {0, 1}, {-1, 0}}

func wildernessAdvance(from [2]int, facing uint8) [2]int {
	delta := wildernessDeltas[facing%4]
	return [2]int{from[0] + delta[0], from[1] + delta[1]}
}

// wildernessOffset 是「GEO 格子 − 野外座標」，模 16。兩者同步走，所以這個差
// 在同一張圖上是常數，可以直接從現況量出來，不必寫死。
func wildernessOffset(a *app) (int, int) {
	memory := a.eventMachine.Memory
	x := (int(a.spawn.X) - int(memory[wildernessX])) % 16
	y := (int(a.spawn.Y) - int(memory[wildernessY])) % 16
	return (x + 16) % 16, (y + 16) % 16
}

// wildernessCanLeave 說站在 cell 上往 facing 踏一步，GEO 的牆讓不讓。跨圖是
// 「走到某一個 X 再踏出去」，而**踏出去那一步一樣會被牆擋**——只看 X 對不對
// 會站在邊界對著牆一直按前進。
func wildernessCanLeave(a *app, cell [2]int, facing uint8) bool {
	offsetX, offsetY := wildernessOffset(a)
	return a.initialMap.Grid.CanMoveDungeonWrapped(
		(cell[0]+offsetX)%16, (cell[1]+offsetY)%16, int(facing)*2)
}

// wildernessRoute 用 BFS 算一條從 from 到 target 的路線，回傳每一步的四方位。
// 可通行判定用的是遊戲自己走路的那一支。走不到就回 nil。
//
// avoid 是「路過會出事的格子」。跨圖的時候要把這張圖的地點全部繞開：踩上去
// 就會跑那一支腳本，而地點腳本動不動就 `NEWECL` 把隊伍帶離野外——實測從圖 27
// 往西跨到 26 之後第二步就踩進 (11,28)，直接被送去區塊 18。
func wildernessRoute(a *app, from, target [2]int, avoid map[[2]int]bool) []uint8 {
	if from == target {
		return nil
	}
	offsetX, offsetY := wildernessOffset(a)
	deltas := wildernessDeltas
	type node struct{ x, y int }
	came := map[node]node{}
	step := map[node]uint8{}
	start := node{from[0], from[1]}
	seen := map[node]bool{start: true}
	queue := []node{start}
	goal := node{target[0], target[1]}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if current == goal {
			var route []uint8
			for at := current; at != start; at = came[at] {
				route = append([]uint8{step[at]}, route...)
			}
			return route
		}
		for facing := 0; facing < 4; facing++ {
			next := node{current.x + deltas[facing][0], current.y + deltas[facing][1]}
			if next.x < wildernessMinX || next.x > wildernessMaxX ||
				next.y < wildernessMinY || next.y > wildernessMaxY || seen[next] {
				continue
			}
			if next != goal && avoid[[2]int{next.x, next.y}] {
				continue
			}
			if !a.initialMap.Grid.CanMoveDungeonWrapped(
				(current.x+offsetX)%16, (current.y+offsetY)%16, facing*2) {
				continue
			}
			seen[next] = true
			came[next] = current
			step[next] = uint8(facing)
			queue = append(queue, next)
		}
	}
	return nil
}

// wildernessNextFacing 決定野外的下一步該面哪一個四方位。第二個回傳值為
// false 代表「沒有目標可瞄」，呼叫端退回亂走。
func wildernessNextFacing(a *app, zipPath string, walk *wildernessWalk) (uint8, bool) {
	block := int(a.eventSession.CurrentBlockID())
	memory := a.eventMachine.Memory
	here := [2]int{int(memory[wildernessX]), int(memory[wildernessY])}
	walk.walked[[3]int{block, here[0], here[1]}] = true
	walk.sheets[block] = true
	if block != walk.block {
		walk.block, walk.route, walk.steps = block, nil, 0
		walk.stall, walk.lastHere = 0, here
	}
	if here == walk.lastHere {
		walk.stall++
	} else {
		walk.stall, walk.lastHere = 0, here
	}
	if walk.stall > wildernessMaxStall {
		walk.gaveUp[[3]int{block, walk.target[0], walk.target[1]}] = true
		walk.route, walk.stall = nil, 0
	}
	switch {
	case len(walk.route) == 0:
	case here == walk.at:
		// 還沒踏出去（這一 tick 在轉向），照原計畫。
	case here == wildernessAdvance(walk.at, walk.route[0]):
		walk.route, walk.at = walk.route[1:], here
	default:
		walk.route = nil // 被腳本搬走了，重規劃
	}
	if len(walk.route) != 0 {
		walk.steps++
		if walk.steps > exploreMaxWildernessTargetSteps {
			walk.gaveUp[[3]int{block, walk.target[0], walk.target[1]}] = true
			walk.route = nil
		}
	}
	if len(walk.route) == 0 {
		walk.route, walk.target = planWildernessRoute(a, zipPath, block, here, walk)
		walk.at, walk.steps = here, 0
	}
	if len(walk.route) == 0 {
		// 已經站在跨圖那一格上：往外踏一步就換圖。
		for _, crossing := range wildernessCrossings[block] {
			if !walk.sheets[crossing.To] && here[0] == crossing.X &&
				wildernessCanLeave(a, here, crossing.Facing) {
				return crossing.Facing, true
			}
		}
		return 0, false
	}
	return walk.route[0], true
}

// planWildernessRoute 規劃下一條路線：先跨到還沒去過的鄰圖，都去過了才走這
// 一張自己的地點。
//
// 順序有意義：一張圖上隨便一個地點都可能立刻 `NEWECL` 把隊伍帶走，先走地點
// 就幾乎不會走到最西邊那一張——而區塊 1、19、28 的入口全部只在圖 25 上。
func planWildernessRoute(a *app, zipPath string, block int, here [2]int,
	walk *wildernessWalk) ([]uint8, [2]int) {
	crossing := func() ([]uint8, [2]int, bool) {
		for _, edge := range wildernessCrossings[block] {
			if walk.sheets[edge.To] || walk.gaveUp[[3]int{block, edge.X, -1}] {
				continue
			}
			if here[0] == edge.X && wildernessCanLeave(a, here, edge.Facing) {
				return nil, [2]int{edge.X, -1}, true
			}
			if route, ok := wildernessCrossingStep(a, zipPath, block, here, edge.X,
				edge.Facing, walk.rowPick+walk.crossings); ok {
				walk.crossings++
				return route, [2]int{edge.X, -1}, true
			}
			walk.gaveUp[[3]int{block, edge.X, -1}] = true
		}
		return nil, [2]int{}, false
	}
	places := func() ([]uint8, [2]int, bool) {
		for {
			target, ok := nearestWildernessPlace(zipPath, block, here, walk)
			if !ok {
				return nil, [2]int{}, false
			}
			if route := wildernessRoute(a, here, target, nil); len(route) != 0 {
				return route, target, true
			}
			walk.gaveUp[[3]int{block, target[0], target[1]}] = true
		}
	}
	order := [2]func() ([]uint8, [2]int, bool){crossing, places}
	if !walk.crossFirst {
		order = [2]func() ([]uint8, [2]int, bool){places, crossing}
	}
	for _, pick := range order {
		if route, target, ok := pick(); ok {
			return route, target
		}
	}
	// 這張圖能走的都走完了：從另一列跨出去再跨回來，換一座迷宮再試一次。
	if walk.reshuffles < wildernessMaxReshuffles {
		walk.reshuffles++
		walk.rowPick++
		walk.sheets = map[int]bool{block: true}
		walk.gaveUp = map[[3]int]bool{}
		if route, target, ok := crossing(); ok {
			return route, target
		}
	}
	return nil, [2]int{}
}

// wildernessCrossingStep 決定「往邊界那一格走」的下一步。四層退讓，順序就是
// 代價由小到大：
//
//  1. 繞開所有地點格走到邊界；
//  2. 繞不過去就允許路過地點。
//
// 繞開地點是因為踩上去就會跑那一支腳本，而地點腳本動不動就把隊伍帶離野外
// ——實測從圖 27 往西跨到 26 之後第二步就踩進 (11,28)，直接被送去區塊 18。
// 但**繞不過去的時候寧可路過也要跨圖**：最西邊那一張圖上有三個區塊的入口，
// 停在原地一個都拿不到。
func wildernessCrossingStep(a *app, zipPath string, block int, here [2]int,
	edge int, facing uint8, rowPick int) ([]uint8, bool) {
	places := wildernessPlaceSet(zipPath, block)
	for _, avoid := range []map[[2]int]bool{places, nil} {
		if route, ok := wildernessCrossingRow(a, here, edge, facing, avoid, rowPick); ok {
			return route, true
		}
	}
	return nil, false
}

// wildernessCrossingRow 列出所有走得到的邊界格，照路線長度排序，挑第
// rowPick 條。跨圖是「走到某一個 X 再踏出去」，同一個 X 有很多列，而**從哪
// 一列跨過去決定了下一張圖的迷宮長什麼樣**（見 wildernessWalk.rowPick）。
func wildernessCrossingRow(a *app, here [2]int, edge int, facing uint8,
	avoid map[[2]int]bool, rowPick int) ([]uint8, bool) {
	var routes [][]uint8
	for y := wildernessMinY; y <= wildernessMaxY; y++ {
		if !wildernessCanLeave(a, [2]int{edge, y}, facing) {
			continue
		}
		if route := wildernessRoute(a, here, [2]int{edge, y}, avoid); len(route) != 0 {
			routes = append(routes, route)
		}
	}
	if len(routes) == 0 {
		return nil, false
	}
	sort.Slice(routes, func(i, j int) bool { return len(routes[i]) < len(routes[j]) })
	return routes[rowPick%len(routes)], true
}

// wildernessPlaceSet 是一張圖上所有地點格子的集合，給 wildernessRoute 當
// avoid 用。
func wildernessPlaceSet(zipPath string, block int) map[[2]int]bool {
	set := map[[2]int]bool{}
	for _, place := range wildernessTargetsFor(zipPath, block) {
		set[place] = true
	}
	return set
}

// nearestWildernessPlace 挑目前這張圖上最近的一個還沒踩到、也還沒放棄的地點。
func nearestWildernessPlace(zipPath string, block int, here [2]int,
	walk *wildernessWalk) ([2]int, bool) {
	best, found, bestCost := [2]int{}, false, 0
	for _, place := range wildernessTargetsFor(zipPath, block) {
		key := [3]int{block, place[0], place[1]}
		if walk.walked[key] || walk.gaveUp[key] {
			continue
		}
		cost := abs(place[0]-here[0]) + abs(place[1]-here[1])
		if !found || cost < bestCost {
			best, bestCost, found = place, cost, true
		}
	}
	return best, found
}

// wildernessTurnKey 把「要面哪裡」換成這一 tick 要按的鍵。
func wildernessTurnKey(facing, want uint8) ebiten.Key {
	switch {
	case facing == want:
		return ebiten.KeyArrowUp
	case (int(want)-int(facing)+4)%4 == 3:
		return ebiten.KeyArrowLeft
	default:
		return ebiten.KeyArrowRight
	}
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

// 野外的規劃真的走得到最西邊那一張圖。
//
// 這一條是 spec 101「缺的 12 個區塊不是被機制擋住」的直接示範：從東邊的登陸
// 點出發，一路往西跨兩次圖，站上圖 25 的 (12,31)——原版在那一格印
// 「YOU HAVE REACHED THE BUCCANEER BASE」，而那一支的結尾就是 `NEWECL 1`。
//
// 亂走走不到這裡：先前的做法在同樣的預算內連圖 26 都跨不過去。
func TestTheWildernessWalkReachesTheWesternSheet(t *testing.T) {
	application := sailEastIntoTheWilderness(t)
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	walk := newWildernessWalk(true, 0)
	sheets := []int{int(application.eventSession.CurrentBlockID())}
	for step := 0; step < 3000 && application.inWildernessOverland(); step++ {
		key := ebiten.KeyArrowRight
		if want, ok := wildernessNextFacing(application, zipPath, walk); ok {
			key = wildernessTurnKey(application.spawn.Facing, want)
		}
		if err := press(application, key); err != nil {
			t.Fatal(err)
		}
		// 地點選單選最後一項：那通常是「離開／不進去」，留在野外才走得完。
		for tick := 0; tick < 200 &&
			(application.cellEventPending || application.cellWaitingMenu); tick++ {
			if application.cellWaitingMenu && len(application.cellMenuOptions) > 1 &&
				application.cellMenuCursor != len(application.cellMenuOptions)-1 {
				press(application, ebiten.KeyArrowRight)
				continue
			}
			press(application, ebiten.KeyEnter)
		}
		if !application.inWilderness() {
			break
		}
		if block := int(application.eventSession.CurrentBlockID()); block != sheets[len(sheets)-1] {
			sheets = append(sheets, block)
		}
	}
	want := []int{27, 26, 25}
	if len(sheets) != len(want) {
		t.Fatalf("走過的野外圖是 %v，預期 %v", sheets, want)
	}
	for index := range want {
		if sheets[index] != want[index] {
			t.Fatalf("走過的野外圖是 %v，預期 %v", sheets, want)
		}
	}
	if !walk.walked[[3]int{25, 12, 31}] {
		var onSheet [][2]int
		for key := range walk.walked {
			if key[0] == 25 {
				onSheet = append(onSheet, [2]int{key[1], key[2]})
			}
		}
		t.Fatalf("沒有站上圖 25 的 (12,31)（海盜基地）；圖 25 上踩過 %d 格：%v",
			len(onSheet), onSheet)
	}
}

// 在區域圖裡走一步，不可以動到野外座標。
//
// 野外那三個區塊各有兩種身分（spec 105）：地形圖上走路時野外座標跟著動，
// 而 `ecl6/25 A46Ch` 寫下 `4A9E = 255` 進區域圖之後，入口 0 的第一行
// `COMPARE @4A9E, 255 → EXIT` 讓整段野外處理不跑——`00FBh`／`00FCh` 因此停在
// 上一次野外移動留下的舊值，前端要是照抄回 `49C3`／`49C4`，隊伍就會被拉回
// 上一個野外位置。
func TestAnAreaMapStepLeavesTheWildernessPositionAlone(t *testing.T) {
	application := sailEastIntoTheWilderness(t)
	memory := application.eventMachine.Memory
	// 先在地形上走一步，讓 `00FBh`／`00FCh` 帶著「上一步」的值。
	application.spawn.Facing = 3
	if err := press(application, ebiten.KeyArrowUp); err != nil {
		t.Fatal(err)
	}
	drainWildernessEvents(application)
	if !application.inWildernessOverland() {
		t.Fatalf("走一步之後不在野外地形上（4A9E = %d）", memory[wildernessArea])
	}
	memory[wildernessArea] = 255
	if application.inWildernessOverland() {
		t.Fatal("4A9E = 255 之後還被當成野外地形")
	}
	before := [2]uint16{memory[wildernessX], memory[wildernessY]}
	// 把 `00FBh`／`00FCh` 換成一組看得出來的值：前端若照抄就會露餡。
	memory[wildernessNextX], memory[wildernessNextY] = 99, 99
	// 一定要真的走到一步才算數——牆擋住的話 `moveForward` 在寫回之前就返回，
	// 測試會因為「什麼都沒發生」而綠。
	moved := 0
	for facing := uint8(0); facing < 4; facing++ {
		application.spawn.Facing = facing
		cell := [2]uint8{application.spawn.X, application.spawn.Y}
		if err := press(application, ebiten.KeyArrowUp); err != nil {
			t.Fatal(err)
		}
		drainWildernessEvents(application)
		if [2]uint8{application.spawn.X, application.spawn.Y} != cell {
			moved++
		}
		if got := [2]uint16{memory[wildernessX], memory[wildernessY]}; got != before {
			t.Fatalf("區域圖裡朝向 %d 走一步，野外座標從 %v 變成 %v", facing, before, got)
		}
	}
	if moved == 0 {
		t.Fatal("四個方向都被牆擋住，這一條什麼都沒驗到")
	}
}

// drainWildernessEvents 把格子事件與選單按掉，讓下一次方向鍵真的送進移動。
func drainWildernessEvents(a *app) {
	for tick := 0; tick < 200 && (a.cellEventPending || a.cellWaitingMenu); tick++ {
		if a.cellWaitingMenu && len(a.cellMenuOptions) > 1 &&
			a.cellMenuCursor != len(a.cellMenuOptions)-1 {
			press(a, ebiten.KeyArrowRight)
			continue
		}
		press(a, ebiten.KeyEnter)
	}
}

// 野外 → 城西緣 → 區塊 18 → 北邊界 → 區塊 9。
//
// 這一條是 spec 101 那張缺口表裡「9→6→3→{4,5}→7 那條邊界鏈」的第一段，
// 而且是**在完整的前端底下走的**：從東邊的登陸點出發，往西跨到野外圖 26，
// 走到 (11,28)（城西緣，spec 105 的地點 4），選 NORTH 進區塊 18，再從
// GEO1/18 的北緣踏出去——`ecl1/18 99D4h` 的 `ON GOTO @C04D` 第 0 支就是
// `6E12 = 2`、`NEWECL 9`。
//
// 世界巡迴走不到這裡不是因為機制不通，是因為 34 趟裡只有兩趟到得了 GEO1/18，
// 而且都是走完野外之後才進去的（見 WORKLIST）。
func TestTheNorthEdgeOfBlockEighteenLeadsToBlockNine(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application := sailEastIntoTheWilderness(t)
	walk := newWildernessWalk(true, 0)
	for step := 0; step < 2000 && int(application.eventSession.CurrentBlockID()) != 26; step++ {
		if !application.inWildernessOverland() {
			t.Fatalf("中途離開野外，block %d", application.eventSession.CurrentBlockID())
		}
		key := ebiten.KeyArrowRight
		if want, ok := wildernessNextFacing(application, zipPath, walk); ok {
			key = wildernessTurnKey(application.spawn.Facing, want)
		}
		if err := press(application, key); err != nil {
			t.Fatal(err)
		}
		drainWildernessEvents(application)
	}
	memory := application.eventMachine.Memory
	// 走到城西緣 (11,28)，選單第一項是 NORTH。
	for step := 0; step < 400 && application.inWildernessOverland(); step++ {
		here := [2]int{int(memory[wildernessX]), int(memory[wildernessY])}
		if here == [2]int{11, 28} {
			break
		}
		route := wildernessRoute(application, here, [2]int{11, 28}, nil)
		if len(route) == 0 {
			t.Fatalf("從 (%d,%d) 走不到 (11,28)", here[0], here[1])
		}
		faceTowards(application, route[0])
		press(application, ebiten.KeyArrowUp)
		for tick := 0; tick < 200 &&
			(application.cellEventPending || application.cellWaitingMenu); tick++ {
			press(application, ebiten.KeyEnter)
		}
	}
	if got := int(application.eventSession.CurrentBlockID()); got != 18 {
		t.Fatalf("城西緣選 NORTH 之後是 block %d，要 18", got)
	}
	// GEO1/18 的北緣：(4,0) 與 (11,0)。走到其中一個再往北踏出去。
	reached := false
	for _, cell := range [][2]int{{4, 0}, {11, 0}} {
		plan := planToCells(application, 0, func(x, y int) bool {
			return x == cell[0] && y == cell[1]
		})
		if len(plan) == 0 {
			continue
		}
		for _, step := range plan {
			faceTowards(application, step.facing)
			press(application, ebiten.KeyArrowUp)
			drainWildernessEvents(application)
		}
		if [2]uint8{application.spawn.X, application.spawn.Y} !=
			[2]uint8{uint8(cell[0]), uint8(cell[1])} {
			continue
		}
		application.spawn.Facing = 0
		press(application, ebiten.KeyArrowUp)
		drainWildernessEvents(application)
		reached = true
		break
	}
	if !reached {
		t.Fatal("GEO1/18 的兩個北緣格都走不到")
	}
	if got := int(application.eventSession.CurrentBlockID()); got != 9 {
		t.Fatalf("從 GEO1/18 北緣踏出去之後是 block %d，要 9", got)
	}
	if got := application.spawn.Map.BlockID; got != 9 {
		t.Fatalf("地圖是 GEO%d/%d，要 GEO2/9", application.spawn.Map.Archive, got)
	}
}
