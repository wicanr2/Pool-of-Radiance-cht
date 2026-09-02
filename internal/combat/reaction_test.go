package combat

import (
	"reflect"
	"testing"
)

// DS:2880h..2883h 的四個代碼。
func TestDisablingEffectCodesMatchTheDump(t *testing.T) {
	want := [4]uint8{0x33, 0x34, 0x35, 0x1F}
	if DisablingEffectCodes != want {
		t.Fatalf("codes %v, want %v", DisablingEffectCodes, want)
	}
}

func TestIsReactionDisabledNeedsOneOfTheFourCodes(t *testing.T) {
	if IsReactionDisabled([]uint8{0x10, 0x20}) {
		t.Fatal("unrelated effects disabled the reaction")
	}
	for _, code := range DisablingEffectCodes {
		if !IsReactionDisabled([]uint8{0x10, code, 0x20}) {
			t.Fatalf("code %02X did not disable the reaction", code)
		}
	}
	if IsReactionDisabled(nil) {
		t.Fatal("an empty effect list disabled the reaction")
	}
}

// 由目前朝向起算 +6..+10，取模 8 就是前後兩格。
func TestReactionFacingsCoverTheNeighbourhood(t *testing.T) {
	got := ReactionFacings(0)
	want := [ReactionFacingWindow]uint8{6, 7, 0, 1, 2}
	if got != want {
		t.Fatalf("facings %v, want %v", got, want)
	}
	got = ReactionFacings(5)
	want = [ReactionFacingWindow]uint8{3, 4, 5, 6, 7}
	if got != want {
		t.Fatalf("facings %v, want %v", got, want)
	}
}

func TestSelectReactionAttackSlotFallsBackToTheSecondSlot(t *testing.T) {
	got := SelectReactionAttackSlot(0, [2]uint8{0, 0})
	if got.Slot != 2 {
		t.Fatalf("slot %d, want 2", got.Slot)
	}
	if got.PhaseCounts != [2]uint8{0, 1} {
		t.Fatalf("phase counts %v, want the chosen slot topped up to 1", got.PhaseCounts)
	}
}

func TestSelectReactionAttackSlotPrefersThePrimaryWhenItHasARate(t *testing.T) {
	got := SelectReactionAttackSlot(2, [2]uint8{0, 0})
	if got.Slot != 1 || got.PhaseCounts != [2]uint8{1, 0} {
		t.Fatalf("got %+v, want slot 1 topped up to 1", got)
	}
}

// 計數大於 0 的槽會覆蓋預設，且後看到的贏。
func TestSelectReactionAttackSlotPrefersASlotThatStillHasPhases(t *testing.T) {
	got := SelectReactionAttackSlot(2, [2]uint8{0, 3})
	if got.Slot != 2 {
		t.Fatalf("slot %d, want 2", got.Slot)
	}
	if got.PhaseCounts != [2]uint8{0, 3} {
		t.Fatalf("phase counts %v, want them untouched", got.PhaseCounts)
	}
	got = SelectReactionAttackSlot(0, [2]uint8{4, 5})
	if got.Slot != 2 {
		t.Fatalf("slot %d, want the later slot to win", got.Slot)
	}
}

func TestLeavingOpponentsIsTheDifference(t *testing.T) {
	leaving := LeavingOpponents([]uint8{3, 7, 9}, []uint8{7})
	if !reflect.DeepEqual(leaving, []uint8{3, 9}) {
		t.Fatalf("leaving %v, want [3 9]", leaving)
	}
	leaving = LeavingOpponents([]uint8{3, 7}, []uint8{3, 7})
	if len(leaving) != 0 {
		t.Fatalf("leaving %v, want none", leaving)
	}
	leaving = LeavingOpponents(nil, []uint8{3})
	if len(leaving) != 0 {
		t.Fatalf("leaving %v, want none", leaving)
	}
}

func TestHasEffectWalksTheList(t *testing.T) {
	if !HasEffect([]uint8{1, 2, 0x33}, 0x33) {
		t.Fatal("a code at the end of the list was missed")
	}
	if HasEffect([]uint8{1, 2}, 0x33) {
		t.Fatal("an absent code was reported present")
	}
}
