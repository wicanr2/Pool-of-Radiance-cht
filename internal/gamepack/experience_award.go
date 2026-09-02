package gamepack

// 經驗值怎麼發（spec 097）。overlay-05 `033Ah` 走過隊伍鏈，
// 每個人依自己的複合職業碼與主屬性決定拿多少。
const (
	// PrimeRequisiteBonusDivisor 是主屬性加成的除數：多拿十分之一。
	PrimeRequisiteBonusDivisor = 10
	// PrimeRequisiteThreshold 是主屬性要超過多少才有加成（`cmp 0Fh` 之後 `jbe` 跳過）。
	PrimeRequisiteThreshold = 15

	// classCodeCleric 等是複合職業碼 `+2Fh` 裡的四個純職業。
	classCodeCleric     = 0
	classCodeFighter    = 2
	classCodeMagicUser  = 5
	classCodeThief      = 6
	// 以下是被除以 2 與除以 3 的複合職業碼區間（`0438h..046Bh`）。
	classCodeHalfShareSingle = 8
	classCodeHalfShareLow    = 10
	classCodeHalfShareHigh   = 14
	classCodeHalfShareExtra  = 16
	classCodeThirdShareLow   = 9
	classCodeThirdShareHigh  = 15
)

// 能力值在 save.Character.Abilities 裡的位置，與角色記錄 `+10h` 起同順序。
const (
	AbilityStrength     = 0
	AbilityIntelligence = 1
	AbilityWisdom       = 2
	AbilityDexterity    = 3
	AbilityConstitution = 4
	AbilityCharisma     = 5
)

// primeRequisiteAbility 回傳這個純職業看哪一個能力值；不是純職業就回 -1。
func primeRequisiteAbility(classCode uint8) int {
	switch classCode {
	case classCodeCleric:
		return AbilityWisdom
	case classCodeFighter:
		return AbilityStrength
	case classCodeMagicUser:
		return AbilityIntelligence
	case classCodeThief:
		return AbilityDexterity
	}
	return -1
}

// ExperienceShare 是一個角色從一筆經驗值裡拿到多少。
//
// 純職業的主屬性超過 15 多拿十分之一；複合職業改成除以職業數。
// 兩者互斥——原版是 if/else if 的鏈，複合職業走不到加成那一段。
func ExperienceShare(amount uint32, classCode uint8, abilities [6]int) uint32 {
	if ability := primeRequisiteAbility(classCode); ability >= 0 {
		if abilities[ability] > PrimeRequisiteThreshold {
			return amount + amount/PrimeRequisiteBonusDivisor
		}
		return amount
	}
	switch {
	case classCode == classCodeThirdShareLow, classCode == classCodeThirdShareHigh:
		return amount / 3
	case classCode == classCodeHalfShareSingle,
		classCode >= classCodeHalfShareLow && classCode <= classCodeHalfShareHigh,
		classCode == classCodeHalfShareExtra:
		return amount / 2
	}
	return amount
}

// DivideExperience 把一場的總經驗值分給有資格的隊員（overlay-05 `0308h`）。
//
// 除數是「隊伍人數減掉沒資格的人」，不是隊伍人數：昏迷、離隊的不佔份額，
// 所以人倒得越多，活著的人分得越多。沒有人有資格就回 0。
func DivideExperience(total uint32, sharers int) uint32 {
	if sharers <= 0 {
		return 0
	}
	return total / uint32(sharers)
}
