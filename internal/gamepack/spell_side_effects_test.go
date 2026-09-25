package gamepack

import "testing"

// 群組 10 的祝福／詛咒、群組 18 的急速／緩速與 entry 20 的覆蓋規則（spec 112，#81）。
func TestSideSpellEffectsFollowTheHandlers(t *testing.T) {
	blessed := EffectList{}.Append(NewEffectNode(BlessEffectCode, 6, 1, false))
	cursed := EffectList{}.Append(NewEffectNode(CurseEffectCode, 6, 1, false))
	if got := HitRollEffectModifier(blessed, nil); got != 1 {
		t.Errorf("祝福 %d，應該 +1", got)
	}
	if got := HitRollEffectModifier(cursed, nil); got != -1 {
		t.Errorf("詛咒 %d，應該 −1", got)
	}
	// 群組 10 問的是攻擊者；祝福掛在目標身上不影響對它出手的人。
	if got := HitRollEffectModifier(nil, blessed); got != 0 {
		t.Errorf("目標身上的祝福改了命中 %d", got)
	}
	// 同一個碼兩個節點：每個群組只問一次最早的那一個。
	if got := HitRollEffectModifier(blessed.Append(NewEffectNode(BlessEffectCode, 6, 1, false)), nil); got != 1 {
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
