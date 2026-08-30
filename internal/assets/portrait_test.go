package assets

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreationPortraitSelectorsRejectOutOfRange(t *testing.T) {
	for _, selectors := range [][2]uint8{{0, 1}, {15, 1}, {1, 0}, {1, 13}} {
		if _, err := ReadCreationPortraitParts("unused.zip", selectors[0], selectors[1]); err == nil {
			t.Fatalf("accepted selectors %v", selectors)
		}
	}
}

func TestRealCreationPortraitDescriptorAnchors(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	if _, err := os.Stat(zipPath); err != nil {
		t.Skip("original DOS ZIP is intentionally not tracked")
	}
	for _, selectors := range [][2]uint8{{1, 1}, {14, 12}} {
		parts, err := ReadCreationPortraitParts(zipPath, selectors[0], selectors[1])
		if err != nil {
			t.Fatal(err)
		}
		if parts.Head.Width() != 88 || parts.Head.Height() != 40 || len(parts.Head.Pixels) != 3520 {
			t.Fatalf("HEAD %d has unexpected shape", selectors[0])
		}
		if parts.Body.Width() != 88 || parts.Body.Height() != 48 || len(parts.Body.Pixels) != 4224 {
			t.Fatalf("BODY %d has unexpected shape", selectors[1])
		}
		composed, err := ComposeCreationPortrait(parts)
		if err != nil {
			t.Fatal(err)
		}
		if composed.Width() != 88 || composed.Height() != 88 || len(composed.Pixels) != 7744 {
			t.Fatalf("composed portrait has unexpected shape")
		}
		for index, value := range parts.Head.Pixels {
			if composed.Pixels[index] != value {
				t.Fatalf("HEAD pixel %d changed during composition", index)
			}
		}
		for index, value := range parts.Body.Pixels {
			if composed.Pixels[88*40+index] != value {
				t.Fatalf("BODY pixel %d changed during composition", index)
			}
		}
	}
}
