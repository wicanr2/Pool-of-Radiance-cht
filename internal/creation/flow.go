package creation

import "fmt"

type Stage uint8

const (
	StageRace Stage = iota
	StageGender
	StageClass
	StageAlignment
	StageRoll
	StageName
	StagePortrait
	StageIcon
	StageIconConfirm
)

type Flow struct {
	Stage          Stage
	RaceIndex      int
	GenderIndex    int
	ClassIndex     int
	AlignmentIndex int
	Name           string
	PortraitHead   uint8
	PortraitBody   uint8
	IconHead       uint8
	IconWeapon     uint8
	IconSize       uint8
	IconPart       uint8
	IconColors     [6][2]uint8
}

func NewFlow() Flow { return Flow{Stage: StageRace} }

func (flow Flow) Options() []string {
	var result []string
	switch flow.Stage {
	case StageRace:
		for _, value := range Races {
			result = append(result, value.Label)
		}
	case StageGender:
		for _, value := range Genders {
			result = append(result, value.Label)
		}
	case StageClass:
		for _, value := range ClassesForRace(Races[flow.RaceIndex].ID) {
			result = append(result, value.Label)
		}
	case StageAlignment:
		for _, value := range Alignments {
			result = append(result, value.Label)
		}
	}
	return result
}

func (flow *Flow) Select(index int) error {
	options := flow.Options()
	if index < 0 || index >= len(options) {
		return fmt.Errorf("creation option %d outside 0..%d", index, len(options)-1)
	}
	switch flow.Stage {
	case StageRace:
		flow.RaceIndex, flow.ClassIndex, flow.Stage = index, 0, StageGender
	case StageGender:
		flow.GenderIndex, flow.Stage = index, StageClass
	case StageClass:
		flow.ClassIndex, flow.Stage = index, StageAlignment
	case StageAlignment:
		flow.AlignmentIndex, flow.Stage = index, StageRoll
	default:
		return fmt.Errorf("creation stage %d does not accept menu selection", flow.Stage)
	}
	return nil
}

// Back is the remake's explicit ESC seam. It mirrors the observed nested DOS
// order without discarding selections from earlier screens.
func (flow *Flow) Back() bool {
	switch flow.Stage {
	case StageGender:
		flow.Stage = StageRace
	case StageClass:
		flow.Stage = StageGender
	case StageAlignment:
		flow.Stage = StageClass
	case StageRoll:
		flow.Stage = StageAlignment
	case StageName:
		flow.Stage = StageRoll
	case StagePortrait:
		flow.Stage = StageName
	case StageIcon:
		flow.Stage = StagePortrait
	case StageIconConfirm:
		flow.Stage = StageIcon
	default:
		return false
	}
	return true
}

func (flow *Flow) AcceptRoll() error {
	if flow.Stage != StageRoll {
		return fmt.Errorf("creation stage %d has no roll to accept", flow.Stage)
	}
	flow.Stage = StageName
	return nil
}

func (flow *Flow) SetName(name string) error {
	if flow.Stage != StageName {
		return fmt.Errorf("creation stage %d does not accept a name", flow.Stage)
	}
	if len(name) < 1 || len(name) > 15 {
		return fmt.Errorf("Pool character name length %d, want 1..15 bytes", len(name))
	}
	flow.Name, flow.PortraitHead, flow.PortraitBody, flow.Stage = name, 1, 1, StagePortrait
	return nil
}

func (flow *Flow) NextPortraitHead() error {
	if flow.Stage != StagePortrait {
		return fmt.Errorf("creation stage %d does not edit a portrait", flow.Stage)
	}
	flow.PortraitHead = flow.PortraitHead%14 + 1
	return nil
}

func (flow *Flow) NextPortraitBody() error {
	if flow.Stage != StagePortrait {
		return fmt.Errorf("creation stage %d does not edit a portrait", flow.Stage)
	}
	flow.PortraitBody = flow.PortraitBody%12 + 1
	return nil
}

func (flow *Flow) KeepPortrait() error {
	if flow.Stage != StagePortrait {
		return fmt.Errorf("creation stage %d has no portrait to keep", flow.Stage)
	}
	flow.IconHead, flow.IconWeapon, flow.IconPart = 0, 0, 0
	flow.IconSize = 2
	if flow.SelectedRace().ID == "dwarf" || flow.SelectedRace().ID == "gnome" || flow.SelectedRace().ID == "halfling" {
		flow.IconSize = 1
	}
	flow.IconColors = [6][2]uint8{{1, 9}, {2, 10}, {3, 11}, {4, 12}, {6, 14}, {7, 15}}
	flow.Stage = StageIcon
	return nil
}

func (flow *Flow) NextIconHead() error {
	if flow.Stage != StageIcon {
		return fmt.Errorf("creation stage %d does not edit an icon", flow.Stage)
	}
	flow.IconHead = (flow.IconHead + 1) % 14
	return nil
}

func (flow *Flow) NextIconWeapon() error {
	if flow.Stage != StageIcon {
		return fmt.Errorf("creation stage %d does not edit an icon", flow.Stage)
	}
	flow.IconWeapon = (flow.IconWeapon + 1) % 32
	return nil
}

func (flow *Flow) SelectNextIconPart() error {
	if flow.Stage != StageIcon {
		return fmt.Errorf("creation stage %d does not edit an icon", flow.Stage)
	}
	flow.IconPart = (flow.IconPart + 1) % 6
	return nil
}

func (flow *Flow) NextIconColor(component int) error {
	if flow.Stage != StageIcon || component < 0 || component > 1 {
		return fmt.Errorf("invalid icon color edit")
	}
	flow.IconColors[flow.IconPart][component] = (flow.IconColors[flow.IconPart][component] + 1) & 0x0F
	return nil
}

func (flow *Flow) ToggleIconSize() error {
	if flow.Stage != StageIcon {
		return fmt.Errorf("creation stage %d does not edit an icon", flow.Stage)
	}
	if flow.IconSize == 1 {
		flow.IconSize = 2
	} else {
		flow.IconSize = 1
	}
	return nil
}

func (flow *Flow) RequestIconConfirmation() error {
	if flow.Stage != StageIcon {
		return fmt.Errorf("creation stage %d does not finish an icon", flow.Stage)
	}
	flow.Stage = StageIconConfirm
	return nil
}

func (flow *Flow) RejectIconConfirmation() error {
	if flow.Stage != StageIconConfirm {
		return fmt.Errorf("creation stage %d does not confirm an icon", flow.Stage)
	}
	flow.Stage = StageIcon
	return nil
}

func (flow Flow) SelectedRace() Race     { return Races[flow.RaceIndex] }
func (flow Flow) SelectedGender() Gender { return Genders[flow.GenderIndex] }
func (flow Flow) SelectedClass() ClassChoice {
	return ClassesForRace(flow.SelectedRace().ID)[flow.ClassIndex]
}
func (flow Flow) SelectedAlignment() Alignment { return Alignments[flow.AlignmentIndex] }

// Roll uses the selections accepted by the original menu order. Calling it
// before the roll page is fail-closed so tests and future frontends cannot
// silently generate a character from default, unconfirmed selections.
func (flow Flow) Roll(roller Roller) (RolledCharacter, error) {
	if flow.Stage != StageRoll {
		return RolledCharacter{}, fmt.Errorf("creation stage %d is not ready to roll", flow.Stage)
	}
	return RollCharacter(roller, flow.SelectedRace(), flow.SelectedGender(), flow.SelectedClass()), nil
}
