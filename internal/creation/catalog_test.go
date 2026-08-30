package creation

import "testing"

func TestRaceClassMenusMatchDOSOracle(t *testing.T) {
	want := map[string][]string{
		"dwarf":    {"Fighter", "Thief", "Fighter/Thief"},
		"elf":      {"Fighter", "Magic-User", "Thief", "Fighter/Magic-User", "Fighter/Thief", "Fighter/Magic-User/Thief", "Magic-User/Thief"},
		"gnome":    {"Fighter", "Thief", "Fighter/Thief"},
		"half-elf": {"Cleric", "Fighter", "Magic-User", "Thief", "Cleric/Fighter", "Cleric/Fighter/Magic-User", "Cleric/Magic-User", "Fighter/Magic-User", "Fighter/Thief", "Fighter/Magic-User/Thief", "Magic-User/Thief"},
		"halfling": {"Fighter", "Thief", "Fighter/Thief"},
		"human":    {"Cleric", "Fighter", "Magic-User", "Thief"},
	}
	if len(Races) != 6 {
		t.Fatalf("race count = %d", len(Races))
	}
	for _, race := range Races {
		got := ClassesForRace(race.ID)
		if len(got) != len(want[race.ID]) {
			t.Fatalf("%s class count = %d", race.ID, len(got))
		}
		for index := range got {
			if got[index].Label != want[race.ID][index] {
				t.Fatalf("%s class %d = %q", race.ID, index, got[index].Label)
			}
		}
	}
}

func TestCatalogReturnsCopiesAndGuidance(t *testing.T) {
	got := ClassesForRace("elf")
	got[0].Label = "changed"
	got[0].Components[0] = "changed"
	if ClassesForRace("elf")[0].Label != "Fighter" {
		t.Fatal("caller mutated class catalog")
	}
	if ClassesForRace("elf")[0].Components[0] != "fighter" {
		t.Fatal("caller mutated nested class components")
	}
	for _, stage := range []string{"race", "class", "alignment", "portrait", "icon"} {
		if HintFor(stage) == "" {
			t.Fatalf("missing hint for %s", stage)
		}
	}
}

func TestClassDOSCodesMatchPairedOriginalCharacterFiles(t *testing.T) {
	want := map[string]uint8{
		"cleric": 0, "fighter": 2, "magic-user": 5, "thief": 6,
		"cleric-fighter": 8, "cleric-fighter-magic-user": 9,
		"cleric-magic-user": 11, "fighter-magic-user": 13,
		"fighter-thief": 14, "fighter-magic-user-thief": 15,
		"magic-user-thief": 16,
	}
	seen := map[string]bool{}
	for _, race := range Races {
		for _, class := range ClassesForRace(race.ID) {
			code, ok := want[class.ID]
			if !ok {
				t.Fatalf("missing original code for %s", class.ID)
			}
			if class.DOSCode != code {
				t.Fatalf("%s DOS code=%d want %d", class.ID, class.DOSCode, code)
			}
			seen[class.ID] = true
		}
	}
	if len(seen) != len(want) {
		t.Fatalf("covered %d class identities, want %d", len(seen), len(want))
	}
}
