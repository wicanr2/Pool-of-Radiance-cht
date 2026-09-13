package gamepack

const (
	// EnergyDrainOneEffectCode／EnergyDrainTwoEffectCode 是 MONnSPC.DAX 的
	// 特殊攻擊代碼（overlay-12 entry 80／81）。
	EnergyDrainOneEffectCode uint8 = 0x55
	EnergyDrainTwoEffectCode uint8 = 0x56
)

// EnergyDrainOutcome 是能量吸取寫回角色記錄的結果（spec 097／112）。
type EnergyDrainOutcome struct {
	Levels           [ClassThac0ClassCount]uint8
	Experience       uint32
	MaxHitPoints     int
	CurrentHitPoints int
	RawHitPoints     int
	DrainedLevels    int
	DrainedHitPoints int
	Killed           bool
}

// DrainEnergy 重現 overlay-12 `21C4h..236Ch` 的玩家狀態變更。
// 每吸一級先以「最大 HP ÷ 全部職業等級總和」扣三份 HP，再依原版 selector
// 同時不降低「目前等級」與「該級門檻」兩個候選值，挑一個職業降一級；55h
// 呼叫一次，56h 呼叫兩次。兩值相同時後面的職業勝出。
func DrainEnergy(levels [ClassThac0ClassCount]uint8, experience uint32,
	maxHitPoints, currentHitPoints, rawHitPoints, drainedLevels, drainedHitPoints, count int,
	thresholds ExperienceTable) EnergyDrainOutcome {
	outcome := EnergyDrainOutcome{
		Levels: levels, Experience: experience,
		MaxHitPoints: maxHitPoints, CurrentHitPoints: currentHitPoints, RawHitPoints: rawHitPoints,
		DrainedLevels: drainedLevels, DrainedHitPoints: drainedHitPoints,
	}
	for drain := 0; drain < count; drain++ {
		totalLevels := 0
		for _, level := range outcome.Levels {
			totalLevels += int(level)
		}
		if totalLevels <= 0 {
			outcome.Killed = true
			outcome.CurrentHitPoints = 0
			break
		}
		loss := outcome.MaxHitPoints / totalLevels
		outcome.DrainedLevels++
		outcome.DrainedHitPoints += loss
		outcome.MaxHitPoints = subtractFloorZero(outcome.MaxHitPoints, loss)
		outcome.CurrentHitPoints = subtractFloorZero(outcome.CurrentHitPoints, loss)
		outcome.RawHitPoints = subtractFloorZero(outcome.RawHitPoints, loss)

		class, threshold, ok := drainClass(outcome.Levels, thresholds)
		if !ok {
			// 原版最高職業等級低於 2 時把角色送進死亡路徑並把最大 HP 歸零。
			outcome.Killed = true
			outcome.MaxHitPoints, outcome.CurrentHitPoints = 0, 0
			break
		}
		outcome.Levels[class]--
		outcome.Experience = threshold
		if outcome.CurrentHitPoints == 0 {
			// overlay-12 在等級與經驗值寫回後檢查 +B1h；吸取本身把目前 HP
			// 扣到零時，也進同一條死亡路徑。
			outcome.Killed = true
			break
		}
	}
	return outcome
}

func drainClass(levels [ClassThac0ClassCount]uint8, thresholds ExperienceTable) (int, uint32, bool) {
	chosen, bestLevel, bestThreshold, ok := 0, uint8(0), uint32(0), false
	for class, level := range levels {
		if level <= 1 {
			continue
		}
		threshold, reachable := thresholds.RequiredExperience(class, int(level))
		if !reachable {
			continue
		}
		if !ok || level >= bestLevel && threshold >= bestThreshold {
			chosen, bestLevel, bestThreshold, ok = class, level, threshold, true
		}
	}
	return chosen, bestThreshold, ok
}

func subtractFloorZero(value, amount int) int {
	if value <= amount {
		return 0
	}
	return value - amount
}

// RestoreDrainedLevel 在還 HP 欠帳之外，把原版挑出的職業等級與經驗值也還回去。
// overlay-22 `2C01h` 挑「升下一級所需門檻最低」的職業。
func RestoreDrainedLevel(levels [ClassThac0ClassCount]uint8, experience uint32,
	drainedLevels, drainedHitPoints int, thresholds ExperienceTable) (RestorationOutcome,
	[ClassThac0ClassCount]uint8, uint32) {
	outcome := Restore(drainedLevels, drainedHitPoints)
	if !outcome.Restored {
		return outcome, levels, experience
	}
	chosen, required, found := 0, uint32(0), false
	for class, level := range levels {
		if level == 0 {
			continue
		}
		next, ok := thresholds.RequiredExperience(class, int(level)+1)
		if !ok {
			continue
		}
		if !found || next < required {
			chosen, required, found = class, next, true
		}
	}
	if found {
		levels[chosen]++
		experience = required
	}
	return outcome, levels, experience
}
