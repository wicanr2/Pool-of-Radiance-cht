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

// 模式 0Ah 的四支分邊（overlay-22 `0F35h`／`2724h`）：祝福留施法者那一邊、剔掉貼身有
// 敵人的；詛咒留對面；急速最多施法者等級個、身上有緩速的解掉緩速、剔掉但額度照扣。
func TestFilterSpellSideFollowsTheHandlers(t *testing.T) {
	sides := map[uint8]uint8{2: 0, 3: 0, 4: 1, 5: 0, 6: 1, 7: 0}
	query := SpellSideQuery{
		Side: func(index uint8) (uint8, bool) {
			side, ok := sides[index]
			return side, ok
		},
		Engaged: func(index uint8) (bool, error) { return index == 3, nil },
		// 5 身上帶著緩速：急速的 `2724h` 推的是 2Ah，把它解掉、這次不加速。
		CancelEffect: func(index uint8, code uint8) bool { return index == 5 && code == SlowEffectCode },
	}
	list := []uint8{2, 3, 4, 5, 6, 7, 9}
	for _, check := range []struct {
		name  string
		id    uint8
		level int
		want  []uint8
	}{
		{"祝福", SpellIDBless, 1, []uint8{2, 5, 7}},
		{"詛咒", SpellIDCurse, 1, []uint8{4, 6}},
		// 額度 3：2、3 各扣一，5 扣一但緩速被解掉、這次不加速，7 沒額度了。
		{"急速", SpellIDHaste, 3, []uint8{2, 3}},
		{"緩速", SpellIDSlow, 1, []uint8{4}},
	} {
		filter, ok := SpellSideFilterFor(check.id)
		if !ok {
			t.Fatalf("%s 沒有分邊常式", check.name)
		}
		got, err := FilterSpellSide(filter, list, 0, check.level, query)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != len(check.want) {
			t.Fatalf("%s 留下 %v，原版是 %v", check.name, got, check.want)
		}
		for index := range got {
			if got[index] != check.want[index] {
				t.Fatalf("%s 留下 %v，原版是 %v", check.name, got, check.want)
			}
		}
	}
	if _, ok := SpellSideFilterFor(SpellIDFireball); ok {
		t.Fatal("火球術不走分邊常式")
	}
}

// 閃電束是模式 8、瞄一點；編號 3Ch 走同一支射線，豁免類別與規則同參數表。
func TestRaySpellsCarryTheHandlerLiterals(t *testing.T) {
	parameters, err := ReadDOSSpellParameters(poolZipPath())
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	if got := parameters[SpellIDLightningBolt].TargetMode(); got != SpellTargetBolt {
		t.Fatalf("閃電束的模式是 %#x", got)
	}
	bolt, err := CastSpell(SpellIDLightningBolt, parameters, 6, maxRoller{})
	if err != nil {
		t.Fatal(err)
	}
	if bolt.Ray == nil || *bolt.Ray != (SpellRayEffect{Length: 8, Damage: bolt.Damage,
		SaveCategory: 4, Surcharge: true}) {
		t.Fatalf("閃電束的射線 %+v（傷害 %d）", bolt.Ray, bolt.Damage)
	}
	ray, err := CastSpell(SpellIDRayDamage, parameters, 6, maxRoller{})
	if err != nil {
		t.Fatal(err)
	}
	if ray.Ray == nil || *ray.Ray != (SpellRayEffect{Length: 3, Damage: 20, SaveCategory: 4}) {
		t.Fatalf("編號 3Ch 的射線 %+v", ray.Ray)
	}
	if ray.Damage != 26 {
		t.Fatalf("編號 3Ch 打瞄準那一格是 Roll(1, 6) + 20，最大 26，拿到 %d", ray.Damage)
	}
}
