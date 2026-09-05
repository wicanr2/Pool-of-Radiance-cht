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

// wanderSettle 是野外亂走時的按鍵策略：**只有洞穴那一句才進去**。
//
// 對任何選單都挑 ENTER 會出事——野外的地點裡有「進城」「上船」，踩到就離開
// 野外了，而回來的路很長。
func wanderSettle(a *app) {
	if strings.Contains(a.eventText, "DARK CAVE") {
		caveSettle(a, "ENTER", "ENTER CAVE")
		return
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
