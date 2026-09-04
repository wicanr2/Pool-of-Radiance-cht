package character

import (
	"encoding/binary"
	"fmt"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/creation"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 把 remake 的角色寫回原版的三個檔：`.sav`（285 bytes 的角色記錄）、
// `.itm`（63 的倍數，物品記錄）、`.spc`（9 的倍數，效果串列）。
// 三個一組是原版預設人物檔的形狀（spec 069）。
//
// **只寫有出處的欄位**，其餘位元組原封不動留著 base 的內容。原版的記錄有
// 285 bytes，remake 只解出其中一部分；把沒解出來的欄位歸零會產出一個
// 「載得進去、行為不對」的檔案，那比缺一個匯出功能糟得多。
//
// 目前**不寫、留給 base** 的已知欄位（都不是漏掉，是 remake 沒有產生端）：
//
//   - `+6Dh..+71h` 五個豁免目標值（spec 075）。remake 只會從記錄「讀」，
//     沒有從職業與等級「算」的那一支。
//   - `+72h` 移動、`+73h` 生命骰、`+76h` 驅散不死欄、`+A9h`／`+111h` 護甲、
//     `+101h` 豁免修正、`+102h` 負重：全部由裝備與等級推導，remake 現算現用，
//     沒有存進角色模型。
//   - `+77h` 起的賊技能：`poolsave.Character.ThiefSkills` 可能是空的，
//     空的時候寫 0 會把原本的技能抹掉，所以只在有值時才覆寫。
//   - `+BFh` 圖示的不透明旗標：WriteDOSIcon 也是刻意保留這一個。
const (
	// 錢是七個連續的 16-bit（spec 040，`+88h + 2×索引`）。索引順序與
	// internal/treasure 相同：銅、銀、琥珀金、金、白金、寶石、珠寶。
	offsetMoney = 0x88
	// MoneySlots 是那七欄。
	MoneySlots = 7

	offsetAge         = 0x30
	offsetAbilities   = 0x10
	offsetExceptional = 0x16
	offsetMaxHP       = 0x32
	offsetThiefSkills = 0x77
	offsetClassLevels = 0x96
	offsetExperience  = 0xAC
	offsetRawHP       = 0xB1
	offsetStatus      = 0x10C
	offsetCurrentHP   = 0x11B

	// AbilityCount 是 `+10h` 起的六個能力值。
	AbilityCount = 6
	// ClassLevelSlots 是 `+96h` 起的每職業等級陣列（spec 097）。
	ClassLevelSlots = 8
	// ThiefSkillSlots 是 `+77h` 起的八個賊技能百分比（spec 095）。
	ThiefSkillSlots = 8
	// NameLimit 是姓名的位元組上限；`+0` 是長度，`+1` 起是內容。
	NameLimit = 15

	// EffectNodeSize 是 `.spc` 的單位；`.itm` 的單位是 carry.go 的
	// ItemRecordSize。
	EffectNodeSize = 9
)

// ExportDOSRecord 把一個 remake 角色寫進 285 bytes 的記錄。
// base 通常是這個人原本的記錄；沒有的話傳一份 285 bytes 的零值。
func ExportDOSRecord(base []byte, character poolsave.Character) ([]byte, error) {
	if len(base) != DOSRecordSize {
		return nil, fmt.Errorf("Pool DOS CHA base length %d, want %d", len(base), DOSRecordSize)
	}
	if len(character.Name) > NameLimit {
		return nil, fmt.Errorf("Pool character name %q is %d bytes, the record holds %d",
			character.Name, len(character.Name), NameLimit)
	}
	race, ok := creation.RaceDOSCode(character.RaceID)
	if !ok {
		return nil, fmt.Errorf("Pool race %q has no DOS code", character.RaceID)
	}
	class, ok := creation.ClassDOSCode(character.ClassID)
	if !ok {
		return nil, fmt.Errorf("Pool class %q has no DOS code", character.ClassID)
	}
	gender, ok := creation.GenderDOSCode(character.GenderID)
	if !ok {
		return nil, fmt.Errorf("Pool gender %q has no DOS code", character.GenderID)
	}
	if len(character.ClassLevels) > ClassLevelSlots {
		return nil, fmt.Errorf("Pool character %q has %d class levels, the record holds %d",
			character.Name, len(character.ClassLevels), ClassLevelSlots)
	}
	if len(character.ThiefSkills) > ThiefSkillSlots {
		return nil, fmt.Errorf("Pool character %q has %d thief skills, the record holds %d",
			character.Name, len(character.ThiefSkills), ThiefSkillSlots)
	}

	record := append([]byte(nil), base...)
	record[0] = byte(len(character.Name))
	for index := 0; index < NameLimit; index++ {
		record[1+index] = 0
	}
	copy(record[1:], character.Name)

	for index, value := range character.Abilities {
		record[offsetAbilities+index] = clampByte(value)
	}
	record[offsetExceptional] = clampByte(character.ExceptionalStrength)
	record[offsetRace] = race
	record[offsetClass] = class
	record[offsetGender] = gender
	record[offsetAge] = clampByte(character.Age)
	record[offsetMaxHP] = clampByte(character.MaxHP)
	record[offsetRawHP] = clampByte(character.RawHP)
	record[offsetCurrentHP] = clampByte(character.CurrentHP)
	record[offsetStatus] = character.Status
	for slot := 0; slot < MoneySlots; slot++ {
		binary.LittleEndian.PutUint16(record[offsetMoney+slot*2:], character.Money[slot])
	}
	// 等級陣列整段重寫：`ClassLevels` 是空的代表「每個組成職業都是第 1 級」
	// （spec 097 的讀取端就這樣解），所以要照那個語意補，不能留 base 的舊值。
	levels, err := classLevelSlots(character)
	if err != nil {
		return nil, err
	}
	copy(record[offsetClassLevels:], levels[:])
	// 賊技能是空的時候不動；那代表 remake 沒有這一份資料，不是「全部 0」。
	copy(record[offsetThiefSkills:], character.ThiefSkills)
	binary.LittleEndian.PutUint32(record[offsetExperience:], character.Experience)

	record[offsetPortraitHead] = character.PortraitHead
	record[offsetPortraitBody] = character.PortraitBody
	record[offsetIconHead] = character.IconHead
	record[offsetIconWeapon] = character.IconWeapon
	record[offsetIconSize] = character.IconSize
	for index, offset := range iconColorOffsets() {
		if index >= len(character.IconColors) {
			break
		}
		record[offset] = packColor(DualColor{
			Color1: character.IconColors[index][0],
			Color2: character.IconColors[index][1],
		})
	}
	return record, nil
}

// classLevelSlots 把 remake 的等級展開成記錄 `+96h` 起的八格。
// 索引是**單一職業的碼**（ComponentClassIndex），不是組合職業的碼。
func classLevelSlots(character poolsave.Character) ([ClassLevelSlots]uint8, error) {
	var levels [ClassLevelSlots]uint8
	if len(character.ClassLevels) > 0 {
		copy(levels[:], character.ClassLevels)
		return levels, nil
	}
	components, ok := creation.ClassComponents(character.ClassID)
	if !ok {
		return levels, fmt.Errorf("Pool class %q has no component list", character.ClassID)
	}
	for _, component := range components {
		index, ok := creation.ComponentClassIndex(component)
		if !ok || int(index) >= ClassLevelSlots {
			return levels, fmt.Errorf("Pool class component %q has no record slot", component)
		}
		levels[index] = 1
	}
	return levels, nil
}

// ExportDOSItems 把背包寫成 `.itm`：一件 63 bytes，順序照背包。
// 每一件直接抄原始記錄——remake 沒有重建那 63 bytes 的能力，抄不到的
//（`Raw` 不是 63 bytes）就是錯誤，不補零蒙混過去。
func ExportDOSItems(items []poolsave.Item) ([]byte, error) {
	out := make([]byte, 0, len(items)*ItemRecordSize)
	for index, item := range items {
		if len(item.Raw) != ItemRecordSize {
			return nil, fmt.Errorf("Pool item %d (%q) has %d raw bytes, want %d",
				index, item.Name, len(item.Raw), ItemRecordSize)
		}
		out = append(out, item.Raw...)
	}
	return out, nil
}

// ExportDOSEffects 把效果碼寫成 `.spc`：一個節點 9 bytes（spec 069）。
//
// 節點的 `+5..+8` 是上次執行時的遠指標，重新載入沒有意義，走訪照檔案順序，
// 所以這裡一律寫 0。`+1`..`+4`（持續、下效果者等級、收尾旗標）在 spec 069
// 仍是 DRAFT，而且 remake 的存檔模型沒有存它們——寫 0 是**已知的缺口**，
// 不是查證過的預設值。
func ExportDOSEffects(codes []uint8) []byte {
	out := make([]byte, len(codes)*EffectNodeSize)
	for index, code := range codes {
		out[index*EffectNodeSize] = code
	}
	return out
}

func iconColorOffsets() []int {
	return []int{offsetColorBody, offsetColorArm, offsetColorLeg,
		offsetColorFace, offsetColorShield, offsetColorWeapon}
}

func clampByte(value int) byte {
	switch {
	case value < 0:
		return 0
	case value > 0xFF:
		return 0xFF
	}
	return byte(value)
}

// 建角寫下的基礎值（spec 063「建角寫下的基礎值」那一節，overlay-16 `0570h`
// 與 `1C0Bh`）。remake 自己建的角色沒有原版記錄可以當 base，這些常數就是
// 那份 base 的內容——不是預設值，是原版逐位元組寫下的。
const (
	// BaseArmourClassOffset 是基礎 AC internal，檯面上是 60 − 50 = 10。
	BaseArmourClassOffset = 0xA9
	// BaseArmourClassValue 是它的值（`32h`）。
	BaseArmourClassValue = 0x32
	// BaseThac0Offset／BaseThac0Value 是基礎 THAC0 internal，檯面 20。
	// 之後由職業等級表重算（gamepack.RecomputeCombatFields）。
	BaseThac0Offset = 0x2D
	BaseThac0Value  = 0x28
	// BaseMovementOffset／BaseMovementValue 是基礎移動力 12。
	BaseMovementOffset = 0x72
	BaseMovementValue  = 0x0C
	// PresenceOffset 是參戰旗標；spec 052 的先攻與 spec 059 的反應攻擊都
	// 以它非 0 為前提。
	PresenceOffset = 0x10D
	// AbilityBonusFlagOffset 是 `+0AAh`：非零才套用兩個力量修正。
	AbilityBonusFlagOffset = 0xAA

	// unnamed6C 是 spec 063 列出、但原版證據還沒替它命名的三個 1 之一
	//（另外兩個是 `+0BBh`／`+0BCh`，那兩格由肖像覆寫）。
	unnamed6C = 0x6C
)

// NewDOSRecordBase 產生一份「剛建好、還沒填任何選擇」的 285-byte 記錄，
// 給 ExportDOSRecord 當 base 用。
//
// 沒有寫的兩個已知欄位：`+0ABh` 是 `random(100h)`（原版每次建角擲一次，
// 語意未閉合，這裡不假造一個定值），`+6Dh..+71h` 的豁免表與 `+73h` 的生命骰
// remake 還沒有產生端。
func NewDOSRecordBase() []byte {
	record := make([]byte, DOSRecordSize)
	record[BaseArmourClassOffset] = BaseArmourClassValue
	record[BaseThac0Offset] = BaseThac0Value
	record[BaseMovementOffset] = BaseMovementValue
	record[PresenceOffset] = 1
	record[unnamed6C] = 1
	// `+0AAh` = 1 是**建角自己寫的**（overlay-16 `1C02h` 是全遊戲唯一一處
	// 寫它的指令，spec 065）。怪物那一側是資料帶的，172 筆記錄裡 123 筆是 0，
	// 所以它不是常數——但玩家角色一律是 1。
	record[AbilityBonusFlagOffset] = 1
	return record
}
