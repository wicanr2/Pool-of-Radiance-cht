package gamepack

import (
	"path/filepath"
	"testing"
)

func poolZipPath() string { return filepath.Join("..", "..", "Pool of Radiance (1988).zip") }

// 四張表逐格對原版的位元組。走得到的四個職業另外與 AD&D 一版對過。
func TestLevelUpTablesMatchTheOriginal(t *testing.T) {
	tables, err := ReadDOSLevelUpTables(poolZipPath())
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	wantCategory := [ClassThac0ClassCount]uint8{
		classCategoryPriest,                        // 0 牧師
		classCategoryPriest,                        // 1 德魯伊
		classCategoryFighter,                       // 2 戰士
		classCategoryFighter | classCategoryPriest, // 3 聖武士
		classCategoryFighter | classCategoryPriest, // 4 遊俠
		classCategorySpell,                         // 5 法師
		classCategoryThief,                         // 6 賊
		classCategoryPriest | classCategoryThief,   // 7 武僧
	}
	if tables.ClassCategory != wantCategory {
		t.Fatalf("職業分類遮罩是 %v，原版是 %v", tables.ClassCategory, wantCategory)
	}
	wantCount := [ClassThac0ClassCount]uint8{1, 1, 1, 1, 2, 1, 1, 2}
	wantSides := [ClassThac0ClassCount]uint8{8, 8, 10, 10, 8, 4, 6, 4}
	if tables.HitDiceCount != wantCount || tables.HitDiceSides != wantSides {
		t.Fatalf("生命骰是 %v d %v，原版是 %v d %v",
			tables.HitDiceCount, tables.HitDiceSides, wantCount, wantSides)
	}
	// 體質加成與 AD&D 一版逐格相同。
	for constitution, want := range map[int]int8{
		3: -2, 4: -1, 5: -1, 6: -1, 7: 0, 10: 0, 14: 0, 15: 1, 16: 2, 17: 2, 18: 2,
	} {
		if got := tables.ConstitutionBonus[constitution]; got != want {
			t.Errorf("體質 %d 的加成是 %d，AD&D 是 %d", constitution, got, want)
		}
	}
}

type fixedRoller struct{ value int }

func (r fixedRoller) Roll(count, sides int) int { return r.value }

type maximumRoller struct{}

func (maximumRoller) Roll(count, sides int) int { return count * sides }

// 第一級的地板只在職業目前是第 1 級時生效，而且是照面數算的。
func TestFirstLevelHitDieFloor(t *testing.T) {
	tables, err := ReadDOSLevelUpTables(poolZipPath())
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	var fighter [ClassThac0ClassCount]uint8
	fighter[2] = 1
	if got := tables.HitDiceRoll(fighter, classCategoryFighter, fixedRoller{1}); got != 6 {
		t.Errorf("第 1 級的戰士擲出 1 應該墊到 6（d10 × 2 ÷ 3），拿到 %d", got)
	}
	var magicUser [ClassThac0ClassCount]uint8
	magicUser[5] = 1
	if got := tables.HitDiceRoll(magicUser, classCategorySpell, fixedRoller{1}); got != 2 {
		t.Errorf("第 1 級的法師擲出 1 應該墊到 2（d4 × 2 ÷ 3），拿到 %d", got)
	}
	// 第 2 級以上沒有地板：訓練所是先加等級再擲，所以走不到。
	fighter[2] = 2
	if got := tables.HitDiceRoll(fighter, classCategoryFighter, fixedRoller{1}); got != 1 {
		t.Errorf("第 2 級的戰士不該有地板，拿到 %d", got)
	}
}

// 分類遮罩對不上的職業不擲。多職業只升被選中的那一類。
func TestHitDiceRollObeysTheCategoryMask(t *testing.T) {
	tables, err := ReadDOSLevelUpTables(poolZipPath())
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	var fighterThief [ClassThac0ClassCount]uint8
	fighterThief[2], fighterThief[6] = 3, 3
	if got := tables.HitDiceRoll(fighterThief, classCategoryFighter, maximumRoller{}); got != 10 {
		t.Errorf("只訓練戰士那一邊應該只擲 d10，拿到 %d", got)
	}
	if got := tables.HitDiceRoll(fighterThief, classCategoryThief, maximumRoller{}); got != 6 {
		t.Errorf("只訓練賊那一邊應該只擲 d6，拿到 %d", got)
	}
	if got := tables.HitDiceRoll(fighterThief, classCategoryFighter|classCategoryThief,
		maximumRoller{}); got != 16 {
		t.Errorf("兩邊一起訓練應該是 d10 加 d6，拿到 %d", got)
	}
}

// 體質加成是「每個有等級的職業各一份」，呼叫端除完剛好一份。
// 純戰士的額外加成比的是複合職業碼，戰士／賊拿不到。
func TestConstitutionBonusPerClassAndTheFighterExtra(t *testing.T) {
	tables, err := ReadDOSLevelUpTables(poolZipPath())
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	var fighter [ClassThac0ClassCount]uint8
	fighter[2] = 3
	if got := tables.ConstitutionHitPointBonus(fighter, 18, PureFighterClassCode); got != 4 {
		t.Errorf("體質 18 的純戰士應該是 +4，拿到 %d", got)
	}
	if got := tables.ConstitutionHitPointBonus(fighter, 17, PureFighterClassCode); got != 3 {
		t.Errorf("體質 17 的純戰士應該是 +3，拿到 %d", got)
	}
	var fighterThief [ClassThac0ClassCount]uint8
	fighterThief[2], fighterThief[6] = 3, 3
	// 複合職業碼 14 是戰士／賊：拿到的是一般的 +2，每個職業各一份。
	if got := tables.ConstitutionHitPointBonus(fighterThief, 18, 14); got != 4 {
		t.Errorf("體質 18 的戰士／賊應該是兩個職業各 +2，拿到 %d", got)
	}
	if got := LevelUpHitPointGain(10, 4, 2).Total; got != 7 {
		t.Errorf("兩個職業、擲 10、加成 4 應該得 7，拿到 %d", got)
	}
}

// HP 至少加 1，而且不含體質加成的那一份分開算。
func TestLevelUpHitPointGainFloors(t *testing.T) {
	if got := LevelUpHitPointGain(1, -6, 1); got.Total != 1 {
		t.Errorf("體質再差也至少加 1，拿到 %d", got.Total)
	}
	if got := LevelUpHitPointGain(1, 0, 3); got.WithoutConstitution != 1 {
		t.Errorf("除下來是 0 也要當 1，拿到 %d", got.WithoutConstitution)
	}
	if got := LevelUpHitPointGain(9, 3, 3); got.WithoutConstitution != 3 || got.Total != 4 {
		t.Errorf("擲 9、加成 3、三職業應該是 3 與 4，拿到 %d 與 %d",
			got.WithoutConstitution, got.Total)
	}
}

// 加 HP 要保留受傷量，不是治好。
func TestApplyLevelUpHitPointsKeepsTheWound(t *testing.T) {
	maximum, current := ApplyLevelUpHitPoints(20, 8, 5)
	if maximum != 25 || current != 13 {
		t.Fatalf("20/8 加 5 應該變成 25/13，拿到 %d/%d", maximum, current)
	}
	if maximum, current := ApplyLevelUpHitPoints(20, 20, 5); maximum != 25 || current != 25 {
		t.Fatalf("沒受傷的加完仍是滿的，拿到 %d/%d", maximum, current)
	}
}
