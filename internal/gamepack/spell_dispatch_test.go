package gamepack_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// 表要剛好 67 格、編號 1..67 一格不缺，而且每一格都指到 overlay-22 的
// 一支真的進入點。
func TestSpellDispatchTableCoversEveryIdentifier(t *testing.T) {
	table, err := gamepack.ReadDOSSpellDispatchTable(dosZIP)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	if len(table) != gamepack.SpellDispatchCount {
		t.Fatalf("dispatch table has %d slots, want %d", len(table), gamepack.SpellDispatchCount)
	}
	for index, entry := range table {
		if entry.SpellID != index+1 {
			t.Fatalf("slot %d carries spell %d", index, entry.SpellID)
		}
		if entry.CodeOffset == 0 {
			t.Fatalf("spell %d has no handler", entry.SpellID)
		}
	}
}

// 共用處理常式的組合，正好落在名稱表裡語意重複的那幾組——牧師與法師各有一份
// 的偵測系、七個防護系、定身術、隱形術、解除魔法。六組全中、一組沒有例外，
// 這就是「這張表是依法術編號派發」最硬的證據；換成別的意義不會這樣對齊。
func TestSpellDispatchSharedHandlersMatchTheDuplicateSpells(t *testing.T) {
	table, err := gamepack.ReadDOSSpellDispatchTable(dosZIP)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	want := [][]int{
		{5, 11, 18, 22, 29},         // Detect Magic ×2、Read Magic、Find Traps、Detect Invisibility
		{6, 7, 16, 17, 52, 53, 54},  // 七個 Protection From …
		{23, 49},                    // Hold Person（牧師／法師）
		{30, 50},                    // Invisibility 與 Invisibility, 10' Radius
		{41, 46},                    // Dispel Magic（牧師／法師）
		{47, 64},                    // Fireball 與編號 64 的無名效果
	}
	got := make([][]int, 0, len(want))
	for _, ids := range gamepack.SpellDispatchGroups(table) {
		got = append(got, ids)
	}
	if len(got) != len(want) {
		t.Fatalf("%d handlers are shared, want %d: %v", len(got), len(want), got)
	}
	for _, group := range want {
		matched := false
		for _, candidate := range got {
			if reflect.DeepEqual(candidate, group) {
				matched = true
				break
			}
		}
		if !matched {
			t.Fatalf("spells %v do not share one handler; shared groups are %v", group, got)
		}
	}
}

// 名稱一模一樣的法術必須共用同一支處理常式。這是可以被推翻的預測：牧師與
// 法師各有一份的 Detect Magic、Protection From Evil、Protection from Good、
// Hold Person、Dispel Magic，五對只要有一對落在不同常式上，「依法術編號派發」
// 的讀法就不成立。
//
// 反過來不成立，所以不檢查：Detect Magic 與 Find Traps 名稱不同卻共用一支，
// 因為它們在戰鬥裡同樣不做事。
func TestSpellDispatchIdenticalNamesShareOneHandler(t *testing.T) {
	names, err := gamepack.ReadDOSSpellNames(dosZIP)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	table, err := gamepack.ReadDOSSpellDispatchTable(dosZIP)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	handler := make(map[int]uint16, len(table))
	for _, entry := range table {
		handler[entry.SpellID] = entry.CodeOffset
	}
	byName := make(map[string][]int, len(names))
	for index, name := range names {
		key := strings.ToLower(strings.TrimSpace(name))
		byName[key] = append(byName[key], index+1)
	}
	pairs := 0
	for name, ids := range byName {
		if len(ids) < 2 {
			continue
		}
		pairs++
		for _, id := range ids[1:] {
			if handler[id] != handler[ids[0]] {
				t.Fatalf("%q is spell %v but dispatches to %#04x and %#04x", name, ids, handler[ids[0]], handler[id])
			}
		}
	}
	if pairs != 5 {
		t.Fatalf("the name table has %d repeated names, want 5", pairs)
	}
}

// 除了那六組，其餘每個編號都有自己的處理常式。
func TestSpellDispatchHandlersAreOtherwiseDistinct(t *testing.T) {
	table, err := gamepack.ReadDOSSpellDispatchTable(dosZIP)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	distinct := make(map[uint16]struct{}, len(table))
	for _, entry := range table {
		distinct[entry.CodeOffset] = struct{}{}
	}
	// 67 格扣掉六組共用多出來的 14 格。
	if len(distinct) != 53 {
		t.Fatalf("table uses %d distinct handlers, want 53", len(distinct))
	}
}
