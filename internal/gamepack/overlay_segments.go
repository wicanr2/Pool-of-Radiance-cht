package gamepack

import (
	"fmt"

	"github.com/wicanr2/golden-box-remake-engine/tpov"
)

// MZHeaderBytes 是 `START.EXE` 的 MZ header 長度；控制記錄的檔案位移要減掉它
// 才換得到 runtime segment。
const MZHeaderBytes = 0x3B0

// OverlayStubSegment 算一顆 overlay 的 stub segment（spec 109）。
//
// 反組譯出來的跨 overlay 呼叫寫成 `call <segment>:<offset>`，那個 segment
// 不是程式碼位址，是這顆 overlay 的 stub 段。每個進入點都算得出同一個值：
//
//	segment = (executable_file_offset − stub_offset − 3B0h) ÷ 16
//
// 進入點之間算出來的值不一致就回錯——那代表清冊或 header 長度的假設壞了。
func OverlayStubSegment(overlay tpov.Overlay) (uint16, error) {
	if len(overlay.Entries) == 0 {
		return 0, fmt.Errorf("overlay has no entries")
	}
	var segment uint16
	for index, entry := range overlay.Entries {
		base := entry.ExecutableOffset - int(entry.StubOffset) - MZHeaderBytes
		if base < 0 || base%16 != 0 {
			return 0, fmt.Errorf("entry %d gives a %d-byte control offset", index, base)
		}
		value := uint16(base / 16)
		if index == 0 {
			segment = value
			continue
		}
		if value != segment {
			return 0, fmt.Errorf("entry %d gives segment %04X, entry 0 gives %04X",
				index, value, segment)
		}
	}
	return segment, nil
}

// ReadDOSOverlayStubSegments 回傳三十八顆 overlay 的 stub segment，索引就是
// overlay 編號。
func ReadDOSOverlayStubSegments(zipPath string) ([]uint16, error) {
	executable, err := readStartExecutable(zipPath)
	if err != nil {
		return nil, err
	}
	overlayFile, err := readArchiveMember(zipPath, "GAME.OVR")
	if err != nil {
		return nil, err
	}
	overlays, err := tpov.Decode(executable, overlayFile)
	if err != nil {
		return nil, fmt.Errorf("decode GAME.OVR: %w", err)
	}
	segments := make([]uint16, 0, len(overlays))
	for index, overlay := range overlays {
		segment, err := OverlayStubSegment(overlay)
		if err != nil {
			return nil, fmt.Errorf("overlay %d: %w", index, err)
		}
		segments = append(segments, segment)
	}
	return segments, nil
}
