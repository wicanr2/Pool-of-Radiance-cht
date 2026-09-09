package main

import (
	"testing"

	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 紮營那兩列指令照原版（spec 135）。
func TestCampCommandRows(t *testing.T) {
	want := []string{"SAVE", "VIEW", "MAGIC", "REST", "ALTER", "EXIT"}
	if len(campCommands) != len(want) {
		t.Fatalf("第一層有 %d 項，原版是 %d 項", len(campCommands), len(want))
	}
	for index, command := range want {
		if campCommands[index] != command {
			t.Errorf("第一層第 %d 項是 %s，原版是 %s", index, campCommands[index], command)
		}
	}
	wantRest := []string{"REST", "DAYS", "HOURS", "MINS", "INC", "DEC", "EXIT"}
	if len(campRestCommands) != len(wantRest) {
		t.Fatalf("第二層有 %d 項，原版是 %d 項", len(campRestCommands), len(wantRest))
	}
	for index, command := range wantRest {
		if campRestCommands[index] != command {
			t.Errorf("第二層第 %d 項是 %s，原版是 %s", index, campRestCommands[index], command)
		}
	}
}

// 高亮的是大寫的那個字母，不是第一個字母——`daYs` 的鍵是 Y（spec 135）。
func TestSplitCommandKeyUsesTheCapital(t *testing.T) {
	cases := []struct{ label, before, key, after string }{
		{"daYs", "da", "Y", "s"},
		{"Hours", "", "H", "ours"},
		{"AREA", "", "A", "REA"},
		{"Y 天", "", "Y", " 天"},
		{"紮營", "", "紮", "營"},
	}
	for _, item := range cases {
		before, key, after := splitCommandKey(item.label)
		if before != item.before || key != item.key || after != item.after {
			t.Errorf("%q 切成 %q/%q/%q，應該是 %q/%q/%q",
				item.label, before, key, after, item.before, item.key, item.after)
		}
	}
}

// 兩層的同一個字母意思不同：第一層的 M 是 MAGIC，第二層的 M 是 MINS。
// 這裡只釘住「兩層是分開的」——按鍵本身由 inn_test 與 memorise_test 走完整條路。
func TestCampStagesAreSeparate(t *testing.T) {
	if campStageMenu == campStageRest {
		t.Fatal("兩層要分得出來")
	}
	application := &app{mode: modeAdventure, campOpen: true, campStage: campStageMenu}
	if got := application.screenName(); got != "camp" {
		t.Errorf("第一層報 %q，應該是 camp", got)
	}
	application.campStage = campStageRest
	if got := application.screenName(); got != "camp-rest" {
		t.Errorf("第二層報 %q，應該是 camp-rest", got)
	}
	// 紮營不是「蓋上來的面板」——指令列那一列還在，只是換了內容。
	if application.panelOpen() {
		t.Error("紮營不該算成面板")
	}
}

// 紮營的選單樹照原版分層（spec 135）。
func TestCampAlterRows(t *testing.T) {
	want := []string{"ORDER", "DROP", "SPEED", "ICON", "PICS", "EXIT"}
	if len(campAlterCommands) != len(want) {
		t.Fatalf("ALTER 那一列有 %d 項，原版是 %d 項", len(campAlterCommands), len(want))
	}
	for index, command := range want {
		if campAlterCommands[index] != command {
			t.Errorf("第 %d 項是 %s，原版是 %s", index, campAlterCommands[index], command)
		}
	}
}

// 速度那一列是組出來的：0 不列 Faster、9 不列 Slower
//（overlay-15 `1AD9h`／`1AF1h` 的兩個比較）。
func TestCampSpeedRowDropsTheEndStop(t *testing.T) {
	application := &app{mode: modeAdventure, campOpen: true, campStage: campStageSpeed}
	cases := []struct {
		speed uint8
		want  []string
	}{
		{campSpeedFastest, []string{"SLOWER", "EXIT"}},
		{5, []string{"FASTER", "SLOWER", "EXIT"}},
		{campSpeedSlowest, []string{"FASTER", "EXIT"}},
	}
	for _, item := range cases {
		application.gameSpeed = item.speed
		got := application.commandBarList()
		if len(got) != len(item.want) {
			t.Errorf("速度 %d 那一列是 %v，應該是 %v", item.speed, got, item.want)
			continue
		}
		for index := range got {
			if got[index] != item.want[index] {
				t.Errorf("速度 %d 那一列是 %v，應該是 %v", item.speed, got, item.want)
				break
			}
		}
	}
}

// PICS 那兩項顯示的是目前狀態，而**兩個開關預設都是開的**——零值要對應
// 預設，不然每一份測試用的 app 都會變成「圖片關掉」。
func TestCampPicsRowShowsTheCurrentState(t *testing.T) {
	application := &app{mode: modeAdventure, campOpen: true, campStage: campStagePics}
	got := application.commandBarList()
	if got[0] != "MONSTERS-ON" || got[1] != "PORTRAITS-ON" {
		t.Errorf("預設是 %v，兩個開關預設都該是開的", got)
	}
	application.monsterPicsHidden, application.portraitsHidden = true, true
	got = application.commandBarList()
	if got[0] != "MONSTERS-OFF" || got[1] != "PORTRAITS-OFF" {
		t.Errorf("關掉之後是 %v", got)
	}
}

// ORDER 把一個人挪到另一個位置，其餘依序遞補。
func TestMoveMemberKeepsEveryoneElseInOrder(t *testing.T) {
	names := func(a *app) string {
		out := ""
		for _, member := range a.state.Party {
			out += member.Name
		}
		return out
	}
	build := func() *app {
		application := &app{}
		for _, name := range []string{"A", "B", "C", "D"} {
			application.state.Party = append(application.state.Party,
				poolsave.Character{Name: name})
		}
		return application
	}
	cases := []struct{ from, to int; want string }{
		{0, 3, "BCDA"},
		{3, 0, "DABC"},
		{1, 2, "ACBD"},
		{2, 2, "ABCD"},
	}
	for _, item := range cases {
		application := build()
		application.moveMember(item.from, item.to)
		if got := names(application); got != item.want {
			t.Errorf("把第 %d 位挪到第 %d 位得到 %s，應該是 %s",
				item.from+1, item.to+1, got, item.want)
		}
	}
	// 越界不動任何人。
	application := build()
	application.moveMember(0, 9)
	if got := names(application); got != "ABCD" {
		t.Errorf("越界之後變成 %s，應該原封不動", got)
	}
}

// 每一層都報得出自己是哪一層，截圖腳本才不用盲按。
func TestCampStageNames(t *testing.T) {
	cases := map[campStage]string{
		campStageMenu:        "camp",
		campStageRest:        "camp-rest",
		campStageAlter:       "camp-alter",
		campStageOrderSelect: "camp-order-select",
		campStageOrderPlace:  "camp-order-place",
		campStageSpeed:       "camp-speed",
		campStagePics:        "camp-pics",
		campStageQuitConfirm: "camp-quit",
		campStageDropConfirm: "camp-drop",
	}
	for stage, want := range cases {
		application := &app{mode: modeAdventure, campOpen: true, campStage: stage}
		if got := application.screenName(); got != want {
			t.Errorf("第 %d 層報 %q，應該是 %q", stage, got, want)
		}
	}
}
