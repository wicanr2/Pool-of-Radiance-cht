package gamepack

import "fmt"

// 訓練所的昇級（spec 097）。原版把它拆成四張靜態表加三段算式，
// 這裡照那個形狀接：表由 START.EXE 解出來，算式逐行對應。
const (
	// ExperienceRecordOffset 是累積經驗值在角色記錄裡的位置，32 bit，
	// 低位字在 `+0ACh`、高位字在 `+0AEh`。
	ExperienceRecordOffset = 0xac
	// ClassCategoryTableOffset 是職業分類遮罩表的 DS 位移。
	ClassCategoryTableOffset = 0x5f2
	// HitDiceCountTableOffset 是每個職業擲幾顆骰子。
	HitDiceCountTableOffset = 0x8ca
	// HitDiceSidesTableOffset 是每顆骰子幾面。
	HitDiceSidesTableOffset = 0x8d2
	// ConstitutionBonusTableOffset 是體質加成表，索引就是體質值。
	ConstitutionBonusTableOffset = 0x4006
	// ConstitutionBonusTableLength 是表裡有意義的長度；體質最高 18，
	// 表到 20 都還是有效值，再往後是別的資料。
	ConstitutionBonusTableLength = 21

	// ConstitutionOffset 是體質在角色記錄裡的位置。
	ConstitutionOffset = 0x14
	// ClassCodeOffset 是複合職業碼 `+2Fh`。體質的額外加成比的是它，
	// 不是單一職業的索引。
	ClassCodeOffset = 0x2f
	// PureFighterClassCode 是「純戰士」的複合職業碼。
	PureFighterClassCode = 2
	// HitPointsWithoutConstitutionOffset 是 `+0B1h`：不含體質加成的那一份，
	// 能量吸取還帳時用它。
	HitPointsWithoutConstitutionOffset = 0xb1
	// DrainedLevelsOffset 是 `+74h`：被能量吸取欠著的等級數。
	DrainedLevelsOffset = 0x74
	// DrainedHitPointsOffset 是 `+75h`：欠著的 HP。
	DrainedHitPointsOffset = 0x75

	// classCategorySpell 是分類位元：施法。
	classCategorySpell = 0x01
	// classCategoryPriest 是分類位元：神職。
	classCategoryPriest = 0x02
	// classCategoryThief 是分類位元：賊。
	classCategoryThief = 0x04
	// classCategoryFighter 是分類位元：戰士。
	classCategoryFighter = 0x08

	// firstLevelHitDieNumerator／Denominator 是第一級的地板：面數 × 2 ÷ 3。
	firstLevelHitDieNumerator   = 2
	firstLevelHitDieDenominator = 3
)

// LevelUpTables 是昇級要用的四張表，全部從 START.EXE 的資料段解出來。
type LevelUpTables struct {
	// ClassCategory 是每個職業的分類遮罩。
	ClassCategory [ClassThac0ClassCount]uint8
	// HitDiceCount 是每個職業擲幾顆。
	HitDiceCount [ClassThac0ClassCount]uint8
	// HitDiceSides 是每顆幾面。
	HitDiceSides [ClassThac0ClassCount]uint8
	// ConstitutionBonus 以體質值為索引；負值以有號 byte 存，這裡轉成 int8。
	ConstitutionBonus [ConstitutionBonusTableLength]int8
}

// ParseLevelUpTables 從 START.EXE 的完整內容解出四張表。
func ParseLevelUpTables(executable []byte) (LevelUpTables, error) {
	var tables LevelUpTables
	read := func(ds, length int) ([]byte, error) {
		start := ds + startDataSegmentFileDelta
		if start < 0 || start+length > len(executable) {
			return nil, fmt.Errorf("Pool START.EXE is %d bytes, DS:%04Xh needs %d more",
				len(executable), ds, start+length-len(executable))
		}
		return executable[start : start+length], nil
	}
	category, err := read(ClassCategoryTableOffset, ClassThac0ClassCount)
	if err != nil {
		return tables, err
	}
	count, err := read(HitDiceCountTableOffset, ClassThac0ClassCount)
	if err != nil {
		return tables, err
	}
	sides, err := read(HitDiceSidesTableOffset, ClassThac0ClassCount)
	if err != nil {
		return tables, err
	}
	bonus, err := read(ConstitutionBonusTableOffset, ConstitutionBonusTableLength)
	if err != nil {
		return tables, err
	}
	copy(tables.ClassCategory[:], category)
	copy(tables.HitDiceCount[:], count)
	copy(tables.HitDiceSides[:], sides)
	for index, value := range bonus {
		tables.ConstitutionBonus[index] = int8(value)
	}
	return tables, nil
}

