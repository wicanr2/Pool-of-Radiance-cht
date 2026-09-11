package main

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/golden-box-remake-engine/geometry"
)

// enterStojanowGate 把隊伍帶到 GEO2/9（斯托亞諾夫城門）的南半邊落點。
//
// 路線與 `TestTheNorthEdgeOfBlockEighteenLeadsToBlockNine` 同一條：搭船進野外
// → 走到 (11,28) 選 NORTH 進區塊 18 → 走出 GEO1/18 的北緣。
func enterStojanowGate(t *testing.T) *app {
	t.Helper()
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
		break
	}
	if got := int(application.eventSession.CurrentBlockID()); got != 9 {
		t.Fatalf("從 GEO1/18 北緣踏出去之後是 block %d，要 9", got)
	}
	return application
}

// castlePilot 是這一組測試共用的戰術地圖駕駛。
var castlePilot = &tacticalPilot{}

// dismissDoorMenu 把鎖住的門那個選單關掉（選 `EXIT`）。
//
// **門選單是 modal 的**（spec 122）：`a.door != nil` 的時候方向鍵與 ENTER 都歸
// 它，`spawn.Facing` 與 `cellMenuCursor` 一個都不會動。治具裡那些「轉到某個
// 朝向」「移到某一項」的迴圈因此永遠轉不出來——**症狀是整條測試逾時，看起來
// 像變慢而不是卡住**，所以走一步之後要先確認門沒開著。
//
// `EXIT` 一定在最後一格（`refreshDoorOptions` 無條件補上它），而游標是環狀的，
// 往左一下就到；四個選項最多兩下就關得掉，guard 給八下。
func dismissDoorMenu(a *app) {
	if a.door != nil && !a.cellEventPending {
		doorMenusDismissed++
	}
	// **`!a.cellEventPending` 是必要條件，不是防禦性的。** 門的分派排在
	// `cellEventPending` **後面**（`main.go`，理由見 spec 122），所以有格子
	// 事件在等的時候按鍵全歸事件，門的游標一下都不會動。少了這個條件，
	// 這支就在原地空轉，而症狀取決於 guard 給多少：給得小是「門沒關掉、
	// 朝向與事件狀態被打亂」（測試走不到該走到的區塊），給得大是「每個
	// tick 空轉幾十次 press」（整條測試逾時）。**兩種都不會說是順序錯了。**
	for guard := 0; a.door != nil && !a.cellEventPending && guard < 32; guard++ {
		if want := len(a.door.Options) - 1; a.door.Cursor != want {
			press(a, ebiten.KeyArrowLeft)
			continue
		}
		press(a, ebiten.KeyEnter)
	}
}

// doorMenusDismissed／faceTowardsMaxTurns 是這一批治具的診斷計數（**只在測試裡用**）。
// 門到底有沒有被撞到、轉向到底用掉幾下，都要用數字回答——2026-09-11 查
// 「城堡走不到區塊 7」的時候，靠推論排過三輪假設都是錯的。
var (
	doorMenusDismissed int
	faceTowardsMaxTurns int
)

// faceTowards 轉到指定朝向。先關掉門選單，理由見 dismissDoorMenu。
//
// guard 的用意是**讓迴圈有終點**，不是限制轉幾次。正常轉向最多三下，但這一支
// 也會在「有格子事件擋著」的時候被呼叫——那時右鍵不轉向，要先把事件按過去，
// 次數沒有上限可推。給得太緊的話迴圈會在朝向還沒轉到的時候就走人，接著那一步
// 就往錯的方向走，症狀是覆蓋少了一塊（實測 guard 8 會讓城堡那條測試走不到
// ECL 區塊 7），而不是任何一條斷言直接指出朝向錯了。
func faceTowards(a *app, facing uint8) {
	dismissDoorMenu(a)
	turns := 0
	for ; a.spawn.Facing != facing && turns < 256; turns++ {
		press(a, ebiten.KeyArrowRight)
	}
	if turns > faceTowardsMaxTurns {
		faceTowardsMaxTurns = turns
	}
}

