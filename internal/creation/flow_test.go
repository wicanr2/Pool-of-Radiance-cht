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

func TestFlowOnlyRollsAfterAllDOSMenusAreAccepted(t *testing.T) {
	flow := NewFlow()
	roller := &fixedRoller{values: []int{}}
	if _, err := flow.Roll(roller); err == nil {
		t.Fatal("rolled before menu completion")
	}
	for _, selected := range []int{5, 0, 1, 0} {
		if err := flow.Select(selected); err != nil {
			t.Fatal(err)
		}
	}
	roller.values = []int{4, 10, 11, 12, 13, 14, 15, 12, 8}
	got, err := flow.Roll(roller)
	if err != nil {
		t.Fatal(err)
	}
	if got.Age != 19 || got.Gold != 120 || got.HP != 8 {
		t.Fatalf("rolled character = %+v", got)
	}
}

func TestFlowContinuesThroughOriginalNameAndPortraitOrder(t *testing.T) {
	flow := NewFlow()
	for _, selected := range []int{0, 0, 0, 0} {
		if err := flow.Select(selected); err != nil {
			t.Fatal(err)
		}
	}
	if err := flow.AcceptRoll(); err != nil || flow.Stage != StageName {
		t.Fatalf("accept roll: stage=%d err=%v", flow.Stage, err)
	}
	for _, name := range []string{"", "1234567890123456"} {
		if err := flow.SetName(name); err == nil {
			t.Fatalf("accepted invalid name %q", name)
		}
	}
	if err := flow.SetName("HERO"); err != nil || flow.Stage != StagePortrait || flow.PortraitHead != 1 || flow.PortraitBody != 1 {
		t.Fatalf("set name: %+v err=%v", flow, err)
	}
	for index := 0; index < 14; index++ {
		if err := flow.NextPortraitHead(); err != nil {
			t.Fatal(err)
		}
	}
	for index := 0; index < 12; index++ {
		if err := flow.NextPortraitBody(); err != nil {
			t.Fatal(err)
		}
	}
	if flow.PortraitHead != 1 || flow.PortraitBody != 1 {
		t.Fatalf("portrait wrap = %d/%d", flow.PortraitHead, flow.PortraitBody)
	}
	if err := flow.KeepPortrait(); err != nil || flow.Stage != StageIcon {
		t.Fatalf("keep portrait: stage=%d err=%v", flow.Stage, err)
	}
	if flow.IconHead != 0 || flow.IconWeapon != 0 || flow.IconSize != 1 {
		t.Fatalf("Dwarf icon defaults = head %d weapon %d size %d", flow.IconHead, flow.IconWeapon, flow.IconSize)
	}
	if err := flow.RequestIconConfirmation(); err != nil || flow.Stage != StageIconConfirm {
		t.Fatalf("icon confirmation: stage=%d err=%v", flow.Stage, err)
	}
	if err := flow.RejectIconConfirmation(); err != nil || flow.Stage != StageIcon {
		t.Fatalf("icon reject: stage=%d err=%v", flow.Stage, err)
	}
}

// OptionIDs 要與 Options 一一對應，否則 UI 會把譯名對到錯的選項。
func TestOptionIDsLineUpWithOptions(t *testing.T) {
	flow := NewFlow()
	for _, stage := range []Stage{StageRace, StageGender, StageClass, StageAlignment} {
		flow.Stage = stage
		labels, ids := flow.Options(), flow.OptionIDs()
		if len(labels) != len(ids) || len(ids) == 0 {
			t.Fatalf("stage %v: %d labels, %d ids", stage, len(labels), len(ids))
		}
		for _, id := range ids {
			if id == "" {
				t.Fatalf("stage %v has an empty option id", stage)
			}
		}
	}
}
