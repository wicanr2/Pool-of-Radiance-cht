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
	if read != 41 || generic != 25 || total != 66 {
		t.Fatalf("逐支讀的 %d 支、純泛型的 %d 支、合計 %d 支；"+
			"文件寫的是 41／25／66，改了實作要一起改", read, generic, total)
	}
	if SpellDispatchCount != 67 {
		t.Fatalf("派發表是 %d 格，spec 寫的是 67", SpellDispatchCount)
	}
}

// 力量術加多少看目標的職業，與 AD&D 逐項相同。
func TestStrengthSpellDieByClass(t *testing.T) {
	var levels [ClassThac0ClassCount]uint8
	levels[ClassSlotMagicUser] = 3
	if count, sides := StrengthSpellDie(levels); count != 1 || sides != 4 {
		t.Errorf("法師應該是 1d4，算出 %dd%d", count, sides)
	}
	levels = [ClassThac0ClassCount]uint8{}
	levels[ClassSlotCleric] = 3
	if _, sides := StrengthSpellDie(levels); sides != 6 {
		t.Errorf("牧師應該是 1d6，算出 d%d", sides)
	}
	levels = [ClassThac0ClassCount]uint8{}
	levels[ClassSlotThief] = 3
	if _, sides := StrengthSpellDie(levels); sides != 6 {
		t.Errorf("賊應該是 1d6，算出 d%d", sides)
	}
	levels = [ClassThac0ClassCount]uint8{}
	levels[ClassSlotFighter] = 3
	if _, sides := StrengthSpellDie(levels); sides != 8 {
		t.Errorf("戰士應該是 1d8，算出 d%d", sides)
	}
	// 戰士／法師：後面的判斷蓋掉前面的，所以取戰士那一個。
	levels[ClassSlotMagicUser] = 3
	if _, sides := StrengthSpellDie(levels); sides != 8 {
		t.Errorf("戰士／法師應該取 1d8，算出 d%d", sides)
	}
	// 沒有職業就沒有骰子。
	if count, _ := StrengthSpellDie([ClassThac0ClassCount]uint8{}); count != 0 {
		t.Error("沒有職業不該擲骰")
	}
}

// 三支後補的：與火球共用的那支、Roll(2,4)+2 的那支、以及整支是空的那支。
func TestTrailingSpellHandlers(t *testing.T) {
	parameters, err := ReadDOSSpellParameters(poolZipPath())
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	alt, _ := CastSpell(SpellIDFireballAlt, parameters, 6, maxRoller{})
	fire, _ := CastSpell(SpellIDFireball, parameters, 6, maxRoller{})
	if alt.Damage != fire.Damage || !alt.Area {
		t.Errorf("64 與火球共用同一支處理常式，算出 %+v vs %+v", alt, fire)
	}
	low, _ := CastSpell(SpellIDMagicMissileAlt, parameters, 6, minRoller{})
	high, _ := CastSpell(SpellIDMagicMissileAlt, parameters, 6, maxRoller{})
	if low.Damage != 4 || high.Damage != 10 {
		t.Errorf("65 是 Roll(2,4)+2（4..10），算出 %d..%d", low.Damage, high.Damage)
	}
	// 66 整支是空的：原版什麼都不做，所以「沒有效果」才是對的答案。
	none, err := CastSpell(SpellIDNoOperation, parameters, 6, maxRoller{})
	if err != nil {
		t.Fatal(err)
	}
	if none.Damage != 0 || none.Heal != 0 || none.SleepBudget != 0 ||
		none.WholeSide || none.Area || len(none.RemoveEffects) != 0 {
		t.Errorf("66 應該什麼都不做，算出 %+v", none)
	}
}

