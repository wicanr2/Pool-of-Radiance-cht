package combat

import "testing"

func TestDexterityInitiativeModifierOriginalTable(t *testing.T) {
	tests := map[uint8]int8{
		0: -4, 2: -4,
		3: -3, 4: -2, 5: -1,
		6: 0, 15: 0,
		16: 1, 17: 2, 18: 3,
		19: 3, 20: 3,
		21: 4, 23: 4,
		24: 5, 25: 5,
		26: 0, 255: 0,
	}
	for dexterity, want := range tests {
		if got := DexterityInitiativeModifier(dexterity); got != want {
			t.Fatalf("DexterityInitiativeModifier(%d)=%d want %d", dexterity, got, want)
		}
	}
}

func TestResolveInitiativeScoreOriginalOrder(t *testing.T) {
	tests := []struct {
		name      string
		modifier  int8
		die       uint8
		surprised bool
		want      uint8
		wantError bool
	}{
		{name: "plain sum", modifier: 4, die: 3, want: 7},
		{name: "minimum clamps before surprise", modifier: -8, die: 1, surprised: true, want: 0},
		{name: "surprise can retain zero", modifier: 5, die: 1, surprised: true, want: 0},
		{name: "surprise subtracts six", modifier: 8, die: 4, surprised: true, want: 6},
		{name: "score above twenty becomes inactive", modifier: 20, die: 1, want: 0},
		{name: "zero die rejected", modifier: 1, wantError: true},
		{name: "die above six rejected", modifier: 1, die: 7, wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ResolveInitiativeScore(test.modifier, test.die, test.surprised)
			if (err != nil) != test.wantError {
				t.Fatalf("ResolveInitiativeScore error=%v wantError=%v", err, test.wantError)
			}
			if err == nil && got != test.want {
				t.Fatalf("ResolveInitiativeScore=%d want %d", got, test.want)
			}
		})
	}
}

func TestSelectInitiativeActorOriginalMaximumAndTie(t *testing.T) {
	tests := []struct {
		name      string
		scores    []uint8
		ties      []uint8
		want      int
		found     bool
		wantError bool
	}{
		{name: "greatest score wins", scores: []uint8{3, 8, 5}, ties: []uint8{99, 1, 100}, want: 1, found: true},
		{name: "greater tie roll replaces", scores: []uint8{8, 8}, ties: []uint8{20, 21}, want: 1, found: true},
		{name: "equal tie roll replaces", scores: []uint8{8, 8}, ties: []uint8{20, 20}, want: 1, found: true},
		{name: "lower tie roll keeps first", scores: []uint8{8, 8}, ties: []uint8{20, 19}, want: 0, found: true},
		{name: "all zero returns none", scores: []uint8{0, 0}, ties: []uint8{1, 100}},
		{name: "length mismatch rejected", scores: []uint8{1}, wantError: true},
		{name: "invalid score rejected", scores: []uint8{21}, ties: []uint8{1}, wantError: true},
		{name: "invalid tie roll rejected", scores: []uint8{1}, ties: []uint8{0}, wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, found, err := SelectInitiativeActor(test.scores, test.ties)
			if (err != nil) != test.wantError {
				t.Fatalf("SelectInitiativeActor error=%v wantError=%v", err, test.wantError)
			}
			if err == nil && (got != test.want || found != test.found) {
				t.Fatalf("SelectInitiativeActor=(%d,%v) want (%d,%v)", got, found, test.want, test.found)
			}
		})
	}
}

func TestApplyCastingTimeInitiativeOriginalFloor(t *testing.T) {
	tests := []struct {
		score     uint8
		casting   uint8
		want      uint8
		wantError bool
	}{
		{score: 12, casting: 4, want: 8},
		{score: 4, casting: 4, want: 1},
		{score: 3, casting: 4, want: 1},
		{score: 0, casting: 0, want: 1},
		{score: 21, casting: 1, wantError: true},
	}
	for _, test := range tests {
		got, err := ApplyCastingTimeInitiative(test.score, test.casting)
		if (err != nil) != test.wantError {
			t.Fatalf("ApplyCastingTimeInitiative(%d,%d) error=%v wantError=%v", test.score, test.casting, err, test.wantError)
		}
		if err == nil && got != test.want {
			t.Fatalf("ApplyCastingTimeInitiative(%d,%d)=%d want %d", test.score, test.casting, got, test.want)
		}
	}
}

