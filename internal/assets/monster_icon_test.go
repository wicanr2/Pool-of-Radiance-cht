package assets

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/golden-box-remake-engine/graphics"
)

// iconRow 把一列像素寫成 dosgolem 截圖那種字串：透明是 '.'，其餘是十六進位色號。
func iconRow(picture graphics.Picture, row int) string {
	width := picture.Width()
	line := make([]byte, width)
	for column := 0; column < width; column++ {
		pixel := picture.Pixels[row*width+column]
		switch {
		case pixel > 15:
			line[column] = '.'
		case pixel < 10:
			line[column] = '0' + pixel
		default:
			line[column] = 'a' + pixel - 10
		}
	}
	return string(line)
}

// 原版第一場遭遇（dosgolem 基準 `85-c`，戰場 (32,56) 那一格）的怪物與 `CPIC1.DAX` 區塊 2 逐像素
// 相同（222／222 個不透明像素）：紅色是圖本身的顏色，不是另外換的配色（spec 166）。
// 這兩列就是那一張截圖第 60、66 列（y = 56 + 4、56 + 10）從 x = 32 起的 24 個像素。
func TestMonsterIconIsTheCPICBlockWithItsOwnColours(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	if _, err := os.Stat(zipPath); err != nil {
		t.Skip("original DOS ZIP is intentionally not tracked")
	}
	goblin, err := ReadMonsterCombatIcon(zipPath, 1, 2, false)
	if err != nil {
		t.Fatal(err)
	}
	if goblin.Width() != 24 || goblin.Height() != 24 {
		t.Fatalf("CPIC1 區塊 2 是 %dx%d，want 24×24", goblin.Width(), goblin.Height())
	}
	for row, want := range map[int]string{
		4:  "..989...8..44...........",
		10: "...86.4448222248846888..",
	} {
		if got := iconRow(goblin, row); got != want {
			t.Fatalf("CPIC1 區塊 2 第 %d 列 %q，原版畫面是 %q", row, got, want)
		}
	}
	// 第二種怪物：最後一戰的 TYRANITHRAXUS（`LOAD MONSTER 42h, 1, 42h`，ECL5）是 CPIC5 區塊 42h，
	// 48×48、佔 2×2；動作圖在 C2h，同樣大小。
	for _, action := range []bool{false, true} {
		dragon, err := ReadMonsterCombatIcon(zipPath, 5, 0x42, action)
		if err != nil {
			t.Fatal(err)
		}
		if dragon.Width() != 48 || dragon.Height() != 48 {
			t.Fatalf("CPIC5 區塊 42h（action=%v）是 %dx%d，want 48×48", action, dragon.Width(), dragon.Height())
		}
		opaque := 0
		for _, pixel := range dragon.Pixels {
			if pixel == 0x0D {
				t.Fatal("換色表把 0Dh 換成 08h，圖上不該還有 0Dh")
			}
			if pixel < 16 {
				opaque++
			}
		}
		if opaque == 0 {
			t.Fatalf("CPIC5 區塊 42h（action=%v）整張透明", action)
		}
	}
	// 兩種怪物的色組不同：哥布林有紅（4／0Ch），巨龍那一張的色組另外算，不是同一套預設配色。
	if colours(goblin) == colours(mustIcon(t, zipPath, 5, 0x42)) {
		t.Fatal("兩種怪物用了同一組顏色，像是套了同一份預設配色")
	}
}

func mustIcon(t *testing.T, zipPath string, archive, block uint8) graphics.Picture {
	t.Helper()
	picture, err := ReadMonsterCombatIcon(zipPath, archive, block, false)
	if err != nil {
		t.Fatal(err)
	}
	return picture
}

func colours(picture graphics.Picture) [16]bool {
	var used [16]bool
	for _, pixel := range picture.Pixels {
		if pixel < 16 {
			used[pixel] = true
		}
	}
	return used
}

// `DS:0CE6h → 0CF6h`：只有 0Dh 換成 08h；透明不動。
func TestMonsterIconRecolourTable(t *testing.T) {
	picture := graphics.Picture{WidthUnits: 1, HeightUnits: 2, ItemCount: 1,
		Pixels: []uint8{0, 1, 4, 8, 0x0C, 0x0D, 0x0F, 16, 0x0D, 2, 3, 5, 6, 7, 9, 0x0E}}
	got := recolourMonsterIcon(picture)
	want := []uint8{0, 1, 4, 8, 0x0C, 0x08, 0x0F, 16, 0x08, 2, 3, 5, 6, 7, 9, 0x0E}
	for index := range want {
		if got.Pixels[index] != want[index] {
			t.Fatalf("pixel %d = %02X, want %02X", index, got.Pixels[index], want[index])
		}
	}
	if picture.Pixels[5] != 0x0D {
		t.Fatal("換色不能改到原本那一份")
	}
	if _, err := ReadMonsterCombatIcon("unused.zip", 0, 2, false); err == nil {
		t.Fatal("檔案組 0 不存在（CPIC1..8），要回錯誤")
	}
}
