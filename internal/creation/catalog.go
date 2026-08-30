package creation

type Race struct {
	ID, Label string
	DOSCode   uint8
}
type Gender struct {
	ID, Label string
	DOSCode   uint8
}
type ClassChoice struct {
	ID, Label  string
	DOSCode    uint8
	Components []string
}
type Alignment struct{ ID, Label string }

var Races = []Race{
	{"dwarf", "Dwarf", 1}, {"elf", "Elf", 2}, {"gnome", "Gnome", 3},
	{"half-elf", "Half-Elf", 4}, {"halfling", "Halfling", 5}, {"human", "Human", 7},
}

var Genders = []Gender{{"male", "Male", 0}, {"female", "Female", 1}}

var Alignments = []Alignment{
	{"lawful-good", "Lawful Good"}, {"lawful-neutral", "Lawful Neutral"}, {"lawful-evil", "Lawful Evil"},
	{"neutral-good", "Neutral Good"}, {"true-neutral", "True Neutral"}, {"neutral-evil", "Neutral Evil"},
	{"chaotic-good", "Chaotic Good"}, {"chaotic-neutral", "Chaotic Neutral"}, {"chaotic-evil", "Chaotic Evil"},
}

var classesByRace = map[string][]ClassChoice{
	"dwarf": {
		{"fighter", "Fighter", 2, []string{"fighter"}}, {"thief", "Thief", 6, []string{"thief"}},
		{"fighter-thief", "Fighter/Thief", 14, []string{"fighter", "thief"}},
	},
	"elf": {
		{"fighter", "Fighter", 2, []string{"fighter"}}, {"magic-user", "Magic-User", 5, []string{"magic-user"}},
		{"thief", "Thief", 6, []string{"thief"}}, {"fighter-magic-user", "Fighter/Magic-User", 13, []string{"fighter", "magic-user"}},
		{"fighter-thief", "Fighter/Thief", 14, []string{"fighter", "thief"}},
		{"fighter-magic-user-thief", "Fighter/Magic-User/Thief", 15, []string{"fighter", "magic-user", "thief"}},
		{"magic-user-thief", "Magic-User/Thief", 16, []string{"magic-user", "thief"}},
	},
	"gnome": {
		{"fighter", "Fighter", 2, []string{"fighter"}}, {"thief", "Thief", 6, []string{"thief"}},
		{"fighter-thief", "Fighter/Thief", 14, []string{"fighter", "thief"}},
	},
	"half-elf": {
		{"cleric", "Cleric", 0, []string{"cleric"}}, {"fighter", "Fighter", 2, []string{"fighter"}},
		{"magic-user", "Magic-User", 5, []string{"magic-user"}}, {"thief", "Thief", 6, []string{"thief"}},
		{"cleric-fighter", "Cleric/Fighter", 8, []string{"cleric", "fighter"}},
		{"cleric-fighter-magic-user", "Cleric/Fighter/Magic-User", 9, []string{"cleric", "fighter", "magic-user"}},
		{"cleric-magic-user", "Cleric/Magic-User", 11, []string{"cleric", "magic-user"}},
		{"fighter-magic-user", "Fighter/Magic-User", 13, []string{"fighter", "magic-user"}},
		{"fighter-thief", "Fighter/Thief", 14, []string{"fighter", "thief"}},
		{"fighter-magic-user-thief", "Fighter/Magic-User/Thief", 15, []string{"fighter", "magic-user", "thief"}},
		{"magic-user-thief", "Magic-User/Thief", 16, []string{"magic-user", "thief"}},
	},
	"halfling": {
		{"fighter", "Fighter", 2, []string{"fighter"}}, {"thief", "Thief", 6, []string{"thief"}},
		{"fighter-thief", "Fighter/Thief", 14, []string{"fighter", "thief"}},
	},
	"human": {
		{"cleric", "Cleric", 0, []string{"cleric"}}, {"fighter", "Fighter", 2, []string{"fighter"}},
		{"magic-user", "Magic-User", 5, []string{"magic-user"}}, {"thief", "Thief", 6, []string{"thief"}},
	},
}

func ClassesForRace(raceID string) []ClassChoice {
	source := classesByRace[raceID]
	result := make([]ClassChoice, len(source))
	for index, value := range source {
		result[index] = value
		result[index].Components = append([]string(nil), value.Components...)
	}
	return result
}

func HintFor(stage string) string {
	switch stage {
	case "race":
		return "RACE LIMITS CLASS OPTIONS; THIS LIST FOLLOWS THE DOS ORIGINAL."
	case "class":
		return "MULTI-CLASS CHARACTERS SPLIT EXPERIENCE BETWEEN ACTIVE CLASSES."
	case "alignment":
		return "ALIGNMENT AFFECTS SOME CLASS AND STORY INTERACTIONS."
	case "portrait":
		return "HEAD AND BODY CYCLE SEPARATELY; KEEP ACCEPTS THE PORTRAIT."
	case "icon":
		return "PREVIEW READY/ACTION WHILE EDITING PARTS, TWO COLORS, AND SIZE."
	default:
		return ""
	}
}
