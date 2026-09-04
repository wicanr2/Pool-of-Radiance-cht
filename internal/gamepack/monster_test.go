package gamepack

import (
	"path/filepath"
	"testing"
)

func TestParseMonsterRecordPreservesUnknownBytes(t *testing.T) {
	payload := make([]byte, monsterRecordSize)
	payload[0] = 3
	copy(payload[1:], "ORC")
	payload[284] = 0xA5
	record, err := parseMonsterRecord(13, payload)
	if err != nil {
		t.Fatal(err)
	}
	if record.ID != 13 || record.Name != "ORC" || record.Raw[284] != 0xA5 {
		t.Fatalf("record=%+v raw tail=%02X", record, record.Raw[284])
	}
}

func TestMonsterCombatAccessorsUsePoolRecordEncoding(t *testing.T) {
	payload := make([]byte, monsterRecordSize)
	payload[0] = 3
	copy(payload[1:], "ORC")
	payload[0x32] = 7
	payload[0x11B] = 5
	payload[0x111] = 54
	payload[0x110] = 41
	payload[0x11C] = 9
	// 傷害骰讀的是**來源欄位**，不是 `+114h..` 那一段執行期副本（spec 051）：
	// 形態 1 是 2d4-1、形態 2 是 3d6+2，攻擊次數編碼 2 與 1。
	payload[0xA1] = 2
	payload[0xA2] = 1
	payload[0xA3] = 2
	payload[0xA5] = 4
	payload[0xA7] = 0xFF
	payload[0xA4] = 3
	payload[0xA6] = 6
	payload[0xA8] = 2
	record, err := parseMonsterRecord(13, payload)
	if err != nil {
		t.Fatal(err)
	}
	if record.MaxHitPoints() != 7 || record.CurrentHitPoints() != 5 {
		t.Fatalf("HP=%d/%d", record.CurrentHitPoints(), record.MaxHitPoints())
	}
	if record.ArmorClass() != 6 || record.THAC0() != 19 {
		t.Fatalf("AC=%d THAC0=%d", record.ArmorClass(), record.THAC0())
	}
	if record.DamageDiceCount() != 2 || record.DamageDieSides() != 4 || record.DamageBonus() != -1 {
		t.Fatalf("damage=%dd%d%+d", record.DamageDiceCount(), record.DamageDieSides(), record.DamageBonus())
	}
	if record.Movement() != 9 {
		t.Fatalf("movement=%d", record.Movement())
	}
	for slot, want := range map[uint8]uint8{1: 2, 2: 1} {
		got, err := record.BaseAttackRate(slot)
		if err != nil || got != want {
			t.Fatalf("slot %d base rate=%d error=%v want %d", slot, got, err, want)
		}
	}
	second, err := record.AttackDamage(2)
	if err != nil || second != (MonsterAttackDamage{Count: 3, Sides: 6, Bonus: 2}) {
		t.Fatalf("slot 2 damage=%+v error=%v", second, err)
	}
	if _, err := record.BaseAttackRate(0); err == nil {
		t.Fatal("attack slot zero was accepted")
	}
	if _, err := record.AttackDamage(3); err == nil {
		t.Fatal("attack slot three was accepted")
	}
}

func TestParseMonsterRecordRejectsMalformedShapeAndName(t *testing.T) {
	if _, err := parseMonsterRecord(1, make([]byte, monsterRecordSize-1)); err == nil {
		t.Fatal("short monster record was accepted")
	}
	for _, length := range []byte{0, 16} {
		payload := make([]byte, monsterRecordSize)
		payload[0] = length
		if _, err := parseMonsterRecord(1, payload); err == nil {
			t.Fatalf("monster name length %d was accepted", length)
		}
	}
}