// answerCellMenus 把等待中的事件按完，選單挑 want 裡認得的那一項。
func answerCellMenus(a *app, want ...string) {
	for tick := 0; tick < 3000; tick++ {
		busy := a.door != nil || a.cellWaitingMenu || a.cellEventPending || a.encounter != nil ||
			a.treasureActive || a.tactical != nil
		if !busy {
			return
		}
		// 門排在格子事件**後面**，與 `main.go` 的分派同一個順序。順序反過來
		// 的話，有事件在等的時候這裡會一直試著關門而事件永遠按不完。
		if a.door != nil && !a.cellEventPending {
			dismissDoorMenu(a)
			continue
		}
		if a.tactical != nil {
			// 只按 Enter 的話全隊都不出手，那一場永遠結束不了——用探索器
			// 那個「自己人怎麼打」的駕駛（coverage_test.go）。
			press(a, castlePilot.key(a))
			continue
		}
		if a.cellWaitingMenu && len(a.cellMenuOptions) != 0 {
			pick := 0
			for index, option := range a.cellMenuOptions {
				for _, label := range want {
					if strings.EqualFold(option, label) {
						pick = index
					}
				}
			}
			for guard := 0; a.cellMenuCursor != pick && guard < 256; guard++ {
				press(a, ebiten.KeyArrowDown)
			}
		}
		press(a, ebiten.KeyEnter)
	}
	// 上限用完就放棄這一次，不是錯誤：像神殿那種「選了也買不到」的選單
	// （`want` 裡沒有它認得的項，於是一直選第一項）本來就不會收斂，而外層
	// 的輪替會帶著隊伍走開。**診斷期間可以把這裡換成帶狀態的 panic**，
	// 2026-09-11 查門選單就是那樣把卡住的狀態問出來的——但收工要換回來，
	// 否則既有的容忍會變成硬失敗。
}

// 付過路費過斯托亞諾夫城門，然後走北緣進城堡。
//
// 原版的路（`ecl2/9`，spec 101）：地形 9 的格子拿到商人的馬車（遭遇選單選
// PARLAY 花 250 金買、選 COMBAT 就殺了他搶走，兩條都是 `B014 OR @4A77 64`），
// 地形 5 的格子付 15 金（`AC4Bh`「THE BUGBEAR TAKES THE MONEY AND THE GATE
// OPENS.」），腳本把隊伍寫到 (8,5)——南北兩半在 GEO 上是不通的，這一段腳本
// 就是唯一的通道。過去之後北緣 (4,0)／(11,0) 往北踏出去就換到城堡。
//
// 這個測試同時釘住 `consumeInitialTransitionResources` 要累積 `Writes`：
// 那三個 `SAVE` 夾在 `PICTURE 255` 與 `CALL 2C90h` 之間，把 result 換掉就等於
// 把傳送丟掉，症狀是「記憶體裡座標對了，隊伍站著不動」。
// 負對照做過：不累積 `Writes` 的時候，付完錢隊伍留在 (6,11)，而
// `C04B`／`C04C` 已經是 8／5。
func TestPayingTheTollAtStojanowGateOpensTheCastle(t *testing.T) {
	application := enterStojanowGate(t)
	terrain := func(x, y int) int {
		return int(application.initialMap.Grid.Cells[y][x].Terrain) & 0x1F
	}
	// 地形 3 是熊地精衛兵：踩到就把 `@4A78` 推到 3，之後索賄那一支必翻臉。
	guards := func(x, y int) bool { return terrain(x, y) == 3 }
	walkTo := func(what string, wanted func(x, y int) bool) {
		plan := planToCellsAvoiding(application, 0, wanted, guards)
		if len(plan) == 0 {
			t.Fatalf("走不到%s（現在 GEO%d/%d (%d,%d)）", what,
				application.spawn.Map.Archive, application.spawn.Map.BlockID,
				application.spawn.X, application.spawn.Y)
		}
		for _, step := range plan {
			faceTowards(application, step.facing)
			press(application, ebiten.KeyArrowUp)
			answerCellMenus(application, "YES")
		}
	}
	walkTo("馬車格", func(x, y int) bool { return terrain(x, y) == 9 })
	if got := application.eventMachine.Memory[0x4A77] & 64; got == 0 {
		t.Fatalf("站上馬車格之後 `@4A77` 的第 6 位沒設起來（%d），馬車沒買到",
			application.eventMachine.Memory[0x4A77])
	}
	walkTo("索賄格", func(x, y int) bool { return terrain(x, y) == 5 })
	if got := [2]int{int(application.spawn.X), int(application.spawn.Y)}; got != [2]int{8, 5} {
		t.Fatalf("付了過路費之後隊伍在 (%d,%d)，原版腳本要把人放到 (8,5)",
			got[0], got[1])
	}
	// 北半邊：走到北緣再往北踏出去。
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
			answerCellMenus(application, "YES")
		}
		if [2]int{int(application.spawn.X), int(application.spawn.Y)} != cell {
			continue
		}
		application.spawn.Facing = 0
		press(application, ebiten.KeyArrowUp)
		answerCellMenus(application, "YES")
		reached = true
		break
	}
	if !reached {
		t.Fatal("過了城門卻走不到北緣的兩格")
	}
	block := int(application.eventSession.CurrentBlockID())
	if block == 9 {
		t.Fatalf("往北踏出去還在區塊 9，位置 (%d,%d)",
			application.spawn.X, application.spawn.Y)
	}
	t.Logf("走進城堡：ECL block %d，GEO%d/%d (%d,%d)", block,
		application.spawn.Map.Archive, application.spawn.Map.BlockID,
		application.spawn.X, application.spawn.Y)
	if application.spawn.X >= geometry.Width || application.spawn.Y >= geometry.Height {
		t.Fatalf("換圖之後的座標超出範圍：(%d,%d)", application.spawn.X, application.spawn.Y)
	}
}

