// Package character owns Pool-specific DOS character record adapters.
package character

import "fmt"

const (
	DOSRecordSize     = 285
	offsetIconHead    = 0xBD
	offsetIconWeapon  = 0xBE
	offsetIconOpaque  = 0xBF
	offsetIconSize    = 0xC0
	offsetColorBody   = 0xC1
	offsetColorArm    = 0xC2
	offsetColorLeg    = 0xC3
	offsetColorFace   = 0xC4
	offsetColorShield = 0xC5
	offsetColorWeapon = 0xC6
)

type DualColor struct{ Color1, Color2 uint8 }

type IconCustomization struct {
	Head, Weapon, OpaqueBF, Size                  uint8
	Body, Arm, Leg, HairFace, Shield, WeaponColor DualColor
}

type DOSCharacter struct {
	Name      string
	Abilities [6]uint8
	Icon      IconCustomization
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
	result.Icon = IconCustomization{
		Head: record[offsetIconHead], Weapon: record[offsetIconWeapon],
		OpaqueBF: record[offsetIconOpaque], Size: record[offsetIconSize],
		Body: unpackColor(record[offsetColorBody]), Arm: unpackColor(record[offsetColorArm]),
		Leg: unpackColor(record[offsetColorLeg]), HairFace: unpackColor(record[offsetColorFace]),
		Shield: unpackColor(record[offsetColorShield]), WeaponColor: unpackColor(record[offsetColorWeapon]),
	}
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
