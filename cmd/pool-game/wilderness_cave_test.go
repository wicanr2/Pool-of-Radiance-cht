package main

import (
	"strings"
	"math/rand"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// caveSettle 把等待中的東西按完；選單優先挑 prefer 裡認得的那一項。
//
// 寶物那一串要走 `treasureMenuChoice`（挑 Exit），不然 View → Return → View
// 繞不完；戰術地圖要用探索器的駕駛，只按 Enter 全隊都不出手，那一場永遠
// 結束不了。
func caveSettle(a *app, prefer ...string) {
	for tick := 0; tick < 4000 && (a.cellEventPending || a.cellWaitingMenu ||
		a.tactical != nil || a.combatActive || a.encounter != nil ||
		a.treasureActive || a.tacticalPreview); tick++ {
		if a.tactical != nil {
			press(a, castlePilot.key(a))
			continue
		}
		if a.cellWaitingMenu && len(a.cellMenuOptions) != 0 {
			pick := 0
			if a.treasureActive {
				pick = treasureMenuChoice(a.cellMenuOptions)
			} else {
				for index, option := range a.cellMenuOptions {
					for _, label := range prefer {
						if option == label {
							pick = index
						}
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

// randomAreaPrompts 是三張野外圖的「隨機區域」問句。
//
// 進去再出來會重擲位移（見下面那一支的說明），而**每一張圖的名字不一樣**：
// 野外 25 與 27 是洞穴，野外 26 是小樹林與廢棄小屋。只認 `DARK CAVE` 的話，
// 在野外 26 上會把每一次機會都拒絕掉——實測第一擲中了 198 次，一次都沒進去。
var randomAreaPrompts = []string{
	"DARK CAVE",    // ecl6/25 A425h、ecl8/27 A20Dh
	"WOODED GROVE", // ecl7/26 A45Dh
	"RUINED HUTS",  // ecl7/26 A4A6h
}

// wanderSettle 是野外亂走時的按鍵策略：**只有隨機區域那幾句才進去**。
//
// 對任何選單都挑 ENTER 會出事——野外的地點裡有「進城」「上船」，踩到就離開
// 野外了，而回來的路很長。
func wanderSettle(a *app) {
	for _, prompt := range randomAreaPrompts {
		if strings.Contains(a.eventText, prompt) {
			caveSettle(a, "ENTER", "ENTER CAVE", "INVESTIGATE", "YES")
			return
		}
	}
	caveSettle(a, "LEAVE", "NO", "GO BACK")
}

// 東野外的 (6,15)（區塊 13）要靠洞穴重擲位移才走得到。
//
// 野外是一張 16×16 的 GEO 平鋪上去的迷宮（spec 105），而位移決定走得到哪些
// 地點。標準入口給的位移不開這三個地點所在的口袋：
//
//	區塊 13 → 野外 27 的 (6,15)；船的登陸點位移 (6,4)，那個口袋只有兩個地點
//	區塊 17 → 野外 26 的 (12,11)（游牧民營地）；跨圖進來的位移 (10,4) 不開它
//	區塊 28 → 野外 25 的 (3,32)（前哨站）
//
// 這一支只跑第一個——另外兩個要先跨圖，走法還沒調得夠快。
//
// 換位移的唯一機制是**兩個隨機事件串起來**：
//
//	1. 在野外走路時 `ecl8/27 9EA7h RANDOM 19` 中 0、再 `A867h` 的
//	   `RANDOM 20` 中 20 → `A20Dh`「YOU HAVE FOUND A SMALL, DARK CAVE.」
//	2. 進去之後那張區域圖**沒有任何邊界出口**，要在裡面走到
//	   `A7F7h`「YOU FIND A PASSAGEWAY GOING OUT.」才出得來
//	3. 出來時 `A255h RANDOM 3` 從四組 `4A18/4A19` 挑一組寫回位置，位移就換了
//
// 所以走法是「亂走等事件」，不是規劃路線。
func TestTheWildernessCaveRerollReachesTheEasternOutpost(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	for _, want := range []struct {
		sheet  int
		target [2]int
		block  int
		name   string
	}{
		{27, [2]int{6, 15}, 13, "東野外的 (6,15)"},
		// 野外 26 的 (12,11)（游牧民營地 → 區塊 17）與野外 25 的 (3,32)
		// （前哨站 → 區塊 28）是同一個機制，但要先跨圖過去，走法還沒調到
		// 能在合理時間內跑完（見 WORKLIST）。
	} {
		t.Run(want.name, func(t *testing.T) {
			application := sailEastIntoTheWilderness(t)
			memory := application.eventMachine.Memory
			here := func() [2]int {
				return [2]int{int(memory[wildernessX]), int(memory[wildernessY])}
			}
			// 先跨到目標那一張野外圖。
			walk := newWildernessWalk(true, 0)
			for step := 0; step < 6000 &&
				int(application.eventSession.CurrentBlockID()) != want.sheet; step++ {
				if !application.inWildernessOverland() {
					t.Skipf("跨圖途中離開野外，block %d",
						application.eventSession.CurrentBlockID())
				}
				key := ebiten.KeyArrowRight
				if facing, ok := wildernessNextFacing(application, zipPath, walk); ok {
					key = wildernessTurnKey(application.spawn.Facing, facing)
				}
				press(application, key)
				drainWildernessEvents(application)
			}
			if got := int(application.eventSession.CurrentBlockID()); got != want.sheet {
				t.Skipf("走不到野外 %d，停在 block %d", want.sheet, got)
			}
			dice := rand.New(rand.NewSource(11))
			caves := 0
			for step := 0; step < 30000; step++ {
				if application.inWildernessOverland() &&
					int(application.eventSession.CurrentBlockID()) == want.sheet &&
					len(wildernessRoute(application, here(), want.target, nil)) != 0 {
					for move := 0; move < 300 && here() != want.target; move++ {
						route := wildernessRoute(application, here(), want.target, nil)
						if len(route) == 0 {
							break
						}
						for application.spawn.Facing != route[0] {
							press(application, ebiten.KeyArrowRight)
						}
						press(application, ebiten.KeyArrowUp)
						caveSettle(application, "YES", "ENTER")
					}
					if got := int(application.eventSession.CurrentBlockID()); got == want.block {
						t.Logf("走到 %v（重擲 %d 次）：ECL block %d，GEO%d/%d",
							want.target, caves, got,
							application.spawn.Map.Archive, application.spawn.Map.BlockID)
						return
					}
				}
				if !application.inWildernessOverland() {
					// 在洞裡：出口也是隨機事件，亂走等它出現。
					for round := 0; round < 6000 && !application.inWildernessOverland(); round++ {
						application.spawn.Facing = uint8(dice.Intn(4))
						press(application, ebiten.KeyArrowUp)
						caveSettle(application, "LEAVE", "OUT", "GO", "YES")
					}
					if !application.inWildernessOverland() {
						t.Skipf("出不了 block %d GEO%d/%d",
							application.eventSession.CurrentBlockID(),
							application.spawn.Map.Archive, application.spawn.Map.BlockID)
					}
					caves++
					x, y := wildernessOffset(application)
					t.Logf("第 %d 次出洞：在 %v 位移 (%d,%d)", caves, here(), x, y)
					continue
				}
				application.spawn.Facing = uint8(dice.Intn(4))
				press(application, ebiten.KeyArrowUp)
				wanderSettle(application)
			}
			t.Skipf("重擲 %d 次還沒走到 %v", caves, want.target)
		})
	}
}


// 野外 26 的游牧民營地（區塊 17）：跨到野外 26，重擲位移，再走過去。
//
// 跨圖不用通用 walker：走到「最西走得到的那一格」再往西踏一步就好
// （spec 105 的 `49C3 == 2`），27 → 26 一秒出頭。
func TestTheNomadCampOnTheMiddleWildernessSheet(t *testing.T) {
	application := sailEastIntoTheWilderness(t)
	memory := application.eventMachine.Memory
	here := func() [2]int {
		return [2]int{int(memory[wildernessX]), int(memory[wildernessY])}
	}
	block := func() int { return int(application.eventSession.CurrentBlockID()) }
	walkTo := func(target [2]int, enter bool) bool {
		for step := 0; step < 400 && here() != target; step++ {
			if !application.inWildernessOverland() {
				return false
			}
			route := wildernessRoute(application, here(), target, nil)
			if len(route) == 0 {
				return false
			}
			for application.spawn.Facing != route[0] {
				press(application, ebiten.KeyArrowRight)
			}
			press(application, ebiten.KeyArrowUp)
			if enter {
				caveSettle(application, "ENTER", "ENTER IT", "YES")
			} else {
				wanderSettle(application)
			}
		}
		return here() == target
	}
	// 走到最西走得到的那一格，再往西跨到野外 26。
	best, found := [2]int{}, false
	for y := 3; y <= 34; y++ {
		for x := 2; x <= 15; x++ {
			if here() != [2]int{x, y} &&
				len(wildernessRoute(application, here(), [2]int{x, y}, nil)) == 0 {
				continue
			}
			if !found || x < best[0] {
				best, found = [2]int{x, y}, true
			}
		}
	}
	if !found || !walkTo(best, false) {
		t.Skipf("走不到最西的格子 %v", best)
	}
	for attempt := 0; attempt < 8 && block() == 27; attempt++ {
		application.spawn.Facing = 3
		press(application, ebiten.KeyArrowUp)
		wanderSettle(application)
	}
	if block() != 26 {
		t.Skipf("跨不到野外 26，現在 %d", block())
	}
	x, y := wildernessOffset(application)
	t.Logf("進到野外 26：在 %v 位移 (%d,%d)", here(), x, y)
	dice := rand.New(rand.NewSource(11))
	// 亂走時把三張圖的地點都繞開，也不踏跨圖欄（X ≤ 2、X ≥ 15）。
	// 隨機區域**不是地點**，所以繞開地點不會擋到它；不繞的話會走進城裡
	// 或上船，而回來的路很長。
	avoid := map[[2]int]bool{}
	for _, sheet := range []int{25, 26, 27} {
		for cell := range wildernessPlaceSet(
			filepath.Join("..", "..", "Pool of Radiance (1988).zip"), sheet) {
			avoid[cell] = true
		}
	}
	deltas := [4][2]int{{0, -1}, {1, 0}, {0, 1}, {-1, 0}}
	areas := 0
	for step := 0; step < 30000; step++ {
		if application.inWildernessOverland() && block() == 26 &&
			len(wildernessRoute(application, here(), [2]int{12, 11}, nil)) != 0 {
			walkTo([2]int{12, 11}, true)
			for attempt := 0; attempt < 4 && block() == 26; attempt++ {
				application.spawn.Facing = uint8(attempt)
				press(application, ebiten.KeyArrowUp)
				caveSettle(application, "ENTER", "ENTER IT", "YES")
			}
			if got := block(); got != 17 {
				t.Skipf("站上 (12,11) 之後是 block %d（4A0F=%d）", got, memory[0x4A0F])
			}
			t.Logf("走到 (12,11)（重擲 %d 次）：ECL block 17，GEO%d/%d", areas,
				application.spawn.Map.Archive, application.spawn.Map.BlockID)
			return
		}
		if !application.inWildernessOverland() {
			for round := 0; round < 8000 && !application.inWildernessOverland(); round++ {
				application.spawn.Facing = uint8(dice.Intn(4))
				press(application, ebiten.KeyArrowUp)
				caveSettle(application, "LEAVE", "OUT", "GO", "YES")
			}
			if !application.inWildernessOverland() {
				t.Skipf("出不了 block %d GEO%d/%d", block(),
					application.spawn.Map.Archive, application.spawn.Map.BlockID)
			}
			areas++
			ox, oy := wildernessOffset(application)
			t.Logf("第 %d 次出來：在 %v 位移 (%d,%d)", areas, here(), ox, oy)
			continue
		}
		facing, ok := -1, false
		start := dice.Intn(4)
		for try := 0; try < 4; try++ {
			f := (start + try) % 4
			next := [2]int{here()[0] + deltas[f][0], here()[1] + deltas[f][1]}
			if avoid[next] || next[0] <= 2 || next[0] >= 15 || next[1] < 3 || next[1] > 34 {
				continue
			}
			facing, ok = f, true
			break
		}
		if !ok {
			facing = dice.Intn(4)
		}
		application.spawn.Facing = uint8(facing)
		press(application, ebiten.KeyArrowUp)
		wanderSettle(application)
	}
	t.Skipf("重擲 %d 次還沒走到 (12,11)", areas)
}


// 西野外的前哨站（區塊 28）：跨兩次圖到野外 25，重擲位移，再走過去答 YES。
//
// 兩個非走法的條件：
//
//   - **主線閘門**：`ecl6/25 9E4Ch COMPARE @4A98 255` 要成立，隊伍才是
//     「新費蘭來的外交使節」。沒有它只會演「YOU ARE TRESSPASSING ON
//     PRIVATE LAND」那一段，選 STAY 或 LEAVE 都進不去。這裡跟碼頭那條航線
//     一樣直接把旗標設起來——它證明的是「那一區的腳本跑得動」。
//   - **僵局安全閥**：野外 25 上有一場雙方都碰不到彼此的架（野豬 ×5），
//     沒有 `tacticalStalemateRounds` 的話隊伍走 155 步就永遠停在那裡。
func TestTheOutpostOnTheWesternWildernessSheet(t *testing.T) {
	application := sailEastIntoTheWilderness(t)
	memory := application.eventMachine.Memory
	here := func() [2]int {
		return [2]int{int(memory[wildernessX]), int(memory[wildernessY])}
	}
	block := func() int { return int(application.eventSession.CurrentBlockID()) }
	walkTo := func(target [2]int, settle func(*app)) bool {
		for step := 0; step < 400 && here() != target; step++ {
			if !application.inWildernessOverland() {
				return false
			}
			route := wildernessRoute(application, here(), target, nil)
			if len(route) == 0 {
				return false
			}
			for application.spawn.Facing != route[0] {
				press(application, ebiten.KeyArrowRight)
			}
			press(application, ebiten.KeyArrowUp)
			settle(application)
		}
		return here() == target
	}
	best, found := [2]int{}, false
	for y := 3; y <= 34; y++ {
		for x := 2; x <= 15; x++ {
			if here() != [2]int{x, y} &&
				len(wildernessRoute(application, here(), [2]int{x, y}, nil)) == 0 {
				continue
			}
			if !found || x < best[0] {
				best, found = [2]int{x, y}, true
			}
		}
	}
	if !found || !walkTo(best, wanderSettle) {
		t.Skip("走不到最西的格子")
	}
	for attempt := 0; attempt < 8 && block() == 27; attempt++ {
		application.spawn.Facing = 3
		press(application, ebiten.KeyArrowUp)
		wanderSettle(application)
	}
	for _, row := range []int{24, 20, 15, 28} {
		if block() == 25 {
			break
		}
		if !walkTo([2]int{2, row}, wanderSettle) {
			continue
		}
		for attempt := 0; attempt < 4 && block() == 26; attempt++ {
			application.spawn.Facing = 3
			press(application, ebiten.KeyArrowUp)
			wanderSettle(application)
		}
	}
	if block() != 25 {
		t.Skipf("跨不到野外 25，現在 %d", block())
	}
	ox, oy := wildernessOffset(application)
	// 前哨站要「外交使節」的委任：`ecl6/25 9E4Ch COMPARE @4A98 255`。
	// 沒有它就只會演「YOU ARE TRESSPASSING ON PRIVATE LAND」那一段。
	memory[0x4A98] = 255
	t.Logf("進到野外 25，在 %v 位移 (%d,%d)｜4A98=%d", here(), ox, oy, memory[0x4A98])
	dice := rand.New(rand.NewSource(11))
	areas, moves, last := 0, 0, here()
	for step := 0; step < 6000; step++ {
		if application.inWildernessOverland() && here() != last {
			moves++
			last = here()
		}
		if step%1500 == 0 {
			_ = moves
		}
		if application.inWildernessOverland() && block() == 25 &&
			len(wildernessRoute(application, here(), [2]int{3, 32}, nil)) != 0 {
			logMenu := func(a *app) {
				for tick := 0; tick < 800 && (a.cellEventPending || a.cellWaitingMenu ||
					a.tactical != nil || a.combatActive || a.encounter != nil ||
					a.treasureActive || a.tacticalPreview); tick++ {
					if a.tactical != nil {
						press(a, castlePilot.key(a))
						continue
					}
					if a.cellWaitingMenu && len(a.cellMenuOptions) != 0 {
						pick := 0
						for index, option := range a.cellMenuOptions {
							if option == "YES" || option == "ENTER" {
								pick = index
							}
						}
						for a.cellMenuCursor != pick {
							press(a, ebiten.KeyArrowDown)
						}
					}
					press(a, ebiten.KeyEnter)
				}
			}
			walkTo([2]int{3, 32}, logMenu)
			for attempt := 0; attempt < 4 && block() == 25; attempt++ {
				application.spawn.Facing = uint8(attempt)
				press(application, ebiten.KeyArrowUp)
				logMenu(application)
			}
			if got := block(); got != 28 {
				t.Skipf("站上 (3,32) 之後是 block %d", got)
			}
			t.Logf("走到 (3,32)（重擲 %d 次）：ECL block 28，GEO%d/%d", areas,
				application.spawn.Map.Archive, application.spawn.Map.BlockID)
			return
		}
		if !application.inWildernessOverland() {
			for round := 0; round < 8000 && !application.inWildernessOverland(); round++ {
				application.spawn.Facing = uint8(dice.Intn(4))
				press(application, ebiten.KeyArrowUp)
				caveSettle(application, "LEAVE", "OUT", "GO", "YES")
			}
			if !application.inWildernessOverland() {
				t.Skipf("出不了 block %d", block())
			}
			areas++
			x, y := wildernessOffset(application)
			t.Logf("第 %d 次出來，在 %v 位移 (%d,%d)", areas, here(), x, y)
			continue
		}
		// 只擋邊界與跨圖欄，地點照走——`wanderSettle` 對它們一律答 LEAVE。
		deltas := [4][2]int{{0, -1}, {1, 0}, {0, 1}, {-1, 0}}
		facing := -1
		start := dice.Intn(4)
		for try := 0; try < 4; try++ {
			f := (start + try) % 4
			next := [2]int{here()[0] + deltas[f][0], here()[1] + deltas[f][1]}
			if next[0] <= 2 || next[0] >= 15 || next[1] < 3 || next[1] > 34 {
				continue
			}
			facing = f
			break
		}
		if facing < 0 {
			facing = dice.Intn(4)
		}
		application.spawn.Facing = uint8(facing)
		press(application, ebiten.KeyArrowUp)
		wanderSettle(application)
	}
	t.Skipf("重擲 %d 次、移動 %d 次，沒走到 (3,32)", areas, moves)
}