// stojanowGateBlockID 是斯托亞諾夫城門的 ECL 區塊編號。
const stojanowGateBlockID = 9

// walkStojanowGateIntoTheCastle 走完整條城門的路，回傳進到城堡之後的 app。
// 走不成就 Skip——那條路本身由上面那個測試釘住。
func walkStojanowGateIntoTheCastle(t *testing.T) *app {
	t.Helper()
	application := enterStojanowGate(t)
	terrain := func(x, y int) int {
		return int(application.initialMap.Grid.Cells[y][x].Terrain) & 0x1F
	}
	guards := func(x, y int) bool { return terrain(x, y) == 3 }
	walkTo := func(wanted func(x, y int) bool) bool {
		plan := planToCellsAvoiding(application, 0, wanted, guards)
		if len(plan) == 0 {
			return false
		}
		for _, step := range plan {
			faceTowards(application, step.facing)
			press(application, ebiten.KeyArrowUp)
			answerCellMenus(application, "YES")
		}
		return true
	}
	if !walkTo(func(x, y int) bool { return terrain(x, y) == 9 }) {
		t.Skip("走不到馬車格")
	}
	if !walkTo(func(x, y int) bool { return terrain(x, y) == 5 }) {
		t.Skip("走不到索賄格")
	}
	if [2]int{int(application.spawn.X), int(application.spawn.Y)} != [2]int{8, 5} {
		t.Skip("過路費沒付成")
	}
	for _, cell := range [][2]int{{4, 0}, {11, 0}} {
		if !walkTo(func(x, y int) bool { return x == cell[0] && y == cell[1] }) {
			continue
		}
		if [2]int{int(application.spawn.X), int(application.spawn.Y)} != cell {
			continue
		}
		application.spawn.Facing = 0
		press(application, ebiten.KeyArrowUp)
		answerCellMenus(application, "YES")
		break
	}
	if int(application.eventSession.CurrentBlockID()) == stojanowGateBlockID {
		t.Skip("沒走出北緣")
	}
	return application
}

