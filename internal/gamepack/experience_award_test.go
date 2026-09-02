package gamepack

import "testing"

func abilitiesWith(index, value int) [6]int {
	var abilities [6]int
	for i := range abilities {
		abilities[i] = 10
	}
	abilities[index] = value
	return abilities
}

// 四個純職業各看自己的主屬性，超過 15 多拿十分之一。
func TestPrimeRequisiteBonus(t *testing.T) {
	for _, testCase := range []struct {
		name      string
		classCode uint8
		ability   int
	}{
		{"牧師看睿智", classCodeCleric, AbilityWisdom},
		{"戰士看力量", classCodeFighter, AbilityStrength},
		{"法師看智力", classCodeMagicUser, AbilityIntelligence},
		{"賊看敏捷", classCodeThief, AbilityDexterity},
	} {
		if got := ExperienceShare(1000, testCase.classCode, abilitiesWith(testCase.ability, 16)); got != 1100 {
			t.Errorf("%s：主屬性 16 應該拿 1100，拿到 %d", testCase.name, got)
		}
		if got := ExperienceShare(1000, testCase.classCode, abilitiesWith(testCase.ability, 15)); got != 1000 {
			t.Errorf("%s：主屬性 15 剛好不夠，應該拿 1000，拿到 %d", testCase.name, got)
		}
		// 看錯屬性就抓不到：把別的屬性拉高不該有加成。
		other := (testCase.ability + 1) % 6
		if got := ExperienceShare(1000, testCase.classCode, abilitiesWith(other, 18)); got != 1000 {
			t.Errorf("%s：拉高別的屬性不該有加成，拿到 %d", testCase.name, got)
		}
	}
}

// 複合職業除以職業數，而且拿不到主屬性加成。
func TestMulticlassSharesAreDivided(t *testing.T) {
	high := abilitiesWith(AbilityStrength, 18)
	for _, classCode := range []uint8{8, 10, 11, 12, 13, 14, 16} {
		if got := ExperienceShare(1000, classCode, high); got != 500 {
			t.Errorf("職業碼 %d 應該拿一半 500，拿到 %d", classCode, got)
		}
	}
	for _, classCode := range []uint8{9, 15} {
		if got := ExperienceShare(1000, classCode, high); got != 333 {
			t.Errorf("職業碼 %d 應該拿三分之一 333，拿到 %d", classCode, got)
		}
	}
}

// 一場的總額先除以有資格的人數，再各自套職業規則。
func TestDivideExperienceUsesTheSharerCount(t *testing.T) {
	if got := DivideExperience(1000, 4); got != 250 {
		t.Errorf("1000 分給 4 個人應該各 250，拿到 %d", got)
	}
	// 倒了兩個之後剩下的人分得更多——除數是有資格的人，不是隊伍人數。
	if got := DivideExperience(1000, 2); got != 500 {
		t.Errorf("1000 分給 2 個人應該各 500，拿到 %d", got)
	}
	if got := DivideExperience(1000, 0); got != 0 {
		t.Errorf("沒有人有資格應該是 0，拿到 %d", got)
	}
}
