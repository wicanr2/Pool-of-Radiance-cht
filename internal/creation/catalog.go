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

// ComponentClassIndex 回傳單職業的 DOS 職業碼。那個碼同時是原版角色記錄裡
// 每職業等級陣列（record +96h 起）與 DS:3C16h THAC0 表的列索引（spec 063），
// 所以多職業要逐個 component 查這裡，不能用組合職業自己的碼。
//
// 值取自本檔既有的單職業條目，不另外維護一份對照表。
func ComponentClassIndex(component string) (uint8, bool) {
	for _, choices := range classesByRace {
		for _, choice := range choices {
			if len(choice.Components) == 1 && choice.Components[0] == component {
				return choice.DOSCode, true
			}
		}
	}
	return 0, false
}

// ClassComponents 回傳某個職業 ID 的組成職業。查不到就回 false，
// 讓呼叫端失敗即關閉，而不是當成單職業硬解。
func ClassComponents(classID string) ([]string, bool) {
	for _, choices := range classesByRace {
		for _, choice := range choices {
			if choice.ID == classID {
				return append([]string(nil), choice.Components...), true
			}
		}
	}
	return nil, false
}

// ClassDOSCode 回傳組合職業自己的 DOS 碼，也就是原版角色記錄 `+2Fh` 那個
// 位元組。七名預設人物對得上：三名戰士是 2、牧師 0、兩名法師 5、賊 6。
//
// 與 ComponentClassIndex 不同：那一支回的是每個組成職業的列索引，
// 這一支回的是整個職業組合的碼（雙職業 8..16）。
func ClassDOSCode(classID string) (uint8, bool) {
	for _, choices := range classesByRace {
		for _, choice := range choices {
			if choice.ID == classID {
				return choice.DOSCode, true
			}
		}
	}
	return 0, false
}

// ClassIDForDOSCode 是 [ClassDOSCode] 的反查：從原版記錄的 `+2Fh` 回到職業 ID。
//
// NPC 沒有經過建角流程，它的職業只存在記錄裡（`36h ADD NPC` 從
// `MON<n>CHA.DAX` 讀一筆 285 位元組的記錄）。少了這一條，NPC 入隊之後任何
// 要查職業的地方都會拿到空字串——**症狀不是顯示錯，是整支查詢報錯**。
func ClassIDForDOSCode(code uint8) (string, bool) {
	for _, choices := range classesByRace {
		for _, choice := range choices {
			if choice.DOSCode == code {
				return choice.ID, true
			}
		}
	}
	return "", false
}

// RaceDOSCode 回傳原版角色記錄 `+2Eh` 那個位元組。碼不連續（人類是 7
// 不是 6），所以一定要查表，不能拿 Races 的索引充當。
func RaceDOSCode(raceID string) (uint8, bool) {
	for _, race := range Races {
		if race.ID == raceID {
			return race.DOSCode, true
		}
	}
	return 0, false
}

// GenderDOSCode 回傳原版角色記錄 `+9Eh` 那個位元組。
func GenderDOSCode(genderID string) (uint8, bool) {
	for _, gender := range Genders {
		if gender.ID == genderID {
			return gender.DOSCode, true
		}
	}
	return 0, false
}
