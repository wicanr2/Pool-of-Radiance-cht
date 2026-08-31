package character

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestParseDOSIconMapping(t *testing.T) {
	record := make([]byte, DOSRecordSize)
	record[0] = 4
	copy(record[1:], "TEST")
	copy(record[0x10:], []byte{14, 13, 11, 13, 15, 13})
	record[offsetRace], record[offsetClass], record[offsetGender] = 1, 2, 1
	record[offsetPortraitHead], record[offsetPortraitBody] = 14, 12
	copy(record[offsetIconHead:], []byte{0, 0, 0x7A, 1, 0x91, 0xA2, 0xB3, 0xC4, 0xE6, 0xF7})
	got, err := ParseDOS(record)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "TEST" || got.Abilities != [6]uint8{14, 13, 11, 13, 15, 13} {
		t.Fatalf("unexpected identity: %+v", got)
	}
	if got.RaceCode != 1 || got.ClassCode != 2 || got.GenderCode != 1 {
		t.Fatalf("unexpected identity codes: %+v", got)
	}
	if got.Portrait != (PortraitSelection{Head: 14, Body: 12}) {
		t.Fatalf("unexpected portrait: %+v", got.Portrait)
	}
	if got.Icon.OpaqueBF != 0x7A || got.Icon.Size != 1 {
		t.Fatalf("unexpected structure: %+v", got.Icon)
	}
	if got.Icon.Body != (DualColor{1, 9}) || got.Icon.Arm != (DualColor{2, 10}) || got.Icon.Leg != (DualColor{3, 11}) || got.Icon.HairFace != (DualColor{4, 12}) || got.Icon.Shield != (DualColor{6, 14}) || got.Icon.WeaponColor != (DualColor{7, 15}) {
		t.Fatalf("unexpected colors: %+v", got.Icon)
	}
}

func TestRealDOSClassPartyStrengthAnchors(t *testing.T) {
	tests := []struct {
		file, sha256 string
		want         PartyStrengthRecord
		contribution uint8
	}{
		{"FEM.CHA", "69292e21b32666e99c4b1e8a48a883915ddea683b8fc98cac4ed056f93a379a6", PartyStrengthRecord{Field96: 0, Field9B: 0, Field110: 40, Field111: 50, Field11B: 7}, 1},
		{"HMU.CHA", "8ba4b517f7d28dd07edc6d8d4c843a92ab8ef2de3b6f937615fc18d584eddb86", PartyStrengthRecord{Field96: 0, Field9B: 1, Field110: 41, Field111: 50, Field11B: 2}, 2},
		{"HTH.CHA", "c81904d74ef8a885f2eee401ca3ce35d62ca9976847877ae52eb34fee35076cd", PartyStrengthRecord{Field96: 0, Field9B: 0, Field110: 40, Field111: 52, Field11B: 4}, 0},
	}
	for _, test := range tests {
		t.Run(test.file, func(t *testing.T) {
			path := filepath.Join("..", "..", "workplace", "oracle", "class-records", test.file)
			record, err := os.ReadFile(path)
			if os.IsNotExist(err) {
				t.Skip("original DOS class oracle is not present")
			}
			if err != nil {
				t.Fatal(err)
			}
			if got := fmt.Sprintf("%x", sha256.Sum256(record)); got != test.sha256 {
				t.Fatalf("SHA-256=%s, want %s", got, test.sha256)
			}
			parsed, err := ParseDOS(record)
			if err != nil {
				t.Fatal(err)
			}
			if parsed.PartyStrength != test.want || parsed.PartyStrength.Contribution() != test.contribution {
				t.Fatalf("fields=%+v contribution=%d, want %+v/%d", parsed.PartyStrength, parsed.PartyStrength.Contribution(), test.want, test.contribution)
			}
		})
	}
}

func TestPartyStrengthFormulaUsesThresholdsIntegerDivisionAndByteWrap(t *testing.T) {
	first := PartyStrengthRecord{Field96: 2, Field9B: 3, Field110: 42, Field111: 63, Field11B: 19}
	// (8 + 24 + 15 + 15 + 19) / 10 = 8.
	if got := first.Contribution(); got != 8 {
		t.Fatalf("contribution=%d, want 8", got)
	}
	if got := (PartyStrengthRecord{Field110: 39, Field111: 60, Field11B: 9}).Contribution(); got != 0 {
		t.Fatalf("threshold contribution=%d, want 0", got)
	}
	records := make([]PartyStrengthRecord, 32)
	for i := range records {
		records[i] = first
	}
	if got := PartyStrength(records); got != 0 {
		t.Fatalf("wrapped total=%d, want 0", got)
	}
}

func TestWriteDOSPortraitPreservesEveryOtherByte(t *testing.T) {
	record := bytes.Repeat([]byte{0x5A}, DOSRecordSize)
	record[0] = 0
	got, err := WriteDOSPortrait(record, PortraitSelection{Head: 14, Body: 12})
	if err != nil {
		t.Fatal(err)
	}
	for index := range record {
		want := record[index]
		if index == offsetPortraitHead {
			want = 14
		} else if index == offsetPortraitBody {
			want = 12
		}
		if got[index] != want {
			t.Fatalf("byte %X = %02X, want %02X", index, got[index], want)
		}
	}
}

func TestWriteDOSPortraitRejectsOutOfRangeSelectors(t *testing.T) {
	record := make([]byte, DOSRecordSize)
	for _, portrait := range []PortraitSelection{{Head: 0, Body: 1}, {Head: 15, Body: 1}, {Head: 1, Body: 0}, {Head: 1, Body: 13}} {
		if _, err := WriteDOSPortrait(record, portrait); err == nil {
			t.Fatalf("accepted out-of-range portrait %+v", portrait)
		}
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
	wantStrength := PartyStrengthRecord{Field96: 0, Field9B: 0, Field110: 40, Field111: 50, Field11B: 6}
	if got.PartyStrength != wantStrength || got.PartyStrength.Contribution() != 1 {
		t.Fatalf("real character strength=%+v contribution=%d", got.PartyStrength, got.PartyStrength.Contribution())
	}
}
