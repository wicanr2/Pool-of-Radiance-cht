package main

import (
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	"github.com/wicanr2/golden-box-remake-engine/geometry"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
	pooltreasure "github.com/wicanr2/Pool-of-Radiance-cht/internal/treasure"
)

// innCells 是菲蘭城區（GEO3/0）的旅店門口。地形索引 9 的七格（spec 102），
// 腳本在 `ecl3/0` 的 `A140h`：問「一枚白金住一晚」，答應就扣錢再推
// `38h PROGRAM` 的值 9。
var innCells = map[[2]int]bool{
	{4, 12}: true, {6, 12}: true, {4, 13}: true, {6, 13}: true,
	{0, 14}: true, {1, 14}: true, {2, 14}: true,
}

// 旅店的過夜：走進去、付一枚白金、開紮營、休息回滿。
//
// `PROGRAM 9` 全遊戲只有這一處（`ecl3/0 A1ADh`），而它的派發鏈第一支是
// overlay-15 entry 1——記憶法術那一支（spec 070／081）。先前 remake 把它
// 當成「問一句再開隊伍管理」，玩家付了白金什麼都沒得到。
func TestTheInnChargesAPlatinumAndOpensCamp(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	money := [7]uint16{}
	money[pooltreasure.Platinum] = 5
	member := poolsave.Character{Name: "HERO", RaceID: "dwarf", GenderID: "male",
		ClassID: "fighter", AlignmentID: "lawful-good",
		Abilities: [6]int{16, 10, 10, 13, 10, 10}, MaxHP: 20, CurrentHP: 4,
		Money: money, PortraitHead: 1, PortraitBody: 1, IconSize: 1}
	application.state = poolsave.State{Schema: poolsave.Schema,
		CharacterLibrary: []poolsave.Character{member}, Party: []poolsave.Character{member}}
	application.saveState = func(poolsave.State) error { return nil }
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if err := press(application, ebiten.KeyB); err != nil {
		t.Fatal(err)
	}
	for tick := 0; tick < 20000 && !application.introDone; tick++ {
		if application.introWaiting || application.tourPage >= 0 {
			if err := press(application, ebiten.KeyEnter); err != nil {
				t.Fatal(err)
			}
			continue
		}
		application.keys = scriptedKeys{}
		if err := application.Update(); err != nil {
			t.Fatal(err)
		}
	}
	if !application.introDone {
		t.Fatal("開場沒有跑完")
	}

	// 走到旅店門口。走進去之後腳本會問「要不要住下」。
	if !walkToCells(t, application, 4000, innCells) {
		t.Fatalf("沒走到旅店，最後在 %+v", application.spawn)
	}
	if !application.cellWaitingMenu || len(application.cellMenuOptions) != 2 {
		t.Fatalf("旅店沒有問話：選單 %v 文字 %q", application.cellMenuOptions, application.eventText)
	}
	t.Logf("旅店問：%q %v", application.eventText, application.cellMenuOptions)

	before := application.state.Party[0].Money[pooltreasure.Platinum]
	// 第一項是 YES。
	application.cellMenuCursor = 0
	for tick := 0; tick < 64 && !application.campOpen; tick++ {
		if err := press(application, ebiten.KeyEnter); err != nil {
			t.Fatalf("答應住下：%v", err)
		}
	}
	if !application.campOpen {
		t.Fatalf("答應之後沒有開紮營：狀態列 %q 選單 %v",
			application.statusLine, application.cellMenuOptions)
	}
	after := application.state.Party[0].Money[pooltreasure.Platinum]
	if after != before-1 {
		t.Errorf("白金 %d → %d，住一晚要扣一枚", before, after)
	}

	// 休息：說明書 p.31 對旅店的保證是「絕對安全而且不會有人中途打擾」與
	// 「你高興休息到什麼時候就待到什麼時候」——**不是自動回滿**。回血一樣是
	// 每二十四小時一點（spec 114），所以要挑夠長的時間。
	// 紮營有兩層（spec 135）：`R` 先進排時間那一層。
	if err := press(application, ebiten.KeyR); err != nil {
		t.Fatalf("進排時間那一層：%v", err)
	}
	if application.campStage != campStageRest {
		t.Fatal("按 R 沒有進排時間那一層")
	}
	missing := application.state.Party[0].MaxHP - application.state.Party[0].CurrentHP
	for day := 0; day < missing; day++ {
		if err := press(application, ebiten.KeyY); err != nil {
			t.Fatalf("選天數那一欄：%v", err)
		}
		if err := press(application, ebiten.KeyI); err != nil {
			t.Fatalf("加一天：%v", err)
		}
	}
	if err := press(application, ebiten.KeyR); err != nil {
		t.Fatalf("休息：%v", err)
	}
	if got := application.state.Party[0].CurrentHP; got != application.state.Party[0].MaxHP {
		t.Errorf("休息完生命值是 %d／%d", got, application.state.Party[0].MaxHP)
	}
	for tick := 0; tick < 64 && (application.campOpen || application.campFromProgram); tick++ {
		if err := press(application, ebiten.KeyEscape); err != nil {
			t.Fatalf("收掉紮營：%v", err)
		}
	}
	if application.campOpen || application.campFromProgram {
		t.Fatal("紮營畫面收不掉")
	}
	if application.mode != modeAdventure {
		t.Fatalf("收掉之後 mode=%d，應該回到地圖", application.mode)
	}
	t.Logf("住完一晚：白金 %d，生命值 %d／%d，位置 GEO%d/%d (%d,%d)",
		after, application.state.Party[0].CurrentHP, application.state.Party[0].MaxHP,
		application.spawn.Map.Archive, application.spawn.Map.BlockID,
		application.spawn.X, application.spawn.Y)
}

