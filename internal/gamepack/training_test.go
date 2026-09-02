package gamepack

import "testing"

func trainingFixtures(t *testing.T) (LevelUpTables, ExperienceTable) {
	t.Helper()
	tables, err := ReadDOSLevelUpTables(poolZipPath())
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	thresholds, err := ReadDOSExperienceTable(poolZipPath())
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	return tables, thresholds
}

// 經驗值不夠就升不動，剛好夠就升得動。戰士升到第 2 級要 2001 點。
func TestTrainingNeedsTheThreshold(t *testing.T) {
	tables, thresholds := trainingFixtures(t)
	var fighter [ClassThac0ClassCount]uint8
	fighter[2] = 1
	if outcome := tables.Train(fighter, 2000, thresholds, 10, PureFighterClassCode, 0,
		maximumRoller{}); outcome.Trained {
		t.Error("2000 點還不夠升第 2 級，卻升了")
	}
	outcome := tables.Train(fighter, 2001, thresholds, 10, PureFighterClassCode, 0, maximumRoller{})
	if !outcome.Trained || outcome.Levels[2] != 2 {
		t.Fatalf("2001 點應該升到第 2 級，拿到 trained=%t 等級 %d", outcome.Trained, outcome.Levels[2])
	}
	// 一顆 d10 擲滿是 10，除以一個職業，體質 10 沒有加成。
	if outcome.HitPointGain != 10 {
		t.Errorf("擲滿的 d10 應該加 10 點，加了 %d", outcome.HitPointGain)
	}
}

// 訓練完經驗值被砍到「升兩級的門檻減一」。戰士第 3 級要 4001，所以砍到 4000。
func TestTrainingCapsExperience(t *testing.T) {
	tables, thresholds := trainingFixtures(t)
	var fighter [ClassThac0ClassCount]uint8
	fighter[2] = 1
	outcome := tables.Train(fighter, 9000, thresholds, 10, PureFighterClassCode, 0, maximumRoller{})
	if !outcome.Trained || outcome.Levels[2] != 2 {
		t.Fatalf("9000 點應該升一級，拿到等級 %d", outcome.Levels[2])
	}
	if outcome.Experience != 4000 {
		t.Errorf("經驗值應該被砍到 4000，變成 %d", outcome.Experience)
	}
	// 一次只升一級：囤了 9000 也不會直接跳到第 3 級。
	if outcome.Levels[2] != 2 {
		t.Errorf("一次只升一級，卻到了第 %d 級", outcome.Levels[2])
	}
}

// 還沒到下一級的門檻就不砍經驗值。
func TestTrainingWithoutACeilingKeepsExperience(t *testing.T) {
	tables, thresholds := trainingFixtures(t)
	var fighter [ClassThac0ClassCount]uint8
	fighter[2] = 1
	outcome := tables.Train(fighter, 2500, thresholds, 10, PureFighterClassCode, 0, maximumRoller{})
	if !outcome.Trained || outcome.Experience != 2500 {
		t.Errorf("2500 點不到第 3 級的門檻，不該被砍，變成 %d", outcome.Experience)
	}
}

// 到了職業上限就升不動了。法師上限是第 6 級。
func TestTrainingStopsAtTheClassCap(t *testing.T) {
	tables, thresholds := trainingFixtures(t)
	var magicUser [ClassThac0ClassCount]uint8
	magicUser[5] = 6
	if outcome := tables.Train(magicUser, 1 << 30, thresholds, 10, 5, 0, maximumRoller{}); outcome.Trained {
		t.Errorf("法師第 6 級是上限，卻還升得動：%+v", outcome.Levels)
	}
}

// 訓練所只收某幾類的話，只升那幾類。戰士／賊在只收戰士的地方只升戰士。
func TestTrainingObeysTheHallMask(t *testing.T) {
	tables, thresholds := trainingFixtures(t)
	var fighterThief [ClassThac0ClassCount]uint8
	fighterThief[2], fighterThief[6] = 1, 1
	// 兩邊的門檻都過得了：戰士 2001、賊 1251。
	outcome := tables.Train(fighterThief, 3000, thresholds, 10, 14, classCategoryFighter,
		maximumRoller{})
	if !outcome.Trained || outcome.Levels[2] != 2 || outcome.Levels[6] != 1 {
		t.Fatalf("只收戰士的地方應該只升戰士，拿到 %v", outcome.Levels)
	}
	// 不限制的話兩邊一起升，HP 除以兩個職業。
	both := tables.Train(fighterThief, 3000, thresholds, 10, 14, 0, maximumRoller{})
	if both.Levels[2] != 2 || both.Levels[6] != 2 {
		t.Fatalf("不限制的話兩邊都要升，拿到 %v", both.Levels)
	}
	// d10 加 d6 擲滿是 16，除以兩個職業是 8。
	if both.HitPointGain != 8 {
		t.Errorf("兩職業一起升應該加 8 點，加了 %d", both.HitPointGain)
	}
}
