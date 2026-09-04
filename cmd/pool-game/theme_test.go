package main

import (
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/golden-box-remake-engine/graphics"
)

// 原版那一套用的就是 EGA 十六色，一格都不能換——換了就不是原版的畫面。
func TestClassicThemeKeepsTheOriginalPalette(t *testing.T) {
	if classicTheme().palette != graphics.EGA16 {
		t.Fatal("原版主題的色盤不是 EGA16")
	}
}

// 現代那一套的第 0 格與第 15 格要與介面的背景、前景同色，
// 否則會出現「文字是米白、圖還是純白」那種兩套配色打架的畫面。
func TestModernThemeTiesArtToTheInterfaceColours(t *testing.T) {
	skin := modernTheme()
	if skin.palette[0] != skin.background {
		t.Fatalf("色盤第 0 格是 %v，介面背景是 %v", skin.palette[0], skin.background)
	}
	if skin.palette[15] != skin.foreground {
		t.Fatalf("色盤第 15 格是 %v，介面前景是 %v", skin.palette[15], skin.foreground)
	}
	// 十六格都要不透明而且互不相同：重複的格子會把原版美術的明暗關係壓平。
	seen := map[color.RGBA]bool{}
	for index, shade := range skin.palette {
		if shade.A != 255 {
			t.Fatalf("色盤第 %d 格不是不透明", index)
		}
		if seen[shade] {
			t.Fatalf("色盤第 %d 格與前面某一格同色", index)
		}
		seen[shade] = true
	}
}

// F2 換主題時，算過一次就存起來的素材要跟著重畫；每格重算的不必管。
func TestSwitchThemeRedrawsTheCachedArt(t *testing.T) {
	title, portrait, icons := 0, 0, 0
	a := &app{
		portrait:     ebiten.NewImage(1, 1),
		iconReady:    ebiten.NewImage(1, 1),
		reloadTitle:  func() error { title++; return nil },
		loadPortrait: func(uint8, uint8) (*ebiten.Image, error) { portrait++; return ebiten.NewImage(1, 1), nil },
		loadIcon: func(uint8, uint8, uint8, bool, [6][2]uint8) (*ebiten.Image, error) {
			icons++
			return ebiten.NewImage(1, 1), nil
		},
	}
	if err := a.switchTheme(); err != nil {
		t.Fatal(err)
	}
	if !a.modern {
		t.Fatal("switchTheme 沒有換過去")
	}
	if title != 1 || portrait != 1 || icons != 2 {
		t.Fatalf("重畫次數：標題 %d 肖像 %d 圖示 %d，預期 1 1 2", title, portrait, icons)
	}
	// 色盤跟著換。
	if a.artPalette() != modernTheme().palette {
		t.Fatal("換主題之後素材色盤沒有跟著換")
	}
	if err := a.switchTheme(); err != nil {
		t.Fatal(err)
	}
	if a.artPalette() != graphics.EGA16 {
		t.Fatal("換回原版之後色盤不是 EGA16")
	}
}

// 沒有素材可重畫時 F2 不該出錯——標題畫面之前 portrait 與 icon 都是 nil。
func TestSwitchThemeWithoutCachedArt(t *testing.T) {
	a := &app{}
	if err := a.switchTheme(); err != nil {
		t.Fatal(err)
	}
	if !a.modern {
		t.Fatal("switchTheme 沒有換過去")
	}
}
