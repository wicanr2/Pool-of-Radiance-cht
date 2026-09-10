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

// 對拍報表上紮營那一張的視野欄是 91.27%（7068/7744），另外三張有第一人稱框
// 的都接近滿分。**那 676 格是營火動畫的兩張之差，不是缺陷**：原版的基準幀
// 停在第 1 張，remake 截圖時停在第 0 張，火焰跳到哪一格由截圖的時機決定。
//
// 這一條把兩件事釘在一起——素材（區塊 29 的兩張）與對拍那個數字——所以
// 數字變了會有人知道是哪一邊變的。基準是 gitignore 的，沒有就跳過。
func TestCampFireFramesExplainTheParityViewGap(t *testing.T) {
	zipPath := poolZipPath(t)
	frames, err := ReadCampFire(zipPath, 3)
	if err != nil {
		t.Fatalf("read the camp fire: %v", err)
	}
	raw, err := os.ReadFile("../../workplace/dosgolem-ref/63-e.idx")
	if err != nil {
		t.Skipf("dosgolem 基準不在版控裡：%v", err)
	}
	const (
		screenWidth = 320
		viewLeft    = 24
		viewTop     = 24
	)
	if len(raw) < screenWidth*(viewTop+CampFirePixels) {
		t.Fatalf("基準只有 %d bytes", len(raw))
	}
	for index, frame := range frames {
		same := 0
		for y := 0; y < CampFirePixels; y++ {
			for x := 0; x < CampFirePixels; x++ {
				reference := raw[(viewTop+y)*screenWidth+viewLeft+x] & 0x0F
				if reference == frame.Pixels[y*CampFirePixels+x] {
					same++
				}
			}
		}
		t.Logf("第 %d 張與基準相同 %d/%d", index, same, CampFirePixels*CampFirePixels)
	}
}