func TestRealMON2CHASlumsOrcs(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	for _, id := range []uint8{13, 4} {
		record, err := ReadDOSMonsterRecord(zipPath, 2, id)
		if err != nil {
			t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
		}
		if record.ID != id || record.Name != "ORC" || len(record.Raw) != monsterRecordSize {
			t.Fatalf("MON2CHA block %d=%+v", id, record)
		}
		if record.MaxHitPoints() != 5 || record.CurrentHitPoints() != 5 || record.ArmorClass() != 6 || record.Movement() != 9 {
			t.Fatalf("MON2CHA block %d combat fields: HP=%d/%d AC=%d movement=%d", id, record.CurrentHitPoints(), record.MaxHitPoints(), record.ArmorClass(), record.Movement())
		}
	}
	// 兩筆 ORC 的傷害骰都是 1d8。block 13 的**執行期區段**寫著 2d4−1，那是
	// 樣板檔裡沒有初始化的殘留（spec 051）；來源欄位 `+0A3h/+0A5h/+0A7h`
	// 兩筆一致，與 AD&D 一版的獸人相同。
	named, _ := ReadDOSMonsterRecord(zipPath, 2, 13)
	if named.THAC0() != 20 || named.DamageDiceCount() != 1 || named.DamageDieSides() != 8 || named.DamageBonus() != 0 {
		t.Fatalf("MON2CHA block 13: THAC0=%d damage=%dd%d%+d", named.THAC0(), named.DamageDiceCount(), named.DamageDieSides(), named.DamageBonus())
	}
	normal, _ := ReadDOSMonsterRecord(zipPath, 2, 4)
	if normal.THAC0() != 19 || normal.DamageDiceCount() != 1 || normal.DamageDieSides() != 8 || normal.DamageBonus() != 0 {
		t.Fatalf("MON2CHA block 4: THAC0=%d damage=%dd%d%+d", normal.THAC0(), normal.DamageDiceCount(), normal.DamageDieSides(), normal.DamageBonus())
	}
	for _, record := range []MonsterRecord{named, normal} {
		primary, err := record.BaseAttackRate(1)
		if err != nil || primary != 2 {
			t.Fatalf("MON2CHA block %d primary base rate=%d error=%v", record.ID, primary, err)
		}
		secondary, err := record.BaseAttackRate(2)
		if err != nil || secondary != 0 {
			t.Fatalf("MON2CHA block %d secondary base rate=%d error=%v", record.ID, secondary, err)
		}
	}
}

// 經驗值是「基礎值加每點生命值的加成」，與 AD&D 一版逐筆相同。
func TestMonsterExperienceMatchesTheOriginal(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	for _, want := range []struct {
		block     uint8
		name      string
		base      uint16
		perHP     uint8
		hitPoints int
		value     uint32
	}{
		{0, "KOBOLD", 5, 1, 3, 8},
		{4, "ORC", 10, 1, 5, 15},
		{8, "OGRE", 90, 5, 21, 195},
		{17, "SPECTRE", 1650, 10, 38, 2030},
	} {
		record, err := ReadDOSMonsterRecord(zipPath, 2, want.block)
		if err != nil {
			t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
		}
		if record.Name != want.name {
			t.Fatalf("block %d 是 %q，預期 %q", want.block, record.Name, want.name)
		}
		if record.ExperienceBase() != want.base || record.ExperiencePerHitPoint() != want.perHP {
			t.Errorf("%s 的經驗值欄位是 %d 加每點 %d，原版是 %d 加每點 %d",
				want.name, record.ExperienceBase(), record.ExperiencePerHitPoint(), want.base, want.perHP)
		}
		if got := record.ExperienceValue(want.hitPoints); got != want.value {
			t.Errorf("%s 有 %d 點生命值應該值 %d，算出 %d",
				want.name, want.hitPoints, want.value, got)
		}
	}
}

// 六份只填第二格傷害骰的記錄。只讀第一格的話牠們每一擊都是 0 點，
// 那一場架就永遠打不完（實測毒蛙對不還手的隊伍打了 10792 次沒扣到血）。
func TestMonstersWithOnlyTheSecondDamageSlot(t *testing.T) {
	for _, want := range []struct {
		archive, block uint8
		name           string
		count, sides   uint8
	}{
		{4, 38, "POISONOUS FROG", 1, 1},
		{5, 49, "MEDUSA", 1, 4},
		{5, 60, "GIANT SNAKE", 3, 6},
		{6, 60, "GIANT SNAKE", 3, 6},
		{7, 69, "DRIDER", 1, 4},
		{8, 116, "PHASE SPIDER", 1, 6},
	} {
		record, err := ReadDOSMonsterRecord(poolZipPath(), want.archive, want.block)
		if err != nil {
			t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
		}
		if record.Name != want.name {
			t.Errorf("MON%d/%d 是 %q，預期 %q", want.archive, want.block, record.Name, want.name)
			continue
		}
		if record.Raw[0x115] != 0 || record.Raw[0x117] != 0 {
			t.Errorf("%s 的第一格不是空的：%d／%d", want.name, record.Raw[0x115], record.Raw[0x117])
		}
		if record.DamageDiceCount() != want.count || record.DamageDieSides() != want.sides {
			t.Errorf("%s 的傷害骰是 %dd%d，預期 %dd%d", want.name,
				record.DamageDiceCount(), record.DamageDieSides(), want.count, want.sides)
		}
	}
}

// 反過來釘住：填了第一格的記錄不受退路影響。
func TestMonstersWithTheFirstDamageSlotAreUnchanged(t *testing.T) {
	record, err := ReadDOSMonsterRecord(poolZipPath(), 1, 2)
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	if record.Name != "GOBLIN GUARD" || record.DamageDiceCount() != 1 || record.DamageDieSides() != 6 {
		t.Fatalf("%q 的傷害骰是 %dd%d，預期 GOBLIN GUARD 1d6",
			record.Name, record.DamageDiceCount(), record.DamageDieSides())
	}
}
