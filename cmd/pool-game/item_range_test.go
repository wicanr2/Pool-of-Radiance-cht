package main

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	"github.com/wicanr2/golden-box-remake-engine/eclvm"
)

// 用物品放的法術，射程照 6 級算（overlay-22 entry 3 `0764h`，spec 098〈物品的射程〉，#85）。
// 編號 60 是物品效果（參數表 +0 = 2），射程 4 + 4 × 等級：施法者等級那一邊是 12（52 格），
// 射程那一邊是 6（28 格）。從 Update() 送 ENTER，Q）UICK 過的隊員自己挑物品、挑目標。

const itemRangeTestSpell = 60

func newItemRangeApp(t *testing.T, foeX uint8) (*app, *tacticalState) {
	t.Helper()
	application, state := newQuickWandApp(t, 5, 0, true)
	raw := application.state.Party[0].Inventory[0].Raw
	// `1B23h`：+3Dh 大於 38h 的減 17h，所以編號 60 存成 60 + 17h。
	raw[gamepack.AIItemSpellOffset] = itemRangeTestSpell + 0x17
	state.Roster[2].X = foeX
	return application, state
}

func TestAIItemRangeUsesLevelSix(t *testing.T) {
	application, _ := newItemRangeApp(t, 45)
	params := application.spellParameters[itemRangeTestSpell]
	if params.Source() != gamepack.SpellSourceItem || params.ItemRange() != 28 || params.Range(12) != 52 {
		t.Fatalf("spell %d is not the item-sourced 4 + 4 × level spell: source %d, range 6 %d, 12 %d",
			itemRangeTestSpell, params.Source(), params.ItemRange(), params.Range(12))
	}
	for _, c := range []struct {
		foeX    uint8
		reached bool
	}{
		{45, false}, // 40 格：12 級的射程搆得到，6 級搆不到
		{20, true},  // 15 格：對照組，兩種算法都搆得到
	} {
		application, state := newItemRangeApp(t, c.foeX)
		hp := state.HitPoints[2]
		if err := press(application, ebiten.KeyEnter); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(state.FoeLog, "USES AN ITEM") {
			t.Fatalf("foe at x=%d: the quick member did not use the item: log %q status %q",
				c.foeX, state.FoeLog, state.Status)
		}
		missed := strings.Contains(state.Status, "finds no reachable target")
		if missed == c.reached {
			t.Fatalf("foe at x=%d: reached %v, want %v (status %q, hp %d → %d)",
				c.foeX, !missed, c.reached, state.Status, hp, state.HitPoints[2])
		}
		if charges := application.state.Party[0].Inventory[0].Raw[gamepack.AIItemChargesOffset]; charges != 4 {
			t.Fatalf("foe at x=%d: the use was not paid for (1BF4h entry 34 → 1C0Eh): %d charges", c.foeX, charges)
		}
	}
}

// 玩家那一側同一支（overlay-19 `1BBDh` 立 `DS:6CB3h` 之後進 overlay-22 entry 5 瞄準）：
// U 開物品選單、U 用那一件，瞄準框的射程是 6 級的 28 格，不是 12 級的 52 格。
func TestPlayerItemAimUsesLevelSixRange(t *testing.T) {
	application, state := newItemRangeApp(t, 20)
	application.state.Party[0].Quick = false
	state.AIDriven[1] = false
	application.combatCommands = turnUseSegments
	pressAll(t, application, ebiten.KeyU)
	pressAll(t, application, ebiten.KeyU)
	if application.castAim == nil || !application.castTargeting {
		t.Fatalf("using the item did not open the aim: status %q", state.Status)
	}
	if application.castAim.reach != 28 {
		t.Fatalf("item aim reach %d, want 28 (4 + 4 × 6)", application.castAim.reach)
	}
}

// overlay-09 entry 3 `0431h`：`[4933h]+1CAh`（ECL @49E5，反魔法區）非 0 時 AI 不用物品，
// 照常走過去打（#85）。同一支魔法飛彈杖，@49E5 = 1 就不用。
func TestAIItemUseStopsInsideTheAntiMagicZone(t *testing.T) {
	for _, zone := range []uint16{0, 1} {
		application, state := newQuickWandApp(t, 2, 0, true)
		application.eventMachine = &eclvm.Machine{Memory: map[uint16]uint16{foeItemAntiMagicAddress: zone}}
		if err := press(application, ebiten.KeyEnter); err != nil {
			t.Fatal(err)
		}
		used := strings.Contains(state.FoeLog, "USES AN ITEM")
		if used == (zone != 0) {
			t.Fatalf("@49E5 = %d: used %v (log %q)", zone, used, state.FoeLog)
		}
	}
}