// 兩支治療與一支有前提的泛型。
func TestHealAndGuardedHandlers(t *testing.T) {
	parameters, err := ReadDOSSpellParameters(poolZipPath())
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	low, _ := CastSpell(SpellIDGreaterHeal, parameters, 6, minRoller{})
	high, _ := CastSpell(SpellIDGreaterHeal, parameters, 6, maxRoller{})
	if low.Heal != 9 || high.Heal != 12 {
		t.Errorf("58 是 Roll(1,4)+8（9..12），算出 %d..%d", low.Heal, high.Heal)
	}
	// 它也走一次解病術那條鏈，外加 16h。
	if len(high.RemoveEffects) != len(CureDiseaseEffectCodes)+1 ||
		high.RemoveEffects[0] != 0x16 {
		t.Errorf("58 應該先解 16h 再解病痛那組，算出 %v", high.RemoveEffects)
	}
	lowLesser, _ := CastSpell(SpellIDLesserHeal, parameters, 6, minRoller{})
	highLesser, _ := CastSpell(SpellIDLesserHeal, parameters, 6, maxRoller{})
	if lowLesser.Heal != 4 || highLesser.Heal != 10 {
		t.Errorf("62 是 Roll(2,4)+2（4..10），算出 %d..%d",
			lowLesser.Heal, highLesser.Heal)
	}
	guarded, _ := CastSpell(SpellIDGuardedGeneric, parameters, 6, maxRoller{})
	if guarded.BlockedByEffect != 0x2a {
		t.Errorf("57 的前提應該是效果碼 2Ah，算出 %#02x", guarded.BlockedByEffect)
	}
	if guarded.Damage != 0 || guarded.Heal != 0 {
		t.Errorf("57 走的是泛型那條，不該有傷害或治療：%+v", guarded)
	}
}

// 急速術掛的效果碼，正好是編號 57 那一支在防的那個。
// 兩邊各自從碼裡讀出來，對上同一個碼。
func TestHasteAndItsGuardAgree(t *testing.T) {
	parameters, err := ReadDOSSpellParameters(poolZipPath())
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	haste, err := CastSpell(SpellIDHaste, parameters, 6, maxRoller{})
	if err != nil {
		t.Fatal(err)
	}
	if !haste.WholeSide || haste.EffectCode != HasteEffectCode {
		t.Errorf("急速術應該是整邊、效果碼 %#02x，算出 %+v", HasteEffectCode, haste)
	}
	guarded, _ := CastSpell(SpellIDGuardedGeneric, parameters, 6, maxRoller{})
	if guarded.BlockedByEffect != haste.EffectCode {
		t.Errorf("57 防的是 %#02x，急速術掛的是 %#02x——兩邊對不上",
			guarded.BlockedByEffect, haste.EffectCode)
	}
	// 參數表也把急速與緩速歸在同一個目標模式（整邊）。
	if !parameters[SpellIDHaste].AffectsWholeSide() ||
		!parameters[SpellIDSlow].AffectsWholeSide() {
		t.Error("急速與緩速在參數表裡都該是整邊模式")
	}
}

// 緩毒術把倒在 0 的人墊回 1。
func TestSlowPoisonRaisesZeroHitPoints(t *testing.T) {
	parameters, err := ReadDOSSpellParameters(poolZipPath())
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	effect, err := CastSpell(SpellIDSlowPoison, parameters, 6, maxRoller{})
	if err != nil {
		t.Fatal(err)
	}
	if effect.MinimumHitPoints != 1 {
		t.Errorf("緩毒術應該把生命值墊到 1，算出 %d", effect.MinimumHitPoints)
	}
	if effect.Damage != 0 || effect.Heal != 0 {
		t.Errorf("緩毒術不是傷害也不是治療：%+v", effect)
	}
	// 等級覆寫推的是 FFh，照實接。
	if effect.CasterLevelOverride != 0xff {
		t.Errorf("等級覆寫應該是 FFh，算出 %#02x", effect.CasterLevelOverride)
	}
}

// 變大術的強度依施法者等級查表。
func TestEnlargeMagnitude(t *testing.T) {
	parameters, err := ReadDOSSpellParameters(poolZipPath())
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	for level, want := range map[int]int{1: 0, 2: 1, 3: 0x33, 4: 0x4c, 5: 0x5b, 6: 0x64} {
		effect, err := CastSpell(SpellIDEnlarge, parameters, level, maxRoller{})
		if err != nil {
			t.Fatal(err)
		}
		if effect.EffectParameter != want {
			t.Errorf("第 %d 級的強度應該是 %#02x，算出 %#02x",
				level, want, effect.EffectParameter)
		}
		if effect.EffectCode != EnlargeEffectCode {
			t.Errorf("效果碼應該是 %#02x，算出 %#02x", EnlargeEffectCode, effect.EffectCode)
		}
		// `12h` 是要設成的力量值 18，不是效果碼；效果碼是參數表的 `0Ch`。
		if effect.StrengthValue != EnlargeStrengthValue ||
			int(effect.StrengthPercentile) != want {
			t.Errorf("第 %d 級應該把力量設成 18/%02d，算出 %d/%02d",
				level, want, effect.StrengthValue, effect.StrengthPercentile)
		}
	}
}

