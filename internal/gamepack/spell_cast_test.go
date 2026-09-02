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
	// 34 是臭雲術：處理常式（1AF6h）還沒讀。
	if _, err := CastSpell(34, parameters, 6, maxRoller{}); err == nil {
		t.Error("臭雲術還沒讀完，卻沒有硬失敗")
	}
	if SpellIsImplemented(34) {
		t.Error("臭雲術不該被當成已實作")
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

// 催眠術的花費表逐段對反組譯（overlay-22 1553h..15AFh）。
func TestSleepHitDiceCostBands(t *testing.T) {
	for _, testCase := range []struct {
		hitDice int
		flag    uint8
		want    int
	}{
		{0, 0, 1}, {1, 0, 1}, {-1, 0, 1}, // 1 以下都算 1
		{2, 0, 2}, {3, 0, 4}, {4, 0, 6},
		{5, 0, 10}, {5, 1, 20}, // 第 5 段看 +2Eh
		{6, 0, 20}, {9, 0, 20}, // 六段以上一律 20，等於放不倒
	} {
		if got := SleepHitDiceCost(testCase.hitDice, testCase.flag); got != testCase.want {
			t.Errorf("生命骰 %d、旗標 %d 應該花 %d，算出 %d",
				testCase.hitDice, testCase.flag, testCase.want, got)
		}
	}
}

// 催眠術的額度是 4d4，效果碼是 35h。
func TestSleepBudgetAndEffectCode(t *testing.T) {
	parameters, err := ReadDOSSpellParameters(poolZipPath())
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	low, err := CastSpell(SpellIDSleep, parameters, 6, minRoller{})
	if err != nil {
		t.Fatal(err)
	}
	high, err := CastSpell(SpellIDSleep, parameters, 6, maxRoller{})
	if err != nil {
		t.Fatal(err)
	}
	if low.SleepBudget != 4 || high.SleepBudget != 16 {
		t.Errorf("額度應該是 4d4（4..16），算出 %d..%d", low.SleepBudget, high.SleepBudget)
	}
	if high.EffectCode != SleepEffectCode {
		t.Errorf("效果碼應該是 %#02x，拿到 %#02x", SleepEffectCode, high.EffectCode)
	}
	// 那個效果碼在 overlay-15 的名稱鏈裡查得到——原版自己叫它 "Funky--"。
	if entry, ok := EffectNameFor(SleepEffectCode); !ok || entry.Name != "Funky--" {
		t.Errorf("效果碼 %#02x 應該叫 Funky--，拿到 %+v（找到 %t）",
			SleepEffectCode, entry, ok)
	}
	// 催眠術不造成傷害。
	if high.Damage != 0 {
		t.Errorf("催眠術不該有傷害，算出 %d", high.Damage)
	}
}

// 版型認得出來的那批要正好是二十五格，而且每一格都有訊息。
// 比對整個版型是關鍵：只看「有沒有呼叫 08BCh」會把會算傷害的那幾支
// 一起收進來，然後傷害就消失了——所以這裡順便釘住魔法飛彈不在裡面。
func TestGenericSpellHandlersMatchTheTemplate(t *testing.T) {
	handlers, err := ReadDOSGenericSpellHandlers(poolZipPath())
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	if len(handlers) != 25 {
		t.Fatalf("版型認出 %d 格，先前量到 25 格", len(handlers))
	}
	ids := map[int]string{}
	for _, handler := range handlers {
		if handler.Message == "" {
			t.Errorf("法術 %d 認成泛型卻沒有訊息", handler.SpellID)
		}
		ids[handler.SpellID] = handler.Message
	}
	// 幾條有名有姓的樣本，訊息逐字對原版。
	for id, want := range map[int]string{
		6: "is protected", 19: "is shielded", 30: "is invisible",
		31: "Knock-Knock", 33: "is weakened", 44: "has been cursed!",
	} {
		if got := ids[id]; got != want {
			t.Errorf("法術 %d 的訊息是 %q，原版是 %q", id, got, want)
		}
	}
	// 會算傷害的那幾支不能被收進來。
	for _, id := range []int{SpellIDMagicMissile, SpellIDFireball, SpellIDLightningBolt,
		SpellIDBurningHands, SpellIDShockingGrasp, SpellIDSleep, SpellIDCureLightWound} {
		if _, ok := ids[id]; ok {
			t.Errorf("法術 %d 有自己的算法，不該被當成純泛型", id)
		}
	}
}

// 泛型那批施得出來，效果就是參數表的效果碼。
func TestSpellCasterCoversTheGenericBatch(t *testing.T) {
	parameters, err := ReadDOSSpellParameters(poolZipPath())
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	caster, err := ReadDOSSpellCaster(poolZipPath())
	if err != nil {
		t.Fatal(err)
	}
	// 6 是 Protection From Evil：純泛型。
	if !caster.Implemented(6) {
		t.Fatal("Protection From Evil 應該施得出來")
	}
	effect, err := caster.Cast(6, parameters, 6, maxRoller{})
	if err != nil {
		t.Fatal(err)
	}
	if effect.Damage != 0 || effect.Heal != 0 {
		t.Errorf("純泛型的不該有傷害或治療，拿到 %+v", effect)
	}
	if effect.EffectCode != parameters[6].EffectCode() {
		t.Errorf("效果碼應該來自參數表 %#02x，拿到 %#02x",
			parameters[6].EffectCode(), effect.EffectCode)
	}
	// 逐支讀過的仍然走自己的算法。
	missile, err := caster.Cast(SpellIDMagicMissile, parameters, 6, maxRoller{})
	if err != nil {
		t.Fatal(err)
	}
	if missile.Damage != 15 {
		t.Errorf("第 6 級的魔法飛彈擲滿應該 15 點，拿到 %d", missile.Damage)
	}
	// 兩邊都沒有的仍然硬失敗。
	if _, err := caster.Cast(34, parameters, 6, maxRoller{}); err == nil {
		t.Error("臭雲術兩邊都沒有，應該硬失敗")
	}
}
