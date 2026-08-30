package creation

// Roller isolates the original game's dice helper so character generation can
// be replayed deterministically in tests and oracle comparisons.
type Roller interface {
	Roll(count, sides int) int
}

type RolledCharacter struct {
	Age                 int
	Abilities           [6]int
	ExceptionalStrength int
	Gold                int
	HP                  int
	RawHP               int
}

type ageRoll struct{ base, count, sides int }

var ages = [8][7]ageRoll{
	1: {{250, 2, 20}, {}, {40, 5, 4}, {}, {}, {}, {75, 3, 6}},
	2: {{650, 10, 10}, {}, {130, 5, 6}, {}, {}, {150, 5, 6}, {100, 5, 6}},
	3: {{300, 3, 12}, {}, {60, 5, 4}, {}, {}, {100, 2, 12}, {80, 5, 4}},
	4: {{40, 2, 4}, {}, {22, 3, 4}, {}, {}, {30, 2, 8}, {22, 3, 8}},
	5: {{}, {}, {20, 3, 4}, {}, {}, {}, {40, 2, 4}},
	6: {{20, 1, 4}, {}, {13, 1, 4}, {}, {}, {}, {20, 2, 4}},
	7: {{18, 1, 4}, {18, 1, 4}, {15, 1, 4}, {17, 1, 4}, {20, 1, 4}, {24, 2, 4}, {18, 1, 4}},
}

var ageThresholds = [8][5]int{
	1: {50, 150, 250, 350, 450}, 2: {175, 550, 875, 1200, 1600},
	3: {90, 300, 450, 600, 750}, 4: {40, 100, 175, 250, 325},
	5: {33, 68, 101, 144, 199}, 6: {15, 30, 45, 60, 80},
	7: {20, 40, 60, 90, 120},
}

// Each row is STR male/female minimum, STR male/female maximum,
// exceptional-STR male/female maximum, then INT/WIS/DEX/CON/CHA min/max.
var raceLimits = [8][16]int{
	1: {8, 8, 18, 17, 99, 0, 3, 18, 3, 18, 3, 17, 12, 19, 3, 16},
	2: {3, 3, 18, 16, 75, 0, 8, 18, 3, 18, 7, 19, 6, 18, 8, 18},
	3: {6, 6, 18, 15, 50, 0, 7, 18, 3, 18, 3, 18, 8, 18, 3, 18},
	4: {3, 3, 18, 17, 90, 0, 4, 18, 3, 18, 6, 18, 6, 18, 3, 18},
	5: {6, 6, 17, 14, 0, 0, 6, 18, 3, 17, 8, 18, 10, 19, 3, 18},
	6: {6, 6, 18, 18, 99, 75, 3, 17, 3, 14, 3, 17, 13, 19, 3, 12},
	7: {3, 3, 18, 18, 100, 50, 3, 18, 3, 18, 3, 18, 3, 18, 3, 18},
}

var classMinimums = [17][6]int{
	0: {6, 6, 9}, 1: {0, 0, 12, 0, 0, 15}, 2: {9, 0, 6, 6, 7},
	3: {13, 9, 12, 0, 9, 17}, 4: {0, 13, 14, 0, 14}, 5: {0, 9, 6, 6},
	6: {6, 6, 0, 9}, 7: {15, 0, 15, 15, 11}, 8: {9, 0, 9},
	9: {9, 9, 9}, 10: {0, 13, 14, 0, 14}, 11: {0, 9, 9},
	12: {0, 0, 9, 9}, 13: {9, 9}, 14: {9, 0, 0, 9},
	15: {9, 9, 0, 9}, 16: {0, 9, 0, 9},
}

var goldDice = [8][2]int{{3, 6}, {3, 6}, {5, 4}, {5, 4}, {5, 4}, {2, 4}, {2, 6}, {5, 4}}
var hpDice = [8][2]int{{1, 8}, {1, 8}, {1, 10}, {1, 10}, {2, 8}, {1, 4}, {1, 6}, {2, 4}}

func RollCharacter(roller Roller, race Race, gender Gender, class ClassChoice) RolledCharacter {
	result := RolledCharacter{}
	result.Age = rollAge(roller, int(race.DOSCode), int(class.DOSCode))
	for ability := range result.Abilities {
		result.Abilities[ability] = roller.Roll(3, 6)
	}
	applyRacialAdjustments(&result.Abilities, int(race.DOSCode))
	applyAgeAdjustments(&result.Abilities, result.Age, int(race.DOSCode))
	applyLimits(&result, roller, int(race.DOSCode), int(gender.DOSCode), int(class.DOSCode), class.Components)
	result.Gold, result.RawHP, result.HP = rollGoldAndHP(roller, class.Components, result.Abilities[4], int(class.DOSCode))
	return result
}

