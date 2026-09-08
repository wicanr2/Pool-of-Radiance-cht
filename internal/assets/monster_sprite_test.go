package assets

import (
	"os"
	"path/filepath"
	"testing"
)

// `COMSPR.DAX` 的每一格都是 24×24——與戰場一格同大（spec 129 量到每格 24 像素）。
// 站立與動作成對，動作那一張的編號是站立加 80h。
func TestMonsterSpritesAreOneBoardCell(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	if _, err := os.Stat(zipPath); err != nil {
		t.Skip("original DOS ZIP is intentionally not tracked")
	}
	blocks := MonsterSpriteBlocks()
	if len(blocks) != 13 {
		t.Fatalf("`COMSPR.DAX` 列了 %d 組，原版是 13 組", len(blocks))
	}
	for _, block := range blocks {
		for _, action := range []bool{false, true} {
			picture, err := ReadMonsterSprite(zipPath, block, action)
			// **這裡不能 skip**：ZIP 在，就代表區塊真的讀不出來。
			// 先前寫成 skip，`0Ch` 不存在時整條測試靜靜地通過了。
			if err != nil {
				t.Fatalf("怪物 sprite %d（action=%v）讀不出來：%v", block, action, err)
			}
			if picture.Width() != 24 || picture.Height() != 24 {
				t.Fatalf("怪物 sprite %d（action=%v）是 %dx%d，戰場一格是 24×24",
					block, action, picture.Width(), picture.Height())
			}
		}
	}
}
