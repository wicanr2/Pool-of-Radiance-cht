// Package character owns Pool-specific DOS character record adapters.
package character

import "fmt"

const (
	DOSRecordSize      = 285
	offsetPortraitHead = 0xBB
	offsetPortraitBody = 0xBC
	offsetIconHead     = 0xBD
	offsetIconWeapon   = 0xBE
	offsetIconOpaque   = 0xBF
	offsetIconSize     = 0xC0
	offsetColorBody    = 0xC1
	offsetColorArm     = 0xC2
	offsetColorLeg     = 0xC3
	offsetColorFace    = 0xC4
	offsetColorShield  = 0xC5
	offsetColorWeapon  = 0xC6
	offsetRace         = 0x2E
	offsetClass        = 0x2F
	offsetGender       = 0x9E
)

type DualColor struct{ Color1, Color2 uint8 }

type PortraitSelection struct{ Head, Body uint8 }

type IconCustomization struct {
	Head, Weapon, OpaqueBF, Size                  uint8
	Body, Arm, Leg, HairFace, Shield, WeaponColor DualColor
}

type DOSCharacter struct {
	Name       string
	Abilities  [6]uint8
	RaceCode   uint8
	ClassCode  uint8
	GenderCode uint8
	Portrait   PortraitSelection
	Icon       IconCustomization
}

func ParseDOS(record []byte) (DOSCharacter, error) {
	if len(record) != DOSRecordSize {
		return DOSCharacter{}, fmt.Errorf("Pool DOS CHA length %d, want %d", len(record), DOSRecordSize)
	}
	nameLength := int(record[0])
	if nameLength > 15 || 1+nameLength > len(record) {
		return DOSCharacter{}, fmt.Errorf("Pool DOS CHA name length %d is invalid", nameLength)
	}
	result := DOSCharacter{Name: string(record[1 : 1+nameLength])}
	copy(result.Abilities[:], record[0x10:0x16])
	result.RaceCode = record[offsetRace]
	result.ClassCode = record[offsetClass]
	result.GenderCode = record[offsetGender]
	result.Portrait = PortraitSelection{Head: record[offsetPortraitHead], Body: record[offsetPortraitBody]}
	result.Icon = IconCustomization{
		Head: record[offsetIconHead], Weapon: record[offsetIconWeapon],
		OpaqueBF: record[offsetIconOpaque], Size: record[offsetIconSize],
		Body: unpackColor(record[offsetColorBody]), Arm: unpackColor(record[offsetColorArm]),
		Leg: unpackColor(record[offsetColorLeg]), HairFace: unpackColor(record[offsetColorFace]),
		Shield: unpackColor(record[offsetColorShield]), WeaponColor: unpackColor(record[offsetColorWeapon]),
	}
	return result, nil
}

// WriteDOSPortrait changes only the two evidence-backed portrait selectors.
func WriteDOSPortrait(record []byte, portrait PortraitSelection) ([]byte, error) {
	if len(record) != DOSRecordSize {
		return nil, fmt.Errorf("Pool DOS CHA length %d, want %d", len(record), DOSRecordSize)
	}
	if portrait.Head < 1 || portrait.Head > 14 {
		return nil, fmt.Errorf("Pool portrait HEAD selector %d, want 1..14", portrait.Head)
	}
	if portrait.Body < 1 || portrait.Body > 12 {
		return nil, fmt.Errorf("Pool portrait BODY selector %d, want 1..12", portrait.Body)
	}
	result := append([]byte(nil), record...)
	result[offsetPortraitHead], result[offsetPortraitBody] = portrait.Head, portrait.Body
	return result, nil
}

// WriteDOSIcon changes only evidence-backed fields and preserves opaque BFh.
func WriteDOSIcon(record []byte, icon IconCustomization) ([]byte, error) {
	if len(record) != DOSRecordSize {
		return nil, fmt.Errorf("Pool DOS CHA length %d, want %d", len(record), DOSRecordSize)
	}
	if icon.Size != 1 && icon.Size != 2 {
		return nil, fmt.Errorf("Pool combat icon size %d, want 1 or 2", icon.Size)
	}
	colors := []DualColor{icon.Body, icon.Arm, icon.Leg, icon.HairFace, icon.Shield, icon.WeaponColor}
	for index, color := range colors {
		if color.Color1 > 0x0F || color.Color2 > 0x0F {
			return nil, fmt.Errorf("Pool combat icon color %d is outside 4-bit range", index)
		}
	}
	result := append([]byte(nil), record...)
	result[offsetIconHead], result[offsetIconWeapon], result[offsetIconSize] = icon.Head, icon.Weapon, icon.Size
	for index, offset := range []int{offsetColorBody, offsetColorArm, offsetColorLeg, offsetColorFace, offsetColorShield, offsetColorWeapon} {
		result[offset] = packColor(colors[index])
	}
	return result, nil
}

func unpackColor(value byte) DualColor { return DualColor{Color1: value & 0x0F, Color2: value >> 4} }
func packColor(value DualColor) byte   { return value.Color1 | value.Color2<<4 }
