package gamepack_test

import (
	"strings"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// 表有 68 筆，編號 0 那筆是走不到的哨兵。
func TestSpellParameterTableShape(t *testing.T) {
	table, err := gamepack.ReadDOSSpellParameters(dosZIP)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	if len(table) != gamepack.SpellParameterCount {
		t.Fatalf("parameter table has %d records, want %d", len(table), gamepack.SpellParameterCount)
	}
	for index, value := range table[0].Raw[:15] {
		if value != 0 {
			t.Fatalf("record 0 byte %d is %#02x, want zero", index, value)
		}
	}
}

// `+0` 為 FFh 的正好是五個「碰觸才生效」的法術。四個 Cause 系加上
// Shocking Grasp，一個不多一個不少——這五個在規則書裡都要先摸到對手。
func TestSpellParameterAttackRollMarksTheTouchSpells(t *testing.T) {
	table, err := gamepack.ReadDOSSpellParameters(dosZIP)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	names, err := gamepack.ReadDOSSpellNames(dosZIP)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	want := map[string]bool{
		"Cause Light Wounds": true,
		"Shocking Grasp":     true,
		"Cause Blindness":    true,
		"Cause Disease":      true,
		"Bestow Curse":       true,
	}
	got := make(map[string]bool, len(want))
	for _, record := range table {
		if !record.RequiresAttackRoll() {
			continue
		}
		if record.SpellID > len(names) {
			t.Fatalf("unnamed spell %d requires an attack roll", record.SpellID)
		}
		got[strings.TrimSpace(names[record.SpellID-1])] = true
	}
	if len(got) != len(want) {
		t.Fatalf("attack-roll spells are %v, want %v", got, want)
	}
	for name := range want {
		if !got[name] {
			t.Fatalf("%s does not require an attack roll", name)
		}
	}
}

// 效果碼為零的，正好是「打完就結束、不留狀態」的那一批：治療與致傷、
// 三個純傷害、Knock、解除魔法、解咒、復原。留下狀態的法術一個都不在裡面
// ——Cure Blindness 在（它是把狀態拿掉），Cause Blindness 不在（它留下失明）。
func TestSpellParameterZeroEffectCodeMeansNoLingeringCondition(t *testing.T) {
	table, err := gamepack.ReadDOSSpellParameters(dosZIP)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	names, err := gamepack.ReadDOSSpellNames(dosZIP)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	instantaneous := map[string]bool{
		"Cure Light Wounds": true, "Cause Light Wounds": true, "Burning Hands": true,
		"Magic Missile": true, "Shocking Grasp": true, "Knock": true,
		"Cure Blindness": true, "Cure Disease": true, "Dispel Magic": true,
		"Remove Curse": true, "Fireball": true, "Lightning Bolt": true,
		"Restoration": true,
	}
	for _, record := range table {
		if record.SpellID < 1 || record.SpellID > len(names) {
			continue
		}
		name := strings.TrimSpace(names[record.SpellID-1])
		if (record.EffectCode() == 0) != instantaneous[name] {
			t.Fatalf("%s has effect code %#02x but instantaneous=%v", name, record.EffectCode(), instantaneous[name])
		}
	}
}

// 兩張表要對得起來：完全同名的法術共用一支處理常式（spec 073），效果碼也
// 必須相同。一邊對、一邊不對，就表示其中一張表的索引讀錯了。
func TestSpellParametersAgreeWithTheDispatchTable(t *testing.T) {
	table, err := gamepack.ReadDOSSpellParameters(dosZIP)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	dispatch, err := gamepack.ReadDOSSpellDispatchTable(dosZIP)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	names, err := gamepack.ReadDOSSpellNames(dosZIP)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	handler := make(map[int]uint16, len(dispatch))
	for _, entry := range dispatch {
		handler[entry.SpellID] = entry.CodeOffset
	}
	byName := make(map[string][]int, len(names))
	for index, name := range names {
		key := strings.ToLower(strings.TrimSpace(name))
		byName[key] = append(byName[key], index+1)
	}
	pairs := 0
	for name, ids := range byName {
		if len(ids) < 2 {
			continue
		}
		pairs++
		for _, id := range ids[1:] {
			if table[id].EffectCode() != table[ids[0]].EffectCode() {
				t.Fatalf("%q is spell %v with effect codes %#02x and %#02x", name, ids, table[ids[0]].EffectCode(), table[id].EffectCode())
			}
			if handler[id] != handler[ids[0]] {
				t.Fatalf("%q is spell %v with handlers %#04x and %#04x", name, ids, handler[ids[0]], handler[id])
			}
		}
	}
	if pairs != 5 {
		t.Fatalf("the name table has %d repeated names, want 5", pairs)
	}
}

// 效果訊息是往回找出來的，所以要有一道獨立的核對：訊息必須和法術名對得上。
func TestSpellDispatchMessagesMatchTheirSpells(t *testing.T) {
	dispatch, err := gamepack.ReadDOSSpellDispatchTable(dosZIP)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	want := map[int]string{
		1: "is Blessed", 2: "is Cursed", 10: "is charmed", 21: "falls asleep",
		23: "is held", 30: "is invisible", 38: "is blind", 48: "is Hasted",
		55: "is Slowed", 61: "is paralyzed",
	}
	got := make(map[int]string, len(dispatch))
	messages := 0
	for _, entry := range dispatch {
		got[entry.SpellID] = entry.Message
		if entry.Message != "" {
			messages++
		}
	}
	for id, text := range want {
		if got[id] != text {
			t.Fatalf("spell %d says %q, want %q", id, got[id], text)
		}
	}
	// 41 支常式帶訊息；其中五支由多個編號共用（偵測系 5、防護系 7、定身 2、
	// 隱形 2、解除魔法 2），攤回編號就是 36 + 18 = 54 個。
	if messages != 54 {
		t.Fatalf("%d spells carry a message, want 54", messages)
	}
}

// `+0`／`+1` 與人工整理的法術目錄逐筆相符：56 個具名法術的職業與等級全中。
// 目錄是照說明書整理的，表是原版的位元組，兩邊獨立——對得起來才表示欄位
// 讀對了，而且順帶把目錄鎖在資料上。
func TestSpellParametersMatchTheCatalogueClassAndLevel(t *testing.T) {
	table, err := gamepack.ReadDOSSpellParameters(dosZIP)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	catalogue, err := gamepack.TraditionalChineseSpells()
	if err != nil {
		t.Fatal(err)
	}
	for id := 1; id <= gamepack.SpellNameCount; id++ {
		spell, err := catalogue.SpellByID(uint8(id))
		if err != nil {
			t.Fatalf("catalogue has no spell %d: %v", id, err)
		}
		want := gamepack.SpellSourceCleric
		if spell.Class == gamepack.SpellClassMagicUser {
			want = gamepack.SpellSourceMagicUser
		}
		if table[id].Source() != want {
			t.Fatalf("spell %d (%s) is %v in the table but %s in the catalogue", id, spell.Name, table[id].Source(), spell.Class)
		}
		if table[id].Level() != spell.Level {
			t.Fatalf("spell %d (%s) is level %d in the table but %d in the catalogue", id, spell.Name, table[id].Level(), spell.Level)
		}
	}
	// 57..67 沒有名字，全部是物品效果。
	for id := gamepack.SpellNameCount + 1; id <= gamepack.SpellDispatchCount; id++ {
		if table[id].Source() != gamepack.SpellSourceItem {
			t.Fatalf("unnamed spell %d is %v, want an item effect", id, table[id].Source())
		}
	}
}

// 持續回合數對得上規則書：固定值加每級增量。取幾個原版與 AD&D 完全一致的
// 來鎖住兩個欄位的位置與先後。
func TestSpellParameterDurations(t *testing.T) {
	table, err := gamepack.ReadDOSSpellParameters(dosZIP)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	for _, item := range []struct {
		id, level, want int
		note            string
	}{
		{1, 5, 6, "Bless 固定 6 回合"},
		{6, 5, 15, "Protection From Evil 每級 3 回合"},
		{19, 4, 20, "Shield 每級 5 回合"},
		{22, 3, 30, "Find Traps 固定 3 turn"},
		{32, 6, 12, "Mirror Image 每級 2 回合"},
		{48, 5, 8, "Haste 3 加每級 1"},
		{30, 9, 0, "Invisibility 不自己結束"},
	} {
		if got := table[item.id].Duration(item.level); got != item.want {
			t.Fatalf("%s: spell %d at caster level %d lasts %d, want %d", item.note, item.id, item.level, got, item.want)
		}
	}
}

// 射程是 `+2 + +3 × 施法者等級`，算成 0 而 `+6` 非零就墊成 1。與規則書
// 逐條對得上：Bless 6 格、Detect Magic 3 格、Fireball 10 加每級 1、
// Lightning Bolt 4 加每級 1、碰觸的是 1 格。
func TestSpellParameterRanges(t *testing.T) {
	table, err := gamepack.ReadDOSSpellParameters(dosZIP)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	for _, item := range []struct {
		id, level, want int
		note            string
	}{
		{1, 6, 6, "Bless 固定 6 格"},
		{5, 6, 3, "Detect Magic 固定 3 格"},
		{3, 6, 1, "Cure Light Wounds 是碰觸"},
		{4, 6, 1, "Cause Light Wounds 的 FFh 也是碰觸"},
		{23, 6, 6, "Hold Person 固定 6 格"},
		{25, 6, 12, "Silence 15' Radius 固定 12 格"},
		{47, 1, 11, "Fireball 10 加每級 1"},
		{47, 6, 16, "Fireball 到 6 級是 16"},
		{51, 6, 10, "Lightning Bolt 4 加每級 1"},
	} {
		if got := table[item.id].Range(item.level); got != item.want {
			t.Fatalf("%s: spell %d at caster level %d reaches %d, want %d", item.note, item.id, item.level, got, item.want)
		}
	}
}


// 豁免成功之後的三種處置（overlay-24 `133Ah`）。少了這一條，火球術豁免成功
// 會照樣打滿，而定身術豁免成功也照樣定住。
func TestDamageAfterSaveFollowsTheOriginalRules(t *testing.T) {
	for _, testCase := range []struct {
		name   string
		rule   uint8
		damage int
		want   int
	}{
		{"規則 1 完全無效", gamepack.SaveRuleNegates, 21, 0},
		{"規則 2 減半（整數除法）", gamepack.SaveRuleHalves, 21, 10},
		{"規則 3 不動傷害", 3, 21, 21},
		{"規則 0 不該走到這裡，但也不動", gamepack.SaveRuleNone, 21, 21},
	} {
		if got := gamepack.DamageAfterSave(testCase.rule, testCase.damage); got != testCase.want {
			t.Errorf("%s：得到 %d，預期 %d", testCase.name, got, testCase.want)
		}
	}
}

// 表裡誰用哪一條規則。這一條是語意上的交叉核對：規則 2 剛好就是火球術與
// 閃電術這兩支「豁免減半」的傷害法術，規則 1 全是狀態類。
func TestSaveRuleAssignmentsInTheOriginalTable(t *testing.T) {
	parameters, err := gamepack.ReadDOSSpellParameters(dosZIP)
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	counts := map[uint8]int{}
	for id := 1; id < len(parameters); id++ {
		counts[parameters[id].SaveRule()]++
	}
	for rule, want := range map[uint8]int{0: 52, 1: 9, 2: 4, 3: 2} {
		if counts[rule] != want {
			t.Errorf("規則 %d 有 %d 支，預期 %d", rule, counts[rule], want)
		}
	}
	for _, id := range []int{gamepack.SpellIDFireball, gamepack.SpellIDLightningBolt} {
		if got := parameters[id].SaveRule(); got != gamepack.SaveRuleHalves {
			t.Errorf("法術 %d 的規則是 %d，預期 %d（豁免減半）", id, got, gamepack.SaveRuleHalves)
		}
	}
}
