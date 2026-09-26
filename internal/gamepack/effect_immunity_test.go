package gamepack

import "testing"

// 群組 9 逐碼的正例與反例（處理常式位址見 effect_immunity.go）。
func TestSpellEffectImmunityFollowsTheHandlers(t *testing.T) {
	fixed := func(value int) func(int, int) int { return func(int, int) int { return value } }
	for _, tc := range []struct {
		name    string
		carried uint8
		code    uint8
		flags   uint8
		level   int
		roll    int
		want    bool
	}{
		{"nothing", 0, BlessEffectCode, 0, 5, 1, false},
		{"69h level 11 roll 50", 0x69, BlessEffectCode, 0, 11, 50, true},
		{"69h level 11 roll 51", 0x69, BlessEffectCode, 0, 11, 51, false},
		{"69h level 13 roll 60", 0x69, BlessEffectCode, 0, 13, 60, true},
		{"6Ah level 8 threshold 0", 0x6a, BlessEffectCode, 0, 8, 1, false},
		{"6Ah level 7 wraps", 0x6a, BlessEffectCode, 0, 7, 100, true},
		{"6Bh sleep 90", 0x6b, SleepEffectCode, 0, 5, 90, true},
		{"6Bh sleep 91", 0x6b, SleepEffectCode, 0, 5, 91, false},
		{"6Bh bless", 0x6b, BlessEffectCode, 0, 5, 1, false},
		{"6Ch charm", 0x6c, CharmPersonEffectCode, 0, 5, 1, true},
		{"6Ch hold", 0x6c, HoldPersonEffectCode, 0, 5, 1, false},
		{"6Dh hold", 0x6d, HoldPersonEffectCode, 0, 5, 1, true},
		{"6Dh charm", 0x6d, CharmPersonEffectCode, 0, 5, 1, false},
		{"6Eh cold", 0x6e, BlessEffectCode, ImmunityColdFlag, 5, 1, true},
		{"6Eh no flag", 0x6e, BlessEffectCode, ImmunityFireFlag, 5, 1, false},
		{"6Fh poison", 0x6f, 0x37, 0, 5, 1, true},
		{"6Fh sleep", 0x6f, SleepEffectCode, 0, 5, 1, false},
		{"70h fire", 0x70, BlessEffectCode, ImmunityFireFlag, 5, 1, true},
		{"70h no flag", 0x70, BlessEffectCode, ImmunityColdFlag, 5, 1, false},
		{"7Ch charm 30", 0x7c, CharmPersonEffectCode, 0, 5, 30, true},
		{"7Ch charm 31", 0x7c, CharmPersonEffectCode, 0, 5, 31, false},
		{"7Dh poison", 0x7d, 0x37, 0, 5, 1, true},
		{"7Dh bless", 0x7d, BlessEffectCode, 0, 5, 1, false},
	} {
		var list EffectList
		if tc.carried != 0 {
			list = list.Append(NewEffectNode(tc.carried, 0, 0, false))
		}
		if got := SpellEffectImmunity(list, tc.code, tc.flags, tc.level, fixed(tc.roll)); got != tc.want {
			t.Errorf("%s: immune %v, want %v", tc.name, got, tc.want)
		}
	}
}

// 魔法抗性只在 `6775h` 還沒被清掉、或 `6777h` 帶位元 3 時才擲骰：前面的碼已經擋掉時
// 不再消耗骰子。
func TestMagicResistanceRollsOnlyWhilePending(t *testing.T) {
	rolls := 0
	counting := func(int, int) int { rolls++; return 100 }
	list := EffectList{}.Append(NewEffectNode(0x69, 0, 0, false))
	SpellEffectImmunity(list, BlessEffectCode, 0, 11, counting)
	if rolls != 1 {
		t.Fatalf("pending code: %d rolls, want 1", rolls)
	}
	rolls = 0
	SpellEffectImmunity(list, 0, 0, 11, counting)
	if rolls != 0 {
		t.Fatalf("nothing pending: %d rolls, want 0", rolls)
	}
	SpellEffectImmunity(list, 0, ImmunityMagicFlag, 11, counting)
	if rolls != 1 {
		t.Fatalf("flag 8: %d rolls, want 1", rolls)
	}
}
