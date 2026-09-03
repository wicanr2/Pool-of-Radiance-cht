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

// 後來讀的四支，逐條對反組譯。
func TestLaterSpellFormulas(t *testing.T) {
	parameters, err := ReadDOSSpellParameters(poolZipPath())
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	// 致傷輕傷：Roll(1, 8)。
	low, _ := CastSpell(SpellIDCauseLightWound, parameters, 6, minRoller{})
	high, _ := CastSpell(SpellIDCauseLightWound, parameters, 6, maxRoller{})
	if low.Damage != 1 || high.Damage != 8 {
		t.Errorf("致傷輕傷應該是 1..8，算出 %d..%d", low.Damage, high.Damage)
	}
	// 鏡影術：Roll(1, 4) 推在施法者等級那一格，不是傷害。
	image, _ := CastSpell(SpellIDMirrorImage, parameters, 6, maxRoller{})
	if image.Damage != 0 {
		t.Errorf("鏡影術不該有傷害，算出 %d", image.Damage)
	}
	if image.CasterLevelOverride != 4 {
		t.Errorf("鏡影術擲滿應該覆寫成 4，算出 %d", image.CasterLevelOverride)
	}
	// 致病術：四個覆寫參數 0／1／0／0，沒有傷害。
	disease, _ := CastSpell(SpellIDCauseDisease, parameters, 6, maxRoller{})
	if disease.Damage != 0 || disease.EffectParameter != 1 {
		t.Errorf("致病術應該沒有傷害而且第二個覆寫參數是 1，算出 %+v", disease)
	}
	if disease.EffectCode != parameters[SpellIDCauseDisease].EffectCode() {
		t.Errorf("致病術的效果碼應該來自參數表")
	}
}

// 詛咒術與祝福術走同一條整邊的路。
func TestCurseMatchesBless(t *testing.T) {
	parameters, err := ReadDOSSpellParameters(poolZipPath())
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	curse, err := CastSpell(SpellIDCurse, parameters, 6, maxRoller{})
	if err != nil {
		t.Fatal(err)
	}
	if !curse.WholeSide || curse.Damage != 0 {
		t.Errorf("詛咒術應該是整邊、沒有傷害，算出 %+v", curse)
	}
	if !SpellIsImplemented(SpellIDCurse) {
		t.Error("詛咒術應該算已實作")
	}
}

// 第三批：祈禱術、靈魂鎚、緩速術、解病術。
func TestThirdBatchSpellFormulas(t *testing.T) {
	parameters, err := ReadDOSSpellParameters(poolZipPath())
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	prayer, err := CastSpell(SpellIDPrayer, parameters, 5, maxRoller{})
	if err != nil {
		t.Fatal(err)
	}
	if prayer.Damage != 0 || prayer.CasterLevelOverride != 5 {
		t.Errorf("祈禱術應該沒有傷害、等級覆寫是 5，算出 %+v", prayer)
	}
	hammer, _ := CastSpell(SpellIDSpiritHammer, parameters, 6, maxRoller{})
	if hammer.Damage != 0 || hammer.EffectParameter != 1 {
		t.Errorf("靈魂鎚的第二個覆寫參數應該是 1，算出 %+v", hammer)
	}
	slow, _ := CastSpell(SpellIDSlow, parameters, 6, maxRoller{})
	if !slow.Area || slow.EffectCode != SlowEffectCode {
		t.Errorf("緩速術應該是範圍、效果碼 %#02x，算出 %+v", SlowEffectCode, slow)
	}
	cure, _ := CastSpell(SpellIDCureDisease, parameters, 6, maxRoller{})
	if len(cure.RemoveEffects) != len(CureDiseaseEffectCodes) {
		t.Fatalf("解病術要拿掉 %d 個效果碼，算出 %d 個",
			len(CureDiseaseEffectCodes), len(cure.RemoveEffects))
	}
	// 拿掉的碼要與 overlay-15 的名稱鏈對得上：2Ch 致病、32h 木乃伊惡疾、1Fh 無助。
	for code, want := range map[uint8]string{
		0x2c: "Cause Disease", 0x32: "Dreaded Mummy Disease", 0x1f: "Helpless",
	} {
		found := false
		for _, value := range cure.RemoveEffects {
			if value == code {
				found = true
			}
		}
		if !found {
			t.Errorf("解病術應該拿掉 %#02x（%s）", code, want)
			continue
		}
		if entry, ok := EffectNameFor(code); !ok || entry.Name != want {
			t.Errorf("效果碼 %#02x 的名稱應該是 %q，拿到 %+v", code, want, entry)
		}
	}
	// 解病術不掛新效果、也沒有傷害。
	if cure.Damage != 0 || cure.Heal != 0 {
		t.Errorf("解病術不該有傷害或治療，算出 %+v", cure)
	}
}