// 恢復術還一級：HP 是「欠的 HP 除以欠的等級數」，還完欠帳各少一份。
func TestRestoreRepaysOneLevel(t *testing.T) {
	// 欠三級、欠 12 點：一次還 4 點，剩兩級、8 點。
	outcome := Restore(3, 12)
	if !outcome.Restored || outcome.HitPoints != 4 ||
		outcome.DrainedLevels != 2 || outcome.DrainedHitPoints != 8 {
		t.Fatalf("還一級的結果不對：%+v", outcome)
	}
	// 連還三次要把欠帳清光。
	second := Restore(outcome.DrainedLevels, outcome.DrainedHitPoints)
	third := Restore(second.DrainedLevels, second.DrainedHitPoints)
	if third.DrainedLevels != 0 || third.DrainedHitPoints != 0 {
		t.Errorf("還三次應該清光，剩 %d 級 %d 點",
			third.DrainedLevels, third.DrainedHitPoints)
	}
	if outcome.HitPoints+second.HitPoints+third.HitPoints != 12 {
		t.Errorf("還回來的總和應該是 12，拿到 %d",
			outcome.HitPoints+second.HitPoints+third.HitPoints)
	}
	// 沒有欠帳就什麼都不做。
	if Restore(0, 0).Restored {
		t.Error("沒有被吸取過就不該還")
	}
}

// 魅惑人類與定身術只對「人」有效：種類 `+9Fh` 不大於 1、體型 `+6Ch` 不大於 1
//（overlay-22 `11DAh`／`174Bh`）。原版資料裡的四個代表值都對過。
func TestSpellAffectsPersonMatchesTheOriginalGate(t *testing.T) {
	for _, want := range []struct {
		name                  string
		creatureType, bodySize uint8
		person                bool
	}{
		{"4TH LVL FIGHTER", 0x00, 0x01, true},
		{"ORC", 0x01, 0x01, true},
		{"BUGBEAR", 0x01, 0x81, false},
		{"HILL GIANT", 0x02, 0x82, false},
		{"SKELETON", 0x04, 0x01, false},
	} {
		if got := SpellAffectsPerson(want.creatureType, want.bodySize); got != want.person {
			t.Errorf("%s（種類 %#02x 體型 %#02x）算不算人：算出 %v，應該是 %v",
				want.name, want.creatureType, want.bodySize, got, want.person)
		}
	}
}

// 定身術的豁免修正看目標數，而且一個目標時兩個編號給的值不同
//（overlay-22 `1656h..168Ch`）。
func TestHoldPersonSaveModifierByTargetCount(t *testing.T) {
	for _, want := range []struct {
		id       uint8
		targets  int
		modifier int
	}{
		{SpellIDHoldPerson, 1, -2},
		{SpellIDHoldPersonAlt, 1, -3},
		{SpellIDHoldPerson, 2, -1},
		{SpellIDHoldPersonAlt, 2, -1},
		{SpellIDHoldPerson, 3, 0},
		{SpellIDHoldPerson, 4, 0},
	} {
		if got := HoldPersonSaveModifier(want.id, want.targets); got != want.modifier {
			t.Errorf("法術 %d 選 %d 個目標的修正是 %d，應該是 %d",
				want.id, want.targets, got, want.modifier)
		}
	}
}

