package gamepack_test

import (
	"strings"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// 表有 68 筆，編號 0 那筆是走不到的哨兵。
func TestSpellParameterTableShape(t *testing.T) {
	table, err := gamepack.ReadDOSSpellParameters(dosZIP)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	if len(table) != gamepack.SpellParameterCount {
		t.Fatalf("parameter table has %d records, want %d", len(table), gamepack.SpellParameterCount)
	}
	for index, value := range table[0].Raw[:15] {
		if value != 0 {
			t.Fatalf("record 0 byte %d is %#02x, want zero", index, value)
		}
	}
}

// `+0` 為 FFh 的正好是五個「碰觸才生效」的法術。四個 Cause 系加上
// Shocking Grasp，一個不多一個不少——這五個在規則書裡都要先摸到對手。
func TestSpellParameterAttackRollMarksTheTouchSpells(t *testing.T) {
	table, err := gamepack.ReadDOSSpellParameters(dosZIP)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	names, err := gamepack.ReadDOSSpellNames(dosZIP)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	want := map[string]bool{
		"Cause Light Wounds": true,
		"Shocking Grasp":     true,
		"Cause Blindness":    true,
		"Cause Disease":      true,
		"Bestow Curse":       true,
	}
	got := make(map[string]bool, len(want))
	for _, record := range table {
		if !record.RequiresAttackRoll() {
			continue
		}
		if record.SpellID > len(names) {
			t.Fatalf("unnamed spell %d requires an attack roll", record.SpellID)
		}
		got[strings.TrimSpace(names[record.SpellID-1])] = true
	}
	if len(got) != len(want) {
		t.Fatalf("attack-roll spells are %v, want %v", got, want)
	}
	for name := range want {
		if !got[name] {
			t.Fatalf("%s does not require an attack roll", name)
		}
	}
}

// 效果碼為零的，正好是「打完就結束、不留狀態」的那一批：治療與致傷、
// 三個純傷害、Knock、解除魔法、解咒、復原。留下狀態的法術一個都不在裡面
// ——Cure Blindness 在（它是把狀態拿掉），Cause Blindness 不在（它留下失明）。
func TestSpellParameterZeroEffectCodeMeansNoLingeringCondition(t *testing.T) {
	table, err := gamepack.ReadDOSSpellParameters(dosZIP)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	names, err := gamepack.ReadDOSSpellNames(dosZIP)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	instantaneous := map[string]bool{
		"Cure Light Wounds": true, "Cause Light Wounds": true, "Burning Hands": true,
		"Magic Missile": true, "Shocking Grasp": true, "Knock": true,
		"Cure Blindness": true, "Cure Disease": true, "Dispel Magic": true,
		"Remove Curse": true, "Fireball": true, "Lightning Bolt": true,
		"Restoration": true,
	}
	for _, record := range table {
		if record.SpellID < 1 || record.SpellID > len(names) {
			continue
		}
		name := strings.TrimSpace(names[record.SpellID-1])
		if (record.EffectCode() == 0) != instantaneous[name] {
			t.Fatalf("%s has effect code %#02x but instantaneous=%v", name, record.EffectCode(), instantaneous[name])
		}
	}
}

// 兩張表要對得起來：完全同名的法術共用一支處理常式（spec 073），效果碼也
// 必須相同。一邊對、一邊不對，就表示其中一張表的索引讀錯了。
func TestSpellParametersAgreeWithTheDispatchTable(t *testing.T) {
	table, err := gamepack.ReadDOSSpellParameters(dosZIP)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	dispatch, err := gamepack.ReadDOSSpellDispatchTable(dosZIP)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	names, err := gamepack.ReadDOSSpellNames(dosZIP)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	handler := make(map[int]uint16, len(dispatch))
	for _, entry := range dispatch {
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
			if table[id].EffectCode() != table[ids[0]].EffectCode() {
				t.Fatalf("%q is spell %v with effect codes %#02x and %#02x", name, ids, table[ids[0]].EffectCode(), table[id].EffectCode())
			}
			if handler[id] != handler[ids[0]] {
				t.Fatalf("%q is spell %v with handlers %#04x and %#04x", name, ids, handler[ids[0]], handler[id])
			}
		}
	}
	if pairs != 5 {
		t.Fatalf("the name table has %d repeated names, want 5", pairs)
	}
}

// 效果訊息是往回找出來的，所以要有一道獨立的核對：訊息必須和法術名對得上。
func TestSpellDispatchMessagesMatchTheirSpells(t *testing.T) {
	dispatch, err := gamepack.ReadDOSSpellDispatchTable(dosZIP)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	want := map[int]string{
		1: "is Blessed", 2: "is Cursed", 10: "is charmed", 21: "falls asleep",
		23: "is held", 30: "is invisible", 38: "is blind", 48: "is Hasted",
		55: "is Slowed", 61: "is paralyzed",
	}
	got := make(map[int]string, len(dispatch))
	messages := 0
	for _, entry := range dispatch {
		got[entry.SpellID] = entry.Message
		if entry.Message != "" {
			messages++
		}
	}
	for id, text := range want {
		if got[id] != text {
			t.Fatalf("spell %d says %q, want %q", id, got[id], text)
		}
	}
	// 41 支常式帶訊息；其中五支由多個編號共用（偵測系 5、防護系 7、定身 2、
	// 隱形 2、解除魔法 2），攤回編號就是 36 + 18 = 54 個。
	if messages != 54 {
		t.Fatalf("%d spells carry a message, want 54", messages)
	}
}