// 第四批：友誼術、解盲術、解除詛咒。
func TestFourthBatchSpellFormulas(t *testing.T) {
	parameters, err := ReadDOSSpellParameters(poolZipPath())
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	low, _ := CastSpell(SpellIDFriends, parameters, 6, minRoller{})
	high, _ := CastSpell(SpellIDFriends, parameters, 6, maxRoller{})
	if low.AbilityBonus.Ability != AbilityCharisma || high.AbilityBonus.Ability != AbilityCharisma {
		t.Error("友誼術加的是魅力")
	}
	if low.AbilityBonus.Amount != 2 || high.AbilityBonus.Amount != 8 {
		t.Errorf("友誼術是 2d4（2..8），算出 %d..%d",
			low.AbilityBonus.Amount, high.AbilityBonus.Amount)
	}
	if high.AbilityBonus.Cap != 25 {
		t.Errorf("上限應該是 25，算出 %d", high.AbilityBonus.Cap)
	}
	blind, _ := CastSpell(SpellIDCureBlindness, parameters, 6, maxRoller{})
	if len(blind.RemoveEffects) != 1 || blind.RemoveEffects[0] != 0x21 {
		t.Errorf("解盲術應該解掉 21h，算出 %v", blind.RemoveEffects)
	}
	// 交叉核對：致盲術（38，純泛型）掛的效果碼就是解盲術解掉的那一個。
	if got := parameters[38].EffectCode(); got != 0x21 {
		t.Errorf("致盲術掛的效果碼是 %#02x，解盲術解的是 21h——兩邊對不上", got)
	}
	curse, _ := CastSpell(SpellIDRemoveCurse, parameters, 6, maxRoller{})
	if len(curse.RemoveEffects) != 1 || curse.RemoveEffects[0] != 0x24 {
		t.Errorf("解除詛咒應該解掉 24h，算出 %v", curse.RemoveEffects)
	}
	// 同樣交叉核對：賦予詛咒（44，純泛型）掛的就是 24h。
	if got := parameters[44].EffectCode(); got != 0x24 {
		t.Errorf("賦予詛咒掛的效果碼是 %#02x，解除詛咒解的是 24h——兩邊對不上", got)
	}
}

// 接得出來的總數。這一條是為了讓文件裡的數字不會過期：改了實作卻沒改
// 文件就會紅。
func TestImplementedSpellCount(t *testing.T) {
	caster, err := ReadDOSSpellCaster(poolZipPath())
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	read, generic, total := 0, 0, 0
	for id := 1; id <= SpellDispatchCount; id++ {
		if !caster.Implemented(uint8(id)) {
			continue
		}
		total++
		if SpellIsImplemented(uint8(id)) {
			read++
		} else {
			generic++
		}
	}
	if read != 19 || generic != 25 || total != 44 {
		t.Fatalf("逐支讀的 %d 支、純泛型的 %d 支、合計 %d 支；"+
			"文件寫的是 19／25／44，改了實作要一起改", read, generic, total)
	}
	if SpellDispatchCount != 67 {
		t.Fatalf("派發表是 %d 格，spec 寫的是 67", SpellDispatchCount)
	}
}
