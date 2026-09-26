package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/treasure"
)

// AI 用物品（overlay-09 entry 3 `03E3h` → overlay-19 entry 8 `1A86h`，spec 096，issue #71）。
// 從 Update() 送鍵：Q）UICK 過的隊員輪到時 tacticalInput 自己分派。

// newQuickWandApp：三級法師（沒記法術）身上一支魔法飛彈杖，charges 次、數量 count。
func newQuickWandApp(t *testing.T, charges, count uint8, readied bool) (*app, *tacticalState) {
	t.Helper()
	application, state := newFoeCastApp(t)
	types, err := gamepack.ReadDOSItemTypeTable(filepath.Join("..", "..", "Pool of Radiance (1988).zip"))
	if err != nil {
		t.Fatal(err)
	}
	application.itemTypes = types
	levels := make([]uint8, gamepack.ClassThac0ClassCount)
	levels[gamepack.ClassSlotMagicUser] = 3
	application.state.Party[0] = poolsave.Character{Name: "A", ClassLevels: levels, Quick: true,
		Inventory: []poolsave.Item{{Name: "WAND", Raw: wandRaw(t, types, charges, count, readied)}}}
	state.AIDriven[1] = true
	state.Mover = 1
	return application, state
}

// wandRaw 是一筆 63-byte 物品記錄：型別取表裡第一個不是卷軸（類別 0Bh..0Dh）的。
func wandRaw(t *testing.T, types *gamepack.ItemTypeTable, charges, count uint8, readied bool) []byte {
	t.Helper()
	raw := make([]byte, 63)
	for index := range types.Entries {
		category := types.Entries[index].Category()
		if category < treasure.ScrollCategoryFirst || category > treasure.ScrollCategoryLast {
			raw[gamepack.ItemTypeOffset] = uint8(index)
			break
		}
	}
	if readied {
		raw[gamepack.ItemReadiedOffset] = 1
	}
	raw[gamepack.ItemCountOffset] = count
	raw[gamepack.AIItemChargesOffset] = charges
	raw[gamepack.AIItemSpellOffset] = gamepack.SpellIDMagicMissile
	return raw
}

// Magic Off（每場的預設）也會用：entry 3 沒有 `DS:6D23h` 那道閘。用一次次數減一，
// 用到 0 就從身上拿掉（overlay-25 entry 17）。
func TestQuickMemberUsesAWandUntilItIsEmpty(t *testing.T) {
	application, state := newQuickWandApp(t, 2, 0, true)
	if state.Casting.MagicOn {
		t.Fatal("magic starts on")
	}
	hp := state.HitPoints[2]
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(state.FoeLog, "USES AN ITEM ITEM:WAND") || state.HitPoints[2] >= hp {
		t.Fatalf("the quick member did not use the wand: %q, foe hp %d → %d",
			state.FoeLog, hp, state.HitPoints[2])
	}
	inventory := application.state.Party[0].Inventory
	if len(inventory) != 1 || inventory[0].Raw[gamepack.AIItemChargesOffset] != 1 {
		t.Fatalf("one charge should be left: %+v", inventory)
	}
	state.Mover = 1
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(state.FoeLog, "USES AN ITEM") {
		t.Fatalf("the second charge was not used: %q", state.FoeLog)
	}
	if len(application.state.Party[0].Inventory) != 0 {
		t.Fatalf("the empty wand is still carried: %+v", application.state.Party[0].Inventory)
	}
}

// 數量大於 1 的（一疊藥水那種）用掉的是一個，次數不動（`1C4Ch`）。
func TestQuickMemberUsesOneOfAStack(t *testing.T) {
	application, state := newQuickWandApp(t, 1, 3, true)
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(state.FoeLog, "USES AN ITEM") {
		t.Fatalf("the stack was not used: %q", state.FoeLog)
	}
	raw := application.state.Party[0].Inventory[0].Raw
	if raw[gamepack.ItemCountOffset] != 2 || raw[gamepack.AIItemChargesOffset] != 1 {
		t.Fatalf("count %d charges %d, want 2 and 1", raw[gamepack.ItemCountOffset], raw[gamepack.AIItemChargesOffset])
	}
}

// 沒穿戴的不用（`04B7h`：`+34h == 0` 跳過）；這一隻照常走接近迴圈。
func TestQuickMemberIgnoresAnUnreadiedWand(t *testing.T) {
	application, state := newQuickWandApp(t, 2, 0, false)
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(state.FoeLog, "USES AN ITEM") {
		t.Fatalf("an unreadied wand was used: %q", state.FoeLog)
	}
	if application.state.Party[0].Inventory[0].Raw[gamepack.AIItemChargesOffset] != 2 {
		t.Fatal("the unreadied wand lost a charge")
	}
}
