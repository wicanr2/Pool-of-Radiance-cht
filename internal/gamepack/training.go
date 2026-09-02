package gamepack

// 訓練所的一次訓練（spec 097）。原版是 overlay-16 `2A25h..3027h`：
// 先走一次職業迴圈挑出「升得動」的，順便算經驗值上限，再升等、擲生命骰、
// 加 HP。這裡照那個順序接。
//
// **兩件事還沒讀完，所以沒有接**：原版第一道閘門是種族／能力值的等級上限
// （`2AF0h..2B6Bh`），以及 `DS:466Eh` 非零時略過經驗值檢查的那個旗標。
// 少了前者，遊俠那類受限職業會升過頭——但 Pool of Radiance 走得到的四個職業
// 沒有那個限制，所以目前不影響。

// TrainingOutcome 是一次訓練的結果。
type TrainingOutcome struct {
	// Trained 是這次真的升了級沒有。
	Trained bool
	// ClassMask 是升了哪幾類（分類位元，不是職業索引）。
	ClassMask uint8
	// Levels 是升完之後的八個職業等級。
	Levels [ClassThac0ClassCount]uint8
	// Experience 是套上上限之後的經驗值。原版訓練完會把它砍到
	//「升兩級的門檻減一」，所以囤不了兩級份。
	Experience uint32
	// HitPointGain 是最大 HP 加了多少。
	HitPointGain int
	// PlainHitPointGain 是不含體質加成的那一份，寫進 `+0B1h`。
	PlainHitPointGain int
}

// EligibleTrainingMask 走一次職業迴圈，回傳「經驗值夠而且還沒到上限」的職業
// 分類遮罩，以及訓練完要套的經驗值上限。
//
// 上限取各個可升職業「升兩級的門檻減一」的最小值；一個都算不出來就回 0，
// 代表不必砍。
func (t LevelUpTables) EligibleTrainingMask(levels [ClassThac0ClassCount]uint8,
	experience uint32, thresholds ExperienceTable) (uint8, uint32) {
	mask, ceiling := uint8(0), uint32(0)
	for class, level := range levels {
		if level == 0 {
			continue
		}
		next, ok := thresholds.RequiredExperience(class, int(level)+1)
		if !ok || experience < next {
			continue
		}
		mask |= t.ClassCategory[class]
		// 上限用的是「再下一級」的門檻減一。到不了那一級就不設上限。
		after, ok := thresholds.RequiredExperience(class, int(level)+2)
		if !ok || experience < after {
			continue
		}
		if candidate := after - 1; ceiling == 0 || candidate < ceiling {
			ceiling = candidate
		}
	}
	return mask, ceiling
}

// Train 對一個角色做一次訓練。mask 是這一家訓練所收的職業分類；
// 傳 0 表示不限制，等同於原版那些每一類都收的訓練所。
func (t LevelUpTables) Train(levels [ClassThac0ClassCount]uint8, experience uint32,
	thresholds ExperienceTable, constitution int, classCode uint8, hallMask uint8,
	roller Roller) TrainingOutcome {
	outcome := TrainingOutcome{Levels: levels, Experience: experience}
	eligible, ceiling := t.EligibleTrainingMask(levels, experience, thresholds)
	if hallMask != 0 {
		eligible &= hallMask
	}
	if eligible == 0 {
		return outcome
	}
	// 原版先數「有等級的職業數」，那是等一下 HP 要除的數，
	// 而且是**升級之前**的職業數。
	classes := LeveledClassCount(levels)
	for class := range outcome.Levels {
		if outcome.Levels[class] == 0 || t.ClassCategory[class]&eligible == 0 {
			continue
		}
		outcome.Levels[class]++
	}
	// 擲生命骰在升等之後，所以第一級的地板走不到（spec 097）。
	rolled := t.HitDiceRoll(outcome.Levels, eligible, roller)
	bonus := t.ConstitutionHitPointBonus(outcome.Levels, constitution, classCode)
	gain := LevelUpHitPointGain(rolled, bonus, classes)
	outcome.Trained, outcome.ClassMask = true, eligible
	outcome.HitPointGain, outcome.PlainHitPointGain = gain.Total, gain.WithoutConstitution
	if ceiling != 0 && ceiling < outcome.Experience {
		outcome.Experience = ceiling
	}
	return outcome
}
