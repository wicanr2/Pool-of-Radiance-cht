package gamepack

import "testing"

// 群組 12 的種族碼：`5Ah` 只在類別 0，`61h` 在類別 2 與 4；加的值看體質（spec 112〈群組 12／6／4／5〉）。
func TestRaceSaveCodesFollowCategoryAndConstitution(t *testing.T) {
	dwarf := EffectList{}.Append(NewEffectNode(RacePoisonSaveEffectCode, 0, EffectUndispellable, false)).
		Append(NewEffectNode(RaceMagicSaveEffectCode, 0, EffectUndispellable, false))
	gnome := EffectList{}.Append(NewEffectNode(RaceMagicSaveEffectCode, 0, EffectUndispellable, false))
	for _, tc := range []struct {
		list         EffectList
		category     uint8
		constitution uint8
		want         uint8
	}{
		{dwarf, 0, 17, 14}, {dwarf, 1, 17, 10}, {dwarf, 2, 17, 14}, {dwarf, 3, 17, 10}, {dwarf, 4, 17, 14},
		{gnome, 0, 17, 10}, {gnome, 4, 17, 14},
		{dwarf, 4, 4, 11}, {dwarf, 4, 10, 12}, {dwarf, 4, 13, 13}, {dwarf, 4, 18, 15}, {dwarf, 4, 20, 15},
		{dwarf, 4, 3, 10}, {dwarf, 4, 0, 10}, {nil, 4, 17, 10},
	} {
		got := SaveRollEffects{Effects: tc.list, Category: tc.category, Constitution: tc.constitution}.Apply(10)
		if got != tc.want {
			t.Errorf("%v category %d CON %d: %d, want %d", tc.list, tc.category, tc.constitution, got, tc.want)
		}
	}
}

// `08h`／`2Dh` 比 `5CF0h` 的陣營 2／5／8，`09h`／`2Eh` 比 0／3／6；不知道陣營就不加。
// `2Dh` 只經作用範圍那一條拿得到時也算。
func TestProtectionCodesReadTheActorAlignment(t *testing.T) {
	evil := EffectList{}.Append(NewEffectNode(ProtectionFromEvilEffectCode, 9, 3, false))
	good := EffectList{}.Append(NewEffectNode(ProtectionFromGoodEffectCode, 9, 3, false))
	area := func(code uint8) (EffectNode, bool) {
		if code == ProtectionFromEvilAreaEffectCode {
			return NewEffectNode(code, 6, 3, false), true
		}
		return EffectNode{}, false
	}
	for alignment := uint8(0); alignment <= 8; alignment++ {
		wantEvil, wantGood := uint8(10), uint8(10)
		switch alignment {
		case 2, 5, 8:
			wantEvil = 12
		case 0, 3, 6:
			wantGood = 12
		}
		save := SaveRollEffects{ActorAlignment: alignment, ActorAlignmentKnown: true}
		save.Effects = evil
		if got := save.Apply(10); got != wantEvil {
			t.Errorf("PfE vs alignment %d: %d, want %d", alignment, got, wantEvil)
		}
		save.Effects = good
		if got := save.Apply(10); got != wantGood {
			t.Errorf("PfG vs alignment %d: %d, want %d", alignment, got, wantGood)
		}
		save.Effects, save.AreaNode = nil, area
		if got := save.Apply(10); got != wantEvil {
			t.Errorf("PfE 10' vs alignment %d: %d, want %d", alignment, got, wantEvil)
		}
	}
	if got := (SaveRollEffects{Effects: good}).Apply(10); got != 10 {
		t.Errorf("unknown alignment still added: %d", got)
	}
}

// 抗寒／抗火：豁免 +3 與傷害減半都要 `DS:6777h` 的那一位。
func TestResistanceReadsTheDamageFlags(t *testing.T) {
	cold := EffectList{}.Append(NewEffectNode(ResistColdEffectCode, 30, 3, false))
	fire := EffectList{}.Append(NewEffectNode(ResistFireEffectCode, 30, 3, false))
	fireball := SpellDamageKind(SpellIDFireball)
	if fireball != 9 || SpellDamageKind(SpellIDBurningHands) != 9 || SpellDamageKind(SpellIDShockingGrasp) != 0x0c ||
		SpellDamageKind(SpellIDMagicMissile) != 8 || SpellDamageKind(SpellIDSleep) != 0 {
		t.Fatal("damage kinds do not match the pushes")
	}
	if got := (SaveRollEffects{Effects: fire, DamageFlags: fireball}).Apply(10); got != 13 {
		t.Errorf("resist fire vs fireball save %d, want 13", got)
	}
	if got := (SaveRollEffects{Effects: cold, DamageFlags: fireball}).Apply(10); got != 10 {
		t.Errorf("resist cold vs fireball save %d, want 10", got)
	}
	if got := (SpellDamageEffects{Effects: fire, Spell: SpellIDFireball, DamageFlags: fireball}).Apply(21).Damage; got != 10 {
		t.Errorf("resist fire vs 21 damage: %d, want 10", got)
	}
	if got := (SpellDamageEffects{Effects: cold, Spell: 1, DamageFlags: DamageFlagCold}).Apply(9).Damage; got != 4 {
		t.Errorf("resist cold vs a cold 9: %d, want 4", got)
	}
}

