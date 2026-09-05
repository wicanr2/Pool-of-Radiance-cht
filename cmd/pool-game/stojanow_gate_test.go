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
		for application.spawn.Facing != route[0] {
			press(application, ebiten.KeyArrowRight)
		}
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
			for application.spawn.Facing != step.facing {
				press(application, ebiten.KeyArrowRight)
			}
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

// answerCellMenus 把等待中的事件按完，選單挑 want 裡認得的那一項。
func answerCellMenus(a *app, want ...string) {
	for tick := 0; tick < 3000; tick++ {
		busy := a.cellWaitingMenu || a.cellEventPending || a.encounter != nil ||
			a.treasureActive || a.tactical != nil
		if !busy {
			return
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
			for a.cellMenuCursor != pick {
				press(a, ebiten.KeyArrowDown)
			}
		}
		press(a, ebiten.KeyEnter)
	}
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
			for application.spawn.Facing != step.facing {
				press(application, ebiten.KeyArrowRight)
			}
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
			for application.spawn.Facing != step.facing {
				press(application, ebiten.KeyArrowRight)
			}
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
			for application.spawn.Facing != step.facing {
				press(application, ebiten.KeyArrowRight)
			}
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
// 這不是門檻測試，是量測：城堡那幾個區塊（3、4、5、6、7）先前一個都沒走到，
// 而世界巡迴到現在也還走不到——巡迴會先去瓦海登墳場，那張圖的
// `ecl4/10 9AD6 ADD 1 @4A00` 把馬車關掉了（class 0 是整份存檔共用的，
// spec 106）。所以這條路要單獨走。
func TestTheCastleBehindStojanowGateHasContent(t *testing.T) {
	blocks := map[int]bool{}
	maps := map[string]bool{}
	// 四種走法取聯集：繞不繞開換圖點、規劃器的方向輪替起點。單一種走法量到的
	// 是那一種走法的下限，聯集才是「這條路後面接得到什麼」。
	for _, pass := range []struct {
		banPass bool
		rotate  int
	}{{true, 0}, {false, 0}, {true, 1}, {false, 1}} {
		application := walkStojanowGateIntoTheCastle(t)
		walked := map[[3]int]bool{}
		avoid := map[[3]int]bool{}
		if pass.banPass {
			// 繞開換圖點是巡迴的作法：踩到就被換走，走不完一張圖。
			for _, key := range boundaryExitKeys(application) {
				avoid[key] = true
			}
		}
		lastBlock := -1
		for step := 0; step < 20000; step++ {
			if application.eventSession != nil {
				id := int(application.eventSession.CurrentBlockID())
				blocks[id] = true
				if id != lastBlock {
					lastBlock = id
					t.Logf("擋換圖 %v 輪替 %d 第 %d 步：ECL block %d，GEO%d/%d (%d,%d)",
						pass.banPass, pass.rotate, step, id,
						application.spawn.Map.Archive, application.spawn.Map.BlockID,
						application.spawn.X, application.spawn.Y)
				}
			}
			if application.initialMap != nil {
				maps[fmt.Sprintf("GEO%d/%d",
					application.spawn.Map.Archive, application.spawn.Map.BlockID)] = true
			}
			busy := application.cellWaitingMenu || application.cellEventPending ||
				application.encounter != nil || application.treasureActive ||
				application.tactical != nil || application.combatActive ||
				application.shopActive || application.templeActive
			if busy {
				answerCellMenus(application, "YES")
				continue
			}
			plan, target := explorePlan(application, walked, avoid, pass.banPass, pass.rotate)
			if len(plan) == 0 {
				break
			}
			walked[target] = true
			for _, s := range plan {
				for application.spawn.Facing != s.facing {
					press(application, ebiten.KeyArrowRight)
				}
				press(application, ebiten.KeyArrowUp)
				if application.cellWaitingMenu || application.cellEventPending ||
					application.tactical != nil {
					break
				}
			}
		}
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
	if len(ids) < 2 {
		t.Fatalf("只走到 %v，至少要有區塊 3 與 5", ids)
	}
}