// ReadDOSLevelUpTables 從遊戲壓縮檔取出四張表。
func ReadDOSLevelUpTables(zipPath string) (LevelUpTables, error) {
	executable, err := readStartExecutable(zipPath)
	if err != nil {
		return LevelUpTables{}, err
	}
	return ParseLevelUpTables(executable)
}

// ClassLevels 取出角色記錄裡八個單一職業的等級。
func ClassLevels(record []byte) ([ClassThac0ClassCount]uint8, error) {
	var levels [ClassThac0ClassCount]uint8
	if len(record) < ClassLevelsOffset+ClassThac0ClassCount {
		return levels, fmt.Errorf("Pool character record is %d bytes, the class levels need %d",
			len(record), ClassLevelsOffset+ClassThac0ClassCount)
	}
	copy(levels[:], record[ClassLevelsOffset:])
	return levels, nil
}

// LeveledClassCount 是有等級的職業數，也就是 HP 要除以的那個數。
func LeveledClassCount(levels [ClassThac0ClassCount]uint8) int {
	total := 0
	for _, level := range levels {
		if level > 0 {
			total++
		}
	}
	return total
}

// Roller 是擲骰。與 `0100h:0048h` 同簽章：擲 count 顆 sides 面。
type Roller interface {
	Roll(count, sides int) int
}

// HitDiceRoll 依「分類遮罩」對每個對得上的職業各擲一次，加總回傳（`4209h`）。
//
// 職業目前是第 1 級時擲出來會墊到 `面數 × 2 ÷ 3`。那只發生在建角寫下第一級的
// 那一次；訓練所是先加等級再擲，走不到這一段。
func (t LevelUpTables) HitDiceRoll(levels [ClassThac0ClassCount]uint8, category uint8, roller Roller) int {
	total := 0
	for class, level := range levels {
		if level == 0 || t.ClassCategory[class]&category == 0 {
			continue
		}
		sides := int(t.HitDiceSides[class])
		rolled := roller.Roll(int(t.HitDiceCount[class]), sides)
		if level == 1 {
			if floor := sides * firstLevelHitDieNumerator / firstLevelHitDieDenominator; rolled < floor {
				rolled = floor
			}
		}
		total += rolled
	}
	return total
}

// ConstitutionHitPointBonus 是 `3F01h`：對每個有等級的職業各加一份體質加成。
// 呼叫端會再除以職業數，所以淨結果是一份。
func (t LevelUpTables) ConstitutionHitPointBonus(levels [ClassThac0ClassCount]uint8,
	constitution int, classCode uint8) int {
	if constitution < 0 || constitution >= ConstitutionBonusTableLength {
		return 0
	}
	perClass := int(t.ConstitutionBonus[constitution])
	// 純戰士在體質 17、18 各再加一。比的是複合職業碼，所以戰士／賊拿不到。
	if constitution > 16 && classCode == PureFighterClassCode {
		perClass++
	}
	if constitution > 17 && classCode == PureFighterClassCode {
		perClass++
	}
	return perClass * LeveledClassCount(levels)
}

// LevelUpHitPoints 是一次昇級加的 HP（`2F80h..3027h`）。
//
// WithoutConstitution 收的是沒有體質加成的那一份，寫進 `+0B1h`；
// Total 是實際加到最大 HP 的那一份。兩者都至少是 1。
type LevelUpHitPoints struct {
	WithoutConstitution int
	Total               int
}

// LevelUpHitPointGain 依擲出來的點數與體質加成算出這一次要加多少 HP。
func LevelUpHitPointGain(rolled, constitutionBonus, leveledClasses int) LevelUpHitPoints {
	if leveledClasses <= 0 {
		return LevelUpHitPoints{WithoutConstitution: 1, Total: 1}
	}
	plain := rolled / leveledClasses
	if plain == 0 {
		plain = 1
	}
	total := (rolled + constitutionBonus) / leveledClasses
	if total < 1 {
		total = 1
	}
	return LevelUpHitPoints{WithoutConstitution: plain, Total: total}
}

// ApplyLevelUpHitPoints 把 HP 加上去，保留原本的受傷量（`2FE9h..3022h`）。
func ApplyLevelUpHitPoints(maxHitPoints, currentHitPoints, gain int) (int, int) {
	missing := maxHitPoints - currentHitPoints
	maxHitPoints += gain
	return maxHitPoints, maxHitPoints - missing
}
