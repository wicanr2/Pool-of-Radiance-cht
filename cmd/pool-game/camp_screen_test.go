package main

import "testing"

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
