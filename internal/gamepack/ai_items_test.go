package gamepack

import "testing"

func aiItem(spell, guard uint8, readied bool) []byte {
	raw := make([]byte, 63)
	raw[AIItemSpellOffset] = spell
	raw[AIItemGuardOffset] = guard
	if readied {
		raw[ItemReadiedOffset] = 1
	}
	return raw
}

// overlay-09 entry 3 的四道過濾與 `04C7h` 的換算。
func TestAIItemSpellFiltersAndFolds(t *testing.T) {
	for _, tc := range []struct {
		name   string
		raw    []byte
		scroll bool
		want   uint8
		ok     bool
	}{
		{"plain", aiItem(0x0F, 0, true), false, 0x0F, true},
		{"38h stays", aiItem(0x38, 0, true), false, 0x38, true},
		{"39h folds to 22h", aiItem(0x39, 0, true), false, 0x22, true},
		{"scroll", aiItem(0x0F, 0, true), true, 0, false},
		{"guard bit 7", aiItem(0x0F, 0x80, true), false, 0, false},
		{"guard 7Fh passes", aiItem(0x0F, 0x7F, true), false, 0x0F, true},
		{"not readied", aiItem(0x0F, 0, false), false, 0, false},
		{"no spell", aiItem(0, 0, true), false, 0, false},
		{"short record", make([]byte, 0x3E), false, 0, false},
	} {
		got, ok := AIItemSpell(tc.raw, tc.scroll)
		if got != tc.want || ok != tc.ok {
			t.Errorf("%s: got %#x %v, want %#x %v", tc.name, got, ok, tc.want, tc.ok)
		}
	}
}

// 門檻 7 起、每輪減一；同一輪依串列順序取第一件；不擲骰（accept 以外沒有呼叫）。
func TestChooseAIItemLowersTheThresholdEachRound(t *testing.T) {
	candidates := []AIItemCandidate{{Index: 0, Spell: 1}, {Index: 2, Spell: 2}}
	var asked []uint8
	chosen, ok := ChooseAIItem(candidates, 3, func(spell, threshold uint8) bool {
		asked = append(asked, threshold)
		return spell == 2 && threshold == 5
	})
	if !ok || chosen.Index != 2 {
		t.Fatalf("chose %+v %v, want the second item in round 3", chosen, ok)
	}
	want := []uint8{7, 7, 6, 6, 5, 5}
	if len(asked) != len(want) {
		t.Fatalf("thresholds %v, want %v", asked, want)
	}
	for index := range want {
		if asked[index] != want[index] {
			t.Fatalf("thresholds %v, want %v", asked, want)
		}
	}
	if _, ok := ChooseAIItem(candidates, 1, func(_, threshold uint8) bool { return threshold < 7 }); ok {
		t.Fatal("one round only tries threshold 7")
	}
}

// overlay-19 `1C3Fh..1C7Bh`。
func TestSpendAIItemUse(t *testing.T) {
	unlimited := aiItem(0x0F, 0, true)
	if SpendAIItemUse(unlimited) || unlimited[AIItemChargesOffset] != 0 {
		t.Fatal("charges 0 means the item never runs out")
	}
	stack := aiItem(0x0F, 0, true)
	stack[AIItemChargesOffset], stack[ItemCountOffset] = 1, 3
	if SpendAIItemUse(stack) || stack[ItemCountOffset] != 2 || stack[AIItemChargesOffset] != 1 {
		t.Fatalf("a stack loses one of its count: %v", stack[0x39:0x3E])
	}
	wand := aiItem(0x0F, 0, true)
	wand[AIItemChargesOffset], wand[ItemCountOffset] = 2, 1
	if SpendAIItemUse(wand) || wand[AIItemChargesOffset] != 1 {
		t.Fatal("a wand loses a charge")
	}
	if !SpendAIItemUse(wand) || wand[AIItemChargesOffset] != 0 {
		t.Fatal("the last charge removes the wand")
	}
}