func TestDelayInitiativeOriginalCommand(t *testing.T) {
	if got := DelayInitiative(); got != 1 {
		t.Fatalf("DelayInitiative()=%d want 1", got)
	}
}

func TestInitiativeAfterAttackSlotsOriginalCompletion(t *testing.T) {
	tests := []struct {
		name      string
		score     uint8
		primary   uint8
		secondary uint8
		want      uint8
		wantError bool
	}{
		{name: "primary remains", score: 12, primary: 1, want: 12},
		{name: "secondary remains", score: 12, secondary: 1, want: 12},
		{name: "both remain", score: 12, primary: 1, secondary: 2, want: 12},
		{name: "both exhausted clears attacker", score: 12, want: 0},
		{name: "invalid score rejected", score: 21, wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := InitiativeAfterAttackSlots(test.score, test.primary, test.secondary)
			if (err != nil) != test.wantError {
				t.Fatalf("InitiativeAfterAttackSlots error=%v wantError=%v", err, test.wantError)
			}
			if err == nil && got != test.want {
				t.Fatalf("InitiativeAfterAttackSlots=%d want %d", got, test.want)
			}
		})
	}
}

func TestInitialMovementBudgetBeforeEffectsOriginalClamp(t *testing.T) {
	tests := []struct {
		base       uint8
		applyBonus bool
		bonus      int16
		want       uint8
	}{
		{base: 9, want: 18},
		{base: 9, applyBonus: true, bonus: 2, want: 22},
		{base: 9, applyBonus: false, bonus: 2, want: 18},
		{base: 1, applyBonus: true, bonus: -1, want: 2},
		{base: 96, want: 192},
		{base: 97, want: 2},
		{base: 255, applyBonus: true, bonus: 2, want: 2},
	}
	for _, test := range tests {
		if got := InitialMovementBudgetBeforeEffects(test.base, test.applyBonus, test.bonus); got != test.want {
			t.Fatalf("InitialMovementBudgetBeforeEffects(%d,%v,%d)=%d want %d", test.base, test.applyBonus, test.bonus, got, test.want)
		}
	}
}

func TestSpendMovementStepOriginalCardinalAndDiagonalCost(t *testing.T) {
	tests := []struct {
		budget    uint8
		direction uint8
		want      uint8
		wantError bool
	}{
		{budget: 18, direction: 0, want: 16},
		{budget: 18, direction: 2, want: 16},
		{budget: 18, direction: 1, want: 15},
		{budget: 18, direction: 7, want: 15},
		{budget: 2, direction: 0, want: 0},
		{budget: 2, direction: 1, want: 0},
		{budget: 1, direction: 0, want: 0},
		{budget: 18, direction: 8, wantError: true},
	}
	for _, test := range tests {
		got, err := SpendMovementStep(test.budget, test.direction)
		if (err != nil) != test.wantError {
			t.Fatalf("SpendMovementStep(%d,%d) error=%v wantError=%v", test.budget, test.direction, err, test.wantError)
		}
		if err == nil && got != test.want {
			t.Fatalf("SpendMovementStep(%d,%d)=%d want %d", test.budget, test.direction, got, test.want)
		}
	}
}

