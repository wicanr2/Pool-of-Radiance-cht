package gamepack

import (
	"bytes"
	"path/filepath"
	"testing"
)

// `DS:05EAh` 的八格要與 START.EXE 逐位元組相同（overlay-16 `0D3Fh` 讀的那一張）。
// 正對照：同一個位移換算讀 `DS:3C16h` 要得到職業 THAC0 表第一列 `28 28 28 28 2A`。
func TestClassUseMasksMatchStartExe(t *testing.T) {
	raw, err := readStartExecutable(filepath.Join("..", "..", "Pool of Radiance (1988).zip"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	thac0 := raw[0x3c16+startDataSegmentFileDelta:][:5]
	if !bytes.Equal(thac0, []byte{0x28, 0x28, 0x28, 0x28, 0x2a}) {
		t.Fatalf("DS delta positive control failed: % X", thac0)
	}
	got := raw[ClassUseMaskAddress+startDataSegmentFileDelta:][:ClassThac0ClassCount]
	if !bytes.Equal(got, ClassUseMasks[:]) {
		t.Fatalf("DS:05EAh is % X, ClassUseMasks is % X", got, ClassUseMasks)
	}
}

func TestClassUseMaskAddsEveryClassWithALevel(t *testing.T) {
	var levels [ClassThac0ClassCount]uint8
	levels[ClassSlotFighter], levels[ClassSlotMagicUser] = 3, 2
	if got := ClassUseMask(levels); got != 0x08|0x01 {
		t.Fatalf("fighter/magic-user mask %02X, want 09", got)
	}
}

// itemTypesFixture 是一張只填了測試要用的幾格的型別表。
func itemTypesFixture() *ItemTypeTable {
	table := &ItemTypeTable{}
	set := func(index uint8, category, hands, mask uint8) {
		table.Entries[index].Raw[0] = category
		table.Entries[index].Raw[ItemTypeHandsOffset] = hands
		table.Entries[index].Raw[ItemTypeClassMaskOffset] = mask
	}
	set(0x01, 0, 1, 0x48) // 單手武器，戰士／聖騎士
	set(0x26, 0, 2, 0x48) // 雙手武器
	set(0x10, 1, 1, 0x4c) // 盾
	set(0x20, 9, 0, 0xff) // 戒指
	set(ItemTypeArrow, 0x0a, 0, 0x48)
	set(0x3d, 0x0b, 2, 0x01) // 法師卷軸
	return table
}

func itemRaw(itemType uint8, readied bool) []byte {
	raw := make([]byte, 63)
	raw[ItemTypeOffset] = itemType
	if readied {
		raw[ItemReadiedOffset] = 1
	}
	return raw
}

// overlay-19 entry 7 的判斷順序：手滿、那一格已有、職業，後面蓋前面。
func TestReadyItemFollowsTheOriginalOrder(t *testing.T) {
	types := itemTypesFixture()
	fighter := uint8(0x08)
	// 已經拿著單手武器，再裝一把：那一格有東西（2），不是自動換手。
	inventory := [][]byte{itemRaw(0x01, true), itemRaw(0x26, false)}
	result, err := ReadyItem(inventory, 1, types, fighter)
	if err != nil || result.Outcome != ReadyAlreadyUsing || result.Blocker != 0 || inventory[1][ItemReadiedOffset] != 0 {
		t.Fatalf("second weapon: %+v %v", result, err)
	}
	// 單手武器＋盾，再裝雙手的：手數 1+1+2 > 2，但武器那一格也有——2 蓋掉 3。
	inventory = [][]byte{itemRaw(0x01, true), itemRaw(0x10, true), itemRaw(0x26, false)}
	if result, _ := ReadyItem(inventory, 2, types, fighter); result.Outcome != ReadyAlreadyUsing {
		t.Fatalf("two-hander over sword and shield: %+v", result)
	}
	// 只有盾，裝雙手的：手滿了。
	inventory = [][]byte{itemRaw(0x10, true), itemRaw(0x26, false)}
	if result, _ := ReadyItem(inventory, 1, types, fighter); result.Outcome != ReadyHandsFull {
		t.Fatalf("two-hander over a shield: %+v", result)
	}
	// 法師拿不了戰士的武器，即使手是空的：職業蓋掉一切。
	inventory = [][]byte{itemRaw(0x01, true), itemRaw(0x26, false)}
	if result, _ := ReadyItem(inventory, 1, types, 0x01); result.Outcome != ReadyWrongClass {
		t.Fatalf("magic-user with a sword: %+v", result)
	}
	// 卸下再裝：成功，+34h 立起來。
	inventory = [][]byte{itemRaw(0x01, true), itemRaw(0x26, false)}
	if result, _ := ReadyItem(inventory, 0, types, fighter); result.Outcome != UnreadyDone || inventory[0][ItemReadiedOffset] != 0 {
		t.Fatalf("unready: %+v", result)
	}
	if result, _ := ReadyItem(inventory, 1, types, fighter); result.Outcome != ReadyDone || inventory[1][ItemReadiedOffset] != 1 {
		t.Fatalf("ready the two-hander: %+v", result)
	}
	// 被詛咒的卸不下來。
	cursed := itemRaw(0x01, true)
	cursed[ItemCursedOffset] = 1
	if result, _ := ReadyItem([][]byte{cursed}, 0, types, fighter); result.Outcome != ReadyCursed || cursed[ItemReadiedOffset] != 1 {
		t.Fatalf("cursed: %+v", result)
	}
	// 兩個戒指槽：第三個戒指擋在第一個（`15C0h` 問 +F4h、印 +F0h）。
	rings := [][]byte{itemRaw(0x20, true), itemRaw(0x20, true), itemRaw(0x20, false)}
	if result, _ := ReadyItem(rings, 2, types, fighter); result.Outcome != ReadyAlreadyUsing || result.Blocker != 0 {
		t.Fatalf("third ring: %+v", result)
	}
}

func TestHalveItemSplitsBehindTheOriginal(t *testing.T) {
	raw := itemRaw(ItemTypeArrow, true)
	raw[ItemCountOffset] = 11
	split, ok := HalveItem(raw)
	if !ok || raw[ItemCountOffset] != 6 || split[ItemCountOffset] != 5 || split[ItemReadiedOffset] != 0 {
		t.Fatalf("halve 11: %v %d %d", ok, raw[ItemCountOffset], split[ItemCountOffset])
	}
	one := itemRaw(ItemTypeArrow, false)
	one[ItemCountOffset] = 1
	if _, ok := HalveItem(one); ok {
		t.Fatal("a single arrow was halved")
	}
}

// 超過 255 時收件的那件設成 255，多出來的留在後面那一件、由它接著收。
func TestJoinItemsCapsAt255(t *testing.T) {
	stack := func(count, charges uint8) []byte {
		raw := itemRaw(ItemTypeArrow, false)
		raw[ItemCountOffset] = count
		raw[AIItemChargesOffset] = charges
		return raw
	}
	inventory := [][]byte{stack(5, 0), stack(5, 0), stack(250, 0), stack(7, 2), itemRaw(0x01, false)}
	removed := JoinItems(inventory, 0)
	if len(removed) != 1 || removed[0] != 1 {
		t.Fatalf("removed %v, want [1]", removed)
	}
	if inventory[0][ItemCountOffset] != 255 || inventory[2][ItemCountOffset] != 5 || inventory[3][ItemCountOffset] != 7 {
		t.Fatalf("counts %d %d %d", inventory[0][ItemCountOffset], inventory[2][ItemCountOffset], inventory[3][ItemCountOffset])
	}
}

func TestScrollReadAndErase(t *testing.T) {
	raw := itemRaw(0x3d, true)
	raw[ItemNameWordOffset+1] = 0xd3 // With 2 Spells
	raw[scrollSpellFirst] = SpellIDMagicMissile
	raw[scrollSpellFirst+1] = 0x80 | 21
	raw[ItemHiddenNameOffset] = 0x02
	if readable, _ := ScrollReadable(raw, 0x0b, false, 5); readable {
		t.Fatal("a cleric read a hidden magic-user scroll")
	}
	readable, revealed := ScrollReadable(raw, 0x0b, true, 0)
	if !readable || !revealed || raw[ItemHiddenNameOffset] != 0 {
		t.Fatal("read magic did not reveal the scroll")
	}
	spells, scribing := ScrollSpells(raw)
	if len(spells) != 2 || spells[0] != SpellIDMagicMissile || spells[1] != 21 || scribing[0] || !scribing[1] {
		t.Fatalf("scroll lines %v %v", spells, scribing)
	}
	if EraseScrollSpell(raw, SpellIDMagicMissile) || raw[scrollSpellFirst] != 0 || raw[ItemNameWordOffset+1] != 0xd2 {
		t.Fatalf("first erase: % X", raw[scrollSpellFirst:scrollSpellLast+1])
	}
	if !EraseScrollSpell(raw, 21) {
		t.Fatal("erasing the last line did not remove the scroll")
	}
}
