package gamepack

import (
	"path/filepath"
	"strings"
	"testing"
)

// 參數表 `+0Bh`..`+0Dh` 在 AI 那一支的讀法（spec 096 entry 4）。正對照挑幾條叫得出
// 名字的：催眠、火球、定身是 7，魔法飛彈 6，治療輕傷 1；營地法術全是 0。
func TestAISpellFieldsFromTheOriginalTable(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	table, err := ReadDOSSpellParameters(zipPath)
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	for _, want := range []struct {
		id       int
		priority uint8
		cost     uint8
		self     bool
	}{
		{1, 2, 3, true},   // Bless：+0Ch 0Ah ÷ 3
		{3, 1, 1, true},   // Cure Light Wounds
		{15, 6, 0, false}, // Magic Missile
		{21, 7, 0, false}, // Sleep
		{23, 7, 1, false}, // Hold Person
		{47, 7, 1, false}, // Fireball
	} {
		p := table[want.id]
		if p.AIPriority() != want.priority || p.CastingCost() != want.cost || p.TargetsCaster() != want.self {
			t.Errorf("spell %d: priority %d cost %d self %v, want %d %d %v", want.id,
				p.AIPriority(), p.CastingCost(), p.TargetsCaster(), want.priority, want.cost, want.self)
		}
	}
	// 營地法術 AI 永遠挑不到：優先度都是 0，而門檻最低降到 1（Roll(1,7) 最多七輪）。
	camp := 0
	for id := 1; id < len(table); id++ {
		if !table[id].CampOnly() {
			continue
		}
		camp++
		if table[id].AIPriority() != 0 {
			t.Errorf("camp-only spell %d has AI priority %d", id, table[id].AIPriority())
		}
	}
	if camp == 0 {
		t.Fatal("no camp-only spell in the table: the +0Bh reading is wrong")
	}
}

// 怪物記錄的法術陣列從 `+17h` 起算（overlay-09 `0557h`），LEVEL 3 MU 的三條放在
// `+18h..+1Ah`：魔法飛彈、催眠、臭雲。
func TestMonsterSpellArrayStartsAt17h(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	record, err := ReadDOSMonsterRecord(zipPath, 2, 94)
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	if !strings.Contains(record.Name, "LEVEL 3 MU") {
		t.Fatalf("mon2/94 is %q, not the level 3 magic-user", record.Name)
	}
	list := AISpellList(record.Raw[AISpellArrayOffset : AISpellArrayOffset+AISpellArraySlots])
	if len(list) != 3 || list[0] != 0x0F || list[1] != 0x15 || list[2] != 0x22 {
		t.Fatalf("mon2/94 spells %x, want 0f 15 22", list)
	}
}

// 挑法的形狀：先擲次數（沒有法術也擲）、每輪三擲、門檻從 7 往下降。
func TestChooseAISpellLowersTheThreshold(t *testing.T) {
	priorities := map[uint8]uint8{0x0F: 6, 0x15: 7}
	accept := func(id, threshold uint8) bool { return priorities[id] >= threshold }
	script := func(values ...int) (func(int, int) int, *[]int) {
		asked := []int{}
		return func(_, sides int) int {
			asked = append(asked, sides)
			value := values[0]
			values = values[1:]
			return value
		}, &asked
	}

	// 沒有法術：次數骰照擲，之後什麼都不擲。
	roll, asked := script(5)
	if got := ChooseAISpell(nil, true, roll, accept); got != 0 || len(*asked) != 1 || (*asked)[0] != 7 {
		t.Fatalf("empty list: got %d asked %v", got, *asked)
	}
	// Magic Off（allowed=false）一樣只擲次數。
	roll, asked = script(5)
	if got := ChooseAISpell([]uint8{0x0F}, false, roll, accept); got != 0 || len(*asked) != 1 {
		t.Fatalf("not allowed: got %d asked %v", got, *asked)
	}
	// 次數 1：只有門檻 7 那一輪，擲三次都是魔法飛彈（6）→ 不放。
	roll, asked = script(1, 1, 1, 1)
	if got := ChooseAISpell([]uint8{0x0F, 0x15}, true, roll, accept); got != 0 || len(*asked) != 4 {
		t.Fatalf("one pass: got %d asked %v", got, *asked)
	}
	// 次數 2：第二輪門檻 6，第一擲就收魔法飛彈。
	roll, _ = script(2, 1, 1, 1, 1)
	if got := ChooseAISpell([]uint8{0x0F, 0x15}, true, roll, accept); got != 0x0F {
		t.Fatalf("two passes: got %d", got)
	}
	// 第一輪第三擲擲到催眠（7）就收。
	roll, _ = script(1, 1, 1, 2)
	if got := ChooseAISpell([]uint8{0x0F, 0x15}, true, roll, accept); got != 0x15 {
		t.Fatalf("third pick: got %d", got)
	}
}

// 陣列裡第 7 位立著的格子照樣佔一個號碼（`0569h` 比的是整個 byte）。
func TestAISpellListKeepsFlaggedSlots(t *testing.T) {
	list := AISpellList([]uint8{0, 0x0F, 0, 0x8F, 0x15})
	if len(list) != 3 || list[1] != 0x8F {
		t.Fatalf("list %x", list)
	}
}
