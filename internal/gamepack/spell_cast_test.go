package gamepack

import "testing"

type maxRoller struct{}

func (maxRoller) Roll(count, sides int) int { return count * sides }

type minRoller struct{}

func (minRoller) Roll(count, sides int) int { return count }

// 七支讀出來的處理常式，逐條對反組譯算出來的公式。
func TestCastSpellFormulas(t *testing.T) {
	parameters, err := ReadDOSSpellParameters(poolZipPath())
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	for _, testCase := range []struct {
		name        string
		id          uint8
		level       int
		wantMinimum int
		wantMaximum int
		heal        bool
	}{
		// 傷害＝施法者等級，沒有擲骰，所以上下限相同。
		{"Burning Hands 第 5 級", SpellIDBurningHands, 5, 5, 5, false},
		// Roll(等級, 6)：第 6 級是 6..36。
		{"Fireball 第 6 級", SpellIDFireball, 6, 6, 36, false},
		{"Lightning Bolt 第 6 級", SpellIDLightningBolt, 6, 6, 36, false},
		// Roll(1, 8) ＋ 等級。
		{"Shocking Grasp 第 4 級", SpellIDShockingGrasp, 4, 5, 12, false},
		// Roll(等級÷2, 4) ＋ 等級÷2：第 6 級是 3 發，每發 1d4+1 → 6..15。
		{"Magic Missile 第 6 級", SpellIDMagicMissile, 6, 6, 15, false},
		// 第 1 級照碼算是 0 發：等級÷2 ＝ 0。與說明書不一致，spec 098 記著。
		{"Magic Missile 第 1 級", SpellIDMagicMissile, 1, 0, 0, false},
		// 治療 Roll(1, 8)。
		{"Cure Light Wounds", SpellIDCureLightWound, 6, 1, 8, true},
	} {
		low, err := CastSpell(testCase.id, parameters, testCase.level, minRoller{})
		if err != nil {
			t.Fatalf("%s: %v", testCase.name, err)
		}
		high, err := CastSpell(testCase.id, parameters, testCase.level, maxRoller{})
		if err != nil {
			t.Fatalf("%s: %v", testCase.name, err)
		}
		gotLow, gotHigh := low.Damage, high.Damage
		if testCase.heal {
			gotLow, gotHigh = low.Heal, high.Heal
		}
		if gotLow != testCase.wantMinimum || gotHigh != testCase.wantMaximum {
			t.Errorf("%s 算出 %d..%d，反組譯的公式是 %d..%d",
				testCase.name, gotLow, gotHigh, testCase.wantMinimum, testCase.wantMaximum)
		}
	}
}

// 沒讀過的法術要硬失敗，不能安靜地什麼都不做——那與「正確地不做事」
// 在報表上分不出來。
func TestUnreadSpellsFailLoudly(t *testing.T) {
	parameters, err := ReadDOSSpellParameters(poolZipPath())
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	// 21 是催眠術：處理常式（1513h）的骨架讀了但效果還沒閉合。
	if _, err := CastSpell(21, parameters, 6, maxRoller{}); err == nil {
		t.Error("催眠術還沒讀完，卻沒有硬失敗")
	}
	if SpellIsImplemented(21) {
		t.Error("催眠術不該被當成已實作")
	}
	if !SpellIsImplemented(SpellIDMagicMissile) {
		t.Error("魔法飛彈讀過了，應該算已實作")
	}
}

// 施法者等級：職業選欄位，物品固定 12，戰術地圖外一律 6。
func TestCasterLevelSelection(t *testing.T) {
	parameters, err := ReadDOSSpellParameters(poolZipPath())
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	magicMissile := parameters[SpellIDMagicMissile]
	if got := CasterLevelFor(magicMissile, 3, 5, false); got != 5 {
		t.Errorf("巫術要讀法師等級 5，拿到 %d", got)
	}
	cure := parameters[SpellIDCureLightWound]
	if got := CasterLevelFor(cure, 3, 5, false); got != 3 {
		t.Errorf("神術要讀牧師等級 3，拿到 %d", got)
	}
	if got := CasterLevelFor(magicMissile, 3, 5, true); got != 6 {
		t.Errorf("戰術地圖外一律當 6，拿到 %d", got)
	}
	// 物品效果固定 12，而且不受「地圖外當 6」影響（原版比的是職業不等於 2）。
	for _, entry := range parameters {
		if entry.Source() == SpellSourceItem {
			if got := CasterLevelFor(entry, 3, 5, true); got != 12 {
				t.Errorf("物品效果應該固定 12，拿到 %d", got)
			}
			break
		}
	}
}
