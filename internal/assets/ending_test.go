package assets_test

import (
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/assets"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

const dosZIP = "../../Pool of Radiance (1988).zip"

// 五個區塊的形狀與位置：位置的單位是 8 像素（spec 108），換算之後三張小圖
// 都完整落在 120×120 那張場景裡。單位讀錯（例如當成 1 像素）會讓它們擠在
// 左上角，讀成 16 像素則會掉出畫面——這條測試兩種都擋得下來。
func TestEndingLayersFitTheScene(t *testing.T) {
	pictures, err := assets.ReadEndingPictures(dosZIP, gamepack.EndingLayerBlocks())
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	want := map[uint8][2]int{
		1: {120, 120}, 3: {120, 120}, 4: {56, 32}, 5: {16, 48}, 6: {16, 48},
	}
	for _, layer := range gamepack.EndingLayers() {
		picture, ok := pictures[layer.Block]
		if !ok {
			t.Fatalf("FINAL5.DAX 少了區塊 %d", layer.Block)
		}
		shape := [2]int{picture.Width(), picture.Height()}
		if shape != want[layer.Block] {
			t.Fatalf("區塊 %d 是 %v，預期 %v", layer.Block, shape, want[layer.Block])
		}
		if err := gamepack.CheckEndingLayerFits(layer, picture.Width(), picture.Height()); err != nil {
			t.Fatal(err)
		}
	}
}

// 後三層依隊伍人數出現（`[4937h]+67Ch`，spec 084／091）。
// 原版的比較是 `>`：人數要大於 1 才畫第三層。
func TestVisibleEndingLayersFollowPartySize(t *testing.T) {
	for _, item := range []struct {
		party  int
		blocks []uint8
	}{
		{1, []uint8{1, 3}},
		{2, []uint8{1, 3, 4}},
		{3, []uint8{1, 3, 4, 5}},
		{4, []uint8{1, 3, 4, 5, 6}},
		{6, []uint8{1, 3, 4, 5, 6}},
	} {
		layers := gamepack.VisibleEndingLayers(item.party)
		if len(layers) != len(item.blocks) {
			t.Fatalf("%d 人畫了 %d 層，預期 %d 層", item.party, len(layers), len(item.blocks))
		}
		for index, block := range item.blocks {
			if layers[index].Block != block {
				t.Fatalf("%d 人的第 %d 層是區塊 %d，預期 %d",
					item.party, index, layers[index].Block, block)
			}
		}
	}
}

// 疊出來的畫面是 120×120，而且真的疊過——只放底圖的話像素會與底圖相同。
func TestComposeEndingSceneOverlaysTheSprites(t *testing.T) {
	pictures, err := assets.ReadEndingPictures(dosZIP, gamepack.EndingLayerBlocks())
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	base, err := assets.ComposeEndingScene(pictures, gamepack.VisibleEndingLayers(1))
	if err != nil {
		t.Fatal(err)
	}
	full, err := assets.ComposeEndingScene(pictures, gamepack.VisibleEndingLayers(6))
	if err != nil {
		t.Fatal(err)
	}
	if full.Width() != gamepack.EndingSceneWidth || full.Height() != gamepack.EndingSceneHeight {
		t.Fatalf("結局畫面是 %dx%d", full.Width(), full.Height())
	}
	same := true
	for index := range base.Pixels {
		if base.Pixels[index] != full.Pixels[index] {
			same = false
			break
		}
	}
	if same {
		t.Fatal("六人隊伍的結局畫面與一人隊伍相同，三張小圖沒有疊上去")
	}
}
