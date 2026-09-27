package main

import (
	"strings"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 除咒術（overlay-22 `2508h`，#66）：身上沒有 24h 時清掉第一件被詛咒物品的 `+36h`，
// 只清一件；有 24h 就只解 24h、物品不動。從 Update() 按 C 施法、瞄準隊友。
func TestRemoveCurseUncursesOneItem(t *testing.T) {
	cursed := func() poolsave.Item {
		raw := make([]byte, 63)
		raw[gamepack.ItemCursedOffset] = 1
		raw[gamepack.ItemReadiedOffset] = 1
		return poolsave.Item{Name: "CURSED SWORD", Raw: raw}
	}
	for _, withCurseEffect := range []bool{false, true} {
		application, state := sideEffectBoard(t, int(gamepack.ClassSlotCleric), 5, gamepack.SpellIDRemoveCurse, 10)
		state.Roster[2].X = 6 // 碰觸（射程 1）
		ally := &application.state.Party[1]
		ally.Inventory = []poolsave.Item{{Name: "DAGGER", Raw: make([]byte, 63)}, cursed(), cursed()}
		if withCurseEffect {
			// 降咒（參數表 +0Ah）。戰場上讀的是盤面那一份（開打時從隊伍複製，#116），
			// 所以兩份一起掛，與正常開打後的狀態相同。
			ally.Effects = poolsave.PermanentEffects(0x24)
			state.Effects[2] = combatEffects(ally.Effects)
		}
		castAtTarget(t, application, state, 2)
		first, second := ally.Inventory[1].Raw[gamepack.ItemCursedOffset], ally.Inventory[2].Raw[gamepack.ItemCursedOffset]
		if withCurseEffect {
			if first == 0 || second == 0 || len(ally.Effects) != 0 {
				t.Fatalf("with 24h: only the effect should go (items %d %d, effects %+v, status %q)",
					first, second, ally.Effects, state.Status)
			}
			continue
		}
		if first != 0 || second == 0 {
			t.Fatalf("the first cursed item should be freed and only that one: %d %d (status %q)",
				first, second, state.Status)
		}
		if !strings.Contains(state.Status, "item is un-cursed") {
			t.Fatalf("no un-cursed message: %q", state.Status)
		}
	}
}

// 神殿的 Remove Curse（overlay-04 entry 9 `081Fh`）：身上有被詛咒的物品就算「有毛病」
// （`0845h` 先看物品，才問 24h），付 3500 之後交給同一支 `2508h`。與其他神殿測試
// （camp_test.go）同一個入口：selectSuneTempleOption 是 ENTER 在神殿裡分派到的那一支。
func TestTempleRemoveCurseFreesACursedItem(t *testing.T) {
	raw := make([]byte, 63)
	raw[gamepack.ItemCursedOffset] = 1
	character := poolsave.Character{Name: "HERO", MaxHP: 12, CurrentHP: 12,
		Money:     [7]uint16{3: 4000},
		Inventory: []poolsave.Item{{Name: "CURSED SWORD", Raw: raw}}}
	application := &app{
		roller: fixedTempleRoller(1),
		state: poolsave.State{Schema: poolsave.Schema,
			CharacterLibrary: []poolsave.Character{character},
			Party:            []poolsave.Character{character}},
		templeActive: true,
	}
	application.saveState = func(poolsave.State) error { return nil }
	application.enterTempleHeal()
	for index, id := range templeHealServiceIDs {
		if id == "remove-curse" {
			application.cellMenuCursor = index
		}
	}
	if err := application.selectSuneTempleOption(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(application.eventText, "3500 gold pieces") {
		t.Fatalf("a cursed item should count as a curse: text %q status %q", application.eventText, application.statusLine)
	}
	if err := application.selectSuneTempleOption(); err != nil {
		t.Fatal(err)
	}
	member := application.state.Party[0]
	// 神殿付完錢把餘額重鑄成白金＋金（overlay-04 經 entry 15，#79）：500 金 → 100 白金。
	left := int(member.Money[3]) + int(member.Money[4])*5
	if left != 500 || member.Inventory[0].Raw[gamepack.ItemCursedOffset] != 0 {
		t.Fatalf("paid %d, item curse %d", 4000-left, member.Inventory[0].Raw[gamepack.ItemCursedOffset])
	}
}
