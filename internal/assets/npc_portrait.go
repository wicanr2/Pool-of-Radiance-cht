package assets

import (
	"archive/zip"
	"fmt"

	"github.com/wicanr2/golden-box-remake-engine/graphics"
)

// NPCPortraitHeadHeight 是 HEAD 記錄的高度；BODY 就從這一列開始畫。
const NPCPortraitHeadHeight = 40

// ReadNPCPortrait 疊出 APPROACH 時畫在第一人稱框裡的 NPC 半身像（spec 117）。
//
// 原版的路徑是 overlay-29 entry 9：先用 `HEAD` ＋ 現行區號組檔名載入 head 區塊，
// 再用 `BODY` ＋ 同一個區號載入 body 區塊；entry 8 接著先畫 head、再畫 body，
// 位置固定 `(3,3)`——那是 8 像素為單位的座標，換算就是 `(24,24)`，剛好是
// 第一人稱內框的原點（spec 047）。所以疊法與建角肖像同一套。
func ReadNPCPortrait(zipPath string, archive, headBlock, bodyBlock uint8) (graphics.Picture, error) {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return graphics.Picture{}, fmt.Errorf("open DOS ZIP: %w", err)
	}
	defer reader.Close()
	head, err := readPortraitBlock(reader.File, fmt.Sprintf("HEAD%d.DAX", archive), headBlock, 88, NPCPortraitHeadHeight)
	if err != nil {
		return graphics.Picture{}, err
	}
	body, err := readPortraitBlock(reader.File, fmt.Sprintf("BODY%d.DAX", archive), bodyBlock, 88, 48)
	if err != nil {
		return graphics.Picture{}, err
	}
	return ComposeCreationPortrait(PortraitParts{Head: head, Body: body})
}
