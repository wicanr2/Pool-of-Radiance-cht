package assets

import (
	"os"
	"testing"
)

// 營火是 `PIC<區號>.DAX` 區塊 29 的兩張 88×88（spec 135）。
func TestReadCampFire(t *testing.T) {
	zipPath := poolZipPath(t)
	frames, err := ReadCampFire(zipPath, 1)
	if err != nil {
		t.Fatalf("read the camp fire: %v", err)
	}
	if len(frames) != CampFireFrames {
		t.Fatalf("got %d frames, want %d", len(frames), CampFireFrames)
	}
	for index, frame := range frames {
		if frame.Width() != CampFirePixels || frame.Height() != CampFirePixels {
			t.Errorf("frame %d is %dx%d, want %dx%d",
				index, frame.Width(), frame.Height(), CampFirePixels, CampFirePixels)
		}
	}
	// 兩張是動畫，不是同一張畫兩次——火焰在跳。
	same := 0
	for i := range frames[0].Pixels {
		if i < len(frames[1].Pixels) && frames[0].Pixels[i] == frames[1].Pixels[i] {
			same++
		}
	}
	if same == len(frames[0].Pixels) {
		t.Error("the two frames are identical; the XOR delta was not applied")
	}
}

// 八個區的區塊 29 都讀得到——原版是照現行區號組檔名的。
func TestEveryArchiveHasTheCampFire(t *testing.T) {
	zipPath := poolZipPath(t)
	for archive := uint8(1); archive <= 8; archive++ {
		if _, err := ReadCampFire(zipPath, archive); err != nil {
			t.Errorf("PIC%d.DAX: %v", archive, err)
		}
	}
}

func poolZipPath(t *testing.T) string {
	t.Helper()
	const path = "../../Pool of Radiance (1988).zip"
	if _, err := os.Stat(path); err != nil {
		t.Skip("原版磁碟映像不在版控裡，跳過")
	}
	return path
}
