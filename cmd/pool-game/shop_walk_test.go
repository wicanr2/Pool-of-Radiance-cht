package main

import (
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
	"github.com/wicanr2/golden-box-remake-engine/geometry"
)

// armouryCells 是菲蘭城區（GEO3/0）走進武具店的格子。
//
// 依據是原始資料，不是攻略：`ecl3/0` 入口 1 在 `99F7h` 做
// `AND 127, @C04F, @6E82`——**每格的事件索引就是地形碼的低七位元**——
// 接著 `9B4Ch ON GOTO @9800` 那張 28 支的表第 22 支是 `A8BBh`，
// 也就是「THE SHOP SPECIALIZES IN ARMS AND ARMOR」那一段（spec 067 的
// `A919h` 就在它下面）。GEO3/0 裡地形碼 & 127 等於 22 的正好這五格。
var armouryCells = map[[2]int]bool{
	{13, 8}: true, {8, 11}: true, {11, 12}: true, {8, 13}: true, {9, 13}: true,
}

var phlanCity = gamepack.MapKey{Archive: 3, BlockID: 0}

// walkToArmoury 只用方向鍵往武具店走，路上遇到文字與選單就按過去。
// 走出城區就放棄——那表示路線判斷錯了，不是店不在。
func walkToArmoury(t *testing.T, application *app, budget int) bool {
	t.Helper()
	for step := 0; step < budget; step++ {
		if application.shopActive {
			return true
		}
		if key, busy := escapeKeyForWalk(application); busy {
			if err := press(application, key); err != nil {
				t.Fatalf("第 %d 步收拾畫面：%v", step, err)
			}
			continue
		}
		if application.spawn.Map != phlanCity {
			t.Logf("第 %d 步走出城區到 GEO%d/%d", step,
				application.spawn.Map.Archive, application.spawn.Map.BlockID)
			return false
		}
		plan := planInsidePhlan(application, step%4)
		if len(plan) == 0 {
			return false
		}
		want := plan[0]
		if application.spawn.Facing != want.facing {
			key := ebiten.KeyArrowRight
			if (int(want.facing)-int(application.spawn.Facing)+4)%4 == 3 {
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
	return application.shopActive
}

// planInsidePhlan 找到最近一格武具店門口的路，**不繞出這張圖**。
//
// `planToCells` 走的是原版的環繞規則（`WrapCoordinate`），而城區的邊界格
// 一踩就換圖：開場結束的位置是 (0,4)，往西一步就繞到 (15,4) 直接離開菲蘭。
// 這裡的路只在 0..15 之內走，所以「走不到」就真的是走不到。
func planInsidePhlan(application *app, rotate int) []exploreStep {
	type node struct{ x, y int }
	start := node{int(application.spawn.X), int(application.spawn.Y)}
	from := map[node]node{start: start}
	via := map[node]uint8{}
	queue := []node{start}
	for len(queue) != 0 {
		current := queue[0]
		queue = queue[1:]
		if current != start && armouryCells[[2]int{current.x, current.y}] {
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

// 裝備的完整玩家鏈：建角拿到金幣 → 走到武具店 → 買 → 按 I 裝上。
//
// 先前只驗到「把預設人物的裝備直接塞進隊伍」與「武具店的庫存與價目對得上
// 真檔」（spec 067），中間那一段——玩家自己走到店裡買——是推論。這一則
// 只用按鍵走完，所以「玩家買得到裝備」變成實測。
func TestNormalKeysBuyAndEquipFromTheWeaponShop(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	text := &scriptedTextKeys{scriptedKeys: scriptedKeys{}}
	application.keys = text
	application.saveState = func(poolsave.State) error { return nil }
	step := func(what string, key ebiten.Key, chars ...rune) {
		t.Helper()
		text.scriptedKeys[key] = true
		text.chars = chars
		if err := application.Update(); err != nil {
			t.Fatalf("%s: %v", what, err)
		}
	}
	idle := func() {
		t.Helper()
		text.chars = nil
		if err := application.Update(); err != nil {
			t.Fatalf("idle: %v", err)
		}
	}

	step("標題", ebiten.KeyEnter)
	step("開始建角", ebiten.KeyC)
	for _, what := range []string{"種族", "性別", "職業", "陣營"} {
		step(what, ebiten.KeyEnter)
	}
	idle()
	step("接受骰值", ebiten.KeyEnter)
	step("輸入姓名", ebiten.KeyEnter, 'H', 'E', 'R', 'O')
	step("保留肖像", ebiten.KeyK)
	step("確認造形", ebiten.KeyEnter)
	step("造形 OK", ebiten.KeyY)
	step("加入隊伍", ebiten.KeyA)
	step("開始冒險", ebiten.KeyB)
	if len(application.state.Party) != 1 {
		t.Fatalf("隊伍有 %d 人", len(application.state.Party))
	}
	purse := application.state.Party[0].Money[3]
	if purse == 0 {
		t.Fatal("建角沒有給金幣，買不了東西")
	}
	t.Logf("建角拿到 %d 金幣", purse)

	for tick := 0; tick < 20000 && !application.introDone; tick++ {
		if application.introWaiting || application.tourPage >= 0 {
			step("推進開場", ebiten.KeyEnter)
			continue
		}
		idle()
	}
	if !application.introDone {
		t.Fatal("開場沒有跑完")
	}
	t.Logf("開場結束時在 GEO%d/%d (%d,%d)", application.spawn.Map.Archive,
		application.spawn.Map.BlockID, application.spawn.X, application.spawn.Y)

	application.keys = scriptedKeys{}
	if !walkToArmoury(t, application, 4000) {
		t.Fatalf("沒走到武具店，最後在 %+v 狀態列 %q", application.spawn, application.statusLine)
	}
	shop := application.shop
	if shop == nil || len(shop.items) == 0 {
		t.Fatal("店開了卻沒有庫存")
	}
	t.Logf("走到店裡：GEO%d/%d (%d,%d)，庫存 %d 件",
		application.spawn.Map.Archive, application.spawn.Map.BlockID,
		application.spawn.X, application.spawn.Y, len(shop.items))

	// 挑一件買得起的，優先挑盾——它裝上去之後 AC 會變，可以順便驗
	// 「裝上了」不只是翻了一個旗標。價目 0 的那幾件（箭、弩矢）原始資料
	// 就寫 0（spec 067），拿它們驗不到扣錢。
	affordable := -1
	wallet := uint32(application.state.Party[0].Money[3])
	for index, record := range shop.items {
		price := uint32(record.Price())
		if price == 0 || price > wallet {
			continue
		}
		if affordable < 0 {
			affordable = index
		}
		if record.Name == "Shield" {
			affordable = index
			break
		}
	}
	if affordable < 0 {
		t.Fatalf("這一家店沒有買得起的東西：金幣 %d", wallet)
	}
	for shop.cursor != affordable {
		if err := press(application, ebiten.KeyDown); err != nil {
			t.Fatalf("移游標：%v", err)
		}
	}
	wanted := shop.items[shop.cursor]
	before := application.state.Party[0].Money[3]
	beforeItems := len(application.state.Party[0].Inventory)
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatalf("買：%v", err)
	}
	after := application.state.Party[0].Money[3]
	if after >= before {
		t.Errorf("買了 %s 金幣沒有變：%d → %d（訊息 %q）", wanted.Name, before, after, shop.message)
	}
	if got := len(application.state.Party[0].Inventory); got != beforeItems+1 {
		t.Fatalf("背包從 %d 變成 %d 件，應該多一件", beforeItems, got)
	}
	t.Logf("買了 %s，花掉 %d 金幣（剩 %d）", wanted.Name, before-after, after)

	if err := press(application, ebiten.KeyEscape); err != nil {
		t.Fatalf("離開店裡：%v", err)
	}
	if application.shopActive {
		t.Fatal("按 Escape 沒有離開店裡")
	}
	for tick := 0; tick < 256 && (application.cellEventPending || application.cellWaitingMenu); tick++ {
		if err := press(application, ebiten.KeyEnter); err != nil {
			t.Fatalf("讀完離店的文字：%v", err)
		}
	}

	// 按 I 開裝備頁，把剛買的裝上。
	if err := press(application, ebiten.KeyI); err != nil {
		t.Fatalf("按 I：%v", err)
	}
	if application.equipment == nil || !application.equipmentOpen {
		t.Fatalf("按 I 沒有開裝備頁：狀態列 %q", application.statusLine)
	}
	inventory := application.state.Party[0].Inventory
	bought := -1
	for index, item := range inventory {
		if item.Name == wanted.Name {
			bought = index
			break
		}
	}
	if bought < 0 {
		t.Fatalf("背包裡找不到剛買的 %s", wanted.Name)
	}
	for application.equipment.item != bought {
		if err := press(application, ebiten.KeyDown); err != nil {
			t.Fatalf("裝備頁移游標：%v", err)
		}
	}
	beforeAC, _, err := application.memberDefenceStats(application.state.Party[0],
		creationArmorClassInternal, creationBaseMovement)
	if err != nil {
		t.Fatalf("裝上之前算 AC：%v", err)
	}
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatalf("裝上：%v", err)
	}
	ready := application.state.Party[0].Inventory[bought].Raw
	if len(ready) <= itemReadyOffset || ready[itemReadyOffset] != 1 {
		t.Fatalf("按 Enter 之後 %s 的 +%02Xh 是 %v，應該是 1（訊息 %q）",
			wanted.Name, itemReadyOffset, ready[itemReadyOffset:], application.equipment.message)
	}
	afterAC, _, err := application.memberDefenceStats(application.state.Party[0],
		creationArmorClassInternal, creationBaseMovement)
	if err != nil {
		t.Fatalf("裝上之後算 AC：%v", err)
	}
	// 旗標翻了不代表規則那一半有接上；盾要真的算進 AC 才算「裝上」。
	if wanted.Name == "Shield" && afterAC <= beforeAC {
		t.Errorf("裝上盾之後 AC 內部值 %d → %d，應該變好", beforeAC, afterAC)
	}
	t.Logf("裝上 %s：AC 內部值 %d → %d（桌上型 %d → %d）",
		wanted.Name, beforeAC, afterAC,
		gamepack.ArmourClassScale-beforeAC, gamepack.ArmourClassScale-afterAC)
}
