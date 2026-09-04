package treasure_test

import (
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/treasure"
)

// 寶石表逐段釘住，而且 1..100 每一點都要落在某一段裡——中間有洞的話，
// 玩家偶爾會遇到「估價 0 金幣」而且沒有人會注意到。
func TestGemValueCoversEveryRoll(t *testing.T) {
	want := map[int]int{1: 10, 25: 10, 26: 50, 50: 50, 51: 100, 70: 100,
		71: 500, 90: 500, 91: 1000, 99: 1000, 100: 5000}
	for roll, value := range want {
		got, err := treasure.GemValue(roll)
		if err != nil || got != value {
			t.Errorf("擲出 %d 值 %d（%v），應該是 %d", roll, got, err, value)
		}
	}
	for roll := 1; roll <= 100; roll++ {
		if _, err := treasure.GemValue(roll); err != nil {
			t.Fatalf("擲出 %d 沒有對應的價值：%v", roll, err)
		}
	}
	// 負對照：界外要回錯，不是回 0。
	for _, roll := range []int{0, 101, -1} {
		if _, err := treasure.GemValue(roll); err == nil {
			t.Errorf("擲出 %d 竟然查得到價值", roll)
		}
	}
}

// 珠寶是「一段範圍 ＋ 底價」，兩端都要對。
func TestJewelryValueSpansEachBand(t *testing.T) {
	lowest := func(int) int { return 0 }
	highest := func(limit int) int { return limit - 1 }
	want := []struct{ roll, low, high int }{
		{1, 100, 999}, {10, 100, 999},
		{11, 200, 1199}, {20, 200, 1199},
		{21, 300, 1799}, {40, 300, 1799},
		{41, 500, 2999}, {50, 500, 2999},
		{51, 1000, 5999}, {70, 1000, 5999},
		{71, 2000, 7999}, {90, 2000, 7999},
		{91, 2000, 11999}, {100, 2000, 11999},
	}
	for _, entry := range want {
		low, err := treasure.JewelryValue(entry.roll, lowest)
		if err != nil || low != entry.low {
			t.Errorf("擲出 %d 的最低價是 %d（%v），應該是 %d", entry.roll, low, err, entry.low)
		}
		high, err := treasure.JewelryValue(entry.roll, highest)
		if err != nil || high != entry.high {
			t.Errorf("擲出 %d 的最高價是 %d（%v），應該是 %d", entry.roll, high, err, entry.high)
		}
	}
	for roll := 1; roll <= 100; roll++ {
		if _, err := treasure.JewelryValue(roll, lowest); err != nil {
			t.Fatalf("擲出 %d 沒有對應的價值：%v", roll, err)
		}
	}
}

// 估價是金幣，賣得的是白金：1 白金 ＝ 5 金，所以除以 5 是幣別換算不是折價。
// `1F0Dh` 的 `div 5` 是整數除。
func TestSellPriceConvertsGoldToPlatinum(t *testing.T) {
	for value, want := range map[int]int{10: 2, 50: 10, 100: 20, 500: 100,
		1000: 200, 5000: 1000, 999: 199, 4: 0} {
		if got := treasure.SellPrice(value); got != want {
			t.Errorf("估價 %d 金幣賣得 %d 白金，應該是 %d", value, got, want)
		}
	}
}
