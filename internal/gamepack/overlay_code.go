package gamepack

import (
	"fmt"

	"github.com/wicanr2/golden-box-remake-engine/tpov"
)

// OverlayCode 取一顆 overlay 的位元組。反組譯的輸入一律由這裡出，
// 才不會有人手工從 ZIP 裡切一段出來、位移差幾個 byte 也看不出來。
func OverlayCode(zipPath string, index int) ([]byte, error) {
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
	if index < 0 || index >= len(overlays) {
		return nil, fmt.Errorf("GAME.OVR has %d overlays, want index %d", len(overlays), index)
	}
	return overlays[index].Code, nil
}
