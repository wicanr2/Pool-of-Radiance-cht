package assets

import (
	"archive/zip"
	"fmt"

	"github.com/wicanr2/golden-box-remake-engine/graphics"
)

// 怪物在戰場上的造形（spec 166）。
//
// **怪物不用 `CBODY.DAX`。** ECL `0Bh LOAD MONSTER` 的處理常式（overlay-03 `044Dh`）在
// `0507h..0529h` 組出 Pascal 字串 "CPIC"（`0448h`），連同第三個運算元（`[bp-3]`）與槽位
// `DS:6D49h` 交給 overlay-33 entry 5（`147h:0039h`，程式 `01CAh`），再把槽位寫進記錄 `+0BFh`
// （`0571h..0577h`）。entry 5 對不是 CHEAD／CBODY／COMSPR／ICON 的名字走 `040Bh`：檔名接上
// `Str(DS:52D4h)`（目前的檔案組，與 MONnCHA 同一個數字），站立圖是那個區塊、動作圖是區塊
// 加 80h，兩張都經 `0CE6h → 0CF6h` 的換色表（`046Dh`、`04D6h`）。
//
// 圖的大小由區塊自己決定：24×24、48×24（兩格寬）、24×48（兩格高）、48×48（2×2），
// 畫的時候從戰鬥員那一格的左上角往右下鋪。

// MonsterIconArchive 是第 archive 組的怪物造形檔名（"CPIC" + 數字）。
func MonsterIconArchive(archive uint8) string {
	return fmt.Sprintf("CPIC%d.DAX", archive)
}

// MonsterIconActionOffset 是動作圖相對於站立圖的區塊位移（entry 5 的 `04ABh add ax,80h`）。
const MonsterIconActionOffset = 0x80

// monsterIconRecolour 是 `START.EXE` 資料段 `DS:0CE6h`（原色）對 `DS:0CF6h`（換成）的那一對表：
// 只有 0Dh 換成 08h，其餘原樣（兩張表都只有 overlay-33 讀，沒有寫入點）。
var monsterIconRecolour = [16]uint8{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0x0A, 0x0B, 0x0C, 0x08, 0x0E, 0x0F}

// ReadMonsterCombatIcon 讀第 archive 組的第 block 號怪物造形，已套上換色表。
func ReadMonsterCombatIcon(zipPath string, archive, block uint8, action bool) (graphics.Picture, error) {
	if archive == 0 || archive > 8 {
		return graphics.Picture{}, fmt.Errorf("Pool monster icon archive %d is outside 1..8", archive)
	}
	id := block
	if action {
		if int(block)+MonsterIconActionOffset > 0xFF {
			return graphics.Picture{}, fmt.Errorf("Pool monster icon %d has no action variant", block)
		}
		id = block + MonsterIconActionOffset
	}
	archiveFile, err := zip.OpenReader(zipPath)
	if err != nil {
		return graphics.Picture{}, fmt.Errorf("open DOS ZIP: %w", err)
	}
	defer archiveFile.Close()
	picture, err := readIconBlock(archiveFile.File, MonsterIconArchive(archive), id)
	if err != nil {
		return graphics.Picture{}, err
	}
	if picture.ItemCount != 1 || (picture.Width() != 24 && picture.Width() != 48) ||
		(picture.Height() != 24 && picture.Height() != 48) {
		return graphics.Picture{}, fmt.Errorf("%s block 0x%02X shape is %dx%dx%d",
			MonsterIconArchive(archive), id, picture.Width(), picture.Height(), picture.ItemCount)
	}
	return recolourMonsterIcon(picture), nil
}

// recolourMonsterIcon 套 `0CE6h → 0CF6h`。透明（16）不動。
func recolourMonsterIcon(picture graphics.Picture) graphics.Picture {
	result := picture
	result.Pixels = append([]uint8(nil), picture.Pixels...)
	for index, pixel := range result.Pixels {
		if pixel < 16 {
			result.Pixels[index] = monsterIconRecolour[pixel]
		}
	}
	return result
}
