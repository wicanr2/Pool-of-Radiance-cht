package gamepack_test

import (
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// 最小值從 FFh 開始、最大值從 0 開始，所以空隊伍原樣寫回去——那是原版的
// 初始值，不是「沒算」。平均用整除。
func TestCheckPartySummaryKeepsTheOriginalInitialValues(t *testing.T) {
	empty := gamepack.CheckPartySummary(nil)
	if empty.Minimum != gamepack.CheckPartyInitialMinimum || empty.Maximum != 0 || empty.Average != 0 {
		t.Fatalf("空隊伍得到 %+v", empty)
	}
	got := gamepack.CheckPartySummary([]uint8{12, 9, 3, 12})
	if got.Minimum != 3 || got.Maximum != 12 {
		t.Fatalf("%+v", got)
	}
	if got.Average != 9 { // (12+9+3+12)/4 = 36/4
		t.Fatalf("平均 %d", got.Average)
	}
	// 整除，不四捨五入。
	if odd := gamepack.CheckPartySummary([]uint8{5, 6}); odd.Average != 5 {
		t.Fatalf("平均 %d，整除應該是 5", odd.Average)
	}
}

// 效果模式找到第一個就算數。
func TestCheckPartyEffectPresent(t *testing.T) {
	effects := [][]uint8{nil, {3, 19}, {7}}
	if gamepack.CheckPartyEffectPresent(effects, 19) != 1 {
		t.Fatal("隊上有 19 卻回 0")
	}
	if gamepack.CheckPartyEffectPresent(effects, 20) != 0 {
		t.Fatal("隊上沒有 20 卻回 1")
	}
	if gamepack.CheckPartyEffectPresent(nil, 19) != 0 {
		t.Fatal("空隊伍不該找到東西")
	}
}

// 只有兩個位址是讀出來的欄位；其餘要失敗，不能猜一個去統計——猜出來的
// 是一個看起來合理的數字，而那分不出對錯。
func TestCheckPartyModeRefusesUnknownSelectors(t *testing.T) {
	if mode, err := gamepack.CheckPartyMode(true, 0); err != nil || mode != gamepack.CheckPartyEffectMode {
		t.Fatalf("字面值 0 應該是效果模式：%d %v", mode, err)
	}
	for _, address := range []uint16{gamepack.CheckPartyFieldFindTraps, gamepack.CheckPartyFieldMovement} {
		if mode, err := gamepack.CheckPartyMode(false, address); err != nil || mode != address {
			t.Fatalf("%#04x：%d %v", address, mode, err)
		}
	}
	if _, err := gamepack.CheckPartyMode(false, 0x1234); err == nil {
		t.Fatal("沒讀過的位址被接受了")
	}
}
