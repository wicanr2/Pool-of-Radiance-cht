package assets_test

import (
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/assets"
)

// suneTempleShot 是 dosgolem 走進蘇恩神殿、答 y 之後的原版畫面
// （`workplace/dosgolem-ref-temple/64-y.png`，320×200；spec 165〈基準〉）。
const suneTempleShot = "../../docs/reference/original-dos/temple/64-y-sune-temple.png"

// TestSuneTemplePriestessMatchesTheDOSTempleShot：蘇恩神殿那一格（ECL3/0 的地點
// 分派索引 7，spec 102）在 `A0BCh` 寫 `SAVE 22 → @6DE1`、`A0C2h` 執行
// `PICTURE 24`。`6DE1h` 不是 `FFh`，走 HEAD／BODY 那條：`HEAD3` 區塊 22 疊
// `BODY3` 區塊 24，與原版那一框逐格相同。
func TestSuneTemplePriestessMatchesTheDOSTempleShot(t *testing.T) {
	portrait, err := assets.ReadNPCPortrait(dosZIP, 3, 22, 24)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	want, err := cropOriginalShot(suneTempleShot, 24, 24, 88, 88)
	if err != nil {
		t.Fatalf("原版截圖讀不出來: %v", err)
	}
	for index := range want {
		if portrait.Pixels[index] != want[index] {
			t.Fatalf("第 %d 格是 %d，原版是 %d（x=%d y=%d）",
				index, portrait.Pixels[index], want[index], index%88, index/88)
		}
	}
	// 負對照：另一座神殿（`A9DBh` `SAVE 57`、`A9E1h` `PICTURE 1`）的祭司對不上。
	other, err := assets.ReadNPCPortrait(dosZIP, 3, 57, 1)
	if err != nil {
		t.Fatal(err)
	}
	same := 0
	for index := range want {
		if other.Pixels[index] == want[index] {
			same++
		}
	}
	if same == len(want) {
		t.Fatal("HEAD3/57 + BODY3/1 也逐格相同，這條比對沒有鑑別力")
	}
}

// TestPICAnimationCarriesItsDelays：每一張前面那 4 bytes 是延遲（spec 135／165）。
// 船（區塊 41）四張 20／15／20／15，營火（區塊 29）兩張都是 2，寶箱（區塊 1）
// 四張 2／4／6／1。每一張都是 88×88，第二張以後已還原成完整圖。
func TestPICAnimationCarriesItsDelays(t *testing.T) {
	for _, want := range []struct {
		block  uint8
		delays []uint32
	}{
		{41, []uint32{20, 15, 20, 15}},
		{29, []uint32{2, 2}},
		{1, []uint32{2, 4, 6, 1}},
	} {
		animation, err := assets.ReadPICAnimation(dosZIP, 3, want.block)
		if err != nil {
			t.Skipf("DOS ZIP unavailable: %v", err)
		}
		if len(animation.Frames) != len(want.delays) || len(animation.Delays) != len(want.delays) {
			t.Fatalf("block %d: %d frames %d delays, want %d", want.block,
				len(animation.Frames), len(animation.Delays), len(want.delays))
		}
		for index, delay := range want.delays {
			if animation.Delays[index] != delay {
				t.Errorf("block %d frame %d delay %d, want %d", want.block, index, animation.Delays[index], delay)
			}
			frame := animation.Frames[index]
			if frame.Width() != 88 || frame.Height() != 88 {
				t.Errorf("block %d frame %d is %dx%d", want.block, index, frame.Width(), frame.Height())
			}
		}
		differs := false
		for index := range animation.Frames[0].Pixels {
			if animation.Frames[0].Pixels[index] != animation.Frames[1].Pixels[index] {
				differs = true
				break
			}
		}
		if !differs {
			t.Errorf("block %d: the first two frames are identical; the XOR delta was not applied", want.block)
		}
	}
	// 營火那一支讀出來的兩張，與這裡讀的是同一組。
	fire, err := assets.ReadCampFire(dosZIP, 3)
	if err != nil {
		t.Fatal(err)
	}
	animation, err := assets.ReadPICAnimation(dosZIP, 3, assets.CampFireBlock)
	if err != nil {
		t.Fatal(err)
	}
	for frame := range fire {
		for index := range fire[frame].Pixels {
			if fire[frame].Pixels[index] != animation.Frames[frame].Pixels[index] {
				t.Fatalf("camp fire frame %d pixel %d differs", frame, index)
			}
		}
	}
}

// 沒有的區塊要失敗，不要回一張空圖。
func TestPICAnimationMissingBlockFails(t *testing.T) {
	if _, err := assets.ReadPICAnimation(dosZIP, 3, 41); err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	if _, err := assets.ReadPICAnimation(dosZIP, 3, 200); err == nil {
		t.Fatal("PIC3 block 200 does not exist but was read")
	}
}
