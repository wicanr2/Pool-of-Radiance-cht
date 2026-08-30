package assets

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/golden-box-remake-engine/graphics"
)

func TestCombatIconBlockFamilies(t *testing.T) {
	cases := []struct {
		selection  CombatIconSelection
		action     bool
		head, body uint8
	}{
		{CombatIconSelection{Head: 13, Body: 31, Size: 2}, false, 0x0D, 0x1F},
		{CombatIconSelection{Head: 13, Body: 31, Size: 1}, false, 0x4D, 0x5F},
		{CombatIconSelection{Head: 13, Body: 31, Size: 2}, true, 0x8D, 0x9F},
		{CombatIconSelection{Head: 13, Body: 31, Size: 1}, true, 0xCD, 0xDF},
	}
	for _, test := range cases {
		head, body, err := CombatIconBlockIDs(test.selection, test.action)
		if err != nil || head != test.head || body != test.body {
			t.Fatalf("%+v action=%t => %02X/%02X, %v", test.selection, test.action, head, body, err)
		}
	}
}

func TestRealCombatIconAnchors(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	if _, err := os.Stat(zipPath); err != nil {
		t.Skip("original DOS ZIP is intentionally not tracked")
	}
	for _, size := range []uint8{1, 2} {
		for _, action := range []bool{false, true} {
			picture, err := ReadCombatIcon(zipPath, CombatIconSelection{Head: 13, Body: 31, Size: size}, action)
			if err != nil {
				t.Fatal(err)
			}
			if picture.Width() != 24 || picture.Height() != 24 || picture.ItemCount != 1 {
				t.Fatalf("size=%d action=%t shape=%dx%dx%d", size, action, picture.Width(), picture.Height(), picture.ItemCount)
			}
		}
	}
}

func TestCombatIconSixDualColorsOnlyReplaceTemplateSlots(t *testing.T) {
	picture := graphics.Picture{WidthUnits: 1, HeightUnits: 2, ItemCount: 1,
		Pixels: []uint8{1, 9, 2, 10, 3, 11, 4, 12, 6, 14, 7, 15, 0, 5, 8, 16}}
	colors := [6][2]uint8{{15, 14}, {13, 12}, {11, 10}, {9, 8}, {7, 6}, {5, 4}}
	got := recolorCombatIcon(picture, colors)
	want := []uint8{15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 0, 5, 8, 16}
	for index := range want {
		if got.Pixels[index] != want[index] {
			t.Fatalf("pixel %d=%d, want %d", index, got.Pixels[index], want[index])
		}
	}
}
