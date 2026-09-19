package assets

import (
	"archive/zip"
	"fmt"

	"github.com/wicanr2/golden-box-remake-engine/graphics"
)

// 戰鬥特效圖形（`COMSPR.DAX`）。
//
// **這一份不是怪物的戰場造形。** 實際解出來的是箭、飛斧、擲石、閃光、爆炸這類
// 特效與狀態圖示（外加一個骷髏交叉骨與一個騎馬人像）；戰場上的怪物用的是
// `CBODY.DAX` 的身體，與玩家角色共用同一套（`cmd/pool-game/sprite_overview.go`
// 的 `drawMonsterOverview` 與 `drawEffectOverview` 分得很清楚）。
// 每一張 24×24，分兩態——站立在 `id`、動作在 `id + 80h`。
//
// `COMSPR.DAX` 有十三組，編號**不是連續的**：`0..0Bh` 與 `19h`
// （動作態各自加 `80h`）。另外 `ICON.DAX` 有三組。
// 編號中間有洞是原版資料本來的樣子，不要當成漏讀。

// MonsterSpriteArchive 是戰鬥特效圖形的檔名。
const MonsterSpriteArchive = "COMSPR.DAX"

// MonsterSpriteActionOffset 是動作態相對於站立態的編號位移。
const MonsterSpriteActionOffset = 0x80

// MonsterSpriteBlocks 是 `COMSPR.DAX` 實際有的站立態編號，量自原版資料。
func MonsterSpriteBlocks() []uint8 {
	return []uint8{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 0x19}
}

// ReadMonsterSprite 讀一隻怪物在戰場上的圖形。
func ReadMonsterSprite(zipPath string, block uint8, action bool) (graphics.Picture, error) {
	return readSpriteFrom(zipPath, MonsterSpriteArchive, block, action)
}

// ReadMonsterSpriteFrom 讓呼叫端指定檔案（`COMSPR.DAX` 或 `ICON.DAX`）。
func ReadMonsterSpriteFrom(zipPath, member string, block uint8, action bool) (graphics.Picture, error) {
	return readSpriteFrom(zipPath, member, block, action)
}

func readSpriteFrom(zipPath, member string, block uint8, action bool) (graphics.Picture, error) {
	id := block
	if action {
		if int(block)+MonsterSpriteActionOffset > 0xFF {
			return graphics.Picture{}, fmt.Errorf("Pool monster sprite %d has no action variant", block)
		}
		id = block + MonsterSpriteActionOffset
	}
	archive, err := zip.OpenReader(zipPath)
	if err != nil {
		return graphics.Picture{}, fmt.Errorf("open DOS ZIP: %w", err)
	}
	defer archive.Close()
	return readCombatIconBlock(archive.File, member, id)
}
