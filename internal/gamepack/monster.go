package gamepack

import (
	"archive/zip"
	"fmt"
)

const monsterRecordSize = 285

// MonsterRecord preserves one Pool MON*CHA block without assigning semantics
// to fields whose original consumers have not yet been closed.
// 285-byte 記錄裡兩個決定「這一隻是不是人」的欄位。
const (
	MonsterBodySizeOffset     = 0x6C
	MonsterCreatureTypeOffset = 0x9F
)

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
// CreatureType 是記錄 `+9Fh`：這一隻算哪一族。四個互相獨立的使用點把語意
// 釘住——死靈術（overlay-22 `2090h`）只對 `0`、迷蛇術（`1927h`）只對 `0Eh`、
// overlay-12 `015Dh` 對 `4` 另外設旗標、魅惑人類與定身術（`11DAh`／`174Bh`）
// 要求不大於 `1`。原版資料裡量得到的值：`0` 人類、`1` 類人（哥布林、獸人、
// 狗頭人、熊地精）、`2` 巨人、`4` 不死、`7`、`0Ah` 巨魔、`0Bh`、`0Ch`、
// `0Eh` 蛇與蠍、`11h`。
func (record MonsterRecord) CreatureType() uint8 { return record.Raw[MonsterCreatureTypeOffset] }

// BodySize 是記錄 `+6Ch`：低位是體型（`1` 與人同大、`2` 大型、`3` 巨大），
// 位元 7 另外標著一批大塊頭（熊地精、食人魔、巨魔、巨人、牛頭人、巨蛇、
// 巨蜥）。魅惑人類與定身術要求整個 byte 不大於 1，所以只有「正好是 1」的
// 才算得上「人」。
func (record MonsterRecord) BodySize() uint8 { return record.Raw[MonsterBodySizeOffset] }

func (record MonsterRecord) MaxHitPoints() uint8     { return record.Raw[0x32] }
func (record MonsterRecord) CurrentHitPoints() uint8 { return record.Raw[0x11B] }
func (record MonsterRecord) ArmorClass() int         { return 60 - int(record.Raw[0x111]) }
func (record MonsterRecord) THAC0() int              { return 60 - int(record.Raw[0x110]) }
// 傷害骰有**兩格**：`+115h`／`+117h` 是第一格，`+116h`／`+118h` 是第二格。
// 兩格都存在是量出來的——overlay-13 有兩處寫入，`3AE5h` 寫第一格的顆數 3、
// `3AEEh` 寫第一格的面數 4，`39E3h`／`39ECh` 寫的是第二格的 2 與 4。
//
// 168 份記錄裡有 162 份填第一格。**剩下六份只填第二格**，而且都是帶特殊攻擊
// 的那幾隻：POISONOUS FROG、MEDUSA、GIANT SNAKE（兩份）、DRIDER、
// PHASE SPIDER。只讀第一格的話牠們每一擊都是 0 點——實測索寇要塞那一場
// 四隻毒蛙對完全不還手的隊伍打了 10792 次、一次都沒扣到血，那一場永遠打不完。
//
// **第一格空的就退到第二格。** 填了第一格的那 162 份第二格都是 0，所以這個
// 退路不會動到牠們。**原版依什麼挑格還沒讀出來**（`0119h` 的加值只在第一格
// 有意義，而角色表的顯示常式 overlay-19 `06F0h` 讀的是第一格），
// 所以這是推論，不是原版規則的重現。
func (record MonsterRecord) DamageDiceCount() uint8 {
	if record.Raw[0x115] == 0 && record.Raw[0x117] == 0 {
		return record.Raw[0x116]
	}
	return record.Raw[0x115]
}

func (record MonsterRecord) DamageDieSides() uint8 {
	if record.Raw[0x115] == 0 && record.Raw[0x117] == 0 {
		return record.Raw[0x118]
	}
	return record.Raw[0x117]
}
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
