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
	if ClassesForRace("elf")[0].Label != "Fighter" {
		t.Fatal("caller mutated class catalog")
	}
	for _, stage := range []string{"race", "class", "alignment", "portrait", "icon"} {
		if HintFor(stage) == "" {
			t.Fatalf("missing hint for %s", stage)
		}
	}
}
