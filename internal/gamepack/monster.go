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
// 傷害骰有**兩種攻擊形態**，每一種各有顆數、面數與加值（spec 051）：
//
//	顆數 `+0A2h + n`   面數 `+0A4h + n`   加值 `+0A6h + n`   （n = 1 或 2）
//
// 攻擊次數在 `+0A0h + n`，編碼是「每回合次數 × 2」，所以 2 是一次、
// 3 是每兩回合三次、4 是兩次。巨魔的 `04 02` 與 `1d4+4`／`2d6` 正是
// AD&D 一版的爪／爪／咬。
//
// **不要讀 `+114h..+11Ah`。** 那一段是原版**執行期**的副本：overlay-25
// `0DF4h` 在排怪時把上面三組欄位抄過去，武器與效果的覆寫也寫在那裡。
// `MONnCHA.DAX` 的樣板記錄裡那一段沒有初始化，殘留的是別的東西——
// 172 筆裡有 53 筆與來源欄位不符，而且殘留值看得出是文字：QUICKLINGS 是
// `41 00 44 00 43 00`（`'A' 'D' 'C'`），讀成骰子就是 65d68+67。
//
// 正負對照擺在一起就分得出來：**用來源欄位算，172 筆沒有一筆的骰子不合理；
// 用執行期那一段算，48 處不合理**（顆數大於 8、面數不是 D&D 的骰面、
// 或加值離譜）。`TestMonsterDamageDiceAreAllPlausible` 釘住這一條。
//
// 第一種形態沒有骰子的有六隻（POISONOUS FROG、MEDUSA、GIANT SNAKE 兩份、
// DRIDER、PHASE SPIDER），牠們的傷害在第二種形態上——第一種是特殊攻擊。
// 只讀第一種的話牠們每一擊都是 0 點：實測索寇要塞那一場四隻毒蛙對完全不
// 還手的隊伍打了 10792 次、一次都沒扣到血，那一場永遠打不完。
const (
	// MonsterAttackRateBase 是攻擊次數的基底，索引 `base + n`。
	MonsterAttackRateBase = 0xA0
	// MonsterDamageCountBase／SidesBase／BonusBase 是三組傷害欄位的基底。
	MonsterDamageCountBase = 0xA2
	MonsterDamageSidesBase = 0xA4
	MonsterDamageBonusBase = 0xA6
	// MonsterAttackSlots 是形態數。
	MonsterAttackSlots = 2
)

// primaryAttackSlot 是「畫面上要顯示哪一種形態」：第一種有骰子就用它，
// 沒有就用第二種。前端還沒把兩種形態都打出來時要用這個（spec 051 的 OPEN）。
func (record MonsterRecord) primaryAttackSlot() uint8 {
	if record.Raw[MonsterDamageCountBase+1] == 0 && record.Raw[MonsterDamageSidesBase+1] == 0 {
		return 2
	}
	return 1
}

func (record MonsterRecord) DamageDiceCount() uint8 {
	return record.Raw[MonsterDamageCountBase+int(record.primaryAttackSlot())]
}

func (record MonsterRecord) DamageDieSides() uint8 {
	return record.Raw[MonsterDamageSidesBase+int(record.primaryAttackSlot())]
}

func (record MonsterRecord) DamageBonus() int8 {
	return int8(record.Raw[MonsterDamageBonusBase+int(record.primaryAttackSlot())])
}

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
	if slot < 1 || slot > MonsterAttackSlots {
		return 0, fmt.Errorf("Pool monster attack slot %d is outside 1..%d", slot, MonsterAttackSlots)
	}
	return record.Raw[MonsterAttackRateBase+slot], nil
}

// AttackDamage 取一種攻擊形態的骰子。讀的是**來源欄位**，不是執行期副本；
// 理由見上面 `DamageDiceCount` 的說明。
func (record MonsterRecord) AttackDamage(slot uint8) (MonsterAttackDamage, error) {
	if slot < 1 || slot > MonsterAttackSlots {
		return MonsterAttackDamage{}, fmt.Errorf("Pool monster attack slot %d is outside 1..%d",
			slot, MonsterAttackSlots)
	}
	index := int(slot)
	return MonsterAttackDamage{
		Count: record.Raw[MonsterDamageCountBase+index],
		Sides: record.Raw[MonsterDamageSidesBase+index],
		Bonus: int8(record.Raw[MonsterDamageBonusBase+index]),
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
