package gamepack

import (
	"archive/zip"
	"fmt"
)

const monsterRecordSize = 285

// MonsterRecord preserves one Pool MON*CHA block without assigning semantics
// to fields whose original consumers have not yet been closed.
type MonsterRecord struct {
	ID   uint8
	Name string
	Raw  [monsterRecordSize]byte
}

type MonsterAttackDamage struct {
	Count uint8
	Sides uint8
	Bonus int8
}

// The accessors below expose only fields closed from Pool's own character
// sheet consumers. Keep Raw as the authority for every field not yet proven.
func (record MonsterRecord) MaxHitPoints() uint8     { return record.Raw[0x32] }
func (record MonsterRecord) CurrentHitPoints() uint8 { return record.Raw[0x11B] }
func (record MonsterRecord) ArmorClass() int         { return 60 - int(record.Raw[0x111]) }
func (record MonsterRecord) THAC0() int              { return 60 - int(record.Raw[0x110]) }
func (record MonsterRecord) DamageDiceCount() uint8  { return record.Raw[0x115] }
func (record MonsterRecord) DamageDieSides() uint8   { return record.Raw[0x117] }
func (record MonsterRecord) DamageBonus() int8       { return int8(record.Raw[0x119]) }
func (record MonsterRecord) Movement() uint8         { return record.Raw[0x11C] }

// 經驗值（spec 097）。overlay-05 entry 2 的 `00C0h..00F4h` 算的是
// `+0B8h + +0BAh × +0B1h`：AD&D 一版的「基礎值加每點生命值的加成」。
//
// `+0B1h` 在活著的戰鬥員身上是牠擲出來的生命值；樣板記錄裡多半與 `+32h`
// 相同，但不是每一筆都填了（HOBGOBLIN 的是 0），所以算的時候要用實際的
// 生命值，不要讀樣板的那一格。
func (record MonsterRecord) ExperienceBase() uint16 {
	return uint16(record.Raw[0xB8]) | uint16(record.Raw[0xB9])<<8
}

// ExperiencePerHitPoint 是每點生命值再加多少。
func (record MonsterRecord) ExperiencePerHitPoint() uint8 { return record.Raw[0xBA] }

// ExperienceValue 是打倒這一隻值多少經驗值。
func (record MonsterRecord) ExperienceValue(hitPoints int) uint32 {
	if hitPoints < 0 {
		hitPoints = 0
	}
	return uint32(record.ExperienceBase()) + uint32(record.ExperiencePerHitPoint())*uint32(hitPoints)
}

func (record MonsterRecord) BaseAttackRate(slot uint8) (uint8, error) {
	if slot < 1 || slot > 2 {
		return 0, fmt.Errorf("Pool monster attack slot %d is outside 1..2", slot)
	}
	return record.Raw[0xA0+slot], nil
}

func (record MonsterRecord) AttackDamage(slot uint8) (MonsterAttackDamage, error) {
	if slot < 1 || slot > 2 {
		return MonsterAttackDamage{}, fmt.Errorf("Pool monster attack slot %d is outside 1..2", slot)
	}
	index := int(slot)
	return MonsterAttackDamage{
		Count: record.Raw[0x114+index],
		Sides: record.Raw[0x116+index],
		Bonus: int8(record.Raw[0x118+index]),
	}, nil
}

// ReadDOSMonsterRecord resolves one ECL LOAD MONSTER ID against the matching
// MONnCHA archive. The ECL icon block remains a separate descriptor field.
func ReadDOSMonsterRecord(zipPath string, archiveNumber, blockID uint8) (MonsterRecord, error) {
	if archiveNumber < 1 || archiveNumber > 8 {
		return MonsterRecord{}, fmt.Errorf("Pool monster archive %d is outside 1..8", archiveNumber)
	}
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return MonsterRecord{}, fmt.Errorf("open DOS ZIP: %w", err)
	}
	defer zr.Close()

	name := fmt.Sprintf("MON%dCHA.DAX", archiveNumber)
	member, err := uniqueMember(zr.File, name)
	if err != nil {
		return MonsterRecord{}, err
	}
	blocks, err := readDAXBlocks(member)
	if err != nil {
		return MonsterRecord{}, fmt.Errorf("%s: %w", name, err)
	}
	payload, ok := blocks[blockID]
	if !ok {
		return MonsterRecord{}, fmt.Errorf("%s has no block %d", name, blockID)
	}
	return parseMonsterRecord(blockID, payload)
}

func parseMonsterRecord(blockID uint8, payload []byte) (MonsterRecord, error) {
	if len(payload) != monsterRecordSize {
		return MonsterRecord{}, fmt.Errorf("Pool monster block %d has %d bytes, want %d", blockID, len(payload), monsterRecordSize)
	}
	nameLength := int(payload[0])
	if nameLength < 1 || nameLength > 15 || 1+nameLength > len(payload) {
		return MonsterRecord{}, fmt.Errorf("Pool monster block %d has invalid name length %d", blockID, nameLength)
	}
	record := MonsterRecord{ID: blockID, Name: string(payload[1 : 1+nameLength])}
	copy(record.Raw[:], payload)
	return record, nil
}
