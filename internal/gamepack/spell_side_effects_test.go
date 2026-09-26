package gamepack

import (
	"fmt"
	"testing"
)

// 群組 10 的祝福／詛咒、群組 18 的急速／緩速與 entry 20 的覆蓋規則（spec 112，#81）。
func TestSideSpellEffectsFollowTheHandlers(t *testing.T) {
	blessed := EffectList{}.Append(NewEffectNode(BlessEffectCode, 6, 1, false))
	cursed := EffectList{}.Append(NewEffectNode(CurseEffectCode, 6, 1, false))
	modifier := func(attacker, target EffectList) int {
		value, _ := HitRollEffects{
			Attacker: HitRollCombatant{Effects: attacker},
			Target:   HitRollCombatant{Effects: target},
		}.Apply(10)
		return int(value) - 10
	}
	if got := modifier(blessed, nil); got != 1 {
		t.Errorf("祝福 %d，應該 +1", got)
	}
	if got := modifier(cursed, nil); got != -1 {
		t.Errorf("詛咒 %d，應該 −1", got)
	}
	// 群組 10 問的是攻擊者；祝福掛在目標身上不影響對它出手的人。
	if got := modifier(nil, blessed); got != 0 {
		t.Errorf("目標身上的祝福改了命中 %d", got)
	}
	// 同一個碼兩個節點：每個群組只問一次最早的那一個。
	if got := modifier(blessed.Append(NewEffectNode(BlessEffectCode, 6, 1, false)), nil); got != 1 {
		t.Errorf("兩個祝福節點疊成 %d", got)
	}

	hasted := RoundRateEffectsOf(EffectList{}.Append(NewEffectNode(HasteEffectCode, 8, 5, false)))
	slowed := RoundRateEffectsOf(EffectList{}.Append(NewEffectNode(SlowEffectCode, 8, 5, false)))
	for _, check := range []struct {
		name        string
		flags       RoundRateEffects
		rate, moves uint8
		wantRate    uint8
		wantMoves   uint8
	}{
		{"急速", hasted, 2, 24, 4, 48},
		{"緩速", slowed, 3, 24, 1, 12},
		{"急速 byte 溢位", hasted, 2, 192, 4, 128},
		{"不動", RoundRateImmobile, 2, 24, 2, 0},
	} {
		if got := AttackRateAfterEffects(check.rate, check.flags); got != check.wantRate {
			t.Errorf("%s：攻擊編碼 %d → %d，應該 %d", check.name, check.rate, got, check.wantRate)
		}
		if got := MovementAfterEffects(check.moves, check.flags); got != check.wantMoves {
			t.Errorf("%s：移動 %d → %d，應該 %d", check.name, check.moves, got, check.wantMoves)
		}
	}

	list := EffectList{}.Append(NewEffectNode(HasteEffectCode, 8, 5, false))
	list, aged := MarkHasteAged(list)
	if !aged || list[0].CasterLevel() != 5 {
		t.Fatalf("第一次要老一歲而且不動等級：aged %v level %d", aged, list[0].CasterLevel())
	}
	if _, again := MarkHasteAged(list); again {
		t.Error("位元 4 立過之後又老一歲")
	}

	// entry 20：舊的剩得比新的短 → 摘；一樣長 → 留舊的、新的照掛；新的不計時 → 摘。
	old := EffectList{}.Append(NewEffectNode(BlessEffectCode, 3, 1, false))
	if got := ApplySpellEffectNode(old, NewEffectNode(BlessEffectCode, 6, 1, false)); len(got) != 1 || got[0].Duration() != 6 {
		t.Errorf("較長的新節點應該取代舊的：%+v", got)
	}
	same := EffectList{}.Append(NewEffectNode(BlessEffectCode, 6, 1, false))
	if got := ApplySpellEffectNode(same, NewEffectNode(BlessEffectCode, 6, 1, false)); len(got) != 2 {
		t.Errorf("一樣長時原版兩個都留：%+v", got)
	}
	if got := ApplySpellEffectNode(same, NewEffectNode(BlessEffectCode, 0, 1, false)); len(got) != 1 || got[0].Duration() != 0 {
		t.Errorf("不計時的新節點應該取代舊的：%+v", got)
	}
}

