package assets

import (
	"encoding/binary"
	"fmt"

	"github.com/wicanr2/golden-box-remake-engine/dax"
	"github.com/wicanr2/golden-box-remake-engine/graphics"
)

// ECL `PICTURE n` 在 head 選擇子 `6DE1h` 是 `FFh` 時畫的那一條（spec 165）。
//
// 原版 overlay-03 `0890h..08C3h`：把 `"PIC"`（`081Eh`）交給 overlay-29 entry 5
// （code `00A8h`）載入 `PIC<區號>.DAX` 的區塊 n 到 `DS:6A1Ch` 那張動畫表，再用
// entry 2 把第 1 張畫在 `(3,3)`——8 像素為單位，就是第一人稱內框的原點。
//
// 動畫表的版面（spec 117〈PIC 容器的版面〉、spec 135〈營火的動畫時序〉）：
//
//	+0        張數
//	每一張：  4 bytes 延遲（32-bit，little-endian）＋ 17 bytes 圖片頭 ＋ 像素
//
// 延遲在選單元件的等鍵迴圈裡用：`經過的 BIOS tick > 延遲 ÷ 7` 就換下一張
// （overlay-26 `0277h..02B8h`）。

// PICAnimationFrameLimit 是區塊 1 的張數上限：overlay-29 `01B5h..01CEh` 在檔名是
// `"PIC"`（`0099h`）而且區塊是 1 時，把超過 4 的張數截成 4。
const PICAnimationFrameLimit = 4

// PICAnimation 是 `PIC<區號>.DAX` 的一個區塊：每一張都是還原過的完整圖。
type PICAnimation struct {
	Frames []graphics.Picture
	// Delays 是每一張前面那 4 bytes，單位給選單元件的等鍵迴圈用（÷ 7 個 BIOS tick）。
	Delays []uint32
}

// ReadPICAnimation 讀 `PIC<區號>.DAX` 的區塊 block。
func ReadPICAnimation(zipPath string, archive, block uint8) (PICAnimation, error) {
	name := fmt.Sprintf("PIC%d.DAX", archive)
	data, err := readArchiveMember(zipPath, name)
	if err != nil {
		return PICAnimation{}, err
	}
	blocks, err := dax.Parse(data)
	if err != nil {
		return PICAnimation{}, fmt.Errorf("parse %s: %w", name, err)
	}
	for _, entry := range blocks {
		if entry.Entry.ID != block {
			continue
		}
		frames, delays, err := parseAnimationWithDelays(entry.Data)
		if err != nil {
			return PICAnimation{}, fmt.Errorf("%s block %d: %w", name, block, err)
		}
		if block == 1 && len(frames) > PICAnimationFrameLimit {
			frames, delays = frames[:PICAnimationFrameLimit], delays[:PICAnimationFrameLimit]
		}
		return PICAnimation{Frames: frames, Delays: delays}, nil
	}
	return PICAnimation{}, fmt.Errorf("%s has no block %d", name, block)
}

// parseAnimationWithDelays 把一個 PIC 區塊拆成每一張與它的延遲，差分張還原成完整圖。
func parseAnimationWithDelays(data []byte) ([]graphics.Picture, []uint32, error) {
	if len(data) < 1 {
		return nil, nil, fmt.Errorf("animation block is empty")
	}
	count := int(data[0])
	if count == 0 {
		return nil, nil, fmt.Errorf("animation block declares no frames")
	}
	result := make([]graphics.Picture, 0, count)
	delays := make([]uint32, 0, count)
	var base graphics.Picture
	pos := 1
	for index := 0; index < count; index++ {
		if pos+4+17 > len(data) {
			return nil, nil, fmt.Errorf("frame %d runs past the block", index)
		}
		delay := binary.LittleEndian.Uint32(data[pos:])
		header := data[pos+4:]
		width := int(uint16(header[2]) | uint16(header[3])<<8)
		height := int(uint16(header[0]) | uint16(header[1])<<8)
		items := int(header[8])
		if width == 0 || height == 0 || items == 0 {
			return nil, nil, fmt.Errorf("frame %d has invalid dimensions", index)
		}
		size := 17 + items*(width*8*height)/2
		if pos+4+size > len(data) {
			return nil, nil, fmt.Errorf("frame %d wants %d bytes, block has %d left",
				index, size, len(data)-pos-4)
		}
		picture, err := graphics.ParsePicture(data[pos+4:pos+4+size], false, 0)
		if err != nil {
			return nil, nil, fmt.Errorf("frame %d: %w", index, err)
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
		delays = append(delays, delay)
		pos += 4 + size
	}
	return result, delays, nil
}
