package gamepack_test

import (
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// spec 112：active 力量效果到期後，仍有效的下一強效果要接管。
func TestStrengthEffectsRestoreAndReactivateOverlappingEffects(t *testing.T) {
	list, value, percentile, raised := gamepack.ApplyStrengthEffect(nil,
		gamepack.EnlargeEffectCode, 30, 17, 0, 18, 51)
	if !raised || value != 18 || percentile != 51 || len(list) != 1 ||
		!list[0].NeedsTeardown() {
		t.Fatalf("第一個力量效果：list=%v value=%d/%d raised=%v",
			list, value, percentile, raised)
	}
	list, value, percentile, raised = gamepack.ApplyStrengthEffect(list,
		gamepack.GiantStrengthEffectCode, 10, value, percentile, 21, 0)
	if !raised || value != 21 || percentile != 0 || len(list) != 2 {
		t.Fatalf("第二個力量效果：list=%v value=%d/%d raised=%v",
			list, value, percentile, raised)
	}

	// 較強且較短的 21 先到期，應回到仍有效的 18/51。
	expired := list[1]
	list = list[:1]
	value, percentile = gamepack.ExpireStrengthEffect(list, expired, value, percentile)
	if value != 18 || percentile != 51 {
		t.Fatalf("較強效果到期後是 %d/%d，預期 18/51", value, percentile)
	}
	// 最後一個到期才回到施法前的 17。
	expired = list[0]
	list = list[:0]
	value, percentile = gamepack.ExpireStrengthEffect(list, expired, value, percentile)
	if value != 17 || percentile != 0 {
		t.Fatalf("全部到期後是 %d/%d，預期 17/0", value, percentile)
	}
}

func TestWeakerStrengthEffectExpiresWithoutChangingTheActiveOne(t *testing.T) {
	list, value, percentile, _ := gamepack.ApplyStrengthEffect(nil,
		gamepack.GiantStrengthEffectCode, 30, 17, 0, 21, 0)
	list, value, percentile, raised := gamepack.ApplyStrengthEffect(list,
		gamepack.EnlargeEffectCode, 10, value, percentile, 18, 76)
	if raised || value != 21 || percentile != 0 {
		t.Fatalf("較弱效果不該蓋過 21，得到 %d/%d raised=%v", value, percentile, raised)
	}
	expired := list[1]
	list = list[:1]
	value, percentile = gamepack.ExpireStrengthEffect(list, expired, value, percentile)
	if value != 21 || percentile != 0 {
		t.Fatalf("inactive 效果到期改動了能力：%d/%d", value, percentile)
	}
}

func TestInactiveOlderStrengthEffectCanExpireBeforeActiveEffect(t *testing.T) {
	list, value, percentile, _ := gamepack.ApplyStrengthEffect(nil,
		gamepack.EnlargeEffectCode, 10, 17, 0, 18, 51)
	list, value, percentile, _ = gamepack.ApplyStrengthEffect(list,
		gamepack.GiantStrengthEffectCode, 30, value, percentile, 21, 0)
	older := list[0]
	list = list[1:]
	value, percentile = gamepack.ExpireStrengthEffect(list, older, value, percentile)
	if value != 21 || percentile != 0 {
		t.Fatalf("較舊 inactive 效果先到期改動了能力：%d/%d", value, percentile)
	}
	value, percentile = gamepack.ExpireStrengthEffect(nil, list[0], value, percentile)
	if value != 17 || percentile != 0 {
		t.Fatalf("最後 active 效果到期沒有回到基礎值：%d/%d", value, percentile)
	}
}
