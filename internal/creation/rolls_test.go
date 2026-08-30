package creation

import "testing"

type fixedRoller struct {
	values []int
	calls  [][2]int
}

func (r *fixedRoller) Roll(count, sides int) int {
	r.calls = append(r.calls, [2]int{count, sides})
	value := r.values[0]
	r.values = r.values[1:]
	return value
}

func TestSingleClassUsesRandomAgeAndOriginalGoldHPDice(t *testing.T) {
	r := &fixedRoller{values: []int{12, 10, 11, 12, 13, 14, 15, 12, 8}}
	got := RollCharacter(r, Races[5], Genders[0], ClassesForRace("human")[1])
	if got.Age != 27 {
		t.Fatalf("age=%d", got.Age)
	}
	if got.Gold != 120 || got.RawHP != 8 || got.HP != 8 {
		t.Fatalf("gold/rawHP/HP=%d/%d/%d", got.Gold, got.RawHP, got.HP)
	}
	if r.calls[0] != [2]int{1, 4} || r.calls[7] != [2]int{5, 4} || r.calls[8] != [2]int{1, 10} {
		t.Fatalf("calls=%v", r.calls)
	}
}

func TestMulticlassAgeIsMaximumAndHPIsAveraged(t *testing.T) {
	class := ClassesForRace("half-elf")[5]
	r := &fixedRoller{values: []int{9, 10, 11, 12, 13, 14, 10, 8, 12, 6, 6, 4}}
	got := RollCharacter(r, Races[3], Genders[0], class)
	if got.Age != 48 {
		t.Fatalf("age=%d", got.Age)
	}
	if r.calls[0] == [2]int{2, 4} {
		t.Fatal("multiclass age must not consume a roll")
	}
	if got.Gold != 93 {
		t.Fatalf("gold=%d calls=%v", got.Gold, r.calls)
	}
	if got.RawHP != 6 || got.HP != 6 {
		t.Fatalf("rawHP/HP=%d/%d", got.RawHP, got.HP)
	}
}

func TestRacialClassLimitsAndExceptionalStrengthCap(t *testing.T) {
	r := &fixedRoller{values: []int{20, 18, 3, 3, 3, 3, 3, 100, 15, 10}}
	got := RollCharacter(r, Races[1], Genders[0], ClassesForRace("elf")[0])
	if got.Abilities != [6]int{18, 8, 6, 7, 7, 8} {
		t.Fatalf("abilities=%v", got.Abilities)
	}
	if got.ExceptionalStrength != 75 {
		t.Fatalf("exceptional=%d", got.ExceptionalStrength)
	}
}

func TestPureFighterGetsExceptionalConstitutionBonusOnly(t *testing.T) {
	if got := constitutionModifier(18, 2); got != 4 {
		t.Fatalf("pure fighter CON18=%d", got)
	}
	if got := constitutionModifier(18, 13); got != 2 {
		t.Fatalf("multiclass fighter CON18=%d", got)
	}
}