// `07C7h` 的六個特例與一般式（#89）。擲骰照原版的引數順序記下來，順便釘住「每個特例各擲
// 一次、一般式不擲」。
func TestSpellEffectDurationFollows07C7h(t *testing.T) {
	var params SpellParameters
	params.Raw[spellParameterFixedDuration], params.Raw[spellParameterLevelDuration] = 4, 5
	cases := []struct {
		id       uint8
		combat   bool
		want     int
		wantRoll string
	}{
		{0x28, true, 30, "1d6"},    // 07D7h Roll(1, 6) × 10
		{0x39, true, 3, "5d4"},     // 07F9h Roll(5, 4)
		{0x3d, true, 3, "5d4"},     // 同上
		{0x3b, true, 70, "1d4"},    // 0811h Roll(1, 4) × 10 + 40
		{0x3f, true, 30, "2d10"},   // 0838h 戰鬥中 Roll(2, 10) × 10
		{0x3f, false, 130, "1d10"}, // 084Fh (Roll(1, 10) + 10) × 10
		{0x43, true, 0x5a0, ""},    // 086Eh
		{0x13, true, 4 + 5*3, ""},  // 0875h +4 + +5 × 等級
	}
	for _, tc := range cases {
		asked := ""
		got := SpellEffectDuration(tc.id, params, 3, tc.combat, func(count, sides int) int {
			asked += fmt.Sprintf("%dd%d", count, sides)
			return 3
		})
		if got != tc.want || asked != tc.wantRoll {
			t.Errorf("07C7h(%02Xh, combat %v) = %d rolling %q, want %d rolling %q",
				tc.id, tc.combat, got, asked, tc.want, tc.wantRoll)
		}
	}
}

// 群組 11／12／6 裡屬於這一批法術的碼（#89）。
func TestEffectOnlyGroupsFollowTheHandlers(t *testing.T) {
	shield := EffectList{}.Append(NewEffectNode(ShieldEffectCode, 5, 3, false))
	blind := EffectList{}.Append(NewEffectNode(BlindnessEffectCode, 0, 3, false))
	cursed := EffectList{}.Append(NewEffectNode(BestowCurseEffectCode, 30, 3, false))
	// 群組 11：`0BC9h` 減 4；`0664h` 墊到 39h，本來就高的不動。
	if got := HitCheckArmourClass(blind, 50); got != 46 {
		t.Errorf("blind AC %d, want 46", got)
	}
	if got := HitCheckArmourClass(shield, 50); got != 0x39 {
		t.Errorf("shield AC %d, want 57", got)
	}
	if got := HitCheckArmourClass(shield, 60); got != 60 {
		t.Errorf("shield lowered AC 60 to %d", got)
	}
	// 群組 12：護盾 +1、失明與降咒 −4、祈禱依節點 +3 位元 4 與擲豁免那一個的邊。
	prayer := EffectList{}.Append(NewEffectNode(PrayerAreaEffectCode, 3, 0x13, false))
	for _, tc := range []struct {
		list EffectList
		side uint8
		want int
	}{
		{shield, 0, 11}, {blind, 0, 6}, {cursed, 1, 6},
		{prayer, 1, 11}, {prayer, 0, 9}, {nil, 0, 10},
	} {
		if got := SaveRollAfterEffects(tc.list, 10, tc.side, nil); got != tc.want {
			t.Errorf("save with %+v side %d = %d, want %d", tc.list, tc.side, got, tc.want)
		}
	}
	// 群組 6：`067Eh` 只擋 `DS:6779h` == 0Fh。
	if got := SpellDamageAfterEffects(shield, SpellIDMagicMissile, 6); got != 0 {
		t.Errorf("shield let magic missile through for %d", got)
	}
	if got := SpellDamageAfterEffects(shield, SpellIDMagicMissileAlt, 6); got != 6 {
		t.Errorf("shield stopped spell 41h: %d", got)
	}
}
