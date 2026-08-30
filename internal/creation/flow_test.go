package creation

import "testing"

func TestFlowUsesRaceSpecificDOSOrder(t *testing.T) {
	flow := NewFlow()
	if err := flow.Select(3); err != nil {
		t.Fatal(err)
	} // Half-Elf
	if err := flow.Select(1); err != nil {
		t.Fatal(err)
	} // Female
	if got := flow.Options(); len(got) != 11 || got[5] != "Cleric/Fighter/Magic-User" {
		t.Fatalf("class options = %v", got)
	}
	if err := flow.Select(5); err != nil {
		t.Fatal(err)
	}
	if err := flow.Select(8); err != nil {
		t.Fatal(err)
	}
	if flow.Stage != StageRoll || flow.SelectedRace().DOSCode != 4 || flow.SelectedGender().DOSCode != 1 || flow.SelectedClass().ID != "cleric-fighter-magic-user" || flow.SelectedAlignment().ID != "chaotic-evil" {
		t.Fatalf("unexpected completed menu selections: %+v", flow)
	}
}

func TestFlowBackAndInvalidSelectionFailClosed(t *testing.T) {
	flow := NewFlow()
	if flow.Back() {
		t.Fatal("race screen unexpectedly consumed ESC")
	}
	if err := flow.Select(6); err == nil {
		t.Fatal("invalid race accepted")
	}
	if err := flow.Select(1); err != nil {
		t.Fatal(err)
	}
	if !flow.Back() || flow.Stage != StageRace {
		t.Fatalf("Back did not return to race: %+v", flow)
	}
}