// 量城門後面走得到多少東西。
//
// 世界巡迴到現在還走不到這裡：它會先去瓦海登墳場，那張圖的
// `ecl4/10 9AD6 ADD 1 @4A00` 把城門的馬車關掉了（class 0 是整份存檔共用的，
// spec 106）。所以這條路要單獨走。
//
// 走法是「逐格、四個朝向各踏一次」——城堡這幾張圖的出口是**朝向敏感**的
// （`ecl5/3` 的入口 0 用 `GETTABLE @9AA6[@C04D]`，東邊出去是區塊 4、南邊是
// 區塊 6；院子裡 (4,8) 朝北是上樓進區塊 5），光是踩過每一格挑不出來。
func TestTheCastleBehindStojanowGateHasContent(t *testing.T) {
	blocks := map[int]bool{}
	maps := map[string]bool{}
	record := func(a *app) {
		blocks[int(a.eventSession.CurrentBlockID())] = true
		maps[fmt.Sprintf("GEO%d/%d", a.spawn.Map.Archive, a.spawn.Map.BlockID)] = true
	}
	// 四個方向輪替起點各掃一趟：規劃器先往哪個方向走，決定先撞到哪一個出口，
	// 而出口一走就換圖，後面那一段就看不到了。聯集才是這條路後面的全貌。
	for _, rotate := range []int{0, 1, 2, 3} {
		application := walkStojanowGateIntoTheCastle(t)
		tried := map[[3]int]bool{}
		last := -1
		for round := 0; round < 3000; round++ {
			record(application)
			if id := int(application.eventSession.CurrentBlockID()); id != last {
				last = id
				t.Logf("輪替 %d 第 %d 圈：ECL block %d，GEO%d/%d (%d,%d)", rotate, round, id,
					application.spawn.Map.Archive, application.spawn.Map.BlockID,
					application.spawn.X, application.spawn.Y)
			}
			key := func(x, y int) [3]int {
				return [3]int{int(application.spawn.Map.Archive),
					int(application.spawn.Map.BlockID), y*100 + x}
			}
			plan := planToCells(application, rotate, func(x, y int) bool {
				return !tried[key(x, y)]
			})
			if len(plan) == 0 {
				break
			}
			for _, step := range plan {
				faceTowards(application, step.facing)
				press(application, ebiten.KeyArrowUp)
				answerCellMenus(application, "YES")
			}
			here := [2]int{int(application.spawn.X), int(application.spawn.Y)}
			tried[key(here[0], here[1])] = true
			for turn := 0; turn < 4; turn++ {
				application.spawn.Facing = uint8(turn)
				press(application, ebiten.KeyArrowUp)
				answerCellMenus(application, "YES")
				record(application)
				if [2]int{int(application.spawn.X), int(application.spawn.Y)} != here {
					break
				}
			}
		}
	}
	// 院子南緣那一個出口深掃時碰不到（先撞到別的出口就換圖了），單獨走一次。
	application := walkStojanowGateIntoTheCastle(t)
	plan := planToCells(application, 0, func(x, y int) bool { return x == 11 && y == 15 })
	for _, step := range plan {
		faceTowards(application, step.facing)
		press(application, ebiten.KeyArrowUp)
		answerCellMenus(application, "YES")
	}
	if [2]int{int(application.spawn.X), int(application.spawn.Y)} == [2]int{11, 15} {
		application.spawn.Facing = 2
		press(application, ebiten.KeyArrowUp)
		answerCellMenus(application, "YES")
		record(application)
		t.Logf("院子南緣 (11,15) 朝南：ECL block %d，GEO%d/%d (%d,%d)",
			application.eventSession.CurrentBlockID(),
			application.spawn.Map.Archive, application.spawn.Map.BlockID,
			application.spawn.X, application.spawn.Y)
	}
	names := make([]string, 0, len(maps))
	for name := range maps {
		names = append(names, name)
	}
	sort.Strings(names)
	ids := make([]int, 0, len(blocks))
	for id := range blocks {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	t.Logf("城門後面走到的地圖 %d 張：%v", len(names), names)
	t.Logf("城門後面走到的 ECL block %d 個：%v", len(ids), ids)
	t.Logf("診斷：關掉門選單 %d 次、轉向最多用掉 %d 下",
		doorMenusDismissed, faceTowardsMaxTurns)
	for _, want := range []int{3, 4, 5, 6, 7} {
		if !blocks[want] {
			t.Errorf("沒走到區塊 %d（只有 %v）", want, ids)
		}
	}
}
