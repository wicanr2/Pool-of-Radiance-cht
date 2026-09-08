package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/guide"
)

func guideApp(t *testing.T) *app {
	t.Helper()
	catalogue, err := guide.TraditionalChinese()
	if err != nil {
		t.Fatal(err)
	}
	// `currentGuideMap` 要有地圖才回得出來——攻略頁畫的是那張圖的格線。
	initialMap := gamepack.GeometryMap{Key: gamepack.MapKey{Archive: 3, BlockID: 0}}
	return &app{
		mode: modeAdventure, introDone: true, guide: catalogue,
		spawn:      gamepack.Spawn{Map: gamepack.MapKey{Archive: 3, BlockID: 0}, X: 0, Y: 4},
		keys:       scriptedKeys{},
		initialMap: &initialMap,
	}
}

// 預設只顯示走過的格子。一打開就攤開整張圖等於替玩家把遊戲玩完了。
func TestGuideStartsFoggedAndNeedsTwoPressesToOpenUp(t *testing.T) {
	application := guideApp(t)
	if application.guideCellSeen(guide.Key(3, 0), 0, 4) {
		t.Fatal("還沒走過就算走過了")
	}
	application.rememberGuideCell()
	if !application.guideCellSeen(guide.Key(3, 0), 0, 4) {
		t.Fatal("走過的格子沒記下來")
	}

	application.guideOpen = true
	if name := application.screenName(); name != "guide" {
		t.Fatalf("剛打開時回報 %q", name)
	}
	// 第一次按 V 只出警告，不攤開——攤開是不可逆的，要先讓玩家知道。
	if err := press(application, ebiten.KeyV); err != nil {
		t.Fatal(err)
	}
	if application.guideFull {
		t.Fatal("第一次按 V 就攤開了，沒有先警告")
	}
	if !application.guideSpoilerWarned {
		t.Fatal("警告沒有記下來")
	}
	// 警告那一步也要有自己的畫面識別字。分不出來的話，截圖腳本只能盲按
	// 兩次 `V`；漏掉其中一次時它會停在 guide 等不到 guide-full，
	// 而那個症狀看起來像 `V` 沒接上，其實是按鍵掉了。
	if name := application.screenName(); name != "guide-warned" {
		t.Fatalf("出過警告之後回報 %q", name)
	}
	if err := press(application, ebiten.KeyV); err != nil {
		t.Fatal(err)
	}
	if !application.guideFull {
		t.Fatal("第二次按 V 沒有攤開")
	}
	if name := application.screenName(); name != "guide-full" {
		t.Fatalf("攤開之後回報 %q", name)
	}
}

// 這張地圖的攻略點要查得到，而且離隊伍最近的排前面。
func TestGuideListsTheNearestPointFirst(t *testing.T) {
	application := guideApp(t)
	definition, ok := application.currentGuideMap()
	if !ok {
		t.Fatal("GEO3/0 應該有攻略")
	}
	labels := definition.Labels(int(application.spawn.X), int(application.spawn.Y))
	if len(labels) == 0 {
		t.Fatal("一個地點都沒有")
	}
	// 隊伍站在 (0,4)，那一格自己就是城門。
	if labels[0].Label != "城門" {
		t.Fatalf("最近的是 %q，隊伍就站在城門那一格", labels[0].Label)
	}
	// 同一個標籤只留一筆，不然清單會被旅店那七格灌滿。
	seen := map[string]bool{}
	for _, point := range labels {
		if seen[point.Label] {
			t.Fatalf("%q 出現兩次", point.Label)
		}
		seen[point.Label] = true
	}
}

// F3 開、ESC 關；攻略開著的時候底部指令列不畫（panelOpen）。
func TestGuideOpensWithF3AndCountsAsAPanel(t *testing.T) {
	application := guideApp(t)
	if err := press(application, ebiten.KeyF3); err != nil {
		t.Fatal(err)
	}
	if !application.guideOpen {
		t.Fatal("F3 沒有打開攻略")
	}
	if !application.panelOpen() {
		t.Fatal("攻略開著卻不算面板，底下的指令列會露出來")
	}
	if err := press(application, ebiten.KeyEscape); err != nil {
		t.Fatal(err)
	}
	if application.guideOpen {
		t.Fatal("ESC 沒有關掉攻略")
	}
}