// 鏡影：一定先擲；擲 1、不是法術、範圍法術都不擋；擋下就少一個影像，用完摘掉。
func TestMirrorImageAbsorbsOnlySingleTargetSpells(t *testing.T) {
	two := EffectList{}.Append(NewEffectNode(MirrorImageEffectCode, 6, 2, false))
	rolled := 0
	roll := func(value int) func(count, sides int) int {
		return func(count, sides int) int {
			rolled++
			if count != 1 || sides != 3 {
				t.Fatalf("mirror image rolled %dd%d, want 1d3", count, sides)
			}
			return value
		}
	}
	for _, tc := range []struct {
		value int
		spell uint8
		area  bool
		want  bool
	}{{1, 15, false, false}, {2, 0, false, false}, {3, 47, true, false}, {2, 15, false, true}} {
		list, lost := MirrorImageAbsorbs(two, tc.spell, tc.area, roll(tc.value))
		if lost != tc.want {
			t.Errorf("%+v: absorbed %v", tc, lost)
		}
		if lost && list[0].Magnitude() != 1 {
			t.Errorf("images left %d, want 1", list[0].Magnitude())
		}
	}
	if rolled != 4 {
		t.Errorf("rolled %d times, want every call", rolled)
	}
	one := EffectList{}.Append(NewEffectNode(MirrorImageEffectCode, 6, 1, false))
	if list, lost := MirrorImageAbsorbs(one, 15, false, func(int, int) int { return 2 }); !lost || len(list) != 0 {
		t.Errorf("the last image did not go: %v %v", list, lost)
	}
}

func TestEnfeeblementTakesAQuarter(t *testing.T) {
	weak := EffectList{}.Append(NewEffectNode(EnfeeblementEffectCode, 3, 3, false))
	for damage, want := range map[int]int{0: 0, 3: 3, 4: 3, 12: 9, 15: 12} {
		if got := MeleeDamageAfterAttackerEffects(weak, damage); got != want {
			t.Errorf("enfeebled %d → %d, want %d", damage, got, want)
		}
	}
	if got := MeleeDamageAfterAttackerEffects(nil, 12); got != 12 {
		t.Errorf("plain 12 → %d", got)
	}
}

// `22h` → `2Bh`＋`2Ch`；各自重掛、各自扣；到底就掛 `1Fh`（只掛一個）。
func TestDiseaseTeardownRenewsItself(t *testing.T) {
	node := NewEffectNode(DiseaseEffectCode, 0, 3, true)
	got := DiseaseTeardownOf(node, nil, 12, 20)
	if len(got.Effects) != 2 || got.Effects[0].Code != DiseaseWeakeningEffectCode || got.Effects[0].Duration() != 0x3c ||
		got.Effects[1].Code != DiseaseWastingEffectCode || got.Effects[1].Duration() != 0x0a ||
		got.Effects[0].Magnitude() != 3 || !got.Effects[1].NeedsTeardown() {
		t.Fatalf("22h handed over %+v", got.Effects)
	}
	if got.Strength != 11 || got.HitPoints != 19 || !got.Weakened || !got.Wasted {
		t.Fatalf("22h: %+v", got)
	}
	low := DiseaseTeardownOf(node, nil, 3, 1)
	if low.Strength != 3 || low.HitPoints != 1 || low.Weakened || low.Wasted ||
		len(low.Effects) != 3 || low.Effects[1].Code != HelplessEffectCode || !low.Effects[1].Undispellable() {
		t.Fatalf("floored: %+v", low)
	}
	if other := DiseaseTeardownOf(NewEffectNode(0x21, 0, 3, true), nil, 12, 20); len(other.Effects) != 0 ||
		other.Strength != 12 || other.HitPoints != 20 {
		t.Fatalf("21h went through the disease path: %+v", other)
	}
}
