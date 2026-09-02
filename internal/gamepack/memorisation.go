package gamepack

import "fmt"

// 記憶法術（spec 070／072／074）。
//
// 三張已經閉合的表湊起來就是規則：記憶陣列在記錄 `+1Fh`（13 格，1-based 編號）、
// 每級可記憶數在 `+0B2h`／`+0B5h`（牧師與法師各三格，含睿智加成）、
// 法術屬於哪一職業哪一級在參數表的 `+0` 與 `+1`。
//
// 原版的可記憶數是**上限**，不是遞減的剩餘量：overlay-15 只讀那六格，
// 沒有任何一處減它（唯一的 `dec +0B1h` 在 overlay-12，那是能量吸取扣 HP）。
// 所以「還能記幾個」要用「上限減掉已經記了幾個」算。
const (
	// SpellSlotRecordBase 是可記憶數的基底。索引方式是
	// `base + 職業組 × 3 + 法術等級`，法術等級 1 起算——所以牧師第 1 級
	// 落在 `+0B2h`、法師第 1 級落在 `+0B5h`，與 spec 072 相符。
	SpellSlotRecordBase = 0xb1
	// SpellSlotGroupStride 是一個職業組佔幾格。
	SpellSlotGroupStride = 3
	// SpellSlotGroupCleric 是牧師那一組。
	SpellSlotGroupCleric = 0
	// SpellSlotGroupMagicUser 是法師那一組。
	SpellSlotGroupMagicUser = 1
	// SpellSlotGroups 是有幾組。參數表 `+0` 的值 2（物品效果）不佔格子。
	SpellSlotGroups = 2
)

// SpellSlotCounts 是兩組各三級的數字。
type SpellSlotCounts [SpellSlotGroups][SpellSlotLevels]int

// spellSlotRecordIndex 是某一組某一級在記錄裡的位移。
func spellSlotRecordIndex(group, level int) int {
	return SpellSlotRecordBase + group*SpellSlotGroupStride + level
}

// RecordSpellSlotMaxima 讀出角色記錄裡的可記憶數上限。
func RecordSpellSlotMaxima(record []byte) (SpellSlotCounts, error) {
	var counts SpellSlotCounts
	last := spellSlotRecordIndex(SpellSlotGroups-1, SpellSlotLevels)
	if len(record) <= last {
		return counts, fmt.Errorf("Pool character record is %d bytes, the spell slots need %d",
			len(record), last+1)
	}
	for group := 0; group < SpellSlotGroups; group++ {
		for level := 1; level <= SpellSlotLevels; level++ {
			counts[group][level-1] = int(record[spellSlotRecordIndex(group, level)])
		}
	}
	return counts, nil
}

// MemorisedCounts 數出記憶陣列裡每一組每一級各記了幾個。
//
// 編號查參數表取職業組與法術等級；落在表外、屬於物品效果、或法術等級超出
// 三級的都不計——那些進不了記憶陣列。
func MemorisedCounts(memorised []uint8, parameters []SpellParameters) SpellSlotCounts {
	var counts SpellSlotCounts
	for _, value := range memorised {
		id := value & memorisedSpellIDMask
		if id == 0 || int(id) >= len(parameters) {
			continue
		}
		entry := parameters[id]
		group := int(entry.Source())
		level := entry.Level()
		if group >= SpellSlotGroups || level < 1 || level > SpellSlotLevels {
			continue
		}
		counts[group][level-1]++
	}
	return counts
}

// FreeSpellSlots 是「還能記幾個」：上限減掉已經記了幾個，不會低於零。
func FreeSpellSlots(maxima, used SpellSlotCounts) SpellSlotCounts {
	var free SpellSlotCounts
	for group := range maxima {
		for level := range maxima[group] {
			if remaining := maxima[group][level] - used[group][level]; remaining > 0 {
				free[group][level] = remaining
			}
		}
	}
	return free
}

// Memorise 把一個法術寫進記憶陣列的第一個空格。
//
// 記憶陣列傳進來會被就地修改。格子滿了或那一級沒有空位就回錯誤，
// 呼叫端要把它當成「這個選擇不合法」，不是當機。
func Memorise(memorised []uint8, id uint8, parameters []SpellParameters,
	maxima SpellSlotCounts) error {
	if id == 0 || int(id) >= len(parameters) {
		return fmt.Errorf("Pool spell %d is outside the parameter table", id)
	}
	entry := parameters[id]
	group, level := int(entry.Source()), entry.Level()
	if group >= SpellSlotGroups || level < 1 || level > SpellSlotLevels {
		return fmt.Errorf("Pool spell %d is not a memorisable class %d level %d spell",
			id, group, level)
	}
	free := FreeSpellSlots(maxima, MemorisedCounts(memorised, parameters))
	if free[group][level-1] <= 0 {
		return fmt.Errorf("Pool character has no free level %d slot for class %d", level, group)
	}
	for index := range memorised {
		if memorised[index] == 0 {
			memorised[index] = id
			return nil
		}
	}
	return fmt.Errorf("Pool memorised spell array is full")
}

// ForgetMemorised 清掉一格。原版清格子寫的是 0（overlay-14 `0698h`）。
func ForgetMemorised(memorised []uint8, slot int) error {
	if slot < 0 || slot >= len(memorised) {
		return fmt.Errorf("Pool memorised slot %d is outside 0..%d", slot, len(memorised)-1)
	}
	memorised[slot] = 0
	return nil
}
