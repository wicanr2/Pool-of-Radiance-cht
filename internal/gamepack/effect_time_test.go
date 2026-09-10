package gamepack_test

import (
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// 持續 0 的節點在 overlay-20 `0168h` 就被跳過：不遞減、也不到期。
// 這是「0 ＝ 永久」在時間這一側的樣子。
func TestPermanentEffectsNeverExpire(t *testing.T) {
	list := gamepack.EffectList{{Code: 0x21}, {Code: 0x37}}
	got, expired := list.AdvanceEffects(60 * 24 * 30)
	if len(expired) != 0 {
		t.Fatalf("持續 0 的節點到期了：%v", expired)
	}
	if len(got) != 2 || got[0].Duration() != 0 || got[1].Duration() != 0 {
		t.Fatalf("持續被動過：%v", got)
	}
}

// `018D` 的判準是 `經過量 >= 持續`，不是大於——所以剛好走完的那一分鐘就到期。
func TestEffectExpiresWhenElapsedReachesTheDuration(t *testing.T) {
	for _, tc := range []struct {
		name     string
		duration uint16
		minutes  int
		expired  bool
		left     uint16
	}{
		{"還沒到", 10, 9, false, 1},
		{"剛好到", 10, 10, true, 0},
		{"超過", 10, 11, true, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			list := gamepack.EffectList{gamepack.NewEffectNode(0x3B, tc.duration, 5, false)}
			got, expired := list.AdvanceEffects(tc.minutes)
			if tc.expired {
				if len(expired) != 1 || len(got) != 0 {
					t.Fatalf("剩 %v，到期 %v", got, expired)
				}
				return
			}
			if len(expired) != 0 || len(got) != 1 {
				t.Fatalf("剩 %v，到期 %v", got, expired)
			}
			if got[0].Duration() != tc.left {
				t.Fatalf("持續剩 %d，預期 %d", got[0].Duration(), tc.left)
			}
		})
	}
}

// `0197h` 是 `sub` 不是 `dec`：一次減掉經過的量，不是固定一。
func TestEffectDurationDropsByTheWholeElapsedAmount(t *testing.T) {
	list := gamepack.EffectList{gamepack.NewEffectNode(0x3B, 300, 5, false)}
	got, _ := list.AdvanceEffects(60)
	if got[0].Duration() != 240 {
		t.Fatalf("推 60 分之後持續是 %d，預期 240", got[0].Duration())
	}
}

// 十分鐘一批（`0x97`）不改變單人的結果——切批只是為了讓到期的收尾能掛新節點。
// 這一條釘住「切批不會多減也不會少減」。
func TestBatchingDoesNotChangeTheTotalDecrement(t *testing.T) {
	long := gamepack.EffectList{gamepack.NewEffectNode(0x3B, 1000, 5, false)}
	got, _ := long.AdvanceEffects(3 * gamepack.EffectTimeBatch)
	if got[0].Duration() != 1000-30 {
		t.Fatalf("推三批之後持續是 %d，預期 %d", got[0].Duration(), 1000-30)
	}
	// 混一個永久節點，證明它不佔批次也不被摘掉。
	mixed := gamepack.EffectList{
		{Code: 0x21},
		gamepack.NewEffectNode(0x3B, 25, 5, false),
	}
	rest, expired := mixed.AdvanceEffects(25)
	if len(expired) != 1 || expired[0].Code != 0x3B {
		t.Fatalf("到期的是 %v", expired)
	}
	if len(rest) != 1 || rest[0].Code != 0x21 {
		t.Fatalf("剩下的是 %v", rest)
	}
}

// 推 0 分鐘什麼都不該動——走一步被牆擋下來的那一次就是這個情況（spec 118）。
func TestAdvancingZeroMinutesChangesNothing(t *testing.T) {
	list := gamepack.EffectList{gamepack.NewEffectNode(0x3B, 7, 5, false)}
	got, expired := list.AdvanceEffects(0)
	if len(expired) != 0 || len(got) != 1 || got[0].Duration() != 7 {
		t.Fatalf("推 0 分之後是 %v／%v", got, expired)
	}
}
