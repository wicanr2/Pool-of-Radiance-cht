package gamepack_test

import (
	"reflect"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// `1000h..102Eh`：戰士等級與種族都要大於 0（有號），否則 1。
func TestSweepLimitFollowsTheEntry7Tail(t *testing.T) {
	for _, tc := range []struct {
		fighter, race, want uint8
	}{
		{8, 7, 8},    // chrdatd1：戰士 8 級
		{0, 7, 1},    // 牧師、法師
		{1, 0, 1},    // 種族 0（ACOLYTE 那種怪物記錄）
		{0x80, 7, 1}, // 有號比較：80h 是負的
		{4, 0x80, 1},
	} {
		if got := gamepack.SweepLimit(tc.fighter, tc.race); got != tc.want {
			t.Fatalf("SweepLimit(%d, %d) = %d, want %d", tc.fighter, tc.race, got, tc.want)
		}
	}
}

func TestSweepTargetsFollowsEntry10(t *testing.T) {
	hitDice := map[uint8]uint8{5: 0, 6: 1, 7: 0, 8: 0, 9: 0}
	base := gamepack.SweepInput{
		Remaining: 1, Limit: 3, TargetHitDice: 0, Distance: 1,
		Nearby: []uint8{5, 6, 7, 8, 9}, Target: 8,
		HitDice: func(who uint8) uint8 { return hitDice[who] },
	}
	// 目標換到第一格（原本第一格的 5 換到目標那一格），跳過 +73h 非 0 的 6，額度 3。
	if got, want := gamepack.SweepTargets(base), []uint8{8, 7, 5}; !reflect.DeepEqual(got, want) {
		t.Fatalf("sweep order = %v, want %v", got, want)
	}
	for name, mutate := range map[string]func(*gamepack.SweepInput){
		"剩下的攻擊次數不少於上限（0E99h）": func(in *gamepack.SweepInput) { in.Remaining = 3 },
		"目標有生命骰（0EAFh）":       func(in *gamepack.SweepInput) { in.TargetHitDice = 1 },
		"不在隔壁（0EC8h）":         func(in *gamepack.SweepInput) { in.Distance = 2 },
		"小怪不比攻擊次數多（0F4Dh）":    func(in *gamepack.SweepInput) { in.Remaining = 2; in.Nearby = []uint8{5, 6, 8} },
	} {
		in := base
		mutate(&in)
		if got := gamepack.SweepTargets(in); got != nil {
			t.Fatalf("%s：應該照一般攻擊打，卻掃了 %v", name, got)
		}
	}
	// 上限 1 的非戰士：攻擊次數至少 1，一定不掃。
	in := base
	in.Limit = 1
	if got := gamepack.SweepTargets(in); got != nil {
		t.Fatalf("上限 1 掃了 %v", got)
	}
}
