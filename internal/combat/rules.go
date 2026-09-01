package combat

import "fmt"

// DexterityInitiativeModifier reproduces overlay-25 entry 11. Values outside
// the original explicit table ranges use the original zero default.
func DexterityInitiativeModifier(dexterity uint8) int8 {
	switch {
	case dexterity <= 2:
		return -4
	case dexterity <= 5:
		return int8(dexterity) - 6
	case dexterity <= 15:
		return 0
	case dexterity <= 18:
		return int8(dexterity) - 15
	case dexterity <= 20:
		return 3
	case dexterity <= 23:
		return 4
	case dexterity <= 25:
		return 5
	default:
		return 0
	}
}

// ResolveInitiativeScore reproduces overlay-13:0084h..0107h after the
// dexterity initiative modifier has been resolved. The surprise penalty is
// applied after the original minimum-one clamp.
func ResolveInitiativeScore(modifier int8, die uint8, surprised bool) (uint8, error) {
	if die < 1 || die > 6 {
		return 0, fmt.Errorf("Pool initiative die %d is outside 1..6", die)
	}
	score := int(modifier) + int(die)
	if score < 1 {
		score = 1
	}
	if surprised {
		score -= 6
	}
	if score < 0 || score > 20 {
		return 0, nil
	}
	return uint8(score), nil
}

// ApplyCastingTimeInitiative reproduces overlay-13 entry 19's update after a
// spell is chosen. A casting cost at least as large as the current score leaves
// one point instead of reaching zero or wrapping.
func ApplyCastingTimeInitiative(score, castingCost uint8) (uint8, error) {
	if score > 20 {
		return 0, fmt.Errorf("Pool initiative score %d is outside 0..20", score)
	}
	if score > castingCost {
		return score - castingCost, nil
	}
	return 1, nil
}

// DelayInitiative reproduces overlay-08 entry 9's D command.
func DelayInitiative() uint8 {
	return 1
}

// InitiativeAfterAttackSlots reproduces overlay-13:17A0h..17EFh for the
// initiative field. The attacker keeps its score while either attack slot has
// work left; exhausting both slots invokes the original runtime clear service.
func InitiativeAfterAttackSlots(score, primaryRemaining, secondaryRemaining uint8) (uint8, error) {
	if score > 20 {
		return 0, fmt.Errorf("Pool initiative score %d is outside 0..20", score)
	}
	if primaryRemaining > 0 || secondaryRemaining > 0 {
		return score, nil
	}
	return 0, nil
}

// InitialMovementBudgetBeforeEffects reproduces overlay-13:0123h..018Eh up to
// the effect-code 12 hook. The original stores the optional signed addition
// back through AL before applying its 1..96 clamp, then doubles the value.
func InitialMovementBudgetBeforeEffects(baseMovement uint8, applyGlobalBonus bool, globalBonus int16) uint8 {
	adjusted := baseMovement
	if applyGlobalBonus {
		adjusted = uint8(int(baseMovement) + int(globalBonus))
	}
	if adjusted < 1 || adjusted > 96 {
		adjusted = 1
	}
	return adjusted * 2
}

// SpendMovementStep reproduces overlay-13 entry 5's budget update. Pool uses
// even direction codes for cardinal movement and odd codes for diagonals.
func SpendMovementStep(budget, direction uint8) (uint8, error) {
	if direction > 7 {
		return 0, fmt.Errorf("Pool movement direction %d is outside 0..7", direction)
	}
	cost := uint8(2)
	if direction&1 != 0 {
		cost = 3
	}
	if cost > budget {
		return 0, nil
	}
	return budget - cost, nil
}

// AdvanceTacticalCoordinate reproduces the signed byte deltas read by
// overlay-13 entry 5 from DS:274Ah and DS:2753h. Destination validation is a
// caller responsibility; the original stores each sum back through AL.
func AdvanceTacticalCoordinate(x, y, direction uint8) (uint8, uint8, error) {
	if direction > 7 {
		return 0, 0, fmt.Errorf("Pool movement direction %d is outside 0..7", direction)
	}
	deltaX := [...]int8{0, 1, 1, 1, 0, -1, -1, -1}
	deltaY := [...]int8{-1, -1, 0, 1, 1, 1, 0, -1}
	return uint8(int(x) + int(deltaX[direction])), uint8(int(y) + int(deltaY[direction])), nil
}

// ApplyMovementEffectIDs reproduces effect-code 12's ordered accumulator
// updates. The identifiers remain numeric until their player-visible spell or
// item producers are independently closed.
func ApplyMovementEffectIDs(budget uint8, effect27, effect2A, effect3A bool) uint8 {
	if effect27 {
		budget *= 2
	}
	if effect2A {
		budget /= 2
	}
	if effect3A {
		budget = 0
	}
	return budget
}

