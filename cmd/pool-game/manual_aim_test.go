package main

import (
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
)

// 離開 Manual 的四個鍵是原版 `cs:2D80h` 那個**集合**解出來的
// （`{掃描碼 0, Return, 'E', 'T'}`，spec 127）。這一條釘住「Return 與 T
// 是同一件事、E 與取消是同一件事」——先前把那 32 個位元組讀成 Pascal 字串
// 的時候，兩種極性都會矛盾。
func TestManualAimExitKeysComeFromTheOriginalSet(t *testing.T) {
	// 位元圖逐位元組照抄 overlay-13 `2D80h`。
	bitmap := [32]byte{0x01, 0x20, 0, 0, 0, 0, 0, 0, 0x20, 0, 0x10}
	var members []int
	for index, value := range bitmap {
		for bit := 0; bit < 8; bit++ {
			if value>>uint(bit)&1 != 0 {
				members = append(members, index*8+bit)
			}
		}
	}
	want := []int{0, 13, 'E', 'T'}
	if len(members) != len(want) {
		t.Fatalf("集合有 %d 個成員 %v，預期 4 個 %v", len(members), members, want)
	}
	for index := range want {
		if members[index] != want[index] {
			t.Errorf("第 %d 個成員是 %d，預期 %d", index, members[index], want[index])
		}
	}
	// 實作要涵蓋這四個：Return／T 確定、E／Esc 取消。
	if len(manualAimConfirmKeys) != 2 || len(manualAimCancelKeys) != 2 {
		t.Errorf("確定 %d 個、取消 %d 個，集合是四個",
			len(manualAimConfirmKeys), len(manualAimCancelKeys))
	}
}

// 游標的八個方向鍵與戰術移動同一組，位移取自同一張九方向表。
func TestManualAimUsesTheTacticalStepKeys(t *testing.T) {
	if len(tacticalStepKeys) != 8 {
		t.Fatalf("方向鍵有 %d 個，原版是 8 個", len(tacticalStepKeys))
	}
	// 原版 `3217h` 起的對應：H0 I1 M2 Q3 P4 O5 K6 G7。
	wants := [8][2]int{{0, -1}, {1, -1}, {1, 0}, {1, 1}, {0, 1}, {-1, 1}, {-1, 0}, {-1, -1}}
	for direction, want := range wants {
		step, err := combat.DirectionStep(uint8(direction))
		if err != nil {
			t.Fatal(err)
		}
		if int(int8(step.X)) != want[0] || int(int8(step.Y)) != want[1] {
			t.Errorf("方向 %d 的位移是 (%d,%d)，預期 (%d,%d)",
				direction, int8(step.X), int8(step.Y), want[0], want[1])
		}
	}
}

// 游標走一格、走到盤面外就不動；停在有人的那一格才選得出目標。
func TestManualAimMovesAndPicksAnOccupant(t *testing.T) {
	state := &tacticalState{
		Roster: []combat.CombatantCell{{},
			{X: 5, Y: 5, FootprintClass: 1},
			{X: 6, Y: 5, FootprintClass: 1}},
	}
	application := &app{tactical: state}
	application.castTargets = []uint8{1}
	application.castTargeting = true
	if !application.beginManualAim() {
		t.Fatal("進不了 Manual")
	}
	if application.castManualX != 5 || application.castManualY != 5 {
		t.Errorf("游標起點是 (%d,%d)，預期停在目前目標 (5,5)",
			application.castManualX, application.castManualY)
	}
	// 往東（方向 2）一格。
	step, _ := combat.DirectionStep(2)
	application.castManualX += int(int8(step.X))
	application.castManualY += int(int8(step.Y))
	if application.castManualX != 6 {
		t.Errorf("往東一格之後 X 是 %d，預期 6", application.castManualX)
	}
	// 那一格站著第 2 個人，確定要選得到他。
	if err := application.confirmManualAim(); err != nil {
		t.Fatal(err)
	}
	if application.castManual {
		t.Error("選到人之後 Manual 應該關掉")
	}

	// 空格子上按確定：什麼都不做，游標留著。
	application2 := &app{tactical: state}
	application2.castTargets = []uint8{1}
	application2.castTargeting = true
	application2.beginManualAim()
	application2.castManualX, application2.castManualY = 12, 12
	if err := application2.confirmManualAim(); err != nil {
		t.Fatal(err)
	}
	if !application2.castManual {
		t.Error("空格子上按確定不該離開 Manual")
	}
}
