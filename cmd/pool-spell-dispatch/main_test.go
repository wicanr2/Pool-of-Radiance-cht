package main

import "testing"

// 法術編號是 1-based（spec 070）。差一格的名字比沒有名字更糟：報表看起來
// 完整，每一條都指到隔壁那個法術。
func TestSpellNameIsOneBased(t *testing.T) {
	names := []string{"Bless", "Burning Hands", "Charm Person"}
	if got := spellName(names, 1); got != "Bless" {
		t.Errorf("編號 1 查到 %q", got)
	}
	if got := spellName(names, 3); got != "Charm Person" {
		t.Errorf("編號 3 查到 %q", got)
	}
	for _, id := range []int{0, -1, 4, 999} {
		if got := spellName(names, id); got != "" {
			t.Errorf("編號 %d 超出範圍卻查到 %q", id, got)
		}
	}
}

// 沒解讀的欄位要原樣留在證據裡，所以格式固定：小寫、每個位元組兩位、空格分隔。
func TestHexBytesKeepsEveryByte(t *testing.T) {
	if got := hexBytes([]byte{0x00, 0x0f, 0xa5, 0xff}); got != "00 0f a5 ff" {
		t.Fatalf("印成 %q", got)
	}
	if got := hexBytes(nil); got != "" {
		t.Fatalf("空的印成 %q", got)
	}
}

// 共用同一支處理常式的法術照**第一個法術編號**排，報表才逐次可比。
func TestSortGroupsOrdersByFirstSpellID(t *testing.T) {
	groups := []group{
		{CodeOffset: "0x30", SpellIDs: []int{9, 12}},
		{CodeOffset: "0x10", SpellIDs: []int{2}},
		{CodeOffset: "0x20", SpellIDs: []int{5, 7}},
	}
	sortGroups(groups)
	want := []int{2, 5, 9}
	for index, group := range groups {
		if group.SpellIDs[0] != want[index] {
			t.Fatalf("第 %d 組的第一個編號是 %d，該是 %d", index, group.SpellIDs[0], want[index])
		}
	}
	// 已經排好的不該被打亂。
	sortGroups(groups)
	for index, group := range groups {
		if group.SpellIDs[0] != want[index] {
			t.Fatalf("排第二次之後第 %d 組是 %d", index, group.SpellIDs[0])
		}
	}
}
