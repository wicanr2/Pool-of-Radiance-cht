package character

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestParseDOSIconMapping(t *testing.T) {
	record := make([]byte, DOSRecordSize)
	record[0] = 4
	copy(record[1:], "TEST")
	copy(record[0x10:], []byte{14, 13, 11, 13, 15, 13})
	copy(record[offsetIconHead:], []byte{0, 0, 0x7A, 1, 0x91, 0xA2, 0xB3, 0xC4, 0xE6, 0xF7})
	got, err := ParseDOS(record)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "TEST" || got.Abilities != [6]uint8{14, 13, 11, 13, 15, 13} {
		t.Fatalf("unexpected identity: %+v", got)
	}
	if got.Icon.OpaqueBF != 0x7A || got.Icon.Size != 1 {
		t.Fatalf("unexpected structure: %+v", got.Icon)
	}
	if got.Icon.Body != (DualColor{1, 9}) || got.Icon.Arm != (DualColor{2, 10}) || got.Icon.Leg != (DualColor{3, 11}) || got.Icon.HairFace != (DualColor{4, 12}) || got.Icon.Shield != (DualColor{6, 14}) || got.Icon.WeaponColor != (DualColor{7, 15}) {
		t.Fatalf("unexpected colors: %+v", got.Icon)
	}
}

func TestWriteDOSIconPreservesUnknownBytes(t *testing.T) {
	record := bytes.Repeat([]byte{0x5A}, DOSRecordSize)
	record[0] = 0
	record[offsetIconSize] = 1
	record[offsetIconOpaque] = 0xA5
	icon := IconCustomization{Head: 1, Weapon: 2, Size: 2, Body: DualColor{2, 10}, Arm: DualColor{3, 11}, Leg: DualColor{4, 12}, HairFace: DualColor{5, 13}, Shield: DualColor{7, 15}, WeaponColor: DualColor{8, 0}}
	got, err := WriteDOSIcon(record, icon)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{1, 2, 0xA5, 2, 0xA2, 0xB3, 0xC4, 0xD5, 0xF7, 0x08}
	if !bytes.Equal(got[offsetIconHead:offsetColorWeapon+1], want) {
		t.Fatalf("icon bytes = % X, want % X", got[offsetIconHead:offsetColorWeapon+1], want)
	}
	for index := range record {
		if index >= offsetIconHead && index <= offsetColorWeapon {
			continue
		}
		if got[index] != record[index] {
			t.Fatalf("unrelated byte %X changed", index)
		}
	}
}

func TestParseDOSRejectsInvalidRecords(t *testing.T) {
	if _, err := ParseDOS(make([]byte, DOSRecordSize-1)); err == nil {
		t.Fatal("short record accepted")
	}
	record := make([]byte, DOSRecordSize)
	record[0] = 16
	if _, err := ParseDOS(record); err == nil {
		t.Fatal("overlong name accepted")
	}
}

func TestRealDOSCharacterAnchor(t *testing.T) {
	path := filepath.Join("..", "..", "workplace", "oracle", "character-diff", "BASE.CHA")
	record, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		t.Skip("original DOS character oracle is not present")
	}
	if err != nil {
		t.Fatal(err)
	}
	got, err := ParseDOS(record)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "BASE" || got.Icon.Size != 1 || got.Icon.Body != (DualColor{1, 9}) || got.Icon.WeaponColor != (DualColor{7, 15}) {
		t.Fatalf("unexpected real character: %+v", got)
	}
}
