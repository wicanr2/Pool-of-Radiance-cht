package gamepack

// 角色記錄 `+77h` 起的八個賊技能百分比（spec 095）。
//
// 順序與數值由預設人物 `chrdatd4`（TINA，職業碼 6、賊等級 9）釘住：
// 80／77／65／80／66／30／98／45，與規則書第 9 級賊的
// 80／77／65／78／63／30／97／45 逐項對得上——五項到個位數相同，
// 另外三項差 1..3，是敏捷與種族修正。其餘六名預設人物這八格全是 0。
//
// **原版沒有任何 overlay 寫這八格**（四種寫入版面在 38 顆 overlay 與
// START.EXE 裡都掃過，零命中），所以它們是隨記錄檔載進來的。
const (
	// ThiefSkillsOffset 是八格的起點。
	ThiefSkillsOffset = 0x77
	// ThiefSkillCount 是格數。
	ThiefSkillCount = 8
)

// 八個技能的索引，順序即記錄裡的順序。
const (
	ThiefSkillPickPockets = iota
	ThiefSkillOpenLocks
	ThiefSkillFindRemoveTraps
	ThiefSkillMoveSilently
	ThiefSkillHideInShadows
	ThiefSkillHearNoise
	ThiefSkillClimbWalls
	ThiefSkillReadLanguages
)

// ThiefSkills 是那八格。
type ThiefSkills [ThiefSkillCount]uint8

// ThiefSkillsFromRecord 取出記錄裡的八格。
func ThiefSkillsFromRecord(record []byte) (ThiefSkills, error) {
	var skills ThiefSkills
	if len(record) < ThiefSkillsOffset+ThiefSkillCount {
		return skills, errThiefSkillsShortRecord(len(record))
	}
	copy(skills[:], record[ThiefSkillsOffset:])
	return skills, nil
}

func errThiefSkillsShortRecord(size int) error {
	return &shortRecordError{size: size, need: ThiefSkillsOffset + ThiefSkillCount, what: "thief skills"}
}

type shortRecordError struct {
	size int
	need int
	what string
}

func (e *shortRecordError) Error() string {
	return "Pool character record is " + itoa(e.size) + " bytes, the " + e.what + " need " + itoa(e.need)
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	digits := ""
	for value > 0 {
		digits = string(rune('0'+value%10)) + digits
		value /= 10
	}
	return digits
}