func rollAge(roller Roller, race, class int) int {
	index := class
	multiclass := class > 7
	switch class {
	case 8, 9, 11, 12:
		index = 0
	case 13, 15, 16:
		index = 5
	case 14:
		index = 6
	}
	spec := ages[race][index]
	if multiclass {
		return spec.base + spec.count*spec.sides
	}
	return spec.base + roller.Roll(spec.count, spec.sides)
}

func applyRacialAdjustments(a *[6]int, race int) {
	switch race {
	case 1:
		a[4]++
		a[5]--
	case 2:
		a[3]++
		a[4]--
	case 5:
		a[0]--
		a[3]++
	case 6:
		a[0]++
		a[4]++
		a[5] -= 2
	}
}

func applyAgeAdjustments(a *[6]int, age, race int) {
	t := ageThresholds[race]
	if age > t[0] {
		a[0]++
	}
	if age > t[1] {
		a[0]--
	}
	if age > t[2] {
		a[0] -= 2
		a[3] -= 2
		a[4]--
	}
	if age > t[3] {
		a[0]--
		a[1]++
		a[2]++
		a[3]--
		a[4]--
	}
	if age <= t[0] {
		a[2]--
	}
	if age > t[1] {
		a[2]++
	}
	if age > t[2] {
		a[2]++
	}
	if age > t[3] {
		a[2]++
	}
}

func applyLimits(result *RolledCharacter, roller Roller, race, gender, class int, components []string) {
	limits := raceLimits[race]
	for i := range result.Abilities {
		min, max := 0, 0
		if i == 0 {
			min, max = limits[gender], limits[2+gender]
		} else {
			min, max = limits[4+i*2], limits[5+i*2]
		}
		if result.Abilities[i] < min {
			result.Abilities[i] = min
		}
		if result.Abilities[i] > max {
			result.Abilities[i] = max
		}
		if result.Abilities[i] < classMinimums[class][i] {
			result.Abilities[i] = classMinimums[class][i]
		}
	}
	// Overlay-16 tests class codes 8..12 through a Pascal set literal and
	// raises WIS to 13. In Pool's visible catalog these are cleric multiclasses.
	if class >= 8 && class <= 12 && result.Abilities[2] < 13 {
		result.Abilities[2] = 13
	}
	if result.Abilities[0] == 18 && hasComponent(components, "fighter") {
		result.ExceptionalStrength = roller.Roll(1, 100)
		if cap := limits[4+gender]; result.ExceptionalStrength > cap {
			result.ExceptionalStrength = cap
		}
	}
}

func rollGoldAndHP(roller Roller, components []string, constitution, classCode int) (gold, rawHP, hp int) {
	indices := componentIndices(components)
	for _, index := range indices {
		gold += roller.Roll(goldDice[index][0], goldDice[index][1])
		value := roller.Roll(hpDice[index][0], hpDice[index][1])
		minimum := 2 * hpDice[index][1] / 3
		if value < minimum {
			value = minimum
		}
		rawHP += value
	}
	gold = gold * 10 / len(indices)
	modifier := 0
	for range indices {
		modifier += constitutionModifier(constitution, classCode)
	}
	hp = (rawHP + modifier) / len(indices)
	if hp < 1 {
		hp = 1
	}
	return gold, rawHP / len(indices), hp
}

func constitutionModifier(score, classCode int) int {
	base := 0
	switch {
	case score >= 15:
		base = 1
	case score >= 7:
		base = 0
	case score >= 4:
		base = -1
	default:
		base = -2
	}
	if score >= 16 {
		base = 2
	}
	if classCode == 2 && score > 16 {
		base++
	}
	if classCode == 2 && score > 17 {
		base++
	}
	return base
}

func componentIndices(components []string) []int {
	result := make([]int, 0, len(components))
	for _, component := range components {
		switch component {
		case "cleric":
			result = append(result, 0)
		case "fighter":
			result = append(result, 2)
		case "magic-user":
			result = append(result, 5)
		case "thief":
			result = append(result, 6)
		}
	}
	return result
}

func hasComponent(components []string, want string) bool {
	for _, component := range components {
		if component == want {
			return true
		}
	}
	return false
}
