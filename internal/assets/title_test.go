package assets

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRealTitlePicturesMatchDOSAnchor(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	if _, err := os.Stat(zipPath); err != nil {
		t.Skip("original DOS ZIP is intentionally not tracked")
	}
	pictures, err := ReadTitlePictures(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	wantMetadata := map[uint8][8]byte{
		1: {0x00, 0x11, 0x22, 0x13, 0x02, 0x21, 0x23, 0x33},
		2: {0x00, 0x11, 0x22, 0x13, 0x22, 0x11, 0x23, 0x33},
	}
	for id, want := range wantMetadata {
		picture, ok := pictures[id]
		if !ok {
			t.Fatalf("missing TITLE block 0x%02X", id)
		}
		if picture.X != 1 || picture.Y != 0 || picture.Metadata != want || len(picture.Pixels) != 64000 {
			t.Fatalf("TITLE block 0x%02X = x=%d y=%d metadata=% X pixels=%d", id, picture.X, picture.Y, picture.Metadata, len(picture.Pixels))
		}
	}
}
