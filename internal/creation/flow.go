package creation

import "fmt"

type Stage uint8

const (
	StageRace Stage = iota
	StageGender
	StageClass
	StageAlignment
	StageRoll
)

type Flow struct {
	Stage          Stage
	RaceIndex      int
	GenderIndex    int
	ClassIndex     int
	AlignmentIndex int
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
	default:
		return false
	}
	return true
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