type MovementProbeAction uint8

const (
	MovementBlocked MovementProbeAction = iota
	MovementEnter
	MovementAttack
)

// ResolveMovementProbe reproduces overlay-08:0BC9h..0C73h after overlay-13
// entry 6 has inspected the destination. A non-zero target takes precedence
// and is routed to the attack wrapper. Otherwise the first byte of the
// destination-class record at DS:2758h must not exceed the remaining budget.
// Spending the cardinal/diagonal step remains a separate operation.
func ResolveMovementProbe(budget, attackTargetID, entryThreshold uint8) MovementProbeAction {
	if attackTargetID != 0 {
		return MovementAttack
	}
	if entryThreshold > budget {
		return MovementBlocked
	}
	return MovementEnter
}

// SelectInitiativeActor reproduces overlay-08:0124h. It chooses the greatest
// non-zero score. Equal scores use one caller-supplied 1d100 result per actor;
// a later actor replaces the current actor when its roll is greater or equal.
func SelectInitiativeActor(scores, tieRolls []uint8) (int, bool, error) {
	if len(scores) != len(tieRolls) {
		return 0, false, fmt.Errorf("Pool initiative has %d scores and %d tie rolls", len(scores), len(tieRolls))
	}
	selected := -1
	var bestScore, bestTie uint8
	for index, score := range scores {
		if score > 20 {
			return 0, false, fmt.Errorf("Pool initiative score %d at index %d is outside 0..20", score, index)
		}
		roll := tieRolls[index]
		if roll < 1 || roll > 100 {
			return 0, false, fmt.Errorf("Pool initiative tie roll %d at index %d is outside 1..100", roll, index)
		}
		if score > bestScore || score == bestScore && roll >= bestTie {
			bestScore = score
			bestTie = roll
			selected = index
		}
	}
	if bestScore == 0 {
		return 0, false, nil
	}
	return selected, true, nil
}

// AdvanceAttackPhase reproduces the byte increment at overlay-08:0879h.
// Callers own the combat-round boundary and must invoke this exactly once.
func AdvanceAttackPhase(counter uint8) uint8 {
	return counter + 1
}

// AttacksThisPhase reproduces overlay-13:0E58h. The increment is intentionally
// byte-sized, including the original wrap before division.
func AttacksThisPhase(encodedRate, phaseBit uint8) (uint8, error) {
	if phaseBit > 1 {
		return 0, fmt.Errorf("Pool attack phase bit %d is outside 0..1", phaseBit)
	}
	if phaseBit == 1 {
		encodedRate++
	}
	return encodedRate / 2, nil
}

// ResolveHit evaluates the Pool attack comparison using the record's internal
// THAC0/AC encodings. Callers must resolve situational modifiers separately.
func ResolveHit(roll, attackerTHAC0Internal uint8, effectiveACInternal, modifier int) (bool, error) {
	if roll < 1 || roll > 20 {
		return false, fmt.Errorf("Pool attack roll %d is outside 1..20", roll)
	}
	if effectiveACInternal < 0 || effectiveACInternal > 255 {
		return false, fmt.Errorf("Pool effective AC encoding %d is outside one byte", effectiveACInternal)
	}
	if modifier < -128 || modifier > 127 {
		return false, fmt.Errorf("Pool attack modifier %d is outside signed byte range", modifier)
	}
	if roll == 1 {
		return false, nil
	}
	score := int(roll)
	if roll == 20 {
		score = 100
	}
	return score+int(attackerTHAC0Internal)+modifier >= effectiveACInternal, nil
}

type DamageDice struct {
	Count uint8
	Sides uint8
	Bonus int8
}

// ResolveDamage consumes already-generated one-based die results in their
// original order. A caller-owned proven multiplier must be explicit.
func ResolveDamage(dice DamageDice, rolls []uint8, multiplier int) (int, error) {
	if dice.Count == 0 || dice.Sides == 0 {
		return 0, fmt.Errorf("Pool damage dice must have non-zero count and sides")
	}
	if len(rolls) != int(dice.Count) {
		return 0, fmt.Errorf("Pool damage has %d rolls, want %d", len(rolls), dice.Count)
	}
	if multiplier < 1 {
		return 0, fmt.Errorf("Pool damage multiplier %d is below 1", multiplier)
	}
	total := int(dice.Bonus)
	for index, roll := range rolls {
		if roll < 1 || roll > dice.Sides {
			return 0, fmt.Errorf("Pool damage roll %d at index %d is outside 1..%d", roll, index, dice.Sides)
		}
		total += int(roll)
	}
	if total < 0 {
		total = 0
	}
	return total * multiplier, nil
}
