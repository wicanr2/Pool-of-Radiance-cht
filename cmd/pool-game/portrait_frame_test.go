package main

import (
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/assets"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 原版人物資料頁的肖像框與整頁外框共用上、右兩邊；以下位置釘住新增的
// 左、下兩邊及三個交點，避免只畫一個看似相近的素色矩形（spec 130）。
func TestSheetPortraitFrameSymbolsMatchTheDOSShot(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	_, band4, err := gamepack.ReadDOSGlobalSymbolBands(zipPath)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	width := int(band4.WidthUnits) * 8
	if width != frameTileSize || int(band4.ItemCount) <= frameHorizontalItem {
		t.Fatalf("第 4 帶是 %d 寬、%d item，接不上肖像框", width, band4.ItemCount)
	}
	symbol := func(item int) []uint8 {
		base := item * width * frameTileSize
		return band4.Pixels[base : base+width*frameTileSize]
	}
	shot := filepath.Join("..", "..", "docs", "reference", "original-dos",
		"character-flow", "14-character-sheet.png")
	cases := []struct {
		name        string
		item        int
		column, row int
	}{
		{"左上交點", frameCornerItem, sheetPortraitLeftTileCol, 0},
		{"左緣第一格", frameVerticalItem, sheetPortraitLeftTileCol, 1},
		{"左緣最後一格", frameVerticalItem, sheetPortraitLeftTileCol, sheetPortraitBottomTileRow - 1},
		{"左下交點", frameCornerItem, sheetPortraitLeftTileCol, sheetPortraitBottomTileRow},
		{"下緣第一格", frameHorizontalItem, sheetPortraitLeftTileCol + 1, sheetPortraitBottomTileRow},
		{"下緣最後一格", frameHorizontalItem, sheetPortraitRightTileCol - 1, sheetPortraitBottomTileRow},
		{"右下交點", frameCornerItem, sheetPortraitRightTileCol, sheetPortraitBottomTileRow},
	}
	for _, testCase := range cases {
		want, err := cropDOSShot(shot, testCase.column*frameTileSize,
			testCase.row*frameTileSize, frameTileSize, frameTileSize)
		if err != nil {
			t.Fatal(err)
		}
		got := symbol(testCase.item)
		if diff := firstDifference(got, want); diff >= 0 {
			t.Errorf("%s（item %#x）第 %d 格不同：素材 %d、截圖 %d\n%s",
				testCase.name, testCase.item, diff, got[diff], want[diff],
				sideBySide(got, want))
		}
	}
}

// 框內剛好是 88×88 native pixels；以 selector 兩端與兩種語言實際走共用
// renderer，確保外框不因肖像組合或繁中欄位而改變幾何。
func TestSheetPortraitFrameFitsSelectorExtremesInBothLanguages(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	if sheetPortraitLeft != (sheetPortraitLeftTileCol+1)*frameTileSize*2 ||
		sheetPortraitTop != frameTileSize*2 ||
		sheetPortraitLeft+88*2 != sheetPortraitRightTileCol*frameTileSize*2 ||
		sheetPortraitTop+88*2 != sheetPortraitBottomTileRow*frameTileSize*2 {
		t.Fatal("88×88 肖像內容沒有精確貼齊框內四邊")
	}
	member := poolsave.Character{Name: "HERO", RaceID: "human", GenderID: "male",
		ClassID: "fighter", AlignmentID: "lawful-good", Age: 21,
		Abilities: [6]int{16, 12, 11, 14, 13, 8}, MaxHP: 8, CurrentHP: 8,
		PortraitHead: 1, PortraitBody: 1, IconSize: 1}
	for _, selectors := range [][2]uint8{{1, 1}, {14, 12}} {
		portrait, err := application.loadPortrait(selectors[0], selectors[1])
		if err != nil {
			t.Fatal(err)
		}
		if got := portrait.Bounds().Size(); got.X != 88 || got.Y != 88 {
			t.Fatalf("肖像 %d/%d 尺寸 %v，預期 88×88", selectors[0], selectors[1], got)
		}
		member.PortraitHead, member.PortraitBody = selectors[0], selectors[1]
		for _, lang := range []language{languageEnglish, languageTraditionalChinese} {
			application.language = lang
			skin := application.currentTheme()
			screen := ebiten.NewImage(logicalWidth, logicalHeight)
			screen.Fill(skin.background)
			application.drawFrame(screen, skin.foreground, skin.accent)
			drawSheetFor(screen, application, member, portrait, skin.foreground, skin.accent)
		}
	}
}

// 88×88 肖像必須放進框內的 native (224,8)..(311,95)，不是從框的左緣
// (216,8) 起畫；後者會把圖往左挪八格，再被左框裁掉（spec 130）。
func TestSheetPortraitPixelsMatchTheDOSInterior(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	parts, err := assets.ReadCreationPortraitParts(zipPath, 1, 1)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	portrait, err := assets.ComposeCreationPortrait(parts)
	if err != nil {
		t.Fatal(err)
	}
	shot := filepath.Join("..", "..", "docs", "reference", "original-dos",
		"character-flow", "14-character-sheet.png")
	want, err := cropDOSShot(shot, 224, 8, 88, 88)
	if err != nil {
		t.Fatal(err)
	}
	if diff := firstDifference(portrait.Pixels, want); diff >= 0 {
		t.Fatalf("肖像內部第 %d 格不同：素材 %d、原版畫面 %d", diff,
			portrait.Pixels[diff], want[diff])
	}
}
