package assets_test

import (
	"image"
	"image/png"
	"os"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/assets"
	"github.com/wicanr2/golden-box-remake-engine/graphics"
)

const rolfShot = "../../docs/reference/original-dos/adventure/03-rolf-approach.png"

// TestNPCPortraitMatchesTheDOSApproachShot 拿原版畫面當 oracle：疊出來的半身像
// 要與 `03-rolf-approach.png` 那一框逐格相同。
//
// 這條測試同時釘住三件事——head 是 `HEAD3` 區塊 8、body 是 `BODY3` 區塊 9、
// 疊的位置是 head 在上 body 在下。任何一項換掉都會有像素對不上。
func TestNPCPortraitMatchesTheDOSApproachShot(t *testing.T) {
	portrait, err := assets.ReadNPCPortrait(dosZIP, 3, 8, 9)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	if portrait.Width() != 88 || portrait.Height() != 88 {
		t.Fatalf("半身像是 %dx%d，預期 88x88", portrait.Width(), portrait.Height())
	}
	want, err := cropOriginalShot(rolfShot, 24, 24, 88, 88)
	if err != nil {
		t.Skipf("原版截圖不可用: %v", err)
	}
	for index := range want {
		if portrait.Pixels[index] != want[index] {
			t.Fatalf("第 %d 格是 %d，原版是 %d（x=%d y=%d）",
				index, portrait.Pixels[index], want[index], index%88, index/88)
		}
	}
}

// cropOriginalShot 從 640×400 的原版截圖裁一塊，換算回 320×200 的 EGA 索引。
func cropOriginalShot(path string, x, y, width, height int) ([]uint8, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	shot, err := png.Decode(file)
	if err != nil {
		return nil, err
	}
	scale := shot.Bounds().Dx() / 320
	pixels := make([]uint8, width*height)
	for row := 0; row < height; row++ {
		for column := 0; column < width; column++ {
			point := image.Pt(shot.Bounds().Min.X+(x+column)*scale, shot.Bounds().Min.Y+(y+row)*scale)
			red, green, blue, _ := shot.At(point.X, point.Y).RGBA()
			index := -1
			for candidate, colour := range graphics.EGA16 {
				if uint32(colour.R) == red>>8 && uint32(colour.G) == green>>8 && uint32(colour.B) == blue>>8 {
					index = candidate
					break
				}
			}
			if index < 0 {
				return nil, errNotEGA
			}
			pixels[row*width+column] = uint8(index)
		}
	}
	return pixels, nil
}

var errNotEGA = errNotEGAType{}

type errNotEGAType struct{}

func (errNotEGAType) Error() string { return "截圖裡有不在 EGA 16 色裡的顏色" }