// walkToCells 只用方向鍵走到指定的任一格，路上遇到文字與選單就按過去；
// 走到目標格會停在它自己開的問話上，所以停止條件是「有選單」或「站上去了」。
func walkToCells(t *testing.T, application *app, budget int, want map[[2]int]bool) bool {
	t.Helper()
	for step := 0; step < budget; step++ {
		if want[[2]int{int(application.spawn.X), int(application.spawn.Y)}] &&
			application.cellWaitingMenu {
			return true
		}
		if key, busy := escapeKeyForWalk(application); busy {
			if err := press(application, key); err != nil {
				t.Fatalf("第 %d 步收拾畫面：%v", step, err)
			}
			continue
		}
		if application.spawn.Map != phlanCity {
			return false
		}
		plan := planToCellsInsideThisMap(application, step%4, want)
		if len(plan) == 0 {
			return false
		}
		next := plan[0]
		if application.spawn.Facing != next.facing {
			key := ebiten.KeyArrowRight
			if (int(next.facing)-int(application.spawn.Facing)+4)%4 == 3 {
				key = ebiten.KeyArrowLeft
			}
			if err := press(application, key); err != nil {
				t.Fatalf("第 %d 步轉向：%v", step, err)
			}
			continue
		}
		if err := press(application, ebiten.KeyArrowUp); err != nil {
			t.Fatalf("第 %d 步前進：%v", step, err)
		}
	}
	return false
}

// planToCellsInsideThisMap 與 planInsidePhlan 同一套（不繞出這張圖），
// 目標格由呼叫端給。
func planToCellsInsideThisMap(application *app, rotate int, want map[[2]int]bool) []exploreStep {
	type node struct{ x, y int }
	start := node{int(application.spawn.X), int(application.spawn.Y)}
	from := map[node]node{start: start}
	via := map[node]uint8{}
	queue := []node{start}
	for len(queue) != 0 {
		current := queue[0]
		queue = queue[1:]
		if current != start && want[[2]int{current.x, current.y}] {
			steps := []exploreStep{}
			for cursor := current; cursor != start; cursor = from[cursor] {
				steps = append([]exploreStep{{facing: via[cursor]}}, steps...)
			}
			return steps
		}
		for offset := 0; offset < 4; offset++ {
			facing := (offset + rotate) % 4
			if !application.initialMap.Grid.CanMoveDungeonWrapped(current.x, current.y, facing*2) {
				continue
			}
			next := node{x: current.x + exploreDeltas[facing][0], y: current.y + exploreDeltas[facing][1]}
			if next.x < 0 || next.x >= geometry.Width || next.y < 0 || next.y >= geometry.Height {
				continue
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

var _ = gamepack.ProgramCamp