// 四支新接的法術各自要帶出來的東西。
func TestCharmAndHoldCastEffects(t *testing.T) {
	parameters, err := ReadDOSSpellParameters(poolZipPath())
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	charm, err := CastSpell(SpellIDCharmPerson, parameters, 6, maxRoller{})
	if err != nil {
		t.Fatalf("魅惑人類：%v", err)
	}
	if !charm.PersonOnly || charm.EffectCode != 0x0b || charm.CasterLevelOverride != 6 {
		t.Errorf("魅惑人類算出 %+v", charm)
	}
	for _, id := range []uint8{SpellIDHoldPerson, SpellIDHoldPersonAlt} {
		hold, err := CastSpell(id, parameters, 6, maxRoller{})
		if err != nil {
			t.Fatalf("定身術 %d：%v", id, err)
		}
		if !hold.PersonOnly || !hold.SaveModifierByTargetCount ||
			hold.EffectCode != HoldPersonEffectCode {
			t.Errorf("定身術 %d 算出 %+v", id, hold)
		}
	}
	snake, err := CastSpell(SpellIDSnakeCharm, parameters, 6, maxRoller{})
	if err != nil {
		t.Fatalf("迷蛇術：%v", err)
	}
	if !snake.CreatureTypeFiltered || snake.CreatureType != CreatureTypeSnake ||
		!snake.HitPointBudgetFromCaster || snake.PersonOnly {
		t.Errorf("迷蛇術算出 %+v", snake)
	}
}

// 種類欄位的四個獨立使用點各自指同一件事，這裡拿原版記錄當正對照：
// 死靈術只認 0（人類）、迷蛇術只認 0Eh，而不死一律是 4。
func TestMonsterCreatureTypeFromOriginalRecords(t *testing.T) {
	for _, want := range []struct {
		archive, block uint8
		name           string
		creatureType   uint8
		bodySize       uint8
	}{
		{1, 4, "ORC", 0x01, 0x01},
		{1, 8, "OGRE", 0x01, 0x82},
		{4, 34, "SKELETON", 0x04, 0x01},
		{5, 60, "GIANT SNAKE", 0x0e, 0x83},
		{1, 41, "4TH LVL FIGHTER", 0x00, 0x01},
	} {
		record, err := ReadDOSMonsterRecord(poolZipPath(), want.archive, want.block)
		if err != nil {
			t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
		}
		if record.Name != want.name {
			t.Fatalf("mon%d/%d 是 %q，不是 %q", want.archive, want.block, record.Name, want.name)
		}
		if record.CreatureType() != want.creatureType || record.BodySize() != want.bodySize {
			t.Errorf("%s 的種類／體型是 %#02x／%#02x，應該是 %#02x／%#02x",
				want.name, record.CreatureType(), record.BodySize(),
				want.creatureType, want.bodySize)
		}
	}
}

// 變大術的效果碼由兩條互相獨立的路對上：參數表 `+0Ah` 與處理常式
// `1331h` 推給掛效果常式的字面值。縮小術要求的正是同一個碼。
func TestEnlargeEffectCodeMatchesTheParameterTable(t *testing.T) {
	parameters, err := ReadDOSSpellParameters(poolZipPath())
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	if got := parameters[SpellIDEnlarge].EffectCode(); got != EnlargeEffectCode {
		t.Errorf("變大術參數表的效果碼是 %#02x，常數寫的是 %#02x",
			got, EnlargeEffectCode)
	}
	if got := parameters[SpellIDGiantStrength].EffectCode(); got != GiantStrengthEffectCode {
		t.Errorf("編號 %d 參數表的效果碼是 %#02x，常數寫的是 %#02x",
			SpellIDGiantStrength, got, GiantStrengthEffectCode)
	}
	reduce, err := CastSpell(SpellIDReduce, parameters, 6, maxRoller{})
	if err != nil {
		t.Fatalf("縮小術：%v", err)
	}
	if reduce.RequiresEffect != EnlargeEffectCode || !reduce.MessageOnly ||
		reduce.EffectCode != 0 {
		t.Errorf("縮小術算出 %+v", reduce)
	}
	giant, err := CastSpell(SpellIDGiantStrength, parameters, 6, maxRoller{})
	if err != nil {
		t.Fatalf("編號 %d：%v", SpellIDGiantStrength, err)
	}
	if giant.StrengthValue != GiantStrengthValue || giant.StrengthPercentile != 0 ||
		giant.EffectCode != GiantStrengthEffectCode {
		t.Errorf("編號 %d 算出 %+v", SpellIDGiantStrength, giant)
	}
}