func TestAdvanceTacticalCoordinateOriginalDirectionTables(t *testing.T) {
	tests := []struct {
		direction uint8
		wantX     uint8
		wantY     uint8
	}{
		{direction: 0, wantX: 10, wantY: 9},
		{direction: 1, wantX: 11, wantY: 9},
		{direction: 2, wantX: 11, wantY: 10},
		{direction: 3, wantX: 11, wantY: 11},
		{direction: 4, wantX: 10, wantY: 11},
		{direction: 5, wantX: 9, wantY: 11},
		{direction: 6, wantX: 9, wantY: 10},
		{direction: 7, wantX: 9, wantY: 9},
	}
	for _, test := range tests {
		gotX, gotY, err := AdvanceTacticalCoordinate(10, 10, test.direction)
		if err != nil {
			t.Fatalf("AdvanceTacticalCoordinate direction %d: %v", test.direction, err)
		}
		if gotX != test.wantX || gotY != test.wantY {
			t.Fatalf("AdvanceTacticalCoordinate direction %d=(%d,%d) want (%d,%d)", test.direction, gotX, gotY, test.wantX, test.wantY)
		}
	}
	gotX, gotY, err := AdvanceTacticalCoordinate(0, 0, 7)
	if err != nil || gotX != 255 || gotY != 255 {
		t.Fatalf("AdvanceTacticalCoordinate byte wrap=(%d,%d,%v) want (255,255,nil)", gotX, gotY, err)
	}
	if _, _, err := AdvanceTacticalCoordinate(10, 10, 8); err == nil {
		t.Fatal("AdvanceTacticalCoordinate accepted direction 8")
	}
}

func TestApplyMovementEffectIDsOriginalOrderAndByteMath(t *testing.T) {
	tests := []struct {
		name     string
		budget   uint8
		effect27 bool
		effect2A bool
		effect3A bool
		want     uint8
	}{
		{name: "no effects", budget: 18, want: 18},
		{name: "effect 27 doubles", budget: 18, effect27: true, want: 36},
		{name: "effect 27 keeps byte wrap", budget: 130, effect27: true, want: 4},
		{name: "effect 2A halves with truncation", budget: 5, effect2A: true, want: 2},
		{name: "27 runs before 2A", budget: 130, effect27: true, effect2A: true, want: 2},
		{name: "effect 3A runs last and zeros", budget: 18, effect27: true, effect2A: true, effect3A: true, want: 0},
	}
	for _, test := range tests {
		if got := ApplyMovementEffectIDs(test.budget, test.effect27, test.effect2A, test.effect3A); got != test.want {
			t.Fatalf("ApplyMovementEffectIDs(%d,%v,%v,%v)=%d want %d", test.budget, test.effect27, test.effect2A, test.effect3A, got, test.want)
		}
	}
}

func TestResolveMovementProbeOriginalBranchOrder(t *testing.T) {
	tests := []struct {
		name      string
		budget    uint8
		target    uint8
		threshold uint8
		want      MovementProbeAction
	}{
		{name: "empty affordable destination enters", budget: 18, threshold: 2, want: MovementEnter},
		{name: "threshold equal to budget enters", budget: 2, threshold: 2, want: MovementEnter},
		{name: "threshold above budget blocks", budget: 2, threshold: 3, want: MovementBlocked},
		{name: "ff destination blocks normal budget", budget: 254, threshold: 0xFF, want: MovementBlocked},
		{name: "target routes to attack before threshold gate", budget: 0, target: 4, threshold: 0xFF, want: MovementAttack},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := ResolveMovementProbe(test.budget, test.target, test.threshold); got != test.want {
				t.Fatalf("ResolveMovementProbe(%d,%d,%d)=%d want %d", test.budget, test.target, test.threshold, got, test.want)
			}
		})
	}
}

func TestAdvanceAttackPhaseOriginalByteIncrement(t *testing.T) {
	if got := AdvanceAttackPhase(0); got != 1 {
		t.Fatalf("AdvanceAttackPhase(0)=%d want 1", got)
	}
	if got := AdvanceAttackPhase(1); got != 2 {
		t.Fatalf("AdvanceAttackPhase(1)=%d want 2", got)
	}
	if got := AdvanceAttackPhase(255); got != 0 {
		t.Fatalf("AdvanceAttackPhase(255)=%d want byte wrap to 0", got)
	}
}

func TestAttacksThisPhaseOriginalRoundingAndWrap(t *testing.T) {
	tests := []struct {
		rate      uint8
		phase     uint8
		want      uint8
		wantError bool
	}{
		{rate: 0, phase: 0, want: 0},
		{rate: 0, phase: 1, want: 0},
		{rate: 1, phase: 0, want: 0},
		{rate: 1, phase: 1, want: 1},
		{rate: 2, phase: 0, want: 1},
		{rate: 2, phase: 1, want: 1},
		{rate: 3, phase: 0, want: 1},
		{rate: 3, phase: 1, want: 2},
		{rate: 255, phase: 1, want: 0},
		{rate: 2, phase: 2, wantError: true},
	}
	for _, test := range tests {
		got, err := AttacksThisPhase(test.rate, test.phase)
		if (err != nil) != test.wantError {
			t.Fatalf("AttacksThisPhase(%d,%d) error=%v wantError=%v", test.rate, test.phase, err, test.wantError)
		}
		if err == nil && got != test.want {
			t.Fatalf("AttacksThisPhase(%d,%d)=%d want %d", test.rate, test.phase, got, test.want)
		}
	}
}

