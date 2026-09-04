package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/creation"
)

func iconApp(raceID string) *app {
	flow := creation.NewFlow()
	for index, race := range creation.Races {
		if race.ID == raceID {
			flow.RaceIndex = index
		}
	}
	flow.Stage = creation.StageIcon
	flow.IconColors = [6][2]uint8{{1, 9}, {2, 10}, {3, 11}, {4, 12}, {6, 14}, {7, 15}}
	flow.IconSize = 2
	if flow.UsesIconSizeMenu() {
		flow.IconSize = 1
	}
	return &app{
		mode: modeCreation, flow: flow, keys: scriptedKeys{},
		loadIcon: func(uint8, uint8, uint8, bool, [6][2]uint8) (*ebiten.Image, error) {
			return ebiten.NewImage(1, 1), nil
		},
	}
}

func iconLabels(a *app) []string {
	var labels []string
	for _, option := range a.iconMenuOptions() {
		labels = append(labels, option.label)
	}
	return labels
}

func sameLabels(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for index := range want {
		if got[index] != want[index] {
			return false
		}
	}
	return true
}

// 頂層與三個子選單的列與順序照 spec 003 第 7..10 步。
func TestIconMenuLevelsMatchTheOriginalScreens(t *testing.T) {
	a := iconApp("dwarf")
	if want := []string{"PARTS", "COLOR-1", "COLOR-2", "SIZE", "EXIT"}; !sameLabels(iconLabels(a), want) {
		t.Fatalf("頂層是 %v，預期 %v", iconLabels(a), want)
	}
	// SIZE 只給預設小號的種族；人類本來就是正常大小。
	human := iconApp("human")
	if want := []string{"PARTS", "COLOR-1", "COLOR-2", "EXIT"}; !sameLabels(iconLabels(human), want) {
		t.Fatalf("人類的頂層是 %v，預期 %v", iconLabels(human), want)
	}
	if err := a.chooseIconMenu("PARTS"); err != nil {
		t.Fatal(err)
	}
	if want := []string{"HEAD", "WEAPON", "EXIT"}; !sameLabels(iconLabels(a), want) {
		t.Fatalf("PARTS 是 %v，預期 %v", iconLabels(a), want)
	}
	if err := a.chooseIconMenu("HEAD"); err != nil {
		t.Fatal(err)
	}
	if want := []string{"NEXT", "PREV", "KEEP", "EXIT"}; !sameLabels(iconLabels(a), want) {
		t.Fatalf("循環層是 %v，預期 %v", iconLabels(a), want)
	}
	// COLOR-1 的六個部位，武器排第一。
	a = iconApp("dwarf")
	if err := a.chooseIconMenu("COLOR-1"); err != nil {
		t.Fatal(err)
	}
	if want := []string{"WEAPON", "BODY", "HAIR", "SHIELD", "ARM", "LEG", "EXIT"}; !sameLabels(iconLabels(a), want) {
		t.Fatalf("COLOR-1 是 %v，預期 %v", iconLabels(a), want)
	}
	// COLOR-2 把 HAIR 換成 FACE，其餘相同。
	a = iconApp("dwarf")
	if err := a.chooseIconMenu("COLOR-2"); err != nil {
		t.Fatal(err)
	}
	if want := []string{"WEAPON", "BODY", "FACE", "SHIELD", "ARM", "LEG", "EXIT"}; !sameLabels(iconLabels(a), want) {
		t.Fatalf("COLOR-2 是 %v，預期 %v", iconLabels(a), want)
	}
}

// 畫面上的順序與記錄裡的順序不同：武器在畫面上排第一，在記錄 `+0C6h` 是最後
// 一格。挑錯就會改到別的部位。
func TestIconColourMenuMapsToTheRecordSlots(t *testing.T) {
	for _, item := range []struct {
		label string
		slot  uint8
	}{{"WEAPON", 5}, {"BODY", 0}, {"HAIR", 3}, {"SHIELD", 4}, {"ARM", 1}, {"LEG", 2}} {
		a := iconApp("dwarf")
		if err := a.chooseIconMenu("COLOR-1"); err != nil {
			t.Fatal(err)
		}
		if err := a.chooseIconMenu(item.label); err != nil {
			t.Fatal(err)
		}
		if a.flow.IconPart != item.slot {
			t.Fatalf("%s 對到第 %d 格，預期第 %d 格", item.label, a.flow.IconPart, item.slot)
		}
	}
}