// 編號 60（原版沒有名字）的傷害是 `Roll(1, 6) + 20`，而且參數表的
// 豁免規則與類別跟處理常式裡寫死的值一致——兩條獨立的路對得上。
func TestRayDamageSpellMatchesTheParameterTable(t *testing.T) {
	parameters, err := ReadDOSSpellParameters(poolZipPath())
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	effect, err := CastSpell(SpellIDRayDamage, parameters, 6, maxRoller{})
	if err != nil {
		t.Fatalf("編號 %d：%v", SpellIDRayDamage, err)
	}
	if effect.Damage != 26 || !effect.Area {
		t.Errorf("最大骰應該是 6 + 20 ＝ 26 的範圍傷害，算出 %+v", effect)
	}
	// `287Ch` 寫死 `push 2`（減半）與 `push 4`（類別），參數表 +8／+9 相同。
	if got := parameters[SpellIDRayDamage].SaveRule(); got != 2 {
		t.Errorf("豁免規則是 %d，處理常式寫死的是 2", got)
	}
	if got := parameters[SpellIDRayDamage].SaveCategory(); got != 4 {
		t.Errorf("豁免類別是 %d，處理常式寫死的是 4", got)
	}
}

// 力量往上調的兩道關卡（overlay-24 entry 18 `1158h`）。
func TestRaiseStrengthOnlyGoesUp(t *testing.T) {
	for _, want := range []struct {
		name                       string
		cur, curPct, val, pct      uint8
		outVal, outPct             uint8
		raised                     bool
	}{
		{"比目前低就不調", 18, 50, 16, 0, 18, 50, false},
		{"同樣是 18 但百分位更低也不調", 18, 90, 18, 51, 18, 90, false},
		{"同樣是 18 百分位更高就調", 18, 50, 18, 91, 18, 91, true},
		{"比目前高就調", 12, 0, 18, 1, 18, 1, true},
		{"21 不比百分位", 18, 100, 21, 0, 21, 0, true},
	} {
		gotVal, gotPct, gotRaised := RaiseStrength(want.cur, want.curPct, want.val, want.pct)
		if gotVal != want.outVal || gotPct != want.outPct || gotRaised != want.raised {
			t.Errorf("%s：算出 %d/%02d raised=%v，應該是 %d/%02d raised=%v",
				want.name, gotVal, gotPct, gotRaised, want.outVal, want.outPct, want.raised)
		}
	}
}

// 力量術的骰子看目標的職業，超過 18 之後只有戰士拿得到百分位
//（overlay-22 `1F16h`）。
func TestStrengthSpellResultFollowsTheTargetClass(t *testing.T) {
	fighter := [ClassThac0ClassCount]uint8{}
	fighter[ClassSlotFighter] = 4
	// 最大骰 1d8 ＝ 8，力量 16 → 24，超出 6 點 → 百分位 60。
	value, percentile := StrengthSpellResult(fighter, 16, 0, maxRoller{})
	if value != 18 || percentile != 60 {
		t.Errorf("戰士應該是 18/60，算出 %d/%02d", value, percentile)
	}
	// 已經有百分位就疊上去，上限 100。
	if _, percentile := StrengthSpellResult(fighter, 16, 50, maxRoller{}); percentile != 100 {
		t.Errorf("疊到上限應該是 100，算出 %d", percentile)
	}
	magicUser := [ClassThac0ClassCount]uint8{}
	magicUser[ClassSlotMagicUser] = 4
	// 最大骰 1d4 ＝ 4，力量 16 → 20，超過 18 但不是戰士 → 18/00。
	if value, percentile := StrengthSpellResult(magicUser, 16, 0, maxRoller{}); value != 18 || percentile != 0 {
		t.Errorf("法師應該是 18/00，算出 %d/%02d", value, percentile)
	}
	// 沒超過 18 就照算，百分位不動。
	if value, percentile := StrengthSpellResult(magicUser, 10, 7, maxRoller{}); value != 14 || percentile != 7 {
		t.Errorf("沒超過 18 應該是 14/07，算出 %d/%02d", value, percentile)
	}
}