func TestResolveHitOriginalBoundaries(t *testing.T) {
	tests := []struct {
		name      string
		roll      uint8
		thac0     uint8
		armor     int
		modifier  int
		want      bool
		wantError bool
	}{
		{name: "natural one always misses", roll: 1, thac0: 255, armor: 0},
		{name: "twenty promotes score to one hundred", roll: 20, thac0: 0, armor: 100, want: true},
		{name: "twenty is not an unconditional return", roll: 20, thac0: 0, armor: 255},
		{name: "one below threshold misses", roll: 12, thac0: 41, armor: 54},
		{name: "threshold hits", roll: 13, thac0: 41, armor: 54, want: true},
		{name: "positive modifier hits", roll: 11, thac0: 41, armor: 54, modifier: 2, want: true},
		{name: "negative modifier misses", roll: 13, thac0: 41, armor: 54, modifier: -1},
		{name: "zero roll rejected", roll: 0, thac0: 41, armor: 54, wantError: true},
		{name: "oversized roll rejected", roll: 21, thac0: 41, armor: 54, wantError: true},
		{name: "negative armor encoding rejected", roll: 10, thac0: 41, armor: -1, wantError: true},
		{name: "modifier outside signed byte rejected", roll: 10, thac0: 41, armor: 54, modifier: 128, wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ResolveHit(test.roll, test.thac0, test.armor, test.modifier)
			if (err != nil) != test.wantError {
				t.Fatalf("ResolveHit error=%v wantError=%v", err, test.wantError)
			}
			if err == nil && got != test.want {
				t.Fatalf("ResolveHit=%v want %v", got, test.want)
			}
		})
	}
}

func TestResolveDamageOriginalOrderAndClamp(t *testing.T) {
	tests := []struct {
		name       string
		dice       DamageDice
		rolls      []uint8
		multiplier int
		want       int
		wantError  bool
	}{
		{name: "normal orc 1d8", dice: DamageDice{Count: 1, Sides: 8}, rolls: []uint8{6}, multiplier: 1, want: 6},
		{name: "named orc 2d4 minus one", dice: DamageDice{Count: 2, Sides: 4, Bonus: -1}, rolls: []uint8{2, 4}, multiplier: 1, want: 5},
		{name: "negative result clamps before multiplier", dice: DamageDice{Count: 1, Sides: 2, Bonus: -3}, rolls: []uint8{1}, multiplier: 4, want: 0},
		{name: "explicit multiplier", dice: DamageDice{Count: 1, Sides: 6, Bonus: 1}, rolls: []uint8{4}, multiplier: 3, want: 15},
		{name: "zero count rejected", dice: DamageDice{Sides: 6}, multiplier: 1, wantError: true},
		{name: "wrong roll count rejected", dice: DamageDice{Count: 2, Sides: 6}, rolls: []uint8{3}, multiplier: 1, wantError: true},
		{name: "zero roll rejected", dice: DamageDice{Count: 1, Sides: 6}, rolls: []uint8{0}, multiplier: 1, wantError: true},
		{name: "roll above sides rejected", dice: DamageDice{Count: 1, Sides: 6}, rolls: []uint8{7}, multiplier: 1, wantError: true},
		{name: "zero multiplier rejected", dice: DamageDice{Count: 1, Sides: 6}, rolls: []uint8{3}, wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ResolveDamage(test.dice, test.rolls, test.multiplier)
			if (err != nil) != test.wantError {
				t.Fatalf("ResolveDamage error=%v wantError=%v", err, test.wantError)
			}
			if err == nil && got != test.want {
				t.Fatalf("ResolveDamage=%d want %d", got, test.want)
			}
		})
	}
}
