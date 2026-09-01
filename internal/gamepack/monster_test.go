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
	payload[0x115] = 2
	payload[0x117] = 4
	payload[0x119] = 0xFF
	payload[0x11C] = 9
	payload[0xA1] = 2
	payload[0xA2] = 1
	payload[0x116] = 3
	payload[0x118] = 6
	payload[0x11A] = 2
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
	named, _ := ReadDOSMonsterRecord(zipPath, 2, 13)
	if named.THAC0() != 20 || named.DamageDiceCount() != 2 || named.DamageDieSides() != 4 || named.DamageBonus() != -1 {
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
