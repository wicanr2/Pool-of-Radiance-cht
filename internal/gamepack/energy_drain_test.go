package gamepack_test

import (
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

func TestOriginalMonsterEffectsIdentifyTheTwoEnergyDrains(t *testing.T) {
	empty, err := gamepack.ReadDOSMonsterEffects(dosZIP, 1, 4)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("missing MON1SPC.DAX produced effects: %+v", empty)
	}
	for _, testCase := range []struct {
		archive uint8
		block   uint8
		code    uint8
	}{
		{archive: 4, block: 20, code: gamepack.EnergyDrainOneEffectCode}, // WIGHT
		{archive: 4, block: 21, code: gamepack.EnergyDrainOneEffectCode}, // WRAITH
		{archive: 4, block: 23, code: gamepack.EnergyDrainTwoEffectCode}, // VAMPIRE
		{archive: 2, block: 17, code: gamepack.EnergyDrainTwoEffectCode}, // SPECTRE
	} {
		effects, err := gamepack.ReadDOSMonsterEffects(dosZIP,
			testCase.archive, testCase.block)
		if err != nil {
			t.Skipf("DOS ZIP unavailable: %v", err)
		}
		if !effects.Has(testCase.code) {
			t.Errorf("MON%dSPC block %d has no effect %02Xh: %+v",
				testCase.archive, testCase.block, testCase.code, effects)
		}
	}
}

func TestEnergyDrainAndRestorationRoundTripACharacterLevel(t *testing.T) {
	var thresholds gamepack.ExperienceTable
	thresholds[2][2], thresholds[2][3] = 2000, 4000
	var levels [gamepack.ClassThac0ClassCount]uint8
	levels[2] = 3

	drained := gamepack.DrainEnergy(levels, 5000, 18, 15, 12, 0, 0, 1, thresholds)
	if drained.Levels[2] != 2 || drained.Experience != 4000 {
		t.Fatalf("吸取後等級／經驗值是 %d／%d", drained.Levels[2], drained.Experience)
	}
	if drained.MaxHitPoints != 12 || drained.CurrentHitPoints != 9 ||
		drained.RawHitPoints != 6 || drained.DrainedLevels != 1 ||
		drained.DrainedHitPoints != 6 || drained.Killed {
		t.Fatalf("吸取結果不符：%+v", drained)
	}

	restored, restoredLevels, restoredExperience := gamepack.RestoreDrainedLevel(
		drained.Levels, drained.Experience, drained.DrainedLevels,
		drained.DrainedHitPoints, thresholds)
	if !restored.Restored || restored.HitPoints != 6 || restored.DrainedLevels != 0 ||
		restored.DrainedHitPoints != 0 || restoredLevels[2] != 3 || restoredExperience != 4000 {
		t.Fatalf("恢復結果不符：%+v levels=%v xp=%d",
			restored, restoredLevels, restoredExperience)
	}
}

func TestEnergyDrainThatConsumesTheLastHitPointKillsTheCharacter(t *testing.T) {
	var thresholds gamepack.ExperienceTable
	thresholds[2][2] = 2000
	var levels [gamepack.ClassThac0ClassCount]uint8
	levels[2] = 2

	drained := gamepack.DrainEnergy(levels, 3000, 10, 4, 4, 0, 0, 1, thresholds)
	if !drained.Killed || drained.CurrentHitPoints != 0 {
		t.Fatalf("吸取扣光最後生命值卻沒有死亡：%+v", drained)
	}
	if drained.Levels[2] != 1 || drained.Experience != 2000 {
		t.Fatalf("死亡前沒有寫回被吸取的等級／經驗值：%+v", drained)
	}
}
