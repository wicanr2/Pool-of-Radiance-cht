package assets_test

import (
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/assets"
)

// weaponShopShot 是 dosgolem 走進菲蘭武具店、按 y 之後的原版畫面
// （`workplace/dosgolem-ref-shop/79-y.png`，320×200；spec 164〈來源〉）。
const weaponShopShot = "../../docs/reference/original-dos/shop/79-y-weapon-shop.png"

// TestShopkeeperPortraitMatchesTheDOSShopShot：武具店的 ECL 在 `A8BFh` 寫
// `SAVE 42 → @6DE1`、`A8C5h` 執行 `PICTURE 9`。`6DE1h` 不是 `FFh`，所以走
// head／body 那條（spec 117）：`HEAD3` 區塊 42 疊 `BODY3` 區塊 9。
// 疊出來的 88×88 要與原版那一框逐格相同。
func TestShopkeeperPortraitMatchesTheDOSShopShot(t *testing.T) {
	portrait, err := assets.ReadNPCPortrait(dosZIP, 3, 42, 9)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	want, err := cropOriginalShot(weaponShopShot, 24, 24, 88, 88)
	if err != nil {
		t.Fatalf("原版截圖讀不出來: %v", err)
	}
	for index := range want {
		if portrait.Pixels[index] != want[index] {
			t.Fatalf("第 %d 格是 %d，原版是 %d（x=%d y=%d）",
				index, portrait.Pixels[index], want[index], index%88, index/88)
		}
	}
	// 負對照：同一個 body 配 Rolf 的 head（8）必須對不上，證明比對有鑑別力。
	rolf, err := assets.ReadNPCPortrait(dosZIP, 3, 8, 9)
	if err != nil {
		t.Fatal(err)
	}
	same := 0
	for index := range want {
		if rolf.Pixels[index] == want[index] {
			same++
		}
	}
	if same == len(want) {
		t.Fatal("Rolf 的 head 也逐格相同，這條比對沒有鑑別力")
	}
}

// TestAllFourShopkeepersCompose：珠寶店（`A838h`）與銀器店（`A950h`）寫的是
// `SAVE 63 → @6DE1` 再 `PICTURE 34`，雜貨店（`A242h`）與武具店同為 42／9。
// 兩組都要疊得出 88×88；缺區塊就會在這裡失敗，而不是在店裡安靜地退回視野。
func TestAllFourShopkeepersCompose(t *testing.T) {
	if _, err := assets.ReadNPCPortrait(dosZIP, 3, 8, 9); err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	for _, pair := range [][2]uint8{{42, 9}, {63, 34}} {
		portrait, err := assets.ReadNPCPortrait(dosZIP, 3, pair[0], pair[1])
		if err != nil {
			t.Fatalf("HEAD3/%d + BODY3/%d: %v", pair[0], pair[1], err)
		}
		if portrait.Width() != 88 || portrait.Height() != 88 {
			t.Fatalf("HEAD3/%d + BODY3/%d is %dx%d", pair[0], pair[1], portrait.Width(), portrait.Height())
		}
	}
}
