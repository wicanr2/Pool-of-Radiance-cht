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
// 的每一張逐格比對，區塊 29 的第 1 張 100% 相同、第 0 張 91.27%，其餘都不到
// 60%（spec 135）。八個檔的區塊 29 內容相同——營火不分區，但原版仍是照現行
// 區號組檔名，這裡照做。
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
func parseAnimation(data []byte) ([]graphics.Picture, error) {
	if len(data) < 1 {
		return nil, fmt.Errorf("animation block is empty")
	}
	count := int(data[0])
	if count == 0 {
		return nil, fmt.Errorf("animation block declares no frames")
	}
	result := make([]graphics.Picture, 0, count)
	var base graphics.Picture
	pos := 1
	for index := 0; index < count; index++ {
		// 每一張前面有 4 bytes 前綴，內容還沒讀出來，這裡只跳過。
		if pos+4+17 > len(data) {
			return nil, fmt.Errorf("frame %d runs past the block", index)
		}
		header := data[pos+4:]
		width := int(uint16(header[2]) | uint16(header[3])<<8)
		height := int(uint16(header[0]) | uint16(header[1])<<8)
		items := int(header[8])
		if width == 0 || height == 0 || items == 0 {
			return nil, fmt.Errorf("frame %d has invalid dimensions", index)
		}
		size := 17 + items*(width*8*height)/2
		if pos+4+size > len(data) {
			return nil, fmt.Errorf("frame %d wants %d bytes, block has %d left",
				index, size, len(data)-pos-4)
		}
		picture, err := graphics.ParsePicture(data[pos+4:pos+4+size], false, 0)
		if err != nil {
			return nil, fmt.Errorf("frame %d: %w", index, err)
		}
		if index == 0 {
			base = picture
		} else {
			restored := base
			restored.Pixels = make([]uint8, len(base.Pixels))
			for i := range base.Pixels {
				if i < len(picture.Pixels) {
					restored.Pixels[i] = base.Pixels[i] ^ picture.Pixels[i]
				}
			}
			picture = restored
		}
		result = append(result, picture)
		pos += 4 + size
	}
	return result, nil
}
