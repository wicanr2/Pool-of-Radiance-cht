package gamepack

import "testing"

// spec 155 的純規則。戰場上的接點另有 cmd/pool-game/death_effects_test.go 從按鍵驗。

func TestBoarRallyHitPointsFollowTheDyingCounter(t *testing.T) {
	for overkill, want := range map[int]int{-1: 0, 0: 6, 1: 5, 5: 1, 6: 0, 9: 0, 12: 0} {
		if got := BoarRallyHitPoints(overkill); got != want {
			t.Errorf("overkill %d: %d, want %d", overkill, got, want)
		}
	}
}

func TestRegenerateAddsThreeUpToTheMaximum(t *testing.T) {
	list := EffectList{NewEffectNode(RegenerationEffectCode, 0, EffectUndispellable, false)}
	for _, tc := range [][3]int{{10, 30, 13}, {29, 30, 30}, {30, 30, 30}} {
		if got := Regenerate(list, tc[0], tc[1]); got != tc[2] {
			t.Errorf("%d/%d → %d, want %d", tc[0], tc[1], got, tc[2])
		}
	}
	if got := Regenerate(nil, 10, 30); got != 10 {
		t.Errorf("without 62h: %d", got)
	}
}

// `0FB7h`：行動者手上沒有武器、或貼身，連骰都不擲；`68h` 60%、`78h` 只接型別 57h／58h 50%。
// `75h` 聖水 Roll(1, 6) ＋ 1；`7Ah` 油瓶 3d8，`6777h & 9` 再加骰數。
func TestMeleeTargetGroupAvoidsAndOverrides(t *testing.T) {
	item := func(itemType uint8) []byte {
		raw := make([]byte, MonsterItemRecordSize)
		raw[ItemTypeOffset] = itemType
		return raw
	}
	rolled := 0
	roll := func(count, sides int) int { rolled++; return count * 2 }
	for _, tc := range []struct {
		name     string
		code     uint8
		wielded  []byte
		distance int
		flags    uint8
		damage   int
		avoided  bool
		rolls    int
	}{
		{"kreen far", ThriKreenDodgeEffectCode, item(0x24), 3, 0, 0, true, 1},
		{"kreen adjacent", ThriKreenDodgeEffectCode, item(0x24), 1, 0, 7, false, 0},
		{"kreen bare hands", ThriKreenDodgeEffectCode, nil, 3, 0, 7, false, 0},
		{"giant catches a rock", GiantCatchEffectCode, item(0x57), 3, 0, 0, true, 1},
		{"giant and a sword", GiantCatchEffectCode, item(0x24), 3, 0, 7, false, 0},
		{"holy water", HolyWaterEffectCode, item(0x55), 1, 0, 3, false, 1},
		{"oil flask", FireVulnerabilityEffectCode, item(0x56), 1, 0, 6, false, 1},
		{"bane on a mummy", FireVulnerabilityEffectCode, item(0x24), 1, CreatureBaneDamageFlags, 7 + 2, false, 0},
	} {
		rolled = 0
		outcome := MeleeTargetEffects{
			Effects:     EffectList{NewEffectNode(tc.code, 0, EffectUndispellable, false)},
			DamageFlags: tc.flags,
			Dice:        2,
			Roll:        roll,
			Hitting:     tc.wielded,
			Wielded:     tc.wielded,
			Distance:    tc.distance,
		}.Apply(7)
		if outcome.Damage != tc.damage || outcome.Avoided != tc.avoided || rolled != tc.rolls {
			t.Errorf("%s: damage %d avoided %v rolls %d, want %d %v %d", tc.name, outcome.Damage,
				outcome.Avoided, rolled, tc.damage, tc.avoided, tc.rolls)
		}
	}
}

// `65h`：身上沒有 `62h`、`3Bh` 才掛 `3Bh`；群組 6 同一支。
func TestTrollWoundArmsRegenerationOnce(t *testing.T) {
	list := EffectList{NewEffectNode(TrollWoundEffectCode, 0, EffectUndispellable, false)}
	list = TrollWounded(list)
	if !list.Has(RegenerationPendingEffectCode) || len(list) != 2 {
		t.Fatalf("first wound: %+v", list)
	}
	if again := TrollWounded(list); len(again) != 2 {
		t.Fatalf("second wound attached another 3Bh: %+v", again)
	}
	outcome := SpellDamageEffects{Effects: EffectList{NewEffectNode(TrollWoundEffectCode, 0, 0xff, false)}}.Apply(5)
	if !outcome.Effects.Has(RegenerationPendingEffectCode) || outcome.Damage != 5 {
		t.Fatalf("group 6: %+v damage %d", outcome.Effects, outcome.Damage)
	}
}

// `225Bh`：`2Bh` 帶走 `2Ch`、`1Fh`，`32h` 帶走 `39h`；只有伴隨碼的不算解到。
func TestCureDiseaseChainTakesTheCompanions(t *testing.T) {
	node := func(code uint8) EffectNode { return NewEffectNode(code, 0, EffectUndispellable, false) }
	list, cured := CureDiseaseChain(EffectList{node(0x2b), node(0x2c), node(0x1f), node(0x32), node(0x39)})
	if !cured || len(list) != 0 {
		t.Fatalf("weakening and rot: cured %v left %+v", cured, list)
	}
	list, cured = CureDiseaseChain(EffectList{node(0x2c), node(0x39)})
	if cured || len(list) != 2 {
		t.Fatalf("companions alone: cured %v left %+v", cured, list)
	}
}

// entry 21：死透（6）的不治、身上有 `32h` 的不治；倒著的瀕死改昏迷，戰鬥外的昏迷站起來。
func TestHealByEntry21(t *testing.T) {
	rot := EffectList{NewEffectNode(NoHealingEffectCode, 0, EffectUndispellable, false)}
	for _, tc := range []struct {
		name     string
		list     EffectList
		state    uint8
		down     bool
		combat   bool
		hp       int
		healed   bool
		newState uint8
		stands   bool
	}{
		{"standing", nil, 0, false, true, 20, true, 0, false},
		{"capped", nil, 0, false, true, 20, true, 0, false},
		{"dead", nil, DeadState, true, true, 0, false, DeadState, false},
		{"rot", rot, 0, false, true, 10, false, 0, false},
		{"dying in combat", nil, DyingState, true, true, 10, true, UnconsciousState, false},
		{"dying in camp", nil, DyingState, true, false, 10, true, UnconsciousState, true},
	} {
		start := 10
		if tc.name == "capped" {
			start = 18
		}
		if tc.state == DeadState {
			start = 0
		}
		result := HealByEntry21(tc.list, tc.state, tc.down, start, 20, 10, tc.combat)
		if result.Healed != tc.healed || result.State != tc.newState || result.StandsUp != tc.stands ||
			(tc.healed && result.HitPoints != min(start+10, 20)) {
			t.Errorf("%s: %+v", tc.name, result)
		}
	}
}

// `15F7h` 的第三個參數沒有讀：`45h` 推的 −2 不作用；持續先擲再豁免。
func TestParalysisAttacksRollTheirDurations(t *testing.T) {
	roll := func(count, sides int) int { return count * sides }
	drider := EffectList{NewEffectNode(0x45, 0, EffectUndispellable, false)}
	if got := ParalysisAttacks(drider, 2, 7, roll); len(got) != 1 || got[0] != 9+10 {
		t.Fatalf("drider: %v", got)
	}
	if got := ParalysisAttacks(drider, 1, 7, roll); len(got) != 0 {
		t.Fatalf("drider first form: %v", got)
	}
}