// KEEP 收下候選，EXIT 退回進來時的值。原版同時給這兩個出口，
// 其中一個不做事的話就是多餘的。
func TestIconCycleKeepAcceptsAndExitReverts(t *testing.T) {
	a := iconApp("dwarf")
	for _, label := range []string{"PARTS", "WEAPON", "NEXT", "NEXT"} {
		if err := a.chooseIconMenu(label); err != nil {
			t.Fatal(err)
		}
	}
	if a.flow.IconWeapon != 2 {
		t.Fatalf("兩次 NEXT 之後武器是 %d", a.flow.IconWeapon)
	}
	if err := a.chooseIconMenu("EXIT"); err != nil {
		t.Fatal(err)
	}
	if a.flow.IconWeapon != 0 {
		t.Fatalf("EXIT 之後武器是 %d，應該退回 0", a.flow.IconWeapon)
	}
	for _, label := range []string{"WEAPON", "PREV", "KEEP"} {
		if err := a.chooseIconMenu(label); err != nil {
			t.Fatal(err)
		}
	}
	if a.flow.IconWeapon != 31 {
		t.Fatalf("PREV 之後武器是 %d，預期 31（0 往前繞）", a.flow.IconWeapon)
	}
	// KEEP 回到 PARTS 那一層，不是回頂層。
	if want := []string{"HEAD", "WEAPON", "EXIT"}; !sameLabels(iconLabels(a), want) {
		t.Fatalf("KEEP 之後停在 %v", iconLabels(a))
	}
}

// SIZE 的 LARGE 把圖示放大，EXIT 退回進來時的大小。
func TestIconSizeMenu(t *testing.T) {
	a := iconApp("dwarf")
	if a.flow.IconSize != 1 {
		t.Fatalf("矮人的預設大小是 %d，預期 1", a.flow.IconSize)
	}
	if err := a.chooseIconMenu("SIZE"); err != nil {
		t.Fatal(err)
	}
	if err := a.chooseIconMenu("LARGE"); err != nil {
		t.Fatal(err)
	}
	if a.flow.IconSize != 2 {
		t.Fatalf("LARGE 之後大小是 %d", a.flow.IconSize)
	}
	if err := a.chooseIconMenu("SIZE"); err != nil {
		t.Fatal(err)
	}
	if err := a.chooseIconMenu("EXIT"); err != nil {
		t.Fatal(err)
	}
	if a.flow.IconSize != 2 {
		t.Fatalf("EXIT 應該退回進來時的 2，得到 %d", a.flow.IconSize)
	}
}

// 頂層 EXIT 進確認頁（spec 003 第 11 步）。
func TestIconTopExitAsksForConfirmation(t *testing.T) {
	a := iconApp("human")
	if err := a.chooseIconMenu("EXIT"); err != nil {
		t.Fatal(err)
	}
	if a.flow.Stage != creation.StageIconConfirm {
		t.Fatalf("頂層 EXIT 之後停在 stage %d", a.flow.Stage)
	}
}

// 方向鍵與首字母兩條路都要走得動。
func TestIconMenuAcceptsArrowsAndInitials(t *testing.T) {
	a := iconApp("human")
	a.keys = scriptedKeys{ebiten.KeyArrowDown: true}
	if err := a.iconMenuInput(); err != nil {
		t.Fatal(err)
	}
	if a.iconMenu.cursor != 1 {
		t.Fatalf("方向鍵之後游標在 %d", a.iconMenu.cursor)
	}
	a.keys = scriptedKeys{ebiten.KeyEnter: true}
	if err := a.iconMenuInput(); err != nil {
		t.Fatal(err)
	}
	if a.iconMenu.level != iconMenuColour || a.iconMenu.component != 0 {
		t.Fatalf("ENTER 選到的是 level %d component %d", a.iconMenu.level, a.iconMenu.component)
	}
	a.keys = scriptedKeys{ebiten.KeyA: true}
	if err := a.iconMenuInput(); err != nil {
		t.Fatal(err)
	}
	if a.flow.IconPart != 1 {
		t.Fatalf("按 A（ARM）選到第 %d 格", a.flow.IconPart)
	}
}
