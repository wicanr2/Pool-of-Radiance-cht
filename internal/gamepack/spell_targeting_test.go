package gamepack

import "testing"

// `20AEh` 的四條路照參數表 `+6` 分：定身術 3 個、定身怪物 4 個、火球術以一點
// 為中心預算 3、催眠術預算 1、祝福術（0Ah）預算 2、沉默術（1Fh）挑一個。
func TestSpellTargetPlanFollowsOverlay13(t *testing.T) {
	parameters, err := ReadDOSSpellParameters(poolZipPath())
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	for _, check := range []struct {
		id   uint8
		want SpellTargetPlan
	}{
		{SpellIDHoldPerson, SpellTargetPlan{Kind: SpellTargetKindCount, Count: 3}},
		{SpellIDHoldPersonAlt, SpellTargetPlan{Kind: SpellTargetKindCount, Count: 4}},
		{SpellIDMagicMissile, SpellTargetPlan{Kind: SpellTargetKindCount, Count: 1}},
		{SpellIDFireball, SpellTargetPlan{Kind: SpellTargetKindArea, AreaBudget: 3, PointAim: true}},
		{SpellIDSleep, SpellTargetPlan{Kind: SpellTargetKindArea, AreaBudget: 1, PointAim: true}},
		{SpellIDBless, SpellTargetPlan{Kind: SpellTargetKindArea, AreaBudget: 2, PointAim: true}},
		{25, SpellTargetPlan{Kind: SpellTargetKindOne}},  // Silence, 15' Radius：+6 = 1Fh
		{5, SpellTargetPlan{Kind: SpellTargetKindSelf}},  // Detect Magic
		{27, SpellTargetPlan{Kind: SpellTargetKindSelf}}, // Snake Charm：+6 = F0h，低四位是 0
	} {
		if got := parameters[check.id].TargetPlan(); got != check.want {
			t.Errorf("法術 %d 的收目標方式是 %+v，原版是 %+v", check.id, got, check.want)
		}
	}
}

// 火球術只在 `@49E6` 為 0 時以預算 2 重收（`2661h`）；別的法術不重收。
func TestFireballRecollectsOnlyWhenTheWalkFlagIsClear(t *testing.T) {
	if got := FireballOutdoorAreaBudget(SpellIDFireball, 0); got != FireballAreaBudget {
		t.Errorf("@49E6 = 0 時應該以 %d 重收，拿到 %d", FireballAreaBudget, got)
	}
	if got := FireballOutdoorAreaBudget(SpellIDFireballAlt, 0); got != FireballAreaBudget {
		t.Errorf("編號 40h 與火球術共用 262Eh，拿到 %d", got)
	}
	if got := FireballOutdoorAreaBudget(SpellIDFireball, 1); got != 0 {
		t.Errorf("@49E6 非 0 時留著 20AEh 的表，拿到 %d", got)
	}
	if got := FireballOutdoorAreaBudget(SpellIDSleep, 0); got != 0 {
		t.Errorf("催眠術不重收，拿到 %d", got)
	}
}
