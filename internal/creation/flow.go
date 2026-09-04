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

// OptionIDs 與 Options 一一對應，回傳每個選項的穩定 ID。UI 靠 ID 決定譯名，
// 不靠英文標籤字串——標籤是給人看的，改了不該連帶改壞翻譯對照。
func (flow Flow) OptionIDs() []string {
	var result []string
	switch flow.Stage {
	case StageRace:
		for _, value := range Races {
			result = append(result, value.ID)
		}
	case StageGender:
		for _, value := range Genders {
			result = append(result, value.ID)
		}
	case StageClass:
		for _, value := range ClassesForRace(Races[flow.RaceIndex].ID) {
			result = append(result, value.ID)
		}
	case StageAlignment:
		for _, value := range Alignments {
			result = append(result, value.ID)
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
	if flow.UsesIconSizeMenu() {
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

// 巢狀 icon menu 需要的反向與直接設定。原版的 `NEXT / PREV / KEEP / EXIT`
// 迴圈兩個方向都走得動（spec 003 第 8 步），而 COLOR-1／COLOR-2 的部位是
// 直接點名的，不是一路循環過去。

// PreviousIconHead 是 HEAD 的 PREV。
func (flow *Flow) PreviousIconHead() error {
	if flow.Stage != StageIcon {
		return fmt.Errorf("creation stage %d does not edit an icon", flow.Stage)
	}
	flow.IconHead = (flow.IconHead + 13) % 14
	return nil
}

// PreviousIconWeapon 是 WEAPON 的 PREV。
func (flow *Flow) PreviousIconWeapon() error {
	if flow.Stage != StageIcon {
		return fmt.Errorf("creation stage %d does not edit an icon", flow.Stage)
	}
	flow.IconWeapon = (flow.IconWeapon + 31) % 32
	return nil
}

// SelectIconPart 直接指定要改顏色的部位。
func (flow *Flow) SelectIconPart(index int) error {
	if flow.Stage != StageIcon {
		return fmt.Errorf("creation stage %d does not edit an icon", flow.Stage)
	}
	if index < 0 || index >= len(flow.IconColors) {
		return fmt.Errorf("icon part %d is outside 0..%d", index, len(flow.IconColors)-1)
	}
	flow.IconPart = uint8(index)
	return nil
}

// PreviousIconColor 是顏色的 PREV。四位元循環，與 NextIconColor 同一組。
func (flow *Flow) PreviousIconColor(component int) error {
	if flow.Stage != StageIcon || component < 0 || component > 1 {
		return fmt.Errorf("invalid icon color edit")
	}
	flow.IconColors[flow.IconPart][component] = (flow.IconColors[flow.IconPart][component] + 0x0F) & 0x0F
	return nil
}

// SetIconSize 直接設定大小。1 是小、2 是正常；SIZE 子選單的 LARGE 選的是 2。
func (flow *Flow) SetIconSize(size uint8) error {
	if flow.Stage != StageIcon {
		return fmt.Errorf("creation stage %d does not edit an icon", flow.Stage)
	}
	if size != 1 && size != 2 {
		return fmt.Errorf("icon size %d, want 1 or 2", size)
	}
	flow.IconSize = size
	return nil
}

// SmallRaceIDs 是建角時圖示預設為小號的種族（KeepPortrait 設 IconSize = 1
// 的那三個）。SIZE 子選單只對他們有意義——別的種族本來就是正常大小。
func SmallRaceIDs() []string { return []string{"dwarf", "gnome", "halfling"} }

// UsesIconSizeMenu 回答這個角色的 icon editor 要不要顯示 SIZE。
func (flow Flow) UsesIconSizeMenu() bool {
	id := flow.SelectedRace().ID
	for _, small := range SmallRaceIDs() {
		if id == small {
			return true
		}
	}
	return false
}
