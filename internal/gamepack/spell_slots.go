package gamepack

import "fmt"

// 每級可記憶幾個法術。overlay-23 是「依職業等級重算衍生數值」那一支：
// 它同時算基礎 THAC0（`3C16h` 的表，spec 063）、最高職業等級、戰士的攻擊次數，
// 以及這裡的法術格數。
//
// 取值是 `base + level × 3 + spellLevel`，spellLevel 是 1..3——**所以有效的
// 列基底比表頭多一個 byte**。先前用 AD&D 的列去搜檔案找不到，原因就在這裡：
// 表頭那六個 byte 是 `FF`（第 1 級不走這條路），而搜尋樣式從第 1 級的
// `1,0,0` 開始，那一列根本不在表裡。
const (
	// ClericSpellSlotTableOffset 是牧師表的 DS 位移，已含程式的 +1 偏移。
	ClericSpellSlotTableOffset = 0x4035
	// MagicUserSpellSlotTableOffset 是法師表的 DS 位移，同樣已含 +1。
	MagicUserSpellSlotTableOffset = 0x414d
	// SpellSlotLevels 是每列的欄數，也就是遊戲支援的法術等級數。
	SpellSlotLevels = 3
	// SpellSlotFirstLevel 是表涵蓋的最低角色等級。overlay-23 對等級 1 直接
	// 跳過這一段（表裡那一列是 FFh），第 1 級的格數由別處寫入。
	SpellSlotFirstLevel = 2
	// SpellSlotLastLevel 是表涵蓋的最高角色等級；再上去是零，
	// 與 spec 071 的等級上限（牧師 6、法師 6）一致。
	SpellSlotLastLevel = 6

	// ClericSpellSlotRecordOffset 是牧師格數在角色記錄裡的起點，
	// 第 1 級在 `+0B2h`。
	ClericSpellSlotRecordOffset = 0xb2
	// MagicUserSpellSlotRecordOffset 同上，第 1 級在 `+0B5h`。
	MagicUserSpellSlotRecordOffset = 0xb5
	// wisdomOffset 是睿智在角色記錄裡的位置。
	wisdomOffset = 0x12
)

// SpellSlots 是三個法術等級的可記憶數。
type SpellSlots [SpellSlotLevels]uint8

// ParseSpellSlotTable 解出一張格數表：從角色等級 2 到 6，每級三個 byte。
func ParseSpellSlotTable(raw []byte) ([]SpellSlots, error) {
	want := (SpellSlotLastLevel + 1) * SpellSlotLevels
	if len(raw) < want {
		return nil, fmt.Errorf("Pool spell slot table has %d bytes, want at least %d", len(raw), want)
	}
	table := make([]SpellSlots, SpellSlotLastLevel+1)
	for level := SpellSlotFirstLevel; level <= SpellSlotLastLevel; level++ {
		for spellLevel := 1; spellLevel <= SpellSlotLevels; spellLevel++ {
			// base 已經含了程式的 +1 偏移，所以這裡用 spellLevel-1 取值；
			// 兩邊都加一次會整批位移一格。
			table[level][spellLevel-1] = raw[level*SpellSlotLevels+spellLevel-1]
		}
	}
	return table, nil
}

// WisdomBonusSlots 是 overlay-23 `01ADh` 的睿智加成。
//
// 原版的兩個細節照抄：加成是**逐段各加一次**（睿智 13 與 14 各給第一級一格），
// 而且**只加在本來就有格子的等級上**（`cmp .., 0; jbe` 跳過）。
// 少了那個守衛，一個第 2 級的牧師會憑空得到第三級法術的格子。
func WisdomBonusSlots(wisdom int, base SpellSlots) SpellSlots {
	out := base
	add := func(index int, when bool) {
		if when && out[index] > 0 {
			out[index]++
		}
	}
	add(0, wisdom > 12)
	add(0, wisdom > 13)
	add(1, wisdom > 14)
	add(1, wisdom > 15)
	add(2, wisdom > 16)
	return out
}

// ReadDOSSpellSlotTables 讀出牧師與法師的格數表。
func ReadDOSSpellSlotTables(zipPath string) (cleric, magicUser []SpellSlots, err error) {
	raw, err := readStartExecutable(zipPath)
	if err != nil {
		return nil, nil, err
	}
	read := func(offset int) ([]SpellSlots, error) {
		start := offset + startDataSegmentFileDelta
		end := start + (SpellSlotLastLevel+1)*SpellSlotLevels
		if len(raw) < end {
			return nil, fmt.Errorf("START.EXE is %d bytes, the slot table needs %d", len(raw), end)
		}
		return ParseSpellSlotTable(raw[start:end])
	}
	if cleric, err = read(ClericSpellSlotTableOffset); err != nil {
		return nil, nil, err
	}
	if magicUser, err = read(MagicUserSpellSlotTableOffset); err != nil {
		return nil, nil, err
	}
	return cleric, magicUser, nil
}

// RecordSpellSlots 讀出角色記錄裡已經算好的格數。
func RecordSpellSlots(record []byte, class SpellClass) (SpellSlots, error) {
	offset := ClericSpellSlotRecordOffset
	if class == SpellClassMagicUser {
		offset = MagicUserSpellSlotRecordOffset
	}
	var slots SpellSlots
	if len(record) < offset+SpellSlotLevels {
		return slots, fmt.Errorf("Pool character record is %d bytes, the slot fields need %d",
			len(record), offset+SpellSlotLevels)
	}
	copy(slots[:], record[offset:offset+SpellSlotLevels])
	return slots, nil
}

// RecordWisdom 取出角色記錄的睿智。
func RecordWisdom(record []byte) (int, error) {
	if len(record) <= wisdomOffset {
		return 0, fmt.Errorf("Pool character record is %d bytes, wisdom is at %#02x", len(record), wisdomOffset)
	}
	return int(record[wisdomOffset]), nil
}
