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
		{"fighter", "Fighter", []string{"fighter"}}, {"thief", "Thief", []string{"thief"}},
		{"fighter-thief", "Fighter/Thief", []string{"fighter", "thief"}},
	},
	"elf": {
		{"fighter", "Fighter", []string{"fighter"}}, {"magic-user", "Magic-User", []string{"magic-user"}},
		{"thief", "Thief", []string{"thief"}}, {"fighter-magic-user", "Fighter/Magic-User", []string{"fighter", "magic-user"}},
		{"fighter-thief", "Fighter/Thief", []string{"fighter", "thief"}},
		{"fighter-magic-user-thief", "Fighter/Magic-User/Thief", []string{"fighter", "magic-user", "thief"}},
		{"magic-user-thief", "Magic-User/Thief", []string{"magic-user", "thief"}},
	},
	"gnome": {
		{"fighter", "Fighter", []string{"fighter"}}, {"thief", "Thief", []string{"thief"}},
		{"fighter-thief", "Fighter/Thief", []string{"fighter", "thief"}},
	},
	"half-elf": {
		{"cleric", "Cleric", []string{"cleric"}}, {"fighter", "Fighter", []string{"fighter"}},
		{"magic-user", "Magic-User", []string{"magic-user"}}, {"thief", "Thief", []string{"thief"}},
		{"cleric-fighter", "Cleric/Fighter", []string{"cleric", "fighter"}},
		{"cleric-fighter-magic-user", "Cleric/Fighter/Magic-User", []string{"cleric", "fighter", "magic-user"}},
		{"cleric-magic-user", "Cleric/Magic-User", []string{"cleric", "magic-user"}},
		{"fighter-magic-user", "Fighter/Magic-User", []string{"fighter", "magic-user"}},
		{"fighter-thief", "Fighter/Thief", []string{"fighter", "thief"}},
		{"fighter-magic-user-thief", "Fighter/Magic-User/Thief", []string{"fighter", "magic-user", "thief"}},
		{"magic-user-thief", "Magic-User/Thief", []string{"magic-user", "thief"}},
	},
	"halfling": {
		{"fighter", "Fighter", []string{"fighter"}}, {"thief", "Thief", []string{"thief"}},
		{"fighter-thief", "Fighter/Thief", []string{"fighter", "thief"}},
	},
	"human": {
		{"cleric", "Cleric", []string{"cleric"}}, {"fighter", "Fighter", []string{"fighter"}},
		{"magic-user", "Magic-User", []string{"magic-user"}}, {"thief", "Thief", []string{"thief"}},
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
		return "種族會限制可選職業；此清單完全依原版，不套用新版 D&D 規則。"
	case "class":
		return "多職業會分配經驗值；READY／ACTION 戰鬥圖示稍後仍可自訂。"
	case "alignment":
		return "陣營會影響部分職業與劇情互動；請依角色概念選擇。"
	case "portrait":
		return "HEAD 與 BODY 可分別循環；KEEP 接受目前肖像。"
	case "icon":
		return "可同時預覽 READY／ACTION，並調整 Parts、兩組顏色與 Size。"
	default:
		return ""
	}
}
