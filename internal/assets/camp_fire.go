package assets

import (
	"fmt"

	"github.com/wicanr2/golden-box-remake-engine/dax"
	"github.com/wicanr2/golden-box-remake-engine/graphics"
)

// 紮營畫面的營火（spec 135）。
//
// 原版按 `E` 之後把第一人稱視野那一塊 88×88 換成一張營火——那是
// `PIC<區號>.DAX` 的**區塊 29**，而 `PIC` 容器的區塊是一段動畫：
//
//	+0        張數
//	每一張：  4 bytes 前綴 ＋ 17 bytes 標準圖片頭 ＋ 像素
//
// 第二張以後是與第一張 XOR 的差分（overlay-29 `0338h..03B7h` 那段 XOR 迴圈
// 只在檔名是 `PIC` 或 `FINAL` 時才跑，見 spec 117）。營火是**兩張**，差的
// 就是火焰跳動的那幾格。
//
// 怎麼確定是區塊 29：拿原版紮營那一幀的 `(24,24)` 88×88 與八個 `PIC*.DAX`
// 的每一張逐格比對，區塊 29 的**第 0 張 100% 相同、第 1 張 7068/7744**，
// 其餘都不到 60%（spec 135）。八個檔的區塊 29 內容相同——營火不分區，但原版
// 仍是照現行區號組檔名，這裡照做。
//
// 那 676 格之差就是火焰跳動的那幾格，也是對拍報表上紮營視野欄在 100% 與
// 91.27% 之間跳的原因：兩邊的截圖各自停在動畫的哪一張，是時機決定的。
// `camp_fire_test.go` 的 `TestCampFireFramesExplainTheParityViewGap` 釘住這
// 兩個數字。
const (
	// CampFireBlock 是營火在 `PIC<區號>.DAX` 裡的區塊編號。
	CampFireBlock = 29
	// CampFirePixels 是那張圖的邊長，與第一人稱視野的內框同寬（spec 047）。
	CampFirePixels = 88
	// CampFireFrames 是營火動畫的張數。
	CampFireFrames = 2
)

// ReadCampFire 讀某一區的營火動畫。回傳的每一張都是還原過的完整圖，
// 不是差分。
func ReadCampFire(zipPath string, archive uint8) ([]graphics.Picture, error) {
	name := fmt.Sprintf("PIC%d.DAX", archive)
	data, err := readArchiveMember(zipPath, name)
	if err != nil {
		return nil, err
	}
	blocks, err := dax.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", name, err)
	}
	for _, block := range blocks {
		if block.Entry.ID != CampFireBlock {
			continue
		}
		pictures, err := parseAnimation(block.Data)
		if err != nil {
			return nil, fmt.Errorf("%s block %d: %w", name, CampFireBlock, err)
		}
		if len(pictures) != CampFireFrames {
			return nil, fmt.Errorf("%s block %d has %d frames, want %d",
				name, CampFireBlock, len(pictures), CampFireFrames)
		}
		for index, picture := range pictures {
			if picture.Width() != CampFirePixels || picture.Height() != CampFirePixels {
				return nil, fmt.Errorf("%s block %d frame %d is %dx%d, want %dx%d",
					name, CampFireBlock, index, picture.Width(), picture.Height(),
					CampFirePixels, CampFirePixels)
			}
		}
		return pictures, nil
	}
	return nil, fmt.Errorf("%s has no block %d", name, CampFireBlock)
}

// parseAnimation 把一個 PIC 區塊拆成每一張，並把差分張還原成完整圖。
// 每一張的延遲由 parseAnimationWithDelays 讀（pic_animation.go，spec 165）。
func parseAnimation(data []byte) ([]graphics.Picture, error) {
	pictures, _, err := parseAnimationWithDelays(data)
	return pictures, err
}
